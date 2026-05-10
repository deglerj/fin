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
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "fin")
}

// CredentialsPath returns the path to the credentials file.
func CredentialsPath() string {
	return filepath.Join(configDir(), "credentials")
}

// CredentialsPath returns the path to the credentials file (method form for use with a Config receiver).
func (c *Config) CredentialsPath() string {
	return CredentialsPath()
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
