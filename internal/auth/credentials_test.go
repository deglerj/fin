package auth_test

import (
	"path/filepath"
	"testing"

	"github.com/deglerj/fin/internal/auth"
	"github.com/stretchr/testify/require"
)

type fixedID struct{}

func (fixedID) MachineID() (string, error) { return "test-machine-id-1234", nil }

type wrongKeyID struct{}

func (wrongKeyID) MachineID() (string, error) { return "different-machine-id", nil }

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	creds := auth.Credentials{
		ServerURL:   "https://jf.example.com",
		UserID:      "abc123",
		AccessToken: "tok-xyz",
	}
	require.NoError(t, auth.Save(creds, path, fixedID{}))
	got, err := auth.LoadCreds(path, fixedID{})
	require.NoError(t, err)
	require.Equal(t, creds.AccessToken, got.AccessToken)
	require.Equal(t, creds.ServerURL, got.ServerURL)
}

func TestLoadMissing(t *testing.T) {
	_, err := auth.LoadCreds("/nonexistent/path", fixedID{})
	require.Error(t, err)
}

func TestWrongKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	creds := auth.Credentials{AccessToken: "secret"}
	require.NoError(t, auth.Save(creds, path, fixedID{}))
	_, err := auth.LoadCreds(path, wrongKeyID{})
	require.Error(t, err)
}
