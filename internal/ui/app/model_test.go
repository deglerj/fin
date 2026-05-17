// internal/ui/app/model_test.go
package app_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

func newAppWithMockServer(t *testing.T, handler http.Handler) app.Model {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := api.New(srv.URL)
	client.SetAuth("u1", "tok")
	return app.New(nil, client, false)
}

func TestFetchVirtualSectionNextUp(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Shows/NextUp" {
			json.NewEncoder(w).Encode(api.ItemsResponse{
				Items: []api.Item{{Id: "ep1", Name: "Episode 1", Type: "Episode"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	_, cmd := m.Update(msg.FetchVirtualSection{ID: "__next_up__"})
	require.NotNil(t, cmd)
	result := cmd()
	push, ok := result.(msg.PushLevel)
	require.True(t, ok, "expected PushLevel, got %T", result)
	require.Equal(t, "Next Up", push.LevelName)
	require.Len(t, push.Items, 1)
}

func TestFetchVirtualSectionResume(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/UserItems/Resume" {
			json.NewEncoder(w).Encode(api.ItemsResponse{
				Items: []api.Item{{Id: "m1", Name: "Inception", Type: "Movie"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	_, cmd := m.Update(msg.FetchVirtualSection{ID: "__resume__"})
	require.NotNil(t, cmd)
	result := cmd()
	push, ok := result.(msg.PushLevel)
	require.True(t, ok, "expected PushLevel, got %T", result)
	require.Equal(t, "Continue Watching", push.LevelName)
	require.Len(t, push.Items, 1)
}

func TestFetchVirtualSectionLatest(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/Items/Latest") {
			json.NewEncoder(w).Encode([]api.Item{{Id: "m2", Name: "Dune", Type: "Movie"}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	_, cmd := m.Update(msg.FetchVirtualSection{ID: "__latest__"})
	require.NotNil(t, cmd)
	result := cmd()
	push, ok := result.(msg.PushLevel)
	require.True(t, ok, "expected PushLevel, got %T", result)
	require.Equal(t, "Recently Added", push.LevelName)
	require.Len(t, push.Items, 1)
}

func TestFetchVirtualSectionFavorites(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Items" && r.URL.Query().Get("isFavorite") == "true" {
			json.NewEncoder(w).Encode(api.ItemsResponse{
				Items: []api.Item{{Id: "m3", Name: "The Matrix", Type: "Movie"}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	_, cmd := m.Update(msg.FetchVirtualSection{ID: "__favorites__"})
	require.NotNil(t, cmd)
	result := cmd()
	push, ok := result.(msg.PushLevel)
	require.True(t, ok, "expected PushLevel, got %T", result)
	require.Equal(t, "Favorites", push.LevelName)
	require.Len(t, push.Items, 1)
}
