package r2go2

import (
	"strings"
	"testing"
	"time"
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
