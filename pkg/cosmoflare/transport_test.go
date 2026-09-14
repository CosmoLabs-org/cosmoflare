package cosmoflare

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- FEAT-039: one transport policy, enforced at one chokepoint ---

func TestControlPlaneClient_HasDefaultTimeout(t *testing.T) {
	c := controlPlaneClient()
	if c == nil {
		t.Fatal("controlPlaneClient returned nil")
	}
	if c.Timeout != DefaultControlPlaneTimeout {
		t.Errorf("control-plane timeout = %v, want %v", c.Timeout, DefaultControlPlaneTimeout)
	}
	if c.Timeout == 0 {
		t.Error("control-plane client must never be timeout-free (http.DefaultClient class of hang)")
	}
}

func TestNewCloudflareAPI_WiresControlPlaneClient(t *testing.T) {
	api, err := newCloudflareAPI("test-token")
	if err != nil {
		t.Fatalf("newCloudflareAPI: %v", err)
	}
	if api == nil {
		t.Fatal("newCloudflareAPI returned nil API")
	}
}

// TestTransportChokepoint_Guard enforces the policy structurally: every
// Cloudflare API construction and every raw HTTP request in this package
// must go through the shared transport helpers. A bare
// cloudflare.NewWithAPIToken or http.DefaultClient call silently opts a
// service out of the timeout policy — this test fails the build the moment
// one sneaks back in (FEAT-039).
func TestTransportChokepoint_Guard(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob *.go: %v (err=%v)", files, err)
	}
	violations := []string{}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") || f == "transport.go" {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		src := string(b)
		// Bare NewWithAPIToken calls (no explicit HTTPClient option) bypass
		// the timeout policy. Calls that pass cloudflare.HTTPClient are
		// deliberate overrides (client.go's WithHTTPClient path, the
		// knowledge.Transport wrapper in ratelimit.go) and are allowed.
		newWith := strings.Count(src, "cloudflare.NewWithAPIToken(")
		withClient := strings.Count(src, "cloudflare.HTTPClient(")
		if newWith > withClient {
			violations = append(violations, f+": bare cloudflare.NewWithAPIToken without HTTPClient (use newCloudflareAPI)")
		}
		if strings.Contains(src, "http.DefaultClient") {
			violations = append(violations, f+": http.DefaultClient (no timeout — use controlPlaneClient)")
		}
	}
	if len(violations) > 0 {
		t.Errorf("transport policy violations (%d):\n  %s", len(violations), strings.Join(violations, "\n  "))
	}
}

// Sanity: the policy constant documents the number users can rely on.
func TestDefaultControlPlaneTimeoutValue(t *testing.T) {
	if DefaultControlPlaneTimeout != 30*time.Second {
		t.Errorf("DefaultControlPlaneTimeout = %v, want 30s (changing it is a documented breaking change)", DefaultControlPlaneTimeout)
	}
}
