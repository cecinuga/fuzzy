package fuzzer

import (
	"strings"
	"testing"
)

func TestResponseMessage(t *testing.T) {
	cases := map[string]struct {
		contentType, body, want string
	}{
		"json is left alone":   {"application/json", `{"error":"<b>boom</b>"}`, `{"error":"<b>boom</b>"}`},
		"text is left alone":   {"text/plain", "<b>a</b>", "<b>a</b>"},
		"no body":              {"text/html", "", "Internal Server Error"},
		"html title and body":  {"text/html; charset=utf-8", "<!doctype html><html><head><title>Oops</title><style>p{color:red}</style></head><body><h1>Server</h1><p>Failed</p></body></html>", "Oops Server Failed"},
		"html scripts":         {"text/html", "<script>if (a<b) { x('</p>') }</script><p>shown</p>", "shown"},
		"html uppercase":       {"text/html", "<SCRIPT>hidden</SCRIPT><P>shown</P>", "shown"},
		"html comment":         {"text/html", "<!-- hidden <p>x</p> --><p>shown</p>", "shown"},
		"html entities":        {"text/html", "<p>a &amp; b &lt;c&gt;</p>", "a & b <c>"},
		"html literal bracket": {"text/html", "<p>1 < 2</p>", "1 < 2"},
		"html quoted bracket":  {"text/html", `<a title="x>y">link</a>`, "link"},
		"html cut in a tag":    {"text/html", `<p>start</p><a href="`, "start"},
		"html cut in script":   {"text/html", `<p>start</p><script>var a`, "start"},
		"html only markup":     {"text/html", "<br><hr>", "<br><hr>"},
		"xhtml":                {"application/xhtml+xml", "<html><body><p>shown</p></body></html>", "shown"},
		"xml pairs":            {"application/xml", `<?xml version="1.0"?><error code="42"><detail>boom</detail></error>`, "code: 42; detail: boom"},
		"xml drops namespace":  {"text/xml", `<a:e xmlns:a="urn:x" xmlns="urn:y" xml:lang="en">v</a:e>`, "e: v"},
		"soap 1.1 fault": {"text/xml; charset=utf-8", `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"><soap:Body><soap:Fault>
			<faultcode>soap:Server</faultcode><faultstring>boom</faultstring></soap:Fault></soap:Body></soap:Envelope>`, "faultcode: soap:Server; faultstring: boom"},
		"soap 1.2 fault":          {"application/soap+xml", `<e:Envelope xmlns:e="u"><e:Body><e:Fault><e:Code><e:Value>e:Sender</e:Value></e:Code><e:Reason><e:Text xml:lang="en">bad</e:Text></e:Reason></e:Fault></e:Body></e:Envelope>`, "Value: e:Sender; Text: bad"},
		"xml mixed content":       {"application/xml", "<p>one <b>two</b> three</p>", "p: one; b: two; three"},
		"xml cut short":           {"application/xml", "<a><b>kept</b><c>lo", "b: kept; c: lo"},
		"xml other charset":       {"application/xml", `<?xml version="1.0" encoding="ISO-8859-1"?><a>kept</a>`, "a: kept"},
		"xml not xml":             {"application/xml", "plain words", "plain words"},
		"xml entity not expanded": {"application/xml", `<!DOCTYPE a [<!ENTITY x "expanded">]><a>&x;</a>`, "a: &x;"},
		"sniffed html":            {"", "<!DOCTYPE html><html><body><p>shown</p></body></html>", "shown"},
		"sniffed xml":             {"", `<?xml version="1.0"?><a>shown</a>`, "a: shown"},
		"sniffed plain":           {"", "<x>not markup</x>", "<x>not markup</x>"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := responseMessage(tc.contentType, []byte(tc.body), 500); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResponseMessageIsBounded(t *testing.T) {
	long := []byte("<p>" + strings.Repeat("word ", 1000) + "</p>")
	if got := responseMessage("text/html", long, 200); len([]rune(got)) != maxMsgRunes+1 {
		t.Errorf("html message has %d runes", len([]rune(got)))
	}
	if got := responseMessage("application/json", []byte(strings.Repeat("x", snippetSize)), 200); len([]rune(got)) != maxMsgRunes+1 {
		t.Errorf("plain message has %d runes", len([]rune(got)))
	}
}
