//go:build windows

package image

// CellSize returns the fallback cell size — Windows terminals do not report
// pixel dimensions via the console API.
func CellSize() (w, h int) { return defaultCellW, defaultCellH }
