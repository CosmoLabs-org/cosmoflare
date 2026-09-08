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

// LifecycleCondition is when a transition fires: Age (maxAge seconds) or Date.
type LifecycleCondition struct {
	Type   string  `json:"type"`             // "Age" | "Date"
	MaxAge *int64  `json:"maxAge,omitempty"` // seconds; required for Age
	Date   *string `json:"date,omitempty"`   // RFC3339; required for Date
}

// LifecycleConditions scopes a lifecycle rule to an object key prefix.
type LifecycleConditions struct {
	Prefix string `json:"prefix"` // empty string scopes the rule to all objects
}

// LifecycleTransition wraps the condition that triggers it.
type LifecycleTransition struct {
	Condition LifecycleCondition `json:"condition"`
}

// StorageClassTransition moves matching objects to another storage class.
// "InfrequentAccess" is currently the only supported storage class.
type StorageClassTransition struct {
	Condition    LifecycleCondition `json:"condition"`
	StorageClass string             `json:"storageClass"`
}

// LifecycleRule expires/aborts/transitions objects under a prefix. The ID is
// a human description that doubles as the rule identifier.
type LifecycleRule struct {
	ID                              string                   `json:"id"`
	Enabled                         bool                     `json:"enabled"`
	Conditions                      LifecycleConditions      `json:"conditions"`
	DeleteObjectsTransition         *LifecycleTransition     `json:"deleteObjectsTransition,omitempty"`
	AbortMultipartUploadsTransition *LifecycleTransition     `json:"abortMultipartUploadsTransition,omitempty"`
	StorageClassTransitions         []StorageClassTransition `json:"storageClassTransitions,omitempty"`
}

const maxLifecycleRules = 1000

// BucketLifecycleService manages R2 object lifecycle rules over the REST API.
// There is no per-rule delete endpoint: Set REPLACES the whole configuration.
type BucketLifecycleService struct {
	accountID    string
	apiToken     string
	httpClient   *http.Client
	baseURL      string
	jurisdiction string // optional cf-r2-jurisdiction header: default|eu|us|fedramp
}

// NewBucketLifecycleService creates a service for managing R2 lifecycle rules.
func NewBucketLifecycleService(accountID, apiToken string, opts ...BucketLifecycleOption) *BucketLifecycleService {
	s := &BucketLifecycleService{
		accountID:  accountID,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.cloudflare.com/client/v4",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// BucketLifecycleOption configures the service.
type BucketLifecycleOption func(*BucketLifecycleService)

// WithBucketLifecycleHTTPClient sets a custom HTTP client.
func WithBucketLifecycleHTTPClient(c *http.Client) BucketLifecycleOption {
	return func(s *BucketLifecycleService) { s.httpClient = c }
}

// WithBucketLifecycleBaseURL overrides the REST API base URL.
func WithBucketLifecycleBaseURL(u string) BucketLifecycleOption {
	return func(s *BucketLifecycleService) { s.baseURL = u }
}

// WithBucketLifecycleJurisdiction sets the cf-r2-jurisdiction header value.
func WithBucketLifecycleJurisdiction(j string) BucketLifecycleOption {
	return func(s *BucketLifecycleService) { s.jurisdiction = j }
}

// bucketLifecycleEnvelope is the standard Cloudflare API response wrapper.
type bucketLifecycleEnvelope struct {
	Success bool                 `json:"success"`
	Errors  []bucketLifecycleErr `json:"errors"`
	Result  json.RawMessage      `json:"result"`
}

type bucketLifecycleErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// do performs an authenticated request against the lifecycle API and decodes
// the standard envelope. When out is non-nil the raw result is unmarshalled
// into it.
func (s *BucketLifecycleService) do(ctx context.Context, op, method, path string, body interface{}, out interface{}) error {
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

	var env bucketLifecycleEnvelope
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

func (s *BucketLifecycleService) lifecyclePath(bucket string) string {
	return fmt.Sprintf("/accounts/%s/r2/buckets/%s/lifecycle", s.accountID, url.PathEscape(bucket))
}

// validateCondition checks an Age/Date condition shape.
func validateCondition(op string, c LifecycleCondition, ageOnly bool) error {
	if ageOnly && c.Type == "Date" {
		return validationError(op, "abortMultipartUploadsTransition only supports Age conditions, not Date")
	}
	switch c.Type {
	case "Age":
		if c.MaxAge == nil || *c.MaxAge <= 0 {
			return validationError(op, fmt.Sprintf("Age condition requires maxAge > 0 seconds (got %v)", c.MaxAge))
		}
	case "Date":
		if c.Date == nil {
			return validationError(op, "Date condition requires a date value")
		}
		if _, err := time.Parse(time.RFC3339, *c.Date); err != nil {
			return validationError(op, fmt.Sprintf("Date condition requires an RFC3339 date, got %q", *c.Date))
		}
	default:
		return validationError(op, fmt.Sprintf("condition type must be \"Age\" or \"Date\", got %q", c.Type))
	}
	return nil
}

// Get returns the bucket's lifecycle rules. The slice is empty and non-nil
// when the bucket has no rules.
func (s *BucketLifecycleService) Get(ctx context.Context, bucket string) ([]LifecycleRule, error) {
	const op = "BucketLifecycleGet"
	var result struct {
		Rules []LifecycleRule `json:"rules"`
	}
	if err := s.do(ctx, op, http.MethodGet, s.lifecyclePath(bucket), nil, &result); err != nil {
		return nil, err
	}
	if result.Rules == nil {
		result.Rules = []LifecycleRule{}
	}
	return result.Rules, nil
}

// Set REPLACES the bucket's whole lifecycle configuration with rules (PUT).
// A nil slice sends {"rules":[]}, clearing every rule.
func (s *BucketLifecycleService) Set(ctx context.Context, bucket string, rules []LifecycleRule) error {
	const op = "BucketLifecycleSet"
	if len(rules) > maxLifecycleRules {
		return validationError(op, fmt.Sprintf("too many rules: %d (maximum %d)", len(rules), maxLifecycleRules))
	}
	for i, rule := range rules {
		if rule.ID == "" {
			return validationError(op, fmt.Sprintf("rule %d: id is required", i))
		}
		if rule.DeleteObjectsTransition != nil {
			if err := validateCondition(op, rule.DeleteObjectsTransition.Condition, false); err != nil {
				return validationError(op, fmt.Sprintf("rule %d (%s): %s", i, rule.ID, err.Error()))
			}
		}
		if rule.AbortMultipartUploadsTransition != nil {
			if err := validateCondition(op, rule.AbortMultipartUploadsTransition.Condition, true); err != nil {
				return validationError(op, fmt.Sprintf("rule %d (%s): %s", i, rule.ID, err.Error()))
			}
		}
		for j, tr := range rule.StorageClassTransitions {
			if err := validateCondition(op, tr.Condition, false); err != nil {
				return validationError(op, fmt.Sprintf("rule %d (%s) storage class transition %d: %s", i, rule.ID, j, err.Error()))
			}
			if tr.StorageClass != "InfrequentAccess" {
				return validationError(op, fmt.Sprintf("rule %d (%s): storageClass must be \"InfrequentAccess\", got %q", i, rule.ID, tr.StorageClass))
			}
		}
	}
	if rules == nil {
		rules = []LifecycleRule{}
	}
	body := struct {
		Rules []LifecycleRule `json:"rules"`
	}{Rules: rules}
	return s.do(ctx, op, http.MethodPut, s.lifecyclePath(bucket), body, nil)
}
