package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// BucketPolicyService manages R2 bucket policy documents over the REST API,
// following the same hand-rolled restClient pattern as lifecycle/notifications.
type BucketPolicyService struct {
	accountID string
	rest      restClient
}

// NewBucketPolicyService creates a service for managing R2 bucket policies.
func NewBucketPolicyService(accountID, apiToken string, opts ...BucketPolicyOption) *BucketPolicyService {
	s := &BucketPolicyService{
		accountID: accountID,
		rest:      newRESTClient(apiToken),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// BucketPolicyOption configures the service.
type BucketPolicyOption func(*BucketPolicyService)

// WithBucketPolicyHTTPClient sets a custom HTTP client.
func WithBucketPolicyHTTPClient(c *http.Client) BucketPolicyOption {
	return func(s *BucketPolicyService) { s.rest.httpClient = c }
}

// WithBucketPolicyBaseURL overrides the REST API base URL.
func WithBucketPolicyBaseURL(u string) BucketPolicyOption {
	return func(s *BucketPolicyService) { s.rest.baseURL = u }
}

// WithBucketPolicyJurisdiction sets the cf-r2-jurisdiction header value.
func WithBucketPolicyJurisdiction(j string) BucketPolicyOption {
	return func(s *BucketPolicyService) { s.rest.jurisdiction = j }
}

func (s *BucketPolicyService) policyPath(bucket string) string {
	return "/accounts/" + s.accountID + "/r2/buckets/" + url.PathEscape(bucket) + "/policy"
}

// GetBucketPolicy returns the bucket's raw policy document.
func (s *BucketPolicyService) GetBucketPolicy(ctx context.Context, bucket string) (json.RawMessage, error) {
	const op = "BucketPolicyGet"
	if bucket == "" {
		return nil, validationError(op, "bucket name is required")
	}
	var result json.RawMessage
	if err := s.rest.do(ctx, op, http.MethodGet, s.policyPath(bucket), nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// SetBucketPolicy replaces the bucket's policy document. The policy is
// validated as well-formed JSON locally before any API call.
func (s *BucketPolicyService) SetBucketPolicy(ctx context.Context, bucket string, policy json.RawMessage) error {
	const op = "BucketPolicySet"
	if bucket == "" {
		return validationError(op, "bucket name is required")
	}
	if len(policy) == 0 {
		return validationError(op, "policy is required")
	}
	if !json.Valid(policy) {
		return validationError(op, "policy is not valid JSON")
	}
	return s.rest.do(ctx, op, http.MethodPut, s.policyPath(bucket), policy, nil)
}
