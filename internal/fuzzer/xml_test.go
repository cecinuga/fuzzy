package fuzzer

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"math/rand/v2"
	"strings"
	"testing"

	"fuzzy/internal/generator"
	"fuzzy/internal/openapi"
)

// parseSchema resolves a components.schemas section and returns the media type of
// a body referring to its schema named Body.
func parseMedia(t *testing.T, schemas string) *openapi.MediaType {
	t.Helper()
	doc, err := openapi.Parse([]byte(`{"openapi":"3.0.0","info":{"title":"t","version":"1"},
		"paths":{"/a":{"post":{"requestBody":{"required":true,"content":{"application/xml":{"schema":{"$ref":"#/components/schemas/Body"}}}}}}},
		"components":{"schemas":` + schemas + `}}`))
	if err != nil {
		t.Fatal(err)
	}
	return doc.Operations[0].RequestBody.Content["application/xml"]
}

func encodeWith(t *testing.T, schemas string, v any) string {
	t.Helper()
	media := parseMedia(t, schemas)
	out := string(encodeXML(newXMLBody(&openapi.Operation{}, "application/xml", media), v))
	body, ok := strings.CutPrefix(out, xml.Header)
	if !ok {
		t.Fatalf("no XML declaration: %s", out)
	}
	checkWellFormed(t, out)
	return body
}

// checkWellFormed parses with namespaces on, so an undeclared prefix would show
// up as an element in the prefix's own namespace, which this also checks for.
func checkWellFormed(t *testing.T, document string) {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(document))
	for {
		token, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return
		}
		if err != nil {
			t.Fatalf("not well-formed: %v\n%s", err, document)
		}
		if start, ok := token.(xml.StartElement); ok && strings.HasPrefix(start.Name.Space, "undeclared") {
			t.Fatalf("undeclared prefix in %s", document)
		}
	}
}

func TestEncodeXML(t *testing.T) {
	cases := map[string]struct {
		schemas string
		value   any
		want    string
	}{
		"root takes the component's name": {
			`{"Body":{"type":"string"}}`, "v", `<Body>v</Body>`,
		},
		"xml.name renames the root": {
			`{"Body":{"type":"string","xml":{"name":"doc"}}}`, "v", `<doc>v</doc>`,
		},
		"properties keep the spec's order": {
			`{"Body":{"type":"object","properties":{"z":{"type":"string"},"a":{"type":"integer"},"m":{"type":"boolean"}}}}`,
			map[string]any{"a": int64(1), "m": true, "z": "x"}, `<Body><z>x</z><a>1</a><m>true</m></Body>`,
		},
		"allOf keeps the base first": {
			`{"Base":{"type":"object","properties":{"z":{"type":"string"}}},"Body":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"a":{"type":"string"}}}]}}`,
			map[string]any{"a": "1", "z": "2"}, `<Body><z>2</z><a>1</a></Body>`,
		},
		"attribute": {
			`{"Body":{"type":"object","properties":{"id":{"type":"integer","xml":{"attribute":true}},"n":{"type":"string"}}}}`,
			map[string]any{"id": int64(7), "n": "x"}, `<Body id="7"><n>x</n></Body>`,
		},
		"renamed attribute": {
			`{"Body":{"type":"object","properties":{"id":{"type":"integer","xml":{"attribute":true,"name":"key"}}}}}`,
			map[string]any{"id": int64(7)}, `<Body key="7"/>`,
		},
		"renamed property": {
			`{"Body":{"type":"object","properties":{"n":{"type":"string","xml":{"name":"label"}}}}}`,
			map[string]any{"n": "x"}, `<Body><label>x</label></Body>`,
		},
		"array repeats its items": {
			`{"Body":{"type":"object","properties":{"tags":{"type":"array","items":{"type":"string"}}}}}`,
			map[string]any{"tags": []any{"a", "b"}}, `<Body><tags>a</tags><tags>b</tags></Body>`,
		},
		"items name": {
			`{"Body":{"type":"object","properties":{"tags":{"type":"array","items":{"type":"string","xml":{"name":"tag"}}}}}}`,
			map[string]any{"tags": []any{"a"}}, `<Body><tag>a</tag></Body>`,
		},
		"name of an unwrapped array has no effect": {
			`{"Body":{"type":"object","properties":{"tags":{"type":"array","xml":{"name":"x"},"items":{"type":"string"}}}}}`,
			map[string]any{"tags": []any{"a"}}, `<Body><tags>a</tags></Body>`,
		},
		"wrapped array": {
			`{"Body":{"type":"object","properties":{"tags":{"type":"array","xml":{"wrapped":true},"items":{"type":"string"}}}}}`,
			map[string]any{"tags": []any{"a", "b"}}, `<Body><tags><tags>a</tags><tags>b</tags></tags></Body>`,
		},
		"wrapped array, both named": {
			`{"Body":{"type":"object","properties":{"tags":{"type":"array","xml":{"wrapped":true,"name":"list"},"items":{"type":"string","xml":{"name":"tag"}}}}}}`,
			map[string]any{"tags": []any{"a"}}, `<Body><list><tag>a</tag></list></Body>`,
		},
		"array at the root": {
			`{"Body":{"type":"array","items":{"type":"string","xml":{"name":"tag"}}}}`,
			[]any{"a", "b"}, `<Body><tag>a</tag><tag>b</tag></Body>`,
		},
		"namespace and prefix": {
			`{"Body":{"type":"object","xml":{"prefix":"p","namespace":"urn:x"},"properties":{"a":{"type":"string","xml":{"prefix":"p"}}}}}`,
			map[string]any{"a": "1"}, `<p:Body xmlns:p="urn:x"><p:a>1</p:a></p:Body>`,
		},
		"default namespace": {
			`{"Body":{"type":"string","xml":{"namespace":"urn:x"}}}`, "v", `<Body xmlns="urn:x">v</Body>`,
		},
		"undeclared prefix is dropped": {
			`{"Body":{"type":"string","xml":{"prefix":"nope"}}}`, "v", `<Body>v</Body>`,
		},
		"prefix doesn't outlive its element": {
			`{"Body":{"type":"object","properties":{"a":{"type":"string","xml":{"prefix":"p","namespace":"urn:x"}},"b":{"type":"string","xml":{"prefix":"p"}}}}}`,
			map[string]any{"a": "1", "b": "2"}, `<Body><p:a xmlns:p="urn:x">1</p:a><b>2</b></Body>`,
		},
		"text is escaped": {
			`{"Body":{"type":"object","properties":{"a":{"type":"string","xml":{"attribute":true}},"b":{"type":"string"}}}}`,
			map[string]any{"a": `"<&`, "b": "</b><x>&"}, `<Body a="&#34;&lt;&amp;"><b>&lt;/b&gt;&lt;x&gt;&amp;</b></Body>`,
		},
		"characters XML can't hold are replaced": {
			`{"Body":{"type":"string"}}`, "a\x01b", "<Body>a\uFFFDb</Body>",
		},
		"keys that aren't names": {
			`{"Body":{"type":"object","additionalProperties":{"type":"string"}}}`,
			map[string]any{"1 bad/key": "x", "": "y"}, `<Body><_>y</_><_1_bad_key>x</_1_bad_key></Body>`,
		},
		"undeclared keys go last": {
			`{"Body":{"type":"object","required":["u"],"properties":{"z":{"type":"string"}}}}`,
			map[string]any{"u": "1", "z": "2"}, `<Body><z>2</z><u>1</u></Body>`,
		},
		"repeated attribute name": {
			`{"Body":{"type":"object","properties":{"a":{"type":"string","xml":{"attribute":true,"name":"k"}},"b":{"type":"string","xml":{"attribute":true,"name":"k"}}}}}`,
			map[string]any{"a": "1", "b": "2"}, `<Body k="1"/>`,
		},
		"oneOf follows the branch of the value": {
			`{"Body":{"oneOf":[{"type":"object","required":["a"],"properties":{"a":{"type":"string"}},"xml":{"name":"first"}},{"type":"object","required":["b"],"properties":{"b":{"type":"string","xml":{"attribute":true}}},"xml":{"name":"second"}}]}}`,
			map[string]any{"b": "1"}, `<second b="1"/>`,
		},
		"discriminator tells branches apart": {
			`{"Cat":{"type":"object","required":["kind"],"properties":{"kind":{"type":"string"},"x":{"type":"string"}},"xml":{"name":"cat"}},
			  "Dog":{"type":"object","required":["kind"],"properties":{"kind":{"type":"string"},"x":{"type":"string"}},"xml":{"name":"dog"}},
			  "Body":{"oneOf":[{"$ref":"#/components/schemas/Cat"},{"$ref":"#/components/schemas/Dog"}],"discriminator":{"propertyName":"kind"}}}`,
			map[string]any{"kind": "Dog", "x": "1"}, `<dog><kind>Dog</kind><x>1</x></dog>`,
		},
		"recursive schema": {
			`{"Body":{"type":"object","properties":{"next":{"$ref":"#/components/schemas/Body"},"v":{"type":"integer"}}}}`,
			map[string]any{"v": int64(1), "next": map[string]any{"v": int64(2)}}, `<Body><next><v>2</v></next><v>1</v></Body>`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := encodeWith(t, tc.schemas, tc.value); got != tc.want {
				t.Errorf("got  %s\nwant %s", got, tc.want)
			}
		})
	}
}

func TestInlineSchemaGetsTheDefaultRoot(t *testing.T) {
	media := &openapi.MediaType{Schema: &openapi.Schema{Type: "string"}}
	got := string(encodeXML(newXMLBody(&openapi.Operation{}, "application/xml", media), "v"))
	if !strings.HasSuffix(got, "<root>v</root>") {
		t.Errorf("got %s", got)
	}
}

func TestSOAPEnvelope(t *testing.T) {
	action := &openapi.Operation{Parameters: []*openapi.Parameter{{Name: "soapaction", In: "header"}}}
	payload := &openapi.MediaType{Schema: &openapi.Schema{Type: "string"}, SchemaName: "GetUser"}
	envelope := &openapi.MediaType{Schema: &openapi.Schema{Type: "object", XML: &openapi.XML{
		Name: "Envelope", Prefix: "s", Namespace: soap11Namespace,
	}}}

	cases := map[string]struct {
		op          *openapi.Operation
		contentType string
		media       *openapi.MediaType
		value       any
		want        string
	}{
		"soap 1.2 by media type": {
			&openapi.Operation{}, "application/soap+xml; charset=utf-8", payload, "v",
			`<soap:Envelope xmlns:soap="` + soap12Namespace + `"><soap:Body><GetUser>v</GetUser></soap:Body></soap:Envelope>`,
		},
		"soap 1.1 by SOAPAction": {
			action, "text/xml", payload, "v",
			`<soap:Envelope xmlns:soap="` + soap11Namespace + `"><soap:Body><GetUser>v</GetUser></soap:Body></soap:Envelope>`,
		},
		"text/xml alone is plain XML": {
			&openapi.Operation{}, "text/xml", payload, "v", `<GetUser>v</GetUser>`,
		},
		"application/xml is plain XML, even with SOAPAction": {
			action, "application/xml", payload, "v", `<GetUser>v</GetUser>`,
		},
		"a schema modeling the envelope isn't wrapped again": {
			&openapi.Operation{}, "application/soap+xml", envelope, map[string]any{},
			`<s:Envelope xmlns:s="` + soap11Namespace + `"/>`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := string(encodeXML(newXMLBody(tc.op, tc.contentType, tc.media), tc.value))
			checkWellFormed(t, got)
			if want := xml.Header + tc.want; got != want {
				t.Errorf("got  %s\nwant %s", got, want)
			}
		})
	}
}

func TestXMLIsTheLastChoiceAmongMediaTypes(t *testing.T) {
	var got capture
	srv := got.server(t, 200)
	targets, _ := compileSpec(t, srv.URL, paths(`{"/b":{"post":{"requestBody":{"required":true,"content":{
		"application/xml":{"schema":{"type":"string"}},
		"text/plain":{"schema":{"type":"string"}}
	}}}}}`))
	Run(context.Background(), options(), targets)
	if contentType := got.requests[0].header.Get("Content-Type"); contentType != "text/plain" {
		t.Errorf("Content-Type = %q", contentType)
	}
}

func TestXMLVariantsAreNotDocuments(t *testing.T) {
	for _, media := range []string{"application/xml-dtd", "application/xml-external-parsed-entity"} {
		if _, _, ok := bodyKindOf(media); ok {
			t.Errorf("%s should not be supported", media)
		}
	}
	for _, media := range []string{"application/xml", "text/xml", "application/soap+xml", "application/atom+xml", "TEXT/XML; charset=utf-8"} {
		if kind, _, ok := bodyKindOf(media); !ok || kind != bodyXML {
			t.Errorf("%s: kind %v, ok %v", media, kind, ok)
		}
	}
}

// A body the server can always parse, for a schema using most of what the
// generator can produce, sent over HTTP with its declared media type.
func TestRunSendsWellFormedXML(t *testing.T) {
	var got capture
	srv := got.server(t, 200)
	targets, skipped := compileSpec(t, srv.URL, `{"openapi":"3.0.0","info":{"title":"t","version":"1"},
		"paths":{"/soap":{"post":{
			"parameters":[{"name":"SOAPAction","in":"header","required":true,"schema":{"type":"string","enum":["urn:get"]}}],
			"requestBody":{"required":true,"content":{"text/xml":{"schema":{"$ref":"#/components/schemas/Pet"}}}}
		}}},
		"components":{"schemas":{"Pet":{"type":"object","required":["id","name","owner"],"xml":{"prefix":"p","namespace":"urn:pets"},"properties":{
			"id":{"type":"integer","xml":{"attribute":true}},
			"name":{"type":"string","xml":{"prefix":"p"}},
			"note":{"type":"string","nullable":true},
			"tags":{"type":"array","minItems":1,"xml":{"wrapped":true},"items":{"type":"string","xml":{"name":"tag"}}},
			"owner":{"oneOf":[{"type":"object","required":["a"],"properties":{"a":{"type":"string"}}},{"type":"object","required":["b"],"properties":{"b":{"type":"number"}}}]},
			"friend":{"$ref":"#/components/schemas/Pet"},
			"extra":{"type":"object","minProperties":1,"additionalProperties":{"type":"boolean"}}
		}}}}}`)
	if len(skipped) != 0 || len(targets) != 1 {
		t.Fatalf("targets %d, skipped %+v", len(targets), skipped)
	}

	opts := options()
	opts.Requests = 50
	Run(context.Background(), opts, targets)
	if len(got.requests) != 50 {
		t.Fatalf("%d requests", len(got.requests))
	}
	for _, r := range got.requests {
		if contentType := r.header.Get("Content-Type"); contentType != "text/xml" {
			t.Errorf("Content-Type = %q", contentType)
		}
		if r.header.Get("SOAPAction") != "urn:get" {
			t.Errorf("SOAPAction = %q", r.header.Get("SOAPAction"))
		}
		checkWellFormed(t, string(r.body))
		for _, want := range []string{`<soap:Envelope xmlns:soap="` + soap11Namespace + `"><soap:Body><p:Pet xmlns:p="urn:pets" id="`, "</p:Pet></soap:Body></soap:Envelope>"} {
			if !bytes.Contains(r.body, []byte(want)) {
				t.Errorf("body lacks %q:\n%s", want, r.body)
			}
		}
	}
}

func TestXMLBodyIsReproducible(t *testing.T) {
	media := parseMedia(t, `{"Body":{"type":"object","minProperties":3,"additionalProperties":{"type":"string"},"properties":{"b":{"type":"string"},"a":{"type":"integer"}}}}`)
	body := newXMLBody(&openapi.Operation{}, "application/xml", media)
	source := generator.New(generator.DefaultBounds(), nil, 1).Compile(media.Schema, generator.InQuery)

	generate := func() string {
		return string(encodeXML(body, source.Generate(rand.New(rand.NewPCG(1, 2)), 3)))
	}
	first := generate()
	for range 20 {
		if again := generate(); again != first {
			t.Fatalf("got %s and %s", first, again)
		}
	}
}
