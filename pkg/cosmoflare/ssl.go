package cosmoflare

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// SSLStatus represents the SSL/TLS encryption mode for a zone.
type SSLStatus struct {
	ID                string `json:"id"`
	Value             string `json:"value"`
	Editable          bool   `json:"editable"`
	CertificateStatus string `json:"certificate_status"`
	ModifiedOn        string `json:"modified_on,omitempty"`
}

// SSLVerification represents a certificate verification entry.
type SSLVerification struct {
	CertificateStatus  string `json:"certificate_status"`
	VerificationType   string `json:"verification_type"`
	ValidationMethod   string `json:"validation_method"`
	CertPackUUID       string `json:"cert_pack_uuid"`
	VerificationStatus bool   `json:"verification_status"`
	BrandCheck         bool   `json:"brand_check"`
}

// SSLSettings holds SSL/TLS-related zone settings.
type SSLSettings struct {
	MinTLSVersion          string `json:"min_tls_version"`
	AlwaysUseHTTPS         bool   `json:"always_use_https"`
	AutomaticHTTPSRewrites bool   `json:"automatic_https_rewrites"`
	UniversalSSL           bool   `json:"universal_ssl"`
}

// SSLOption is a functional option for SSL settings updates.
type SSLOption func(*sslConfig)

type sslConfig struct {
	minTLSVersion *string
	alwaysHTTPS   *bool
	autoRewrites  *bool
}

// WithMinTLSVersion sets the minimum TLS version ("1.0", "1.1", "1.2", "1.3").
func WithMinTLSVersion(v string) SSLOption {
	return func(c *sslConfig) { c.minTLSVersion = &v }
}

// WithAlwaysHTTPS enables or disables Always Use HTTPS.
func WithAlwaysHTTPS(b bool) SSLOption {
	return func(c *sslConfig) { c.alwaysHTTPS = &b }
}

// WithAutoHTTPSRewrites enables or disables Automatic HTTPS Rewrites.
func WithAutoHTTPSRewrites(b bool) SSLOption {
	return func(c *sslConfig) { c.autoRewrites = &b }
}

// SSLService implements SSL/TLS operations.
// SSL is zone-scoped — it uses a zone ID, not an account ID.
type SSLService struct {
	cf     *cloudflare.API
	zoneID string
}

// NewSSLService creates a new SSL service client.
func NewSSLService(api *cloudflare.API, zoneID string) (*SSLService, error) {
	if api == nil {
		return nil, validationError("NewSSLService", "cloudflare API client is required")
	}
	if zoneID == "" {
		return nil, validationError("NewSSLService", "zone ID is required")
	}
	return &SSLService{cf: api, zoneID: zoneID}, nil
}

// NewSSLServiceFromCreds creates an SSLService from zone ID and API token.
func NewSSLServiceFromCreds(zoneID, apiToken string) (*SSLService, error) {
	if zoneID == "" {
		return nil, validationError("NewSSLService", "zone ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewSSLService", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewSSLService", "failed to create Cloudflare API client", err)
	}
	return &SSLService{cf: cf, zoneID: zoneID}, nil
}

// GetSSL retrieves the current SSL/TLS encryption mode for the zone.
func (s *SSLService) GetSSL(ctx context.Context) (*SSLStatus, error) {
	resp, err := s.cf.ZoneSSLSettings(ctx, s.zoneID)
	if err != nil {
		return nil, newError("SSLService.GetSSL", "failed to get SSL settings", err)
	}
	return &SSLStatus{
		ID:                resp.ID,
		Value:             resp.Value,
		Editable:          resp.Editable,
		CertificateStatus: resp.CertificateStatus,
		ModifiedOn:        resp.ModifiedOn,
	}, nil
}

// UpdateSSL updates the SSL/TLS encryption mode for the zone.
// Valid values: "off", "flexible", "full", "strict".
func (s *SSLService) UpdateSSL(ctx context.Context, value string) (*SSLStatus, error) {
	if value == "" {
		return nil, validationError("SSLService.UpdateSSL", "SSL mode value is required")
	}

	resp, err := s.cf.UpdateZoneSSLSettings(ctx, s.zoneID, value)
	if err != nil {
		return nil, newError("SSLService.UpdateSSL", fmt.Sprintf("failed to update SSL mode to %q", value), err)
	}
	return &SSLStatus{
		ID:                resp.ID,
		Value:             resp.Value,
		Editable:          resp.Editable,
		CertificateStatus: resp.CertificateStatus,
		ModifiedOn:        resp.ModifiedOn,
	}, nil
}

// GetVerification retrieves Universal SSL verification details for the zone.
func (s *SSLService) GetVerification(ctx context.Context) ([]*SSLVerification, error) {
	results, err := s.cf.UniversalSSLVerificationDetails(ctx, s.zoneID)
	if err != nil {
		return nil, newError("SSLService.GetVerification", "failed to get SSL verification details", err)
	}

	verifications := make([]*SSLVerification, 0, len(results))
	for _, v := range results {
		verifications = append(verifications, &SSLVerification{
			CertificateStatus:  v.CertificateStatus,
			VerificationType:   v.VerificationType,
			ValidationMethod:   v.ValidationMethod,
			CertPackUUID:       v.CertPackUUID,
			VerificationStatus: v.VerificationStatus,
			BrandCheck:         v.BrandCheck,
		})
	}
	return verifications, nil
}

// GetSettings retrieves SSL/TLS-related zone settings.
func (s *SSLService) GetSettings(ctx context.Context) (*SSLSettings, error) {
	resp, err := s.cf.ZoneSettings(ctx, s.zoneID)
	if err != nil {
		return nil, newError("SSLService.GetSettings", "failed to get zone settings", err)
	}

	settings := &SSLSettings{}
	for _, setting := range resp.Result {
		switch setting.ID {
		case "min_tls_version":
			if v, ok := setting.Value.(string); ok {
				settings.MinTLSVersion = v
			}
		case "always_use_https":
			if v, ok := setting.Value.(string); ok {
				settings.AlwaysUseHTTPS = v == "on"
			}
		case "automatic_https_rewrites":
			if v, ok := setting.Value.(string); ok {
				settings.AutomaticHTTPSRewrites = v == "on"
			}
		}
	}

	ussl, err := s.cf.UniversalSSLSettingDetails(ctx, s.zoneID)
	if err == nil {
		settings.UniversalSSL = ussl.Enabled
	}

	return settings, nil
}

// UpdateSettings updates SSL/TLS-related zone settings.
func (s *SSLService) UpdateSettings(ctx context.Context, opts ...SSLOption) error {
	if len(opts) == 0 {
		return validationError("SSLService.UpdateSettings", "at least one setting option is required")
	}

	cfg := &sslConfig{}
	for _, o := range opts {
		o(cfg)
	}

	var settings []cloudflare.ZoneSetting

	if cfg.minTLSVersion != nil {
		settings = append(settings, cloudflare.ZoneSetting{
			ID:    "min_tls_version",
			Value: *cfg.minTLSVersion,
		})
	}
	if cfg.alwaysHTTPS != nil {
		val := "off"
		if *cfg.alwaysHTTPS {
			val = "on"
		}
		settings = append(settings, cloudflare.ZoneSetting{
			ID:    "always_use_https",
			Value: val,
		})
	}
	if cfg.autoRewrites != nil {
		val := "off"
		if *cfg.autoRewrites {
			val = "on"
		}
		settings = append(settings, cloudflare.ZoneSetting{
			ID:    "automatic_https_rewrites",
			Value: val,
		})
	}

	if len(settings) == 0 {
		return validationError("SSLService.UpdateSettings", "no settings to update")
	}

	_, err := s.cf.UpdateZoneSettings(ctx, s.zoneID, settings)
	if err != nil {
		return newError("SSLService.UpdateSettings", "failed to update SSL settings", err)
	}
	return nil
}
