package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deglerj/fin/internal/auth"
	"github.com/deglerj/fin/internal/config"
	"github.com/deglerj/fin/internal/ui/app"
	"github.com/stretchr/testify/require"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load(t.TempDir())
	require.NoError(t, err)
	return cfg
}

func TestInitialModelStartsAtLoginOnFirstRun(t *testing.T) {
	m := initialModel(testConfig(t), false).(app.Model)
	require.Equal(t, app.ScreenLogin, m.Screen())
	require.NotContains(t, m.View(), "Error:", "a missing credentials file is not an error")
}

func TestInitialModelRestoresSavedSession(t *testing.T) {
	cfg := testConfig(t)
	require.NoError(t, auth.Save(auth.Credentials{
		ServerURL: "https://jf.example.com", UserID: "u1", AccessToken: "tok",
	}, cfg.CredentialsPath(), auth.DefaultMachineID{}))

	m := initialModel(cfg, false).(app.Model)
	require.Equal(t, app.ScreenBrowser, m.Screen())
}

func TestInitialModelReportsUnreadableCredentials(t *testing.T) {
	cfg := testConfig(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(cfg.CredentialsPath()), 0o700))
	require.NoError(t, os.WriteFile(cfg.CredentialsPath(), []byte("not encrypted at all"), 0o600))

	m := initialModel(cfg, false).(app.Model)
	require.Equal(t, app.ScreenLogin, m.Screen())
	require.Contains(t, strings.ToLower(m.View()), "saved credentials could not be read",
		"an undecryptable credentials file must be explained, not silently ignored")
}
