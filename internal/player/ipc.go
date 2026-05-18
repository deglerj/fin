package player

import (
	"bufio"
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/deglerj/fin/internal/api"
)

// Monitor connects to mpv's IPC socket and reports playback progress to Jellyfin
// until mpv exits. Intended to be run as a goroutine alongside tea.ExecProcess.
// Exits silently if the socket never appears (non-mpv player or mpv crash at launch).
func Monitor(socketPath string, client *api.Client, item api.Item, startSec int64) {
	conn, err := waitForSocket(socketPath, 3*time.Second)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	sessionID := newSessionID()
	ctx := context.Background()
	startTicks := startSec * 10_000_000
	report := api.PlaybackReport{
		ItemId:        item.Id,
		PlaySessionId: sessionID,
		MediaSourceId: item.Id,
		PositionTicks: startTicks,
		CanSeek:       true,
		PlayMethod:    "DirectStream",
		RepeatMode:    "RepeatNone",
	}

	_ = client.ReportPlaybackStart(ctx, report)

	scanner := bufio.NewScanner(conn)
	lastTicks := startTicks
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		ticks, err := queryPosition(conn, scanner)
		if err != nil {
			break
		}
		lastTicks = ticks
		report.PositionTicks = ticks
		_ = client.ReportPlaybackProgress(ctx, report)
	}

	report.PositionTicks = lastTicks
	_ = client.ReportPlaybackStopped(ctx, report)
}

func waitForSocket(path string, timeout time.Duration) (net.Conn, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("unix", path)
		if err == nil {
			return conn, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("socket %s did not appear within %s", path, timeout)
}

// queryPosition sends a get_property command to mpv and returns the current
// playback position in Jellyfin ticks (100-nanosecond intervals).
// Returns an error if the connection is closed (mpv has exited).
func queryPosition(conn net.Conn, scanner *bufio.Scanner) (int64, error) {
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	_, err := fmt.Fprintf(conn, "{\"command\":[\"get_property\",\"playback-time\"],\"request_id\":1}\n")
	if err != nil {
		return 0, err
	}

	for scanner.Scan() {
		var resp struct {
			Data      *float64 `json:"data"`
			RequestID int      `json:"request_id"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			continue // skip unparseable lines (mpv event messages, etc.)
		}
		if resp.RequestID == 1 && resp.Data != nil {
			return int64(*resp.Data * 10_000_000), nil
		}
	}
	return 0, fmt.Errorf("connection closed: %w", scanner.Err())
}

func newSessionID() string {
	b := make([]byte, 16)
	_, _ = crand.Read(b)
	return hex.EncodeToString(b)
}
