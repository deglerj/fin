// internal/ui/app/model_test.go
package app_test

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/ui/app"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/stretchr/testify/require"
)

func TestStartsAtLogin(t *testing.T) {
	m := app.New(nil, nil, false)
	require.Equal(t, app.ScreenLogin, m.Screen())
}

func TestLoginSuccessTransition(t *testing.T) {
	m := app.New(nil, nil, false)
	m2, _ := m.Update(msg.LoginSuccess{ServerURL: "http://jf", UserID: "u1", AccessToken: "tok"})
	am := m2.(app.Model)
	require.Equal(t, app.ScreenBrowser, am.Screen())
}

func TestErrorDisplayed(t *testing.T) {
	m := app.New(nil, nil, false)
	m2, _ := m.Update(msg.AppError{Err: fmt.Errorf("network timeout")})
	view := m2.(app.Model).View()
	require.True(t, strings.Contains(view, "network timeout"), "error not in view: %q", view)
}

func TestAppErrorResetsBrowserLoading(t *testing.T) {
	m := app.New(nil, nil, false)
	// Transition to browser screen
	m2, _ := m.Update(msg.LoginSuccess{ServerURL: "http://jf", UserID: "u1", AccessToken: "tok"})
	// Push a series into the browser
	m3, _ := m2.(app.Model).Update(msg.PushLevel{
		Items:     []api.Item{{Id: "s1", Name: "Breaking Bad", Type: "Series"}},
		LevelName: "Shows",
	})
	// Press enter — browser sets loading=true
	m4, _ := m3.(app.Model).Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Contains(t, m4.(app.Model).View(), "Loading...")
	// Send AppError through app — should clear browser loading
	m5, _ := m4.(app.Model).Update(msg.AppError{Err: fmt.Errorf("network error")})
	require.NotContains(t, m5.(app.Model).View(), "Loading...")
}

func TestLibrariesLoadedPrependsVirtualSections(t *testing.T) {
	m := app.New(nil, nil, false)
	// Go to browser screen first
	m2, _ := m.Update(msg.LoginSuccess{ServerURL: "http://jf", UserID: "u1", AccessToken: "tok"})
	// Set terminal size so items are rendered in View
	m3, _ := m2.(app.Model).Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	// Send libraries
	m4, _ := m3.(app.Model).Update(msg.LibrariesLoaded{
		Libraries: []api.Library{{Id: "lib1", Name: "Movies"}},
	})
	view := m4.(app.Model).View()
	require.Contains(t, view, "Next Up")
	require.Contains(t, view, "Continue Watching")
	require.Contains(t, view, "Recently Added")
	require.Contains(t, view, "Favorites")
	require.Contains(t, view, "Movies")
}
