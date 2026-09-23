package cosmoflare

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// TurnstileService implements Cloudflare Turnstile widget management.
//
// Turnstile widgets are ACCOUNT-scoped: every call targets the account
// passed to the constructor. Requires an API token with the account-level
// "Turnstile" permission (verified against the Qwen dataset): Turnstile:
// Read for listings, Turnstile: Edit for writes.
//
// The widget secret is returned by the API exactly once, on creation, and
// is never repeated by get/list — surface it to the operator immediately
// and tell them to store it. Rotate via the Cloudflare dashboard.
type TurnstileService struct {
	cf        *cloudflare.API
	accountID string
}

// NewTurnstileService creates a Turnstile service client for an account.
func NewTurnstileService(api *cloudflare.API, accountID string) (*TurnstileService, error) {
	if api == nil {
		return nil, validationError("NewTurnstileService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewTurnstileService", "account ID is required")
	}
	return &TurnstileService{cf: api, accountID: accountID}, nil
}

// NewTurnstileServiceFromCreds creates a TurnstileService from an account
// ID and API token.
func NewTurnstileServiceFromCreds(accountID, apiToken string) (*TurnstileService, error) {
	if accountID == "" {
		return nil, validationError("NewTurnstileServiceFromCreds", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewTurnstileServiceFromCreds", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewTurnstileServiceFromCreds", "failed to create Cloudflare API client", err)
	}
	return &TurnstileService{cf: cf, accountID: accountID}, nil
}

// TurnstileWidget is a Turnstile challenge widget. Secret is only ever
// populated by Create — the API never repeats it and get/list strip it.
type TurnstileWidget struct {
	SiteKey      string     `json:"sitekey"`
	Secret       string     `json:"secret,omitempty"`
	Name         string     `json:"name,omitempty"`
	Domains      []string   `json:"domains,omitempty"`
	Mode         string     `json:"mode,omitempty"`
	BotFightMode bool       `json:"bot_fight_mode,omitempty"`
	Region       string     `json:"region,omitempty"`
	OffLabel     bool       `json:"offlabel,omitempty"`
	CreatedOn    *time.Time `json:"created_on,omitempty"`
	ModifiedOn   *time.Time `json:"modified_on,omitempty"`
}

// TurnstileWidgetOption is a functional option for widget create/update.
// The update path treats every field as optional (patch semantics); the
// create path requires at least name and one hostname.
type TurnstileWidgetOption func(*turnstileWidgetConfig)

type turnstileWidgetConfig struct {
	name         *string
	hostnames    *[]string
	mode         *string
	botFightMode *bool
	region       *string
	offLabel     *bool
}

// WithTurnstileName sets the widget name.
func WithTurnstileName(name string) TurnstileWidgetOption {
	return func(c *turnstileWidgetConfig) { c.name = &name }
}

// WithTurnstileHostnames sets the hostnames the widget is served on.
func WithTurnstileHostnames(hostnames []string) TurnstileWidgetOption {
	return func(c *turnstileWidgetConfig) { c.hostnames = &hostnames }
}

// WithTurnstileMode sets the widget mode: managed, non-interactive, or
// invisible.
func WithTurnstileMode(mode string) TurnstileWidgetOption {
	return func(c *turnstileWidgetConfig) { c.mode = &mode }
}

// WithTurnstileBotFightMode enables the automatic blocking of likely bots.
func WithTurnstileBotFightMode(enabled bool) TurnstileWidgetOption {
	return func(c *turnstileWidgetConfig) { c.botFightMode = &enabled }
}

// WithTurnstileRegion sets the widget region (world by default).
func WithTurnstileRegion(region string) TurnstileWidgetOption {
	return func(c *turnstileWidgetConfig) { c.region = &region }
}

// WithTurnstileOffLabel hides the Cloudflare branding from the widget.
func WithTurnstileOffLabel(offLabel bool) TurnstileWidgetOption {
	return func(c *turnstileWidgetConfig) { c.offLabel = &offLabel }
}

// Create creates a Turnstile widget. The name and at least one hostname
// are required. The returned widget carries the secret key — the API shows
// it exactly once, so surface it immediately.
func (s *TurnstileService) Create(ctx context.Context, opts ...TurnstileWidgetOption) (*TurnstileWidget, error) {
	var cfg turnstileWidgetConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.name == nil || *cfg.name == "" {
		return nil, validationError("TurnstileService.Create", "widget name is required")
	}
	if cfg.hostnames == nil || len(*cfg.hostnames) == 0 {
		return nil, validationError("TurnstileService.Create", "at least one hostname is required")
	}

	params := cloudflare.CreateTurnstileWidgetParams{
		Name:    *cfg.name,
		Domains: *cfg.hostnames,
	}
	if cfg.mode != nil {
		params.Mode = *cfg.mode
	}
	if cfg.botFightMode != nil {
		params.BotFightMode = *cfg.botFightMode
	}
	if cfg.region != nil {
		params.Region = *cfg.region
	}
	if cfg.offLabel != nil {
		params.OffLabel = *cfg.offLabel
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	widget, err := s.cf.CreateTurnstileWidget(ctx, rc, params)
	if err != nil {
		return nil, newError("TurnstileService.Create", fmt.Sprintf("failed to create turnstile widget %q", *cfg.name), err)
	}
	return mapTurnstileWidget(widget, false), nil
}

// List lists the account's Turnstile widgets. Secrets are never included.
func (s *TurnstileService) List(ctx context.Context) ([]*TurnstileWidget, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	widgets, _, err := s.cf.ListTurnstileWidgets(ctx, rc, cloudflare.ListTurnstileWidgetParams{})
	if err != nil {
		return nil, newError("TurnstileService.List", fmt.Sprintf("failed to list turnstile widgets in account %q", s.accountID), err)
	}

	results := make([]*TurnstileWidget, 0, len(widgets))
	for _, widget := range widgets {
		results = append(results, mapTurnstileWidget(widget, true))
	}
	return results, nil
}

// Get retrieves a single Turnstile widget by its site key. The secret is
// never included — the API only returns it at creation time.
func (s *TurnstileService) Get(ctx context.Context, siteKey string) (*TurnstileWidget, error) {
	if siteKey == "" {
		return nil, validationError("TurnstileService.Get", "site key is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	widget, err := s.cf.GetTurnstileWidget(ctx, rc, siteKey)
	if err != nil {
		return nil, newError("TurnstileService.Get", fmt.Sprintf("failed to get turnstile widget %q", siteKey), err)
	}
	return mapTurnstileWidget(widget, true), nil
}

// Update updates a Turnstile widget by its site key. Only the fields
// covered by the given options change.
func (s *TurnstileService) Update(ctx context.Context, siteKey string, opts ...TurnstileWidgetOption) (*TurnstileWidget, error) {
	if siteKey == "" {
		return nil, validationError("TurnstileService.Update", "site key is required")
	}

	var cfg turnstileWidgetConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.isEmpty() {
		return nil, validationError("TurnstileService.Update", "at least one update option is required (name, hostname, mode, bot-fight-mode, off-label)")
	}

	// The API's update endpoint does not accept a region change, so it is
	// ignored here rather than silently failing server-side.
	params := cloudflare.UpdateTurnstileWidgetParams{SiteKey: siteKey}
	params.Name = cfg.name
	params.Domains = cfg.hostnames
	params.Mode = cfg.mode
	params.BotFightMode = cfg.botFightMode
	params.OffLabel = cfg.offLabel

	rc := cloudflare.AccountIdentifier(s.accountID)
	widget, err := s.cf.UpdateTurnstileWidget(ctx, rc, params)
	if err != nil {
		return nil, newError("TurnstileService.Update", fmt.Sprintf("failed to update turnstile widget %q", siteKey), err)
	}
	return mapTurnstileWidget(widget, true), nil
}

// Delete deletes a Turnstile widget by its site key. This action is
// irreversible.
func (s *TurnstileService) Delete(ctx context.Context, siteKey string) error {
	if siteKey == "" {
		return validationError("TurnstileService.Delete", "site key is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	if err := s.cf.DeleteTurnstileWidget(ctx, rc, siteKey); err != nil {
		return newError("TurnstileService.Delete", fmt.Sprintf("failed to delete turnstile widget %q", siteKey), err)
	}
	return nil
}

func (c *turnstileWidgetConfig) isEmpty() bool {
	return c.name == nil && c.hostnames == nil && c.mode == nil &&
		c.botFightMode == nil && c.region == nil && c.offLabel == nil
}

// mapTurnstileWidget maps an API widget onto the CLI shape. stripSecret
// drops the secret everywhere except the create response, where the API
// returns it exactly once.
func mapTurnstileWidget(widget cloudflare.TurnstileWidget, stripSecret bool) *TurnstileWidget {
	mapped := &TurnstileWidget{
		SiteKey:      widget.SiteKey,
		Secret:       widget.Secret,
		Name:         widget.Name,
		Domains:      widget.Domains,
		Mode:         widget.Mode,
		BotFightMode: widget.BotFightMode,
		Region:       widget.Region,
		OffLabel:     widget.OffLabel,
		CreatedOn:    widget.CreatedOn,
		ModifiedOn:   widget.ModifiedOn,
	}
	if stripSecret {
		mapped.Secret = ""
	}
	return mapped
}
