package openapi

import (
	"path/filepath"
	"strings"
	"testing"
)

func mustParse(t *testing.T, spec string) *Document {
	t.Helper()
	doc, err := Parse([]byte(spec))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return doc
}

func wrapPaths(paths, components string) string {
	return `{"openapi":"3.0.3","info":{"title":"t","version":"1"},"paths":` + paths + `,"components":` + components + `}`
}

func TestParseRejectsUnsupportedDocuments(t *testing.T) {
	cases := map[string]struct{ spec, want string }{
		"yaml":       {"openapi: 3.0.0", "not valid JSON"},
		"swagger 2":  {`{"swagger":"2.0","paths":{}}`, "swagger 2.0"},
		"openapi31":  {`{"openapi":"3.1.0","paths":{}}`, "3.1.0"},
		"no version": {`{"paths":{}}`, "missing version"},
		"no paths":   {`{"openapi":"3.0.0","paths":{}}`, "no paths"},
		"no ops":     {`{"openapi":"3.0.0","paths":{"/a":{}}}`, "no operations"},
		"bad schema ref": {wrapPaths(`{"/a":{"get":{"parameters":[{"name":"x","in":"query","schema":{"$ref":"#/components/schemas/Nope"}}]}}}`, `{}`),
			"unresolvable schema reference"},
		"external ref": {wrapPaths(`{"/a":{"get":{"parameters":[{"name":"x","in":"query","schema":{"$ref":"other.json#/X"}}]}}}`, `{}`),
			"unsupported schema reference"},
		"ref cycle": {wrapPaths(`{"/a":{"get":{}}}`, `{"schemas":{"A":{"$ref":"#/components/schemas/B"},"B":{"$ref":"#/components/schemas/A"}}}`),
			"cycle"},
		"bad location": {wrapPaths(`{"/a":{"get":{"parameters":[{"name":"x","in":"body"}]}}}`, `{}`), "invalid location"},
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

func TestOperationsAreSortedAndMergeParameters(t *testing.T) {
	doc := mustParse(t, wrapPaths(`{
		"/b/{id}/{undeclared}": {
			"parameters": [
				{"name":"id","in":"path","schema":{"type":"string"}},
				{"name":"shared","in":"query","schema":{"type":"string"}}
			],
			"post": {"parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"integer"}}]},
			"get": {"parameters":[
				{"name":"Accept","in":"header","schema":{"type":"string"}},
				{"name":"own","in":"query","schema":{"type":"boolean"}}
			]}
		},
		"/a": {"get": {}}
	}`, `{}`))

	var got []string
	for _, op := range doc.Operations {
		got = append(got, op.Method+" "+op.Path)
	}
	want := "GET /a,GET /b/{id}/{undeclared},POST /b/{id}/{undeclared}"
	if strings.Join(got, ",") != want {
		t.Fatalf("operations = %v, want %s", got, want)
	}

	get := doc.Operations[1]
	names := map[string]*Parameter{}
	for _, p := range get.Parameters {
		names[p.Name] = p
	}
	if _, ok := names["Accept"]; ok {
		t.Error("the Accept header parameter must be ignored")
	}
	if p := names["id"]; p == nil || !p.Required {
		t.Error("path parameters are always required")
	}
	if p := names["undeclared"]; p == nil || p.In != "path" || !p.Required {
		t.Errorf("undeclared template parameter was not synthesized: %+v", p)
	}
	if names["shared"] == nil || names["own"] == nil {
		t.Errorf("parameters not merged: %v", names)
	}

	post := doc.Operations[2]
	for _, p := range post.Parameters {
		if p.Name == "id" && p.Schema.Type != "integer" {
			t.Errorf("operation parameter must override the path-level one, got type %q", p.Schema.Type)
		}
	}
}

func TestBasePathComesFromMostSpecificServer(t *testing.T) {
	doc := mustParse(t, `{"openapi":"3.0.0","servers":[{"url":"https://x.io/{v}/api/","variables":{"v":{"default":"v2"}}}],
		"paths":{
			"/a":{"get":{}},
			"/b":{"servers":[{"url":"/path-level"}],"get":{},"post":{"servers":[{"url":"http://y.io/op-level"}]}}
		}}`)
	want := map[string]string{"GET /a": "/v2/api", "GET /b": "/path-level", "POST /b": "/op-level"}
	for _, op := range doc.Operations {
		if got := op.BasePath; got != want[op.Method+" "+op.Path] {
			t.Errorf("%s %s: base path %q, want %q", op.Method, op.Path, got, want[op.Method+" "+op.Path])
		}
	}
}

func TestRefsAndAllOfAreResolved(t *testing.T) {
	doc, err := Load(filepath.Join("..", "..", "testdata", "api.json"))
	if err != nil {
		t.Fatal(err)
	}
	newUser := doc.Components.Schemas["NewUser"]
	if newUser.AllOf != nil {
		t.Fatal("allOf should have been merged")
	}
	for _, name := range []string{"email", "age", "pet", "manager"} {
		if newUser.Properties[name] == nil {
			t.Errorf("merged schema lacks property %q", name)
		}
	}
	for _, name := range []string{"email", "age", "pet"} {
		found := false
		for _, r := range newUser.Required {
			found = found || r == name
		}
		if !found {
			t.Errorf("merged schema lacks required %q", name)
		}
	}

	// A recursive component resolves to a pointer cycle, not to a copy.
	employee := doc.Components.Schemas["Employee"]
	if employee.Properties["reports"].Items != employee {
		t.Error("recursive $ref should point back at the component")
	}

	for _, op := range doc.Operations {
		if op.Path == "/user" {
			if op.RequestBody == nil || op.RequestBody.Content["application/json"] == nil {
				t.Error("requestBody $ref not resolved")
			}
		}
	}
}

func TestVariantsApplyDiscriminator(t *testing.T) {
	doc, err := Load(filepath.Join("..", "..", "testdata", "api.json"))
	if err != nil {
		t.Fatal(err)
	}
	variants := doc.Components.Schemas["Pet"].Variants()
	if len(variants) != 2 {
		t.Fatalf("got %d variants", len(variants))
	}
	if got := variants[0].Properties["kind"].Enum[0]; got != "kitty" {
		t.Errorf("mapped discriminator = %v, want kitty", got)
	}
	if got := variants[1].Properties["kind"].Enum[0]; got != "Dog" {
		t.Errorf("implicit discriminator = %v, want the schema name Dog", got)
	}
	if doc.Components.Schemas["Cat"].Properties["kind"].Enum != nil {
		t.Error("building variants must not alter the shared branch schema")
	}
}

func TestMergeKeepsTighterBounds(t *testing.T) {
	doc := mustParse(t, wrapPaths(`{"/a":{"get":{"parameters":[{"name":"x","in":"query","schema":{
		"allOf":[
			{"type":"integer","minimum":1,"maximum":100,"enum":[1,5,7]},
			{"minimum":5,"maximum":200,"enum":[5,7,9]}
		]}}]}}}`, `{}`))
	s := doc.Operations[0].Parameters[0].Schema
	if Float(s.Minimum) != 5 || Float(s.Maximum) != 100 {
		t.Errorf("bounds = %v..%v, want 5..100", s.Minimum, s.Maximum)
	}
	if len(s.Enum) != 2 {
		t.Errorf("enum should be intersected, got %v", s.Enum)
	}
}

func TestIntKeepsPrecisionAndRoundsInward(t *testing.T) {
	doc := mustParse(t, wrapPaths(`{"/a":{"get":{"parameters":[{"name":"x","in":"query","schema":{
		"type":"integer","minimum":0.5,"maximum":9223372036854775807}}]}}}`, `{}`))
	s := doc.Operations[0].Parameters[0].Schema
	if got := Int(s.Maximum, false); got != 9223372036854775807 {
		t.Errorf("max = %d: int64 precision lost", got)
	}
	if got := Int(s.Minimum, true); got != 1 {
		t.Errorf("min = %d, want 0.5 rounded up to 1", got)
	}
}

func TestAllOfCombinesOverlappingProperties(t *testing.T) {
	doc := mustParse(t, wrapPaths(`{"/a":{"post":{"requestBody":{"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Refined"}}}}}}}`, `{"schemas":{
		"Refined":{"allOf":[
			{"$ref":"#/components/schemas/Base"},
			{"properties":{"n":{"minimum":5},"inner":{"allOf":[{"$ref":"#/components/schemas/Leaf"}]}}}
		]},
		"Base":{"type":"object","properties":{"n":{"type":"integer","maximum":9},"inner":{"type":"object"}}},
		"Leaf":{"required":["x"],"properties":{"x":{"type":"string"}}},
		"LoopA":{"type":"object","properties":{"next":{"$ref":"#/components/schemas/LoopA"}}},
		"LoopB":{"allOf":[{"$ref":"#/components/schemas/LoopA"},{"properties":{"next":{"$ref":"#/components/schemas/LoopB"}}}]}
	}}`))

	refined := doc.Components.Schemas["Refined"]
	n := refined.Properties["n"]
	if n.Type != "integer" || Float(n.Minimum) != 5 || Float(n.Maximum) != 9 {
		t.Errorf("n should carry the constraints of both definitions: %+v", n)
	}
	inner := refined.Properties["inner"]
	if inner.AllOf != nil || inner.Properties["x"] == nil || inner.Type != "object" {
		t.Errorf("a property's own allOf must survive the merge: %+v", inner)
	}
	if base := doc.Components.Schemas["Base"].Properties["n"]; base.Minimum != nil {
		t.Error("merging must not alter the source schema")
	}
}

func TestVariantsDropUnsatisfiableBranches(t *testing.T) {
	doc := mustParse(t, wrapPaths(`{"/a":{"post":{"requestBody":{"content":{"application/json":{"schema":{
		"type":"object","additionalProperties":false,"properties":{"a":{"type":"string"}},
		"oneOf":[{"required":["a"]},{"required":["ghost"]}]
	}}}}}}}`, `{}`))
	schema := doc.Operations[0].RequestBody.Content["application/json"].Schema
	variants := schema.Variants()
	if len(variants) != 1 || variants[0].Required[0] != "a" {
		t.Fatalf("variants = %+v", variants)
	}
}

func TestInlineBranchesRequireTheDiscriminator(t *testing.T) {
	doc := mustParse(t, wrapPaths(`{"/a":{"post":{"requestBody":{"content":{"application/json":{"schema":{
		"type":"object","properties":{"status":{"type":"string","enum":["a","b","c"]}},
		"discriminator":{"propertyName":"status"},
		"oneOf":[{"properties":{"status":{"enum":["a"]}},"required":["status"]},{"properties":{"status":{"enum":["b","c"]}}}]
	}}}}}}}`, `{}`))
	variants := doc.Operations[0].RequestBody.Content["application/json"].Schema.Variants()
	if len(variants) != 2 {
		t.Fatalf("got %d variants", len(variants))
	}
	second := variants[1]
	found := false
	for _, name := range second.Required {
		found = found || name == "status"
	}
	if !found {
		t.Error("the discriminator must be required")
	}
	if prop := second.Properties["status"]; prop.Type != "string" || len(prop.Enum) != 2 {
		t.Errorf("the branch must narrow the parent's property: %+v", prop)
	}
}

func TestEffectiveTypeInference(t *testing.T) {
	cases := map[string]string{
		`{"type":"boolean","minimum":1}`:             "boolean", // a declared type always wins
		`{"properties":{"a":{}}}`:                    "object",
		`{"required":["a"]}`:                         "object",
		`{"additionalProperties":{"type":"string"}}`: "object",
		`{"additionalProperties":false}`:             "object",
		`{"minProperties":1}`:                        "object",
		`{"maxProperties":1}`:                        "object",
		`{"items":{"type":"string"}}`:                "array",
		`{"minItems":1}`:                             "array",
		`{"maxItems":1}`:                             "array",
		`{"uniqueItems":true}`:                       "array",
		`{"format":"int32"}`:                         "integer",
		`{"format":"int64","minimum":1}`:             "integer",
		`{"format":"float"}`:                         "number",
		`{"format":"double"}`:                        "number",
		`{"minimum":1}`:                              "number",
		`{"maximum":1}`:                              "number",
		`{"multipleOf":2}`:                           "number",
		`{"enum":[1,2]}`:                             "integer",
		`{"enum":[1.5]}`:                             "number",
		`{"enum":[true]}`:                            "boolean",
		`{"enum":["a"]}`:                             "string",
		`{"enum":[null,3]}`:                          "integer",
		`{"enum":[[1]]}`:                             "array",
		`{"enum":[{"a":1}]}`:                         "object",
		`{"minLength":1}`:                            "string",
		`{"pattern":"^a"}`:                           "string",
		`{"format":"uuid"}`:                          "string",
		`{}`:                                         "string",
	}
	for schema, want := range cases {
		doc := mustParse(t, wrapPaths(`{"/a":{"get":{"parameters":[{"name":"x","in":"query","schema":`+schema+`}]}}}`, `{}`))
		if got := doc.Operations[0].Parameters[0].Schema.EffectiveType(); got != want {
			t.Errorf("%s: type %q, want %q", schema, got, want)
		}
	}
}

func TestXMLKeywordsAndPropertyOrder(t *testing.T) {
	doc, err := Parse([]byte(`{"openapi":"3.0.0","info":{"title":"t","version":"1"},
		"paths":{"/a":{"post":{"requestBody":{"content":{
			"application/xml":{"schema":{"$ref":"#/components/schemas/Body"}},
			"text/xml":{"schema":{"type":"string"}}
		}}}}},
		"components":{"schemas":{
			"Base":{"type":"object","properties":{"z":{"type":"string"},"y":{"type":"integer","maximum":9007199254740993}}},
			"Body":{"allOf":[{"$ref":"#/components/schemas/Base"},{"properties":{"b":{"type":"string"},"a":{"type":"string","xml":{"attribute":true,"name":"n","namespace":"urn:x","prefix":"p","wrapped":true}}}}],"xml":{"name":"doc"}}
		}}}`))
	if err != nil {
		t.Fatal(err)
	}

	content := doc.Operations[0].RequestBody.Content
	media := content["application/xml"]
	if media.SchemaName != "Body" || content["text/xml"].SchemaName != "" {
		t.Errorf("schema names %q and %q", media.SchemaName, content["text/xml"].SchemaName)
	}
	body := media.Schema
	if got := strings.Join(body.OrderedProperties(), ","); got != "z,y,b,a" {
		t.Errorf("order = %s", got)
	}
	if body.XML == nil || body.XML.Name != "doc" {
		t.Errorf("xml = %+v", body.XML)
	}
	want := XML{Name: "n", Namespace: "urn:x", Prefix: "p", Attribute: true, Wrapped: true}
	if got := body.Properties["a"].XML; got == nil || *got != want {
		t.Errorf("property xml = %+v", got)
	}
	// A custom decoder must keep numbers exact, as the document's own decoder does.
	if got := body.Properties["y"].Maximum.String(); got != "9007199254740993" {
		t.Errorf("maximum = %s", got)
	}
}

func TestOrderedPropertiesCoversNamesWithoutAnOrder(t *testing.T) {
	s := &Schema{
		Properties:    map[string]*Schema{"c": {}, "a": {}, "b": {}},
		PropertyOrder: []string{"b", "gone", "b"},
	}
	if got := strings.Join(s.OrderedProperties(), ","); got != "b,a,c" {
		t.Errorf("order = %s", got)
	}
}

func TestVariantsKeepPropertyOrderApart(t *testing.T) {
	doc, err := Parse([]byte(`{"openapi":"3.0.0","info":{"title":"t","version":"1"},
		"paths":{"/a":{"post":{"requestBody":{"content":{"application/json":{"schema":{
			"properties":{"base":{"type":"string"}},
			"oneOf":[{"properties":{"one":{"type":"string"}}},{"properties":{"two":{"type":"string"}}}]
		}}}}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	variants := doc.Operations[0].RequestBody.Content["application/json"].Schema.Variants()
	for i, want := range []string{"base,one", "base,two"} {
		if got := strings.Join(variants[i].OrderedProperties(), ","); got != want {
			t.Errorf("variant %d order = %s, want %s", i, got, want)
		}
	}
}
