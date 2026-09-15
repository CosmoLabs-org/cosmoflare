package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var (
	errZones   = fmt.Errorf("zones unavailable")
	errR2      = fmt.Errorf("r2 unavailable")
	errWorkers = fmt.Errorf("workers unavailable")
	errKV      = fmt.Errorf("kv unavailable")
)

func TestMetricsProducer_PublishesOnTick(t *testing.T) {
	src := &fakeSource{
		zones:   []*cosmoflare.Zone{{Name: "example.com"}},
		r2:      []*cosmoflare.Bucket{{Name: "bucket-1"}, {Name: "bucket-2"}},
		workers: []*cosmoflare.Worker{{Name: "worker-1"}},
		kv:      []*cosmoflare.KVNamespace{{Title: "ns-1"}},
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
			if len(payload.Zones) != 1 {
				t.Fatalf("zones = %v, want 1-element array", payload.Zones)
			}
			return
		}
	}
	t.Fatal("no data line in frame")
}

func TestMetricsProducer_SkipsWhenNoSubscribers(t *testing.T) {
	src := &fakeSource{
		zones: []*cosmoflare.Zone{{Name: "z"}},
		r2:    []*cosmoflare.Bucket{},
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
	src := &fakeSource{zones: []*cosmoflare.Zone{{Name: "z"}}}
	s := New(Config{Token: "t", Version: "test"})
	s.SetData(src)

	ctx, cancel := context.WithCancel(context.Background())
	p := NewMetricsProducer(s, src, 50*time.Millisecond)
	p.Start(ctx)
	cancel()
	time.Sleep(100 * time.Millisecond)
}

func TestMetricsPartialSnapshot(t *testing.T) {
	src := &metricsSource{
		zones:    []*cosmoflare.Zone{{Name: "example.com"}},
		r2Err:    errR2,
		workers:  []*cosmoflare.Worker{{Name: "worker-1"}},
		kv:       []*cosmoflare.KVNamespace{{Title: "ns-1"}},
	}
	s, url := newSourcedServer(t, src)

	ch := subscribeMetrics(t, url)
	waitForSubscriber(t, s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	NewMetricsProducer(s, src, 50*time.Millisecond).Start(ctx)

	snap := expectMetricsSnapshot(t, ch, 3*time.Second)
	if snap.Errors["r2_buckets"] == "" {
		t.Fatalf("Errors[r2_buckets] not set: %+v", snap.Errors)
	}
	if len(snap.Zones) != 1 {
		t.Fatalf("zones = %v, want 1-element array", snap.Zones)
	}
	if snap.Workers == nil || snap.KVNamespaces == nil {
		t.Fatalf("healthy sources dropped: workers=%v kv=%v", snap.Workers, snap.KVNamespaces)
	}
}

func TestMetricsProfilePopulated(t *testing.T) {
	src := &metricsSource{
		profile: "default",
		zones:   []*cosmoflare.Zone{{Name: "example.com"}},
	}
	s, url := newSourcedServer(t, src)

	ch := subscribeMetrics(t, url)
	waitForSubscriber(t, s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	NewMetricsProducer(s, src, 50*time.Millisecond).Start(ctx)

	snap := expectMetricsSnapshot(t, ch, 3*time.Second)
	if snap.Profile != "default" {
		t.Fatalf("profile = %q, want default", snap.Profile)
	}
}

func TestMetricsDeltaDetection(t *testing.T) {
	src := &metricsSource{
		zones: []*cosmoflare.Zone{{Name: "example.com"}},
	}
	s, url := newSourcedServer(t, src)

	ch := subscribeMetrics(t, url)
	waitForSubscriber(t, s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	NewMetricsProducer(s, src, 40*time.Millisecond).Start(ctx)

	// First poll publishes.
	expectMetricsSnapshot(t, ch, 3*time.Second)
	// Identical data on subsequent polls → no re-publish.
	expectNoMetricsSnapshot(t, ch, 300*time.Millisecond)
	// Changed data → exactly one more publish.
	src.setZones([]*cosmoflare.Zone{{Name: "example.com"}, {Name: "example.org"}})
	snap := expectMetricsSnapshot(t, ch, 3*time.Second)
	if len(snap.Zones) != 2 {
		t.Fatalf("zones = %v, want 2-element array", snap.Zones)
	}
}

func TestMetricsAllSourcesFail(t *testing.T) {
	src := &metricsSource{
		zoneErr:   errZones,
		r2Err:     errR2,
		workerErr: errWorkers,
		kvErr:     errKV,
	}
	s, url := newSourcedServer(t, src)

	ch := subscribeMetrics(t, url)
	waitForSubscriber(t, s)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	NewMetricsProducer(s, src, 50*time.Millisecond).Start(ctx)

	snap := expectMetricsSnapshot(t, ch, 3*time.Second)
	if len(snap.Errors) != 4 {
		t.Fatalf("Errors = %v, want 4 entries", snap.Errors)
	}
	for _, k := range []string{"zones", "r2_buckets", "workers", "kv_namespaces"} {
		if snap.Errors[k] == "" {
			t.Fatalf("Errors[%q] not set: %v", k, snap.Errors)
		}
	}
}
