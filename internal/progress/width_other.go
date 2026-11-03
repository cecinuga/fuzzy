//go:build !(linux || darwin || freebsd || netbsd || openbsd || dragonfly)

package progress

import "os"

// terminalWidth has no dependency-free answer on these systems. Consoles there
// are normally at least this wide.
func terminalWidth(*os.File) int { return defaultWidth }

// terminalHeight reports no terminal: the endpoint list needs ANSI sequences,
// which older Windows consoles only accept after enabling them.
func terminalHeight(*os.File) (rows int, ok bool) { return 0, false }
