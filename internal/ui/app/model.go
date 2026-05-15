// internal/ui/app/model.go
package app

import (
	"context"
	"fmt"
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

	overlay        overlayKind
	selectedItemID string
	errorMsg       string
	width          int
	height         int
}

func New(cfg *config.Config, client *api.Client, imageCapable bool) Model {
	return Model{
		cfg:          cfg,
		client:       client,
		imageCapable: imageCapable,
		screen:       ScreenLogin,
		login:        login.New(client),
		browser:      browser.New(client, 80, 24),
		details:      details.New(imageCapable),
		search:       search.New(client),
	}
}

func (m Model) Screen() ScreenKind { return m.screen }

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
		return m, nil

	case msg.LoginSuccess:
		m.client = api.New(message.ServerURL)
		m.client.SetAuth(message.UserID, message.AccessToken)
		bw := m.browserWidth()
		m.browser = browser.New(m.client, bw, m.height)
		m.search = search.New(m.client)
		m.details = m.details.WithClient(m.client)
		m.screen = ScreenBrowser
		if m.cfg != nil {
			go saveCredentials(message, m.cfg)
		}
		return m, m.fetchLibraries()

	case msg.LibrariesLoaded:
		items := make([]api.Item, len(message.Libraries))
		for i, lib := range message.Libraries {
			items[i] = api.Item{Id: lib.Id, Name: lib.Name, Type: "Folder"}
		}
		return m.updateBrowser(msg.PushLevel{Items: items, LevelName: "Libraries"})

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
		if m.cfg == nil || m.client == nil {
			return m, nil
		}
		url := m.client.StreamURL(message.Item)
		return m, player.Play(m.cfg.Player.Command, m.cfg.Player.ExtraArgs, url, message.Item.MediaTitle())

	case msg.PlayerDone:
		if message.Err != nil {
			m.errorMsg = message.Err.Error()
		}
		return m, nil

	case msg.AppError:
		m.errorMsg = message.Err.Error()
		m.browser, _ = asBrowserModel(m.browser.Update(message))
		return m, nil

	case msg.DismissError:
		m.errorMsg = ""
		return m, nil

	case tea.KeyMsg:
		if message.String() == "q" || message.String() == "ctrl+c" {
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
		if !m.browser.IsLoading() && m.details.HasImage() {
			// Place image after text so it renders on top of the reserved blank rows.
			// \x1b[2;{bw+2}H = row 2 (below details top border), col bw+2 (inside left border).
			sb.WriteString("\x1b7")
			fmt.Fprintf(&sb, "\x1b[2;%dH", bw+2)
			sb.WriteString(image.Encode(m.details.ImageData()))
			sb.WriteString("\x1b8")
		} else {
			sb.WriteString(image.Delete())
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

func saveCredentials(ls msg.LoginSuccess, cfg *config.Config) {
	creds := auth.Credentials{
		ServerURL:   ls.ServerURL,
		UserID:      ls.UserID,
		AccessToken: ls.AccessToken,
	}
	_ = auth.Save(creds, cfg.CredentialsPath(), auth.DefaultMachineID{})
}

func asBrowserModel(m tea.Model, cmd tea.Cmd) (browser.Model, tea.Cmd) {
	return m.(browser.Model), cmd
}
func asDetailsModel(m tea.Model, cmd tea.Cmd) (details.Model, tea.Cmd) {
	return m.(details.Model), cmd
}
func asSearchModel(m tea.Model, cmd tea.Cmd) (search.Model, tea.Cmd) {
	return m.(search.Model), cmd
}
