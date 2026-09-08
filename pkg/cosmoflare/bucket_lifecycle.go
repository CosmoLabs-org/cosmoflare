package cosmoflare

import (
	"context"
	"fmt"
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
	accountID string
	rest      restClient
}

// NewBucketLifecycleService creates a service for managing R2 lifecycle rules.
func NewBucketLifecycleService(accountID, apiToken string, opts ...BucketLifecycleOption) *BucketLifecycleService {
	s := &BucketLifecycleService{
		accountID: accountID,
		rest:      newRESTClient(apiToken),
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
	return func(s *BucketLifecycleService) { s.rest.httpClient = c }
}

// WithBucketLifecycleBaseURL overrides the REST API base URL.
func WithBucketLifecycleBaseURL(u string) BucketLifecycleOption {
	return func(s *BucketLifecycleService) { s.rest.baseURL = u }
}

// WithBucketLifecycleJurisdiction sets the cf-r2-jurisdiction header value.
func WithBucketLifecycleJurisdiction(j string) BucketLifecycleOption {
	return func(s *BucketLifecycleService) { s.rest.jurisdiction = j }
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
	if err := s.rest.do(ctx, op, http.MethodGet, s.lifecyclePath(bucket), nil, &result); err != nil {
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
	return s.rest.do(ctx, op, http.MethodPut, s.lifecyclePath(bucket), body, nil)
}
