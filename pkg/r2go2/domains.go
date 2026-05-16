package r2go2

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// DomainStatus represents a zone enriched with health indicators.
type DomainStatus struct {
	Zone         *Zone  `json:"zone"`
	DNSStatus    string `json:"dns_status"`    // "ok", "warn", "err"
	SSLStatus    string `json:"ssl_status"`    // "valid", "expiring", "expired", "none"
	HealthStatus string `json:"health_status"` // "up", "down", "unknown"
	RecordCount  int    `json:"record_count"`
	NSStatus     string `json:"ns_status"` // "cloudflare", "external", "mismatch"
}

// DomainDetail provides extended information for a single domain.
type DomainDetail struct {
	DomainStatus
	NameServers []string `json:"name_servers"`
	RecordTypes map[string]int `json:"record_types"` // "A" -> 12, "CNAME" -> 8, etc.
	SSLMode     string   `json:"ssl_mode,omitempty"`
	SSLExpiry   string   `json:"ssl_expiry,omitempty"`
	ResponseTime string  `json:"response_time,omitempty"`
}

// DomainListOptions configures domain listing behavior.
type DomainListOptions struct {
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
	Filter  string `json:"filter,omitempty"`  // "active", "paused"
	Name    string `json:"name,omitempty"`    // substring/glob filter
	Sort    string `json:"sort,omitempty"`    // "name", "status", "records"
}

// Pagination holds pagination metadata for list responses.
type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// DomainService provides domain overview and health enrichment.
type DomainService struct {
	zones  *ZoneService
	ssl    *SSLService
	dns    *DNSService
	doctor *DoctorService
}

// NewDomainService creates a DomainService from its component services.
// The ssl, dns, and doctor parameters are optional — if nil, the corresponding
// health enrichment is skipped.
func NewDomainService(zones *ZoneService, ssl *SSLService, dns *DNSService, doctor *DoctorService) (*DomainService, error) {
	if zones == nil {
		return nil, validationError("NewDomainService", "zone service is required")
	}
	return &DomainService{
		zones:  zones,
		ssl:    ssl,
		dns:    dns,
		doctor: doctor,
	}, nil
}

// List returns domains with health indicators, supporting pagination and filtering.
func (s *DomainService) List(ctx context.Context, opts DomainListOptions) ([]*DomainStatus, *Pagination, error) {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PerPage <= 0 {
		opts.PerPage = 50
	}

	zones, err := s.zones.List(ctx)
	if err != nil {
		return nil, nil, newError("DomainService.List", "failed to list zones", err)
	}

	// Apply filters.
	filtered := s.applyFilters(zones, opts)

	// Apply sorting.
	s.applySorting(filtered, opts.Sort)

	// Paginate.
	total := len(filtered)
	totalPages := (total + opts.PerPage - 1) / opts.PerPage
	start := (opts.Page - 1) * opts.PerPage
	end := start + opts.PerPage
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	page := filtered[start:end]

	// Enrich with health indicators.
	results := make([]*DomainStatus, 0, len(page))
	for _, z := range page {
		ds := &DomainStatus{
			Zone:         z,
			DNSStatus:    "ok",
			SSLStatus:    "none",
			HealthStatus: "unknown",
			NSStatus:     classifyNameservers(z.NameServers),
		}
		results = append(results, ds)
	}

	pagination := &Pagination{
		Page:       opts.Page,
		PerPage:    opts.PerPage,
		Total:      total,
		TotalPages: totalPages,
	}

	return results, pagination, nil
}

// GetDetail retrieves extended information for a single domain.
func (s *DomainService) GetDetail(ctx context.Context, zoneID string) (*DomainDetail, error) {
	if zoneID == "" {
		return nil, validationError("DomainService.GetDetail", "zone ID is required")
	}

	zone, err := s.zones.Get(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	detail := &DomainDetail{
		DomainStatus: DomainStatus{
			Zone:         zone,
			DNSStatus:    "ok",
			SSLStatus:    "none",
			HealthStatus: "unknown",
			NSStatus:     classifyNameservers(zone.NameServers),
		},
		NameServers: zone.NameServers,
		RecordTypes: make(map[string]int),
	}

	// Enrich with DNS record counts if dns service is available.
	if s.dns != nil {
		records, err := s.dns.List(ctx)
		if err == nil {
			detail.RecordCount = len(records)
			for _, r := range records {
				detail.RecordTypes[r.Type]++
			}
		}
	}

	// Enrich with SSL status if ssl service is available.
	if s.ssl != nil {
		sslStatus, err := s.ssl.GetSSL(ctx)
		if err == nil {
			detail.SSLMode = sslStatus.Value
		}
	}

	// Enrich with HTTP probe if doctor service is available.
	if s.doctor != nil {
		httpResult, err := s.doctor.CheckHTTP(ctx, zone.Name)
		if err == nil && httpResult.Error == "" {
			detail.HealthStatus = "up"
			detail.ResponseTime = fmt.Sprintf("%dms", httpResult.ResponseTimeMs)
		} else {
			detail.HealthStatus = "down"
		}

		sslResult, err := s.doctor.CheckSSL(ctx, zone.Name)
		if err == nil && sslResult.Error == "" {
			detail.SSLStatus = classifySSLStatus(sslResult.DaysLeft, sslResult.Valid)
			if !sslResult.NotAfter.IsZero() {
				detail.SSLExpiry = sslResult.NotAfter.Format("2006-01-02")
			}
		}
	}

	return detail, nil
}

// EnrichWithHealth runs lightweight probes on each domain status to fill in
// DNS, SSL, and Health indicators. This is optional and expensive for large lists.
func (s *DomainService) EnrichWithHealth(ctx context.Context, domains []*DomainStatus) {
	if s.doctor == nil {
		return
	}
	for _, ds := range domains {
		if ds.Zone == nil {
			continue
		}

		httpResult, err := s.doctor.CheckHTTP(ctx, ds.Zone.Name)
		if err == nil && httpResult.Error == "" {
			if httpResult.StatusCode >= 200 && httpResult.StatusCode < 400 {
				ds.HealthStatus = "up"
			} else {
				ds.HealthStatus = "down"
			}
		}

		sslResult, err := s.doctor.CheckSSL(ctx, ds.Zone.Name)
		if err == nil && sslResult.Error == "" {
			ds.SSLStatus = classifySSLStatus(sslResult.DaysLeft, sslResult.Valid)
		}
	}
}

func (s *DomainService) applyFilters(zones []*Zone, opts DomainListOptions) []*Zone {
	if opts.Filter == "" && opts.Name == "" {
		return zones
	}

	filtered := make([]*Zone, 0, len(zones))
	for _, z := range zones {
		if opts.Filter != "" {
			if opts.Filter == "active" && z.Status != "active" {
				continue
			}
			if opts.Filter == "paused" && !z.Paused {
				continue
			}
		}
		if opts.Name != "" {
			if !matchesNameFilter(z.Name, opts.Name) {
				continue
			}
		}
		filtered = append(filtered, z)
	}
	return filtered
}

func (s *DomainService) applySorting(zones []*Zone, sortBy string) {
	switch sortBy {
	case "status":
		sort.Slice(zones, func(i, j int) bool {
			return zones[i].Status < zones[j].Status
		})
	case "name", "":
		sort.Slice(zones, func(i, j int) bool {
			return zones[i].Name < zones[j].Name
		})
	}
}

// matchesNameFilter checks if a domain name matches a filter pattern.
// Supports simple glob patterns (*.com) and substring matching.
func matchesNameFilter(name, pattern string) bool {
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(name, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(name, pattern[:len(pattern)-1])
	}
	return strings.Contains(name, pattern)
}

func classifyNameservers(ns []string) string {
	if len(ns) == 0 {
		return "external"
	}
	for _, n := range ns {
		if strings.Contains(strings.ToLower(n), "cloudflare") {
			return "cloudflare"
		}
	}
	return "external"
}

func classifySSLStatus(daysLeft int, valid bool) string {
	if !valid {
		return "expired"
	}
	if daysLeft < 30 {
		return "expiring"
	}
	return "valid"
}

// FormatDomainTable returns a formatted table string for domain status list.
// Useful for both CLI output and agent consumption.
func FormatDomainTable(domains []*DomainStatus) string {
	if len(domains) == 0 {
		return "No domains found"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%-25s %-10s %-6s %-10s %-8s %-8s %-12s\n",
		"DOMAIN", "STATUS", "DNS", "SSL", "HEALTH", "RECORDS", "NS"))

	for _, d := range domains {
		name := d.Zone.Name
		if len(name) > 25 {
			name = name[:22] + "..."
		}
		status := d.Zone.Status
		records := fmt.Sprintf("%d", d.RecordCount)
		if d.Zone.Paused {
			status = "paused"
		}

		b.WriteString(fmt.Sprintf("%-25s %-10s %-6s %-10s %-8s %-8s %-12s\n",
			name, status, d.DNSStatus, d.SSLStatus, d.HealthStatus, records, d.NSStatus))
	}

	return b.String()
}

// FormatDomainDetail returns a formatted detail card for a single domain.
func FormatDomainDetail(d *DomainDetail) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s (%s)\n", d.Zone.Name, d.Zone.Status))

	if len(d.NameServers) > 0 {
		b.WriteString(fmt.Sprintf("  Nameservers: %s\n", strings.Join(d.NameServers, ", ")))
	}

	if len(d.RecordTypes) > 0 {
		var parts []string
		for t, count := range d.RecordTypes {
			parts = append(parts, fmt.Sprintf("%d %s", count, t))
		}
		sort.Strings(parts)
		b.WriteString(fmt.Sprintf("  Records:     %d (%s)\n", d.RecordCount, strings.Join(parts, ", ")))
	}

	if d.SSLMode != "" {
		sslLine := fmt.Sprintf("  SSL:         %s", d.SSLMode)
		if d.SSLExpiry != "" {
			expiry, err := time.Parse("2006-01-02", d.SSLExpiry)
			if err == nil {
				days := int(time.Until(expiry).Hours() / 24)
				sslLine += fmt.Sprintf(" — expires %s (%d days)", d.SSLExpiry, days)
			}
		}
		b.WriteString(sslLine + "\n")
	}

	if d.ResponseTime != "" {
		b.WriteString(fmt.Sprintf("  Health:      %s (%s)\n", d.HealthStatus, d.ResponseTime))
	}

	return b.String()
}
