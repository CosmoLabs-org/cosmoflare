package cosmoflare

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestProber() *RedirectProber {
	return NewRedirectProber(
		WithProbeConcurrency(4),
		WithProbeTimeout(2*time.Second),
		WithProbeHopCap(5),
	)
}

// TestRedirectProberOK TestRedirectProberOK verifies that probing a URL
// returning 200 with no redirects yields a clean result: no loop, no error,
// not skipped.
func TestRedirectProberOK(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "fine")
	}))
	defer srv.Close()

	res := newTestProber().Probe(context.Background(), srv.URL)
	if res.Status != http.StatusOK || res.Loop || res.Err != "" || res.Skipped {
		t.Fatalf("Probe = %+v, want 200 clean", res)
	}
}

// TestRedirectProber4xx TestRedirectProber4xx verifies that a 404 response is
// reported as-is rather than treated as a transport failure.
func TestRedirectProber4xx(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer srv.Close()

	res := newTestProber().Probe(context.Background(), srv.URL)
	if res.Status != http.StatusNotFound {
		t.Fatalf("Status = %d, want 404", res.Status)
	}
}

// TestRedirectProberLoop TestRedirectProberLoop verifies that a URL
// redirecting to itself is detected as a redirect loop.
func TestRedirectProberLoop(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/again", http.StatusFound) // /again → /again → loop
	}))
	defer srv.Close()

	res := newTestProber().Probe(context.Background(), srv.URL+"/again")
	if !res.Loop {
		t.Fatalf("Probe = %+v, want Loop=true", res)
	}
}

// TestRedirectProberHopCap TestRedirectProberHopCap verifies that an
// endlessly-redirecting chain that never repeats a URL terminates at the hop
// cap with an error that is explicitly not a loop.
func TestRedirectProberHopCap(t *testing.T) {
	t.Parallel()
	hops := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hops++
		http.Redirect(w, r, fmt.Sprintf("/hop-%d", hops), http.StatusFound) // never repeats a URL, never terminates
	}))
	defer srv.Close()

	res := newTestProber().Probe(context.Background(), srv.URL+"/start")
	if res.Loop || res.Err == "" {
		t.Fatalf("Probe = %+v, want non-loop error at hop cap (Err set, Loop false)", res)
	}
}

// TestRedirectProberSkipsCaptures TestRedirectProberSkipsCaptures verifies
// that URLs containing regex capture placeholders like $1 are skipped instead
// of probed.
func TestRedirectProberSkipsCaptures(t *testing.T) {
	t.Parallel()
	res := newTestProber().Probe(context.Background(), "https://example.com/$1/x")
	if !res.Skipped {
		t.Fatalf("Probe = %+v, want Skipped=true for $1 capture", res)
	}
}

// TestRedirectProberUnreachable TestRedirectProberUnreachable verifies that a
// transport-level failure yields an error result with status 0.
func TestRedirectProberUnreachable(t *testing.T) {
	t.Parallel()
	res := newTestProber().Probe(context.Background(), "http://127.0.0.1:1/nope")
	if res.Err == "" || res.Status != 0 {
		t.Fatalf("Probe = %+v, want transport error with Status 0", res)
	}
}

// TestRedirectProberProbeAll TestRedirectProberProbeAll verifies that
// ProbeAll deduplicates the input URL list and returns per-URL results keyed
// by URL.
func TestRedirectProberProbeAll(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			http.Error(w, "x", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	results := newTestProber().ProbeAll(context.Background(), []string{srv.URL, srv.URL + "/bad", srv.URL})
	if len(results) != 2 {
		t.Fatalf("ProbeAll must dedupe, got %d entries: %+v", len(results), results)
	}
	if results[srv.URL].Status != 200 || results[srv.URL+"/bad"].Status != 500 {
		t.Fatalf("ProbeAll results wrong: %+v", results)
	}
}
