package report

import (
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"

	"fuzzy/internal/fuzzer"
)

// writeHTML renders the document as one self-contained page, built by hand like
// the other formats: html/template would make the linker keep every method of
// every package, which costs each run 3 MB of binary and 2 MB of memory, HTML
// report or not.
//
// Everything from the remote or the spec passes through htmlText, and only into
// element content. The page's policy also allows no scripts, images or requests at
// all, so a hostile response stays text even if a browser disagrees.
func writeHTML(w io.Writer, doc document) error {
	var b strings.Builder

	b.WriteString(htmlHead)
	if doc.Interrupted {
		b.WriteString("<p class=\"partial\">The run was interrupted or is still going: these results are partial.</p>\n")
	}
	writeHTMLOverview(&b, doc.Overview)

	b.WriteString("<h2>Summary</h2>\n")
	htmlTableStart(&b, "Endpoint", "Sent", "Reported", "Status codes", "Response time")
	for _, row := range doc.Summary {
		htmlRow(&b, htmlText(row.Endpoint), strconv.Itoa(row.Sent), strconv.Itoa(row.Reported), countsLine(row.StatusCodes), row.ResponseTime.short())
	}
	b.WriteString(htmlTableEnd)

	for i := range doc.Endpoints {
		writeHTMLEndpoint(&b, &doc.Endpoints[i])
	}
	if len(doc.Skipped) > 0 {
		b.WriteString("<h2>Skipped endpoints</h2>\n")
		htmlTableStart(&b, "Endpoint", "Reason")
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
	b.WriteString("</body>\n</html>\n")

	_, err := io.WriteString(w, b.String())
	return err
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

func writeHTMLEndpoint(b *strings.Builder, e *endpoint) {
	fmt.Fprintf(b, "<section>\n<h2>%s %s</h2>\n", htmlText(e.Method), htmlText(e.Path))
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
		htmlTableStart(b, "Name", "In", "Type", "Required", "Constraints")
		for _, p := range e.Parameters {
			htmlRow(b, htmlText(p.Name), htmlText(p.In), htmlText(p.Type), yesNo(p.Required), htmlText(p.Constraints))
		}
		b.WriteString(htmlTableEnd)
	}

	res := &e.Results
	fmt.Fprintf(b, "<h3>Results</h3>\n<p>%d sent, %d reported (filter: %s). Status codes: %s.</p>\n<p>Response time: %s.</p>\n",
		res.Sent, res.Reported, htmlText(res.Filter), countsLine(res.StatusCodes), res.ResponseTime.text())
	if len(res.Items) > 0 {
		htmlTableStart(b, "Endpoint", "Status", "Message", "Input", "Time")
		for _, item := range res.Items {
			fmt.Fprintf(b, "<tr><td>%s</td><td class=\"%s\">%s</td><td>%s</td><td><code>%s</code></td><td>%s</td></tr>\n",
				htmlText(item.Endpoint), statusClass(item.Status), statusLabel(int(item.Status)),
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

func htmlTableStart(b *strings.Builder, headings ...string) {
	b.WriteString("<div class=\"scroll\"><table>\n<tr>")
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

const htmlHead = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'">
<title>Fuzzy report</title>
<style>
:root { color-scheme: light dark; --line: #8884; --red: #c62828; --orange: #b26a00; --green: #2e7d32; }
@media (prefers-color-scheme: dark) { :root { --red: #ef9a9a; --orange: #ffcc80; --green: #a5d6a7; } }
body { font: 15px/1.5 system-ui, sans-serif; margin: 0 auto; max-width: 1100px; padding: 16px; }
h2 { margin-top: 2.5em; overflow-wrap: anywhere; }
.scroll { overflow-x: auto; }
table { border-collapse: collapse; margin: 1em 0; width: 100%; }
th, td { border: 1px solid var(--line); padding: 4px 8px; text-align: left; vertical-align: top; }
td { overflow-wrap: anywhere; }
code { font-family: ui-monospace, monospace; font-size: 13px; }
.partial { border: 1px solid var(--orange); color: var(--orange); font-weight: 600; padding: 8px 12px; }
.failed, .server-error { color: var(--red); font-weight: 600; }
.client-error { color: var(--orange); }
.success { color: var(--green); }
</style>
</head>
<body>
<h1>Fuzzy report</h1>
`
