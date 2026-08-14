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

// Version is reported to Jellyfin in the client header and shows up in the
// server's device list. main overrides it with the real build version.
var Version = "dev"

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
	return fmt.Sprintf(`MediaBrowser Client="fin", Device="terminal", DeviceId="%s", Version="%s"`, deviceID(), Version)
}

type Client struct {
	baseURL    string
	userID     string
	token      string
	httpClient *http.Client
}

// New returns a client for baseURL. A URL with no scheme is assumed to be
// https, so "jellyfin.example.com" works as well as the full form — otherwise
// the request URL parses as relative and Do fails with an opaque
// "unsupported protocol scheme" error.
func New(baseURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL != "" && !strings.Contains(baseURL, "://") {
		baseURL = "https://" + baseURL
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) SetAuth(userID, token string) {
	c.userID = userID
	c.token = token
}

// do issues an authenticated request and maps HTTP status codes to errors. On
// success the caller owns the response body and must close it.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Emby-Authorization", clientHeader())
	if c.token != "" {
		req.Header.Set("X-Emby-Token", c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		_ = resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, ErrUnauthorized
		case http.StatusNotFound:
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("jellyfin: HTTP %d for %s", resp.StatusCode, path)
	}
	return resp, nil
}

// decode issues a request and JSON-decodes the response into out. Pass a nil
// out for endpoints whose response body is not needed.
func (c *Client) decode(ctx context.Context, method, path string, body io.Reader, out any) error {
	resp, err := c.do(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.decode(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) getRaw(ctx context.Context, path string) ([]byte, error) {
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return io.ReadAll(resp.Body)
}
