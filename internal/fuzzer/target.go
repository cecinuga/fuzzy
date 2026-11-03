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
			body, err := compileBody(op.RequestBody, gen)
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

func compileBody(b *openapi.RequestBody, gen *generator.Generator) (*targetBody, error) {
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
	loc := generator.InBody
	if bestKind == bodyForm || bestKind == bodyRaw {
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
	var input strings.Builder
	var query []string
	headers := make(http.Header)

	for i := range t.params {
		p := &t.params[i]
		if !p.Required && r.IntN(2) == 0 {
			continue
		}
		v := p.source.Generate(r, iter)
		writeInput(&input, p.Name, v)

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
		writeInput(&input, "body", v)

		encoded, contentType, err := encodeBody(t.body, v)
		if err != nil {
			return nil, input.String(), fmt.Errorf("encode body: %w", err)
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
		return nil, input.String(), err
	}
	req.Header = headers
	return req, input.String(), nil
}

func writeInput(b *strings.Builder, name string, v any) {
	if b.Len() > 0 {
		b.WriteString(", ")
	}
	b.WriteString(name)
	b.WriteByte('=')
	switch v.(type) {
	case []any, map[string]any, nil:
		b.Write(toJSON(v))
	default:
		b.WriteString(scalar(v))
	}
}
