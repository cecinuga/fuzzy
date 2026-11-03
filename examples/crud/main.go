// Command crud is a small in-memory CRUD API, used as a fuzzing target for Fuzzy.
//
// At startup it writes its OpenAPI 3.0 spec to artifacts/openapi.json.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"maps"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

const openAPIVersion = "3.0.3"

// Bounds of the API, shared by the request validation and the spec.
const (
	nameMinLen        = 1
	nameMaxLen        = 64
	descriptionMaxLen = 256
	priceAbove        = 0.0
	priceMax          = 10000.0
	quantityMin       = 0
	quantityMax       = 1000
	limitMin          = 1
	limitMax          = 100
	limitDefault      = 20
	offsetMin         = 0
	offsetDefault     = 0
	itemIDAbove       = 0
)

type ItemIn struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Price       float64 `json:"price"`
	Quantity    int64   `json:"quantity"`
}

type Item struct {
	ItemIn
	ID int64 `json:"id"`
}

// server holds the items. Requests run concurrently, so every access to the
// store goes through mu.
type server struct {
	mu     sync.RWMutex
	items  map[int64]Item
	nextID int64
}

// route is one operation: it registers the handler and describes it in the spec.
type route struct {
	method, path       string
	summary, operation string
	handler            func(*server, http.ResponseWriter, *http.Request)
	parameters         []any
	hasBody            bool
	status             int
	response           object // schema of the success response, nil if it has no content
}

var (
	itemIDParameters = []any{object{
		"name": "item_id", "in": "path", "required": true,
		"schema": object{"type": "integer", "minimum": itemIDAbove, "exclusiveMinimum": true, "title": "Item Id"},
	}}
	listParameters = []any{
		object{
			"name": "limit", "in": "query", "required": false,
			"schema": object{"type": "integer", "minimum": limitMin, "maximum": limitMax, "default": limitDefault, "title": "Limit"},
		},
		object{
			"name": "offset", "in": "query", "required": false,
			"schema": object{"type": "integer", "minimum": offsetMin, "default": offsetDefault, "title": "Offset"},
		},
	}
	listResponse = object{"type": "array", "items": schemaRef("Item"), "title": "Response List Items Items Get"}

	routes = []route{
		{
			method: http.MethodPost, path: "/items", summary: "Create Item", operation: "create_item_items_post",
			handler: (*server).createItem, hasBody: true, status: http.StatusCreated, response: schemaRef("Item"),
		},
		{
			method: http.MethodGet, path: "/items", summary: "List Items", operation: "list_items_items_get",
			handler: (*server).listItems, parameters: listParameters, status: http.StatusOK, response: listResponse,
		},
		{
			method: http.MethodGet, path: "/items/{item_id}", summary: "Get Item", operation: "get_item_items__item_id__get",
			handler: (*server).getItem, parameters: itemIDParameters, status: http.StatusOK, response: schemaRef("Item"),
		},
		{
			method: http.MethodPut, path: "/items/{item_id}", summary: "Update Item", operation: "update_item_items__item_id__put",
			handler: (*server).updateItem, parameters: itemIDParameters, hasBody: true, status: http.StatusOK, response: schemaRef("Item"),
		},
		{
			method: http.MethodDelete, path: "/items/{item_id}", summary: "Delete Item", operation: "delete_item_items__item_id__delete",
			handler: (*server).deleteItem, parameters: itemIDParameters, status: http.StatusNoContent,
		},
	}
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8000", "address to listen on")
	flag.Parse()

	spec, err := marshalSpec()
	if err != nil {
		log.Fatalf("[!] build spec: %v", err)
	}
	path := defaultSpecPath()
	if err := writeSpec(path, spec); err != nil {
		log.Fatalf("[!] write spec: %v", err)
	}
	log.Printf("[+] spec written to %s", path)
	log.Printf("[+] listening on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, newHandler(spec)))
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

func newHandler(spec []byte) http.Handler {
	s := &server{items: make(map[int64]Item)}
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

func (s *server) createItem(w http.ResponseWriter, r *http.Request) {
	in, errs := readItemIn(r)
	if len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	s.mu.Lock()
	s.nextID++
	item := Item{ItemIn: in, ID: s.nextID}
	s.items[item.ID] = item
	s.mu.Unlock()

	writeJSON(w, http.StatusCreated, item)
}

func (s *server) listItems(w http.ResponseWriter, r *http.Request) {
	limit, limitErrs := parseLimit(r)
	offset, offsetErrs := parseOffset(r)
	if errs := slices.Concat(limitErrs, offsetErrs); len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	// IDs only grow, so sorting them gives the insertion order.
	ids := slices.Sorted(maps.Keys(s.items))
	start := min(offset, int64(len(ids)))
	end := start + min(limit, int64(len(ids))-start)
	page := make([]Item, 0, end-start)
	for _, id := range ids[start:end] {
		page = append(page, s.items[id])
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *server) getItem(w http.ResponseWriter, r *http.Request) {
	id, errs := parseItemID(r)
	if len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	s.mu.RLock()
	item, ok := s.items[id]
	s.mu.RUnlock()
	if !ok {
		writeNotFound(w, id)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *server) updateItem(w http.ResponseWriter, r *http.Request) {
	id, idErrs := parseItemID(r)
	in, bodyErrs := readItemIn(r)
	if errs := slices.Concat(idErrs, bodyErrs); len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	item := Item{ItemIn: in, ID: id}
	s.mu.Lock()
	_, ok := s.items[id]
	if ok {
		s.items[id] = item
	}
	s.mu.Unlock()
	if !ok {
		writeNotFound(w, id)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *server) deleteItem(w http.ResponseWriter, r *http.Request) {
	id, errs := parseItemID(r)
	if len(errs) > 0 {
		writeValidationErrors(w, errs)
		return
	}

	s.mu.Lock()
	_, ok := s.items[id]
	delete(s.items, id)
	s.mu.Unlock()
	if !ok {
		writeNotFound(w, id)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func parseItemID(r *http.Request) (int64, []fieldError) {
	raw := r.PathValue("item_id")
	loc := []any{"path", "item_id"}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, []fieldError{intParsingError(loc, raw)}
	}
	if id <= itemIDAbove {
		return 0, []fieldError{{
			Type: "greater_than", Loc: loc, Input: raw,
			Msg: fmt.Sprintf("Input should be greater than %d", itemIDAbove),
			Ctx: object{"gt": itemIDAbove},
		}}
	}
	return id, nil
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

func readItemIn(r *http.Request) (ItemIn, []fieldError) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return ItemIn{}, []fieldError{{Type: "body_read", Loc: []any{"body"}, Msg: "Unable to read the request body"}}
	}
	// A body sent as anything but JSON stays an opaque string, not an object.
	if len(data) > 0 && !isJSONContentType(r.Header.Get("Content-Type")) {
		return ItemIn{}, []fieldError{notAnObjectError(string(data))}
	}
	return parseItemIn(data)
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

// parseItemIn validates the body field by field, so that a 422 lists every
// problem and a missing field stays distinct from an explicit null.
func parseItemIn(data []byte) (ItemIn, []fieldError) {
	body := bytes.TrimSpace(data)
	switch {
	case len(body) == 0:
		return ItemIn{}, []fieldError{{Type: "missing", Loc: []any{"body"}, Msg: "Field required"}}
	case !json.Valid(body):
		return ItemIn{}, []fieldError{{Type: "json_invalid", Loc: []any{"body"}, Msg: "JSON decode error", Input: object{}}}
	}
	var fields map[string]json.RawMessage
	// null decodes into a map without an error, leaving it nil.
	if err := json.Unmarshal(body, &fields); err != nil || fields == nil {
		return ItemIn{}, []fieldError{notAnObjectError(json.RawMessage(body))}
	}

	var in ItemIn
	var errs []fieldError
	report := func(err fieldError) { errs = append(errs, err) }

	loc := []any{"body", "name"}
	if raw, ok := fields["name"]; !ok {
		report(fieldError{Type: "missing", Loc: loc, Msg: "Field required", Input: json.RawMessage(body)})
	} else if name, ok := decodeField[string](raw); !ok {
		report(stringTypeError(loc, raw))
	} else if length := utf8.RuneCountInString(name); length < nameMinLen {
		report(fieldError{
			Type: "string_too_short", Loc: loc, Input: raw,
			Msg: fmt.Sprintf("String should have at least %d character", nameMinLen),
			Ctx: object{"min_length": nameMinLen},
		})
	} else if length > nameMaxLen {
		report(stringTooLongError(loc, raw, nameMaxLen))
	} else {
		in.Name = name
	}

	loc = []any{"body", "description"}
	if raw, ok := fields["description"]; ok && string(raw) != "null" {
		if description, ok := decodeField[string](raw); !ok {
			report(stringTypeError(loc, raw))
		} else if utf8.RuneCountInString(description) > descriptionMaxLen {
			report(stringTooLongError(loc, raw, descriptionMaxLen))
		} else {
			in.Description = &description
		}
	}

	loc = []any{"body", "price"}
	if raw, ok := fields["price"]; !ok {
		report(fieldError{Type: "missing", Loc: loc, Msg: "Field required", Input: json.RawMessage(body)})
	} else if price, ok := decodeField[float64](raw); !ok {
		report(fieldError{Type: "float_type", Loc: loc, Msg: "Input should be a valid number", Input: raw})
	} else if price <= priceAbove {
		report(fieldError{
			Type: "greater_than", Loc: loc, Input: raw,
			Msg: fmt.Sprintf("Input should be greater than %v", priceAbove),
			Ctx: object{"gt": priceAbove},
		})
	} else if price > priceMax {
		report(tooLargeError(loc, raw, priceMax))
	} else {
		in.Price = price
	}

	loc = []any{"body", "quantity"}
	if raw, ok := fields["quantity"]; ok {
		if quantity, ok := decodeField[int64](raw); !ok {
			report(fieldError{Type: "int_type", Loc: loc, Msg: "Input should be a valid integer", Input: raw})
		} else if quantity < quantityMin {
			report(tooSmallError(loc, raw, quantityMin))
		} else if quantity > quantityMax {
			report(tooLargeError(loc, raw, quantityMax))
		} else {
			in.Quantity = quantity
		}
	}

	return in, errs
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

func notAnObjectError(input any) fieldError {
	return fieldError{
		Type: "model_attributes_type", Loc: []any{"body"}, Input: input,
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

func tooLargeError[T int | float64](loc []any, input any, maximum T) fieldError {
	return fieldError{
		Type: "less_than_equal", Loc: loc, Input: input,
		Msg: fmt.Sprintf("Input should be less than or equal to %v", maximum),
		Ctx: object{"le": maximum},
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
	writeJSON(w, http.StatusNotFound, object{"detail": fmt.Sprintf("item %d not found", id)})
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
		operation := object{
			"summary":     rt.summary,
			"operationId": rt.operation,
			"responses": object{
				strconv.Itoa(rt.status): success,
				"422": object{
					"description": "Validation Error",
					"content":     jsonContent(schemaRef("HTTPValidationError")),
				},
			},
		}
		if rt.parameters != nil {
			operation["parameters"] = rt.parameters
		}
		if rt.hasBody {
			operation["requestBody"] = object{"required": true, "content": jsonContent(schemaRef("ItemIn"))}
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
		"info":    object{"title": "Fuzzy CRUD example", "version": "1.0.0"},
		"paths":   paths,
		"components": object{"schemas": object{
			"ItemIn": object{
				"type": "object", "title": "ItemIn",
				"properties": itemProperties(),
				"required":   []string{"name", "price"},
			},
			"Item": object{
				"type": "object", "title": "Item",
				"properties": func() object {
					properties := itemProperties()
					properties["id"] = object{"type": "integer", "title": "Id"}
					return properties
				}(),
				"required": []string{"name", "price", "id"},
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

func itemProperties() object {
	return object{
		"name": object{"type": "string", "minLength": nameMinLen, "maxLength": nameMaxLen, "title": "Name"},
		"description": object{
			"type": "string", "maxLength": descriptionMaxLen, "nullable": true, "title": "Description",
		},
		"price": object{
			"type": "number", "minimum": priceAbove, "exclusiveMinimum": true, "maximum": priceMax, "title": "Price",
		},
		"quantity": object{
			"type": "integer", "minimum": quantityMin, "maximum": quantityMax, "default": 0, "title": "Quantity",
		},
	}
}

func schemaRef(name string) object {
	return object{"$ref": "#/components/schemas/" + name}
}

func jsonContent(schema object) object {
	return object{"application/json": object{"schema": schema}}
}
