//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package progress

import (
	"os"
	"syscall"
	"unsafe"
)

// terminalWidth asks the terminal on every call, so that a resized window is
// picked up by the next frame.
func terminalWidth(f *os.File) int {
	var size struct{ rows, cols, xpixel, ypixel uint16 }
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCGWINSZ, uintptr(unsafe.Pointer(&size)))
	if errno != 0 || size.cols == 0 {
		return defaultWidth
	}
	return int(size.cols)
}
