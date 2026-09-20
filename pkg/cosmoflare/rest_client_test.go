package cosmoflare

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// noSleep is a sleepFn that records the requested delays without actually
// sleeping, so retry tests run instantly.
func noSleep(record *[]time.Duration) func(time.Duration) {
	return func(d time.Duration) {
		*record = append(*record, d)
	}
}

func restTestClient(baseURL string, opts ...restClientOption) restClient {
	c := newRESTClient("test-token", opts...)
	c.baseURL = baseURL
	return c
}

func TestRESTClientRetriesOn429WithRetryAfter(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&requests, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"success":false,"errors":[{"code":10000,"message":"rate limited"}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"errors":[],"result":{}}`))
	}))
	defer srv.Close()

	var sleeps []time.Duration
	c := restTestClient(srv.URL)
	c.sleepFn = noSleep(&sleeps)

	err := c.do(t.Context(), "test.op", http.MethodGet, "/", nil, nil)
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	if got := atomic.LoadInt32(&requests); got != 2 {
		t.Fatalf("expected exactly 2 requests, got %d", got)
	}
	if len(sleeps) != 1 || sleeps[0] != 0 {
		t.Fatalf("expected exactly one sleep of 0s (from Retry-After: 0), got %v", sleeps)
	}
}

func TestRESTClientRetriesThriceThenSucceeds(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&requests, 1)
		if n <= 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"success":false,"errors":[{"code":10000,"message":"rate limited"}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"errors":[],"result":{}}`))
	}))
	defer srv.Close()

	var sleeps []time.Duration
	c := restTestClient(srv.URL, withMaxRetries(3))
	c.sleepFn = noSleep(&sleeps)

	err := c.do(t.Context(), "test.op", http.MethodGet, "/", nil, nil)
	if err != nil {
		t.Fatalf("expected success on 4th attempt, got error: %v", err)
	}
	if got := atomic.LoadInt32(&requests); got != 4 {
		t.Fatalf("expected exactly 4 requests, got %d", got)
	}
	if len(sleeps) != 3 {
		t.Fatalf("expected exactly 3 sleeps, got %d", len(sleeps))
	}
}

func TestRESTClientMaxRetriesZeroDisablesRetries(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"success":false,"errors":[{"code":10000,"message":"rate limited"}]}`))
	}))
	defer srv.Close()

	var sleeps []time.Duration
	c := restTestClient(srv.URL, withMaxRetries(0))
	c.sleepFn = noSleep(&sleeps)

	err := c.do(t.Context(), "test.op", http.MethodGet, "/", nil, nil)
	if err == nil {
		t.Fatal("expected error with maxRetries=0, got nil")
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Fatalf("expected exactly 1 request with maxRetries=0, got %d", got)
	}
	if len(sleeps) != 0 {
		t.Fatalf("expected no sleeps with maxRetries=0, got %v", sleeps)
	}
}

func TestRESTClientPostNotRetriedOn503(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"success":false,"errors":[{"code":10001,"message":"unavailable"}]}`))
	}))
	defer srv.Close()

	var sleeps []time.Duration
	c := restTestClient(srv.URL)
	c.sleepFn = noSleep(&sleeps)

	err := c.do(t.Context(), "test.op", http.MethodPost, "/", map[string]string{"k": "v"}, nil)
	if err == nil {
		t.Fatal("expected error, POST on 503 should not be retried")
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Fatalf("expected exactly 1 request (no retry for POST), got %d", got)
	}
	if len(sleeps) != 0 {
		t.Fatalf("expected no sleeps for POST/503, got %v", sleeps)
	}
}

func TestRESTClientGetRetriedOn503(t *testing.T) {
	var requests int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&requests, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"success":false,"errors":[{"code":10001,"message":"unavailable"}]}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"errors":[],"result":{}}`))
	}))
	defer srv.Close()

	var sleeps []time.Duration
	c := restTestClient(srv.URL)
	c.sleepFn = noSleep(&sleeps)

	err := c.do(t.Context(), "test.op", http.MethodGet, "/", nil, nil)
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}
	if got := atomic.LoadInt32(&requests); got != 3 {
		t.Fatalf("expected exactly 3 requests, got %d", got)
	}
	if len(sleeps) != 2 {
		t.Fatalf("expected exactly 2 backoff sleeps, got %d: %v", len(sleeps), sleeps)
	}
}

func TestRESTClientBackoffJitterBounds(t *testing.T) {
	c := newRESTClient("test-token", withRetryBaseDelay(1*time.Second))
	for attempt := 1; attempt <= 6; attempt++ {
		for i := 0; i < 50; i++ {
			d := c.backoffDelay(attempt)
			shift := attempt - 1
			base := 1 * time.Second * time.Duration(1<<uint(shift))
			if base > maxRetryBackoff {
				base = maxRetryBackoff
			}
			lower := time.Duration(float64(base) * 0.75)
			upper := time.Duration(float64(base) * 1.25)
			if upper > maxRetryBackoff {
				upper = maxRetryBackoff
			}
			if d < lower || d > upper {
				t.Fatalf("attempt %d: backoff %v out of jitter bounds [%v, %v]", attempt, d, lower, upper)
			}
			if d > maxRetryBackoff {
				t.Fatalf("attempt %d: backoff %v exceeds cap %v", attempt, d, maxRetryBackoff)
			}
		}
	}
}

func TestParseRetryAfterSeconds(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"0", 0, true},
		{"5", 5 * time.Second, true},
		{"", 0, false},
		{"-1", 0, false},
		{"not-a-number-or-date", 0, false},
	}
	for _, tc := range cases {
		got, ok := parseRetryAfter(tc.in)
		if ok != tc.ok {
			t.Errorf("parseRetryAfter(%q) ok = %v, want %v", tc.in, ok, tc.ok)
			continue
		}
		if ok && got != tc.want {
			t.Errorf("parseRetryAfter(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseRetryAfterHTTPDate(t *testing.T) {
	future := time.Now().Add(10 * time.Second).UTC().Format(http.TimeFormat)
	d, ok := parseRetryAfter(future)
	if !ok {
		t.Fatalf("expected HTTP-date Retry-After to parse, got ok=false")
	}
	if d <= 0 || d > 11*time.Second {
		t.Fatalf("expected delay close to 10s, got %v", d)
	}
}

func TestIsIdempotentRESTMethod(t *testing.T) {
	idempotent := []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete}
	for _, m := range idempotent {
		if !isIdempotentRESTMethod(m) {
			t.Errorf("expected %s to be idempotent", m)
		}
	}
	nonIdempotent := []string{http.MethodPost, http.MethodPatch}
	for _, m := range nonIdempotent {
		if isIdempotentRESTMethod(m) {
			t.Errorf("expected %s to NOT be idempotent", m)
		}
	}
}

func TestWithRetryBaseDelayOption(t *testing.T) {
	c := newRESTClient("t", withRetryBaseDelay(2*time.Second))
	if c.retryBaseDelay != 2*time.Second {
		t.Fatalf("expected retryBaseDelay=2s, got %v", c.retryBaseDelay)
	}
}

func TestExhaustedRetriesErrorNamesAttemptsAndElapsed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"success":false,"errors":[{"code":10000,"message":"rate limited"}]}`))
	}))
	defer srv.Close()

	var sleeps []time.Duration
	c := restTestClient(srv.URL, withMaxRetries(2))
	c.sleepFn = noSleep(&sleeps)

	err := c.do(t.Context(), "test.op", http.MethodGet, "/", nil, nil)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	msg := err.Error()
	if !strings.Contains(msg, "attempt") || !strings.Contains(msg, strconv.Itoa(3)) {
		t.Fatalf("expected error to name attempt count, got: %s", msg)
	}
}

// TestDecodeEnvelope_ClassifiedErrors pins TASK-012's contract: envelope
// failures on auth/quota statuses surface as their typed error classes with
// the HTTP status attached, so the daemon's mapError classifies them
// instead of reporting a blanket 502 upstream_error.
func TestDecodeEnvelope_ClassifiedErrors(t *testing.T) {
	body := []byte(`{"success":false,"errors":[{"code":9109,"message":"Unauthorized to access requested resource"}]}`)
	tests := []struct {
		name       string
		statusCode int
		check      func(*testing.T, error)
	}{
		{
			name:       "401 becomes R2AuthError",
			statusCode: http.StatusUnauthorized,
			check: func(t *testing.T, err error) {
				var authErr *R2AuthError
				if !errors.As(err, &authErr) {
					t.Fatalf("want *R2AuthError, got %T", err)
				}
				if ErrorStatus(err) != http.StatusUnauthorized {
					t.Errorf("ErrorStatus = %d, want 401", ErrorStatus(err))
				}
			},
		},
		{
			name:       "403 becomes R2AccessDeniedError",
			statusCode: http.StatusForbidden,
			check: func(t *testing.T, err error) {
				var deniedErr *R2AccessDeniedError
				if !errors.As(err, &deniedErr) {
					t.Fatalf("want *R2AccessDeniedError, got %T", err)
				}
				if ErrorStatus(err) != http.StatusForbidden {
					t.Errorf("ErrorStatus = %d, want 403", ErrorStatus(err))
				}
			},
		},
		{
			name:       "429 becomes R2QuotaError",
			statusCode: http.StatusTooManyRequests,
			check: func(t *testing.T, err error) {
				var quotaErr *R2QuotaError
				if !errors.As(err, &quotaErr) {
					t.Fatalf("want *R2QuotaError, got %T", err)
				}
				if ErrorStatus(err) != http.StatusTooManyRequests {
					t.Errorf("ErrorStatus = %d, want 429", ErrorStatus(err))
				}
			},
		},
		{
			name:       "500 stays plain R2Error with status",
			statusCode: http.StatusInternalServerError,
			check: func(t *testing.T, err error) {
				var r2e *R2Error
				if !errors.As(err, &r2e) {
					t.Fatalf("want *R2Error, got %T", err)
				}
				if ErrorStatus(err) != http.StatusInternalServerError {
					t.Errorf("ErrorStatus = %d, want 500", ErrorStatus(err))
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := decodeEnvelope("TestOp", body, tc.statusCode, nil)
			if err == nil {
				t.Fatal("decodeEnvelope returned nil error for failed envelope")
			}
			tc.check(t, err)
		})
	}
}
