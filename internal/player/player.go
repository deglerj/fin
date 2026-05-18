package player

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

type DoneMsg struct{ Err error }

func SocketPath() string {
	return fmt.Sprintf("/tmp/mpv-fin-%d.sock", os.Getpid())
}

func BuildCmd(command string, extraArgs []string, url, title string, startSec int64, socketPath string) *exec.Cmd {
	args := []string{url}
	if filepath.Base(command) == "mpv" {
		args = append(args, fmt.Sprintf("--title=%s", title), fmt.Sprintf("--force-media-title=%s", title))
		if startSec > 0 {
			args = append(args, fmt.Sprintf("--start=%d", startSec))
		}
		if socketPath != "" {
			args = append(args, fmt.Sprintf("--input-ipc-server=%s", socketPath))
		}
	}
	args = append(args, extraArgs...)
	return exec.Command(command, args...)
}

func Play(command string, extraArgs []string, url, title string, startSec int64, socketPath string) tea.Cmd {
	cmd := BuildCmd(command, extraArgs, url, title, startSec, socketPath)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return DoneMsg{Err: err}
	})
}
