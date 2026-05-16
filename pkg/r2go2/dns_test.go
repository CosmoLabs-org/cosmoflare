package r2go2

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

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
