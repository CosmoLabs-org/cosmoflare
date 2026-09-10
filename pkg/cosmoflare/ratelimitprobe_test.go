package cosmoflare

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestExpressionPath verifies literal path extraction from the two
// documented expression forms, and non-path expressions yield "".
func TestExpressionPath(t *testing.T) {
	tests := []struct{ in, want string }{
		{`path eq "/catalog.json"`, "/catalog.json"},
		{`http.request.uri.path eq "/a/b"`, "/a/b"},
		{`(http.host eq "x.com" and path eq "/x")`, ""}, // composite: not a literal prefix
		{`http.request.method eq "GET"`, ""},
		{``, ""},
	}
	for _, tt := range tests {
		if got := ExpressionPath(tt.in); got != tt.want {
			t.Errorf("ExpressionPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestClassifyProbe drives the pure verdict engine directly — the test must
// not re-derive the verdict rules inline (tautological-test guard).
func TestClassifyProbe(t *testing.T) {
	t.Run("any 429 trips", func(t *testing.T) {
		v, _ := classifyProbe(map[int]int{200: 19, 429: 1}, 20, 20)
		if v != VerdictTripped {
			t.Fatalf("verdict = %q, want tripped", v)
		}
	})
	t.Run("all ok means not counted with matrix explanation", func(t *testing.T) {
		v, expl := classifyProbe(map[int]int{200: 20}, 20, 20)
		if v != VerdictNotCounted {
			t.Fatalf("verdict = %q, want not-counted", v)
		}
		if !strings.Contains(expl, "cache-hit-static-asset") {
			t.Fatalf("explanation must cite the skipped classes: %q", expl)
		}
	})
	t.Run("transport errors dominate means inconclusive", func(t *testing.T) {
		v, _ := classifyProbe(map[int]int{-1: 15, 200: 5}, 5, 20)
		if v != VerdictInconclusive {
			t.Fatalf("verdict = %q, want inconclusive", v)
		}
	})
	t.Run("short send means inconclusive", func(t *testing.T) {
		v, _ := classifyProbe(map[int]int{200: 10}, 10, 20)
		if v != VerdictInconclusive {
			t.Fatalf("verdict = %q, want inconclusive", v)
		}
	})
	t.Run("403 also trips", func(t *testing.T) {
		v, _ := classifyProbe(map[int]int{403: 3, 200: 7}, 10, 10)
		if v != VerdictTripped {
			t.Fatalf("verdict = %q, want tripped", v)
		}
	})
	t.Run("all 404 is inconclusive, not not-counted", func(t *testing.T) {
		v, _ := classifyProbe(map[int]int{404: 10}, 10, 10)
		if v != VerdictInconclusive {
			t.Fatalf("verdict = %q, want inconclusive", v)
		}
	})
}

// TestRateLimitProbeBurst exercises the live prober against httptest
// servers: 429-server trips, 200-server reports not-counted, dead server is
// inconclusive. Also asserts the cache-buster is present on odd requests.
func TestRateLimitProbeBurst(t *testing.T) {
	t.Run("always 429", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer srv.Close()
		res := NewRateLimitProber(WithRateLimitProbeTimeout(5*time.Second)).Probe(context.Background(), srv.URL, 6)
		if !res.Tripped || res.Verdict != VerdictTripped {
			t.Fatalf("want tripped, got %+v", res)
		}
	})
	t.Run("always 200 with cache busters", func(t *testing.T) {
		var total, busted int64
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&total, 1)
			if strings.Contains(r.URL.RawQuery, "cfprobe=") {
				atomic.AddInt64(&busted, 1)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		res := NewRateLimitProber(WithRateLimitProbeTimeout(5*time.Second)).Probe(context.Background(), srv.URL+"/catalog.json", 6)
		if res.Verdict != VerdictNotCounted {
			t.Fatalf("want not-counted, got %+v", res)
		}
		if busted != 3 || total != 6 {
			t.Fatalf("want 6 requests / 3 cache-busted, got %d/%d", total, busted)
		}
	})
	t.Run("dead server is inconclusive", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close()
		res := NewRateLimitProber(WithRateLimitProbeTimeout(2*time.Second)).Probe(context.Background(), url, 4)
		if res.Verdict != VerdictInconclusive {
			t.Fatalf("want inconclusive, got %+v", res)
		}
	})
	t.Run("requests clamp to MaxProbeRequests", func(t *testing.T) {
		var n int64
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&n, 1)
		}))
		defer srv.Close()
		res := NewRateLimitProber(WithRateLimitProbeTimeout(5*time.Second)).Probe(context.Background(), srv.URL, 500)
		if res.Requests != MaxProbeRequests {
			t.Fatalf("requests must clamp to %d, got %d", MaxProbeRequests, res.Requests)
		}
	})
}

// TestRateLimitProberDefaults verifies the option defaults construct a
// usable prober (non-nil client, sane timeout/concurrency).
func TestRateLimitProberDefaults(t *testing.T) {
	p := NewRateLimitProber()
	if p == nil || p.httpClient == nil || p.concurrency < 1 || p.timeout <= 0 {
		t.Fatalf("bad defaults: %+v", p)
	}
}
