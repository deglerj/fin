// internal/api/endpoints.go
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

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
	err = c.post(ctx, "/Users/AuthenticateByName", bytes.NewReader(b), &resp)
	return resp, err
}

func (c *Client) ValidateToken(ctx context.Context) error {
	var result map[string]any
	return c.get(ctx, fmt.Sprintf("/Users/%s", c.userID), &result)
}

func (c *Client) GetLibraries(ctx context.Context) ([]Library, error) {
	var resp LibraryResponse
	err := c.get(ctx, "/Library/MediaFolders", &resp)
	return resp.Items, err
}

func (c *Client) GetItems(ctx context.Context, parentID string, itemTypes []string) ([]Item, error) {
	const pageSize = 500
	typeFilter := ""
	if len(itemTypes) > 0 {
		typeFilter = "&IncludeItemTypes=" + strings.Join(itemTypes, ",")
	}
	var all []Item
	for startIndex := 0; ; startIndex += pageSize {
		q := fmt.Sprintf("/Users/%s/Items?ParentId=%s&Limit=%d&StartIndex=%d&Fields=Overview,People,CommunityRating,ProductionYear%s",
			c.userID, parentID, pageSize, startIndex, typeFilter)
		var resp ItemsResponse
		if err := c.get(ctx, q, &resp); err != nil {
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
	err := c.get(ctx, fmt.Sprintf("/Users/%s/Items/%s", c.userID, id), &item)
	return item, err
}

func (c *Client) Search(ctx context.Context, term string) ([]Item, error) {
	q := fmt.Sprintf("/Items?searchTerm=%s&IncludeItemTypes=Movie,Series,Episode&Recursive=true&UserId=%s&Limit=20&Fields=Overview,People,CommunityRating,ProductionYear",
		url.QueryEscape(term), c.userID)
	var resp ItemsResponse
	err := c.get(ctx, q, &resp)
	return resp.Items, err
}

func (c *Client) GetImage(ctx context.Context, itemID string, maxWidth, maxHeight int, tag string) ([]byte, error) {
	path := fmt.Sprintf("/Items/%s/Images/Primary?MaxWidth=%d&MaxHeight=%d", itemID, maxWidth, maxHeight)
	if tag != "" {
		path += "&tag=" + tag
	}
	return c.getRaw(ctx, path)
}

func (c *Client) GetNextUp(ctx context.Context) ([]Item, error) {
	q := fmt.Sprintf("/Shows/NextUp?UserId=%s&Limit=20&Fields=Overview,People,CommunityRating,ProductionYear", c.userID)
	var resp ItemsResponse
	err := c.get(ctx, q, &resp)
	return resp.Items, err
}

func (c *Client) GetResumeItems(ctx context.Context) ([]Item, error) {
	q := fmt.Sprintf("/UserItems/Resume?UserId=%s&Limit=20&Fields=Overview,People,CommunityRating,ProductionYear", c.userID)
	var resp ItemsResponse
	err := c.get(ctx, q, &resp)
	return resp.Items, err
}

func (c *Client) GetLatestMedia(ctx context.Context) ([]Item, error) {
	q := fmt.Sprintf("/Users/%s/Items/Latest?Limit=20&Fields=Overview,People,CommunityRating,ProductionYear", c.userID)
	var items []Item
	err := c.get(ctx, q, &items)
	return items, err
}

func (c *Client) GetFavorites(ctx context.Context) ([]Item, error) {
	q := fmt.Sprintf("/Items?isFavorite=true&Recursive=true&UserId=%s&Limit=20&Fields=Overview,People,CommunityRating,ProductionYear", c.userID)
	var resp ItemsResponse
	err := c.get(ctx, q, &resp)
	return resp.Items, err
}

func (c *Client) StreamURL(item Item) string {
	if item.Type == "Audio" || item.MediaType == "Audio" {
		return fmt.Sprintf("%s/Audio/%s/stream?api_key=%s&static=true", c.baseURL, item.Id, c.token)
	}
	return fmt.Sprintf("%s/Videos/%s/stream?api_key=%s&static=true", c.baseURL, item.Id, c.token)
}

func (c *Client) MarkPlayed(ctx context.Context, itemID string) error {
	return c.doNoResponse(ctx, "POST", fmt.Sprintf("/Users/%s/PlayedItems/%s", c.userID, itemID))
}

func (c *Client) MarkUnplayed(ctx context.Context, itemID string) error {
	return c.doNoResponse(ctx, "DELETE", fmt.Sprintf("/Users/%s/PlayedItems/%s", c.userID, itemID))
}
