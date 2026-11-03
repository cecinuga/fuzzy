package openapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

const (
	definitionRefPrefix       = "#/definitions/"
	sharedParameterRefPrefix  = "#/parameters/"
	formURLEncoded            = "application/x-www-form-urlencoded"
	multipartFormData         = "multipart/form-data"
	defaultSwaggerContentType = "application/json"
)

// swaggerDocument is the subset of a Swagger 2.0 document that fuzzing needs. The
// host and schemes are left out on purpose: the target is always the remote given
// on the command line.
type swaggerDocument struct {
	Info        Info                         `json:"info"`
	BasePath    string                       `json:"basePath"`
	Consumes    []string                     `json:"consumes"`
	Paths       map[string]*swaggerPathItem  `json:"paths"`
	Definitions map[string]*Schema           `json:"definitions"`
	Parameters  map[string]*swaggerParameter `json:"parameters"`
}

type swaggerPathItem struct {
	Parameters []*swaggerParameter `json:"parameters"`

	Get     *swaggerOperation `json:"get"`
	Put     *swaggerOperation `json:"put"`
	Post    *swaggerOperation `json:"post"`
	Delete  *swaggerOperation `json:"delete"`
	Options *swaggerOperation `json:"options"`
	Head    *swaggerOperation `json:"head"`
	Patch   *swaggerOperation `json:"patch"`
}

type swaggerOperation struct {
	OperationID string              `json:"operationId"`
	Summary     string              `json:"summary"`
	Tags        []string            `json:"tags"`
	Parameters  []*swaggerParameter `json:"parameters"`
	Consumes    []string            `json:"consumes"` // an empty list clears the global one
}

// swaggerParameter is any parameter of a Swagger 2.0 operation. Only a body
// parameter has a schema: the others carry the schema keywords directly.
type swaggerParameter struct {
	Ref              string  `json:"$ref"`
	Name             string  `json:"name"`
	In               string  `json:"in"`
	Required         bool    `json:"required"`
	Schema           *Schema `json:"schema"`
	CollectionFormat string  `json:"collectionFormat"`

	keywords *Schema
	file     bool
}

// UnmarshalJSON also reads the parameter as a schema. Its "required" is a boolean
// there, where a schema has a list of names, so that key stays out.
func (p *swaggerParameter) UnmarshalJSON(data []byte) error {
	type plain swaggerParameter
	if err := json.Unmarshal(data, (*plain)(p)); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	delete(fields, "required")
	delete(fields, "schema")
	delete(fields, "$ref")
	keywords, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	p.keywords = &Schema{}
	if err := json.Unmarshal(keywords, p.keywords); err != nil {
		return err
	}
	if p.keywords.Type == "file" {
		p.file = true
		p.keywords.Type, p.keywords.Format = "string", "binary"
	}
	return nil
}

// decodeSwagger converts a Swagger 2.0 document into the model OpenAPI 3.0 decodes
// to. References are still unresolved at this point, so from here on both versions
// share the same code.
func decodeSwagger(data []byte) (*Document, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber() // int64 limits and enum members must not lose precision
	var spec swaggerDocument
	if err := dec.Decode(&spec); err != nil {
		return nil, fmt.Errorf("decode spec: %w", err)
	}
	c := &swaggerConverter{spec: &spec, rewritten: map[*Schema]bool{}}
	return c.document()
}

type swaggerConverter struct {
	spec      *swaggerDocument
	rewritten map[*Schema]bool
}

func (c *swaggerConverter) document() (*Document, error) {
	doc := &Document{
		Format:     "swagger 2.0",
		Info:       c.spec.Info,
		Paths:      make(map[string]*PathItem, len(c.spec.Paths)),
		Components: Components{Schemas: c.spec.Definitions},
	}
	if c.spec.BasePath != "" {
		doc.Servers = []Server{{URL: c.spec.BasePath}}
	}
	for _, name := range sortedKeys(c.spec.Definitions) {
		if err := c.schema(c.spec.Definitions[name]); err != nil {
			return nil, fmt.Errorf("definitions.%s: %w", name, err)
		}
	}
	applyInheritance(doc.Components.Schemas)

	for _, path := range sortedKeys(c.spec.Paths) {
		item := c.spec.Paths[path]
		if item == nil {
			continue
		}
		out := &PathItem{}
		for _, entry := range []struct {
			method string
			from   *swaggerOperation
			to     **Operation
		}{
			{"GET", item.Get, &out.Get}, {"POST", item.Post, &out.Post}, {"PUT", item.Put, &out.Put}, {"PATCH", item.Patch, &out.Patch},
			{"DELETE", item.Delete, &out.Delete}, {"HEAD", item.Head, &out.Head}, {"OPTIONS", item.Options, &out.Options},
		} {
			if entry.from == nil {
				continue
			}
			op, err := c.operation(item, entry.from)
			if err != nil {
				return nil, fmt.Errorf("%s %s: %w", entry.method, path, err)
			}
			*entry.to = op
		}
		doc.Paths[path] = out
	}
	return doc, nil
}

// operation merges the path-level parameters in, since a body or form parameter
// declared there has to become part of each operation's request body.
func (c *swaggerConverter) operation(item *swaggerPathItem, from *swaggerOperation) (*Operation, error) {
	params, err := c.parameters(item.Parameters, from.Parameters)
	if err != nil {
		return nil, err
	}

	op := &Operation{OperationID: from.OperationID, Summary: from.Summary, Tags: from.Tags}
	var body, form []*swaggerParameter
	for _, p := range params {
		switch p.In {
		case "body":
			body = append(body, p)
		case "formData":
			form = append(form, p)
		case "path", "query", "header":
			param, err := c.parameter(p)
			if err != nil {
				return nil, err
			}
			op.Parameters = append(op.Parameters, param)
		default:
			return nil, fmt.Errorf("parameter %q: invalid location %q", p.Name, p.In)
		}
	}

	consumes := c.spec.Consumes
	if from.Consumes != nil {
		consumes = from.Consumes
	}
	switch {
	case len(body) > 0 && len(form) > 0:
		return nil, errors.New("body and formData parameters cannot be combined")
	case len(body) > 0:
		op.RequestBody, err = c.bodyParameter(body[0], consumes)
	case len(form) > 0:
		op.RequestBody, err = c.formParameters(form, consumes)
	}
	return op, err
}

// parameters applies the same override rule as mergeParameters. There can only be
// one body, so a later body replaces an earlier one whatever its name.
func (c *swaggerConverter) parameters(shared, own []*swaggerParameter) ([]*swaggerParameter, error) {
	merged := make([]*swaggerParameter, 0, len(shared)+len(own))
	for _, p := range slices.Concat(shared, own) {
		p, err := c.follow(p)
		if err != nil {
			return nil, err
		}
		if i := slices.IndexFunc(merged, func(m *swaggerParameter) bool {
			return m.In == p.In && (m.Name == p.Name || p.In == "body")
		}); i >= 0 {
			merged[i] = p
		} else {
			merged = append(merged, p)
		}
	}
	return merged, nil
}

func (c *swaggerConverter) follow(p *swaggerParameter) (*swaggerParameter, error) {
	for hops := 0; p != nil && p.Ref != ""; hops++ {
		name, ok := strings.CutPrefix(p.Ref, sharedParameterRefPrefix)
		target := c.spec.Parameters[name]
		if !ok || target == nil || hops > len(c.spec.Parameters) {
			return nil, fmt.Errorf("unresolvable parameter reference %q", p.Ref)
		}
		p = target
	}
	if p == nil {
		return nil, errors.New("null parameter")
	}
	return p, nil
}

func (c *swaggerConverter) parameter(p *swaggerParameter) (*Parameter, error) {
	if err := c.schema(p.keywords); err != nil {
		return nil, fmt.Errorf("parameter %q: %w", p.Name, err)
	}
	out := &Parameter{Name: p.Name, In: p.In, Required: p.Required, Schema: p.keywords}
	if p.keywords.Type == "array" {
		explode, delimiter, err := collection(p.CollectionFormat)
		if err != nil {
			return nil, fmt.Errorf("parameter %q: %w", p.Name, err)
		}
		out.Explode, out.Delimiter = &explode, delimiter
	}
	return out, nil
}

// collection says how a collectionFormat writes an array. Unlike OpenAPI 3.0, the
// default is one comma-separated value rather than a repeated parameter. tsv has no
// matching style, and neither have ssv and pipes outside the query, so the separator
// is kept as it is.
func collection(format string) (explode bool, delimiter string, err error) {
	switch format {
	case "", "csv":
		return false, "", nil
	case "multi":
		return true, "", nil
	case "ssv":
		return false, " ", nil
	case "tsv":
		return false, "\t", nil
	case "pipes":
		return false, "|", nil
	}
	return false, "", fmt.Errorf("unsupported collectionFormat %q", format)
}

// bodyParameter offers the body in every media type the operation consumes. With
// none declared, Swagger says nothing about the format and JSON is the safe choice.
func (c *swaggerConverter) bodyParameter(p *swaggerParameter, consumes []string) (*RequestBody, error) {
	if p.Schema != nil {
		if err := c.schema(p.Schema); err != nil {
			return nil, fmt.Errorf("parameter %q: %w", p.Name, err)
		}
	}
	if len(consumes) == 0 {
		consumes = []string{defaultSwaggerContentType}
	}
	body := &RequestBody{Required: p.Required, Content: make(map[string]*MediaType, len(consumes))}
	for _, contentType := range consumes {
		body.Content[contentType] = &MediaType{Schema: p.Schema}
	}
	return body, nil
}

// formParameters gathers the formData parameters into the object schema OpenAPI 3.0
// would use for the body. A file can only be sent in a multipart body, and media
// types that aren't forms can't carry these parameters at all, whatever consumes says.
func (c *swaggerConverter) formParameters(params []*swaggerParameter, consumes []string) (*RequestBody, error) {
	schema := &Schema{Type: "object", Properties: make(map[string]*Schema, len(params))}
	encoding := map[string]*Encoding{}
	body := &RequestBody{Content: map[string]*MediaType{}}
	hasFile := false
	for _, p := range params {
		if p.Name == "" {
			return nil, errors.New("parameter in formData without a name")
		}
		if err := c.schema(p.keywords); err != nil {
			return nil, fmt.Errorf("parameter %q: %w", p.Name, err)
		}
		schema.Properties[p.Name] = p.keywords
		schema.PropertyOrder = append(schema.PropertyOrder, p.Name)
		if p.Required {
			schema.Required = append(schema.Required, p.Name)
			body.Required = true
		}
		hasFile = hasFile || p.file
		if p.keywords.Type == "array" {
			explode, delimiter, err := collection(p.CollectionFormat)
			if err != nil {
				return nil, fmt.Errorf("parameter %q: %w", p.Name, err)
			}
			encoding[p.Name] = &Encoding{Explode: &explode, Delimiter: delimiter}
		}
	}

	var contentTypes []string
	for _, contentType := range consumes {
		media, _, _ := strings.Cut(strings.ToLower(contentType), ";")
		media = strings.TrimSpace(media)
		if media == multipartFormData || (media == formURLEncoded && !hasFile) {
			contentTypes = append(contentTypes, contentType)
		}
	}
	if len(contentTypes) == 0 {
		contentTypes = []string{formURLEncoded}
		if hasFile {
			contentTypes = []string{multipartFormData}
		}
	}
	for _, contentType := range contentTypes {
		body.Content[contentType] = &MediaType{Schema: schema, Encoding: encoding}
	}
	return body, nil
}

// schema points the references below s at the components the resolver looks in.
// Errors name the reference as the spec spells it. Nothing shares a schema before
// resolving except a parameter used by several operations, hence the guard.
func (c *swaggerConverter) schema(s *Schema) error {
	if s == nil {
		return errors.New("null schema")
	}
	if c.rewritten[s] {
		return nil
	}
	c.rewritten[s] = true

	if s.Ref != "" {
		name, ok := strings.CutPrefix(s.Ref, definitionRefPrefix)
		if !ok {
			return fmt.Errorf("unsupported schema reference %q: only %s* is allowed", s.Ref, definitionRefPrefix)
		}
		if c.spec.Definitions[pointerToken(name)] == nil {
			return fmt.Errorf("unresolvable schema reference %q", s.Ref)
		}
		s.Ref = schemaRefPrefix + name
	}
	if s.Items != nil {
		if err := c.schema(s.Items); err != nil {
			return err
		}
	}
	for _, name := range sortedKeys(s.Properties) {
		if err := c.schema(s.Properties[name]); err != nil {
			return fmt.Errorf("property %q: %w", name, err)
		}
	}
	if extra := s.AdditionalProperties; extra != nil && extra.Schema != nil {
		if err := c.schema(extra.Schema); err != nil {
			return err
		}
	}
	for _, list := range [][]*Schema{s.AllOf, s.OneOf, s.AnyOf} {
		for _, member := range list {
			if err := c.schema(member); err != nil {
				return err
			}
		}
	}
	return nil
}

// applyInheritance gives a Swagger 2.0 discriminator its meaning. There it is just
// a property name on a base schema, its subtypes extend the base through allOf, and
// the property must hold the name of the definition the value is an instance of.
//
// OpenAPI 3.0 models that as a oneOf, so a schema with subtypes moves to another
// name for them to extend, and its own name becomes a oneOf of the concrete subtypes:
// what refers to Pet gets a Cat or a Dog, and a Cat is still a Pet. Only the leaves
// are branches because a base is often abstract, and a server would reject its name.
// A leaf also pins the property, since it can be referred to directly.
func applyInheritance(schemas map[string]*Schema) {
	defs := maps.Clone(schemas)
	subtypes := map[string][]string{}
	for _, name := range sortedKeys(defs) {
		for _, member := range defs[name].AllOf {
			parent, ok := strings.CutPrefix(member.Ref, schemaRefPrefix)
			if parent = pointerToken(parent); ok && defs[parent] != nil {
				subtypes[parent] = append(subtypes[parent], name)
			}
		}
	}

	moved := map[string]bool{}
	for _, root := range sortedKeys(defs) {
		d := defs[root].Discriminator
		if d == nil || d.PropertyName == "" {
			continue
		}
		family := descendants(root, subtypes)
		leaves := slices.DeleteFunc(slices.Clone(family), func(name string) bool { return len(subtypes[name]) > 0 })
		if len(leaves) == 0 {
			continue
		}
		tag := d.PropertyName

		// A property can't hold a name its enum doesn't list. When that happens the
		// spec's enum wins, and the subtypes are a plain oneOf that keeps it.
		var allowed []any
		if property := defs[root].Properties[tag]; property != nil {
			allowed = property.Enum
		}
		tagged := allowed == nil || !slices.ContainsFunc(leaves, func(name string) bool { return !slices.Contains(allowed, any(name)) })
		if tagged {
			for _, leaf := range leaves {
				pinProperty(defs[leaf], tag, leaf)
			}
		}

		for _, name := range family {
			if len(subtypes[name]) == 0 || moved[name] {
				continue
			}
			moved[name] = true
			extended := name + "$base"
			for schemas[extended] != nil {
				extended += "_"
			}
			schemas[extended] = defs[name]
			for _, sub := range subtypes[name] {
				for _, member := range defs[sub].AllOf {
					if pointerToken(strings.TrimPrefix(member.Ref, schemaRefPrefix)) == name {
						member.Ref = schemaRefPrefix + extended
					}
				}
			}

			union := &Schema{XML: defs[name].XML}
			if tagged {
				union.Discriminator = &Discriminator{PropertyName: tag}
			}
			for _, leaf := range descendants(name, subtypes) {
				if len(subtypes[leaf]) == 0 {
					union.OneOf = append(union.OneOf, &Schema{Ref: schemaRefPrefix + leaf})
				}
			}
			schemas[name] = union
		}
	}
}

// descendants returns name and every schema extending it, however indirectly. Cycles
// are illegal but mustn't hang a run, and a family with no leaf is left alone.
func descendants(name string, subtypes map[string][]string) []string {
	var family []string
	seen := map[string]bool{}
	var visit func(string)
	visit = func(n string) {
		if seen[n] {
			return
		}
		seen[n] = true
		family = append(family, n)
		for _, sub := range subtypes[n] {
			visit(sub)
		}
	}
	visit(name)
	return family
}

// pinProperty makes s require a property holding exactly value. It goes through
// allOf so that it merges with the definition the property inherits.
func pinProperty(s *Schema, name, value string) {
	s.AllOf = append(s.AllOf, &Schema{
		Properties:    map[string]*Schema{name: {Type: "string", Enum: []any{value}}},
		PropertyOrder: []string{name},
		Required:      []string{name},
	})
}
