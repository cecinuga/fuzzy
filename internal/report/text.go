package report

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// writeText renders the document without any markup, to read in a terminal or a
// pager. Everything from the remote or the spec passes through oneLine, so a
// response can't forge a section or send escape sequences to a terminal.
func writeText(w io.Writer, doc document) error {
	var b strings.Builder

	b.WriteString("Fuzzy report\n============\n\n")
	if doc.Interrupted {
		b.WriteString("WARNING: The run was interrupted or is still going: these results are partial.\n\n")
	}

	o := doc.Overview
	dictionary := o.Dictionary
	if dictionary == "" {
		dictionary = "none (random values)"
	}
	for _, row := range [][2]string{
		{"Target", oneLine(o.Target)},
		{"Spec", oneLine(fmt.Sprintf("%s (%s %s)", o.Spec.Path, o.Spec.Title, o.Spec.Version))},
		{"Dictionary", oneLine(dictionary)},
		{"Random bounds", strings.Join(o.Bounds.lines(), ", ")},
		{"Charset", oneLine(o.Charset)},
		{"Status filter", oneLine(o.Filter)},
		{"Seed", strconv.FormatUint(o.Seed, 10)},
		{"Started", o.Started},
		{"Duration", o.Duration},
		{"Requests sent", strconv.Itoa(o.RequestsSent)},
		{"Results reported", strconv.Itoa(o.ResultsReported)},
	} {
		fmt.Fprintf(&b, "%-17s %s\n", row[0]+":", row[1])
	}
	if s := o.Selection; len(s.Include)+len(s.Exclude) > 0 {
		if len(s.Include) > 0 {
			fmt.Fprintf(&b, "%-17s %s\n", "Included:", oneLine(strings.Join(s.Include, "; ")))
		}
		if len(s.Exclude) > 0 {
			fmt.Fprintf(&b, "%-17s %s\n", "Excluded:", oneLine(strings.Join(s.Exclude, "; ")))
		}
		fmt.Fprintf(&b, "%-17s %d\n", "Left out:", s.LeftOut)
	}

	textHeading(&b, "Summary")
	for _, row := range doc.Summary {
		fmt.Fprintf(&b, "%s: %d sent, %d reported. Status codes: %s. Response time: %s.\n",
			oneLine(row.Endpoint), row.Sent, row.Reported, countsLine(row.StatusCodes), row.ResponseTime.short())
	}
	for i := range doc.Endpoints {
		writeTextEndpoint(&b, &doc.Endpoints[i])
	}
	if len(doc.Skipped) > 0 {
		textHeading(&b, "Skipped endpoints")
		for _, s := range doc.Skipped {
			fmt.Fprintf(&b, "%s: %s\n", oneLine(s.Endpoint), oneLine(s.Reason))
		}
	}
	if len(doc.Warnings) > 0 {
		textHeading(&b, "Warnings")
		for _, warning := range doc.Warnings {
			fmt.Fprintf(&b, "- %s\n", oneLine(warning))
		}
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func writeTextEndpoint(b *strings.Builder, e *endpoint) {
	textHeading(b, oneLine(e.Method+" "+e.Path))
	if e.Summary != "" {
		fmt.Fprintf(b, "%s\n\n", oneLine(e.Summary))
	}
	fmt.Fprintf(b, "HTTP verb: %s\nPath: %s\n", oneLine(e.Method), oneLine(e.FullPath))
	if e.OperationID != "" {
		fmt.Fprintf(b, "Operation ID: %s\n", oneLine(e.OperationID))
	}

	b.WriteString("\nParameters:\n")
	if len(e.Parameters) == 0 {
		b.WriteString("  None.\n")
	}
	for _, p := range e.Parameters {
		required := "optional"
		if p.Required {
			required = "required"
		}
		fmt.Fprintf(b, "  %s (%s): %s, %s", oneLine(p.Name), oneLine(p.In), oneLine(p.Type), required)
		if p.Constraints != "" {
			fmt.Fprintf(b, ", %s", oneLine(p.Constraints))
		}
		b.WriteByte('\n')
	}

	res := &e.Results
	fmt.Fprintf(b, "\nResults: %d sent, %d reported (filter: %s). Status codes: %s.\nResponse time: %s.\n",
		res.Sent, res.Reported, oneLine(res.Filter), countsLine(res.StatusCodes), res.ResponseTime.text())
	for _, item := range res.Items {
		fmt.Fprintf(b, "\n  %s %s (%s)\n    Message: %s\n    Input: %s\n",
			statusLabel(int(item.Status)), oneLine(item.Endpoint), millisText(item.DurationMs), oneLine(item.Message), oneLine(item.Input))
	}
}

func textHeading(b *strings.Builder, title string) {
	fmt.Fprintf(b, "\n%s\n%s\n", title, strings.Repeat("-", len([]rune(title))))
}
