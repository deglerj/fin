//go:build !windows

package image

import (
	"os"

	"golang.org/x/term"
)

func makeRaw() (any, error) {
	return term.MakeRaw(int(os.Stdin.Fd()))
}

func restore(state any) {
	if s, ok := state.(*term.State); ok {
		_ = term.Restore(int(os.Stdin.Fd()), s)
	}
}
