// internal/ui/keys/keys.go
package keys

import "github.com/charmbracelet/bubbles/key"

type Bindings struct {
	Up         key.Binding
	Down       key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	Open       key.Binding
	Back       key.Binding
	Search     key.Binding
	Random     key.Binding
	MarkPlayed key.Binding
}

var Default = Bindings{
	Up:         key.NewBinding(key.WithKeys("up")),
	Down:       key.NewBinding(key.WithKeys("down")),
	PageUp:     key.NewBinding(key.WithKeys("pgup")),
	PageDown:   key.NewBinding(key.WithKeys("pgdown")),
	Open:       key.NewBinding(key.WithKeys("enter", "right")),
	Back:       key.NewBinding(key.WithKeys("esc", "left", "backspace")),
	Search:     key.NewBinding(key.WithKeys("/")),
	Random:     key.NewBinding(key.WithKeys("r")),
	MarkPlayed: key.NewBinding(key.WithKeys("m")),
}
