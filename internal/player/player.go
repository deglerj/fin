package player

import (
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type DoneMsg struct{ Err error }

func BuildCmd(command string, extraArgs []string, url, title string) *exec.Cmd {
	args := []string{url, fmt.Sprintf("--title=%s", title), fmt.Sprintf("--force-media-title=%s", title)}
	args = append(args, extraArgs...)
	return exec.Command(command, args...)
}

func Play(command string, extraArgs []string, url, title string) tea.Cmd {
	cmd := BuildCmd(command, extraArgs, url, title)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return DoneMsg{Err: err}
	})
}
