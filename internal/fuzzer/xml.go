package fuzzer

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"fuzzy/internal/openapi"
)

const (
	soap11Namespace = "http://schemas.xmlsoap.org/soap/envelope/"
	soap12Namespace = "http://www.w3.org/2003/05/soap-envelope"

	defaultXMLRoot = "root"
	soapEnvelope   = "Envelope"
	maxXMLVariants = 8 // oneOf directly inside oneOf, as far as the generator goes
)

// xmlBody says how to write the values of one XML request body.
type xmlBody struct {
	schema *openapi.Schema
	root   string // used when the schema has no xml.name
	soap   string // namespace of the SOAP envelope to wrap it in; empty for none
}

func newXMLBody(op *openapi.Operation, contentType string, media *openapi.MediaType) *xmlBody {
	body := &xmlBody{schema: media.Schema, root: media.SchemaName, soap: soapNamespace(op, contentType)}
	if body.root == "" {
		body.root = defaultXMLRoot
	}
	return body
}

// soapNamespace recognizes a SOAP operation. SOAP 1.2 has its own media type,
// while SOAP 1.1 uses text/xml like any XML API, and only its SOAPAction header
// tells them apart.
func soapNamespace(op *openapi.Operation, contentType string) string {
	media, _, _ := strings.Cut(strings.ToLower(contentType), ";")
	switch strings.TrimSpace(media) {
	case "application/soap+xml":
		return soap12Namespace
	case "text/xml":
		if slices.ContainsFunc(op.Parameters, func(p *openapi.Parameter) bool {
			return p.In == "header" && strings.EqualFold(p.Name, "SOAPAction")
		}) {
			return soap11Namespace
		}
	}
	return ""
}

// encodeXML writes v following the schema's xml keywords, which OAS defines for
// this purpose. The walk follows the value rather than the schema, so a recursive
// schema is no problem: generated values are always finite.
func encodeXML(body *xmlBody, v any) []byte {
	e := &xmlEncoder{}
	e.b.WriteString(xml.Header)

	schema := pickVariant(body.schema, v)
	root := xmlName(schema, body.root)
	wrap := body.soap != "" && localName(root) != soapEnvelope
	if wrap {
		fmt.Fprintf(&e.b, `<soap:%s xmlns:soap="%s"><soap:Body>`, soapEnvelope, body.soap)
		e.declared = append(e.declared, "soap")
	}
	if list, ok := v.([]any); ok {
		e.list(root, schema, list, true) // a document has one root, so it always wraps
	} else {
		e.element(root, schema, v)
	}
	if wrap {
		fmt.Fprintf(&e.b, `</soap:Body></soap:%s>`, soapEnvelope)
	}
	return e.b.Bytes()
}

type xmlEncoder struct {
	b        bytes.Buffer
	declared []string // namespace prefixes in scope at the current element
}

// member writes one value of an object or a list, under name unless the schema
// renames it. As OAS says, a list repeats its items unless it is wrapped, and the
// xml.name of an unwrapped list has no effect.
func (e *xmlEncoder) member(name string, s *openapi.Schema, v any) {
	s = pickVariant(s, v)
	if list, ok := v.([]any); ok {
		e.list(name, s, list, s.XML != nil && s.XML.Wrapped)
		return
	}
	e.element(xmlName(s, name), s, v)
}

func (e *xmlEncoder) list(name string, s *openapi.Schema, list []any, wrapped bool) {
	if !wrapped {
		for _, item := range list {
			e.member(name, s.Items, item)
		}
		return
	}
	outer := xmlName(s, name)
	mark := e.open(outer, s, nil)
	e.b.WriteByte('>')
	for _, item := range list {
		e.member(outer, s.Items, item) // items take the wrapper's name by default
	}
	e.close(outer, s, mark)
}

// element writes v as a single element: the members of an object become its
// attributes and children, and anything else its text.
func (e *xmlEncoder) element(name string, s *openapi.Schema, v any) {
	object, isObject := v.(map[string]any)
	var children []string
	var attributes [][2]string
	if isObject {
		for _, key := range memberOrder(s, object) {
			child := memberSchema(s, key)
			if child != nil && child.XML != nil && child.XML.Attribute {
				attributes = append(attributes, [2]string{xmlName(child, key), scalar(object[key])})
			} else {
				children = append(children, key)
			}
		}
	}

	mark := e.open(name, s, attributes)
	text := ""
	if !isObject {
		text = scalar(v)
	}
	if len(children) == 0 && text == "" {
		e.b.WriteString("/>")
		e.declared = e.declared[:mark]
		return
	}
	e.b.WriteByte('>')
	for _, key := range children {
		e.member(key, memberSchema(s, key), object[key])
	}
	_ = xml.EscapeText(&e.b, []byte(text)) // a bytes.Buffer can't fail
	e.close(name, s, mark)
}

// open writes a start tag without its closing bracket and returns the mark that
// close needs to end the scope of its namespace declaration.
func (e *xmlEncoder) open(name string, s *openapi.Schema, attributes [][2]string) int {
	mark := len(e.declared)
	declaration := ""
	if s != nil && s.XML != nil && s.XML.Namespace != "" {
		if s.XML.Prefix != "" {
			e.declared = append(e.declared, s.XML.Prefix)
			declaration = "xmlns:" + s.XML.Prefix
		} else {
			declaration = "xmlns"
		}
	}

	e.b.WriteByte('<')
	e.b.WriteString(e.qualified(name, s))
	if declaration != "" {
		e.attribute(declaration, s.XML.Namespace)
	}
	seen := make(map[string]bool, len(attributes))
	for _, attribute := range attributes {
		if !seen[attribute[0]] { // repeating an attribute isn't well-formed XML
			seen[attribute[0]] = true
			e.attribute(attribute[0], attribute[1])
		}
	}
	return mark
}

func (e *xmlEncoder) attribute(name, value string) {
	e.b.WriteByte(' ')
	e.b.WriteString(name)
	e.b.WriteString(`="`)
	_ = xml.EscapeText(&e.b, []byte(value)) // a bytes.Buffer can't fail
	e.b.WriteByte('"')
}

func (e *xmlEncoder) close(name string, s *openapi.Schema, mark int) {
	e.b.WriteString("</")
	e.b.WriteString(e.qualified(name, s))
	e.b.WriteByte('>')
	e.declared = e.declared[:mark]
}

// qualified adds the schema's prefix. A prefix nothing declares is left out,
// because a namespace-aware parser would refuse the whole document.
func (e *xmlEncoder) qualified(name string, s *openapi.Schema) string {
	if s != nil && s.XML != nil && s.XML.Prefix != "" && slices.Contains(e.declared, s.XML.Prefix) {
		return s.XML.Prefix + ":" + name
	}
	return name
}

// memberOrder lists the keys of object as the schema declares them, because XML
// consumers often enforce an order, followed by any others, sorted.
func memberOrder(s *openapi.Schema, object map[string]any) []string {
	keys := make([]string, 0, len(object))
	if s != nil {
		for _, name := range s.OrderedProperties() {
			if _, ok := object[name]; ok {
				keys = append(keys, name)
			}
		}
	}
	declared := len(keys)
	for key := range object {
		if !slices.Contains(keys[:declared], key) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys[declared:])
	return keys
}

func memberSchema(s *openapi.Schema, key string) *openapi.Schema {
	switch {
	case s == nil:
		return nil
	case s.Properties[key] != nil:
		return s.Properties[key]
	case s.AdditionalProperties != nil:
		return s.AdditionalProperties.Schema
	}
	return nil
}

// xmlName returns the element name for a schema, made valid: fallback may be any
// JSON key, and a generated or oddly named one must not break the document.
func xmlName(s *openapi.Schema, fallback string) string {
	name := fallback
	if s != nil && s.XML != nil && s.XML.Name != "" {
		name = s.XML.Name
	}
	return validXMLName(name)
}

func validXMLName(name string) string {
	valid := strings.Map(func(c rune) rune {
		if unicode.IsLetter(c) || unicode.IsDigit(c) || strings.ContainsRune("_:-.", c) {
			return c
		}
		return '_'
	}, name)
	if valid == "" {
		return "_"
	}
	if first := rune(valid[0]); first < 0x80 && !unicode.IsLetter(first) && first != '_' {
		return "_" + valid
	}
	return valid
}

func localName(name string) string {
	return name[strings.LastIndexByte(name, ':')+1:]
}

// pickVariant returns the oneOf or anyOf branch that v was generated from, since
// its xml keywords and property order are the ones to follow. The value doesn't
// say which it was, so this finds the first branch that accepts it.
func pickVariant(s *openapi.Schema, v any) *openapi.Schema {
	for range maxXMLVariants {
		if s == nil {
			return &openapi.Schema{}
		}
		variants := s.Variants()
		if len(variants) == 0 {
			return s
		}
		next := slices.IndexFunc(variants, func(variant *openapi.Schema) bool { return acceptsValue(variant, v) })
		s = variants[max(next, 0)]
	}
	return s
}

// acceptsValue is a quick check, not a validation: it only has to tell branches
// apart, and they are distinct in their shape, their required members or the
// fixed value of their discriminator.
func acceptsValue(s *openapi.Schema, v any) bool {
	if len(s.OneOf)+len(s.AnyOf) > 0 {
		return true // decided by the next round of pickVariant
	}
	kind := s.EffectiveType()
	switch t := v.(type) {
	case []any:
		return kind == "array"
	case map[string]any:
		if kind != "object" {
			return false
		}
		for _, name := range s.Required {
			if _, ok := t[name]; !ok {
				return false
			}
		}
		for key, value := range t {
			property := s.Properties[key]
			if property == nil {
				if s.AdditionalProperties == nil || s.AdditionalProperties.Forbidden {
					return false
				}
				continue
			}
			if len(property.Enum) > 0 && !slices.ContainsFunc(property.Enum, func(member any) bool {
				return fmt.Sprint(member) == fmt.Sprint(value)
			}) {
				return false
			}
		}
		return true
	}
	return kind != "array" && kind != "object"
}
