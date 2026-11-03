package config

import (
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
)

func TestParseDefaults(t *testing.T) {
	cfg, err := Parse([]string{"-remote", "http://localhost:8080", "-spec", "api.json"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	b := cfg.Bounds
	if b.StringMax != 10 || b.IntMax != 1000 || b.FloatMax != 1000.0 {
		t.Errorf("default bounds = %+v, want string 10, int 1000, float 1000.0", b)
	}
	if cfg.Requests != 0 || cfg.Concurrency < 1 || cfg.Output != "fuzzy-report.md" || cfg.Seed == 0 {
		t.Errorf("config = %+v", cfg)
	}
	if !cfg.Filter.Match(204) || !cfg.Filter.Match(500) {
		t.Error("by default every status is reported")
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
			cfg, err := Parse([]string{"-remote", raw, "-spec", "s.json"}, io.Discard)
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
	cfg, err := Parse([]string{
		"-remote", "https://api.example.com/v1", "-spec", "s.json", "-dict", "d.txt", "-status", "200,5xx",
		"-string-min", "2", "-string-max", "4", "-int-min", "-5", "-int-max", "5", "-float-min", "0.5", "-float-max", "1.5",
		"-charset", "alnum", "-n", "7", "-c", "3", "-timeout", "2s", "-o", "out.md", "-k", "-seed", "9",
		"-H", "Authorization: Bearer abc:def", "-H", "X-A: 1", "-H", "X-A: 2",
	}, io.Discard)
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
	if cfg.Filter.Match(404) || !cfg.Filter.Match(503) {
		t.Error("status filter not applied")
	}
}

func TestParseRejectsBadInput(t *testing.T) {
	base := []string{"-remote", "http://x.io", "-spec", "s.json"}
	cases := map[string]struct {
		args []string
		want string
	}{
		"no remote":     {[]string{"-spec", "s.json"}, "-remote is required"},
		"no spec":       {[]string{"-remote", "http://x.io"}, "-spec is required"},
		"ftp":           {[]string{"-remote", "ftp://x.io", "-spec", "s"}, "-remote"},
		"no host":       {[]string{"-remote", "http://", "-spec", "s"}, "-remote"},
		"only a port":   {[]string{"-remote", ":8080", "-spec", "s"}, "-remote"},
		"port too big":  {[]string{"-remote", "x.io:70000", "-spec", "s"}, "invalid port"},
		"port zero":     {[]string{"-remote", "x.io:0", "-spec", "s"}, "invalid port"},
		"port not num":  {[]string{"-remote", "x.io:abc", "-spec", "s"}, "-remote"},
		"space in host": {[]string{"-remote", "x y.io", "-spec", "s"}, "-remote"},
		"unknown flag":  {append(base, "-nope"), "not defined"},
		"stray arg":     {append(base, "extra"), "unexpected argument"},
		"bad status":    {append(base, "-status", "abc"), "-status"},
		"bad strings":   {append(base, "-string-min", "9", "-string-max", "2"), "string minimum"},
		"negative len":  {append(base, "-string-min", "-1"), "negative"},
		"bad ints":      {append(base, "-int-min", "9", "-int-max", "2"), "int minimum"},
		"bad floats":    {append(base, "-float-min", "9", "-float-max", "2"), "float minimum"},
		"bad charset":   {append(base, "-charset", "x"), "charset"},
		"negative n":    {append(base, "-n", "-1"), "-n"},
		"zero workers":  {append(base, "-c", "0"), "-c"},
		"zero timeout":  {append(base, "-timeout", "0s"), "-timeout"},
		"bad header":    {append(base, "-H", "nocolon"), "Name: value"},
		"empty header":  {append(base, "-H", ": v"), "Name: value"},
		"non-numeric n": {append(base, "-n", "many"), "invalid value"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse(tc.args, io.Discard)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestHelp(t *testing.T) {
	var out strings.Builder
	_, err := Parse([]string{"-h"}, &out)
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("err = %v", err)
	}
	for _, name := range []string{"-remote", "-spec", "-status", "-dict", "-string-max", "-int-max", "-float-max"} {
		if !strings.Contains(out.String(), name) {
			t.Errorf("help lacks %s", name)
		}
	}
}
