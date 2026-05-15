// internal/ui/details/model.go
package details

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/api"
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
	GetImage(ctx context.Context, itemID string, maxWidth, maxHeight int, tag string) ([]byte, error)
}

func New(imageCapable bool) Model {
	return Model{imageCapable: imageCapable}
}

func (m Model) WithClient(c apiClient) Model { m.client = c; return m }
func (m Model) WithSize(w, h int) Model      { m.width = w; m.height = h; return m }
func (m Model) HasImage() bool               { return len(m.imageData) > 0 }
func (m Model) ImageData() []byte            { return m.imageData }

func (m Model) ImageRows() int {
	r := (m.height - 2) / 2
	if r < 4 {
		r = 4
	}
	return r
}

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
		tag := message.Item.ImageTags["Primary"]
		// ImageTags == nil means no tag data (e.g. library folders constructed without it) — attempt fetch anyway.
		hasImage := message.Item.ImageTags == nil || tag != ""
		if m.imageCapable && m.client != nil && hasImage {
			itemID := message.Item.Id
			c := m.client
			imgCols := m.width - 4
			if imgCols < 8 {
				imgCols = 8
			}
			imgRows := m.ImageRows()
			// Multiply terminal cells by approximate pixel dimensions (9×20) to get real pixel size.
			maxW := imgCols * 9
			maxH := imgRows * 20
			cmd = func() tea.Msg {
				data, err := c.GetImage(context.Background(), itemID, maxW, maxH, tag)
				if err != nil {
					return nil
				}
				return msg.ImageLoaded{Data: data, ItemId: itemID}
			}
		}
		return m, cmd
	case msg.ImageLoaded:
		if message.ItemId == m.item.Id {
			m.imageData = message.Data
		}
	}
	return m, nil
}

func (m Model) View() string {
	textWidth := m.width - 4
	if textWidth <= 0 {
		textWidth = 76
	}

	var content strings.Builder

	if m.HasImage() {
		for i := 0; i < m.ImageRows(); i++ {
			content.WriteByte('\n')
		}
	}

	if m.item.Id == "" {
		content.WriteString(styles.Dim.Render("Select an item\nto see details"))
	} else {
		content.WriteString(styles.Title.Render(m.item.Name))
		if m.item.ProductionYear > 0 {
			content.WriteString(styles.Subtitle.Render(fmt.Sprintf(" (%d)", m.item.ProductionYear)))
		}
		content.WriteByte('\n')

		if m.item.RunTimeTicks > 0 {
			mins := m.item.RunTimeTicks / 600_000_000
			content.WriteString(styles.Dim.Render(fmt.Sprintf("%dh%02dm", mins/60, mins%60)))
			content.WriteByte('\n')
		}

		if m.item.CommunityRating > 0 {
			content.WriteString(styles.Dim.Render(fmt.Sprintf("★ %.1f", m.item.CommunityRating)))
			content.WriteByte('\n')
		}

		if m.item.Overview != "" {
			content.WriteByte('\n')
			content.WriteString(wordWrap(m.item.Overview, textWidth))
			content.WriteByte('\n')
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
			content.WriteString(styles.Label.Render("Director: ") + strings.Join(directors, ", ") + "\n")
		}
		if len(cast) > 0 {
			content.WriteString(styles.Label.Render("Cast: ") + strings.Join(cast, ", ") + "\n")
		}
	}

	if m.height > 0 && m.width > 0 {
		return styles.Overlay.Width(m.width - 2).Height(m.height - 2).MaxHeight(m.height).Render(content.String())
	}
	return styles.Overlay.Render(content.String())
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
