// internal/api/types.go
package api

import "fmt"

type AuthResponse struct {
	User        UserInfo `json:"User"`
	AccessToken string   `json:"AccessToken"`
}

type UserInfo struct {
	Id   string `json:"Id"`
	Name string `json:"Name"`
}

type Item struct {
	Id                string            `json:"Id"`
	Name              string            `json:"Name"`
	Type              string            `json:"Type"` // Movie, Series, Season, Episode, Audio
	MediaType         string            `json:"MediaType"`
	SeriesName        string            `json:"SeriesName"`
	SeasonName        string            `json:"SeasonName"`
	IndexNumber       int               `json:"IndexNumber"`
	ParentIndexNumber int               `json:"ParentIndexNumber"`
	RunTimeTicks      int64             `json:"RunTimeTicks"`
	Overview          string            `json:"Overview"`
	CommunityRating   float64           `json:"CommunityRating"`
	ProductionYear    int               `json:"ProductionYear"`
	People            []Person          `json:"People"`
	UserData          UserData          `json:"UserData"`
	ImageTags         map[string]string `json:"ImageTags"`
}

func (item Item) MediaTitle() string {
	if item.Type == "Episode" {
		return fmt.Sprintf("%s S%02dE%02d %s", item.SeriesName, item.ParentIndexNumber, item.IndexNumber, item.Name)
	}
	return item.Name
}

type Person struct {
	Name string `json:"Name"`
	Role string `json:"Role"`
	Type string `json:"Type"`
}

type UserData struct {
	Played                bool  `json:"Played"`
	PlaybackPositionTicks int64 `json:"PlaybackPositionTicks"`
}

type ItemsResponse struct {
	Items            []Item `json:"Items"`
	TotalRecordCount int    `json:"TotalRecordCount"`
}

type Library struct {
	Id             string `json:"Id"`
	Name           string `json:"Name"`
	CollectionType string `json:"CollectionType"`
}

type LibraryResponse struct {
	Items []Library `json:"Items"`
}
