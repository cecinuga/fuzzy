package generator

import (
	"math/rand/v2"
	"regexp"
	"regexp/syntax"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxRepeat       = 8  // cap for the unbounded * and + operators
	patternAttempts = 16 // anchors in odd places can make a candidate fail to match
	classAttempts   = 32
)

// regexGen produces strings that match a pattern by walking its syntax tree.
type regexGen struct {
	tree *syntax.Regexp
	re   *regexp.Regexp
}

func newRegexGen(pattern string) (*regexGen, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	tree, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return nil, err
	}
	return &regexGen{tree: tree.Simplify(), re: re}, nil
}

// generate prefers a match whose length is inside [lo, hi]. If the pattern
// cannot produce one, a match that respects the schema's own length limits is
// still valid, so it is returned rather than giving up.
func (g *regexGen) generate(r *rand.Rand, lo, hi int, schemaLo, schemaHi limit[int]) string {
	var fallback, last string
	for range patternAttempts {
		var b strings.Builder
		g.walk(r, g.tree, &b)
		last = b.String()
		// An unanchored pattern can match inside a longer string, so a candidate
		// that came out too short may just need padding to reach the minimum.
		if missing := lo - utf8.RuneCountInString(last); missing > 0 {
			last += randString(r, missing, alnumRunes)
		}
		if !g.re.MatchString(last) {
			continue
		}
		if fitsLength(last, limit[int]{lo, true}, limit[int]{hi, true}) {
			return last
		}
		if fallback == "" && fitsLength(last, schemaLo, schemaHi) {
			fallback = last
		}
	}
	if fallback != "" {
		return fallback
	}
	return last
}

func (g *regexGen) walk(r *rand.Rand, re *syntax.Regexp, b *strings.Builder) {
	switch re.Op {
	case syntax.OpLiteral:
		for _, c := range re.Rune {
			b.WriteRune(c)
		}
	case syntax.OpCharClass:
		b.WriteRune(pickFromClass(r, re.Rune))
	case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		b.WriteRune(alnumRunes[r.IntN(len(alnumRunes))])
	case syntax.OpCapture, syntax.OpConcat:
		for _, sub := range re.Sub {
			g.walk(r, sub, b)
		}
	case syntax.OpAlternate:
		g.walk(r, re.Sub[r.IntN(len(re.Sub))], b)
	case syntax.OpQuest:
		if r.IntN(2) == 0 {
			g.walk(r, re.Sub[0], b)
		}
	case syntax.OpStar:
		g.repeat(r, re.Sub[0], r.IntN(maxRepeat+1), b)
	case syntax.OpPlus:
		g.repeat(r, re.Sub[0], 1+r.IntN(maxRepeat), b)
	case syntax.OpRepeat: // Simplify expands these, but stay correct regardless
		count := re.Min
		if re.Max > re.Min {
			count += r.IntN(re.Max - re.Min + 1)
		} else if re.Max < 0 {
			count += r.IntN(maxRepeat + 1)
		}
		g.repeat(r, re.Sub[0], count, b)
	}
	// Anchors and empty matches emit nothing.
}

func (g *regexGen) repeat(r *rand.Rand, sub *syntax.Regexp, count int, b *strings.Builder) {
	for range count {
		g.walk(r, sub, b)
	}
}

// pickFromClass favors plain alphanumerics, since a negated class such as [^/]
// also contains every control character, then any printable member.
func pickFromClass(r *rand.Rand, pairs []rune) rune {
	for range classAttempts {
		if c := alnumRunes[r.IntN(len(alnumRunes))]; inClass(c, pairs) {
			return c
		}
	}
	var c rune
	for range classAttempts {
		i := r.IntN(len(pairs)/2) * 2
		c = pairs[i] + rune(r.Int32N(pairs[i+1]-pairs[i]+1))
		if unicode.IsPrint(c) {
			return c
		}
	}
	return c
}

func inClass(c rune, pairs []rune) bool {
	for i := 0; i+1 < len(pairs); i += 2 {
		if c >= pairs[i] && c <= pairs[i+1] {
			return true
		}
	}
	return false
}
