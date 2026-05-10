# fin — Jellyfin TUI Client: Design Spec

**Date:** 2026-05-10  
**Language:** Go  
**TUI framework:** bubbletea + lipgloss (Charm ecosystem)  
**Target platform:** Linux primary; macOS and Windows nice-to-have

---

## 1. Overview

`fin` is a text-mode terminal UI client for Jellyfin. It covers browsing libraries, drilling into TV shows, global search, and random playback. Video and audio are delegated to mpv. The UI runs entirely in the terminal using bubbletea's Elm-style architecture.

### In scope (v1)
- Library browsing (movies, TV shows; other types navigable but not primary)
- Breadcrumb drill-down: Libraries → Collection → Show → Season → Episodes
- Details overlay: thumbnail (kitty graphics protocol), description, cast, rating, runtime
- Global search with debounced live results
- Random playback (context-sensitive: picks from current list)
- Playback via mpv (video and audio)
- Secure credential storage (encrypted token, never plaintext password)
- XDG-compliant config file (TOML, plaintext except credentials)
- Graceful image fallback when kitty protocol unavailable

### Out of scope (v1)
- Built-in audio/video playback
- Multi-server support
- Transcoding control
- Queue / playlist management
- Admin functions

---

## 2. Architecture

### Module layout

```
cmd/fin/             entry point, flag parsing, tea.NewProgram
internal/
  api/               Jellyfin HTTP client (hand-rolled)
  auth/              encrypted credential storage
  config/            TOML config loader (XDG paths)
  player/            mpv subprocess launcher
  image/             kitty graphics protocol encoder + capability detection
  ui/
    app/             root model — router, holds active page
    login/           login screen model
    browser/         library + drill-down browser model
    search/          global search overlay model
    details/         details/metadata overlay model
```

### Root model (router)

The `app.Model` holds the current active page as a `tea.Model` interface value and a navigation stack for the browser drill-down. Overlays (search, details) are stored as optional fields on the root model; they render on top of the browser via lipgloss layering. Navigation events are communicated upward via typed `tea.Msg` return values — no global state or channels.

---

## 3. Startup & Authentication

1. Load `$XDG_CONFIG_HOME/fin/config.toml`
2. Check for `$XDG_CONFIG_HOME/fin/credentials` (mode 0600)
3. If found: decrypt → validate token with a lightweight Jellyfin API ping (`GET /Users/{id}/Items`)
4. If valid: navigate directly to browser
5. If missing or invalid: show login screen
6. On successful login: store encrypted credentials, navigate to browser
7. Logout clears the credentials file and returns to login screen

### Credential encryption

- Key derivation: HKDF-SHA256 from `/etc/machine-id` (Linux) or equivalent
- Encryption: AES-256-GCM
- Stored payload (JSON): `{ "server_url", "user_id", "access_token" }`
- The user's password is **never stored**
- File permissions enforced at write time: `0600`

---

## 4. UI Screens

### Browser (persistent base screen)

Single-panel list. Breadcrumb header line at top shows current path (e.g. `Movies > Dune`). Status bar at bottom shows item count and keybindings hint.

**Drill-down levels:**
```
Libraries
  └─ Movies / TV Shows / Music / Photos / …
       └─ [TV Show]
            └─ Season N
                 └─ Episodes
```

Each level is a new list pushed onto a navigation stack. `←` / `Esc` pops one level (when no overlay is open). `→` / `Enter` pushes the next level or triggers playback for leaf items (movies, episodes). When an overlay is open, `Esc` closes the overlay first; the browser stack is unaffected.

`r` at any level picks a random item from the current list and plays it immediately.

### Details overlay (`i`)

Slides in over the browser. Shows:
- Thumbnail image (kitty protocol if supported; omitted otherwise)
- Title, year, duration
- Synopsis / description
- Director / cast (truncated to fit)
- Community rating
- `Enter` to play, `Esc` to close

### Search overlay (`/`)

Full-width overlay. Input field at top; results update debounced (~300ms) as user types. Results are cross-type (movies, shows, episodes). `Enter` on a result closes search and navigates the browser to that item. `Esc` closes search without navigating.

API call: `GET /Items?searchTerm=...&includeItemTypes=Movie,Series,Episode&limit=20`

### Login screen

Fields: Server URL, Username, Password (masked). Arrow keys / Tab to move between fields, `Enter` to submit. Displays inline error on failure.

### Help overlay (`?`)

Static keybindings reference. `Esc` or `?` to close.

---

## 5. Jellyfin API Client (`internal/api/`)

Hand-rolled HTTP client using `net/http`. No auto-generated SDK — only the endpoints actually used are implemented.

| Endpoint | Purpose |
|----------|---------|
| `POST /Users/AuthenticateByName` | Login, returns access token |
| `GET /Users/{id}/Items` | List items (with `ParentId`, type filters) |
| `GET /Library/MediaFolders` | Top-level libraries |
| `GET /Items?searchTerm=...` | Global search |
| `GET /Users/{id}/Items/{itemId}` | Item details |
| `GET /Items/{itemId}/Images/Primary` | Thumbnail image bytes |
| `{server}/Videos/{itemId}/stream?api_key={token}&static=true` | Direct stream URL for mpv |

All requests include `X-Emby-Token` header. Responses decoded into typed structs. All API calls are issued as bubbletea `Cmd`s (goroutines); results returned as `tea.Msg`. Loading state shown via spinner in status bar.

---

## 6. mpv Integration (`internal/player/`)

Uses bubbletea's `tea.ExecProcess` to hand terminal control to mpv and resume fin cleanly on exit.

```go
url := fmt.Sprintf("%s/Videos/%s/stream?api_key=%s&static=true", server, itemID, token)
cmd := exec.Command("mpv", url, "--title="+title)
return tea.ExecProcess(cmd, func(err error) tea.Msg { ... })
```

mpv is assumed to be installed. If not found, show an error message in the status bar. Audio files (music, podcasts) use the same mechanism — mpv handles both.

---

## 7. Image Display (`internal/image/`)

### Capability detection

On startup, probe the terminal by writing a kitty graphics APC query (`\x1b_Ga=q;\x1b\\`) to stdout and reading stdin with a short timeout (~100ms). Any terminal implementing the kitty graphics protocol responds. Result is stored once and reused throughout the session.

`$KITTY_WINDOW_ID` being set means we're running inside Kitty — treat as capable and skip the probe entirely.

### Display

Thumbnail is fetched from Jellyfin's image API as JPEG. Encoded as base64 kitty graphics protocol chunks and written directly to the terminal within the details overlay's `View()` output. Image dimensions are constrained to fit within the overlay (max ~20 columns wide on the left side).

### Fallback

If capability detection returns false, the image area is omitted. The details overlay fills the freed space with additional text metadata. No error shown to the user.

---

## 8. Configuration (`internal/config/`)

File: `$XDG_CONFIG_HOME/fin/config.toml` (falls back to `~/.config/fin/config.toml`)

```toml
[server]
url = "https://jellyfin.example.com"

[ui]
date_format = "2006-01-02"   # Go time format

[player]
command = "mpv"              # path or binary name
extra_args = []              # extra args passed to player

[keybindings]
# Optional overrides — defaults shown
# play    = "enter"
# back    = "esc"
# details = "i"
# search  = "/"
# random  = "r"
# quit    = "q"
# help    = "?"
```

Credentials are stored separately in `$XDG_CONFIG_HOME/fin/credentials` (binary, encrypted — not TOML).

---

## 9. Keybindings

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate list |
| `→` / `Enter` | Open / drill in / play |
| `←` / `Esc` | Back / close overlay |
| `i` | Open details overlay |
| `/` | Open search overlay |
| `r` | Random item from current list |
| `?` | Toggle help overlay |
| `q` | Quit |

Arrow keys are primary. Vim keys (`j`/`k`/`h`/`l`) are not bound by default but can be added via config.

---

## 10. Error Handling

Errors surface as a dismissible status bar message (red highlight, bottom of screen). The app never panics on API or network failures. After showing the error the UI remains fully navigable. `Esc` dismisses the error message.

Specific cases:
- **Network error / timeout**: show message, stay on current screen
- **Auth token expired**: clear credentials, redirect to login
- **mpv not found**: show error, do not attempt playback
- **Image fetch failure**: silently omit image (text fallback)
- **Config parse error**: show error at startup, exit with code 1

---

## 11. Testing

- `internal/api/` — table-driven tests against `httptest.NewServer` mock
- `internal/auth/` — encrypt/decrypt round-trip, key derivation determinism
- `internal/config/` — parse valid and invalid TOML inputs
- `internal/image/` — probe timeout behavior, base64 encoding correctness
- `internal/ui/*` — bubbletea model unit tests: send `tea.Msg`, assert model state and `View()` output
- No CI integration tests against a live Jellyfin instance

---

## 12. Cross-Platform Notes

| Feature | Linux | macOS | Windows |
|---------|-------|-------|---------|
| Config path | `$XDG_CONFIG_HOME` | `~/Library/Application Support` | `%APPDATA%` |
| Machine ID (key derivation) | `/etc/machine-id` | `IOPlatformUUID` (sysctl) | Registry MachineGuid |
| Terminal images | kitty / WezTerm / Ghostty | iTerm2 does not support kitty protocol | Unlikely to work |
| mpv | package manager | Homebrew | mpv.io installer |

Platform-specific machine-ID code is isolated behind an interface in `internal/auth/`. Windows and macOS support is best-effort for v1.
