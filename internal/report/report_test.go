package report

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"fuzzy/internal/fuzzer"
	"fuzzy/internal/generator"
	"fuzzy/internal/openapi"
)

const spec = `{"openapi":"3.0.0","info":{"title":"Demo","version":"2"},"paths":{
	"/user/{id}":{"get":{"operationId":"getUser","summary":"Fetch | a user","parameters":[
		{"name":"id","in":"path","required":true,"schema":{"type":"integer","format":"int32","minimum":1,"maximum":9}},
		{"name":"tag","in":"query","schema":{"type":"array","items":{"type":"string","enum":["a","b"]},"maxItems":2}}
	]}},
	"/user":{"post":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/User"}}}}}},
	"/none":{"get":{}}
},"components":{"schemas":{"User":{"type":"object","required":["name"],"properties":{
	"name":{"type":"string","minLength":2,"pattern":"^a"},
	"secret":{"type":"string","readOnly":true},
	"friend":{"$ref":"#/components/schemas/User"},
	"address":{"type":"object","required":["city"],"properties":{"city":{"type":"string","maxLength":9}}},
	"pet":{"oneOf":[{"type":"integer"},{"type":"boolean"}],"nullable":true},
	"score":{"type":"number","minimum":0,"exclusiveMinimum":true,"multipleOf":0.5}
}}}}}`

func render(t *testing.T, handler http.HandlerFunc, status string) string {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	doc, err := openapi.Parse([]byte(spec))
	if err != nil {
		t.Fatal(err)
	}
	remote, _ := url.Parse(srv.URL)
	targets, skipped := fuzzer.Compile(doc, generator.New(generator.DefaultBounds(), nil, 1), remote)
	filter, err := fuzzer.ParseStatusFilter(status)
	if err != nil {
		t.Fatal(err)
	}
	outcomes := fuzzer.Run(context.Background(), fuzzer.Options{Requests: 5, Concurrency: 2, Timeout: time.Second, Filter: filter}, targets)

	var b strings.Builder
	err = Write(&b, Report{
		SpecTitle: doc.Info.Title, SpecVersion: doc.Info.Version,
		Remote: "http://target", Spec: "spec.json", Filter: filter, Bounds: "bounds-text", Seed: 77,
		Started: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), Duration: 1500 * time.Millisecond,
		Outcomes: outcomes, Skipped: skipped, Warnings: []string{"odd | pattern"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func mustContain(t *testing.T, doc string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(doc, want) {
			t.Errorf("report lacks %q\n---\n%s", want, doc)
		}
	}
}

func TestReportDescribesEveryEndpoint(t *testing.T) {
	doc := render(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(404) }, "")

	mustContain(t, doc,
		"# Fuzzy report",
		"| **Target** | http://target |",
		"spec.json (Demo 2)",
		"| **Seed** | 77 |",
		"| **Duration** | 1.5s |",
		"| **Requests sent** | 15 |",
		"| **Dictionary** | none (random values) |",
		"bounds-text",

		"## GET /user/{id}",
		"- **HTTP verb:** GET",
		"- **Operation ID:** getUser",
		`Fetch \| a user`,
		"| id | path | integer (int32) | yes | min 1; max 9 |",
		`| tag | query | array of string | no | max items 2 |`,

		"## POST /user",
		"| (body) | application/json | object | yes |",
		`| name | application/json | string | yes | min length 2; pattern: ^a |`,
		`| score | application/json | number | no | exclusive min 0; multiple of 0.5 |`,
		"oneOf(integer, boolean) (nullable)",
		"| address.city | application/json | string | yes | max length 9 |", // nested fields are listed
		"| friend | application/json | object | no |  |",                    // recursion stops

		"## GET /none",
		"None.",

		"| Endpoint | Status | Message | Input |",
		"| 404 | Not Found | id=",
		"404 ×5",
		"odd \\| pattern",
	)
	if strings.Contains(doc, "secret") {
		t.Error("readOnly properties are not request parameters")
	}
	if strings.Count(doc, "## POST /user") != 1 {
		t.Error("recursive schema must not repeat the section")
	}
}

func TestResultsFollowTheStatusFilter(t *testing.T) {
	doc := render(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
	}, "5xx")

	mustContain(t, doc,
		"| **Status filter** | 5xx |",
		"| **Results reported** | 5 |",
		"| POST /user | 500 | Internal Server Error |",
		"5 sent, 0 reported (filter: 5xx). Status codes: 200 ×5.", // counts survive the filter
	)
	if strings.Contains(doc, "| 200 | OK |") {
		t.Error("200 results must be filtered out")
	}
}

func TestHostileResponseCannotBreakTheDocument(t *testing.T) {
	doc := render(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("| x | y |\n## Fake heading\n<img src=x onerror=alert(1)> `code` [a](http://evil) \x1b[31m"))
	}, "")

	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(line, "## Fake") {
			t.Error("a newline in a response escaped its table row")
		}
	}
	if strings.Contains(doc, "<img") {
		t.Error("raw HTML from a response reached the report")
	}
	if strings.Contains(doc, "[a](") {
		t.Error("a link from a response reached the report")
	}
	if strings.Contains(doc, "\x1b") {
		t.Error("control characters from a response reached the report")
	}
	mustContain(t, doc, `\| x \| y \|`, "&lt;img")

	// Every table row of a section must keep its column count.
	for _, line := range strings.Split(doc, "\n") {
		if strings.Contains(line, "&lt;img") {
			cells := strings.Count(strings.ReplaceAll(line, `\|`, ""), "|")
			if cells != 5 {
				t.Errorf("row has %d separators: %s", cells, line)
			}
		}
	}
}

func TestInterruptedAndSkippedAreVisible(t *testing.T) {
	var b strings.Builder
	op := &openapi.Operation{Method: "POST", Path: "/legacy"}
	err := Write(&b, Report{Interrupted: true, Skipped: []fuzzer.Skipped{{Op: op, Reason: "no supported media type"}}})
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, b.String(), "interrupted", "## Skipped endpoints", "| POST /legacy | no supported media type |")
}

func TestWriteReportsWriterErrors(t *testing.T) {
	if err := Write(failingWriter{}, Report{}); err == nil {
		t.Error("a failed write must not pass silently")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, http.ErrHandlerTimeout }
