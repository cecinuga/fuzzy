package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

const selectionSpec = `{"openapi":"3.0.3","info":{"title":"t","version":"1"},"paths":{
	"/reports":{"post":{"requestBody":{"required":true,"content":{"application/protobuf":{"schema":{"type":"string"}}}}}},
	"/users":{
		"get":{"tags":["reads"]},
		"post":{"tags":["writes","users"]}
	},
	"/users/{id}":{
		"parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"integer"}}],
		"get":{"tags":["reads"]},
		"delete":{}
	}
}}`

func writeSelectionSpec(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(path, []byte(selectionSpec), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// recorder is a server that remembers the endpoints it was called on.
func recorder(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	seen := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		path := r.URL.Path
		if strings.HasPrefix(path, "/users/") {
			path = "/users/{id}"
		}
		seen[r.Method+" "+path] = true
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		called := make([]string, 0, len(seen))
		for endpoint := range seen {
			called = append(called, endpoint)
		}
		slices.Sort(called)
		return called
	}
}

func TestSelectionDecidesWhatIsCalled(t *testing.T) {
	cases := map[string]struct {
		args []string
		want []string
	}{
		"everything": {nil, []string{"DELETE /users/{id}", "GET /users", "GET /users/{id}", "POST /users"}},
		"method":     {[]string{"--include", "GET"}, []string{"GET /users", "GET /users/{id}"}},
		"glob":       {[]string{"--include", "/users/*"}, []string{"DELETE /users/{id}", "GET /users/{id}"}},
		"pasted":     {[]string{"--include", "GET /users/{id}"}, []string{"GET /users/{id}"}},
		"tag":        {[]string{"--include", "tag:users"}, []string{"POST /users"}},
		"no deletes": {[]string{"--exclude", "DELETE"}, []string{"GET /users", "GET /users/{id}", "POST /users"}},
		"both":       {[]string{"--include", "/users/**", "--exclude", "GET,DELETE"}, []string{"POST /users"}},
		"old style":  {[]string{"-include", "GET /users"}, []string{"GET /users"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv, called := recorder(t)
			args := append([]string{"--remote", srv.URL, "--spec", writeSelectionSpec(t), "-n", "2", "-o", filepath.Join(t.TempDir(), "r.md")}, tc.args...)
			code, _, stderr := runCLI(t, args...)
			if code != exitOK {
				t.Fatalf("exit %d: %s", code, stderr)
			}
			if got := called(); !slices.Equal(got, tc.want) {
				t.Errorf("called %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSelectionIsShownInTheOutputAndReport(t *testing.T) {
	srv, _ := recorder(t)
	report := filepath.Join(t.TempDir(), "r.json")
	code, stdout, stderr := runCLI(t, "--remote", srv.URL, "--spec", writeSelectionSpec(t), "-n", "1", "--format", "json", "-o", report,
		"--include", "/users/**", "--exclude", "DELETE")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "fuzzing 3 of 5 endpoints") {
		t.Errorf("stdout = %s", stdout)
	}

	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Overview struct {
			Selection struct {
				Include, Exclude []string
				LeftOut          int `json:"endpointsLeftOut"`
			}
		}
		Endpoints []struct{ Path string }
		Skipped   []struct{ Endpoint string }
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	// The unfuzzable endpoint counts as left out, not as skipped: it wasn't selected.
	s := doc.Overview.Selection
	if !slices.Equal(s.Include, []string{"/users/**"}) || !slices.Equal(s.Exclude, []string{"DELETE"}) || s.LeftOut != 2 {
		t.Errorf("selection = %+v", s)
	}
	if len(doc.Endpoints) != 3 || len(doc.Skipped) != 0 {
		t.Errorf("%d endpoints, skipped %v", len(doc.Endpoints), doc.Skipped)
	}
}

func TestWithoutASelectionTheStartLineIsUnchanged(t *testing.T) {
	srv, _ := recorder(t)
	_, stdout, stderr := runCLI(t, "--remote", srv.URL, "--spec", writeSelectionSpec(t), "-n", "1", "-o", filepath.Join(t.TempDir(), "r.md"))
	if !strings.Contains(stdout, "[+] fuzzing 4 endpoints of ") {
		t.Errorf("stdout = %s, stderr = %s", stdout, stderr)
	}
}

func TestSelectionErrors(t *testing.T) {
	cases := map[string]struct {
		args []string
		code int
		want string
	}{
		"bad selector":      {[]string{"--include", "FETCH /a"}, exitUsage, "--include: invalid selector"},
		"bad exclude":       {[]string{"--exclude", "/a*"}, exitUsage, "--exclude: invalid selector"},
		"typo":              {[]string{"--exclude", "DELETE /usres/*"}, exitFailure, `no endpoint of the spec matches --exclude "DELETE /usres/*"`},
		"nothing left":      {[]string{"--exclude", "/**"}, exitFailure, "leave no endpoint to fuzz"},
		"only a skipped":    {[]string{"--include", "POST /reports"}, exitFailure, "leave no endpoint to fuzz"},
		"listing too":       {[]string{"--list", "--exclude", "/nope"}, exitFailure, "matches"},
		"listing nothing":   {[]string{"--list", "--exclude", "/**"}, exitFailure, "leave no endpoint"},
		"listing bad flags": {[]string{"--list", "--include", ""}, exitUsage, "--include"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srv, called := recorder(t)
			dir := t.TempDir()
			t.Chdir(dir)
			args := append([]string{"--remote", srv.URL, "--spec", writeSelectionSpec(t)}, tc.args...)
			code, _, stderr := runCLI(t, args...)
			if code != tc.code || !strings.Contains(stderr, tc.want) {
				t.Errorf("exit %d (want %d), stderr %q (want %q)", code, tc.code, stderr, tc.want)
			}
			if got := called(); len(got) != 0 {
				t.Errorf("requests were sent: %v", got)
			}
			if entries, _ := os.ReadDir(dir); len(entries) != 0 {
				t.Errorf("no report should be created, found %v", entries)
			}
		})
	}
}

func TestListSendsNothingAndWritesNoReport(t *testing.T) {
	srv, called := recorder(t)
	spec := writeSelectionSpec(t)
	dir := t.TempDir()
	t.Chdir(dir)

	cases := map[string]struct {
		args   []string
		stdout string
		stderr []string
	}{
		"without remote": {
			[]string{"--spec", spec, "--list"},
			"GET /users\treads\nPOST /users\twrites, users\nGET /users/{id}\treads\nDELETE /users/{id}\n",
			[]string{"4 of 5 endpoints would be fuzzed", "skipped POST /reports"},
		},
		"with remote": {
			[]string{"--remote", srv.URL, "--spec", spec, "--list"},
			"GET /users\treads\nPOST /users\twrites, users\nGET /users/{id}\treads\nDELETE /users/{id}\n",
			[]string{"4 of 5"},
		},
		"with a selection": {
			[]string{"--spec", spec, "--list", "--exclude", "DELETE", "--exclude", "POST"},
			"GET /users\treads\nGET /users/{id}\treads\n",
			[]string{"2 of 5"},
		},
		"old style": {
			[]string{"-spec", spec, "-list", "-include", "DELETE"},
			"DELETE /users/{id}\n",
			[]string{"1 of 5"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := runCLI(t, tc.args...)
			if code != exitOK {
				t.Fatalf("exit %d: %s", code, stderr)
			}
			if stdout != tc.stdout {
				t.Errorf("stdout = %q, want %q", stdout, tc.stdout)
			}
			for _, want := range tc.stderr {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr %q lacks %q", stderr, want)
				}
			}
		})
	}

	if got := called(); len(got) != 0 {
		t.Errorf("requests were sent: %v", got)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("--list must not write a report, found %v", entries)
	}
}

func TestListStillValidatesARemoteGiven(t *testing.T) {
	code, _, stderr := runCLI(t, "--spec", writeSelectionSpec(t), "--list", "--remote", "ftp://x")
	if code != exitUsage || !strings.Contains(stderr, "--remote") {
		t.Errorf("exit %d, stderr %q", code, stderr)
	}
}

func TestRemoteIsRequiredWithoutList(t *testing.T) {
	code, _, stderr := runCLI(t, "--spec", writeSelectionSpec(t), "--include", "GET")
	if code != exitUsage || !strings.Contains(stderr, `required flag(s) "remote" not set`) {
		t.Errorf("exit %d, stderr %q", code, stderr)
	}
}

func TestListNeutralizesControlCharactersFromTheSpec(t *testing.T) {
	path := filepath.Join(t.TempDir(), "spec.json")
	spec := `{"openapi":"3.0.3","paths":{"/a\u001b[31mb":{"get":{"tags":["x\u001b[2Jy"]}}}}`
	if err := os.WriteFile(path, []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI(t, "--spec", path, "--list")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if strings.Contains(stdout, "\x1b") {
		t.Errorf("a raw escape sequence reached the terminal: %q", stdout)
	}
}

// The way a failing endpoint gets replayed: same seed, only that endpoint.
func TestSeedReplaysOneEndpointOfABiggerRun(t *testing.T) {
	spec := writeSelectionSpec(t)
	srv, _ := recorder(t)
	dir := t.TempDir()

	inputs := func(report string, extra ...string) map[string][]string {
		t.Helper()
		args := append([]string{"--remote", srv.URL, "--spec", spec, "--seed", "42", "-n", "20", "-c", "4", "--format", "json", "-o", report}, extra...)
		if code, _, stderr := runCLI(t, args...); code != exitOK {
			t.Fatalf("exit %d: %s", code, stderr)
		}
		data, err := os.ReadFile(report)
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Endpoints []struct {
				Method, Path string
				Results      struct {
					Items []struct{ Endpoint, Input string }
				}
			}
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		out := map[string][]string{}
		for _, e := range doc.Endpoints {
			for _, item := range e.Results.Items {
				out[e.Method+" "+e.Path] = append(out[e.Method+" "+e.Path], item.Endpoint+" "+item.Input)
			}
		}
		return out
	}

	const endpoint = "DELETE /users/{id}"
	full := inputs(filepath.Join(dir, "full.json"))
	replay := inputs(filepath.Join(dir, "replay.json"), "--include", endpoint)
	if len(full[endpoint]) != 20 || !slices.Equal(replay[endpoint], full[endpoint]) {
		t.Errorf("replay differs from the full run:\n%v\n%v", replay[endpoint], full[endpoint])
	}
}

func TestSelectionFlagsHaveShortNames(t *testing.T) {
	srv, called := recorder(t)
	args := []string{"--rem", srv.URL, "--spc", writeSelectionSpec(t), "-n", "1", "-o", filepath.Join(t.TempDir(), "r.md"), "--icl", "/users/**", "--exc", "GET,DELETE"}
	if code, _, stderr := runCLI(t, args...); code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if got, want := called(), []string{"POST /users"}; !slices.Equal(got, want) {
		t.Errorf("called %v, want %v", got, want)
	}

	code, stdout, stderr := runCLI(t, "--spc", writeSelectionSpec(t), "--lst", "--icl", "DELETE")
	if code != exitOK || stdout != "DELETE /users/{id}\n" {
		t.Errorf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

// A preview must leave nothing behind, whatever the other flags ask for.
func TestListCreatesNoLogOrReport(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	code, _, stderr := runCLI(t, "--spec", writeSelectionSpec(t), "--list", "--log", "run.log", "-o", "r.json", "--incremental")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("--list left files behind: %v", entries)
	}
}

func TestEveryFormatShowsTheSelection(t *testing.T) {
	cases := map[string][]string{
		"md":   {"Included", "Excluded", `/users/\*\*`, "DELETE"},
		"txt":  {"Included:", "Excluded:", "/users/**", "DELETE"},
		"html": {"Included", "Excluded", "/users/**", "DELETE"},
		"xml":  {"<include>/users/**</include>", "<exclude>DELETE</exclude>", "<endpointsLeftOut>2</endpointsLeftOut>"},
		"json": {`"endpointsLeftOut": 2`, "/users/**", "DELETE"},
	}
	for extension, wants := range cases {
		t.Run(extension, func(t *testing.T) {
			srv, _ := recorder(t)
			report := filepath.Join(t.TempDir(), "r."+extension)
			code, _, stderr := runCLI(t, "--remote", srv.URL, "--spec", writeSelectionSpec(t), "-n", "1", "-o", report, "--include", "/users/**", "--exclude", "DELETE")
			if code != exitOK {
				t.Fatalf("exit %d: %s", code, stderr)
			}
			data, err := os.ReadFile(report)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range wants {
				if !strings.Contains(string(data), want) {
					t.Errorf("report lacks %q:\n%s", want, data)
				}
			}
		})
	}
}

// The README promises this: -n only says how far along the same inputs a run goes.
func TestASmallerRunSendsTheFirstInputsOfABiggerOne(t *testing.T) {
	spec := writeSelectionSpec(t)
	srv, _ := recorder(t)
	dir := t.TempDir()

	inputs := func(requests string) map[string][]string {
		t.Helper()
		report := filepath.Join(dir, requests+".json")
		code, _, stderr := runCLI(t, "--remote", srv.URL, "--spec", spec, "--seed", "42", "-n", requests, "-c", "4", "--format", "json", "-o", report)
		if code != exitOK {
			t.Fatalf("exit %d: %s", code, stderr)
		}
		data, err := os.ReadFile(report)
		if err != nil {
			t.Fatal(err)
		}
		var parsed struct {
			Endpoints []struct {
				Method, Path string
				Results      struct {
					Items []struct{ Input string }
				}
			}
		}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatal(err)
		}
		out := map[string][]string{}
		for _, e := range parsed.Endpoints {
			for _, item := range e.Results.Items {
				out[e.Method+" "+e.Path] = append(out[e.Method+" "+e.Path], item.Input)
			}
		}
		return out
	}

	small, big := inputs("10"), inputs("30")
	if len(small) == 0 {
		t.Fatal("no results")
	}
	for endpoint, first := range small {
		if len(first) != 10 || len(big[endpoint]) != 30 || !slices.Equal(first, big[endpoint][:10]) {
			t.Errorf("%s: %v is not the start of %v", endpoint, first, big[endpoint])
		}
	}
}
