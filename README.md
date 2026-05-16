# fin

A terminal UI client for [Jellyfin](https://jellyfin.org), built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- Browse libraries, series, seasons, and episodes
- Full-text search with debounce
- Play media via mpv
- Item details overlay
- Inline cover art via [Kitty graphics protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/) (optional)
- Encrypted credential storage (AES-GCM, keyed by machine ID)
- Remembers login across sessions

## Requirements

- [mpv](https://mpv.io) — media playback
- A running Jellyfin server
- A [Kitty terminal](https://sw.kovidgoyal.net/kitty/) for inline images (optional)

## Install

**Pre-built binaries** — download from the [releases page](../../releases).

**From source:**

```sh
go install github.com/deglerj/fin/cmd/fin@latest
```

## Configuration

Config file: `~/.config/fin/config.toml` (XDG-aware)

```toml
[player]
command = "mpv"   # default; override with e.g. "vlc"
```

Credentials are stored encrypted at `~/.config/fin/credentials`. Delete this file to log out.

## Keybindings

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate |
| `→` / `Enter` | Select / open |
| `←` / `Esc` / `Backspace` | Back |
| `Enter` | Play (on episode) |
| `/` | Search |
| `r` | Random item |
| `?` | Help |
| `q` / `Ctrl+C` | Quit |

## License

[MIT](LICENSE)
