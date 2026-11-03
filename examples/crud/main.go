// Command crud is a small in-memory invoice API, used as a fuzzing target for Fuzzy.
//
// At startup it loads its invoices from db.json, which it never writes back, and
// writes its OpenAPI 3.0 spec to artifacts/openapi.json.
package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"maps"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const openAPIVersion = "3.0.3"

// seedData is the starting content of the store. It is embedded so that a restart
// always goes back to the same invoices, however much a run changed them.
//
//go:embed db.json
var seedData []byte

// Bounds of the API, shared by the request validation and the spec.
const (
	customerMinLen    = 1
	customerMaxLen    = 64
	notesMaxLen       = 256
	descriptionMinLen = 1
	descriptionMaxLen = 64
	taxRateMin        = 0
	taxRateMax        = 100
	linesMin          = 1
	linesMax          = 20
	quantityMin       = 1
	quantityMax       = 1000
	unitPriceMin      = 1
	unitPriceMax      = 1000000
	limitMin          = 1
	limitMax          = 100
	limitDefault      = 20
	offsetMin         = 0
	offsetDefault     = 0
	invoiceIDAbove    = 0
)

const (
	statusDraft = "draft"
	statusSent  = "sent"
	statusPaid  = "paid"
	statusVoid  = "void"
)

var (
	currencies      = []string{"EUR", "USD", "GBP"}
	invoiceStatuses = []string{statusDraft, statusSent, statusPaid, statusVoid}
	// targetStatuses are the ones a request can ask for: nothing goes back to draft.
	targetStatuses = []string{statusSent, statusPaid, statusVoid}
	// statusTransitions is the invoice lifecycle. Paid and void are final.
	statusTransitions = map[string][]string{
		statusDraft: {statusSent},
		statusSent:  {statusPaid, statusVoid},
	}
)

type InvoiceLine struct {
	Description string `json:"description"`
	Quantity    int64  `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"` // in cents
}

// InvoiceIn is what a client can set. Dates are kept as validated YYYY-MM-DD
// strings, which compare the same way the dates do.
type InvoiceIn struct {
	Customer  string        `json:"customer"`
	Currency  string        `json:"currency"`
	IssueDate string        `json:"issue_date"`
	DueDate   string        `json:"due_date"`
	TaxRate   float64       `json:"tax_rate"` // a percentage
	Notes     *string       `json:"notes"`
	Lines     []InvoiceLine `json:"lines"`
}

// Invoice adds what only the server decides. Amounts are in cents.
type Invoice struct {
	InvoiceIn
	ID       int64  `json:"id"`
	Status   string `json:"status"`
	Number   *int64 `json:"number"` // nil until the invoice is sent
	Subtotal int64  `json:"subtotal"`
	Tax      int64  `json:"tax"`
	Total    int64  `json:"total"`
}

// newInvoice computes the amounts, so they can never disagree with the lines.
func newInvoice(in InvoiceIn, id int64, status string, number *int64) Invoice {
	var subtotal int64
	for _, line := range in.Lines {
		subtotal += line.Quantity * line.UnitPrice
	}
	tax := int64(math.Round(float64(subtotal) * in.TaxRate / 100))
	return Invoice{
		InvoiceIn: in, ID: id, Status: status, Number: number,
		Subtotal: subtotal, Tax: tax, Total: subtotal + tax,
	}
}

func dueBeforeIssue(in InvoiceIn) bool {
	return in.DueDate < in.IssueDate
}

func dueBeforeIssueMessage(in InvoiceIn) string {
	return fmt.Sprintf("due_date %s is before issue_date %s", in.DueDate, in.IssueDate)
}

// server holds the invoices. Requests run concurrently, so every access to the
// store goes through mu.
type server struct {
	mu         sync.RWMutex
	invoices   map[int64]Invoice
	nextID     int64
	nextNumber int64
}

// newServer loads the seed through the same validation as the requests, so a bad
// edit of db.json stops the server instead of breaking a rule at runtime.
func newServer(seed []byte) (*server, error) {
	var db struct {
		Invoices []json.RawMessage `json:"invoices"`
	}
	if err := json.Unmarshal(seed, &db); err != nil {
		return nil, err
	}

	s := &server{invoices: make(map[int64]Invoice, len(db.Invoices))}
	numbers := make(map[int64]bool, len(db.Invoices))
	for i, raw := range db.Invoices {
		invoice, err := parseSeedInvoice(raw)
		if err == nil {
			_, duplicate := s.invoices[invoice.ID]
			if duplicate {
				err = fmt.Errorf("duplicate id %d", invoice.ID)
			} else if invoice.Number != nil && numbers[*invoice.Number] {
				err = fmt.Errorf("duplicate number %d", *invoice.Number)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("invoice at index %d: %w", i, err)
		}
		s.invoices[invoice.ID] = invoice
		s.nextID = max(s.nextID, invoice.ID)
		if invoice.Number != nil {
			numbers[*invoice.Number] = true
			s.nextNumber = max(s.nextNumber, *invoice.Number)
		}
	}
	return s, nil
}

func parseSeedInvoice(raw json.RawMessage) (Invoice, error) {
	in, errs := parseInvoiceIn(raw)
	if len(errs) > 0 {
		return Invoice{}, fmt.Errorf("%s at %v", errs[0].Msg, errs[0].Loc)
	}
	var meta struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
		Number *int64 `json:"number"`
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		return Invoice{}, err
	}
	switch {
	case meta.ID <= invoiceIDAbove:
		return Invoice{}, fmt.Errorf("id %d is not greater than %d", meta.ID, invoiceIDAbove)
	case !slices.Contains(invoiceStatuses, meta.Status):
		return Invoice{}, fmt.Errorf("unknown status %q", meta.Status)
	case (meta.Status == statusDraft) != (meta.Number == nil):
		return Invoice{}, errors.New("a number is required on every invoice but drafts, and only there")
	case dueBeforeIssue(in):
		return Invoice{}, errors.New(dueBeforeIssueMessage(in))
	}
	return newInvoice(in, meta.ID, meta.Status, meta.Number), nil
}

// route is one operation: it registers the handler and describes it in the spec.
type route struct {
	method, path       string
	summary, operation string
	handler            func(*server, http.ResponseWriter, *http.Request)
	parameters         []any
	request            object // schema of the request body, nil if it has none
	status             int
	response           object // schema of the success response, nil if it has no content
	conflict           bool   // whether a business rule can refuse the request with a 409
}

var (
	invoiceIDParameters = []any{object{
		"name": "invoice_id", "in": "path", "required": true,
		"schema": object{"type": "integer", "minimum": invoiceIDAbove, "exclusiveMinimum": true, "title": "Invoice Id"},
	}}
	listParameters = []any{
		object{
			"name": "status", "in": "query", "required": false,
			"schema": object{"type": "string", "enum": invoiceStatuses, "title": "Status"},
		},
		object{
			"name": "limit", "in": "query", "required": false,
			"schema": object{"type": "integer", "minimum": limitMin, "maximum": limitMax, "default": limitDefault, "title": "Limit"},
		},
		object{
			"name": "offset", "in": "query", "required": false,
			"schema": object{"type": "integer", "minimum": offsetMin, "default": offsetDefault, "title": "Offset"},
		},
	}
	listResponse = object{"type": "array", "items": schemaRef("Invoice"), "title": "Response List Invoices Invoices Get"}

	routes = []route{
		{
			method: http.MethodPost, path: "/invoices", summary: "Create Invoice", operation: "create_invoice_invoices_post",
			handler: (*server).createInvoice, request: schemaRef("InvoiceIn"),
			status: http.StatusCreated, response: schemaRef("Invoice"), conflict: true,
		},
		{
			method: http.MethodGet, path: "/invoices", summary: "List Invoices", operation: "list_invoices_invoices_get",
			handler: (*server).listInvoices, parameters: listParameters, status: http.StatusOK, response: listResponse,
		},
		{
			method: http.MethodGet, path: "/invoices/{invoice_id}", summary: "Get Invoice", operation: "get_invoice_invoices__invoice_id__get",
			handler: (*server).getInvoice, parameters: invoiceIDParameters, status: http.StatusOK, response: schemaRef("Invoice"),
		},
		{
			method: http.MethodPut, path: "/invoices/{invoice_id}", summary: "Update Invoice", operation: "update_invoice_invoices__invoice_id__put",
			handler: (*server).updateInvoice, parameters: invoiceIDParameters, request: schemaRef("InvoiceIn"),
			status: http.StatusOK, response: schemaRef("Invoice"), conflict: true,
		},
		{
			method: http.MethodDelete, path: "/invoices/{invoice_id}", summary: "Delete Invoice", operation: "delete_invoice_invoices__invoice_id__delete",
			handler: (*server).deleteInvoice, parameters: invoiceIDParameters, status: http.StatusNoContent, conflict: true,
		},
		{
			method: http.MethodPut, path: "/invoices/{invoice_id}/status", summary: "Set Invoice Status", operation: "set_invoice_status_invoices__invoice_id__status_put",
			handler: (*server).setInvoiceStatus, parameters: invoiceIDParameters, request: schemaRef("StatusChange"),
			status: http.StatusOK, response: schemaRef("Invoice"), conflict: true,
		},
	}
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8000", "address to listen on")
	flag.Parse()

	s, err := newServer(seedData)
	if err != nil {
		log.Fatalf("[!] load db.json: %v", err)
	}
	spec, err := marshalSpec()
	if err != nil {
		log.Fatalf("[!] build spec: %v", err)
	}
	path := defaultSpecPath()
	if err := writeSpec(path, spec); err != nil {
		log.Fatalf("[!] write spec: %v", err)
	}
	log.Printf("[+] loaded %d invoices from db.json", len(s.invoices))
	log.Printf("[+] spec written to %s", path)
	log.Printf("[+] listening on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, newHandler(spec, s)))
}

// defaultSpecPath points at artifacts/ next to this source file, wherever the
// server is started from.
func defaultSpecPath() string {
	_, source, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(source), "artifacts", "openapi.json")
}

func writeSpec(path string, spec []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, spec, 0o644)
}

func newHandler(spec []byte, s *server) http.Handler {
	mux := http.NewServeMux()
	for _, rt := range routes {
		handler := rt.handler
		mux.HandleFunc(rt.method+" "+rt.path, func(w http.ResponseWriter, r *http.Request) {
			handler(s, w, r)
		})
	}
	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(spec)
	})
	return mux
}

func (s *server) createInvoice(w http.ResponseWriter, r *http.Request) {
	in, errs := readInvoiceIn(r)
	if len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}
	if dueBeforeIssue(in) {
		writeConflict(w, dueBeforeIssueMessage(in))
		return
	}

	s.mu.Lock()
	s.nextID++
	invoice := newInvoice(in, s.nextID, statusDraft, nil)
	s.invoices[invoice.ID] = invoice
	s.mu.Unlock()

	writeJSON(w, http.StatusCreated, invoice)
}

func (s *server) listInvoices(w http.ResponseWriter, r *http.Request) {
	status, statusErrs := parseStatusFilter(r)
	limit, limitErrs := parseLimit(r)
	offset, offsetErrs := parseOffset(r)
	if errs := slices.Concat(statusErrs, limitErrs, offsetErrs); len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	// IDs only grow, so sorting them gives the insertion order.
	ids := slices.DeleteFunc(slices.Sorted(maps.Keys(s.invoices)), func(id int64) bool {
		return status != "" && s.invoices[id].Status != status
	})
	start := min(offset, int64(len(ids)))
	end := start + min(limit, int64(len(ids))-start)
	page := make([]Invoice, 0, end-start)
	for _, id := range ids[start:end] {
		page = append(page, s.invoices[id])
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *server) getInvoice(w http.ResponseWriter, r *http.Request) {
	id, errs := parseInvoiceID(r)
	if len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	s.mu.RLock()
	invoice, ok := s.invoices[id]
	s.mu.RUnlock()
	if !ok {
		writeNotFound(w, id)
		return
	}
	writeJSON(w, http.StatusOK, invoice)
}

// updateInvoice replaces the content of a draft. The checks are ordered from the
// request alone (422) to the stored invoice (404, then 409).
func (s *server) updateInvoice(w http.ResponseWriter, r *http.Request) {
	id, idErrs := parseInvoiceID(r)
	in, bodyErrs := readInvoiceIn(r)
	if errs := slices.Concat(idErrs, bodyErrs); len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.invoices[id]
	switch {
	case !ok:
		writeNotFound(w, id)
	case current.Status != statusDraft:
		writeConflict(w, notDraftMessage(current))
	case dueBeforeIssue(in):
		writeConflict(w, dueBeforeIssueMessage(in))
	default:
		invoice := newInvoice(in, id, statusDraft, nil)
		s.invoices[id] = invoice
		writeJSON(w, http.StatusOK, invoice)
	}
}

func (s *server) deleteInvoice(w http.ResponseWriter, r *http.Request) {
	id, errs := parseInvoiceID(r)
	if len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.invoices[id]
	switch {
	case !ok:
		writeNotFound(w, id)
	case current.Status != statusDraft:
		writeConflict(w, notDraftMessage(current))
	default:
		delete(s.invoices, id)
		w.WriteHeader(http.StatusNoContent)
	}
}

// setInvoiceStatus moves an invoice along its lifecycle. Sending is what gives it
// a number, so deleted drafts never leave a gap in the numbering.
func (s *server) setInvoiceStatus(w http.ResponseWriter, r *http.Request) {
	id, idErrs := parseInvoiceID(r)
	target, bodyErrs := readStatusChange(r)
	if errs := slices.Concat(idErrs, bodyErrs); len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	invoice, ok := s.invoices[id]
	switch {
	case !ok:
		writeNotFound(w, id)
		return
	case !slices.Contains(statusTransitions[invoice.Status], target):
		writeConflict(w, fmt.Sprintf("invoice %d cannot go from %s to %s", id, invoice.Status, target))
		return
	}
	if target == statusSent {
		s.nextNumber++
		number := s.nextNumber
		invoice.Number = &number
	}
	invoice.Status = target
	s.invoices[id] = invoice
	writeJSON(w, http.StatusOK, invoice)
}

func notDraftMessage(invoice Invoice) string {
	return fmt.Sprintf("invoice %d is %s: only drafts can be changed or deleted", invoice.ID, invoice.Status)
}

// fieldError is one entry of a 422 response, shaped like the ValidationError
// schema of the spec.
type fieldError struct {
	Type  string `json:"type"`
	Loc   []any  `json:"loc"`
	Msg   string `json:"msg"`
	Input any    `json:"input"`
	Ctx   object `json:"ctx,omitempty"`
}

func parseInvoiceID(r *http.Request) (int64, []fieldError) {
	raw := r.PathValue("invoice_id")
	loc := []any{"path", "invoice_id"}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, []fieldError{intParsingError(loc, raw)}
	}
	if id <= invoiceIDAbove {
		return 0, []fieldError{{
			Type: "greater_than", Loc: loc, Input: raw,
			Msg: fmt.Sprintf("Input should be greater than %d", invoiceIDAbove),
			Ctx: object{"gt": invoiceIDAbove},
		}}
	}
	return id, nil
}

// parseStatusFilter returns "" when the query doesn't ask for a status.
func parseStatusFilter(r *http.Request) (string, []fieldError) {
	raw, ok := lastQueryValue(r, "status")
	if !ok {
		return "", nil
	}
	if !slices.Contains(invoiceStatuses, raw) {
		return "", []fieldError{enumError([]any{"query", "status"}, raw, invoiceStatuses)}
	}
	return raw, nil
}

func parseLimit(r *http.Request) (int64, []fieldError) {
	raw, ok := lastQueryValue(r, "limit")
	if !ok {
		return limitDefault, nil
	}
	loc := []any{"query", "limit"}
	limit, err := strconv.ParseInt(raw, 10, 64)
	switch {
	case err != nil:
		return 0, []fieldError{intParsingError(loc, raw)}
	case limit < limitMin:
		return 0, []fieldError{tooSmallError(loc, raw, limitMin)}
	case limit > limitMax:
		return 0, []fieldError{tooLargeError(loc, raw, limitMax)}
	}
	return limit, nil
}

func parseOffset(r *http.Request) (int64, []fieldError) {
	raw, ok := lastQueryValue(r, "offset")
	if !ok {
		return offsetDefault, nil
	}
	loc := []any{"query", "offset"}
	offset, err := strconv.ParseInt(raw, 10, 64)
	switch {
	case err != nil:
		return 0, []fieldError{intParsingError(loc, raw)}
	case offset < offsetMin:
		return 0, []fieldError{tooSmallError(loc, raw, offsetMin)}
	}
	return offset, nil
}

// lastQueryValue returns the last value of a repeated query parameter.
func lastQueryValue(r *http.Request, name string) (string, bool) {
	values := r.URL.Query()[name]
	if len(values) == 0 {
		return "", false
	}
	return values[len(values)-1], true
}

// readBody returns the raw body of a request that claims to carry JSON.
func readBody(r *http.Request) ([]byte, []fieldError) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, []fieldError{{Type: "body_read", Loc: []any{"body"}, Msg: "Unable to read the request body"}}
	}
	// A body sent as anything but JSON stays an opaque string, not an object.
	if len(data) > 0 && !isJSONContentType(r.Header.Get("Content-Type")) {
		return nil, []fieldError{notAnObjectError([]any{"body"}, string(data))}
	}
	return data, nil
}

func readInvoiceIn(r *http.Request) (InvoiceIn, []fieldError) {
	data, errs := readBody(r)
	if len(errs) > 0 {
		return InvoiceIn{}, errs
	}
	return parseInvoiceIn(data)
}

func readStatusChange(r *http.Request) (string, []fieldError) {
	data, errs := readBody(r)
	if len(errs) > 0 {
		return "", errs
	}
	fields, errs := parseBodyObject(data)
	if len(errs) > 0 {
		return "", errs
	}
	status := readField(fields, []any{"body"}, "status", true, oneOf(targetStatuses), &errs)
	return status, errs
}

// isJSONContentType also accepts a missing header, in which case the body is
// assumed to be JSON.
func isJSONContentType(header string) bool {
	if header == "" {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(header)
	if err != nil {
		return false
	}
	subtype, ok := strings.CutPrefix(mediaType, "application/")
	return ok && (subtype == "json" || strings.HasSuffix(subtype, "+json"))
}

func parseBodyObject(data []byte) (map[string]json.RawMessage, []fieldError) {
	body := bytes.TrimSpace(data)
	switch {
	case len(body) == 0:
		return nil, []fieldError{{Type: "missing", Loc: []any{"body"}, Msg: "Field required"}}
	case !json.Valid(body):
		return nil, []fieldError{{Type: "json_invalid", Loc: []any{"body"}, Msg: "JSON decode error", Input: object{}}}
	}
	return decodeObject(body, []any{"body"})
}

func decodeObject(raw json.RawMessage, loc []any) (map[string]json.RawMessage, []fieldError) {
	var fields map[string]json.RawMessage
	// null decodes into a map without an error, leaving it nil.
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, []fieldError{notAnObjectError(loc, raw)}
	}
	return fields, nil
}

// parseInvoiceIn validates the body field by field, so that a 422 lists every
// problem and a missing field stays distinct from an explicit null.
func parseInvoiceIn(data []byte) (InvoiceIn, []fieldError) {
	fields, errs := parseBodyObject(data)
	if len(errs) > 0 {
		return InvoiceIn{}, errs
	}
	body := []any{"body"}
	return InvoiceIn{
		Customer:  readField(fields, body, "customer", true, stringIn(customerMinLen, customerMaxLen), &errs),
		Currency:  readField(fields, body, "currency", true, oneOf(currencies), &errs),
		IssueDate: readField(fields, body, "issue_date", true, parseDate, &errs),
		DueDate:   readField(fields, body, "due_date", true, parseDate, &errs),
		TaxRate:   readField(fields, body, "tax_rate", false, parseTaxRate, &errs),
		Notes:     readField(fields, body, "notes", false, parseNotes, &errs),
		Lines:     readField(fields, body, "lines", true, parseLines, &errs),
	}, errs
}

// fieldParser turns the raw value of the field at loc into its Go value.
type fieldParser[T any] func(raw json.RawMessage, loc []any) (T, []fieldError)

// readField parses one field of an object into errs. An absent optional field
// keeps its zero value.
func readField[T any](fields map[string]json.RawMessage, parent []any, name string, required bool, parse fieldParser[T], errs *[]fieldError) T {
	var value T
	loc := append(slices.Clone(parent), name)
	raw, ok := fields[name]
	if !ok {
		if required {
			*errs = append(*errs, fieldError{Type: "missing", Loc: loc, Msg: "Field required"})
		}
		return value
	}
	value, fieldErrs := parse(raw, loc)
	*errs = append(*errs, fieldErrs...)
	return value
}

func stringIn(minLen, maxLen int) fieldParser[string] {
	return func(raw json.RawMessage, loc []any) (string, []fieldError) {
		value, ok := decodeField[string](raw)
		if !ok {
			return "", []fieldError{stringTypeError(loc, raw)}
		}
		length := utf8.RuneCountInString(value)
		if length < minLen {
			return "", []fieldError{{
				Type: "string_too_short", Loc: loc, Input: raw,
				Msg: fmt.Sprintf("String should have at least %d character", minLen),
				Ctx: object{"min_length": minLen},
			}}
		}
		if length > maxLen {
			return "", []fieldError{stringTooLongError(loc, raw, maxLen)}
		}
		return value, nil
	}
}

func intIn(lo, hi int) fieldParser[int64] {
	return func(raw json.RawMessage, loc []any) (int64, []fieldError) {
		value, ok := decodeField[int64](raw)
		switch {
		case !ok:
			return 0, []fieldError{{Type: "int_type", Loc: loc, Msg: "Input should be a valid integer", Input: raw}}
		case value < int64(lo):
			return 0, []fieldError{tooSmallError(loc, raw, lo)}
		case value > int64(hi):
			return 0, []fieldError{tooLargeError(loc, raw, hi)}
		}
		return value, nil
	}
}

func oneOf(allowed []string) fieldParser[string] {
	return func(raw json.RawMessage, loc []any) (string, []fieldError) {
		value, ok := decodeField[string](raw)
		if !ok {
			return "", []fieldError{stringTypeError(loc, raw)}
		}
		if !slices.Contains(allowed, value) {
			return "", []fieldError{enumError(loc, raw, allowed)}
		}
		return value, nil
	}
}

func parseDate(raw json.RawMessage, loc []any) (string, []fieldError) {
	value, ok := decodeField[string](raw)
	if !ok {
		return "", []fieldError{stringTypeError(loc, raw)}
	}
	if _, err := time.Parse(time.DateOnly, value); err != nil {
		return "", []fieldError{{Type: "date_parsing", Loc: loc, Msg: "Input should be a valid date in the format YYYY-MM-DD", Input: raw}}
	}
	return value, nil
}

func parseTaxRate(raw json.RawMessage, loc []any) (float64, []fieldError) {
	value, ok := decodeField[float64](raw)
	switch {
	case !ok:
		return 0, []fieldError{{Type: "float_type", Loc: loc, Msg: "Input should be a valid number", Input: raw}}
	case value < taxRateMin:
		return 0, []fieldError{tooSmallError(loc, raw, taxRateMin)}
	case value > taxRateMax:
		return 0, []fieldError{tooLargeError(loc, raw, taxRateMax)}
	}
	return value, nil
}

func parseNotes(raw json.RawMessage, loc []any) (*string, []fieldError) {
	if string(raw) == "null" {
		return nil, nil
	}
	value, errs := stringIn(0, notesMaxLen)(raw, loc)
	if len(errs) > 0 {
		return nil, errs
	}
	return &value, nil
}

func parseLines(raw json.RawMessage, loc []any) ([]InvoiceLine, []fieldError) {
	elements, ok := decodeField[[]json.RawMessage](raw)
	switch {
	case !ok:
		return nil, []fieldError{{Type: "list_type", Loc: loc, Msg: "Input should be a valid list", Input: raw}}
	case len(elements) < linesMin:
		return nil, []fieldError{{
			Type: "too_short", Loc: loc, Input: raw,
			Msg: fmt.Sprintf("List should have at least %d item", linesMin),
			Ctx: object{"min_length": linesMin},
		}}
	case len(elements) > linesMax:
		return nil, []fieldError{{
			Type: "too_long", Loc: loc, Input: raw,
			Msg: fmt.Sprintf("List should have at most %d items", linesMax),
			Ctx: object{"max_length": linesMax},
		}}
	}

	lines := make([]InvoiceLine, len(elements))
	var errs []fieldError
	for i, element := range elements {
		lineLoc := append(slices.Clone(loc), i)
		fields, lineErrs := decodeObject(element, lineLoc)
		if len(lineErrs) > 0 {
			errs = append(errs, lineErrs...)
			continue
		}
		lines[i] = InvoiceLine{
			Description: readField(fields, lineLoc, "description", true, stringIn(descriptionMinLen, descriptionMaxLen), &errs),
			Quantity:    readField(fields, lineLoc, "quantity", true, intIn(quantityMin, quantityMax), &errs),
			UnitPrice:   readField(fields, lineLoc, "unit_price", true, intIn(unitPriceMin, unitPriceMax), &errs),
		}
	}
	return lines, errs
}

// decodeField reports false for null too, since json.Unmarshal would silently
// accept it and leave the zero value.
func decodeField[T any](raw json.RawMessage) (T, bool) {
	var value T
	if string(raw) == "null" {
		return value, false
	}
	return value, json.Unmarshal(raw, &value) == nil
}

func notAnObjectError(loc []any, input any) fieldError {
	return fieldError{
		Type: "model_attributes_type", Loc: loc, Input: input,
		Msg: "Input should be a valid dictionary or object to extract fields from",
	}
}

func intParsingError(loc []any, input string) fieldError {
	return fieldError{
		Type: "int_parsing", Loc: loc, Input: input,
		Msg: "Input should be a valid integer, unable to parse string as an integer",
	}
}

func tooSmallError(loc []any, input any, minimum int) fieldError {
	return fieldError{
		Type: "greater_than_equal", Loc: loc, Input: input,
		Msg: fmt.Sprintf("Input should be greater than or equal to %d", minimum),
		Ctx: object{"ge": minimum},
	}
}

func tooLargeError(loc []any, input any, maximum int) fieldError {
	return fieldError{
		Type: "less_than_equal", Loc: loc, Input: input,
		Msg: fmt.Sprintf("Input should be less than or equal to %d", maximum),
		Ctx: object{"le": maximum},
	}
}

func enumError(loc []any, input any, allowed []string) fieldError {
	return fieldError{
		Type: "enum", Loc: loc, Input: input,
		Msg: "Input should be one of: " + strings.Join(allowed, ", "),
		Ctx: object{"expected": allowed},
	}
}

func stringTypeError(loc []any, input json.RawMessage) fieldError {
	return fieldError{Type: "string_type", Loc: loc, Msg: "Input should be a valid string", Input: input}
}

func stringTooLongError(loc []any, input json.RawMessage, maximum int) fieldError {
	return fieldError{
		Type: "string_too_long", Loc: loc, Input: input,
		Msg: fmt.Sprintf("String should have at most %d characters", maximum),
		Ctx: object{"max_length": maximum},
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(buf.Bytes())
}

func writeValidationErrors(w http.ResponseWriter, errs []fieldError) {
	writeJSON(w, http.StatusUnprocessableEntity, object{"detail": errs})
}

func writeNotFound(w http.ResponseWriter, id int64) {
	writeJSON(w, http.StatusNotFound, object{"detail": fmt.Sprintf("invoice %d not found", id)})
}

// writeConflict answers a request that is well formed but breaks a business rule.
func writeConflict(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusConflict, object{"detail": message})
}

// object is a JSON object of the spec.
type object = map[string]any

func marshalSpec() ([]byte, error) {
	data, err := json.MarshalIndent(buildSpec(), "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// buildSpec derives the paths from the same route table that registers the
// handlers, so the spec can't list an endpoint the server doesn't have.
func buildSpec() object {
	paths := object{}
	for _, rt := range routes {
		success := object{"description": "Successful Response"}
		if rt.response != nil {
			success["content"] = jsonContent(rt.response)
		}
		responses := object{
			strconv.Itoa(rt.status): success,
			"422": object{
				"description": "Validation Error",
				"content":     jsonContent(schemaRef("HTTPValidationError")),
			},
		}
		if rt.conflict {
			responses["409"] = object{
				"description": "Business Rule Violation",
				"content":     jsonContent(schemaRef("BusinessRuleError")),
			}
		}
		operation := object{
			"summary":     rt.summary,
			"operationId": rt.operation,
			"responses":   responses,
		}
		if rt.parameters != nil {
			operation["parameters"] = rt.parameters
		}
		if rt.request != nil {
			operation["requestBody"] = object{"required": true, "content": jsonContent(rt.request)}
		}

		item, ok := paths[rt.path].(object)
		if !ok {
			item = object{}
			paths[rt.path] = item
		}
		item[strings.ToLower(rt.method)] = operation
	}

	return object{
		"openapi": openAPIVersion,
		"info":    object{"title": "Fuzzy CRUD invoice example", "version": "2.0.0"},
		"paths":   paths,
		"components": object{"schemas": object{
			"InvoiceLine": object{
				"type": "object", "title": "InvoiceLine",
				"properties": object{
					"description": object{"type": "string", "minLength": descriptionMinLen, "maxLength": descriptionMaxLen, "title": "Description"},
					"quantity":    object{"type": "integer", "minimum": quantityMin, "maximum": quantityMax, "title": "Quantity"},
					"unit_price":  object{"type": "integer", "minimum": unitPriceMin, "maximum": unitPriceMax, "title": "Unit Price"},
				},
				"required": []string{"description", "quantity", "unit_price"},
			},
			"InvoiceIn": object{
				"type": "object", "title": "InvoiceIn",
				"properties": invoiceProperties(),
				"required":   []string{"customer", "currency", "issue_date", "due_date", "lines"},
			},
			"Invoice": object{
				"type": "object", "title": "Invoice",
				"properties": func() object {
					properties := invoiceProperties()
					properties["id"] = object{"type": "integer", "title": "Id"}
					properties["status"] = object{"type": "string", "enum": invoiceStatuses, "title": "Status"}
					properties["number"] = object{"type": "integer", "nullable": true, "title": "Number"}
					properties["subtotal"] = object{"type": "integer", "title": "Subtotal"}
					properties["tax"] = object{"type": "integer", "title": "Tax"}
					properties["total"] = object{"type": "integer", "title": "Total"}
					return properties
				}(),
				"required": []string{"customer", "currency", "issue_date", "due_date", "lines", "id", "status", "subtotal", "tax", "total"},
			},
			"StatusChange": object{
				"type": "object", "title": "StatusChange",
				"properties": object{
					"status": object{"type": "string", "enum": targetStatuses, "title": "Status"},
				},
				"required": []string{"status"},
			},
			"BusinessRuleError": object{
				"type": "object", "title": "BusinessRuleError",
				"properties": object{
					"detail": object{"type": "string", "title": "Detail"},
				},
			},
			"HTTPValidationError": object{
				"type": "object", "title": "HTTPValidationError",
				"properties": object{
					"detail": object{"type": "array", "items": schemaRef("ValidationError"), "title": "Detail"},
				},
			},
			"ValidationError": object{
				"type": "object", "title": "ValidationError",
				"properties": object{
					"loc": object{
						"type": "array", "title": "Location",
						"items": object{"anyOf": []any{object{"type": "string"}, object{"type": "integer"}}},
					},
					"msg":   object{"type": "string", "title": "Message"},
					"type":  object{"type": "string", "title": "Error Type"},
					"input": object{"title": "Input"},
					"ctx":   object{"type": "object", "title": "Context"},
				},
				"required": []string{"loc", "msg", "type"},
			},
		}},
	}
}

func invoiceProperties() object {
	return object{
		"customer":   object{"type": "string", "minLength": customerMinLen, "maxLength": customerMaxLen, "title": "Customer"},
		"currency":   object{"type": "string", "enum": currencies, "title": "Currency"},
		"issue_date": object{"type": "string", "format": "date", "title": "Issue Date"},
		"due_date":   object{"type": "string", "format": "date", "title": "Due Date"},
		"tax_rate": object{
			"type": "number", "minimum": taxRateMin, "maximum": taxRateMax, "default": 0, "title": "Tax Rate",
		},
		"notes": object{"type": "string", "maxLength": notesMaxLen, "nullable": true, "title": "Notes"},
		"lines": object{
			"type": "array", "minItems": linesMin, "maxItems": linesMax, "items": schemaRef("InvoiceLine"), "title": "Lines",
		},
	}
}

func schemaRef(name string) object {
	return object{"$ref": "#/components/schemas/" + name}
}

func jsonContent(schema object) object {
	return object{"application/json": object{"schema": schema}}
}
