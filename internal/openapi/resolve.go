package openapi

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

const (
	schemaRefPrefix    = "#/components/schemas/"
	parameterRefPrefix = "#/components/parameters/"
	bodyRefPrefix      = "#/components/requestBodies/"
)

// resolver rewrites the decoded document in place. Schemas are walked as a
// graph because recursive components are legal.
type resolver struct {
	doc     *Document
	walked  map[*Schema]bool
	merging map[*Schema]bool
	merged  map[*Schema]bool
}

func newResolver(doc *Document) *resolver {
	return &resolver{
		doc:     doc,
		walked:  map[*Schema]bool{},
		merging: map[*Schema]bool{},
		merged:  map[*Schema]bool{},
	}
}

func (r *resolver) resolveAll() error {
	for _, name := range sortedKeys(r.doc.Components.Schemas) {
		s, err := r.schema(r.doc.Components.Schemas[name])
		if err != nil {
			return fmt.Errorf("components.schemas.%s: %w", name, err)
		}
		r.doc.Components.Schemas[name] = s
	}
	for _, path := range sortedKeys(r.doc.Paths) {
		item := r.doc.Paths[path]
		if item == nil {
			continue
		}
		if err := r.parameters(item.Parameters); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		for _, op := range []*Operation{item.Get, item.Post, item.Put, item.Patch, item.Delete, item.Head, item.Options, item.Trace} {
			if op == nil {
				continue
			}
			if err := r.parameters(op.Parameters); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			if op.RequestBody != nil {
				body, err := r.body(op.RequestBody)
				if err != nil {
					return fmt.Errorf("%s: %w", path, err)
				}
				op.RequestBody = body
			}
		}
	}
	for s := range r.walked {
		r.flatten(s)
	}
	return nil
}

func (r *resolver) parameters(params []*Parameter) error {
	for i, p := range params {
		resolved, err := r.parameter(p)
		if err != nil {
			return err
		}
		params[i] = resolved
	}
	return nil
}

func (r *resolver) parameter(p *Parameter) (*Parameter, error) {
	for hops := 0; p.Ref != ""; hops++ {
		name, ok := strings.CutPrefix(p.Ref, parameterRefPrefix)
		target := r.doc.Components.Parameters[name]
		if !ok || target == nil || hops > len(r.doc.Components.Parameters) {
			return nil, fmt.Errorf("unresolvable parameter reference %q", p.Ref)
		}
		p = target
	}

	if p.Schema == nil && len(p.Content) > 0 {
		contentType := preferredContent(p.Content)
		p.ContentType, p.Schema = contentType, p.Content[contentType].Schema
	}
	if p.Schema == nil {
		p.Schema = &Schema{Type: "string"}
	}
	if p.In == "path" {
		p.Required = true
	}

	s, err := r.schema(p.Schema)
	if err != nil {
		return nil, fmt.Errorf("parameter %q: %w", p.Name, err)
	}
	p.Schema = s
	return p, nil
}

// preferredContent prefers JSON, the only media type parameters really use.
func preferredContent(content map[string]*MediaType) string {
	keys := sortedKeys(content)
	for _, key := range keys {
		if strings.Contains(key, "json") {
			return key
		}
	}
	return keys[0]
}

func (r *resolver) body(b *RequestBody) (*RequestBody, error) {
	for hops := 0; b.Ref != ""; hops++ {
		name, ok := strings.CutPrefix(b.Ref, bodyRefPrefix)
		target := r.doc.Components.RequestBodies[name]
		if !ok || target == nil || hops > len(r.doc.Components.RequestBodies) {
			return nil, fmt.Errorf("unresolvable requestBody reference %q", b.Ref)
		}
		b = target
	}
	for contentType, media := range b.Content {
		if media == nil {
			delete(b.Content, contentType)
			continue
		}
		if media.Schema == nil {
			media.Schema = &Schema{}
		}
		if media.Schema.Ref != "" {
			media.SchemaName = lastSegment(media.Schema.Ref)
		}
		s, err := r.schema(media.Schema)
		if err != nil {
			return nil, fmt.Errorf("requestBody %s: %w", contentType, err)
		}
		media.Schema = s
	}
	return b, nil
}

// schema follows s to its final target and rewrites everything reachable from it.
func (r *resolver) schema(s *Schema) (*Schema, error) {
	s, err := r.follow(s)
	if err != nil {
		return nil, err
	}
	if r.walked[s] {
		return s, nil
	}
	r.walked[s] = true

	if s.Items, err = r.optional(s.Items); err != nil {
		return nil, err
	}
	for name, prop := range s.Properties {
		resolved, err := r.schema(prop)
		if err != nil {
			return nil, fmt.Errorf("property %q: %w", name, err)
		}
		s.Properties[name] = resolved
	}
	if extra := s.AdditionalProperties; extra != nil {
		if extra.Schema, err = r.optional(extra.Schema); err != nil {
			return nil, err
		}
	}
	if err := r.branches(s, s.OneOf); err != nil {
		return nil, err
	}
	if err := r.branches(s, s.AnyOf); err != nil {
		return nil, err
	}
	if err := r.list(s.AllOf); err != nil {
		return nil, err
	}
	return s, nil
}

func (r *resolver) follow(s *Schema) (*Schema, error) {
	for hops := 0; s.Ref != ""; hops++ {
		name, ok := strings.CutPrefix(s.Ref, schemaRefPrefix)
		if !ok {
			return nil, fmt.Errorf("unsupported schema reference %q: only %s* is allowed", s.Ref, schemaRefPrefix)
		}
		target := r.doc.Components.Schemas[strings.NewReplacer("~1", "/", "~0", "~").Replace(name)]
		if target == nil {
			return nil, fmt.Errorf("unresolvable schema reference %q", s.Ref)
		}
		if hops > len(r.doc.Components.Schemas) {
			return nil, fmt.Errorf("schema reference cycle at %q", s.Ref)
		}
		s = target
	}
	return s, nil
}

func (r *resolver) optional(s *Schema) (*Schema, error) {
	if s == nil {
		return nil, nil
	}
	return r.schema(s)
}

func (r *resolver) list(schemas []*Schema) error {
	for i, s := range schemas {
		resolved, err := r.schema(s)
		if err != nil {
			return err
		}
		schemas[i] = resolved
	}
	return nil
}

// branches resolves oneOf/anyOf members. Refs are only available before the
// rewrite, so this is where discriminator values are bound to their branch.
func (r *resolver) branches(parent *Schema, branches []*Schema) error {
	refs := make([]string, len(branches))
	for i, b := range branches {
		refs[i] = b.Ref
	}
	if err := r.list(branches); err != nil {
		return err
	}
	if parent.Discriminator == nil || parent.Discriminator.PropertyName == "" {
		return nil
	}
	if parent.discriminatorValues == nil {
		parent.discriminatorValues = map[*Schema]string{}
	}
	for i, ref := range refs {
		if ref != "" {
			parent.discriminatorValues[branches[i]] = discriminatorValue(parent.Discriminator, ref)
		}
	}
	return nil
}

func discriminatorValue(d *Discriminator, ref string) string {
	for _, key := range sortedKeys(d.Mapping) {
		if target := d.Mapping[key]; target == ref || target == lastSegment(ref) {
			return key
		}
	}
	return lastSegment(ref)
}

// flatten merges allOf members into s. The in-progress guard stops a schema
// that (indirectly) includes itself from recursing forever.
func (r *resolver) flatten(s *Schema) {
	if r.merged[s] || r.merging[s] {
		return
	}
	r.merging[s] = true
	// Properties first: merge may copy them, and a copy would miss a later flatten.
	for _, prop := range s.Properties {
		r.flatten(prop)
	}
	for _, member := range s.AllOf {
		r.flatten(member)
		merge(s, member)
	}
	s.AllOf = nil
	delete(r.merging, s)
	r.merged[s] = true
}

func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
