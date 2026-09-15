package cosmoflare

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// defaultResolvers defines the public DNS resolvers used for propagation checks.
var defaultResolvers = []struct {
	Address string
	Name    string
}{
	{"8.8.8.8", "Google"},
	{"8.8.4.4", "Google Secondary"},
	{"1.1.1.1", "Cloudflare"},
	{"1.0.0.1", "Cloudflare Secondary"},
	{"208.67.222.222", "OpenDNS"},
	{"9.9.9.9", "Quad9"},
}

// DoctorService provides diagnostic probes for domain health checks.
// All probes use Go stdlib only — no Cloudflare API credentials required.
type DoctorService struct {
	httpClient *http.Client
	timeout    time.Duration
}

// NewDoctorService creates a new DoctorService with the given timeout.
// If timeout is 0, defaults to 10 seconds.
func NewDoctorService(timeout time.Duration) *DoctorService {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &DoctorService{
		httpClient: &http.Client{Timeout: timeout},
		timeout:    timeout,
	}
}

// ---------------------------------------------------------------------------
// Probe 1: DNS Propagation (P-02)
// ---------------------------------------------------------------------------

// DNSProbeResult holds DNS lookup results from a single resolver.
type DNSProbeResult struct {
	Resolver     string              `json:"resolver"`
	ResolverName string              `json:"resolver_name"`
	Records      map[string][]string `json:"records"`
	LatencyMs    int64               `json:"latency_ms"`
	Error        string              `json:"error,omitempty"`
}

// DNSPropagationResult aggregates DNS propagation results across all resolvers.
type DNSPropagationResult struct {
	Domain     string           `json:"domain"`
	Consistent bool             `json:"consistent"`
	Results    []DNSProbeResult `json:"results"`
	Summary    string           `json:"summary"`
}

// CheckDNSPropagation queries multiple public resolvers in parallel and
// compares A records to detect propagation inconsistencies.
func (d *DoctorService) CheckDNSPropagation(ctx context.Context, domain string) (*DNSPropagationResult, error) {
	if domain == "" {
		return nil, validationError("DoctorService.CheckDNSPropagation", "domain is required")
	}

	results := make([]DNSProbeResult, len(defaultResolvers))
	var wg sync.WaitGroup

	for i, res := range defaultResolvers {
		wg.Add(1)
		go func(idx int, addr, name string) {
			defer wg.Done()
			results[idx] = d.queryResolver(ctx, domain, addr, name)
		}(i, res.Address, res.Name)
	}

	wg.Wait()

	// Compare A records across resolvers for consistency.
	consistent := true
	var firstA []string
	firstSet := false
	resolversReached := 0

	for _, r := range results {
		if r.Error != "" {
			continue
		}
		resolversReached++
		aRecords := r.Records["A"]
		if !firstSet {
			firstA = aRecords
			firstSet = true
			continue
		}
		if !stringSlicesEqual(sortedCopy(firstA), sortedCopy(aRecords)) {
			consistent = false
			break
		}
	}

	if resolversReached == 0 {
		consistent = false
	}

	summary := fmt.Sprintf("DNS propagation check for %s: queried %d resolvers, %d responded", domain, len(defaultResolvers), resolversReached)
	if resolversReached == 0 {
		summary += " — all resolvers failed"
	} else if consistent {
		summary += " — all A records consistent"
	} else {
		summary += " — A record inconsistency detected across resolvers"
	}

	return &DNSPropagationResult{
		Domain:     domain,
		Consistent: consistent,
		Results:    results,
		Summary:    summary,
	}, nil
}

// queryResolver performs DNS lookups against a single resolver.
func (d *DoctorService) queryResolver(ctx context.Context, domain, addr, name string) DNSProbeResult {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialer := net.Dialer{Timeout: d.timeout}
			return dialer.DialContext(ctx, "udp", addr+":53")
		},
	}

	result := DNSProbeResult{
		Resolver:     addr,
		ResolverName: name,
		Records:      make(map[string][]string),
	}

	start := time.Now()

	// A and AAAA records via LookupHost.
	hosts, err := resolver.LookupHost(ctx, domain)
	if err == nil {
		var ipv4, ipv6 []string
		for _, h := range hosts {
			if ip := net.ParseIP(h); ip != nil {
				if ip.To4() != nil {
					ipv4 = append(ipv4, h)
				} else {
					ipv6 = append(ipv6, h)
				}
			}
		}
		if len(ipv4) > 0 {
			result.Records["A"] = ipv4
		}
		if len(ipv6) > 0 {
			result.Records["AAAA"] = ipv6
		}
	}

	// MX records.
	mxRecords, err := resolver.LookupMX(ctx, domain)
	if err == nil && len(mxRecords) > 0 {
		mx := make([]string, 0, len(mxRecords))
		for _, m := range mxRecords {
			mx = append(mx, fmt.Sprintf("%s (priority %d)", m.Host, m.Pref))
		}
		result.Records["MX"] = mx
	}

	// NS records.
	nsRecords, err := resolver.LookupNS(ctx, domain)
	if err == nil && len(nsRecords) > 0 {
		ns := make([]string, 0, len(nsRecords))
		for _, n := range nsRecords {
			ns = append(ns, n.Host)
		}
		result.Records["NS"] = ns
	}

	result.LatencyMs = time.Since(start).Milliseconds()

	// If we got zero records at all, record the last error.
	if len(result.Records) == 0 {
		result.Error = fmt.Sprintf("no DNS records found for %s via %s", domain, addr)
	}

	return result
}

// ---------------------------------------------------------------------------
// Probe 2: SSL Certificate (P-03)
// ---------------------------------------------------------------------------

// SSLProbeResult holds SSL/TLS certificate inspection results.
type SSLProbeResult struct {
	Valid       bool      `json:"valid"`
	Issuer      string    `json:"issuer"`
	Subject     string    `json:"subject"`
	NotBefore   time.Time `json:"not_before"`
	NotAfter    time.Time `json:"not_after"`
	DaysLeft    int       `json:"days_left"`
	TLSVersion  string    `json:"tls_version"`
	ChainLength int       `json:"chain_length"`
	HSTS        bool      `json:"hsts"`
	Error       string    `json:"error,omitempty"`
}

// CheckSSL connects to a domain over TLS and inspects the certificate chain.
func (d *DoctorService) CheckSSL(ctx context.Context, domain string) (*SSLProbeResult, error) {
	if domain == "" {
		return nil, validationError("DoctorService.CheckSSL", "domain is required")
	}

	result := &SSLProbeResult{}

	// TLS connection.
	dialer := &net.Dialer{Timeout: d.timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", domain+":443", &tls.Config{
		InsecureSkipVerify: false,
	})
	if err != nil {
		result.Error = fmt.Sprintf("TLS connection failed: %v", err)
		return result, nil
	}
	defer conn.Close()

	state := conn.ConnectionState()

	if len(state.PeerCertificates) == 0 {
		result.Error = "no peer certificates received"
		return result, nil
	}

	cert := state.PeerCertificates[0]
	now := time.Now()

	result.Valid = now.After(cert.NotBefore) && now.Before(cert.NotAfter)
	result.Issuer = cert.Issuer.String()
	result.Subject = cert.Subject.String()
	result.NotBefore = cert.NotBefore
	result.NotAfter = cert.NotAfter
	result.DaysLeft = int(time.Until(cert.NotAfter).Hours() / 24)
	result.TLSVersion = tlsVersionString(state.Version)
	result.ChainLength = len(state.PeerCertificates)

	// Check HSTS header via HTTPS GET.
	result.HSTS = d.checkHSTS(ctx, domain)

	return result, nil
}

// tlsVersionString maps a TLS version constant to a human-readable string.
func tlsVersionString(v uint16) string {
	switch v {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("unknown (0x%04x)", v)
	}
}

// checkHSTS performs an HTTPS HEAD request and checks for the Strict-Transport-Security header.
func (d *DoctorService) checkHSTS(ctx context.Context, domain string) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://"+domain, nil)
	if err != nil {
		return false
	}
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.Header.Get("Strict-Transport-Security") != ""
}

// ---------------------------------------------------------------------------
// Probe 3: HTTP Response (P-04)
// ---------------------------------------------------------------------------

// HTTPProbeResult holds HTTP response diagnostic information.
type HTTPProbeResult struct {
	StatusCode     int      `json:"status_code"`
	ResponseTimeMs int64    `json:"response_time_ms"`
	RedirectChain  []string `json:"redirect_chain,omitempty"`
	CloudflareRay  string   `json:"cloudflare_ray,omitempty"`
	Server         string   `json:"server,omitempty"`
	Error          string   `json:"error,omitempty"`
}

// CheckHTTP performs an HTTPS GET against the domain and captures response metadata.
func (d *DoctorService) CheckHTTP(ctx context.Context, domain string) (*HTTPProbeResult, error) {
	if domain == "" {
		return nil, validationError("DoctorService.CheckHTTP", "domain is required")
	}

	var redirects []string
	client := &http.Client{
		Timeout: d.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirects = append(redirects, req.URL.String())
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+domain, nil)
	if err != nil {
		return nil, newError("DoctorService.CheckHTTP", "failed to create request", err)
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		return &HTTPProbeResult{
			ResponseTimeMs: elapsed.Milliseconds(),
			RedirectChain:  redirects,
			Error:          fmt.Sprintf("HTTP request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	return &HTTPProbeResult{
		StatusCode:     resp.StatusCode,
		ResponseTimeMs: elapsed.Milliseconds(),
		RedirectChain:  redirects,
		CloudflareRay:  resp.Header.Get("cf-ray"),
		Server:         resp.Header.Get("Server"),
	}, nil
}

// ---------------------------------------------------------------------------
// Probe 4: Nameserver Consistency (P-05)
// ---------------------------------------------------------------------------

// NSProbeResult holds nameserver consistency check results.
type NSProbeResult struct {
	Expected  []string `json:"expected"`
	Actual    []string `json:"actual"`
	Match     bool     `json:"match"`
	DNSSEC    bool     `json:"dnssec"`
	PrimaryNS string   `json:"primary_ns,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// CheckNameservers looks up NS records and compares them to the expected set.
func (d *DoctorService) CheckNameservers(ctx context.Context, domain string, expected []string) (*NSProbeResult, error) {
	if domain == "" {
		return nil, validationError("DoctorService.CheckNameservers", "domain is required")
	}
	if len(expected) == 0 {
		return nil, validationError("DoctorService.CheckNameservers", "expected nameservers list is required")
	}

	result := &NSProbeResult{
		Expected: expected,
	}

	nsRecords, err := net.DefaultResolver.LookupNS(ctx, domain)
	if err != nil {
		result.Error = fmt.Sprintf("NS lookup failed: %v", err)
		return result, nil
	}

	actual := make([]string, 0, len(nsRecords))
	for _, ns := range nsRecords {
		actual = append(actual, ns.Host)
	}
	result.Actual = actual

	// Compare: case-insensitive, trailing-dot-agnostic.
	result.Match = compareNameservers(expected, actual)

	if len(actual) > 0 {
		result.PrimaryNS = actual[0]
	}

	// DNSSEC detection is not available via Go stdlib (no DS/RRSIG record access).
	// Default to false with this noted limitation.
	result.DNSSEC = false

	return result, nil
}

// compareNameservers checks whether two NS lists are equivalent,
// ignoring case and trailing dots.
func compareNameservers(expected, actual []string) bool {
	norm := func(list []string) []string {
		out := make([]string, len(list))
		for i, s := range list {
			out[i] = strings.ToLower(strings.TrimSuffix(s, "."))
		}
		sort.Strings(out)
		return out
	}
	return stringSlicesEqual(norm(expected), norm(actual))
}

// ---------------------------------------------------------------------------
// Full Diagnostic Runner
// ---------------------------------------------------------------------------

// DiagnosticReport aggregates all probe results with issue analysis.
type DiagnosticReport struct {
	Domain          string                `json:"domain"`
	Timestamp       time.Time             `json:"timestamp"`
	DNS             *DNSPropagationResult `json:"dns"`
	SSL             *SSLProbeResult       `json:"ssl"`
	HTTP            *HTTPProbeResult      `json:"http"`
	Nameservers     *NSProbeResult        `json:"nameservers,omitempty"`
	RedirectTargets []RedirectProbeResult `json:"redirect_targets,omitempty"`
	Issues          []DiagnosticIssue     `json:"issues"`
	Score           string                `json:"score"`
}

// DiagnosticIssue represents a single problem found during diagnostics.
type DiagnosticIssue struct {
	Probe    string `json:"probe"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Fix      string `json:"fix,omitempty"`
}

// RunDiagnostics executes all probes and produces a consolidated report.
// If expectedNS is nil, the nameserver consistency check is skipped.
func (d *DoctorService) RunDiagnostics(ctx context.Context, domain string, expectedNS []string) (*DiagnosticReport, error) {
	if domain == "" {
		return nil, validationError("DoctorService.RunDiagnostics", "domain is required")
	}

	report := &DiagnosticReport{
		Domain:    domain,
		Timestamp: time.Now(),
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	var probeErrors []string

	// DNS probe.
	wg.Add(1)
	go func() {
		defer wg.Done()
		dns, err := d.CheckDNSPropagation(ctx, domain)
		mu.Lock()
		report.DNS = dns
		if err != nil {
			probeErrors = append(probeErrors, fmt.Sprintf("dns: %v", err))
		}
		mu.Unlock()
	}()

	// SSL probe.
	wg.Add(1)
	go func() {
		defer wg.Done()
		ssl, err := d.CheckSSL(ctx, domain)
		mu.Lock()
		report.SSL = ssl
		if err != nil {
			probeErrors = append(probeErrors, fmt.Sprintf("ssl: %v", err))
		}
		mu.Unlock()
	}()

	// HTTP probe.
	wg.Add(1)
	go func() {
		defer wg.Done()
		httpResult, err := d.CheckHTTP(ctx, domain)
		mu.Lock()
		report.HTTP = httpResult
		if err != nil {
			probeErrors = append(probeErrors, fmt.Sprintf("http: %v", err))
		}
		mu.Unlock()
	}()

	// NS probe (optional).
	if len(expectedNS) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ns, err := d.CheckNameservers(ctx, domain, expectedNS)
			mu.Lock()
			report.Nameservers = ns
			if err != nil {
				probeErrors = append(probeErrors, fmt.Sprintf("ns: %v", err))
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Surface hard probe errors as critical issues.
	for _, pe := range probeErrors {
		report.Issues = append(report.Issues, DiagnosticIssue{
			Probe:    "system",
			Severity: "critical",
			Message:  fmt.Sprintf("Probe error: %s", pe),
		})
	}

	// Analyze results and populate issues.
	report.Issues = append(report.Issues, d.analyzeIssues(report)...)
	report.Score = computeScore(report.Issues)

	return report, nil
}

// analyzeIssues inspects probe results and returns detected issues.
func (d *DoctorService) analyzeIssues(report *DiagnosticReport) []DiagnosticIssue {
	var issues []DiagnosticIssue

	issues = appendDNSIssues(issues, report.DNS)
	issues = appendSSLIssues(issues, report.SSL)
	issues = appendHTTPIssues(issues, report.HTTP)
	issues = appendNSIssues(issues, report.Nameservers)

	return issues
}

// appendDNSIssues appends issues from the DNS propagation probe.
func appendDNSIssues(issues []DiagnosticIssue, dns *DNSPropagationResult) []DiagnosticIssue {
	if dns == nil {
		return issues
	}
	if !dns.Consistent {
		issues = append(issues, DiagnosticIssue{
			Probe:    "dns",
			Severity: "warning",
			Message:  "A records are inconsistent across DNS resolvers",
			Fix:      "Wait for DNS propagation to complete, or verify records are correct in your DNS provider",
		})
	}
	return issues
}

// appendSSLIssues appends issues from the SSL certificate probe.
func appendSSLIssues(issues []DiagnosticIssue, ssl *SSLProbeResult) []DiagnosticIssue {
	if ssl == nil {
		return issues
	}
	if ssl.Error != "" {
		return append(issues, DiagnosticIssue{
			Probe:    "ssl",
			Severity: "critical",
			Message:  fmt.Sprintf("SSL probe failed: %s", ssl.Error),
			Fix:      "Verify the domain has a valid SSL certificate and is accessible on port 443",
		})
	}
	if ssl.DaysLeft < 0 {
		issues = append(issues, DiagnosticIssue{
			Probe:    "ssl",
			Severity: "critical",
			Message:  fmt.Sprintf("SSL certificate expired %d days ago", -ssl.DaysLeft),
			Fix:      "Renew the SSL certificate immediately",
		})
	} else if ssl.DaysLeft < 30 {
		issues = append(issues, DiagnosticIssue{
			Probe:    "ssl",
			Severity: "warning",
			Message:  fmt.Sprintf("SSL certificate expires in %d days", ssl.DaysLeft),
			Fix:      "Renew the SSL certificate before expiration",
		})
	}
	if ssl.TLSVersion == "TLS 1.0" || ssl.TLSVersion == "TLS 1.1" {
		issues = append(issues, DiagnosticIssue{
			Probe:    "ssl",
			Severity: "warning",
			Message:  fmt.Sprintf("TLS version %s is outdated and insecure", ssl.TLSVersion),
			Fix:      "cosmoflare ssl update ZONE_ID --min-tls=1.2",
		})
	}
	return issues
}

// appendHTTPIssues appends issues from the HTTP response probe.
func appendHTTPIssues(issues []DiagnosticIssue, httpResult *HTTPProbeResult) []DiagnosticIssue {
	if httpResult == nil {
		return issues
	}
	if httpResult.Error != "" {
		return append(issues, DiagnosticIssue{
			Probe:    "http",
			Severity: "critical",
			Message:  fmt.Sprintf("HTTP probe failed: %s", httpResult.Error),
		})
	}
	if httpResult.StatusCode >= 500 {
		issues = append(issues, DiagnosticIssue{
			Probe:    "http",
			Severity: "critical",
			Message:  fmt.Sprintf("HTTP returned server error status %d", httpResult.StatusCode),
		})
	} else if httpResult.StatusCode >= 400 {
		issues = append(issues, DiagnosticIssue{
			Probe:    "http",
			Severity: "warning",
			Message:  fmt.Sprintf("HTTP returned client error status %d", httpResult.StatusCode),
		})
	}
	if httpResult.CloudflareRay == "" {
		issues = append(issues, DiagnosticIssue{
			Probe:    "http",
			Severity: "info",
			Message:  "No cf-ray header detected — domain may not be proxied through Cloudflare",
		})
	}
	return issues
}

// appendNSIssues appends issues from the nameserver consistency probe.
func appendNSIssues(issues []DiagnosticIssue, ns *NSProbeResult) []DiagnosticIssue {
	if ns == nil {
		return issues
	}
	if ns.Error != "" {
		return append(issues, DiagnosticIssue{
			Probe:    "ns",
			Severity: "critical",
			Message:  fmt.Sprintf("Nameserver probe failed: %s", ns.Error),
		})
	}
	if !ns.Match {
		issues = append(issues, DiagnosticIssue{
			Probe:    "ns",
			Severity: "critical",
			Message:  "Nameservers do not match expected configuration",
			Fix:      "Update nameservers at your registrar",
		})
	}
	return issues
}

// computeScore determines the overall health score from the issues list.
func computeScore(issues []DiagnosticIssue) string {
	for _, issue := range issues {
		if issue.Severity == "critical" {
			return "critical"
		}
	}
	for _, issue := range issues {
		if issue.Severity == "warning" {
			return "warning"
		}
	}
	return "healthy"
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// stringSlicesEqual checks if two sorted string slices are identical.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// sortedCopy returns a sorted copy of the input slice without modifying it.
func sortedCopy(s []string) []string {
	if s == nil {
		return nil
	}
	c := make([]string, len(s))
	copy(c, s)
	sort.Strings(c)
	return c
}
