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
	Long: `Deep diagnostic probes for domain health analysis.

Runs 4 probes against a domain or zone:
  1. DNS Propagation — queries 6 public resolvers for record consistency
  2. SSL Certificate — inspects TLS certificate chain, expiry, and HSTS
  3. HTTP Response — checks status code, response time, redirect chain, cf-ray
  4. Nameserver Consistency — compares assigned NS with DNS-returned NS

Results are presented with pass/warn/fail indicators and an overall health score.
Use --fix to see actionable cosmoflare commands for each issue found.

This command is designed for both human operators and AI agents:
  - Human-readable output with colored status indicators by default
  - --json provides structured diagnostic data for automated pipelines
  - --fix suggestions are valid cosmoflare commands that can be executed directly
  - Exit code reflects health: 0=healthy/warning, 1=critical

Examples:
  cosmoflare doctor example.com              # Full diagnostics for a domain
  cosmoflare doctor example.com --fix        # Include fix suggestions
  cosmoflare doctor example.com --json       # Machine-readable output
  cosmoflare doctor --all                    # Run on all domains (slow)
  cosmoflare doctor --all --json             # All domains, JSON output

  # Use with other cosmoflare commands:
  cosmoflare doctor example.com --fix --json | jq '.issues[].fix'
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
	doctor := cosmoflare.NewDoctorService(10 * time.Second)

	if doctorAll {
		return runDoctorAll(ctx, doctor)
	}

	return runDoctorSingle(ctx, doctor, args[0])
}

func runDoctorAll(ctx context.Context, doctor *cosmoflare.DoctorService) error {
	zoneSvc, err := getZoneService()
	if err != nil {
		return fmt.Errorf("failed to create zone service: %w", err)
	}

	zones, err := zoneSvc.List(ctx)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list zones: %v", err))
		}
		return fmt.Errorf("failed to list zones: %w", err)
	}

	if len(zones) == 0 {
		if JSONOutput {
			return printSuccessJSON("No zones found", nil)
		}
		printInfo("No zones found in account")
		return nil
	}

	// For JSON output, collect all reports into a slice.
	type allDomainsReport struct {
		Reports []*cosmoflare.DiagnosticReport `json:"reports"`
		Summary string                    `json:"summary"`
	}
	var reports []*cosmoflare.DiagnosticReport
	var hasWarning, hasCritical bool

	for i, zone := range zones {
		if !JSONOutput && i > 0 {
			fmt.Fprintln(os.Stdout)
		}

		report, err := doctor.RunDiagnostics(ctx, zone.Name, zone.NameServers)
		if err != nil {
			if JSONOutput {
				// Continue collecting; include error in reports.
				reports = append(reports, &cosmoflare.DiagnosticReport{
					Domain:    zone.Name,
					Timestamp: time.Now(),
					Score:     "critical",
					Issues: []cosmoflare.DiagnosticIssue{{
						Probe:    "system",
						Severity: "critical",
						Message:  fmt.Sprintf("Failed to run diagnostics: %v", err),
					}},
				})
				hasCritical = true
				continue
			}
			printWarning("Failed to diagnose %s: %v", zone.Name, err)
			continue
		}

		reports = append(reports, report)
		switch report.Score {
		case "critical":
			hasCritical = true
		case "warning":
			hasWarning = true
		}

		if !JSONOutput {
			printDoctorReport(report)
		}
	}

	if JSONOutput {
		summary := fmt.Sprintf("Diagnosed %d domain(s)", len(reports))
		return printJSON(allDomainsReport{
			Reports: reports,
			Summary: summary,
		})
	}

	if hasCritical {
		return fmt.Errorf("one or more domains have critical health issues")
	}
	if hasWarning {
		printWarning("One or more domains have warnings — review output above")
	}
	return nil
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
