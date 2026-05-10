package auth_test

import (
	"path/filepath"
	"testing"

	"github.com/deglerj/fin/internal/auth"
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
	if err := auth.Save(creds, path, fixedID{}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := auth.LoadCreds(path, fixedID{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.AccessToken != creds.AccessToken {
		t.Errorf("token mismatch: got %q", got.AccessToken)
	}
	if got.ServerURL != creds.ServerURL {
		t.Errorf("url mismatch: got %q", got.ServerURL)
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := auth.LoadCreds("/nonexistent/path", fixedID{})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestWrongKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "credentials")
	creds := auth.Credentials{AccessToken: "secret"}
	if err := auth.Save(creds, path, fixedID{}); err != nil {
		t.Fatal(err)
	}
	_, err := auth.LoadCreds(path, wrongKeyID{})
	if err == nil {
		t.Fatal("expected decryption to fail with different key")
	}
}
