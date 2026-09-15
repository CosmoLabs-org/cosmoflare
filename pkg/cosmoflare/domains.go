package cosmoflare

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// DomainStatus represents a zone enriched with health indicators.
type DomainStatus struct {
	Zone          *Zone  `json:"zone"`
	DNSStatus     string `json:"dns_status"`    // "ok", "warn", "err"
	SSLStatus     string `json:"ssl_status"`    // "valid", "expiring", "expired", "none"
	HealthStatus  string `json:"health_status"` // "up", "down", "unknown"
	RecordCount   int    `json:"record_count"`
	NSStatus      string `json:"ns_status"`                // "cloudflare", "external", "mismatch"
	RedirectIssue string `json:"redirect_issue,omitempty"` // "" | "loop" | "http-4xx" | "http-5xx" | "unreachable"
}

// DomainDetail provides extended information for a single domain.
type DomainDetail struct {
	DomainStatus
	NameServers  []string       `json:"name_servers"`
	RecordTypes  map[string]int `json:"record_types"` // "A" -> 12, "CNAME" -> 8, etc.
	SSLMode      string         `json:"ssl_mode,omitempty"`
	SSLExpiry    string         `json:"ssl_expiry,omitempty"`
	ResponseTime string         `json:"response_time,omitempty"`
	Redirects    []RedirectRule `json:"redirects,omitempty"`
	Registrar    *RegistrarInfo `json:"registrar,omitempty"`
}

// DomainListOptions configures domain listing behavior.
type DomainListOptions struct {
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
	Filter  string `json:"filter,omitempty"` // "active", "paused"
	Name    string `json:"name,omitempty"`   // substring/glob filter
	Sort    string `json:"sort,omitempty"`   // "name", "status", "records"
}

// Pagination holds pagination metadata for list responses.
type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// PageRuleLister is the consumer-side surface for legacy Page Rules.
type PageRuleLister interface {
	List(ctx context.Context) ([]*PageRule, error)
}

// DomainService provides domain overview and health enrichment.
type DomainService struct {
	zones     *ZoneService
	ssl       *SSLService
	dns       *DNSService
	doctor    *DoctorService
	redirects *RedirectService
	registrar *RegistrarService
	pageRules func(zoneID string) PageRuleLister
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

// WithRedirects wires modern Redirect Rules enrichment into GetDetail. It returns
// the receiver for fluent chaining and leaves the base constructor untouched.
func (s *DomainService) WithRedirects(r *RedirectService) *DomainService {
	s.redirects = r
	return s
}

// WithRegistrar wires registration-overlay enrichment into GetDetail. It returns
// the receiver for fluent chaining and leaves the base constructor untouched.
func (s *DomainService) WithRegistrar(r *RegistrarService) *DomainService {
	s.registrar = r
	return s
}

// WithPageRules wires legacy forwarding-rule enrichment into GetDetail.
// The factory is invoked per zone (PageRuleService is zone-scoped at
// construction); a nil factory, nil lister, or list failure silently skips
// legacy rows — modern rules still show (partial-failure doctrine).
func (s *DomainService) WithPageRules(factory func(zoneID string) PageRuleLister) *DomainService {
	s.pageRules = factory
	return s
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

	filtered := s.applyFilters(zones, opts)
	s.applySorting(filtered, opts.Sort)

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

	s.enrichDNSRecords(ctx, detail)
	s.enrichSSLMode(ctx, detail)
	s.enrichDoctorProbes(ctx, zone.Name, detail)
	s.enrichRedirects(ctx, zoneID, detail)
	s.enrichRegistrar(ctx, zone.Name, detail)

	return detail, nil
}

// enrichDNSRecords fills the record count and per-type breakdown from the
// DNS service. A nil service or a list failure leaves the counts zero.
func (s *DomainService) enrichDNSRecords(ctx context.Context, detail *DomainDetail) {
	if s.dns == nil {
		return
	}
	records, err := s.dns.List(ctx)
	if err == nil {
		detail.RecordCount = len(records)
		for _, r := range records {
			detail.RecordTypes[r.Type]++
		}
	}
}

// enrichSSLMode fills the zone SSL mode from the SSL service. A nil service
// or a fetch failure leaves the field empty.
func (s *DomainService) enrichSSLMode(ctx context.Context, detail *DomainDetail) {
	if s.ssl == nil {
		return
	}
	sslStatus, err := s.ssl.GetSSL(ctx)
	if err == nil {
		detail.SSLMode = sslStatus.Value
	}
}

// enrichDoctorProbes runs the doctor's HTTP and SSL probes against the
// domain to fill health, SSL status, response time, and SSL expiry. A nil
// doctor leaves the defaults from GetDetail in place.
func (s *DomainService) enrichDoctorProbes(ctx context.Context, zoneName string, detail *DomainDetail) {
	if s.doctor == nil {
		return
	}

	httpResult, err := s.doctor.CheckHTTP(ctx, zoneName)
	if err == nil && httpResult.Error == "" {
		detail.HealthStatus = "up"
		detail.ResponseTime = fmt.Sprintf("%dms", httpResult.ResponseTimeMs)
	} else {
		detail.HealthStatus = "down"
	}

	sslResult, err := s.doctor.CheckSSL(ctx, zoneName)
	if err == nil && sslResult.Error == "" {
		detail.SSLStatus = classifySSLStatus(sslResult.DaysLeft, sslResult.Valid)
		if !sslResult.NotAfter.IsZero() {
			detail.SSLExpiry = sslResult.NotAfter.Format("2006-01-02")
		}
	}
}

// enrichRedirects attaches redirect rules to the detail — modern Redirect
// Rules for the zone, then legacy forwarding_url Page Rules merged after
// modern rules so redirect visibility is complete regardless of which
// system created the rule. Failures skip rows silently.
func (s *DomainService) enrichRedirects(ctx context.Context, zoneID string, detail *DomainDetail) {
	// Optional redirect enrichment — modern Redirect Rules for the zone.
	if s.redirects != nil {
		if rules, err := s.redirects.List(ctx, zoneID); err == nil {
			detail.Redirects = rules
		}
	}

	// Optional legacy enrichment — forwarding_url Page Rules.
	if s.pageRules != nil {
		if lister := s.pageRules(zoneID); lister != nil {
			if legacy, err := lister.List(ctx); err == nil {
				detail.Redirects = append(detail.Redirects, legacyForwardingRules(zoneID, legacy)...)
			}
		}
	}
}

// enrichRegistrar attaches the registration overlay keyed by domain name.
// A domain absent from the registrar map is registered elsewhere ("external").
// NOTE: RegistrarInfo.AutoRenew is always false (cloudflare-go v0.116.0 exposes
// no auto-renew flag on the read model) — display layers must not render it as
// "auto-renew off".
func (s *DomainService) enrichRegistrar(ctx context.Context, zoneName string, detail *DomainDetail) {
	if s.registrar == nil {
		return
	}
	if all, err := s.registrar.List(ctx); err == nil {
		if info, ok := all[zoneName]; ok {
			detail.Registrar = &info
		} else {
			detail.Registrar = &RegistrarInfo{Registrar: "external"}
		}
	}
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

// DomainSummary aggregates counts across a list of domains plus an attention list.
type DomainSummary struct {
	Total          int            `json:"total"`
	ByNSStatus     map[string]int `json:"by_ns_status"`
	BySSLStatus    map[string]int `json:"by_ssl_status"`
	ByRegistrar    map[string]int `json:"by_registrar"`
	NeedsAttention int            `json:"needs_attention"`
	Attention      []string       `json:"attention"`
}

// SummarizeDomains computes aggregate stats and the attention list over a set of
// domain statuses. ByRegistrar is intentionally left empty here: DomainStatus
// carries no registrar data (that lives on the enriched DomainDetail), so a
// breakdown would be fabricated. Callers needing it should summarize details.
func SummarizeDomains(ds []*DomainStatus) DomainSummary {
	s := DomainSummary{
		ByNSStatus:  map[string]int{},
		BySSLStatus: map[string]int{},
		ByRegistrar: map[string]int{},
	}
	for _, d := range ds {
		if d == nil {
			continue
		}
		s.Total++
		s.ByNSStatus[d.NSStatus]++
		s.BySSLStatus[d.SSLStatus]++
		if domainNeedsAttention(d) {
			s.NeedsAttention++
			if d.Zone != nil {
				s.Attention = append(s.Attention, d.Zone.Name)
			}
		}
	}
	return s
}

// domainNeedsAttention reports whether a domain has a condition an operator should
// look at: non-Cloudflare or mismatched nameservers, an SSL problem, or a failing
// health check.
func domainNeedsAttention(d *DomainStatus) bool {
	return d.NSStatus == "external" || d.NSStatus == "mismatch" ||
		d.SSLStatus == "expired" || d.SSLStatus == "expiring" || d.SSLStatus == "none" ||
		d.HealthStatus == "down" ||
		d.RedirectIssue != ""
}

// classifyRedirectIssue reduces a domain's probe results to the worst
// redirect issue: loop > http-5xx > http-4xx > unreachable > none.
// Skipped destinations contribute nothing.
func classifyRedirectIssue(results []RedirectProbeResult) string {
	worst := ""
	rank := map[string]int{"": 0, "unreachable": 1, "http-4xx": 2, "http-5xx": 3, "loop": 4}
	for _, r := range results {
		issue := ""
		switch {
		case r.Skipped:
			continue
		case r.Loop:
			issue = "loop"
		case r.Status >= 500:
			issue = "http-5xx"
		case r.Status >= 400:
			issue = "http-4xx"
		case r.Err != "":
			issue = "unreachable"
		}
		if rank[issue] > rank[worst] {
			worst = issue
		}
	}
	return worst
}

// ClassifyRedirectIssues is the exported form of classifyRedirectIssue for
// cmd/TUI consumers.
func ClassifyRedirectIssues(results []RedirectProbeResult) string {
	return classifyRedirectIssue(results)
}

// classifyRegistrarStatus returns "cloudflare" if the domain is present in the
// registrar map as a Cloudflare-registered domain, otherwise "external".
func classifyRegistrarStatus(name string, reg map[string]RegistrarInfo) string {
	if reg != nil {
		if info, ok := reg[name]; ok && info.Registrar == "cloudflare" {
			return "cloudflare"
		}
	}
	return "external"
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
