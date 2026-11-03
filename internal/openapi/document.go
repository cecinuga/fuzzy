// Package openapi loads an OpenAPI 3.0 JSON document into a resolved model:
// every local $ref is replaced by a pointer to its target and every allOf is
// merged, so consumers only ever walk plain schemas (which may form cycles).
package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
)

// Document is the subset of an OpenAPI 3.0 document that fuzzing needs.
type Document struct {
	OpenAPI    string               `json:"openapi"`
	Info       Info                 `json:"info"`
	Servers    []Server             `json:"servers"`
	Paths      map[string]*PathItem `json:"paths"`
	Components Components           `json:"components"`

	// Operations lists every operation, sorted by path then by verb.
	Operations []*Operation `json:"-"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type Server struct {
	URL       string                    `json:"url"`
	Variables map[string]ServerVariable `json:"variables"`
}

type ServerVariable struct {
	Default string `json:"default"`
}

type Components struct {
	Schemas       map[string]*Schema      `json:"schemas"`
	Parameters    map[string]*Parameter   `json:"parameters"`
	RequestBodies map[string]*RequestBody `json:"requestBodies"`
}

type PathItem struct {
	Parameters []*Parameter `json:"parameters"`
	Servers    []Server     `json:"servers"`

	Get     *Operation `json:"get"`
	Put     *Operation `json:"put"`
	Post    *Operation `json:"post"`
	Delete  *Operation `json:"delete"`
	Options *Operation `json:"options"`
	Head    *Operation `json:"head"`
	Patch   *Operation `json:"patch"`
	Trace   *Operation `json:"trace"`
}

// Operation is one verb on one path, with path-level parameters already merged.
type Operation struct {
	OperationID string       `json:"operationId"`
	Summary     string       `json:"summary"`
	Tags        []string     `json:"tags"`
	Parameters  []*Parameter `json:"parameters"`
	RequestBody *RequestBody `json:"requestBody"`
	Servers     []Server     `json:"servers"`

	Method   string `json:"-"`
	Path     string `json:"-"`
	BasePath string `json:"-"` // path component of the applicable server URL
}

// Parameter is a path, query, header or cookie parameter. A parameter declared
// through `content` is normalized at load time: Schema holds the chosen media
// type's schema and ContentType its name.
type Parameter struct {
	Ref      string                `json:"$ref"`
	Name     string                `json:"name"`
	In       string                `json:"in"`
	Required bool                  `json:"required"`
	Schema   *Schema               `json:"schema"`
	Style    string                `json:"style"`
	Explode  *bool                 `json:"explode"`
	Content  map[string]*MediaType `json:"content"`

	ContentType string `json:"-"`
}

type RequestBody struct {
	Ref      string                `json:"$ref"`
	Required bool                  `json:"required"`
	Content  map[string]*MediaType `json:"content"`
}

type MediaType struct {
	Schema   *Schema              `json:"schema"`
	Encoding map[string]*Encoding `json:"encoding"`

	// SchemaName is the component the schema referred to, if any. References are
	// gone once resolved, and an XML body takes its root element's name from it.
	SchemaName string `json:"-"`
}

// Encoding says how one property of a form body is serialized.
type Encoding struct {
	Style   string `json:"style"`
	Explode *bool  `json:"explode"`
}

// Load reads and resolves the JSON document stored at path.
func Load(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}
	return Parse(data)
}

// Parse decodes and resolves a JSON document. Only OpenAPI 3.0.x is accepted:
// Swagger 2.0 and OpenAPI 3.1 describe schemas differently, and misreading them
// would silently produce invalid requests.
func Parse(data []byte) (*Document, error) {
	var probe struct {
		OpenAPI string `json:"openapi"`
		Swagger string `json:"swagger"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, fmt.Errorf("spec is not valid JSON (YAML is not supported): %w", err)
	}
	if !strings.HasPrefix(probe.OpenAPI, "3.0") {
		found := probe.OpenAPI
		if found == "" && probe.Swagger != "" {
			found = "swagger " + probe.Swagger
		}
		if found == "" {
			found = "missing version field"
		}
		return nil, fmt.Errorf("unsupported spec version (%s): OpenAPI 3.0.x is required", found)
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber() // int64 limits and enum members must not lose precision
	var doc Document
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode spec: %w", err)
	}
	if len(doc.Paths) == 0 {
		return nil, errors.New("spec defines no paths")
	}

	r := newResolver(&doc)
	if err := r.resolveAll(); err != nil {
		return nil, err
	}
	if err := doc.buildOperations(r); err != nil {
		return nil, err
	}
	if len(doc.Operations) == 0 {
		return nil, errors.New("spec defines no operations")
	}
	return &doc, nil
}

var pathParamRe = regexp.MustCompile(`\{([^{}]+)\}`)

func (d *Document) buildOperations(r *resolver) error {
	paths := make([]string, 0, len(d.Paths))
	for p := range d.Paths {
		paths = append(paths, p)
	}
	slices.Sort(paths)

	for _, path := range paths {
		item := d.Paths[path]
		if item == nil {
			continue
		}
		for _, entry := range []struct {
			method string
			op     *Operation
		}{
			{"GET", item.Get}, {"POST", item.Post}, {"PUT", item.Put}, {"PATCH", item.Patch},
			{"DELETE", item.Delete}, {"HEAD", item.Head}, {"OPTIONS", item.Options}, {"TRACE", item.Trace},
		} {
			if entry.op == nil {
				continue
			}
			op := entry.op
			op.Method, op.Path = entry.method, path
			op.BasePath = serverBasePath(op.Servers, item.Servers, d.Servers)

			params, err := mergeParameters(path, item.Parameters, op.Parameters)
			if err != nil {
				return fmt.Errorf("%s %s: %w", op.Method, path, err)
			}
			op.Parameters = params
			d.Operations = append(d.Operations, op)
		}
	}
	return nil
}

// mergeParameters applies the spec's override rule (an operation parameter
// replaces the path-level one with the same name and location) and then makes
// sure every {placeholder} of the path template has a parameter, because
// sending a literal "{id}" could never be a valid request.
func mergeParameters(path string, shared, own []*Parameter) ([]*Parameter, error) {
	merged := slices.Clone(shared)
	for _, p := range own {
		if i := slices.IndexFunc(merged, func(m *Parameter) bool {
			return m.Name == p.Name && m.In == p.In
		}); i >= 0 {
			merged[i] = p
		} else {
			merged = append(merged, p)
		}
	}

	out := merged[:0]
	for _, p := range merged {
		switch p.In {
		case "path", "query", "header", "cookie":
		default:
			return nil, fmt.Errorf("parameter %q: invalid location %q", p.Name, p.In)
		}
		if p.Name == "" {
			return nil, fmt.Errorf("parameter in %s without a name", p.In)
		}
		// OAS 3.0: these header parameters "shall be ignored".
		if p.In == "header" && slices.Contains([]string{"accept", "content-type", "authorization"}, strings.ToLower(p.Name)) {
			continue
		}
		out = append(out, p)
	}

	for _, m := range pathParamRe.FindAllStringSubmatch(path, -1) {
		if !slices.ContainsFunc(out, func(p *Parameter) bool { return p.In == "path" && p.Name == m[1] }) {
			out = append(out, &Parameter{Name: m[1], In: "path", Required: true, Schema: &Schema{Type: "string"}})
		}
	}
	return out, nil
}

// serverBasePath returns the path of the first applicable server. The host is
// discarded on purpose: the target is always the remote given on the command line.
func serverBasePath(levels ...[]Server) string {
	for _, servers := range levels {
		if len(servers) == 0 {
			continue
		}
		raw := servers[0].URL
		for name, v := range servers[0].Variables {
			raw = strings.ReplaceAll(raw, "{"+name+"}", v.Default)
		}
		u, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		return strings.TrimRight(u.Path, "/")
	}
	return ""
}
