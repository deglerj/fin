package player_test

import (
	"testing"

	"github.com/deglerj/fin/internal/player"
	"github.com/stretchr/testify/require"
)

func TestBuildCmd(t *testing.T) {
	cmd := player.BuildCmd("mpv", []string{"--really-quiet"}, "https://example.com/video", "Test Movie")
	args := cmd.Args
	require.Equal(t, "mpv", args[0])
	var hasURL, hasTitle bool
	for _, a := range args {
		if a == "https://example.com/video" {
			hasURL = true
		}
		if a == "--title=Test Movie" {
			hasTitle = true
		}
	}
	require.True(t, hasURL, "URL not in args")
	require.True(t, hasTitle, "title not in args")
}
