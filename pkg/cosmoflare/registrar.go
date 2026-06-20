package cosmoflare

import (
	"context"
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
