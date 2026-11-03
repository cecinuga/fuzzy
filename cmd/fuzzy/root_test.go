package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

const oneEndpointSpec = `{"openapi":"3.0.0","paths":{"/s":{"get":{"parameters":[
	{"name":"q","in":"query","required":true,"schema":{"type":"integer"}}
]}}}}`

func writeSpec(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(path, []byte(oneEndpointSpec), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCanonicalArgs(t *testing.T) {
	cases := map[string]struct{ in, want []string }{
		"no args":           {[]string{}, []string{}},
		"modern untouched":  {[]string{"--remote", "x", "-n", "5", "-k"}, []string{"--remote", "x", "-n", "5", "-k"}},
		"single-dash long":  {[]string{"-remote", "x", "-string-max", "4"}, []string{"--remote", "x", "--string-max", "4"}},
		"double-dash short": {[]string{"--n", "5", "--k", "--H", "A: b"}, []string{"-n", "5", "-k", "-H", "A: b"}},
		"inline value":      {[]string{"-status=5xx", "--n=5"}, []string{"--status=5xx", "-n=5"}},
		"negative value":    {[]string{"-int-min", "-5", "-int-max=-1"}, []string{"--int-min", "-5", "--int-max=-1"}},
		"value like a flag": {[]string{"-o", "-remote", "-spec", "s"}, []string{"-o", "-remote", "--spec", "s"}},
		"bool takes none":   {[]string{"-k", "-remote", "x"}, []string{"-k", "--remote", "x"}},
		"short name":        {[]string{"--rem", "x", "--sta=5xx", "--inc", "--spc", "s"}, []string{"--rem", "x", "--sta=5xx", "--inc", "--spc", "s"}},
		"selection":         {[]string{"-include", "GET", "-exclude=/a", "-list", "-spec", "s"}, []string{"--include", "GET", "--exclude=/a", "--list", "--spec", "s"}},
		"help":              {[]string{"-help"}, []string{"--help"}},
		"version":           {[]string{"-version"}, []string{"--version"}},
		"after terminator":  {[]string{"-k", "--", "-remote"}, []string{"-k", "--", "-remote"}},
		"unknown long":      {[]string{"-nope"}, []string{"-nope"}},
		"short bundle":      {[]string{"-kc", "3"}, []string{"-kc", "3"}},
		"subcommand":        {[]string{"completion", "-remote"}, []string{"completion", "-remote"}},
		"shell completion":  {[]string{"__complete", "--c"}, []string{"__complete", "--c"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := canonicalArgs(newRootCommand(), tc.in); !slices.Equal(got, tc.want) {
				t.Errorf("canonicalArgs(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}

	if canonicalArgs(newRootCommand(), nil) == nil {
		t.Error("nil args must stay non-nil, or Cobra reads os.Args instead")
	}
}

// Commands written for the flag package must keep working, in both of its spellings.
func TestPreCobraCommandLinesStillWork(t *testing.T) {
	spec := writeSpec(t)

	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("X-Token") != "a:b" {
			t.Errorf("X-Token = %q", r.Header.Get("X-Token"))
		}
	}))
	defer srv.Close()

	cases := map[string]func(report string) []string{
		"single dash": func(report string) []string {
			return []string{
				"-remote", srv.URL, "-spec", spec, "-status", "2xx", "-string-min", "1", "-string-max", "4",
				"-int-min", "-5", "-int-max", "-1", "-float-min", "0", "-float-max", "1", "-charset", "alnum",
				"-n", "3", "-c", "2", "-timeout", "5s", "-o", report, "-k", "-seed", "9", "-H", "X-Token: a:b",
			}
		},
		"double dash": func(report string) []string {
			return []string{
				"--remote=" + srv.URL, "--spec=" + spec, "--status=2xx", "--string-min=1", "--string-max=4",
				"--int-min=-5", "--int-max=-1", "--float-min=0", "--float-max=1", "--charset=alnum",
				"--n=3", "--c=2", "--timeout=5s", "--o=" + report, "--k", "--seed=9", "--H=X-Token: a:b",
			}
		},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			requests.Store(0)
			report := filepath.Join(t.TempDir(), "r.md")

			code, stdout, stderr := runCLI(t, args(report)...)
			if code != exitOK {
				t.Fatalf("exit %d: %s", code, stderr)
			}
			if requests.Load() != 3 || !strings.Contains(stdout, "(seed 9)") {
				t.Errorf("%d requests, stdout = %s", requests.Load(), stdout)
			}
			if _, err := os.Stat(report); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestUsageErrorsExitWithAHint(t *testing.T) {
	spec := writeSpec(t)
	base := []string{"--remote", "http://x.io", "--spec", spec}
	cases := map[string]struct {
		args []string
		want string
	}{
		"no flags":     {nil, `required flag(s) "remote", "spec" not set`},
		"no spec":      {[]string{"--remote", "http://x.io"}, `required flag(s) "spec" not set`},
		"unknown flag": {append(slices.Clone(base), "--nope"), "unknown flag: --nope"},
		"stray arg":    {append(slices.Clone(base), "extra"), `unknown command "extra"`},
		"invalid":      {append(slices.Clone(base), "-c", "0"), "--concurrency"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := runCLI(t, tc.args...)
			if code != exitUsage {
				t.Errorf("exit %d, want %d", code, exitUsage)
			}
			if !strings.Contains(stderr, "[!] ") || !strings.Contains(stderr, tc.want) || !strings.Contains(stderr, "--help") {
				t.Errorf("stderr = %q, want the error %q and a hint", stderr, tc.want)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want nothing", stdout)
			}
		})
	}
}

func TestHelpListsEveryFlag(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}, {"-help"}} {
		code, stdout, stderr := runCLI(t, args...)
		if code != exitOK || stderr != "" {
			t.Errorf("%v: exit %d, stderr %q", args, code, stderr)
		}
		for _, flag := range []string{
			"--rem, -remote", "--spc, -spec", "--sta, -status", "--dic, -dict", "--smn, -string-min", "--smx, -string-max", "--imn, -int-min", "--imx, -int-max",
			"--fmn, -float-min", "--fmx, -float-max", "--chr, -charset", "--n, -requests", "--c, -concurrency", "--tmo, -timeout", "--o, -output", "--fmt, -format",
			"--inc, -incremental", "--log, -log", "--H, -header", "--k, -insecure", "--sd, -seed", "--h, -help", "--v, -version",
			"--icl, -include", "--exc, -exclude", "--lst, -list",
		} {
			if !strings.Contains(stdout, flag) {
				t.Errorf("%v: help lacks %s", args, flag)
			}
		}
	}
}

func TestShortNamesWorkOnTheCommandLine(t *testing.T) {
	var (
		mu    sync.Mutex
		token string
		count int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		token = r.Header.Get("X-Token")
		count++
	}))
	defer srv.Close()
	dir := t.TempDir()
	report := filepath.Join(dir, "r.txt")

	code, stdout, stderr := runCLI(t, "--rem", srv.URL, "--spc", writeSpec(t), "--sta", "2xx", "--smn", "2", "--smx=3", "--imn", "-5", "--imx", "5",
		"--fmn", "1", "--fmx", "2", "--chr", "alnum", "--n", "3", "--c", "2", "--tmo", "2s", "--o", report, "--fmt", "json", "--inc",
		"--log", filepath.Join(dir, "run.log"), "--H", "X-Token: t", "--k", "--sd", "7")
	if code != exitOK {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if count != 3 || token != "t" {
		t.Errorf("%d requests with X-Token %q, want 3 with t", count, token)
	}
	if !strings.Contains(stdout, "3 requests each (seed 7)") {
		t.Errorf("stdout = %s", stdout)
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Errorf("--fmt was ignored, report is:\n%s", data)
	}
}

func TestHelpOfASubcommandKeepsItsSpellings(t *testing.T) {
	_, stdout, _ := runCLI(t, "completion", "bash", "--help")
	if !strings.Contains(stdout, "--no-descriptions") || !strings.Contains(stdout, "-h, --help") {
		t.Errorf("help = %s", stdout)
	}
}

func TestVersionFlag(t *testing.T) {
	code, stdout, stderr := runCLI(t, "--version")
	if code != exitOK || !strings.HasPrefix(stdout, "fuzzy version ") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestShellCompletion(t *testing.T) {
	t.Run("script", func(t *testing.T) {
		code, stdout, stderr := runCLI(t, "completion", "bash")
		if code != exitOK || !strings.Contains(stdout, "bash completion") {
			t.Errorf("exit %d, stderr %q", code, stderr)
		}
	})
	t.Run("charset values", func(t *testing.T) {
		_, stdout, _ := runCLI(t, "__complete", "--charset", "")
		if !strings.Contains(stdout, "unicode") || !strings.Contains(stdout, "alnum") {
			t.Errorf("stdout = %q", stdout)
		}
	})
	t.Run("format values", func(t *testing.T) {
		_, stdout, _ := runCLI(t, "__complete", "--format", "")
		for _, format := range []string{"markdown", "json", "txt", "csv", "xml", "html"} {
			if !strings.Contains(stdout, format) {
				t.Errorf("stdout lacks %s: %q", format, stdout)
			}
		}
	})
	t.Run("partial flag name", func(t *testing.T) {
		_, stdout, _ := runCLI(t, "__complete", "--c")
		if !strings.Contains(stdout, "--charset") || !strings.Contains(stdout, "--concurrency") {
			t.Errorf("stdout = %q", stdout)
		}
	})
}

func TestFinalLineShowsTheAbsoluteReportPath(t *testing.T) {
	spec := writeSpec(t)
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	dir := t.TempDir()
	t.Chdir(dir)

	cases := map[string]string{
		"relative": "r.md",
		"absolute": filepath.Join(dir, "r.md"),
	}
	for name, output := range cases {
		t.Run(name, func(t *testing.T) {
			code, stdout, stderr := runCLI(t, "--remote", srv.URL, "--spec", spec, "-n", "1", "-o", output)
			if code != exitOK {
				t.Fatalf("exit %d: %s", code, stderr)
			}
			if want := "Report: " + filepath.Join(dir, "r.md") + "\n"; !strings.HasSuffix(stdout, want) {
				t.Errorf("stdout = %q, want it to end with %q", stdout, want)
			}
		})
	}
}

func TestFormatFlagPicksTheReportFormat(t *testing.T) {
	spec := writeSpec(t)
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	cases := map[string]struct {
		args []string
		file string
		json bool
	}{
		"default":        {nil, "fuzzy-report.md", false},
		"markdown":       {[]string{"--format", "markdown"}, "fuzzy-report.md", false},
		"json":           {[]string{"--format", "json"}, "fuzzy-report.json", true},
		"old spelling":   {[]string{"-format", "json"}, "fuzzy-report.json", true},
		"explicit path":  {[]string{"--format=json", "-o", "custom.out"}, "custom.out", true},
		"path no format": {[]string{"-o", "custom.json"}, "custom.json", true},
		"flag over path": {[]string{"--format", "markdown", "-o", "custom.json"}, "custom.json", false},
		"unknown path":   {[]string{"-o", "custom.out"}, "custom.out", false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)

			args := append([]string{"--remote", srv.URL, "--spec", spec, "-n", "2"}, tc.args...)
			code, stdout, stderr := runCLI(t, args...)
			if code != exitOK {
				t.Fatalf("exit %d: %s", code, stderr)
			}
			if want := "Report: " + filepath.Join(dir, tc.file) + "\n"; !strings.HasSuffix(stdout, want) {
				t.Errorf("stdout = %q, want it to end with %q", stdout, want)
			}

			data, err := os.ReadFile(tc.file)
			if err != nil {
				t.Fatal(err)
			}
			if tc.json != json.Valid(data) {
				t.Errorf("JSON expected: %v, got:\n%s", tc.json, data)
			}
			if !tc.json && !strings.HasPrefix(string(data), "# Fuzzy report") {
				t.Errorf("not a Markdown report:\n%s", data)
			}
		})
	}
}

func TestBadFormatIsAUsageErrorAndWritesNothing(t *testing.T) {
	t.Chdir(t.TempDir())

	code, _, stderr := runCLI(t, "--remote", "http://localhost:1", "--spec", writeSpec(t), "--format", "yaml")
	if code != exitUsage || !strings.Contains(stderr, "--format") {
		t.Errorf("exit %d, stderr %q", code, stderr)
	}
	if entries, _ := os.ReadDir("."); len(entries) != 0 {
		t.Errorf("no report should be created, found %v", entries)
	}
}

func TestCancelledRunMarksTheJSONReportAsPartial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) { once.Do(cancel) }))
	defer srv.Close()
	output := filepath.Join(t.TempDir(), "r.json")

	args := []string{"--remote", srv.URL, "--spec", writeSpec(t), "-n", "1000000", "--format", "json", "-o", output}
	if code := run(ctx, args, io.Discard, io.Discard); code != exitOK {
		t.Fatalf("exit %d", code)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct{ Interrupted bool }
	if err := json.Unmarshal(data, &doc); err != nil || !doc.Interrupted {
		t.Errorf("interrupted = %v, err = %v:\n%s", doc.Interrupted, err, data)
	}
}

func TestOutputExtensionPicksTheFormat(t *testing.T) {
	spec := writeSpec(t)
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	for file, start := range map[string]string{
		"r.md":   "# Fuzzy report",
		"r.json": "{",
		"r.txt":  "Fuzzy report\n=====",
		"r.csv":  "operation,endpoint,status,",
		"r.xml":  "<?xml",
		"r.html": "<!doctype html>",
		"R.HTM":  "<!doctype html>",
	} {
		t.Run(file, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), file)
			code, _, stderr := runCLI(t, "--remote", srv.URL, "--spec", spec, "-n", "2", "-o", path)
			if code != exitOK {
				t.Fatalf("exit %d: %s", code, stderr)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(string(data), start) {
				t.Errorf("starts with %.40q, want %q", data, start)
			}
		})
	}
}

func TestLogFlagKeepsEveryRequest(t *testing.T) {
	spec := writeSpec(t)
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "run.log")

	// Twice, because a log appends where a report starts over.
	for range 2 {
		code, stdout, stderr := runCLI(t, "--remote", srv.URL, "--spec", spec, "-n", "3", "--status", "5xx", "-o", filepath.Join(dir, "r.csv"), "--log", logPath)
		if code != exitOK {
			t.Fatalf("exit %d: %s", code, stderr)
		}
		if !strings.Contains(stdout, "Log: "+logPath+"\n") {
			t.Errorf("stdout = %q", stdout)
		}
		if !strings.Contains(stdout, "Report: "+filepath.Join(dir, "r.csv")+"\n") {
			t.Errorf("stdout = %q", stdout)
		}
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	log := string(data)
	if got := strings.Count(log, "msg=\"run started\""); got != 2 {
		t.Errorf("%d runs started:\n%s", got, log)
	}
	if got := strings.Count(log, "msg=\"run finished\""); got != 2 {
		t.Errorf("%d runs finished:\n%s", got, log)
	}
	// --status 5xx keeps none of these in the report, and all of them in the log.
	if got := strings.Count(log, "status=404"); got < 6 {
		t.Errorf("%d requests logged:\n%s", got, log)
	}
	report, err := os.ReadFile(filepath.Join(dir, "r.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(report), "\n") != 1 {
		t.Errorf("the filter should leave only the header:\n%s", report)
	}
}

func TestLogFlagOldSpellingAndBadPath(t *testing.T) {
	spec := writeSpec(t)
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	dir := t.TempDir()

	report := filepath.Join(dir, "r.md")
	if err := os.WriteFile(report, []byte("old report"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := runCLI(t, "-remote", srv.URL, "-spec", spec, "-o", report, "-log", filepath.Join(dir, "no", "dir", "run.log"))
	if code != exitFailure || !strings.Contains(stderr, "open log") {
		t.Errorf("exit %d, stderr %q", code, stderr)
	}
	if data, _ := os.ReadFile(report); string(data) != "old report" {
		t.Errorf("a bad log path must not cost the previous report: %q", data)
	}

	path := filepath.Join(dir, "run.log")
	if code, _, stderr := runCLI(t, "-remote", srv.URL, "-spec", spec, "-n", "1", "-o", report, "-log", path); code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if _, err := os.Stat(path); err != nil {
		t.Error(err)
	}
}
