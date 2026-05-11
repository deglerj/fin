package image

import (
	"encoding/base64"
	"fmt"
	"os"
	"time"
)

var capable *bool

// Probe sends a kitty graphics query and waits for a response.
// Returns true if the terminal supports kitty graphics protocol.
// Result cached after first call.
func Probe() bool {
	if capable != nil {
		return *capable
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		t := true
		capable = &t
		return true
	}
	result := probeTerminal(100 * time.Millisecond)
	capable = &result
	return result
}

func probeTerminal(timeout time.Duration) bool {
	oldState, err := makeRaw()
	if err != nil {
		return false
	}
	defer restore(oldState)

	fmt.Fprint(os.Stdout, "\x1b_Ga=q,s=1,v=1,i=1;\x1b\\")

	ch := make(chan bool, 1)
	go func() {
		buf := make([]byte, 64)
		n, _ := os.Stdin.Read(buf)
		ch <- n > 0 && containsKittyResponse(buf[:n])
	}()

	select {
	case result := <-ch:
		return result
	case <-time.After(timeout):
		return false
	}
}

func containsKittyResponse(b []byte) bool {
	for i := 0; i < len(b)-2; i++ {
		if b[i] == 0x1b && b[i+1] == '_' && b[i+2] == 'G' {
			return true
		}
	}
	return false
}

// Encode encodes image bytes as a kitty graphics protocol string.
// cols and rows are the desired terminal cell dimensions.
func Encode(data []byte, cols, rows int) string {
	const chunkSize = 4096
	encoded := base64.StdEncoding.EncodeToString(data)
	out := ""
	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]
		m := 1
		if end == len(encoded) {
			m = 0
		}
		if i == 0 {
			out += fmt.Sprintf("\x1b_Ga=T,f=100,c=%d,r=%d,m=%d;%s\x1b\\", cols, rows, m, chunk)
		} else {
			out += fmt.Sprintf("\x1b_Gm=%d;%s\x1b\\", m, chunk)
		}
	}
	return out
}
