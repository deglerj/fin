// internal/ui/search/model_test.go
package search_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/deglerj/fin/internal/ui/search"
	"github.com/stretchr/testify/require"
)

func TestSearchShowsResults(t *testing.T) {
	m := search.New(nil)
	results := []api.Item{{Id: "1", Name: "Dune"}, {Id: "2", Name: "Dune: Part Two"}}
	m2, _ := m.Update(msg.SearchResults{Items: results})
	view := m2.(search.Model).View()
	require.True(t, strings.Contains(view, "Dune"), "results not in view: %q", view)
}

func TestEscClosesSearch(t *testing.T) {
	m := search.New(nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd, "expected close command")
	result := cmd()
	_, ok := result.(msg.CloseOverlay)
	require.True(t, ok, "expected CloseOverlay, got %T", result)
}
