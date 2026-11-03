package fuzzer

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"fuzzy/internal/generator"
	"fuzzy/internal/openapi"
)

func compileBoth(t *testing.T, spec string) []*Target {
	t.Helper()
	targets, skipped := compileSpec(t, "http://h", spec)
	if len(skipped) > 0 {
		t.Fatalf("skipped: %+v", skipped)
	}
	return targets
}

func request(t *testing.T, target *Target, iter int) (*http.Request, string, string) {
	t.Helper()
	req, input, err := target.build(context.Background(), rand.New(rand.NewPCG(7, uint64(iter))), iter)
	if err != nil {
		t.Fatal(err)
	}
	var body strings.Builder
	buf := make([]byte, 1<<16)
	for {
		n, err := req.Body.Read(buf)
		body.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return req, input, body.String()
}

// A spec written in either version must lead to the same requests, for the same seed.
func TestSwaggerAndOpenAPISendTheSameRequests(t *testing.T) {
	const swagger = `{"swagger":"2.0","info":{"title":"t","version":"1"},"host":"prod.example.com","basePath":"/v2",
	"consumes":["application/json"],
	"parameters":{"limit":{"name":"limit","in":"query","type":"integer","minimum":1,"maximum":50}},
	"definitions":{
		"Base":{"type":"object","required":["id"],"properties":{"id":{"type":"integer","format":"int64","minimum":1},"next":{"$ref":"#/definitions/Item"}}},
		"Item":{"allOf":[{"$ref":"#/definitions/Base"},{"type":"object","required":["name"],"properties":{
			"name":{"type":"string","minLength":2,"maxLength":5},
			"kind":{"type":"string","enum":["a","b"]},
			"tags":{"type":"array","minItems":1,"maxItems":3,"items":{"type":"string","format":"uuid"}}
		}}]}
	},
	"paths":{
		"/items/{id}":{
			"parameters":[{"name":"id","in":"path","required":true,"type":"integer","minimum":1,"maximum":9}],
			"get":{"parameters":[
				{"$ref":"#/parameters/limit"},
				{"name":"csv","in":"query","type":"array","items":{"type":"integer"}},
				{"name":"multi","in":"query","required":true,"type":"array","collectionFormat":"multi","minItems":1,"items":{"type":"string"}},
				{"name":"X-Trace","in":"header","required":true,"type":"string","format":"uuid"}
			]},
			"put":{"parameters":[{"name":"item","in":"body","required":true,"schema":{"$ref":"#/definitions/Item"}}]}
		},
		"/form":{"post":{"parameters":[
			{"name":"a","in":"formData","required":true,"type":"string","maxLength":4},
			{"name":"list","in":"formData","required":true,"type":"array","minItems":1,"items":{"type":"integer"}}
		]}},
		"/upload":{"post":{"consumes":["multipart/form-data"],"parameters":[
			{"name":"doc","in":"formData","required":true,"type":"file"},
			{"name":"note","in":"formData","type":"string"}
		]}}
	}}`
	const oas = `{"openapi":"3.0.3","info":{"title":"t","version":"1"},"servers":[{"url":"https://prod.example.com/v2"}],
	"components":{
		"parameters":{"limit":{"name":"limit","in":"query","schema":{"type":"integer","minimum":1,"maximum":50}}},
		"schemas":{
			"Base":{"type":"object","required":["id"],"properties":{"id":{"type":"integer","format":"int64","minimum":1},"next":{"$ref":"#/components/schemas/Item"}}},
			"Item":{"allOf":[{"$ref":"#/components/schemas/Base"},{"type":"object","required":["name"],"properties":{
				"name":{"type":"string","minLength":2,"maxLength":5},
				"kind":{"type":"string","enum":["a","b"]},
				"tags":{"type":"array","minItems":1,"maxItems":3,"items":{"type":"string","format":"uuid"}}
			}}]}
		}
	},
	"paths":{
		"/items/{id}":{
			"parameters":[{"name":"id","in":"path","required":true,"schema":{"type":"integer","minimum":1,"maximum":9}}],
			"get":{"parameters":[
				{"$ref":"#/components/parameters/limit"},
				{"name":"csv","in":"query","explode":false,"schema":{"type":"array","items":{"type":"integer"}}},
				{"name":"multi","in":"query","required":true,"explode":true,"schema":{"type":"array","minItems":1,"items":{"type":"string"}}},
				{"name":"X-Trace","in":"header","required":true,"schema":{"type":"string","format":"uuid"}}
			]},
			"put":{"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/Item"}}}}}
		},
		"/form":{"post":{"requestBody":{"required":true,"content":{"application/x-www-form-urlencoded":{
			"schema":{"type":"object","required":["a","list"],"properties":{
				"a":{"type":"string","maxLength":4},
				"list":{"type":"array","minItems":1,"items":{"type":"integer"}}
			}},
			"encoding":{"list":{"explode":false}}
		}}}}},
		"/upload":{"post":{"requestBody":{"required":true,"content":{"multipart/form-data":{
			"schema":{"type":"object","required":["doc"],"properties":{
				"doc":{"type":"string","format":"binary"},
				"note":{"type":"string"}
			}}
		}}}}}
	}}`

	fromSwagger, fromOpenAPI := compileBoth(t, swagger), compileBoth(t, oas)
	if len(fromSwagger) != len(fromOpenAPI) || len(fromSwagger) != 4 {
		t.Fatalf("targets: swagger %d, openapi %d, want 4", len(fromSwagger), len(fromOpenAPI))
	}
	for i := range fromSwagger {
		a, b := fromSwagger[i], fromOpenAPI[i]
		for iter := range 200 {
			reqA, inputA, bodyA := request(t, a, iter)
			reqB, inputB, bodyB := request(t, b, iter)
			if reqA.Method != reqB.Method || reqA.URL.String() != reqB.URL.String() {
				t.Fatalf("%d: %s %s vs %s %s", iter, reqA.Method, reqA.URL, reqB.Method, reqB.URL)
			}
			if !strings.HasPrefix(reqA.URL.Path, "/v2/") || reqA.URL.Host != "h" {
				t.Fatalf("URL %s must use the spec's base path and the given host", reqA.URL)
			}
			if inputA != inputB {
				t.Fatalf("%s input:\n%s\nvs\n%s", reqA.URL.Path, inputA, inputB)
			}
			if reqA.Header.Get("X-Trace") != reqB.Header.Get("X-Trace") {
				t.Fatalf("%s headers: %v vs %v", reqA.URL.Path, reqA.Header, reqB.Header)
			}
			// A multipart body and its Content-Type embed a random boundary.
			if a.body != nil && a.body.kind != bodyMultipart {
				if bodyA != bodyB || reqA.Header.Get("Content-Type") != reqB.Header.Get("Content-Type") {
					t.Fatalf("%s body:\n%s\nvs\n%s", reqA.URL.Path, bodyA, bodyB)
				}
			}
		}
	}
}

func TestSwaggerCollectionFormatsOnTheWire(t *testing.T) {
	const ab = `"type":"array","minItems":2,"maxItems":2,"items":{"type":"string","enum":["a b"]}`
	targets := compileBoth(t, `{"swagger":"2.0","paths":{"/q/{csv}/{ssv}/{tsv}/{pipes}":{"post":{
		"parameters":[
			{"name":"csv","in":"path","required":true,`+ab+`},
			{"name":"ssv","in":"path","required":true,`+ab+`,"collectionFormat":"ssv"},
			{"name":"tsv","in":"path","required":true,`+ab+`,"collectionFormat":"tsv"},
			{"name":"pipes","in":"path","required":true,`+ab+`,"collectionFormat":"pipes"},
			{"name":"qcsv","in":"query","required":true,`+ab+`},
			{"name":"qssv","in":"query","required":true,`+ab+`,"collectionFormat":"ssv"},
			{"name":"qtsv","in":"query","required":true,`+ab+`,"collectionFormat":"tsv"},
			{"name":"qpipes","in":"query","required":true,`+ab+`,"collectionFormat":"pipes"},
			{"name":"qmulti","in":"query","required":true,`+ab+`,"collectionFormat":"multi"},
			{"name":"X-Csv","in":"header","required":true,`+ab+`},
			{"name":"X-Pipes","in":"header","required":true,`+ab+`,"collectionFormat":"pipes"},
			{"name":"X-Tsv","in":"header","required":true,`+ab+`,"collectionFormat":"tsv"},
			{"name":"fpipes","in":"formData","required":true,`+ab+`,"collectionFormat":"pipes"},
			{"name":"fmulti","in":"formData","required":true,`+ab+`,"collectionFormat":"multi"}
		]
	}}}}`)
	req, _, body := request(t, targets[0], 0)

	if want := "/q/a%20b,a%20b/a%20b%20a%20b/a%20b%09a%20b/a%20b%7Ca%20b"; req.URL.EscapedPath() != want {
		t.Errorf("path = %s, want %s", req.URL.EscapedPath(), want)
	}
	for _, want := range []string{"qcsv=a+b,a+b", "qssv=a+b%20a+b", "qtsv=a+b%09a+b", "qpipes=a+b%7Ca+b", "qmulti=a+b&qmulti=a+b"} {
		if !strings.Contains(req.URL.RawQuery, want) {
			t.Errorf("query %q misses %q", req.URL.RawQuery, want)
		}
	}
	for name, want := range map[string]string{"X-Csv": "a b,a b", "X-Pipes": "a b|a b", "X-Tsv": "a b\ta b"} {
		if got := req.Header.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if want := "fmulti=a+b&fmulti=a+b&fpipes=a+b%7Ca+b"; body != want {
		t.Errorf("form = %q, want %q", body, want)
	}
	if got := req.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q", got)
	}
}

func TestSwaggerFileIsSentAsAFilePart(t *testing.T) {
	targets := compileBoth(t, `{"swagger":"2.0","paths":{"/u":{"post":{"parameters":[
		{"name":"doc","in":"formData","required":true,"type":"file"},
		{"name":"note","in":"formData","required":true,"type":"string"}
	]}}}}`)
	req, _, body := request(t, targets[0], 0)
	mediaType, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		t.Fatalf("Content-Type = %q, %v", req.Header.Get("Content-Type"), err)
	}
	form, err := multipart.NewReader(strings.NewReader(body), params["boundary"]).ReadForm(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(form.File["doc"]) != 1 || len(form.Value["note"]) != 1 {
		t.Errorf("files %v, values %v, want doc as a file and note as a field", form.File, form.Value)
	}
}

func TestSwaggerUnsupportedConsumesSkipsARequiredBody(t *testing.T) {
	_, skipped := compileSpec(t, "http://h", `{"swagger":"2.0","paths":{"/a":{"post":{"consumes":["application/x-protobuf"],
		"parameters":[{"name":"b","in":"body","required":true,"schema":{"type":"string"}}]}}}}`)
	if len(skipped) != 1 || !strings.Contains(skipped[0].Reason, "application/x-protobuf") {
		t.Errorf("skipped = %+v", skipped)
	}
}

// Every request must carry a tag and the fields of the subtype that tag names.
func TestSwaggerPolymorphicBodiesAreValid(t *testing.T) {
	targets := compileBoth(t, `{"swagger":"2.0","definitions":{
		"Pet":{"type":"object","discriminator":"petType","required":["name","petType"],"properties":{
			"name":{"type":"string"},"petType":{"type":"string"},"friend":{"$ref":"#/definitions/Pet"}
		}},
		"Cat":{"allOf":[{"$ref":"#/definitions/Pet"},{"type":"object","required":["lives"],"properties":{"lives":{"type":"integer"}}}]},
		"Dog":{"allOf":[{"$ref":"#/definitions/Pet"},{"type":"object","required":["bark"],"properties":{"bark":{"type":"string"}}}]}
	},"paths":{"/pets":{"post":{"parameters":[{"name":"b","in":"body","required":true,"schema":{"$ref":"#/definitions/Pet"}}]}}}}`)

	seen := map[string]int{}
	var check func(t *testing.T, value map[string]any)
	check = func(t *testing.T, value map[string]any) {
		kind, _ := value["petType"].(string)
		seen[kind]++
		if _, ok := value["name"]; !ok {
			t.Errorf("no base field: %v", value)
		}
		switch kind {
		case "Cat":
			if _, ok := value["lives"]; !ok {
				t.Errorf("a Cat without lives: %v", value)
			}
		case "Dog":
			if _, ok := value["bark"]; !ok {
				t.Errorf("a Dog without bark: %v", value)
			}
		default:
			t.Errorf("petType = %q, want the name of a concrete definition", kind)
		}
		if friend, ok := value["friend"].(map[string]any); ok {
			check(t, friend)
		}
	}
	for iter := range 300 {
		_, _, body := request(t, targets[0], iter)
		var value map[string]any
		if err := json.Unmarshal([]byte(body), &value); err != nil {
			t.Fatalf("%s: %v", body, err)
		}
		check(t, value)
	}
	if seen["Cat"] == 0 || seen["Dog"] == 0 {
		t.Errorf("tags sent = %v, want both subtypes", seen)
	}
}

func TestSwaggerPolymorphicXMLBodyKeepsTheRootName(t *testing.T) {
	doc, err := openapi.Parse([]byte(`{"swagger":"2.0","consumes":["application/xml"],"definitions":{
		"Pet":{"type":"object","discriminator":"petType","required":["petType"],"properties":{"petType":{"type":"string"}}},
		"Cat":{"allOf":[{"$ref":"#/definitions/Pet"},{"type":"object"}]}
	},"paths":{"/pets":{"post":{"parameters":[{"name":"b","in":"body","required":true,"schema":{"$ref":"#/definitions/Pet"}}]}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse("http://h")
	targets, _ := Compile(doc, generator.New(generator.DefaultBounds(), nil, 1), u)
	_, _, body := request(t, targets[0], 0)
	if !strings.Contains(body, "<Pet>") || !strings.Contains(body, "<petType>Cat</petType>") {
		t.Errorf("body = %s", body)
	}
}
