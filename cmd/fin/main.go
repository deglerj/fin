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
