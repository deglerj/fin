package image_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	finimage "github.com/deglerj/fin/internal/image"
	"github.com/stretchr/testify/require"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func TestEncodeContainsAPC(t *testing.T) {
	out := finimage.Encode(pngBytes(t, 1, 1), 4, 4)
	require.True(t, strings.Contains(out, "\x1b_G"), "encoded output missing kitty APC sequence")
}

// The placement must claim exactly the cells the caller reserved, so it can
// never spill onto the text below it.
func TestEncodeClaimsRequestedCells(t *testing.T) {
	cellW, cellH := finimage.CellSize()
	data := pngBytes(t, 320, 480)

	cols, rows := finimage.Fit(data, 56, 24)
	require.LessOrEqual(t, rows, 24)

	out := finimage.Encode(data, cols, rows)
	require.Contains(t, out, fmt.Sprintf("c=%d,r=%d", cols, rows))
	// Transferred pixels fill the cell box exactly (padded), so the terminal's
	// scaling into c/r is a no-op regardless of its real cell size.
	require.Contains(t, out, fmt.Sprintf("s=%d,v=%d", cols*cellW, rows*cellH))
}

func TestEncodeRejectsUndecodableData(t *testing.T) {
	require.Empty(t, finimage.Encode([]byte("not an image"), 4, 4))

	cols, rows := finimage.Fit([]byte("not an image"), 56, 24)
	require.Zero(t, cols)
	require.Zero(t, rows)
}
