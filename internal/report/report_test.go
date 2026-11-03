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

// runReport fuzzes the test spec against handler and returns the report's input.
func runReport(t *testing.T, handler http.HandlerFunc, status string) Report {
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

	return Report{
		SpecTitle: doc.Info.Title, SpecVersion: doc.Info.Version,
		Remote: "http://target", Spec: "spec.json", Filter: filter, Seed: 77,
		Bounds:  generator.Bounds{StringMin: 1, StringMax: 2, IntMin: -3, IntMax: 4, FloatMin: 0.5, FloatMax: 1.5, Charset: generator.CharsetAlnum},
		Started: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), Duration: 1500 * time.Millisecond,
		Outcomes: outcomes, Skipped: skipped, Warnings: []string{"odd | pattern"},
	}
}

func render(t *testing.T, handler http.HandlerFunc, status string) string {
	t.Helper()
	var b strings.Builder
	if err := Write(&b, FormatMarkdown, runReport(t, handler, status)); err != nil {
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
		"| **Random bounds** | <ul><li>string: {min: 1, max: 2}</li><li>int: {min: -3, max: 4}</li><li>float: {min: 0.5, max: 1.5}</li></ul> |",
		"| **Charset** | alnum |",

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
		`| 404 | Not Found | {"path":{"id":`,
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
			if cells != 6 {
				t.Errorf("row has %d separators: %s", cells, line)
			}
		}
	}
}

func TestInterruptedAndSkippedAreVisible(t *testing.T) {
	var b strings.Builder
	op := &openapi.Operation{Method: "POST", Path: "/legacy"}
	err := Write(&b, FormatMarkdown, Report{Interrupted: true, Skipped: []fuzzer.Skipped{{Op: op, Reason: "no supported media type"}}})
	if err != nil {
		t.Fatal(err)
	}
	mustContain(t, b.String(), "interrupted", "## Skipped endpoints", "| POST /legacy | no supported media type |")
}

func TestWriteReportsWriterErrors(t *testing.T) {
	for _, format := range []Format{FormatMarkdown, FormatJSON} {
		if err := Write(failingWriter{}, format, Report{}); err == nil {
			t.Errorf("%s: a failed write must not pass silently", format)
		}
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, http.ErrHandlerTimeout }

func TestReportShowsInferredTypes(t *testing.T) {
	cases := map[string]string{
		`{"enum":[1,2]}`:                "integer",
		`{"format":"int64"}`:            "integer (int64)",
		`{"minimum":1}`:                 "number",
		`{"minItems":1}`:                "array of any",
		`{"items":{"format":"double"}}`: "array of number (double)",
		`{"required":["a"]}`:            "object",
		`{}`:                            "string",
	}
	for schema, want := range cases {
		doc, err := openapi.Parse([]byte(`{"openapi":"3.0.0","paths":{"/a":{"get":{"parameters":[{"name":"x","in":"query","schema":` + schema + `}]}}}}`))
		if err != nil {
			t.Fatal(err)
		}
		if got := typeName(doc.Operations[0].Parameters[0].Schema, 0); got != want {
			t.Errorf("%s: %q, want %q", schema, got, want)
		}
	}
}
