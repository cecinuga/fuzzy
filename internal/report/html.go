package report

import (
	"cmp"
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"fuzzy/internal/fuzzer"
)

// The page's style and script, inlined so the report stays a single file that
// works offline. They are text, never templates: see writeHTML.
var (
	//go:embed page/report.css
	htmlStyle string
	//go:embed page/report.js
	htmlScript string
)

// writeHTML renders the document as one self-contained page, built by hand like
// the other formats: html/template would make the linker keep every method of
// every package, which costs each run 3 MB of binary and 2 MB of memory, HTML
// report or not.
//
// Everything from the remote or the spec passes through htmlText, into element
// content or a quoted attribute. The page carries the figures its script needs in
// data-* attributes, and its policy allows that one script by its hash, and no
// other script, image or request at all: a hostile response stays text even if a
// browser disagrees, and the script can't send anything anywhere. The script only
// reads the page and writes text nodes; without it, the page is the plain report.
func writeHTML(w io.Writer, doc document) error {
	var b strings.Builder

	writeHTMLHead(&b, doc)
	if doc.Interrupted {
		b.WriteString("<p class=\"partial\">The run was interrupted or is still going: these results are partial.</p>\n")
	}
	writeHTMLOverview(&b, doc.Overview)

	b.WriteString("<h2 id=\"summary\">Summary</h2>\n")
	htmlTableStart(&b, "summary", "Endpoint", "Sent", "Reported", "Status codes", "Response time")
	for i, row := range doc.Summary {
		t := row.ResponseTime
		fmt.Fprintf(&b, "<tr data-ep=\"%d\" data-method=\"%s\" data-sent=\"%d\" data-reported=\"%d\" data-codes=\"%s\""+
			" data-count=\"%d\" data-min=\"%s\" data-mean=\"%s\" data-p50=\"%s\" data-p95=\"%s\" data-p99=\"%s\" data-max=\"%s\">",
			i, htmlText(doc.Endpoints[i].Method), row.Sent, row.Reported, countsData(row.StatusCodes),
			t.Count, number(t.Min), number(t.Mean), number(t.P50), number(t.P95), number(t.P99), number(t.Max))
		fmt.Fprintf(&b, "<td><a href=\"#ep-%d\">%s</a></td><td>%d</td><td>%d</td><td>%s</td><td>%s</td></tr>\n",
			i, htmlText(row.Endpoint), row.Sent, row.Reported, countsLine(row.StatusCodes), t.short())
	}
	b.WriteString(htmlTableEnd)

	b.WriteString("<div id=\"endpoints\">\n")
	for i := range doc.Endpoints {
		writeHTMLEndpoint(&b, i, &doc.Endpoints[i])
	}
	b.WriteString("</div>\n")
	if len(doc.Skipped) > 0 {
		b.WriteString("<h2>Skipped endpoints</h2>\n")
		htmlTableStart(&b, "", "Endpoint", "Reason")
		for _, s := range doc.Skipped {
			htmlRow(&b, htmlText(s.Endpoint), htmlText(s.Reason))
		}
		b.WriteString(htmlTableEnd)
	}
	if len(doc.Warnings) > 0 {
		b.WriteString("<h2>Warnings</h2>\n<ul>\n")
		for _, warning := range doc.Warnings {
			fmt.Fprintf(&b, "<li>%s</li>\n", htmlText(warning))
		}
		b.WriteString("</ul>\n")
	}
	b.WriteString("<script>")
	b.WriteString(htmlScript)
	b.WriteString("</script>\n</body>\n</html>\n")

	_, err := io.WriteString(w, b.String())
	return err
}

// writeHTMLHead opens the page. The body carries what the script can't read from
// the tables: how long the run took and how many endpoints it skipped.
func writeHTMLHead(b *strings.Builder, doc document) {
	durationMs := int64(0)
	if d, err := time.ParseDuration(doc.Overview.Duration); err == nil {
		durationMs = d.Milliseconds()
	}
	fmt.Fprintf(b, `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="%s">
<title>Fuzzy report</title>
<style>
%s</style>
</head>
<body data-duration-ms="%d" data-skipped="%d">
<h1>Fuzzy report</h1>
`, htmlPolicy(), htmlStyle, durationMs, len(doc.Skipped))
}

// htmlPolicy allows inline style and the page's own script, pinned by its hash:
// nothing else may run or load.
func htmlPolicy() string {
	sum := sha256.Sum256([]byte(htmlScript))
	return "default-src 'none'; style-src 'unsafe-inline'; script-src 'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

func writeHTMLOverview(b *strings.Builder, o overview) {
	dictionary := o.Dictionary
	if dictionary == "" {
		dictionary = "none (random values)"
	}
	b.WriteString("<div class=\"scroll\"><table>\n")
	htmlField(b, "Target", htmlText(o.Target))
	htmlField(b, "Spec", htmlText(fmt.Sprintf("%s (%s %s)", o.Spec.Path, o.Spec.Title, o.Spec.Version)))
	htmlField(b, "Dictionary", htmlText(dictionary))
	htmlField(b, "Random bounds", htmlLines(o.Bounds.lines()))
	htmlField(b, "Charset", htmlText(o.Charset))
	htmlField(b, "Status filter", htmlText(o.Filter))
	if s := o.Selection; len(s.Include)+len(s.Exclude) > 0 {
		if len(s.Include) > 0 {
			htmlField(b, "Included", htmlLines(s.Include))
		}
		if len(s.Exclude) > 0 {
			htmlField(b, "Excluded", htmlLines(s.Exclude))
		}
		htmlField(b, "Endpoints left out", strconv.Itoa(s.LeftOut))
	}
	htmlField(b, "Seed", strconv.FormatUint(o.Seed, 10))
	htmlField(b, "Started", htmlText(o.Started))
	htmlField(b, "Duration", htmlText(o.Duration))
	htmlField(b, "Requests sent", strconv.Itoa(o.RequestsSent))
	htmlField(b, "Results reported", strconv.Itoa(o.ResultsReported))
	b.WriteString(htmlTableEnd)
}

func writeHTMLEndpoint(b *strings.Builder, i int, e *endpoint) {
	fmt.Fprintf(b, "<section id=\"ep-%d\">\n<h2>%s %s</h2>\n", i, htmlText(e.Method), htmlText(e.Path))
	if e.Summary != "" {
		fmt.Fprintf(b, "<p>%s</p>\n", htmlText(e.Summary))
	}
	fmt.Fprintf(b, "<ul>\n<li><strong>HTTP verb:</strong> %s</li>\n<li><strong>Path:</strong> %s</li>\n", htmlText(e.Method), htmlText(e.FullPath))
	if e.OperationID != "" {
		fmt.Fprintf(b, "<li><strong>Operation ID:</strong> %s</li>\n", htmlText(e.OperationID))
	}
	b.WriteString("</ul>\n<h3>Parameters</h3>\n")
	if len(e.Parameters) == 0 {
		b.WriteString("<p>None.</p>\n")
	} else {
		htmlTableStart(b, "params", "Name", "In", "Type", "Required", "Constraints")
		for _, p := range e.Parameters {
			htmlRow(b, htmlText(p.Name), htmlText(p.In), htmlText(p.Type), yesNo(p.Required), htmlText(p.Constraints))
		}
		b.WriteString(htmlTableEnd)
	}

	res := &e.Results
	fmt.Fprintf(b, "<h3>Results</h3>\n<p class=\"stats\">%d sent, %d reported (filter: %s). Status codes: %s.</p>\n<p>Response time: %s.</p>\n",
		res.Sent, res.Reported, htmlText(res.Filter), countsLine(res.StatusCodes), res.ResponseTime.text())
	if len(res.Items) > 0 {
		htmlTableStart(b, "results", "Endpoint", "Status", "Message", "Input", "Time")
		for _, item := range res.Items {
			fmt.Fprintf(b, "<tr data-s=\"%d\" data-ms=\"%s\"><td>%s</td><td class=\"%s\">%s</td><td>%s</td><td><code>%s</code></td><td>%s</td></tr>\n",
				int(item.Status), number(item.DurationMs), htmlText(item.Endpoint), statusClass(item.Status), statusLabel(int(item.Status)),
				htmlText(item.Message), htmlText(item.Input), millisText(item.DurationMs))
		}
		b.WriteString(htmlTableEnd)
	}
	b.WriteString("</section>\n")
}

// htmlText makes untrusted text safe as element content.
func htmlText(s string) string {
	return html.EscapeString(oneLine(s))
}

func htmlLines(lines []string) string {
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(htmlText(line))
		b.WriteString("<br>")
	}
	return b.String()
}

func htmlField(b *strings.Builder, name, safeValue string) {
	fmt.Fprintf(b, "<tr><th>%s</th><td>%s</td></tr>\n", name, safeValue)
}

// htmlTableStart opens a table; class, a fixed name, tells the script which it is.
func htmlTableStart(b *strings.Builder, class string, headings ...string) {
	if class == "" {
		b.WriteString("<div class=\"scroll\"><table>\n<tr>")
	} else {
		fmt.Fprintf(b, "<div class=\"scroll\"><table class=\"%s\">\n<tr>", class)
	}
	for _, heading := range headings {
		fmt.Fprintf(b, "<th>%s</th>", heading)
	}
	b.WriteString("</tr>\n")
}

func htmlRow(b *strings.Builder, safeCells ...string) {
	b.WriteString("<tr>")
	for _, cell := range safeCells {
		fmt.Fprintf(b, "<td>%s</td>", cell)
	}
	b.WriteString("</tr>\n")
}

const htmlTableEnd = "</table></div>\n"

// countsData renders status counts for the script, e.g. "200:3 404:97 0:2", where
// 0 means no response.
func countsData(counts map[int]int) string {
	statuses := slices.SortedFunc(maps.Keys(counts), cmp.Compare[int])
	parts := make([]string, len(statuses))
	for i, status := range statuses {
		parts[i] = strconv.Itoa(status) + ":" + strconv.Itoa(counts[status])
	}
	return strings.Join(parts, " ")
}

// statusClass returns the style of a result's status. It is a fixed name, never
// text from the response.
func statusClass(s statusCode) string {
	switch {
	case s == fuzzer.TransportError:
		return "failed"
	case s >= 500:
		return "server-error"
	case s >= 400:
		return "client-error"
	}
	return "success"
}
