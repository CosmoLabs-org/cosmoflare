package cosmoflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// BucketDomainStatus is the activation state of an attached custom domain.
type BucketDomainStatus struct {
	Ownership string `json:"ownership"` // pending|active|deactivated|blocked|error|unknown
	SSL       string `json:"ssl"`       // initializing|pending|active|deactivated|error|unknown
}

// BucketDomain is a custom domain attached to an R2 bucket.
type BucketDomain struct {
	Domain   string              `json:"domain"`
	Enabled  bool                `json:"enabled"`
	Status   *BucketDomainStatus `json:"status,omitempty"`
	Ciphers  []string            `json:"ciphers,omitempty"`
	MinTLS   string              `json:"minTLS,omitempty"`
	ZoneID   string              `json:"zoneId,omitempty"`
	ZoneName string              `json:"zoneName,omitempty"`
}

// AttachBucketDomainRequest attaches a domain to a bucket.
type AttachBucketDomainRequest struct {
	Domain  string   `json:"domain"`
	Enabled bool     `json:"enabled"`
	ZoneID  string   `json:"zoneId"`
	Ciphers []string `json:"ciphers,omitempty"`
	MinTLS  string   `json:"minTLS,omitempty"` // 1.0|1.1|1.2|1.3, default API-side 1.0
}

// UpdateBucketDomainRequest changes settings of an attached domain.
type UpdateBucketDomainRequest struct {
	Enabled *bool     `json:"enabled,omitempty"`
	Ciphers []string  `json:"ciphers,omitempty"`
	MinTLS  *string   `json:"minTLS,omitempty"`
}

// BucketDomainService manages R2 bucket custom domains over the REST API.
type BucketDomainService struct {
	accountID    string
	apiToken     string
	httpClient   *http.Client
	baseURL      string
	jurisdiction string // optional cf-r2-jurisdiction header: default|eu|us|fedramp
	pollInterval time.Duration
	onPoll       func(*BucketDomain) // optional observer invoked after each Verify poll
}

// NewBucketDomainService creates a service for managing R2 bucket custom domains.
func NewBucketDomainService(accountID, apiToken string, opts ...BucketDomainOption) *BucketDomainService {
	s := &BucketDomainService{
		accountID:    accountID,
		apiToken:     apiToken,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		baseURL:      "https://api.cloudflare.com/client/v4",
		pollInterval: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// BucketDomainOption configures the service.
type BucketDomainOption func(*BucketDomainService)

// WithBucketDomainHTTPClient sets a custom HTTP client.
func WithBucketDomainHTTPClient(c *http.Client) BucketDomainOption {
	return func(s *BucketDomainService) { s.httpClient = c }
}

// WithBucketDomainBaseURL overrides the REST API base URL.
func WithBucketDomainBaseURL(u string) BucketDomainOption {
	return func(s *BucketDomainService) { s.baseURL = u }
}

// WithBucketDomainJurisdiction sets the cf-r2-jurisdiction header value.
func WithBucketDomainJurisdiction(j string) BucketDomainOption {
	return func(s *BucketDomainService) { s.jurisdiction = j }
}

// WithBucketDomainPollInterval sets the polling interval used by Verify.
func WithBucketDomainPollInterval(d time.Duration) BucketDomainOption {
	return func(s *BucketDomainService) { s.pollInterval = d }
}

// WithBucketDomainOnPoll registers an observer invoked with the domain state
// after every poll performed by Verify. Used by the CLI to print live status.
func WithBucketDomainOnPoll(fn func(*BucketDomain)) BucketDomainOption {
	return func(s *BucketDomainService) { s.onPoll = fn }
}

// envelope is the standard Cloudflare API response wrapper.
type bucketDomainEnvelope struct {
	Success bool              `json:"success"`
	Errors  []bucketDomainErr `json:"errors"`
	Result  json.RawMessage   `json:"result"`
}

type bucketDomainErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// do performs an authenticated request against the custom domains API and
// decodes the standard envelope. When out is non-nil the raw result is
// unmarshalled into it.
func (s *BucketDomainService) do(ctx context.Context, op, method, path string, body interface{}, out interface{}) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return newError(op, "failed to encode request body", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, reader)
	if err != nil {
		return newError(op, "failed to build request", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	if s.jurisdiction != "" {
		req.Header.Set("cf-r2-jurisdiction", s.jurisdiction)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return newError(op, "request failed", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return newError(op, "failed to read response body", err)
	}

	var env bucketDomainEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return newError(op, fmt.Sprintf("unexpected response (HTTP %d)", resp.StatusCode), err)
	}
	if !env.Success {
		msg := fmt.Sprintf("API returned errors (HTTP %d)", resp.StatusCode)
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

func (s *BucketDomainService) domainsPath(bucket string) string {
	return fmt.Sprintf("/accounts/%s/r2/buckets/%s/domains/custom", s.accountID, url.PathEscape(bucket))
}

// Attach attaches a custom domain to a bucket.
func (s *BucketDomainService) Attach(ctx context.Context, bucket string, req AttachBucketDomainRequest) (*BucketDomain, error) {
	const op = "BucketDomainAttach"
	if req.Domain == "" {
		return nil, validationError(op, "domain is required")
	}
	if req.ZoneID == "" {
		return nil, validationError(op, "zone ID is required (pass --zone-id or rely on auto-resolution)")
	}
	if req.MinTLS != "" && req.MinTLS != "1.0" && req.MinTLS != "1.1" && req.MinTLS != "1.2" && req.MinTLS != "1.3" {
		return nil, validationError(op, fmt.Sprintf("invalid minTLS %q: must be one of 1.0, 1.1, 1.2, 1.3", req.MinTLS))
	}
	var d BucketDomain
	if err := s.do(ctx, op, http.MethodPost, s.domainsPath(bucket), req, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// List returns all custom domains attached to a bucket.
func (s *BucketDomainService) List(ctx context.Context, bucket string) ([]BucketDomain, error) {
	const op = "BucketDomainList"
	var result struct {
		Domains []BucketDomain `json:"domains"`
	}
	if err := s.do(ctx, op, http.MethodGet, s.domainsPath(bucket), nil, &result); err != nil {
		return nil, err
	}
	if result.Domains == nil {
		result.Domains = []BucketDomain{}
	}
	return result.Domains, nil
}

// Get returns a single attached custom domain including its status.
func (s *BucketDomainService) Get(ctx context.Context, bucket, domain string) (*BucketDomain, error) {
	const op = "BucketDomainGet"
	var d BucketDomain
	path := s.domainsPath(bucket) + "/" + url.PathEscape(domain)
	if err := s.do(ctx, op, http.MethodGet, path, nil, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// Update changes settings of an attached custom domain.
func (s *BucketDomainService) Update(ctx context.Context, bucket, domain string, req UpdateBucketDomainRequest) (*BucketDomain, error) {
	const op = "BucketDomainUpdate"
	var d BucketDomain
	path := s.domainsPath(bucket) + "/" + url.PathEscape(domain)
	if err := s.do(ctx, op, http.MethodPut, path, req, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// Detach removes a custom domain from a bucket.
func (s *BucketDomainService) Detach(ctx context.Context, bucket, domain string) error {
	const op = "BucketDomainDetach"
	path := s.domainsPath(bucket) + "/" + url.PathEscape(domain)
	return s.do(ctx, op, http.MethodDelete, path, nil, nil)
}

// Verify polls Get until both ownership and SSL are active, a terminal state
// is reached, the context is done, or the timeout elapses (timeout <= 0 waits
// for the context only). It always returns the last observed domain; on
// timeout the error mentions both statuses.
func (s *BucketDomainService) Verify(ctx context.Context, bucket, domain string, timeout time.Duration) (*BucketDomain, error) {
	const op = "BucketDomainVerify"
	deadline := time.Time{}
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
	}

	var last *BucketDomain
	for {
		d, err := s.Get(ctx, bucket, domain)
		if err != nil {
			if last != nil && ctx.Err() != nil {
				return last, newError(op, "verification cancelled", ctx.Err())
			}
			return last, err
		}
		last = d
		if s.onPoll != nil {
			s.onPoll(d)
		}

		if d.Status != nil && d.Status.Ownership == "active" && d.Status.SSL == "active" {
			return d, nil
		}
		if terminal := d.Status != nil &&
			(isTerminalDomainState(d.Status.Ownership) || isTerminalDomainState(d.Status.SSL)); terminal {
			return d, newError(op, fmt.Sprintf("domain verification reached terminal state: ownership=%s ssl=%s", d.Status.Ownership, d.Status.SSL), nil)
		}

		if !deadline.IsZero() && !time.Now().Add(s.pollInterval).Before(deadline) {
			return d, s.verifyTimeoutError(d)
		}

		select {
		case <-ctx.Done():
			return d, newError(op, "verification cancelled", ctx.Err())
		case <-time.After(s.pollInterval):
		}
	}
}

func (s *BucketDomainService) verifyTimeoutError(d *BucketDomain) error {
	ownership, ssl := "unknown", "unknown"
	if d != nil && d.Status != nil {
		ownership, ssl = d.Status.Ownership, d.Status.SSL
	}
	return newError("BucketDomainVerify", fmt.Sprintf("verification timed out: ownership=%s ssl=%s", ownership, ssl), nil)
}

func isTerminalDomainState(state string) bool {
	switch state {
	case "deactivated", "blocked", "error":
		return true
	}
	return false
}
