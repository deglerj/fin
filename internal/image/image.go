package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	imglib "image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	termimg "github.com/blacktop/go-termimg"
	xdraw "golang.org/x/image/draw"
)

// Fallback cell size for terminals that report no pixel dimensions.
const (
	defaultCellW = 9
	defaultCellH = 20
)

// Probe detects whether the terminal supports kitty graphics protocol.
// Result is cached after first call by go-termimg.
func Probe() bool {
	return termimg.KittySupported()
}

// Delete returns the kitty graphics protocol escape to delete all visible image placements.
func Delete() string {
	return termimg.ClearAllString()
}

// Fit reports how many terminal cells an image should occupy, preserving its
// aspect ratio and never exceeding maxCols/maxRows. Returns 0, 0 for images
// that cannot be decoded, so callers reserve no space for them.
func Fit(data []byte, maxCols, maxRows int) (cols, rows int) {
	cfg, _, err := imglib.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0
	}
	cellW, cellH := CellSize()
	return fitCells(cfg.Width, cfg.Height, cellW, cellH, maxCols, maxRows)
}

func fitCells(imgW, imgH, cellW, cellH, maxCols, maxRows int) (cols, rows int) {
	if imgW <= 0 || imgH <= 0 || cellW <= 0 || cellH <= 0 || maxCols <= 0 || maxRows <= 0 {
		return 0, 0
	}
	// Shrink to whichever axis binds first; never enlarge past the source.
	scale := min(float64(maxCols*cellW)/float64(imgW), float64(maxRows*cellH)/float64(imgH), 1)
	cols = ceilDiv(int(float64(imgW)*scale), cellW)
	rows = ceilDiv(int(float64(imgH)*scale), cellH)
	return max(cols, 1), max(rows, 1)
}

func ceilDiv(a, b int) int { return (a + b - 1) / b }

// Encode encodes image bytes as a kitty graphics protocol string occupying
// exactly cols×rows cells. The image is scaled to fit that box and padded with
// transparent pixels to fill it, so the terminal's own scaling into c/r cannot
// distort it — and the placement can never spill past the rows the caller
// reserved, whatever the real cell size turns out to be.
func Encode(data []byte, cols, rows int) string {
	if cols <= 0 || rows <= 0 {
		return ""
	}
	src, _, err := imglib.Decode(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	cellW, cellH := CellSize()
	boxW, boxH := cols*cellW, rows*cellH

	b := src.Bounds()
	scale := min(float64(boxW)/float64(b.Dx()), float64(boxH)/float64(b.Dy()))
	dst := imglib.NewRGBA(imglib.Rect(0, 0, boxW, boxH))
	xdraw.CatmullRom.Scale(dst, imglib.Rect(0, 0, int(float64(b.Dx())*scale), int(float64(b.Dy())*scale)), src, b, xdraw.Src, nil)

	encoded := base64.StdEncoding.EncodeToString(dst.Pix)

	const chunkSize = 4096
	var out strings.Builder
	for i := 0; i < len(encoded); i += chunkSize {
		end := min(i+chunkSize, len(encoded))
		chunk := encoded[i:end]
		m := 1
		if end == len(encoded) {
			m = 0
		}
		if i == 0 {
			fmt.Fprintf(&out, "\x1b_Ga=T,f=32,s=%d,v=%d,c=%d,r=%d,q=2,m=%d;%s\x1b\\", boxW, boxH, cols, rows, m, chunk)
		} else {
			fmt.Fprintf(&out, "\x1b_Gm=%d,q=2;%s\x1b\\", m, chunk)
		}
	}
	return out.String()
}
