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
// A file above streamThreshold stays on disk: lines replaces Strings, Ints and
// Floats, and only Bools is kept in memory.
type Dictionary struct {
	Strings []string
	Ints    []int64
	Floats  []float64
	Bools   []bool

	lines *wordlist
}

// numberRe excludes what ParseFloat accepts but JSON does not (inf, nan, hex).
var numberRe = regexp.MustCompile(`^[+-]?(\d+(\.\d*)?|\.\d+)([eE][+-]?\d+)?$`)

// LoadDictionary reads a wordlist with one value per line. Blank lines are
// skipped; every other line is kept verbatim and classified by what it parses as.
// A regular file above streamThreshold is indexed instead of read, and keeps its
// duplicate lines, because finding them takes memory for every line. Anything
// else, such as a pipe, can't be read by position and is always loaded.
func LoadDictionary(path string) (*Dictionary, error) {
	if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() && info.Size() > streamThreshold {
		return openWordlist(path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dictionary: %w", err)
	}

	d := &Dictionary{}
	seen := map[string]bool{}
	for raw := range strings.Lines(string(data)) {
		line, ok := cleanLine(raw)
		if !ok || seen[line] {
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

// cleanLine strips a raw line's terminator. ok is false for a blank line.
func cleanLine(raw string) (line string, ok bool) {
	line = strings.TrimRight(raw, "\r\n")
	return line, strings.TrimSpace(line) != ""
}

func (d *Dictionary) add(line string) {
	d.Strings = append(d.Strings, line)

	switch line {
	case "true":
		d.Bools = append(d.Bools, true)
	case "false":
		d.Bools = append(d.Bools, false)
	}
	if i, ok := parseInt(line); ok {
		d.Ints = append(d.Ints, i)
	}
	if f, ok := parseFloat(line); ok {
		d.Floats = append(d.Floats, f)
	}
}

func parseString(line string) (string, bool) { return line, true }

func parseInt(line string) (int64, bool) {
	if !isNumber(line) {
		return 0, false
	}
	i, err := strconv.ParseInt(line, 10, 64)
	return i, err == nil
}

func parseFloat(line string) (float64, bool) {
	if !isNumber(line) {
		return 0, false
	}
	f, err := strconv.ParseFloat(line, 64)
	return f, err == nil
}

// isNumber checks the first byte before the regexp: most lines of a wordlist
// are words, and a big one is parsed again on every read.
func isNumber(line string) bool {
	return line != "" && strings.IndexByte("+-.0123456789", line[0]) >= 0 && numberRe.MatchString(line)
}

// Size returns how many values the wordlist holds: the distinct ones, or every
// non-blank line of a file above streamThreshold.
func (d *Dictionary) Size() int {
	switch {
	case d == nil:
		return 0
	case d.lines != nil:
		return d.lines.size
	}
	return len(d.Strings)
}

// Err returns the first error met reading a wordlist after it was opened.
func (d *Dictionary) Err() error {
	if d == nil || d.lines == nil {
		return nil
	}
	return d.lines.err()
}

// Close releases the file of a wordlist that stays on disk.
func (d *Dictionary) Close() {
	if d != nil && d.lines != nil {
		_ = d.lines.file.Close()
	}
}
