// internal/ui/msg/msg.go
package msg

import "github.com/deglerj/fin/internal/api"

// Screen navigation
type LoginSuccess struct {
	ServerURL   string
	UserID      string
	AccessToken string
}
type LoginError struct{ Err error }
type TokenValid struct{}
type TokenInvalid struct{}

// Browser navigation
type LibrariesLoaded struct{ Libraries []api.Library }
type ItemsLoaded struct {
	Items     []api.Item
	ParentID  string
	LevelName string
}
type PushLevel struct {
	Items     []api.Item
	LevelName string
	ParentID  string
}
type PopLevel struct{}
type FetchVirtualSection struct{ ID string }
type RefreshLevel struct {
	Items    []api.Item
	ParentID string
}

// Overlays
type OpenDetails struct{ Item api.Item }
type ItemDetailLoaded struct{ Item api.Item }
type ImageLoaded struct {
	Data   []byte
	ItemId string
}
type OpenSearch struct{}
type SearchResults struct{ Items []api.Item }
type NavigateToItem struct{ Item api.Item }
type CloseOverlay struct{}

// Playback
type PlayItem struct{ Item api.Item }
type PlayerDone struct{ Err error }

// Played status
type PlayedToggled struct {
	ItemID string
	Played bool
}

// Error
type AppError struct{ Err error }
type DismissError struct{}
