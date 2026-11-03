package fuzzer

import (
	"fmt"
	"slices"
	"strings"

	"fuzzy/internal/openapi"
)

var httpMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "TRACE"}

const tagPrefix = "tag:"

// Selection decides which endpoints of the spec a run fuzzes: those matching any
// include (or all of them, without includes) and no exclude, so an exclude always
// wins. The zero value selects everything.
type Selection struct {
	include, exclude []selector
}

// selector matches endpoints. An empty part matches anything.
type selector struct {
	raw     string
	methods []string
	path    []string // the pattern's segments
	tag     string
}

// ParseSelection parses the --include and --exclude selectors. A selector is a
// method list, a path pattern, a tag, or a method list followed by either:
//
//	GET    GET,POST    /users/*    /users/**    tag:billing    GET /user/{id}
func ParseSelection(include, exclude []string) (Selection, error) {
	var s Selection
	var err error
	if s.include, err = parseSelectors(include); err != nil {
		return s, fmt.Errorf("--include: %w", err)
	}
	if s.exclude, err = parseSelectors(exclude); err != nil {
		return s, fmt.Errorf("--exclude: %w", err)
	}
	return s, nil
}

func parseSelectors(raw []string) ([]selector, error) {
	selectors := make([]selector, len(raw))
	for i, entry := range raw {
		sel, err := parseSelector(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid selector %q: %w", entry, err)
		}
		selectors[i] = sel
	}
	return selectors, nil
}

func parseSelector(raw string) (selector, error) {
	sel := selector{raw: strings.TrimSpace(raw)}
	rest := sel.raw
	if rest == "" {
		return sel, fmt.Errorf("it is empty")
	}

	// Only what comes first can be methods. The rest is kept whole, because a tag
	// may contain spaces.
	if !strings.HasPrefix(rest, "/") && !strings.HasPrefix(rest, tagPrefix) {
		list, after, _ := strings.Cut(rest, " ")
		for method := range strings.SplitSeq(list, ",") {
			method = strings.ToUpper(method)
			if !slices.Contains(httpMethods, method) {
				return sel, fmt.Errorf("want a method, a /path or %sname, e.g. 'GET /users/*'", tagPrefix)
			}
			sel.methods = append(sel.methods, method)
		}
		rest = strings.TrimSpace(after)
	}

	switch {
	case rest == "":
	case strings.HasPrefix(rest, "/"):
		sel.path = splitPath(rest)
		for _, segment := range sel.path {
			if strings.Contains(segment, "*") && segment != "*" && segment != "**" {
				return sel, fmt.Errorf("a wildcard must be a whole path segment, got %q", segment)
			}
		}
	case strings.HasPrefix(rest, tagPrefix):
		sel.tag = strings.TrimSpace(strings.TrimPrefix(rest, tagPrefix))
		if sel.tag == "" {
			return sel, fmt.Errorf("%s needs a name", tagPrefix)
		}
	default:
		return sel, fmt.Errorf("after the methods, want a /path or %sname", tagPrefix)
	}
	return sel, nil
}

func splitPath(path string) []string {
	return strings.Split(strings.TrimPrefix(path, "/"), "/")
}

func (sel *selector) matches(op *openapi.Operation) bool {
	return (sel.methods == nil || slices.Contains(sel.methods, op.Method)) &&
		(sel.path == nil || matchSegments(sel.path, splitPath(op.Path))) &&
		(sel.tag == "" || slices.Contains(op.Tags, sel.tag))
}

// matchSegments compares a path with a pattern, where * stands for one segment
// and ** for any number of them, including none. Anything else, {id} included,
// matches literally, so a path can be pasted from the report as it is.
func matchSegments(pattern, path []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			for i := range len(path) + 1 {
				if matchSegments(pattern[1:], path[i:]) {
					return true
				}
			}
			return false
		}
		if len(path) == 0 {
			return false
		}
		if pattern[0] == "*" {
			if path[0] == "" {
				return false
			}
		} else if pattern[0] != path[0] {
			return false
		}
		pattern, path = pattern[1:], path[1:]
	}
	return len(path) == 0
}

// Active reports whether the selection can leave any endpoint out.
func (s Selection) Active() bool { return len(s.include)+len(s.exclude) > 0 }

// Include returns the --include selectors as given on the command line.
func (s Selection) Include() []string { return rawSelectors(s.include) }

// Exclude returns the --exclude selectors as given on the command line.
func (s Selection) Exclude() []string { return rawSelectors(s.exclude) }

func rawSelectors(selectors []selector) []string {
	raw := make([]string, len(selectors))
	for i := range selectors {
		raw[i] = selectors[i].raw
	}
	return raw
}

// Apply keeps the selected targets and skipped operations, so an endpoint left
// out doesn't show up as one that couldn't be fuzzed.
//
// It must run after Compile, on everything Compile returned. Compiling consumes
// the seeded generator's offsets, so leaving operations out of Compile would give
// the remaining ones different dictionary values for the same seed.
//
// A selector matching no endpoint is an error: otherwise a typo in an --exclude
// would silently fuzz the endpoints it was meant to protect.
func (s Selection) Apply(targets []*Target, skipped []Skipped) ([]*Target, []Skipped, error) {
	if !s.Active() {
		return targets, skipped, nil
	}

	included := make([]bool, len(s.include))
	excluded := make([]bool, len(s.exclude))
	selected := func(op *openapi.Operation) bool {
		// No early return: every selector has to learn whether it matched.
		in, out := len(s.include) == 0, false
		for i := range s.include {
			if s.include[i].matches(op) {
				in, included[i] = true, true
			}
		}
		for i := range s.exclude {
			if s.exclude[i].matches(op) {
				out, excluded[i] = true, true
			}
		}
		return in && !out
	}

	keptTargets := make([]*Target, 0, len(targets))
	for _, t := range targets {
		if selected(t.Op) {
			keptTargets = append(keptTargets, t)
		}
	}
	keptSkipped := make([]Skipped, 0, len(skipped))
	for _, sk := range skipped {
		if selected(sk.Op) {
			keptSkipped = append(keptSkipped, sk)
		}
	}

	var unmatched []string
	for i := range s.include {
		if !included[i] {
			unmatched = append(unmatched, fmt.Sprintf("--include %q", s.include[i].raw))
		}
	}
	for i := range s.exclude {
		if !excluded[i] {
			unmatched = append(unmatched, fmt.Sprintf("--exclude %q", s.exclude[i].raw))
		}
	}
	if len(unmatched) > 0 {
		return nil, nil, fmt.Errorf("no endpoint of the spec matches %s", strings.Join(unmatched, ", "))
	}
	return keptTargets, keptSkipped, nil
}
