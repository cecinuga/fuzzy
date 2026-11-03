package progress

import (
	"bytes"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"fuzzy/internal/fuzzer"
)

// live returns a Display that draws as on a terminal of the given width.
func live(width int) (*Display, *bytes.Buffer) {
	var out bytes.Buffer
	d := New(&out)
	d.width = func() int { return width }
	return d, &out
}

var midRun = fuzzer.Progress{Done: 300, Total: 600, ServerErrors: 17, TransportErrors: 2, Elapsed: 3 * time.Second}

func TestPlainOutputWhenNotATerminal(t *testing.T) {
	var out bytes.Buffer
	d := New(&out)
	if d.Interval() != plainInterval {
		t.Errorf("interval = %s, want %s", d.Interval(), plainInterval)
	}
	d.Update(midRun)
	d.Finish()
	if got, want := out.String(), "[+] 300/600 requests\n"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestStatusLineContent(t *testing.T) {
	got := statusLine(midRun, '*', 200)
	want := "* [██████████░░░░░░░░░░]  50% 300/600 req · 100 req/s · ETA 3s · 5xx 17 · err 2"
	if got != want {
		t.Errorf("line =\n%q, want\n%q", got, want)
	}
}

func TestStatusLineBeforeAnyResponse(t *testing.T) {
	got := statusLine(fuzzer.Progress{Total: 600, Elapsed: time.Second}, '*', 200)
	if !strings.Contains(got, "  0% 0/600 req") || !strings.Contains(got, "ETA --") {
		t.Errorf("line = %q", got)
	}
}

func TestStatusLineEnds(t *testing.T) {
	got := statusLine(fuzzer.Progress{Done: 600, Total: 600, Elapsed: time.Second}, '*', 200)
	if !strings.Contains(got, "["+strings.Repeat("█", barWidth)+"] 100% 600/600 req") || !strings.Contains(got, "ETA 0s") {
		t.Errorf("line = %q", got)
	}
}

func TestStatusLineWithoutTotal(t *testing.T) {
	if got := statusLine(fuzzer.Progress{}, '*', 200); !strings.Contains(got, "0% 0/0 req") {
		t.Errorf("line = %q", got)
	}
}

func TestStatusLineNeverExceedsWidth(t *testing.T) {
	big := fuzzer.Progress{Done: 123456, Total: 9999999, ServerErrors: 54321, TransportErrors: 54321, Elapsed: time.Minute}
	for width := 0; width <= 120; width++ {
		if got := utf8.RuneCountInString(statusLine(big, '*', width)); got > width {
			t.Errorf("width %d: line has %d columns", width, got)
		}
	}
}

func TestStatusLineDropsLeastUsefulSegmentsFirst(t *testing.T) {
	cases := []struct {
		width         int
		kept, dropped []string
	}{
		{70, []string{"[", "300/600", "ETA", "5xx 17"}, []string{"req/s"}},
		{60, []string{"[", "300/600", "5xx 17"}, []string{"req/s", "ETA"}},
		{40, []string{"300/600", "5xx 17", "err 2"}, []string{"req/s", "ETA", "["}},
		{20, []string{"300/600"}, []string{"req/s", "ETA", "[", "5xx"}},
	}
	for _, tc := range cases {
		got := statusLine(midRun, '*', tc.width)
		for _, s := range tc.kept {
			if !strings.Contains(got, s) {
				t.Errorf("width %d: %q lacks %q", tc.width, got, s)
			}
		}
		for _, s := range tc.dropped {
			if strings.Contains(got, s) {
				t.Errorf("width %d: %q still has %q", tc.width, got, s)
			}
		}
	}
}

func TestFramesStayOnOneLine(t *testing.T) {
	d, out := live(80)
	if d.Interval() != liveInterval {
		t.Errorf("interval = %s, want %s", d.Interval(), liveInterval)
	}
	d.Update(midRun)
	d.Update(midRun)
	if got := out.String(); strings.Contains(got, "\n") || !strings.HasPrefix(got, "\r") || !strings.HasSuffix(got, "\r") {
		t.Errorf("frames = %q", got)
	}
}

func TestSpinnerMoves(t *testing.T) {
	d, out := live(80)
	d.Update(midRun)
	first := out.String()
	out.Reset()
	d.Update(midRun)
	if out.String() == first {
		t.Errorf("two frames are identical: %q", first)
	}
}

func TestShorterFrameCoversTheLongerOne(t *testing.T) {
	d, out := live(80)
	d.Update(midRun)
	long := d.drawn

	out.Reset()
	d.width = func() int { return 20 }
	d.Update(midRun)
	frame := strings.Trim(out.String(), "\r")
	if got := utf8.RuneCountInString(frame); got != long {
		t.Errorf("frame covers %d columns, want %d: %q", got, long, frame)
	}
}

func TestFinishErasesTheBar(t *testing.T) {
	d, out := live(80)
	d.Update(midRun)
	columns := d.drawn

	out.Reset()
	d.Finish()
	if got, want := out.String(), "\r"+strings.Repeat(" ", columns)+"\r"; got != want {
		t.Errorf("finish = %q, want %q", got, want)
	}
}

func TestFinishWithoutFramesWritesNothing(t *testing.T) {
	d, out := live(80)
	d.Finish()
	if out.Len() != 0 {
		t.Errorf("finish wrote %q", out.String())
	}
}
