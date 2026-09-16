package cosmoflare

import (
	"context"
	"fmt"
	"time"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

// Supported WAF list kinds.
const (
	WAFListKindIP       = "ip"
	WAFListKindASN      = "asn"
	WAFListKindRedirect = "redirect"
	WAFListKindHostname = "hostname"
)

// WAFList represents an account-level WAF rules list (IP, ASN, redirect, ...).
type WAFList struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Kind        string `json:"kind"`
	NumItems    int    `json:"num_items"`
	CreatedOn   string `json:"created_on,omitempty"`
	ModifiedOn  string `json:"modified_on,omitempty"`
}

// WAFListItem represents a single entry in a WAF list. The meaningful field
// depends on the list kind: IP for "ip"/"hostname" lists, ASN for "asn" lists.
type WAFListItem struct {
	ID      string `json:"id"`
	IP      string `json:"ip,omitempty"`
	ASN     uint32 `json:"asn,omitempty"`
	Comment string `json:"comment,omitempty"`
}

// WAFManagedRuleset summarizes the result of a managed-ruleset phase update.
type WAFManagedRuleset struct {
	Phase     string `json:"phase"`
	RulesetID string `json:"ruleset_id,omitempty"`
	Mode      string `json:"mode"`
	NumRules  int    `json:"num_rules"`
}

// WAFListService implements account-level WAF list and managed-ruleset
// operations.
type WAFListService struct {
	cf        *cloudflare.API
	accountID string
}

// NewWAFListService creates a new WAF list service client.
func NewWAFListService(api *cloudflare.API, accountID string) (*WAFListService, error) {
	if api == nil {
		return nil, validationError("NewWAFListService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewWAFListService", "account ID is required")
	}
	return &WAFListService{cf: api, accountID: accountID}, nil
}

// NewWAFListServiceFromCreds creates a WAFListService from account ID and API token.
func NewWAFListServiceFromCreds(accountID, apiToken string) (*WAFListService, error) {
	if accountID == "" {
		return nil, validationError("NewWAFListService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewWAFListService", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewWAFListService", "failed to create Cloudflare API client", err)
	}
	return &WAFListService{cf: cf, accountID: accountID}, nil
}

// wafListFromCF maps an SDK List into our shape.
func wafListFromCF(l cloudflare.List) *WAFList {
	out := &WAFList{
		ID:          l.ID,
		Name:        l.Name,
		Description: l.Description,
		Kind:        l.Kind,
		NumItems:    l.NumItems,
	}
	if l.CreatedOn != nil {
		out.CreatedOn = l.CreatedOn.Format(time.RFC3339)
	}
	if l.ModifiedOn != nil {
		out.ModifiedOn = l.ModifiedOn.Format(time.RFC3339)
	}
	return out
}

// wafListItemFromCF maps an SDK ListItem into our shape.
func wafListItemFromCF(i cloudflare.ListItem) *WAFListItem {
	out := &WAFListItem{ID: i.ID, Comment: i.Comment}
	if i.IP != nil {
		out.IP = *i.IP
	}
	if i.ASN != nil {
		out.ASN = *i.ASN
	}
	return out
}

// wafListItemsToCF maps our items into the SDK create/replace request shape.
func wafListItemsToCF(items []WAFListItem) []cloudflare.ListItemCreateRequest {
	out := make([]cloudflare.ListItemCreateRequest, 0, len(items))
	for _, it := range items {
		req := cloudflare.ListItemCreateRequest{Comment: it.Comment}
		if it.IP != "" {
			ip := it.IP
			req.IP = &ip
		}
		if it.ASN != 0 {
			asn := it.ASN
			req.ASN = &asn
		}
		out = append(out, req)
	}
	return out
}

// ListLists returns all WAF lists on the account.
func (s *WAFListService) ListLists(ctx context.Context) ([]*WAFList, error) {
	results, err := s.cf.ListLists(ctx, cloudflare.AccountIdentifier(s.accountID), cloudflare.ListListsParams{})
	if err != nil {
		return nil, newError("WAFListService.ListLists", "failed to list WAF lists", err)
	}
	lists := make([]*WAFList, 0, len(results))
	for _, l := range results {
		lists = append(lists, wafListFromCF(l))
	}
	return lists, nil
}

// CreateList creates a new WAF list. kind must be one of ip, asn, redirect,
// hostname.
func (s *WAFListService) CreateList(ctx context.Context, name, kind, description string) (*WAFList, error) {
	if name == "" {
		return nil, validationError("WAFListService.CreateList", "list name is required")
	}
	if kind == "" {
		return nil, validationError("WAFListService.CreateList", "list kind is required (ip, asn, redirect, hostname)")
	}
	switch kind {
	case WAFListKindIP, WAFListKindASN, WAFListKindRedirect, WAFListKindHostname:
	default:
		return nil, validationError("WAFListService.CreateList",
			fmt.Sprintf("invalid list kind %q (must be ip, asn, redirect, or hostname)", kind))
	}
	l, err := s.cf.CreateList(ctx, cloudflare.AccountIdentifier(s.accountID), cloudflare.ListCreateParams{
		Name:        name,
		Description: description,
		Kind:        kind,
	})
	if err != nil {
		return nil, newError("WAFListService.CreateList", fmt.Sprintf("failed to create WAF list %q", name), err)
	}
	return wafListFromCF(l), nil
}

// UpdateList updates a list's description. The Cloudflare API does not support
// renaming lists, so name must be empty (or equal to the current name).
func (s *WAFListService) UpdateList(ctx context.Context, id, name, description string) (*WAFList, error) {
	if id == "" {
		return nil, validationError("WAFListService.UpdateList", "list ID is required")
	}
	if name != "" {
		return nil, validationError("WAFListService.UpdateList",
			"renaming a WAF list is not supported by the Cloudflare API; only the description can be updated")
	}
	l, err := s.cf.UpdateList(ctx, cloudflare.AccountIdentifier(s.accountID), cloudflare.ListUpdateParams{
		ID:          id,
		Description: description,
	})
	if err != nil {
		return nil, newError("WAFListService.UpdateList", fmt.Sprintf("failed to update WAF list %q", id), err)
	}
	return wafListFromCF(l), nil
}

// DeleteList removes a WAF list.
func (s *WAFListService) DeleteList(ctx context.Context, id string) error {
	if id == "" {
		return validationError("WAFListService.DeleteList", "list ID is required")
	}
	_, err := s.cf.DeleteList(ctx, cloudflare.AccountIdentifier(s.accountID), id)
	if err != nil {
		return newError("WAFListService.DeleteList", fmt.Sprintf("failed to delete WAF list %q", id), err)
	}
	return nil
}

// ListItems returns all items in a WAF list.
func (s *WAFListService) ListItems(ctx context.Context, listID string) ([]*WAFListItem, error) {
	if listID == "" {
		return nil, validationError("WAFListService.ListItems", "list ID is required")
	}
	results, err := s.cf.ListListItems(ctx, cloudflare.AccountIdentifier(s.accountID), cloudflare.ListListItemsParams{ID: listID})
	if err != nil {
		return nil, newError("WAFListService.ListItems", fmt.Sprintf("failed to list items for WAF list %q", listID), err)
	}
	items := make([]*WAFListItem, 0, len(results))
	for _, i := range results {
		items = append(items, wafListItemFromCF(i))
	}
	return items, nil
}

// AddItems appends items to a WAF list and returns the list's full item set.
func (s *WAFListService) AddItems(ctx context.Context, listID string, items []WAFListItem) ([]*WAFListItem, error) {
	if listID == "" {
		return nil, validationError("WAFListService.AddItems", "list ID is required")
	}
	if len(items) == 0 {
		return nil, validationError("WAFListService.AddItems", "at least one item is required")
	}
	results, err := s.cf.CreateListItems(ctx, cloudflare.AccountIdentifier(s.accountID), cloudflare.ListCreateItemsParams{
		ID:    listID,
		Items: wafListItemsToCF(items),
	})
	if err != nil {
		return nil, newError("WAFListService.AddItems", fmt.Sprintf("failed to add items to WAF list %q", listID), err)
	}
	out := make([]*WAFListItem, 0, len(results))
	for _, i := range results {
		out = append(out, wafListItemFromCF(i))
	}
	return out, nil
}

// ReplaceItems replaces the entire item set of a WAF list and returns the
// resulting items.
func (s *WAFListService) ReplaceItems(ctx context.Context, listID string, items []WAFListItem) ([]*WAFListItem, error) {
	if listID == "" {
		return nil, validationError("WAFListService.ReplaceItems", "list ID is required")
	}
	if len(items) == 0 {
		return nil, validationError("WAFListService.ReplaceItems", "at least one item is required")
	}
	results, err := s.cf.ReplaceListItems(ctx, cloudflare.AccountIdentifier(s.accountID), cloudflare.ListReplaceItemsParams{
		ID:    listID,
		Items: wafListItemsToCF(items),
	})
	if err != nil {
		return nil, newError("WAFListService.ReplaceItems", fmt.Sprintf("failed to replace items in WAF list %q", listID), err)
	}
	out := make([]*WAFListItem, 0, len(results))
	for _, i := range results {
		out = append(out, wafListItemFromCF(i))
	}
	return out, nil
}

// UpdateManagedRuleset turns the managed ruleset in the given account phase
// (e.g. http_request_firewall_managed) on or off by rewriting the phase
// entrypoint rules' enabled flag. The phase must already contain a managed
// ruleset entrypoint; this call cannot install one.
func (s *WAFListService) UpdateManagedRuleset(ctx context.Context, phase, mode string) (*WAFManagedRuleset, error) {
	if phase == "" {
		return nil, validationError("WAFListService.UpdateManagedRuleset", "ruleset phase is required")
	}
	if mode != "on" && mode != "off" {
		return nil, validationError("WAFListService.UpdateManagedRuleset", fmt.Sprintf("invalid mode %q (must be on or off)", mode))
	}
	rc := cloudflare.AccountIdentifier(s.accountID)
	rs, err := s.cf.GetEntrypointRuleset(ctx, rc, phase)
	if err != nil {
		if isNotFound(err) {
			return nil, validationError("WAFListService.UpdateManagedRuleset",
				fmt.Sprintf("no managed ruleset is installed in phase %q; nothing to enable or disable", phase))
		}
		return nil, newError("WAFListService.UpdateManagedRuleset",
			fmt.Sprintf("failed to read entrypoint for phase %q", phase), err)
	}
	if len(rs.Rules) == 0 {
		return nil, validationError("WAFListService.UpdateManagedRuleset",
			fmt.Sprintf("phase %q entrypoint has no rules; nothing to enable or disable", phase))
	}
	enabled := mode == "on"
	rules := make([]cloudflare.RulesetRule, len(rs.Rules))
	copy(rules, rs.Rules)
	for i := range rules {
		rules[i].Enabled = &enabled
	}
	updated, err := s.cf.UpdateEntrypointRuleset(ctx, rc, cloudflare.UpdateEntrypointRulesetParams{
		Phase: phase,
		Rules: rules,
	})
	if err != nil {
		return nil, newError("WAFListService.UpdateManagedRuleset",
			fmt.Sprintf("failed to update managed ruleset phase %q", phase), err)
	}
	return &WAFManagedRuleset{
		Phase:     phase,
		RulesetID: updated.ID,
		Mode:      mode,
		NumRules:  len(updated.Rules),
	}, nil
}
