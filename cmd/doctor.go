package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor [domain]",
	Short: "Run diagnostic health checks on a domain",
	Long: `Deep diagnostic probes for domain health analysis, and a fleet-wide
protection matrix across every zone.

Single-domain mode runs 4 network probes against a domain or zone:
  1. DNS Propagation — queries 6 public resolvers for record consistency
  2. SSL Certificate — inspects TLS certificate chain, expiry, and HSTS
  3. HTTP Response — checks status code, response time, redirect chain, cf-ray
  4. Nameserver Consistency — compares assigned NS with DNS-returned NS

Results are presented with pass/warn/fail indicators and an overall health score.
Use --fix to see actionable cosmoflare commands for each issue found.

Fleet mode (--all) skips the network probes and instead reads Cloudflare's
own API-reported state for every zone on the account in one pass: zone
active status, DNSSEC, Universal SSL, certificate expiry, minimum TLS
version, security level, and development mode. It stays fast across
hundreds of zones because it never leaves the Cloudflare API.

This command is designed for both human operators and AI agents:
  - Human-readable output with colored status indicators by default
  - --json provides structured diagnostic data for automated pipelines
  - --fix suggestions are valid cosmoflare commands that can be executed directly
  - Exit code reflects health: 0=healthy/warning, 1=critical (or degraded, in fleet mode)

Examples:
  cosmoflare doctor example.com              # Full diagnostics for a domain
  cosmoflare doctor example.com --fix        # Include fix suggestions
  cosmoflare doctor example.com --json       # Machine-readable output
  cosmoflare doctor --all                    # Fleet-wide protection matrix (fast, API-state)
  cosmoflare doctor --all --json             # Fleet matrix, machine-readable output

  # Use with other cosmoflare commands:
  cosmoflare doctor example.com --fix --json | jq '.issues[].fix'
  cosmoflare doctor --all --json | jq '.zones[] | select(.issues != [])'
  cosmoflare domains --json | jq -r '.domains[].zone.name' | xargs -I{} cosmoflare doctor {}`,
	RunE: runDoctor,
}

var (
	doctorFix bool
	doctorAll bool
)

// zoneIDPattern matches a 32-character hex string (Cloudflare zone ID format).
var zoneIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

func init() {
	rootCmd.AddCommand(doctorCmd)

	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Show actionable fix commands for each issue")
	doctorCmd.Flags().BoolVar(&doctorAll, "all", false, "Run diagnostics on all domains (slow)")
}

func runDoctor(cmd *cobra.Command, args []string) error {
	if !doctorAll && len(args) == 0 {
		return fmt.Errorf("domain argument is required (or use --all for all domains)")
	}

	ctx := context.Background()

	if doctorAll {
		return runDoctorAll(ctx)
	}

	doctor := cosmoflare.NewDoctorService(10 * time.Second)
	return runDoctorSingle(ctx, doctor, args[0])
}

// runDoctorAll runs the fleet-wide protection matrix: Cloudflare's own
// API-reported state (DNSSEC, Universal SSL, certificate expiry, zone
// settings) for every zone in one bounded-concurrency pass. This is
// distinct from single-domain doctor, which network-probes a domain from
// the outside — --all reads account-side state instead, so it stays fast
// even across hundreds of zones.
func runDoctorAll(ctx context.Context) error {
	fleet, err := cosmoflare.NewFleetStatusServiceFromCreds(AccountID, APIToken)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create fleet status service: %v", err))
		}
		return fmt.Errorf("failed to create fleet status service: %w", err)
	}

	snap, err := fleet.Snapshot(ctx)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to collect fleet status: %v", err))
		}
		return fmt.Errorf("failed to collect fleet status: %w", err)
	}

	if len(snap.Zones) == 0 {
		if JSONOutput {
			return printSuccessJSON("No zones found", nil)
		}
		printInfo("No zones found in account")
		return nil
	}

	if JSONOutput {
		if err := printJSON(snap); err != nil {
			return err
		}
	} else {
		printFleetTable(snap)
	}

	if snap.DegradedCount > 0 {
		return fmt.Errorf("%d of %d zone(s) degraded", snap.DegradedCount, len(snap.Zones))
	}
	return nil
}

// printFleetTable renders the fleet-wide protection matrix as a table:
// ZONE | STATUS | DNSSEC | SSL | CERT | TLS | WAF | ISSUES, followed by a
// one-line summary.
func printFleetTable(snap *cosmoflare.FleetStatus) {
	fmt.Fprintf(os.Stdout, "%-32s %-9s %-9s %-6s %-9s %-5s %-8s %s\n",
		"ZONE", "STATUS", "DNSSEC", "SSL", "CERT", "TLS", "WAF", "ISSUES")

	for _, z := range snap.Zones {
		status := "active"
		if !z.ZoneActive {
			status = "inactive"
		}
		ssl := "off"
		if z.UniversalSSL {
			ssl = "on"
		}
		fmt.Fprintf(os.Stdout, "%-32s %-9s %-9s %-6s %-9s %-5s %-8s %s\n",
			z.Zone, status, z.DNSSECStatus, ssl, formatCertExpiry(z.CertExpiresIn),
			z.MinTLS, z.SecurityLevel, strings.Join(z.Issues, "; "))
	}

	fmt.Fprintln(os.Stdout)
	fmt.Fprintf(os.Stdout, "%d zones: %d healthy, %d degraded\n",
		len(snap.Zones), snap.HealthyCount, snap.DegradedCount)
}

// formatCertExpiry renders a certificate expiry field for the fleet table:
// "—" when unknown, "expired" when past due, else "<n>d".
func formatCertExpiry(days *int) string {
	if days == nil {
		return "—"
	}
	if *days < 0 {
		return "expired"
	}
	return fmt.Sprintf("%dd", *days)
}

func runDoctorSingle(ctx context.Context, doctor *cosmoflare.DoctorService, target string) error {
	domain := target
	var expectedNS []string

	// If the target looks like a zone ID, resolve it to a domain name.
	if zoneIDPattern.MatchString(target) {
		zoneSvc, err := getZoneService()
		if err != nil {
			return fmt.Errorf("failed to create zone service: %w", err)
		}
		zone, err := zoneSvc.Get(ctx, target)
		if err != nil {
			if JSONOutput {
				return printErrorJSON(fmt.Sprintf("failed to resolve zone ID: %v", err))
			}
			return fmt.Errorf("failed to resolve zone ID %s: %w", target, err)
		}
		domain = zone.Name
		expectedNS = zone.NameServers
	} else {
		// Try to get expected nameservers from the Cloudflare API.
		// This is best-effort — if it fails, we just skip the NS check.
		zoneSvc, zoneErr := getZoneService()
		if zoneErr == nil {
			zones, listErr := zoneSvc.List(ctx)
			if listErr == nil {
				for _, z := range zones {
					if z.Name == domain {
						expectedNS = z.NameServers
						break
					}
				}
			}
		}
	}

	report, err := doctor.RunDiagnostics(ctx, domain, expectedNS)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("diagnostics failed: %v", err))
		}
		return fmt.Errorf("diagnostics failed for %s: %w", domain, err)
	}

	// Redirect-target section (opt-in by having redirects): fetch the
	// domain's destinations via DomainService (CF API), probe them with the
	// stdlib RedirectProber, attach to the report. DoctorService itself
	// stays credential-free — the cmd layer supplies the data.
	if zoneSvc, err := getZoneService(); err == nil {
		if domSvc, err := cosmoflare.NewDomainService(zoneSvc, nil, nil, nil); err == nil {
			if rs, err := cosmoflare.NewRedirectServiceFromCreds(AccountID, APIToken); err == nil {
				domSvc = domSvc.WithRedirects(rs)
				if zones, err := zoneSvc.List(ctx); err == nil {
					for _, z := range zones {
						if z.Name != domain {
							continue
						}
						if detail, err := domSvc.GetDetail(ctx, z.ID); err == nil && len(detail.Redirects) > 0 {
							seen := map[string]bool{}
							dests := make([]string, 0, len(detail.Redirects))
							for _, r := range detail.Redirects {
								if r.Destination != "" && !seen[r.Destination] {
									seen[r.Destination] = true
									dests = append(dests, r.Destination)
								}
							}
							results := cosmoflare.NewRedirectProber().ProbeAll(ctx, dests)
							for _, d := range dests {
								report.RedirectTargets = append(report.RedirectTargets, results[d])
							}
							if issue := cosmoflare.ClassifyRedirectIssues(report.RedirectTargets); issue != "" {
								report.Issues = append(report.Issues, cosmoflare.DiagnosticIssue{
									Probe: "redirect-target", Severity: "warning",
									Message: "redirect target problem: " + issue,
									Fix:     "inspect the redirect destination — it loops, errors, or is unreachable",
								})
							}
						}
						break
					}
				}
			}
		}
	}

	if JSONOutput {
		return printJSON(report)
	}

	printDoctorReport(report)

	if report.Score == "critical" {
		return fmt.Errorf("domain health is critical")
	}
	return nil
}

// printDoctorReport renders a human-readable diagnostic report.
func printDoctorReport(report *cosmoflare.DiagnosticReport) {
	fmt.Fprintf(os.Stdout, "Diagnostics for %s\n", report.Domain)
	fmt.Fprintln(os.Stdout, strings.Repeat("=", 30+len(report.Domain)))
	fmt.Fprintln(os.Stdout)

	// DNS Propagation
	printDNSSection(report)

	// SSL Certificate
	printSSLSection(report)

	// HTTP Response
	printHTTPSection(report)

	// Nameservers
	printNSSection(report)

	// Issues
	printIssuesSection(report)

	// Score
	issueCount := len(report.Issues)
	switch report.Score {
	case "healthy":
		printSuccess("Score: healthy (%d issues)", issueCount)
	case "warning":
		printWarning("Score: warning (%d issues)", issueCount)
	case "critical":
		fmt.Fprintf(os.Stdout, "❌ Score: critical (%d issues)\n", issueCount)
	}
}

func printDNSSection(report *cosmoflare.DiagnosticReport) {
	if report.DNS == nil {
		fmt.Fprintln(os.Stdout, "DNS Propagation  (skipped)")
		fmt.Fprintln(os.Stdout)
		return
	}

	status := "✓"
	if !report.DNS.Consistent {
		status = "✗"
	}
	fmt.Fprintf(os.Stdout, "DNS Propagation %s\n", status)

	responded := 0
	for _, r := range report.DNS.Results {
		if r.Error == "" {
			responded++
		}
	}
	if report.DNS.Consistent {
		fmt.Fprintf(os.Stdout, "  Queried %d resolvers, all A records consistent\n", len(report.DNS.Results))
	} else {
		fmt.Fprintf(os.Stdout, "  Queried %d resolvers, %d responded — inconsistency detected\n", len(report.DNS.Results), responded)
	}

	for _, r := range report.DNS.Results {
		if r.Error != "" {
			fmt.Fprintf(os.Stdout, "  %s (%s): error — %s\n", r.ResolverName, r.Resolver, r.Error)
			continue
		}
		aRecords := r.Records["A"]
		if len(aRecords) > 0 {
			fmt.Fprintf(os.Stdout, "  %s (%s): %s (%dms)\n", r.ResolverName, r.Resolver, strings.Join(aRecords, ", "), r.LatencyMs)
		} else {
			fmt.Fprintf(os.Stdout, "  %s (%s): no A records (%dms)\n", r.ResolverName, r.Resolver, r.LatencyMs)
		}
	}
	fmt.Fprintln(os.Stdout)
}

func printSSLSection(report *cosmoflare.DiagnosticReport) {
	if report.SSL == nil {
		fmt.Fprintln(os.Stdout, "SSL Certificate  (skipped)")
		fmt.Fprintln(os.Stdout)
		return
	}

	if report.SSL.Error != "" {
		fmt.Fprintf(os.Stdout, "SSL Certificate ✗\n")
		fmt.Fprintf(os.Stdout, "  Error: %s\n", report.SSL.Error)
		fmt.Fprintln(os.Stdout)
		return
	}

	status := "✓"
	if !report.SSL.Valid || report.SSL.DaysLeft < 30 {
		status = "⚠"
	}
	if !report.SSL.Valid || report.SSL.DaysLeft < 0 {
		status = "✗"
	}
	fmt.Fprintf(os.Stdout, "SSL Certificate %s\n", status)
	fmt.Fprintf(os.Stdout, "  Valid: %s\n", report.SSL.Subject)
	fmt.Fprintf(os.Stdout, "  Issuer: %s\n", report.SSL.Issuer)
	fmt.Fprintf(os.Stdout, "  Expires: %s (%d days)\n", report.SSL.NotAfter.Format("2006-01-02"), report.SSL.DaysLeft)
	fmt.Fprintf(os.Stdout, "  TLS: %s\n", report.SSL.TLSVersion)
	hstsStr := "no"
	if report.SSL.HSTS {
		hstsStr = "yes"
	}
	fmt.Fprintf(os.Stdout, "  HSTS: %s\n", hstsStr)
	fmt.Fprintln(os.Stdout)
}

func printHTTPSection(report *cosmoflare.DiagnosticReport) {
	if report.HTTP == nil {
		fmt.Fprintln(os.Stdout, "HTTP Response  (skipped)")
		fmt.Fprintln(os.Stdout)
		return
	}

	if report.HTTP.Error != "" {
		fmt.Fprintf(os.Stdout, "HTTP Response ✗\n")
		fmt.Fprintf(os.Stdout, "  Error: %s\n", report.HTTP.Error)
		fmt.Fprintln(os.Stdout)
		return
	}

	status := "✓"
	if report.HTTP.StatusCode >= 400 {
		status = "⚠"
	}
	if report.HTTP.StatusCode >= 500 {
		status = "✗"
	}
	fmt.Fprintf(os.Stdout, "HTTP Response %s\n", status)
	fmt.Fprintf(os.Stdout, "  Status: %d\n", report.HTTP.StatusCode)
	fmt.Fprintf(os.Stdout, "  Response time: %dms\n", report.HTTP.ResponseTimeMs)
	if report.HTTP.Server != "" {
		fmt.Fprintf(os.Stdout, "  Server: %s\n", report.HTTP.Server)
	}
	if report.HTTP.CloudflareRay != "" {
		fmt.Fprintf(os.Stdout, "  CF-Ray: %s\n", report.HTTP.CloudflareRay)
	}
	if len(report.HTTP.RedirectChain) > 0 {
		fmt.Fprintf(os.Stdout, "  Redirects: %s\n", strings.Join(report.HTTP.RedirectChain, " -> "))
	}
	fmt.Fprintln(os.Stdout)
}

func printNSSection(report *cosmoflare.DiagnosticReport) {
	if report.Nameservers == nil {
		fmt.Fprintln(os.Stdout, "Nameservers  (skipped — no expected NS available)")
		fmt.Fprintln(os.Stdout)
		return
	}

	if report.Nameservers.Error != "" {
		fmt.Fprintf(os.Stdout, "Nameservers ✗\n")
		fmt.Fprintf(os.Stdout, "  Error: %s\n", report.Nameservers.Error)
		fmt.Fprintln(os.Stdout)
		return
	}

	status := "✓"
	if !report.Nameservers.Match {
		status = "✗"
	}
	fmt.Fprintf(os.Stdout, "Nameservers %s\n", status)
	fmt.Fprintf(os.Stdout, "  Expected: %s\n", strings.Join(report.Nameservers.Expected, ", "))
	fmt.Fprintf(os.Stdout, "  Actual:   %s\n", strings.Join(report.Nameservers.Actual, ", "))
	matchStr := "yes"
	if !report.Nameservers.Match {
		matchStr = "no"
	}
	fmt.Fprintf(os.Stdout, "  Match: %s\n", matchStr)
	fmt.Fprintln(os.Stdout)
}

func printIssuesSection(report *cosmoflare.DiagnosticReport) {
	if len(report.Issues) == 0 {
		return
	}

	fmt.Fprintf(os.Stdout, "ISSUES (%d):\n", len(report.Issues))
	for _, issue := range report.Issues {
		indicator := "?"
		switch issue.Severity {
		case "critical":
			indicator = "✗"
		case "warning":
			indicator = "⚠"
		case "info":
			indicator = "ℹ"
		}
		fmt.Fprintf(os.Stdout, "  %s [%s] %s\n", indicator, issue.Probe, issue.Message)
		if doctorFix && issue.Fix != "" {
			fmt.Fprintf(os.Stdout, "    FIX: %s\n", issue.Fix)
		}
	}
	fmt.Fprintln(os.Stdout)
}
