//go:build !(linux || darwin || freebsd || netbsd || openbsd || dragonfly)

package progress

import "os"

// terminalWidth has no dependency-free answer on these systems. Consoles there
// are normally at least this wide.
func terminalWidth(*os.File) int { return defaultWidth }
