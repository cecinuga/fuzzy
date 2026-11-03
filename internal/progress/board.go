package progress

import (
	"fmt"
	"strings"
	"unicode"

	"fuzzy/internal/fuzzer"
)

const (
	colorRed    = "\x1b[31m"
	colorYellow = "\x1b[33m"
	colorWhite  = "\x1b[37m"
	colorReset  = "\x1b[0m"

	clearLine = "\x1b[K"
	clearDown = "\x1b[J"
	// With autowrap off a long line is clipped by the terminal instead of taking
	// two rows, which would make the next frame start from the wrong one.
	wrapOff = "\x1b[?7l"
	wrapOn  = "\x1b[?7h"
)

// renderBoard returns what draws the endpoint list over the previous one, with
// the cursor left where the status bar goes. Autowrap is restored in the same
// frame, so that a kill between two frames can't leave it off.
func (d *Display) renderBoard(endpoints []fuzzer.EndpointProgress) string {
	if d.height == nil || len(endpoints) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(wrapOff)
	if d.boardLines > 0 {
		fmt.Fprintf(&b, "\x1b[%dA", d.boardLines)
	}
	b.WriteByte('\r')
	// A frame taller than the screen can't be redrawn in place. The bar takes a row.
	d.boardLines = writeEndpoints(&b, endpoints, max(d.height()-1, 1))
	b.WriteString(clearDown + wrapOn)
	return b.String()
}

// writeEndpoints writes at most slots lines and returns how many. When all the
// endpoints fit they keep the order of the spec; otherwise the ones still moving
// come first and the last line counts those left out.
func writeEndpoints(b *strings.Builder, endpoints []fuzzer.EndpointProgress, slots int) int {
	if len(endpoints) <= slots {
		for _, e := range endpoints {
			writeEndpoint(b, e)
		}
		return len(endpoints)
	}

	shown := 0
	var hidden [fuzzer.EndpointDone + 1]int
	for _, state := range [...]fuzzer.EndpointState{fuzzer.EndpointTesting, fuzzer.EndpointPending, fuzzer.EndpointDone} {
		for _, e := range endpoints {
			switch {
			case e.State != state:
			case shown < slots-1:
				writeEndpoint(b, e)
				shown++
			default:
				hidden[state]++
			}
		}
	}
	fmt.Fprintf(b, "... and %d more: %s%d not tested%s, %s%d testing%s, %s%d sent%s%s\n",
		len(endpoints)-shown,
		colorRed, hidden[fuzzer.EndpointPending], colorReset,
		colorYellow, hidden[fuzzer.EndpointTesting], colorReset,
		colorWhite, hidden[fuzzer.EndpointDone], colorReset,
		clearLine)
	return shown + 1
}

func writeEndpoint(b *strings.Builder, e fuzzer.EndpointProgress) {
	fmt.Fprintf(b, "%s%s%s%s\n", stateColor(e.State), TerminalSafe(e.Name), colorReset, clearLine)
}

func stateColor(state fuzzer.EndpointState) string {
	switch state {
	case fuzzer.EndpointTesting:
		return colorYellow
	case fuzzer.EndpointDone:
		return colorWhite
	}
	return colorRed
}

// TerminalSafe neutralizes control characters: paths come from the spec, and an
// escape sequence hidden in one must not reach the terminal.
func TerminalSafe(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '?'
		}
		return r
	}, s)
}
