package player

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/deglerj/fin/internal/api"
)

type DoneMsg struct{ Err error }

// Options describes one playback invocation.
type Options struct {
	Command    string
	ExtraArgs  []string
	URL        string
	Title      string
	StartSec   int64
	SocketPath string
	Chapters   []api.ChapterInfo
}

// SocketPath returns the mpv IPC socket for this process, or "" on platforms
// where a unix socket in the temp dir is not usable — progress reporting is
// simply skipped there.
func SocketPath() string {
	if runtime.GOOS == "windows" {
		return ""
	}
	return filepath.Join(os.TempDir(), fmt.Sprintf("mpv-fin-%d.sock", os.Getpid()))
}

// BuildCmd assembles the player command. It returns the temp files the caller
// must delete once playback finishes.
//
// The stream URL carries the Jellyfin access token, so for mpv it is handed
// over in a 0600 playlist file rather than as an argv element, where every
// other user on the machine could read it out of `ps`. Players other than mpv
// get the URL on the command line — fin knows no equivalent indirection for
// them.
func BuildCmd(o Options) (*exec.Cmd, []string) {
	if filepath.Base(o.Command) != "mpv" {
		return exec.Command(o.Command, append([]string{o.URL}, o.ExtraArgs...)...), nil
	}

	var args, temps []string
	if path, err := writeTemp("fin-playlist-*.m3u", o.URL+"\n"); err == nil {
		temps = append(temps, path)
		args = append(args, "--playlist="+path)
	} else {
		// Falling back to argv beats refusing to play; the token is exposed.
		args = append(args, o.URL)
	}
	args = append(args, "--title="+o.Title, "--force-media-title="+o.Title)
	if o.StartSec > 0 {
		args = append(args, fmt.Sprintf("--start=%d", o.StartSec))
	}
	if o.SocketPath != "" {
		args = append(args, "--input-ipc-server="+o.SocketPath)
	}
	if len(o.Chapters) > 0 {
		if path, err := writeTemp("fin-chapters-*.txt", chapterFile(o.Chapters)); err == nil {
			temps = append(temps, path)
			args = append(args, "--chapters-file="+path)
		}
	}
	args = append(args, o.ExtraArgs...)
	return exec.Command(o.Command, args...), temps
}

// Play returns the command that runs the player, plus the temp files to remove
// when DoneMsg arrives.
func Play(o Options) (tea.Cmd, []string) {
	cmd, temps := BuildCmd(o)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return DoneMsg{Err: err}
	}), temps
}

// MergeIntroChapters folds Intro Skipper's timestamps into an episode's chapter
// list as seekable marks, keeping the result sorted by position.
func MergeIntroChapters(chapters []api.ChapterInfo, ts api.IntroTimestamps) []api.ChapterInfo {
	merged := make([]api.ChapterInfo, 0, len(chapters)+2)
	merged = append(merged, chapters...)
	merged = append(merged,
		api.ChapterInfo{StartPositionTicks: int64(ts.IntroStart * 10_000_000), Name: "Intro"},
		api.ChapterInfo{StartPositionTicks: int64(ts.IntroEnd * 10_000_000), Name: "After Intro"},
	)
	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].StartPositionTicks < merged[j].StartPositionTicks
	})
	return merged
}

// chapterFile renders chapters in mpv's --chapters-file format.
func chapterFile(chapters []api.ChapterInfo) string {
	var sb []byte
	for i, c := range chapters {
		ms := c.StartPositionTicks / 10_000
		n := i + 1
		sb = fmt.Appendf(sb, "CHAPTER%02d=%02d:%02d:%02d.%03d\nCHAPTER%02dNAME=%s\n",
			n, ms/3_600_000, (ms%3_600_000)/60_000, (ms%60_000)/1_000, ms%1_000, n, c.Name)
	}
	return string(sb)
}

func writeTemp(pattern, content string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}
