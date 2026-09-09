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

// ---------------------------------------------------------------------------
// matchesNameFilter
// ---------------------------------------------------------------------------

func TestMatchesNameFilter(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		pattern string
		want    bool
	}{
		// suffix glob: *suffix → domain must end with suffix
		{name: "suffix glob matches .com", domain: "example.com", pattern: "*.com", want: true},
		{name: "suffix glob no match .org vs .com", domain: "example.org", pattern: "*.com", want: false},
		{name: "suffix glob matches .dev", domain: "api.cosmo.dev", pattern: "*.dev", want: true},
		{name: "suffix glob wildcard only matches everything", domain: "anything", pattern: "*", want: true},
		{name: "suffix glob empty suffix after star", domain: "example.com", pattern: "*", want: true},

		// prefix glob: prefix* → domain must start with prefix
		{name: "prefix glob matches", domain: "cosmo.dev", pattern: "cosmo*", want: true},
		{name: "prefix glob no match", domain: "example.dev", pattern: "cosmo*", want: false},
		{name: "prefix glob single char prefix", domain: "example.com", pattern: "e*", want: true},
		{name: "prefix glob single char no match", domain: "example.com", pattern: "z*", want: false},

		// contains (no wildcard)
		{name: "contains substring match", domain: "example.com", pattern: "xamp", want: true},
		{name: "contains full exact name", domain: "example.com", pattern: "example.com", want: true},
		{name: "contains no match", domain: "example.com", pattern: "nothere", want: false},
		{name: "contains empty pattern matches all", domain: "example.com", pattern: "", want: true},
		{name: "contains dot in pattern", domain: "example.com", pattern: ".com", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesNameFilter(tt.domain, tt.pattern)
			if got != tt.want {
				t.Errorf("matchesNameFilter(%q, %q) = %v; want %v", tt.domain, tt.pattern, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// classifyNameservers
// ---------------------------------------------------------------------------

func TestClassifyNameservers(t *testing.T) {
	tests := []struct {
		name string
		ns   []string
		want string
	}{
		{name: "nil slice returns external", ns: nil, want: "external"},
		{name: "empty slice returns external", ns: []string{}, want: "external"},
		{name: "cloudflare ns returns cloudflare", ns: []string{"anita.ns.cloudflare.com", "bob.ns.cloudflare.com"}, want: "cloudflare"},
		{name: "mixed with cloudflare returns cloudflare", ns: []string{"ns1.example.com", "anita.ns.cloudflare.com"}, want: "cloudflare"},
		{name: "uppercase cloudflare returns cloudflare", ns: []string{"ANITA.NS.CLOUDFLARE.COM"}, want: "cloudflare"},
		{name: "mixed case cloudflare returns cloudflare", ns: []string{"Anita.Ns.Cloudflare.Com"}, want: "cloudflare"},
		{name: "all external returns external", ns: []string{"ns1.example.com", "ns2.example.com"}, want: "external"},
		{name: "single external returns external", ns: []string{"ns.host.io"}, want: "external"},
		{name: "cloudflare is first of two returns cloudflare", ns: []string{"anita.ns.cloudflare.com", "ns1.example.com"}, want: "cloudflare"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyNameservers(tt.ns)
			if got != tt.want {
				t.Errorf("classifyNameservers(%v) = %q; want %q", tt.ns, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// classifySSLStatus
// ---------------------------------------------------------------------------

func TestClassifySSLStatus(t *testing.T) {
	tests := []struct {
		name     string
		daysLeft int
		valid    bool
		want     string
	}{
		// invalid always → expired, regardless of daysLeft
		{name: "invalid cert with many days → expired", daysLeft: 90, valid: false, want: "expired"},
		{name: "invalid cert zero days → expired", daysLeft: 0, valid: false, want: "expired"},
		{name: "invalid cert negative days → expired", daysLeft: -5, valid: false, want: "expired"},
		// valid, boundary at 30
		{name: "valid cert exactly 30 days → valid", daysLeft: 30, valid: true, want: "valid"},
		{name: "valid cert 31 days → valid", daysLeft: 31, valid: true, want: "valid"},
		{name: "valid cert 90 days → valid", daysLeft: 90, valid: true, want: "valid"},
		{name: "valid cert 365 days → valid", daysLeft: 365, valid: true, want: "valid"},
		// valid, below 30 → expiring
		{name: "valid cert 29 days → expiring", daysLeft: 29, valid: true, want: "expiring"},
		{name: "valid cert 1 day → expiring", daysLeft: 1, valid: true, want: "expiring"},
		{name: "valid cert 0 days → expiring", daysLeft: 0, valid: true, want: "expiring"},
		{name: "valid cert negative days → expiring", daysLeft: -1, valid: true, want: "expiring"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifySSLStatus(tt.daysLeft, tt.valid)
			if got != tt.want {
				t.Errorf("classifySSLStatus(%d, %v) = %q; want %q", tt.daysLeft, tt.valid, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FormatDomainTable
// ---------------------------------------------------------------------------

func TestFormatDomainTable(t *testing.T) {
	t.Run("nil input returns sentinel string", func(t *testing.T) {
		got := FormatDomainTable(nil)
		if got != "No domains found" {
			t.Errorf("FormatDomainTable(nil) = %q; want %q", got, "No domains found")
		}
	})

	t.Run("empty slice returns sentinel string", func(t *testing.T) {
		got := FormatDomainTable([]*DomainStatus{})
		if got != "No domains found" {
			t.Errorf("FormatDomainTable([]) = %q; want %q", got, "No domains found")
		}
	})

	t.Run("output contains all header columns", func(t *testing.T) {
		domains := []*DomainStatus{
			{Zone: &Zone{Name: "example.com", Status: "active"}, NSStatus: "cloudflare"},
		}
		got := FormatDomainTable(domains)
		for _, col := range []string{"DOMAIN", "STATUS", "DNS", "SSL", "HEALTH", "RECORDS", "NS"} {
			if !strings.Contains(got, col) {
				t.Errorf("expected header column %q in output:\n%s", col, got)
			}
		}
	})

	t.Run("output contains domain data values", func(t *testing.T) {
		domains := []*DomainStatus{
			{
				Zone:         &Zone{Name: "example.com", Status: "active"},
				DNSStatus:    "ok",
				SSLStatus:    "valid",
				HealthStatus: "up",
				RecordCount:  42,
				NSStatus:     "cloudflare",
			},
		}
		got := FormatDomainTable(domains)
		for _, val := range []string{"example.com", "active", "ok", "valid", "up", "42", "cloudflare"} {
			if !strings.Contains(got, val) {
				t.Errorf("expected value %q in output:\n%s", val, got)
			}
		}
	})

	t.Run("paused zone shows paused status overriding zone status", func(t *testing.T) {
		domains := []*DomainStatus{
			{
				Zone:     &Zone{Name: "paused.com", Status: "active", Paused: true},
				NSStatus: "external",
			},
		}
		got := FormatDomainTable(domains)
		if !strings.Contains(got, "paused") {
			t.Errorf("expected 'paused' status for paused zone in output:\n%s", got)
		}
	})

	t.Run("multiple domains all appear in output", func(t *testing.T) {
		domains := []*DomainStatus{
			{Zone: &Zone{Name: "alpha.com", Status: "active"}, NSStatus: "cloudflare"},
			{Zone: &Zone{Name: "beta.org", Status: "active"}, NSStatus: "external"},
			{Zone: &Zone{Name: "gamma.io", Status: "inactive"}, NSStatus: "external"},
		}
		got := FormatDomainTable(domains)
		for _, name := range []string{"alpha.com", "beta.org", "gamma.io"} {
			if !strings.Contains(got, name) {
				t.Errorf("expected domain %q in output:\n%s", name, got)
			}
		}
	})

	t.Run("domain name longer than 25 chars is truncated with ellipsis", func(t *testing.T) {
		longName := "this-is-a-very-long-domain-name-that-exceeds-25-chars.example.com"
		domains := []*DomainStatus{
			{Zone: &Zone{Name: longName, Status: "active"}, NSStatus: "external"},
		}
		got := FormatDomainTable(domains)
		if strings.Contains(got, longName) {
			t.Errorf("expected long domain name to be truncated, but full name appears in:\n%s", got)
		}
		if !strings.Contains(got, "...") {
			t.Errorf("expected '...' truncation marker in:\n%s", got)
		}
	})

	t.Run("domain name of exactly 25 chars is not truncated", func(t *testing.T) {
		name := "1234567890123456789012345" // exactly 25 chars
		domains := []*DomainStatus{
			{Zone: &Zone{Name: name, Status: "active"}, NSStatus: "external"},
		}
		got := FormatDomainTable(domains)
		if !strings.Contains(got, name) {
			t.Errorf("25-char name should not be truncated, output:\n%s", got)
		}
	})

	t.Run("record count of zero is rendered", func(t *testing.T) {
		domains := []*DomainStatus{
			{
				Zone:        &Zone{Name: "empty.com", Status: "active"},
				RecordCount: 0,
				NSStatus:    "external",
			},
		}
		got := FormatDomainTable(domains)
		if !strings.Contains(got, "0") {
			t.Errorf("expected '0' record count in output:\n%s", got)
		}
	})
}

// ---------------------------------------------------------------------------
// FormatDomainDetail
// ---------------------------------------------------------------------------

func TestFormatDomainDetail(t *testing.T) {
	t.Run("zone name and status are always present", func(t *testing.T) {
		d := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone: &Zone{Name: "example.com", Status: "active"},
			},
		}
		got := FormatDomainDetail(d)
		if !strings.Contains(got, "example.com") {
			t.Errorf("expected zone name in output: %q", got)
		}
		if !strings.Contains(got, "active") {
			t.Errorf("expected zone status in output: %q", got)
		}
	})

	t.Run("nameservers line shown when present", func(t *testing.T) {
		d := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone: &Zone{Name: "example.com", Status: "active"},
			},
			NameServers: []string{"anita.ns.cloudflare.com", "bob.ns.cloudflare.com"},
		}
		got := FormatDomainDetail(d)
		if !strings.Contains(got, "Nameservers") {
			t.Errorf("expected 'Nameservers' label in output: %q", got)
		}
		if !strings.Contains(got, "anita.ns.cloudflare.com") {
			t.Errorf("expected first nameserver in output: %q", got)
		}
		if !strings.Contains(got, "bob.ns.cloudflare.com") {
			t.Errorf("expected second nameserver in output: %q", got)
		}
	})

	t.Run("nameservers line absent when nil", func(t *testing.T) {
		d := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone: &Zone{Name: "example.com", Status: "active"},
			},
			NameServers: nil,
		}
		got := FormatDomainDetail(d)
		if strings.Contains(got, "Nameservers") {
			t.Errorf("expected no 'Nameservers' line when nil, got: %q", got)
		}
	})

	t.Run("nameservers line absent when empty slice", func(t *testing.T) {
		d := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone: &Zone{Name: "example.com", Status: "active"},
			},
			NameServers: []string{},
		}
		got := FormatDomainDetail(d)
		if strings.Contains(got, "Nameservers") {
			t.Errorf("expected no 'Nameservers' line when empty slice, got: %q", got)
		}
	})

	t.Run("records line shows count and types when map has entries", func(t *testing.T) {
		d := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone:        &Zone{Name: "example.com", Status: "active"},
				RecordCount: 15,
			},
			RecordTypes: map[string]int{"A": 10, "CNAME": 5},
		}
		got := FormatDomainDetail(d)
		if !strings.Contains(got, "Records") {
			t.Errorf("expected 'Records' label in output: %q", got)
		}
		if !strings.Contains(got, "15") {
			t.Errorf("expected total record count '15' in output: %q", got)
		}
		if !strings.Contains(got, "A") {
			t.Errorf("expected record type 'A' in output: %q", got)
		}
		if !strings.Contains(got, "CNAME") {
			t.Errorf("expected record type 'CNAME' in output: %q", got)
		}
	})

	t.Run("records line absent when map is empty", func(t *testing.T) {
		d := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone: &Zone{Name: "example.com", Status: "active"},
			},
			RecordTypes: map[string]int{},
		}
		got := FormatDomainDetail(d)
		if strings.Contains(got, "Records") {
			t.Errorf("expected no 'Records' line when map is empty, got: %q", got)
		}
	})

	t.Run("records line absent when map is nil", func(t *testing.T) {
		d := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone: &Zone{Name: "example.com", Status: "active"},
			},
			RecordTypes: nil,
		}
		got := FormatDomainDetail(d)
		if strings.Contains(got, "Records") {
			t.Errorf("expected no 'Records' line when map is nil, got: %q", got)
		}
	})

	t.Run("SSL line shown when SSLMode set", func(t *testing.T) {
		d := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone: &Zone{Name: "example.com", Status: "active"},
			},
			SSLMode: "full_strict",
		}
		got := FormatDomainDetail(d)
		if !strings.Contains(got, "SSL") {
			t.Errorf("expected 'SSL' label in output: %q", got)
		}
		if !strings.Contains(got, "full_strict") {
			t.Errorf("expected SSL mode value in output: %q", got)
		}
	})

	t.Run("SSL line absent when SSLMode is empty", func(t *testing.T) {
		d := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone: &Zone{Name: "example.com", Status: "active"},
			},
			SSLMode: "",
		}
		got := FormatDomainDetail(d)
		if strings.Contains(got, "SSL:") {
			t.Errorf("expected no SSL line when mode empty, got: %q", got)
		}
	})

	t.Run("SSL expiry date rendered when set", func(t *testing.T) {
		d := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone: &Zone{Name: "example.com", Status: "active"},
			},
			SSLMode:   "full",
			SSLExpiry: "2099-12-31",
		}
		got := FormatDomainDetail(d)
		if !strings.Contains(got, "2099-12-31") {
			t.Errorf("expected SSL expiry date '2099-12-31' in output: %q", got)
		}
		if !strings.Contains(got, "expires") {
			t.Errorf("expected 'expires' text in output: %q", got)
		}
	})

	t.Run("SSL line without expiry when SSLExpiry empty", func(t *testing.T) {
		d := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone: &Zone{Name: "example.com", Status: "active"},
			},
			SSLMode:   "flexible",
			SSLExpiry: "",
		}
		got := FormatDomainDetail(d)
		if !strings.Contains(got, "flexible") {
			t.Errorf("expected SSL mode 'flexible' in output: %q", got)
		}
		if strings.Contains(got, "expires") {
			t.Errorf("expected no 'expires' text when SSLExpiry empty, got: %q", got)
		}
	})

	t.Run("health line shown when ResponseTime set", func(t *testing.T) {
		d := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone:         &Zone{Name: "example.com", Status: "active"},
				HealthStatus: "up",
			},
			ResponseTime: "42ms",
		}
		got := FormatDomainDetail(d)
		if !strings.Contains(got, "Health") {
			t.Errorf("expected 'Health' label in output: %q", got)
		}
		if !strings.Contains(got, "up") {
			t.Errorf("expected health status 'up' in output: %q", got)
		}
		if !strings.Contains(got, "42ms") {
			t.Errorf("expected response time '42ms' in output: %q", got)
		}
	})

	t.Run("health line absent when ResponseTime empty", func(t *testing.T) {
		d := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone:         &Zone{Name: "example.com", Status: "active"},
				HealthStatus: "unknown",
			},
			ResponseTime: "",
		}
		got := FormatDomainDetail(d)
		if strings.Contains(got, "Health:") {
			t.Errorf("expected no Health line when ResponseTime empty, got: %q", got)
		}
	})

	t.Run("full detail with all fields populates complete output", func(t *testing.T) {
		d := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone:         &Zone{Name: "full.example.com", Status: "active"},
				HealthStatus: "up",
				RecordCount:  8,
			},
			NameServers:  []string{"anna.ns.cloudflare.com", "bob.ns.cloudflare.com"},
			RecordTypes:  map[string]int{"A": 5, "CNAME": 3},
			SSLMode:      "full",
			SSLExpiry:    time.Now().AddDate(0, 3, 0).Format("2006-01-02"),
			ResponseTime: "142ms",
		}
		got := FormatDomainDetail(d)
		for _, want := range []string{
			"full.example.com", "active",
			"Nameservers", "anna.ns.cloudflare.com",
			"Records", "8",
			"SSL", "full",
			"Health", "up", "142ms",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("expected %q in full detail output:\n%s", want, got)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// NewDomainService
// ---------------------------------------------------------------------------

func TestNewDomainService(t *testing.T) {
	t.Run("nil zones returns error and nil service", func(t *testing.T) {
		svc, err := NewDomainService(nil, nil, nil, nil)
		if err == nil {
			t.Error("NewDomainService(nil zones) expected error, got nil")
		}
		if svc != nil {
			t.Error("NewDomainService(nil zones) expected nil service, got non-nil")
		}
	})

	t.Run("non-nil zones with all optionals nil returns service", func(t *testing.T) {
		zoneSvc := &ZoneService{}
		svc, err := NewDomainService(zoneSvc, nil, nil, nil)
		if err != nil {
			t.Fatalf("NewDomainService(zoneSvc, nil, nil, nil) unexpected error: %v", err)
		}
		if svc == nil {
			t.Fatal("NewDomainService returned nil service unexpectedly")
		}
		if svc.zones != zoneSvc {
			t.Error("svc.zones does not match provided ZoneService")
		}
		if svc.ssl != nil {
			t.Error("expected svc.ssl to be nil")
		}
		if svc.dns != nil {
			t.Error("expected svc.dns to be nil")
		}
		if svc.doctor != nil {
			t.Error("expected svc.doctor to be nil")
		}
	})

	t.Run("all optional services are stored when provided", func(t *testing.T) {
		zoneSvc := &ZoneService{}
		sslSvc := &SSLService{}
		dnsSvc := &DNSService{}
		doctorSvc := &DoctorService{}

		svc, err := NewDomainService(zoneSvc, sslSvc, dnsSvc, doctorSvc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svc.zones != zoneSvc {
			t.Error("svc.zones mismatch")
		}
		if svc.ssl != sslSvc {
			t.Error("svc.ssl mismatch")
		}
		if svc.dns != dnsSvc {
			t.Error("svc.dns mismatch")
		}
		if svc.doctor != doctorSvc {
			t.Error("svc.doctor mismatch")
		}
	})

	t.Run("only zones provided, ssl nil, returns service with nil ssl", func(t *testing.T) {
		zoneSvc := &ZoneService{}
		svc, err := NewDomainService(zoneSvc, nil, &DNSService{}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svc.ssl != nil {
			t.Error("expected svc.ssl to be nil")
		}
		if svc.dns == nil {
			t.Error("expected svc.dns to be non-nil")
		}
	})
}

// ---------------------------------------------------------------------------
// DomainStatus field tests
// ---------------------------------------------------------------------------

func TestDomainStatusFields(t *testing.T) {
	t.Run("all fields populate correctly", func(t *testing.T) {
		zone := &Zone{Name: "example.com", Status: "active"}
		ds := &DomainStatus{
			Zone:         zone,
			DNSStatus:    "ok",
			SSLStatus:    "valid",
			HealthStatus: "up",
			RecordCount:  15,
			NSStatus:     "cloudflare",
		}
		if ds.Zone.Name != "example.com" {
			t.Errorf("expected zone name example.com, got %s", ds.Zone.Name)
		}
		if ds.DNSStatus != "ok" {
			t.Errorf("expected dns_status ok, got %s", ds.DNSStatus)
		}
		if ds.SSLStatus != "valid" {
			t.Errorf("expected ssl_status valid, got %s", ds.SSLStatus)
		}
		if ds.HealthStatus != "up" {
			t.Errorf("expected health_status up, got %s", ds.HealthStatus)
		}
		if ds.RecordCount != 15 {
			t.Errorf("expected record_count 15, got %d", ds.RecordCount)
		}
		if ds.NSStatus != "cloudflare" {
			t.Errorf("expected ns_status cloudflare, got %s", ds.NSStatus)
		}
	})

	t.Run("zero values for optional fields", func(t *testing.T) {
		ds := &DomainStatus{
			Zone: &Zone{Name: "minimal.com", Status: "inactive"},
		}
		if ds.DNSStatus != "" {
			t.Errorf("expected empty dns_status, got %q", ds.DNSStatus)
		}
		if ds.RecordCount != 0 {
			t.Errorf("expected 0 record_count, got %d", ds.RecordCount)
		}
	})
}

// ---------------------------------------------------------------------------
// DomainStatus JSON serialization
// ---------------------------------------------------------------------------

func TestDomainStatusJSONSerialization(t *testing.T) {
	t.Run("round-trip preserves all fields", func(t *testing.T) {
		original := &DomainStatus{
			Zone:         &Zone{Name: "serial.dev", Status: "active"},
			DNSStatus:    "warn",
			SSLStatus:    "expiring",
			HealthStatus: "down",
			RecordCount:  7,
			NSStatus:     "external",
		}
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var decoded DomainStatus
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if decoded.Zone.Name != "serial.dev" {
			t.Errorf("decoded zone name = %q, want serial.dev", decoded.Zone.Name)
		}
		if decoded.DNSStatus != "warn" {
			t.Errorf("decoded dns_status = %q, want warn", decoded.DNSStatus)
		}
		if decoded.SSLStatus != "expiring" {
			t.Errorf("decoded ssl_status = %q, want expiring", decoded.SSLStatus)
		}
		if decoded.HealthStatus != "down" {
			t.Errorf("decoded health_status = %q, want down", decoded.HealthStatus)
		}
		if decoded.RecordCount != 7 {
			t.Errorf("decoded record_count = %d, want 7", decoded.RecordCount)
		}
		if decoded.NSStatus != "external" {
			t.Errorf("decoded ns_status = %q, want external", decoded.NSStatus)
		}
	})

	t.Run("JSON field names use snake_case", func(t *testing.T) {
		ds := &DomainStatus{
			Zone:         &Zone{Name: "test.com", Status: "active"},
			DNSStatus:    "ok",
			SSLStatus:    "valid",
			HealthStatus: "up",
			RecordCount:  3,
			NSStatus:     "cloudflare",
		}
		data, err := json.Marshal(ds)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		jsonStr := string(data)
		for _, field := range []string{`"dns_status"`, `"ssl_status"`, `"health_status"`, `"record_count"`, `"ns_status"`} {
			if !strings.Contains(jsonStr, field) {
				t.Errorf("expected JSON field %s in output: %s", field, jsonStr)
			}
		}
	})
}

func TestDomainDetailJSONSerialization(t *testing.T) {
	t.Run("round-trip preserves extended fields", func(t *testing.T) {
		original := &DomainDetail{
			DomainStatus: DomainStatus{
				Zone:         &Zone{Name: "detail.io", Status: "active"},
				RecordCount:  10,
				HealthStatus: "up",
			},
			NameServers:  []string{"ns1.cf.com", "ns2.cf.com"},
			RecordTypes:  map[string]int{"A": 5, "MX": 3, "TXT": 2},
			SSLMode:      "full_strict",
			SSLExpiry:    "2099-06-15",
			ResponseTime: "89ms",
		}
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var decoded DomainDetail
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if len(decoded.NameServers) != 2 {
			t.Fatalf("expected 2 nameservers, got %d", len(decoded.NameServers))
		}
		if decoded.NameServers[0] != "ns1.cf.com" {
			t.Errorf("expected first ns ns1.cf.com, got %s", decoded.NameServers[0])
		}
		if decoded.RecordTypes["A"] != 5 {
			t.Errorf("expected A=5, got %d", decoded.RecordTypes["A"])
		}
		if decoded.SSLMode != "full_strict" {
			t.Errorf("expected ssl_mode full_strict, got %s", decoded.SSLMode)
		}
		if decoded.ResponseTime != "89ms" {
			t.Errorf("expected response_time 89ms, got %s", decoded.ResponseTime)
		}
	})
}

// ---------------------------------------------------------------------------
// Empty domain results
// ---------------------------------------------------------------------------

func TestDomainListOptionsDefaults(t *testing.T) {
	t.Run("zero-value options", func(t *testing.T) {
		opts := DomainListOptions{}
		if opts.Page != 0 {
			t.Errorf("expected default page 0, got %d", opts.Page)
		}
		if opts.PerPage != 0 {
			t.Errorf("expected default per_page 0, got %d", opts.PerPage)
		}
		if opts.Filter != "" {
			t.Errorf("expected empty filter, got %q", opts.Filter)
		}
		if opts.Name != "" {
			t.Errorf("expected empty name, got %q", opts.Name)
		}
		if opts.Sort != "" {
			t.Errorf("expected empty sort, got %q", opts.Sort)
		}
	})

	t.Run("JSON round-trip", func(t *testing.T) {
		opts := DomainListOptions{
			Page:    2,
			PerPage: 25,
			Filter:  "active",
			Name:    "*.dev",
			Sort:    "name",
		}
		data, err := json.Marshal(opts)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		var decoded DomainListOptions
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if decoded.Page != 2 || decoded.PerPage != 25 || decoded.Filter != "active" || decoded.Name != "*.dev" || decoded.Sort != "name" {
			t.Errorf("round-trip mismatch: %+v", decoded)
		}
	})
}

func TestPaginationFields(t *testing.T) {
	p := Pagination{Page: 3, PerPage: 20, Total: 100, TotalPages: 5}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded Pagination
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Page != 3 || decoded.PerPage != 20 || decoded.Total != 100 || decoded.TotalPages != 5 {
		t.Errorf("pagination round-trip mismatch: %+v", decoded)
	}
}

// -----------------------------------------------------------------------------
// P-03: DomainService enrichment + summary helpers
// -----------------------------------------------------------------------------

func TestSummarizeDomains_Attention(t *testing.T) {
	ds := []*DomainStatus{
		{Zone: &Zone{Name: "ok.com"}, NSStatus: "cloudflare", SSLStatus: "valid", HealthStatus: "up"},
		{Zone: &Zone{Name: "badns.com"}, NSStatus: "external", SSLStatus: "valid"},
		{Zone: &Zone{Name: "expirssl.com"}, NSStatus: "cloudflare", SSLStatus: "expired"},
	}
	summary := SummarizeDomains(ds)

	if summary.Total != 3 {
		t.Fatalf("Total = %d, want 3", summary.Total)
	}
	if summary.NeedsAttention != 2 { // badns (external NS) + expirssl (expired SSL)
		t.Fatalf("NeedsAttention = %d, want 2", summary.NeedsAttention)
	}
	if summary.ByNSStatus["cloudflare"] != 2 || summary.ByNSStatus["external"] != 1 {
		t.Errorf("ByNSStatus = %+v", summary.ByNSStatus)
	}
	if len(summary.Attention) != 2 {
		t.Errorf("Attention = %v, want 2 entries", summary.Attention)
	}
}

func TestSummarizeDomains_SkipsNil(t *testing.T) {
	summary := SummarizeDomains([]*DomainStatus{nil, {Zone: &Zone{Name: "a.com"}, NSStatus: "cloudflare", SSLStatus: "valid"}})
	if summary.Total != 1 {
		t.Errorf("Total = %d, want 1 (nil entry skipped)", summary.Total)
	}
}

func TestRegistrarStatusFor(t *testing.T) {
	reg := map[string]RegistrarInfo{"a.com": {Registrar: "cloudflare"}}
	if got := classifyRegistrarStatus("a.com", reg); got != "cloudflare" {
		t.Errorf("a.com = %q, want cloudflare", got)
	}
	if got := classifyRegistrarStatus("b.com", reg); got != "external" {
		t.Errorf("b.com = %q, want external", got)
	}
	if got := classifyRegistrarStatus("x.com", nil); got != "external" {
		t.Errorf("nil map = %q, want external", got)
	}
}

func TestDomainServiceEnrichmentWiring(t *testing.T) {
	svc, err := NewDomainService(&ZoneService{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewDomainService: %v", err)
	}
	if svc.redirects != nil || svc.registrar != nil {
		t.Fatal("expected redirects/registrar unset on a fresh service")
	}

	rs := NewRedirectService(nil, "acct-1")
	rg := NewRegistrarService(nil, "acct-1")
	if got := svc.WithRedirects(rs).WithRegistrar(rg); got != svc {
		t.Error("setters should return the receiver for fluent chaining")
	}
	if svc.redirects != rs {
		t.Error("WithRedirects did not wire the RedirectService")
	}
	if svc.registrar != rg {
		t.Error("WithRegistrar did not wire the RegistrarService")
	}
}

// Suppress unused import warning
var _ = time.Now

func TestClassifyRedirectIssue(t *testing.T) {
	tests := []struct {
		name    string
		results []RedirectProbeResult
		want    string
	}{
		{"empty", nil, ""},
		{"all clean", []RedirectProbeResult{{Status: 200}}, ""},
		{"skipped only", []RedirectProbeResult{{Skipped: true}}, ""},
		{"4xx", []RedirectProbeResult{{Status: 200}, {Status: 404}}, "http-4xx"},
		{"5xx", []RedirectProbeResult{{Status: 503}}, "http-5xx"},
		{"loop", []RedirectProbeResult{{Status: 404}, {Loop: true}}, "loop"},
		{"unreachable", []RedirectProbeResult{{Err: "timeout"}}, "unreachable"},
		{"loop beats 5xx", []RedirectProbeResult{{Status: 500}, {Loop: true}}, "loop"},
		{"5xx beats 4xx", []RedirectProbeResult{{Status: 404}, {Status: 500}}, "http-5xx"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyRedirectIssue(tt.results); got != tt.want {
				t.Fatalf("classifyRedirectIssue = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDomainNeedsAttentionRedirectIssue(t *testing.T) {
	d := &DomainStatus{NSStatus: "cloudflare", SSLStatus: "valid", HealthStatus: "up"}
	if domainNeedsAttention(d) {
		t.Fatal("clean domain must not need attention")
	}
	d.RedirectIssue = "loop"
	if !domainNeedsAttention(d) {
		t.Fatal("RedirectIssue must trigger attention")
	}
}

// ---------------------------------------------------------------------------
// GetDetail redirect enrichment (modern + legacy Page Rules)
// ---------------------------------------------------------------------------

// newDetailTestService builds a zones-backed DomainService over an httptest
// server answering GET /zones/z1 with the "example.com" zone. ssl/dns/doctor
// are nil so GetDetail exercises only the zone + redirect paths.
func newDetailTestService(t *testing.T) *DomainService {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/zones/z1" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"errors":  []interface{}{},
				"result": map[string]interface{}{
					"id":           "z1",
					"name":         "example.com",
					"status":       "active",
					"paused":       false,
					"name_servers": []string{"ns1.cloudflare.com", "ns2.cloudflare.com"},
				},
			})
			return
		}
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	cf, err := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to build cloudflare client: %v", err)
	}
	zones, err := NewZoneService(cf, "acct-test-123")
	if err != nil {
		t.Fatalf("NewZoneService: %v", err)
	}
	svc, err := NewDomainService(zones, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewDomainService: %v", err)
	}
	return svc
}

// newFakeRedirectService builds a RedirectService whose httptest server serves
// the /zones/z1/rulesets list + phase-ruleset GET pair (TestRedirectService_List
// handler shape), returning the given rules from the redirect phase.
func newFakeRedirectService(rules []RedirectRule) *RedirectService {
	encoded := make([]map[string]interface{}, 0, len(rules))
	for _, r := range rules {
		encoded = append(encoded, map[string]interface{}{
			"id":         r.ID,
			"expression": r.When,
			"enabled":    r.Enabled,
			"action":     "redirect",
			"action_parameters": map[string]interface{}{
				"from_value": map[string]interface{}{
					"status_code": r.StatusCode,
					"target_url":  map[string]interface{}{"value": r.Destination},
				},
			},
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones/z1/rulesets":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":  true,
				"errors":   []interface{}{},
				"messages": []interface{}{},
				"result": []map[string]interface{}{
					{"id": "redirect-rs", "phase": "http_request_dynamic_redirect", "name": "Redirect Rules"},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/zones/z1/rulesets/redirect-rs":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":  true,
				"errors":   []interface{}{},
				"messages": []interface{}{},
				"result": map[string]interface{}{
					"id":    "redirect-rs",
					"phase": "http_request_dynamic_redirect",
					"name":  "Redirect Rules",
					"rules": encoded,
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	return NewRedirectService(cf, "account-test")
}

// fakePageRuleLister is a static PageRuleLister for tests.
type fakePageRuleLister struct {
	rules []*PageRule
	err   error
}

func (f fakePageRuleLister) List(ctx context.Context) ([]*PageRule, error) {
	return f.rules, f.err
}

func TestGetDetailMergesLegacyPageRules(t *testing.T) {
	// Zones-only service (ssl/dns/doctor nil) serving one zone "z1"/"example.com",
	// then wire both redirect sources.
	svc := newDetailTestService(t)

	svc = svc.
		WithRedirects(newFakeRedirectService([]RedirectRule{
			{ID: "r1", ZoneID: "z1", When: "starts_with(\"/a\")", Destination: "https://example.com/b", Enabled: true},
		})).
		WithPageRules(func(zoneID string) PageRuleLister {
			return fakePageRuleLister{rules: []*PageRule{{
				ID: "pr1", Status: "active",
				Targets: []PageRuleTarget{{Constraint: PageRuleConstraint{Value: "*example.com/old/*"}}},
				Actions: []PageRuleAction{{ID: "forwarding_url", Value: map[string]interface{}{"url": "https://example.com/new", "status_code": float64(302)}}},
			}}}
		})

	detail, err := svc.GetDetail(context.Background(), "z1")
	if err != nil {
		t.Fatalf("GetDetail: %v", err)
	}
	if len(detail.Redirects) != 2 {
		t.Fatalf("merged redirects = %d, want 2 (modern + legacy): %+v", len(detail.Redirects), detail.Redirects)
	}
	if detail.Redirects[0].Source != "" || detail.Redirects[0].ID != "r1" {
		t.Fatalf("modern rule must come first with empty Source: %+v", detail.Redirects[0])
	}
	if detail.Redirects[1].Source != "pagerules" || detail.Redirects[1].Destination != "https://example.com/new" {
		t.Fatalf("legacy rule wrong: %+v", detail.Redirects[1])
	}
}

func TestGetDetailPageRuleFactoryFailureSkipsLegacy(t *testing.T) {
	svc := newDetailTestService(t).
		WithRedirects(newFakeRedirectService([]RedirectRule{{ID: "r1", ZoneID: "z1"}})).
		WithPageRules(func(zoneID string) PageRuleLister { return nil }) // nil lister → skip

	detail, err := svc.GetDetail(context.Background(), "z1")
	if err != nil {
		t.Fatalf("GetDetail: %v", err)
	}
	if len(detail.Redirects) != 1 {
		t.Fatalf("nil lister must skip legacy, got %d: %+v", len(detail.Redirects), detail.Redirects)
	}
}
