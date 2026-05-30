package cosmoflare

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestNewDevServer(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		ds := NewDevServer()
		if ds == nil {
			t.Fatal("expected non-nil DevServer")
		}
		if ds.Port() != 8787 {
			t.Errorf("expected default port 8787, got %d", ds.Port())
		}
		if !ds.Watch() {
			t.Error("expected watch to be true by default")
		}
	})

	t.Run("with custom port", func(t *testing.T) {
		ds := NewDevServer(WithDevPort(3000))
		if ds.Port() != 3000 {
			t.Errorf("expected port 3000, got %d", ds.Port())
		}
	})

	t.Run("with services filter", func(t *testing.T) {
		ds := NewDevServer(WithDevServices([]string{"r2", "kv"}))
		services := ds.Services()
		if len(services) != 2 {
			t.Fatalf("expected 2 services, got %d", len(services))
		}
		if services[0] != "r2" || services[1] != "kv" {
			t.Errorf("expected [r2 kv], got %v", services)
		}
	})

	t.Run("with watch disabled", func(t *testing.T) {
		ds := NewDevServer(WithDevWatch(false))
		if ds.Watch() {
			t.Error("expected watch to be false")
		}
	})

	t.Run("with profile", func(t *testing.T) {
		ds := NewDevServer(WithDevProfile("staging"))
		if ds.Profile() != "staging" {
			t.Errorf("expected profile 'staging', got %q", ds.Profile())
		}
	})
}

func TestDevServerStartStop(t *testing.T) {
	t.Run("starts and stops cleanly", func(t *testing.T) {
		ds := NewDevServer(WithDevPort(0)) // port 0 = OS picks a free port
		ctx, cancel := context.WithCancel(context.Background())

		errCh := make(chan error, 1)
		go func() {
			errCh <- ds.Start(ctx)
		}()

		// Wait for server to be ready
		if !waitForReady(ds, 2*time.Second) {
			t.Fatal("server did not become ready within timeout")
		}

		// Verify it's serving
		addr := ds.Addr()
		if addr == "" {
			t.Fatal("expected non-empty address after start")
		}

		resp, err := http.Get(fmt.Sprintf("http://%s/health", addr))
		if err != nil {
			t.Fatalf("health check failed: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 from /health, got %d", resp.StatusCode)
		}

		// Stop via context cancel
		cancel()

		select {
		case err := <-errCh:
			if err != nil {
				t.Errorf("expected nil error on clean shutdown, got: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("server did not stop within 3 seconds")
		}
	})

	t.Run("stop is idempotent", func(t *testing.T) {
		ds := NewDevServer(WithDevPort(0))
		ctx, cancel := context.WithCancel(context.Background())

		go func() { _ = ds.Start(ctx) }()
		waitForReady(ds, 2*time.Second)
		cancel()
		time.Sleep(100 * time.Millisecond)

		// Second stop should not panic or error
		ds.Stop()
	})
}

func TestDevServerRouting(t *testing.T) {
	t.Run("proxies to configured services", func(t *testing.T) {
		ds := NewDevServer(
			WithDevPort(0),
			WithDevServices([]string{"r2", "kv"}),
		)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() { _ = ds.Start(ctx) }()
		if !waitForReady(ds, 2*time.Second) {
			t.Fatal("server did not become ready")
		}

		base := fmt.Sprintf("http://%s", ds.Addr())

		// Known service routes should return non-404
		for _, svc := range []string{"r2", "kv"} {
			resp, err := http.Get(fmt.Sprintf("%s/%s", base, svc))
			if err != nil {
				t.Fatalf("request to /%s failed: %v", svc, err)
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound {
				t.Errorf("expected /%s to be routed (not 404)", svc)
			}
		}

		// Non-configured service should 404
		resp, err := http.Get(fmt.Sprintf("%s/workers", base))
		if err != nil {
			t.Fatalf("request to /workers failed: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected /workers to return 404, got %d", resp.StatusCode)
		}
	})
}

func TestDevServerJSON(t *testing.T) {
	t.Run("startup event has defaults", func(t *testing.T) {
		ds := NewDevServer()
		event := ds.StartupEvent()
		if event.Port != 8787 {
			t.Errorf("expected port 8787 in startup event, got %d", event.Port)
		}
		if event.Services == nil {
			t.Error("expected non-nil services in startup event")
		}
		if !event.Watch {
			t.Error("expected watch=true in startup event")
		}
	})

	t.Run("startup event reflects actual port after start", func(t *testing.T) {
		ds := NewDevServer(WithDevPort(0))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() { _ = ds.Start(ctx) }()
		if !waitForReady(ds, 2*time.Second) {
			t.Fatal("server did not become ready")
		}

		event := ds.StartupEvent()
		if event.Port == 0 {
			t.Error("expected non-zero port after start")
		}
		if event.Address == "" {
			t.Error("expected non-empty address after start")
		}
	})
}

// waitForReady polls the dev server's Ready() method until true or timeout.
func waitForReady(ds *DevServer, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ds.Ready() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
