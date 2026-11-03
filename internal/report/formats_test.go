package report

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"fuzzy/internal/fuzzer"
	"fuzzy/internal/openapi"
)

func TestFormatOfPath(t *testing.T) {
	for path, want := range map[string]Format{
		"a.md": FormatMarkdown, "a.markdown": FormatMarkdown, "a.json": FormatJSON, "a.txt": FormatText,
		"a.csv": FormatCSV, "a.xml": FormatXML, "a.html": FormatHTML, "a.htm": FormatHTML,
		"A.CSV": FormatCSV, "dir.json/a.tar.xml": FormatXML,
	} {
		if got, ok := FormatOfPath(path); !ok || got != want {
			t.Errorf("FormatOfPath(%q) = %q, %v", path, got, ok)
		}
	}
	for _, path := range []string{"", "a", "a.out", "a.yaml", "dir.json/a", ".", "a."} {
		if got, ok := FormatOfPath(path); ok {
			t.Errorf("FormatOfPath(%q) = %q, want none", path, got)
		}
	}
}

func TestFormatExtensionsLeadBackToTheFormat(t *testing.T) {
	for _, format := range Formats() {
		if got, ok := FormatOfPath("report" + format.Extension()); !ok || got != format {
			t.Errorf("%s: extension %s reads as %q", format, format.Extension(), got)
		}
		if _, err := ParseFormat(string(format)); err != nil {
			t.Errorf("%s: %v", format, err)
		}
	}
}

// Every format but CSV renders the whole document, so everything in JSON has to
// be there too, written the way each format writes text.
func TestReadableFormatsCarryTheSameInformation(t *testing.T) {
	handler := func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(time.Millisecond)
		w.WriteHeader(404)
	}
	r := runReport(t, handler, "")
	var decoded any
	if err := json.Unmarshal([]byte(writeString(t, FormatJSON, r)), &decoded); err != nil {
		t.Fatal(err)
	}

	forms := map[Format]func(string) string{
		FormatText: oneLine,
		FormatHTML: html.EscapeString,
		FormatXML:  xmlEscaped,
	}
	for format, written := range forms {
		t.Run(string(format), func(t *testing.T) {
			out := writeString(t, format, r)
			checked := 0
			leaves(decoded, func(value string) {
				checked++
				if !strings.Contains(out, written(value)) {
					t.Errorf("JSON has %q, the %s report doesn't", value, format)
				}
			})
			if checked < 50 {
				t.Errorf("only %d values compared", checked)
			}
		})
	}
}

func xmlEscaped(s string) string {
	var b strings.Builder
	if err := xml.EscapeText(&b, []byte(s)); err != nil {
		panic(err)
	}
	return b.String()
}

func TestEveryFormatMarksAPartialRun(t *testing.T) {
	r := runReport(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }, "")
	for format, want := range map[Format]string{
		FormatMarkdown: "these results are partial",
		FormatText:     "these results are partial",
		FormatHTML:     "these results are partial",
		FormatJSON:     `"interrupted": true`,
		FormatXML:      "<interrupted>true</interrupted>",
		FormatCSV:      ",true\n",
	} {
		r.Interrupted = false
		if strings.Contains(writeString(t, format, r), want) {
			t.Errorf("%s: a complete run contains %q", format, want)
		}
		r.Interrupted = true
		if !strings.Contains(writeString(t, format, r), want) {
			t.Errorf("%s: a partial run lacks %q", format, want)
		}
	}
}

func TestEveryFormatReportsWriterErrors(t *testing.T) {
	r := runReport(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }, "")
	for _, format := range Formats() {
		if err := Write(failingWriter{}, format, r); err == nil {
			t.Errorf("%s: want an error", format)
		}
	}
}

func TestEveryFormatHandlesAnEmptyReport(t *testing.T) {
	for _, format := range Formats() {
		if err := Write(io.Discard, format, Report{}); err != nil {
			t.Errorf("%s: %v", format, err)
		}
	}
}

// hostileReport has text from the server, and from the spec, trying to break out.
func hostileReport(message string) Report {
	op := &openapi.Operation{Method: "GET", Path: "/a</td><script>x</script>", Summary: "</p><img src=x>"}
	return Report{
		Warnings: []string{"<script>warning</script>"},
		Skipped:  []fuzzer.Skipped{{Op: op, Reason: "<script>reason</script>"}},
		Outcomes: []fuzzer.Outcome{{
			Target: &fuzzer.Target{Op: op}, Sent: 1, Counts: map[int]int{500: 1},
			Results: []fuzzer.Result{{Endpoint: "GET /a", Status: 500, Message: message, Input: "q=\xff" + message}},
		}},
	}
}

const hostileMessage = "<script>alert(1)</script></td>\n## Forged\nForged\n-----\n\x1b[31m\u202e&amp; \"q\" ]]>"

func TestHTMLNeutralizesHostileText(t *testing.T) {
	out := writeString(t, FormatHTML, hostileReport(hostileMessage))
	for _, banned := range []string{"<script", "<img", "\x1b", "\u202e"} {
		if strings.Contains(out, banned) {
			t.Errorf("found %q:\n%s", banned, out)
		}
	}
	if !strings.Contains(out, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Errorf("the message should stay readable:\n%s", out)
	}
	if !strings.Contains(out, `Content-Security-Policy`) {
		t.Error("the page lost its policy")
	}
	if !strings.Contains(out, `class="server-error"`) {
		t.Errorf("status has no style:\n%s", out)
	}
}

func TestHTMLStatusStyles(t *testing.T) {
	for status, want := range map[statusCode]string{0: "failed", 500: "server-error", 503: "server-error", 404: "client-error", 200: "success", 302: "success"} {
		if got := statusClass(status); got != want {
			t.Errorf("statusClass(%d) = %q, want %q", status, got, want)
		}
	}
}

func TestTextNeutralizesHostileText(t *testing.T) {
	out := writeString(t, FormatText, hostileReport(hostileMessage))
	for _, banned := range []string{"\x1b", "\u202e", "\nForged", "\n## Forged", "Forged\n"} {
		if strings.Contains(out, banned) {
			t.Errorf("found %q:\n%s", banned, out)
		}
	}
	if !strings.Contains(out, "<script>alert(1)</script>") {
		t.Errorf("plain text has no markup to escape:\n%s", out)
	}
}

func TestXMLKeepsHostileTextInsideItsElement(t *testing.T) {
	out := writeString(t, FormatXML, hostileReport(hostileMessage))

	var doc struct {
		Endpoints []struct {
			Path    string `xml:"path"`
			Results struct {
				Items []struct {
					Status  string `xml:"status"`
					Message string `xml:"message"`
					Input   string `xml:"input"`
				} `xml:"items>result"`
			} `xml:"results"`
		} `xml:"endpoints>endpoint"`
		Warnings []string `xml:"warnings>warning"`
	}
	if err := xml.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("not well-formed: %v\n%s", err, out)
	}
	if strings.Contains(out, "\x1b") {
		t.Error("a raw control character reached the file")
	}
	item := doc.Endpoints[0].Results.Items[0]
	if item.Status != "500" || !strings.Contains(item.Message, "<script>alert(1)</script></td>") || !strings.Contains(item.Message, "]]>") {
		t.Errorf("item = %+v", item)
	}
	if !strings.HasPrefix(item.Input, "q=\uFFFD") {
		t.Errorf("input = %q, want invalid UTF-8 replaced", item.Input)
	}
	if doc.Endpoints[0].Path != "/a</td><script>x</script>" || doc.Warnings[0] != "<script>warning</script>" {
		t.Errorf("doc = %+v", doc)
	}
}

func TestEmptyXMLReportKeepsItsShape(t *testing.T) {
	out := writeString(t, FormatXML, Report{})
	for _, want := range []string{"<summary></summary>", "<endpoints></endpoints>", "<skipped></skipped>", "<warnings></warnings>", "<dictionary></dictionary>"} {
		if !strings.Contains(out, want) {
			t.Errorf("lacks %s:\n%s", want, out)
		}
	}
	op := &openapi.Operation{Method: "GET", Path: "/none"}
	out = writeString(t, FormatXML, Report{Outcomes: []fuzzer.Outcome{{Target: &fuzzer.Target{Op: op}}}})
	for _, want := range []string{"<parameters></parameters>", "<items></items>", "<statusCodes></statusCodes>", "<count>0</count>"} {
		if !strings.Contains(out, want) {
			t.Errorf("an endpoint without results lacks %s:\n%s", want, out)
		}
	}
}

func TestXMLStatusCounts(t *testing.T) {
	op := &openapi.Operation{Method: "GET", Path: "/a"}
	out := writeString(t, FormatXML, Report{Outcomes: []fuzzer.Outcome{{
		Target:  &fuzzer.Target{Op: op},
		Counts:  map[int]int{500: 1, fuzzer.TransportError: 2, 404: 3},
		Results: []fuzzer.Result{{Status: fuzzer.TransportError, Message: "m"}},
	}}})
	for _, want := range []string{`<count status="ERR">2</count>`, `<count status="404">3</count>`, `<status>ERR</status>`} {
		if !strings.Contains(out, want) {
			t.Errorf("lacks %s:\n%s", want, out)
		}
	}
	if strings.Index(out, `status="ERR"`) > strings.Index(out, `status="404"`) || strings.Index(out, `status="404"`) > strings.Index(out, `status="500"`) {
		t.Errorf("counts are out of order:\n%s", out)
	}
}

func readCSV(t *testing.T, r Report) [][]string {
	t.Helper()
	rows, err := csv.NewReader(strings.NewReader(writeString(t, FormatCSV, r))).ReadAll()
	if err != nil {
		t.Fatalf("not valid CSV: %v", err)
	}
	return rows
}

func TestCSVRows(t *testing.T) {
	get := &openapi.Operation{Method: "GET", Path: "/user/{id}"}
	post := &openapi.Operation{Method: "POST", Path: "/user"}
	rows := readCSV(t, Report{
		Seed: 18446744073709551557,
		Outcomes: []fuzzer.Outcome{
			{Target: &fuzzer.Target{Op: get}, Sent: 3, Counts: map[int]int{404: 1, 500: 1, 0: 1}, Results: []fuzzer.Result{
				{Endpoint: "GET /user/1", Status: 500, Message: "a, \"b\"", Input: "id=1", Duration: 1500 * time.Microsecond},
				{Endpoint: "GET /user/2", Status: fuzzer.TransportError, Message: "timeout", Input: "id=2", Duration: time.Second},
			}},
			{Target: &fuzzer.Target{Op: post}, Sent: 1, Counts: map[int]int{404: 1}}, // nothing reported
		},
	})
	want := [][]string{
		{"operation", "endpoint", "status", "message", "input", "durationMs", "seed", "interrupted"},
		{"GET /user/{id}", "GET /user/1", "500", `a, "b"`, "id=1", "1.5", "18446744073709551557", "false"},
		{"GET /user/{id}", "GET /user/2", "ERR", "timeout", "id=2", "1000", "18446744073709551557", "false"},
	}
	if fmt.Sprint(rows) != fmt.Sprint(want) {
		t.Errorf("got  %q\nwant %q", rows, want)
	}
}

func TestCSVWithoutResultsStillHasItsHeader(t *testing.T) {
	if rows := readCSV(t, Report{}); len(rows) != 1 || rows[0][0] != "operation" {
		t.Errorf("rows = %q", rows)
	}
}

// A spreadsheet runs these as formulas, so a response must not be able to start one.
func TestCSVDefusesFormulas(t *testing.T) {
	for _, formula := range []string{"=1+1", "+1", "-1", "@SUM(A1)", "\t=1", "\r=1"} {
		op := &openapi.Operation{Method: "GET", Path: "/a"}
		rows := readCSV(t, Report{Outcomes: []fuzzer.Outcome{{
			Target:  &fuzzer.Target{Op: op},
			Results: []fuzzer.Result{{Endpoint: formula, Status: 500, Message: formula, Input: formula}},
		}}})
		for _, column := range []int{1, 3, 4} {
			if cell := rows[1][column]; !strings.HasPrefix(cell, "'") {
				t.Errorf("column %d = %q", column, cell)
			}
		}
	}
	if got := csvCell("plain = text"); got != "plain = text" {
		t.Errorf("csvCell = %q: only the start of a cell is dangerous", got)
	}
	if got := csvCell(""); got != "" {
		t.Errorf("csvCell(empty) = %q", got)
	}
	if got := csvCell("a\xff"); got != "a\uFFFD" {
		t.Errorf("csvCell = %q, want invalid UTF-8 replaced", got)
	}
}

func TestResponseTimeInReports(t *testing.T) {
	op := &openapi.Operation{Method: "GET", Path: "/a"}
	report := Report{Outcomes: []fuzzer.Outcome{{Target: &fuzzer.Target{Op: op}}}}
	for _, format := range []Format{FormatMarkdown, FormatText, FormatHTML} {
		if out := writeString(t, format, report); !strings.Contains(out, "no responses") {
			t.Errorf("%s: an endpoint without responses should say so:\n%s", format, out)
		}
	}

	r := runReport(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }, "")
	var doc struct {
		Summary []struct {
			ResponseTime responseTime `json:"responseTimeMs"`
		} `json:"summary"`
		Endpoints []struct {
			Results struct {
				ResponseTime responseTime `json:"responseTimeMs"`
				Items        []struct {
					DurationMs float64 `json:"durationMs"`
				} `json:"items"`
			} `json:"results"`
		} `json:"endpoints"`
	}
	if err := json.Unmarshal([]byte(writeString(t, FormatJSON, r)), &doc); err != nil {
		t.Fatal(err)
	}
	for i, row := range doc.Summary {
		times := doc.Endpoints[i].Results.ResponseTime
		if row.ResponseTime != times || times.Count != 5 || times.Min <= 0 || times.Min > times.Mean || times.Mean > times.Max || times.P50 > times.P99 {
			t.Errorf("endpoint %d: summary %+v, results %+v", i, row.ResponseTime, times)
		}
		for _, item := range doc.Endpoints[i].Results.Items {
			if item.DurationMs <= 0 {
				t.Errorf("endpoint %d: item took %v", i, item.DurationMs)
			}
		}
	}
}

func TestResponseTimeText(t *testing.T) {
	one := responseTime{Count: 1, Min: 1.5, Mean: 1.5, P50: 1.5, P95: 1.5, P99: 1.5, Max: 1.5}
	if got := one.text(); !strings.Contains(got, "max 1.5 ms (1 response)") {
		t.Errorf("text = %q", got)
	}
	two := responseTime{Count: 2, Mean: 2, P95: 3, Max: 4}
	if got := two.short(); got != "mean 2 ms, p95 3 ms, max 4 ms" {
		t.Errorf("short = %q", got)
	}
	if got := millis(1234567 * time.Nanosecond); got != 1.235 {
		t.Errorf("millis = %v", got)
	}
}
