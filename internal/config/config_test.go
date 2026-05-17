package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/deglerj/fin/internal/config"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg, err := config.Load("")
	require.NoError(t, err)
	require.Equal(t, "mpv", cfg.Player.Command)
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "fin"), 0755))
	toml := `[server]
url = "https://jf.example.com"
[player]
command = "vlc"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "fin", "config.toml"), []byte(toml), 0644))
	cfg, err := config.Load("")
	require.NoError(t, err)
	require.Equal(t, "https://jf.example.com", cfg.Server.URL)
	require.Equal(t, "vlc", cfg.Player.Command)
}

func TestLoadWithExplicitDir(t *testing.T) {
	dir := t.TempDir()
	toml := `[player]
command = "vlc"
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.toml"), []byte(toml), 0644))
	cfg, err := config.Load(dir)
	require.NoError(t, err)
	require.Equal(t, "vlc", cfg.Player.Command)
	require.Equal(t, filepath.Join(dir, "credentials"), cfg.CredentialsPath())
}

func TestCredentialsPathFromConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg, err := config.Load("")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "fin", "credentials"), cfg.CredentialsPath())
}
