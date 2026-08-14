package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Player    PlayerConfig `toml:"player"`
	configDir string
}

type PlayerConfig struct {
	Command   string   `toml:"command"`
	ExtraArgs []string `toml:"extra_args"`
}

func defaults() *Config {
	return &Config{
		Player: PlayerConfig{Command: "mpv"},
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

// Load reads config.toml from dir, falling back to the XDG location when dir is
// empty. Unknown sections are ignored, so a config carrying settings from an
// older version still loads.
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
