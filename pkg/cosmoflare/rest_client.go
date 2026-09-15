package cosmoflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// defaultRESTBaseURL is the Cloudflare v4 API root shared by the REST
// services that the cloudflare-go dependency does not wrap.
const defaultRESTBaseURL = "https://api.cloudflare.com/client/v4"

// defaultMaxRetries and defaultRetryBaseDelay tune the rate-limit-aware
// retry behavior shared by every restClient. maxRetryBackoff caps the
// exponential backoff delay (before jitter) at 30s per the Cloudflare API
// rate-limit corpus (docs/research/2026-09-10-cf-limits-corpus).
const (
	defaultMaxRetries     = 3
	defaultRetryBaseDelay = 1 * time.Second
	maxRetryBackoff       = 30 * time.Second
)

// restClient is the shared transport for the hand-rolled REST services
// (bucket domains, lifecycle, notifications): bearer auth, the optional
// cf-r2-jurisdiction header, and the standard response envelope. Services
// embed it and add only their paths, validation, and typed methods.
type restClient struct {
	apiToken     string
	httpClient   *http.Client
	baseURL      string
	jurisdiction string // optional cf-r2-jurisdiction: default|eu|us|fedramp

	maxRetries     int
	retryBaseDelay time.Duration
	sleepFn        func(time.Duration)
	jitterFn       func() float64
}

// restClientOption configures retry behavior on a restClient. restClient
// is unexported (an internal transport embedded by REST-backed services),
// so these options are unexported too — named withMaxRetries /
// withRetryBaseDelay rather than the exported WithMaxRetries /
// WithInitialDelay already used by the generic retry.Do helper, to avoid a
// naming collision with that unrelated (currently unwired) retry package.
// sleepFn/jitterFn are test-only seams set directly by the package's own
// tests.
type restClientOption func(*restClient)

// withMaxRetries sets the maximum number of retry attempts for a request
// that receives a retryable status (429, or 502/503/504 on idempotent
// methods). withMaxRetries(0) disables retries deterministically.
func withMaxRetries(n int) restClientOption {
	return func(c *restClient) { c.maxRetries = n }
}

// withRetryBaseDelay sets the base delay used for exponential backoff when
// no Retry-After header is present (default 1s, doubled per retry, capped
// at 30s, ±25% jitter).
func withRetryBaseDelay(d time.Duration) restClientOption {
	return func(c *restClient) { c.retryBaseDelay = d }
}

func newRESTClient(apiToken string, opts ...restClientOption) restClient {
	c := restClient{
		apiToken:       apiToken,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		baseURL:        defaultRESTBaseURL,
		maxRetries:     defaultMaxRetries,
		retryBaseDelay: defaultRetryBaseDelay,
		sleepFn:        time.Sleep,
		jitterFn:       defaultJitter,
	}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// defaultJitter returns a multiplier in [0.75, 1.25] (±25% jitter).
func defaultJitter() float64 {
	return 0.75 + rand.Float64()*0.5
}

// isIdempotentRESTMethod reports whether method is safe to retry on a
// 502/503/504 response (the request may not have been processed for these
// codes, but only for idempotent methods is a blind retry safe).
func isIdempotentRESTMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

// parseRetryAfter parses a Retry-After header value: either an integer
// number of seconds, or an HTTP-date. Returns ok=false when the value is
// empty or unparseable, signaling the caller to fall back to backoff.
func parseRetryAfter(value string) (time.Duration, bool) {
	if value == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(value); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(value); err == nil {
		d := time.Until(t)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

// backoffDelay computes the exponential-backoff-with-jitter delay for the
// given attempt (1-indexed: the Nth request made, i.e. the delay before
// retry N). Base delay doubles per attempt, capped at maxRetryBackoff
// before jitter is applied.
func (c *restClient) backoffDelay(attempt int) time.Duration {
	base := c.retryBaseDelay
	if base <= 0 {
		base = defaultRetryBaseDelay
	}
	shift := attempt - 1
	if shift < 0 {
		shift = 0
	}
	if shift > 20 { // guard against overflow on pathological attempt counts
		shift = 20
	}
	d := base * time.Duration(1<<uint(shift))
	if d > maxRetryBackoff {
		d = maxRetryBackoff
	}
	jitter := 1.0
	if c.jitterFn != nil {
		jitter = c.jitterFn()
	}
	scaled := time.Duration(float64(d) * jitter)
	if scaled > maxRetryBackoff {
		scaled = maxRetryBackoff
	}
	if scaled < 0 {
		scaled = 0
	}
	return scaled
}

// shouldRetry decides whether the given response status is retryable for
// method, and if so, the delay to wait before retrying:
//   - 429: always retryable (the request was never processed). Honors an
//     exact Retry-After header value when present, else falls back to
//     backoff.
//   - 502/503/504: retryable only for idempotent methods (GET, HEAD, PUT,
//     DELETE) — a non-idempotent method like POST may have already been
//     processed, so it is not retried.
//   - anything else: not retryable.
func (c *restClient) shouldRetry(statusCode int, method string, header http.Header, attempt int) (bool, time.Duration) {
	switch statusCode {
	case http.StatusTooManyRequests:
		if d, ok := parseRetryAfter(header.Get("Retry-After")); ok {
			return true, d
		}
		return true, c.backoffDelay(attempt)
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		if !isIdempotentRESTMethod(method) {
			return false, 0
		}
		return true, c.backoffDelay(attempt)
	default:
		return false, 0
	}
}

// apiEnvelope is the standard Cloudflare API response wrapper.
type apiEnvelope struct {
	Success bool            `json:"success"`
	Errors  []apiErrorItem  `json:"errors"`
	Result  json.RawMessage `json:"result"`
}

type apiErrorItem struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// do performs an authenticated request against the API and decodes the
// standard envelope. When out is non-nil the raw result is unmarshalled
// into it.
//
// Rate-limit-aware retries: a 429 response is always retried (the request
// was never processed by the API), honoring an exact Retry-After value
// when the server sends one. A 502/503/504 response is retried only for
// idempotent methods (GET, HEAD, PUT, DELETE) — POST is not retried since
// the request may have already been processed. Both cases otherwise use
// exponential backoff with jitter, up to maxRetries additional attempts.
func (c *restClient) do(ctx context.Context, op, method, path string, body interface{}, out interface{}) error {
	bodyData, err := marshalBody(op, body)
	if err != nil {
		return err
	}

	data, statusCode, err := c.sendWithRetry(ctx, op, method, path, bodyData)
	if err != nil {
		return err
	}

	return decodeEnvelope(op, data, statusCode, out)
}

// marshalBody encodes a JSON request body, returning nil for a nil body.
func marshalBody(op string, body interface{}) ([]byte, error) {
	if body == nil {
		return nil, nil
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, newError(op, "failed to encode request body", err)
	}
	return data, nil
}

// buildRequest constructs one authenticated request attempt from the
// pre-marshalled body, applying the jurisdiction header when set.
func (c *restClient) buildRequest(ctx context.Context, method, path string, bodyData []byte) (*http.Request, error) {
	var reader io.Reader
	if bodyData != nil {
		reader = bytes.NewReader(bodyData)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	if c.jurisdiction != "" {
		req.Header.Set("cf-r2-jurisdiction", c.jurisdiction)
	}
	if bodyData != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// sendWithRetry performs the request attempt loop, returning the response
// body and status of the final (non-retried) attempt. See do for the
// retry policy.
func (c *restClient) sendWithRetry(ctx context.Context, op, method, path string, bodyData []byte) ([]byte, int, error) {
	sleepFn := c.sleepFn
	if sleepFn == nil {
		sleepFn = time.Sleep
	}

	start := time.Now()
	var data []byte
	var statusCode int
	attempt := 0
	for {
		attempt++

		req, err := c.buildRequest(ctx, method, path, bodyData)
		if err != nil {
			return nil, 0, newError(op, "failed to build request", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, 0, newError(op, "request failed", err)
		}

		// Always drain and close the body so the underlying connection is
		// reusable, whether or not this attempt is going to be retried.
		respData, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, 0, newError(op, "failed to read response body", readErr)
		}
		data = respData
		statusCode = resp.StatusCode

		retryable, delay := c.shouldRetry(statusCode, method, resp.Header, attempt)
		if retryable && attempt <= c.maxRetries {
			sleepFn(delay)
			continue
		}
		if retryable {
			elapsed := time.Since(start)
			return nil, 0, newError(op, fmt.Sprintf("giving up after %d attempt(s) over %s: HTTP %d", attempt, elapsed, statusCode), nil)
		}
		break
	}
	return data, statusCode, nil
}

// decodeEnvelope unmarshals the standard response envelope into out (when
// non-nil), translating envelope failures into op-scoped errors.
func decodeEnvelope(op string, data []byte, statusCode int, out interface{}) error {
	var env apiEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return newError(op, fmt.Sprintf("unexpected response (HTTP %d)", statusCode), err)
	}
	if !env.Success {
		msg := fmt.Sprintf("API returned errors (HTTP %d)", statusCode)
		if len(env.Errors) > 0 {
			msg = env.Errors[0].Message
		}
		return newError(op, msg, nil)
	}
	if out != nil && len(env.Result) > 0 {
		if err := json.Unmarshal(env.Result, out); err != nil {
			return newError(op, "failed to decode result", err)
		}
	}
	return nil
}
