package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const sampleSpec = "../../testdata/api.json"

// strictAPI implements testdata/api.json and rejects, with a 422, any request
// its spec would not allow. A clean run therefore proves the requests were valid.
func strictAPI(violations *[]string, mu *sync.Mutex) http.Handler {
	fail := func(r *http.Request, w http.ResponseWriter, format string, args ...any) {
		msg := r.Method + " " + r.URL.String() + ": " + fmt.Sprintf(format, args...)
		mu.Lock()
		*violations = append(*violations, msg)
		mu.Unlock()
		http.Error(w, msg, http.StatusUnprocessableEntity)
	}
	intIn := func(raw string, lo, hi int) bool {
		v, err := strconv.Atoi(raw)
		return err == nil && v >= lo && v <= hi
	}
	skuRe := regexp.MustCompile(`^[A-Z]{3}-[0-9]{4}$`)
	uuidRe := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	runes := func(s string) int { return len([]rune(s)) }

	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/user/{id}", func(w http.ResponseWriter, r *http.Request) {
		if !intIn(r.PathValue("id"), 1, 500) {
			fail(r, w, "id")
			return
		}
		if id := r.Header.Get("X-Request-Id"); id != "" && !uuidRe.MatchString(id) {
			fail(r, w, "X-Request-Id %q", id)
			return
		}
		if fields := r.URL.Query().Get("fields"); r.URL.Query().Has("fields") {
			seen := map[string]bool{}
			for f := range strings.SplitSeq(fields, ",") {
				if (f != "name" && f != "email" && f != "age") || seen[f] {
					fail(r, w, "fields %q", fields)
					return
				}
				seen[f] = true
			}
		}
		w.WriteHeader(404)
	})

	mux.HandleFunc("DELETE /v1/user/{id}", func(w http.ResponseWriter, r *http.Request) {
		force := r.URL.Query().Get("force")
		if !intIn(r.PathValue("id"), 1, 500) || (r.URL.Query().Has("force") && force != "true" && force != "false") {
			fail(r, w, "delete")
			return
		}
		w.WriteHeader(204)
	})

	mux.HandleFunc("POST /v1/user", func(w http.ResponseWriter, r *http.Request) {
		var u map[string]any
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&u); err != nil || r.Header.Get("Content-Type") != "application/json" {
			fail(r, w, "bad JSON: %v", err)
			return
		}
		if age, ok := u["age"].(float64); !ok || age < 18 || age > 120 || age != float64(int(age)) {
			fail(r, w, "age %v", u["age"])
			return
		}
		if email, ok := u["email"].(string); !ok || !strings.Contains(email, "@") {
			fail(r, w, "email %v", u["email"])
			return
		}
		if _, ok := u["id"]; ok {
			fail(r, w, "readOnly id sent")
			return
		}
		if name, present := u["name"]; present && name != nil {
			if s, ok := name.(string); !ok || runes(s) < 1 || runes(s) > 30 {
				fail(r, w, "name %v", name)
				return
			}
		}
		if score, present := u["score"]; present {
			if f, ok := score.(float64); !ok || f < 0 || f > 10 || f*2 != float64(int(f*2)) {
				fail(r, w, "score %v", score)
				return
			}
		}
		pet, ok := u["pet"].(map[string]any)
		switch {
		case !ok:
			fail(r, w, "pet %v", u["pet"])
			return
		case pet["kind"] == "kitty":
			if lives, ok := pet["lives"].(float64); !ok || lives < 1 || lives > 9 {
				fail(r, w, "cat %v", pet)
				return
			}
		case pet["kind"] == "Dog":
			if _, ok := pet["bark"].(bool); !ok {
				fail(r, w, "dog %v", pet)
				return
			}
		default:
			fail(r, w, "discriminator %v", pet["kind"])
			return
		}
		w.WriteHeader(201)
	})

	mux.HandleFunc("GET /v1/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if n := runes(q.Get("q")); n < 2 || n > 20 {
			fail(r, w, "q")
			return
		}
		if ratio, err := strconv.ParseFloat(q.Get("ratio"), 64); err != nil || ratio < 0 || ratio >= 1 {
			fail(r, w, "ratio")
			return
		}
		if !skuRe.MatchString(q.Get("sku")) {
			fail(r, w, "sku")
			return
		}
		if s := q.Get("sort"); s != "asc" && s != "desc" {
			fail(r, w, "sort")
			return
		}
		if q.Has("page[cursor]") && !q.Has("page[size]") {
			fail(r, w, "page.size is required")
			return
		}
		if q.Has("page[size]") && !intIn(q.Get("page[size]"), 1, 50) {
			fail(r, w, "page.size")
			return
		}
		if c, err := r.Cookie("session"); err == nil {
			value, _ := url.QueryUnescape(c.Value)
			if runes(value) > 16 {
				fail(r, w, "session")
				return
			}
		}
		w.Write([]byte("found something"))
	})

	mux.HandleFunc("POST /v1/upload", func(w http.ResponseWriter, r *http.Request) {
		media, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || media != "multipart/form-data" {
			fail(r, w, "content type")
			return
		}
		form, err := multipart.NewReader(r.Body, params["boundary"]).ReadForm(1 << 20)
		if err != nil {
			fail(r, w, "form: %v", err)
			return
		}
		if kind := form.Value["kind"]; len(kind) != 1 || (kind[0] != "avatar" && kind[0] != "document") {
			fail(r, w, "kind %v", kind)
			return
		}
		if len(form.File["file"]) != 1 {
			fail(r, w, "file part missing")
			return
		}
		w.WriteHeader(200)
	})

	mux.HandleFunc("POST /v1/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			fail(r, w, "content type")
			return
		}
		if n := runes(r.PostFormValue("username")); n < 3 || n > 12 {
			fail(r, w, "username")
			return
		}
		if n := runes(r.PostFormValue("password")); n < 8 || n > 32 {
			fail(r, w, "password")
			return
		}
		w.WriteHeader(401)
	})

	mux.HandleFunc("POST /v1/legacy", func(w http.ResponseWriter, r *http.Request) {
		if contentType := r.Header.Get("Content-Type"); contentType != "application/xml" {
			fail(r, w, "Content-Type %q", contentType)
			return
		}
		var document struct {
			XMLName xml.Name
			Members []struct {
				XMLName xml.Name
			} `xml:",any"`
		}
		if err := xml.NewDecoder(r.Body).Decode(&document); err != nil {
			fail(r, w, "body is not well-formed XML: %v", err)
			return
		}
		// The schema is an inline object, so the root has the default name, and
		// an object in a body needs at least one member to serialize to anything.
		if document.XMLName.Local != "root" || len(document.Members) == 0 {
			fail(r, w, "unexpected document: %+v", document)
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fail(r, w, "no such route")
	})
	return mux
}

func runCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(context.Background(), args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestEveryRequestIsValidAgainstAStrictServer(t *testing.T) {
	var violations []string
	var mu sync.Mutex
	srv := httptest.NewServer(strictAPI(&violations, &mu))
	defer srv.Close()
	report := filepath.Join(t.TempDir(), "report.md")

	code, stdout, stderr := runCLI(t, "--remote", srv.URL, "--spec", sampleSpec, "-n", "400", "-c", "16", "-o", report, "--seed", "1")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if len(violations) > 0 {
		t.Fatalf("%d invalid requests, first ones:\n%s", len(violations), strings.Join(violations[:min(10, len(violations))], "\n"))
	}
	if !strings.Contains(stdout, "2800 requests sent, 0 without a response") {
		t.Errorf("stdout = %s", stdout)
	}

	doc, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## GET /user/{id}", "## POST /user", "## DELETE /user/{id}", "## GET /search", "## POST /upload", "## POST /login", "## POST /legacy"} {
		if !strings.Contains(string(doc), want) {
			t.Errorf("report lacks %q", want)
		}
	}
}

func TestDictionaryValuesAreUsedAndUncoveredTypesRandomized(t *testing.T) {
	var mu sync.Mutex
	got := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		for key, values := range r.URL.Query() {
			got[key+"="+values[0]] = true
		}
		got["path="+r.URL.Path] = true
	}))
	defer srv.Close()

	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.json")
	os.WriteFile(spec, []byte(`{"openapi":"3.0.0","paths":{"/u/{id}":{"get":{"parameters":[
		{"name":"id","in":"path","required":true,"schema":{"type":"integer","minimum":5000,"maximum":5001}},
		{"name":"who","in":"query","required":true,"schema":{"type":"string"}}
	]}}}}`), 0o600)
	dict := filepath.Join(dir, "words.txt")
	os.WriteFile(dict, []byte("alice\nbob\n<script>\n7\n"), 0o600)

	// No -n: the run grows to cover the dictionary if needed.
	code, _, stderr := runCLI(t, "--remote", srv.URL, "--spec", spec, "--dict", dict, "-o", filepath.Join(dir, "r.md"))
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	for _, want := range []string{"who=alice", "who=bob", "who=<script>", "who=7"} {
		if !got[want] {
			t.Errorf("dictionary value %q was never sent: %v", want, got)
		}
	}
	// 7 is the only int in the dictionary and is outside the schema's range.
	if got["path=/u/7"] {
		t.Error("a dictionary value outside the schema must not be used")
	}
	if !got["path=/u/5000"] || !got["path=/u/5001"] {
		t.Errorf("expected random values within the schema's range: %v", got)
	}
}

func TestDefaultRequestCountCoversTheDictionary(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen[r.URL.Query().Get("w")] = true
		mu.Unlock()
	}))
	defer srv.Close()

	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.json")
	os.WriteFile(spec, []byte(`{"openapi":"3.0.0","paths":{"/s":{"get":{"parameters":[{"name":"w","in":"query","required":true,"schema":{"type":"string"}}]}}}}`), 0o600)
	var words strings.Builder
	for i := range 350 {
		fmt.Fprintf(&words, "word%d\n", i)
	}
	dict := filepath.Join(dir, "words.txt")
	os.WriteFile(dict, []byte(words.String()), 0o600)

	code, stdout, stderr := runCLI(t, "--remote", srv.URL, "--spec", spec, "--dict", dict, "-o", filepath.Join(dir, "r.md"))
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if len(seen) != 350 {
		t.Errorf("%d distinct words sent, want 350. stdout: %s", len(seen), stdout)
	}
}

func TestStatusFlagFiltersTheReport(t *testing.T) {
	var violations []string
	var mu sync.Mutex
	srv := httptest.NewServer(strictAPI(&violations, &mu))
	defer srv.Close()
	report := filepath.Join(t.TempDir(), "report.md")

	code, _, stderr := runCLI(t, "--remote", srv.URL, "--spec", sampleSpec, "-n", "20", "--status", "201,401", "-o", report)
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	data, _ := os.ReadFile(report)
	doc := string(data)
	for _, want := range []string{"| POST /v1/user | 201 |", "| POST /v1/login | 401 |"} {
		if !strings.Contains(doc, want) {
			t.Errorf("report lacks %q", want)
		}
	}
	for _, reject := range []string{"| 204 |", "| 404 |", "| 200 |"} {
		if strings.Contains(doc, reject) {
			t.Errorf("filtered status present: %q", reject)
		}
	}
}

func TestRemoteAsHostPortReachesTheServer(t *testing.T) {
	var violations []string
	var mu sync.Mutex
	srv := httptest.NewServer(strictAPI(&violations, &mu))
	defer srv.Close()

	hostPort := strings.TrimPrefix(srv.URL, "http://")
	code, stdout, stderr := runCLI(t, "--remote", hostPort, "--spec", sampleSpec, "-n", "5", "-o", filepath.Join(t.TempDir(), "r.md"))
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "0 without a response") {
		t.Errorf("stdout = %s", stdout)
	}
	if len(violations) > 0 {
		t.Errorf("invalid requests: %v", violations[:min(5, len(violations))])
	}
}

func TestExitCodes(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		args []string
		code int
		want string
	}{
		{"help", []string{"-h"}, exitOK, ""},
		{"missing flags", nil, exitUsage, `required flag(s) "remote", "spec" not set`},
		{"bad status", []string{"--remote", "http://x.io", "--spec", sampleSpec, "--status", "nope"}, exitUsage, "--status"},
		{"missing spec", []string{"--remote", "http://x.io", "--spec", filepath.Join(dir, "no.json")}, exitFailure, "read spec"},
		{"missing dict", []string{"--remote", "http://x.io", "--spec", sampleSpec, "--dict", filepath.Join(dir, "no.txt")}, exitFailure, "read dictionary"},
		{"bad output", []string{"--remote", "http://x.io", "--spec", sampleSpec, "-o", filepath.Join(dir, "no", "dir", "r.md")}, exitFailure, "create report"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, _, stderr := runCLI(t, tc.args...)
			if code != tc.code || !strings.Contains(stderr, tc.want) {
				t.Errorf("exit %d (want %d), stderr %q (want %q)", code, tc.code, stderr, tc.want)
			}
		})
	}
}

func TestAnUnreachableTargetStillProducesAReport(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	remote := srv.URL
	srv.Close()
	report := filepath.Join(t.TempDir(), "report.md")

	code, stdout, _ := runCLI(t, "--remote", remote, "--spec", sampleSpec, "-n", "1", "-o", report)
	if code != exitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stdout, "7 requests sent, 7 without a response") {
		t.Errorf("stdout = %s", stdout)
	}
	data, _ := os.ReadFile(report)
	if !strings.Contains(string(data), "ERR ×1") {
		t.Errorf("report does not show the failures:\n%s", data)
	}
}

func TestCancelledRunWritesAPartialReport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { once.Do(cancel) }))
	defer srv.Close()
	report := filepath.Join(t.TempDir(), "report.md")

	code := run(ctx, []string{"--remote", srv.URL, "--spec", sampleSpec, "-n", "1000000", "-o", report}, io.Discard, io.Discard)
	if code != exitOK {
		t.Fatalf("exit %d", code)
	}
	data, _ := os.ReadFile(report)
	if !strings.Contains(string(data), "interrupted") {
		t.Errorf("a partial report must say so:\n%s", data)
	}
}

// fuzzOneEndpoint runs against a one-endpoint spec, calling onRequest with the
// number of each request, and returns the CLI's exit code and stderr.
func fuzzOneEndpoint(t *testing.T, requests int, output string, onRequest func(n int), extra ...string) (int, string) {
	t.Helper()
	spec := filepath.Join(t.TempDir(), "spec.json")
	body := `{"openapi": "3.0.3", "info": {"title": "t", "version": "1"},
		"paths": {"/a": {"get": {"responses": {"200": {"description": "ok"}}}}}}`
	if err := os.WriteFile(spec, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		n++
		if onRequest != nil {
			onRequest(n)
		}
	}))
	defer srv.Close()
	args := append([]string{"--remote", srv.URL, "--spec", spec, "--format", "json", "-c", "1", "-n", strconv.Itoa(requests), "-o", output}, extra...)
	code, _, stderr := runCLI(t, args...)
	return code, stderr
}

type reportState struct {
	Interrupted bool `json:"interrupted"`
	Overview    struct {
		RequestsSent int `json:"requestsSent"`
	} `json:"overview"`
}

func readReportState(t *testing.T, path string) reportState {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var state reportState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("not a complete report: %v\n%s", err, data)
	}
	return state
}

func TestReportIsOnlyWrittenAtTheEndByDefault(t *testing.T) {
	report := filepath.Join(t.TempDir(), "report.json")

	size := int64(-1)
	code, _ := fuzzOneEndpoint(t, 110, report, func(n int) {
		if n == 100 {
			info, err := os.Stat(report)
			if err != nil {
				t.Error(err)
				return
			}
			size = info.Size()
		}
	})
	if code != exitOK {
		t.Fatalf("exit %d", code)
	}

	if size != 0 {
		t.Errorf("report had %d bytes during the run, want none", size)
	}
	if final := readReportState(t, report); final.Interrupted || final.Overview.RequestsSent != 110 {
		t.Errorf("final report = %+v, want a complete one with 110 requests", final)
	}
}

func TestDefaultKeepsAHardLinkToTheReport(t *testing.T) {
	dir := t.TempDir()
	report := filepath.Join(dir, "report.json")
	if err := os.WriteFile(report, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Link(report, link); err != nil {
		t.Skip(err)
	}

	if code, _ := fuzzOneEndpoint(t, 20, report, nil); code != exitOK {
		t.Fatalf("exit %d", code)
	}
	if state := readReportState(t, link); state.Overview.RequestsSent != 20 {
		t.Errorf("the file was replaced instead of written: the link has %+v", state)
	}
}

func TestReportIsWrittenDuringTheRun(t *testing.T) {
	report := filepath.Join(t.TempDir(), "report.json")

	// One worker can't get past the checkpoint at 50 while it's being written, as
	// the records it sends fill up, so that one is on disk when request 60 arrives.
	// The next needs request 60 to complete, and this handler is holding it up.
	var midRun reportState
	code, _ := fuzzOneEndpoint(t, 110, report, func(n int) {
		if n == 60 {
			midRun = readReportState(t, report)
		}
	}, "--incremental")
	if code != exitOK {
		t.Fatalf("exit %d", code)
	}

	if !midRun.Interrupted || midRun.Overview.RequestsSent != 50 {
		t.Errorf("mid-run report = %+v, want a partial one with 50 requests", midRun)
	}
	final := readReportState(t, report)
	if final.Interrupted || final.Overview.RequestsSent != 110 {
		t.Errorf("final report = %+v, want a complete one with 110 requests", final)
	}
}

func TestCheckpointsLeaveNoTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	if code, _ := fuzzOneEndpoint(t, 110, filepath.Join(dir, "report.json"), nil, "--incremental"); code != exitOK {
		t.Fatalf("exit %d", code)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("directory holds %v, want only the report", entries)
	}
}

func TestReportKeepsItsModeAndSymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.json")
	if err := os.WriteFile(real, nil, 0o640); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(real, link); err != nil {
		t.Skip(err)
	}

	if code, _ := fuzzOneEndpoint(t, 20, link, nil, "--incremental"); code != exitOK {
		t.Fatalf("exit %d", code)
	}

	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the link was replaced: %v, %v", info, err)
	}
	info, err := os.Stat(real)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %o, want 640", info.Mode().Perm())
	}
	if state := readReportState(t, real); state.Overview.RequestsSent != 20 {
		t.Errorf("the link's target has %+v", state)
	}
}

func TestDeviceOutputIsNotReplaced(t *testing.T) {
	if _, err := os.Stat(os.DevNull); err != nil {
		t.Skip(err)
	}
	code, stderr := fuzzOneEndpoint(t, 20, os.DevNull, nil, "--incremental")
	if code != exitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr, "only written at the end") {
		t.Errorf("no warning that checkpoints are off, stderr = %q", stderr)
	}
	info, err := os.Stat(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().IsRegular() {
		t.Errorf("%s became a regular file", os.DevNull)
	}
}

func TestReportInAReadOnlyDirectoryIsStillWritten(t *testing.T) {
	dir := t.TempDir()
	report := filepath.Join(dir, "report.json")
	if err := os.WriteFile(report, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })
	if canCreateIn(dir) {
		t.Skip("permissions aren't enforced here, for example when running as root")
	}

	if code, _ := fuzzOneEndpoint(t, 20, report, nil, "--incremental"); code != exitOK {
		t.Fatalf("exit %d", code)
	}
	if state := readReportState(t, report); state.Interrupted || state.Overview.RequestsSent != 20 {
		t.Errorf("report = %+v", state)
	}
}
