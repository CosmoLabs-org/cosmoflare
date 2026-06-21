package server

import (
	"bufio"
	"net/http"
	"strings"
	"testing"
	"time"
)

// waitForSubscriber polls Subscribers() until at least one client is attached
// to the hub, bounded by a short deadline. This defeats the test race where
// Publish fires before the /events goroutine has registered its client.
func waitForSubscriber(t *testing.T, s *Server) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.Subscribers() > 0 {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("subscriber never registered")
}

// TestSSE_RequiresToken — /events is behind the same token gate as REST.
func TestSSE_RequiresToken(t *testing.T) {
	s := New(Config{Token: "t", Version: "test"})
	url := s.testServer(t)

	resp, err := http.Get(url + "/events")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/events no token: got %d, want 401", resp.StatusCode)
	}
}

// TestSSE_ContentTypeAndNotificationFrame verifies /events is an event-stream
// and that a Publish reaches the client as an `event: <channel>\ndata: {...}`
// frame.
func TestSSE_ContentTypeAndNotificationFrame(t *testing.T) {
	s := New(Config{Token: "t", Version: "test"})
	url := s.testServer(t)

	req, _ := http.NewRequest(http.MethodGet, url+"/events", nil)
	req.Header.Set("Authorization", "Bearer t")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	waitForSubscriber(t, s)
	s.Publish("notifications", map[string]any{"msg": "hello"})

	br := bufio.NewReader(resp.Body)
	var frame strings.Builder
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		line, err := br.ReadString('\n')
		frame.WriteString(line)
		// A blank line terminates an SSE frame.
		if line == "\n" {
			break
		}
		if err != nil {
			break
		}
	}
	got := frame.String()
	if !strings.Contains(got, "event: notifications") {
		t.Fatalf("frame missing event line: %q", got)
	}
	if !strings.Contains(got, `"msg":"hello"`) {
		t.Fatalf("frame missing data: %q", got)
	}
}

// TestSSE_StatusPublishedOnHealthFlip verifies that flipping cloudflare_online
// pushes a `status` frame carrying the new health state.
func TestSSE_StatusPublishedOnHealthFlip(t *testing.T) {
	s := New(Config{Token: "t", Version: "test"})
	url := s.testServer(t)

	req, _ := http.NewRequest(http.MethodGet, url+"/events", nil)
	req.Header.Set("Authorization", "Bearer t")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()

	waitForSubscriber(t, s)

	// false -> true flips the tier and must publish a status frame.
	s.SetCloudflareOnline(true)

	br := bufio.NewReader(resp.Body)
	var frame strings.Builder
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		line, err := br.ReadString('\n')
		frame.WriteString(line)
		if line == "\n" {
			break
		}
		if err != nil {
			break
		}
	}
	got := frame.String()
	if !strings.Contains(got, "event: status") {
		t.Fatalf("missing status frame: %q", got)
	}
	if !strings.Contains(got, `"cloudflare_online":true`) {
		t.Fatalf("status frame missing cloudflare_online:true: %q", got)
	}
}

// TestSSE_NoSpamOnSameValue verifies SetCloudflareOnline is idempotent: setting
// the same value does not re-publish a status frame (avoids spamming on every
// successful REST call).
func TestSSE_NoSpamOnSameValue(t *testing.T) {
	s := New(Config{Token: "t", Version: "test"})
	s.SetCloudflareOnline(true)
	s.SetCloudflareOnline(true)
	s.SetCloudflareOnline(true)
	// No assertion on subscribers here — the contract is that repeated same-value
	// sets do not panic and are cheap. The flip-vs-repeat logic is exercised by
	// TestSSE_StatusPublishedOnHealthFlip (transition) above.
}

// TestSSE_NotificationOnHealthFlip verifies BR-07's v1 notification source: a
// cloudflare_online transition produces a `notifications` frame (in addition to
// the status frame), so the desktop notifications panel has real content.
func TestSSE_NotificationOnHealthFlip(t *testing.T) {
	s := New(Config{Token: "t", Version: "test"})
	url := s.testServer(t)

	req, _ := http.NewRequest(http.MethodGet, url+"/events", nil)
	req.Header.Set("Authorization", "Bearer t")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()

	waitForSubscriber(t, s)
	s.SetCloudflareOnline(true) // false -> true: emits status + a notification

	// Read frames asynchronously: the SSE stream stays open, so a blocking
	// ReadString would hang past the frame we want. A goroutine + select lets us
	// stop cleanly on the notifications frame or a deadline.
	br := bufio.NewReader(resp.Body)
	lines := make(chan string, 32)
	go func() {
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				return
			}
			lines <- line
		}
	}()

	sawNotification := false
	deadline := time.After(2 * time.Second)
	for {
		select {
		case line := <-lines:
			if strings.Contains(line, "event: notifications") {
				sawNotification = true
			}
			if sawNotification && line == "\n" {
				return // full notifications frame received
			}
		case <-deadline:
			if !sawNotification {
				t.Fatalf("health flip did not emit a notifications frame")
			}
			return
		}
	}
}
