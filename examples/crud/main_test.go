package main

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"fuzzy/internal/openapi"
)

// The seed in db.json, as the tests rely on it.
const (
	seedCount   = 5
	draftID     = "1"
	sentID      = "2"
	paidID      = "3"
	voidID      = "4"
	newestSeed  = 5.0
	lastNumber  = 3.0
	invoicePath = "/invoices/"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	spec, err := marshalSpec()
	if err != nil {
		t.Fatal(err)
	}
	s, err := newServer(seedData)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(newHandler(spec, s))
	t.Cleanup(ts.Close)
	return ts
}

// call sends a request and decodes the JSON response, if there is one.
func call(t *testing.T, ts *httptest.Server, method, path, body string) (int, any) {
	t.Helper()
	req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded any
	if resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
			t.Fatalf("%s %s: decode response: %v", method, path, err)
		}
	}
	return resp.StatusCode, decoded
}

func validFields() map[string]any {
	return map[string]any{
		"customer":   "a",
		"currency":   "EUR",
		"issue_date": "2026-01-01",
		"due_date":   "2026-01-31",
		"lines":      []any{map[string]any{"description": "d", "quantity": 1, "unit_price": 100}},
	}
}

func encode(t *testing.T, fields map[string]any) string {
	t.Helper()
	data, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// bodyWith is a valid body with some fields replaced.
func bodyWith(t *testing.T, changes map[string]any) string {
	t.Helper()
	fields := validFields()
	maps.Copy(fields, changes)
	return encode(t, fields)
}

func bodyWithout(t *testing.T, name string) string {
	t.Helper()
	fields := validFields()
	delete(fields, name)
	return encode(t, fields)
}

// bodyWithLine is a valid body whose only line has one field replaced.
func bodyWithLine(t *testing.T, name string, value any) string {
	t.Helper()
	line := map[string]any{"description": "d", "quantity": 1, "unit_price": 100}
	line[name] = value
	return bodyWith(t, map[string]any{"lines": []any{line}})
}

func field(t *testing.T, invoice any, name string) any {
	t.Helper()
	fields, ok := invoice.(map[string]any)
	if !ok {
		t.Fatalf("got %v, want an object", invoice)
	}
	return fields[name]
}

func TestCRUDLifecycle(t *testing.T) {
	ts := newTestServer(t)

	status, created := call(t, ts, "POST", "/invoices", bodyWith(t, map[string]any{"tax_rate": 10}))
	if status != http.StatusCreated {
		t.Fatalf("create: status %d, body %v", status, created)
	}
	id := field(t, created, "id")
	if id != newestSeed+1 {
		t.Fatalf("create: id %v must follow the seed", id)
	}
	if got := field(t, created, "status"); got != statusDraft {
		t.Errorf("create: status %v, want draft", got)
	}
	if got := field(t, created, "number"); got != nil {
		t.Errorf("create: number %v, want none until sent", got)
	}
	path := fmt.Sprintf("%s%v", invoicePath, id)

	if status, got := call(t, ts, "GET", path, ""); status != http.StatusOK || fmt.Sprint(got) != fmt.Sprint(created) {
		t.Fatalf("get: status %d, body %v", status, got)
	}

	status, updated := call(t, ts, "PUT", path, bodyWith(t, map[string]any{"customer": "b", "notes": "n"}))
	if status != http.StatusOK || field(t, updated, "customer") != "b" || field(t, updated, "notes") != "n" || field(t, updated, "id") != id {
		t.Fatalf("update: status %d, body %v", status, updated)
	}

	if status, _ := call(t, ts, "DELETE", path, ""); status != http.StatusNoContent {
		t.Fatalf("delete: status %d", status)
	}
	for _, c := range []struct{ method, path, body string }{
		{"GET", path, ""},
		{"DELETE", path, ""},
		{"PUT", path, bodyWith(t, nil)},
		{"PUT", path + "/status", `{"status":"sent"}`},
	} {
		if status, _ := call(t, ts, c.method, c.path, c.body); status != http.StatusNotFound {
			t.Errorf("%s %s after delete: status %d", c.method, c.path, status)
		}
	}
	_, next := call(t, ts, "POST", "/invoices", bodyWith(t, nil))
	if got := field(t, next, "id"); got != newestSeed+2 {
		t.Errorf("IDs must not be reused, got %v", got)
	}
}

func TestSeedIsServed(t *testing.T) {
	ts := newTestServer(t)

	_, got := call(t, ts, "GET", "/invoices", "")
	if page := got.([]any); len(page) != seedCount {
		t.Fatalf("got %d invoices, want %d", len(page), seedCount)
	}

	// Totals aren't in db.json: 2 × 49900 = 99800, and 8.25% of it is 8233.5.
	_, paid := call(t, ts, "GET", invoicePath+paidID, "")
	for name, want := range map[string]any{
		"status": statusPaid, "number": 2.0, "subtotal": 99800.0, "tax": 8234.0, "total": 108034.0,
	} {
		if got := field(t, paid, name); got != want {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
}

func TestSeedValidation(t *testing.T) {
	invoice := func(id int, status, number, dueDate string) string {
		return fmt.Sprintf(`{"id":%d,"status":%q,"number":%s,"customer":"a","currency":"EUR","issue_date":"2026-01-10",`+
			`"due_date":%q,"lines":[{"description":"d","quantity":1,"unit_price":1}]}`, id, status, number, dueDate)
	}
	const okDate = "2026-01-20"

	tests := []struct {
		name     string
		invoices []string
	}{
		{"id zero", []string{invoice(0, statusDraft, "null", okDate)}},
		{"duplicate id", []string{invoice(1, statusDraft, "null", okDate), invoice(1, statusDraft, "null", okDate)}},
		{"duplicate number", []string{invoice(1, statusSent, "1", okDate), invoice(2, statusPaid, "1", okDate)}},
		{"unknown status", []string{invoice(1, "late", "null", okDate)}},
		{"sent without a number", []string{invoice(1, statusSent, "null", okDate)}},
		{"draft with a number", []string{invoice(1, statusDraft, "1", okDate)}},
		{"due before issue", []string{invoice(1, statusDraft, "null", "2026-01-01")}},
		{"invalid field", []string{`{"id":1,"status":"draft","number":null}`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seed := fmt.Sprintf(`{"invoices":[%s]}`, strings.Join(tt.invoices, ","))
			if _, err := newServer([]byte(seed)); err == nil {
				t.Error("want an error")
			}
		})
	}

	if _, err := newServer([]byte(`{`)); err == nil {
		t.Error("malformed seed: want an error")
	}
	s, err := newServer([]byte(fmt.Sprintf(`{"invoices":[%s]}`, invoice(7, statusSent, "4", okDate))))
	if err != nil {
		t.Fatal(err)
	}
	if s.nextID != 7 || s.nextNumber != 4 {
		t.Errorf("counters at %d and %d, want them to continue from the seed", s.nextID, s.nextNumber)
	}
}

func TestAmounts(t *testing.T) {
	ts := newTestServer(t)
	body := bodyWith(t, map[string]any{
		"tax_rate": 22,
		"lines": []any{
			map[string]any{"description": "a", "quantity": 8, "unit_price": 7500},
			map[string]any{"description": "b", "quantity": 1, "unit_price": 2500},
		},
		// What the client says about these fields must be ignored.
		"id": 999, "status": statusPaid, "number": 999, "subtotal": 1, "tax": 1, "total": 1,
	})
	_, got := call(t, ts, "POST", "/invoices", body)
	for name, want := range map[string]any{
		"id": newestSeed + 1, "status": statusDraft, "number": nil, "subtotal": 62500.0, "tax": 13750.0, "total": 76250.0,
	} {
		if value := field(t, got, name); value != want {
			t.Errorf("%s: got %v, want %v", name, value, want)
		}
	}

	_, updated := call(t, ts, "PUT", invoicePath+draftID, bodyWith(t, nil))
	if got := field(t, updated, "total"); got != 100.0 {
		t.Errorf("update must recompute the amounts, got total %v", got)
	}
}

func TestDueDateRule(t *testing.T) {
	ts := newTestServer(t)
	early := bodyWith(t, map[string]any{"issue_date": "2026-02-01", "due_date": "2026-01-31"})
	same := bodyWith(t, map[string]any{"issue_date": "2026-02-01", "due_date": "2026-02-01"})

	if status, got := call(t, ts, "POST", "/invoices", early); status != http.StatusConflict {
		t.Errorf("create: status %d, body %v", status, got)
	}
	if status, got := call(t, ts, "PUT", invoicePath+draftID, early); status != http.StatusConflict {
		t.Errorf("update: status %d, body %v", status, got)
	}
	if status, got := call(t, ts, "PUT", "/invoices/999", early); status != http.StatusNotFound {
		t.Errorf("update of a missing invoice: status %d, body %v", status, got)
	}
	if status, got := call(t, ts, "POST", "/invoices", same); status != http.StatusCreated {
		t.Errorf("due on the issue date: status %d, body %v", status, got)
	}
}

func TestOnlyDraftsChange(t *testing.T) {
	ts := newTestServer(t)
	for _, id := range []string{sentID, paidID, voidID} {
		_, before := call(t, ts, "GET", invoicePath+id, "")
		if status, got := call(t, ts, "PUT", invoicePath+id, bodyWith(t, nil)); status != http.StatusConflict {
			t.Errorf("update %s: status %d, body %v", id, status, got)
		}
		if status, got := call(t, ts, "DELETE", invoicePath+id, ""); status != http.StatusConflict {
			t.Errorf("delete %s: status %d, body %v", id, status, got)
		}
		_, after := call(t, ts, "GET", invoicePath+id, "")
		if fmt.Sprint(before) != fmt.Sprint(after) {
			t.Errorf("invoice %s changed: %v became %v", id, before, after)
		}
	}
}

func TestStatusTransitions(t *testing.T) {
	seeded := map[string]string{statusDraft: draftID, statusSent: sentID, statusPaid: paidID, statusVoid: voidID}
	allowed := map[string]bool{
		statusDraft + ">" + statusSent: true,
		statusSent + ">" + statusPaid:  true,
		statusSent + ">" + statusVoid:  true,
	}
	for from, id := range seeded {
		for _, to := range targetStatuses {
			t.Run(from+" to "+to, func(t *testing.T) {
				ts := newTestServer(t)
				status, got := call(t, ts, "PUT", invoicePath+id+"/status", fmt.Sprintf(`{"status":%q}`, to))
				want := http.StatusConflict
				if allowed[from+">"+to] {
					want = http.StatusOK
				}
				if status != want {
					t.Fatalf("status %d, want %d, body %v", status, want, got)
				}
				_, stored := call(t, ts, "GET", invoicePath+id, "")
				wantStatus := from
				if want == http.StatusOK {
					wantStatus = to
				}
				if field(t, stored, "status") != wantStatus {
					t.Errorf("stored status %v, want %v", field(t, stored, "status"), wantStatus)
				}
			})
		}
	}
}

func TestNumberIsAssignedWhenSent(t *testing.T) {
	ts := newTestServer(t)

	// A draft deleted along the way must not leave a gap in the numbering.
	_, skipped := call(t, ts, "POST", "/invoices", bodyWith(t, nil))
	call(t, ts, "DELETE", fmt.Sprintf("%s%v", invoicePath, field(t, skipped, "id")), "")

	for i, id := range []string{"5", draftID} {
		_, sent := call(t, ts, "PUT", invoicePath+id+"/status", `{"status":"sent"}`)
		if got, want := field(t, sent, "number"), lastNumber+float64(i+1); got != want {
			t.Errorf("invoice %s: number %v, want %v", id, got, want)
		}
	}

	// Later transitions keep the number.
	_, paid := call(t, ts, "PUT", invoicePath+draftID+"/status", `{"status":"paid"}`)
	if got := field(t, paid, "number"); got != lastNumber+2 {
		t.Errorf("number %v after paying", got)
	}
}

func TestListPagination(t *testing.T) {
	ts := newTestServer(t)
	for range 25 - seedCount {
		call(t, ts, "POST", "/invoices", bodyWith(t, nil))
	}

	tests := []struct {
		query   string
		count   int
		firstID float64
	}{
		{"", limitDefault, 1},
		{"?limit=5", 5, 1},
		{"?limit=5&offset=3", 5, 4},
		{"?limit=100&offset=20", 5, 21},
		{"?offset=25", 0, 0},
		{"?offset=9223372036854775807&limit=100", 0, 0},
		{"?limit=1&limit=2", 2, 1},
		{"?status=paid", 1, 3},
		{"?status=draft&limit=2&offset=1", 2, 5},
		{"?status=sent&offset=1", 0, 0},
	}
	for _, tt := range tests {
		status, got := call(t, ts, "GET", "/invoices"+tt.query, "")
		page, ok := got.([]any)
		if status != http.StatusOK || !ok || len(page) != tt.count {
			t.Errorf("%q: status %d, body %v, want %d invoices", tt.query, status, got, tt.count)
			continue
		}
		if tt.count > 0 && page[0].(map[string]any)["id"] != tt.firstID {
			t.Errorf("%q: first invoice %v, want id %v", tt.query, page[0], tt.firstID)
		}
	}
}

func TestValidationErrors(t *testing.T) {
	ts := newTestServer(t)
	long := func(n int) string { return strings.Repeat("é", n) }
	manyLines := make([]any, linesMax+1)
	for i := range manyLines {
		manyLines[i] = map[string]any{"description": "d", "quantity": 1, "unit_price": 1}
	}

	tests := []struct {
		name, method, path, body string
		wantType                 string
		wantLoc                  string
	}{
		{"empty body", "POST", "/invoices", ``, "missing", "[body]"},
		{"null body", "POST", "/invoices", `null`, "model_attributes_type", "[body]"},
		{"malformed body", "POST", "/invoices", `{`, "json_invalid", "[body]"},
		{"array body", "POST", "/invoices", `[]`, "model_attributes_type", "[body]"},
		{"missing customer", "POST", "/invoices", bodyWithout(t, "customer"), "missing", "[body customer]"},
		{"null customer", "POST", "/invoices", bodyWith(t, map[string]any{"customer": nil}), "string_type", "[body customer]"},
		{"numeric customer", "POST", "/invoices", bodyWith(t, map[string]any{"customer": 1}), "string_type", "[body customer]"},
		{"empty customer", "POST", "/invoices", bodyWith(t, map[string]any{"customer": ""}), "string_too_short", "[body customer]"},
		{"long customer", "POST", "/invoices", bodyWith(t, map[string]any{"customer": long(customerMaxLen + 1)}), "string_too_long", "[body customer]"},
		{"missing currency", "POST", "/invoices", bodyWithout(t, "currency"), "missing", "[body currency]"},
		{"unknown currency", "POST", "/invoices", bodyWith(t, map[string]any{"currency": "JPY"}), "enum", "[body currency]"},
		{"missing issue date", "POST", "/invoices", bodyWithout(t, "issue_date"), "missing", "[body issue_date]"},
		{"date with a time", "POST", "/invoices", bodyWith(t, map[string]any{"issue_date": "2026-01-01T00:00:00Z"}), "date_parsing", "[body issue_date]"},
		{"impossible date", "POST", "/invoices", bodyWith(t, map[string]any{"due_date": "2026-02-30"}), "date_parsing", "[body due_date]"},
		{"numeric date", "POST", "/invoices", bodyWith(t, map[string]any{"due_date": 20260101}), "string_type", "[body due_date]"},
		{"null tax rate", "POST", "/invoices", bodyWith(t, map[string]any{"tax_rate": nil}), "float_type", "[body tax_rate]"},
		{"negative tax rate", "POST", "/invoices", bodyWith(t, map[string]any{"tax_rate": -0.5}), "greater_than_equal", "[body tax_rate]"},
		{"large tax rate", "POST", "/invoices", bodyWith(t, map[string]any{"tax_rate": 100.5}), "less_than_equal", "[body tax_rate]"},
		{"long notes", "POST", "/invoices", bodyWith(t, map[string]any{"notes": long(notesMaxLen + 1)}), "string_too_long", "[body notes]"},
		{"numeric notes", "POST", "/invoices", bodyWith(t, map[string]any{"notes": 1}), "string_type", "[body notes]"},
		{"missing lines", "POST", "/invoices", bodyWithout(t, "lines"), "missing", "[body lines]"},
		{"null lines", "POST", "/invoices", bodyWith(t, map[string]any{"lines": nil}), "list_type", "[body lines]"},
		{"lines not a list", "POST", "/invoices", bodyWith(t, map[string]any{"lines": "x"}), "list_type", "[body lines]"},
		{"no lines", "POST", "/invoices", bodyWith(t, map[string]any{"lines": []any{}}), "too_short", "[body lines]"},
		{"too many lines", "POST", "/invoices", bodyWith(t, map[string]any{"lines": manyLines}), "too_long", "[body lines]"},
		{"line not an object", "POST", "/invoices", bodyWith(t, map[string]any{"lines": []any{1}}), "model_attributes_type", "[body lines 0]"},
		{"empty line description", "POST", "/invoices", bodyWithLine(t, "description", ""), "string_too_short", "[body lines 0 description]"},
		{"long line description", "POST", "/invoices", bodyWithLine(t, "description", long(descriptionMaxLen+1)), "string_too_long", "[body lines 0 description]"},
		{"fractional quantity", "POST", "/invoices", bodyWithLine(t, "quantity", 1.5), "int_type", "[body lines 0 quantity]"},
		{"zero quantity", "POST", "/invoices", bodyWithLine(t, "quantity", 0), "greater_than_equal", "[body lines 0 quantity]"},
		{"large quantity", "POST", "/invoices", bodyWithLine(t, "quantity", quantityMax+1), "less_than_equal", "[body lines 0 quantity]"},
		{"zero unit price", "POST", "/invoices", bodyWithLine(t, "unit_price", 0), "greater_than_equal", "[body lines 0 unit_price]"},
		{"large unit price", "POST", "/invoices", bodyWithLine(t, "unit_price", unitPriceMax+1), "less_than_equal", "[body lines 0 unit_price]"},
		{"limit zero", "GET", "/invoices?limit=0", ``, "greater_than_equal", "[query limit]"},
		{"limit large", "GET", "/invoices?limit=101", ``, "less_than_equal", "[query limit]"},
		{"limit empty", "GET", "/invoices?limit=", ``, "int_parsing", "[query limit]"},
		{"offset negative", "GET", "/invoices?offset=-1", ``, "greater_than_equal", "[query offset]"},
		{"unknown status filter", "GET", "/invoices?status=late", ``, "enum", "[query status]"},
		{"id zero", "GET", "/invoices/0", ``, "greater_than", "[path invoice_id]"},
		{"id text", "DELETE", "/invoices/abc", ``, "int_parsing", "[path invoice_id]"},
		{"missing target status", "PUT", "/invoices/1/status", `{}`, "missing", "[body status]"},
		{"back to draft", "PUT", "/invoices/1/status", `{"status":"draft"}`, "enum", "[body status]"},
		{"status body not an object", "PUT", "/invoices/1/status", `"sent"`, "model_attributes_type", "[body]"},
		{"validation wins over 404", "PUT", "/invoices/999", `{}`, "missing", "[body customer]"},
		{"validation wins over 409", "PUT", "/invoices/3", `{}`, "missing", "[body customer]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, got := call(t, ts, tt.method, tt.path, tt.body)
			if status != http.StatusUnprocessableEntity {
				t.Fatalf("status %d, body %v", status, got)
			}
			first := got.(map[string]any)["detail"].([]any)[0].(map[string]any)
			if first["type"] != tt.wantType || fmt.Sprint(first["loc"]) != tt.wantLoc {
				t.Errorf("got %v at %v, want %s at %s", first["type"], first["loc"], tt.wantType, tt.wantLoc)
			}
		})
	}
}

func TestContentType(t *testing.T) {
	tests := []struct {
		header string
		want   int
	}{
		{"", http.StatusCreated},
		{"application/json", http.StatusCreated},
		{"application/json; charset=utf-8", http.StatusCreated},
		{"application/vnd.api+json", http.StatusCreated},
		{"text/plain", http.StatusUnprocessableEntity},
		{"text/json", http.StatusUnprocessableEntity},
		{"application/x-www-form-urlencoded", http.StatusUnprocessableEntity},
		{"not a media type", http.StatusUnprocessableEntity},
	}
	ts := newTestServer(t)
	for _, tt := range tests {
		req, err := http.NewRequest("POST", ts.URL+"/invoices", strings.NewReader(bodyWith(t, nil)))
		if err != nil {
			t.Fatal(err)
		}
		if tt.header != "" {
			req.Header.Set("Content-Type", tt.header)
		}
		resp, err := ts.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != tt.want {
			t.Errorf("%q: status %d, want %d", tt.header, resp.StatusCode, tt.want)
		}
	}
}

func TestBoundsAreInclusive(t *testing.T) {
	ts := newTestServer(t)
	lines := make([]any, linesMax)
	for i := range lines {
		lines[i] = map[string]any{
			"description": strings.Repeat("é", descriptionMaxLen),
			"quantity":    quantityMax,
			"unit_price":  unitPriceMax,
		}
	}
	body := bodyWith(t, map[string]any{
		"customer": strings.Repeat("é", customerMaxLen),
		"notes":    strings.Repeat("é", notesMaxLen),
		"tax_rate": taxRateMax,
		"lines":    lines,
	})
	status, got := call(t, ts, "POST", "/invoices", body)
	if status != http.StatusCreated {
		t.Fatalf("status %d, body %v", status, got)
	}
	// The largest invoice the API allows has to fit the amounts exactly.
	want := float64(2 * linesMax * quantityMax * unitPriceMax)
	if total := field(t, got, "total"); total != want {
		t.Errorf("total %v, want %v", total, want)
	}
}

func TestPutReportsPathAndBodyErrorsTogether(t *testing.T) {
	ts := newTestServer(t)
	_, got := call(t, ts, "PUT", "/invoices/0", bodyWithout(t, "lines"))
	if detail := got.(map[string]any)["detail"].([]any); len(detail) != 2 {
		t.Errorf("got %v, want the id and lines errors", detail)
	}
}

func TestConcurrentWrites(t *testing.T) {
	ts := newTestServer(t)
	const writers = 50
	body := bodyWith(t, nil)

	var wg sync.WaitGroup
	for range writers {
		wg.Go(func() {
			req, _ := http.NewRequest("POST", ts.URL+"/invoices", strings.NewReader(body))
			resp, err := ts.Client().Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			resp.Body.Close()
		})
	}
	wg.Wait()

	_, got := call(t, ts, "GET", "/invoices?limit=100", "")
	page := got.([]any)
	if len(page) != seedCount+writers {
		t.Fatalf("got %d invoices, want %d", len(page), seedCount+writers)
	}
	for i, invoice := range page {
		if id := invoice.(map[string]any)["id"]; id != float64(i+1) {
			t.Fatalf("invoice %d has id %v: IDs must be unique and listed in order", i, id)
		}
	}
}

// Concurrent sends must never hand out the same number twice.
func TestConcurrentSends(t *testing.T) {
	ts := newTestServer(t)
	const drafts = 50
	body := bodyWith(t, nil)

	ids := make([]string, drafts)
	for i := range ids {
		_, created := call(t, ts, "POST", "/invoices", body)
		ids[i] = fmt.Sprint(field(t, created, "id"))
	}

	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Go(func() {
			req, _ := http.NewRequest("PUT", ts.URL+invoicePath+id+"/status", strings.NewReader(`{"status":"sent"}`))
			resp, err := ts.Client().Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			resp.Body.Close()
		})
	}
	wg.Wait()

	numbers := make(map[any]bool, drafts)
	for _, id := range ids {
		_, sent := call(t, ts, "GET", invoicePath+id, "")
		numbers[field(t, sent, "number")] = true
	}
	if len(numbers) != drafts {
		t.Errorf("got %d distinct numbers, want %d", len(numbers), drafts)
	}
}

// The spec has to be something Fuzzy itself accepts, with every route in it.
func TestSpecLoadsInFuzzy(t *testing.T) {
	spec, err := marshalSpec()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "nested", "openapi.json")
	if err := writeSpec(path, spec); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	doc, err := openapi.Parse(written)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(doc.OpenAPI, "3.0") {
		t.Errorf("version %s", doc.OpenAPI)
	}
	if len(doc.Operations) != len(routes) {
		t.Errorf("got %d operations, want %d", len(doc.Operations), len(routes))
	}
}

func TestOpenAPIRouteServesTheSpec(t *testing.T) {
	ts := newTestServer(t)
	status, got := call(t, ts, "GET", "/openapi.json", "")
	if status != http.StatusOK || got.(map[string]any)["openapi"] != openAPIVersion {
		t.Errorf("status %d, body %v", status, got)
	}
}
