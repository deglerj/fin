package player

import (
	"bufio"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// listenUnix creates a temporary Unix socket listener and registers cleanup.
func listenUnix(t *testing.T) (net.Listener, string) {
	t.Helper()
	addr := filepath.Join(t.TempDir(), "mpv.sock")
	ln, err := net.Listen("unix", addr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })
	return ln, addr
}

func TestQueryPositionReturnsExpectedTicks(t *testing.T) {
	ln, addr := listenUnix(t)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			_, _ = fmt.Fprintf(conn, "{\"request_id\":1,\"data\":60.0,\"error\":\"success\"}\n")
		}
	}()

	conn, err := net.Dial("unix", addr)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	ticks, err := queryPosition(conn, bufio.NewScanner(conn))
	require.NoError(t, err)
	require.Equal(t, int64(600_000_000), ticks) // 60s × 10_000_000 ticks/s
}

func TestQueryPositionSkipsEventLines(t *testing.T) {
	ln, addr := listenUnix(t)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			// Send an mpv event first, then the actual response.
			_, _ = fmt.Fprintf(conn, "{\"event\":\"property-change\",\"name\":\"pause\"}\n")
			_, _ = fmt.Fprintf(conn, "{\"request_id\":1,\"data\":30.5,\"error\":\"success\"}\n")
		}
	}()

	conn, err := net.Dial("unix", addr)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	ticks, err := queryPosition(conn, bufio.NewScanner(conn))
	require.NoError(t, err)
	require.Equal(t, int64(305_000_000), ticks) // 30.5s × 10_000_000
}

func TestQueryPositionErrorOnClosedSocket(t *testing.T) {
	ln, addr := listenUnix(t)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		_ = conn.Close() // close immediately — simulates mpv exit
	}()

	conn, err := net.Dial("unix", addr)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()

	_, err = queryPosition(conn, bufio.NewScanner(conn))
	require.Error(t, err)
}

func TestWaitForSocketConnectsAfterDelay(t *testing.T) {
	addr := filepath.Join(t.TempDir(), "delayed.sock")

	go func() {
		time.Sleep(50 * time.Millisecond)
		ln, err := net.Listen("unix", addr)
		if err != nil {
			return
		}
		defer func() { _ = ln.Close() }()
		conn, _ := ln.Accept()
		if conn != nil {
			_ = conn.Close()
		}
	}()

	conn, err := waitForSocket(addr, 2*time.Second)
	require.NoError(t, err)
	_ = conn.Close()
}

func TestWaitForSocketTimesOut(t *testing.T) {
	addr := filepath.Join(t.TempDir(), "nonexistent.sock")
	_, err := waitForSocket(addr, 150*time.Millisecond)
	require.Error(t, err)
}
