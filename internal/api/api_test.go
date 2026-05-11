package api_test

import (
	"encoding/json"
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
		json.NewEncoder(w).Encode(api.AuthResponse{
			User:        api.UserInfo{Id: "uid1", Name: "alice"},
			AccessToken: "tok123",
		})
	}))
	resp, err := client.Authenticate("alice", "password")
	require.NoError(t, err)
	require.Equal(t, "tok123", resp.AccessToken)
}

func TestGetLibraries(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.LibraryResponse{
			Items: []api.Library{{Id: "lib1", Name: "Movies", CollectionType: "movies"}},
		})
	}))
	client.SetAuth("uid1", "tok123")
	libs, err := client.GetLibraries()
	require.NoError(t, err)
	require.Len(t, libs, 1)
	require.Equal(t, "Movies", libs[0].Name)
}

func TestGetItems(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "lib1", r.URL.Query().Get("ParentId"))
		json.NewEncoder(w).Encode(api.ItemsResponse{
			Items: []api.Item{{Id: "m1", Name: "Dune", Type: "Movie"}},
		})
	}))
	client.SetAuth("uid1", "tok123")
	items, err := client.GetItems("lib1", nil)
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func TestSearch(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "dune", r.URL.Query().Get("searchTerm"))
		json.NewEncoder(w).Encode(api.ItemsResponse{
			Items: []api.Item{{Id: "m1", Name: "Dune"}},
		})
	}))
	client.SetAuth("uid1", "tok123")
	items, err := client.Search("dune")
	require.NoError(t, err)
	require.Len(t, items, 1)
}

func TestStreamURL(t *testing.T) {
	client := api.New("https://jf.example.com")
	client.SetAuth("uid1", "tok")
	item := api.Item{Id: "m1", Type: "Movie"}
	require.Equal(t, "https://jf.example.com/Videos/m1/stream?api_key=tok&static=true", client.StreamURL(item))
}
