//go:build disabled

package domain

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// Manager handles custom domain operations for Cloudflare R2
type Manager struct {
	ctx        context.Context
	cf         *cloudflare.API
	accountID  string
	httpClient *http.Client
}

// NewManager creates a new domain manager
func NewManager(cf *cloudflare.API, accountID string) *Manager {
	return &Manager{
		ctx:        context.Background(),
		cf:         cf,
		accountID:  accountID,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Domain represents a custom domain configuration
type Domain struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Bucket     string    `json:"bucket"`
	Status     string    `json:"status"`
	SSLStatus  string    `json:"ssl_status"`
	DNSStatus  string    `json:"dns_status"`
	ZoneID     string    `json:"zone_id"`
	ZoneName   string    `json:"zone_name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	RecordID   string    `json:"record_id,omitempty"`
	RecordType string    `json:"record_type"`
	RecordName string    `json:"record_name"`
	RecordValue string   `json:"record_value"`
}

// SSLMode represents SSL configuration options
type SSLMode string

const (
	SSLModeOff     SSLMode = "off"
	SSLModeFlexible SSLMode = "flexible"
	SSLModeFull    SSLMode = "full"
	SSLModeStrict  SSLMode = "strict"
)

// AttachRequest represents a domain attachment request
type AttachRequest struct {
	Domain   string  `json:"domain"`
	Bucket   string  `json:"bucket"`
	SSLMode  SSLMode `json:"ssl_mode"`
	ZoneID   string  `json:"zone_id,omitempty"`
	Force    bool    `json:"force"`
}

// AttachResponse represents the result of domain attachment
type AttachResponse struct {
	Domain     *Domain             `json:"domain"`
	DNSRecord  *cloudflare.DNSRecord `json:"dns_record"`
	SSLCert    *cloudflare.CustomSSL  `json:"ssl_cert,omitempty"`
	PurgeToken string              `json:"purge_token,omitempty"`
}

// ListOptions for domain listing
type ListOptions struct {
	Bucket string `json:"bucket,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// DetachRequest represents a domain detachment request
type DetachRequest struct {
	Domain string `json:"domain"`
	Force  bool   `json:"force"`
}

// PurgeRequest represents a cache purge request
type PurgeRequest struct {
	Domain     string   `json:"domain"`
	Path       string   `json:"path,omitempty"`
	Everything bool     `json:"everything"`
	Tags       []string `json:"tags,omitempty"`
}

// VerificationResult represents domain verification results
type VerificationResult struct {
	Domain      string           `json:"domain"`
	Checks      []VerificationCheck `json:"checks"`
	OverallStatus string         `json:"overall_status"`
	Details     string           `json:"details"`
	Score       int              `json:"score"`
}

// VerificationCheck represents a single verification check
type VerificationCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "pass", "fail", "warn"
	Info   string `json:"info"`
	Error  string `json:"error,omitempty"`
}

// Attach attaches a custom domain to an R2 bucket
func (m *Manager) Attach(req *AttachRequest) (*AttachResponse, error) {
	// Validate input
	if req.Domain == "" {
		return nil, fmt.Errorf("domain name is required")
	}
	if req.Bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	// Get or create zone
	zoneID := req.ZoneID
	var zone *cloudflare.Zone
	var err error

	if zoneID == "" {
		// Find zone by domain
		zone, err = m.findZoneByDomain(req.Domain)
		if err != nil {
			return nil, fmt.Errorf("failed to find zone for domain %s: %w", req.Domain, err)
		}
		zoneID = zone.ID
	} else {
		// Get zone by ID
		zone, err = m.cf.ZoneDetails(m.ctx, zoneID)
		if err != nil {
			return nil, fmt.Errorf("failed to get zone %s: %w", zoneID, err)
		}
	}

	// Check if domain already exists
	existingDomain, err := m.getDomainByName(req.Domain)
	if err == nil && existingDomain != nil && !req.Force {
		return nil, fmt.Errorf("domain %s is already attached to bucket %s", req.Domain, existingDomain.Bucket)
	}

	// Create R2 bucket custom hostname
	hostname := cloudflare.CreateR2BucketCustomHostnameParams{
		AccountID: m.accountID,
		BucketName: req.Bucket,
		Hostname:  req.Domain,
	}

	customHostname, err := m.cf.CreateR2BucketCustomHostname(m.ctx, hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to create R2 custom hostname: %w", err)
	}

	// Create DNS record
	r2Endpoint := fmt.Sprintf("bucket.%s.r2.cloudflarestorage.com", m.accountID)
	dnsRecord := cloudflare.DNSRecord{
		Type:    "CNAME",
		Name:    req.Domain,
		Content: r2Endpoint,
		TTL:     300, // 5 minutes
		Proxied: cloudflare.BoolPtr(true),
	}

	record, err := m.cf.CreateDNSRecord(m.ctx, cloudflare.ZoneIdentifier(zoneID), dnsRecord)
	if err != nil {
		// Rollback custom hostname creation
		m.cf.DeleteR2BucketCustomHostname(m.ctx, cloudflare.DeleteR2BucketCustomHostnameParams{
			AccountID: m.accountID,
			BucketName: req.Bucket,
			Hostname:  req.Domain,
		})
		return nil, fmt.Errorf("failed to create DNS record: %w", err)
	}

	// Configure SSL if requested
	var sslCert *cloudflare.CustomSSL
	if req.SSLMode != SSLModeOff {
		sslConfig := cloudflare.CustomSSLOptions{
			BundleMethod: "ubiquitous",
			Type:         string(req.SSLMode),
		}
		sslCert, err = m.cf.CreateCustomSSL(m.ctx, zoneID, sslConfig)
		if err != nil {
			// Log warning but don't fail
			fmt.Printf("Warning: Failed to configure SSL: %v\n", err)
		}
	}

	// Wait for DNS propagation
	domain := &Domain{
		ID:          customHostname.Hostname,
		Name:        req.Domain,
		Bucket:      req.Bucket,
		Status:      "pending",
		SSLStatus:   "pending",
		DNSStatus:   "configuring",
		ZoneID:      zoneID,
		ZoneName:    zone.Name,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		RecordID:    record.ID,
		RecordType:  record.Type,
		RecordName:  record.Name,
		RecordValue: record.Content,
	}

	if sslCert != nil {
		domain.SSLStatus = "active"
	}

	// Verify DNS configuration
	if err := m.verifyDNSRecord(zoneID, req.Domain, r2Endpoint); err == nil {
		domain.DNSStatus = "active"
		domain.Status = "active"
	}

	// Purge cache
	purgeToken, err := m.purgeDomainCache(req.Domain)
	if err != nil {
		fmt.Printf("Warning: Failed to purge cache: %v\n", err)
	}

	return &AttachResponse{
		Domain:     domain,
		DNSRecord:  &record,
		SSLCert:    sslCert,
		PurgeToken: purgeToken,
	}, nil
}

// List lists all custom domains
func (m *Manager) List(opts *ListOptions) ([]*Domain, error) {
	var result []*Domain

	// Get all custom hostnames from R2
	hostnames, err := m.cf.ListR2BucketCustomHostnames(m.ctx, cloudflare.ListR2BucketCustomHostnameParams{
		AccountID: m.accountID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list R2 custom hostnames: %w", err)
	}

	// Convert to domain objects and add DNS record info
	for _, hostname := range hostnames {
		// Get DNS record for this domain
		zone, err := m.findZoneByDomain(hostname.Hostname)
		if err != nil {
			continue // Skip if zone not found
		}

		records, _, err := m.cf.ListDNSRecords(m.ctx, cloudflare.ZoneIdentifier(zone.ID), cloudflare.ListDNSRecordsParams{
			Type: "CNAME",
			Name: hostname.Hostname,
		})
		if err != nil {
			continue
		}

		var record *cloudflare.DNSRecord
		if len(records) > 0 {
			record = &records[0]
		}

		domain := &Domain{
			ID:          hostname.Hostname,
			Name:        hostname.Hostname,
			Bucket:      hostname.BucketName,
			Status:      "active", // R2 returns only active hostnames
			SSLStatus:   "active",
			DNSStatus:   "active",
			ZoneID:      zone.ID,
			ZoneName:    zone.Name,
			CreatedAt:   time.Now().UTC(), // R2 API doesn't provide creation time
			UpdatedAt:   time.Now().UTC(),
		}

		if record != nil {
			domain.RecordID = record.ID
			domain.RecordType = record.Type
			domain.RecordName = record.Name
			domain.RecordValue = record.Content
		}

		// Apply filters
		if opts != nil {
			if opts.Bucket != "" && domain.Bucket != opts.Bucket {
				continue
			}
			if opts.Status != "" && domain.Status != opts.Status {
				continue
			}
		}

		result = append(result, domain)
	}

	// Apply limit
	if opts != nil && opts.Limit > 0 && len(result) > opts.Limit {
		result = result[:opts.Limit]
	}

	return result, nil
}

// Get retrieves a specific domain
func (m *Manager) Get(domainName string) (*Domain, error) {
	domains, err := m.List(&ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, domain := range domains {
		if domain.Name == domainName {
			return domain, nil
		}
	}

	return nil, fmt.Errorf("domain %s not found", domainName)
}

// Detach detaches a custom domain from an R2 bucket
func (m *Manager) Detach(req *DetachRequest) error {
	if req.Domain == "" {
		return fmt.Errorf("domain name is required")
	}

	// Get domain details
	domain, err := m.Get(req.Domain)
	if err != nil {
		return fmt.Errorf("failed to get domain details: %w", err)
	}

	// Delete DNS record
	if domain.RecordID != "" {
		err = m.cf.DeleteDNSRecord(m.ctx, cloudflare.ZoneIdentifier(domain.ZoneID), domain.RecordID)
		if err != nil {
			fmt.Printf("Warning: Failed to delete DNS record: %v\n", err)
		}
	}

	// Delete R2 custom hostname
	err = m.cf.DeleteR2BucketCustomHostname(m.ctx, cloudflare.DeleteR2BucketCustomHostnameParams{
		AccountID: m.accountID,
		BucketName: domain.Bucket,
		Hostname:  req.Domain,
	})
	if err != nil {
		return fmt.Errorf("failed to delete R2 custom hostname: %w", err)
	}

	// Purge cache
	_, err = m.purgeDomainCache(req.Domain)
	if err != nil {
		fmt.Printf("Warning: Failed to purge cache: %v\n", err)
	}

	return nil
}

// Verify verifies the configuration of a custom domain
func (m *Manager) Verify(domainName string) (*VerificationResult, error) {
	if domainName == "" {
		return nil, fmt.Errorf("domain name is required")
	}

	result := &VerificationResult{
		Domain: domainName,
		Checks: []VerificationCheck{},
		Score:  0,
	}

	domain, err := m.Get(domainName)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain details: %w", err)
	}

	// Check 1: DNS Configuration
	r2Endpoint := fmt.Sprintf("bucket.%s.r2.cloudflarestorage.com", m.accountID)
	if err := m.verifyDNSRecord(domain.ZoneID, domainName, r2Endpoint); err == nil {
		result.Checks = append(result.Checks, VerificationCheck{
			Name:   "DNS Configuration",
			Status: "pass",
			Info:   "CNAME record correctly configured",
		})
		result.Score += 25
	} else {
		result.Checks = append(result.Checks, VerificationCheck{
			Name:   "DNS Configuration",
			Status: "fail",
			Info:   "CNAME record not found or incorrect",
			Error:  err.Error(),
		})
	}

	// Check 2: R2 Bucket Access
	if err := m.verifyR2Access(domain.Bucket); err == nil {
		result.Checks = append(result.Checks, VerificationCheck{
			Name:   "R2 Bucket Access",
			Status: "pass",
			Info:   "R2 bucket is accessible",
		})
		result.Score += 25
	} else {
		result.Checks = append(result.Checks, VerificationCheck{
			Name:   "R2 Bucket Access",
			Status: "fail",
			Info:   "Cannot access R2 bucket",
			Error:  err.Error(),
		})
	}

	// Check 3: HTTPS Connectivity
	if err := m.verifyHTTPSConnectivity(domainName); err == nil {
		result.Checks = append(result.Checks, VerificationCheck{
			Name:   "HTTPS Connectivity",
			Status: "pass",
			Info:   "Domain responds to HTTPS requests",
		})
		result.Score += 25
	} else {
		result.Checks = append(result.Checks, VerificationCheck{
			Name:   "HTTPS Connectivity",
			Status: "fail",
			Info:   "Domain does not respond to HTTPS",
			Error:  err.Error(),
		})
	}

	// Check 4: SSL Certificate
	if err := m.verifySSLCertificate(domain.ZoneID, domainName); err == nil {
		result.Checks = append(result.Checks, VerificationCheck{
			Name:   "SSL Certificate",
			Status: "pass",
			Info:   "Valid SSL certificate installed",
		})
		result.Score += 25
	} else {
		result.Checks = append(result.Checks, VerificationCheck{
			Name:   "SSL Certificate",
			Status: "warn",
			Info:   "SSL certificate may still be provisioning",
			Error:  err.Error(),
		})
	}

	// Determine overall status
	if result.Score >= 75 {
		result.OverallStatus = "pass"
		result.Details = "Domain is properly configured and ready for use"
	} else if result.Score >= 50 {
		result.OverallStatus = "warn"
		result.Details = "Domain configuration is in progress, some checks failed"
	} else {
		result.OverallStatus = "fail"
		result.Details = "Domain configuration has serious issues"
	}

	return result, nil
}

// Purge purges the CDN cache for a domain
func (m *Manager) Purge(req *PurgeRequest) error {
	if req.Domain == "" {
		return fmt.Errorf("domain name is required")
	}

	// Build purge request
	var files []string
	var hosts []string = []string{req.Domain}

	if req.Everything {
		// Purge everything for this host
		purgeReq := cloudflare.PurgeCacheRequest{
			Everything: true,
			Hosts:      hosts,
		}

		_, err := m.cf.PurgeCache(m.ctx, cloudflare.ZoneIdentifier(req.Domain), purgeReq)
		return err
	} else if req.Path != "" {
		// Purge specific path
		if !strings.HasPrefix(req.Path, "/") {
			req.Path = "/" + req.Path
		}
		files = []string{"https://" + req.Domain + req.Path}

		purgeReq := cloudflare.PurgeCacheRequest{
			Files: files,
		}

		_, err := m.cf.PurgeCache(m.ctx, cloudflare.ZoneIdentifier(req.Domain), purgeReq)
		return err
	} else if len(req.Tags) > 0 {
		// Purge by tags
		purgeReq := cloudflare.PurgeCacheRequest{
			Tags: req.Tags,
			Hosts: hosts,
		}

		_, err := m.cf.PurgeCache(m.ctx, cloudflare.ZoneIdentifier(req.Domain), purgeReq)
		return err
	}

	return fmt.Errorf("must specify one of: --everything, --path, or --tags")
}

// Helper functions

func (m *Manager) findZoneByDomain(domain string) (*cloudflare.Zone, error) {
	// Extract zone name (remove subdomains)
	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid domain format")
	}

	zoneName := strings.Join(parts[len(parts)-2:], ".")
	zones, err := m.cf.ListZones(m.ctx, cloudflare.WithZoneFilters(zoneName, ""))
	if err != nil {
		return nil, err
	}

	if len(zones) == 0 {
		return nil, fmt.Errorf("zone %s not found", zoneName)
	}

	return &zones[0], nil
}

func (m *Manager) getDomainByName(domainName string) (*Domain, error) {
	domains, err := m.List(nil)
	if err != nil {
		return nil, err
	}

	for _, domain := range domains {
		if domain.Name == domainName {
			return domain, nil
		}
	}

	return nil, fmt.Errorf("domain %s not found", domainName)
}

func (m *Manager) verifyDNSRecord(zoneID, domain, expectedValue string) error {
	records, _, err := m.cf.ListDNSRecords(m.ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{
		Type: "CNAME",
		Name: domain,
	})
	if err != nil {
		return err
	}

	for _, record := range records {
		if record.Content == expectedValue {
			return nil
		}
	}

	return fmt.Errorf("DNS record not found or incorrect")
}

func (m *Manager) verifyR2Access(bucketName string) error {
	// This would use the S3 client to verify bucket access
	// For now, we'll just check if the bucket exists
	_, err := m.cf.GetR2Bucket(m.ctx, cloudflare.GetR2BucketParams{
		AccountID: m.accountID,
		BucketName: bucketName,
	})
	return err
}

func (m *Manager) verifyHTTPSConnectivity(domain string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://" + domain)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (m *Manager) verifySSLCertificate(zoneID, domain string) error {
	// Get SSL settings for the zone
	ssl, err := m.cf.CustomSSL(m.ctx, zoneID)
	if err != nil {
		return err
	}

	for _, cert := range ssl {
		if cert.Hosts == domain || strings.Contains(cert.Hosts, domain) {
			return nil
		}
	}

	return fmt.Errorf("SSL certificate not found for domain %s", domain)
}

func (m *Manager) purgeDomainCache(domain string) (string, error) {
	purgeReq := cloudflare.PurgeCacheRequest{
		Everything: true,
		Hosts:      []string{domain},
	}

	// Find zone for domain
	zone, err := m.findZoneByDomain(domain)
	if err != nil {
		return "", err
	}

	purgeResp, err := m.cf.PurgeCache(m.ctx, cloudflare.ZoneIdentifier(zone.ID), purgeReq)
	if err != nil {
		return "", err
	}

	return purgeResp.ID, nil
}