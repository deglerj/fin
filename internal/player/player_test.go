package player_test

import (
	"testing"

	"github.com/deglerj/fin/internal/player"
	"github.com/stretchr/testify/require"
)

func TestBuildCmd(t *testing.T) {
	cmd := player.BuildCmd("mpv", []string{"--really-quiet"}, "https://example.com/video", "Test Movie", 0, "/tmp/test.sock")
	args := cmd.Args
	require.Equal(t, "mpv", args[0])
	var hasURL, hasTitle, hasSocket bool
	for _, a := range args {
		if a == "https://example.com/video" {
			hasURL = true
		}
		if a == "--title=Test Movie" {
			hasTitle = true
		}
		if a == "--input-ipc-server=/tmp/test.sock" {
			hasSocket = true
		}
	}
	require.True(t, hasURL, "URL not in args")
	require.True(t, hasTitle, "title not in args")
	require.True(t, hasSocket, "--input-ipc-server not in args")
}

func TestBuildCmdWithResume(t *testing.T) {
	cmd := player.BuildCmd("mpv", nil, "https://example.com/video", "Test Movie", 120, "")
	var hasStart bool
	for _, a := range cmd.Args {
		if a == "--start=120" {
			hasStart = true
		}
	}
	require.True(t, hasStart, "--start=120 not in args: %v", cmd.Args)
}

func TestBuildCmdNoStartFlagWhenZero(t *testing.T) {
	cmd := player.BuildCmd("mpv", nil, "https://example.com/video", "Test Movie", 0, "")
	for _, a := range cmd.Args {
		require.NotContains(t, a, "--start=", "unexpected --start flag when startSec=0: %v", cmd.Args)
	}
}

func TestBuildCmdNoSocketWhenPathEmpty(t *testing.T) {
	cmd := player.BuildCmd("mpv", nil, "https://example.com/video", "Test Movie", 0, "")
	for _, a := range cmd.Args {
		require.NotContains(t, a, "--input-ipc-server", "unexpected socket flag when socketPath empty: %v", cmd.Args)
	}
}

func TestBuildCmdNonMpvNoSocketOrTitle(t *testing.T) {
	cmd := player.BuildCmd("vlc", nil, "https://example.com/video", "Test Movie", 0, "/tmp/test.sock")
	for _, a := range cmd.Args {
		require.NotContains(t, a, "--input-ipc-server", "non-mpv should not get IPC socket arg")
		require.NotContains(t, a, "--title=", "non-mpv should not get title arg")
	}
}

func TestSocketPath(t *testing.T) {
	path := player.SocketPath()
	require.Contains(t, path, "mpv-fin-")
	require.Contains(t, path, ".sock")
}
