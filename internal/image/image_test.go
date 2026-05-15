package image_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	finimage "github.com/deglerj/fin/internal/image"
	"github.com/stretchr/testify/require"
)

func TestEncodeContainsAPC(t *testing.T) {
	// Encode a minimal 1x1 white PNG so go-termimg can decode it.
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	out := finimage.Encode(buf.Bytes())
	require.True(t, strings.Contains(out, "\x1b_G"), "encoded output missing kitty APC sequence")
}
