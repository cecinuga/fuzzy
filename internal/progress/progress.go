// Package progress shows how far a run has got: a status bar that updates in
// place on a terminal, plain periodic lines anywhere else.
package progress

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"fuzzy/internal/fuzzer"
)

const (
	liveInterval  = 100 * time.Millisecond
	plainInterval = 5 * time.Second

	defaultWidth  = 80
	defaultHeight = 24
	barWidth      = 20
	separator     = " · "
)

var spinner = []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")

// Display draws a run's progress. It isn't safe for concurrent use: Update and
// Finish must not overlap.
type Display struct {
	w     io.Writer
	width func() int // nil when w is not a terminal
	frame int
	drawn int // columns the last frame occupies

	height     func() int // nil when the endpoint list can't be drawn on w
	boardLines int        // lines the endpoint list occupies above the status bar
}

// New returns a Display that writes to w, drawing a status bar if w is a
// terminal that can overwrite its current line.
func New(w io.Writer) *Display {
	d := &Display{w: w}
	if f, ok := w.(*os.File); ok && isTerminal(f) {
		d.width = func() int { return terminalWidth(f) }
		// The list is told apart only by its colors, so without them there's
		// just the bar. /dev/null ends up here too, having no height.
		if _, ok := terminalHeight(f); ok && os.Getenv("NO_COLOR") == "" {
			d.height = func() int {
				if rows, ok := terminalHeight(f); ok {
					return rows
				}
				return defaultHeight
			}
		}
	}
	return d
}

// isTerminal also accepts /dev/null, which is a character device too: drawing
// there is harmless.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0 && os.Getenv("TERM") != "dumb"
}

// Interval is how often Update should be called.
func (d *Display) Interval() time.Duration {
	if d.width != nil {
		return liveInterval
	}
	return plainInterval
}

// Update shows a new snapshot. The final one only matters on a terminal, where
// it leaves the endpoint list in its last state.
func (d *Display) Update(p fuzzer.Progress) {
	if d.width == nil {
		if p.Final {
			return
		}
		fmt.Fprintf(d.w, "[+] %d/%d requests\n", p.Done, p.Total)
		return
	}
	// The last column is left free: filling it makes some terminals wrap.
	line := statusLine(p, spinner[d.frame%len(spinner)], d.width()-1)
	d.frame++
	d.redraw(d.renderBoard(p.Endpoints), line)
}

// Finish removes the status bar, leaving the cursor at the start of its line so
// that what comes next is printed cleanly. The endpoint list stays on screen.
func (d *Display) Finish() {
	if d.drawn > 0 {
		d.redraw("", "")
	}
}

// redraw replaces the current line. It pads with spaces instead of using an
// ANSI erase, which needs enabling on older Windows consoles. The cursor goes
// back to the start, so a ^C echoed by the terminal is overwritten by the next
// frame. Everything goes out in one write to keep frames from tearing. above is
// what renderBoard wants drawn over the bar.
func (d *Display) redraw(above, line string) {
	columns := utf8.RuneCountInString(line)
	padding := max(d.drawn-columns, 0)
	io.WriteString(d.w, above+"\r"+line+strings.Repeat(" ", padding)+"\r")
	d.drawn = columns
}

// statusLine renders a snapshot in at most width columns, dropping the least
// useful segments first when they don't all fit.
func statusLine(p fuzzer.Progress, frame rune, width int) string {
	const (
		bar = iota
		count
		speed
		eta
		failures
		numSegments
	)
	segments := [numSegments]string{
		bar:      renderBar(p),
		count:    fmt.Sprintf("%3d%% %d/%d req", percent(p), p.Done, p.Total),
		speed:    fmt.Sprintf("%.0f req/s", rate(p)),
		eta:      "ETA " + remaining(p),
		failures: fmt.Sprintf("5xx %d%serr %d", p.ServerErrors, separator, p.TransportErrors),
	}

	line := joinSegments(frame, segments[:])
	for _, drop := range []int{speed, eta, bar, failures} {
		if utf8.RuneCountInString(line) <= width {
			return line
		}
		segments[drop] = ""
		line = joinSegments(frame, segments[:])
	}
	return truncate(line, width)
}

func joinSegments(frame rune, segments []string) string {
	var b strings.Builder
	b.WriteRune(frame)
	// The bar and the count read as one unit, so no separator between them.
	if segments[0] != "" {
		b.WriteString(" " + segments[0])
	}
	b.WriteString(" " + segments[1])
	for _, segment := range segments[2:] {
		if segment != "" {
			b.WriteString(separator + segment)
		}
	}
	return b.String()
}

func renderBar(p fuzzer.Progress) string {
	filled := barWidth * percent(p) / 100
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled) + "]"
}

func percent(p fuzzer.Progress) int {
	if p.Total <= 0 {
		return 0
	}
	return min(p.Done*100/p.Total, 100)
}

func rate(p fuzzer.Progress) float64 {
	if p.Elapsed <= 0 {
		return 0
	}
	return float64(p.Done) / p.Elapsed.Seconds()
}

// remaining extrapolates from the average speed so far, which unlike the speed
// of the last moment doesn't make the estimate jump around.
func remaining(p fuzzer.Progress) string {
	speed := rate(p)
	if speed == 0 {
		return "--"
	}
	left := time.Duration(float64(p.Total-p.Done) / speed * float64(time.Second))
	return left.Round(time.Second).String()
}

func truncate(line string, width int) string {
	runes := []rune(line)
	if len(runes) <= width {
		return line
	}
	return string(runes[:max(width, 0)])
}
