package openapi

import (
	"bytes"
	"encoding/json"
	"maps"
	"math"
	"reflect"
	"slices"
	"strings"
)

const maxMergeDepth = 8

// Schema is the subset of an OpenAPI 3.0 schema object that constrains values.
type Schema struct {
	Ref    string `json:"$ref"`
	Type   string `json:"type"`
	Format string `json:"format"`
	Enum   []any  `json:"enum"`

	Nullable bool `json:"nullable"`
	ReadOnly bool `json:"readOnly"`

	Minimum          *json.Number `json:"minimum"`
	Maximum          *json.Number `json:"maximum"`
	ExclusiveMinimum bool         `json:"exclusiveMinimum"`
	ExclusiveMaximum bool         `json:"exclusiveMaximum"`
	MultipleOf       *json.Number `json:"multipleOf"`

	MinLength *int   `json:"minLength"`
	MaxLength *int   `json:"maxLength"`
	Pattern   string `json:"pattern"`

	Items       *Schema `json:"items"`
	MinItems    *int    `json:"minItems"`
	MaxItems    *int    `json:"maxItems"`
	UniqueItems bool    `json:"uniqueItems"`

	Properties           map[string]*Schema `json:"properties"`
	Required             []string           `json:"required"`
	MinProperties        *int               `json:"minProperties"`
	MaxProperties        *int               `json:"maxProperties"`
	AdditionalProperties *Additional        `json:"additionalProperties"`

	AllOf []*Schema `json:"allOf"`
	OneOf []*Schema `json:"oneOf"`
	AnyOf []*Schema `json:"anyOf"`

	Discriminator *Discriminator `json:"discriminator"`

	// discriminatorValues maps each resolved oneOf/anyOf branch to the value
	// its discriminator property must carry.
	discriminatorValues map[*Schema]string
}

// Additional is the additionalProperties keyword: a schema or a boolean.
type Additional struct {
	Schema    *Schema
	Forbidden bool
}

func (a *Additional) UnmarshalJSON(data []byte) error {
	switch string(bytes.TrimSpace(data)) {
	case "true":
		return nil
	case "false":
		a.Forbidden = true
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	a.Schema = &Schema{}
	return dec.Decode(a.Schema)
}

type Discriminator struct {
	PropertyName string            `json:"propertyName"`
	Mapping      map[string]string `json:"mapping"`
}

// EffectiveType returns the declared type or infers it from the keywords in
// use. Untyped schemas accept anything, so they are treated as strings.
func (s *Schema) EffectiveType() string {
	switch {
	case s.Type != "":
		return s.Type
	case len(s.Properties) > 0 || s.AdditionalProperties != nil || s.MinProperties != nil || s.MaxProperties != nil:
		return "object"
	case s.Items != nil:
		return "array"
	case s.Minimum != nil || s.Maximum != nil || s.MultipleOf != nil:
		return "number"
	}
	return "string"
}

// Variants returns one fully merged schema per oneOf/anyOf branch, or nil when
// the schema has no branches. Each variant carries the sibling keywords of the
// parent and, if a discriminator applies, a fixed value for its property.
func (s *Schema) Variants() []*Schema {
	branches := s.OneOf
	if len(branches) == 0 {
		branches = s.AnyOf
	}
	if len(branches) == 0 {
		return nil
	}

	base := *s
	base.OneOf, base.AnyOf, base.Discriminator, base.discriminatorValues = nil, nil, nil, nil

	variants := make([]*Schema, 0, len(branches))
	for i, branch := range branches {
		v := base
		v.Properties = maps.Clone(base.Properties)
		v.Required = slices.Clone(base.Required)
		merge(&v, branch)

		// oneOf demands exactly one match, and branches often differ only in what
		// they require: sending what a sibling requires would match both.
		if len(s.OneOf) > 0 {
			for j, sibling := range branches {
				for _, name := range sibling.Required {
					isTag := s.Discriminator != nil && s.Discriminator.PropertyName == name
					if j != i && !isTag && !slices.Contains(v.Required, name) {
						delete(v.Properties, name)
					}
				}
			}
		}

		if d := s.Discriminator; d != nil && d.PropertyName != "" {
			if tag, ok := s.discriminatorValues[branch]; ok {
				if v.Properties == nil {
					v.Properties = map[string]*Schema{}
				}
				v.Properties[d.PropertyName] = &Schema{Type: "string", Enum: []any{tag}}
			}
			// The discriminator must always be present, inline branches included.
			if v.Properties[d.PropertyName] != nil && !slices.Contains(v.Required, d.PropertyName) {
				v.Required = append(v.Required, d.PropertyName)
			}
		}
		variants = append(variants, &v)
	}

	satisfiable := slices.DeleteFunc(slices.Clone(variants), (*Schema).unsatisfiable)
	if len(satisfiable) > 0 {
		return satisfiable
	}
	return variants
}

// unsatisfiable reports whether s requires a property it also forbids.
func (s *Schema) unsatisfiable() bool {
	if s.AdditionalProperties == nil || !s.AdditionalProperties.Forbidden {
		return false
	}
	return slices.ContainsFunc(s.Required, func(name string) bool { return s.Properties[name] == nil })
}

// merge folds src into dst keeping the tighter constraint on each keyword, so
// that a value generated from dst also satisfies src. src is left untouched.
func merge(dst, src *Schema) { mergeAt(dst, src, 0) }

func mergeAt(dst, src *Schema, depth int) {
	if dst.Type == "" {
		dst.Type = src.Type
	}
	if dst.Format == "" {
		dst.Format = src.Format
	}
	if dst.Pattern == "" {
		dst.Pattern = src.Pattern
	}
	if dst.Items == nil {
		dst.Items = src.Items
	}
	if len(dst.Enum) == 0 {
		dst.Enum = src.Enum
	} else if len(src.Enum) > 0 {
		dst.Enum = slices.DeleteFunc(slices.Clone(dst.Enum), func(v any) bool {
			return !slices.ContainsFunc(src.Enum, func(o any) bool { return reflect.DeepEqual(o, v) })
		})
	}

	dst.Minimum, dst.ExclusiveMinimum = tighterNumber(dst.Minimum, dst.ExclusiveMinimum, src.Minimum, src.ExclusiveMinimum, 1)
	dst.Maximum, dst.ExclusiveMaximum = tighterNumber(dst.Maximum, dst.ExclusiveMaximum, src.Maximum, src.ExclusiveMaximum, -1)
	if dst.MultipleOf == nil {
		dst.MultipleOf = src.MultipleOf
	}

	dst.MinLength = tighterInt(dst.MinLength, src.MinLength, 1)
	dst.MaxLength = tighterInt(dst.MaxLength, src.MaxLength, -1)
	dst.MinItems = tighterInt(dst.MinItems, src.MinItems, 1)
	dst.MaxItems = tighterInt(dst.MaxItems, src.MaxItems, -1)
	dst.UniqueItems = dst.UniqueItems || src.UniqueItems

	if len(src.Properties) > 0 {
		if dst.Properties == nil {
			dst.Properties = make(map[string]*Schema, len(src.Properties))
		}
		for name, prop := range src.Properties {
			// Both constraints apply to a property defined twice. Recursive schemas
			// could overlap forever, so past the limit the later definition wins.
			if own := dst.Properties[name]; own != nil && own != prop && depth < maxMergeDepth {
				combined := *own
				combined.Properties = maps.Clone(own.Properties)
				combined.Required = slices.Clone(own.Required)
				mergeAt(&combined, prop, depth+1)
				prop = &combined
			}
			dst.Properties[name] = prop
		}
	}
	for _, name := range src.Required {
		if !slices.Contains(dst.Required, name) {
			dst.Required = append(dst.Required, name)
		}
	}
	dst.MinProperties = tighterInt(dst.MinProperties, src.MinProperties, 1)
	dst.MaxProperties = tighterInt(dst.MaxProperties, src.MaxProperties, -1)
	if dst.AdditionalProperties == nil {
		dst.AdditionalProperties = src.AdditionalProperties
	}

	if len(dst.OneOf) == 0 && len(src.OneOf) > 0 {
		dst.OneOf, dst.Discriminator, dst.discriminatorValues = src.OneOf, src.Discriminator, src.discriminatorValues
	}
	if len(dst.AnyOf) == 0 && len(src.AnyOf) > 0 {
		dst.AnyOf, dst.Discriminator, dst.discriminatorValues = src.AnyOf, src.Discriminator, src.discriminatorValues
	}
}

// tighterNumber picks the stricter numeric bound; sign is +1 for minimums
// (larger wins) and -1 for maximums (smaller wins).
func tighterNumber(a *json.Number, aExcl bool, b *json.Number, bExcl bool, sign float64) (*json.Number, bool) {
	if a == nil {
		return b, bExcl
	}
	if b == nil {
		return a, aExcl
	}
	af, bf := Float(a), Float(b)
	switch {
	case af == bf:
		return a, aExcl || bExcl
	case (bf-af)*sign > 0:
		return b, bExcl
	}
	return a, aExcl
}

func tighterInt(a, b *int, sign int) *int {
	if a == nil {
		return b
	}
	if b == nil || (*a-*b)*sign >= 0 {
		return a
	}
	return b
}

// Float converts a bound to float64.
func Float(n *json.Number) float64 {
	f, _ := n.Float64()
	return f
}

// Int converts a bound to int64, rounding towards the inside of the range:
// up for lower bounds (roundUp) and down for upper bounds.
func Int(n *json.Number, roundUp bool) int64 {
	if i, err := n.Int64(); err == nil {
		return i
	}
	f := Float(n)
	if roundUp {
		f = math.Ceil(f)
	} else {
		f = math.Floor(f)
	}
	switch {
	case f >= math.MaxInt64:
		return math.MaxInt64
	case f <= math.MinInt64:
		return math.MinInt64
	}
	return int64(f)
}

// lastSegment returns the component name a "#/components/.../Name" ref points to.
func lastSegment(ref string) string {
	return ref[strings.LastIndexByte(ref, '/')+1:]
}
