package report

import (
	"encoding/csv"
	"io"
	"strconv"
	"strings"
)

// csvHeader names the columns as JSON names the same values. seed and interrupted
// repeat on every row because a table has nowhere else to keep them, and a partial
// run must stay recognizable once it is imported elsewhere.
var csvHeader = []string{"operation", "endpoint", "status", "message", "input", "durationMs", "seed", "interrupted"}

// writeCSV exports the results as one table, with one row per reported result. It
// is the one format that leaves out the rest of the document: the overview, the
// parameters and the totals of each endpoint don't fit in rows of results.
func writeCSV(w io.Writer, doc document) error {
	out := csv.NewWriter(w)
	if err := out.Write(csvHeader); err != nil {
		return err
	}
	seed := strconv.FormatUint(doc.Overview.Seed, 10)
	interrupted := strconv.FormatBool(doc.Interrupted)
	for i := range doc.Endpoints {
		e := &doc.Endpoints[i]
		operation := csvCell(e.Method + " " + e.Path)
		for _, item := range e.Results.Items {
			err := out.Write([]string{
				operation, csvCell(item.Endpoint), statusLabel(int(item.Status)),
				csvCell(item.Message), csvCell(item.Input), number(item.DurationMs), seed, interrupted,
			})
			if err != nil {
				return err
			}
		}
	}
	out.Flush()
	return out.Error()
}

// csvCell defuses formula injection: spreadsheets run a cell starting with one of
// these characters, and these cells hold what a server answered. The leading quote
// makes it text. Line breaks are safe, since the writer quotes them.
func csvCell(s string) string {
	s = strings.ToValidUTF8(s, "�")
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}
