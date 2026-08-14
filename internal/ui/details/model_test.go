// internal/ui/details/model_test.go
package details_test

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"

	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/ui/details"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/stretchr/testify/require"
)

func TestDetailsView(t *testing.T) {
	m := details.New(false).WithSize(40, 20)
	m2, _ := m.Update(msg.OpenDetails{Item: api.Item{
		Id: "m1", Name: "Dune", ProductionYear: 2021,
		Overview: "A noble family becomes embroiled in a war.",
	}})
	view := m2.(details.Model).View()
	require.True(t, strings.Contains(view, "Dune"), "title not in view: %q", view)
	require.True(t, strings.Contains(view, "2021"), "year not in view: %q", view)
}

// The blank rows the view reserves must match the cells the placement claims,
// at load time and after a resize.
func TestImageRowsMatchReservedBlankLines(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 600)) // portrait poster
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))

	m := details.New(true).WithSize(40, 40)
	m2, _ := m.Update(msg.OpenDetails{Item: api.Item{Id: "m1", Name: "Dune"}})
	m3, _ := m2.(details.Model).Update(msg.ImageLoaded{Data: buf.Bytes(), ItemId: "m1"})
	d := m3.(details.Model)

	require.True(t, d.HasImage())
	require.LessOrEqual(t, d.ImageRows(), (40-2)/2)
	require.Equal(t, d.ImageRows(), leadingBlankLines(d.View()))

	small := d.WithSize(30, 16)
	require.LessOrEqual(t, small.ImageRows(), (16-2)/2)
	require.Equal(t, small.ImageRows(), leadingBlankLines(small.View()))
}

// leadingBlankLines counts empty content lines between the overlay's top
// border and its first line of text.
func leadingBlankLines(view string) int {
	lines := strings.Split(view, "\n")
	n := 0
	for _, l := range lines[1:] {
		if strings.TrimSpace(strings.Trim(l, "│")) != "" {
			break
		}
		n++
	}
	return n
}
