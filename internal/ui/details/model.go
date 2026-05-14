// internal/ui/details/model.go
package details

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/image"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/deglerj/fin/internal/ui/styles"
)

type Model struct {
	item         api.Item
	imageCapable bool
	imageData    []byte
	client       apiClient
	width        int
	height       int
}

type apiClient interface {
	GetImage(ctx context.Context, itemID string, maxWidth int) ([]byte, error)
}

func New(imageCapable bool) Model {
	return Model{imageCapable: imageCapable}
}

func (m Model) WithClient(c apiClient) Model { m.client = c; return m }
func (m Model) WithSize(w, h int) Model      { m.width = w; m.height = h; return m }
func (m Model) HasImage() bool               { return len(m.imageData) > 0 }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case msg.OpenDetails:
		if message.Item.Id == m.item.Id {
			return m, nil
		}
		m.item = message.Item
		m.imageData = nil
		var cmd tea.Cmd
		if m.imageCapable && m.client != nil {
			itemID := message.Item.Id
			c := m.client
			cmd = func() tea.Msg {
				data, err := c.GetImage(context.Background(), itemID, 200)
				if err != nil {
					return nil
				}
				return msg.ImageLoaded{Data: data}
			}
		}
		return m, cmd
	case msg.ImageLoaded:
		m.imageData = message.Data
	}
	return m, nil
}

func (m Model) View() string {
	if m.item.Id == "" {
		return styles.Overlay.Width(m.width - 4).Render(styles.Dim.Render("Select an item to see details"))
	}

	var imgStr string
	if m.imageCapable && len(m.imageData) > 0 {
		imgStr = image.Encode(m.imageData, 20, 10) + "\n"
	}

	var sb strings.Builder
	sb.WriteString(styles.Title.Render(m.item.Name))
	if m.item.ProductionYear > 0 {
		sb.WriteString(styles.Subtitle.Render(fmt.Sprintf(" (%d)", m.item.ProductionYear)))
	}
	sb.WriteByte('\n')

	if m.item.RunTimeTicks > 0 {
		mins := m.item.RunTimeTicks / 600_000_000
		sb.WriteString(styles.Dim.Render(fmt.Sprintf("%dh%02dm", mins/60, mins%60)))
		sb.WriteByte('\n')
	}

	if m.item.CommunityRating > 0 {
		sb.WriteString(styles.Dim.Render(fmt.Sprintf("★ %.1f", m.item.CommunityRating)))
		sb.WriteByte('\n')
	}

	if m.item.Overview != "" {
		sb.WriteByte('\n')
		sb.WriteString(wordWrap(m.item.Overview, m.width-4))
		sb.WriteByte('\n')
	}

	var directors, cast []string
	for _, p := range m.item.People {
		switch p.Type {
		case "Director":
			directors = append(directors, p.Name)
		case "Actor":
			if len(cast) < 5 {
				cast = append(cast, p.Name)
			}
		}
	}
	if len(directors) > 0 {
		sb.WriteString(styles.Label.Render("Director: ") + strings.Join(directors, ", ") + "\n")
	}
	if len(cast) > 0 {
		sb.WriteString(styles.Label.Render("Cast: ") + strings.Join(cast, ", ") + "\n")
	}

	return imgStr + styles.Overlay.Width(m.width-4).Render(sb.String())
}

func wordWrap(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	var result strings.Builder
	words := strings.Fields(s)
	line := 0
	for i, w := range words {
		if line+len(w)+1 > width && line > 0 {
			result.WriteByte('\n')
			line = 0
		}
		if i > 0 && line > 0 {
			result.WriteByte(' ')
			line++
		}
		result.WriteString(w)
		line += len(w)
	}
	return result.String()
}
