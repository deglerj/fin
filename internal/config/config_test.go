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
