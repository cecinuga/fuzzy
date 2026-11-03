package generator

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func writeWordlist(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "words.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func openTestWordlist(t *testing.T, content string) *Dictionary {
	t.Helper()
	d, err := openWordlist(writeWordlist(t, content))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(d.Close)
	return d
}

// mixedWords spans many blocks and has no duplicates, so both modes hold the
// same values in the same order.
func mixedWords() string {
	var b strings.Builder
	for i := range 12000 {
		switch i % 7 {
		case 0:
			fmt.Fprintf(&b, "%d\n", i)
		case 1:
			fmt.Fprintf(&b, "%d.5\r\n", i)
		case 2:
			fmt.Fprintf(&b, "word%d\n", i)
		case 3:
			fmt.Fprintf(&b, "\n   \nè%d-ünicode\n", i)
		case 4:
			fmt.Fprintf(&b, "user%d@example.com\n", i)
		case 5:
			fmt.Fprintf(&b, "%s%d\n", strings.Repeat("long", 20), i)
		case 6:
			fmt.Fprintf(&b, "192.168.%d.%d\n", i%256, i%251)
		}
	}
	b.WriteString("true\nfalse\nlast-without-newline")
	return b.String()
}

var streamedSchemas = []struct {
	schema string
	loc    Location
}{
	{`{"type":"string"}`, InBody},
	{`{"type":"string"}`, InQuery}, // a second node over the same list, with its own offset
	{`{"type":"string","maxLength":8}`, InBody},
	{`{"type":"string","minLength":70}`, InQuery},
	{`{"type":"string"}`, InHeader},
	{`{"type":"string"}`, InPath},
	{`{"type":"string","pattern":"^word[0-9]+$"}`, InBody},
	{`{"type":"string","format":"email"}`, InBody},
	{`{"type":"string","format":"ipv4"}`, InQuery},
	{`{"type":"string","maxLength":0}`, InBody},
	{`{"type":"integer"}`, InBody},
	{`{"type":"integer","minimum":100,"maximum":5000,"multipleOf":7}`, InQuery},
	{`{"type":"integer","minimum":999999}`, InBody},
	{`{"type":"number"}`, InBody},
	{`{"type":"number","minimum":10.5,"maximum":900.5}`, InBody},
	{`{"type":"boolean"}`, InBody},
	{`{"type":"array","items":{"type":"string","maxLength":8},"minItems":1}`, InBody},
	{`{"type":"object","required":["a","b"],"properties":{"a":{"type":"integer"},"b":{"type":"string"}}}`, InBody},
}

func TestWordlistGeneratesTheSameValuesAsALoadedDictionary(t *testing.T) {
	content := mixedWords()
	loaded := writeDict(t, content)
	streamed := openTestWordlist(t, content)

	if loaded.Size() != streamed.Size() {
		t.Fatalf("Size: loaded %d, streamed %d", loaded.Size(), streamed.Size())
	}
	if len(streamed.lines.offsets) < 10 {
		t.Fatalf("the content must span many blocks, got %d", len(streamed.lines.offsets)-1)
	}

	const seed = 7
	a, b := New(DefaultBounds(), loaded, seed), New(DefaultBounds(), streamed, seed)
	for _, tc := range streamedSchemas {
		s := parseSchema(t, tc.schema, "")
		sa, sb := a.Compile(s, tc.loc), b.Compile(s, tc.loc)
		for iter := 0; iter < 100_000; iter += 97 { // spread out, to reach every block and wrap around
			ra := rand.New(rand.NewPCG(seed, uint64(iter)))
			rb := rand.New(rand.NewPCG(seed, uint64(iter)))
			got, want := sb.Generate(rb, iter), sa.Generate(ra, iter)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%s iter %d: streamed %#v, loaded %#v", tc.schema, iter, got, want)
			}
		}
	}
	if err := streamed.Err(); err != nil {
		t.Error(err)
	}
}

func TestWordlistKeepsDuplicatesAndSkipsBlankLines(t *testing.T) {
	d := openTestWordlist(t, "a\n\n  \t \na\r\nb\ntrue\ntrue\n")
	if d.Size() != 5 {
		t.Errorf("Size = %d, want 5: duplicates are kept, blank lines are not", d.Size())
	}
	if !reflect.DeepEqual(d.Bools, []bool{true}) {
		t.Errorf("Bools = %v, want each one once", d.Bools)
	}
}

func TestWordlistRejectsAnEmptyFile(t *testing.T) {
	for _, content := range []string{"", "\n\n   \r\n"} {
		if _, err := openWordlist(writeWordlist(t, content)); err == nil {
			t.Errorf("%q: an empty dictionary is a mistake worth reporting", content)
		}
	}
	if _, err := openWordlist(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("a missing dictionary must fail")
	}
}

func TestLoadDictionaryStreamsOnlyAboveTheThreshold(t *testing.T) {
	sized := func(size int64) string {
		path := writeWordlist(t, "")
		if err := os.Truncate(path, size); err != nil {
			t.Fatal(err)
		}
		return path
	}

	at, err := LoadDictionary(sized(streamThreshold))
	if err != nil {
		t.Fatal(err)
	}
	if at.lines != nil {
		t.Error("a file at the threshold must be loaded")
	}

	above, err := LoadDictionary(sized(streamThreshold + 1))
	if err != nil {
		t.Fatal(err)
	}
	defer above.Close()
	if above.lines == nil {
		t.Error("a file above the threshold must stay on disk")
	}
}

func TestWordlistHandlesLinesBiggerThanABlockAndTheIndexBuffer(t *testing.T) {
	huge := strings.Repeat("x", 2*indexBufferSize+123)
	spaces := strings.Repeat(" ", 2*indexBufferSize)
	d := openTestWordlist(t, "first\n"+huge+"\n"+spaces+"\ntrue"+spaces+"\nlast\n")

	if d.Size() != 4 {
		t.Fatalf("Size = %d, want 4", d.Size())
	}
	if len(d.Bools) != 0 {
		t.Errorf("Bools = %v: a long line that starts with true isn't a boolean", d.Bools)
	}
	got := map[string]bool{}
	source := New(DefaultBounds(), d, 1).Compile(parseSchema(t, `{"type":"string","minLength":5,"maxLength":5}`, ""), InBody)
	for iter := range 2 {
		got[source.Generate(rand.New(rand.NewPCG(1, 1)), iter).(string)] = true
	}
	if !got["first"] {
		t.Errorf("got %v", got)
	}
	source = New(DefaultBounds(), d, 1).Compile(parseSchema(t, fmt.Sprintf(`{"type":"string","minLength":%d}`, len(huge)), ""), InBody)
	if v := source.Generate(rand.New(rand.NewPCG(1, 1)), 0); v != huge {
		t.Errorf("the long line changed: %d bytes", len(v.(string)))
	}
}

func TestSchemasWithTheSameConstraintsShareOneList(t *testing.T) {
	for name, d := range map[string]*Dictionary{
		"loaded":   writeDict(t, mixedWords()),
		"streamed": openTestWordlist(t, mixedWords()),
	} {
		t.Run(name, func(t *testing.T) {
			g := New(DefaultBounds(), d, 1)
			offsets := map[any]bool{}
			for range 20 {
				// A new schema each time: what's shared are the constraints, not the object.
				n := g.Compile(parseSchema(t, `{"type":"string","maxLength":8}`, ""), InQuery)
				offsets[n.Generate(rand.New(rand.NewPCG(1, 1)), 0)] = true
			}
			g.Compile(parseSchema(t, `{"type":"string","maxLength":9}`, ""), InQuery)
			if lists := len(g.fitting) + len(g.views); lists != 2 {
				t.Errorf("%d lists, want 2", lists)
			}
			if len(offsets) < 2 {
				t.Error("each schema must still start from its own offset")
			}
		})
	}
}

func TestWordlistIsSafeAndStableUnderConcurrencyWithEvictions(t *testing.T) {
	d := openTestWordlist(t, mixedWords())
	d.lines.cache.limit = 2 * blockSize // a cache this small keeps evicting
	source := New(DefaultBounds(), d, 3).Compile(parseSchema(t, `{"type":"string","maxLength":12}`, ""), InBody)

	const iters = 4000
	want := make([]any, iters)
	for i := range want {
		want[i] = source.Generate(rand.New(rand.NewPCG(1, uint64(i))), i)
	}

	var workers sync.WaitGroup
	for w := range 8 {
		workers.Go(func() {
			for i := range iters {
				i := (i + w*97) % iters // workers ask for different blocks at the same time
				if got := source.Generate(rand.New(rand.NewPCG(1, uint64(i))), i); got != want[i] {
					t.Errorf("iter %d: got %v, want %v", i, got, want[i])
					return
				}
			}
		})
	}
	workers.Wait()

	if d.lines.cache.used > d.lines.cache.limit {
		t.Errorf("the cache holds %d bytes, over its limit of %d", d.lines.cache.used, d.lines.cache.limit)
	}
}

func TestWordlistFallsBackToValidValuesWhenTheFileGoesBad(t *testing.T) {
	var content strings.Builder
	for i := range 20000 {
		fmt.Fprintf(&content, "%d\n", 1000+i)
	}
	stringSchema := `{"type":"string","minLength":2,"maxLength":4}`
	intSchema := `{"type":"integer","minimum":1000,"maximum":30000}`

	bad := map[string]func(t *testing.T, path string){
		"truncated": func(t *testing.T, path string) {
			if err := os.Truncate(path, 0); err != nil {
				t.Fatal(err)
			}
		},
		"rewritten": func(t *testing.T, path string) {
			other := strings.Repeat("x", content.Len()-1) + "\n"
			if err := os.WriteFile(path, []byte(other), 0o600); err != nil {
				t.Fatal(err)
			}
		},
	}
	for name, spoil := range bad {
		t.Run(name, func(t *testing.T) {
			path := writeWordlist(t, content.String())
			d, err := openWordlist(path)
			if err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			g := New(DefaultBounds(), d, 1)
			str := g.Compile(parseSchema(t, stringSchema, ""), InBody)
			num := g.Compile(parseSchema(t, intSchema, ""), InBody)
			if w := g.Warnings(); len(w) != 0 {
				t.Fatalf("warnings before anything failed: %v", w)
			}

			spoil(t, path)
			r := rand.New(rand.NewPCG(1, 1))
			for iter := range 50 {
				if v, ok := str.Generate(r, iter).(string); !ok || len([]rune(v)) < 2 || len([]rune(v)) > 4 {
					t.Fatalf("invalid fallback string %q", v)
				}
				if v, ok := num.Generate(r, iter).(int64); !ok || v < 1000 || v > 30000 {
					t.Fatalf("invalid fallback integer %v", v)
				}
			}
			if d.Err() == nil {
				t.Error("the failure must be recorded")
			}
			if w := g.Warnings(); len(w) != 1 || !strings.Contains(w[0], "dictionary values were replaced") {
				t.Errorf("warnings = %v", w)
			}
		})
	}
}

func TestViewPassOverABadFileStillCompiles(t *testing.T) {
	d := openTestWordlist(t, mixedWords())
	d.Close() // every read now fails
	source := New(DefaultBounds(), d, 1).Compile(parseSchema(t, `{"type":"string","maxLength":5}`, ""), InBody)
	if v, ok := source.Generate(rand.New(rand.NewPCG(1, 1)), 0).(string); !ok || len([]rune(v)) > 5 {
		t.Errorf("want a random valid value, got %v", v)
	}
	if d.Err() == nil {
		t.Error("the failure must be recorded")
	}
}

func TestNilDictionaryMethodsAreSafe(t *testing.T) {
	var d *Dictionary
	d.Close()
	if d.Err() != nil || d.Size() != 0 {
		t.Error("a nil dictionary has no error and no values")
	}
	if w := New(DefaultBounds(), nil, 1).Warnings(); len(w) != 0 {
		t.Errorf("warnings = %v", w)
	}
}

func BenchmarkWordlistLookup(b *testing.B) {
	var content strings.Builder
	for i := range 200000 {
		fmt.Fprintf(&content, "word%d\n", i)
	}
	path := filepath.Join(b.TempDir(), "words.txt")
	if err := os.WriteFile(path, []byte(content.String()), 0o600); err != nil {
		b.Fatal(err)
	}
	d, err := openWordlist(path)
	if err != nil {
		b.Fatal(err)
	}
	defer d.Close()
	v := d.lines.view(func(string) bool { return true }, func(line string) any { return line })
	b.ResetTimer()
	for i := range b.N {
		v.at(i % v.size())
	}
}
