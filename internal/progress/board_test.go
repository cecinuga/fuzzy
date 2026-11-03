package progress

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"fuzzy/internal/fuzzer"
)

func liveBoard(rows int) (*Display, *bytes.Buffer) {
	d, out := live(80)
	d.height = func() int { return rows }
	return d, out
}

func colored(color, text string) string { return color + text + colorReset }

func snapshot(endpoints ...fuzzer.EndpointProgress) fuzzer.Progress {
	p := midRun
	p.Endpoints = endpoints
	return p
}

func TestEndpointColorsFollowTheirState(t *testing.T) {
	d, out := liveBoard(40)
	d.Update(snapshot(
		fuzzer.EndpointProgress{Name: "GET /a", State: fuzzer.EndpointPending},
		fuzzer.EndpointProgress{Name: "GET /b", State: fuzzer.EndpointTesting},
		fuzzer.EndpointProgress{Name: "GET /c", State: fuzzer.EndpointDone},
	))

	got := out.String()
	for _, want := range []string{
		colored(colorRed, "GET /a"),
		colored(colorYellow, "GET /b"),
		colored(colorWhite, "GET /c"),
		"300/600 req", // the status bar is still there
	} {
		if !strings.Contains(got, want) {
			t.Errorf("frame lacks %q: %q", want, got)
		}
	}
	if strings.Index(got, "GET /c") > strings.Index(got, "300/600 req") {
		t.Errorf("the list must be above the bar: %q", got)
	}
}

func TestEachFrameGoesBackOverTheList(t *testing.T) {
	d, out := liveBoard(40)
	endpoints := []fuzzer.EndpointProgress{{Name: "GET /a"}, {Name: "GET /b"}}

	d.Update(snapshot(endpoints...))
	if strings.Contains(out.String(), "A\r") {
		t.Errorf("first frame moved up: %q", out.String())
	}

	out.Reset()
	d.Update(snapshot(endpoints...))
	got := out.String()
	if !strings.Contains(got, "\x1b[2A\r") {
		t.Errorf("second frame must go back 2 lines: %q", got)
	}
	if n := strings.Count(got, "\n"); n != 2 {
		t.Errorf("frame ends %d lines, want 2: the bar stays on the last one", n)
	}
	if !strings.Contains(got, wrapOn) || strings.Index(got, wrapOn) < strings.Index(got, wrapOff) {
		t.Errorf("autowrap must be restored within the frame: %q", got)
	}
}

func TestListNeverOutgrowsTheScreen(t *testing.T) {
	const rows = 6
	var endpoints []fuzzer.EndpointProgress
	for i := range 10 {
		endpoints = append(endpoints, fuzzer.EndpointProgress{Name: fmt.Sprintf("GET /e%d", i)})
	}
	endpoints[0].State, endpoints[1].State = fuzzer.EndpointDone, fuzzer.EndpointDone
	endpoints[7].State = fuzzer.EndpointTesting // past the rows that fit

	d, out := liveBoard(rows)
	d.Update(snapshot(endpoints...))

	got := out.String()
	if n := strings.Count(got, "\n"); n != rows-1 {
		t.Errorf("list takes %d lines on a %d-row terminal, plus the bar: %q", n, rows, got)
	}
	if !strings.Contains(got, colored(colorYellow, "GET /e7")) {
		t.Errorf("the endpoint in progress must come first: %q", got)
	}
	want := "... and 6 more: " + colored(colorRed, "4 not tested") + ", " + colored(colorYellow, "0 testing") + ", " + colored(colorWhite, "2 sent")
	if !strings.Contains(got, want) {
		t.Errorf("frame lacks %q: %q", want, got)
	}
}

func TestSpecOrderIsKeptWhenEverythingFits(t *testing.T) {
	d, out := liveBoard(40)
	d.Update(snapshot(
		fuzzer.EndpointProgress{Name: "GET /a", State: fuzzer.EndpointDone},
		fuzzer.EndpointProgress{Name: "GET /b", State: fuzzer.EndpointTesting},
	))
	if got := out.String(); strings.Index(got, "GET /a") > strings.Index(got, "GET /b") {
		t.Errorf("order changed: %q", got)
	}
}

func TestEscapesFromTheSpecDontReachTheTerminal(t *testing.T) {
	d, out := liveBoard(40)
	d.Update(snapshot(fuzzer.EndpointProgress{Name: "GET /a\x1b[2Jb\u009b31m\n"}))

	got := out.String()
	if !strings.Contains(got, "GET /a?[2Jb?31m?") {
		t.Errorf("name not neutralized: %q", got)
	}
	if strings.Contains(got, "\x1b[2J") || strings.ContainsRune(got, '\u009b') {
		t.Errorf("control sequence in the frame: %q", got)
	}
}

func TestFinishKeepsTheListAndErasesTheBar(t *testing.T) {
	d, out := liveBoard(40)
	d.Update(snapshot(fuzzer.EndpointProgress{Name: "GET /a", State: fuzzer.EndpointDone}))
	columns := d.drawn

	out.Reset()
	d.Finish()
	if got, want := out.String(), "\r"+strings.Repeat(" ", columns)+"\r"; got != want {
		t.Errorf("finish = %q, want %q", got, want)
	}
}

func TestNoListWithoutHeight(t *testing.T) {
	d, out := live(80) // what Windows and NO_COLOR get: just the bar
	d.Update(snapshot(fuzzer.EndpointProgress{Name: "GET /a"}))
	if strings.Contains(out.String(), "GET /a") || strings.Contains(out.String(), "\x1b") {
		t.Errorf("frame = %q", out.String())
	}
}

func TestFinalSnapshotIsSilentInPlainOutput(t *testing.T) {
	var out bytes.Buffer
	d := New(&out)
	p := snapshot(fuzzer.EndpointProgress{Name: "GET /a"})
	p.Final = true
	d.Update(p)
	if out.Len() != 0 {
		t.Errorf("output = %q", out.String())
	}
}

func TestNullDeviceGetsNoList(t *testing.T) {
	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if d := New(f); d.height != nil {
		t.Error("/dev/null has no height, so no list")
	}
}
