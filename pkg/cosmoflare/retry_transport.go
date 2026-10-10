package cosmoflare

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// RetryTransport (FEAT-061) gives the Cloudflare control plane bounded
// retry: 2 retries with exponential backoff and jitter on 429 (the server
// did not process the request — any method) and on 5xx for replay-safe
// requests only. It delegates the loop to the shared retry.Do so the
// backoff, jitter and context-abort semantics stay in exactly one place.
//
// A 5xx after a POST/PATCH is NOT retried here: the server may have
// processed the request, and duplicating a create is worse than surfacing
// the error. Requests whose body cannot be replayed (no http.Request.GetBody)
// are one-shot for the same reason. Errors from the underlying transport
// flow through retry.Do's classification unchanged.
type RetryTransport struct {
	Base http.RoundTripper
}

// retryableResponseError carries a response status into retry.Do's
// isRetryable classification (429 and >=500 are retryable there).
type retryableResponseError struct {
	status int
}

func (e *retryableResponseError) Error() string   { return fmt.Sprintf("retryable status %d", e.status) }
func (e *retryableResponseError) StatusCode() int { return e.status }

// nonRetryableResponseError marks a response the policy must not retry
// (5xx on a non-idempotent method) — plain text so isRetryable rejects it.
type nonRetryableResponseError struct {
	status int
}

func (e *nonRetryableResponseError) Error() string { return fmt.Sprintf("status %d on non-idempotent request", e.status) }

// idempotentMethods are the methods whose 5xx responses may be retried: a
// repeated GET/HEAD/OPTIONS/PUT/DELETE cannot create a second resource.
var idempotentMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
	http.MethodPut:     true,
	http.MethodDelete:  true,
}

func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	// A body that cannot be rewound makes every retry unsafe — run once.
	replayable := req.Body == nil || req.GetBody != nil
	if !replayable {
		return base.RoundTrip(req)
	}

	var last *http.Response
	err := Do(req.Context(), func() error {
		if req.Body != nil && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return err
			}
			req.Body = body
		}
		resp, err := base.RoundTrip(req)
		if err != nil {
			return err // transport error: retry.Do classifies (net timeouts retry)
		}
		switch {
		case resp.StatusCode == http.StatusTooManyRequests:
			drainAndClose(resp)
			return &retryableResponseError{status: resp.StatusCode}
		case resp.StatusCode >= 500 && idempotentMethods[req.Method]:
			drainAndClose(resp)
			return &retryableResponseError{status: resp.StatusCode}
		case resp.StatusCode >= 500:
			// Non-idempotent method: hand the response back unretried.
			last = resp
			return &nonRetryableResponseError{status: resp.StatusCode}
		default:
			last = resp
			return nil
		}
	},
		WithMaxRetries(2),
		WithInitialDelay(250*time.Millisecond),
		WithMaxDelay(5*time.Second),
		WithJitter(true),
	)
	if err == nil {
		return last, nil
	}
	// A non-retryable 5xx already parked its response in `last`; surface the
	// response itself, not the synthesized stop-error.
	if last != nil {
		return last, nil
	}
	// Exhausted retries or transport failure: nothing to surface but the error.
	return nil, err
}

// drainAndClose reads and discards a response body so the connection can be
// reused before the next attempt.
func drainAndClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	resp.Body.Close()
}
