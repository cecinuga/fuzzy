package report

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"fuzzy/internal/fuzzer"
	"fuzzy/internal/openapi"
)

// withoutPageScript checks that the page has its own script exactly once, and
// returns the page without it, so that a test can look for any other script.
func withoutPageScript(t *testing.T, page string) string {
	t.Helper()
	own := "<script>" + htmlScript + "</script>"
	if n := strings.Count(page, own); n != 1 {
		t.Fatalf("the page has its script %d times", n)
	}
	return strings.Replace(page, own, "", 1)
}

var policyPattern = regexp.MustCompile(`<meta http-equiv="Content-Security-Policy" content="([^"]*)">`)

func TestHTMLPolicyAllowsOnlyThePageScript(t *testing.T) {
	page := writeString(t, FormatHTML, hostileReport(hostileMessage))
	found := policyPattern.FindStringSubmatch(page)
	if found == nil {
		t.Fatalf("no policy:\n%.600s", page)
	}
	policy := found[1]

	start := strings.Index(page, "<script>")
	end := strings.Index(page, "</script>")
	if start < 0 || end < start {
		t.Fatal("no script")
	}
	sum := sha256.Sum256([]byte(page[start+len("<script>") : end]))
	want := "script-src 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
	if !strings.Contains(policy, want) {
		t.Errorf("policy %q doesn't pin the script that is on the page (want %q)", policy, want)
	}
	if !strings.HasPrefix(policy, "default-src 'none';") {
		t.Errorf("policy %q must deny everything it doesn't name", policy)
	}
	for _, loose := range []string{"unsafe-eval", "unsafe-hashes", "strict-dynamic", "http", "data:", "*"} {
		if strings.Contains(policy, loose) {
			t.Errorf("policy %q allows %s", policy, loose)
		}
	}
	if scriptSrc := policy[strings.Index(policy, "script-src"):]; strings.Contains(scriptSrc, "unsafe-inline") {
		t.Errorf("policy %q runs any inline script", policy)
	}
}

// The script works on text from the server and the spec. It must only ever write
// it as text, and has no business making requests.
func TestHTMLScriptStaysInert(t *testing.T) {
	for _, banned := range []string{
		"innerHTML", "outerHTML", "insertAdjacentHTML", "createContextualFragment", "DOMParser", "document.write",
		"eval(", "Function(", "srcdoc", "javascript:",
		"fetch(", "XMLHttpRequest", "WebSocket", "EventSource", "sendBeacon", "import(", "window.open", "location.assign",
		"</script", "<!--",
	} {
		if strings.Contains(htmlScript, banned) {
			t.Errorf("the script contains %q", banned)
		}
	}
	if strings.Contains(htmlStyle, "</style") || strings.Contains(htmlStyle, "url(") {
		t.Error("the style must not close its element or load anything")
	}
}

func TestHTMLCarriesTheFiguresTheScriptReads(t *testing.T) {
	r := runReport(t, func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(404)
	}, "")
	page := writeString(t, FormatHTML, r)
	doc := newDocument(r)

	mustContain(t, page, `<body data-duration-ms="1500" data-skipped="0">`, `<h2 id="summary">`,
		`<table class="summary">`, `<div id="endpoints">`, `<table class="params">`, `<table class="results">`)
	for i, row := range doc.Summary {
		rt := row.ResponseTime
		mustContain(t, page,
			fmt.Sprintf(`<tr data-ep="%d" data-method="%s" data-sent="5" data-reported="5" data-codes="%s" data-count="5" data-min="%s" data-mean="%s" data-p50="%s" data-p95="%s" data-p99="%s" data-max="%s">`,
				i, doc.Endpoints[i].Method, countsData(row.StatusCodes), number(rt.Min), number(rt.Mean), number(rt.P50), number(rt.P95), number(rt.P99), number(rt.Max)),
			fmt.Sprintf(`<a href="#ep-%d">%s</a>`, i, row.Endpoint),
			fmt.Sprintf(`<section id="ep-%d">`, i))
		for _, item := range doc.Endpoints[i].Results.Items {
			mustContain(t, page, fmt.Sprintf(`<tr data-s="%d" data-ms="%s">`, item.Status, number(item.DurationMs)))
		}
	}
	if got := strings.Count(page, `data-codes="500:5"`); got != 1 {
		t.Errorf("%d endpoints got 500 ×5, want the POST one", got)
	}
}

func TestHTMLMarksRequestsWithoutAResponse(t *testing.T) {
	op := &openapi.Operation{Method: "GET", Path: "/a"}
	page := writeString(t, FormatHTML, Report{Outcomes: []fuzzer.Outcome{{
		Target: &fuzzer.Target{Op: op}, Sent: 3, Counts: map[int]int{fuzzer.TransportError: 2, 200: 1},
		Results: []fuzzer.Result{{Endpoint: "GET /a", Status: fuzzer.TransportError, Message: "refused"}},
	}}})
	mustContain(t, page, `data-codes="0:2 200:1"`, `<tr data-s="0" data-ms="0">`, `<body data-duration-ms="0" data-skipped="0">`)
}

func TestHTMLAttributesStayQuoted(t *testing.T) {
	op := &openapi.Operation{Method: `GET" onmouseover="alert(1)`, Path: "/a"}
	page := writeString(t, FormatHTML, Report{Outcomes: []fuzzer.Outcome{{Target: &fuzzer.Target{Op: op}, Sent: 1, Counts: map[int]int{200: 1}}}})
	if strings.Contains(page, `" onmouseover="`) {
		t.Errorf("a method broke out of its attribute:\n%s", page)
	}
	mustContain(t, page, `data-method="GET&#34; onmouseover=&#34;alert(1)"`)
}

func TestCountsData(t *testing.T) {
	if got := countsData(map[int]int{500: 1, 0: 2, 200: 30}); got != "0:2 200:30 500:1" {
		t.Errorf("countsData = %q", got)
	}
	if got := countsData(nil); got != "" {
		t.Errorf("countsData(nil) = %q", got)
	}
}
