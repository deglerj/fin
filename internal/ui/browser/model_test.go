// internal/ui/browser/model_test.go
package browser_test

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	require.Equal(t, 1, bm.Depth())
	require.Equal(t, "A", bm.SelectedItem().Name)
}

func TestNavigateDown(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeItems("A", "B", "C"), LevelName: "Movies"})
	m3, _ := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyDown})
	bm := m3.(browser.Model)
	require.Equal(t, "B", bm.SelectedItem().Name)
}

func TestPopLevel(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeItems("A"), LevelName: "Level1"})
	m3, _ := m2.(browser.Model).Update(msg.PushLevel{Items: makeItems("X"), LevelName: "Level2"})
	m4, _ := m3.(browser.Model).Update(msg.PopLevel{})
	bm := m4.(browser.Model)
	require.Equal(t, 1, bm.Depth())
}

func TestRandomSelection(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m2, _ := m.Update(msg.PushLevel{Items: makeItems("A", "B", "C", "D", "E"), LevelName: "Movies"})
	_, cmd := m2.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	require.NotNil(t, cmd, "expected a command from random key")
}
