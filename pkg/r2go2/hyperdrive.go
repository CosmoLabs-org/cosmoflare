package r2go2

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// HyperdriveConfig represents a Cloudflare Hyperdrive configuration.
type HyperdriveConfig struct {
	ID      string                  `json:"id"`
	Name    string                  `json:"name"`
	Origin  HyperdriveOrigin       `json:"origin"`
	Caching HyperdriveCaching      `json:"caching"`
}

// HyperdriveOrigin holds the origin database connection details.
type HyperdriveOrigin struct {
	Database       string `json:"database"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Scheme         string `json:"scheme"`
	User           string `json:"user"`
	AccessClientID string `json:"access_client_id,omitempty"`
}

// HyperdriveCaching holds caching settings for a Hyperdrive config.
type HyperdriveCaching struct {
	Disabled             *bool `json:"disabled,omitempty"`
	MaxAge               int   `json:"max_age,omitempty"`
	StaleWhileRevalidate int   `json:"stale_while_revalidate,omitempty"`
}

// HyperdriveOriginConfig holds the origin details including secrets for create/update.
type HyperdriveOriginConfig struct {
	Database           string `json:"database"`
	Host               string `json:"host"`
	Port               int    `json:"port"`
	Scheme             string `json:"scheme"`
	User               string `json:"user"`
	Password           string `json:"password"`
	AccessClientID     string `json:"access_client_id,omitempty"`
	AccessClientSecret string `json:"access_client_secret,omitempty"`
}

// HyperdriveUpdateParams holds the parameters for updating a Hyperdrive config.
type HyperdriveUpdateParams struct {
	Name    string                 `json:"name,omitempty"`
	Origin  HyperdriveOriginConfig `json:"origin"`
	Caching *HyperdriveCaching     `json:"caching,omitempty"`
}

// HyperdriveService implements Cloudflare Hyperdrive operations.
type HyperdriveService struct {
	cf        *cloudflare.API
	accountID string
}

// NewHyperdriveService creates a new Hyperdrive service client.
func NewHyperdriveService(api *cloudflare.API, accountID string) (*HyperdriveService, error) {
	if api == nil {
		return nil, validationError("NewHyperdriveService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewHyperdriveService", "account ID is required")
	}
	return &HyperdriveService{cf: api, accountID: accountID}, nil
}

// NewHyperdriveServiceFromCreds creates a HyperdriveService from account ID and API token.
func NewHyperdriveServiceFromCreds(accountID, apiToken string) (*HyperdriveService, error) {
	if accountID == "" {
		return nil, validationError("NewHyperdriveService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewHyperdriveService", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewHyperdriveService", "failed to create Cloudflare API client", err)
	}
	return &HyperdriveService{cf: cf, accountID: accountID}, nil
}

// List returns all Hyperdrive configs in the account.
func (s *HyperdriveService) List(ctx context.Context) ([]*HyperdriveConfig, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	results, err := s.cf.ListHyperdriveConfigs(ctx, rc, cloudflare.ListHyperdriveConfigParams{})
	if err != nil {
		return nil, newError("HyperdriveService.List", "failed to list Hyperdrive configs", err)
	}

	configs := make([]*HyperdriveConfig, 0, len(results))
	for _, c := range results {
		configs = append(configs, mapHyperdriveConfig(c))
	}
	return configs, nil
}

// Get retrieves a single Hyperdrive config by ID.
func (s *HyperdriveService) Get(ctx context.Context, configID string) (*HyperdriveConfig, error) {
	if configID == "" {
		return nil, validationError("HyperdriveService.Get", "config ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.GetHyperdriveConfig(ctx, rc, configID)
	if err != nil {
		return nil, newError("HyperdriveService.Get", fmt.Sprintf("failed to get Hyperdrive config %q", configID), err)
	}

	return mapHyperdriveConfig(result), nil
}

// Create creates a new Hyperdrive config.
func (s *HyperdriveService) Create(ctx context.Context, name string, origin HyperdriveOriginConfig) (*HyperdriveConfig, error) {
	if name == "" {
		return nil, validationError("HyperdriveService.Create", "config name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	params := cloudflare.CreateHyperdriveConfigParams{
		Name: name,
		Origin: cloudflare.HyperdriveConfigOriginWithSecrets{
			HyperdriveConfigOrigin: cloudflare.HyperdriveConfigOrigin{
				Database:       origin.Database,
				Host:           origin.Host,
				Port:           origin.Port,
				Scheme:         origin.Scheme,
				User:           origin.User,
				AccessClientID: origin.AccessClientID,
			},
			Password:           origin.Password,
			AccessClientSecret: origin.AccessClientSecret,
		},
	}

	result, err := s.cf.CreateHyperdriveConfig(ctx, rc, params)
	if err != nil {
		return nil, newError("HyperdriveService.Create", fmt.Sprintf("failed to create Hyperdrive config %q", name), err)
	}

	return mapHyperdriveConfig(result), nil
}

// Update updates an existing Hyperdrive config.
func (s *HyperdriveService) Update(ctx context.Context, configID string, params HyperdriveUpdateParams) (*HyperdriveConfig, error) {
	if configID == "" {
		return nil, validationError("HyperdriveService.Update", "config ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	cfParams := cloudflare.UpdateHyperdriveConfigParams{
		HyperdriveID: configID,
		Name:         params.Name,
		Origin: cloudflare.HyperdriveConfigOriginWithSecrets{
			HyperdriveConfigOrigin: cloudflare.HyperdriveConfigOrigin{
				Database:       params.Origin.Database,
				Host:           params.Origin.Host,
				Port:           params.Origin.Port,
				Scheme:         params.Origin.Scheme,
				User:           params.Origin.User,
				AccessClientID: params.Origin.AccessClientID,
			},
			Password:           params.Origin.Password,
			AccessClientSecret: params.Origin.AccessClientSecret,
		},
	}
	if params.Caching != nil {
		cfParams.Caching = cloudflare.HyperdriveConfigCaching{
			Disabled:             params.Caching.Disabled,
			MaxAge:               params.Caching.MaxAge,
			StaleWhileRevalidate: params.Caching.StaleWhileRevalidate,
		}
	}

	result, err := s.cf.UpdateHyperdriveConfig(ctx, rc, cfParams)
	if err != nil {
		return nil, newError("HyperdriveService.Update", fmt.Sprintf("failed to update Hyperdrive config %q", configID), err)
	}

	return mapHyperdriveConfig(result), nil
}

// Delete deletes a Hyperdrive config.
func (s *HyperdriveService) Delete(ctx context.Context, configID string) error {
	if configID == "" {
		return validationError("HyperdriveService.Delete", "config ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	err := s.cf.DeleteHyperdriveConfig(ctx, rc, configID)
	if err != nil {
		return newError("HyperdriveService.Delete", fmt.Sprintf("failed to delete Hyperdrive config %q", configID), err)
	}
	return nil
}

// mapHyperdriveConfig converts a cloudflare.HyperdriveConfig to our HyperdriveConfig type.
func mapHyperdriveConfig(c cloudflare.HyperdriveConfig) *HyperdriveConfig {
	return &HyperdriveConfig{
		ID:   c.ID,
		Name: c.Name,
		Origin: HyperdriveOrigin{
			Database:       c.Origin.Database,
			Host:           c.Origin.Host,
			Port:           c.Origin.Port,
			Scheme:         c.Origin.Scheme,
			User:           c.Origin.User,
			AccessClientID: c.Origin.AccessClientID,
		},
		Caching: HyperdriveCaching{
			Disabled:             c.Caching.Disabled,
			MaxAge:               c.Caching.MaxAge,
			StaleWhileRevalidate: c.Caching.StaleWhileRevalidate,
		},
	}
}
