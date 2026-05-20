package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deglerj/fin/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, handler http.Handler) (*httptest.Server, *api.Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, api.New(srv.URL)
}

func TestAuthenticate(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/Users/AuthenticateByName", r.URL.Path)
		assert.NoError(t, json.NewEncoder(w).Encode(api.AuthResponse{
			User:        api.UserInfo{Id: "uid1", Name: "alice"},
			AccessToken: "tok123",
		}))
	}))
	resp, err := client.Authenticate(context.Background(), "alice", "password")
	require.NoError(t, err)
	require.Equal(t, "tok123", resp.AccessToken)
}

func TestGetLibraries(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewEncoder(w).Encode(api.LibraryResponse{
			Items: []api.Library{{Id: "lib1", Name: "Movies", CollectionType: "movies"}},
		}))
	}))
	client.SetAuth("uid1", "tok123")
	libs, err := client.GetLibraries(context.Background())
	require.NoError(t, err)
	require.Len(t, libs, 1)
	require.Equal(t, "Movies", libs[0].Name)
}

func TestGetItems(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "lib1", r.URL.Query().Get("ParentId"))
		assert.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
			Items: []api.Item{{Id: "m1", Name: "Dune", Type: "Movie"}},
		}))
	}))
	client.SetAuth("uid1", "tok123")
	items, err := client.GetItems(context.Background(), "lib1", nil)
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func TestGetItemsPagination(t *testing.T) {
	var capturedStartIndexes []string
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		si := r.URL.Query().Get("StartIndex")
		capturedStartIndexes = append(capturedStartIndexes, si)
		var items []api.Item
		if si == "0" {
			items = make([]api.Item, 500)
			for i := range items {
				items[i] = api.Item{Id: fmt.Sprintf("id%d", i), Name: fmt.Sprintf("Item %d", i), Type: "Movie"}
			}
		} else {
			items = []api.Item{{Id: "id500", Name: "Item 500", Type: "Movie"}}
		}
		assert.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{Items: items, TotalRecordCount: 501}))
	}))
	client.SetAuth("uid1", "tok123")
	items, err := client.GetItems(context.Background(), "lib1", nil)
	require.NoError(t, err)
	require.Len(t, items, 501)
	require.Equal(t, []string{"0", "500"}, capturedStartIndexes)
}

func TestSearch(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "dune", r.URL.Query().Get("searchTerm"))
		assert.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
			Items: []api.Item{{Id: "m1", Name: "Dune"}},
		}))
	}))
	client.SetAuth("uid1", "tok123")
	items, err := client.Search(context.Background(), "dune")
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func TestSearchSpecialCharsURLEncoded(t *testing.T) {
	var captured string
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.URL.Query().Get("searchTerm")
		assert.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{}))
	}))
	client.SetAuth("uid1", "tok123")
	_, err := client.Search(context.Background(), "breaking&bad")
	require.NoError(t, err)
	require.Equal(t, "breaking&bad", captured)
}

func TestStreamURL(t *testing.T) {
	client := api.New("https://jf.example.com")
	client.SetAuth("uid1", "tok")
	item := api.Item{Id: "m1", Type: "Movie"}
	require.Equal(t, "https://jf.example.com/Videos/m1/stream?api_key=tok&static=true", client.StreamURL(item))
}

func TestGetNextUp(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/Shows/NextUp", r.URL.Path)
		assert.Equal(t, "uid1", r.URL.Query().Get("UserId"))
		assert.Equal(t, "20", r.URL.Query().Get("Limit"))
		assert.Equal(t, "Overview,People,CommunityRating,ProductionYear", r.URL.Query().Get("Fields"))
		assert.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
			Items: []api.Item{{Id: "ep1", Name: "Episode 1", Type: "Episode"}},
		}))
	}))
	client.SetAuth("uid1", "tok123")
	items, err := client.GetNextUp(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "Episode 1", items[0].Name)
}

func TestGetResumeItems(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/UserItems/Resume", r.URL.Path)
		assert.Equal(t, "uid1", r.URL.Query().Get("UserId"))
		assert.Equal(t, "20", r.URL.Query().Get("Limit"))
		assert.Equal(t, "Overview,People,CommunityRating,ProductionYear", r.URL.Query().Get("Fields"))
		assert.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
			Items: []api.Item{{Id: "m1", Name: "Inception", Type: "Movie"}},
		}))
	}))
	client.SetAuth("uid1", "tok123")
	items, err := client.GetResumeItems(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "Inception", items[0].Name)
}

func TestGetLatestMedia(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/Users/uid1/Items/Latest", r.URL.Path)
		assert.Equal(t, "20", r.URL.Query().Get("Limit"))
		assert.Equal(t, "Overview,People,CommunityRating,ProductionYear", r.URL.Query().Get("Fields"))
		assert.NoError(t, json.NewEncoder(w).Encode([]api.Item{
			{Id: "m2", Name: "Dune", Type: "Movie"},
		}))
	}))
	client.SetAuth("uid1", "tok123")
	items, err := client.GetLatestMedia(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "Dune", items[0].Name)
}

func TestGetFavorites(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/Items", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("isFavorite"))
		assert.Equal(t, "true", r.URL.Query().Get("Recursive"))
		assert.Equal(t, "uid1", r.URL.Query().Get("UserId"))
		assert.Equal(t, "20", r.URL.Query().Get("Limit"))
		assert.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
			Items: []api.Item{{Id: "m3", Name: "The Matrix", Type: "Movie"}},
		}))
	}))
	client.SetAuth("uid1", "tok123")
	items, err := client.GetFavorites(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "The Matrix", items[0].Name)
}

func TestMarkPlayed(t *testing.T) {
	var gotMethod, gotPath string
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	client.SetAuth("uid1", "tok123")
	err := client.MarkPlayed(context.Background(), "item42")
	require.NoError(t, err)
	require.Equal(t, "POST", gotMethod)
	require.Equal(t, "/Users/uid1/PlayedItems/item42", gotPath)
}

func TestMarkUnplayed(t *testing.T) {
	var gotMethod, gotPath string
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	client.SetAuth("uid1", "tok123")
	err := client.MarkUnplayed(context.Background(), "item42")
	require.NoError(t, err)
	require.Equal(t, "DELETE", gotMethod)
	require.Equal(t, "/Users/uid1/PlayedItems/item42", gotPath)
}

func TestMarkPlayedHTTPError(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	client.SetAuth("uid1", "tok123")
	err := client.MarkPlayed(context.Background(), "item42")
	require.Error(t, err)
	require.Contains(t, err.Error(), "500")
}

func TestReportPlaybackStart(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody api.PlaybackReport
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	client.SetAuth("uid1", "tok123")
	report := api.PlaybackReport{
		ItemId:        "item1",
		PlaySessionId: "sess1",
		MediaSourceId: "item1",
		PositionTicks: 0,
		CanSeek:       true,
		PlayMethod:    "DirectStream",
		RepeatMode:    "RepeatNone",
	}
	err := client.ReportPlaybackStart(context.Background(), report)
	require.NoError(t, err)
	require.Equal(t, "POST", gotMethod)
	require.Equal(t, "/Sessions/Playing", gotPath)
	require.Equal(t, "item1", gotBody.ItemId)
	require.Equal(t, "sess1", gotBody.PlaySessionId)
	require.True(t, gotBody.CanSeek)
}

func TestReportPlaybackProgress(t *testing.T) {
	var gotPath string
	var gotBody api.PlaybackReport
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	client.SetAuth("uid1", "tok123")
	report := api.PlaybackReport{
		ItemId:        "item1",
		PlaySessionId: "sess1",
		PositionTicks: 100_000_000,
	}
	err := client.ReportPlaybackProgress(context.Background(), report)
	require.NoError(t, err)
	require.Equal(t, "/Sessions/Playing/Progress", gotPath)
	require.Equal(t, int64(100_000_000), gotBody.PositionTicks)
}

func TestReportPlaybackStopped(t *testing.T) {
	var gotPath string
	var gotBody api.PlaybackReport
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	client.SetAuth("uid1", "tok123")
	report := api.PlaybackReport{
		ItemId:        "item1",
		PlaySessionId: "sess1",
		PositionTicks: 900_000_000,
	}
	err := client.ReportPlaybackStopped(context.Background(), report)
	require.NoError(t, err)
	require.Equal(t, "/Sessions/Playing/Stopped", gotPath)
	require.Equal(t, int64(900_000_000), gotBody.PositionTicks)
}

func TestReportPlaybackHTTPError(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	client.SetAuth("uid1", "tok123")
	err := client.ReportPlaybackStart(context.Background(), api.PlaybackReport{ItemId: "item1"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "500")
}

func TestGetRandomLeaf(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/Users/uid1/Items", r.URL.Path)
		assert.Equal(t, "series1", r.URL.Query().Get("ParentId"))
		assert.Equal(t, "true", r.URL.Query().Get("Recursive"))
		assert.Equal(t, "Random", r.URL.Query().Get("SortBy"))
		assert.Equal(t, "1", r.URL.Query().Get("Limit"))
		assert.Equal(t, "Movie,Episode,Audio", r.URL.Query().Get("IncludeItemTypes"))
		assert.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{
			Items: []api.Item{{Id: "ep42", Name: "The One Where It Works", Type: "Episode"}},
		}))
	}))
	client.SetAuth("uid1", "tok123")
	item, err := client.GetRandomLeaf(context.Background(), "series1")
	require.NoError(t, err)
	require.Equal(t, "ep42", item.Id)
}

func TestGetRandomLeafEmptyResponseReturnsError(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewEncoder(w).Encode(api.ItemsResponse{}))
	}))
	client.SetAuth("uid1", "tok123")
	_, err := client.GetRandomLeaf(context.Background(), "series1")
	require.Error(t, err)
}

func TestGetItemUnmarshalsChapters(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewEncoder(w).Encode(api.Item{
			Id:   "m1",
			Name: "Dune",
			Type: "Movie",
			Chapters: []api.ChapterInfo{
				{StartPositionTicks: 0, Name: "Intro"},
				{StartPositionTicks: 50_000_000, Name: "Chapter 1"},
			},
		}))
	}))
	client.SetAuth("uid1", "tok123")
	item, err := client.GetItem(context.Background(), "m1")
	require.NoError(t, err)
	require.Len(t, item.Chapters, 2)
	require.Equal(t, "Intro", item.Chapters[0].Name)
	require.Equal(t, int64(0), item.Chapters[0].StartPositionTicks)
	require.Equal(t, "Chapter 1", item.Chapters[1].Name)
	require.Equal(t, int64(50_000_000), item.Chapters[1].StartPositionTicks)
}
