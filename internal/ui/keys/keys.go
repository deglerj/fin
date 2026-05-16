// internal/ui/keys/keys.go
package keys

import "github.com/charmbracelet/bubbles/key"

type Bindings struct {
	Up     key.Binding
	Down   key.Binding
	Right  key.Binding
	Left   key.Binding
	Play   key.Binding
	Back   key.Binding
	Search key.Binding
	Random key.Binding
	Help   key.Binding
	Quit   key.Binding
}

var Default = Bindings{
	Up:     key.NewBinding(key.WithKeys("up")),
	Down:   key.NewBinding(key.WithKeys("down")),
	Right:  key.NewBinding(key.WithKeys("right", "enter")),
	Left:   key.NewBinding(key.WithKeys("left", "esc")),
	Play:   key.NewBinding(key.WithKeys("enter")),
	Back:   key.NewBinding(key.WithKeys("esc", "left", "backspace")),
	Search: key.NewBinding(key.WithKeys("/")),
	Random: key.NewBinding(key.WithKeys("r")),
	Help:   key.NewBinding(key.WithKeys("?")),
	Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c")),
}
