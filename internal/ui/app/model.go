// internal/ui/app/model.go
package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/auth"
	"github.com/deglerj/fin/internal/config"
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
	overlayDetails
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

	overlay  overlayKind
	errorMsg string
	width    int
	height   int
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
	return tea.Batch(m.login.Init(), m.browser.Init())
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width = message.Width
		m.height = message.Height
		m.browser = browser.New(m.client, m.width, m.height-1)
		m.details = m.details.WithSize(m.width, m.height)
		return m, nil

	case msg.LoginSuccess:
		m.client = api.New(message.ServerURL)
		m.client.SetAuth(message.UserID, message.AccessToken)
		m.browser = browser.New(m.client, m.width, m.height-1)
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
		var cmd tea.Cmd
		m.browser, cmd = asBrowserModel(m.browser.Update(msg.PushLevel{
			Items: items, LevelName: "Libraries",
		}))
		return m, cmd

	case msg.NavigateToItem:
		// Search result selected — close overlay and push item into browser
		m.overlay = overlayNone
		var cmd tea.Cmd
		m.browser, cmd = asBrowserModel(m.browser.Update(msg.PushLevel{
			Items: []api.Item{message.Item}, LevelName: message.Item.Name,
		}))
		return m, cmd

	case msg.OpenDetails:
		m.overlay = overlayDetails
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
		return m, player.Play(m.cfg.Player.Command, m.cfg.Player.ExtraArgs, url, message.Item.Name)

	case msg.PlayerDone:
		if message.Err != nil {
			m.errorMsg = message.Err.Error()
		}
		return m, nil

	case msg.AppError:
		m.errorMsg = message.Err.Error()
		return m, nil

	case msg.DismissError:
		m.errorMsg = ""
		return m, nil

	case tea.KeyMsg:
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
	if m.overlay == overlayDetails {
		var cmd tea.Cmd
		m.details, cmd = asDetailsModel(m.details.Update(message))
		return m, cmd
	}
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
	updated, cmd := m.browser.Update(message)
	m.browser = updated.(browser.Model)
	return m, cmd
}

func (m Model) View() string {
	var base string
	if m.screen == ScreenLogin {
		base = m.login.View()
	} else {
		base = m.browser.View()
	}

	var sb strings.Builder
	sb.WriteString(base)

	switch m.overlay {
	case overlayDetails:
		sb.WriteString("\n" + m.details.View())
	case overlaySearch:
		sb.WriteString("\n" + m.search.View())
	case overlayHelp:
		sb.WriteString("\n" + help.View())
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
		libs, err := c.GetLibraries()
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
