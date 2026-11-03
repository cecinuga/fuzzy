package openapi

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func swaggerSpec(body string) string {
	return `{"swagger":"2.0","info":{"title":"t","version":"1"},` + body + `}`
}

func mustFindOperation(t *testing.T, doc *Document, method, path string) *Operation {
	t.Helper()
	for _, op := range doc.Operations {
		if op.Method == method && op.Path == path {
			return op
		}
	}
	t.Fatalf("no operation %s %s", method, path)
	return nil
}

func TestParseRoutesByDeclaredVersion(t *testing.T) {
	const paths = `"paths":{"/a":{"get":{}}}`
	accepted := map[string]string{
		`{"openapi":"3.0.3",` + paths + `}`: "openapi 3.0.3",
		`{"openapi":"3.0.0",` + paths + `}`: "openapi 3.0.0",
		`{"swagger":"2.0",` + paths + `}`:   "swagger 2.0",
		// An OpenAPI document with a leftover swagger field is still OpenAPI.
		`{"openapi":"3.0.0","swagger":"2.0",` + paths + `}`: "openapi 3.0.0",
	}
	for spec, want := range accepted {
		doc, err := Parse([]byte(spec))
		if err != nil {
			t.Errorf("Parse(%s): %v", spec, err)
			continue
		}
		if doc.Format != want {
			t.Errorf("Parse(%s).Format = %q, want %q", spec, doc.Format, want)
		}
	}

	refused := map[string]string{
		`{"swagger":"1.2",` + paths + `}`:                   "swagger 1.2",
		`{"swagger":"3.0",` + paths + `}`:                   "swagger 3.0",
		`{"openapi":"3.1.0",` + paths + `}`:                 "3.1.0",
		`{"openapi":"3.1.0","swagger":"2.0",` + paths + `}`: "3.1.0",
		`{` + paths + `}`:                                   "missing version",
	}
	for spec, want := range refused {
		_, err := Parse([]byte(spec))
		if err == nil || !strings.Contains(err.Error(), want) || !strings.Contains(err.Error(), "Swagger 2.0 is required") {
			t.Errorf("Parse(%s) error = %v, want it to mention %q and what is supported", spec, err, want)
		}
	}
}

func TestSwaggerRejectsInvalidDocuments(t *testing.T) {
	op := func(parameters string) string {
		return swaggerSpec(`"paths":{"/a":{"post":{"parameters":[` + parameters + `]}}}`)
	}
	cases := map[string]struct{ spec, want string }{
		"no paths": {swaggerSpec(`"paths":{}`), "no paths"},
		"no ops":   {swaggerSpec(`"paths":{"/a":{}}`), "no operations"},
		"external ref": {op(`{"name":"b","in":"body","schema":{"$ref":"other.json#/X"}}`),
			`unsupported schema reference "other.json#/X": only #/definitions/* is allowed`},
		"openapi ref in swagger": {op(`{"name":"b","in":"body","schema":{"$ref":"#/components/schemas/X"}}`),
			"unsupported schema reference"},
		"unknown definition": {op(`{"name":"b","in":"body","schema":{"$ref":"#/definitions/Nope"}}`),
			`unresolvable schema reference "#/definitions/Nope"`},
		"unknown ref in a definition": {swaggerSpec(`"paths":{"/a":{"get":{}}},"definitions":{"A":{"properties":{"x":{"$ref":"#/definitions/Nope"}}}}`),
			`definitions.A: property "x"`},
		"null definition":   {swaggerSpec(`"paths":{"/a":{"get":{}}},"definitions":{"A":null}`), "null schema"},
		"unknown parameter": {op(`{"$ref":"#/parameters/nope"}`), `unresolvable parameter reference "#/parameters/nope"`},
		"parameter cycle": {swaggerSpec(`"paths":{"/a":{"get":{"parameters":[{"$ref":"#/parameters/x"}]}}},"parameters":{"x":{"$ref":"#/parameters/x"}}`),
			"unresolvable parameter reference"},
		"null parameter":  {op(`null`), "null parameter"},
		"cookie location": {op(`{"name":"c","in":"cookie","type":"string"}`), `parameter "c": invalid location "cookie"`},
		"body and form": {op(`{"name":"b","in":"body","schema":{"type":"object"}},{"name":"f","in":"formData","type":"string"}`),
			"cannot be combined"},
		"form without a name": {op(`{"in":"formData","type":"string"}`), "without a name"},
		"bad collectionFormat": {op(`{"name":"q","in":"query","type":"array","items":{"type":"string"},"collectionFormat":"nope"}`),
			`unsupported collectionFormat "nope"`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(tc.spec))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestSwaggerBasePathAndHost(t *testing.T) {
	doc := mustParse(t, swaggerSpec(`"host":"prod.example.com","schemes":["https"],"basePath":"/api/v2/","paths":{"/a":{"get":{}}}`))
	if got := doc.Operations[0].BasePath; got != "/api/v2" {
		t.Errorf("BasePath = %q, want the spec's path without the host", got)
	}
	doc = mustParse(t, swaggerSpec(`"host":"prod.example.com","paths":{"/a":{"get":{}}}`))
	if got := doc.Operations[0].BasePath; got != "" {
		t.Errorf("BasePath = %q: the host must never be used", got)
	}
}

func TestSwaggerParametersKeepTheirKeywords(t *testing.T) {
	doc := mustParse(t, swaggerSpec(`"parameters":{
		"limit":{"name":"limit","in":"query","required":true,"type":"integer","format":"int64","minimum":9007199254740993,"maximum":9007199254740995}
	},"paths":{"/a/{id}":{
		"parameters":[
			{"$ref":"#/parameters/limit"},
			{"name":"over","in":"query","type":"string"},
			{"name":"id","in":"path","type":"integer","required":false}
		],
		"get":{"operationId":"getA","summary":"s","tags":["x"],"parameters":[
			{"name":"over","in":"query","type":"string","enum":["a","b"],"pattern":"^[ab]$","minLength":1,"maxLength":1},
			{"name":"X-Ignored","in":"header","type":"string"},
			{"name":"Authorization","in":"header","type":"string"}
		]}
	}}`))
	op := mustFindOperation(t, doc, "GET", "/a/{id}")
	if op.OperationID != "getA" || op.Summary != "s" || !slices.Equal(op.Tags, []string{"x"}) {
		t.Errorf("operation metadata lost: %+v", op)
	}

	byName := map[string]*Parameter{}
	for _, p := range op.Parameters {
		byName[p.Name] = p
	}
	if _, ok := byName["Authorization"]; ok {
		t.Error("the headers OpenAPI ignores must be ignored here too")
	}
	limit := byName["limit"]
	if limit == nil || !limit.Required || limit.Schema.Type != "integer" || limit.Schema.Format != "int64" {
		t.Fatalf("shared parameter lost its keywords: %+v", limit)
	}
	if got := limit.Schema.Minimum.String(); got != "9007199254740993" {
		t.Errorf("minimum = %s: int64 limits must not lose precision", got)
	}
	if over := byName["over"]; len(over.Schema.Enum) != 2 || over.Schema.Pattern != "^[ab]$" || *over.Schema.MaxLength != 1 {
		t.Errorf("the operation's parameter must override the path's: %+v", over.Schema)
	}
	if id := byName["id"]; !id.Required || id.Schema.Type != "integer" {
		t.Errorf("a path parameter is always required: %+v", id)
	}
	if len(op.Parameters) != 4 {
		t.Errorf("parameters = %d, want 4", len(op.Parameters))
	}
}

func TestSwaggerCollectionFormats(t *testing.T) {
	for format, want := range map[string]struct {
		explode   bool
		delimiter string
	}{
		"":      {false, ""},
		"csv":   {false, ""},
		"multi": {true, ""},
		"ssv":   {false, " "},
		"tsv":   {false, "\t"},
		"pipes": {false, "|"},
	} {
		field := ""
		if format != "" {
			field = `,"collectionFormat":"` + format + `"`
		}
		array := `"type":"array","items":{"type":"string"}` + field
		doc := mustParse(t, swaggerSpec(`"paths":{"/a":{"post":{"parameters":[
			{"name":"q","in":"query",`+array+`},
			{"name":"f","in":"formData",`+array+`}
		]}}}`))
		op := doc.Operations[0]
		q := op.Parameters[0]
		if q.Explode == nil || *q.Explode != want.explode || q.Delimiter != want.delimiter {
			t.Errorf("%q: query explode=%v delimiter=%q, want %+v", format, q.Explode, q.Delimiter, want)
		}
		f := op.RequestBody.Content[formURLEncoded].Encoding["f"]
		if f == nil || *f.Explode != want.explode || f.Delimiter != want.delimiter {
			t.Errorf("%q: form field encoding = %+v, want %+v", format, f, want)
		}
	}

	doc := mustParse(t, swaggerSpec(`"paths":{"/a":{"get":{"parameters":[{"name":"q","in":"query","type":"string"}]}}}`))
	if q := doc.Operations[0].Parameters[0]; q.Explode != nil || q.Delimiter != "" {
		t.Errorf("a scalar has no collection format: %+v", q)
	}
}

func TestSwaggerBodyParameter(t *testing.T) {
	spec := swaggerSpec(`"consumes":["application/json","application/xml"],
	"definitions":{"Item":{"type":"object","required":["id"],"properties":{"id":{"type":"integer"},"next":{"$ref":"#/definitions/Item"}}}},
	"parameters":{"shared":{"name":"item","in":"body","required":true,"schema":{"$ref":"#/definitions/Item"}}},
	"paths":{
		"/global":{"post":{"parameters":[{"$ref":"#/parameters/shared"}]}},
		"/own":{"post":{"consumes":["text/plain"],"parameters":[{"name":"b","in":"body","schema":{"type":"string"}}]}},
		"/cleared":{"post":{"consumes":[],"parameters":[{"name":"b","in":"body","schema":{"type":"string"}}]}},
		"/shared":{
			"parameters":[{"name":"old","in":"body","schema":{"type":"string"}}],
			"get":{},
			"post":{"parameters":[{"name":"new","in":"body","schema":{"type":"integer"}}]}
		}
	}`)
	doc := mustParse(t, spec)

	body := mustFindOperation(t, doc, "POST", "/global").RequestBody
	if !body.Required || len(body.Content) != 2 {
		t.Fatalf("body = %+v, want required, in the two global media types", body)
	}
	for contentType, media := range body.Content {
		if media.SchemaName != "Item" {
			t.Errorf("%s: SchemaName = %q: an XML root takes its name from it", contentType, media.SchemaName)
		}
		if media.Schema.Properties["next"] != media.Schema {
			t.Errorf("%s: a recursive definition must resolve to itself", contentType)
		}
	}
	if slices.Compare(sortedKeys(mustFindOperation(t, doc, "POST", "/own").RequestBody.Content), []string{"text/plain"}) != 0 {
		t.Error("an operation's consumes must replace the global one")
	}
	if slices.Compare(sortedKeys(mustFindOperation(t, doc, "POST", "/cleared").RequestBody.Content), []string{"application/json"}) != 0 {
		t.Error("an empty consumes clears the global one, and JSON is the default")
	}

	// A path-level body reaches the operations, and any body of their own replaces it.
	if got := mustFindOperation(t, doc, "GET", "/shared").RequestBody.Content["application/json"].Schema.Type; got != "string" {
		t.Errorf("inherited body type = %q", got)
	}
	if got := mustFindOperation(t, doc, "POST", "/shared").RequestBody.Content["application/json"].Schema.Type; got != "integer" {
		t.Errorf("overriding body type = %q", got)
	}
	for _, p := range mustFindOperation(t, doc, "POST", "/shared").Parameters {
		t.Errorf("a body must not stay a parameter: %+v", p)
	}
}

func TestSwaggerFormParameters(t *testing.T) {
	doc := mustParse(t, swaggerSpec(`"consumes":["application/json"],"paths":{
		"/form":{"post":{"parameters":[
			{"name":"zeta","in":"formData","type":"string","required":true,"maxLength":3},
			{"name":"alpha","in":"formData","type":"integer"}
		]}},
		"/optional":{"post":{"parameters":[{"name":"a","in":"formData","type":"string"}]}},
		"/upload":{"post":{"consumes":["application/x-www-form-urlencoded"],"parameters":[
			{"name":"doc","in":"formData","type":"file","required":true},
			{"name":"note","in":"formData","type":"string"}
		]}},
		"/both":{"post":{"consumes":["multipart/form-data; boundary=x","application/x-www-form-urlencoded"],"parameters":[
			{"name":"a","in":"formData","type":"string"}
		]}}
	}`))

	form := mustFindOperation(t, doc, "POST", "/form").RequestBody
	if !form.Required || len(form.Content) != 1 {
		t.Fatalf("form body = %+v, want required and only urlencoded: JSON can't carry form fields", form)
	}
	schema := form.Content[formURLEncoded].Schema
	if schema.Type != "object" || !slices.Equal(schema.Required, []string{"zeta"}) || *schema.Properties["zeta"].MaxLength != 3 {
		t.Errorf("form schema = %+v", schema)
	}
	if got := schema.OrderedProperties(); !slices.Equal(got, []string{"zeta", "alpha"}) {
		t.Errorf("properties = %v, want the spec's order", got)
	}
	if mustFindOperation(t, doc, "POST", "/optional").RequestBody.Required {
		t.Error("a body with no required field is optional")
	}

	upload := mustFindOperation(t, doc, "POST", "/upload").RequestBody
	if len(upload.Content) != 1 || upload.Content[multipartFormData] == nil {
		t.Fatalf("upload content = %v, want only multipart: nothing else carries a file", sortedKeys(upload.Content))
	}
	if file := upload.Content[multipartFormData].Schema.Properties["doc"]; file.Type != "string" || file.Format != "binary" {
		t.Errorf("file field = %+v, want a binary string", file)
	}

	both := mustFindOperation(t, doc, "POST", "/both").RequestBody
	if want := []string{"application/x-www-form-urlencoded", "multipart/form-data; boundary=x"}; !slices.Equal(sortedKeys(both.Content), want) {
		t.Errorf("content = %v, want %v", sortedKeys(both.Content), want)
	}
}

// bodySchema returns the resolved schema of the only body of a spec's POST /a.
func bodySchema(t *testing.T, definitions, ref string) *Schema {
	t.Helper()
	doc := mustParse(t, swaggerSpec(`"definitions":{`+definitions+`},"paths":{"/a":{"post":{"parameters":[
		{"name":"b","in":"body","required":true,"schema":{"$ref":"#/definitions/`+ref+`"}}
	]}}}`))
	return doc.Operations[0].RequestBody.Content[defaultSwaggerContentType].Schema
}

const petDefinitions = `
	"Pet":{"type":"object","discriminator":"petType","required":["name","petType"],"xml":{"name":"animal"},"properties":{
		"name":{"type":"string"},"petType":{"type":"string"},"friend":{"$ref":"#/definitions/Pet"}
	}},
	"Cat":{"allOf":[{"$ref":"#/definitions/Pet"},{"type":"object","required":["lives"],"properties":{"lives":{"type":"integer"}}}]},
	"Dog":{"allOf":[{"$ref":"#/definitions/Pet"},{"type":"object","properties":{"bark":{"type":"string"}}}]},
	"Puppy":{"allOf":[{"$ref":"#/definitions/Dog"},{"type":"object","properties":{"age":{"type":"integer"}}}]}`

// variantsByTag indexes the variants of s by the single value their tag can have.
func variantsByTag(t *testing.T, s *Schema, tag string) map[string]*Schema {
	t.Helper()
	out := map[string]*Schema{}
	for _, v := range s.Variants() {
		property := v.Properties[tag]
		if property == nil || len(property.Enum) != 1 {
			t.Fatalf("variant without a fixed %s: %+v", tag, v)
		}
		if !slices.Contains(v.Required, tag) {
			t.Errorf("%v: the discriminator must be required", property.Enum)
		}
		out[property.Enum[0].(string)] = v
	}
	return out
}

func TestSwaggerDiscriminatorBecomesTheConcreteSubtypes(t *testing.T) {
	pet := bodySchema(t, petDefinitions, "Pet")
	variants := variantsByTag(t, pet, "petType")
	if got := sortedKeys(variants); !slices.Equal(got, []string{"Cat", "Puppy"}) {
		t.Fatalf("variants = %v, want the leaves: a base may be abstract", got)
	}
	cat := variants["Cat"]
	if cat.Properties["name"] == nil || cat.Properties["lives"] == nil || !slices.Contains(cat.Required, "lives") || !slices.Contains(cat.Required, "name") {
		t.Errorf("a Cat must have its own and its base's fields: %+v", cat)
	}
	puppy := variants["Puppy"]
	if puppy.Properties["name"] == nil || puppy.Properties["bark"] == nil || puppy.Properties["age"] == nil {
		t.Errorf("a Puppy must have the fields of the whole chain: %+v", puppy)
	}

	// Recursive references stay polymorphic.
	if friend := cat.Properties["friend"]; len(friend.OneOf) != 2 {
		t.Errorf("a reference to the base from a subtype must be polymorphic too: %+v", friend)
	}
	// The XML root keeps the base's name and its xml keyword.
	if pet.XML == nil || pet.XML.Name != "animal" {
		t.Errorf("xml keyword of the base lost: %+v", pet.XML)
	}
}

func TestSwaggerDiscriminatorPinsDirectReferences(t *testing.T) {
	cat := bodySchema(t, petDefinitions, "Cat")
	tag := cat.Properties["petType"]
	if tag == nil || len(tag.Enum) != 1 || tag.Enum[0] != "Cat" || !slices.Contains(cat.Required, "petType") {
		t.Errorf("a reference to a subtype must fix the tag: %+v", tag)
	}
	if len(cat.OneOf) != 0 {
		t.Errorf("a leaf is not polymorphic: %+v", cat.OneOf)
	}

	// An intermediate type is polymorphic over what extends it.
	dog := bodySchema(t, petDefinitions, "Dog")
	if got := sortedKeys(variantsByTag(t, dog, "petType")); !slices.Equal(got, []string{"Puppy"}) {
		t.Errorf("Dog variants = %v, want its leaves", got)
	}
}

func TestSwaggerDiscriminatorWithoutSubtypes(t *testing.T) {
	s := bodySchema(t, `"Lone":{"type":"object","discriminator":"kind","properties":{"kind":{"type":"string"}}}`, "Lone")
	tag := s.Properties["kind"]
	if len(s.OneOf) != 0 || len(tag.Enum) != 1 || tag.Enum[0] != "Lone" || !slices.Contains(s.Required, "kind") {
		t.Errorf("the only possible value is the schema's own name: %+v", s)
	}
}

func TestSwaggerDiscriminatorRespectsTheEnum(t *testing.T) {
	// The enum lists both names: each subtype narrows it to its own.
	s := bodySchema(t, `
		"B":{"discriminator":"k","properties":{"k":{"type":"string","enum":["C","D"]}}},
		"C":{"allOf":[{"$ref":"#/definitions/B"}]},
		"D":{"allOf":[{"$ref":"#/definitions/B"}]}`, "B")
	if got := sortedKeys(variantsByTag(t, s, "k")); !slices.Equal(got, []string{"C", "D"}) {
		t.Errorf("variants = %v", got)
	}

	// The enum doesn't use the definitions' names: its values are what is valid.
	s = bodySchema(t, `
		"B":{"discriminator":"k","properties":{"k":{"type":"string","enum":["see","dee"]}}},
		"C":{"allOf":[{"$ref":"#/definitions/B"}]},
		"D":{"allOf":[{"$ref":"#/definitions/B"}]}`, "B")
	variants := s.Variants()
	if len(variants) != 2 {
		t.Fatalf("variants = %d, want 2", len(variants))
	}
	for _, v := range variants {
		if got := v.Properties["k"].Enum; len(got) != 2 || got[0] != "see" {
			t.Errorf("enum = %v, want the spec's", got)
		}
	}
}

func TestSwaggerInheritanceCyclesTerminate(t *testing.T) {
	bodySchema(t, `
		"A":{"discriminator":"k","allOf":[{"$ref":"#/definitions/B"}]},
		"B":{"allOf":[{"$ref":"#/definitions/A"}]},
		"Self":{"discriminator":"k","allOf":[{"$ref":"#/definitions/Self"}]}`, "A")
}

func TestSwaggerNamesDoNotCollide(t *testing.T) {
	s := bodySchema(t, `
		"B":{"discriminator":"k","properties":{"k":{"type":"string"},"mine":{"type":"integer"}}},
		"B$base":{"type":"object","properties":{"theirs":{"type":"integer"}}},
		"C":{"allOf":[{"$ref":"#/definitions/B"}]}`, "B$base")
	if s.Properties["theirs"] == nil || s.Properties["mine"] != nil {
		t.Errorf("a definition with the moved base's name must be untouched: %+v", s.Properties)
	}
}

func TestOpenAPIDiscriminatorObjectStillDecodes(t *testing.T) {
	var d Discriminator
	if err := json.Unmarshal([]byte(`{"propertyName":"k","mapping":{"a":"#/x"}}`), &d); err != nil {
		t.Fatal(err)
	}
	if d.PropertyName != "k" || d.Mapping["a"] != "#/x" {
		t.Errorf("object form = %+v", d)
	}
	var s Discriminator
	if err := json.Unmarshal([]byte(`"k"`), &s); err != nil || s.PropertyName != "k" {
		t.Errorf("string form = %+v, %v", s, err)
	}
}
