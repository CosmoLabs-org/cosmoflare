package cosmoflare

import (
	"context"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// NewDoctorService
// ---------------------------------------------------------------------------

func TestDoctorService_NewDefault(t *testing.T) {
	svc := NewDoctorService(0)
	if svc == nil {
		t.Fatal("expected non-nil DoctorService")
	}
	if svc.timeout != 10*time.Second {
		t.Errorf("expected default timeout 10s, got %v", svc.timeout)
	}
	if svc.httpClient == nil {
		t.Fatal("expected non-nil httpClient")
	}
}

func TestDoctorService_NewCustomTimeout(t *testing.T) {
	svc := NewDoctorService(5 * time.Second)
	if svc.timeout != 5*time.Second {
		t.Errorf("expected timeout 5s, got %v", svc.timeout)
	}
}

// ---------------------------------------------------------------------------
// Validation — empty domain returns error
// ---------------------------------------------------------------------------

func TestDoctorService_CheckDNSPropagation_EmptyDomain(t *testing.T) {
	svc := NewDoctorService(0)
	_, err := svc.CheckDNSPropagation(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty domain")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Errorf("expected R2ValidationError, got %T", err)
	}
}

func TestDoctorService_CheckSSL_EmptyDomain(t *testing.T) {
	svc := NewDoctorService(0)
	_, err := svc.CheckSSL(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty domain")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Errorf("expected R2ValidationError, got %T", err)
	}
}

func TestDoctorService_CheckHTTP_EmptyDomain(t *testing.T) {
	svc := NewDoctorService(0)
	_, err := svc.CheckHTTP(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty domain")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Errorf("expected R2ValidationError, got %T", err)
	}
}

func TestDoctorService_CheckNameservers_EmptyDomain(t *testing.T) {
	svc := NewDoctorService(0)
	_, err := svc.CheckNameservers(context.Background(), "", []string{"ns1.example.com"})
	if err == nil {
		t.Fatal("expected error for empty domain")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Errorf("expected R2ValidationError, got %T", err)
	}
}

func TestDoctorService_CheckNameservers_EmptyExpected(t *testing.T) {
	svc := NewDoctorService(0)
	_, err := svc.CheckNameservers(context.Background(), "example.com", nil)
	if err == nil {
		t.Fatal("expected error for empty expected list")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Errorf("expected R2ValidationError, got %T", err)
	}
}

func TestDoctorService_RunDiagnostics_EmptyDomain(t *testing.T) {
	svc := NewDoctorService(0)
	_, err := svc.RunDiagnostics(context.Background(), "", nil)
	if err == nil {
		t.Fatal("expected error for empty domain")
	}
	if _, ok := err.(*R2ValidationError); !ok {
		t.Errorf("expected R2ValidationError, got %T", err)
	}
}

// ---------------------------------------------------------------------------
// compareNameservers — unit tests for NS comparison logic
// ---------------------------------------------------------------------------

func TestDoctor_CompareNameservers_Match(t *testing.T) {
	expected := []string{"ns1.example.com", "ns2.example.com"}
	actual := []string{"ns1.example.com", "ns2.example.com"}
	if !compareNameservers(expected, actual) {
		t.Error("expected match for identical lists")
	}
}

func TestDoctor_CompareNameservers_MatchUnordered(t *testing.T) {
	expected := []string{"ns2.example.com", "ns1.example.com"}
	actual := []string{"ns1.example.com", "ns2.example.com"}
	if !compareNameservers(expected, actual) {
		t.Error("expected match for unordered identical lists")
	}
}

func TestDoctor_CompareNameservers_CaseInsensitive(t *testing.T) {
	expected := []string{"NS1.EXAMPLE.COM", "ns2.example.com"}
	actual := []string{"ns1.example.com", "NS2.EXAMPLE.COM"}
	if !compareNameservers(expected, actual) {
		t.Error("expected case-insensitive match")
	}
}

func TestDoctor_CompareNameservers_TrailingDot(t *testing.T) {
	expected := []string{"ns1.example.com.", "ns2.example.com."}
	actual := []string{"ns1.example.com", "ns2.example.com"}
	if !compareNameservers(expected, actual) {
		t.Error("expected match when trailing dots differ")
	}
}

func TestDoctor_CompareNameservers_MixedDotsAndCase(t *testing.T) {
	expected := []string{"NS1.Example.Com.", "NS2.Example.Com"}
	actual := []string{"ns1.example.com", "ns2.example.com."}
	if !compareNameservers(expected, actual) {
		t.Error("expected match with mixed case and trailing dots")
	}
}

func TestDoctor_CompareNameservers_Mismatch(t *testing.T) {
	expected := []string{"ns1.example.com", "ns2.example.com"}
	actual := []string{"ns1.different.com", "ns2.different.com"}
	if compareNameservers(expected, actual) {
		t.Error("expected mismatch for different NS lists")
	}
}

func TestDoctor_CompareNameservers_DifferentLengths(t *testing.T) {
	expected := []string{"ns1.example.com", "ns2.example.com"}
	actual := []string{"ns1.example.com"}
	if compareNameservers(expected, actual) {
		t.Error("expected mismatch when list lengths differ")
	}
}

func TestDoctor_CompareNameservers_PartialOverlap(t *testing.T) {
	expected := []string{"ns1.example.com", "ns2.example.com"}
	actual := []string{"ns1.example.com", "ns3.example.com"}
	if compareNameservers(expected, actual) {
		t.Error("expected mismatch for partially overlapping lists")
	}
}

// ---------------------------------------------------------------------------
// DiagnosticIssue construction & scoring
// ---------------------------------------------------------------------------

func TestDoctor_ComputeScore_Healthy(t *testing.T) {
	issues := []DiagnosticIssue{
		{Severity: "info", Message: "informational only"},
	}
	if score := computeScore(issues); score != "healthy" {
		t.Errorf("expected healthy, got %s", score)
	}
}

func TestDoctor_ComputeScore_HealthyEmpty(t *testing.T) {
	if score := computeScore(nil); score != "healthy" {
		t.Errorf("expected healthy for nil issues, got %s", score)
	}
}

func TestDoctor_ComputeScore_Warning(t *testing.T) {
	issues := []DiagnosticIssue{
		{Severity: "info", Message: "ok"},
		{Severity: "warning", Message: "something off"},
	}
	if score := computeScore(issues); score != "warning" {
		t.Errorf("expected warning, got %s", score)
	}
}

func TestDoctor_ComputeScore_Critical(t *testing.T) {
	issues := []DiagnosticIssue{
		{Severity: "warning", Message: "minor"},
		{Severity: "critical", Message: "major"},
	}
	if score := computeScore(issues); score != "critical" {
		t.Errorf("expected critical, got %s", score)
	}
}

// ---------------------------------------------------------------------------
// analyzeIssues — unit tests via synthetic reports
// ---------------------------------------------------------------------------

func TestDoctor_AnalyzeIssues_DNSInconsistent(t *testing.T) {
	svc := NewDoctorService(0)
	report := &DiagnosticReport{
		DNS: &DNSPropagationResult{Consistent: false},
		SSL: &SSLProbeResult{Valid: true, DaysLeft: 90, TLSVersion: "TLS 1.3"},
		HTTP: &HTTPProbeResult{StatusCode: 200, CloudflareRay: "abc123"},
	}
	issues := svc.analyzeIssues(report)
	found := false
	for _, iss := range issues {
		if iss.Probe == "dns" && iss.Severity == "warning" {
			found = true
		}
	}
	if !found {
		t.Error("expected DNS warning for inconsistent records")
	}
}

func TestDoctor_AnalyzeIssues_SSLExpired(t *testing.T) {
	svc := NewDoctorService(0)
	report := &DiagnosticReport{
		DNS: &DNSPropagationResult{Consistent: true},
		SSL: &SSLProbeResult{Valid: false, DaysLeft: -5, TLSVersion: "TLS 1.2"},
		HTTP: &HTTPProbeResult{StatusCode: 200, CloudflareRay: "abc123"},
	}
	issues := svc.analyzeIssues(report)
	found := false
	for _, iss := range issues {
		if iss.Probe == "ssl" && iss.Severity == "critical" {
			found = true
		}
	}
	if !found {
		t.Error("expected critical SSL issue for expired cert")
	}
}

func TestDoctor_AnalyzeIssues_SSLExpiringSoon(t *testing.T) {
	svc := NewDoctorService(0)
	report := &DiagnosticReport{
		DNS: &DNSPropagationResult{Consistent: true},
		SSL: &SSLProbeResult{Valid: true, DaysLeft: 15, TLSVersion: "TLS 1.3"},
		HTTP: &HTTPProbeResult{StatusCode: 200, CloudflareRay: "abc123"},
	}
	issues := svc.analyzeIssues(report)
	found := false
	for _, iss := range issues {
		if iss.Probe == "ssl" && iss.Severity == "warning" && iss.Message != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for cert expiring in <30 days")
	}
}

func TestDoctor_AnalyzeIssues_OldTLS(t *testing.T) {
	svc := NewDoctorService(0)
	report := &DiagnosticReport{
		DNS: &DNSPropagationResult{Consistent: true},
		SSL: &SSLProbeResult{Valid: true, DaysLeft: 90, TLSVersion: "TLS 1.0"},
		HTTP: &HTTPProbeResult{StatusCode: 200, CloudflareRay: "abc123"},
	}
	issues := svc.analyzeIssues(report)
	found := false
	for _, iss := range issues {
		if iss.Probe == "ssl" && iss.Severity == "warning" && iss.Fix != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for TLS 1.0")
	}
}

func TestDoctor_AnalyzeIssues_HTTPServerError(t *testing.T) {
	svc := NewDoctorService(0)
	report := &DiagnosticReport{
		DNS: &DNSPropagationResult{Consistent: true},
		SSL: &SSLProbeResult{Valid: true, DaysLeft: 90, TLSVersion: "TLS 1.3"},
		HTTP: &HTTPProbeResult{StatusCode: 503},
	}
	issues := svc.analyzeIssues(report)
	found := false
	for _, iss := range issues {
		if iss.Probe == "http" && iss.Severity == "critical" {
			found = true
		}
	}
	if !found {
		t.Error("expected critical issue for HTTP 503")
	}
}

func TestDoctor_AnalyzeIssues_NoCFRay(t *testing.T) {
	svc := NewDoctorService(0)
	report := &DiagnosticReport{
		DNS: &DNSPropagationResult{Consistent: true},
		SSL: &SSLProbeResult{Valid: true, DaysLeft: 90, TLSVersion: "TLS 1.3"},
		HTTP: &HTTPProbeResult{StatusCode: 200, CloudflareRay: ""},
	}
	issues := svc.analyzeIssues(report)
	found := false
	for _, iss := range issues {
		if iss.Probe == "http" && iss.Severity == "info" {
			found = true
		}
	}
	if !found {
		t.Error("expected info issue for missing cf-ray header")
	}
}

func TestDoctor_AnalyzeIssues_NSMismatch(t *testing.T) {
	svc := NewDoctorService(0)
	report := &DiagnosticReport{
		DNS: &DNSPropagationResult{Consistent: true},
		SSL: &SSLProbeResult{Valid: true, DaysLeft: 90, TLSVersion: "TLS 1.3"},
		HTTP: &HTTPProbeResult{StatusCode: 200, CloudflareRay: "abc123"},
		Nameservers: &NSProbeResult{
			Expected: []string{"ns1.cf.com"},
			Actual:   []string{"ns1.other.com"},
			Match:    false,
		},
	}
	issues := svc.analyzeIssues(report)
	found := false
	for _, iss := range issues {
		if iss.Probe == "ns" && iss.Severity == "critical" {
			found = true
		}
	}
	if !found {
		t.Error("expected critical issue for NS mismatch")
	}
}

func TestDoctor_AnalyzeIssues_AllHealthy(t *testing.T) {
	svc := NewDoctorService(0)
	report := &DiagnosticReport{
		DNS: &DNSPropagationResult{Consistent: true},
		SSL: &SSLProbeResult{Valid: true, DaysLeft: 90, TLSVersion: "TLS 1.3"},
		HTTP: &HTTPProbeResult{StatusCode: 200, CloudflareRay: "abc123"},
		Nameservers: &NSProbeResult{Match: true},
	}
	issues := svc.analyzeIssues(report)
	if len(issues) != 0 {
		t.Errorf("expected no issues for healthy report, got %d: %v", len(issues), issues)
	}
}

// ---------------------------------------------------------------------------
// TLS version string mapping
// ---------------------------------------------------------------------------

func TestDoctor_TLSVersionString(t *testing.T) {
	tests := []struct {
		version  uint16
		expected string
	}{
		{0x0301, "TLS 1.0"},
		{0x0302, "TLS 1.1"},
		{0x0303, "TLS 1.2"},
		{0x0304, "TLS 1.3"},
		{0x0000, "unknown (0x0000)"},
	}
	for _, tt := range tests {
		got := tlsVersionString(tt.version)
		if got != tt.expected {
			t.Errorf("tlsVersionString(0x%04x) = %q, want %q", tt.version, got, tt.expected)
		}
	}
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestDoctor_StringSlicesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a, b     []string
		expected bool
	}{
		{"both nil", nil, nil, true},
		{"both empty", []string{}, []string{}, true},
		{"equal", []string{"a", "b"}, []string{"a", "b"}, true},
		{"different length", []string{"a"}, []string{"a", "b"}, false},
		{"different content", []string{"a", "b"}, []string{"a", "c"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stringSlicesEqual(tt.a, tt.b); got != tt.expected {
				t.Errorf("stringSlicesEqual(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

func TestDoctor_SortedCopy(t *testing.T) {
	original := []string{"c", "a", "b"}
	sorted := sortedCopy(original)
	if sorted[0] != "a" || sorted[1] != "b" || sorted[2] != "c" {
		t.Errorf("expected [a b c], got %v", sorted)
	}
	// Verify original is unchanged.
	if original[0] != "c" {
		t.Error("sortedCopy modified the original slice")
	}
}

func TestDoctor_SortedCopy_Nil(t *testing.T) {
	if got := sortedCopy(nil); got != nil {
		t.Errorf("expected nil for nil input, got %v", got)
	}
}
