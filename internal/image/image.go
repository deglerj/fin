package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	imglib "image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	termimg "github.com/blacktop/go-termimg"
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

// Encode encodes image bytes as a kitty graphics protocol string.
// Omits c/r cell hints so the terminal uses its actual font size, ensuring correct aspect ratio.
func Encode(data []byte) string {
	src, _, err := imglib.Decode(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	bounds := src.Bounds()
	rgba := imglib.NewRGBA(bounds)
	draw.Draw(rgba, bounds, src, bounds.Min, draw.Src)

	pixelW, pixelH := bounds.Dx(), bounds.Dy()
	encoded := base64.StdEncoding.EncodeToString(rgba.Pix)

	const chunkSize = 4096
	var out strings.Builder
	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]
		m := 1
		if end == len(encoded) {
			m = 0
		}
		if i == 0 {
			fmt.Fprintf(&out, "\x1b_Ga=T,f=32,s=%d,v=%d,q=2,m=%d;%s\x1b\\", pixelW, pixelH, m, chunk)
		} else {
			fmt.Fprintf(&out, "\x1b_Gm=%d,q=2;%s\x1b\\", m, chunk)
		}
	}
	return out.String()
}
