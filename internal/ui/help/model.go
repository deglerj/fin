// internal/ui/help/model.go
package help

import (
	"github.com/deglerj/fin/internal/ui/styles"
)

const helpText = `
  ↑ / ↓       Navigate list
  PgUp/PgDn   Page through list
  → / Enter   Open / drill in / play
  ⌫ / ← / Esc  Back / close overlay
  J / K       Scroll details text
  /           Search
  r           Play random item (recurses into series/seasons)
  ?           Toggle this help
  q           Quit
`

func View() string {
	return styles.Overlay.Render(styles.Dim.Render(helpText))
}
