package cosmoflare

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// NotificationAction is a single R2 object event action.
type NotificationAction string

// The exact action names accepted by the R2 event notifications API.
const (
	ActionPutObject               NotificationAction = "PutObject"
	ActionCopyObject              NotificationAction = "CopyObject"
	ActionDeleteObject            NotificationAction = "DeleteObject"
	ActionCompleteMultipartUpload NotificationAction = "CompleteMultipartUpload"
	ActionLifecycleDeletion       NotificationAction = "LifecycleDeletion"
)

// validNotificationActions is the closed set of actions the API accepts.
var validNotificationActions = map[NotificationAction]bool{
	ActionPutObject:               true,
	ActionCopyObject:              true,
	ActionDeleteObject:            true,
	ActionCompleteMultipartUpload: true,
	ActionLifecycleDeletion:       true,
}

// NotificationRule filters which objects send which actions to the queue.
type NotificationRule struct {
	Actions     []NotificationAction `json:"actions"`
	Description string               `json:"description,omitempty"`
	Prefix      string               `json:"prefix,omitempty"`
	Suffix      string               `json:"suffix,omitempty"`
}

// NotificationRuleResult is a rule as returned by the API, with identity
// metadata the server assigns.
type NotificationRuleResult struct {
	NotificationRule
	RuleID    string    `json:"ruleId"`
	CreatedAt time.Time `json:"createdAt"`
}

// QueueNotification is one queue's rule set on a bucket (GET/list result).
type QueueNotification struct {
	QueueID   string                   `json:"queueId"`
	QueueName string                   `json:"queueName"`
	Rules     []NotificationRuleResult `json:"rules"`
}

// BucketNotificationService manages R2 event notification rules, routing
// per-object events (created/deleted/copied/multipart-completed/
// lifecycle-expired) to Cloudflare Queues over the REST API.
type BucketNotificationService struct {
	accountID string
	rest      restClient
}

// NewBucketNotificationService creates a service for managing R2 event
// notification rules. The target queue must already exist.
func NewBucketNotificationService(accountID, apiToken string, opts ...BucketNotificationOption) *BucketNotificationService {
	s := &BucketNotificationService{
		accountID: accountID,
		rest:      newRESTClient(apiToken),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// BucketNotificationOption configures the service.
type BucketNotificationOption func(*BucketNotificationService)

// WithBucketNotificationHTTPClient sets a custom HTTP client.
func WithBucketNotificationHTTPClient(c *http.Client) BucketNotificationOption {
	return func(s *BucketNotificationService) { s.rest.httpClient = c }
}

// WithBucketNotificationBaseURL overrides the REST API base URL.
func WithBucketNotificationBaseURL(u string) BucketNotificationOption {
	return func(s *BucketNotificationService) { s.rest.baseURL = u }
}

// WithBucketNotificationJurisdiction sets the cf-r2-jurisdiction header value.
func WithBucketNotificationJurisdiction(j string) BucketNotificationOption {
	return func(s *BucketNotificationService) { s.rest.jurisdiction = j }
}

func (s *BucketNotificationService) configPath(bucket string) string {
	return fmt.Sprintf("/accounts/%s/event_notifications/r2/%s/configuration", s.accountID, url.PathEscape(bucket))
}

// queuePath addresses one queue's rule set under the bucket configuration.
func (s *BucketNotificationService) queuePath(bucket, queueID string) string {
	return s.configPath(bucket) + "/queues/" + url.PathEscape(queueID)
}

// validateRules checks the rule set client-side so obvious mistakes fail
// before any HTTP call: at least one rule, non-empty unique known actions.
func validateNotificationRules(op string, rules []NotificationRule) error {
	if len(rules) == 0 {
		return validationError(op, "at least one rule is required")
	}
	for i, rule := range rules {
		if len(rule.Actions) == 0 {
			return validationError(op, fmt.Sprintf("rule %d: at least one action is required", i))
		}
		seen := make(map[NotificationAction]bool, len(rule.Actions))
		for _, a := range rule.Actions {
			if !validNotificationActions[a] {
				return validationError(op, fmt.Sprintf("rule %d: invalid action %q (valid: PutObject, CopyObject, DeleteObject, CompleteMultipartUpload, LifecycleDeletion)", i, a))
			}
			if seen[a] {
				return validationError(op, fmt.Sprintf("rule %d: duplicate action %q (actions must be unique)", i, a))
			}
			seen[a] = true
		}
	}
	return nil
}

// List returns every queue wired to the bucket.
func (s *BucketNotificationService) List(ctx context.Context, bucket string) ([]QueueNotification, error) {
	const op = "BucketNotificationList"
	var result struct {
		BucketName string              `json:"bucketName"`
		Queues     []QueueNotification `json:"queues"`
	}
	if err := s.rest.do(ctx, op, http.MethodGet, s.configPath(bucket), nil, &result); err != nil {
		return nil, err
	}
	if result.Queues == nil {
		result.Queues = []QueueNotification{}
	}
	return result.Queues, nil
}

// Get returns one queue's rules; unknown queue → API error surfaces.
func (s *BucketNotificationService) Get(ctx context.Context, bucket, queueID string) (*QueueNotification, error) {
	const op = "BucketNotificationGet"
	var result struct {
		BucketName string              `json:"bucketName"`
		Queues     []QueueNotification `json:"queues"`
	}
	if err := s.rest.do(ctx, op, http.MethodGet, s.queuePath(bucket, queueID), nil, &result); err != nil {
		return nil, err
	}
	// The single-queue endpoint wraps the queue in the same queues array.
	for i := range result.Queues {
		if result.Queues[i].QueueID == queueID || len(result.Queues) == 1 {
			q := result.Queues[i]
			if q.Rules == nil {
				q.Rules = []NotificationRuleResult{}
			}
			return &q, nil
		}
	}
	return nil, newError(op, fmt.Sprintf("queue %s not found in response", queueID), nil)
}

// Set creates/replaces the queue's rule set (PUT). len(rules)==0 → validation
// error; duplicate actions within a rule → validation error.
func (s *BucketNotificationService) Set(ctx context.Context, bucket, queueID string, rules []NotificationRule) error {
	const op = "BucketNotificationSet"
	if err := validateNotificationRules(op, rules); err != nil {
		return err
	}
	body := struct {
		Rules []NotificationRule `json:"rules"`
	}{Rules: rules}
	return s.rest.do(ctx, op, http.MethodPut, s.queuePath(bucket, queueID), body, nil)
}

// Delete removes ruleIds from the queue, or the whole queue config when ids
// is empty.
func (s *BucketNotificationService) Delete(ctx context.Context, bucket, queueID string, ruleIDs []string) error {
	const op = "BucketNotificationDelete"
	var body interface{}
	if len(ruleIDs) > 0 {
		body = struct {
			RuleIDs []string `json:"ruleIds"`
		}{RuleIDs: ruleIDs}
	}
	return s.rest.do(ctx, op, http.MethodDelete, s.queuePath(bucket, queueID), body, nil)
}
