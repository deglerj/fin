// internal/api/api_test.go
package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/deglerj/fin/internal/api"
)

func newTestServer(t *testing.T, handler http.Handler) (*httptest.Server, *api.Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := api.New(srv.URL)
	return srv, c
}

func TestAuthenticate(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Users/AuthenticateByName" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(api.AuthResponse{
			User:        api.UserInfo{Id: "uid1", Name: "alice"},
			AccessToken: "tok123",
		})
	}))
	resp, err := client.Authenticate("alice", "password")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if resp.AccessToken != "tok123" {
		t.Errorf("expected tok123, got %q", resp.AccessToken)
	}
}

func TestGetLibraries(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(api.LibraryResponse{
			Items: []api.Library{{Id: "lib1", Name: "Movies", CollectionType: "movies"}},
		})
	}))
	client.SetAuth("uid1", "tok123")
	libs, err := client.GetLibraries()
	if err != nil {
		t.Fatalf("GetLibraries: %v", err)
	}
	if len(libs) != 1 || libs[0].Name != "Movies" {
		t.Errorf("unexpected libraries: %+v", libs)
	}
}

func TestGetItems(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ParentId") != "lib1" {
			t.Errorf("expected ParentId=lib1, got %q", r.URL.Query().Get("ParentId"))
		}
		json.NewEncoder(w).Encode(api.ItemsResponse{
			Items: []api.Item{{Id: "m1", Name: "Dune", Type: "Movie"}},
		})
	}))
	client.SetAuth("uid1", "tok123")
	items, err := client.GetItems("lib1", nil)
	if err != nil {
		t.Fatalf("GetItems: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestSearch(t *testing.T) {
	_, client := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("searchTerm") != "dune" {
			t.Errorf("expected searchTerm=dune, got %q", r.URL.Query().Get("searchTerm"))
		}
		json.NewEncoder(w).Encode(api.ItemsResponse{
			Items: []api.Item{{Id: "m1", Name: "Dune"}},
		})
	}))
	client.SetAuth("uid1", "tok123")
	items, err := client.Search("dune")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 result")
	}
}

func TestStreamURL(t *testing.T) {
	client := api.New("https://jf.example.com")
	client.SetAuth("uid1", "tok")
	item := api.Item{Id: "m1", Type: "Movie"}
	url := client.StreamURL(item)
	expected := "https://jf.example.com/Videos/m1/stream?api_key=tok&static=true"
	if url != expected {
		t.Errorf("got %q", url)
	}
}
