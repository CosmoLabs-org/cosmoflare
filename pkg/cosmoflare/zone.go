package cosmoflare

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// Zone represents a Cloudflare DNS zone.
type Zone struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Type        string    `json:"type"`
	Paused      bool      `json:"paused"`
	NameServers []string  `json:"name_servers"`
	OriginalNS  []string  `json:"original_name_servers,omitempty"`
	Plan        ZonePlan  `json:"plan"`
	CreatedOn   time.Time `json:"created_on"`
	ModifiedOn  time.Time `json:"modified_on"`
}

// ZonePlan contains the plan information for a zone.
type ZonePlan struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ZoneSetting contains a single zone setting entry.
type ZoneSetting struct {
	ID         string      `json:"id"`
	Value      interface{} `json:"value"`
	Editable   bool        `json:"editable"`
	ModifiedOn string      `json:"modified_on,omitempty"`
}

// ZoneService implements zone management operations.
type ZoneService struct {
	cf        *cloudflare.API
	accountID string
}

// NewZoneService creates a new Zone service client.
func NewZoneService(api *cloudflare.API, accountID string) (*ZoneService, error) {
	if api == nil {
		return nil, validationError("NewZoneService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewZoneService", "account ID is required")
	}
	return &ZoneService{cf: api, accountID: accountID}, nil
}

// NewZoneServiceFromCreds creates a ZoneService from account ID and API token.
// Convenience helper for CLI usage.
func NewZoneServiceFromCreds(accountID, apiToken string) (*ZoneService, error) {
	if accountID == "" {
		return nil, validationError("NewZoneService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewZoneService", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewZoneService", "failed to create Cloudflare API client", err)
	}
	return &ZoneService{cf: cf, accountID: accountID}, nil
}

// Create adds a new zone to the account.
// zoneType must be "full" or "partial".
func (s *ZoneService) Create(ctx context.Context, name string, zoneType string) (*Zone, error) {
	if name == "" {
		return nil, validationError("ZoneService.Create", "zone name is required")
	}
	if zoneType == "" {
		zoneType = "full"
	}

	z, err := s.cf.CreateZone(ctx, name, false, cloudflare.Account{ID: s.accountID}, zoneType)
	if err != nil {
		return nil, newError("ZoneService.Create", fmt.Sprintf("failed to create zone %q", name), err)
	}

	return cfZoneToZone(z), nil
}

// List returns all zones accessible to the account.
func (s *ZoneService) List(ctx context.Context) ([]*Zone, error) {
	results, err := s.cf.ListZones(ctx)
	if err != nil {
		return nil, newError("ZoneService.List", "failed to list zones", err)
	}

	zones := make([]*Zone, 0, len(results))
	for _, z := range results {
		zones = append(zones, cfZoneToZone(z))
	}
	return zones, nil
}

// Get retrieves a single zone by ID.
func (s *ZoneService) Get(ctx context.Context, zoneID string) (*Zone, error) {
	if zoneID == "" {
		return nil, validationError("ZoneService.Get", "zone ID is required")
	}

	z, err := s.cf.ZoneDetails(ctx, zoneID)
	if err != nil {
		return nil, notFound("ZoneService.Get", "", zoneID, err)
	}

	return cfZoneToZone(z), nil
}

// Delete removes a zone by ID.
func (s *ZoneService) Delete(ctx context.Context, zoneID string) error {
	if zoneID == "" {
		return validationError("ZoneService.Delete", "zone ID is required")
	}

	_, err := s.cf.DeleteZone(ctx, zoneID)
	if err != nil {
		return newError("ZoneService.Delete", fmt.Sprintf("failed to delete zone %q", zoneID), err)
	}
	return nil
}

// GetSettings retrieves all settings for a zone.
func (s *ZoneService) GetSettings(ctx context.Context, zoneID string) ([]*ZoneSetting, error) {
	if zoneID == "" {
		return nil, validationError("ZoneService.GetSettings", "zone ID is required")
	}

	resp, err := s.cf.ZoneSettings(ctx, zoneID)
	if err != nil {
		return nil, newError("ZoneService.GetSettings", fmt.Sprintf("failed to get settings for zone %q", zoneID), err)
	}

	settings := make([]*ZoneSetting, 0, len(resp.Result))
	for _, s := range resp.Result {
		settings = append(settings, &ZoneSetting{
			ID:         s.ID,
			Value:      s.Value,
			Editable:   s.Editable,
			ModifiedOn: s.ModifiedOn,
		})
	}
	return settings, nil
}

// cfZoneToZone converts a cloudflare.Zone to our Zone type.
func cfZoneToZone(z cloudflare.Zone) *Zone {
	return &Zone{
		ID:          z.ID,
		Name:        z.Name,
		Status:      z.Status,
		Type:        z.Type,
		Paused:      z.Paused,
		NameServers: z.NameServers,
		OriginalNS:  z.OriginalNS,
		Plan:        ZonePlan{ID: z.Plan.ID, Name: z.Plan.Name},
		CreatedOn:   z.CreatedOn,
		ModifiedOn:  z.ModifiedOn,
	}
}
