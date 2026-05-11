package image_test

import (
	"strings"
	"testing"

	"github.com/deglerj/fin/internal/image"
	"github.com/stretchr/testify/require"
)

func TestEncodeContainsAPC(t *testing.T) {
	data := []byte("fake-jpeg-data")
	out := image.Encode(data, 20, 10)
	require.True(t, strings.Contains(out, "\x1b_G"), "encoded output missing kitty APC sequence")
}
