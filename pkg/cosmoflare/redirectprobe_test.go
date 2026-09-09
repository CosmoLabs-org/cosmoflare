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

func TestRedirectProberOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "fine")
	}))
	defer srv.Close()

	res := newTestProber().Probe(context.Background(), srv.URL)
	if res.Status != http.StatusOK || res.Loop || res.Err != "" || res.Skipped {
		t.Fatalf("Probe = %+v, want 200 clean", res)
	}
}

func TestRedirectProber4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	}))
	defer srv.Close()

	res := newTestProber().Probe(context.Background(), srv.URL)
	if res.Status != http.StatusNotFound {
		t.Fatalf("Status = %d, want 404", res.Status)
	}
}

func TestRedirectProberLoop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/again", http.StatusFound) // /again → /again → loop
	}))
	defer srv.Close()

	res := newTestProber().Probe(context.Background(), srv.URL+"/again")
	if !res.Loop {
		t.Fatalf("Probe = %+v, want Loop=true", res)
	}
}

func TestRedirectProberHopCap(t *testing.T) {
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

func TestRedirectProberSkipsCaptures(t *testing.T) {
	res := newTestProber().Probe(context.Background(), "https://example.com/$1/x")
	if !res.Skipped {
		t.Fatalf("Probe = %+v, want Skipped=true for $1 capture", res)
	}
}

func TestRedirectProberUnreachable(t *testing.T) {
	res := newTestProber().Probe(context.Background(), "http://127.0.0.1:1/nope")
	if res.Err == "" || res.Status != 0 {
		t.Fatalf("Probe = %+v, want transport error with Status 0", res)
	}
}

func TestRedirectProberProbeAll(t *testing.T) {
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
