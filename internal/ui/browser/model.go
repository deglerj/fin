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
	"github.com/charmbracelet/x/ansi"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/ui/keys"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/deglerj/fin/internal/ui/styles"
)

type level struct {
	name         string
	parentID     string
	items        []api.Item
	cursor       int
	offset       int
	hasBack      bool
	revalidating bool
}

func (l level) displayLen() int {
	if l.hasBack {
		return len(l.items) + 1
	}
	return len(l.items)
}

// itemAt maps a display index to a real item. Returns (item, true) for real
// items and (zero, false) for the virtual "..." back entry.
func (l level) itemAt(displayIdx int) (api.Item, bool) {
	if l.hasBack {
		if displayIdx == 0 {
			return api.Item{}, false
		}
		displayIdx--
	}
	if displayIdx < 0 || displayIdx >= len(l.items) {
		return api.Item{}, false
	}
	return l.items[displayIdx], true
}

type Model struct {
	client      *api.Client
	cache       map[string][]api.Item
	stack       []level
	width       int
	height      int
	loading     bool
	spinner     spinner.Model
	cancelFetch context.CancelFunc
}

func New(client *api.Client, width, height int) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return Model{client: client, cache: make(map[string][]api.Item), width: width, height: height, spinner: sp}
}

// maxCachedLevels bounds the per-level item cache, which otherwise grows for
// every folder visited in a session.
// ponytail: clear-all rather than LRU — the stack keeps its own copy of the
// levels on screen, so a flush costs at most one refetch.
const maxCachedLevels = 64

func (m *Model) cacheLevel(parentID string, items []api.Item) {
	if parentID == "" {
		return
	}
	if len(m.cache) >= maxCachedLevels {
		clear(m.cache)
	}
	m.cache[parentID] = items
}

// abandonFetch cancels the in-flight child fetch, if any. Called whenever the
// level it was loading stops being the one the user is looking at; cancelling
// an already-finished request is a no-op.
func (m *Model) abandonFetch() {
	if m.cancelFetch != nil {
		m.cancelFetch()
		m.cancelFetch = nil
	}
}

// pop drops the top level and abandons any fetch that was filling it.
func (m *Model) pop() {
	if len(m.stack) > 1 {
		m.stack = m.stack[:len(m.stack)-1]
		m.abandonFetch()
	}
}

// clampCursor keeps cursor and offset inside a level whose item list changed
// size under it, so a shrinking refresh cannot strand the cursor past the end.
func clampCursor(l level, visible int) level {
	maxIdx := l.displayLen() - 1
	if maxIdx < 0 {
		l.cursor, l.offset = 0, 0
		return l
	}
	if l.cursor > maxIdx {
		l.cursor = maxIdx
	}
	if visible > 0 && l.cursor >= l.offset+visible {
		l.offset = l.cursor - visible + 1
	}
	if l.offset > l.cursor {
		l.offset = l.cursor
	}
	if l.offset < 0 {
		l.offset = 0
	}
	return l
}

func (m Model) SelectedItem() api.Item {
	if len(m.stack) == 0 {
		return api.Item{}
	}
	top := m.stack[len(m.stack)-1]
	item, ok := top.itemAt(top.cursor)
	if !ok {
		return api.Item{}
	}
	return item
}

func (m Model) visibleHeight() int {
	return m.height - 4 // breadcrumb + border + 2 status lines
}

func (m Model) WithSize(w, h int) Model { m.width = w; m.height = h; return m }
func (m Model) IsLoading() bool         { return m.loading || len(m.stack) == 0 }

func (m Model) CurrentLevelParentID() string {
	if len(m.stack) == 0 {
		return ""
	}
	return m.stack[len(m.stack)-1].parentID
}

func (m Model) Init() tea.Cmd { return m.spinner.Tick }

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case msg.PushLevel:
		m.loading = false
		m.abandonFetch()
		hasBack := len(m.stack) > 0
		cursor := 0
		if hasBack {
			cursor = 1
		}
		m.cacheLevel(message.ParentID, message.Items)
		m.stack = append(m.stack, level{name: message.LevelName, parentID: message.ParentID, items: message.Items, hasBack: hasBack, cursor: cursor})
		return m, nil

	case msg.RefreshLevel:
		m.cacheLevel(message.ParentID, message.Items)
		for i, l := range m.stack {
			if l.parentID == message.ParentID {
				m.stack[i].items = message.Items
				m.stack[i].revalidating = false
				m.stack[i] = clampCursor(m.stack[i], m.visibleHeight())
			}
		}
		return m, nil

	case msg.AppError:
		m.loading = false
		if len(m.stack) > 0 {
			m.stack[len(m.stack)-1].revalidating = false
		}
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
			} else {
				top.cursor = top.displayLen() - 1
				top.offset = top.cursor - m.visibleHeight() + 1
				if top.offset < 0 {
					top.offset = 0
				}
			}
			m.stack[len(m.stack)-1] = top
		case key.Matches(message, keys.Default.Down):
			if top.cursor < top.displayLen()-1 {
				top.cursor++
				if top.cursor >= top.offset+m.visibleHeight() {
					top.offset++
				}
			} else {
				top.cursor = 0
				top.offset = 0
			}
			m.stack[len(m.stack)-1] = top
		case key.Matches(message, keys.Default.PageDown):
			if top.displayLen() == 0 {
				break
			}
			newCursor := top.cursor + m.visibleHeight()
			if newCursor >= top.displayLen() {
				newCursor = top.displayLen() - 1
			}
			top.cursor = newCursor
			if top.cursor >= top.offset+m.visibleHeight() {
				top.offset = top.cursor - m.visibleHeight() + 1
				if top.offset < 0 {
					top.offset = 0
				}
			}
			m.stack[len(m.stack)-1] = top
		case key.Matches(message, keys.Default.PageUp):
			if top.displayLen() == 0 {
				break
			}
			newCursor := top.cursor - m.visibleHeight()
			if newCursor < 0 {
				newCursor = 0
			}
			top.cursor = newCursor
			if top.cursor < top.offset {
				top.offset = top.cursor
			}
			m.stack[len(m.stack)-1] = top
		case key.Matches(message, keys.Default.Open):
			item, isReal := top.itemAt(top.cursor)
			if !isReal {
				m.pop()
				return m, nil
			}
			if isLeaf(item) {
				return m, func() tea.Msg { return msg.PlayItem{Item: item} }
			}
			if item.Type == "VirtualSection" {
				m.loading = true
				return m, func() tea.Msg { return msg.FetchVirtualSection{ID: item.Id} }
			}
			m.abandonFetch()
			ctx, cancel := context.WithCancel(context.Background())
			m.cancelFetch = cancel
			if cached, ok := m.cache[item.Id]; ok {
				hasBack := len(m.stack) > 0
				cursor := 0
				if hasBack {
					cursor = 1
				}
				m.stack = append(m.stack, level{
					name:         item.Name,
					parentID:     item.Id,
					items:        cached,
					hasBack:      hasBack,
					cursor:       cursor,
					revalidating: true,
				})
				return m, m.fetchChildren(ctx, item, true)
			}
			m.loading = true
			return m, m.fetchChildren(ctx, item, false)
		case key.Matches(message, keys.Default.Back):
			m.pop()
		case key.Matches(message, keys.Default.Search):
			return m, func() tea.Msg { return msg.OpenSearch{} }
		case key.Matches(message, keys.Default.Random):
			if len(m.stack) <= 1 {
				return m, nil
			}
			items := top.items
			if len(items) == 0 {
				return m, nil
			}
			var leaves []api.Item
			for _, it := range items {
				if isLeaf(it) {
					leaves = append(leaves, it)
				}
			}
			if len(leaves) > 0 {
				item := leaves[rand.Intn(len(leaves))]
				return m, func() tea.Msg { return msg.PlayItem{Item: item} }
			}
			if m.client == nil {
				return m, func() tea.Msg { return msg.AppError{Err: fmt.Errorf("no client configured")} }
			}
			client := m.client
			parentID := top.parentID
			if strings.HasPrefix(parentID, "__") {
				parentID = items[rand.Intn(len(items))].Id
			}
			return m, func() tea.Msg {
				item, err := client.GetRandomLeaf(context.Background(), parentID)
				if err != nil {
					return msg.AppError{Err: err}
				}
				return msg.PlayItem{Item: item}
			}
		case key.Matches(message, keys.Default.MarkPlayed):
			item, isReal := top.itemAt(top.cursor)
			if !isReal || !isLeaf(item) {
				return m, nil
			}
			client := m.client
			if client == nil {
				return m, func() tea.Msg { return msg.AppError{Err: fmt.Errorf("no client configured")} }
			}
			targetPlayed := !item.UserData.Played
			itemID := item.Id
			return m, func() tea.Msg {
				var err error
				if targetPlayed {
					err = client.MarkPlayed(context.Background(), itemID)
				} else {
					err = client.MarkUnplayed(context.Background(), itemID)
				}
				if err != nil {
					return msg.AppError{Err: err}
				}
				return msg.PlayedToggled{ItemID: itemID, Played: targetPlayed}
			}
		}

	case msg.PlayedToggled:
		if len(m.stack) == 0 {
			return m, nil
		}
		top := m.stack[len(m.stack)-1]
		for i, item := range top.items {
			if item.Id == message.ItemID {
				m.stack[len(m.stack)-1].items[i].UserData.Played = message.Played
				break
			}
		}
		if top.parentID != "" {
			if cached, ok := m.cache[top.parentID]; ok {
				for i, item := range cached {
					if item.Id == message.ItemID {
						m.cache[top.parentID][i].UserData.Played = message.Played
						break
					}
				}
			}
		}
		return m, nil
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

func (m Model) fetchChildren(ctx context.Context, item api.Item, isRefresh bool) tea.Cmd {
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
		items, err := client.GetItems(ctx, item.Id, itemTypes)
		if err != nil {
			return msg.AppError{Err: err}
		}
		if isRefresh {
			return msg.RefreshLevel{ParentID: item.Id, Items: items}
		}
		return msg.PushLevel{ParentID: item.Id, Items: items, LevelName: item.Name}
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
	if end > top.displayLen() {
		end = top.displayLen()
	}
	for i := top.offset; i < end; i++ {
		var line string
		item, isReal := top.itemAt(i)
		if !isReal {
			line = "..."
		} else {
			line = formatItem(item)
		}
		if m.width > 0 {
			line = ansi.Truncate(line, m.width, "…")
		}
		if i == top.cursor {
			sb.WriteString(styles.Selected.Width(m.width).Render(line))
		} else {
			sb.WriteString(styles.Dim.Render(line))
		}
		sb.WriteByte('\n')
	}

	count := fmt.Sprintf("%d items", len(top.items))
	revalidating := ""
	if top.revalidating {
		revalidating = " [~]"
	}
	line1 := count + revalidating + "  ↑↓ navigate · enter open · ⌫/esc/← back"
	line2 := "/ search · m mark · q quit"
	if len(m.stack) > 1 {
		line2 = "/ search · r random · m mark · q quit"
	}
	sb.WriteString(styles.StatusBar.Width(m.width).Render(line1 + "\n" + line2))
	return sb.String()
}

// pad right-pads s to width terminal columns. Measuring display width rather
// than bytes or runes keeps accented and double-width names aligned.
func pad(s string, width int) string {
	if n := width - ansi.StringWidth(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

func formatItem(item api.Item) string {
	name := item.Name
	if item.Type == "Episode" && item.IndexNumber > 0 {
		name = fmt.Sprintf("S%02dE%02d – %s", item.ParentIndexNumber, item.IndexNumber, item.Name)
	}
	if item.Type == "Episode" && item.SeriesName != "" {
		name = item.SeriesName + ": " + name
	}
	if item.UserData.Played {
		name = "✓ " + name
	}
	if item.RunTimeTicks > 0 {
		mins := item.RunTimeTicks / 600_000_000
		return fmt.Sprintf("%s %dm", pad(name, 50), mins)
	}
	return name
}
