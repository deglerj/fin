package player_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/deglerj/fin/internal/api"
	"github.com/deglerj/fin/internal/player"
	"github.com/stretchr/testify/require"
)

// build runs BuildCmd and schedules the temp files it created for removal.
func build(t *testing.T, o player.Options) []string {
	t.Helper()
	cmd, temps := player.BuildCmd(o)
	t.Cleanup(func() {
		for _, p := range temps {
			_ = os.Remove(p)
		}
	})
	return cmd.Args
}

// argValue returns the value of the first --flag=value argument for flag.
func argValue(args []string, flag string) string {
	for _, a := range args {
		if v, ok := strings.CutPrefix(a, flag+"="); ok {
			return v
		}
	}
	return ""
}

func TestBuildCmd(t *testing.T) {
	args := build(t, player.Options{
		Command:    "mpv",
		ExtraArgs:  []string{"--really-quiet"},
		URL:        "https://example.com/video",
		Title:      "Test Movie",
		SocketPath: "/tmp/test.sock",
	})
	require.Equal(t, "mpv", args[0])
	require.Contains(t, args, "--title=Test Movie")
	require.Contains(t, args, "--input-ipc-server=/tmp/test.sock")
	require.Contains(t, args, "--really-quiet")
}

func TestBuildCmdKeepsStreamURLOutOfArgv(t *testing.T) {
	url := "https://example.com/video?api_key=secrettoken&static=true"
	args := build(t, player.Options{Command: "mpv", URL: url, Title: "Test Movie"})

	for _, a := range args {
		require.NotContains(t, a, "secrettoken", "access token leaked into argv: %v", args)
	}

	playlist := argValue(args, "--playlist")
	require.NotEmpty(t, playlist, "expected a --playlist file: %v", args)
	data, err := os.ReadFile(playlist)
	require.NoError(t, err)
	require.Equal(t, url+"\n", string(data))
}

func TestBuildCmdWithResume(t *testing.T) {
	args := build(t, player.Options{Command: "mpv", URL: "https://example.com/video", StartSec: 120})
	require.Contains(t, args, "--start=120")
}

func TestBuildCmdNoStartFlagWhenZero(t *testing.T) {
	for _, a := range build(t, player.Options{Command: "mpv", URL: "https://example.com/video"}) {
		require.NotContains(t, a, "--start=")
	}
}

func TestBuildCmdNoSocketWhenPathEmpty(t *testing.T) {
	for _, a := range build(t, player.Options{Command: "mpv", URL: "https://example.com/video"}) {
		require.NotContains(t, a, "--input-ipc-server")
	}
}

func TestBuildCmdNonMpvGetsPlainURL(t *testing.T) {
	args := build(t, player.Options{
		Command:    "vlc",
		URL:        "https://example.com/video",
		Title:      "Test Movie",
		SocketPath: "/tmp/test.sock",
	})
	require.Contains(t, args, "https://example.com/video")
	for _, a := range args {
		require.NotContains(t, a, "--input-ipc-server")
		require.NotContains(t, a, "--title=")
		require.NotContains(t, a, "--playlist=")
	}
}

func TestBuildCmdWritesChapterFile(t *testing.T) {
	args := build(t, player.Options{
		Command: "mpv",
		URL:     "https://example.com/video",
		Chapters: []api.ChapterInfo{
			{StartPositionTicks: 0, Name: "Intro"},
			{StartPositionTicks: 100_000_000, Name: "Act 1"},  // 10 seconds
			{StartPositionTicks: 600_000_000, Name: "Climax"}, // 60 seconds
		},
	})
	path := argValue(args, "--chapters-file")
	require.NotEmpty(t, path, "expected a --chapters-file: %v", args)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, strings.Join([]string{
		"CHAPTER01=00:00:00.000",
		"CHAPTER01NAME=Intro",
		"CHAPTER02=00:00:10.000",
		"CHAPTER02NAME=Act 1",
		"CHAPTER03=00:01:00.000",
		"CHAPTER03NAME=Climax",
		"",
	}, "\n"), string(data))
}

func TestBuildCmdNoChapterFileWithoutChapters(t *testing.T) {
	for _, a := range build(t, player.Options{Command: "mpv", URL: "https://example.com/video"}) {
		require.NotContains(t, a, "--chapters-file")
	}
}

func TestBuildCmdTempFilesAreReported(t *testing.T) {
	cmd, temps := player.BuildCmd(player.Options{
		Command:  "mpv",
		URL:      "https://example.com/video",
		Chapters: []api.ChapterInfo{{StartPositionTicks: 0, Name: "Intro"}},
	})
	require.Len(t, temps, 2, "playlist and chapter file should both be reported: %v", cmd.Args)
	for _, p := range temps {
		require.FileExists(t, p)
		_ = os.Remove(p)
	}
}

func TestMergeIntroChaptersSortsByPosition(t *testing.T) {
	merged := player.MergeIntroChapters(
		[]api.ChapterInfo{{StartPositionTicks: 0, Name: "Cold Open"}},
		api.IntroTimestamps{Valid: true, IntroStart: 10, IntroEnd: 90},
	)
	require.Len(t, merged, 3)
	require.Equal(t, "Cold Open", merged[0].Name)
	require.Equal(t, "Intro", merged[1].Name)
	require.Equal(t, int64(100_000_000), merged[1].StartPositionTicks)
	require.Equal(t, "After Intro", merged[2].Name)
	require.Equal(t, int64(900_000_000), merged[2].StartPositionTicks)
}

func TestMergeIntroChaptersDoesNotMutateInput(t *testing.T) {
	original := []api.ChapterInfo{{StartPositionTicks: 0, Name: "Cold Open"}}
	player.MergeIntroChapters(original, api.IntroTimestamps{IntroStart: 10, IntroEnd: 90})
	require.Len(t, original, 1)
}

func TestSocketPath(t *testing.T) {
	path := player.SocketPath()
	require.Contains(t, path, "mpv-fin-")
	require.Contains(t, path, ".sock")
	// Clean: macOS reports the temp dir with a trailing slash, which Join strips.
	require.Equal(t, filepath.Clean(os.TempDir()), filepath.Dir(path),
		"socket must live in the temp dir, not a hardcoded /tmp")
}
