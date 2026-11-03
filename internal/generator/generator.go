package generator

import (
	"encoding/base64"
	"fmt"
	"maps"
	"math"
	"math/rand/v2"
	"net/mail"
	"net/netip"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"fuzzy/internal/openapi"
)

// Location says where a value travels, since that narrows what is valid in it.
type Location int

const (
	InBody Location = iota // JSON and multipart bodies, the only places null can travel
	InPath
	InQuery
	InHeader // also used for cookies: both are restricted to ASCII
)

const (
	softDepth  = 4  // past this depth, optional content is dropped
	hardDepth  = 12 // past this depth, even required content is cut (unsatisfiable cycles)
	maxNesting = 8  // oneOf directly inside oneOf
	extraItems = 3  // arrays get at most minItems+extraItems elements
)

// Generator compiles schemas into value sources. Compiling happens once, up
// front, and is not safe for concurrent use; the resulting Sources are.
type Generator struct {
	bounds   Bounds
	dict     *Dictionary
	offsets  *rand.Rand
	memo     map[memoKey]*lazyNode
	warnings []string
}

type memoKey struct {
	schema *openapi.Schema
	loc    Location
}

// New returns a Generator. dict may be nil. The seed fixes where each value
// list starts.
func New(bounds Bounds, dict *Dictionary, seed uint64) *Generator {
	return &Generator{
		bounds:  bounds,
		dict:    dict,
		offsets: rand.New(rand.NewPCG(seed, 0x667a7a79)),
		memo:    map[memoKey]*lazyNode{},
	}
}

// Warnings lists the schema constraints that could not be honored.
func (g *Generator) Warnings() []string { return g.warnings }

// Source produces values for one schema.
type Source struct{ root node }

// Compile prepares a Source for a value of schema s traveling in loc.
func (g *Generator) Compile(s *openapi.Schema, loc Location) Source {
	return Source{root: g.compile(s, loc, 0)}
}

// Generate returns a value. It only reads the compiled schema, so it may be
// called concurrently as long as each caller has its own r. iter advances the
// walk through dictionary and enum values: consecutive iterations visit
// consecutive values, so every value gets its turn.
func (s Source) Generate(r *rand.Rand, iter int) any {
	return s.root.generate(ctx{r: r, iter: iter})
}

type ctx struct {
	r     *rand.Rand
	iter  int
	depth int
}

type node interface{ generate(c ctx) any }

// lazyNode ties the knot for recursive schemas: it is memoized before its
// target exists, so a cycle compiles to a cycle of nodes.
type lazyNode struct{ node }

func (g *Generator) compile(s *openapi.Schema, loc Location, nesting int) node {
	if s == nil {
		s = &openapi.Schema{}
	}
	key := memoKey{s, loc}
	if n, ok := g.memo[key]; ok {
		return n
	}
	n := &lazyNode{}
	g.memo[key] = n
	n.node = g.build(s, loc, nesting)
	return n
}

func (g *Generator) build(s *openapi.Schema, loc Location, nesting int) node {
	// The list is complete: with an enum, nullable allows null only if it is listed.
	if len(s.Enum) > 0 {
		return g.pick(enumValues(s, loc))
	}

	var n node
	switch {
	case len(s.OneOf)+len(s.AnyOf) > 0 && nesting < maxNesting:
		n = g.variants(s, loc, nesting)
	default:
		n = g.byType(s, loc, nesting)
	}
	if s.Nullable && loc == InBody {
		return nullableNode{n}
	}
	return n
}

func enumValues(s *openapi.Schema, loc Location) []any {
	vals := make([]any, 0, len(s.Enum))
	for _, v := range s.Enum {
		if v != nil || loc == InBody { // null has no representation outside a body
			vals = append(vals, v)
		}
	}
	return vals
}

func (g *Generator) variants(s *openapi.Schema, loc Location, nesting int) node {
	variants := s.Variants()
	branches := make(variantNode, len(variants))
	for i, v := range variants {
		branches[i] = g.compile(v, loc, nesting+1)
	}
	return branches
}

func (g *Generator) byType(s *openapi.Schema, loc Location, nesting int) node {
	switch s.EffectiveType() {
	case "integer":
		return g.integer(s)
	case "number":
		return g.number(s)
	case "boolean":
		if g.dict != nil && len(g.dict.Bools) > 0 {
			return g.pick(toAny(g.dict.Bools))
		}
		return boolNode{}
	case "array":
		return g.array(s, loc, nesting)
	case "object":
		return g.object(s, loc, nesting)
	default:
		return g.str(s, loc)
	}
}

func (g *Generator) pick(vals []any) node {
	if len(vals) == 0 {
		return nullNode{}
	}
	return pickNode{vals: vals, offset: g.offsets.IntN(len(vals))}
}

func toAny[T any](vals []T) []any {
	out := make([]any, len(vals))
	for i, v := range vals {
		out[i] = v
	}
	return out
}

func filter[T any](vals []T, keep func(T) bool) []any {
	var out []any
	for _, v := range vals {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

// --- integers ---------------------------------------------------------------

func (g *Generator) integer(s *openapi.Schema) node {
	var lo, hi limit[int64]
	if s.Minimum != nil {
		lo = limit[int64]{openapi.Int(s.Minimum, true), true}
		if s.ExclusiveMinimum && lo.value < math.MaxInt64 {
			lo.value++
		}
	}
	if s.Maximum != nil {
		hi = limit[int64]{openapi.Int(s.Maximum, false), true}
		if s.ExclusiveMaximum && hi.value > math.MinInt64 {
			hi.value--
		}
	}
	if s.Format == "int32" {
		if !lo.set || lo.value < math.MinInt32 {
			lo = limit[int64]{math.MinInt32, true}
		}
		if !hi.set || hi.value > math.MaxInt32 {
			hi = limit[int64]{math.MaxInt32, true}
		}
	}
	var step int64 = 1
	if s.MultipleOf != nil {
		step = max(openapi.Int(s.MultipleOf, true), 1)
	}

	if g.dict != nil {
		fits := filter(g.dict.Ints, func(v int64) bool {
			return (!lo.set || v >= lo.value) && (!hi.set || v <= hi.value) && v%step == 0
		})
		if len(fits) > 0 {
			return g.pick(fits)
		}
	}

	from, to := effectiveRange(lo, hi, g.bounds.IntMin, g.bounds.IntMax)
	return intNode{lo: from, hi: to, step: step}
}

type intNode struct{ lo, hi, step int64 }

func (n intNode) generate(c ctx) any {
	if n.step == 1 {
		return randInt(c.r, n.lo, n.hi)
	}
	first := ceilDiv(n.lo, n.step)
	last := floorDiv(n.hi, n.step)
	if first > last {
		return first * n.step // no multiple in range: prefer the schema's multipleOf
	}
	return randInt(c.r, first, last) * n.step
}

func randInt(r *rand.Rand, lo, hi int64) int64 {
	width := uint64(hi) - uint64(lo) + 1
	if width == 0 { // the whole int64 range
		return int64(r.Uint64())
	}
	return int64(uint64(lo) + r.Uint64N(width))
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

func ceilDiv(a, b int64) int64 {
	q := a / b
	if a%b != 0 && (a < 0) == (b < 0) {
		q++
	}
	return q
}

// --- floats -----------------------------------------------------------------

func (g *Generator) number(s *openapi.Schema) node {
	var lo, hi limit[float64]
	if s.Minimum != nil {
		lo = limit[float64]{openapi.Float(s.Minimum), true}
		if s.ExclusiveMinimum {
			lo.value = math.Nextafter(lo.value, math.Inf(1))
		}
	}
	if s.Maximum != nil {
		hi = limit[float64]{openapi.Float(s.Maximum), true}
		if s.ExclusiveMaximum {
			hi.value = math.Nextafter(hi.value, math.Inf(-1))
		}
	}
	if s.Format == "float" {
		if !lo.set || lo.value < -math.MaxFloat32 {
			lo = limit[float64]{-math.MaxFloat32, true}
		}
		if !hi.set || hi.value > math.MaxFloat32 {
			hi = limit[float64]{math.MaxFloat32, true}
		}
	}
	var step float64
	if s.MultipleOf != nil {
		step = openapi.Float(s.MultipleOf)
	}

	if g.dict != nil {
		fits := filter(g.dict.Floats, func(v float64) bool {
			return (!lo.set || v >= lo.value) && (!hi.set || v <= hi.value) && isMultiple(v, step)
		})
		if len(fits) > 0 {
			return g.pick(fits)
		}
	}

	from, to := effectiveRange(lo, hi, g.bounds.FloatMin, g.bounds.FloatMax)
	return floatNode{lo: from, hi: to, step: step}
}

func isMultiple(v, step float64) bool {
	if step <= 0 {
		return true
	}
	q := v / step
	return math.Abs(q-math.Round(q)) < 1e-9
}

type floatNode struct{ lo, hi, step float64 }

func (n floatNode) generate(c ctx) any {
	if n.step > 0 {
		first, last := math.Ceil(n.lo/n.step), math.Floor(n.hi/n.step)
		k := first
		if last > first {
			k += math.Floor(c.r.Float64() * (last - first + 1))
		}
		return roundLike(k*n.step, n.step)
	}
	v := n.lo + c.r.Float64()*(n.hi-n.lo)
	// Three decimals keep inputs readable, unless the range is tighter than that.
	if rounded := math.Round(v*1000) / 1000; rounded >= n.lo && rounded <= n.hi {
		return rounded
	}
	return min(max(v, n.lo), n.hi)
}

// roundLike removes binary noise (0.1*3 = 0.30000000000000004) from v using the
// number of decimals of step, so that a multipleOf check on the server passes.
func roundLike(v, step float64) float64 {
	text := strconv.FormatFloat(step, 'f', -1, 64)
	_, decimals, _ := strings.Cut(text, ".")
	scale := math.Pow10(len(decimals))
	if rounded := math.Round(v*scale) / scale; !math.IsInf(rounded, 0) && !math.IsNaN(rounded) {
		return rounded
	}
	return v
}

// --- strings ----------------------------------------------------------------

func (g *Generator) str(s *openapi.Schema, loc Location) node {
	var lo, hi limit[int]
	if s.MinLength != nil {
		lo = limit[int]{*s.MinLength, true}
	}
	if s.MaxLength != nil {
		hi = limit[int]{*s.MaxLength, true}
	}
	if loc == InPath && (!lo.set || lo.value < 1) {
		lo = limit[int]{1, true} // an empty segment would change the route
	}

	var re *regexp.Regexp
	var pattern *regexGen
	if s.Pattern != "" {
		var err error
		if pattern, err = newRegexGen(s.Pattern); err != nil {
			g.warnings = append(g.warnings, fmt.Sprintf("pattern %q is not supported and was ignored: %v", s.Pattern, err))
		} else {
			re = pattern.re
		}
	}

	if g.dict != nil {
		fits := filter(g.dict.Strings, func(v string) bool {
			return fitsLength(v, lo, hi) && (re == nil || re.MatchString(v)) && fitsFormat(v, s.Format) && fitsLocation(v, loc)
		})
		if len(fits) > 0 {
			return g.pick(fits)
		}
	}

	from, to := effectiveRange(lo, hi, g.bounds.StringMin, g.bounds.StringMax)
	n := stringNode{lo: max(from, 0), hi: max(to, 0), alphabet: g.alphabet(loc), schemaLo: lo, schemaHi: hi}
	switch {
	case pattern != nil:
		n.pattern = pattern
	default:
		n.format = formatGenerators[s.Format]
	}
	return n
}

func (g *Generator) alphabet(loc Location) []rune {
	if g.bounds.Charset == CharsetAlnum || loc == InHeader {
		return alnumRunes
	}
	return unicodeRunes
}

func fitsLength(v string, lo, hi limit[int]) bool {
	n := utf8.RuneCountInString(v)
	return (!lo.set || n >= lo.value) && (!hi.set || n <= hi.value)
}

func fitsLocation(v string, loc Location) bool {
	switch loc {
	case InPath:
		return v != "." && v != ".." // would be normalized away, changing the route
	case InHeader:
		for i := 0; i < len(v); i++ {
			if v[i] < 0x20 || v[i] > 0x7e {
				return false
			}
		}
	}
	return true
}

func fitsFormat(v, format string) bool {
	switch format {
	case "date":
		_, err := time.Parse(time.DateOnly, v)
		return err == nil
	case "date-time":
		_, err := time.Parse(time.RFC3339, v)
		return err == nil
	case "uuid":
		return uuidRe.MatchString(v)
	case "email":
		a, err := mail.ParseAddress(v)
		return err == nil && a.Address == v
	case "uri", "url":
		u, err := url.Parse(v)
		return err == nil && u.Scheme != ""
	case "hostname":
		return hostnameRe.MatchString(v)
	case "ipv4":
		a, err := netip.ParseAddr(v)
		return err == nil && a.Is4()
	case "ipv6":
		a, err := netip.ParseAddr(v)
		return err == nil && a.Is6()
	case "byte":
		_, err := base64.StdEncoding.DecodeString(v)
		return err == nil
	}
	return true
}

var (
	uuidRe     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	hostnameRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
)

type stringNode struct {
	lo, hi             int
	schemaLo, schemaHi limit[int]
	alphabet           []rune
	pattern            *regexGen
	format             func(c ctx) string
}

func (n stringNode) generate(c ctx) any {
	switch {
	case n.pattern != nil:
		return n.pattern.generate(c.r, n.lo, n.hi, n.schemaLo, n.schemaHi)
	case n.format != nil:
		return n.format(c)
	}
	return randString(c.r, n.lo+c.r.IntN(n.hi-n.lo+1), n.alphabet)
}

func randString(r *rand.Rand, length int, alphabet []rune) string {
	var b strings.Builder
	b.Grow(length)
	for range length {
		b.WriteRune(alphabet[r.IntN(len(alphabet))])
	}
	return b.String()
}

var alnumRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// unicodeRunes holds alphanumerics from several scripts and byte widths.
// Controls, marks and bidi overrides are left out: they are noise, not input.
var unicodeRunes = func() []rune {
	out := append([]rune(nil), alnumRunes...)
	for _, block := range [][2]rune{
		{0x00C0, 0x00FF},   // Latin-1
		{0x0391, 0x03C9},   // Greek
		{0x0410, 0x044F},   // Cyrillic
		{0x0627, 0x064A},   // Arabic
		{0x0905, 0x0939},   // Devanagari
		{0x3041, 0x3096},   // Hiragana
		{0x4E00, 0x4E7F},   // CJK
		{0xAC00, 0xAC7F},   // Hangul
		{0x20000, 0x2007F}, // CJK extension B, four bytes in UTF-8
	} {
		for c := block[0]; c <= block[1]; c++ {
			if unicode.IsLetter(c) || unicode.IsDigit(c) {
				out = append(out, c)
			}
		}
	}
	return out
}()

var formatGenerators = map[string]func(c ctx) string{
	"uuid": func(c ctx) string {
		hi, lo := c.r.Uint64(), c.r.Uint64()
		hi = hi&^0xF000 | 0x4000           // version 4
		lo = lo&^(0b11<<62) | (0b10 << 62) // RFC 4122 variant
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", hi>>32, (hi>>16)&0xFFFF, hi&0xFFFF, lo>>48, lo&0xFFFFFFFFFFFF)
	},
	"date":      func(c ctx) string { return randTime(c.r).Format(time.DateOnly) },
	"date-time": func(c ctx) string { return randTime(c.r).Format(time.RFC3339) },
	"email":     func(c ctx) string { return randString(c.r, 4+c.r.IntN(5), alnumRunes) + "@example.com" },
	"uri":       randURL,
	"url":       randURL,
	"hostname":  func(c ctx) string { return randString(c.r, 4+c.r.IntN(5), alnumRunes) + ".example.com" },
	"ipv4": func(c ctx) string {
		v := c.r.Uint32()
		return netip.AddrFrom4([4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}).String()
	},
	"ipv6": func(c ctx) string {
		var a [16]byte
		for i := range a {
			a[i] = byte(c.r.Uint32())
		}
		a[0] = 0x20 // stay out of the IPv4-mapped range, which prints as IPv4
		return netip.AddrFrom16(a).String()
	},
	"byte": func(c ctx) string {
		raw := make([]byte, 1+c.r.IntN(16))
		for i := range raw {
			raw[i] = byte(c.r.Uint32())
		}
		return base64.StdEncoding.EncodeToString(raw)
	},
}

func randURL(c ctx) string {
	return "https://example.com/" + randString(c.r, 4+c.r.IntN(5), alnumRunes)
}

func randTime(r *rand.Rand) time.Time {
	const y2038 = 1 << 31
	return time.Unix(r.Int64N(y2038), 0).UTC()
}

// --- composites -------------------------------------------------------------

func (g *Generator) array(s *openapi.Schema, loc Location, nesting int) node {
	n := &arrayNode{item: g.compile(s.Items, loc, nesting), unique: s.UniqueItems}
	if s.MinItems != nil {
		n.lo = max(*s.MinItems, 0)
	}
	if loc != InBody && n.lo == 0 {
		n.lo = 1 // an empty array does not serialize to a parameter value
	}
	n.hi = n.lo + extraItems
	if s.MaxItems != nil {
		n.hi = min(n.hi, *s.MaxItems)
		n.lo = min(n.lo, *s.MaxItems)
	}
	n.hi = max(n.hi, n.lo)
	return n
}

type arrayNode struct {
	item   node
	lo, hi int
	unique bool
}

func (n *arrayNode) generate(c ctx) any {
	if c.depth >= hardDepth {
		return []any{}
	}
	count := n.lo
	if c.depth < softDepth {
		count += c.r.IntN(n.hi - n.lo + 1)
	}
	out := make([]any, 0, count)
	seen := map[string]bool{}

	item := c
	item.depth++
	// Bounded retries: the item schema may simply not have enough distinct values.
	for i := 0; len(out) < count && i < count*8; i++ {
		item.iter = c.iter + i
		v := n.item.generate(item)
		if n.unique {
			key := fmt.Sprint(v)
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		out = append(out, v)
	}
	return out
}

func (g *Generator) object(s *openapi.Schema, loc Location, nesting int) node {
	n := &objectNode{maxProps: math.MaxInt}
	if s.MinProperties != nil {
		n.minProps = *s.MinProperties
	}
	if s.MaxProperties != nil {
		n.maxProps = *s.MaxProperties
	}
	if loc != InBody {
		n.minProps = max(n.minProps, 1) // an empty object does not serialize to anything
	}

	var extraSchema *openapi.Schema
	extraAllowed := s.AdditionalProperties == nil || !s.AdditionalProperties.Forbidden
	if extraAllowed && s.AdditionalProperties != nil {
		extraSchema = s.AdditionalProperties.Schema
	}
	// Only needed when the named properties cannot reach minProperties.
	if extraAllowed && n.minProps > 0 {
		n.extra = g.compile(extraSchema, loc, nesting)
	}
	for _, name := range sortedNames(s.Properties) {
		prop := s.Properties[name]
		if prop.ReadOnly {
			continue // readOnly properties must not be sent in a request
		}
		n.props = append(n.props, objectProp{
			name:     name,
			required: slices.Contains(s.Required, name),
			value:    g.compile(prop, loc, nesting),
		})
	}
	// `required` may name properties the schema never defines.
	for _, name := range s.Required {
		if s.Properties[name] == nil && extraAllowed {
			n.props = append(n.props, objectProp{name: name, required: true, value: g.compile(extraSchema, loc, nesting)})
		}
	}
	return n
}

type objectProp struct {
	name     string
	required bool
	value    node
}

type objectNode struct {
	props              []objectProp
	minProps, maxProps int
	extra              node // value of generated keys; nil if they are not allowed
}

func (n *objectNode) generate(c ctx) any {
	out := make(map[string]any, len(n.props))
	if c.depth >= hardDepth {
		return out
	}
	child := c
	child.depth++

	for _, p := range n.props {
		if p.required {
			out[p.name] = p.value.generate(child)
		}
	}
	for _, p := range n.props {
		if !p.required && len(out) < n.maxProps && c.depth < softDepth && c.r.IntN(2) == 0 {
			out[p.name] = p.value.generate(child)
		}
	}
	// Top up to minProperties: first with what was skipped, then with new keys.
	for _, p := range n.props {
		if len(out) >= n.minProps {
			break
		}
		if _, set := out[p.name]; !set {
			out[p.name] = p.value.generate(child)
		}
	}
	for i := 0; n.extra != nil && len(out) < n.minProps; i++ {
		key := "key" + strconv.Itoa(i)
		if _, taken := out[key]; !taken {
			out[key] = n.extra.generate(child)
		}
	}
	return out
}

// --- leaves -----------------------------------------------------------------

type pickNode struct {
	vals   []any
	offset int
}

func (n pickNode) generate(c ctx) any {
	return n.vals[(c.iter+n.offset)%len(n.vals)]
}

type variantNode []node

func (n variantNode) generate(c ctx) any {
	return n[c.r.IntN(len(n))].generate(c)
}

type nullableNode struct{ node }

func (n nullableNode) generate(c ctx) any {
	if c.r.IntN(10) == 0 {
		return nil
	}
	return n.node.generate(c)
}

type boolNode struct{}

func (boolNode) generate(c ctx) any { return c.r.IntN(2) == 0 }

type nullNode struct{}

func (nullNode) generate(ctx) any { return nil }

func sortedNames(m map[string]*openapi.Schema) []string {
	return slices.Sorted(maps.Keys(m))
}
