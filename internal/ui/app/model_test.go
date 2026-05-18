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
	"github.com/deglerj/fin/internal/player"
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
			require.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
				Items: []api.Item{{Id: "ep1", Name: "Episode 1", Type: "Episode"}},
			}))
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
			require.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
				Items: []api.Item{{Id: "m1", Name: "Inception", Type: "Movie"}},
			}))
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
			require.NoError(t, json.NewEncoder(w).Encode([]api.Item{{Id: "m2", Name: "Dune", Type: "Movie"}}))
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
			require.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
				Items: []api.Item{{Id: "m3", Name: "The Matrix", Type: "Movie"}},
			}))
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

func TestPlayerDoneRefreshesPlayedStatus(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/Items/movie1") {
			require.NoError(t, json.NewEncoder(w).Encode(api.Item{
				Id:       "movie1",
				Type:     "Movie",
				UserData: api.UserData{Played: true},
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	// PlayItem stores the item even when cfg is nil (no actual player launched).
	m2, _ := m.Update(msg.PlayItem{Item: api.Item{Id: "movie1", Name: "Dune", Type: "Movie"}})

	// PlayerDone should return a cmd that fetches the updated item.
	_, cmd := m2.(app.Model).Update(player.DoneMsg{})
	require.NotNil(t, cmd)

	result := cmd()
	toggled, ok := result.(msg.PlayedToggled)
	require.True(t, ok, "expected PlayedToggled, got %T", result)
	require.Equal(t, "movie1", toggled.ItemID)
	require.True(t, toggled.Played)
}

func TestPlayerDoneNoItemStoredReturnsNilCmd(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	// No PlayItem sent — playingItem is nil.
	_, cmd := m.Update(player.DoneMsg{})
	require.Nil(t, cmd)
}

func TestPlayerDoneGetItemFailsGracefully(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	m2, _ := m.Update(msg.PlayItem{Item: api.Item{Id: "movie1", Type: "Movie"}})
	_, cmd := m2.(app.Model).Update(player.DoneMsg{})
	require.NotNil(t, cmd)

	result := cmd()
	// GetItem fails → nil result, no PlayedToggled emitted.
	require.Nil(t, result)
}

func TestPlayerDoneWithErrorStillRefreshes(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/Items/movie1") {
			require.NoError(t, json.NewEncoder(w).Encode(api.Item{
				Id:       "movie1",
				Type:     "Movie",
				UserData: api.UserData{Played: false},
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	m2, _ := m.Update(msg.PlayItem{Item: api.Item{Id: "movie1", Type: "Movie"}})

	// Error from player — refresh still happens.
	_, cmd := m2.(app.Model).Update(player.DoneMsg{Err: fmt.Errorf("mpv crashed")})
	require.NotNil(t, cmd)

	result := cmd()
	_, ok := result.(msg.PlayedToggled)
	require.True(t, ok, "expected PlayedToggled even on player error, got %T", result)
}

func TestFetchVirtualSectionNextUpSetsParentID(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Shows/NextUp" {
			require.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
				Items: []api.Item{{Id: "ep1", Name: "Ep 1", Type: "Episode"}},
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	_, cmd := m.Update(msg.FetchVirtualSection{ID: "__next_up__"})
	require.NotNil(t, cmd)
	push, ok := cmd().(msg.PushLevel)
	require.True(t, ok)
	require.Equal(t, "__next_up__", push.ParentID)
}

func TestFetchVirtualSectionResumeSetsParentID(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/UserItems/Resume" {
			require.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
				Items: []api.Item{{Id: "m1", Name: "Inception", Type: "Movie"}},
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	_, cmd := m.Update(msg.FetchVirtualSection{ID: "__resume__"})
	require.NotNil(t, cmd)
	push, ok := cmd().(msg.PushLevel)
	require.True(t, ok)
	require.Equal(t, "__resume__", push.ParentID)
}

func TestFetchVirtualSectionLatestSetsParentID(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/Items/Latest") {
			require.NoError(t, json.NewEncoder(w).Encode([]api.Item{{Id: "m2", Name: "Dune", Type: "Movie"}}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	_, cmd := m.Update(msg.FetchVirtualSection{ID: "__latest__"})
	require.NotNil(t, cmd)
	push, ok := cmd().(msg.PushLevel)
	require.True(t, ok)
	require.Equal(t, "__latest__", push.ParentID)
}

func TestFetchVirtualSectionFavoritesSetsParentID(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Items" && r.URL.Query().Get("isFavorite") == "true" {
			require.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
				Items: []api.Item{{Id: "m3", Name: "Matrix", Type: "Movie"}},
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	_, cmd := m.Update(msg.FetchVirtualSection{ID: "__favorites__"})
	require.NotNil(t, cmd)
	push, ok := cmd().(msg.PushLevel)
	require.True(t, ok)
	require.Equal(t, "__favorites__", push.ParentID)
}

// executeBatch executes cmd and all sub-commands in a batch, returning their msgs.
// If cmd is not a batch, returns []tea.Msg{cmd()}.
func executeBatch(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	result := cmd()
	if batch, ok := result.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, c := range batch {
			if c != nil {
				msgs = append(msgs, c())
			}
		}
		return msgs
	}
	return []tea.Msg{result}
}

func TestPlayedToggledRefreshesVirtualSection(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Shows/NextUp" {
			require.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
				Items: []api.Item{{Id: "ep1", Name: "Ep 1 Updated", Type: "Episode"}},
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	// Push virtual section level directly (simulates result of FetchVirtualSection after Task 2).
	// Do NOT call LoginSuccess — that would replace the mock client.
	m2, _ := m.Update(msg.PushLevel{
		ParentID:  "__next_up__",
		LevelName: "Next Up",
		Items:     []api.Item{{Id: "ep1", Name: "Ep 1", Type: "Episode"}},
	})
	_, cmd := m2.(app.Model).Update(msg.PlayedToggled{ItemID: "ep1", Played: true})
	require.NotNil(t, cmd)
	msgs := executeBatch(cmd)
	var found bool
	for _, result := range msgs {
		if rl, ok := result.(msg.RefreshLevel); ok {
			require.Equal(t, "__next_up__", rl.ParentID)
			require.Len(t, rl.Items, 1)
			found = true
		}
	}
	require.True(t, found, "expected RefreshLevel{ParentID: __next_up__} in batch cmds")
}

func TestPlayedToggledRefreshesRealFolder(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Users/u1/Items" && r.URL.Query().Get("ParentId") == "season1" {
			require.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
				Items: []api.Item{{Id: "ep1", Name: "Ep 1", Type: "Episode"}},
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	m2, _ := m.Update(msg.PushLevel{
		ParentID:  "season1",
		LevelName: "Season 1",
		Items:     []api.Item{{Id: "ep1", Name: "Ep 1", Type: "Episode"}},
	})
	_, cmd := m2.(app.Model).Update(msg.PlayedToggled{ItemID: "ep1", Played: true})
	require.NotNil(t, cmd)
	msgs := executeBatch(cmd)
	var found bool
	for _, result := range msgs {
		if rl, ok := result.(msg.RefreshLevel); ok {
			require.Equal(t, "season1", rl.ParentID)
			found = true
		}
	}
	require.True(t, found, "expected RefreshLevel{ParentID: season1} in batch cmds")
}

func TestPlayedToggledEmptyParentIDNoRefresh(t *testing.T) {
	// Client is non-nil but parentID is empty — tests the parentID=="" guard.
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	m2, _ := m.Update(msg.PushLevel{
		LevelName: "Libraries",
		Items:     []api.Item{{Id: "lib1", Name: "Movies", Type: "Folder"}},
	})
	_, cmd := m2.(app.Model).Update(msg.PlayedToggled{ItemID: "lib1", Played: true})
	msgs := executeBatch(cmd)
	for _, result := range msgs {
		_, isRefresh := result.(msg.RefreshLevel)
		require.False(t, isRefresh, "should not emit RefreshLevel when parentID is empty")
	}
}
