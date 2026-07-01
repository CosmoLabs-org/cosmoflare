package server

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestMetricsProducer_PublishesOnTick(t *testing.T) {
	src := &fakeSource{
		zones:   []map[string]any{{"name": "example.com"}},
		r2:      []map[string]any{{"name": "bucket-1"}, {"name": "bucket-2"}},
		workers: []map[string]any{{"name": "worker-1"}},
		kv:      []map[string]any{{"name": "ns-1"}},
	}
	s, url := newSourcedServer(t, src)

	req, _ := http.NewRequest(http.MethodGet, url+"/events", nil)
	req.Header.Set("Authorization", "Bearer t")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	waitForSubscriber(t, s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewMetricsProducer(s, src, 50*time.Millisecond)
	p.Start(ctx)

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
	if !strings.Contains(got, "event: metrics") {
		t.Fatalf("expected metrics event, got: %q", got)
	}

	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(line, "data: ") {
			var payload MetricsSnapshot
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &payload); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if payload.Profile != "" {
				t.Fatalf("profile = %q, want empty (default)", payload.Profile)
			}
			zones, ok := payload.Zones.([]any)
			if !ok || len(zones) != 1 {
				t.Fatalf("zones = %v, want 1-element array", payload.Zones)
			}
			return
		}
	}
	t.Fatal("no data line in frame")
}

func TestMetricsProducer_SkipsWhenNoSubscribers(t *testing.T) {
	src := &fakeSource{
		zones: []map[string]any{{"name": "z"}},
		r2:    []map[string]any{},
	}
	s := New(Config{Token: "t", Version: "test"})
	s.SetData(src)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := NewMetricsProducer(s, src, 50*time.Millisecond)
	p.Start(ctx)

	time.Sleep(150 * time.Millisecond)
	cancel()

	if src.seenProf != "" {
		t.Fatalf("producer called source with no subscribers (profile=%q)", src.seenProf)
	}
}

func TestMetricsProducer_StopsOnContextCancel(t *testing.T) {
	src := &fakeSource{zones: []map[string]any{{"name": "z"}}}
	s := New(Config{Token: "t", Version: "test"})
	s.SetData(src)

	ctx, cancel := context.WithCancel(context.Background())
	p := NewMetricsProducer(s, src, 50*time.Millisecond)
	p.Start(ctx)
	cancel()
	time.Sleep(100 * time.Millisecond)
}
