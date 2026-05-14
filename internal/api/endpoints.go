// internal/api/endpoints.go
package api

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

func (c *Client) Authenticate(ctx context.Context, username, password string) (AuthResponse, error) {
	body := fmt.Sprintf(`{"Username":%q,"Pw":%q}`, username, password)
	var resp AuthResponse
	err := c.post(ctx, "/Users/AuthenticateByName", strings.NewReader(body), &resp)
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
		q := fmt.Sprintf("/Users/%s/Items?ParentId=%s&Limit=%d&StartIndex=%d%s",
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
	q := fmt.Sprintf("/Items?searchTerm=%s&IncludeItemTypes=Movie,Series,Episode&Recursive=true&UserId=%s&Limit=20",
		url.QueryEscape(term), c.userID)
	var resp ItemsResponse
	err := c.get(ctx, q, &resp)
	return resp.Items, err
}

func (c *Client) GetImage(ctx context.Context, itemID string, maxWidth int) ([]byte, error) {
	return c.getRaw(ctx, fmt.Sprintf("/Items/%s/Images/Primary?MaxWidth=%d", itemID, maxWidth))
}

func (c *Client) StreamURL(item Item) string {
	if item.Type == "Audio" || item.MediaType == "Audio" {
		return fmt.Sprintf("%s/Audio/%s/stream?api_key=%s&static=true", c.baseURL, item.Id, c.token)
	}
	return fmt.Sprintf("%s/Videos/%s/stream?api_key=%s&static=true", c.baseURL, item.Id, c.token)
}
