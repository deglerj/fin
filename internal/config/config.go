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
	configDir   string
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

func defaultConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "fin")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "fin")
}

func (c *Config) CredentialsPath() string {
	return filepath.Join(c.configDir, "credentials")
}

func Load(dir string) (*Config, error) {
	if dir == "" {
		dir = defaultConfigDir()
	}
	cfg := defaults()
	cfg.configDir = dir
	path := filepath.Join(dir, "config.toml")
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
