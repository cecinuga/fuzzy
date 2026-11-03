package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"fuzzy/internal/fuzzer"
)

// bigDictionary writes a wordlist just above the size that keeps it on disk.
func bigDictionary(t *testing.T) string {
	t.Helper()
	var words bytes.Buffer
	for i := 0; words.Len() <= 25<<20; i++ {
		fmt.Fprintf(&words, "payload-%d\n", i)
	}
	path := filepath.Join(t.TempDir(), "big.txt")
	if err := os.WriteFile(path, words.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestBigDictionaryIsReproducibleAtAnyConcurrency(t *testing.T) {
	dict := bigDictionary(t)
	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.json")
	os.WriteFile(spec, []byte(`{"openapi":"3.0.0","paths":{"/u":{"get":{"parameters":[
		{"name":"who","in":"query","required":true,"schema":{"type":"string"}},
		{"name":"short","in":"query","required":true,"schema":{"type":"string","maxLength":10}}
	]}}}}`), 0o600)

	sent := func(concurrency string) []string {
		var mu sync.Mutex
		var queries []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			defer mu.Unlock()
			queries = append(queries, r.URL.RawQuery)
		}))
		defer srv.Close()

		code, _, stderr := runCLI(t, "--remote", srv.URL, "--spec", spec, "--dict", dict, "--seed", "5",
			"-n", "300", "-c", concurrency, "-o", filepath.Join(dir, "r.md"))
		if code != exitOK {
			t.Fatalf("exit %d: %s", code, stderr)
		}
		slices.Sort(queries)
		return queries
	}

	one, many := sent("1"), sent("16")
	if len(one) != 300 {
		t.Fatalf("sent %d requests, want 300", len(one))
	}
	if !slices.Equal(one, many) {
		t.Error("a seed must reproduce the inputs of a big dictionary at any concurrency")
	}
	if !strings.Contains(one[0], "who=payload-") {
		t.Errorf("the dictionary wasn't used: %s", one[0])
	}
}

func TestListingWorksWithABigDictionary(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.json")
	os.WriteFile(spec, []byte(`{"openapi":"3.0.0","paths":{"/u":{"get":{}}}}`), 0o600)

	code, _, stderr := runCLI(t, "--remote", "http://127.0.0.1:1", "--spec", spec, "--dict", bigDictionary(t), "--list")
	if code != exitOK {
		t.Fatalf("exit %d: %s", code, stderr)
	}
}

func TestRetentionWarning(t *testing.T) {
	all := fuzzer.StatusFilter{}
	some, err := fuzzer.ParseStatusFilter("5xx,err")
	if err != nil {
		t.Fatal(err)
	}

	if got := retentionWarning(retentionWarningResults-1, all); got != "" {
		t.Errorf("a small run needs no warning: %q", got)
	}
	if got := retentionWarning(retentionWarningResults, some); got != "" {
		t.Errorf("--status already limits the results: %q", got)
	}
	got := retentionWarning(12_000_000, all)
	for _, want := range []string{"12000000", "12.0 GB", "--status", "-n"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q lacks %q", got, want)
		}
	}
}
