package cosmoflare

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// DNSRecord represents a Cloudflare DNS record.
type DNSRecord struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Name       string    `json:"name"`
	Content    string    `json:"content"`
	TTL        int       `json:"ttl"`
	Proxied    bool      `json:"proxied"`
	Priority   *uint16   `json:"priority,omitempty"`
	Comment    string    `json:"comment,omitempty"`
	ZoneID     string    `json:"zone_id"`
	Proxiable  bool      `json:"proxiable"`
	CreatedOn  time.Time `json:"created_on"`
	ModifiedOn time.Time `json:"modified_on"`
}

// DNSOption is a functional option for DNS record operations.
type DNSOption func(*dnsConfig)

type dnsConfig struct {
	ttl      int
	proxied  *bool
	priority *uint16
	comment  *string
}

// WithDNSProxied sets the proxied status for a DNS record.
func WithDNSProxied(proxied bool) DNSOption {
	return func(c *dnsConfig) { c.proxied = boolPtr(proxied) }
}

// WithDNSTTL sets the TTL for a DNS record (1 = automatic).
func WithDNSTTL(ttl int) DNSOption {
	return func(c *dnsConfig) { c.ttl = ttl }
}

// WithDNSPriority sets the priority for MX/SRV records.
func WithDNSPriority(priority uint16) DNSOption {
	return func(c *dnsConfig) { c.priority = &priority }
}

// WithDNSComment sets a comment on a DNS record.
func WithDNSComment(comment string) DNSOption {
	return func(c *dnsConfig) { c.comment = &comment }
}

// DNSListOption is a functional option for DNS record list operations.
type DNSListOption func(*dnsListConfig)

type dnsListConfig struct {
	recordType string
	name       string
	content    string
}

// WithDNSType filters listed records by type (A, AAAA, CNAME, MX, etc.).
func WithDNSType(t string) DNSListOption {
	return func(c *dnsListConfig) { c.recordType = t }
}

// WithDNSName filters listed records by name.
func WithDNSName(name string) DNSListOption {
	return func(c *dnsListConfig) { c.name = name }
}

// WithDNSContent filters listed records by content.
func WithDNSContent(content string) DNSListOption {
	return func(c *dnsListConfig) { c.content = content }
}

// DNSService implements DNS record operations.
type DNSService struct {
	cf     *cloudflare.API
	zoneID string
}

// NewDNSService creates a new DNS service client.
func NewDNSService(api *cloudflare.API, zoneID string) (*DNSService, error) {
	if api == nil {
		return nil, validationError("NewDNSService", "cloudflare API client is required")
	}
	if zoneID == "" {
		return nil, validationError("NewDNSService", "zone ID is required")
	}
	return &DNSService{cf: api, zoneID: zoneID}, nil
}

// NewDNSServiceFromCreds creates a DNSService from zone ID and API token.
// Convenience helper for CLI usage.
func NewDNSServiceFromCreds(zoneID, apiToken string) (*DNSService, error) {
	if zoneID == "" {
		return nil, validationError("NewDNSService", "zone ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewDNSService", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewDNSService", "failed to create Cloudflare API client", err)
	}
	return &DNSService{cf: cf, zoneID: zoneID}, nil
}

// Create creates a new DNS record.
func (s *DNSService) Create(ctx context.Context, recordType, name, content string, opts ...DNSOption) (*DNSRecord, error) {
	if recordType == "" {
		return nil, validationError("DNSService.Create", "record type is required")
	}
	if name == "" {
		return nil, validationError("DNSService.Create", "record name is required")
	}
	if content == "" {
		return nil, validationError("DNSService.Create", "record content is required")
	}

	cfg := &dnsConfig{}
	for _, o := range opts {
		o(cfg)
	}

	var commentStr string
	if cfg.comment != nil {
		commentStr = *cfg.comment
	}

	params := cloudflare.CreateDNSRecordParams{
		Type:     recordType,
		Name:     name,
		Content:  content,
		TTL:      cfg.ttl,
		Proxied:  cfg.proxied,
		Priority: cfg.priority,
		Comment:  commentStr,
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	resp, err := s.cf.CreateDNSRecord(ctx, rc, params)
	if err != nil {
		return nil, newError("DNSService.Create", fmt.Sprintf("failed to create DNS record %s %s", recordType, name), err)
	}

	return cfDNSToRecord(resp, s.zoneID), nil
}

// List returns DNS records in the zone with optional filtering.
func (s *DNSService) List(ctx context.Context, opts ...DNSListOption) ([]*DNSRecord, error) {
	cfg := &dnsListConfig{}
	for _, o := range opts {
		o(cfg)
	}

	params := cloudflare.ListDNSRecordsParams{
		Type:    cfg.recordType,
		Name:    cfg.name,
		Content: cfg.content,
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	results, _, err := s.cf.ListDNSRecords(ctx, rc, params)
	if err != nil {
		return nil, newError("DNSService.List", "failed to list DNS records", err)
	}

	records := make([]*DNSRecord, 0, len(results))
	for _, r := range results {
		records = append(records, cfDNSToRecord(r, s.zoneID))
	}
	return records, nil
}

// Get retrieves a single DNS record by ID.
func (s *DNSService) Get(ctx context.Context, recordID string) (*DNSRecord, error) {
	if recordID == "" {
		return nil, validationError("DNSService.Get", "record ID is required")
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	resp, err := s.cf.GetDNSRecord(ctx, rc, recordID)
	if err != nil {
		return nil, notFound("DNSService.Get", "", recordID, err)
	}

	return cfDNSToRecord(resp, s.zoneID), nil
}

// Update modifies an existing DNS record.
func (s *DNSService) Update(ctx context.Context, recordID string, opts ...DNSOption) (*DNSRecord, error) {
	if recordID == "" {
		return nil, validationError("DNSService.Update", "record ID is required")
	}

	cfg := &dnsConfig{}
	for _, o := range opts {
		o(cfg)
	}

	params := cloudflare.UpdateDNSRecordParams{
		ID:       recordID,
		TTL:      cfg.ttl,
		Proxied:  cfg.proxied,
		Priority: cfg.priority,
		Comment:  cfg.comment,
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	resp, err := s.cf.UpdateDNSRecord(ctx, rc, params)
	if err != nil {
		return nil, newError("DNSService.Update", fmt.Sprintf("failed to update DNS record %q", recordID), err)
	}

	return cfDNSToRecord(resp, s.zoneID), nil
}

// Delete removes a DNS record.
func (s *DNSService) Delete(ctx context.Context, recordID string) error {
	if recordID == "" {
		return validationError("DNSService.Delete", "record ID is required")
	}

	rc := cloudflare.ZoneIdentifier(s.zoneID)
	err := s.cf.DeleteDNSRecord(ctx, rc, recordID)
	if err != nil {
		return newError("DNSService.Delete", fmt.Sprintf("failed to delete DNS record %q", recordID), err)
	}
	return nil
}

// cfDNSToRecord maps a cloudflare.DNSRecord to our DNSRecord type.
func cfDNSToRecord(r cloudflare.DNSRecord, zoneID string) *DNSRecord {
	return &DNSRecord{
		ID:         r.ID,
		Type:       r.Type,
		Name:       r.Name,
		Content:    r.Content,
		TTL:        r.TTL,
		Proxied:    boolVal(r.Proxied),
		Priority:   r.Priority,
		Comment:    r.Comment,
		ZoneID:     zoneID,
		Proxiable:  r.Proxiable,
		CreatedOn:  r.CreatedOn,
		ModifiedOn: r.ModifiedOn,
	}
}

func boolVal(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func boolPtr(b bool) *bool { return &b }
