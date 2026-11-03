package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"fuzzy/internal/openapi"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	spec, err := marshalSpec()
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(newHandler(spec))
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

func TestCRUDLifecycle(t *testing.T) {
	ts := newTestServer(t)

	status, created := call(t, ts, "POST", "/items", `{"name":"a","price":1.5}`)
	if status != http.StatusCreated {
		t.Fatalf("create: status %d", status)
	}
	want := map[string]any{"id": 1.0, "name": "a", "description": nil, "price": 1.5, "quantity": 0.0}
	if fmt.Sprint(created) != fmt.Sprint(want) {
		t.Fatalf("create: got %v, want %v", created, want)
	}

	if status, got := call(t, ts, "GET", "/items/1", ""); status != http.StatusOK || fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("get: status %d, body %v", status, got)
	}

	status, updated := call(t, ts, "PUT", "/items/1", `{"name":"b","description":"d","price":2,"quantity":3}`)
	want = map[string]any{"id": 1.0, "name": "b", "description": "d", "price": 2.0, "quantity": 3.0}
	if status != http.StatusOK || fmt.Sprint(updated) != fmt.Sprint(want) {
		t.Fatalf("update: status %d, body %v", status, updated)
	}

	if status, _ := call(t, ts, "DELETE", "/items/1", ""); status != http.StatusNoContent {
		t.Fatalf("delete: status %d", status)
	}
	for _, c := range []struct{ method, body string }{
		{"GET", ""}, {"DELETE", ""}, {"PUT", `{"name":"a","price":1}`},
	} {
		if status, _ := call(t, ts, c.method, "/items/1", c.body); status != http.StatusNotFound {
			t.Errorf("%s after delete: status %d", c.method, status)
		}
	}
	if status, _ := call(t, ts, "POST", "/items", `{"name":"a","price":1}`); status != http.StatusCreated {
		t.Fatalf("create after delete: status %d", status)
	}
	if _, got := call(t, ts, "GET", "/items/2", ""); got.(map[string]any)["id"] != 2.0 {
		t.Errorf("IDs must not be reused, got %v", got)
	}
}

func TestListPagination(t *testing.T) {
	ts := newTestServer(t)
	for range 25 {
		call(t, ts, "POST", "/items", `{"name":"a","price":1}`)
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
	}
	for _, tt := range tests {
		status, got := call(t, ts, "GET", "/items"+tt.query, "")
		page, ok := got.([]any)
		if status != http.StatusOK || !ok || len(page) != tt.count {
			t.Errorf("%q: status %d, body %v, want %d items", tt.query, status, got, tt.count)
			continue
		}
		if tt.count > 0 && page[0].(map[string]any)["id"] != tt.firstID {
			t.Errorf("%q: first item %v, want id %v", tt.query, page[0], tt.firstID)
		}
	}
}

func TestValidationErrors(t *testing.T) {
	ts := newTestServer(t)
	call(t, ts, "POST", "/items", `{"name":"a","price":1}`)
	long := func(n int) string { return strings.Repeat("é", n) }

	tests := []struct {
		name, method, path, body string
		wantType                 string
		wantLoc                  string
	}{
		{"empty body", "POST", "/items", ``, "missing", "[body]"},
		{"null body", "POST", "/items", `null`, "model_attributes_type", "[body]"},
		{"malformed body", "POST", "/items", `{`, "json_invalid", "[body]"},
		{"array body", "POST", "/items", `[]`, "model_attributes_type", "[body]"},
		{"missing name", "POST", "/items", `{"price":1}`, "missing", "[body name]"},
		{"null name", "POST", "/items", `{"name":null,"price":1}`, "string_type", "[body name]"},
		{"numeric name", "POST", "/items", `{"name":1,"price":1}`, "string_type", "[body name]"},
		{"empty name", "POST", "/items", `{"name":"","price":1}`, "string_too_short", "[body name]"},
		{"long name", "POST", "/items", fmt.Sprintf(`{"name":%q,"price":1}`, long(65)), "string_too_long", "[body name]"},
		{"long description", "POST", "/items", fmt.Sprintf(`{"name":"a","description":%q,"price":1}`, long(257)), "string_too_long", "[body description]"},
		{"missing price", "POST", "/items", `{"name":"a"}`, "missing", "[body price]"},
		{"null price", "POST", "/items", `{"name":"a","price":null}`, "float_type", "[body price]"},
		{"zero price", "POST", "/items", `{"name":"a","price":0}`, "greater_than", "[body price]"},
		{"large price", "POST", "/items", `{"name":"a","price":10000.5}`, "less_than_equal", "[body price]"},
		{"null quantity", "POST", "/items", `{"name":"a","price":1,"quantity": null }`, "int_type", "[body quantity]"},
		{"fractional quantity", "POST", "/items", `{"name":"a","price":1,"quantity":1.5}`, "int_type", "[body quantity]"},
		{"negative quantity", "POST", "/items", `{"name":"a","price":1,"quantity":-1}`, "greater_than_equal", "[body quantity]"},
		{"large quantity", "POST", "/items", `{"name":"a","price":1,"quantity":1001}`, "less_than_equal", "[body quantity]"},
		{"limit zero", "GET", "/items?limit=0", ``, "greater_than_equal", "[query limit]"},
		{"limit large", "GET", "/items?limit=101", ``, "less_than_equal", "[query limit]"},
		{"limit empty", "GET", "/items?limit=", ``, "int_parsing", "[query limit]"},
		{"offset negative", "GET", "/items?offset=-1", ``, "greater_than_equal", "[query offset]"},
		{"id zero", "GET", "/items/0", ``, "greater_than", "[path item_id]"},
		{"id text", "DELETE", "/items/abc", ``, "int_parsing", "[path item_id]"},
		{"validation wins over 404", "PUT", "/items/999", `{}`, "missing", "[body name]"},
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
		req, err := http.NewRequest("POST", ts.URL+"/items", strings.NewReader(`{"name":"a","price":1}`))
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
	body := fmt.Sprintf(`{"name":%q,"description":%q,"price":%v,"quantity":%d}`,
		strings.Repeat("é", nameMaxLen), strings.Repeat("é", descriptionMaxLen), priceMax, quantityMax)
	if status, got := call(t, ts, "POST", "/items", body); status != http.StatusCreated {
		t.Errorf("status %d, body %v", status, got)
	}
}

func TestPutReportsPathAndBodyErrorsTogether(t *testing.T) {
	ts := newTestServer(t)
	_, got := call(t, ts, "PUT", "/items/0", `{"name":"a"}`)
	if detail := got.(map[string]any)["detail"].([]any); len(detail) != 2 {
		t.Errorf("got %v, want the id and price errors", detail)
	}
}

func TestConcurrentWrites(t *testing.T) {
	ts := newTestServer(t)
	const writers = 50

	var wg sync.WaitGroup
	for range writers {
		wg.Go(func() {
			req, _ := http.NewRequest("POST", ts.URL+"/items", strings.NewReader(`{"name":"a","price":1}`))
			resp, err := ts.Client().Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			resp.Body.Close()
		})
	}
	wg.Wait()

	_, got := call(t, ts, "GET", "/items?limit=100", "")
	page := got.([]any)
	if len(page) != writers {
		t.Fatalf("got %d items, want %d", len(page), writers)
	}
	for i, item := range page {
		if id := item.(map[string]any)["id"]; id != float64(i+1) {
			t.Fatalf("item %d has id %v: IDs must be unique and listed in order", i, id)
		}
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
