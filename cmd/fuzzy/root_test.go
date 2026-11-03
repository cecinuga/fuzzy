package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
			"--remote", "--spec", "--status", "--dict", "--string-min", "--string-max", "--int-min", "--int-max", "--float-min", "--float-max", "--charset",
			"-n, --requests", "-c, --concurrency", "--timeout", "-o, --output", "-H, --header", "-k, --insecure", "--seed",
		} {
			if !strings.Contains(stdout, flag) {
				t.Errorf("%v: help lacks %s", args, flag)
			}
		}
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
	t.Run("partial flag name", func(t *testing.T) {
		_, stdout, _ := runCLI(t, "__complete", "--c")
		if !strings.Contains(stdout, "--charset") || !strings.Contains(stdout, "--concurrency") {
			t.Errorf("stdout = %q", stdout)
		}
	})
}
