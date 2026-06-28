// internal/api/client.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var ErrUnauthorized = errors.New("jellyfin: unauthorized")
var ErrNotFound = errors.New("jellyfin: not found")

var (
	_deviceID   string
	_deviceOnce sync.Once
)

func deviceID() string {
	_deviceOnce.Do(func() {
		for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
			if b, err := os.ReadFile(p); err == nil {
				id := strings.TrimSpace(string(b))
				if len(id) > 16 {
					id = id[:16]
				}
				_deviceID = "fin-" + id
				return
			}
		}
		if h, err := os.Hostname(); err == nil {
			_deviceID = "fin-" + h
		} else {
			_deviceID = "fin-cli"
		}
	})
	return _deviceID
}

func clientHeader() string {
	return fmt.Sprintf(`MediaBrowser Client="fin", Device="terminal", DeviceId="%s", Version="1.0.0"`, deviceID())
}

type Client struct {
	baseURL    string
	userID     string
	token      string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) SetAuth(userID, token string) {
	c.userID = userID
	c.token = token
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Emby-Authorization", clientHeader())
	if c.token != "" {
		req.Header.Set("X-Emby-Token", c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == 401 {
		return ErrUnauthorized
	}
	if resp.StatusCode == 404 {
		return ErrNotFound
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("jellyfin: HTTP %d for %s", resp.StatusCode, path)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) getRaw(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Emby-Authorization", clientHeader())
	if c.token != "" {
		req.Header.Set("X-Emby-Token", c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == 401 {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("jellyfin: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) post(ctx context.Context, path string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Emby-Authorization", clientHeader())
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == 401 {
		return ErrUnauthorized
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("jellyfin: HTTP %d for %s", resp.StatusCode, path)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) doNoResponse(ctx context.Context, method, path string) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Emby-Authorization", clientHeader())
	if c.token != "" {
		req.Header.Set("X-Emby-Token", c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == 401 {
		return ErrUnauthorized
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("jellyfin: HTTP %d for %s", resp.StatusCode, path)
	}
	return nil
}

func (c *Client) postNoResponse(ctx context.Context, path string, body io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Emby-Authorization", clientHeader())
	if c.token != "" {
		req.Header.Set("X-Emby-Token", c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == 401 {
		return ErrUnauthorized
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("jellyfin: HTTP %d for %s", resp.StatusCode, path)
	}
	return nil
}
