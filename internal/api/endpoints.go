// internal/api/endpoints.go
package api

import (
	"fmt"
	"strings"
)

func (c *Client) Authenticate(username, password string) (AuthResponse, error) {
	body := fmt.Sprintf(`{"Username":%q,"Pw":%q}`, username, password)
	var resp AuthResponse
	err := c.post("/Users/AuthenticateByName", strings.NewReader(body), &resp)
	return resp, err
}

func (c *Client) ValidateToken() error {
	var result map[string]any
	return c.get(fmt.Sprintf("/Users/%s", c.userID), &result)
}

func (c *Client) GetLibraries() ([]Library, error) {
	var resp LibraryResponse
	err := c.get("/Library/MediaFolders", &resp)
	return resp.Items, err
}

func (c *Client) GetItems(parentID string, itemTypes []string) ([]Item, error) {
	q := fmt.Sprintf("/Users/%s/Items?ParentId=%s&Limit=500", c.userID, parentID)
	if len(itemTypes) > 0 {
		q += "&IncludeItemTypes=" + strings.Join(itemTypes, ",")
	}
	var resp ItemsResponse
	err := c.get(q, &resp)
	return resp.Items, err
}

func (c *Client) GetItem(id string) (Item, error) {
	var item Item
	err := c.get(fmt.Sprintf("/Users/%s/Items/%s", c.userID, id), &item)
	return item, err
}

func (c *Client) Search(term string) ([]Item, error) {
	q := fmt.Sprintf("/Items?searchTerm=%s&IncludeItemTypes=Movie,Series,Episode&Recursive=true&UserId=%s&Limit=20",
		term, c.userID)
	var resp ItemsResponse
	err := c.get(q, &resp)
	return resp.Items, err
}

func (c *Client) GetImage(itemID string, maxWidth int) ([]byte, error) {
	return c.getRaw(fmt.Sprintf("/Items/%s/Images/Primary?MaxWidth=%d", itemID, maxWidth))
}

func (c *Client) StreamURL(item Item) string {
	if item.Type == "Audio" || item.MediaType == "Audio" {
		return fmt.Sprintf("%s/Audio/%s/stream?api_key=%s&static=true", c.baseURL, item.Id, c.token)
	}
	return fmt.Sprintf("%s/Videos/%s/stream?api_key=%s&static=true", c.baseURL, item.Id, c.token)
}
