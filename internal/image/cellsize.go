//go:build !windows

package image

import (
	"os"

	"golang.org/x/sys/unix"
)

// CellSize returns the pixel dimensions of one terminal cell, queried via
// TIOCGWINSZ. Terminals that leave the pixel fields zero (or a non-tty stdout)
// fall back to defaultCellW/defaultCellH.
func CellSize() (w, h int) {
	ws, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 || ws.Row == 0 || ws.Xpixel == 0 || ws.Ypixel == 0 {
		return defaultCellW, defaultCellH
	}
	return int(ws.Xpixel) / int(ws.Col), int(ws.Ypixel) / int(ws.Row)
}
