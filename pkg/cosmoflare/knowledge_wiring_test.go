package cosmoflare

// FEAT-044: knowledge.Transport must be wired at the shared client
// factories, not just the rate-limit service. These tests exercise the
// full construction path (NewClient / newCloudflareAPI / restClient) and
// pin the chokepoint count.

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"
)

// wiringCountingTripper is the wrap-always marker: a RoundTripper that
// counts every call that reaches it.
type wiringCountingTripper struct{ calls int32 }

func (w *wiringCountingTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	atomic.AddInt32(&w.calls, 1)
	return http.DefaultTransport.RoundTrip(req)
}

// wiringHitServer returns an httptest server that counts requests. Reaching
// it proves a request left the process.
func wiringHitServer(t *testing.T) (*httptest.Server, *int32) {
	t.Helper()
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

// TestKnowledgeTransportFullPathBlock builds a real NewClient and fires an
// in-scope UNREGISTERED route through its control-plane http.Client. The
// request must be blocked in-process by knowledge.Transport — the httptest
// base must see zero requests.
func TestKnowledgeTransportFullPathBlock(t *testing.T) {
	srv, hits := wiringHitServer(t)

	c, err := NewClient(WithAccountID("test-account"), WithAPIToken("test-token"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	cl, ok := c.(*client)
	if !ok {
		t.Fatalf("NewClient returned %T, want *client", c)
	}

	req, err := http.NewRequest(http.MethodDelete,
		srv.URL+"/zones/00000000000000000000000000000000/rulesets/phases/http_ratelimit/entrypoint", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := cl.httpClient.Do(req)
	if err == nil {
		resp.Body.Close()
		t.Fatal("in-scope unregistered route passed through; expected knowledge block")
	}
	if !strings.Contains(err.Error(), "not a registered endpoint") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "not a registered endpoint")
	}
	if got := atomic.LoadInt32(hits); got != 0 {
		t.Errorf("httptest base saw %d requests, want 0 (block must happen in-process)", got)
	}
}

// TestKnowledgeTransportWrapAlways pins the WithHTTPClient semantics: the
// caller's custom RoundTripper stays in the chain (out-of-scope traffic
// still reaches it) while the registry still blocks in-scope unregistered
// routes.
func TestKnowledgeTransportWrapAlways(t *testing.T) {
	srv, hits := wiringHitServer(t)

	marker := &wiringCountingTripper{}
	c, err := NewClient(WithAccountID("test-account"), WithAPIToken("test-token"),
		WithHTTPClient(&http.Client{Transport: marker}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	cl := c.(*client)

	// The control-plane client wraps the caller's transport (wrap-always),
	// preserving the chain via knowledge.Transport.Base.
	kt, ok := cl.httpClient.Transport.(*knowledge.Transport)
	if !ok {
		t.Fatalf("control-plane transport = %T, want *knowledge.Transport", cl.httpClient.Transport)
	}
	if kt.Base == nil || kt.Base != http.RoundTripper(marker) {
		t.Errorf("knowledge.Transport.Base = %v, want the caller's marker RoundTripper", kt.Base)
	}

	// Out-of-scope route passes through and increments the marker.
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/zones", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := cl.httpClient.Do(req)
	if err != nil {
		t.Fatalf("out-of-scope GET /zones: %v", err)
	}
	resp.Body.Close()
	if got := atomic.LoadInt32(&marker.calls); got != 1 {
		t.Errorf("marker RoundTripper calls = %d, want 1 (caller transport must stay in the chain)", got)
	}
	if got := atomic.LoadInt32(hits); got != 1 {
		t.Errorf("httptest base hits = %d, want 1", got)
	}

	// In-scope unregistered route still errors — the registry applies even
	// on the caller-supplied client.
	blocked, err := http.NewRequest(http.MethodDelete,
		srv.URL+"/zones/00000000000000000000000000000000/rulesets/phases/http_ratelimit/entrypoint", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if _, err := cl.httpClient.Do(blocked); err == nil {
		t.Error("in-scope unregistered route passed through the wrapped client; expected knowledge block")
	} else if !strings.Contains(err.Error(), "not a registered endpoint") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "not a registered endpoint")
	}
	if got := atomic.LoadInt32(&marker.calls); got != 1 {
		t.Errorf("marker calls after blocked route = %d, want 1 (blocked requests never reach the base)", got)
	}

	// The caller's original client struct is never mutated: the data plane
	// (initS3) keeps using it unwrapped.
	if cl.cfg.httpClient == nil || cl.cfg.httpClient.Transport != http.RoundTripper(marker) {
		t.Error("caller-owned client was mutated or lost; the R2 data plane must keep the original transport (BUG-042/FEAT-044)")
	}
}

// TestKnowledgeWiring_ChokepointCount pins FEAT-044's structural invariant:
// exactly one cloudflare.NewWithAPIToken construction site across the
// package's non-test files (transport.go, the chokepoint). ratelimit.go and
// client.go must never construct cloudflare-go clients directly again.
func TestKnowledgeWiring_ChokepointCount(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("glob *.go: %v (err=%v)", files, err)
	}
	total := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if n := strings.Count(string(b), "cloudflare.NewWithAPIToken("); n > 0 && f != "transport.go" {
			t.Errorf("%s constructs a cloudflare-go client directly (%d); use newCloudflareAPI", f, n)
		}
		total += strings.Count(string(b), "cloudflare.NewWithAPIToken(")
	}
	if total != 1 {
		t.Errorf("cloudflare.NewWithAPIToken sites in non-test files = %d, want exactly 1 (transport.go only)", total)
	}
}
