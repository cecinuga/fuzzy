package report

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"time"

	"fuzzy/internal/fuzzer"
	"fuzzy/internal/openapi"
)

// document is the content of a report. Every format renders this same value, so
// they can't drift apart in structure or information. Its strings are raw: each
// renderer makes them safe for its own format.
//
// Lists are never nil and every key is always present, so that a JSON consumer
// sees the same shape whatever happened in the run.
type document struct {
	Interrupted bool              `json:"interrupted"`
	Overview    overview          `json:"overview"`
	Summary     []summaryRow      `json:"summary"`
	Endpoints   []endpoint        `json:"endpoints"`
	Skipped     []skippedEndpoint `json:"skipped"`
	Warnings    []string          `json:"warnings"`
}

type overview struct {
	Target     string       `json:"target"`
	Spec       specInfo     `json:"spec"`
	Dictionary string       `json:"dictionary"` // empty when values are only random
	Bounds     randomBounds `json:"randomBounds"`
	Charset    string       `json:"charset"`
	Filter     string       `json:"statusFilter"`
	Selection  selection    `json:"selection"`
	// A string in JSON: seeds use all 64 bits, and many JSON readers keep numbers
	// as doubles, which would round the seed and so fail to reproduce the run.
	Seed            uint64 `json:"seed,string"`
	Started         string `json:"started"`
	Duration        string `json:"duration"`
	RequestsSent    int    `json:"requestsSent"`
	ResultsReported int    `json:"resultsReported"`
}

// randomBounds holds the limits of random values for each data type. For strings
// they limit the length.
type randomBounds struct {
	String bound[int]     `json:"string"`
	Int    bound[int64]   `json:"int"`
	Float  bound[float64] `json:"float"`
}

type bound[T int | int64 | float64] struct {
	Min T `json:"min"`
	Max T `json:"max"`
}

// text renders the bound for Markdown with the same shape it has in JSON.
func (b bound[T]) text() string {
	return fmt.Sprintf("{min: %v, max: %v}", b.Min, b.Max)
}

// lines renders the bounds of each data type as text, one entry per type.
func (b randomBounds) lines() []string {
	return []string{"string: " + b.String.text(), "int: " + b.Int.text(), "float: " + b.Float.text()}
}

// selection says which endpoints the run was limited to, so that a partial run
// can't be mistaken for one that covered the whole spec.
type selection struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
	LeftOut int      `json:"endpointsLeftOut"`
}

type specInfo struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Version string `json:"version"`
}

type summaryRow struct {
	Endpoint     string       `json:"endpoint"`
	Sent         int          `json:"sent"`
	Reported     int          `json:"reported"`
	StatusCodes  statusCounts `json:"statusCodes"`
	ResponseTime responseTime `json:"responseTimeMs"`
}

// responseTime summarizes an endpoint's response times in milliseconds. It covers
// every response, whatever the status filter keeps, and only responses: a request
// without one would report how long the timeout is, not how the API performs.
type responseTime struct {
	Count int     `json:"count"`
	Min   float64 `json:"min"`
	Mean  float64 `json:"mean"`
	P50   float64 `json:"p50"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
	Max   float64 `json:"max"`
}

func newResponseTime(l *fuzzer.Latency) responseTime {
	return responseTime{
		Count: l.Count,
		Min:   millis(l.Min),
		Mean:  millis(l.Mean()),
		P50:   millis(l.Percentile(50)),
		P95:   millis(l.Percentile(95)),
		P99:   millis(l.Percentile(99)),
		Max:   millis(l.Max),
	}
}

// text renders every figure, for the formats meant to be read.
func (t responseTime) text() string {
	if t.Count == 0 {
		return "no responses"
	}
	noun := "responses"
	if t.Count == 1 {
		noun = "response"
	}
	return fmt.Sprintf("min %s, mean %s, p50 %s, p95 %s, p99 %s, max %s (%d %s)",
		millisText(t.Min), millisText(t.Mean), millisText(t.P50), millisText(t.P95), millisText(t.P99), millisText(t.Max), t.Count, noun)
}

// short renders the figures that fit a summary table.
func (t responseTime) short() string {
	if t.Count == 0 {
		return "no responses"
	}
	return fmt.Sprintf("mean %s, p95 %s, max %s", millisText(t.Mean), millisText(t.P95), millisText(t.Max))
}

// millis converts a duration to milliseconds, to the microsecond.
func millis(d time.Duration) float64 {
	return math.Round(float64(d)/float64(time.Microsecond)) / 1000
}

func millisText(ms float64) string {
	return number(ms) + " ms"
}

// number renders a value the way JSON does, without an exponent.
func number(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

type endpoint struct {
	Method      string      `json:"method"`
	Path        string      `json:"path"`
	FullPath    string      `json:"fullPath"`
	Summary     string      `json:"summary"`
	OperationID string      `json:"operationId"`
	Parameters  []parameter `json:"parameters"`
	Results     results     `json:"results"`
}

type parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Constraints string `json:"constraints"`
}

type results struct {
	Sent         int          `json:"sent"`
	Reported     int          `json:"reported"`
	Filter       string       `json:"filter"`
	StatusCodes  statusCounts `json:"statusCodes"`
	ResponseTime responseTime `json:"responseTimeMs"`
	Items        []result     `json:"items"`
}

type result struct {
	Endpoint string     `json:"endpoint"`
	Status   statusCode `json:"status"`
	Message  string     `json:"message"`
	Input    string     `json:"input"`
	// The time the call took, also for a request without a response: how long it
	// took to fail is what tells a timeout from a refused connection.
	DurationMs float64 `json:"durationMs"`
}

type skippedEndpoint struct {
	Endpoint string `json:"endpoint"`
	Reason   string `json:"reason"`
}

// statusCode is an HTTP status, or fuzzer.TransportError when no response came.
type statusCode int

// MarshalJSON writes the code as a number, and null when there was no response:
// 0 would read as a status, and "ERR" would make the field's type vary.
func (s statusCode) MarshalJSON() ([]byte, error) {
	if s == fuzzer.TransportError {
		return []byte("null"), nil
	}
	return strconv.AppendInt(nil, int64(s), 10), nil
}

// statusCounts holds how many requests got each status.
type statusCounts map[int]int

// MarshalJSON writes an object keyed by status label, e.g. {"404":83,"ERR":2}.
func (c statusCounts) MarshalJSON() ([]byte, error) {
	labeled := make(map[string]int, len(c))
	for status, count := range c {
		labeled[statusLabel(status)] = count
	}
	return json.Marshal(labeled)
}

func newDocument(r Report) document {
	doc := document{
		Interrupted: r.Interrupted,
		Summary:     make([]summaryRow, len(r.Outcomes)),
		Endpoints:   make([]endpoint, len(r.Outcomes)),
		Skipped:     make([]skippedEndpoint, len(r.Skipped)),
		Warnings:    append([]string{}, r.Warnings...),
	}

	sent, kept := 0, 0
	for i := range r.Outcomes {
		o := &r.Outcomes[i]
		sent += o.Sent
		kept += len(o.Results)
		doc.Summary[i] = summaryRow{
			Endpoint:     o.Target.Op.Method + " " + o.Target.Op.Path,
			Sent:         o.Sent,
			Reported:     len(o.Results),
			StatusCodes:  o.Counts,
			ResponseTime: newResponseTime(&o.Latency),
		}
		doc.Endpoints[i] = newEndpoint(o, r.Filter)
	}
	for i, s := range r.Skipped {
		doc.Skipped[i] = skippedEndpoint{Endpoint: s.Op.Method + " " + s.Op.Path, Reason: s.Reason}
	}

	doc.Overview = overview{
		Target:     r.Remote,
		Spec:       specInfo{Path: r.Spec, Title: r.SpecTitle, Version: r.SpecVersion},
		Dictionary: r.Dictionary,
		Bounds: randomBounds{
			String: bound[int]{r.Bounds.StringMin, r.Bounds.StringMax},
			Int:    bound[int64]{r.Bounds.IntMin, r.Bounds.IntMax},
			Float:  bound[float64]{r.Bounds.FloatMin, r.Bounds.FloatMax},
		},
		Charset:         string(r.Bounds.Charset),
		Filter:          r.Filter.String(),
		Selection:       selection{r.Selection.Include(), r.Selection.Exclude(), r.LeftOut},
		Seed:            r.Seed,
		Started:         r.Started.Format(time.RFC3339),
		Duration:        r.Duration.Round(time.Millisecond).String(),
		RequestsSent:    sent,
		ResultsReported: kept,
	}
	return doc
}

func newEndpoint(o *fuzzer.Outcome, filter fuzzer.StatusFilter) endpoint {
	op := o.Target.Op
	items := make([]result, len(o.Results))
	for i, res := range o.Results {
		items[i] = result{
			Endpoint: res.Endpoint, Status: statusCode(res.Status), Message: res.Message, Input: res.Input,
			DurationMs: millis(res.Duration),
		}
	}
	return endpoint{
		Method:      op.Method,
		Path:        op.Path,
		FullPath:    op.BasePath + op.Path,
		Summary:     op.Summary,
		OperationID: op.OperationID,
		Parameters:  parametersOf(op),
		Results: results{
			Sent:         o.Sent,
			Reported:     len(o.Results),
			Filter:       filter.String(),
			StatusCodes:  o.Counts,
			ResponseTime: newResponseTime(&o.Latency),
			Items:        items,
		},
	}
}

// parametersOf lists the parameters of op, followed by each body and its fields.
func parametersOf(op *openapi.Operation) []parameter {
	params := []parameter{}
	for _, p := range op.Parameters {
		params = append(params, parameter{p.Name, p.In, typeName(p.Schema, 0), p.Required, constraints(p.Schema)})
	}
	if op.RequestBody != nil {
		for _, contentType := range slices.Sorted(maps.Keys(op.RequestBody.Content)) {
			schema := op.RequestBody.Content[contentType].Schema
			params = append(params, parameter{"(body)", contentType, typeName(schema, 0), op.RequestBody.Required, constraints(schema)})
			flattenProperties(schema, "", map[*openapi.Schema]bool{}, func(name string, s *openapi.Schema, required bool) {
				params = append(params, parameter{name, contentType, typeName(s, 0), required, constraints(s)})
			})
		}
	}
	return params
}
