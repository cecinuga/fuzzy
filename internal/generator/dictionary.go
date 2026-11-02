package generator

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Dictionary holds the values of a wordlist, grouped by the types they are
// valid for. A line can serve several types: "42" is an int, a float and a string.
type Dictionary struct {
	Strings []string
	Ints    []int64
	Floats  []float64
	Bools   []bool
}

// numberRe excludes what ParseFloat accepts but JSON does not (inf, nan, hex).
var numberRe = regexp.MustCompile(`^[+-]?(\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?$`)

// LoadDictionary reads a wordlist with one value per line. Blank lines are
// skipped; every other line is kept verbatim and classified by what it parses as.
func LoadDictionary(path string) (*Dictionary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dictionary: %w", err)
	}

	d := &Dictionary{}
	seen := map[string]bool{}
	for line := range strings.Lines(string(data)) {
		line = strings.TrimRight(line, "\r\n")
		if strings.TrimSpace(line) == "" || seen[line] {
			continue
		}
		seen[line] = true
		d.add(line)
	}
	if len(d.Strings) == 0 {
		return nil, errors.New("dictionary is empty: " + path)
	}
	return d, nil
}

func (d *Dictionary) add(line string) {
	d.Strings = append(d.Strings, line)

	switch line {
	case "true":
		d.Bools = append(d.Bools, true)
	case "false":
		d.Bools = append(d.Bools, false)
	}
	if !numberRe.MatchString(line) {
		return
	}
	if i, err := strconv.ParseInt(line, 10, 64); err == nil {
		d.Ints = append(d.Ints, i)
	}
	if f, err := strconv.ParseFloat(line, 64); err == nil {
		d.Floats = append(d.Floats, f)
	}
}

// Size returns how many distinct values the wordlist holds.
func (d *Dictionary) Size() int {
	if d == nil {
		return 0
	}
	return len(d.Strings)
}
