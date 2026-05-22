// internal/ui/app/model.go
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/auth"
	"github.com/deglerj/fin/internal/config"
	"github.com/deglerj/fin/internal/image"
	"github.com/deglerj/fin/internal/player"
	"github.com/deglerj/fin/internal/ui/browser"
	"github.com/deglerj/fin/internal/ui/details"
	"github.com/deglerj/fin/internal/ui/help"
	"github.com/deglerj/fin/internal/ui/login"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/deglerj/fin/internal/ui/search"
	"github.com/deglerj/fin/internal/ui/styles"
)

type ScreenKind int

const (
	ScreenLogin ScreenKind = iota
	ScreenBrowser
)

type overlayKind int

const (
	overlayNone overlayKind = iota
	overlaySearch
	overlayHelp
)

type Model struct {
	cfg          *config.Config
	client       *api.Client
	imageCapable bool

	screen  ScreenKind
	login   login.Model
	browser browser.Model
	details details.Model
	search  search.Model

	overlay            overlayKind
	selectedItemID     string
	playingItem        *api.Item
	playingChapterFile string
	errorMsg           string
	width              int
	height             int
}

func New(cfg *config.Config, client *api.Client, imageCapable bool) Model {
	return Model{
		cfg:          cfg,
		client:       client,
		imageCapable: imageCapable,
		screen:       ScreenLogin,
		login:        login.New(),
		browser:      browser.New(client, 80, 24),
		details:      details.New(imageCapable),
		search:       search.New(client),
	}
}

func (m Model) Screen() ScreenKind         { return m.screen }
func (m Model) PlayingChapterFile() string { return m.playingChapterFile }

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.login.Init(), m.browser.Init()}
	if m.screen == ScreenBrowser && m.client != nil {
		cmds = append(cmds, m.fetchLibraries())
	}
	return tea.Batch(cmds...)
}

func (m Model) browserWidth() int {
	bw := m.width * 2 / 3
	if m.width > 0 && bw < 20 {
		bw = 20
	}
	return bw
}

func (m Model) syncDetailsToSelection() (Model, tea.Cmd) {
	sel := m.browser.SelectedItem()
	if sel.Id == m.selectedItemID {
		return m, nil
	}
	m.selectedItemID = sel.Id
	var cmd tea.Cmd
	m.details, cmd = asDetailsModel(m.details.Update(msg.OpenDetails{Item: sel}))
	return m, cmd
}

func (m Model) updateBrowser(message tea.Msg) (Model, tea.Cmd) {
	var browserCmd tea.Cmd
	m.browser, browserCmd = asBrowserModel(m.browser.Update(message))
	var detailsCmd tea.Cmd
	m, detailsCmd = m.syncDetailsToSelection()
	return m, tea.Batch(browserCmd, detailsCmd)
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		bw := m.browserWidth()
		m.browser = m.browser.WithSize(bw, m.height)
		m.details = m.details.WithSize(m.width-bw, m.height)
		m.search = m.search.WithWidth(bw)
		return m, nil

	case msg.LoginSuccess:
		m.client = api.New(message.ServerURL)
		m.client.SetAuth(message.UserID, message.AccessToken)
		bw := m.browserWidth()
		m.browser = browser.New(m.client, bw, m.height)
		m.search = search.New(m.client).WithWidth(bw)
		m.details = m.details.WithClient(m.client)
		m.screen = ScreenBrowser
		if m.cfg != nil {
			go saveCredentials(message, m.cfg)
		}
		return m, m.fetchLibraries()

	case msg.LibrariesLoaded:
		virtual := []api.Item{
			{Id: "__next_up__", Name: "Next Up", Type: "VirtualSection"},
			{Id: "__resume__", Name: "Continue Watching", Type: "VirtualSection"},
			{Id: "__latest__", Name: "Recently Added", Type: "VirtualSection"},
			{Id: "__favorites__", Name: "Favorites", Type: "VirtualSection"},
		}
		libs := make([]api.Item, len(message.Libraries))
		for i, lib := range message.Libraries {
			libs[i] = api.Item{Id: lib.Id, Name: lib.Name, Type: "Folder"}
		}
		return m.updateBrowser(msg.PushLevel{Items: append(virtual, libs...), LevelName: "Libraries"})

	case msg.PushLevel:
		return m.updateBrowser(message)

	case msg.NavigateToItem:
		m.overlay = overlayNone
		return m.updateBrowser(msg.PushLevel{Items: []api.Item{message.Item}, LevelName: message.Item.Name})

	case msg.ImageLoaded:
		var cmd tea.Cmd
		m.details, cmd = asDetailsModel(m.details.Update(message))
		return m, cmd

	case msg.OpenSearch:
		m.overlay = overlaySearch
		return m, nil

	case msg.CloseOverlay:
		m.overlay = overlayNone
		return m, nil

	case msg.PlayItem:
		item := message.Item
		m.playingItem = &item
		if m.cfg == nil || m.client == nil {
			return m, nil
		}
		client := m.client
		originalItem := item
		return m, func() tea.Msg {
			fetched, err := client.GetItem(context.Background(), originalItem.Id)
			if err != nil {
				return msg.ItemReadyToPlay{Item: originalItem}
			}
			return msg.ItemReadyToPlay{Item: fetched}
		}

	case msg.ItemReadyToPlay:
		item := message.Item
		if m.cfg == nil || m.client == nil {
			return m, nil
		}
		url := m.client.StreamURL(item)
		startSec := item.UserData.PlaybackPositionTicks / 10_000_000
		socketPath := player.SocketPath()
		client := m.client
		go player.Monitor(socketPath, client, item, startSec)

		extraArgs := append([]string{}, m.cfg.Player.ExtraArgs...)
		if isVideoItem(item) && len(item.Chapters) > 0 {
			if path, err := WriteChapterFile(item.Chapters); err == nil {
				m.playingChapterFile = path
				extraArgs = append(extraArgs, "--chapter-list="+path)
			}
		}
		return m, player.Play(m.cfg.Player.Command, extraArgs, url, item.MediaTitle(), startSec, socketPath)

	case player.DoneMsg:
		if m.playingChapterFile != "" {
			_ = os.Remove(m.playingChapterFile)
			m.playingChapterFile = ""
		}
		if message.Err != nil {
			m.errorMsg = message.Err.Error()
		}
		item := m.playingItem
		m.playingItem = nil
		if item != nil && m.client != nil {
			client := m.client
			itemID := item.Id
			return m, func() tea.Msg {
				updated, err := client.GetItem(context.Background(), itemID)
				if err != nil {
					return nil
				}
				return msg.PlayedToggled{ItemID: itemID, Played: updated.UserData.Played}
			}
		}
		return m, nil

	case msg.PlayedToggled:
		m, cmd := m.updateBrowser(message)
		return m, tea.Batch(cmd, m.refreshCurrentLevel())

	case msg.AppError:
		m.errorMsg = message.Err.Error()
		m.browser, _ = asBrowserModel(m.browser.Update(message))
		return m, nil

	case msg.DismissError:
		m.errorMsg = ""
		return m, nil

	case msg.FetchVirtualSection:
		c := m.client
		if c == nil {
			return m, func() tea.Msg { return msg.AppError{Err: fmt.Errorf("no client configured")} }
		}
		switch message.ID {
		case "__next_up__", "__resume__", "__latest__", "__favorites__":
			id := message.ID
			lname := virtualSectionName(id)
			return m, func() tea.Msg {
				items, err := fetchVirtualItems(c, id)
				if err != nil {
					return msg.AppError{Err: err}
				}
				return msg.PushLevel{Items: items, LevelName: lname, ParentID: id}
			}
		default:
			return m, nil
		}

	case tea.KeyMsg:
		if message.String() == "ctrl+c" || (message.String() == "q" && m.screen != ScreenLogin) {
			return m, tea.Quit
		}
		if message.String() == "?" && m.overlay == overlayNone {
			m.overlay = overlayHelp
			return m, nil
		}
		if message.String() == "?" && m.overlay == overlayHelp {
			m.overlay = overlayNone
			return m, nil
		}
		if message.String() == "esc" && m.errorMsg != "" {
			m.errorMsg = ""
			return m, nil
		}
	}

	// Delegate to active overlay or screen
	if m.overlay == overlaySearch {
		var cmd tea.Cmd
		m.search, cmd = asSearchModel(m.search.Update(message))
		if sel := m.search.SelectedItem(); sel != nil && sel.Id != m.selectedItemID {
			m.selectedItemID = sel.Id
			var detCmd tea.Cmd
			m.details, detCmd = asDetailsModel(m.details.Update(msg.OpenDetails{Item: *sel}))
			cmd = tea.Batch(cmd, detCmd)
		}
		return m, cmd
	}
	if m.screen == ScreenLogin {
		updated, cmd := m.login.Update(message)
		m.login = updated.(login.Model)
		return m, cmd
	}
	return m.updateBrowser(message)
}

func (m Model) View() string {
	if m.screen == ScreenLogin {
		var sb strings.Builder
		if m.imageCapable {
			sb.WriteString(image.Delete())
		}
		sb.WriteString(m.login.View())
		if m.errorMsg != "" {
			sb.WriteString("\n" + styles.Error.Render("Error: "+m.errorMsg+"  [esc to dismiss]"))
		}
		return sb.String()
	}

	var sb strings.Builder

	bw := m.browserWidth()

	var leftView string
	switch m.overlay {
	case overlaySearch:
		leftView = lipgloss.NewStyle().Width(bw).Height(m.height).Render(m.search.View())
	case overlayHelp:
		leftView = lipgloss.NewStyle().Width(bw).Height(m.height).Render(help.View())
	default:
		leftView = lipgloss.NewStyle().Width(bw).Height(m.height).Render(m.browser.View())
	}

	if m.browser.IsLoading() {
		sb.WriteString(leftView)
	} else {
		sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftView, m.details.View()))
	}

	if m.imageCapable {
		// Always delete before placing: bubbletea batches frames at 60fps, so the
		// intermediate "no image" frame (with only Delete) may be skipped when
		// ImageLoaded arrives within the same tick as OpenDetails.
		sb.WriteString(image.Delete())
		if !m.browser.IsLoading() && m.details.HasImage() {
			// Place image after text so it renders on top of the reserved blank rows.
			// \x1b[2;{bw+2}H = row 2 (below details top border), col bw+2 (inside left border).
			sb.WriteString("\x1b7")
			fmt.Fprintf(&sb, "\x1b[2;%dH", bw+2)
			sb.WriteString(image.Encode(m.details.ImageData()))
			sb.WriteString("\x1b8")
		}
	}

	if m.errorMsg != "" {
		sb.WriteString("\n" + styles.Error.Render("Error: "+m.errorMsg+"  [esc to dismiss]"))
	}
	return sb.String()
}

func (m Model) fetchLibraries() tea.Cmd {
	c := m.client
	if c == nil {
		return nil
	}
	return func() tea.Msg {
		libs, err := c.GetLibraries(context.Background())
		if err != nil {
			return msg.AppError{Err: err}
		}
		return msg.LibrariesLoaded{Libraries: libs}
	}
}

func (m Model) refreshCurrentLevel() tea.Cmd {
	parentID := m.browser.CurrentLevelParentID()
	c := m.client
	if c == nil || parentID == "" {
		return nil
	}
	switch parentID {
	case "__next_up__", "__resume__", "__latest__", "__favorites__":
		return func() tea.Msg {
			items, err := fetchVirtualItems(c, parentID)
			if err != nil {
				return msg.AppError{Err: err}
			}
			return msg.RefreshLevel{ParentID: parentID, Items: items}
		}
	default:
		return func() tea.Msg {
			items, err := c.GetItems(context.Background(), parentID, nil)
			if err != nil {
				return msg.AppError{Err: err}
			}
			return msg.RefreshLevel{ParentID: parentID, Items: items}
		}
	}
}

func fetchVirtualItems(c *api.Client, parentID string) ([]api.Item, error) {
	switch parentID {
	case "__next_up__":
		return c.GetNextUp(context.Background())
	case "__resume__":
		return c.GetResumeItems(context.Background())
	case "__latest__":
		return c.GetLatestMedia(context.Background())
	case "__favorites__":
		return c.GetFavorites(context.Background())
	}
	return nil, nil
}

func virtualSectionName(id string) string {
	switch id {
	case "__next_up__":
		return "Next Up"
	case "__resume__":
		return "Continue Watching"
	case "__latest__":
		return "Recently Added"
	case "__favorites__":
		return "Favorites"
	}
	return ""
}

func saveCredentials(ls msg.LoginSuccess, cfg *config.Config) {
	creds := auth.Credentials{
		ServerURL:   ls.ServerURL,
		UserID:      ls.UserID,
		AccessToken: ls.AccessToken,
	}
	_ = auth.Save(creds, cfg.CredentialsPath(), auth.DefaultMachineID{})
}

func asBrowserModel(m tea.Model, cmd tea.Cmd) (browser.Model, tea.Cmd) {
	bm, ok := m.(browser.Model)
	if !ok {
		panic(fmt.Sprintf("expected browser.Model, got %T", m))
	}
	return bm, cmd
}
func asDetailsModel(m tea.Model, cmd tea.Cmd) (details.Model, tea.Cmd) {
	dm, ok := m.(details.Model)
	if !ok {
		panic(fmt.Sprintf("expected details.Model, got %T", m))
	}
	return dm, cmd
}
func asSearchModel(m tea.Model, cmd tea.Cmd) (search.Model, tea.Cmd) {
	sm, ok := m.(search.Model)
	if !ok {
		panic(fmt.Sprintf("expected search.Model, got %T", m))
	}
	return sm, cmd
}

func isVideoItem(item api.Item) bool {
	return item.Type == "Movie" || item.Type == "Episode"
}

func WriteChapterFile(chapters []api.ChapterInfo) (string, error) {
	type mpvChapter struct {
		Title string  `json:"title"`
		Time  float64 `json:"time"`
	}
	type mpvChapterList struct {
		Chapters []mpvChapter `json:"chapters"`
	}
	list := mpvChapterList{Chapters: make([]mpvChapter, len(chapters))}
	for i, c := range chapters {
		list.Chapters[i] = mpvChapter{
			Title: c.Name,
			Time:  float64(c.StartPositionTicks) / 10_000_000,
		}
	}
	f, err := os.CreateTemp("", "fin-chapters-*.json")
	if err != nil {
		return "", err
	}
	if err := json.NewEncoder(f).Encode(list); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}
