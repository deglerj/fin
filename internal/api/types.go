// internal/api/types.go
package api

type AuthResponse struct {
	User        UserInfo `json:"User"`
	AccessToken string   `json:"AccessToken"`
}

type UserInfo struct {
	Id   string `json:"Id"`
	Name string `json:"Name"`
}

type Item struct {
	Id                string   `json:"Id"`
	Name              string   `json:"Name"`
	Type              string   `json:"Type"`   // Movie, Series, Season, Episode, Audio
	MediaType         string   `json:"MediaType"`
	SeriesName        string   `json:"SeriesName"`
	SeasonName        string   `json:"SeasonName"`
	IndexNumber       int      `json:"IndexNumber"`
	ParentIndexNumber int      `json:"ParentIndexNumber"`
	RunTimeTicks      int64    `json:"RunTimeTicks"`
	Overview          string   `json:"Overview"`
	CommunityRating   float64  `json:"CommunityRating"`
	ProductionYear    int      `json:"ProductionYear"`
	People            []Person `json:"People"`
	UserData          UserData `json:"UserData"`
}

type Person struct {
	Name string `json:"Name"`
	Role string `json:"Role"`
	Type string `json:"Type"`
}

type UserData struct {
	Played bool `json:"Played"`
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
