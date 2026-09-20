package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// TestRunDoctorAll_MissingCreds verifies that fleet mode surfaces the
// fleet-status service construction error before any API request when
// credentials are absent.
func TestRunDoctorAll_MissingCreds(t *testing.T) {
	runGlobalsSnapshot(t)

	err := runDoctorAll(context.Background())
	if err == nil {
		t.Fatal("runDoctorAll with empty credentials should return an error")
	}
	if !strings.Contains(err.Error(), "failed to create fleet status service") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to create fleet status service")
	}
}

// TestRunDoctor_AllRoutesToDoctorAll verifies runDoctor routes to the fleet
// path when --all is set, even with no domain argument.
func TestRunDoctor_AllRoutesToDoctorAll(t *testing.T) {
	runGlobalsSnapshot(t)

	origAll := doctorAll
	doctorAll = true
	t.Cleanup(func() { doctorAll = origAll })

	err := runDoctor(doctorCmd, []string{})
	if err == nil {
		t.Fatal("runDoctor --all with missing creds should return an error")
	}
	if !strings.Contains(err.Error(), "failed to create fleet status service") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to create fleet status service")
	}
}

// TestRunDoctorSingle_ZoneIDTargetWithoutCreds verifies the single-domain path
// fails fast with the zone-service construction error when the target looks
// like a zone ID and credentials are missing — before any network probe runs.
func TestRunDoctorSingle_ZoneIDTargetWithoutCreds(t *testing.T) {
	runGlobalsSnapshot(t)

	doctor := cosmoflare.NewDoctorService(time.Second)
	err := runDoctorSingle(context.Background(), doctor, "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6")
	if err == nil {
		t.Fatal("runDoctorSingle with zone-ID target and no creds should return an error")
	}
	if !strings.Contains(err.Error(), "failed to create zone service") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to create zone service")
	}
}

// TestResolveDoctorTarget verifies target resolution for domain inputs and
// the zone-ID error path.
func TestResolveDoctorTarget(t *testing.T) {
	t.Run("plain domain passes through with no expected NS offline", func(t *testing.T) {
		runGlobalsSnapshot(t)

		domain, ns, err := resolveDoctorTarget(context.Background(), "example.com")
		if err != nil {
			t.Fatalf("resolveDoctorTarget(domain) returned error: %v", err)
		}
		if domain != "example.com" {
			t.Errorf("domain = %q, want %q", domain, "example.com")
		}
		if ns != nil {
			t.Errorf("expectedNS = %v, want nil when zone service is unavailable", ns)
		}
	})

	t.Run("zone ID target without creds errors", func(t *testing.T) {
		runGlobalsSnapshot(t)

		_, _, err := resolveDoctorTarget(context.Background(), "0123456789abcdef0123456789abcdef")
		if err == nil {
			t.Fatal("resolveDoctorTarget(zoneID) without creds should return an error")
		}
		if !strings.Contains(err.Error(), "failed to create zone service") {
			t.Errorf("error = %q, want it to contain %q", err.Error(), "failed to create zone service")
		}
	})

	t.Run("empty target is not treated as zone ID", func(t *testing.T) {
		runGlobalsSnapshot(t)

		domain, _, err := resolveDoctorTarget(context.Background(), "")
		if err != nil {
			t.Fatalf("resolveDoctorTarget(\"\") returned error: %v", err)
		}
		if domain != "" {
			t.Errorf("domain = %q, want empty passthrough", domain)
		}
	})
}

// TestEnrichDoctorReport_NoCredsIsNoop verifies that the best-effort redirect
// enrichment leaves the report untouched (and does not panic) when the zone
// service cannot be constructed.
func TestEnrichDoctorReport_NoCredsIsNoop(t *testing.T) {
	runGlobalsSnapshot(t)

	report := &cosmoflare.DiagnosticReport{
		Domain: "example.com",
		Issues: []cosmoflare.DiagnosticIssue{},
	}

	enrichDoctorReport(context.Background(), report, "example.com")

	if report.RedirectTargets != nil {
		t.Errorf("RedirectTargets = %v, want nil when enrichment is skipped", report.RedirectTargets)
	}
	if len(report.Issues) != 0 {
		t.Errorf("Issues = %v, want unchanged when enrichment is skipped", report.Issues)
	}
}

// TestPrintDNSSection verifies DNS section rendering for the skipped,
// consistent and inconsistent cases.
func TestPrintDNSSection(t *testing.T) {
	t.Run("nil DNS prints skipped", func(t *testing.T) {
		out := capturePrint(t, func() { printDNSSection(&cosmoflare.DiagnosticReport{}) })
		if !strings.Contains(out, "DNS Propagation  (skipped)") {
			t.Errorf("output should mark DNS as skipped, got: %q", out)
		}
	})

	t.Run("consistent results render checkmark and summary", func(t *testing.T) {
		report := &cosmoflare.DiagnosticReport{
			DNS: &cosmoflare.DNSPropagationResult{
				Domain:     "example.com",
				Consistent: true,
				Results: []cosmoflare.DNSProbeResult{
					{Resolver: "8.8.8.8", ResolverName: "Google", Records: map[string][]string{"A": {"203.0.113.1"}}, LatencyMs: 12},
					{Resolver: "1.1.1.1", ResolverName: "Cloudflare", Records: map[string][]string{"A": {"203.0.113.1"}}, LatencyMs: 9},
				},
			},
		}
		out := capturePrint(t, func() { printDNSSection(report) })
		for _, want := range []string{"DNS Propagation ✓", "Queried 2 resolvers, all A records consistent", "203.0.113.1"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q, got: %q", want, out)
			}
		}
	})

	t.Run("inconsistent results render cross, errors and empty record sets", func(t *testing.T) {
		report := &cosmoflare.DiagnosticReport{
			DNS: &cosmoflare.DNSPropagationResult{
				Domain:     "example.com",
				Consistent: false,
				Results: []cosmoflare.DNSProbeResult{
					{Resolver: "8.8.8.8", ResolverName: "Google", Records: map[string][]string{"A": {"203.0.113.1"}}, LatencyMs: 12},
					{Resolver: "9.9.9.9", ResolverName: "Quad9", Error: "i/o timeout"},
					{Resolver: "1.1.1.1", ResolverName: "Cloudflare", Records: map[string][]string{}, LatencyMs: 9},
				},
			},
		}
		out := capturePrint(t, func() { printDNSSection(report) })
		for _, want := range []string{
			"DNS Propagation ✗",
			"Queried 3 resolvers, 2 responded — inconsistency detected",
			"error — i/o timeout",
			"no A records",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q, got: %q", want, out)
			}
		}
	})
}

// TestPrintSSLSection verifies SSL section rendering for the skipped, error
// and healthy/expiring/expired certificate states.
func TestPrintSSLSection(t *testing.T) {
	t.Run("nil SSL prints skipped", func(t *testing.T) {
		out := capturePrint(t, func() { printSSLSection(&cosmoflare.DiagnosticReport{}) })
		if !strings.Contains(out, "SSL Certificate  (skipped)") {
			t.Errorf("output should mark SSL as skipped, got: %q", out)
		}
	})

	t.Run("probe error renders cross and message", func(t *testing.T) {
		report := &cosmoflare.DiagnosticReport{
			SSL: &cosmoflare.SSLProbeResult{Error: "connection refused"},
		}
		out := capturePrint(t, func() { printSSLSection(report) })
		for _, want := range []string{"SSL Certificate ✗", "Error: connection refused"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q, got: %q", want, out)
			}
		}
	})

	t.Run("valid long-lived certificate renders checkmark", func(t *testing.T) {
		report := &cosmoflare.DiagnosticReport{
			SSL: &cosmoflare.SSLProbeResult{
				Valid:      true,
				Subject:    "example.com",
				Issuer:     "Let's Encrypt",
				NotAfter:   time.Now().Add(90 * 24 * time.Hour),
				DaysLeft:   90,
				TLSVersion: "1.3",
				HSTS:       true,
			},
		}
		out := capturePrint(t, func() { printSSLSection(report) })
		for _, want := range []string{"SSL Certificate ✓", "Valid: example.com", "Issuer: Let's Encrypt", "HSTS: yes"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q, got: %q", want, out)
			}
		}
	})

	t.Run("expiring soon renders warning", func(t *testing.T) {
		report := &cosmoflare.DiagnosticReport{
			SSL: &cosmoflare.SSLProbeResult{Valid: true, DaysLeft: 10, NotAfter: time.Now().Add(10 * 24 * time.Hour)},
		}
		out := capturePrint(t, func() { printSSLSection(report) })
		if !strings.Contains(out, "SSL Certificate ⚠") {
			t.Errorf("output should carry warning indicator, got: %q", out)
		}
	})

	t.Run("expired certificate renders cross", func(t *testing.T) {
		report := &cosmoflare.DiagnosticReport{
			SSL: &cosmoflare.SSLProbeResult{Valid: true, DaysLeft: -3, NotAfter: time.Now().Add(-72 * time.Hour), HSTS: false},
		}
		out := capturePrint(t, func() { printSSLSection(report) })
		for _, want := range []string{"SSL Certificate ✗", "HSTS: no"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q, got: %q", want, out)
			}
		}
	})
}

// TestPrintHTTPSection verifies HTTP section rendering for the skipped,
// error and success/client-error/server-error status classes.
func TestPrintHTTPSection(t *testing.T) {
	t.Run("nil HTTP prints skipped", func(t *testing.T) {
		out := capturePrint(t, func() { printHTTPSection(&cosmoflare.DiagnosticReport{}) })
		if !strings.Contains(out, "HTTP Response  (skipped)") {
			t.Errorf("output should mark HTTP as skipped, got: %q", out)
		}
	})

	t.Run("probe error renders cross and message", func(t *testing.T) {
		report := &cosmoflare.DiagnosticReport{
			HTTP: &cosmoflare.HTTPProbeResult{Error: "dial tcp: timeout"},
		}
		out := capturePrint(t, func() { printHTTPSection(report) })
		for _, want := range []string{"HTTP Response ✗", "Error: dial tcp: timeout"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q, got: %q", want, out)
			}
		}
	})

	t.Run("success renders checkmark with metadata", func(t *testing.T) {
		report := &cosmoflare.DiagnosticReport{
			HTTP: &cosmoflare.HTTPProbeResult{
				StatusCode:     200,
				ResponseTimeMs: 120,
				Server:         "cloudflare",
				CloudflareRay:  "8f3a1b2c3d4e5f6a",
				RedirectChain:  []string{"http://example.com", "https://example.com"},
			},
		}
		out := capturePrint(t, func() { printHTTPSection(report) })
		for _, want := range []string{
			"HTTP Response ✓",
			"Status: 200",
			"Response time: 120ms",
			"Server: cloudflare",
			"CF-Ray: 8f3a1b2c3d4e5f6a",
			"Redirects: http://example.com -> https://example.com",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q, got: %q", want, out)
			}
		}
	})

	t.Run("client error renders warning", func(t *testing.T) {
		report := &cosmoflare.DiagnosticReport{
			HTTP: &cosmoflare.HTTPProbeResult{StatusCode: 404, ResponseTimeMs: 30},
		}
		out := capturePrint(t, func() { printHTTPSection(report) })
		if !strings.Contains(out, "HTTP Response ⚠") {
			t.Errorf("output should carry warning indicator, got: %q", out)
		}
	})

	t.Run("server error renders cross", func(t *testing.T) {
		report := &cosmoflare.DiagnosticReport{
			HTTP: &cosmoflare.HTTPProbeResult{StatusCode: 503, ResponseTimeMs: 30},
		}
		out := capturePrint(t, func() { printHTTPSection(report) })
		if !strings.Contains(out, "HTTP Response ✗") {
			t.Errorf("output should carry failure indicator, got: %q", out)
		}
	})

	t.Run("empty optional fields are omitted", func(t *testing.T) {
		report := &cosmoflare.DiagnosticReport{
			HTTP: &cosmoflare.HTTPProbeResult{StatusCode: 200, ResponseTimeMs: 5},
		}
		out := capturePrint(t, func() { printHTTPSection(report) })
		for _, unwanted := range []string{"Server:", "CF-Ray:", "Redirects:"} {
			if strings.Contains(out, unwanted) {
				t.Errorf("output should not contain %q when field is empty, got: %q", unwanted, out)
			}
		}
	})
}

// TestPrintNSSection verifies nameserver section rendering for the skipped,
// error, matching and mismatching cases.
func TestPrintNSSection(t *testing.T) {
	t.Run("nil NS prints skipped", func(t *testing.T) {
		out := capturePrint(t, func() { printNSSection(&cosmoflare.DiagnosticReport{}) })
		if !strings.Contains(out, "Nameservers  (skipped — no expected NS available)") {
			t.Errorf("output should mark NS as skipped, got: %q", out)
		}
	})

	t.Run("probe error renders cross and message", func(t *testing.T) {
		report := &cosmoflare.DiagnosticReport{
			Nameservers: &cosmoflare.NSProbeResult{Error: "lookup failed"},
		}
		out := capturePrint(t, func() { printNSSection(report) })
		for _, want := range []string{"Nameservers ✗", "Error: lookup failed"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q, got: %q", want, out)
			}
		}
	})

	t.Run("matching nameservers render checkmark", func(t *testing.T) {
		report := &cosmoflare.DiagnosticReport{
			Nameservers: &cosmoflare.NSProbeResult{
				Expected: []string{"ns1.cloudflare.com", "ns2.cloudflare.com"},
				Actual:   []string{"ns1.cloudflare.com", "ns2.cloudflare.com"},
				Match:    true,
			},
		}
		out := capturePrint(t, func() { printNSSection(report) })
		for _, want := range []string{
			"Nameservers ✓",
			"Expected: ns1.cloudflare.com, ns2.cloudflare.com",
			"Actual:   ns1.cloudflare.com, ns2.cloudflare.com",
			"Match: yes",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q, got: %q", want, out)
			}
		}
	})

	t.Run("mismatching nameservers render cross", func(t *testing.T) {
		report := &cosmoflare.DiagnosticReport{
			Nameservers: &cosmoflare.NSProbeResult{
				Expected: []string{"ns1.cloudflare.com"},
				Actual:   []string{"ns1.old-dns.example"},
				Match:    false,
			},
		}
		out := capturePrint(t, func() { printNSSection(report) })
		for _, want := range []string{"Nameservers ✗", "Match: no"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q, got: %q", want, out)
			}
		}
	})
}
