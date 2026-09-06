package cmd

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/server"
	"github.com/CosmoLabs-org/cosmoflare/internal/webhook"
)

// FEAT-008: the serve daemon bridges webhook.Manager alert triggers onto the
// SSE hub's notifications channel, so desktop clients subscribed to /events
// receive alert_triggered frames live — with zero webhooks configured.

func waitForServeSubscriber(t *testing.T, s *server.Server) {
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

func TestServeAlertBridge_PublishesAlertToNotificationsChannel(t *testing.T) {
	srv := server.New(server.Config{Token: "t", Version: "test"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/events", nil)
	req.Header.Set("Authorization", "Bearer t")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer resp.Body.Close()

	// Subscribe BEFORE triggering — Publish to a hub with no clients drops
	// the frame by design (live channel, not a queue).
	waitForServeSubscriber(t, srv)

	m := newServeAlertBridge(srv)
	alert := &webhook.Alert{
		ID:        "a-1",
		Name:      "error-rate",
		Type:      webhook.AlertTypeThreshold,
		Threshold: 5,
		Enabled:   true,
	}
	if err := m.TriggerAlert(alert, 7.5, "error rate above threshold", nil); err != nil {
		t.Fatalf("TriggerAlert: %v", err)
	}

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
	if !strings.Contains(got, "event: notifications") {
		t.Fatalf("frame missing notifications event line: %q", got)
	}
	if !strings.Contains(got, `"event":"alert_triggered"`) {
		t.Fatalf("frame missing alert_triggered payload: %q", got)
	}
	if !strings.Contains(got, `"name":"error-rate"`) {
		t.Fatalf("frame missing alert name: %q", got)
	}
	if !strings.Contains(got, `"value":7.5`) {
		t.Fatalf("frame missing trigger value: %q", got)
	}
}
