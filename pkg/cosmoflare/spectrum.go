package cosmoflare

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// SpectrumAppInfo is the public view of a zone-scoped Spectrum
// application. Timestamps are rendered as RFC 3339 strings, empty when the
// API reports none. Spectrum proxies arbitrary TCP/UDP/HTTP traffic
// through Cloudflare's edge.
//
// Spectrum permission pending FEAT-011 dataset.
type SpectrumAppInfo struct {
	ID               string   `json:"id"`
	Protocol         string   `json:"protocol"`
	DNSName          string   `json:"dns_name"`
	DNSType          string   `json:"dns_type,omitempty"`
	TrafficType      string   `json:"traffic_type,omitempty"`
	TLS              string   `json:"tls,omitempty"`
	ProxyProtocol    string   `json:"proxy_protocol,omitempty"`
	OriginDirect     []string `json:"origin_direct,omitempty"`
	OriginDNSName    string   `json:"origin_dns_name,omitempty"`
	OriginPort       string   `json:"origin_port,omitempty"`
	ArgoSmartRouting bool     `json:"argo_smart_routing,omitempty"`
	IPv4             bool     `json:"ipv4,omitempty"`
	IPFirewall       bool     `json:"ip_firewall,omitempty"`
	CreatedOn        string   `json:"created_on,omitempty"`
	ModifiedOn       string   `json:"modified_on,omitempty"`
}

// SpectrumAppCreate carries the fields for application creation. Name,
// Protocol and OriginPort are required; OriginDirect or OriginDNSName
// supplies the origin address (at least one is required). DNSType defaults
// to "CNAME", TrafficType to "tcp", TLS to "" (API default: full).
type SpectrumAppCreate struct {
	Name             string
	Protocol         string
	OriginPort       string
	OriginDirect     []string
	OriginDNSName    string
	DNSType          string
	TrafficType      string
	TLS              string
	ProxyProtocol    string
	ArgoSmartRouting bool
	IPv4             bool
	IPFirewall       bool
}

// SpectrumAppUpdate carries optional changes to an existing application.
// nil fields are left untouched; the service merges the patch onto the
// app's current state before the PUT (Spectrum update is a full replace).
type SpectrumAppUpdate struct {
	Protocol         *string
	OriginDirect     *[]string
	OriginDNSName    *string
	OriginPort       *string
	DNSType          *string
	TrafficType      *string
	TLS              *string
	ProxyProtocol    *string
	ArgoSmartRouting *bool
	IPv4             *bool
	IPFirewall       *bool
}

// SpectrumService implements zone-scoped Spectrum application operations on
// top of the typed cloudflare-go resources.
//
// Spectrum permission pending FEAT-011 dataset.
type SpectrumService struct {
	cf     *cloudflare.API
	zoneID string
}

// NewSpectrumService creates a new Spectrum service client.
func NewSpectrumService(api *cloudflare.API, zoneID string) (*SpectrumService, error) {
	if api == nil {
		return nil, validationError("NewSpectrumService", "cloudflare API client is required")
	}
	if zoneID == "" {
		return nil, validationError("NewSpectrumService", "zone ID is required")
	}
	return &SpectrumService{cf: api, zoneID: zoneID}, nil
}

// NewSpectrumServiceFromCreds creates a SpectrumService from zone ID and
// API token. Convenience helper for CLI usage.
func NewSpectrumServiceFromCreds(zoneID, apiToken string) (*SpectrumService, error) {
	if zoneID == "" {
		return nil, validationError("NewSpectrumServiceFromCreds", "zone ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewSpectrumServiceFromCreds", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewSpectrumServiceFromCreds", "failed to create Cloudflare API client", err)
	}
	return &SpectrumService{cf: cf, zoneID: zoneID}, nil
}

// Create adds a new Spectrum application to the zone. The returned app
// carries the ID Cloudflare assigned.
func (s *SpectrumService) Create(ctx context.Context, opts SpectrumAppCreate) (*SpectrumAppInfo, error) {
	if opts.Name == "" {
		return nil, validationError("SpectrumService.Create", "name is required")
	}
	if opts.Protocol == "" {
		return nil, validationError("SpectrumService.Create", "protocol is required (e.g. tcp/80)")
	}
	if opts.OriginPort == "" {
		return nil, validationError("SpectrumService.Create", "origin port is required")
	}
	if len(opts.OriginDirect) == 0 && opts.OriginDNSName == "" {
		return nil, validationError("SpectrumService.Create", "an origin is required (--origin-direct or --origin-dns)")
	}

	originPort, err := parseSpectrumOriginPort(opts.OriginPort)
	if err != nil {
		return nil, validationError("SpectrumService.Create", fmt.Sprintf("invalid origin port %q: %v", opts.OriginPort, err))
	}
	_ = originPort

	dnsType := opts.DNSType
	if dnsType == "" {
		dnsType = "CNAME"
	}
	app := cloudflare.SpectrumApplication{
		Protocol: opts.Protocol,
		DNS: cloudflare.SpectrumApplicationDNS{
			Type: dnsType,
			Name: opts.Name,
		},
		OriginDirect:     opts.OriginDirect,
		OriginPort:       originPort,
		TrafficType:      opts.TrafficType,
		TLS:              opts.TLS,
		ProxyProtocol:    cloudflare.ProxyProtocol(opts.ProxyProtocol),
		ArgoSmartRouting: opts.ArgoSmartRouting,
		IPv4:             opts.IPv4,
		IPFirewall:       opts.IPFirewall,
	}
	if opts.OriginDNSName != "" {
		app.OriginDNS = &cloudflare.SpectrumApplicationOriginDNS{Name: opts.OriginDNSName}
	}

	created, createErr := s.cf.CreateSpectrumApplication(ctx, s.zoneID, app)
	if createErr != nil {
		return nil, newError("SpectrumService.Create", fmt.Sprintf("failed to create Spectrum app %q", opts.Name), createErr)
	}
	mapped := spectrumAppFromCF(created)
	return &mapped, nil
}

// List returns every Spectrum application in the zone.
func (s *SpectrumService) List(ctx context.Context) ([]SpectrumAppInfo, error) {
	apps, err := s.cf.SpectrumApplications(ctx, s.zoneID)
	if err != nil {
		return nil, newError("SpectrumService.List", fmt.Sprintf("failed to list Spectrum apps in zone %q", s.zoneID), err)
	}
	out := make([]SpectrumAppInfo, 0, len(apps))
	for i := range apps {
		out = append(out, spectrumAppFromCF(apps[i]))
	}
	return out, nil
}

// Get fetches a single Spectrum application by ID.
func (s *SpectrumService) Get(ctx context.Context, appID string) (*SpectrumAppInfo, error) {
	if appID == "" {
		return nil, validationError("SpectrumService.Get", "application ID is required")
	}

	app, err := s.cf.SpectrumApplication(ctx, s.zoneID, appID)
	if err != nil {
		return nil, newError("SpectrumService.Get", fmt.Sprintf("failed to get Spectrum app %q", appID), err)
	}
	mapped := spectrumAppFromCF(app)
	return &mapped, nil
}

// Update applies a partial change to a Spectrum application. The app's
// current state is fetched first and the requested fields merged onto it,
// because the Spectrum update endpoint is a full PUT replace.
func (s *SpectrumService) Update(ctx context.Context, appID string, opts SpectrumAppUpdate) (*SpectrumAppInfo, error) {
	if appID == "" {
		return nil, validationError("SpectrumService.Update", "application ID is required")
	}

	current, err := s.cf.SpectrumApplication(ctx, s.zoneID, appID)
	if err != nil {
		return nil, newError("SpectrumService.Update", fmt.Sprintf("failed to fetch Spectrum app %q before update", appID), err)
	}

	patched := current
	if opts.Protocol != nil {
		patched.Protocol = *opts.Protocol
	}
	if opts.OriginDirect != nil {
		patched.OriginDirect = *opts.OriginDirect
	}
	if opts.OriginDNSName != nil {
		if *opts.OriginDNSName == "" {
			patched.OriginDNS = nil
		} else {
			patched.OriginDNS = &cloudflare.SpectrumApplicationOriginDNS{Name: *opts.OriginDNSName}
		}
	}
	if opts.OriginPort != nil {
		originPort, err := parseSpectrumOriginPort(*opts.OriginPort)
		if err != nil {
			return nil, validationError("SpectrumService.Update", fmt.Sprintf("invalid origin port %q: %v", *opts.OriginPort, err))
		}
		patched.OriginPort = originPort
	}
	if opts.DNSType != nil {
		patched.DNS.Type = *opts.DNSType
	}
	if opts.TrafficType != nil {
		patched.TrafficType = *opts.TrafficType
	}
	if opts.TLS != nil {
		patched.TLS = *opts.TLS
	}
	if opts.ProxyProtocol != nil {
		patched.ProxyProtocol = cloudflare.ProxyProtocol(*opts.ProxyProtocol)
	}
	if opts.ArgoSmartRouting != nil {
		patched.ArgoSmartRouting = *opts.ArgoSmartRouting
	}
	if opts.IPv4 != nil {
		patched.IPv4 = *opts.IPv4
	}
	if opts.IPFirewall != nil {
		patched.IPFirewall = *opts.IPFirewall
	}

	app, err := s.cf.UpdateSpectrumApplication(ctx, s.zoneID, appID, patched)
	if err != nil {
		return nil, newError("SpectrumService.Update", fmt.Sprintf("failed to update Spectrum app %q", appID), err)
	}
	mapped := spectrumAppFromCF(app)
	return &mapped, nil
}

// Delete removes a Spectrum application from the zone.
func (s *SpectrumService) Delete(ctx context.Context, appID string) error {
	if appID == "" {
		return validationError("SpectrumService.Delete", "application ID is required")
	}

	if err := s.cf.DeleteSpectrumApplication(ctx, s.zoneID, appID); err != nil {
		return newError("SpectrumService.Delete", fmt.Sprintf("failed to delete Spectrum app %q", appID), err)
	}
	return nil
}

// parseSpectrumOriginPort parses "port" or "start-end" into the
// cloudflare-go origin-port union. cloudflare-go keeps its parser
// unexported, so the same grammar is implemented here.
func parseSpectrumOriginPort(s string) (*cloudflare.SpectrumApplicationOriginPort, error) {
	start, end, err := splitPortRange(s)
	if err != nil {
		return nil, err
	}
	p := &cloudflare.SpectrumApplicationOriginPort{}
	if end > 0 {
		p.Start, p.End = start, end
	} else {
		p.Port = start
	}
	return p, nil
}

// splitPortRange parses a single uint16 port or a "start-end" range,
// returning (port, 0) for a single port and (start, end) for a range.
func splitPortRange(s string) (uint16, uint16, error) {
	parts := strings.Split(s, "-")
	switch len(parts) {
	case 1:
		v, err := strconv.ParseUint(parts[0], 10, 16)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid port %q", s)
		}
		return uint16(v), 0, nil
	case 2:
		start, err := strconv.ParseUint(parts[0], 10, 16)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid range %q", s)
		}
		end, err := strconv.ParseUint(parts[1], 10, 16)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid range %q", s)
		}
		if start >= end {
			return 0, 0, fmt.Errorf("range start must be below end in %q", s)
		}
		return uint16(start), uint16(end), nil
	default:
		return 0, 0, fmt.Errorf("invalid port or range %q", s)
	}
}

// spectrumAppFromCF maps the cloudflare-go Spectrum application to the
// public SpectrumAppInfo view, rendering timestamps as RFC3339 strings and
// the origin port union as "port" or "start-end".
func spectrumAppFromCF(a cloudflare.SpectrumApplication) SpectrumAppInfo {
	created, modified := "", ""
	if a.CreatedOn != nil {
		created = a.CreatedOn.Format(time.RFC3339)
	}
	if a.ModifiedOn != nil {
		modified = a.ModifiedOn.Format(time.RFC3339)
	}
	info := SpectrumAppInfo{
		ID:               a.ID,
		Protocol:         a.Protocol,
		DNSName:          a.DNS.Name,
		DNSType:          a.DNS.Type,
		TrafficType:      a.TrafficType,
		TLS:              a.TLS,
		ProxyProtocol:    string(a.ProxyProtocol),
		OriginDirect:     a.OriginDirect,
		ArgoSmartRouting: a.ArgoSmartRouting,
		IPv4:             a.IPv4,
		IPFirewall:       a.IPFirewall,
		CreatedOn:        created,
		ModifiedOn:       modified,
	}
	if a.OriginDNS != nil {
		info.OriginDNSName = a.OriginDNS.Name
	}
	if a.OriginPort != nil {
		if a.OriginPort.End > 0 {
			info.OriginPort = fmt.Sprintf("%d-%d", a.OriginPort.Start, a.OriginPort.End)
		} else {
			info.OriginPort = fmt.Sprintf("%d", a.OriginPort.Port)
		}
	}
	return info
}
