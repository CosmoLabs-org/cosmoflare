package server

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// testServer starts an httptest server backed by s.Handler() and registers a
// cleanup. Returns the base URL for clients to use.
func (s *Server) testServer(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return srv.URL
}

// healthzCloudflare fetches /healthz with the given token and returns the
// reported cloudflare_online value.
func healthzCloudflare(t *testing.T, url, token string) bool {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url+"/healthz", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode /healthz: %v", err)
	}
	online, _ := out["cloudflare_online"].(bool)
	return online
}

// metricsSource is a ServeSource for the metrics-producer tests. Unlike
// fakeSource (rest_test.go) it supports per-source error injection, mutable
// data (for delta detection), and the optional ProfileNameResolver seam, so
// it exercises the producer's partial-snapshot and change-detection paths.
type metricsSource struct {
	mu        sync.Mutex
	profile   string
	zones     any
	r2        any
	workers   any
	kv        any
	zoneErr   error
	r2Err     error
	workerErr error
	kvErr     error
}

func (m *metricsSource) Accounts(_ context.Context) (any, error) { return nil, nil }

func (m *metricsSource) CurrentProfileName() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.profile
}

func (m *metricsSource) Zones(_ context.Context, _ string) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.zones, m.zoneErr
}

func (m *metricsSource) R2Buckets(_ context.Context, _ string) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.r2, m.r2Err
}

func (m *metricsSource) Workers(_ context.Context, _ string) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.workers, m.workerErr
}

func (m *metricsSource) KV(_ context.Context, _ string) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.kv, m.kvErr
}

// setZones swaps the zones payload between polls; used for delta detection.
func (m *metricsSource) setZones(v any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.zones = v
}

// subscribeMetrics opens one SSE subscription and returns a channel that
// receives every published metrics snapshot. A watchdog fails the test if the
// read goroutine stalls.
func subscribeMetrics(t *testing.T, url string) <-chan MetricsSnapshot {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url+"/events", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer t")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	ch := make(chan MetricsSnapshot, 16)
	go func() {
		br := bufio.NewReader(resp.Body)
		var frame strings.Builder
		for {
			line, err := br.ReadString('\n')
			if err != nil {
				close(ch)
				return
			}
			if line == "\n" {
				if strings.Contains(frame.String(), "event: metrics") {
					for _, l := range strings.Split(frame.String(), "\n") {
						if strings.HasPrefix(l, "data: ") {
							var snap MetricsSnapshot
							if json.Unmarshal([]byte(strings.TrimPrefix(l, "data: ")), &snap) == nil {
								ch <- snap
							}
						}
					}
				}
				frame.Reset()
				continue
			}
			frame.WriteString(line)
		}
	}()
	return ch
}

// expectMetricsSnapshot waits up to d for the next snapshot, failing the test
// on timeout.
func expectMetricsSnapshot(t *testing.T, ch <-chan MetricsSnapshot, d time.Duration) MetricsSnapshot {
	t.Helper()
	select {
	case snap, ok := <-ch:
		if !ok {
			t.Fatal("metrics stream closed")
		}
		return snap
	case <-time.After(d):
		t.Fatalf("no metrics snapshot within %s", d)
		return MetricsSnapshot{}
	}
}

// expectNoMetricsSnapshot asserts no snapshot arrives within d.
func expectNoMetricsSnapshot(t *testing.T, ch <-chan MetricsSnapshot, d time.Duration) {
	t.Helper()
	select {
	case snap := <-ch:
		t.Fatalf("unexpected metrics snapshot within %s: %+v", d, snap)
	case <-time.After(d):
	}
}
