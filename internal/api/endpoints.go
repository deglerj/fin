// internal/api/endpoints.go
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// itemFields is the detail set every item listing asks for, so the details
// pane can render without a second round trip.
const itemFields = "Overview,People,CommunityRating,ProductionYear"

// userItemsPath builds /Users/{id}/Items with the standard detail fields.
func (c *Client) userItemsPath(q url.Values) string {
	q.Set("Fields", itemFields)
	return "/Users/" + url.PathEscape(c.userID) + "/Items?" + q.Encode()
}

// itemsPath builds the user-scoped /Items query with the standard detail fields.
func (c *Client) itemsPath(q url.Values) string {
	q.Set("UserId", c.userID)
	q.Set("Fields", itemFields)
	return "/Items?" + q.Encode()
}

func (c *Client) Authenticate(ctx context.Context, username, password string) (AuthResponse, error) {
	payload := struct {
		Username string `json:"Username"`
		Pw       string `json:"Pw"`
	}{username, password}
	b, err := json.Marshal(payload)
	if err != nil {
		return AuthResponse{}, err
	}
	var resp AuthResponse
	err = c.decode(ctx, http.MethodPost, "/Users/AuthenticateByName", bytes.NewReader(b), &resp)
	return resp, err
}

func (c *Client) ValidateToken(ctx context.Context) error {
	var result map[string]any
	return c.get(ctx, "/Users/"+url.PathEscape(c.userID), &result)
}

func (c *Client) GetLibraries(ctx context.Context) ([]Library, error) {
	var resp LibraryResponse
	err := c.get(ctx, "/Library/MediaFolders", &resp)
	return resp.Items, err
}

func (c *Client) GetItems(ctx context.Context, parentID string, itemTypes []string) ([]Item, error) {
	const pageSize = 500
	var all []Item
	for startIndex := 0; ; startIndex += pageSize {
		q := url.Values{
			"ParentId":   {parentID},
			"Limit":      {strconv.Itoa(pageSize)},
			"StartIndex": {strconv.Itoa(startIndex)},
		}
		if len(itemTypes) > 0 {
			q.Set("IncludeItemTypes", strings.Join(itemTypes, ","))
		}
		var resp ItemsResponse
		if err := c.get(ctx, c.userItemsPath(q), &resp); err != nil {
			return nil, err
		}
		all = append(all, resp.Items...)
		if len(resp.Items) < pageSize || len(all) >= resp.TotalRecordCount {
			break
		}
	}
	return all, nil
}

func (c *Client) GetItem(ctx context.Context, id string) (Item, error) {
	var item Item
	err := c.get(ctx, "/Users/"+url.PathEscape(c.userID)+"/Items/"+url.PathEscape(id), &item)
	return item, err
}

func (c *Client) Search(ctx context.Context, term string) ([]Item, error) {
	q := url.Values{
		"searchTerm":       {term},
		"IncludeItemTypes": {"Movie,Series,Episode"},
		"Recursive":        {"true"},
		"Limit":            {"20"},
	}
	var resp ItemsResponse
	err := c.get(ctx, c.itemsPath(q), &resp)
	return resp.Items, err
}

func (c *Client) GetImage(ctx context.Context, itemID string, maxWidth, maxHeight int, tag string) ([]byte, error) {
	q := url.Values{
		"MaxWidth":  {strconv.Itoa(maxWidth)},
		"MaxHeight": {strconv.Itoa(maxHeight)},
	}
	if tag != "" {
		q.Set("tag", tag)
	}
	return c.getRaw(ctx, "/Items/"+url.PathEscape(itemID)+"/Images/Primary?"+q.Encode())
}

func (c *Client) GetNextUp(ctx context.Context) ([]Item, error) {
	q := url.Values{"UserId": {c.userID}, "Limit": {"20"}, "Fields": {itemFields}}
	var resp ItemsResponse
	err := c.get(ctx, "/Shows/NextUp?"+q.Encode(), &resp)
	return resp.Items, err
}

func (c *Client) GetResumeItems(ctx context.Context) ([]Item, error) {
	q := url.Values{"UserId": {c.userID}, "Limit": {"20"}, "Fields": {itemFields}}
	var resp ItemsResponse
	err := c.get(ctx, "/UserItems/Resume?"+q.Encode(), &resp)
	return resp.Items, err
}

func (c *Client) GetLatestMedia(ctx context.Context) ([]Item, error) {
	q := url.Values{"Limit": {"20"}, "Fields": {itemFields}}
	var items []Item
	err := c.get(ctx, "/Users/"+url.PathEscape(c.userID)+"/Items/Latest?"+q.Encode(), &items)
	return items, err
}

func (c *Client) GetFavorites(ctx context.Context) ([]Item, error) {
	q := url.Values{"isFavorite": {"true"}, "Recursive": {"true"}, "Limit": {"20"}}
	var resp ItemsResponse
	err := c.get(ctx, c.itemsPath(q), &resp)
	return resp.Items, err
}

func (c *Client) GetRandomLeaf(ctx context.Context, parentID string) (Item, error) {
	q := url.Values{
		"ParentId":         {parentID},
		"Recursive":        {"true"},
		"IncludeItemTypes": {"Movie,Episode,Audio"},
		"SortBy":           {"Random"},
		"Limit":            {"1"},
	}
	var resp ItemsResponse
	if err := c.get(ctx, c.userItemsPath(q), &resp); err != nil {
		return Item{}, err
	}
	if len(resp.Items) == 0 {
		return Item{}, fmt.Errorf("no playable items found")
	}
	return resp.Items[0], nil
}

func (c *Client) GetIntroTimestamps(ctx context.Context, itemID string) (IntroTimestamps, error) {
	var ts IntroTimestamps
	err := c.get(ctx, "/Episode/"+url.PathEscape(itemID)+"/IntroTimestamps", &ts)
	if errors.Is(err, ErrNotFound) {
		return IntroTimestamps{}, nil
	}
	return ts, err
}

// StreamURL embeds the access token as a query parameter. Callers must keep it
// out of a player's argv — see player.BuildCmd, which routes it through a
// playlist file so the token never lands in `ps` output.
func (c *Client) StreamURL(item Item) string {
	kind := "Videos"
	if item.Type == "Audio" || item.MediaType == "Audio" {
		kind = "Audio"
	}
	q := url.Values{"api_key": {c.token}, "static": {"true"}}
	return c.baseURL + "/" + kind + "/" + url.PathEscape(item.Id) + "/stream?" + q.Encode()
}

func (c *Client) MarkPlayed(ctx context.Context, itemID string) error {
	return c.decode(ctx, http.MethodPost, c.playedItemPath(itemID), nil, nil)
}

func (c *Client) MarkUnplayed(ctx context.Context, itemID string) error {
	return c.decode(ctx, http.MethodDelete, c.playedItemPath(itemID), nil, nil)
}

func (c *Client) playedItemPath(itemID string) string {
	return "/Users/" + url.PathEscape(c.userID) + "/PlayedItems/" + url.PathEscape(itemID)
}

func (c *Client) ReportPlaybackStart(ctx context.Context, r PlaybackReport) error {
	return c.reportPlayback(ctx, "/Sessions/Playing", r)
}

func (c *Client) ReportPlaybackProgress(ctx context.Context, r PlaybackReport) error {
	return c.reportPlayback(ctx, "/Sessions/Playing/Progress", r)
}

func (c *Client) ReportPlaybackStopped(ctx context.Context, r PlaybackReport) error {
	return c.reportPlayback(ctx, "/Sessions/Playing/Stopped", r)
}

func (c *Client) reportPlayback(ctx context.Context, path string, r PlaybackReport) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return c.decode(ctx, http.MethodPost, path, bytes.NewReader(b), nil)
}
