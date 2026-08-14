// internal/ui/app/model_test.go
package app_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/config"
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
	for _, result := range executeBatch(cmd) {
		_, isRefresh := result.(msg.RefreshLevel)
		require.False(t, isRefresh, "should not emit RefreshLevel when parentID is empty")
	}
}

func TestPlayedToggledRefreshesVirtualSectionResume(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/UserItems/Resume" {
			require.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
				Items: []api.Item{{Id: "m1", Name: "Inception", Type: "Movie"}},
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	m2, _ := m.Update(msg.PushLevel{
		ParentID:  "__resume__",
		LevelName: "Continue Watching",
		Items:     []api.Item{{Id: "m1", Name: "Inception", Type: "Movie"}},
	})
	_, cmd := m2.(app.Model).Update(msg.PlayedToggled{ItemID: "m1", Played: true})
	require.NotNil(t, cmd)
	msgs := executeBatch(cmd)
	var found bool
	for _, result := range msgs {
		if rl, ok := result.(msg.RefreshLevel); ok {
			require.Equal(t, "__resume__", rl.ParentID)
			found = true
		}
	}
	require.True(t, found, "expected RefreshLevel{ParentID: __resume__} in batch cmds")
}

func TestPlayedToggledRefreshesVirtualSectionLatest(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/Items/Latest") {
			require.NoError(t, json.NewEncoder(w).Encode([]api.Item{{Id: "m2", Name: "Dune", Type: "Movie"}}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	m2, _ := m.Update(msg.PushLevel{
		ParentID:  "__latest__",
		LevelName: "Recently Added",
		Items:     []api.Item{{Id: "m2", Name: "Dune", Type: "Movie"}},
	})
	_, cmd := m2.(app.Model).Update(msg.PlayedToggled{ItemID: "m2", Played: true})
	require.NotNil(t, cmd)
	msgs := executeBatch(cmd)
	var found bool
	for _, result := range msgs {
		if rl, ok := result.(msg.RefreshLevel); ok {
			require.Equal(t, "__latest__", rl.ParentID)
			found = true
		}
	}
	require.True(t, found, "expected RefreshLevel{ParentID: __latest__} in batch cmds")
}

func TestPlayedToggledRefreshesVirtualSectionFavorites(t *testing.T) {
	m := newAppWithMockServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Items" && r.URL.Query().Get("isFavorite") == "true" {
			require.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
				Items: []api.Item{{Id: "m3", Name: "Matrix", Type: "Movie"}},
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	m2, _ := m.Update(msg.PushLevel{
		ParentID:  "__favorites__",
		LevelName: "Favorites",
		Items:     []api.Item{{Id: "m3", Name: "Matrix", Type: "Movie"}},
	})
	_, cmd := m2.(app.Model).Update(msg.PlayedToggled{ItemID: "m3", Played: true})
	require.NotNil(t, cmd)
	msgs := executeBatch(cmd)
	var found bool
	for _, result := range msgs {
		if rl, ok := result.(msg.RefreshLevel); ok {
			require.Equal(t, "__favorites__", rl.ParentID)
			found = true
		}
	}
	require.True(t, found, "expected RefreshLevel{ParentID: __favorites__} in batch cmds")
}

func newAppWithCfgAndMockServer(t *testing.T, cfg *config.Config, handler http.Handler) app.Model {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := api.New(srv.URL)
	client.SetAuth("u1", "tok")
	return app.New(cfg, client, false)
}

func TestPlayItemFetchesItemBeforePlay(t *testing.T) {
	var fetched bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/Items/movie1") {
			fetched = true
			require.NoError(t, json.NewEncoder(w).Encode(api.Item{
				Id:   "movie1",
				Type: "Movie",
				Name: "Dune",
				Chapters: []api.ChapterInfo{
					{StartPositionTicks: 0, Name: "Intro"},
				},
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	cfg := &config.Config{Player: config.PlayerConfig{Command: "mpv"}}
	m := newAppWithCfgAndMockServer(t, cfg, handler)

	_, cmd := m.Update(msg.PlayItem{Item: api.Item{Id: "movie1", Type: "Movie", Name: "Dune"}})
	require.NotNil(t, cmd)

	result := cmd()
	ready, ok := result.(msg.ItemReadyToPlay)
	require.True(t, ok, "expected ItemReadyToPlay, got %T", result)
	require.Equal(t, "movie1", ready.Item.Id)
	require.True(t, fetched, "GetItem was not called")
	require.Len(t, ready.Item.Chapters, 1)
}

func TestPlayItemFetchFailsFallsBackToOriginal(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	cfg := &config.Config{Player: config.PlayerConfig{Command: "mpv"}}
	m := newAppWithCfgAndMockServer(t, cfg, handler)

	_, cmd := m.Update(msg.PlayItem{Item: api.Item{Id: "movie1", Type: "Movie", Name: "Dune"}})
	require.NotNil(t, cmd)

	result := cmd()
	ready, ok := result.(msg.ItemReadyToPlay)
	require.True(t, ok, "expected ItemReadyToPlay fallback on error, got %T", result)
	require.Equal(t, "movie1", ready.Item.Id)
	require.Empty(t, ready.Item.Chapters)
}

// chapterFileOf returns the chapter temp file among the player's temp files, or "".
func chapterFileOf(m app.Model) string {
	for _, p := range m.PlayingTempFiles() {
		if strings.Contains(p, "fin-chapters-") {
			return p
		}
	}
	return ""
}

func cleanupTempFiles(t *testing.T, m app.Model) {
	t.Helper()
	t.Cleanup(func() {
		for _, p := range m.PlayingTempFiles() {
			_ = os.Remove(p)
		}
	})
}

func TestItemReadyToPlayCreatesChapterFileForVideo(t *testing.T) {
	cfg := &config.Config{Player: config.PlayerConfig{Command: "mpv"}}
	m := newAppWithCfgAndMockServer(t, cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	item := api.Item{
		Id:   "movie1",
		Type: "Movie",
		Name: "Dune",
		Chapters: []api.ChapterInfo{
			{StartPositionTicks: 0, Name: "Intro"},
			{StartPositionTicks: 300_000_000, Name: "Act 1"},
		},
	}
	m2, _ := m.Update(msg.ItemReadyToPlay{Item: item})
	am := m2.(app.Model)
	cleanupTempFiles(t, am)

	chapterFile := chapterFileOf(am)
	require.NotEmpty(t, chapterFile, "chapter file expected for video with chapters")
	require.FileExists(t, chapterFile)
}

func TestItemReadyToPlayNoChapterFileWhenNoChapters(t *testing.T) {
	cfg := &config.Config{Player: config.PlayerConfig{Command: "mpv"}}
	m := newAppWithCfgAndMockServer(t, cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	item := api.Item{Id: "movie1", Type: "Movie", Name: "Dune"}
	m2, _ := m.Update(msg.ItemReadyToPlay{Item: item})
	am := m2.(app.Model)
	cleanupTempFiles(t, am)
	require.Empty(t, chapterFileOf(am), "no chapter file expected when item has no chapters")
}

func TestItemReadyToPlayNoChapterFileForAudio(t *testing.T) {
	cfg := &config.Config{Player: config.PlayerConfig{Command: "mpv"}}
	m := newAppWithCfgAndMockServer(t, cfg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	item := api.Item{
		Id:   "audio1",
		Type: "Audio",
		Name: "Track 1",
		Chapters: []api.ChapterInfo{
			{StartPositionTicks: 0, Name: "Part 1"},
		},
	}
	m2, _ := m.Update(msg.ItemReadyToPlay{Item: item})
	am := m2.(app.Model)
	cleanupTempFiles(t, am)
	require.Empty(t, chapterFileOf(am), "no chapter file expected for audio items")
}

func TestPlayItemInjectsIntroChaptersForEpisode(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/Items/ep1") {
			require.NoError(t, json.NewEncoder(w).Encode(api.Item{
				Id: "ep1", Type: "Episode", Name: "Pilot",
				Chapters: []api.ChapterInfo{{StartPositionTicks: 0, Name: "Cold Open"}},
			}))
			return
		}
		if r.URL.Path == "/Episode/ep1/IntroTimestamps" {
			require.NoError(t, json.NewEncoder(w).Encode(api.IntroTimestamps{
				Valid: true, IntroStart: 10.0, IntroEnd: 90.0,
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	cfg := &config.Config{Player: config.PlayerConfig{Command: "mpv"}}
	m := newAppWithCfgAndMockServer(t, cfg, handler)

	_, cmd := m.Update(msg.PlayItem{Item: api.Item{Id: "ep1", Type: "Episode", Name: "Pilot"}})
	result := cmd()
	ready, ok := result.(msg.ItemReadyToPlay)
	require.True(t, ok)
	require.Len(t, ready.Item.Chapters, 3)
	require.Equal(t, "Cold Open", ready.Item.Chapters[0].Name)
	require.Equal(t, "Intro", ready.Item.Chapters[1].Name)
	require.Equal(t, int64(100_000_000), ready.Item.Chapters[1].StartPositionTicks)
	require.Equal(t, "After Intro", ready.Item.Chapters[2].Name)
	require.Equal(t, int64(900_000_000), ready.Item.Chapters[2].StartPositionTicks)
}

func TestPlayItemSkipsIntroTimestampsWhenPluginAbsent(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/Items/ep1") {
			require.NoError(t, json.NewEncoder(w).Encode(api.Item{
				Id: "ep1", Type: "Episode", Name: "Pilot",
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	cfg := &config.Config{Player: config.PlayerConfig{Command: "mpv"}}
	m := newAppWithCfgAndMockServer(t, cfg, handler)

	_, cmd := m.Update(msg.PlayItem{Item: api.Item{Id: "ep1", Type: "Episode", Name: "Pilot"}})
	result := cmd()
	ready, ok := result.(msg.ItemReadyToPlay)
	require.True(t, ok)
	require.Empty(t, ready.Item.Chapters)
}

func TestDoneMsgDeletesTempFiles(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/Items/movie1") {
			require.NoError(t, json.NewEncoder(w).Encode(api.Item{
				Id:       "movie1",
				Type:     "Movie",
				UserData: api.UserData{Played: false},
			}))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	cfg := &config.Config{Player: config.PlayerConfig{Command: "mpv"}}
	m := newAppWithCfgAndMockServer(t, cfg, handler)

	// PlayItem sets m.playingItem
	m2, _ := m.Update(msg.PlayItem{Item: api.Item{Id: "movie1", Type: "Movie", Name: "Dune"}})

	// ItemReadyToPlay writes chapter file and sets m.playingChapterFile
	itemWithChapters := api.Item{
		Id:   "movie1",
		Type: "Movie",
		Name: "Dune",
		Chapters: []api.ChapterInfo{
			{StartPositionTicks: 0, Name: "Intro"},
		},
	}
	m3, _ := m2.(app.Model).Update(msg.ItemReadyToPlay{Item: itemWithChapters})
	temps := m3.(app.Model).PlayingTempFiles()
	require.NotEmpty(t, temps, "playlist and chapter file expected")
	for _, p := range temps {
		require.FileExists(t, p, "temp file should exist before DoneMsg")
	}

	// DoneMsg deletes every temp file the player was reading.
	m4, _ := m3.(app.Model).Update(player.DoneMsg{})
	require.Empty(t, m4.(app.Model).PlayingTempFiles())
	for _, p := range temps {
		require.NoFileExists(t, p, "temp file should be deleted after DoneMsg")
	}
}

func TestSearchOverlayKeepsSingleKeyShortcuts(t *testing.T) {
	m := app.New(nil, nil, false)
	m2, _ := m.Update(msg.LoginSuccess{ServerURL: "http://jf", UserID: "u1", AccessToken: "tok"})
	m3, _ := m2.(app.Model).Update(msg.OpenSearch{})

	for _, k := range []string{"q", "?"} {
		_, cmd := m3.(app.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)})
		if cmd == nil {
			continue
		}
		_, isQuit := cmd().(tea.QuitMsg)
		require.False(t, isQuit, "typing %q in the search box must not quit", k)
	}
}

func TestLoginScreenKeepsSingleKeyShortcuts(t *testing.T) {
	m := app.New(nil, nil, false)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd != nil {
		_, isQuit := cmd().(tea.QuitMsg)
		require.False(t, isQuit, "typing \"q\" into the login form must not quit")
	}
}

func TestQuitsFromBrowser(t *testing.T) {
	m := app.New(nil, nil, false)
	m2, _ := m.Update(msg.LoginSuccess{ServerURL: "http://jf", UserID: "u1", AccessToken: "tok"})
	_, cmd := m2.(app.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	require.NotNil(t, cmd)
	_, isQuit := cmd().(tea.QuitMsg)
	require.True(t, isQuit)
}

func TestErrorBannerDoesNotAddARow(t *testing.T) {
	m := app.New(nil, nil, false)
	m2, _ := m.Update(msg.LoginSuccess{ServerURL: "http://jf", UserID: "u1", AccessToken: "tok"})
	m3, _ := m2.(app.Model).Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m4, _ := m3.(app.Model).Update(msg.PushLevel{Items: makeMovies(3), LevelName: "L", ParentID: "p"})

	clean := m4.(app.Model).View()
	m5, _ := m4.(app.Model).Update(msg.AppError{Err: fmt.Errorf("network timeout")})
	withError := m5.(app.Model).View()

	require.Contains(t, withError, "network timeout")
	require.Equal(t, strings.Count(clean, "\n"), strings.Count(withError, "\n"),
		"error banner must replace the last row, not push the view past the screen")
}

func makeMovies(n int) []api.Item {
	out := make([]api.Item, n)
	for i := range out {
		out[i] = api.Item{Id: fmt.Sprintf("m%d", i), Name: fmt.Sprintf("Movie %d", i), Type: "Movie"}
	}
	return out
}
