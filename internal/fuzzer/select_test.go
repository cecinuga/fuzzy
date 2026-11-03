package fuzzer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fuzzy/internal/generator"
	"fuzzy/internal/openapi"
)

func TestParseSelectionRefusesBadSelectors(t *testing.T) {
	cases := map[string]string{
		"":              "empty",
		"   ":           "empty",
		"FETCH":         "want a method",
		"users":         "want a method",
		"GET,":          "want a method",
		"GET,FETCH /a":  "want a method",
		"GET POST":      "after the methods",
		"GET users":     "after the methods",
		"tag:":          "needs a name",
		"GET tag:  ":    "needs a name",
		"/users*":       "whole path segment",
		"/a/**b/c":      "whole path segment",
		"GET /a/*.json": "whole path segment",
	}
	for raw, want := range cases {
		for flag, parse := range map[string]func() (Selection, error){
			"--include": func() (Selection, error) { return ParseSelection([]string{raw}, nil) },
			"--exclude": func() (Selection, error) { return ParseSelection(nil, []string{raw}) },
		} {
			_, err := parse()
			if err == nil || !strings.Contains(err.Error(), want) || !strings.HasPrefix(err.Error(), flag) {
				t.Errorf("%s %q: error = %v, want %s and %q", flag, raw, err, flag, want)
			}
		}
	}
}

func TestSelectorMatches(t *testing.T) {
	cases := []struct {
		selector string
		method   string
		path     string
		tags     []string
		want     bool
	}{
		{"GET", "GET", "/a", nil, true},
		{"get", "GET", "/a", nil, true},
		{"GET", "POST", "/a", nil, false},
		{"GET,POST", "POST", "/a", nil, true},
		{"GET,post /a", "POST", "/a", nil, true},
		{"GET  /a", "GET", "/a", nil, true},

		{"/a", "GET", "/a", nil, true},
		{"/a", "GET", "/a/b", nil, false},
		{"/a", "GET", "/A", nil, false},
		{"/a/", "GET", "/a", nil, false},
		{"/a/*", "GET", "/a/b", nil, true},
		{"/a/*", "GET", "/a/{id}", nil, true},
		{"/a/*", "GET", "/a", nil, false},
		{"/a/*", "GET", "/a/", nil, false},
		{"/a/*", "GET", "/a/b/c", nil, false},
		{"/*/c", "GET", "/a/c", nil, true},

		{"/a/**", "GET", "/a", nil, true},
		{"/a/**", "GET", "/a/b/c/d", nil, true},
		{"/a/**", "GET", "/ab", nil, false},
		{"/**", "GET", "/", nil, true},
		{"/**", "GET", "/a/b", nil, true},
		{"/a/**/z", "GET", "/a/z", nil, true},
		{"/a/**/z", "GET", "/a/b/c/z", nil, true},
		{"/a/**/z", "GET", "/a/b/c", nil, false},
		{"/**/b/**", "GET", "/a/b/c", nil, true},

		{"/user/{id}", "GET", "/user/{id}", nil, true},
		{"/user/{id}", "GET", "/user/{name}", nil, false},
		{"GET /user/{id}", "GET", "/user/{id}", nil, true},
		{"GET /user/{id}", "PUT", "/user/{id}", nil, false},

		{"tag:billing", "GET", "/a", []string{"users", "billing"}, true},
		{"tag:billing", "GET", "/a", []string{"users"}, false},
		{"tag:billing", "GET", "/a", nil, false},
		{"tag:Billing", "GET", "/a", []string{"billing"}, false},
		{"tag:User Management", "GET", "/a", []string{"User Management"}, true},
		{"GET tag:User Management", "GET", "/a", []string{"User Management"}, true},
		{"POST tag:billing", "GET", "/a", []string{"billing"}, false},
	}
	for _, tc := range cases {
		sel, err := parseSelector(tc.selector)
		if err != nil {
			t.Errorf("%q: %v", tc.selector, err)
			continue
		}
		op := &openapi.Operation{Method: tc.method, Path: tc.path, Tags: tc.tags}
		if got := sel.matches(op); got != tc.want {
			t.Errorf("%q on %s %s %v = %v, want %v", tc.selector, tc.method, tc.path, tc.tags, got, tc.want)
		}
	}
}

const selectionSpec = `{"openapi":"3.0.0","paths":{
	"/a":{
		"get":{"tags":["reads"]},
		"post":{"requestBody":{"required":true,"content":{"application/protobuf":{"schema":{"type":"string"}}}}},
		"delete":{}
	},
	"/b":{"get":{"tags":["reads","slow"]}}
}}`

// names returns "METHOD /path" for what Apply kept, targets then skipped.
func names(targets []*Target, skipped []Skipped) (fuzzed, notFuzzed []string) {
	fuzzed, notFuzzed = []string{}, []string{}
	for _, target := range targets {
		fuzzed = append(fuzzed, target.Op.Method+" "+target.Op.Path)
	}
	for _, s := range skipped {
		notFuzzed = append(notFuzzed, s.Op.Method+" "+s.Op.Path)
	}
	return fuzzed, notFuzzed
}

func TestApply(t *testing.T) {
	cases := map[string]struct {
		include, exclude []string
		fuzzed, skipped  []string
	}{
		"no selection": {nil, nil, []string{"DELETE /a", "GET /a", "GET /b"}, []string{"POST /a"}},
		"include":      {[]string{"GET"}, nil, []string{"GET /a", "GET /b"}, []string{}},
		"includes add up": {
			[]string{"DELETE", "/b"}, nil, []string{"DELETE /a", "GET /b"}, []string{},
		},
		"exclude":         {nil, []string{"DELETE"}, []string{"GET /a", "GET /b"}, []string{"POST /a"}},
		"exclude wins":    {[]string{"/a"}, []string{"GET /a"}, []string{"DELETE /a"}, []string{"POST /a"}},
		"tag":             {[]string{"tag:slow"}, nil, []string{"GET /b"}, []string{}},
		"excluded skip":   {nil, []string{"POST /a"}, []string{"DELETE /a", "GET /a", "GET /b"}, []string{}},
		"everything out":  {nil, []string{"/**"}, []string{}, []string{}},
		"selected a skip": {[]string{"POST"}, nil, []string{}, []string{"POST /a"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			selection, err := ParseSelection(tc.include, tc.exclude)
			if err != nil {
				t.Fatal(err)
			}
			targets, skipped, err := selection.Apply(compileSpec(t, "http://x", selectionSpec))
			if err != nil {
				t.Fatal(err)
			}
			fuzzed, notFuzzed := names(targets, skipped)
			slices.Sort(fuzzed)
			if !slices.Equal(fuzzed, tc.fuzzed) || !slices.Equal(notFuzzed, tc.skipped) {
				t.Errorf("fuzzed %v, skipped %v; want %v and %v", fuzzed, notFuzzed, tc.fuzzed, tc.skipped)
			}
		})
	}
}

// A typo in an --exclude must never quietly fuzz what it was meant to protect.
func TestApplyRefusesASelectorMatchingNothing(t *testing.T) {
	cases := map[string]struct {
		include, exclude []string
		want             []string
	}{
		"exclude typo":      {nil, []string{"DELETE /aa"}, []string{`--exclude "DELETE /aa"`}},
		"include typo":      {[]string{"/nope"}, nil, []string{`--include "/nope"`}},
		"one of many":       {[]string{"GET", "/nope"}, []string{"DELETE"}, []string{`--include "/nope"`}},
		"all named":         {[]string{"/x"}, []string{"/y"}, []string{`--include "/x"`, `--exclude "/y"`}},
		"no such method":    {[]string{"PATCH"}, nil, []string{`--include "PATCH"`}},
		"no such tag":       {[]string{"tag:nope"}, nil, []string{`--include "tag:nope"`}},
		"verb and path off": {nil, []string{"PUT /a"}, []string{`--exclude "PUT /a"`}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			selection, err := ParseSelection(tc.include, tc.exclude)
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = selection.Apply(compileSpec(t, "http://x", selectionSpec))
			if err == nil {
				t.Fatal("want an error")
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q lacks %s", err, want)
				}
			}
		})
	}

	// An exclude only has to match the spec, not what the includes kept.
	selection, err := ParseSelection([]string{"GET"}, []string{"DELETE"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := selection.Apply(compileSpec(t, "http://x", selectionSpec)); err != nil {
		t.Errorf("an exclude matching an endpoint the includes left out: %v", err)
	}
}

// Leaving endpoints out must not change the inputs of those that stay: that is
// what lets a seed replay one failing endpoint of a bigger run.
func TestSelectionKeepsAnEndpointsInputs(t *testing.T) {
	const spec = `{"openapi":"3.0.0","paths":{
		"/a":{"get":{"parameters":[{"name":"w","in":"query","required":true,"schema":{"type":"string"}}]}},
		"/b":{"post":{"requestBody":{"required":true,"content":{"application/protobuf":{"schema":{"type":"string"}}}}}},
		"/c":{"get":{"parameters":[{"name":"w","in":"query","required":true,"schema":{"type":"string","maxLength":3}}]}},
		"/d":{"get":{"parameters":[
			{"name":"w","in":"query","required":true,"schema":{"type":"string"}},
			{"name":"n","in":"query","required":true,"schema":{"type":"integer"}}
		]}}
	}}`
	// With a dictionary, because compiling then consumes the seeded offsets.
	words := filepath.Join(t.TempDir(), "words.txt")
	if err := os.WriteFile(words, []byte("alice\nbob\ncarol\ndave\nerin\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dict, err := generator.LoadDictionary(words)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()
	remote, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	inputs := func(selection Selection) map[string][]string {
		doc, err := openapi.Parse([]byte(spec))
		if err != nil {
			t.Fatal(err)
		}
		targets, skipped := Compile(doc, generator.New(generator.DefaultBounds(), dict, 7), remote)
		targets, _, err = selection.Apply(targets, skipped)
		if err != nil {
			t.Fatal(err)
		}
		opts := options()
		opts.Seed = 7
		out := map[string][]string{}
		for _, outcome := range Run(context.Background(), opts, targets) {
			for _, result := range outcome.Results {
				out[outcome.Target.Op.Path] = append(out[outcome.Target.Op.Path], result.Input)
			}
		}
		return out
	}

	full := inputs(Selection{})
	if len(full) != 3 {
		t.Fatalf("full run reached %d endpoints: %v", len(full), full)
	}
	if slices.Equal(full["/a"], full["/d"]) {
		t.Fatal("endpoints share their inputs, so this test can't tell them apart")
	}

	for _, path := range []string{"/a", "/c", "/d"} {
		only, err := ParseSelection([]string{path}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := inputs(only)[path]; !slices.Equal(got, full[path]) {
			t.Errorf("%s alone got other inputs than in the full run:\n%v\n%v", path, got, full[path])
		}
	}
}
