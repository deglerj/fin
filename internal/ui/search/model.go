// internal/ui/search/model.go
package search

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/deglerj/fin/internal/ui/styles"
)

type debounceMsg struct{ seq int }

type Model struct {
	input   textinput.Model
	results []api.Item
	cursor  int
	client  *api.Client
	seq     int
}

func New(client *api.Client) Model {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()
	return Model{input: ti, client: client}
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case msg.SearchResults:
		m.results = message.Items
		m.cursor = 0
		return m, nil

	case debounceMsg:
		if message.seq != m.seq {
			return m, nil
		}
		term := m.input.Value()
		if term == "" {
			m.results = nil
			return m, nil
		}
		c := m.client
		if c == nil {
			return m, nil
		}
		return m, func() tea.Msg {
			items, err := c.Search(context.Background(), term)
			if err != nil {
				return msg.AppError{Err: err}
			}
			return msg.SearchResults{Items: items}
		}

	case tea.KeyMsg:
		switch message.String() {
		case "esc":
			return m, func() tea.Msg { return msg.CloseOverlay{} }
		case "enter":
			if m.cursor < len(m.results) {
				item := m.results[m.cursor]
				return m, func() tea.Msg { return msg.NavigateToItem{Item: item} }
			}
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
		default:
			m.seq++
			seq := m.seq
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(message)
			return m, tea.Batch(cmd, tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
				return debounceMsg{seq: seq}
			}))
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(message)
	return m, cmd
}

func (m Model) View() string {
	var sb strings.Builder
	sb.WriteString(styles.Label.Render("Search: "))
	sb.WriteString(m.input.View())
	sb.WriteByte('\n')
	sb.WriteString(strings.Repeat("─", 40))
	sb.WriteByte('\n')
	for i, item := range m.results {
		line := item.Name
		if item.Type == "Episode" {
			line = item.SeriesName + " – " + item.Name
		}
		if i == m.cursor {
			sb.WriteString(styles.Selected.Render(line))
		} else {
			sb.WriteString(styles.Dim.Render(line))
		}
		sb.WriteByte('\n')
	}
	if len(m.results) == 0 && m.input.Value() != "" {
		sb.WriteString(styles.Dim.Render("No results"))
	}
	sb.WriteString(styles.Dim.Render("\nenter to open · esc to close"))
	return styles.Overlay.Render(sb.String())
}
