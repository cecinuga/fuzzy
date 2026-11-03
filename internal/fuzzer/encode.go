package fuzzer

import (
	"bytes"
	"encoding/json"
	"io"
	"maps"
	"mime"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"fuzzy/internal/openapi"
)

// scalar renders a primitive; composites nested too deep for a style become JSON.
func scalar(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	case json.Number:
		return t.String()
	}
	return string(toJSON(v))
}

// toJSON leaves <, > and & alone: a payload from a dictionary should reach the
// server, and the report, exactly as written.
func toJSON(v any) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil
	}
	return bytes.TrimRight(b.Bytes(), "\n")
}

// styleOf applies the OAS 3.0 defaults for style and explode.
func styleOf(p *openapi.Parameter) (style string, explode bool) {
	style = p.Style
	if style == "" {
		switch p.In {
		case "query", "cookie":
			style = "form"
		default:
			style = "simple"
		}
	}
	explode = style == "form"
	if p.Explode != nil {
		explode = *p.Explode
	}
	return style, explode
}

// rendered is a parameter value in the shape serialization styles work on.
type rendered struct {
	items []string    // array members, or the single scalar
	pairs [][2]string // object members, sorted by key
	array bool
}

func render(p *openapi.Parameter, v any) rendered {
	if p.ContentType != "" {
		return rendered{items: []string{string(toJSON(v))}}
	}
	switch t := v.(type) {
	case []any:
		out := rendered{array: true, items: make([]string, len(t))}
		for i, item := range t {
			out.items[i] = scalar(item)
		}
		return out
	case map[string]any:
		out := rendered{pairs: make([][2]string, 0, len(t))}
		for _, key := range slices.Sorted(maps.Keys(t)) {
			out.pairs = append(out.pairs, [2]string{key, scalar(t[key])})
		}
		return out
	}
	return rendered{items: []string{scalar(v)}}
}

// flat returns object members as k,v,k,v in the non-exploded form, otherwise k=v.
func (r rendered) flat(explode bool, escape func(string) string) []string {
	out := make([]string, 0, len(r.items)+2*len(r.pairs))
	for _, item := range r.items {
		out = append(out, escape(item))
	}
	for _, pair := range r.pairs {
		if explode {
			out = append(out, escape(pair[0])+"="+escape(pair[1]))
		} else {
			out = append(out, escape(pair[0]), escape(pair[1]))
		}
	}
	return out
}

// encodePath serializes a path parameter per its simple, label or matrix style.
func encodePath(p *openapi.Parameter, v any) string {
	style, explode := styleOf(p)
	r := render(p, v)
	flat := r.flat(explode, url.PathEscape)
	name := url.PathEscape(p.Name)

	switch style {
	case "label":
		return "." + strings.Join(flat, ".")
	case "matrix":
		switch {
		case r.pairs != nil && explode:
			return ";" + strings.Join(flat, ";")
		case r.array && explode:
			return ";" + name + "=" + strings.Join(flat, ";"+name+"=")
		}
		return ";" + name + "=" + strings.Join(flat, ",")
	}
	return strings.Join(flat, ",")
}

// encodeQuery serializes a query parameter into already-escaped name=value pairs.
func encodeQuery(p *openapi.Parameter, v any) []string {
	style, explode := styleOf(p)
	r := render(p, v)
	name := url.QueryEscape(p.Name)

	if style == "deepObject" && p.ContentType == "" {
		return deepObject(name, v, nil)
	}
	if r.pairs != nil {
		out := make([]string, 0, len(r.pairs))
		if explode {
			for _, pair := range r.pairs {
				out = append(out, url.QueryEscape(pair[0])+"="+url.QueryEscape(pair[1]))
			}
			return out
		}
		return []string{name + "=" + strings.Join(r.flat(false, url.QueryEscape), ",")}
	}

	if r.array && explode {
		out := make([]string, len(r.items))
		for i, item := range r.items {
			out[i] = name + "=" + url.QueryEscape(item)
		}
		return out
	}

	sep := ","
	switch style {
	case "spaceDelimited":
		sep = "%20"
	case "pipeDelimited":
		sep = "%7C"
	}
	return []string{name + "=" + strings.Join(r.flat(false, url.QueryEscape), sep)}
}

// deepObject serializes v as name[key]=value. OAS only defines this style for
// flat objects; nested objects and arrays (name[a][b], name[0]) follow the
// bracket convention of the APIs using it. Indexes rather than a bare name[]
// keep the members of an array of objects apart.
func deepObject(name string, v any, out []string) []string {
	switch t := v.(type) {
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(t)) {
			out = deepObject(name+"%5B"+url.QueryEscape(key)+"%5D", t[key], out)
		}
	case []any:
		for i, item := range t {
			out = deepObject(name+"%5B"+strconv.Itoa(i)+"%5D", item, out)
		}
	default:
		out = append(out, name+"="+url.QueryEscape(scalar(v)))
	}
	return out
}

// encodeHeader serializes a header parameter (always the simple style).
func encodeHeader(p *openapi.Parameter, v any) string {
	_, explode := styleOf(p)
	return strings.Join(render(p, v).flat(explode, func(s string) string { return s }), ",")
}

// encodeCookie serializes a cookie parameter as a single name=value pair.
func encodeCookie(p *openapi.Parameter, v any) string {
	return p.Name + "=" + strings.Join(render(p, v).flat(false, url.QueryEscape), ",")
}

type bodyKind int

const (
	bodyJSON bodyKind = iota
	bodyForm
	bodyMultipart
	bodyRaw
)

// bodyKindOf classifies a media type; ok is false when no valid body can be
// produced for it (XML, for one, cannot be derived from the schema alone).
func bodyKindOf(contentType string) (kind bodyKind, rank int, ok bool) {
	media, _, _ := strings.Cut(strings.ToLower(contentType), ";")
	media = strings.TrimSpace(media)
	switch {
	case media == "application/json":
		return bodyJSON, 0, true
	case strings.HasSuffix(media, "+json"):
		return bodyJSON, 1, true
	case media == "application/x-www-form-urlencoded":
		return bodyForm, 2, true
	case media == "multipart/form-data":
		return bodyMultipart, 3, true
	case strings.Contains(media, "xml"):
		return 0, 0, false
	case strings.HasPrefix(media, "text/"):
		return bodyRaw, 4, true
	case media == "application/octet-stream":
		return bodyRaw, 5, true
	case media == "*/*" || media == "application/*":
		return bodyJSON, 6, true
	}
	return 0, 0, false
}

// encodeBody serializes v and returns the Content-Type to send with it.
func encodeBody(b *targetBody, v any) ([]byte, string, error) {
	kind, contentType := b.kind, b.contentType
	switch kind {
	case bodyJSON:
		if strings.Contains(contentType, "*") {
			contentType = "application/json"
		}
		return toJSON(v), contentType, nil
	case bodyForm:
		return encodeForm(b.fields, v), contentType, nil
	case bodyMultipart:
		return encodeMultipart(b.binary, v)
	}
	return []byte(scalar(v)), contentType, nil
}

// encodeForm serializes a form body. Its fields follow the rules of query
// parameters, with the `encoding` object of the media type in place of the
// parameter's style and explode.
func encodeForm(fields map[string]*openapi.Parameter, v any) []byte {
	object, ok := v.(map[string]any)
	if !ok {
		return []byte("value=" + url.QueryEscape(scalar(v)))
	}
	pairs := make([]string, 0, len(object))
	for _, key := range slices.Sorted(maps.Keys(object)) {
		field := fields[key]
		if field == nil {
			field = &openapi.Parameter{Name: key, In: "query"}
		}
		pairs = append(pairs, encodeQuery(field, object[key])...)
	}
	return []byte(strings.Join(pairs, "&"))
}

// createNullPart declares the part as JSON: as plain text, "null" would just be a
// four-letter string, and an empty part would be an empty one.
func createNullPart(w *multipart.Writer, name string) (io.Writer, error) {
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": name}))
	header.Set("Content-Type", "application/json")
	return w.CreatePart(header)
}

func encodeMultipart(binary map[string]bool, v any) ([]byte, string, error) {
	fields, ok := v.(map[string]any)
	if !ok {
		fields = map[string]any{"value": v}
	}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		var part io.Writer
		var err error
		content := scalar(fields[key])
		switch {
		case fields[key] == nil:
			part, err = createNullPart(w, key)
			content = "null"
		case binary[key]:
			part, err = w.CreateFormFile(key, key+".bin")
		default:
			part, err = w.CreateFormField(key)
		}
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write([]byte(content)); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return b.Bytes(), w.FormDataContentType(), nil
}
