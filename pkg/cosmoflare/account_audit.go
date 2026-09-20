package cosmoflare

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// AccountAuditAction is what was done in an audit-log entry.
type AccountAuditAction struct {
	Type   string `json:"type"`
	Result bool   `json:"result"`
}

// AccountAuditActor is who performed an audit-log action.
type AccountAuditActor struct {
	ID    string `json:"id,omitempty"`
	Email string `json:"email,omitempty"`
	IP    string `json:"ip,omitempty"`
	Type  string `json:"type,omitempty"`
}

// AccountAuditResource is what an audit-log action was performed on.
type AccountAuditResource struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
}

// AccountAuditLog is the public view of one Cloudflare account audit-log
// entry. This is the Cloudflare account audit trail (who changed what in
// your account), distinct from cosmoflare's local mutation audit log.
type AccountAuditLog struct {
	ID       string                 `json:"id"`
	When     string                 `json:"when"`
	Action   AccountAuditAction     `json:"action"`
	Actor    AccountAuditActor      `json:"actor"`
	Resource AccountAuditResource   `json:"resource"`
	OwnerID  string                 `json:"ownerId,omitempty"`
	OldValue string                 `json:"oldValue,omitempty"`
	NewValue string                 `json:"newValue,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AccountAuditFilter carries the query filters supported by the
// /accounts/{id}/logs/audit endpoint. Since/Before accept RFC3339
// timestamps or bare YYYY-MM-DD dates; PerPage (1-100) defaults to the
// API's 20.
type AccountAuditFilter struct {
	ActorID   string
	ActorType string
	Action    string
	Zone      string
	Since     string
	Before    string
	PerPage   int
}

// AccountAuditService queries the Cloudflare account audit log over the
// REST API (cloudflare-go wraps only the legacy /audit_logs endpoint, not
// the /logs/audit surface, so this service uses the shared REST client and
// decodes into cloudflare-go's typed AuditLog resource).
type AccountAuditService struct {
	accountID string
	rest      restClient
}

// AccountAuditOption configures the service.
type AccountAuditOption func(*AccountAuditService)

// WithAccountAuditHTTPClient sets a custom HTTP client.
func WithAccountAuditHTTPClient(c *http.Client) AccountAuditOption {
	return func(s *AccountAuditService) { s.rest.httpClient = c }
}

// WithAccountAuditBaseURL overrides the REST API base URL.
func WithAccountAuditBaseURL(u string) AccountAuditOption {
	return func(s *AccountAuditService) { s.rest.baseURL = u }
}

// NewAccountAuditService creates an audit-log query service.
func NewAccountAuditService(accountID, apiToken string, opts ...AccountAuditOption) *AccountAuditService {
	s := &AccountAuditService{
		accountID: accountID,
		rest:      newRESTClient(apiToken),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// NewAccountAuditServiceFromCreds creates an audit-log query service,
// validating up front that credentials were provided so callers fail fast
// instead of issuing an unauthenticated request.
func NewAccountAuditServiceFromCreds(accountID, apiToken string, opts ...AccountAuditOption) (*AccountAuditService, error) {
	if accountID == "" {
		return nil, validationError("NewAccountAuditService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewAccountAuditService", "API token is required")
	}
	return NewAccountAuditService(accountID, apiToken, opts...), nil
}

// Logs queries the account audit log with the given filters.
func (s *AccountAuditService) Logs(ctx context.Context, f AccountAuditFilter) ([]AccountAuditLog, error) {
	const op = "AccountAuditLogs"
	query, err := f.toQuery(op)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/accounts/%s/logs/audit", s.accountID)
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	var entries []cloudflare.AuditLog
	if err := s.rest.do(ctx, op, http.MethodGet, path, nil, &entries); err != nil {
		return nil, err
	}
	return accountAuditLogsFromCF(entries), nil
}

// History returns the change history of a single audit-log entry: the
// chain of prior states that led to the entry's current value.
func (s *AccountAuditService) History(ctx context.Context, logID string) ([]AccountAuditLog, error) {
	const op = "AccountAuditHistory"
	if logID == "" {
		return nil, validationError(op, "audit log ID is required")
	}
	path := fmt.Sprintf("/accounts/%s/logs/audit/%s/history", s.accountID, url.PathEscape(logID))

	var entries []cloudflare.AuditLog
	if err := s.rest.do(ctx, op, http.MethodGet, path, nil, &entries); err != nil {
		return nil, err
	}
	return accountAuditLogsFromCF(entries), nil
}

// toQuery renders the filter as HTTP query parameters, validating the
// timestamp and page-size inputs first.
func (f AccountAuditFilter) toQuery(op string) (url.Values, error) {
	v := url.Values{}
	set := func(key, value string) {
		if value != "" {
			v.Add(key, value)
		}
	}
	set("actor.id", f.ActorID)
	set("actor.type", f.ActorType)
	set("action", f.Action)
	set("zone", f.Zone)
	for _, ts := range []struct{ key, value string }{
		{"since", f.Since},
		{"before", f.Before},
	} {
		if ts.value == "" {
			continue
		}
		if err := validateAuditTimestamp(ts.value); err != nil {
			return nil, validationError(op, fmt.Sprintf("invalid --%s value %q: %v", ts.key, ts.value, err))
		}
		v.Add(ts.key, ts.value)
	}
	if f.PerPage != 0 {
		if f.PerPage < 1 || f.PerPage > 100 {
			return nil, validationError(op, fmt.Sprintf("invalid per-page %d: must be between 1 and 100", f.PerPage))
		}
		v.Add("per_page", strconv.Itoa(f.PerPage))
	}
	return v, nil
}

// validateAuditTimestamp accepts RFC3339 timestamps or bare YYYY-MM-DD
// dates, both of which the audit-log endpoint understands.
func validateAuditTimestamp(value string) error {
	if _, err := time.Parse(time.RFC3339, value); err == nil {
		return nil
	}
	if _, err := time.Parse("2006-01-02", value); err == nil {
		return nil
	}
	return fmt.Errorf("must be an RFC3339 timestamp or a YYYY-MM-DD date")
}

// accountAuditLogsFromCF maps the cloudflare-go audit-log type to the
// public view, rendering timestamps as RFC3339 strings.
func accountAuditLogsFromCF(entries []cloudflare.AuditLog) []AccountAuditLog {
	out := make([]AccountAuditLog, 0, len(entries))
	for _, e := range entries {
		out = append(out, AccountAuditLog{
			ID:     e.ID,
			When:   e.When.Format(time.RFC3339),
			Action: AccountAuditAction{Type: e.Action.Type, Result: e.Action.Result},
			Actor: AccountAuditActor{
				ID:    e.Actor.ID,
				Email: e.Actor.Email,
				IP:    e.Actor.IP,
				Type:  e.Actor.Type,
			},
			Resource: AccountAuditResource{ID: e.Resource.ID, Type: e.Resource.Type},
			OwnerID:  e.Owner.ID,
			OldValue: e.OldValue,
			NewValue: e.NewValue,
			Metadata: e.Metadata,
		})
	}
	return out
}
