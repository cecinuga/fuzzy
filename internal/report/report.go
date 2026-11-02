// Package report renders the outcome of a run as a Markdown document.
package report

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"fuzzy/internal/fuzzer"
	"fuzzy/internal/openapi"
)

const maxDescribeDepth = 4

// Report is the input of Write.
type Report struct {
	SpecTitle   string
	SpecVersion string
	Remote      string
	Spec        string
	Dictionary  string
	Filter      fuzzer.StatusFilter
	Bounds      string
	Seed        uint64
	Started     time.Time
	Duration    time.Duration
	Interrupted bool
	Warnings    []string
	Outcomes    []fuzzer.Outcome
	Skipped     []fuzzer.Skipped
}

// Write renders the report to w. Everything that comes from the remote or the
// spec passes through escape, so a response cannot break out of its table cell
// or inject markup into the document.
func Write(w io.Writer, r Report) error {
	var b strings.Builder

	b.WriteString("# Fuzzy report\n\n")
	if r.Interrupted {
		b.WriteString("> ⚠️ The run was interrupted: these results are partial.\n\n")
	}
	writeOverview(&b, r)
	writeSummary(&b, r)
	for i := range r.Outcomes {
		writeEndpoint(&b, &r.Outcomes[i], r.Filter)
	}
	if len(r.Skipped) > 0 {
		b.WriteString("## Skipped endpoints\n\n| Endpoint | Reason |\n|---|---|\n")
		for _, s := range r.Skipped {
			fmt.Fprintf(&b, "| %s %s | %s |\n", s.Op.Method, escape(s.Op.Path), escape(s.Reason))
		}
		b.WriteString("\n")
	}
	if len(r.Warnings) > 0 {
		b.WriteString("## Warnings\n\n")
		for _, warning := range r.Warnings {
			fmt.Fprintf(&b, "- %s\n", escape(warning))
		}
		b.WriteString("\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func writeOverview(b *strings.Builder, r Report) {
	sent, kept := 0, 0
	for i := range r.Outcomes {
		sent += r.Outcomes[i].Sent
		kept += len(r.Outcomes[i].Results)
	}
	dictionary := r.Dictionary
	if dictionary == "" {
		dictionary = "none (random values)"
	}

	b.WriteString("| | |\n|---|---|\n")
	for _, row := range [][2]string{
		{"Target", r.Remote},
		{"Spec", fmt.Sprintf("%s (%s %s)", r.Spec, r.SpecTitle, r.SpecVersion)},
		{"Dictionary", dictionary},
		{"Random bounds", r.Bounds},
		{"Status filter", r.Filter.String()},
		{"Seed", strconv.FormatUint(r.Seed, 10)},
		{"Started", r.Started.Format(time.RFC3339)},
		{"Duration", r.Duration.Round(time.Millisecond).String()},
		{"Requests sent", strconv.Itoa(sent)},
		{"Results reported", strconv.Itoa(kept)},
	} {
		fmt.Fprintf(b, "| **%s** | %s |\n", row[0], escape(row[1]))
	}
	b.WriteString("\n")
}

func writeSummary(b *strings.Builder, r Report) {
	b.WriteString("## Summary\n\n| Endpoint | Sent | Reported | Status codes |\n|---|---|---|---|\n")
	for i := range r.Outcomes {
		o := &r.Outcomes[i]
		fmt.Fprintf(b, "| %s %s | %d | %d | %s |\n",
			o.Target.Op.Method, escape(o.Target.Op.Path), o.Sent, len(o.Results), countsLine(o.Counts))
	}
	b.WriteString("\n")
}

func writeEndpoint(b *strings.Builder, o *fuzzer.Outcome, filter fuzzer.StatusFilter) {
	op := o.Target.Op
	fmt.Fprintf(b, "## %s %s\n\n", op.Method, escape(op.Path))
	if op.Summary != "" {
		fmt.Fprintf(b, "%s\n\n", escape(op.Summary))
	}
	fmt.Fprintf(b, "- **HTTP verb:** %s\n- **Path:** %s\n", op.Method, escape(op.BasePath+op.Path))
	if op.OperationID != "" {
		fmt.Fprintf(b, "- **Operation ID:** %s\n", escape(op.OperationID))
	}
	b.WriteString("\n")

	writeParameters(b, op)

	fmt.Fprintf(b, "### Results\n\n%d sent, %d reported (filter: %s). Status codes: %s.\n\n",
		o.Sent, len(o.Results), escape(filter.String()), countsLine(o.Counts))
	if len(o.Results) == 0 {
		return
	}
	b.WriteString("| Endpoint | Status | Message | Input |\n|---|---|---|---|\n")
	for _, res := range o.Results {
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n",
			escape(res.Endpoint), statusLabel(res.Status), escape(res.Message), escape(res.Input))
	}
	b.WriteString("\n")
}

func writeParameters(b *strings.Builder, op *openapi.Operation) {
	type row struct{ name, in, kind, required, constraints string }
	var rows []row

	for _, p := range op.Parameters {
		rows = append(rows, row{p.Name, p.In, typeName(p.Schema, 0), yesNo(p.Required), constraints(p.Schema)})
	}
	if op.RequestBody != nil {
		for _, contentType := range slices.Sorted(maps.Keys(op.RequestBody.Content)) {
			schema := op.RequestBody.Content[contentType].Schema
			rows = append(rows, row{"(body)", contentType, typeName(schema, 0), yesNo(op.RequestBody.Required), constraints(schema)})
			flattenProperties(schema, "", map[*openapi.Schema]bool{}, func(name string, s *openapi.Schema, required bool) {
				rows = append(rows, row{name, contentType, typeName(s, 0), yesNo(required), constraints(s)})
			})
		}
	}

	b.WriteString("### Parameters\n\n")
	if len(rows) == 0 {
		b.WriteString("None.\n\n")
		return
	}
	b.WriteString("| Name | In | Type | Required | Constraints |\n|---|---|---|---|---|\n")
	for _, r := range rows {
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n",
			escape(r.name), escape(r.in), escape(r.kind), r.required, escape(r.constraints))
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
	s = strings.ToValidUTF8(s, "�")
	s = strings.Map(func(c rune) rune {
		switch {
		case c == '\n' || c == '\r' || c == '\t':
			return ' '
		case unicode.IsControl(c) || unicode.Is(unicode.Cf, c):
			return -1
		}
		return c
	}, s)
	return escaper.Replace(s)
}
