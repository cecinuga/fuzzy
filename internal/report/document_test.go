package report

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"fuzzy/internal/fuzzer"
	"fuzzy/internal/openapi"
)

func writeString(t *testing.T, format Format, r Report) string {
	t.Helper()
	var b strings.Builder
	if err := Write(&b, format, r); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// leaves calls visit with every string and number in a decoded JSON value.
func leaves(value any, visit func(string)) {
	switch v := value.(type) {
	case map[string]any:
		for _, item := range v {
			leaves(item, visit)
		}
	case []any:
		for _, item := range v {
			leaves(item, visit)
		}
	case string:
		visit(v)
	case float64:
		visit(fmt.Sprint(v))
	}
}

func TestBothFormatsCarryTheSameInformation(t *testing.T) {
	r := runReport(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(404) }, "")
	markdown := writeString(t, FormatMarkdown, r)

	var decoded any
	if err := json.Unmarshal([]byte(writeString(t, FormatJSON, r)), &decoded); err != nil {
		t.Fatal(err)
	}
	checked := 0
	leaves(decoded, func(value string) {
		checked++
		if !strings.Contains(markdown, escape(value)) {
			t.Errorf("JSON has %q, the Markdown report doesn't", value)
		}
	})
	if checked < 50 {
		t.Errorf("only %d values compared: the walk is not reaching the document", checked)
	}
}

// The schema is a promise to whoever parses the report: changing this document
// breaks them, so it must be a deliberate choice.
func TestJSONSchema(t *testing.T) {
	filter, err := fuzzer.ParseStatusFilter("5xx,err")
	if err != nil {
		t.Fatal(err)
	}
	op := &openapi.Operation{
		Method: "GET", Path: "/user/{id}", BasePath: "/v1", Summary: "Fetch a user", OperationID: "getUser",
		Parameters: []*openapi.Parameter{{
			Name: "id", In: "path", Required: true,
			Schema: &openapi.Schema{Type: "integer", Format: "int32", MaxLength: new(int)},
		}},
	}
	got := writeString(t, FormatJSON, Report{
		SpecTitle: "Demo", SpecVersion: "2", Remote: "http://target", Spec: "spec.json", Dictionary: "values.txt",
		Filter: filter, Bounds: []string{"int: 0..9"},
		Seed:    18446744073709551557, // doesn't fit a double
		Started: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), Duration: 1500 * time.Millisecond,
		Interrupted: true, Warnings: []string{"odd pattern"},
		Outcomes: []fuzzer.Outcome{{
			Target: &fuzzer.Target{Op: op},
			Sent:   4,
			Counts: map[int]int{fuzzer.TransportError: 1, 404: 2, 500: 1},
			Results: []fuzzer.Result{
				{Endpoint: "GET /v1/user/1", Status: 500, Message: "boom", Input: "id=1"},
				{Endpoint: "GET /v1/user/2", Status: fuzzer.TransportError, Message: "timeout", Input: "id=2"},
			},
		}},
		Skipped: []fuzzer.Skipped{{Op: &openapi.Operation{Method: "POST", Path: "/legacy"}, Reason: "XML only"}},
	})

	want := `{
  "interrupted": true,
  "overview": {
    "target": "http://target",
    "spec": {
      "path": "spec.json",
      "title": "Demo",
      "version": "2"
    },
    "dictionary": "values.txt",
    "randomBounds": [
      "int: 0..9"
    ],
    "statusFilter": "5xx,err",
    "seed": "18446744073709551557",
    "started": "2026-01-02T03:04:05Z",
    "duration": "1.5s",
    "requestsSent": 4,
    "resultsReported": 2
  },
  "summary": [
    {
      "endpoint": "GET /user/{id}",
      "sent": 4,
      "reported": 2,
      "statusCodes": {
        "404": 2,
        "500": 1,
        "ERR": 1
      }
    }
  ],
  "endpoints": [
    {
      "method": "GET",
      "path": "/user/{id}",
      "fullPath": "/v1/user/{id}",
      "summary": "Fetch a user",
      "operationId": "getUser",
      "parameters": [
        {
          "name": "id",
          "in": "path",
          "type": "integer (int32)",
          "required": true,
          "constraints": "max length 0"
        }
      ],
      "results": {
        "sent": 4,
        "reported": 2,
        "filter": "5xx,err",
        "statusCodes": {
          "404": 2,
          "500": 1,
          "ERR": 1
        },
        "items": [
          {
            "endpoint": "GET /v1/user/1",
            "status": 500,
            "message": "boom",
            "input": "id=1"
          },
          {
            "endpoint": "GET /v1/user/2",
            "status": null,
            "message": "timeout",
            "input": "id=2"
          }
        ]
      }
    }
  ],
  "skipped": [
    {
      "endpoint": "POST /legacy",
      "reason": "XML only"
    }
  ],
  "warnings": [
    "odd pattern"
  ]
}
`
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestEmptyJSONReportKeepsItsShape(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(writeString(t, FormatJSON, Report{})), &doc); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"summary", "endpoints", "skipped", "warnings"} {
		if list, ok := doc[key].([]any); !ok || len(list) != 0 {
			t.Errorf("%s = %#v, want an empty list", key, doc[key])
		}
	}
	if bounds := doc["overview"].(map[string]any)["randomBounds"]; bounds == nil {
		t.Error("randomBounds must be an empty list, not null")
	}

	op := &openapi.Operation{Method: "GET", Path: "/none"}
	out := writeString(t, FormatJSON, Report{Outcomes: []fuzzer.Outcome{{Target: &fuzzer.Target{Op: op}}}})
	for _, want := range []string{`"parameters": []`, `"items": []`, `"statusCodes": {}`} {
		if !strings.Contains(out, want) {
			t.Errorf("an endpoint without results lacks %s:\n%s", want, out)
		}
	}
}

// Markdown has to neutralize a response. JSON only has to encode it, so the
// consumer gets back what the server really said.
func TestJSONKeepsHostileValuesIntact(t *testing.T) {
	hostile := "| x |\n## Fake\n<img src=x> `code` [a](http://evil) \x1b[31m \"quoted\" a&b"
	op := &openapi.Operation{Method: "GET", Path: "/a_b"}
	out := writeString(t, FormatJSON, Report{Outcomes: []fuzzer.Outcome{{
		Target:  &fuzzer.Target{Op: op},
		Results: []fuzzer.Result{{Status: 500, Message: hostile, Input: "q=\xff"}},
	}}})

	if strings.Contains(out, "\x1b") {
		t.Error("a raw control character reached the file")
	}
	if !strings.Contains(out, "<img src=x>") || !strings.Contains(out, "a&b") {
		t.Errorf("markup should stay readable:\n%s", out)
	}

	var doc struct {
		Endpoints []struct {
			Path    string
			Results struct {
				Items []struct{ Message, Input string }
			}
		}
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out)
	}
	item := doc.Endpoints[0].Results.Items[0]
	if item.Message != hostile {
		t.Errorf("message = %q, want it unchanged", item.Message)
	}
	if item.Input != "q=\uFFFD" {
		t.Errorf("input = %q, want invalid UTF-8 replaced", item.Input)
	}
	if doc.Endpoints[0].Path != "/a_b" {
		t.Errorf("path = %q: Markdown escaping leaked into JSON", doc.Endpoints[0].Path)
	}
}

func TestParseFormat(t *testing.T) {
	for name, want := range map[string]Format{"markdown": FormatMarkdown, "json": FormatJSON} {
		if got, err := ParseFormat(name); err != nil || got != want {
			t.Errorf("ParseFormat(%q) = %q, %v", name, got, err)
		}
	}
	for _, name := range []string{"", "md", "JSON", "xml"} {
		if _, err := ParseFormat(name); err == nil {
			t.Errorf("ParseFormat(%q) must fail", name)
		}
	}
	if err := Write(&strings.Builder{}, Format("xml"), Report{}); err == nil {
		t.Error("Write must refuse an unknown format")
	}
}
