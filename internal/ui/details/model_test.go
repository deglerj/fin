// internal/ui/details/model_test.go
package details_test

import (
	"bytes"
	"context"
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

func TestLongOverviewIsScrollable(t *testing.T) {
	m := details.New(false).WithSize(40, 12)
	long := strings.Repeat("Ein sehr langer Text über den Film. ", 40)
	m2, _ := m.Update(msg.OpenDetails{Item: api.Item{Id: "m1", Name: "Dune", Overview: long}})
	dm := m2.(details.Model)

	top := dm.View()
	require.Contains(t, top, "Dune")
	require.Contains(t, top, "J/K scroll", "scroll hint expected when content overflows")

	scrolled := dm.Scroll(5).View()
	require.NotEqual(t, top, scrolled, "J should move the details text")
	require.Equal(t, strings.Count(top, "\n"), strings.Count(scrolled, "\n"),
		"scrolling must not change the pane height")
}

func TestShortOverviewShowsNoScrollHint(t *testing.T) {
	m := details.New(false).WithSize(40, 20)
	m2, _ := m.Update(msg.OpenDetails{Item: api.Item{Id: "m1", Name: "Dune", Overview: "Kurz."}})
	require.NotContains(t, m2.(details.Model).View(), "J/K scroll")
}

func TestOverviewWrapsOnDisplayWidth(t *testing.T) {
	m := details.New(false).WithSize(24, 20)
	m2, _ := m.Update(msg.OpenDetails{Item: api.Item{Id: "m1", Name: "x", Overview: "üüüüüüüüüüüüüüüü"}})
	require.Contains(t, m2.(details.Model).View(), "üüüüüüüüüüüüüüüü")
}

// fakeImages counts GetImage calls and always returns the same PNG.
type fakeImages struct {
	calls int
	png   []byte
}

func (f *fakeImages) GetImage(_ context.Context, _ string, _, _ int, _ string) ([]byte, error) {
	f.calls++
	return f.png, nil
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 40, 60))))
	return buf.Bytes()
}

// Moving the cursor emits a debounce tick per item, but only the tick matching
// the item the user actually settled on may fetch.
func TestImageFetchIsDebounced(t *testing.T) {
	f := &fakeImages{png: testPNG(t)}
	m := details.New(true).WithClient(f).WithSize(40, 20)

	var ticks []msg.ImageDebounce
	for _, id := range []string{"a", "b", "c"} {
		next, cmd := m.Update(msg.OpenDetails{Item: api.Item{Id: id, Name: id}})
		m = next.(details.Model)
		require.NotNil(t, cmd, "expected a debounce tick for %q", id)
		// tea.Tick's timer starts when the Cmd is built, so invoke it exactly once.
		produced := cmd()
		tick, ok := produced.(msg.ImageDebounce)
		require.True(t, ok, "expected ImageDebounce, got %T", produced)
		ticks = append(ticks, tick)
	}

	for _, tick := range ticks[:2] {
		_, cmd := m.Update(tick)
		require.Nil(t, cmd, "a superseded selection must not fetch")
	}
	_, cmd := m.Update(ticks[2])
	require.NotNil(t, cmd)
	require.NotNil(t, cmd())
	require.Equal(t, 1, f.calls, "only the settled selection should fetch")
}

func TestImageIsServedFromCacheOnRevisit(t *testing.T) {
	f := &fakeImages{png: testPNG(t)}
	m := details.New(true).WithClient(f).WithSize(40, 20)
	item := api.Item{Id: "a", Name: "a"}

	next, cmd := m.Update(msg.OpenDetails{Item: item})
	tick, ok := cmd().(msg.ImageDebounce)
	require.True(t, ok)
	next, cmd = next.(details.Model).Update(tick)
	next, _ = next.(details.Model).Update(cmd())
	require.Equal(t, 1, f.calls)

	// Move away, then back.
	next, _ = next.(details.Model).Update(msg.OpenDetails{Item: api.Item{Id: "b", Name: "b"}})
	back, cmd := next.(details.Model).Update(msg.OpenDetails{Item: item})
	require.Nil(t, cmd, "a cached poster must not schedule a fetch")
	require.True(t, back.(details.Model).HasImage())
	require.Equal(t, 1, f.calls)
}
