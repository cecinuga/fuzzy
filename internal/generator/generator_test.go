package generator

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"fuzzy/internal/openapi"
)

// parseSchema loads a schema through the real loader, so $ref and allOf work.
func parseSchema(t *testing.T, schema, components string) *openapi.Schema {
	t.Helper()
	if components == "" {
		components = "{}"
	}
	doc, err := openapi.Parse([]byte(`{"openapi":"3.0.0","paths":{"/a":{"post":{"requestBody":{"content":{"application/json":{"schema":` +
		schema + `}}}}}},"components":{"schemas":` + components + `}}`))
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	return doc.Operations[0].RequestBody.Content["application/json"].Schema
}

// validate checks v against s independently of how v was generated.
func validate(s *openapi.Schema, v any) error {
	if v == nil {
		if slices.Contains(s.Enum, nil) || (len(s.Enum) == 0 && s.Nullable) {
			return nil
		}
		return fmt.Errorf("null is not allowed here")
	}
	if len(s.Enum) > 0 {
		if !slices.ContainsFunc(s.Enum, func(e any) bool { return fmt.Sprint(e) == fmt.Sprint(v) }) {
			return fmt.Errorf("%v not in enum %v", v, s.Enum)
		}
		return nil
	}
	if variants := s.Variants(); len(variants) > 0 {
		var last error
		for _, variant := range variants {
			if last = validate(variant, v); last == nil {
				return nil
			}
		}
		return fmt.Errorf("no variant matches %v: %w", v, last)
	}

	switch s.EffectiveType() {
	case "integer":
		i, ok := v.(int64)
		if !ok {
			return fmt.Errorf("%v (%T) is not an integer", v, v)
		}
		return validateNumber(s, float64(i))
	case "number":
		f, ok := v.(float64)
		if !ok {
			return fmt.Errorf("%v (%T) is not a number", v, v)
		}
		return validateNumber(s, f)
	case "boolean":
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("%v is not a boolean", v)
		}
	case "string":
		str, ok := v.(string)
		if !ok {
			return fmt.Errorf("%v is not a string", v)
		}
		n := utf8.RuneCountInString(str)
		if s.MinLength != nil && n < *s.MinLength {
			return fmt.Errorf("%q shorter than %d", str, *s.MinLength)
		}
		if s.MaxLength != nil && n > *s.MaxLength {
			return fmt.Errorf("%q longer than %d", str, *s.MaxLength)
		}
		if s.Pattern != "" && !regexp.MustCompile(s.Pattern).MatchString(str) {
			return fmt.Errorf("%q does not match %s", str, s.Pattern)
		}
		if !fitsFormat(str, s.Format) {
			return fmt.Errorf("%q is not a valid %s", str, s.Format)
		}
	case "array":
		items, ok := v.([]any)
		if !ok {
			return fmt.Errorf("%v is not an array", v)
		}
		if s.MinItems != nil && len(items) < *s.MinItems {
			return fmt.Errorf("%d items, want at least %d", len(items), *s.MinItems)
		}
		if s.MaxItems != nil && len(items) > *s.MaxItems {
			return fmt.Errorf("%d items, want at most %d", len(items), *s.MaxItems)
		}
		seen := map[string]bool{}
		for _, item := range items {
			if s.UniqueItems && seen[fmt.Sprint(item)] {
				return fmt.Errorf("duplicate item %v", item)
			}
			seen[fmt.Sprint(item)] = true
			if err := validate(s.Items, item); err != nil {
				return err
			}
		}
	case "object":
		obj, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("%v is not an object", v)
		}
		for _, name := range s.Required {
			if _, present := obj[name]; !present {
				return fmt.Errorf("missing required %q in %v", name, obj)
			}
		}
		if s.MinProperties != nil && len(obj) < *s.MinProperties {
			return fmt.Errorf("%d properties, want at least %d", len(obj), *s.MinProperties)
		}
		if s.MaxProperties != nil && len(obj) > *s.MaxProperties {
			return fmt.Errorf("%d properties, want at most %d", len(obj), *s.MaxProperties)
		}
		for name, value := range obj {
			prop := s.Properties[name]
			if prop == nil {
				if extra := s.AdditionalProperties; extra != nil && extra.Schema != nil {
					if err := validate(extra.Schema, value); err != nil {
						return fmt.Errorf("%s: %w", name, err)
					}
					continue
				}
				return fmt.Errorf("unknown property %q", name)
			}
			if prop.ReadOnly {
				return fmt.Errorf("readOnly property %q was sent", name)
			}
			if err := validate(prop, value); err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
		}
	}
	return nil
}

func validateNumber(s *openapi.Schema, f float64) error {
	if s.Minimum != nil {
		if lo := openapi.Float(s.Minimum); f < lo || (s.ExclusiveMinimum && f == lo) {
			return fmt.Errorf("%v below minimum %v", f, lo)
		}
	}
	if s.Maximum != nil {
		if hi := openapi.Float(s.Maximum); f > hi || (s.ExclusiveMaximum && f == hi) {
			return fmt.Errorf("%v above maximum %v", f, hi)
		}
	}
	if s.MultipleOf != nil && !isMultiple(f, openapi.Float(s.MultipleOf)) {
		return fmt.Errorf("%v not a multiple of %v", f, s.MultipleOf)
	}
	return nil
}

func TestGeneratedValuesSatisfyTheirSchema(t *testing.T) {
	components := `{
		"Node":{"type":"object","required":["value"],"properties":{"value":{"type":"integer"},"next":{"$ref":"#/components/schemas/Node"},"kids":{"type":"array","items":{"$ref":"#/components/schemas/Node"}}}},
		"Cat":{"type":"object","required":["kind","lives"],"properties":{"kind":{"type":"string"},"lives":{"type":"integer","minimum":1,"maximum":9}}},
		"Dog":{"type":"object","required":["kind"],"properties":{"kind":{"type":"string"},"bark":{"type":"boolean"}}}
	}`
	cases := map[string]string{
		"int":            `{"type":"integer"}`,
		"int range":      `{"type":"integer","minimum":5,"maximum":7}`,
		"int exclusive":  `{"type":"integer","minimum":5,"maximum":7,"exclusiveMinimum":true,"exclusiveMaximum":true}`,
		"int multiple":   `{"type":"integer","minimum":-100,"maximum":100,"multipleOf":7}`,
		"int above cfg":  `{"type":"integer","minimum":5000}`,
		"int below cfg":  `{"type":"integer","maximum":-5000}`,
		"int32 full":     `{"type":"integer","format":"int32","minimum":2147483640}`,
		"int64 huge":     `{"type":"integer","minimum":-9223372036854775808,"maximum":9223372036854775807}`,
		"float":          `{"type":"number"}`,
		"float tight":    `{"type":"number","minimum":0.0001,"maximum":0.0002}`,
		"float excl":     `{"type":"number","minimum":0,"maximum":1,"exclusiveMinimum":true,"exclusiveMaximum":true}`,
		"float multiple": `{"type":"number","minimum":0,"maximum":1,"multipleOf":0.1}`,
		"bool":           `{"type":"boolean"}`,
		"string":         `{"type":"string"}`,
		"string len":     `{"type":"string","minLength":3,"maxLength":4}`,
		"string long":    `{"type":"string","minLength":50}`,
		"string empty":   `{"type":"string","maxLength":0}`,
		"enum":           `{"type":"string","enum":["a","b"]}`,
		"enum nullable":  `{"type":"string","enum":["a","b"],"nullable":true}`,
		"enum with null": `{"type":"string","enum":["a",null],"nullable":true}`,
		"int enum":       `{"type":"integer","enum":[1,2,3]}`,
		"pattern sku":    `{"type":"string","pattern":"^[A-Z]{3}-[0-9]{4}$"}`,
		"pattern negate": `{"type":"string","pattern":"^[^/]+$"}`,
		"pattern alt":    `{"type":"string","pattern":"^(foo|bar)+\\d?$","maxLength":30}`,
		"pattern free":   `{"type":"string","pattern":"abc"}`,
		"pattern padded": `{"type":"string","pattern":"^a","minLength":5,"maxLength":6}`,
		"pattern fixed":  `{"type":"string","pattern":"^[a-f0-9]{32}$"}`,
		"uuid":           `{"type":"string","format":"uuid"}`,
		"date":           `{"type":"string","format":"date"}`,
		"date-time":      `{"type":"string","format":"date-time"}`,
		"email":          `{"type":"string","format":"email"}`,
		"uri":            `{"type":"string","format":"uri"}`,
		"hostname":       `{"type":"string","format":"hostname"}`,
		"ipv4":           `{"type":"string","format":"ipv4"}`,
		"ipv6":           `{"type":"string","format":"ipv6"}`,
		"byte":           `{"type":"string","format":"byte"}`,
		"nullable":       `{"type":"string","nullable":true}`,
		"array":          `{"type":"array","items":{"type":"integer"},"minItems":2,"maxItems":3}`,
		"array unique":   `{"type":"array","items":{"type":"string","enum":["x","y","z"]},"minItems":3,"uniqueItems":true}`,
		"object":         `{"type":"object","required":["a"],"properties":{"a":{"type":"integer"},"b":{"type":"string"},"r":{"type":"string","readOnly":true}}}`,
		"min props":      `{"type":"object","minProperties":2,"properties":{"a":{"type":"integer"},"b":{"type":"integer"},"c":{"type":"integer"}}}`,
		"max props":      `{"type":"object","maxProperties":1,"required":["a"],"properties":{"a":{"type":"integer"},"b":{"type":"integer"},"c":{"type":"integer"}}}`,
		"map type":       `{"type":"object","minProperties":3,"additionalProperties":{"type":"integer","minimum":5,"maximum":6}}`,
		"allOf":          `{"allOf":[{"type":"object","required":["a"],"properties":{"a":{"type":"integer"}}},{"required":["b"],"properties":{"b":{"type":"boolean"}}}]}`,
		"anyOf":          `{"anyOf":[{"type":"integer"},{"type":"boolean"}]}`,
		"oneOf parent":   `{"type":"object","required":["p"],"properties":{"p":{"type":"integer"}},"oneOf":[{"required":["a"],"properties":{"a":{"type":"string"}}},{"required":["b"],"properties":{"b":{"type":"boolean"}}}]}`,
		"discriminator":  `{"oneOf":[{"$ref":"#/components/schemas/Cat"},{"$ref":"#/components/schemas/Dog"}],"discriminator":{"propertyName":"kind"}}`,
		"recursive":      `{"$ref":"#/components/schemas/Node"}`,
		"untyped":        `{}`,
	}

	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			schema := parseSchema(t, spec, components)
			source := New(DefaultBounds(), nil, 1).Compile(schema, InBody)
			r := rand.New(rand.NewPCG(1, 2))
			for iter := range 2000 {
				if err := validate(schema, source.Generate(r, iter)); err != nil {
					t.Fatalf("iteration %d: %v", iter, err)
				}
			}
		})
	}
}

func TestRandomValuesRespectBounds(t *testing.T) {
	bounds := Bounds{StringMin: 2, StringMax: 4, IntMin: -3, IntMax: 3, FloatMin: 1.5, FloatMax: 2.5, Charset: CharsetAlnum}
	g := New(bounds, nil, 1)
	r := rand.New(rand.NewPCG(1, 2))

	str := g.Compile(parseSchema(t, `{"type":"string"}`, ""), InBody)
	integer := g.Compile(parseSchema(t, `{"type":"integer"}`, ""), InBody)
	float := g.Compile(parseSchema(t, `{"type":"number"}`, ""), InBody)
	sawMin, sawMax := false, false

	for iter := range 3000 {
		s := str.Generate(r, iter).(string)
		if n := utf8.RuneCountInString(s); n < 2 || n > 4 {
			t.Fatalf("string %q has length %d", s, n)
		}
		for _, c := range s {
			if c > 127 {
				t.Fatalf("alnum string has non-ASCII rune in %q", s)
			}
		}
		i := integer.Generate(r, iter).(int64)
		if i < -3 || i > 3 {
			t.Fatalf("int %d out of bounds", i)
		}
		sawMin, sawMax = sawMin || i == -3, sawMax || i == 3
		if f := float.Generate(r, iter).(float64); f < 1.5 || f > 2.5 {
			t.Fatalf("float %v out of bounds", f)
		}
	}
	if !sawMin || !sawMax {
		t.Error("integer bounds should be inclusive")
	}
}

func TestSchemaWinsOverDisjointBounds(t *testing.T) {
	cases := []struct {
		lo, hi limit[int64]
		want   [2]int64
	}{
		{limit[int64]{5000, true}, limit[int64]{}, [2]int64{5000, 6000}},
		{limit[int64]{}, limit[int64]{-5000, true}, [2]int64{-6000, -5000}},
		{limit[int64]{2000, true}, limit[int64]{3000, true}, [2]int64{2000, 3000}},
		{limit[int64]{500, true}, limit[int64]{}, [2]int64{500, 1000}},
		{limit[int64]{math.MaxInt64, true}, limit[int64]{}, [2]int64{math.MaxInt64, math.MaxInt64}},
	}
	for _, tc := range cases {
		from, to := effectiveRange(tc.lo, tc.hi, 0, 1000)
		if from != tc.want[0] || to != tc.want[1] {
			t.Errorf("effectiveRange(%v, %v) = %d..%d, want %v", tc.lo, tc.hi, from, to, tc.want)
		}
	}
}

func TestUnicodeCharsetProducesNonASCII(t *testing.T) {
	source := New(DefaultBounds(), nil, 1).Compile(parseSchema(t, `{"type":"string"}`, ""), InBody)
	r := rand.New(rand.NewPCG(1, 2))
	for iter := range 50 {
		s := source.Generate(r, iter).(string)
		if !utf8.ValidString(s) {
			t.Fatalf("invalid UTF-8: %q", s)
		}
		if strings.IndexFunc(s, func(c rune) bool { return c > 127 }) >= 0 {
			return
		}
	}
	t.Fatal("no non-ASCII rune in 50 strings")
}

func TestHeaderStringsStayASCII(t *testing.T) {
	source := New(DefaultBounds(), nil, 1).Compile(parseSchema(t, `{"type":"string"}`, ""), InHeader)
	r := rand.New(rand.NewPCG(1, 2))
	for iter := range 2000 {
		if s := source.Generate(r, iter).(string); strings.IndexFunc(s, func(c rune) bool { return c > 127 }) >= 0 {
			t.Fatalf("non-ASCII header value %q", s)
		}
	}
}

func TestPathStringsAreNeverEmpty(t *testing.T) {
	bounds := DefaultBounds()
	bounds.StringMin = 0
	source := New(bounds, nil, 1).Compile(parseSchema(t, `{"type":"string"}`, ""), InPath)
	r := rand.New(rand.NewPCG(1, 2))
	for iter := range 2000 {
		if source.Generate(r, iter) == "" {
			t.Fatal("an empty path value changes the route")
		}
	}
}

func writeDict(t *testing.T, content string) *Dictionary {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dict.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := LoadDictionary(path)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestLoadDictionaryClassifiesLines(t *testing.T) {
	d := writeDict(t, "admin\r\n42\n3.5\ntrue\n\n   \nadmin\ninf\nNaN\n0x10\n1e3\n99999999999999999999\n")

	wantStrings := []string{"admin", "42", "3.5", "true", "inf", "NaN", "0x10", "1e3", "99999999999999999999"}
	if !slices.Equal(d.Strings, wantStrings) {
		t.Errorf("Strings = %q", d.Strings)
	}
	if !slices.Equal(d.Ints, []int64{42}) {
		t.Errorf("Ints = %v", d.Ints)
	}
	if !slices.Equal(d.Floats, []float64{42, 3.5, 1000, 99999999999999999999}) {
		t.Errorf("Floats = %v (inf, NaN and hex must not count: they are not JSON numbers)", d.Floats)
	}
	if !slices.Equal(d.Bools, []bool{true}) {
		t.Errorf("Bools = %v", d.Bools)
	}
}

func TestLoadDictionaryRejectsEmptyAndMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.txt")
	if err := os.WriteFile(path, []byte("\n  \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDictionary(path); err == nil {
		t.Error("an empty dictionary is a mistake worth reporting")
	}
	if _, err := LoadDictionary(path + ".nope"); err == nil {
		t.Error("a missing dictionary must fail")
	}
}

func TestDictionaryOnlyYieldsValuesThatFit(t *testing.T) {
	dict := writeDict(t, strings.Join([]string{
		"admin", "toolongvalue", "42", "2000", "2.5", "true", "../etc", ".", "a@b.io",
		"6ba7b810-9dad-11d1-80b4-00c04fd430c8", "2024-02-30", "2024-02-29", "ABC-1234", "héllo",
	}, "\n"))

	cases := []struct {
		name   string
		schema string
		loc    Location
		want   []any
	}{
		{"short string", `{"type":"string","maxLength":5}`, InBody, []any{"admin", "42", "2000", "2.5", "true", ".", "héllo"}},
		{"int in range", `{"type":"integer","minimum":0,"maximum":1000}`, InBody, []any{int64(42)}},
		{"int no range", `{"type":"integer"}`, InBody, []any{int64(42), int64(2000)}},
		{"int multiple", `{"type":"integer","multipleOf":100}`, InBody, []any{int64(2000)}},
		{"float", `{"type":"number","maximum":100}`, InBody, []any{float64(42), 2.5}},
		{"bool", `{"type":"boolean"}`, InBody, []any{true}},
		{"email", `{"type":"string","format":"email"}`, InBody, []any{"a@b.io"}},
		{"uuid", `{"type":"string","format":"uuid"}`, InBody, []any{"6ba7b810-9dad-11d1-80b4-00c04fd430c8"}},
		{"date is real", `{"type":"string","format":"date"}`, InBody, []any{"2024-02-29"}},
		{"pattern", `{"type":"string","pattern":"^[A-Z]{3}-[0-9]{4}$"}`, InBody, []any{"ABC-1234"}},
		{"enum beats dict", `{"type":"string","enum":["x"]}`, InBody, []any{"x"}},
		{"path skips dots", `{"type":"string","maxLength":1}`, InPath, nil},
		{"header is ascii", `{"type":"string","maxLength":5}`, InHeader, []any{"admin", "42", "2000", "2.5", "true", "."}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := New(DefaultBounds(), dict, 1).Compile(parseSchema(t, tc.schema, ""), tc.loc)
			r := rand.New(rand.NewPCG(1, 2))
			seen := map[any]bool{}
			for iter := range 200 {
				seen[source.Generate(r, iter)] = true
			}
			if tc.want == nil {
				for v := range seen {
					if v == "." {
						t.Errorf("%q must not be used as a path value", v)
					}
				}
				return
			}
			if len(seen) != len(tc.want) {
				t.Errorf("generated %v, want exactly %v", seen, tc.want)
			}
			for _, v := range tc.want {
				if !seen[v] {
					t.Errorf("value %v was never used (generated %v)", v, seen)
				}
			}
		})
	}
}

func TestDictionaryFallsBackToRandomForUncoveredTypes(t *testing.T) {
	dict := writeDict(t, "only\nstrings\nhere\n")
	g := New(DefaultBounds(), dict, 1)
	r := rand.New(rand.NewPCG(1, 2))

	if _, ok := g.Compile(parseSchema(t, `{"type":"integer"}`, ""), InBody).Generate(r, 0).(int64); !ok {
		t.Error("an integer must be generated when the dictionary has none")
	}
	if _, ok := g.Compile(parseSchema(t, `{"type":"number"}`, ""), InBody).Generate(r, 0).(float64); !ok {
		t.Error("a float must be generated when the dictionary has none")
	}
	if _, ok := g.Compile(parseSchema(t, `{"type":"boolean"}`, ""), InBody).Generate(r, 0).(bool); !ok {
		t.Error("a boolean must be generated when the dictionary has none")
	}
	source := g.Compile(parseSchema(t, `{"type":"string","maxLength":2}`, ""), InBody)
	for iter := range 100 {
		if s := source.Generate(r, iter).(string); utf8.RuneCountInString(s) > 2 {
			t.Fatalf("no word fits maxLength 2, so a random fit was expected, got %q", s)
		}
	}
}

func TestEveryDictionaryValueGetsItsTurn(t *testing.T) {
	dict := writeDict(t, "a\nb\nc\nd\ne\nf\ng\n")
	source := New(DefaultBounds(), dict, 99).Compile(parseSchema(t, `{"type":"string"}`, ""), InBody)
	r := rand.New(rand.NewPCG(1, 2))
	seen := map[any]bool{}
	for iter := range 7 {
		seen[source.Generate(r, iter)] = true
	}
	if len(seen) != 7 {
		t.Fatalf("7 iterations visited only %v", seen)
	}
}

func TestSameSeedSameValues(t *testing.T) {
	schema := parseSchema(t, `{"type":"object","required":["a","b","c"],"properties":{"a":{"type":"string"},"b":{"type":"number"},"c":{"type":"array","items":{"type":"integer"}}}}`, "")
	run := func() string {
		source := New(DefaultBounds(), nil, 7).Compile(schema, InBody)
		var out strings.Builder
		for iter := range 50 {
			r := rand.New(rand.NewPCG(7, uint64(iter)))
			encoded, _ := json.Marshal(source.Generate(r, iter))
			out.Write(encoded)
		}
		return out.String()
	}
	if run() != run() {
		t.Fatal("equal seeds must produce equal values")
	}
}

func TestUnsupportedPatternWarnsAndStillGenerates(t *testing.T) {
	g := New(DefaultBounds(), nil, 1)
	schema := parseSchema(t, `{"type":"string","pattern":"^(?!admin).*$"}`, "")
	source := g.Compile(schema, InBody)
	if len(g.Warnings()) != 1 {
		t.Fatalf("warnings = %v", g.Warnings())
	}
	if _, ok := source.Generate(rand.New(rand.NewPCG(1, 2)), 0).(string); !ok {
		t.Fatal("a string should still be generated")
	}
}

func TestBoundsValidate(t *testing.T) {
	bad := map[string]func(*Bounds){
		"negative length": func(b *Bounds) { b.StringMin = -1 },
		"string inverted": func(b *Bounds) { b.StringMin, b.StringMax = 5, 2 },
		"int inverted":    func(b *Bounds) { b.IntMin, b.IntMax = 5, 2 },
		"float inverted":  func(b *Bounds) { b.FloatMin, b.FloatMax = 5, 2 },
		"float NaN":       func(b *Bounds) { b.FloatMin = math.NaN() },
		"float infinite":  func(b *Bounds) { b.FloatMax = math.Inf(1) },
		"float -infinite": func(b *Bounds) { b.FloatMin = math.Inf(-1) },
		"charset":         func(b *Bounds) { b.Charset = "ebcdic" },
	}
	for name, mutate := range bad {
		b := DefaultBounds()
		mutate(&b)
		if b.Validate() == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if err := DefaultBounds().Validate(); err != nil {
		t.Errorf("default bounds: %v", err)
	}
}

func TestOneOfWithExclusiveRequiredMatchesOneBranch(t *testing.T) {
	schema := parseSchema(t, `{"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"integer"},"c":{"type":"integer"}},
		"oneOf":[{"required":["a"]},{"required":["b"]}]}`, "")
	source := New(DefaultBounds(), nil, 1).Compile(schema, InBody)
	r := rand.New(rand.NewPCG(1, 2))
	sawC := false
	for iter := range 2000 {
		obj := source.Generate(r, iter).(map[string]any)
		_, a := obj["a"]
		_, b := obj["b"]
		if a == b {
			t.Fatalf("value must have exactly one of a and b: %v", obj)
		}
		_, c := obj["c"]
		sawC = sawC || c
	}
	if !sawC {
		t.Error("properties no branch requires must remain available")
	}
	if schema.Properties["a"] == nil || schema.Properties["b"] == nil {
		t.Error("building variants must not alter the parent schema")
	}
}

func TestMinPropertiesWithForbiddenAdditionalStopsAtNamedOnes(t *testing.T) {
	schema := parseSchema(t, `{"type":"object","minProperties":5,"additionalProperties":false,"properties":{"a":{"type":"integer"}}}`, "")
	source := New(DefaultBounds(), nil, 1).Compile(schema, InBody)
	obj := source.Generate(rand.New(rand.NewPCG(1, 2)), 0).(map[string]any)
	if len(obj) != 1 {
		t.Fatalf("an unsatisfiable minProperties must not invent forbidden keys: %v", obj)
	}
}

func TestRequiredPropertyWithoutDefinitionIsStillSent(t *testing.T) {
	schema := parseSchema(t, `{"type":"object","required":["ghost"],"additionalProperties":{"type":"integer"}}`, "")
	obj := New(DefaultBounds(), nil, 1).Compile(schema, InBody).Generate(rand.New(rand.NewPCG(1, 2)), 0).(map[string]any)
	if _, ok := obj["ghost"].(int64); !ok {
		t.Fatalf("got %v", obj)
	}
}

func TestParameterAndFormObjectsAreNeverEmpty(t *testing.T) {
	schema := parseSchema(t, `{"type":"object","properties":{"a":{"type":"integer"},"b":{"type":"integer"}}}`, "")
	source := New(DefaultBounds(), nil, 1).Compile(schema, InQuery)
	r := rand.New(rand.NewPCG(1, 2))
	for iter := range 500 {
		if obj := source.Generate(r, iter).(map[string]any); len(obj) == 0 {
			t.Fatal("an empty object cannot be serialized as form data")
		}
	}
}

func TestEnumIsTheOnlySourceOfValues(t *testing.T) {
	dict := writeDict(t, "admin\n42\ntrue\n3.5\nx\n")
	cases := []struct {
		name, schema string
		want         []string
	}{
		{"beats dictionary", `{"type":"string","enum":["a","b"]}`, []string{"a", "b"}},
		{"integers", `{"type":"integer","enum":[7,8]}`, []string{"7", "8"}},
		{"beats bounds", `{"type":"integer","enum":[5000]}`, []string{"5000"}},
		{"beats format", `{"type":"string","format":"uuid","enum":["not-a-uuid"]}`, []string{"not-a-uuid"}},
		{"untyped", `{"enum":["a",1,true]}`, []string{"a", "1", "true"}},
		{"nullable adds no null", `{"type":"string","enum":["a"],"nullable":true}`, []string{"a"}},
		{"listed null is used", `{"type":"string","enum":["a",null],"nullable":true}`, []string{"a", "<nil>"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := New(DefaultBounds(), dict, 1).Compile(parseSchema(t, tc.schema, ""), InBody)
			r := rand.New(rand.NewPCG(1, 2))
			seen := map[string]bool{}
			for iter := range 500 {
				seen[fmt.Sprint(source.Generate(r, iter))] = true
			}
			if got := slices.Sorted(maps.Keys(seen)); !slices.Equal(got, slices.Sorted(slices.Values(tc.want))) {
				t.Errorf("generated %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUntypedSchemasGenerateTheInferredType(t *testing.T) {
	cases := map[string]struct {
		schema string
		check  func(any) bool
	}{
		"properties": {`{"properties":{"a":{"type":"integer"}},"required":["a"]}`, func(v any) bool {
			_, ok := v.(map[string]any)["a"].(int64)
			return ok
		}},
		"required only": {`{"required":["a"]}`, func(v any) bool { _, ok := v.(map[string]any)["a"]; return ok }},
		"items":         {`{"items":{"type":"integer"}}`, func(v any) bool { _, ok := v.([]any); return ok }},
		"minItems":      {`{"minItems":2}`, func(v any) bool { a, ok := v.([]any); return ok && len(a) >= 2 }},
		"uniqueItems":   {`{"uniqueItems":true}`, func(v any) bool { _, ok := v.([]any); return ok }},
		"minimum":       {`{"minimum":5,"maximum":6}`, func(v any) bool { f, ok := v.(float64); return ok && f >= 5 && f <= 6 }},
		"int32 range":   {`{"minimum":5,"maximum":9,"format":"int32"}`, func(v any) bool { i, ok := v.(int64); return ok && i >= 5 && i <= 9 }},
		"int64":         {`{"format":"int64"}`, func(v any) bool { _, ok := v.(int64); return ok }},
		"double":        {`{"format":"double"}`, func(v any) bool { _, ok := v.(float64); return ok }},
		"minLength":     {`{"minLength":3,"maxLength":3}`, func(v any) bool { s, ok := v.(string); return ok && utf8.RuneCountInString(s) == 3 }},
		"no hint":       {`{}`, func(v any) bool { _, ok := v.(string); return ok }},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			source := New(DefaultBounds(), nil, 1).Compile(parseSchema(t, tc.schema, ""), InBody)
			r := rand.New(rand.NewPCG(1, 2))
			for iter := range 500 {
				if v := source.Generate(r, iter); !tc.check(v) {
					t.Fatalf("iteration %d: unexpected value %#v", iter, v)
				}
			}
		})
	}
}

func TestNullOnlyTravelsInBodies(t *testing.T) {
	schema := parseSchema(t, `{"type":"object","required":["a"],"properties":{"a":{"type":"string","nullable":true}}}`, "")
	nullable := parseSchema(t, `{"type":"integer","nullable":true}`, "")

	for name, loc := range map[string]Location{"path": InPath, "query": InQuery, "header": InHeader} {
		t.Run(name, func(t *testing.T) {
			g := New(DefaultBounds(), nil, 1)
			scalar, nested := g.Compile(nullable, loc), g.Compile(schema, loc)
			r := rand.New(rand.NewPCG(1, 2))
			for iter := range 2000 {
				if scalar.Generate(r, iter) == nil {
					t.Fatal("a null scalar was generated")
				}
				if nested.Generate(r, iter).(map[string]any)["a"] == nil {
					t.Fatal("a nested null was generated")
				}
			}
		})
	}

	g := New(DefaultBounds(), nil, 1)
	source := g.Compile(nullable, InBody)
	r := rand.New(rand.NewPCG(1, 2))
	nulls := 0
	for iter := range 2000 {
		if source.Generate(r, iter) == nil {
			nulls++
		}
	}
	if nulls == 0 || nulls == 2000 {
		t.Errorf("%d/2000 nulls in a body, want a mix", nulls)
	}
}
