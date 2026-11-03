package report

import (
	"cmp"
	"encoding/xml"
	"io"
	"maps"
	"slices"
	"strconv"
)

// writeXML renders the document with the element names of the JSON keys. Walking
// the document by hand, instead of marshaling it, keeps every element present when
// a list is empty, as JSON keeps its keys.
//
// The encoder escapes all text, and replaces what XML can't represent at all, such
// as most control characters, so a response can't break the document.
func writeXML(w io.Writer, doc document) error {
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	x := &xmlWriter{enc: xml.NewEncoder(w)}
	x.enc.Indent("", "  ")

	x.group("report", func() {
		x.text("interrupted", strconv.FormatBool(doc.Interrupted))
		x.overview(doc.Overview)
		x.group("summary", func() {
			for _, row := range doc.Summary {
				x.group("row", func() {
					x.text("endpoint", row.Endpoint)
					x.number("sent", row.Sent)
					x.number("reported", row.Reported)
					x.counts(row.StatusCodes)
					x.responseTime(row.ResponseTime)
				})
			}
		})
		x.group("endpoints", func() {
			for i := range doc.Endpoints {
				x.endpoint(&doc.Endpoints[i])
			}
		})
		x.group("skipped", func() {
			for _, s := range doc.Skipped {
				x.group("endpoint", func() {
					x.text("name", s.Endpoint)
					x.text("reason", s.Reason)
				})
			}
		})
		x.group("warnings", func() {
			for _, warning := range doc.Warnings {
				x.text("warning", warning)
			}
		})
	})
	if x.err != nil {
		return x.err
	}
	if err := x.enc.Close(); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

// xmlWriter stops at the first error, so that the walk doesn't have to check each
// element, and an element can't be left half written with others after it.
type xmlWriter struct {
	enc *xml.Encoder
	err error
}

func (x *xmlWriter) token(t xml.Token) {
	if x.err == nil {
		x.err = x.enc.EncodeToken(t)
	}
}

func (x *xmlWriter) group(name string, content func(), attributes ...xml.Attr) {
	start := xml.StartElement{Name: xml.Name{Local: name}, Attr: attributes}
	x.token(start)
	content()
	x.token(start.End())
}

func (x *xmlWriter) text(name, value string) {
	x.group(name, func() { x.token(xml.CharData(value)) })
}

func (x *xmlWriter) number(name string, value int) {
	x.text(name, strconv.Itoa(value))
}

func (x *xmlWriter) overview(o overview) {
	x.group("overview", func() {
		x.text("target", o.Target)
		x.group("spec", func() {
			x.text("path", o.Spec.Path)
			x.text("title", o.Spec.Title)
			x.text("version", o.Spec.Version)
		})
		x.text("dictionary", o.Dictionary)
		x.group("randomBounds", func() {
			x.bound("string", number(float64(o.Bounds.String.Min)), number(float64(o.Bounds.String.Max)))
			x.bound("int", strconv.FormatInt(o.Bounds.Int.Min, 10), strconv.FormatInt(o.Bounds.Int.Max, 10))
			x.bound("float", number(o.Bounds.Float.Min), number(o.Bounds.Float.Max))
		})
		x.text("charset", o.Charset)
		x.text("statusFilter", o.Filter)
		x.text("seed", strconv.FormatUint(o.Seed, 10))
		x.text("started", o.Started)
		x.text("duration", o.Duration)
		x.number("requestsSent", o.RequestsSent)
		x.number("resultsReported", o.ResultsReported)
	})
}

func (x *xmlWriter) bound(name, low, high string) {
	x.group(name, func() {
		x.text("min", low)
		x.text("max", high)
	})
}

func (x *xmlWriter) endpoint(e *endpoint) {
	x.group("endpoint", func() {
		x.text("method", e.Method)
		x.text("path", e.Path)
		x.text("fullPath", e.FullPath)
		x.text("summary", e.Summary)
		x.text("operationId", e.OperationID)
		x.group("parameters", func() {
			for _, p := range e.Parameters {
				x.group("parameter", func() {
					x.text("name", p.Name)
					x.text("in", p.In)
					x.text("type", p.Type)
					x.text("required", strconv.FormatBool(p.Required))
					x.text("constraints", p.Constraints)
				})
			}
		})
		res := &e.Results
		x.group("results", func() {
			x.number("sent", res.Sent)
			x.number("reported", res.Reported)
			x.text("filter", res.Filter)
			x.counts(res.StatusCodes)
			x.responseTime(res.ResponseTime)
			x.group("items", func() {
				for _, item := range res.Items {
					x.group("result", func() {
						x.text("endpoint", item.Endpoint)
						x.text("status", statusLabel(int(item.Status)))
						x.text("message", item.Message)
						x.text("input", item.Input)
						x.text("durationMs", number(item.DurationMs))
					})
				}
			})
		})
	})
}

// counts writes each status as an attribute, in ascending order: a status can't
// name an element, since neither "404" nor a name made of it would read well.
func (x *xmlWriter) counts(counts statusCounts) {
	x.group("statusCodes", func() {
		for _, status := range slices.SortedFunc(maps.Keys(counts), cmp.Compare[int]) {
			attribute := xml.Attr{Name: xml.Name{Local: "status"}, Value: statusLabel(status)}
			x.group("count", func() { x.token(xml.CharData(strconv.Itoa(counts[status]))) }, attribute)
		}
	})
}

func (x *xmlWriter) responseTime(t responseTime) {
	x.group("responseTimeMs", func() {
		x.number("count", t.Count)
		x.text("min", number(t.Min))
		x.text("mean", number(t.Mean))
		x.text("p50", number(t.P50))
		x.text("p95", number(t.P95))
		x.text("p99", number(t.P99))
		x.text("max", number(t.Max))
	})
}
