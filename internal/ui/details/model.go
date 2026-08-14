// internal/ui/details/model.go
package details

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/image"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/deglerj/fin/internal/ui/styles"
)

type Model struct {
	item         api.Item
	imageCapable bool
	imageData    []byte
	imageCols    int
	imageRows    int
	client       apiClient
	width        int
	height       int
	// vp scrolls the text below the poster; long overviews and cast lists
	// would otherwise be silently cut off at the pane's bottom edge.
	vp viewport.Model
}

type apiClient interface {
	GetImage(ctx context.Context, itemID string, maxWidth, maxHeight int, tag string) ([]byte, error)
}

func New(imageCapable bool) Model {
	return Model{imageCapable: imageCapable, vp: viewport.New(0, 0)}
}

func (m Model) WithClient(c apiClient) Model { m.client = c; return m }

func (m Model) WithSize(w, h int) Model {
	m.width, m.height = w, h
	return m.fitImage().refresh()
}

func (m Model) HasImage() bool    { return m.imageRows > 0 }
func (m Model) ImageData() []byte { return m.imageData }

// ImageCols and ImageRows are the cells the placed image occupies — the same
// numbers View reserves blank lines for, so the image cannot cover text.
func (m Model) ImageCols() int { return m.imageCols }
func (m Model) ImageRows() int { return m.imageRows }

// Scroll moves the text by n lines; negative scrolls up. The poster stays put,
// since it is placed at a fixed terminal row by the parent model.
func (m Model) Scroll(n int) Model {
	switch {
	case n > 0:
		m.vp.ScrollDown(n)
	case n < 0:
		m.vp.ScrollUp(-n)
	}
	return m
}

// fitImage recomputes the image's cell footprint for the current pane size.
func (m Model) fitImage() Model {
	m.imageCols, m.imageRows = 0, 0
	if len(m.imageData) > 0 {
		m.imageCols, m.imageRows = image.Fit(m.imageData, m.maxImageCols(), m.maxImageRows())
	}
	return m
}

// refresh resizes the viewport around the current poster footprint and rebuilds
// its content.
func (m Model) refresh() Model {
	m.vp.Width = max(m.textWidth(), 1)
	// -2 for the border, -1 for the hint row View always reserves.
	m.vp.Height = max(m.height-3-m.imageRows, 1)
	m.vp.SetContent(m.body())
	return m
}

func (m Model) textWidth() int {
	if w := m.width - 4; w > 0 {
		return w
	}
	return 76
}

func (m Model) maxImageCols() int {
	c := m.width - 4
	if c < 8 {
		c = 8
	}
	return c
}

func (m Model) maxImageRows() int {
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
		m.imageCols, m.imageRows = 0, 0
		m = m.refresh()
		m.vp.SetYOffset(0)
		var cmd tea.Cmd
		tag := message.Item.ImageTags["Primary"]
		// ImageTags == nil means no tag data (e.g. library folders constructed without it) — attempt fetch anyway.
		hasImage := message.Item.ImageTags == nil || tag != ""
		if m.imageCapable && m.client != nil && hasImage {
			itemID := message.Item.Id
			c := m.client
			// Ask the server for the pixel size the pane's cell budget really is.
			cellW, cellH := image.CellSize()
			maxW := m.maxImageCols() * cellW
			maxH := m.maxImageRows() * cellH
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
			return m.fitImage().refresh(), nil
		}
	}
	return m, nil
}

// body renders the scrollable text of the pane, excluding the poster.
func (m Model) body() string {
	if m.item.Id == "" {
		return styles.Dim.Render("Select an item\nto see details")
	}

	var content strings.Builder
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
		content.WriteString(ansi.Wrap(m.item.Overview, m.textWidth(), ""))
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
	return content.String()
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	var content strings.Builder
	for i := 0; i < m.imageRows; i++ {
		content.WriteByte('\n')
	}
	content.WriteString(m.vp.View())
	content.WriteByte('\n')
	if m.vp.TotalLineCount() > m.vp.Height {
		content.WriteString(styles.Dim.Render("J/K scroll"))
	}

	return styles.Overlay.Width(m.width - 2).Height(m.height - 2).MaxHeight(m.height).Render(content.String())
}
