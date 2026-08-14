// cmd/fin/main.go
package main

import (
	"errors"
	"flag"
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

var version = "dev"

func main() {
	var showVersion bool
	var configDir string

	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.BoolVar(&showVersion, "v", false, "Print version and exit")
	flag.StringVar(&configDir, "config", "", "Config `DIR` (default: ~/.config/fin)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "fin — A terminal UI client for Jellyfin\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  fin [--config DIR] [--version]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fmt.Fprintf(os.Stderr, "  --config DIR   Config directory (default: ~/.config/fin)\n")
		fmt.Fprintf(os.Stderr, "  --version, -v  Print version and exit\n")
		fmt.Fprintf(os.Stderr, "  -h, --help     Show this help\n\n")
		fmt.Fprintf(os.Stderr, "Config:      ~/.config/fin/config.toml\n")
		fmt.Fprintf(os.Stderr, "Credentials: ~/.config/fin/credentials\n")
	}

	flag.Parse()

	api.Version = version

	if showVersion {
		fmt.Println("fin — A terminal UI client for Jellyfin")
		fmt.Println("Version:", version)
		return
	}

	cfg, err := config.Load(configDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(cfg, image.Probe()), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// initialModel restores a saved session when one is available, so fin starts in
// the browser rather than at the login form.
func initialModel(cfg *config.Config, imageCapable bool) tea.Model {
	creds, err := auth.LoadCreds(cfg.CredentialsPath(), auth.DefaultMachineID{})
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		return app.New(cfg, nil, imageCapable) // first run
	default:
		// The file exists but will not decrypt — usually a changed machine ID,
		// occasionally a truncated write. Say so rather than presenting an
		// empty login form with no explanation.
		return app.New(cfg, nil, imageCapable).
			WithError("saved credentials could not be read (" + err.Error() + ") — please log in again")
	}

	client := api.New(creds.ServerURL)
	client.SetAuth(creds.UserID, creds.AccessToken)
	restored, _ := app.New(cfg, client, imageCapable).Update(msg.LoginSuccess{
		ServerURL:   creds.ServerURL,
		UserID:      creds.UserID,
		AccessToken: creds.AccessToken,
		Restored:    true,
	})
	return restored
}
