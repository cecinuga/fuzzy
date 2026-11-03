// Package report renders the outcome of a run as a document: Markdown, JSON, plain
// text, CSV, XML or HTML.
package report

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"fuzzy/internal/fuzzer"
	"fuzzy/internal/generator"
	"fuzzy/internal/openapi"
)

const maxDescribeDepth = 4

// Format is the file format of a report.
type Format string

const (
	FormatMarkdown Format = "markdown"
	FormatJSON     Format = "json"
	FormatText     Format = "txt"
	FormatCSV      Format = "csv"
	FormatXML      Format = "xml"
	FormatHTML     Format = "html"
)

// Formats lists every format.
func Formats() []Format {
	return []Format{FormatMarkdown, FormatJSON, FormatText, FormatCSV, FormatXML, FormatHTML}
}

// ParseFormat returns the format with this name.
func ParseFormat(name string) (Format, error) {
	if format := Format(name); slices.Contains(Formats(), format) {
		return format, nil
	}
	return "", fmt.Errorf("unknown format %q (want markdown, json, txt, csv, xml or html)", name)
}

// FormatOfPath returns the format a file's extension asks for, ignoring case. ok
// is false for an extension that doesn't name a format, or for none at all.
func FormatOfPath(path string) (format Format, ok bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		return FormatMarkdown, true
	case ".json":
		return FormatJSON, true
	case ".txt":
		return FormatText, true
	case ".csv":
		return FormatCSV, true
	case ".xml":
		return FormatXML, true
	case ".html", ".htm":
		return FormatHTML, true
	}
	return "", false
}

// Extension returns the file extension of the format, with its dot.
func (f Format) Extension() string {
	if f == FormatMarkdown {
		return ".md"
	}
	return "." + string(f)
}

// Report is the input of Write.
type Report struct {
	SpecTitle   string
	SpecVersion string
	Remote      string
	Spec        string
	Dictionary  string
	Filter      fuzzer.StatusFilter
	Bounds      generator.Bounds
	Seed        uint64
	Started     time.Time
	Duration    time.Duration
	Interrupted bool // the results are partial: the run was stopped, or is still going
	Warnings    []string
	Outcomes    []fuzzer.Outcome
	Skipped     []fuzzer.Skipped
}

// Write renders the report to w in the given format.
func Write(w io.Writer, format Format, r Report) error {
	switch format {
	case FormatMarkdown:
		return writeMarkdown(w, newDocument(r))
	case FormatJSON:
		return writeJSON(w, newDocument(r))
	case FormatText:
		return writeText(w, newDocument(r))
	case FormatCSV:
		return writeCSV(w, newDocument(r))
	case FormatXML:
		return writeXML(w, newDocument(r))
	case FormatHTML:
		return writeHTML(w, newDocument(r))
	}
	return fmt.Errorf("unknown report format %q", format)
}

// writeJSON keeps <, > and & as they are: the report is not meant to be embedded
// in a page, and the inputs that triggered a result should stay readable.
func writeJSON(w io.Writer, doc document) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(doc)
}

// writeMarkdown renders the document. Everything that comes from the remote or
// the spec passes through escape, so a response cannot break out of its table
// cell or inject markup into the document.
func writeMarkdown(w io.Writer, doc document) error {
	var b strings.Builder

	b.WriteString("# Fuzzy report\n\n")
	if doc.Interrupted {
		b.WriteString("> ⚠️ The run was interrupted or is still going: these results are partial.\n\n")
	}
	writeOverview(&b, doc.Overview)
	writeSummary(&b, doc.Summary)
	for i := range doc.Endpoints {
		writeEndpoint(&b, &doc.Endpoints[i])
	}
	if len(doc.Skipped) > 0 {
		b.WriteString("## Skipped endpoints\n\n| Endpoint | Reason |\n|---|---|\n")
		for _, s := range doc.Skipped {
			fmt.Fprintf(&b, "| %s | %s |\n", escape(s.Endpoint), escape(s.Reason))
		}
		b.WriteString("\n")
	}
	if len(doc.Warnings) > 0 {
		b.WriteString("## Warnings\n\n")
		for _, warning := range doc.Warnings {
			fmt.Fprintf(&b, "- %s\n", escape(warning))
		}
		b.WriteString("\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func writeOverview(b *strings.Builder, o overview) {
	dictionary := o.Dictionary
	if dictionary == "" {
		dictionary = "none (random values)"
	}

	b.WriteString("| | |\n|---|---|\n")
	for _, row := range [][2]string{
		{"Target", escape(o.Target)},
		{"Spec", escape(fmt.Sprintf("%s (%s %s)", o.Spec.Path, o.Spec.Title, o.Spec.Version))},
		{"Dictionary", escape(dictionary)},
		{"Random bounds", cellList(o.Bounds.lines())},
		{"Charset", escape(o.Charset)},
		{"Status filter", escape(o.Filter)},
		{"Seed", strconv.FormatUint(o.Seed, 10)},
		{"Started", o.Started},
		{"Duration", o.Duration},
		{"Requests sent", strconv.Itoa(o.RequestsSent)},
		{"Results reported", strconv.Itoa(o.ResultsReported)},
	} {
		fmt.Fprintf(b, "| **%s** | %s |\n", row[0], row[1])
	}
	b.WriteString("\n")
}

// cellList renders items as a list inside a table cell. It has to be HTML:
// Markdown lists can't span the lines of a table row.
func cellList(items []string) string {
	var b strings.Builder
	b.WriteString("<ul>")
	for _, item := range items {
		fmt.Fprintf(&b, "<li>%s</li>", escape(item))
	}
	b.WriteString("</ul>")
	return b.String()
}

func writeSummary(b *strings.Builder, rows []summaryRow) {
	b.WriteString("## Summary\n\n| Endpoint | Sent | Reported | Status codes | Response time |\n|---|---|---|---|---|\n")
	for _, row := range rows {
		fmt.Fprintf(b, "| %s | %d | %d | %s | %s |\n", escape(row.Endpoint), row.Sent, row.Reported, countsLine(row.StatusCodes), row.ResponseTime.short())
	}
	b.WriteString("\n")
}

func writeEndpoint(b *strings.Builder, e *endpoint) {
	fmt.Fprintf(b, "## %s %s\n\n", e.Method, escape(e.Path))
	if e.Summary != "" {
		fmt.Fprintf(b, "%s\n\n", escape(e.Summary))
	}
	fmt.Fprintf(b, "- **HTTP verb:** %s\n- **Path:** %s\n", e.Method, escape(e.FullPath))
	if e.OperationID != "" {
		fmt.Fprintf(b, "- **Operation ID:** %s\n", escape(e.OperationID))
	}
	b.WriteString("\n")

	writeParameters(b, e.Parameters)

	res := &e.Results
	fmt.Fprintf(b, "### Results\n\n%d sent, %d reported (filter: %s). Status codes: %s.\n\nResponse time: %s.\n\n",
		res.Sent, res.Reported, escape(res.Filter), countsLine(res.StatusCodes), res.ResponseTime.text())
	if len(res.Items) == 0 {
		return
	}
	b.WriteString("| Endpoint | Status | Message | Input | Time |\n|---|---|---|---|---|\n")
	for _, item := range res.Items {
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n",
			escape(item.Endpoint), statusLabel(int(item.Status)), escape(item.Message), escape(item.Input), millisText(item.DurationMs))
	}
	b.WriteString("\n")
}

func writeParameters(b *strings.Builder, params []parameter) {
	b.WriteString("### Parameters\n\n")
	if len(params) == 0 {
		b.WriteString("None.\n\n")
		return
	}
	b.WriteString("| Name | In | Type | Required | Constraints |\n|---|---|---|---|---|\n")
	for _, p := range params {
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n",
			escape(p.Name), escape(p.In), escape(p.Type), yesNo(p.Required), escape(p.Constraints))
	}
	b.WriteString("\n")
}

// flattenProperties lists the fields of a body schema with dotted names. The
// path set stops at recursive schemas.
func flattenProperties(s *openapi.Schema, prefix string, path map[*openapi.Schema]bool, emit func(string, *openapi.Schema, bool)) {
	if s == nil || path[s] || len(path) >= maxDescribeDepth {
		return
	}
	path[s] = true
	defer delete(path, s)

	if s.EffectiveType() == "array" && s.Items != nil {
		flattenProperties(s.Items, prefix+"[]", path, emit)
		return
	}
	for _, name := range slices.Sorted(maps.Keys(s.Properties)) {
		prop := s.Properties[name]
		if prop.ReadOnly {
			continue
		}
		full := name
		if prefix != "" {
			full = prefix + "." + name
		}
		emit(full, prop, slices.Contains(s.Required, name))
		flattenProperties(prop, full, path, emit)
	}
}

func typeName(s *openapi.Schema, depth int) string {
	if s == nil {
		return "any"
	}
	if depth >= maxDescribeDepth {
		return "…"
	}
	suffix := ""
	if s.Nullable {
		suffix = " (nullable)"
	}
	branches, word := s.OneOf, "oneOf"
	if len(branches) == 0 {
		branches, word = s.AnyOf, "anyOf"
	}
	if len(branches) > 0 {
		names := make([]string, len(branches))
		for i, branch := range branches {
			names[i] = typeName(branch, depth+1)
		}
		return word + "(" + strings.Join(names, ", ") + ")" + suffix
	}

	name := s.EffectiveType()
	switch {
	case name == "array":
		return "array of " + typeName(s.Items, depth+1) + suffix
	case s.Format != "":
		return name + " (" + s.Format + ")" + suffix
	}
	return name + suffix
}

func constraints(s *openapi.Schema) string {
	if s == nil {
		return ""
	}
	var parts []string
	add := func(format string, args ...any) { parts = append(parts, fmt.Sprintf(format, args...)) }
	if len(s.Enum) > 0 {
		values := make([]string, len(s.Enum))
		for i, v := range s.Enum {
			encoded, _ := json.Marshal(v)
			values[i] = string(encoded)
		}
		add("enum: %s", strings.Join(values, ", "))
	}
	if s.Minimum != nil {
		add("%s %s", exclusiveWord(s.ExclusiveMinimum, "min"), s.Minimum)
	}
	if s.Maximum != nil {
		add("%s %s", exclusiveWord(s.ExclusiveMaximum, "max"), s.Maximum)
	}
	if s.MultipleOf != nil {
		add("multiple of %s", s.MultipleOf)
	}
	if s.MinLength != nil {
		add("min length %d", *s.MinLength)
	}
	if s.MaxLength != nil {
		add("max length %d", *s.MaxLength)
	}
	if s.Pattern != "" {
		add("pattern: %s", s.Pattern)
	}
	if s.MinItems != nil {
		add("min items %d", *s.MinItems)
	}
	if s.MaxItems != nil {
		add("max items %d", *s.MaxItems)
	}
	if s.UniqueItems {
		add("unique items")
	}
	if s.MinProperties != nil {
		add("min properties %d", *s.MinProperties)
	}
	if s.MaxProperties != nil {
		add("max properties %d", *s.MaxProperties)
	}
	return strings.Join(parts, "; ")
}

func exclusiveWord(exclusive bool, word string) string {
	if exclusive {
		return "exclusive " + word
	}
	return word
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func statusLabel(status int) string {
	if status == fuzzer.TransportError {
		return "ERR"
	}
	return strconv.Itoa(status)
}

// countsLine renders status counts in ascending order, e.g. "200 ×3, 404 ×97".
func countsLine(counts map[int]int) string {
	if len(counts) == 0 {
		return "none"
	}
	statuses := slices.SortedFunc(maps.Keys(counts), cmp.Compare[int])
	parts := make([]string, len(statuses))
	for i, status := range statuses {
		parts[i] = fmt.Sprintf("%s ×%d", statusLabel(status), counts[status])
	}
	return strings.Join(parts, ", ")
}

var escaper = strings.NewReplacer(
	`\`, `\\`, "|", `\|`, "`", "\\`", "*", `\*`, "_", `\_`, "~", `\~`,
	"[", `\[`, "]", `\]`, "&", "&amp;", "<", "&lt;", ">", "&gt;",
)

// escape makes untrusted text safe for a one-line Markdown table cell.
func escape(s string) string {
	return escaper.Replace(oneLine(s))
}

// oneLine keeps untrusted text on its line and takes out what could change how it
// displays: control characters such as terminal escapes, and invisible format
// characters such as bidirectional overrides.
func oneLine(s string) string {
	s = strings.ToValidUTF8(s, "�")
	return strings.Map(func(c rune) rune {
		switch {
		case c == '\n' || c == '\r' || c == '\t':
			return ' '
		case unicode.IsControl(c) || unicode.Is(unicode.Cf, c):
			return -1
		}
		return c
	}, s)
}
