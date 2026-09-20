package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// RegistrarInfo is the registration overlay for a single domain, derived from
// the Cloudflare Registrar API.
//
// NOTE on AutoRenew: cloudflare-go v0.116.0 does not expose an auto-renew flag
// on the read model (cloudflare.RegistrarDomain). The field exists only on the
// write-side cloudflare.RegistrarDomainConfiguration. There is therefore no
// readable signal for auto-renew in the list/get responses, so AutoRenew is
// always reported as false by List. The field is retained on this struct so
// callers (and future SDK versions that surface the value) have a stable shape.
type RegistrarInfo struct {
	Registrar     string     `json:"registrar"` // "cloudflare" | "external"
	RegistrarName string     `json:"registrar_name"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"` // nil when external/unknown
	AutoRenew     bool       `json:"auto_renew"`
	TransferLock  bool       `json:"transfer_lock"`
}

// RegistrarService reads Cloudflare Registrar registration data for the domains
// in a single account.
type RegistrarService struct {
	cf        *cloudflare.API
	accountID string
}

// NewRegistrarService constructs a RegistrarService bound to a Cloudflare API
// client and account ID.
func NewRegistrarService(cf *cloudflare.API, accountID string) *RegistrarService {
	return &RegistrarService{cf: cf, accountID: accountID}
}

// NewRegistrarServiceFromCreds builds a RegistrarService from an account ID and
// API token, constructing the underlying Cloudflare client (mirrors the other
// *FromCreds constructors in this package).
func NewRegistrarServiceFromCreds(accountID, apiToken string) (*RegistrarService, error) {
	if accountID == "" {
		return nil, validationError("NewRegistrarService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewRegistrarService", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewRegistrarService", "failed to create Cloudflare API client", err)
	}
	return &RegistrarService{cf: cf, accountID: accountID}, nil
}

// List returns registration info for every Cloudflare-registered domain in the
// account, keyed by domain name. Domains registered elsewhere are absent from
// the map — the caller marks those "external".
func (s *RegistrarService) List(ctx context.Context) (map[string]RegistrarInfo, error) {
	if s.accountID == "" {
		return nil, validationError("RegistrarService.List", "account ID is required")
	}
	domains, err := s.cf.RegistrarDomains(ctx, s.accountID)
	if err != nil {
		return nil, newError("RegistrarService.List", "list registrar domains", err)
	}
	out := make(map[string]RegistrarInfo, len(domains))
	for _, d := range domains {
		info := RegistrarInfo{
			Registrar:     "cloudflare",
			RegistrarName: d.CurrentRegistrar,
			TransferLock:  d.Locked,
		}
		// cloudflare.RegistrarDomain.ExpiresAt is a time.Time; a zero value
		// signals "unknown / not set" so we only attach a non-nil pointer when
		// a real expiry is present.
		if !d.ExpiresAt.IsZero() {
			expires := d.ExpiresAt
			info.ExpiresAt = &expires
		}
		// d.ID carries the domain name in the Registrar API responses.
		out[d.ID] = info
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// FEAT-030 — Registrar operations depth.
//
// cloudflare-go v0.116.0 covers GET list/get, transfer-in, cancel-transfer
// and the PUT domain-configuration endpoints. Register (POST
// /accounts/{account_id}/registrar/domains), renew, contacts and DNSSEC are
// hand-rolled via the client's raw-request seam (cf.Raw), mirroring the
// dataset recommendation that Registrar endpoints bypass the lagging SDK.
//
// Permissions: the Registrar API has no dedicated UI permission group — it
// falls back to standard Account/Billing scopes (read: "Account Settings"
// Read; register/transfer/renew also need "Account Settings" Edit plus
// "Billing"). See
// docs/research/2026-09-17-cf-perms-next-waves-tiers/qwen-results.md (Wave 1).
// ---------------------------------------------------------------------------

// RegistrarTransferSteps mirrors the transfer-in checklist the API returns
// for a domain. Each step string is a status (e.g. "pending", "complete").
type RegistrarTransferSteps struct {
	UnlockDomain      string `json:"unlock_domain"`
	DisablePrivacy    string `json:"disable_privacy"`
	EnterAuthCode     string `json:"enter_auth_code"`
	ApproveTransfer   string `json:"approve_transfer"`
	AcceptFoa         string `json:"accept_foa"`
	CanCancelTransfer bool   `json:"can_cancel_transfer"`
}

// RegistrarContact is a single registration contact (registrant, billing,
// admin or tech), mirroring cloudflare-go's RegistrantContact shape.
type RegistrarContact struct {
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Organization string `json:"organization,omitempty"`
	Address      string `json:"address,omitempty"`
	Address2     string `json:"address2,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	Zip          string `json:"zip,omitempty"`
	Country      string `json:"country,omitempty"`
	Phone        string `json:"phone,omitempty"`
	Email        string `json:"email,omitempty"`
	Fax          string `json:"fax,omitempty"`
}

// RegistrarContacts is the full contact set for a domain. Nil roles are
// omitted from the update payload so untouched roles keep their current
// values.
type RegistrarContacts struct {
	Registrant *RegistrarContact `json:"registrant,omitempty"`
	Billing    *RegistrarContact `json:"billing,omitempty"`
	Admin      *RegistrarContact `json:"admin,omitempty"`
	Tech       *RegistrarContact `json:"tech,omitempty"`
}

// RegistrarDSRecord is a single DS (Delegation Signer) record that must be
// published at the parent zone for DNSSEC delegation.
type RegistrarDSRecord struct {
	KeyTag     int    `json:"key_tag"`
	Algorithm  int    `json:"algorithm"`
	DigestType int    `json:"digest_type"`
	Digest     string `json:"digest"`
}

// RegistrarDNSSEC is the DNSSEC state for a registrar domain plus the DS
// records the parent zone must publish while delegation is active.
type RegistrarDNSSEC struct {
	Status    string              `json:"status"`
	DSRecords []RegistrarDSRecord `json:"ds_records,omitempty"`
}

// RegistrarDomainDetail is the full read model for a single registrar
// domain. Unlike the read-only List overlay (RegistrarInfo), this exposes
// the transfer-in checklist, registry statuses and the registrant contact.
type RegistrarDomainDetail struct {
	Name             string                 `json:"name"`
	Available        bool                   `json:"available"`
	SupportedTLD     bool                   `json:"supported_tld"`
	CanRegister      bool                   `json:"can_register"`
	TransferIn       RegistrarTransferSteps `json:"transfer_in"`
	CurrentRegistrar string                 `json:"current_registrar"`
	ExpiresAt        *time.Time             `json:"expires_at,omitempty"`
	RegistryStatuses string                 `json:"registry_statuses,omitempty"`
	Locked           bool                   `json:"locked"`
	CreatedAt        *time.Time             `json:"created_at,omitempty"`
	UpdatedAt        *time.Time             `json:"updated_at,omitempty"`
	Registrant       *RegistrarContact      `json:"registrant_contact,omitempty"`
}

// validate checks the service invariants shared by every operation.
func (s *RegistrarService) validate(domain string) error {
	if s.accountID == "" {
		return validationError("RegistrarService", "account ID is required")
	}
	if domain == "" {
		return validationError("RegistrarService", "domain name is required")
	}
	return nil
}

// registrarRaw issues a raw Cloudflare API request through the client's
// Raw seam and decodes the result envelope into out (when non-nil).
func (s *RegistrarService) registrarRaw(ctx context.Context, op, method, path string, body, out any) error {
	resp, err := s.cf.Raw(ctx, method, path, body, nil)
	if err != nil {
		return newError("RegistrarService."+op, strings.ToLower(method)+" "+path, err)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(resp.Result, out); err != nil {
		return newError("RegistrarService."+op, "decode response for "+path, err)
	}
	return nil
}

func registrarContactFromCF(c cloudflare.RegistrantContact) *RegistrarContact {
	return &RegistrarContact{
		FirstName:    c.FirstName,
		LastName:     c.LastName,
		Organization: c.Organization,
		Address:      c.Address,
		Address2:     c.Address2,
		City:         c.City,
		State:        c.State,
		Zip:          c.Zip,
		Country:      c.Country,
		Phone:        c.Phone,
		Email:        c.Email,
		Fax:          c.Fax,
	}
}

func registrarContactToCF(c RegistrarContact) cloudflare.RegistrantContact {
	return cloudflare.RegistrantContact{
		FirstName:    c.FirstName,
		LastName:     c.LastName,
		Organization: c.Organization,
		Address:      c.Address,
		Address2:     c.Address2,
		City:         c.City,
		State:        c.State,
		Zip:          c.Zip,
		Country:      c.Country,
		Phone:        c.Phone,
		Email:        c.Email,
		Fax:          c.Fax,
	}
}

// registrarDetailFromCF maps the SDK read model to the exported detail
// struct, normalising zero timestamps to nil (the API omits them for
// domains not yet registered with Cloudflare).
func registrarDetailFromCF(d cloudflare.RegistrarDomain) RegistrarDomainDetail {
	out := RegistrarDomainDetail{
		Name:         d.ID,
		Available:    d.Available,
		SupportedTLD: d.SupportedTLD,
		CanRegister:  d.CanRegister,
		TransferIn: RegistrarTransferSteps{
			UnlockDomain:      d.TransferIn.UnlockDomain,
			DisablePrivacy:    d.TransferIn.DisablePrivacy,
			EnterAuthCode:     d.TransferIn.EnterAuthCode,
			ApproveTransfer:   d.TransferIn.ApproveTransfer,
			AcceptFoa:         d.TransferIn.AcceptFoa,
			CanCancelTransfer: d.TransferIn.CanCancelTransfer,
		},
		CurrentRegistrar: d.CurrentRegistrar,
		RegistryStatuses: d.RegistryStatuses,
		Locked:           d.Locked,
		Registrant:       registrarContactFromCF(d.RegistrantContact),
	}
	if !d.ExpiresAt.IsZero() {
		t := d.ExpiresAt
		out.ExpiresAt = &t
	}
	if !d.CreatedAt.IsZero() {
		t := d.CreatedAt
		out.CreatedAt = &t
	}
	if !d.UpdatedAt.IsZero() {
		t := d.UpdatedAt
		out.UpdatedAt = &t
	}
	return out
}

// Get returns the full registrar state for a single domain, including the
// transfer-in checklist and registry statuses.
func (s *RegistrarService) Get(ctx context.Context, domain string) (RegistrarDomainDetail, error) {
	if err := s.validate(domain); err != nil {
		return RegistrarDomainDetail{}, err
	}
	d, err := s.cf.RegistrarDomain(ctx, s.accountID, domain)
	if err != nil {
		return RegistrarDomainDetail{}, newError("RegistrarService.Get", "get registrar domain "+domain, err)
	}
	return registrarDetailFromCF(d), nil
}

// Register registers (or initiates a transfer-in of) a domain. The API
// returns the domain with a pending state that callers poll via Get.
func (s *RegistrarService) Register(ctx context.Context, domain string) (RegistrarDomainDetail, error) {
	if err := s.validate(domain); err != nil {
		return RegistrarDomainDetail{}, err
	}
	var result cloudflare.RegistrarDomain
	err := s.registrarRaw(ctx, "Register", http.MethodPost,
		fmt.Sprintf("/accounts/%s/registrar/domains", s.accountID),
		map[string]string{"name": domain}, &result)
	if err != nil {
		return RegistrarDomainDetail{}, err
	}
	return registrarDetailFromCF(result), nil
}

// TransferPreflight returns the sequencing warnings that must be resolved
// before initiating a transfer-in. Transfers require the domain to be
// unlocked (and privacy disabled) at the current registrar. The check is
// best-effort: a failed lookup degrades to a warning rather than blocking.
func (s *RegistrarService) TransferPreflight(ctx context.Context, domain string) ([]string, error) {
	if err := s.validate(domain); err != nil {
		return nil, err
	}
	d, err := s.cf.RegistrarDomain(ctx, s.accountID, domain)
	if err != nil {
		return []string{fmt.Sprintf("could not verify transfer readiness: %v — confirm the domain is unlocked at its current registrar before initiating", err)}, nil
	}
	var warnings []string
	if d.Locked {
		warnings = append(warnings, "domain is locked — unlock it at the current registrar before transferring (transfers REQUIRE an unlocked domain)")
	}
	if step := d.TransferIn.UnlockDomain; step != "" && !strings.EqualFold(step, "complete") {
		warnings = append(warnings, fmt.Sprintf("transfer-in step unlock_domain is %q — complete it at the current registrar first", step))
	}
	if step := d.TransferIn.DisablePrivacy; step != "" && !strings.EqualFold(step, "complete") {
		warnings = append(warnings, fmt.Sprintf("transfer-in step disable_privacy is %q — WHOIS privacy must be off at the current registrar", step))
	}
	return warnings, nil
}

// Transfer initiates a transfer of a domain from its current registrar into
// Cloudflare Registrar. Transfers are asynchronous: poll Get (or
// TransferIn.CanCancelTransfer) for the pending state.
func (s *RegistrarService) Transfer(ctx context.Context, domain string) (RegistrarDomainDetail, error) {
	if err := s.validate(domain); err != nil {
		return RegistrarDomainDetail{}, err
	}
	domains, err := s.cf.TransferRegistrarDomain(ctx, s.accountID, domain)
	if err != nil {
		return RegistrarDomainDetail{}, newError("RegistrarService.Transfer", "transfer domain "+domain, err)
	}
	if len(domains) == 0 {
		return RegistrarDomainDetail{}, newError("RegistrarService.Transfer", "transfer domain "+domain, fmt.Errorf("empty response"))
	}
	return registrarDetailFromCF(domains[0]), nil
}

// CancelTransfer cancels a pending transfer-in for a domain.
func (s *RegistrarService) CancelTransfer(ctx context.Context, domain string) (RegistrarDomainDetail, error) {
	if err := s.validate(domain); err != nil {
		return RegistrarDomainDetail{}, err
	}
	domains, err := s.cf.CancelRegistrarDomainTransfer(ctx, s.accountID, domain)
	if err != nil {
		return RegistrarDomainDetail{}, newError("RegistrarService.CancelTransfer", "cancel transfer for "+domain, err)
	}
	if len(domains) == 0 {
		return RegistrarDomainDetail{}, newError("RegistrarService.CancelTransfer", "cancel transfer for "+domain, fmt.Errorf("empty response"))
	}
	return registrarDetailFromCF(domains[0]), nil
}

// Renew renews a domain for one year. Renewals are only accepted inside the
// domain's renewable window; the API error is surfaced verbatim when it is
// not (e.g. "not within the renewal window").
func (s *RegistrarService) Renew(ctx context.Context, domain string) (RegistrarDomainDetail, error) {
	if err := s.validate(domain); err != nil {
		return RegistrarDomainDetail{}, err
	}
	var result cloudflare.RegistrarDomain
	err := s.registrarRaw(ctx, "Renew", http.MethodPost,
		fmt.Sprintf("/accounts/%s/registrar/domains/%s/renew", s.accountID, domain),
		nil, &result)
	if err != nil {
		return RegistrarDomainDetail{}, err
	}
	return registrarDetailFromCF(result), nil
}

// SetAutoRenew toggles auto-renew for a domain via a scoped PATCH so the
// other domain settings (lock, privacy) are not clobbered.
func (s *RegistrarService) SetAutoRenew(ctx context.Context, domain string, enabled bool) (RegistrarDomainDetail, error) {
	if err := s.validate(domain); err != nil {
		return RegistrarDomainDetail{}, err
	}
	var result cloudflare.RegistrarDomain
	err := s.registrarRaw(ctx, "SetAutoRenew", http.MethodPatch,
		fmt.Sprintf("/accounts/%s/registrar/domains/%s", s.accountID, domain),
		map[string]bool{"auto_renew": enabled}, &result)
	if err != nil {
		return RegistrarDomainDetail{}, err
	}
	return registrarDetailFromCF(result), nil
}

// SetLock toggles the transfer lock for a domain (locked domains cannot be
// transferred away).
func (s *RegistrarService) SetLock(ctx context.Context, domain string, locked bool) (RegistrarDomainDetail, error) {
	if err := s.validate(domain); err != nil {
		return RegistrarDomainDetail{}, err
	}
	var result cloudflare.RegistrarDomain
	err := s.registrarRaw(ctx, "SetLock", http.MethodPatch,
		fmt.Sprintf("/accounts/%s/registrar/domains/%s", s.accountID, domain),
		map[string]bool{"locked": locked}, &result)
	if err != nil {
		return RegistrarDomainDetail{}, err
	}
	return registrarDetailFromCF(result), nil
}

// GetContacts returns the registrant contact for a domain. The Registrar
// read model only exposes the registrant role; billing/admin/tech are
// write-only via UpdateContacts.
func (s *RegistrarService) GetContacts(ctx context.Context, domain string) (RegistrarContacts, error) {
	detail, err := s.Get(ctx, domain)
	if err != nil {
		return RegistrarContacts{}, err
	}
	return RegistrarContacts{Registrant: detail.Registrant}, nil
}

// UpdateContacts replaces the supplied contact roles for a domain. Nil
// roles are omitted from the payload and keep their current values.
func (s *RegistrarService) UpdateContacts(ctx context.Context, domain string, contacts RegistrarContacts) (RegistrarContacts, error) {
	if err := s.validate(domain); err != nil {
		return RegistrarContacts{}, err
	}
	if contacts.Registrant == nil && contacts.Billing == nil && contacts.Admin == nil && contacts.Tech == nil {
		return RegistrarContacts{}, validationError("RegistrarService.UpdateContacts", "at least one contact role is required")
	}
	var updated RegistrarContacts
	err := s.registrarRaw(ctx, "UpdateContacts", http.MethodPut,
		fmt.Sprintf("/accounts/%s/registrar/domains/%s/contacts", s.accountID, domain),
		contacts, &updated)
	if err != nil {
		return RegistrarContacts{}, err
	}
	return updated, nil
}

// GetDNSSEC returns the DNSSEC status for a domain and the DS records that
// must be published at the parent zone while delegation is active.
func (s *RegistrarService) GetDNSSEC(ctx context.Context, domain string) (RegistrarDNSSEC, error) {
	if err := s.validate(domain); err != nil {
		return RegistrarDNSSEC{}, err
	}
	var result RegistrarDNSSEC
	err := s.registrarRaw(ctx, "GetDNSSEC", http.MethodGet,
		fmt.Sprintf("/accounts/%s/registrar/domains/%s/dnssec", s.accountID, domain),
		nil, &result)
	if err != nil {
		return RegistrarDNSSEC{}, err
	}
	return result, nil
}

// EnableDNSSEC enables DNSSEC for a domain. The response carries the DS
// records that MUST be added at the parent zone's registrar for delegation
// to complete — callers surface them as a follow-up note, not a blocker.
func (s *RegistrarService) EnableDNSSEC(ctx context.Context, domain string) (RegistrarDNSSEC, error) {
	if err := s.validate(domain); err != nil {
		return RegistrarDNSSEC{}, err
	}
	var result RegistrarDNSSEC
	err := s.registrarRaw(ctx, "EnableDNSSEC", http.MethodPut,
		fmt.Sprintf("/accounts/%s/registrar/domains/%s/dnssec", s.accountID, domain),
		map[string]string{"status": "active"}, &result)
	if err != nil {
		return RegistrarDNSSEC{}, err
	}
	return result, nil
}

// DisableDNSSECPreflight returns the sequencing warnings that must be
// resolved before disabling DNSSEC: DS records must be removed at the
// parent zone first, or the domain becomes unresolvable. Best-effort — a
// failed lookup degrades to a warning.
func (s *RegistrarService) DisableDNSSECPreflight(ctx context.Context, domain string) ([]string, error) {
	current, err := s.GetDNSSEC(ctx, domain)
	if err != nil {
		return []string{fmt.Sprintf("could not verify DNSSEC state: %v — confirm DS records are removed at the parent zone before disabling", err)}, nil
	}
	var warnings []string
	if !strings.EqualFold(current.Status, "disabled") && len(current.DSRecords) > 0 {
		warnings = append(warnings, fmt.Sprintf("%d DS record(s) are published at the parent zone — remove them there BEFORE disabling DNSSEC or the domain will stop resolving", len(current.DSRecords)))
	}
	return warnings, nil
}

// DisableDNSSEC disables DNSSEC for a domain. Run DisableDNSSECPreflight
// first: DS records must already be removed at the parent zone.
func (s *RegistrarService) DisableDNSSEC(ctx context.Context, domain string) (RegistrarDNSSEC, error) {
	if err := s.validate(domain); err != nil {
		return RegistrarDNSSEC{}, err
	}
	var result RegistrarDNSSEC
	err := s.registrarRaw(ctx, "DisableDNSSEC", http.MethodPut,
		fmt.Sprintf("/accounts/%s/registrar/domains/%s/dnssec", s.accountID, domain),
		map[string]string{"status": "disabled"}, &result)
	if err != nil {
		return RegistrarDNSSEC{}, err
	}
	return result, nil
}
