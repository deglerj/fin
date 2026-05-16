// internal/ui/login/model.go
package login

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/deglerj/fin/internal/ui/styles"
)

type field int

const (
	fieldServer field = iota
	fieldUsername
	fieldPassword
	fieldCount
)

type Model struct {
	inputs   []textinput.Model
	focused  field
	loading  bool
	spinner  spinner.Model
	errorMsg string
}

func New() Model {
	inputs := make([]textinput.Model, int(fieldCount))
	for i := range inputs {
		inputs[i] = textinput.New()
	}
	inputs[fieldServer].Placeholder = "https://jellyfin.example.com"
	inputs[fieldServer].Focus()
	inputs[fieldUsername].Placeholder = "username"
	inputs[fieldPassword].Placeholder = "password"
	inputs[fieldPassword].EchoMode = textinput.EchoPassword
	inputs[fieldPassword].EchoCharacter = '•'

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return Model{inputs: inputs, spinner: sp}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case msg.LoginError:
		m.loading = false
		m.errorMsg = message.Err.Error()
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(message)
			return m, cmd
		}

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}
		switch message.String() {
		case "tab", "down":
			m.focused = (m.focused + 1) % fieldCount
			syncFocus(m.inputs, m.focused)
		case "shift+tab", "up":
			m.focused = (m.focused + fieldCount - 1) % fieldCount
			syncFocus(m.inputs, m.focused)
		case "enter":
			if m.focused < fieldPassword {
				m.focused++
				syncFocus(m.inputs, m.focused)
			} else {
				return submit(m)
			}
		}
	}

	var cmds []tea.Cmd
	for i := range m.inputs {
		var cmd tea.Cmd
		m.inputs[i], cmd = m.inputs[i].Update(message)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func syncFocus(inputs []textinput.Model, focused field) {
	for i := range inputs {
		if field(i) == focused {
			inputs[i].Focus()
		} else {
			inputs[i].Blur()
		}
	}
}

func submit(m Model) (tea.Model, tea.Cmd) {
	serverURL := m.inputs[fieldServer].Value()
	username := m.inputs[fieldUsername].Value()
	password := m.inputs[fieldPassword].Value()
	if serverURL == "" || username == "" || password == "" {
		m.errorMsg = "all fields required"
		return m, nil
	}
	m.loading = true
	m.errorMsg = ""
	client := api.New(serverURL)
	return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
		resp, err := client.Authenticate(context.Background(), username, password)
		if err != nil {
			return msg.LoginError{Err: err}
		}
		return msg.LoginSuccess{
			ServerURL:   serverURL,
			UserID:      resp.User.Id,
			AccessToken: resp.AccessToken,
		}
	})
}

func (m Model) View() string {
	title := styles.Title.Render("fin — Jellyfin TUI")
	form := fmt.Sprintf(
		"%s\n%s\n\n%s\n%s\n\n%s\n%s",
		styles.Label.Render("Server URL"),
		m.inputs[fieldServer].View(),
		styles.Label.Render("Username"),
		m.inputs[fieldUsername].View(),
		styles.Label.Render("Password"),
		m.inputs[fieldPassword].View(),
	)
	hint := styles.Dim.Render("tab/↑↓ to move · enter to confirm · enter on password to login")
	bottom := hint
	if m.loading {
		bottom = m.spinner.View() + " Authenticating..."
	} else if m.errorMsg != "" {
		bottom = styles.Error.Render("Error: " + m.errorMsg)
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", form, "", bottom,
	)
}
