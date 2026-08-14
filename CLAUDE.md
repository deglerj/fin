# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`fin` — a terminal UI client for Jellyfin. Go module: `github.com/deglerj/fin`. Binary: `./fin`.

## Commands

```bash
go build ./...              # compile all
go build -o fin ./cmd/fin/  # build runnable binary
go test ./...               # run all tests
go test ./internal/api/...  # run a single package's tests
go test ./... -run TestName # run a single test
golangci-lint run           # lint (must pass before commit — CI enforces this)
```

## Architecture

bubbletea MVU throughout. The root model (`internal/ui/app`) owns a `screen` (login or browser), an `overlay` (none, search, help) covering the left pane, and a permanent `details` pane on the right. It delegates `Update` calls to whichever screen/overlay is active and returns the re-cast concrete type. **All Jellyfin API calls are `tea.Cmd` functions** — never call the API client synchronously inside `Update`.

Key design rule: shared `tea.Msg` types live in `internal/ui/msg` so that sub-models can emit messages without importing each other (prevents import cycles).

### Package map

| Package | Role |
|---|---|
| `internal/config` | TOML config, XDG paths, `Load()` |
| `internal/auth` | `Credentials` struct, AES-GCM encrypt/decrypt keyed by machine ID (HKDF) |
| `internal/api` | `Client` with `SetAuth`, typed endpoints for Jellyfin REST API |
| `internal/player` | `Play()` returns `tea.ExecProcess` cmd for mpv |
| `internal/image` | Kitty graphics protocol encoder, terminal capability probe |
| `internal/ui/msg` | All shared `tea.Msg` types |
| `internal/ui/styles` | lipgloss style vars |
| `internal/ui/keys` | `key.Binding` constants |
| `internal/ui/app` | Root model — screen router + overlay manager |
| `internal/ui/login` | Login form with spinner |
| `internal/ui/browser` | Navigation stack (library → series → season → episode) |
| `internal/ui/details` | Details side pane — scrollable text plus optional kitty image |
| `internal/ui/search` | Search overlay with 300ms debounce |
| `internal/ui/help` | Static help overlay |

### Startup flow

`main.go` loads config → probes kitty capability → `initialModel` tries to load saved credentials. If they decrypt, it synthesizes a `LoginSuccess{Restored: true}` so the app starts directly in the browser; the token is not validated up front, because the first API call's 401 already maps to `TokenInvalid` and returns to the login screen. A credentials file that exists but cannot be decrypted is reported to the user rather than silently ignored.

## Config

`~/.config/fin/config.toml` (XDG-aware). Credentials stored encrypted at `~/.config/fin/credentials`.

Default player: `mpv`. Override via `[player] command = "vlc"`.

## Tests

Use `github.com/stretchr/testify` (`require`/`assert`) — not manual `if/t.Errorf`. All model tests drive the `Update` method directly with message types from `internal/ui/msg`.
