package cosmoflare

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// zoneConcurrency bounds the parallel per-zone status fetches in Snapshot.
// Same bound as limits.go's dnsConcurrency — one CF-API-friendly default
// shared across fleet-wide collectors.
const zoneConcurrency = 8

// certExpiryWarningDays flags a certificate as an issue once it is this
// close to expiring, but does not by itself count the zone as degraded.
const certExpiryWarningDays = 30

// ZoneStatus is the fleet-wide protection snapshot for one zone.
type ZoneStatus struct {
	ZoneID        string   `json:"zone_id"`
	Zone          string   `json:"zone"`
	Plan          string   `json:"plan"`
	ZoneActive    bool     `json:"zone_active"`
	NameserversOK bool     `json:"nameservers_ok"`
	DNSSECStatus  string   `json:"dnssec_status"` // active|pending|disabled|unknown
	UniversalSSL  bool     `json:"universal_ssl_active"`
	CertExpiresIn *int     `json:"cert_expires_in_days,omitempty"` // nil = unknown
	MinTLS        string   `json:"min_tls_version"`
	SecurityLevel string   `json:"security_level"`
	DevMode       bool     `json:"dev_mode"`
	Issues        []string `json:"issues"`
}

// FleetStatus is the result of one fleet-wide collection pass.
type FleetStatus struct {
	Zones         []ZoneStatus `json:"zones"`
	HealthyCount  int          `json:"healthy_count"`
	DegradedCount int          `json:"degraded_count"`
	CollectedAt   string       `json:"collected_at"`
}

// FleetStatusService collects the API-reported protection status for every
// zone on an account: DNSSEC, Universal SSL and certificate expiry, and the
// zone settings (min TLS, security level, dev mode) that most affect a
// domain's protection posture. Unlike DoctorService, which network-probes a
// single domain from the outside, FleetStatusService reads Cloudflare's own
// account-side state for every zone in one pass.
type FleetStatusService struct {
	cf    *cloudflare.API
	zones *ZoneService
}

// NewFleetStatusService creates a fleet status collector from an existing
// Cloudflare API client and account ID.
func NewFleetStatusService(api *cloudflare.API, accountID string) (*FleetStatusService, error) {
	if api == nil {
		return nil, validationError("NewFleetStatusService", "cloudflare API client is required")
	}
	zones, err := NewZoneService(api, accountID)
	if err != nil {
		return nil, err
	}
	return &FleetStatusService{cf: api, zones: zones}, nil
}

// NewFleetStatusServiceFromCreds creates a FleetStatusService from account ID
// and API token. Convenience helper for CLI usage.
func NewFleetStatusServiceFromCreds(accountID, apiToken string) (*FleetStatusService, error) {
	if accountID == "" {
		return nil, validationError("NewFleetStatusService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewFleetStatusService", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewFleetStatusService", "failed to create Cloudflare API client", err)
	}
	return NewFleetStatusService(cf, accountID)
}

// Snapshot collects the protection status of every zone on the account.
// Per-zone reads run concurrently (bounded by zoneConcurrency) since they are
// independent — one slow or failing zone never blocks the rest. A failing
// per-zone sub-check (DNSSEC, SSL, settings) is recorded as an issue on that
// zone rather than failing the whole zone; only a failed zone list, or every
// zone individually erroring before any status could be built, is a hard
// failure.
func (s *FleetStatusService) Snapshot(ctx context.Context) (*FleetStatus, error) {
	const op = "FleetStatusSnapshot"

	zones, err := s.zones.List(ctx)
	if err != nil {
		return nil, err
	}

	statuses := make([]ZoneStatus, len(zones))
	failed := make([]bool, len(zones))
	sem := make(chan struct{}, zoneConcurrency)
	var wg sync.WaitGroup
	for i, z := range zones {
		wg.Add(1)
		go func(idx int, zone *Zone) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			statuses[idx], failed[idx] = s.zoneStatus(ctx, zone)
		}(i, z)
	}
	wg.Wait()

	if len(zones) > 0 {
		allFailed := true
		for _, f := range failed {
			if !f {
				allFailed = false
				break
			}
		}
		if allFailed {
			return nil, newError(op, "every zone's status checks failed", nil)
		}
	}

	sortZoneStatuses(statuses)

	snap := &FleetStatus{Zones: statuses, CollectedAt: nowISO8601()}
	for _, z := range statuses {
		if zoneIsHealthy(z) {
			snap.HealthyCount++
		} else {
			snap.DegradedCount++
		}
	}
	return snap, nil
}

// zoneStatus builds one zone's protection status. Individual sub-check
// failures are recorded as an issue string and leave the corresponding field
// at its zero value / "unknown" — never fabricated — so one flaky endpoint
// doesn't hide the rest of the zone's state. failed reports whether every
// one of the four API sub-checks errored — Snapshot treats "every zone
// failed this way" as a hard error, mirroring LimitsService's
// zero-rows-with-errors rule.
func (s *FleetStatusService) zoneStatus(ctx context.Context, zone *Zone) (ZoneStatus, bool) {
	// CF only reports zone status "active" once the account's assigned
	// nameservers are actually in place at the registrar — an active zone
	// implies correctly pointed nameservers, so the two fields share one
	// source signal here (there is no separate "nameservers verified" flag
	// on the Zone resource).
	zs := ZoneStatus{
		ZoneID:        zone.ID,
		Zone:          zone.Name,
		Plan:          zone.Plan.LegacyID,
		ZoneActive:    zone.Status == "active",
		NameserversOK: zone.Status == "active",
		DNSSECStatus:  "unknown",
	}

	errCount := 0

	if dnssec, err := s.cf.ZoneDNSSECSetting(ctx, zone.ID); err != nil {
		if isNotFound(err) {
			zs.DNSSECStatus = "disabled"
		} else {
			zs.Issues = append(zs.Issues, "dnssec check failed: "+err.Error())
			errCount++
		}
	} else if dnssec.Status != "" {
		zs.DNSSECStatus = dnssec.Status
	} else {
		zs.DNSSECStatus = "disabled"
	}

	if ussl, err := s.cf.UniversalSSLSettingDetails(ctx, zone.ID); err != nil {
		zs.Issues = append(zs.Issues, "universal SSL check failed: "+err.Error())
		errCount++
	} else {
		zs.UniversalSSL = ussl.Enabled
	}

	if days, err := s.certExpiresInDays(ctx, zone.ID); err != nil {
		zs.Issues = append(zs.Issues, "certificate check failed: "+err.Error())
		errCount++
	} else {
		zs.CertExpiresIn = days
	}

	if settings, err := s.cf.ZoneSettings(ctx, zone.ID); err != nil {
		zs.Issues = append(zs.Issues, "zone settings check failed: "+err.Error())
		errCount++
	} else {
		applyZoneSettings(&zs, settings)
	}

	zs.Issues = append(zs.Issues, protectionIssues(zs)...)
	return zs, errCount == 4
}

// certExpiresInDays returns the number of days until the fleet's active
// Universal/dedicated certificate expires, using the earliest expiry among
// the zone's active certificate packs' active certificates. nil, nil means
// no active certificate was found (unknown, not fabricated as expired).
func (s *FleetStatusService) certExpiresInDays(ctx context.Context, zoneID string) (*int, error) {
	packs, err := s.cf.ListCertificatePacks(ctx, zoneID)
	if err != nil {
		return nil, err
	}

	var earliest *time.Time
	for _, pack := range packs {
		if pack.Status != "active" {
			continue
		}
		for _, cert := range pack.Certificates {
			if cert.Status != "active" {
				continue
			}
			if earliest == nil || cert.ExpiresOn.Before(*earliest) {
				t := cert.ExpiresOn
				earliest = &t
			}
		}
	}
	if earliest == nil {
		return nil, nil
	}
	days := int(time.Until(*earliest).Hours() / 24)
	return &days, nil
}

// applyZoneSettings joins the fields FleetStatus needs from one ZoneSettings
// batch call, following the same setting-ID switch pattern as
// SSLService.GetSettings.
func applyZoneSettings(zs *ZoneStatus, settings *cloudflare.ZoneSettingResponse) {
	for _, setting := range settings.Result {
		v, ok := setting.Value.(string)
		if !ok {
			continue
		}
		switch setting.ID {
		case "min_tls_version":
			zs.MinTLS = v
		case "security_level":
			zs.SecurityLevel = v
		case "development_mode":
			zs.DevMode = v == "on"
		}
	}
}

// protectionIssues returns the informational issue strings for a zone's
// already-collected status. DNSSEC disabled and an approaching cert expiry
// are informational — reported but not by themselves degrading; zoneIsHealthy
// decides what actually counts as broken.
func protectionIssues(zs ZoneStatus) []string {
	var issues []string
	if !zs.ZoneActive {
		issues = append(issues, "zone not active")
	}
	if zs.DNSSECStatus == "disabled" {
		issues = append(issues, "DNSSEC disabled")
	}
	if zs.CertExpiresIn != nil {
		if *zs.CertExpiresIn < 0 {
			issues = append(issues, "certificate expired")
		} else if *zs.CertExpiresIn < certExpiryWarningDays {
			issues = append(issues, "certificate expiring soon")
		}
	}
	if zs.DevMode {
		issues = append(issues, "development mode enabled")
	}
	return issues
}

// zoneIsHealthy reports whether a zone counts toward HealthyCount: active,
// a non-expired certificate (or unknown — never treated as broken), and
// Universal SSL active. DNSSEC disabled and dev mode are informational only.
func zoneIsHealthy(zs ZoneStatus) bool {
	if !zs.ZoneActive {
		return false
	}
	if !zs.UniversalSSL {
		return false
	}
	if zs.CertExpiresIn != nil && *zs.CertExpiresIn < 0 {
		return false
	}
	return true
}

// sortZoneStatuses orders zones with issues first, then alphabetically by
// name within each group.
func sortZoneStatuses(zones []ZoneStatus) {
	sort.SliceStable(zones, func(i, j int) bool {
		iHas, jHas := len(zones[i].Issues) > 0, len(zones[j].Issues) > 0
		if iHas != jHas {
			return iHas
		}
		return zones[i].Zone < zones[j].Zone
	})
}

// nowISO8601 returns the current time as a full ISO8601+timezone timestamp.
func nowISO8601() string {
	return time.Now().Format(time.RFC3339)
}
