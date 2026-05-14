// internal/ui/browser/model.go
package browser

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/ui/keys"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/deglerj/fin/internal/ui/styles"
)

type level struct {
	name   string
	items  []api.Item
	cursor int
	offset int
}

type Model struct {
	client  *api.Client
	stack   []level
	width   int
	height  int
	loading bool
	spinner spinner.Model
}

func New(client *api.Client, width, height int) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return Model{client: client, width: width, height: height, spinner: sp}
}

func (m Model) Depth() int { return len(m.stack) }

func (m Model) SelectedItem() api.Item {
	if len(m.stack) == 0 {
		return api.Item{}
	}
	top := m.stack[len(m.stack)-1]
	if top.cursor >= len(top.items) {
		return api.Item{}
	}
	return top.items[top.cursor]
}

func (m Model) visibleHeight() int {
	return m.height - 3 // breadcrumb + status + hint rows
}

func (m Model) WithSize(w, h int) Model { m.width = w; m.height = h; return m }

func (m Model) Init() tea.Cmd { return m.spinner.Tick }

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case msg.PushLevel:
		m.loading = false
		m.stack = append(m.stack, level{name: message.LevelName, items: message.Items})
		return m, nil

	case msg.PopLevel:
		if len(m.stack) > 0 {
			m.stack = m.stack[:len(m.stack)-1]
		}
		return m, nil

	case msg.AppError:
		m.loading = false
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(message)
			return m, cmd
		}

	case tea.KeyMsg:
		if len(m.stack) == 0 {
			return m, nil
		}
		// Copy the top level, mutate it, then put it back so the value receiver works correctly.
		top := m.stack[len(m.stack)-1]
		switch {
		case key.Matches(message, keys.Default.Up):
			if top.cursor > 0 {
				top.cursor--
				if top.cursor < top.offset {
					top.offset = top.cursor
				}
			}
			m.stack[len(m.stack)-1] = top
		case key.Matches(message, keys.Default.Down):
			if top.cursor < len(top.items)-1 {
				top.cursor++
				if top.cursor >= top.offset+m.visibleHeight() {
					top.offset++
				}
			}
			m.stack[len(m.stack)-1] = top
		case message.String() == "enter" || message.String() == "right":
			item := m.SelectedItem()
			if isLeaf(item) {
				return m, func() tea.Msg { return msg.PlayItem{Item: item} }
			}
			m.loading = true
			return m, m.fetchChildren(item)
		case message.String() == "esc" || message.String() == "left":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
			}
		case message.String() == "i":
			item := m.SelectedItem()
			return m, func() tea.Msg { return msg.OpenDetails{Item: item} }
		case message.String() == "/":
			return m, func() tea.Msg { return msg.OpenSearch{} }
		case message.String() == "r":
			items := top.items
			if len(items) == 0 {
				return m, nil
			}
			item := items[rand.Intn(len(items))]
			return m, func() tea.Msg { return msg.PlayItem{Item: item} }
		}
	}
	return m, nil
}

func isLeaf(item api.Item) bool {
	switch item.Type {
	case "Movie", "Episode", "Audio":
		return true
	}
	return false
}

func (m Model) fetchChildren(item api.Item) tea.Cmd {
	client := m.client
	if client == nil {
		return func() tea.Msg { return msg.AppError{Err: fmt.Errorf("no client configured")} }
	}
	return func() tea.Msg {
		var itemTypes []string
		switch item.Type {
		case "Series":
			itemTypes = []string{"Season"}
		case "Season":
			itemTypes = []string{"Episode"}
		}
		items, err := client.GetItems(context.Background(), item.Id, itemTypes)
		if err != nil {
			return msg.AppError{Err: err}
		}
		return msg.PushLevel{Items: items, LevelName: item.Name}
	}
}

func (m Model) breadcrumb() string {
	parts := make([]string, len(m.stack))
	for i, l := range m.stack {
		parts[i] = l.name
	}
	return styles.Breadcrumb.Render(strings.Join(parts, " › "))
}

func (m Model) View() string {
	if m.loading {
		return m.spinner.View() + " Loading..."
	}
	if len(m.stack) == 0 {
		return styles.Dim.Render("No content. Loading libraries...")
	}
	top := m.stack[len(m.stack)-1]
	var sb strings.Builder
	sb.WriteString(m.breadcrumb())
	sb.WriteByte('\n')

	visible := m.visibleHeight()
	end := top.offset + visible
	if end > len(top.items) {
		end = len(top.items)
	}
	for i := top.offset; i < end; i++ {
		item := top.items[i]
		line := formatItem(item)
		if i == top.cursor {
			sb.WriteString(styles.Selected.Width(m.width).Render(line))
		} else {
			sb.WriteString(styles.Dim.Render(line))
		}
		sb.WriteByte('\n')
	}

	count := fmt.Sprintf("%d items", len(top.items))
	hint := "↑↓ navigate · enter open/play · i details · / search · r random · q quit"
	sb.WriteString(styles.StatusBar.Render(count + "  " + hint))
	return sb.String()
}

func formatItem(item api.Item) string {
	name := item.Name
	if item.Type == "Episode" && item.IndexNumber > 0 {
		name = fmt.Sprintf("S%02dE%02d – %s", item.ParentIndexNumber, item.IndexNumber, item.Name)
	}
	if item.RunTimeTicks > 0 {
		mins := item.RunTimeTicks / 600_000_000
		return fmt.Sprintf("%-50s %dm", name, mins)
	}
	return name
}
