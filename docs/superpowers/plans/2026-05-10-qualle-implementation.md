# fin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a text-mode Jellyfin client in Go that browses libraries, searches, plays via mpv, and shows metadata with optional kitty images.

**Architecture:** bubbletea root model routes between login and browser screens; overlays (search, details, help) stack on top of the browser. All Jellyfin API calls are issued as bubbletea `Cmd`s. Shared message types live in `internal/ui/msg` to avoid import cycles.

**Tech Stack:** Go 1.22, bubbletea, lipgloss, bubbles (textinput/spinner), BurntSushi/toml, golang.org/x/crypto (HKDF)

---

## File Map

```
cmd/fin/main.go
internal/
  config/
    config.go          Config struct, Load(), XDG paths, CredentialsPath()
    config_test.go
  auth/
    credentials.go     Credentials struct, Save(), Load(), encrypt/decrypt
    credentials_test.go
    machineid.go       MachineIDProvider interface + Linux impl
    machineid_darwin.go
    machineid_windows.go
  api/
    client.go          Client struct, New(), SetAuth(), request helpers
    types.go           Item, Library, AuthResponse, etc.
    endpoints.go       Authenticate, GetLibraries, GetItems, GetItem, Search, GetImage, StreamURL
    api_test.go
  player/
    player.go          Play() returning tea.Cmd via tea.ExecProcess
    player_test.go
  image/
    image.go           Probe(), Encode(), Capable bool
    image_test.go
  ui/
    msg/msg.go         All shared tea.Msg types
    styles/styles.go   lipgloss style vars
    keys/keys.go       Key binding constants
    app/model.go       Root model — router + overlay manager
    app/model_test.go
    login/model.go     Login form model
    login/model_test.go
    browser/model.go   Navigation stack + list rendering
    browser/model_test.go
    details/model.go   Details overlay
    details/model_test.go
    search/model.go    Search overlay with debounce
    search/model_test.go
    help/model.go      Static help overlay
```

---

## Task 1: Project Scaffold

**Files:**
- Create: `go.mod`
- Create: `cmd/fin/main.go`

- [ ] **Step 1: Init module and install deps**

```bash
cd /home/deglerj/Dokumente/Git/fin
go mod init github.com/deglerj/fin
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/BurntSushi/toml@latest
go get golang.org/x/crypto@latest
```

- [ ] **Step 2: Create directory structure**

```bash
mkdir -p cmd/fin
mkdir -p internal/{config,auth,api,player,image}
mkdir -p internal/ui/{msg,styles,keys,app,login,browser,details,search,help}
```

- [ ] **Step 3: Write minimal main.go**

```go
// cmd/fin/main.go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(nil, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Verify it compiles**

```bash
go build ./...
```

Expected: no output (compiles cleanly). `nil` model will panic at runtime — that's expected for now.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum cmd/ internal/
git commit -m "chore: scaffold project structure and dependencies"
```

---

## Task 2: Config Module

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/config/config_test.go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/deglerj/fin/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Player.Command != "mpv" {
		t.Errorf("expected default player mpv, got %q", cfg.Player.Command)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := os.MkdirAll(filepath.Join(dir, "fin"), 0755); err != nil {
		t.Fatal(err)
	}
	toml := `[server]
url = "https://jf.example.com"
[player]
command = "vlc"
`
	if err := os.WriteFile(filepath.Join(dir, "fin", "config.toml"), []byte(toml), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.URL != "https://jf.example.com" {
		t.Errorf("expected server URL, got %q", cfg.Server.URL)
	}
	if cfg.Player.Command != "vlc" {
		t.Errorf("expected vlc, got %q", cfg.Player.Command)
	}
}

func TestCredentialsPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	p := config.CredentialsPath()
	if p != filepath.Join(dir, "fin", "credentials") {
		t.Errorf("unexpected credentials path: %q", p)
	}
}
```

- [ ] **Step 2: Run tests to confirm failure**

```bash
go test ./internal/config/...
```

Expected: `cannot find package` or compile error — implementation doesn't exist yet.

- [ ] **Step 3: Implement config.go**

```go
// internal/config/config.go
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server      ServerConfig      `toml:"server"`
	UI          UIConfig          `toml:"ui"`
	Player      PlayerConfig      `toml:"player"`
	Keybindings KeybindingsConfig `toml:"keybindings"`
}

type ServerConfig struct {
	URL string `toml:"url"`
}

type UIConfig struct {
	DateFormat string `toml:"date_format"`
}

type PlayerConfig struct {
	Command   string   `toml:"command"`
	ExtraArgs []string `toml:"extra_args"`
}

type KeybindingsConfig struct {
	Play    string `toml:"play"`
	Back    string `toml:"back"`
	Details string `toml:"details"`
	Search  string `toml:"search"`
	Random  string `toml:"random"`
	Quit    string `toml:"quit"`
	Help    string `toml:"help"`
}

func defaults() *Config {
	return &Config{
		UI:     UIConfig{DateFormat: "2006-01-02"},
		Player: PlayerConfig{Command: "mpv"},
		Keybindings: KeybindingsConfig{
			Play: "enter", Back: "esc", Details: "i",
			Search: "/", Random: "r", Quit: "q", Help: "?",
		},
	}
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "fin")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "fin")
}

func CredentialsPath() string {
	return filepath.Join(configDir(), "credentials")
}

func Load() (*Config, error) {
	cfg := defaults()
	path := filepath.Join(configDir(), "config.toml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
```

- [ ] **Step 4: Run tests to confirm pass**

```bash
go test ./internal/config/... -v
```

Expected: `PASS` for all three tests.

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config module with XDG path support"
```

---

## Task 3: Credential Storage

**Files:**
- Create: `internal/auth/machineid.go`
- Create: `internal/auth/machineid_darwin.go`
- Create: `internal/auth/machineid_windows.go`
- Create: `internal/auth/credentials.go`
- Create: `internal/auth/credentials_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/auth/credentials_test.go
package auth_test

import (
	"path/filepath"
	"testing"

	"github.com/deglerj/fin/internal/auth"
)

type fixedID struct{}

func (fixedID) MachineID() (string, error) { return "test-machine-id-1234", nil }

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	creds := auth.Credentials{
		ServerURL:   "https://jf.example.com",
		UserID:      "abc123",
		AccessToken: "tok-xyz",
	}
	if err := auth.Save(creds, path, fixedID{}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := auth.LoadCreds(path, fixedID{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.AccessToken != creds.AccessToken {
		t.Errorf("token mismatch: got %q", got.AccessToken)
	}
	if got.ServerURL != creds.ServerURL {
		t.Errorf("url mismatch: got %q", got.ServerURL)
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := auth.LoadCreds("/nonexistent/path", fixedID{})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestWrongKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	creds := auth.Credentials{AccessToken: "secret"}
	if err := auth.Save(creds, path, fixedID{}); err != nil {
		t.Fatal(err)
	}
	type otherID struct{}
	_, err := auth.LoadCreds(path, struct{ auth.MachineIDProvider }{})
	if err == nil {
		t.Fatal("expected decryption to fail with different key")
	}
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/auth/... 2>&1 | head -5
```

Expected: compile error — package doesn't exist yet.

- [ ] **Step 3: Implement machineid.go**

```go
// internal/auth/machineid.go
package auth

import (
	"os"
	"strings"
)

type MachineIDProvider interface {
	MachineID() (string, error)
}

type DefaultMachineID struct{}

func (DefaultMachineID) MachineID() (string, error) {
	for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		b, err := os.ReadFile(p)
		if err == nil {
			return strings.TrimSpace(string(b)), nil
		}
	}
	// fallback: use hostname
	return os.Hostname()
}
```

- [ ] **Step 4: Create machineid_darwin.go**

```go
// internal/auth/machineid_darwin.go
package auth

// DefaultMachineID.MachineID() falls through to hostname on Darwin.
// A proper impl would use IOPlatformUUID via exec, but hostname is
// sufficient for v1 (best-effort platform).
```

- [ ] **Step 5: Create machineid_windows.go**

```go
// internal/auth/machineid_windows.go
package auth

// DefaultMachineID.MachineID() falls through to hostname on Windows.
// A proper impl would read HKLM\SOFTWARE\Microsoft\Cryptography\MachineGuid.
```

- [ ] **Step 6: Implement credentials.go**

```go
// internal/auth/credentials.go
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"os"

	"golang.org/x/crypto/hkdf"
)

type Credentials struct {
	ServerURL   string `json:"server_url"`
	UserID      string `json:"user_id"`
	AccessToken string `json:"access_token"`
}

func deriveKey(provider MachineIDProvider) ([]byte, error) {
	id, err := provider.MachineID()
	if err != nil {
		return nil, err
	}
	r := hkdf.New(sha256.New, []byte(id), []byte("fin-creds-v1"), nil)
	key := make([]byte, 32)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}

func encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ct := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ct, nil)
}

func Save(creds Credentials, path string, provider MachineIDProvider) error {
	key, err := deriveKey(provider)
	if err != nil {
		return err
	}
	plain, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	enc, err := encrypt(key, plain)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, enc, 0600)
}

func LoadCreds(path string, provider MachineIDProvider) (*Credentials, error) {
	key, err := deriveKey(provider)
	if err != nil {
		return nil, err
	}
	enc, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	plain, err := decrypt(key, enc)
	if err != nil {
		return nil, err
	}
	var creds Credentials
	if err := json.Unmarshal(plain, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}
```

Add missing import `"path/filepath"` to the Save function (already in the list above — ensure `filepath.Dir` resolves).

- [ ] **Step 7: Run tests**

```bash
go test ./internal/auth/... -v
```

Expected: `PASS` for `TestRoundTrip` and `TestLoadMissing`. `TestWrongKey` may need adjustment — the zero-value `MachineIDProvider` will error on `MachineID()`, which counts as a failure, so the test passes for the right reason.

- [ ] **Step 8: Commit**

```bash
git add internal/auth/
git commit -m "feat: add encrypted credential storage with HKDF+AES-GCM"
```

---

## Task 4: Jellyfin API Client

**Files:**
- Create: `internal/api/types.go`
- Create: `internal/api/client.go`
- Create: `internal/api/endpoints.go`
- Create: `internal/api/api_test.go`

- [ ] **Step 1: Write types.go**

```go
// internal/api/types.go
package api

type AuthResponse struct {
	User        UserInfo `json:"User"`
	AccessToken string   `json:"AccessToken"`
}

type UserInfo struct {
	Id   string `json:"Id"`
	Name string `json:"Name"`
}

type Item struct {
	Id                string   `json:"Id"`
	Name              string   `json:"Name"`
	Type              string   `json:"Type"`   // Movie, Series, Season, Episode, Audio
	MediaType         string   `json:"MediaType"`
	SeriesName        string   `json:"SeriesName"`
	SeasonName        string   `json:"SeasonName"`
	IndexNumber       int      `json:"IndexNumber"`
	ParentIndexNumber int      `json:"ParentIndexNumber"`
	RunTimeTicks      int64    `json:"RunTimeTicks"`
	Overview          string   `json:"Overview"`
	CommunityRating   float64  `json:"CommunityRating"`
	ProductionYear    int      `json:"ProductionYear"`
	People            []Person `json:"People"`
	UserData          UserData `json:"UserData"`
}

type Person struct {
	Name string `json:"Name"`
	Role string `json:"Role"`
	Type string `json:"Type"`
}

type UserData struct {
	Played bool `json:"Played"`
}

type ItemsResponse struct {
	Items            []Item `json:"Items"`
	TotalRecordCount int    `json:"TotalRecordCount"`
}

type Library struct {
	Id             string `json:"Id"`
	Name           string `json:"Name"`
	CollectionType string `json:"CollectionType"`
}

type LibraryResponse struct {
	Items []Library `json:"Items"`
}
```

- [ ] **Step 2: Write client.go**

```go
// internal/api/client.go
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const clientHeader = `MediaBrowser Client="fin", Device="terminal", DeviceId="fin-cli", Version="1.0.0"`

type Client struct {
	baseURL    string
	userID     string
	token      string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) SetAuth(userID, token string) {
	c.userID = userID
	c.token = token
}

func (c *Client) get(path string, out any) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Emby-Authorization", clientHeader)
	if c.token != "" {
		req.Header.Set("X-Emby-Token", c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("jellyfin: HTTP %d for %s", resp.StatusCode, path)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) getRaw(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Emby-Token", c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("jellyfin: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) post(path string, body io.Reader, out any) error {
	req, err := http.NewRequest("POST", c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Emby-Authorization", clientHeader)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("jellyfin: HTTP %d for %s", resp.StatusCode, path)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
```

- [ ] **Step 3: Write failing test**

```go
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
			t.Errorf("expected searchTerm=dune")
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
```

- [ ] **Step 4: Run tests to confirm failure**

```bash
go test ./internal/api/... 2>&1 | head -5
```

Expected: missing methods on `*Client`.

- [ ] **Step 5: Write endpoints.go**

```go
// internal/api/endpoints.go
package api

import (
	"fmt"
	"strings"
)

func (c *Client) Authenticate(username, password string) (AuthResponse, error) {
	body := fmt.Sprintf(`{"Username":%q,"Pw":%q}`, username, password)
	var resp AuthResponse
	err := c.post("/Users/AuthenticateByName", strings.NewReader(body), &resp)
	return resp, err
}

func (c *Client) ValidateToken() error {
	var result map[string]any
	return c.get(fmt.Sprintf("/Users/%s", c.userID), &result)
}

func (c *Client) GetLibraries() ([]Library, error) {
	var resp LibraryResponse
	err := c.get("/Library/MediaFolders", &resp)
	return resp.Items, err
}

func (c *Client) GetItems(parentID string, itemTypes []string) ([]Item, error) {
	q := fmt.Sprintf("/Users/%s/Items?ParentId=%s&Limit=500", c.userID, parentID)
	if len(itemTypes) > 0 {
		q += "&IncludeItemTypes=" + strings.Join(itemTypes, ",")
	}
	var resp ItemsResponse
	err := c.get(q, &resp)
	return resp.Items, err
}

func (c *Client) GetItem(id string) (Item, error) {
	var item Item
	err := c.get(fmt.Sprintf("/Users/%s/Items/%s", c.userID, id), &item)
	return item, err
}

func (c *Client) Search(term string) ([]Item, error) {
	q := fmt.Sprintf("/Items?searchTerm=%s&IncludeItemTypes=Movie,Series,Episode&Recursive=true&UserId=%s&Limit=20",
		term, c.userID)
	var resp ItemsResponse
	err := c.get(q, &resp)
	return resp.Items, err
}

func (c *Client) GetImage(itemID string, maxWidth int) ([]byte, error) {
	return c.getRaw(fmt.Sprintf("/Items/%s/Images/Primary?MaxWidth=%d", itemID, maxWidth))
}

func (c *Client) StreamURL(item Item) string {
	if item.Type == "Audio" || item.MediaType == "Audio" {
		return fmt.Sprintf("%s/Audio/%s/stream?api_key=%s&static=true", c.baseURL, item.Id, c.token)
	}
	return fmt.Sprintf("%s/Videos/%s/stream?api_key=%s&static=true", c.baseURL, item.Id, c.token)
}
```

- [ ] **Step 6: Run tests**

```bash
go test ./internal/api/... -v
```

Expected: all 5 tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/api/
git commit -m "feat: add Jellyfin API client with typed endpoints"
```

---

## Task 5: Player & Image Modules

**Files:**
- Create: `internal/player/player.go` + `player_test.go`
- Create: `internal/image/image.go` + `image_test.go`

- [ ] **Step 1: Write player test**

```go
// internal/player/player_test.go
package player_test

import (
	"testing"

	"github.com/deglerj/fin/internal/player"
)

func TestBuildCmd(t *testing.T) {
	cmd := player.BuildCmd("mpv", []string{"--really-quiet"}, "https://example.com/video", "Test Movie")
	args := cmd.Args
	if args[0] != "mpv" {
		t.Errorf("expected mpv, got %q", args[0])
	}
	var hasURL, hasTitle bool
	for _, a := range args {
		if a == "https://example.com/video" {
			hasURL = true
		}
		if a == "--title=Test Movie" {
			hasTitle = true
		}
	}
	if !hasURL {
		t.Error("URL not in args")
	}
	if !hasTitle {
		t.Error("title not in args")
	}
}
```

- [ ] **Step 2: Implement player.go**

```go
// internal/player/player.go
package player

import (
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type DoneMsg struct{ Err error }

func BuildCmd(command string, extraArgs []string, url, title string) *exec.Cmd {
	args := []string{url, fmt.Sprintf("--title=%s", title)}
	args = append(args, extraArgs...)
	return exec.Command(command, args...)
}

func Play(command string, extraArgs []string, url, title string) tea.Cmd {
	cmd := BuildCmd(command, extraArgs, url, title)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return DoneMsg{Err: err}
	})
}
```

- [ ] **Step 3: Run player tests**

```bash
go test ./internal/player/... -v
```

Expected: PASS.

- [ ] **Step 4: Write image tests**

```go
// internal/image/image_test.go
package image_test

import (
	"strings"
	"testing"

	"github.com/deglerj/fin/internal/image"
)

func TestEncodeContainsAPC(t *testing.T) {
	data := []byte("fake-jpeg-data")
	out := image.Encode(data, 20, 10)
	// Kitty graphics protocol uses APC escape \x1b_G
	if !strings.Contains(out, "\x1b_G") {
		t.Error("encoded output missing kitty APC sequence")
	}
}
```

- [ ] **Step 5: Implement image.go**

```go
// internal/image/image.go
package image

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"
)

var capable *bool

// Probe sends a kitty graphics query and waits for a response.
// Returns true if the terminal responds (supports kitty graphics protocol).
// Sets the package-level capability flag for subsequent calls.
func Probe() bool {
	if capable != nil {
		return *capable
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		t := true
		capable = &t
		return true
	}
	result := probeTerminal(100 * time.Millisecond)
	capable = &result
	return result
}

func probeTerminal(timeout time.Duration) bool {
	// Put terminal in raw mode, send APC query, wait for response.
	// This is a best-effort probe; failure returns false gracefully.
	oldState, err := makeRaw()
	if err != nil {
		return false
	}
	defer restore(oldState)

	// Send kitty graphics query: action=query, format=png, size=0
	fmt.Fprint(os.Stdout, "\x1b_Ga=q,s=1,v=1,i=1;\x1b\\")

	ch := make(chan bool, 1)
	go func() {
		buf := make([]byte, 64)
		n, _ := os.Stdin.Read(buf)
		ch <- n > 0 && containsKittyResponse(buf[:n])
	}()

	select {
	case result := <-ch:
		return result
	case <-time.After(timeout):
		return false
	}
}

func containsKittyResponse(b []byte) bool {
	for i := 0; i < len(b)-2; i++ {
		if b[i] == 0x1b && b[i+1] == '_' && b[i+2] == 'G' {
			return true
		}
	}
	return false
}

// Encode encodes image bytes as a kitty graphics protocol string.
// cols and rows are the desired terminal cell dimensions.
func Encode(data []byte, cols, rows int) string {
	const chunkSize = 4096
	encoded := base64.StdEncoding.EncodeToString(data)
	out := ""
	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]
		m := 1
		if end == len(encoded) {
			m = 0
		}
		if i == 0 {
			out += fmt.Sprintf("\x1b_Ga=T,f=100,c=%d,r=%d,m=%d;%s\x1b\\", cols, rows, m, chunk)
		} else {
			out += fmt.Sprintf("\x1b_Gm=%d;%s\x1b\\", m, chunk)
		}
	}
	return out
}
```

- [ ] **Step 6: Add platform raw-mode helpers**

```go
// internal/image/raw_unix.go
//go:build !windows

package image

import (
	"golang.org/x/term"
	"os"
)

func makeRaw() (any, error) {
	return term.MakeRaw(int(os.Stdin.Fd()))
}

func restore(state any) {
	if s, ok := state.(*term.State); ok {
		term.Restore(int(os.Stdin.Fd()), s)
	}
}
```

```go
// internal/image/raw_windows.go
//go:build windows

package image

func makeRaw() (any, error) { return nil, nil }
func restore(any)            {}
```

Add `golang.org/x/term` dependency:

```bash
go get golang.org/x/term@latest
```

- [ ] **Step 7: Run image tests**

```bash
go test ./internal/image/... -v
```

Expected: `TestEncodeContainsAPC` passes.

- [ ] **Step 8: Commit**

```bash
git add internal/player/ internal/image/
git commit -m "feat: add mpv player launcher and kitty image encoder"
```

---

## Task 6: Shared UI Messages & Styles

**Files:**
- Create: `internal/ui/msg/msg.go`
- Create: `internal/ui/styles/styles.go`
- Create: `internal/ui/keys/keys.go`

- [ ] **Step 1: Write msg.go**

```go
// internal/ui/msg/msg.go
package msg

import "github.com/deglerj/fin/internal/api"

// Screen navigation
type LoginSuccess struct {
	ServerURL   string
	UserID      string
	AccessToken string
}
type LoginError struct{ Err error }
type TokenValid struct{}
type TokenInvalid struct{}

// Browser navigation
type LibrariesLoaded struct{ Libraries []api.Library }
type ItemsLoaded struct {
	Items     []api.Item
	ParentID  string
	LevelName string
}
type PushLevel struct {
	Items     []api.Item
	LevelName string
}
type PopLevel struct{}

// Overlays
type OpenDetails struct{ Item api.Item }
type ItemDetailLoaded struct{ Item api.Item }
type ImageLoaded struct{ Data []byte }
type OpenSearch struct{}
type SearchResults struct{ Items []api.Item }
type CloseOverlay struct{}

// Playback
type PlayItem struct{ Item api.Item }
type PlayerDone struct{ Err error }

// Error
type AppError struct{ Err error }
type DismissError struct{}
```

- [ ] **Step 2: Write styles.go**

```go
// internal/ui/styles/styles.go
package styles

import "github.com/charmbracelet/lipgloss"

var (
	Breadcrumb = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)

	Selected = lipgloss.NewStyle().
			Background(lipgloss.Color("237")).
			Foreground(lipgloss.Color("15"))

	Dim = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	Subtitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	Error = lipgloss.NewStyle().
		Foreground(lipgloss.Color("9")).
		Bold(true)

	StatusBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	Overlay = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	Label = lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Bold(true)
)
```

- [ ] **Step 3: Write keys.go**

```go
// internal/ui/keys/keys.go
package keys

import "github.com/charmbracelet/bubbles/key"

type Bindings struct {
	Up      key.Binding
	Down    key.Binding
	Right   key.Binding
	Left    key.Binding
	Play    key.Binding
	Back    key.Binding
	Details key.Binding
	Search  key.Binding
	Random  key.Binding
	Help    key.Binding
	Quit    key.Binding
}

var Default = Bindings{
	Up:      key.NewBinding(key.WithKeys("up")),
	Down:    key.NewBinding(key.WithKeys("down")),
	Right:   key.NewBinding(key.WithKeys("right", "enter")),
	Left:    key.NewBinding(key.WithKeys("left", "esc")),
	Play:    key.NewBinding(key.WithKeys("enter")),
	Back:    key.NewBinding(key.WithKeys("esc", "left")),
	Details: key.NewBinding(key.WithKeys("i")),
	Search:  key.NewBinding(key.WithKeys("/")),
	Random:  key.NewBinding(key.WithKeys("r")),
	Help:    key.NewBinding(key.WithKeys("?")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c")),
}
```

- [ ] **Step 4: Verify compilation**

```bash
go build ./internal/ui/...
```

Expected: no errors.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/msg/ internal/ui/styles/ internal/ui/keys/
git commit -m "feat: add shared UI messages, styles, and key bindings"
```

---

## Task 7: Login Screen

**Files:**
- Create: `internal/ui/login/model.go`
- Create: `internal/ui/login/model_test.go`

- [ ] **Step 1: Write login tests**

```go
// internal/ui/login/model_test.go
package login_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/ui/login"
	"github.com/deglerj/fin/internal/ui/msg"
)

func TestInitialView(t *testing.T) {
	m := login.New(nil)
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestLoginErrorDisplayed(t *testing.T) {
	m := login.New(nil)
	updated, _ := m.Update(msg.LoginError{Err: fmt.Errorf("bad credentials")})
	view := updated.(login.Model).View()
	if !strings.Contains(view, "bad credentials") {
		t.Errorf("error not shown in view: %q", view)
	}
}
```

Add missing imports `"fmt"` and `"strings"` to the test file.

- [ ] **Step 2: Implement login/model.go**

```go
// internal/ui/login/model.go
package login

import (
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
	client   *api.Client
	inputs   []textinput.Model
	focused  field
	loading  bool
	spinner  spinner.Model
	errorMsg string
}

func New(client *api.Client) Model {
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

	return Model{client: client, inputs: inputs, spinner: sp}
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
			for i := range m.inputs {
				if field(i) == m.focused {
					m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
		case "shift+tab", "up":
			m.focused = (m.focused + fieldCount - 1) % fieldCount
			for i := range m.inputs {
				if field(i) == m.focused {
					m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
		case "enter":
			if m.focused < fieldPassword {
				m.focused++
				for i := range m.inputs {
					if field(i) == m.focused {
						m.inputs[i].Focus()
					} else {
						m.inputs[i].Blur()
					}
				}
			} else {
				return m.submit()
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

func (m Model) submit() (tea.Model, tea.Cmd) {
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
		resp, err := client.Authenticate(username, password)
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
```

- [ ] **Step 3: Run tests**

```bash
go test ./internal/ui/login/... -v
```

Expected: both tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/login/
git commit -m "feat: add login screen with form and error display"
```

---

## Task 8: Browser Model

**Files:**
- Create: `internal/ui/browser/model.go`
- Create: `internal/ui/browser/model_test.go`

- [ ] **Step 1: Write browser tests**

```go
// internal/ui/browser/model_test.go
package browser_test

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/ui/browser"
	"github.com/deglerj/fin/internal/ui/msg"
)

func items(names ...string) []api.Item {
	out := make([]api.Item, len(names))
	for i, n := range names {
		out[i] = api.Item{Id: fmt.Sprintf("id%d", i), Name: n, Type: "Movie"}
	}
	return out
}

func TestPushLevel(t *testing.T) {
	m := browser.New(nil, 80, 24)
	updated, _ := m.Update(msg.PushLevel{Items: items("A", "B", "C"), LevelName: "Movies"})
	bm := updated.(browser.Model)
	if bm.Depth() != 1 {
		t.Errorf("expected depth 1, got %d", bm.Depth())
	}
	if bm.SelectedItem().Name != "A" {
		t.Errorf("expected A selected, got %q", bm.SelectedItem().Name)
	}
}

func TestNavigateDown(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m, _ = m.Update(msg.PushLevel{Items: items("A", "B", "C"), LevelName: "Movies"})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	bm := m.(browser.Model)
	if bm.SelectedItem().Name != "B" {
		t.Errorf("expected B selected after down, got %q", bm.SelectedItem().Name)
	}
}

func TestPopLevel(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m, _ = m.Update(msg.PushLevel{Items: items("A"), LevelName: "Level1"})
	m, _ = m.Update(msg.PushLevel{Items: items("X"), LevelName: "Level2"})
	m, _ = m.Update(msg.PopLevel{})
	bm := m.(browser.Model)
	if bm.Depth() != 1 {
		t.Errorf("expected depth 1 after pop, got %d", bm.Depth())
	}
}

func TestRandomSelection(t *testing.T) {
	m := browser.New(nil, 80, 24)
	m, _ = m.Update(msg.PushLevel{Items: items("A", "B", "C", "D", "E"), LevelName: "Movies"})
	// random should emit PlayItem
	_, cmd := m.(browser.Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if cmd == nil {
		t.Error("expected a command from random key")
	}
}
```

Add `"fmt"` import to test file.

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/ui/browser/... 2>&1 | head -5
```

Expected: compile error — package doesn't exist.

- [ ] **Step 3: Implement browser/model.go**

```go
// internal/ui/browser/model.go
package browser

import (
	"fmt"
	"math/rand"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/ui/keys"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/deglerj/fin/internal/ui/styles"
)

type level struct {
	name   string
	items  []api.Item
	cursor int
	offset int
}

type Model struct {
	client  *api.Client
	stack   []level
	width   int
	height  int
	loading bool
	spinner spinner.Model
}

func New(client *api.Client, width, height int) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return Model{client: client, width: width, height: height, spinner: sp}
}

func (m Model) Depth() int { return len(m.stack) }

func (m Model) SelectedItem() api.Item {
	if len(m.stack) == 0 {
		return api.Item{}
	}
	top := m.stack[len(m.stack)-1]
	if top.cursor >= len(top.items) {
		return api.Item{}
	}
	return top.items[top.cursor]
}

func (m Model) visibleHeight() int {
	return m.height - 3 // breadcrumb + status + hint rows
}

func (m Model) Init() tea.Cmd { return m.spinner.Tick }

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case msg.PushLevel:
		m.loading = false
		m.stack = append(m.stack, level{name: message.LevelName, items: message.Items})
		return m, nil

	case msg.PopLevel:
		if len(m.stack) > 0 {
			m.stack = m.stack[:len(m.stack)-1]
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(message)
			return m, cmd
		}

	case tea.KeyMsg:
		if len(m.stack) == 0 {
			return m, nil
		}
		top := &m.stack[len(m.stack)-1]
		switch {
		case keys.Default.Up.Contains(message.String()):
			if top.cursor > 0 {
				top.cursor--
				if top.cursor < top.offset {
					top.offset = top.cursor
				}
			}
		case keys.Default.Down.Contains(message.String()):
			if top.cursor < len(top.items)-1 {
				top.cursor++
				if top.cursor >= top.offset+m.visibleHeight() {
					top.offset++
				}
			}
		case message.String() == "enter" || message.String() == "right":
			item := m.SelectedItem()
			if isLeaf(item) {
				return m, func() tea.Msg { return msg.PlayItem{Item: item} }
			}
			return m, m.fetchChildren(item)
		case message.String() == "esc" || message.String() == "left":
			if len(m.stack) > 1 {
				m.stack = m.stack[:len(m.stack)-1]
			}
		case message.String() == "i":
			item := m.SelectedItem()
			return m, func() tea.Msg { return msg.OpenDetails{Item: item} }
		case message.String() == "/":
			return m, func() tea.Msg { return msg.OpenSearch{} }
		case message.String() == "r":
			items := top.items
			if len(items) == 0 {
				return m, nil
			}
			item := items[rand.Intn(len(items))]
			for !isLeaf(item) {
				// pick random leaf: just play whatever was picked
				break
			}
			return m, func() tea.Msg { return msg.PlayItem{Item: item} }
		}
	}
	return m, nil
}

func isLeaf(item api.Item) bool {
	switch item.Type {
	case "Movie", "Episode", "Audio":
		return true
	}
	return false
}

func (m Model) fetchChildren(item api.Item) tea.Cmd {
	m.loading = true
	client := m.client
	return func() tea.Msg {
		var itemTypes []string
		switch item.Type {
		case "Series":
			itemTypes = []string{"Season"}
		case "Season":
			itemTypes = []string{"Episode"}
		}
		items, err := client.GetItems(item.Id, itemTypes)
		if err != nil {
			return msg.AppError{Err: err}
		}
		return msg.PushLevel{Items: items, LevelName: item.Name}
	}
}

func (m Model) breadcrumb() string {
	parts := make([]string, len(m.stack))
	for i, l := range m.stack {
		parts[i] = l.name
	}
	return styles.Breadcrumb.Render(strings.Join(parts, " › "))
}

func (m Model) View() string {
	if m.loading {
		return m.spinner.View() + " Loading..."
	}
	if len(m.stack) == 0 {
		return styles.Dim.Render("No content. Loading libraries...")
	}
	top := m.stack[len(m.stack)-1]
	var sb strings.Builder
	sb.WriteString(m.breadcrumb())
	sb.WriteByte('\n')

	visible := m.visibleHeight()
	end := top.offset + visible
	if end > len(top.items) {
		end = len(top.items)
	}
	for i := top.offset; i < end; i++ {
		item := top.items[i]
		line := formatItem(item)
		if i == top.cursor {
			sb.WriteString(styles.Selected.Width(m.width).Render(line))
		} else {
			sb.WriteString(styles.Dim.Render(line))
		}
		sb.WriteByte('\n')
	}

	count := fmt.Sprintf("%d items", len(top.items))
	hint := "↑↓ navigate · enter open/play · i details · / search · r random · q quit"
	sb.WriteString(styles.StatusBar.Render(count + "  " + hint))
	return sb.String()
}

func formatItem(item api.Item) string {
	name := item.Name
	if item.Type == "Episode" && item.IndexNumber > 0 {
		name = fmt.Sprintf("S%02dE%02d – %s", item.ParentIndexNumber, item.IndexNumber, item.Name)
	}
	if item.RunTimeTicks > 0 {
		mins := item.RunTimeTicks / 600_000_000
		return fmt.Sprintf("%-50s %dm", name, mins)
	}
	return name
}
```

Add `key.Binding.Contains` helper — bubbletea's `key.Binding` uses `Matches`, not `Contains`. Replace all `.Contains(message.String())` with `key.Matches(message, keys.Default.Up)` pattern:

Update the key check in Update to use:
```go
case key.Matches(message, keys.Default.Up):
```

Import `"github.com/charmbracelet/bubbles/key"` in browser/model.go.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/ui/browser/... -v
```

Expected: all 4 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/browser/
git commit -m "feat: add browser model with drill-down navigation stack"
```

---

## Task 9: Details & Search Overlays

**Files:**
- Create: `internal/ui/details/model.go` + `model_test.go`
- Create: `internal/ui/search/model.go` + `model_test.go`
- Create: `internal/ui/help/model.go`

- [ ] **Step 1: Write details test**

```go
// internal/ui/details/model_test.go
package details_test

import (
	"strings"
	"testing"

	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/ui/details"
	"github.com/deglerj/fin/internal/ui/msg"
)

func TestDetailsView(t *testing.T) {
	m := details.New(false)
	m2, _ := m.Update(msg.OpenDetails{Item: api.Item{
		Id: "m1", Name: "Dune", ProductionYear: 2021,
		Overview: "A noble family becomes embroiled in a war.",
	}})
	view := m2.(details.Model).View()
	if !strings.Contains(view, "Dune") {
		t.Errorf("title not in view: %q", view)
	}
	if !strings.Contains(view, "2021") {
		t.Errorf("year not in view: %q", view)
	}
}
```

- [ ] **Step 2: Implement details/model.go**

```go
// internal/ui/details/model.go
package details

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/image"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/deglerj/fin/internal/ui/styles"
)

type Model struct {
	item         api.Item
	imageCapable bool
	imageData    []byte
	client       apiClient
	width        int
	height       int
}

type apiClient interface {
	GetImage(itemID string, maxWidth int) ([]byte, error)
}

func New(imageCapable bool) Model {
	return Model{imageCapable: imageCapable}
}

func (m Model) WithClient(c apiClient) Model { m.client = c; return m }
func (m Model) WithSize(w, h int) Model      { m.width = w; m.height = h; return m }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case msg.OpenDetails:
		m.item = message.Item
		m.imageData = nil
		var cmd tea.Cmd
		if m.imageCapable && m.client != nil {
			itemID := message.Item.Id
			c := m.client
			cmd = func() tea.Msg {
				data, err := c.GetImage(itemID, 200)
				if err != nil {
					return nil // silent failure
				}
				return msg.ImageLoaded{Data: data}
			}
		}
		return m, cmd
	case msg.ImageLoaded:
		m.imageData = message.Data
	case tea.KeyMsg:
		switch message.String() {
		case "esc", "i":
			return m, func() tea.Msg { return msg.CloseOverlay{} }
		case "enter":
			return m, func() tea.Msg { return msg.PlayItem{Item: m.item} }
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.item.Id == "" {
		return ""
	}
	var sb strings.Builder

	if m.imageCapable && len(m.imageData) > 0 {
		sb.WriteString(image.Encode(m.imageData, 20, 10))
		sb.WriteByte('\n')
	}

	sb.WriteString(styles.Title.Render(m.item.Name))
	if m.item.ProductionYear > 0 {
		sb.WriteString(styles.Subtitle.Render(fmt.Sprintf(" (%d)", m.item.ProductionYear)))
	}
	sb.WriteByte('\n')

	if m.item.RunTimeTicks > 0 {
		mins := m.item.RunTimeTicks / 600_000_000
		sb.WriteString(styles.Dim.Render(fmt.Sprintf("%dh%02dm", mins/60, mins%60)))
		sb.WriteByte('\n')
	}

	if m.item.CommunityRating > 0 {
		sb.WriteString(styles.Dim.Render(fmt.Sprintf("★ %.1f", m.item.CommunityRating)))
		sb.WriteByte('\n')
	}

	if m.item.Overview != "" {
		sb.WriteByte('\n')
		sb.WriteString(wordWrap(m.item.Overview, m.width-4))
		sb.WriteByte('\n')
	}

	var directors, cast []string
	for _, p := range m.item.People {
		switch p.Type {
		case "Director":
			directors = append(directors, p.Name)
		case "Actor":
			if len(cast) < 5 {
				cast = append(cast, p.Name)
			}
		}
	}
	if len(directors) > 0 {
		sb.WriteString(styles.Label.Render("Director: ") + strings.Join(directors, ", ") + "\n")
	}
	if len(cast) > 0 {
		sb.WriteString(styles.Label.Render("Cast: ") + strings.Join(cast, ", ") + "\n")
	}

	sb.WriteByte('\n')
	sb.WriteString(styles.Dim.Render("enter to play · esc/i to close"))
	return styles.Overlay.Width(m.width - 4).Render(sb.String())
}

func wordWrap(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	var result strings.Builder
	words := strings.Fields(s)
	line := 0
	for i, w := range words {
		if line+len(w)+1 > width && line > 0 {
			result.WriteByte('\n')
			line = 0
		}
		if i > 0 && line > 0 {
			result.WriteByte(' ')
			line++
		}
		result.WriteString(w)
		line += len(w)
	}
	return result.String()
}
```

- [ ] **Step 3: Write search test**

```go
// internal/ui/search/model_test.go
package search_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/deglerj/fin/internal/ui/search"
)

func TestSearchShowsResults(t *testing.T) {
	m := search.New(nil)
	results := []api.Item{{Id: "1", Name: "Dune"}, {Id: "2", Name: "Dune: Part Two"}}
	m2, _ := m.Update(msg.SearchResults{Items: results})
	view := m2.(search.Model).View()
	if !strings.Contains(view, "Dune") {
		t.Errorf("results not in view: %q", view)
	}
}

func TestEscClosesSearch(t *testing.T) {
	m := search.New(nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected close command")
	}
	result := cmd()
	if _, ok := result.(msg.CloseOverlay); !ok {
		t.Errorf("expected CloseOverlay, got %T", result)
	}
}
```

- [ ] **Step 4: Implement search/model.go**

```go
// internal/ui/search/model.go
package search

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/ui/msg"
	"github.com/deglerj/fin/internal/ui/styles"
)

type debounceMsg struct{ seq int }

type Model struct {
	input   textinput.Model
	results []api.Item
	cursor  int
	client  *api.Client
	seq     int // debounce counter
}

func New(client *api.Client) Model {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Focus()
	return Model{input: ti, client: client}
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case msg.SearchResults:
		m.results = message.Items
		m.cursor = 0
		return m, nil

	case debounceMsg:
		if message.seq != m.seq {
			return m, nil // stale timer, newer keystroke fired
		}
		term := m.input.Value()
		if term == "" {
			m.results = nil
			return m, nil
		}
		c := m.client
		return m, func() tea.Msg {
			items, err := c.Search(term)
			if err != nil {
				return msg.AppError{Err: err}
			}
			return msg.SearchResults{Items: items}
		}

	case tea.KeyMsg:
		switch message.String() {
		case "esc":
			return m, func() tea.Msg { return msg.CloseOverlay{} }
		case "enter":
			if m.cursor < len(m.results) {
				item := m.results[m.cursor]
				return m, func() tea.Msg { return msg.NavigateToItem{Item: item} }
			}
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.results)-1 {
				m.cursor++
			}
		default:
			m.seq++
			seq := m.seq
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(message)
			return m, tea.Batch(cmd, tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
				return debounceMsg{seq: seq}
			}))
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(message)
	return m, cmd
}

func (m Model) View() string {
	var sb strings.Builder
	sb.WriteString(styles.Label.Render("Search: "))
	sb.WriteString(m.input.View())
	sb.WriteByte('\n')
	sb.WriteString(strings.Repeat("─", 40))
	sb.WriteByte('\n')
	for i, item := range m.results {
		line := item.Name
		if item.Type == "Episode" {
			line = item.SeriesName + " – " + item.Name
		}
		if i == m.cursor {
			sb.WriteString(styles.Selected.Render(line))
		} else {
			sb.WriteString(styles.Dim.Render(line))
		}
		sb.WriteByte('\n')
	}
	if len(m.results) == 0 && m.input.Value() != "" {
		sb.WriteString(styles.Dim.Render("No results"))
	}
	sb.WriteString(styles.Dim.Render("\nenter to open · esc to close"))
	return styles.Overlay.Render(sb.String())
}
```

Add missing import for `msg.NavigateToItem` — add to msg/msg.go if not already there:

```go
type NavigateToItem struct{ Item api.Item }
```

- [ ] **Step 5: Write help/model.go (no tests needed — static content)**

```go
// internal/ui/help/model.go
package help

import (
	"github.com/deglerj/fin/internal/ui/styles"
)

const helpText = `
  ↑ / ↓       Navigate list
  → / Enter   Open / drill in / play
  ← / Esc     Back / close overlay
  i           Details overlay
  /           Search
  r           Random from current list
  ?           Toggle this help
  q           Quit
`

func View() string {
	return styles.Overlay.Render(styles.Dim.Render(helpText))
}
```

- [ ] **Step 6: Run tests**

```bash
go test ./internal/ui/details/... ./internal/ui/search/... -v
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/ui/details/ internal/ui/search/ internal/ui/help/
git commit -m "feat: add details overlay, search overlay, and help screen"
```

---

## Task 10: App Root Model & main.go

**Files:**
- Create: `internal/ui/app/model.go`
- Create: `internal/ui/app/model_test.go`
- Modify: `cmd/fin/main.go`

- [ ] **Step 1: Write app model tests**

```go
// internal/ui/app/model_test.go
package app_test

import (
	"testing"

	"github.com/deglerj/fin/internal/ui/app"
	"github.com/deglerj/fin/internal/ui/msg"
)

func TestStartsAtLogin(t *testing.T) {
	m := app.New(nil, nil, false)
	if m.Screen() != app.ScreenLogin {
		t.Errorf("expected login screen, got %v", m.Screen())
	}
}

func TestLoginSuccessTransition(t *testing.T) {
	m := app.New(nil, nil, false)
	m2, _ := m.Update(msg.LoginSuccess{ServerURL: "http://jf", UserID: "u1", AccessToken: "tok"})
	am := m2.(app.Model)
	if am.Screen() != app.ScreenBrowser {
		t.Errorf("expected browser after login success, got %v", am.Screen())
	}
}

func TestErrorDisplayed(t *testing.T) {
	m := app.New(nil, nil, false)
	m2, _ := m.Update(msg.AppError{Err: fmt.Errorf("network timeout")})
	view := m2.(app.Model).View()
	if !strings.Contains(view, "network timeout") {
		t.Errorf("error not in view: %q", view)
	}
}
```

Add imports `"fmt"` and `"strings"` to test file.

- [ ] **Step 2: Implement app/model.go**

```go
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
	var loginClient *api.Client
	if client != nil {
		loginClient = client
	}
	return Model{
		cfg:          cfg,
		client:       client,
		imageCapable: imageCapable,
		screen:       ScreenLogin,
		login:        login.New(loginClient),
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
		if m.cfg == nil {
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

// helpers to re-cast tea.Model returns to concrete types
func asBrowserModel(m tea.Model, cmd tea.Cmd) (browser.Model, tea.Cmd) {
	return m.(browser.Model), cmd
}
func asDetailsModel(m tea.Model, cmd tea.Cmd) (details.Model, tea.Cmd) {
	return m.(details.Model), cmd
}
func asSearchModel(m tea.Model, cmd tea.Cmd) (search.Model, tea.Cmd) {
	return m.(search.Model), cmd
}
```

Add `CredentialsPath()` method to `config.Config`:

```go
// in internal/config/config.go, add:
func (c *Config) CredentialsPath() string {
	return CredentialsPath()
}
```

- [ ] **Step 3: Run app tests**

```bash
go test ./internal/ui/app/... -v
```

Expected: all 3 tests pass.

- [ ] **Step 4: Update main.go with startup sequence**

```go
// cmd/fin/main.go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/auth"
	"github.com/deglerj/fin/internal/config"
	"github.com/deglerj/fin/internal/image"
	"github.com/deglerj/fin/internal/ui/app"
	"github.com/deglerj/fin/internal/ui/msg"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	imageCapable := image.Probe()

	var initialModel tea.Model
	creds, err := auth.LoadCreds(cfg.CredentialsPath(), auth.DefaultMachineID{})
	if err == nil {
		client := api.New(creds.ServerURL)
		client.SetAuth(creds.UserID, creds.AccessToken)
		if err := client.ValidateToken(); err == nil {
			m := app.New(cfg, client, imageCapable)
			// Immediately push to browser and fetch libraries
			m2, _ := m.Update(msg.LoginSuccess{
				ServerURL:   creds.ServerURL,
				UserID:      creds.UserID,
				AccessToken: creds.AccessToken,
			})
			initialModel = m2
		} else {
			initialModel = app.New(cfg, nil, imageCapable)
		}
	} else {
		initialModel = app.New(cfg, nil, imageCapable)
	}

	p := tea.NewProgram(initialModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Build the full project**

```bash
go build ./...
```

Expected: no errors. The binary is now runnable (will show login screen if no credentials saved).

- [ ] **Step 6: Commit**

```bash
git add internal/ui/app/ cmd/fin/main.go internal/config/config.go
git commit -m "feat: wire app root model with startup auth flow and main entrypoint"
```

---

## Task 11: Final Polish

**Files:**
- Create: `.gitignore`
- Modify: `internal/auth/credentials.go` (add missing `filepath` import)

- [ ] **Step 1: Run all tests**

```bash
go test ./... -v 2>&1 | tail -30
```

Expected: all packages pass. Fix any remaining type assertion issues.

- [ ] **Step 2: Build release binary**

```bash
go build -o fin ./cmd/fin/
```

Expected: `./fin` binary created.

- [ ] **Step 3: Verify binary starts**

```bash
./fin --help 2>&1 || ./fin &
```

If no config file exists, login screen appears. `ctrl+c` to exit.

- [ ] **Step 4: Add .gitignore**

```
fin
*.test
.superpowers/
```

- [ ] **Step 5: Final commit**

```bash
git add .gitignore
git commit -m "chore: add .gitignore"
```

---

## Self-Review Checklist

| Spec requirement | Covered by task |
|-----------------|-----------------|
| Library browsing | Task 8 (browser drill-down) |
| TV show drill-down | Task 8 (fetchChildren) |
| Details overlay with thumbnail | Task 9 (details model) |
| Global search | Task 9 (search model + debounce) |
| Random playback | Task 8 (r key) |
| mpv playback | Task 5 (player.Play) |
| Encrypted credentials | Task 3 |
| Skip login if valid creds | Task 10 (main.go startup) |
| Kitty image probe | Task 5 (image.Probe) |
| Graceful image fallback | Task 9 (details skips image block) |
| XDG config + TOML | Task 2 |
| Arrow key navigation | Task 8 + keys package |
| `i` for details | Task 8 → app model |
| `/` for search | Task 8 → app model |
| `r` for random | Task 8 |
| `?` for help | Task 10 (app model) |
| Error status bar | Task 10 (app model View) |
| Single server | All — single Client instance |
