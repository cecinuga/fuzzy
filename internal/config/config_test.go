package config

import (
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"fuzzy/internal/report"
)

// parse runs args through the real flag definitions and their validation.
func parse(args ...string) (*Config, error) {
	var flags Flags
	fs := pflag.NewFlagSet("fuzzy", pflag.ContinueOnError)
	fs.Usage = func() {}
	flags.Register(fs)
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return flags.Config()
}

func TestParseDefaults(t *testing.T) {
	cfg, err := parse("--remote", "http://localhost:8080", "--spec", "api.json")
	if err != nil {
		t.Fatal(err)
	}
	b := cfg.Bounds
	if b.StringMin != 1 || b.StringMax != 10 || b.IntMin != 0 || b.IntMax != 1000 || b.FloatMin != 0 || b.FloatMax != 1000.0 || b.Charset != "unicode" {
		t.Errorf("default bounds = %+v, want string 1..10, int 0..1000, float 0..1000.0, unicode", b)
	}
	if cfg.Requests != 0 || cfg.Concurrency != 20 || cfg.Timeout.Seconds() != 10 || cfg.Output != "fuzzy-report.md" || cfg.Seed == 0 {
		t.Errorf("config = %+v", cfg)
	}
	if !cfg.Filter.Match(204) || !cfg.Filter.Match(500) {
		t.Error("by default every status is reported")
	}
	if cfg.Format != report.FormatMarkdown {
		t.Errorf("default format = %q, want markdown", cfg.Format)
	}
}

func TestReportFormatAndOutput(t *testing.T) {
	cases := map[string]struct {
		args   []string
		format report.Format
		output string
	}{
		"default":           {nil, report.FormatMarkdown, "fuzzy-report.md"},
		"markdown":          {[]string{"--format", "markdown"}, report.FormatMarkdown, "fuzzy-report.md"},
		"json":              {[]string{"--format", "json"}, report.FormatJSON, "fuzzy-report.json"},
		"json to a path":    {[]string{"--format", "json", "-o", "out.txt"}, report.FormatJSON, "out.txt"},
		"path keeps format": {[]string{"-o", "out.json"}, report.FormatMarkdown, "out.json"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cfg, err := parse(append([]string{"--remote", "http://x.io", "--spec", "s.json"}, tc.args...)...)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Format != tc.format || cfg.Output != tc.output {
				t.Errorf("format %q, output %q, want %q, %q", cfg.Format, cfg.Output, tc.format, tc.output)
			}
		})
	}
}

func TestParseRemoteForms(t *testing.T) {
	cases := map[string]string{
		// Full URLs are kept as written.
		"http://localhost:8080":      "http://localhost:8080",
		"https://api.example.com/v3": "https://api.example.com/v3",
		"http://api.example.com":     "http://api.example.com",
		"https://localhost:8443":     "https://localhost:8443",
		// FQDN.
		"api.example.com":     "https://api.example.com",
		"api.example.com.":    "https://api.example.com.",
		"api.example.com/v3":  "https://api.example.com/v3",
		"api.example.com:443": "https://api.example.com:443",
		// host:port.
		"localhost:8080":       "http://localhost:8080",
		"127.0.0.1:8080":       "http://127.0.0.1:8080",
		"[::1]:8080":           "http://[::1]:8080",
		"api.example.com:8080": "http://api.example.com:8080",
		"localhost:8080/v1":    "http://localhost:8080/v1",
	}
	for raw, want := range cases {
		t.Run(raw, func(t *testing.T) {
			cfg, err := parse("--remote", raw, "--spec", "s.json")
			if err != nil {
				t.Fatal(err)
			}
			if got := cfg.Remote.String(); got != want {
				t.Errorf("remote = %q, want %q", got, want)
			}
		})
	}
}

func TestParseAllFlags(t *testing.T) {
	long := []string{
		"--remote", "https://api.example.com/v1", "--spec", "s.json", "--dict", "d.txt", "--status", "200,5xx",
		"--string-min", "2", "--string-max", "4", "--int-min", "-5", "--int-max", "5", "--float-min", "0.5", "--float-max", "1.5",
		"--charset", "alnum", "--requests", "7", "--concurrency", "3", "--timeout", "2s", "--output", "out.md", "--insecure", "--seed", "9",
		"--header", "Authorization: Bearer abc:def", "--header", "X-A: 1", "--header", "X-A: 2",
		"--header", "Accept: text/html, application/json",
	}
	short := []string{
		"--remote", "https://api.example.com/v1", "--spec", "s.json", "--dict", "d.txt", "--status", "200,5xx",
		"--string-min", "2", "--string-max", "4", "--int-min", "-5", "--int-max", "5", "--float-min", "0.5", "--float-max", "1.5",
		"--charset", "alnum", "-n", "7", "-c", "3", "--timeout", "2s", "-o", "out.md", "-k", "--seed", "9",
		"-H", "Authorization: Bearer abc:def", "-H", "X-A: 1", "-H", "X-A: 2",
		"-H", "Accept: text/html, application/json",
	}
	for name, args := range map[string][]string{"long": long, "short": short} {
		t.Run(name, func(t *testing.T) {
			cfg, err := parse(args...)
			if err != nil {
				t.Fatal(err)
			}
			if b := cfg.Bounds; b.StringMin != 2 || b.StringMax != 4 || b.IntMin != -5 || b.IntMax != 5 || b.FloatMin != 0.5 || b.FloatMax != 1.5 || b.Charset != "alnum" {
				t.Errorf("bounds = %+v", b)
			}
			if cfg.Requests != 7 || cfg.Concurrency != 3 || cfg.Timeout.Seconds() != 2 || cfg.Output != "out.md" || !cfg.Insecure || cfg.Seed != 9 || cfg.DictPath != "d.txt" {
				t.Errorf("config = %+v", cfg)
			}
			if got := cfg.Headers.Get("Authorization"); got != "Bearer abc:def" {
				t.Errorf("Authorization = %q", got)
			}
			if got := cfg.Headers.Values("X-A"); len(got) != 2 {
				t.Errorf("X-A = %v", got)
			}
			if got := cfg.Headers.Values("Accept"); len(got) != 1 || got[0] != "text/html, application/json" {
				t.Errorf("a comma must not split a header value: Accept = %q", got)
			}
			if cfg.Filter.Match(404) || !cfg.Filter.Match(503) {
				t.Error("status filter not applied")
			}
		})
	}
}

func TestParseRejectsBadInput(t *testing.T) {
	base := []string{"--remote", "http://x.io", "--spec", "s.json"}
	with := func(args ...string) []string { return append(append([]string{}, base...), args...) }
	cases := map[string]struct {
		args []string
		want string
	}{
		"empty remote":  {[]string{"--remote", "", "--spec", "s.json"}, "--remote"},
		"empty spec":    {[]string{"--remote", "http://x.io", "--spec", ""}, "--spec"},
		"ftp":           {[]string{"--remote", "ftp://x.io", "--spec", "s"}, "--remote"},
		"no host":       {[]string{"--remote", "http://", "--spec", "s"}, "--remote"},
		"only a port":   {[]string{"--remote", ":8080", "--spec", "s"}, "--remote"},
		"port too big":  {[]string{"--remote", "x.io:70000", "--spec", "s"}, "invalid port"},
		"port zero":     {[]string{"--remote", "x.io:0", "--spec", "s"}, "invalid port"},
		"port not num":  {[]string{"--remote", "x.io:abc", "--spec", "s"}, "--remote"},
		"space in host": {[]string{"--remote", "x y.io", "--spec", "s"}, "--remote"},
		"unknown flag":  {with("--nope"), "unknown flag"},
		"bad status":    {with("--status", "abc"), "--status"},
		"bad strings":   {with("--string-min", "9", "--string-max", "2"), "string minimum"},
		"negative len":  {with("--string-min", "-1"), "negative"},
		"bad ints":      {with("--int-min", "9", "--int-max", "2"), "int minimum"},
		"bad floats":    {with("--float-min", "9", "--float-max", "2"), "float minimum"},
		"bad charset":   {with("--charset", "x"), "charset"},
		"negative n":    {with("-n", "-1"), "--requests"},
		"zero workers":  {with("-c", "0"), "--concurrency"},
		"zero timeout":  {with("--timeout", "0s"), "--timeout"},
		"bad format":    {with("--format", "xml"), "--format"},
		"bad header":    {with("-H", "nocolon"), "Name: value"},
		"empty header":  {with("-H", ": v"), "Name: value"},
		"non-numeric n": {with("-n", "many"), "invalid argument"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := parse(tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}
