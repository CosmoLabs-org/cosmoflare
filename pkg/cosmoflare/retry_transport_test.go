package cosmoflare

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// rtFunc adapts a function to http.RoundTripper for tests.
type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// retryTransportHarness counts attempts and serves canned statuses in order
// (the last repeats). Real (tiny) delays keep the test faithful to retry.Do.
type retryTransportHarness struct {
	statuses []int
	attempts int
	bodies   []string // bodies received per attempt (checks replay)
}

func (h *retryTransportHarness) transport() http.RoundTripper {
	index := 0
	return rtFunc(func(r *http.Request) (*http.Response, error) {
		h.attempts++
		if r.Body != nil {
			b, _ := io.ReadAll(r.Body)
			h.bodies = append(h.bodies, string(b))
		} else {
			h.bodies = append(h.bodies, "")
		}
		status := h.statuses[len(h.statuses)-1]
		if index < len(h.statuses) {
			status = h.statuses[index]
			index++
		}
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Request:    r,
		}, nil
	})
}

func retryTransportRequest(t *testing.T, method, url string, body string) *http.Request {
	t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, rd)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	return req
}

func TestRetryTransport429RetriesToSuccess(t *testing.T) {
	h := &retryTransportHarness{statuses: []int{429, 429, 200}}
	rt := &RetryTransport{Base: h.transport()}
	req := retryTransportRequest(t, "GET", "https://api.example.com/zones", "")
	res, err := rt.RoundTrip(req.WithContext(context.Background()))
	if err != nil {
		t.Fatalf("round trip: %v", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("status = %d, want 200 after two 429 retries", res.StatusCode)
	}
	if h.attempts != 3 {
		t.Fatalf("attempts = %d, want 3 (bounded at 2 retries)", h.attempts)
	}
}

func TestRetryTransportExhaustedSurfacesError(t *testing.T) {
	h := &retryTransportHarness{statuses: []int{429}}
	rt := &RetryTransport{Base: h.transport()}
	req := retryTransportRequest(t, "GET", "https://api.example.com/zones", "")
	_, err := rt.RoundTrip(req.WithContext(context.Background()))
	if err == nil {
		t.Fatal("exhausted 429s must surface an error, not a synthetic response")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Fatalf("error should name the status: %v", err)
	}
	if h.attempts != 3 {
		t.Fatalf("attempts = %d, want exactly 3 (bounded)", h.attempts)
	}
}

func TestRetryTransport500GateByMethod(t *testing.T) {
	h := &retryTransportHarness{statuses: []int{500, 200}}
	rt := &RetryTransport{Base: h.transport()}
	req := retryTransportRequest(t, "GET", "https://api.example.com/zones", "")
	res, err := rt.RoundTrip(req.WithContext(context.Background()))
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("GET 500→200 should retry: res=%v err=%v", res, err)
	}
	if h.attempts != 2 {
		t.Fatalf("GET attempts = %d, want 2", h.attempts)
	}

	h2 := &retryTransportHarness{statuses: []int{500, 200}}
	rt2 := &RetryTransport{Base: h2.transport()}
	req2 := retryTransportRequest(t, "POST", "https://api.example.com/zones", `{"name":"x"}`)
	res2, err2 := rt2.RoundTrip(req2.WithContext(context.Background()))
	if err2 != nil || res2.StatusCode != 500 {
		t.Fatalf("POST 500 must surface unretried: res=%v err=%v", res2, err2)
	}
	if h2.attempts != 1 {
		t.Fatalf("POST attempts = %d, want 1 (5xx may have processed it)", h2.attempts)
	}
}

func TestRetryTransport429RetriesPOSTWithBodyReplay(t *testing.T) {
	h := &retryTransportHarness{statuses: []int{429, 201}}
	rt := &RetryTransport{Base: h.transport()}
	req := retryTransportRequest(t, "POST", "https://api.example.com/zones", `{"name":"x"}`)
	res, err := rt.RoundTrip(req.WithContext(context.Background()))
	if err != nil || res.StatusCode != 201 {
		t.Fatalf("429 POST should retry to 201: res=%v err=%v", res, err)
	}
	if h.attempts != 2 {
		t.Fatalf("attempts = %d, want 2", h.attempts)
	}
	// The retried attempt must carry the SAME body (GetBody rewind).
	if len(h.bodies) != 2 || h.bodies[0] != `{"name":"x"}` || h.bodies[1] != `{"name":"x"}` {
		t.Fatalf("bodies = %v, want the same replayed body on both attempts", h.bodies)
	}
}

func TestRetryTransportNonReplayableBodyIsOneShot(t *testing.T) {
	h := &retryTransportHarness{statuses: []int{429, 200}}
	rt := &RetryTransport{Base: h.transport()}
	req := retryTransportRequest(t, "POST", "https://api.example.com/zones", `{"name":"x"}`)
	req.GetBody = nil // simulate a body that cannot be rewound
	res, err := rt.RoundTrip(req.WithContext(context.Background()))
	if err != nil || res.StatusCode != 429 {
		t.Fatalf("non-replayable body must not retry: res=%v err=%v", res, err)
	}
	if h.attempts != 1 {
		t.Fatalf("attempts = %d, want 1", h.attempts)
	}
}

func TestRetryTransportSuccessFirstTry(t *testing.T) {
	h := &retryTransportHarness{statuses: []int{200}}
	rt := &RetryTransport{Base: h.transport()}
	req := retryTransportRequest(t, "GET", "https://api.example.com/zones", "")
	start := time.Now()
	res, err := rt.RoundTrip(req.WithContext(context.Background()))
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("first-try success: res=%v err=%v", res, err)
	}
	if h.attempts != 1 {
		t.Fatalf("attempts = %d, want 1", h.attempts)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("success path must not sleep: took %v", elapsed)
	}
}
