package fuzzer

import (
	"fmt"
	"strconv"
	"strings"
)

// TransportError is the status recorded when no HTTP response was received.
const TransportError = 0

// StatusFilter decides which results are kept for the report. The zero value
// keeps everything.
type StatusFilter struct {
	raw    string
	ranges [][2]int
	errs   bool
}

// ParseStatusFilter parses a comma separated list of exact codes (404), classes
// (5xx), ranges (400-404) and "err" for requests that got no response at all.
// An empty list keeps every result.
func ParseStatusFilter(list string) (StatusFilter, error) {
	f := StatusFilter{raw: strings.TrimSpace(list)}
	if f.raw == "" {
		return f, nil
	}
	for token := range strings.SplitSeq(f.raw, ",") {
		token = strings.ToLower(strings.TrimSpace(token))
		switch {
		case token == "err":
			f.errs = true
		case len(token) == 3 && strings.HasSuffix(token, "xx"):
			class, err := strconv.Atoi(token[:1])
			if err != nil || class < 1 || class > 5 {
				return f, fmt.Errorf("invalid status class %q", token)
			}
			f.ranges = append(f.ranges, [2]int{class * 100, class*100 + 99})
		default:
			from, to, isRange := strings.Cut(token, "-")
			if !isRange {
				to = from
			}
			lo, errLo := strconv.Atoi(from)
			hi, errHi := strconv.Atoi(to)
			if errLo != nil || errHi != nil || lo < 100 || hi > 599 || lo > hi {
				return f, fmt.Errorf("invalid status %q (want e.g. 200, 5xx, 400-404, err)", token)
			}
			f.ranges = append(f.ranges, [2]int{lo, hi})
		}
	}
	return f, nil
}

// Match reports whether a result with this status belongs in the report.
func (f StatusFilter) Match(status int) bool {
	if f.raw == "" {
		return true
	}
	if status == TransportError {
		return f.errs
	}
	for _, r := range f.ranges {
		if status >= r[0] && status <= r[1] {
			return true
		}
	}
	return false
}

// String returns the filter as given on the command line.
func (f StatusFilter) String() string {
	if f.raw == "" {
		return "all"
	}
	return f.raw
}
