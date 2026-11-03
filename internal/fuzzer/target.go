package fuzzer

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"fuzzy/internal/generator"
	"fuzzy/internal/openapi"
)

// Target is an operation compiled for fuzzing.
type Target struct {
	Op      *openapi.Operation
	index   int // position among all compiled targets, whatever a Selection keeps
	baseURL string
	params  []targetParam
	body    *targetBody
}

type targetParam struct {
	*openapi.Parameter
	source generator.Source
}

type targetBody struct {
	required    bool
	contentType string
	kind        bodyKind
	binary      map[string]bool               // multipart fields sent as file parts
	fields      map[string]*openapi.Parameter // form fields with a declared encoding
	xml         *xmlBody                      // how to write an XML body
	source      generator.Source
}

// Skipped is an operation no valid request can be built for.
type Skipped struct {
	Op     *openapi.Operation
	Reason string
}

// Compile prepares every operation of doc. A remote without a path takes the
// base path from the spec's servers; a remote with a path is used as given,
// since the user has then already said where the API lives.
func Compile(doc *openapi.Document, gen *generator.Generator, remote *url.URL) ([]*Target, []Skipped) {
	root := strings.TrimRight(remote.String(), "/")
	useBasePath := strings.Trim(remote.Path, "/") == ""

	var targets []*Target
	var skipped []Skipped
	for _, op := range doc.Operations {
		t := &Target{Op: op, index: len(targets), baseURL: root}
		if useBasePath {
			t.baseURL += op.BasePath
		}
		for _, p := range op.Parameters {
			t.params = append(t.params, targetParam{p, gen.Compile(p.Schema, locationOf(p))})
		}
		if op.RequestBody != nil {
			body, err := compileBody(op, gen)
			if err != nil && op.RequestBody.Required {
				skipped = append(skipped, Skipped{op, err.Error()})
				continue
			}
			t.body = body // nil for an optional body we cannot encode: it is just left out
		}
		targets = append(targets, t)
	}
	return targets, skipped
}

func locationOf(p *openapi.Parameter) generator.Location {
	switch p.In {
	case "path":
		return generator.InPath
	case "header", "cookie":
		return generator.InHeader
	}
	return generator.InQuery
}

func compileBody(op *openapi.Operation, gen *generator.Generator) (*targetBody, error) {
	b := op.RequestBody
	best := ""
	var bestKind bodyKind
	bestRank := -1
	for _, contentType := range sortedContentTypes(b.Content) {
		if kind, rank, ok := bodyKindOf(contentType); ok && (bestRank < 0 || rank < bestRank) {
			best, bestKind, bestRank = contentType, kind, rank
		}
	}
	if bestRank < 0 {
		return nil, fmt.Errorf("no supported media type among: %s", strings.Join(sortedContentTypes(b.Content), ", "))
	}

	media := b.Content[best]
	schema := media.Schema
	body := &targetBody{required: b.Required, contentType: best, kind: bestKind}
	if bestKind == bodyForm {
		body.fields = map[string]*openapi.Parameter{}
		for name, enc := range media.Encoding {
			body.fields[name] = &openapi.Parameter{Name: name, In: "query", Style: enc.Style, Explode: enc.Explode}
		}
	}
	if bestKind == bodyMultipart {
		body.binary = map[string]bool{}
		for name, prop := range schema.Properties {
			if prop.Format == "binary" || prop.Format == "base64" {
				body.binary[name] = true
			}
		}
	}
	if bestKind == bodyXML {
		body.xml = newXMLBody(op, best, media)
	}
	loc := generator.InBody
	if bestKind == bodyForm || bestKind == bodyRaw || bestKind == bodyXML {
		loc = generator.InQuery // like query parameters, these cannot represent null
	}
	body.source = gen.Compile(schema, loc)
	return body, nil
}

func sortedContentTypes(content map[string]*openapi.MediaType) []string {
	return slices.Sorted(maps.Keys(content))
}

// build generates one request. Equal r and iter yield equal input.
func (t *Target) build(ctx context.Context, r *rand.Rand, iter int) (*http.Request, string, error) {
	path := t.Op.Path
	input := requestInput{}
	var query []string
	headers := make(http.Header)

	for i := range t.params {
		p := &t.params[i]
		if !p.Required && r.IntN(2) == 0 {
			continue
		}
		v := p.source.Generate(r, iter)
		input.addParam(p.In, p.Name, v)

		switch p.In {
		case "path":
			path = strings.ReplaceAll(path, "{"+p.Name+"}", encodePath(p.Parameter, v))
		case "query":
			query = append(query, encodeQuery(p.Parameter, v)...)
		case "header":
			headers.Set(p.Name, encodeHeader(p.Parameter, v))
		case "cookie":
			headers.Add("Cookie", encodeCookie(p.Parameter, v))
		}
	}

	var body []byte
	if t.body != nil && (t.body.required || r.IntN(4) != 0) {
		v := t.body.source.Generate(r, iter)
		input["body"] = v

		encoded, contentType, err := encodeBody(t.body, v)
		if err != nil {
			return nil, input.text(), fmt.Errorf("encode body: %w", err)
		}
		body = encoded
		headers.Set("Content-Type", contentType)
	}

	target := t.baseURL + path
	if len(query) > 0 {
		target += "?" + strings.Join(query, "&")
	}
	req, err := http.NewRequestWithContext(ctx, t.Op.Method, target, bytes.NewReader(body))
	if err != nil {
		return nil, input.text(), err
	}
	req.Header = headers
	return req, input.text(), nil
}

// requestInput holds the values of a request by where they went: "path", "query",
// "header" and "cookie" map names to values, and "body" holds the body. The same
// name can appear in two locations, so a flat object would lose values.
type requestInput map[string]any

func (in requestInput) addParam(location, name string, v any) {
	values, ok := in[location].(map[string]any)
	if !ok {
		values = map[string]any{}
		in[location] = values
	}
	values[name] = v
}

// text returns the input as a JSON object with sorted keys, so that equal inputs
// yield equal text.
func (in requestInput) text() string {
	return string(toJSON(map[string]any(in)))
}
