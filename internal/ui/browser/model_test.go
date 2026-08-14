// internal/ui/browser/model_test.go
package browser_test

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/ui/browser"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/stretchr/testify/require"
)

func makeItems(names ...string) []api.Item {
	out := make([]api.Item, len(names))
	for i, n := range names {
		out[i] = api.Item{Id: fmt.Sprintf("id%d", i), Name: n, Type: "Movie"}
	}
	return out
}

func TestPushLevel(t *testing.T) {
	m := browser.New(nil, 80, 24)
	updated, _ := m.Update(msg.PushLevel{Items: makeItems("A", "B", "C"), LevelName: "Movies"})
	bm := updated.(browser.Model)
	require.Equal(t, "A", bm.SelectedItem().Name)
	require.Contains(t, bm.View(), "Movies")
}

func TestNavigateDown(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeItems("A", "B", "C"), LevelName: "Movies"})
	m3, _ := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	bm := m3.(browser.Model)
	require.Equal(t, "B", bm.SelectedItem().Name)
}

func TestBackKeyPopsLevel(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeItems("A"), LevelName: "Level1"})
	m3, _ := m2.(browser.Model).Update(msg.PushLevel{Items: makeItems("X"), LevelName: "Level2"})
	require.Equal(t, "X", m3.(browser.Model).SelectedItem().Name)

	m4, _ := m3.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	bm := m4.(browser.Model)
	require.Equal(t, "A", bm.SelectedItem().Name)
	require.NotContains(t, bm.View(), "Level2", "breadcrumb should have dropped the popped level")
}

func TestBackKeyStopsAtTheRootLevel(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeItems("A"), LevelName: "Level1"})
	m3, _ := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.Equal(t, "A", m3.(browser.Model).SelectedItem().Name)
}

func TestRandomSelectionOnLeaves(t *testing.T) {
	m := browser.New(nil, 80, 24)
	// depth 1: library level
	m2, _ := m.Update(msg.PushLevel{LevelName: "Libraries", Items: []api.Item{{Id: "lib1", Name: "Movies"}}})
	// depth 2: movie leaf items
	m3, _ := m2.(browser.Model).Update(msg.PushLevel{ParentID: "lib1", LevelName: "Movies", Items: makeItems("A", "B", "C", "D", "E")})
	_, cmd := m3.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	require.NotNil(t, cmd, "expected a command from random key on leaf items")
}

func TestRandomIgnoredAtDepth1(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeItems("A", "B"), LevelName: "Libraries"})
	_, cmd := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	require.Nil(t, cmd, "r at depth 1 should be a noop")
}

func TestRandomOnNonLeavesNilClientReturnsError(t *testing.T) {
	m := browser.New(nil, 80, 24)
	// depth 1: library level
	m2, _ := m.Update(msg.PushLevel{LevelName: "Libraries", Items: []api.Item{{Id: "lib1", Name: "TV Shows"}}})
	// depth 2: series items (non-leaves)
	series := []api.Item{{Id: "s1", Name: "Breaking Bad", Type: "Series"}}
	m3, _ := m2.(browser.Model).Update(msg.PushLevel{ParentID: "lib1", LevelName: "TV Shows", Items: series})
	_, cmd := m3.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	require.NotNil(t, cmd, "r on non-leaf level should return a fetch cmd")
	result := cmd()
	_, isErr := result.(msg.AppError)
	require.True(t, isErr, "nil client should produce AppError, got %T", result)
}

func TestRandomStatusBarHiddenAtDepth1(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeItems("A"), LevelName: "Libraries"})
	view := m2.(browser.Model).View()
	require.NotContains(t, view, "r random", "status bar should hide r random hint at depth 1")
}

func TestRandomOnVirtualSectionNonLeavesNilClientReturnsError(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{LevelName: "Libraries", Items: []api.Item{{Id: "lib1", Name: "Home"}}})
	// virtual section with non-leaf items (e.g. favorited Series)
	series := []api.Item{{Id: "s1", Name: "Breaking Bad", Type: "Series"}}
	m3, _ := m2.(browser.Model).Update(msg.PushLevel{ParentID: "__favorites__", LevelName: "Favorites", Items: series})
	_, cmd := m3.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	require.NotNil(t, cmd, "r on virtual section with non-leaves should return a cmd")
	result := cmd()
	_, isErr := result.(msg.AppError)
	require.True(t, isErr, "nil client should produce AppError, got %T", result)
}

func TestRandomStatusBarShownAtDepth2(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{LevelName: "Libraries", Items: []api.Item{{Id: "lib1", Name: "Movies"}}})
	m3, _ := m2.(browser.Model).Update(msg.PushLevel{ParentID: "lib1", LevelName: "Movies", Items: makeItems("A")})
	view := m3.(browser.Model).View()
	require.Contains(t, view, "r random", "status bar should show r random hint at depth 2+")
}

func TestLoadingSpinnerActivatesOnDrillIn(t *testing.T) {
	m := browser.New(nil, 80, 24)
	series := []api.Item{{Id: "s1", Name: "Breaking Bad", Type: "Series"}}
	m2, _ := m.Update(msg.PushLevel{Items: series, LevelName: "Shows"})
	m3, _ := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Contains(t, m3.(browser.Model).View(), "Loading...")
}

func TestLoadingResetOnAppError(t *testing.T) {
	m := browser.New(nil, 80, 24)
	series := []api.Item{{Id: "s1", Name: "Breaking Bad", Type: "Series"}}
	m2, _ := m.Update(msg.PushLevel{Items: series, LevelName: "Shows"})
	m3, _ := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	// loading is now true
	m4, _ := m3.(browser.Model).Update(msg.AppError{Err: fmt.Errorf("network error")})
	require.NotContains(t, m4.(browser.Model).View(), "Loading...")
}

func TestNilClientDrillIn(t *testing.T) {
	m := browser.New(nil, 80, 24)
	series := []api.Item{{Id: "s1", Name: "Breaking Bad", Type: "Series"}}
	m2, _ := m.Update(msg.PushLevel{Items: series, LevelName: "Shows"})
	// pressing enter on a Series with nil client should return an error cmd, not panic
	_, cmd := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "expected error cmd from nil client drill-in")
	result := cmd()
	_, isErr := result.(msg.AppError)
	require.True(t, isErr, "expected AppError from nil client drill-in, got %T", result)
}

func makeSWRModel(t *testing.T) (browser.Model, []api.Item) {
	t.Helper()
	m := browser.New(nil, 80, 24)
	series := api.Item{Id: "s1", Name: "Breaking Bad", Type: "Series"}
	m2, _ := m.Update(msg.PushLevel{LevelName: "Libraries", Items: []api.Item{series}})
	seasons := makeItems("Season 1", "Season 2")
	// PushLevel with ParentID populates the cache
	m3, _ := m2.(browser.Model).Update(msg.PushLevel{ParentID: "s1", LevelName: "Seasons", Items: seasons})
	m4, _ := m3.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyEsc})
	return m4.(browser.Model), seasons
}

func TestSWRCacheHitSkipsLoadingSpinner(t *testing.T) {
	m, seasons := makeSWRModel(t)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	bm := m2.(browser.Model)
	require.False(t, bm.IsLoading(), "cache hit should skip loading spinner")
	require.Equal(t, seasons[0].Name, bm.SelectedItem().Name)
}

func TestSWRCacheHitTriggersBackgroundRefresh(t *testing.T) {
	m, _ := makeSWRModel(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "cache hit should return background refresh cmd")
}

func TestRefreshLevelUpdatesCurrentLevel(t *testing.T) {
	m, _ := makeSWRModel(t)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	freshSeasons := []api.Item{
		{Id: "new1", Name: "Season 1 Updated", Type: "Season"},
		{Id: "new2", Name: "Season 2 New", Type: "Season"},
	}
	m3, _ := m2.(browser.Model).Update(msg.RefreshLevel{ParentID: "s1", Items: freshSeasons})
	bm := m3.(browser.Model)
	require.Equal(t, "Season 1 Updated", bm.SelectedItem().Name)
}

func TestSWRRevalidatingIndicatorShown(t *testing.T) {
	m, _ := makeSWRModel(t)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Contains(t, m2.(browser.Model).View(), "[~]", "revalidating indicator should be shown")
}

func TestSWRRevalidatingIndicatorClearedAfterRefresh(t *testing.T) {
	m, _ := makeSWRModel(t)
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3, _ := m2.(browser.Model).Update(msg.RefreshLevel{ParentID: "s1", Items: makeItems("Season 1 Updated")})
	require.NotContains(t, m3.(browser.Model).View(), "[~]", "revalidating indicator should clear after refresh")
}

func makeLeafItems() []api.Item {
	return []api.Item{
		{Id: "m1", Name: "Dune", Type: "Movie", UserData: api.UserData{Played: false}},
		{Id: "m2", Name: "Arrival", Type: "Movie", UserData: api.UserData{Played: true}},
	}
}

func TestMarkPlayedEmitsCmd(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeLeafItems(), LevelName: "Movies"})
	_, cmd := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	require.NotNil(t, cmd, "expected a cmd from m key on leaf item")
}

func TestMarkPlayedNilClientReturnsAppError(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeLeafItems(), LevelName: "Movies"})
	_, cmd := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	require.NotNil(t, cmd)
	result := cmd()
	_, isErr := result.(msg.AppError)
	require.True(t, isErr, "nil client should produce AppError, got %T", result)
}

func TestMarkPlayedNonLeafIsNoop(t *testing.T) {
	m := browser.New(nil, 80, 24)
	series := []api.Item{{Id: "s1", Name: "Breaking Bad", Type: "Series"}}
	m2, _ := m.Update(msg.PushLevel{Items: series, LevelName: "Shows"})
	_, cmd := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	require.Nil(t, cmd, "m on non-leaf should produce no cmd")
}

func TestPlayedToggledUpdatesStack(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeLeafItems(), LevelName: "Movies"})
	// Mark m1 as played
	m3, _ := m2.(browser.Model).Update(msg.PlayedToggled{ItemID: "m1", Played: true})
	require.Contains(t, m3.(browser.Model).View(), "✓")
}

func TestPlayedToggledUnmarkRemovesIndicator(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeLeafItems(), LevelName: "Movies"})
	// Navigate down to select Arrival (index 1)
	m3, _ := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	// Confirm ✓ is shown initially (Arrival starts as Played: true)
	require.Contains(t, m3.(browser.Model).View(), "✓")
	// Unmark
	m4, _ := m3.(browser.Model).Update(msg.PlayedToggled{ItemID: "m2", Played: false})
	require.NotContains(t, m4.(browser.Model).View(), "✓")
}

func TestPlayedToggledOnUnknownItemIsNoop(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeLeafItems(), LevelName: "Movies"})
	// Toggling an ID that doesn't exist in the stack must not panic
	m3, cmd := m2.(browser.Model).Update(msg.PlayedToggled{ItemID: "nonexistent", Played: true})
	require.Nil(t, cmd)
	_ = m3.(browser.Model).View() // must not panic
}

func TestVirtualSectionEnterEmitsFetchMsg(t *testing.T) {
	m := browser.New(nil, 80, 24)
	items := []api.Item{{Id: "__next_up__", Name: "Next Up", Type: "VirtualSection"}}
	m2, _ := m.Update(msg.PushLevel{Items: items, LevelName: "Libraries"})
	m3, cmd := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "expected a cmd from entering VirtualSection")
	result := cmd()
	fetch, ok := result.(msg.FetchVirtualSection)
	require.True(t, ok, "expected FetchVirtualSection, got %T", result)
	require.Equal(t, "__next_up__", fetch.ID)
	require.True(t, m3.(browser.Model).IsLoading(), "browser should be loading after entering VirtualSection")
}

func TestPlayedItemShowsCheckmark(t *testing.T) {
	m := browser.New(nil, 80, 24)
	items := []api.Item{
		{Id: "m1", Name: "Dune", Type: "Movie", UserData: api.UserData{Played: true}},
	}
	m2, _ := m.Update(msg.PushLevel{Items: items, LevelName: "Movies"})
	require.Contains(t, m2.(browser.Model).View(), "✓")
}

func TestUnplayedItemNoCheckmark(t *testing.T) {
	m := browser.New(nil, 80, 24)
	items := []api.Item{
		{Id: "m1", Name: "Dune", Type: "Movie", UserData: api.UserData{Played: false}},
	}
	m2, _ := m.Update(msg.PushLevel{Items: items, LevelName: "Movies"})
	require.NotContains(t, m2.(browser.Model).View(), "✓")
}

func TestCurrentLevelParentIDEmptyStack(t *testing.T) {
	m := browser.New(nil, 80, 24)
	require.Equal(t, "", m.CurrentLevelParentID())
}

func TestCurrentLevelParentIDAfterPush(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{ParentID: "season1", LevelName: "Season 1", Items: makeItems("Ep 1")})
	require.Equal(t, "season1", m2.(browser.Model).CurrentLevelParentID())
}

func TestCurrentLevelParentIDReturnsTopOfStack(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{ParentID: "series1", LevelName: "Series", Items: makeItems("S1")})
	m3, _ := m2.(browser.Model).Update(msg.PushLevel{ParentID: "season1", LevelName: "Season 1", Items: makeItems("Ep 1")})
	require.Equal(t, "season1", m3.(browser.Model).CurrentLevelParentID())
}

func makeNItems(n int) []api.Item {
	items := make([]api.Item, n)
	for i := range items {
		items[i] = api.Item{Id: fmt.Sprintf("id%d", i), Name: fmt.Sprintf("item%02d", i), Type: "Movie"}
	}
	return items
}

// height=24 → visibleHeight=20
func TestPageDownMovesFullPage(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeNItems(30), LevelName: "Movies"})
	m3, _ := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyPgDown})
	require.Equal(t, "item20", m3.(browser.Model).SelectedItem().Name)
}

func TestPageDownClampsAtEnd(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeNItems(30), LevelName: "Movies"})
	m3, _ := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m4, _ := m3.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyPgDown})
	require.Equal(t, "item29", m4.(browser.Model).SelectedItem().Name)
}

func TestPageUpMovesFullPage(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeNItems(30), LevelName: "Movies"})
	m3, _ := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyPgDown})
	// cursor now at 20; page up should land at 0
	m4, _ := m3.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyPgUp})
	require.Equal(t, "item00", m4.(browser.Model).SelectedItem().Name)
}

func TestPageUpClampsAtTop(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeNItems(30), LevelName: "Movies"})
	// already at top; page up must stay at item00
	m3, _ := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyPgUp})
	require.Equal(t, "item00", m3.(browser.Model).SelectedItem().Name)
}

func TestUpWrapsToLastItem(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeItems("A", "B", "C"), LevelName: "Movies"})
	// cursor starts at 0; Up should wrap to last item C
	m3, _ := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyUp})
	require.Equal(t, "C", m3.(browser.Model).SelectedItem().Name)
}

func TestDownWrapsToFirstItem(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeItems("A", "B", "C"), LevelName: "Movies"})
	// navigate to last item C
	m3, _ := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	m4, _ := m3.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	require.Equal(t, "C", m4.(browser.Model).SelectedItem().Name)
	// Down on last item should wrap to first item A
	m5, _ := m4.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	require.Equal(t, "A", m5.(browser.Model).SelectedItem().Name)
}

// height=24 → visibleHeight=20; 30 items → last item index=29, offset should be 10
func TestUpWrapAdjustsOffsetForLongList(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeNItems(30), LevelName: "Movies"})
	m3, _ := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyUp})
	require.Equal(t, "item29", m3.(browser.Model).SelectedItem().Name)
}

func TestDownWrapAdjustsOffsetForLongList(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeNItems(30), LevelName: "Movies"})
	// go to last item via PgDown twice
	m3, _ := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m4, _ := m3.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyPgDown})
	require.Equal(t, "item29", m4.(browser.Model).SelectedItem().Name)
	// Down on last item wraps to item00
	m5, _ := m4.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	require.Equal(t, "item00", m5.(browser.Model).SelectedItem().Name)
}

func TestRefreshLevelClampsCursorWhenListShrinks(t *testing.T) {
	items := make([]api.Item, 10)
	for i := range items {
		items[i] = api.Item{Id: fmt.Sprintf("id%d", i), Name: "x", Type: "Movie"}
	}
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: items, LevelName: "L", ParentID: "p"})
	bm := m2.(browser.Model)
	for i := 0; i < 8; i++ {
		next, _ := bm.Update(tea.KeyMsg{Type: tea.KeyDown})
		bm = next.(browser.Model)
	}
	require.Equal(t, "id8", bm.SelectedItem().Id)

	m3, _ := bm.Update(msg.RefreshLevel{ParentID: "p", Items: items[:3]})
	sel := m3.(browser.Model).SelectedItem()
	require.Equal(t, "id2", sel.Id, "cursor should land on the last surviving item")
}

func TestRefreshLevelHandlesEmptyResult(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: []api.Item{{Id: "a", Type: "Movie"}}, LevelName: "L", ParentID: "p"})
	m3, _ := m2.(browser.Model).Update(msg.RefreshLevel{ParentID: "p", Items: nil})
	require.Equal(t, api.Item{}, m3.(browser.Model).SelectedItem())
	require.NotPanics(t, func() { _ = m3.(browser.Model).View() })
}

func TestFormatItemAlignsRuntimeOnDisplayWidth(t *testing.T) {
	m := browser.New(nil, 200, 24)
	m2, _ := m.Update(msg.PushLevel{Items: []api.Item{
		{Id: "a", Name: "東京物語", Type: "Movie", RunTimeTicks: 60 * 600_000_000},
		{Id: "b", Name: "Aaaaaaaa", Type: "Movie", RunTimeTicks: 60 * 600_000_000},
	}, LevelName: "L"})

	var cols []int
	for _, line := range strings.Split(m2.(browser.Model).View(), "\n") {
		if i := strings.Index(line, "60m"); i >= 0 {
			cols = append(cols, ansi.StringWidth(line[:i]))
		}
	}
	require.Len(t, cols, 2)
	require.Equal(t, cols[0], cols[1], "wide characters must not shift the runtime column")
}
