// Package generator turns OpenAPI schemas into generators of valid values,
// drawn from a dictionary when it has a fitting value and at random otherwise.
package generator

import "fmt"

// Charset selects the characters of random strings.
type Charset string

const (
	CharsetUnicode Charset = "unicode" // alphanumeric plus letters of other scripts
	CharsetAlnum   Charset = "alnum"   // ASCII letters and digits only
)

// Bounds limits randomly generated values. Schema constraints always win: a
// value is drawn from the intersection of both ranges, and when the two do not
// overlap the schema's range is used, because the request must stay valid.
type Bounds struct {
	StringMin, StringMax int
	IntMin, IntMax       int64
	FloatMin, FloatMax   float64
	Charset              Charset
}

// DefaultBounds returns the limits used when no flag overrides them.
func DefaultBounds() Bounds {
	return Bounds{
		StringMin: 1, StringMax: 10,
		IntMin: 0, IntMax: 1000,
		FloatMin: 0, FloatMax: 1000.0,
		Charset: CharsetUnicode,
	}
}

// Validate rejects ranges that cannot produce a value.
func (b Bounds) Validate() error {
	switch {
	case b.StringMin < 0:
		return fmt.Errorf("string minimum length must not be negative (got %d)", b.StringMin)
	case b.StringMin > b.StringMax:
		return fmt.Errorf("string minimum length %d exceeds maximum %d", b.StringMin, b.StringMax)
	case b.IntMin > b.IntMax:
		return fmt.Errorf("int minimum %d exceeds maximum %d", b.IntMin, b.IntMax)
	case !(b.FloatMin <= b.FloatMax): // also catches NaN
		return fmt.Errorf("float minimum %v exceeds maximum %v", b.FloatMin, b.FloatMax)
	case b.Charset != CharsetUnicode && b.Charset != CharsetAlnum:
		return fmt.Errorf("unknown charset %q (want %s or %s)", b.Charset, CharsetUnicode, CharsetAlnum)
	}
	return nil
}

// limit is an optional schema bound.
type limit[T int | int64 | float64] struct {
	value T
	set   bool
}

// effectiveRange intersects the schema's range with the configured one. When
// they are disjoint the schema's range wins, and a side the schema leaves open
// keeps the width of the configured window.
func effectiveRange[T int | int64 | float64](lo, hi limit[T], cfgLo, cfgHi T) (T, T) {
	from, to := cfgLo, cfgHi
	if lo.set && lo.value > from {
		from = lo.value
	}
	if hi.set && hi.value < to {
		to = hi.value
	}
	if from <= to {
		return from, to
	}

	span := cfgHi - cfgLo
	switch {
	case lo.set && hi.set:
		return lo.value, max(lo.value, hi.value)
	case lo.set:
		if lo.value+span < lo.value { // overflow
			return lo.value, lo.value
		}
		return lo.value, lo.value + span
	default:
		if hi.value-span > hi.value { // overflow
			return hi.value, hi.value
		}
		return hi.value - span, hi.value
	}
}
