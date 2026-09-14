package cosmoflare

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// WAFPackage represents a WAF managed ruleset package.
type WAFPackage struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	DetectionMode string `json:"detection_mode"`
	Sensitivity   string `json:"sensitivity"`
	ActionMode    string `json:"action_mode"`
}

// WAFRule represents a single WAF rule within a package.
type WAFRule struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	PackageID   string `json:"package_id"`
	Group       struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"group"`
	Mode         string   `json:"mode"`
	DefaultMode  string   `json:"default_mode"`
	AllowedModes []string `json:"allowed_modes"`
}

// WAFGroup represents a WAF rule group within a package.
type WAFGroup struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	RulesCount         int    `json:"rules_count"`
	ModifiedRulesCount int    `json:"modified_rules_count"`
	PackageID          string `json:"package_id"`
	Mode               string `json:"mode"`
}

// FirewallRule represents a zone-level IP access rule.
type FirewallRule struct {
	ID            string `json:"id"`
	Mode          string `json:"mode"`
	Notes         string `json:"notes"`
	Configuration struct {
		Target string `json:"target"`
		Value  string `json:"value"`
	} `json:"configuration"`
}

// WAFService implements WAF and firewall operations.
type WAFService struct {
	cf     *cloudflare.API
	zoneID string
}

// NewWAFService creates a new WAF service client.
func NewWAFService(api *cloudflare.API, zoneID string) (*WAFService, error) {
	if api == nil {
		return nil, validationError("NewWAFService", "cloudflare API client is required")
	}
	if zoneID == "" {
		return nil, validationError("NewWAFService", "zone ID is required")
	}
	return &WAFService{cf: api, zoneID: zoneID}, nil
}

// NewWAFServiceFromCreds creates a WAFService from zone ID and API token.
func NewWAFServiceFromCreds(zoneID, apiToken string) (*WAFService, error) {
	if zoneID == "" {
		return nil, validationError("NewWAFService", "zone ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewWAFService", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewWAFService", "failed to create Cloudflare API client", err)
	}
	return &WAFService{cf: cf, zoneID: zoneID}, nil
}

// ListPackages returns all WAF packages for the zone.
func (s *WAFService) ListPackages(ctx context.Context) ([]*WAFPackage, error) {
	results, err := s.cf.ListWAFPackages(ctx, s.zoneID)
	if err != nil {
		return nil, newError("WAFService.ListPackages", "failed to list WAF packages", err)
	}
	packages := make([]*WAFPackage, 0, len(results))
	for _, p := range results {
		packages = append(packages, &WAFPackage{
			ID: p.ID, Name: p.Name, Description: p.Description,
			DetectionMode: p.DetectionMode, Sensitivity: p.Sensitivity, ActionMode: p.ActionMode,
		})
	}
	return packages, nil
}

// GetPackage retrieves a single WAF package.
func (s *WAFService) GetPackage(ctx context.Context, packageID string) (*WAFPackage, error) {
	if packageID == "" {
		return nil, validationError("WAFService.GetPackage", "package ID is required")
	}
	p, err := s.cf.WAFPackage(ctx, s.zoneID, packageID)
	if err != nil {
		return nil, notFound("WAFService.GetPackage", "", packageID, err)
	}
	return &WAFPackage{
		ID: p.ID, Name: p.Name, Description: p.Description,
		DetectionMode: p.DetectionMode, Sensitivity: p.Sensitivity, ActionMode: p.ActionMode,
	}, nil
}

// ListRules returns all WAF rules in a package.
func (s *WAFService) ListRules(ctx context.Context, packageID string) ([]*WAFRule, error) {
	if packageID == "" {
		return nil, validationError("WAFService.ListRules", "package ID is required")
	}
	results, err := s.cf.ListWAFRules(ctx, s.zoneID, packageID)
	if err != nil {
		return nil, newError("WAFService.ListRules", "failed to list WAF rules", err)
	}
	rules := make([]*WAFRule, 0, len(results))
	for _, r := range results {
		rule := &WAFRule{
			ID: r.ID, Description: r.Description, Priority: r.Priority,
			PackageID: r.PackageID, Mode: r.Mode, DefaultMode: r.DefaultMode,
			AllowedModes: r.AllowedModes,
		}
		rule.Group.ID = r.Group.ID
		rule.Group.Name = r.Group.Name
		rules = append(rules, rule)
	}
	return rules, nil
}

// GetRule retrieves a single WAF rule.
func (s *WAFService) GetRule(ctx context.Context, packageID, ruleID string) (*WAFRule, error) {
	if packageID == "" {
		return nil, validationError("WAFService.GetRule", "package ID is required")
	}
	if ruleID == "" {
		return nil, validationError("WAFService.GetRule", "rule ID is required")
	}
	r, err := s.cf.WAFRule(ctx, s.zoneID, packageID, ruleID)
	if err != nil {
		return nil, notFound("WAFService.GetRule", "", ruleID, err)
	}
	rule := &WAFRule{
		ID: r.ID, Description: r.Description, Priority: r.Priority,
		PackageID: r.PackageID, Mode: r.Mode, DefaultMode: r.DefaultMode,
		AllowedModes: r.AllowedModes,
	}
	rule.Group.ID = r.Group.ID
	rule.Group.Name = r.Group.Name
	return rule, nil
}

// UpdateRule changes the mode of a WAF rule (e.g., "block", "simulate", "disable").
func (s *WAFService) UpdateRule(ctx context.Context, packageID, ruleID, mode string) (*WAFRule, error) {
	if packageID == "" {
		return nil, validationError("WAFService.UpdateRule", "package ID is required")
	}
	if ruleID == "" {
		return nil, validationError("WAFService.UpdateRule", "rule ID is required")
	}
	if mode == "" {
		return nil, validationError("WAFService.UpdateRule", "mode is required")
	}
	r, err := s.cf.UpdateWAFRule(ctx, s.zoneID, packageID, ruleID, mode)
	if err != nil {
		return nil, newError("WAFService.UpdateRule", fmt.Sprintf("failed to update WAF rule %q", ruleID), err)
	}
	rule := &WAFRule{
		ID: r.ID, Description: r.Description, Priority: r.Priority,
		PackageID: r.PackageID, Mode: r.Mode, DefaultMode: r.DefaultMode,
		AllowedModes: r.AllowedModes,
	}
	return rule, nil
}

// ListAccessRules returns zone-level IP access rules.
func (s *WAFService) ListAccessRules(ctx context.Context) ([]*FirewallRule, error) {
	resp, err := s.cf.ListZoneAccessRules(ctx, s.zoneID, cloudflare.AccessRule{}, 1)
	if err != nil {
		return nil, newError("WAFService.ListAccessRules", "failed to list access rules", err)
	}
	rules := make([]*FirewallRule, 0, len(resp.Result))
	for _, r := range resp.Result {
		rule := &FirewallRule{
			ID: r.ID, Mode: r.Mode, Notes: r.Notes,
		}
		rule.Configuration.Target = r.Configuration.Target
		rule.Configuration.Value = r.Configuration.Value
		rules = append(rules, rule)
	}
	return rules, nil
}

// CreateAccessRule creates a zone-level IP access rule.
func (s *WAFService) CreateAccessRule(ctx context.Context, target, value, mode, notes string) (*FirewallRule, error) {
	if target == "" || value == "" {
		return nil, validationError("WAFService.CreateAccessRule", "target and value are required")
	}
	if mode == "" {
		return nil, validationError("WAFService.CreateAccessRule", "mode is required (block, challenge, whitelist, js_challenge)")
	}
	rule := cloudflare.AccessRule{
		Mode:  mode,
		Notes: notes,
		Configuration: cloudflare.AccessRuleConfiguration{
			Target: target,
			Value:  value,
		},
	}
	resp, err := s.cf.CreateZoneAccessRule(ctx, s.zoneID, rule)
	if err != nil {
		return nil, newError("WAFService.CreateAccessRule", "failed to create access rule", err)
	}
	r := resp.Result
	result := &FirewallRule{ID: r.ID, Mode: r.Mode, Notes: r.Notes}
	result.Configuration.Target = r.Configuration.Target
	result.Configuration.Value = r.Configuration.Value
	return result, nil
}

// DeleteAccessRule removes a zone-level IP access rule.
func (s *WAFService) DeleteAccessRule(ctx context.Context, ruleID string) error {
	if ruleID == "" {
		return validationError("WAFService.DeleteAccessRule", "rule ID is required")
	}
	_, err := s.cf.DeleteZoneAccessRule(ctx, s.zoneID, ruleID)
	if err != nil {
		return newError("WAFService.DeleteAccessRule", fmt.Sprintf("failed to delete access rule %q", ruleID), err)
	}
	return nil
}
