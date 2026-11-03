package report

import (
	"html/template"
	"io"
	"sync"

	"fuzzy/internal/fuzzer"
)

// writeHTML renders the document as one self-contained page. html/template escapes
// everything by context, and the page's policy allows no scripts, images or
// requests at all, so a hostile response stays text even if a browser disagrees.
func writeHTML(w io.Writer, doc document) error {
	return htmlReport().Execute(w, doc)
}

// htmlReport parses the page on first use: most runs never write HTML, and parsing
// it costs a run more memory than all of its requests.
var htmlReport = sync.OnceValue(func() *template.Template {
	return template.Must(template.New("report").Funcs(template.FuncMap{
		"clean":        oneLine,
		"counts":       countsLine,
		"status":       func(s statusCode) string { return statusLabel(int(s)) },
		"statusClass":  statusClass,
		"milliseconds": millisText,
		"yesNo":        yesNo,
		"bounds":       randomBounds.lines,
		"timeText":     responseTime.text,
		"timeShort":    responseTime.short,
	}).Parse(htmlPage))
})

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

const htmlPage = `<!doctype html>
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
{{- if .Interrupted}}
<p class="partial">The run was interrupted or is still going: these results are partial.</p>
{{- end}}
{{- with .Overview}}
<div class="scroll"><table>
<tr><th>Target</th><td>{{clean .Target}}</td></tr>
<tr><th>Spec</th><td>{{clean .Spec.Path}} ({{clean .Spec.Title}} {{clean .Spec.Version}})</td></tr>
<tr><th>Dictionary</th><td>{{if .Dictionary}}{{clean .Dictionary}}{{else}}none (random values){{end}}</td></tr>
<tr><th>Random bounds</th><td>{{range bounds .Bounds}}{{.}}<br>{{end}}</td></tr>
<tr><th>Charset</th><td>{{clean .Charset}}</td></tr>
<tr><th>Status filter</th><td>{{clean .Filter}}</td></tr>
<tr><th>Seed</th><td>{{.Seed}}</td></tr>
<tr><th>Started</th><td>{{.Started}}</td></tr>
<tr><th>Duration</th><td>{{.Duration}}</td></tr>
<tr><th>Requests sent</th><td>{{.RequestsSent}}</td></tr>
<tr><th>Results reported</th><td>{{.ResultsReported}}</td></tr>
</table></div>
{{- end}}
<h2>Summary</h2>
<div class="scroll"><table>
<tr><th>Endpoint</th><th>Sent</th><th>Reported</th><th>Status codes</th><th>Response time</th></tr>
{{- range .Summary}}
<tr><td>{{clean .Endpoint}}</td><td>{{.Sent}}</td><td>{{.Reported}}</td><td>{{counts .StatusCodes}}</td><td>{{timeShort .ResponseTime}}</td></tr>
{{- end}}
</table></div>
{{- range .Endpoints}}
<section>
<h2>{{clean .Method}} {{clean .Path}}</h2>
{{- if .Summary}}
<p>{{clean .Summary}}</p>
{{- end}}
<ul>
<li><strong>HTTP verb:</strong> {{clean .Method}}</li>
<li><strong>Path:</strong> {{clean .FullPath}}</li>
{{- if .OperationID}}
<li><strong>Operation ID:</strong> {{clean .OperationID}}</li>
{{- end}}
</ul>
<h3>Parameters</h3>
{{- if .Parameters}}
<div class="scroll"><table>
<tr><th>Name</th><th>In</th><th>Type</th><th>Required</th><th>Constraints</th></tr>
{{- range .Parameters}}
<tr><td>{{clean .Name}}</td><td>{{clean .In}}</td><td>{{clean .Type}}</td><td>{{yesNo .Required}}</td><td>{{clean .Constraints}}</td></tr>
{{- end}}
</table></div>
{{- else}}
<p>None.</p>
{{- end}}
<h3>Results</h3>
{{- with .Results}}
<p>{{.Sent}} sent, {{.Reported}} reported (filter: {{clean .Filter}}). Status codes: {{counts .StatusCodes}}.</p>
<p>Response time: {{timeText .ResponseTime}}.</p>
{{- if .Items}}
<div class="scroll"><table>
<tr><th>Endpoint</th><th>Status</th><th>Message</th><th>Input</th><th>Time</th></tr>
{{- range .Items}}
<tr><td>{{clean .Endpoint}}</td><td class="{{statusClass .Status}}">{{status .Status}}</td><td>{{clean .Message}}</td><td><code>{{clean .Input}}</code></td><td>{{milliseconds .DurationMs}}</td></tr>
{{- end}}
</table></div>
{{- end}}
{{- end}}
</section>
{{- end}}
{{- if .Skipped}}
<h2>Skipped endpoints</h2>
<div class="scroll"><table>
<tr><th>Endpoint</th><th>Reason</th></tr>
{{- range .Skipped}}
<tr><td>{{clean .Endpoint}}</td><td>{{clean .Reason}}</td></tr>
{{- end}}
</table></div>
{{- end}}
{{- if .Warnings}}
<h2>Warnings</h2>
<ul>
{{- range .Warnings}}
<li>{{clean .}}</li>
{{- end}}
</ul>
{{- end}}
</body>
</html>
`
