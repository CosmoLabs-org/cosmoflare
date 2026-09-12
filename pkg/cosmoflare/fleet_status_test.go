package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// fleetMockSetup builds a FleetStatusService against an httptest server,
// mirroring zoneMockSetup's cloudflare.BaseURL injection pattern.
func fleetMockSetup(handler http.HandlerFunc) (*FleetStatusService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewFleetStatusService(cf, "acct-test-123")
	return svc, server
}

func fleetWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// fleetFakeZone builds one zone's worth of API responses. certDaysFromNow
// nil means no certificate pack is returned.
type fleetFakeZone struct {
	id             string
	name           string
	status         string
	planLegacyID   string
	dnssecStatus   string // "" -> handler returns 404
	universalSSL   bool
	certDaysFromNow *int
	minTLS         string
	securityLevel  string
	devMode        string // "on" | "off"
}

func fleetTestServer(t *testing.T, zones []fleetFakeZone) *httptest.Server {
	t.Helper()
	byID := make(map[string]fleetFakeZone, len(zones))
	for _, z := range zones {
		byID[z.id] = z
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/zones":
			result := make([]map[string]interface{}, 0, len(zones))
			for _, z := range zones {
				result = append(result, map[string]interface{}{
					"id":     z.id,
					"name":   z.name,
					"status": z.status,
					"plan":   map[string]string{"id": "p1", "legacy_id": z.planLegacyID, "name": z.planLegacyID},
				})
			}
			fleetWriteJSON(w, map[string]interface{}{"success": true, "result": result})
			return
		case strings.HasSuffix(r.URL.Path, "/dnssec"):
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/zones/"), "/dnssec")
			z, ok := byID[id]
			if !ok || z.dnssecStatus == "" {
				w.WriteHeader(http.StatusNotFound)
				fleetWriteJSON(w, map[string]interface{}{"success": false, "errors": []map[string]interface{}{{"code": 1000, "message": "not found"}}})
				return
			}
			fleetWriteJSON(w, map[string]interface{}{"success": true, "result": map[string]interface{}{"status": z.dnssecStatus}})
			return
		case strings.HasSuffix(r.URL.Path, "/ssl/universal/settings"):
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/zones/"), "/ssl/universal/settings")
			z := byID[id]
			fleetWriteJSON(w, map[string]interface{}{"success": true, "result": map[string]interface{}{"enabled": z.universalSSL}})
			return
		case strings.HasSuffix(r.URL.Path, "/settings"):
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/zones/"), "/settings")
			z := byID[id]
			result := []map[string]interface{}{
				{"id": "min_tls_version", "value": z.minTLS},
				{"id": "security_level", "value": z.securityLevel},
				{"id": "development_mode", "value": z.devMode},
			}
			fleetWriteJSON(w, map[string]interface{}{"success": true, "result": result})
			return
		case strings.HasSuffix(r.URL.Path, "/ssl/certificate_packs"):
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/zones/"), "/ssl/certificate_packs")
			z := byID[id]
			if z.certDaysFromNow == nil {
				fleetWriteJSON(w, map[string]interface{}{"success": true, "result": []interface{}{}})
				return
			}
			expires := time.Now().Add(time.Duration(*z.certDaysFromNow) * 24 * time.Hour)
			fleetWriteJSON(w, map[string]interface{}{"success": true, "result": []map[string]interface{}{
				{
					"id":     "cert1",
					"type":   "universal",
					"status": "active",
					"certificates": []map[string]interface{}{
						{"id": "c1", "status": "active", "expires_on": expires.Format(time.RFC3339)},
					},
				},
			}})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			fleetWriteJSON(w, map[string]interface{}{"success": false})
		}
	}))
}

func TestNewFleetStatusServiceValidation(t *testing.T) {
	if _, err := NewFleetStatusService(nil, "acct"); err == nil {
		t.Fatal("nil API must fail validation")
	}
	cf, _ := cloudflare.NewWithAPIToken("tok")
	if _, err := NewFleetStatusService(cf, ""); err == nil {
		t.Fatal("empty account ID must fail validation")
	}
}

func TestNewFleetStatusServiceFromCredsValidation(t *testing.T) {
	if _, err := NewFleetStatusServiceFromCreds("", "tok"); err == nil {
		t.Fatal("empty account ID must fail validation")
	}
	if _, err := NewFleetStatusServiceFromCreds("acct", ""); err == nil {
		t.Fatal("empty API token must fail validation")
	}
}

func TestFleetSnapshotHealthyZone(t *testing.T) {
	days := 90
	srv := fleetTestServer(t, []fleetFakeZone{
		{id: "z1", name: "healthy.example", status: "active", planLegacyID: "pro",
			dnssecStatus: "active", universalSSL: true, certDaysFromNow: &days,
			minTLS: "1.2", securityLevel: "medium", devMode: "off"},
	})
	defer srv.Close()
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(srv.URL))
	svc, err := NewFleetStatusService(cf, "acct-test-123")
	if err != nil {
		t.Fatalf("NewFleetStatusService: %v", err)
	}

	snap, err := svc.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(snap.Zones) != 1 {
		t.Fatalf("len(Zones) = %d, want 1", len(snap.Zones))
	}
	z := snap.Zones[0]
	if !z.ZoneActive || !z.NameserversOK {
		t.Fatalf("zone should be active with OK nameservers: %+v", z)
	}
	if z.DNSSECStatus != "active" {
		t.Fatalf("DNSSECStatus = %q, want active", z.DNSSECStatus)
	}
	if !z.UniversalSSL {
		t.Fatal("UniversalSSL should be true")
	}
	if z.CertExpiresIn == nil || *z.CertExpiresIn < 89 || *z.CertExpiresIn > 90 {
		t.Fatalf("CertExpiresIn = %v, want ~90", z.CertExpiresIn)
	}
	if z.MinTLS != "1.2" || z.SecurityLevel != "medium" || z.DevMode {
		t.Fatalf("settings not joined correctly: %+v", z)
	}
	if len(z.Issues) != 0 {
		t.Fatalf("healthy zone should have no issues, got %v", z.Issues)
	}
	if snap.HealthyCount != 1 || snap.DegradedCount != 0 {
		t.Fatalf("HealthyCount=%d DegradedCount=%d, want 1/0", snap.HealthyCount, snap.DegradedCount)
	}
	if snap.CollectedAt == "" {
		t.Fatal("CollectedAt must be set")
	}
}

func TestFleetSnapshotDegradedZone(t *testing.T) {
	expiring := 5
	srv := fleetTestServer(t, []fleetFakeZone{
		{id: "z1", name: "broken.example", status: "pending", planLegacyID: "free",
			dnssecStatus: "", universalSSL: false, certDaysFromNow: &expiring,
			minTLS: "1.0", securityLevel: "low", devMode: "on"},
	})
	defer srv.Close()
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(srv.URL))
	svc, _ := NewFleetStatusService(cf, "acct-test-123")

	snap, err := svc.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	z := snap.Zones[0]
	if z.ZoneActive || z.NameserversOK {
		t.Fatalf("pending zone should not be active/OK: %+v", z)
	}
	if z.DNSSECStatus != "disabled" {
		t.Fatalf("DNSSECStatus = %q, want disabled (404 -> disabled)", z.DNSSECStatus)
	}
	if z.DevMode != true {
		t.Fatal("DevMode should be true")
	}
	if len(z.Issues) == 0 {
		t.Fatal("degraded zone should have issues")
	}
	if snap.DegradedCount != 1 || snap.HealthyCount != 0 {
		t.Fatalf("HealthyCount=%d DegradedCount=%d, want 0/1", snap.HealthyCount, snap.DegradedCount)
	}
}

func TestFleetSnapshotSortsIssuesFirstThenName(t *testing.T) {
	days := 90
	srv := fleetTestServer(t, []fleetFakeZone{
		{id: "z1", name: "aaa-healthy.example", status: "active", planLegacyID: "pro",
			dnssecStatus: "active", universalSSL: true, certDaysFromNow: &days,
			minTLS: "1.2", securityLevel: "medium", devMode: "off"},
		{id: "z2", name: "zzz-broken.example", status: "pending", planLegacyID: "free",
			dnssecStatus: "", universalSSL: false, certDaysFromNow: nil,
			minTLS: "1.0", securityLevel: "low", devMode: "on"},
	})
	defer srv.Close()
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(srv.URL))
	svc, _ := NewFleetStatusService(cf, "acct-test-123")

	snap, err := svc.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(snap.Zones) != 2 {
		t.Fatalf("len(Zones) = %d, want 2", len(snap.Zones))
	}
	if snap.Zones[0].Zone != "zzz-broken.example" {
		t.Fatalf("Zones[0] = %q, want the zone with issues first", snap.Zones[0].Zone)
	}
}

func TestFleetSnapshotUnknownCertExpiry(t *testing.T) {
	srv := fleetTestServer(t, []fleetFakeZone{
		{id: "z1", name: "nocert.example", status: "active", planLegacyID: "pro",
			dnssecStatus: "active", universalSSL: true, certDaysFromNow: nil,
			minTLS: "1.2", securityLevel: "medium", devMode: "off"},
	})
	defer srv.Close()
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(srv.URL))
	svc, _ := NewFleetStatusService(cf, "acct-test-123")

	snap, err := svc.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.Zones[0].CertExpiresIn != nil {
		t.Fatalf("CertExpiresIn = %v, want nil (unknown)", snap.Zones[0].CertExpiresIn)
	}
}

func TestFleetSnapshotAllZonesFailIsHardError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fleetWriteJSON(w, map[string]interface{}{"success": false})
	}))
	defer srv.Close()
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(srv.URL))
	svc, _ := NewFleetStatusService(cf, "acct-test-123")

	if _, err := svc.Snapshot(context.Background()); err == nil {
		t.Fatal("zone list failure must be a hard error")
	}
}

func TestFleetSnapshotEmptyFleet(t *testing.T) {
	srv := fleetTestServer(t, nil)
	defer srv.Close()
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(srv.URL))
	svc, _ := NewFleetStatusService(cf, "acct-test-123")

	snap, err := svc.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(snap.Zones) != 0 || snap.HealthyCount != 0 || snap.DegradedCount != 0 {
		t.Fatalf("empty fleet should yield zero counts: %+v", snap)
	}
}
