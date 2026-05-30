package cosmoflare

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

// dnsMockSetup creates a mock Cloudflare API server and DNS service for testing.
func dnsMockSetup(handler http.HandlerFunc) (*DNSService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewDNSService(cf, "zone-test-123")
	return svc, server
}

func dnsWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- Constructor validation ---

func TestNewDNSServiceValidation(t *testing.T) {
	_, err := NewDNSService(nil, "zone123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewDNSService(cf, "")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	_, err = NewDNSService(nil, "")
	if err == nil {
		t.Error("expected error when both API client and zoneID are empty")
	}
}

func TestNewDNSServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewDNSService(cf, "zone123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.zoneID != "zone123" {
		t.Errorf("expected zoneID=zone123, got %s", svc.zoneID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

func TestNewDNSServiceFromCredsValidation(t *testing.T) {
	_, err := NewDNSServiceFromCreds("", "test-token")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	_, err = NewDNSServiceFromCreds("zone123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}

	_, err = NewDNSServiceFromCreds("", "")
	if err == nil {
		t.Error("expected error when both zoneID and apiToken are empty")
	}
}

func TestNewDNSServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewDNSServiceFromCreds("zone123", "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.zoneID != "zone123" {
		t.Errorf("expected zoneID=zone123, got %s", svc.zoneID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

// --- Error type assertions ---

func TestDNSConstructorValidationErrorType(t *testing.T) {
	_, err := NewDNSService(nil, "zone123")
	if err == nil {
		t.Fatal("expected error")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewDNSService(cf, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

func TestDNSFromCredsValidationErrorType(t *testing.T) {
	_, err := NewDNSServiceFromCreds("", "test-token")
	if err == nil {
		t.Fatal("expected error")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty zoneID, got %T", err)
	}

	_, err = NewDNSServiceFromCreds("zone123", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty apiToken, got %T", err)
	}
}

// --- Functional options ---

func TestDNSOptions(t *testing.T) {
	cfg := &dnsConfig{}

	WithDNSProxied(true)(cfg)
	if cfg.proxied == nil || !*cfg.proxied {
		t.Error("expected proxied=true")
	}

	WithDNSProxied(false)(cfg)
	if cfg.proxied == nil || *cfg.proxied {
		t.Error("expected proxied=false")
	}

	WithDNSTTL(300)(cfg)
	if cfg.ttl != 300 {
		t.Errorf("expected ttl=300, got %d", cfg.ttl)
	}

	WithDNSTTL(1)(cfg)
	if cfg.ttl != 1 {
		t.Errorf("expected ttl=1 (automatic), got %d", cfg.ttl)
	}

	WithDNSPriority(10)(cfg)
	if cfg.priority == nil || *cfg.priority != 10 {
		t.Errorf("expected priority=10, got %v", cfg.priority)
	}

	WithDNSComment("test comment")(cfg)
	if cfg.comment == nil || *cfg.comment != "test comment" {
		t.Errorf("expected comment='test comment', got %v", cfg.comment)
	}

	WithDNSComment("")(cfg)
	if cfg.comment == nil || *cfg.comment != "" {
		t.Error("expected comment to be set to empty string")
	}
}

func TestDNSListOptions(t *testing.T) {
	cfg := &dnsListConfig{}

	WithDNSType("A")(cfg)
	if cfg.recordType != "A" {
		t.Errorf("expected recordType=A, got %s", cfg.recordType)
	}

	WithDNSType("CNAME")(cfg)
	if cfg.recordType != "CNAME" {
		t.Errorf("expected recordType=CNAME, got %s", cfg.recordType)
	}

	WithDNSName("www.example.com")(cfg)
	if cfg.name != "www.example.com" {
		t.Errorf("expected name=www.example.com, got %s", cfg.name)
	}

	WithDNSContent("1.2.3.4")(cfg)
	if cfg.content != "1.2.3.4" {
		t.Errorf("expected content=1.2.3.4, got %s", cfg.content)
	}
}

// --- Method input validation ---

func TestDNSCreateValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewDNSService(cf, "zone123")
	ctx := context.Background()

	_, err := svc.Create(ctx, "", "name", "content")
	if err == nil {
		t.Error("expected error when record type is empty")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty type, got %T", err)
	}

	_, err = svc.Create(ctx, "A", "", "content")
	if err == nil {
		t.Error("expected error when name is empty")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty name, got %T", err)
	}

	_, err = svc.Create(ctx, "A", "name", "")
	if err == nil {
		t.Error("expected error when content is empty")
	}
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError for empty content, got %T", err)
	}
}

func TestDNSGetValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewDNSService(cf, "zone123")
	ctx := context.Background()

	_, err := svc.Get(ctx, "")
	if err == nil {
		t.Error("expected error when recordID is empty")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

func TestDNSUpdateValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewDNSService(cf, "zone123")
	ctx := context.Background()

	_, err := svc.Update(ctx, "")
	if err == nil {
		t.Error("expected error when recordID is empty")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

func TestDNSDeleteValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewDNSService(cf, "zone123")
	ctx := context.Background()

	err := svc.Delete(ctx, "")
	if err == nil {
		t.Error("expected error when recordID is empty")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected *R2ValidationError, got %T", err)
	}
}

// --- Helper function tests ---

func TestBoolVal(t *testing.T) {
	if boolVal(nil) != false {
		t.Error("expected boolVal(nil) to return false")
	}

	tr := true
	if boolVal(&tr) != true {
		t.Error("expected boolVal(&true) to return true")
	}

	fl := false
	if boolVal(&fl) != false {
		t.Error("expected boolVal(&false) to return false")
	}
}

func TestBoolPtr(t *testing.T) {
	p := boolPtr(true)
	if p == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *p != true {
		t.Error("expected *boolPtr(true) to be true")
	}

	p = boolPtr(false)
	if p == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *p != false {
		t.Error("expected *boolPtr(false) to be false")
	}
}

func TestCfDNSToRecord(t *testing.T) {
	proxied := true
	priority := uint16(10)
	now := time.Now()

	cfRecord := cloudflare.DNSRecord{
		ID:         "rec-abc123",
		Type:       "MX",
		Name:       "mail.example.com",
		Content:    "mx1.example.com",
		TTL:        3600,
		Proxied:    &proxied,
		Priority:   &priority,
		Comment:    "mail server",
		Proxiable:  true,
		CreatedOn:  now,
		ModifiedOn: now,
	}

	record := cfDNSToRecord(cfRecord, "zone-xyz")

	if record.ID != "rec-abc123" {
		t.Errorf("expected ID=rec-abc123, got %s", record.ID)
	}
	if record.Type != "MX" {
		t.Errorf("expected Type=MX, got %s", record.Type)
	}
	if record.Name != "mail.example.com" {
		t.Errorf("expected Name=mail.example.com, got %s", record.Name)
	}
	if record.Content != "mx1.example.com" {
		t.Errorf("expected Content=mx1.example.com, got %s", record.Content)
	}
	if record.TTL != 3600 {
		t.Errorf("expected TTL=3600, got %d", record.TTL)
	}
	if record.Proxied != true {
		t.Error("expected Proxied=true")
	}
	if record.Priority == nil || *record.Priority != 10 {
		t.Errorf("expected Priority=10, got %v", record.Priority)
	}
	if record.Comment != "mail server" {
		t.Errorf("expected Comment='mail server', got %s", record.Comment)
	}
	if record.ZoneID != "zone-xyz" {
		t.Errorf("expected ZoneID=zone-xyz, got %s", record.ZoneID)
	}
	if record.Proxiable != true {
		t.Error("expected Proxiable=true")
	}
	if !record.CreatedOn.Equal(now) {
		t.Errorf("expected CreatedOn=%v, got %v", now, record.CreatedOn)
	}
	if !record.ModifiedOn.Equal(now) {
		t.Errorf("expected ModifiedOn=%v, got %v", now, record.ModifiedOn)
	}
}

func TestCfDNSToRecordNilOptionals(t *testing.T) {
	cfRecord := cloudflare.DNSRecord{
		ID:      "rec-def456",
		Type:    "A",
		Name:    "example.com",
		Content: "93.184.216.34",
		TTL:     1,
		Proxied: nil,
	}

	record := cfDNSToRecord(cfRecord, "zone-abc")

	if record.Proxied != false {
		t.Error("expected Proxied=false when source is nil")
	}
	if record.Priority != nil {
		t.Errorf("expected Priority=nil, got %v", record.Priority)
	}
	if record.Comment != "" {
		t.Errorf("expected empty Comment, got %s", record.Comment)
	}
}

// --- DNSRecord type tests ---

func TestDNSRecordType(t *testing.T) {
	rec := &DNSRecord{
		ID:       "rec-001",
		Type:     "AAAA",
		Name:     "ipv6.example.com",
		Content:  "2001:db8::1",
		TTL:      300,
		Proxied:  false,
		ZoneID:   "zone-test",
		Comment:  "IPv6 record",
	}
	if rec.ID != "rec-001" {
		t.Errorf("unexpected ID: %s", rec.ID)
	}
	if rec.Type != "AAAA" {
		t.Errorf("unexpected Type: %s", rec.Type)
	}
	if rec.TTL != 300 {
		t.Errorf("unexpected TTL: %d", rec.TTL)
	}
	if rec.Comment != "IPv6 record" {
		t.Errorf("unexpected Comment: %s", rec.Comment)
	}
}

// --- httptest-based API mock tests ---

func TestDNSCreateWithMock(t *testing.T) {
	svc, server := dnsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		proxied := true
		dnsWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":         "rec-new-001",
				"type":       "A",
				"name":       "test.example.com",
				"content":    "1.2.3.4",
				"ttl":        300,
				"proxied":    proxied,
				"proxiable":  true,
				"created_on": time.Now().Format(time.RFC3339),
				"modified_on": time.Now().Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	rec, err := svc.Create(ctx, "A", "test.example.com", "1.2.3.4",
		WithDNSTTL(300), WithDNSProxied(true), WithDNSComment("test"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.ID != "rec-new-001" {
		t.Errorf("expected ID=rec-new-001, got %s", rec.ID)
	}
	if rec.Type != "A" {
		t.Errorf("expected Type=A, got %s", rec.Type)
	}
	if rec.Name != "test.example.com" {
		t.Errorf("expected Name=test.example.com, got %s", rec.Name)
	}
	if rec.Content != "1.2.3.4" {
		t.Errorf("expected Content=1.2.3.4, got %s", rec.Content)
	}
}

func TestDNSListWithMock(t *testing.T) {
	svc, server := dnsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		dnsWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{
					"id":      "rec-001",
					"type":    "A",
					"name":    "a.example.com",
					"content": "1.1.1.1",
					"ttl":     1,
					"proxied": true,
				},
				{
					"id":      "rec-002",
					"type":    "AAAA",
					"name":    "aaaa.example.com",
					"content": "::1",
					"ttl":     300,
					"proxied": false,
				},
			},
			"result_info": map[string]interface{}{"page": 1, "total_pages": 1, "count": 2},
		})
	})
	defer server.Close()

	ctx := context.Background()
	records, err := svc.List(ctx, WithDNSType("A"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].ID != "rec-001" {
		t.Errorf("expected first ID=rec-001, got %s", records[0].ID)
	}
	if records[1].Type != "AAAA" {
		t.Errorf("expected second Type=AAAA, got %s", records[1].Type)
	}
}

func TestDNSGetWithMock(t *testing.T) {
	svc, server := dnsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		dnsWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":      "rec-get-001",
				"type":    "CNAME",
				"name":    "www.example.com",
				"content": "example.com",
				"ttl":     1,
				"proxied": true,
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	rec, err := svc.Get(ctx, "rec-get-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.ID != "rec-get-001" {
		t.Errorf("expected ID=rec-get-001, got %s", rec.ID)
	}
	if rec.Type != "CNAME" {
		t.Errorf("expected Type=CNAME, got %s", rec.Type)
	}
}

func TestDNSGetNotFound(t *testing.T) {
	svc, server := dnsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		dnsWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "DNS record not found"}},
		})
	})
	defer server.Close()

	ctx := context.Background()
	_, err := svc.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestDNSUpdateWithMock(t *testing.T) {
	svc, server := dnsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		dnsWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":      "rec-upd-001",
				"type":    "A",
				"name":    "test.example.com",
				"content": "5.6.7.8",
				"ttl":     600,
				"proxied": false,
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	rec, err := svc.Update(ctx, "rec-upd-001", WithDNSTTL(600), WithDNSProxied(false))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.ID != "rec-upd-001" {
		t.Errorf("expected ID=rec-upd-001, got %s", rec.ID)
	}
}

func TestDNSDeleteWithMock(t *testing.T) {
	svc, server := dnsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		dnsWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"id": "rec-del-001"},
		})
	})
	defer server.Close()

	ctx := context.Background()
	err := svc.Delete(ctx, "rec-del-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDNSCreateAPIError(t *testing.T) {
	svc, server := dnsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		dnsWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1001, "message": "invalid zone"}},
		})
	})
	defer server.Close()

	ctx := context.Background()
	_, err := svc.Create(ctx, "A", "test.example.com", "1.2.3.4")
	if err == nil {
		t.Fatal("expected error from API")
	}
	if _, ok := err.(*R2Error); !ok {
		t.Errorf("expected *R2Error, got %T", err)
	}
}

func TestDNSCreateWithPriority(t *testing.T) {
	svc, server := dnsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["type"] != "MX" {
			t.Errorf("expected type=MX, got %v", body["type"])
		}
		priority := float64(10)
		dnsWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id": "rec-mx-001", "type": "MX", "name": "mail.example.com",
				"content": "mx1.example.com", "ttl": 3600, "priority": priority,
				"proxied": false,
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	rec, err := svc.Create(ctx, "MX", "mail.example.com", "mx1.example.com",
		WithDNSTTL(3600), WithDNSPriority(10),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.ID != "rec-mx-001" {
		t.Errorf("expected ID=rec-mx-001, got %s", rec.ID)
	}
}

func TestDNSListEmpty(t *testing.T) {
	svc, server := dnsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		dnsWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  []interface{}{},
			"result_info": map[string]interface{}{"page": 1, "total_pages": 1, "count": 0},
		})
	})
	defer server.Close()

	ctx := context.Background()
	records, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}
}

func TestDNSRecordJSONMarshal(t *testing.T) {
	p := uint16(20)
	rec := &DNSRecord{
		ID: "rec-json-001", Type: "MX", Name: "mail.example.com",
		Content: "mx.example.com", TTL: 3600, Proxied: false, Priority: &p,
		Comment: "json test", ZoneID: "zone-json", Proxiable: true,
		CreatedOn: time.Now(), ModifiedOn: time.Now(),
	}
	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded DNSRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.ID != "rec-json-001" {
		t.Errorf("ID mismatch: got %q", decoded.ID)
	}
	if decoded.Type != "MX" {
		t.Errorf("Type mismatch: got %q", decoded.Type)
	}
	if decoded.Priority == nil || *decoded.Priority != 20 {
		t.Errorf("Priority mismatch: got %v", decoded.Priority)
	}
}

func TestDNSNewFromCredsSuccess(t *testing.T) {
	svc, err := NewDNSServiceFromCreds("zone123", fmt.Sprintf("test-token-%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.zoneID != "zone123" {
		t.Errorf("expected zoneID=zone123, got %s", svc.zoneID)
	}
}
