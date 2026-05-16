package r2go2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func zoneMockSetup(handler http.HandlerFunc) (*ZoneService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewZoneService(cf, "acct-test-123")
	return svc, server
}

func zoneWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func TestNewZoneServiceValidation(t *testing.T) {
	_, err := NewZoneService(nil, "acct123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewZoneService(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewZoneService(nil, "")
	if err == nil {
		t.Error("expected error when both API client and accountID are empty")
	}
}

func TestNewZoneServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewZoneService(cf, "acct123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

func TestNewZoneServiceFromCredsValidation(t *testing.T) {
	_, err := NewZoneServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewZoneServiceFromCreds("acct123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}

	_, err = NewZoneServiceFromCreds("", "")
	if err == nil {
		t.Error("expected error when both accountID and apiToken are empty")
	}
}

func TestNewZoneServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewZoneServiceFromCreds("acct123", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

func TestZoneCreateValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewZoneService(cf, "acct123")

	_, err := svc.Create(context.Background(), "", "full")
	if err == nil {
		t.Error("expected error when zone name is empty")
	}

	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

func TestZoneCreateDefaultType(t *testing.T) {
	// Verify that Create defaults zoneType to "full" when empty.
	// We can't test the actual API call, but we verify no validation
	// error is returned for the empty zoneType — the error will come
	// from the API call itself (no real credentials), not from validation.
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewZoneService(cf, "acct123")

	_, err := svc.Create(context.Background(), "example.com", "")
	// The call will fail at the API level (no real creds), but it should
	// NOT fail with a validation error about zoneType being empty.
	if err != nil {
		var valErr *R2ValidationError
		if errors.As(err, &valErr) {
			t.Errorf("empty zoneType should default to 'full', not produce validation error: %v", err)
		}
		// Any other error (API/network) is expected — no real credentials
	}
}

func TestZoneGetValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewZoneService(cf, "acct123")

	_, err := svc.Get(context.Background(), "")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

func TestZoneDeleteValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewZoneService(cf, "acct123")

	err := svc.Delete(context.Background(), "")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

func TestZoneGetSettingsValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewZoneService(cf, "acct123")

	_, err := svc.GetSettings(context.Background(), "")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

func TestZoneConstructorErrorTypes(t *testing.T) {
	// NewZoneService with nil API returns validation error
	_, err := NewZoneService(nil, "acct123")
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for nil API, got %T", err)
	}

	// NewZoneService with empty accountID returns validation error
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewZoneService(cf, "")
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty accountID, got %T", err)
	}

	// NewZoneServiceFromCreds with empty accountID returns validation error
	_, err = NewZoneServiceFromCreds("", "test-token")
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty accountID in FromCreds, got %T", err)
	}

	// NewZoneServiceFromCreds with empty apiToken returns validation error
	_, err = NewZoneServiceFromCreds("acct123", "")
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty apiToken in FromCreds, got %T", err)
	}
}

func TestCfZoneToZone(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-24 * time.Hour)

	cfZone := cloudflare.Zone{
		ID:          "zone-abc-123",
		Name:        "example.com",
		Status:      "active",
		Type:        "full",
		Paused:      false,
		NameServers: []string{"ns1.cloudflare.com", "ns2.cloudflare.com"},
		OriginalNS:  []string{"ns1.original.com"},
		Plan: cloudflare.ZonePlan{
			ZonePlanCommon: cloudflare.ZonePlanCommon{
				ID:   "plan-free",
				Name: "Free Website",
			},
		},
		CreatedOn:  earlier,
		ModifiedOn: now,
	}

	zone := cfZoneToZone(cfZone)

	if zone.ID != "zone-abc-123" {
		t.Errorf("expected ID=zone-abc-123, got %s", zone.ID)
	}
	if zone.Name != "example.com" {
		t.Errorf("expected Name=example.com, got %s", zone.Name)
	}
	if zone.Status != "active" {
		t.Errorf("expected Status=active, got %s", zone.Status)
	}
	if zone.Type != "full" {
		t.Errorf("expected Type=full, got %s", zone.Type)
	}
	if zone.Paused != false {
		t.Errorf("expected Paused=false, got %v", zone.Paused)
	}
	if len(zone.NameServers) != 2 {
		t.Errorf("expected 2 name servers, got %d", len(zone.NameServers))
	} else {
		if zone.NameServers[0] != "ns1.cloudflare.com" {
			t.Errorf("expected first NS=ns1.cloudflare.com, got %s", zone.NameServers[0])
		}
		if zone.NameServers[1] != "ns2.cloudflare.com" {
			t.Errorf("expected second NS=ns2.cloudflare.com, got %s", zone.NameServers[1])
		}
	}
	if len(zone.OriginalNS) != 1 {
		t.Errorf("expected 1 original NS, got %d", len(zone.OriginalNS))
	} else if zone.OriginalNS[0] != "ns1.original.com" {
		t.Errorf("expected original NS=ns1.original.com, got %s", zone.OriginalNS[0])
	}
	if zone.Plan.ID != "plan-free" {
		t.Errorf("expected Plan.ID=plan-free, got %s", zone.Plan.ID)
	}
	if zone.Plan.Name != "Free Website" {
		t.Errorf("expected Plan.Name=Free Website, got %s", zone.Plan.Name)
	}
	if !zone.CreatedOn.Equal(earlier) {
		t.Errorf("expected CreatedOn=%v, got %v", earlier, zone.CreatedOn)
	}
	if !zone.ModifiedOn.Equal(now) {
		t.Errorf("expected ModifiedOn=%v, got %v", now, zone.ModifiedOn)
	}
}

func TestCfZoneToZonePaused(t *testing.T) {
	cfZone := cloudflare.Zone{
		ID:     "zone-paused",
		Name:   "paused.com",
		Paused: true,
	}
	zone := cfZoneToZone(cfZone)
	if zone.Paused != true {
		t.Errorf("expected Paused=true, got %v", zone.Paused)
	}
}

func TestZoneTypes(t *testing.T) {
	z := &Zone{
		ID:          "z1",
		Name:        "test.com",
		Status:      "active",
		Type:        "full",
		Paused:      false,
		NameServers: []string{"ns1.cf.com"},
		OriginalNS:  []string{"ns1.old.com"},
		Plan:        ZonePlan{ID: "free", Name: "Free"},
		CreatedOn:   time.Now(),
		ModifiedOn:  time.Now(),
	}
	if z.ID != "z1" {
		t.Errorf("unexpected ID: %s", z.ID)
	}
	if z.Name != "test.com" {
		t.Errorf("unexpected Name: %s", z.Name)
	}
	if z.Status != "active" {
		t.Errorf("unexpected Status: %s", z.Status)
	}

	s := &ZoneSetting{
		ID:         "ssl",
		Value:      "full",
		Editable:   true,
		ModifiedOn: "2026-01-01T00:00:00Z",
	}
	if s.ID != "ssl" {
		t.Errorf("unexpected setting ID: %s", s.ID)
	}
	if s.Value != "full" {
		t.Errorf("unexpected setting Value: %v", s.Value)
	}
	if !s.Editable {
		t.Error("expected Editable=true")
	}
}

// --- httptest-based API mock tests ---

func TestZoneCreateWithMock(t *testing.T) {
	svc, server := zoneMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		now := time.Now()
		zoneWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":           "zone-new-001",
				"name":         "newsite.com",
				"status":       "pending",
				"type":         "full",
				"paused":       false,
				"name_servers": []string{"ns1.cloudflare.com", "ns2.cloudflare.com"},
				"plan":         map[string]interface{}{"id": "free", "name": "Free"},
				"created_on":   now.Format(time.RFC3339),
				"modified_on":  now.Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	zone, err := svc.Create(ctx, "newsite.com", "full")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if zone.ID != "zone-new-001" {
		t.Errorf("expected ID=zone-new-001, got %s", zone.ID)
	}
	if zone.Name != "newsite.com" {
		t.Errorf("expected Name=newsite.com, got %s", zone.Name)
	}
	if zone.Status != "pending" {
		t.Errorf("expected Status=pending, got %s", zone.Status)
	}
}

func TestZoneListWithMock(t *testing.T) {
	svc, server := zoneMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		zoneWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{
					"id": "zone-001", "name": "site1.com", "status": "active",
					"type": "full", "paused": false,
					"name_servers": []string{"ns1.cf.com"},
					"plan":         map[string]interface{}{"id": "free", "name": "Free"},
				},
				{
					"id": "zone-002", "name": "site2.com", "status": "active",
					"type": "full", "paused": false,
					"name_servers": []string{"ns2.cf.com"},
					"plan":         map[string]interface{}{"id": "pro", "name": "Pro"},
				},
			},
			"result_info": map[string]interface{}{"page": 1, "total_pages": 1, "count": 2},
		})
	})
	defer server.Close()

	ctx := context.Background()
	zones, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(zones) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(zones))
	}
	if zones[0].Name != "site1.com" {
		t.Errorf("expected first zone=site1.com, got %s", zones[0].Name)
	}
	if zones[1].Name != "site2.com" {
		t.Errorf("expected second zone=site2.com, got %s", zones[1].Name)
	}
}

func TestZoneGetWithMock(t *testing.T) {
	svc, server := zoneMockSetup(func(w http.ResponseWriter, r *http.Request) {
		zoneWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id": "zone-get-001", "name": "gettest.com", "status": "active",
				"type": "full", "paused": false,
				"name_servers": []string{"ns1.cf.com"},
				"plan":         map[string]interface{}{"id": "free", "name": "Free"},
				"created_on":   time.Now().Format(time.RFC3339),
				"modified_on":  time.Now().Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	zone, err := svc.Get(ctx, "zone-get-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if zone.ID != "zone-get-001" {
		t.Errorf("expected ID=zone-get-001, got %s", zone.ID)
	}
	if zone.Name != "gettest.com" {
		t.Errorf("expected Name=gettest.com, got %s", zone.Name)
	}
}

func TestZoneGetNotFound(t *testing.T) {
	svc, server := zoneMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		zoneWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "zone not found"}},
		})
	})
	defer server.Close()

	ctx := context.Background()
	_, err := svc.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestZoneDeleteWithMock(t *testing.T) {
	svc, server := zoneMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		zoneWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"id": "zone-del-001"},
		})
	})
	defer server.Close()

	ctx := context.Background()
	err := svc.Delete(ctx, "zone-del-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestZoneGetSettingsWithMock(t *testing.T) {
	svc, server := zoneMockSetup(func(w http.ResponseWriter, r *http.Request) {
		zoneWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "ssl", "value": "full", "editable": true, "modified_on": "2026-01-01T00:00:00Z"},
				{"id": "always_use_https", "value": "on", "editable": true},
				{"id": "min_tls_version", "value": "1.2", "editable": true},
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	settings, err := svc.GetSettings(ctx, "zone-get-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(settings) < 2 {
		t.Fatalf("expected at least 2 settings, got %d", len(settings))
	}
	found := false
	for _, s := range settings {
		if s.ID == "ssl" {
			found = true
			if s.Value != "full" {
				t.Errorf("expected ssl value=full, got %v", s.Value)
			}
		}
	}
	if !found {
		t.Error("expected ssl setting to be present")
	}
}

func TestZoneCreateAPIError(t *testing.T) {
	svc, server := zoneMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		zoneWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1001, "message": "invalid request"}},
		})
	})
	defer server.Close()

	ctx := context.Background()
	_, err := svc.Create(ctx, "fail.com", "full")
	if err == nil {
		t.Fatal("expected error from API")
	}
	if _, ok := err.(*R2Error); !ok {
		t.Errorf("expected *R2Error, got %T", err)
	}
}

func TestZoneJSONMarshal(t *testing.T) {
	z := &Zone{
		ID: "z-json", Name: "json.com", Status: "active", Type: "full",
		NameServers: []string{"ns1.cf.com"}, Plan: ZonePlan{ID: "free", Name: "Free"},
		CreatedOn: time.Now(), ModifiedOn: time.Now(),
	}
	data, err := json.Marshal(z)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded Zone
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.ID != "z-json" {
		t.Errorf("ID mismatch: got %q", decoded.ID)
	}
	if decoded.Name != "json.com" {
		t.Errorf("Name mismatch: got %q", decoded.Name)
	}
}

func TestZoneNewFromCredsSuccess(t *testing.T) {
	svc, err := NewZoneServiceFromCreds("acct123", fmt.Sprintf("test-token-%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
}
