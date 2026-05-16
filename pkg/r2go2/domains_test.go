package r2go2

import (
	"testing"
	"time"
)

func TestNewDomainServiceValidation(t *testing.T) {
	_, err := NewDomainService(nil, nil, nil, nil)
	if err == nil {
		t.Error("expected error when zone service is nil")
	}
}

func TestDomainListOptionsDefaults(t *testing.T) {
	opts := DomainListOptions{}
	if opts.Page != 0 {
		t.Errorf("expected default Page=0, got %d", opts.Page)
	}
	if opts.PerPage != 0 {
		t.Errorf("expected default PerPage=0, got %d", opts.PerPage)
	}
}

func TestClassifyNameservers(t *testing.T) {
	tests := []struct {
		name     string
		ns       []string
		expected string
	}{
		{"empty", nil, "external"},
		{"cloudflare ns", []string{"anna.ns.cloudflare.com", "bob.ns.cloudflare.com"}, "cloudflare"},
		{"external ns", []string{"ns1.example.com", "ns2.example.com"}, "external"},
		{"mixed case", []string{"anna.ns.CLOUDFLARE.com"}, "cloudflare"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyNameservers(tt.ns)
			if got != tt.expected {
				t.Errorf("classifyNameservers(%v) = %q, want %q", tt.ns, got, tt.expected)
			}
		})
	}
}

func TestClassifySSLStatus(t *testing.T) {
	tests := []struct {
		name     string
		daysLeft int
		valid    bool
		expected string
	}{
		{"valid long", 90, true, "valid"},
		{"expiring soon", 15, true, "expiring"},
		{"expiring boundary", 29, true, "expiring"},
		{"valid boundary", 30, true, "valid"},
		{"expired", -5, false, "expired"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifySSLStatus(tt.daysLeft, tt.valid)
			if got != tt.expected {
				t.Errorf("classifySSLStatus(%d, %v) = %q, want %q", tt.daysLeft, tt.valid, got, tt.expected)
			}
		})
	}
}

func TestMatchesNameFilter(t *testing.T) {
	tests := []struct {
		name    string
		domain  string
		pattern string
		match   bool
	}{
		{"exact substring", "example.com", "example", true},
		{"no match", "example.com", "foobar", false},
		{"suffix glob", "example.com", "*.com", true},
		{"suffix glob no match", "example.com", "*.org", false},
		{"prefix glob", "example.com", "example*", true},
		{"prefix glob no match", "example.com", "test*", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesNameFilter(tt.domain, tt.pattern)
			if got != tt.match {
				t.Errorf("matchesNameFilter(%q, %q) = %v, want %v", tt.domain, tt.pattern, got, tt.match)
			}
		})
	}
}

func TestPagination(t *testing.T) {
	p := &Pagination{
		Page:       2,
		PerPage:    25,
		Total:      127,
		TotalPages: 6,
	}
	if p.Page != 2 {
		t.Errorf("expected Page=2, got %d", p.Page)
	}
	if p.TotalPages != 6 {
		t.Errorf("expected TotalPages=6, got %d", p.TotalPages)
	}
}

func TestFormatDomainTableEmpty(t *testing.T) {
	result := FormatDomainTable(nil)
	if result != "No domains found" {
		t.Errorf("expected 'No domains found', got %q", result)
	}
}

func TestFormatDomainTable(t *testing.T) {
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
	result := FormatDomainTable(domains)
	if result == "" {
		t.Error("expected non-empty table output")
	}
	if !contains(result, "example.com") {
		t.Error("expected table to contain domain name")
	}
	if !contains(result, "DOMAIN") {
		t.Error("expected table to contain header")
	}
}

func TestFormatDomainDetail(t *testing.T) {
	detail := &DomainDetail{
		DomainStatus: DomainStatus{
			Zone:         &Zone{Name: "example.com", Status: "active"},
			HealthStatus: "up",
			RecordCount:  8,
		},
		NameServers:  []string{"anna.ns.cloudflare.com", "bob.ns.cloudflare.com"},
		RecordTypes:  map[string]int{"A": 5, "CNAME": 3},
		SSLMode:      "full",
		SSLExpiry:    time.Now().AddDate(0, 3, 0).Format("2006-01-02"),
		ResponseTime: "142ms",
	}
	result := FormatDomainDetail(detail)
	if !contains(result, "example.com") {
		t.Error("expected detail to contain domain name")
	}
	if !contains(result, "anna.ns.cloudflare.com") {
		t.Error("expected detail to contain nameserver")
	}
	if !contains(result, "full") {
		t.Error("expected detail to contain SSL mode")
	}
	if !contains(result, "142ms") {
		t.Error("expected detail to contain response time")
	}
}

func TestDomainStatusTypes(t *testing.T) {
	ds := &DomainStatus{
		Zone:         &Zone{Name: "test.com"},
		DNSStatus:    "ok",
		SSLStatus:    "valid",
		HealthStatus: "up",
		RecordCount:  10,
		NSStatus:     "cloudflare",
	}
	if ds.Zone.Name != "test.com" {
		t.Errorf("expected zone name test.com, got %s", ds.Zone.Name)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
