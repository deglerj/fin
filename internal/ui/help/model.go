// internal/ui/help/model.go
package help

import (
	"github.com/deglerj/fin/internal/ui/styles"
)

const helpText = `
  ↑ / ↓       Navigate list
  → / Enter   Open / drill in / play
  ← / Esc     Back / close overlay
  i           Details overlay
  /           Search
  r           Random from current list
  ?           Toggle this help
  q           Quit
`

func View() string {
	return styles.Overlay.Render(styles.Dim.Render(helpText))
}
