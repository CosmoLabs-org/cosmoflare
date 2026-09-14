package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli/ux"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var bucketDomainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Manage R2 bucket custom domains",
	Long: `Manage custom domains attached to Cloudflare R2 buckets.

A custom domain serves bucket traffic on a zone you own, with Cloudflare
managing the certificate and routing. Attaching a domain requires the domain
to exist as a zone in your account (or be a subdomain of one).

Commands:
  attach     Attach a custom domain to a bucket
  list       List custom domains attached to a bucket
  get        Show details of one attached domain
  verify     Wait for domain ownership and SSL to become active
  update     Change enabled/cipher/TLS settings of a domain
  detach     Remove a custom domain from a bucket

Examples:
  cosmoflare bucket domain attach my-bucket --domain cdn.example.com
  cosmoflare bucket domain list my-bucket --json`,
}

var bucketDomainAttachCmd = &cobra.Command{
	Use:   "attach [bucket]",
	Short: "Attach a custom domain to a bucket",
	Long: `Attach a custom domain to an R2 bucket.

The domain must resolve through a Cloudflare zone in your account. When
--zone-id is omitted, the zone is auto-resolved by matching the full domain
name first, then its parent (first label dropped).

Minimum TLS versions: 1.0 (API default), 1.1, 1.2, 1.3.

Examples:
  cosmoflare bucket domain attach my-bucket --domain cdn.example.com --min-tls 1.2`,
	RunE: runBucketDomainAttach,
}

var bucketDomainListCmd = &cobra.Command{
	Use:   "list [bucket]",
	Short: "List custom domains attached to a bucket",
	Long: `List all custom domains attached to an R2 bucket.

Shows the domain name, enabled state, ownership and SSL activation statuses,
and the minimum TLS version. Use --json for machine-readable output.

Examples:
  cosmoflare bucket domain list my-bucket`,
	RunE: runBucketDomainList,
}

var bucketDomainGetCmd = &cobra.Command{
	Use:   "get [bucket] [domain]",
	Short: "Show details of an attached custom domain",
	Long: `Show full details of one custom domain attached to an R2 bucket,
including ownership and SSL activation statuses.

Examples:
  cosmoflare bucket domain get my-bucket cdn.example.com --json`,
	RunE: runBucketDomainGet,
}

var bucketDomainVerifyCmd = &cobra.Command{
	Use:   "verify [bucket] [domain]",
	Short: "Wait for a custom domain to become active",
	Long: `Poll a custom domain until both ownership and SSL are active
(or reach a terminal state), printing the live statuses on each poll.

Examples:
  cosmoflare bucket domain verify my-bucket cdn.example.com --timeout 120s`,
	RunE: runBucketDomainVerify,
}

var bucketDomainUpdateCmd = &cobra.Command{
	Use:   "update [bucket] [domain]",
	Short: "Update settings of an attached custom domain",
	Long: `Update the enabled state, cipher list, or minimum TLS version
of an attached custom domain. Only the flags you pass are sent to the API.

Examples:
  cosmoflare bucket domain update my-bucket cdn.example.com --min-tls 1.3`,
	RunE: runBucketDomainUpdate,
}

var bucketDomainDetachCmd = &cobra.Command{
	Use:   "detach [bucket] [domain]",
	Short: "Remove a custom domain from a bucket",
	Long: `Detach a custom domain from an R2 bucket.

The zone and its DNS records are left untouched; the bucket simply stops
serving traffic on that domain.

Examples:
  cosmoflare bucket domain detach my-bucket cdn.example.com --force`,
	RunE: runBucketDomainDetach,
}

func init() {
	bucketCmd.AddCommand(bucketDomainCmd)

	bucketDomainCmd.AddCommand(bucketDomainAttachCmd)
	bucketDomainCmd.AddCommand(bucketDomainListCmd)
	bucketDomainCmd.AddCommand(bucketDomainGetCmd)
	bucketDomainCmd.AddCommand(bucketDomainVerifyCmd)
	bucketDomainCmd.AddCommand(bucketDomainUpdateCmd)
	bucketDomainCmd.AddCommand(bucketDomainDetachCmd)

	bucketDomainAttachCmd.Flags().String("domain", "", "Domain name to attach (required)")
	bucketDomainAttachCmd.Flags().String("zone-id", "", "Cloudflare zone ID (auto-resolved from the domain when omitted)")
	bucketDomainAttachCmd.Flags().String("min-tls", "", "Minimum TLS version: 1.0, 1.1, 1.2 or 1.3")
	bucketDomainAttachCmd.Flags().StringSlice("cipher", []string{}, "TLS cipher suites (repeatable)")
	bucketDomainAttachCmd.Flags().Bool("disabled", false, "Attach the domain in disabled state")

	bucketDomainVerifyCmd.Flags().String("timeout", "120s", "Maximum time to wait (Go duration, e.g. 120s or 5m)")

	bucketDomainUpdateCmd.Flags().Bool("enabled", false, "Enable the domain")
	bucketDomainUpdateCmd.Flags().Bool("disabled", false, "Disable the domain")
	bucketDomainUpdateCmd.Flags().String("min-tls", "", "Minimum TLS version: 1.0, 1.1, 1.2 or 1.3")
	bucketDomainUpdateCmd.Flags().StringSlice("cipher", []string{}, "TLS cipher suites (repeatable; replaces existing list)")

	bucketDomainDetachCmd.Flags().Bool("force", false, "Skip confirmation prompt")
}

// getBucketDomainService creates the bucket custom-domain service using the
// same global credentials the bucket commands rely on.
func getBucketDomainService(opts ...cosmoflare.BucketDomainOption) *cosmoflare.BucketDomainService {
	return cosmoflare.NewBucketDomainService(AccountID, APIToken, opts...)
}

func resolveZoneID(ctx context.Context, domain string) (string, error) {
	zoneSvc, err := getZoneService()
	if err != nil {
		return "", fmt.Errorf("failed to create zone service: %w", err)
	}
	return zoneSvc.ResolveIDForDomain(ctx, domain)
}

func bucketDomainStatuses(d *cosmoflare.BucketDomain) (ownership, ssl string) {
	if d == nil || d.Status == nil {
		return "unknown", "unknown"
	}
	if d.Status.Ownership == "" {
		d.Status.Ownership = "unknown"
	}
	if d.Status.SSL == "" {
		d.Status.SSL = "unknown"
	}
	return d.Status.Ownership, d.Status.SSL
}

func runBucketDomainAttach(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucket := args[0]
	domain, _ := cmd.Flags().GetString("domain")
	zoneID, _ := cmd.Flags().GetString("zone-id")
	minTLS, _ := cmd.Flags().GetString("min-tls")
	ciphers, _ := cmd.Flags().GetStringSlice("cipher")
	disabled, _ := cmd.Flags().GetBool("disabled")

	if domain == "" {
		return fmt.Errorf("--domain is required")
	}

	ctx := context.Background()
	if zoneID == "" {
		printInfo("Auto-resolving zone for %s ...", domain)
		resolved, err := resolveZoneID(ctx, domain)
		if err != nil {
			return outErrf("%s", err)
		}
		zoneID = resolved
		printInfo("Resolved zone: %s", zoneID)
	}

	svc := getBucketDomainService()

	req := cosmoflare.AttachBucketDomainRequest{
		Domain:  domain,
		Enabled: !disabled,
		ZoneID:  zoneID,
		Ciphers: ciphers,
		MinTLS:  minTLS,
	}
	d, err := svc.Attach(ctx, bucket, req)
	if err != nil {
		return outErr("failed to attach domain", err)
	}

	return outResult(d, func() {
		printSuccess("Domain '%s' attached to bucket '%s'", domain, bucket)
		printInfo("Ownership and SSL activate asynchronously; run 'cosmoflare bucket domain verify %s %s' to wait for activation", bucket, domain)
	})
}

func runBucketDomainList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("bucket name is required")
	}
	bucket := args[0]

	svc := getBucketDomainService()
	domains, err := svc.List(context.Background(), bucket)
	if err != nil {
		return outErr("failed to list domains", err)
	}

	return outResult(domains, func() {
		if len(domains) == 0 {
			printInfo("No custom domains attached to bucket '%s'", bucket)
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "DOMAIN\tENABLED\tOWNERSHIP\tSSL\tMIN-TLS")
		for _, d := range domains {
			ownership, ssl := bucketDomainStatuses(&d)
			enabled := "false"
			if d.Enabled {
				enabled = "true"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", d.Domain, enabled, ownership, ssl, d.MinTLS)
		}
		w.Flush()
		printInfo("Total: %d domain(s)", len(domains))
	})
}

func runBucketDomainGet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("bucket name and domain are required")
	}
	bucket, domain := args[0], args[1]

	svc := getBucketDomainService()
	d, err := svc.Get(context.Background(), bucket, domain)
	if err != nil {
		return outErr("failed to get domain", err)
	}

	return outResult(d, func() {
		ownership, ssl := bucketDomainStatuses(d)
		fmt.Printf("Domain:   %s\n", d.Domain)
		fmt.Printf("Enabled:  %t\n", d.Enabled)
		fmt.Printf("Ownership: %s\n", ownership)
		fmt.Printf("SSL:      %s\n", ssl)
		if d.MinTLS != "" {
			fmt.Printf("Min-TLS:  %s\n", d.MinTLS)
		}
		if d.ZoneID != "" {
			fmt.Printf("Zone ID:  %s\n", d.ZoneID)
		}
		if d.ZoneName != "" {
			fmt.Printf("Zone:     %s\n", d.ZoneName)
		}
		if len(d.Ciphers) > 0 {
			fmt.Printf("Ciphers:  %s\n", strings.Join(d.Ciphers, ", "))
		}
	})
}

func runBucketDomainVerify(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("bucket name and domain are required")
	}
	bucket, domain := args[0], args[1]

	timeoutStr, _ := cmd.Flags().GetString("timeout")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return fmt.Errorf("invalid --timeout duration %q: use a Go duration like 120s or 5m", timeoutStr)
	}

	svc := getBucketDomainService(cosmoflare.WithBucketDomainOnPoll(func(d *cosmoflare.BucketDomain) {
		ownership, ssl := bucketDomainStatuses(d)
		printInfo("ownership=%s ssl=%s", ownership, ssl)
	}))

	printInfo("Verifying domain '%s' on bucket '%s' (timeout %s) ...", domain, bucket, timeout)
	d, err := svc.Verify(context.Background(), bucket, domain, timeout)
	ownership, ssl := bucketDomainStatuses(d)
	if err != nil {
		printError("Verification did not complete: %v", err)
		fmt.Printf("ownership=%s ssl=%s\n", ownership, ssl)
		return outErrf("domain verification failed: %v (ownership=%s ssl=%s)", err, ownership, ssl)
	}

	return outResult(d, func() {
		printSuccess("Domain '%s' is active", domain)
		fmt.Printf("ownership=%s ssl=%s\n", ownership, ssl)
	})
}

func runBucketDomainUpdate(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("bucket name and domain are required")
	}
	bucket, domain := args[0], args[1]

	enabledSet, _ := cmd.Flags().GetBool("enabled")
	disabledSet, _ := cmd.Flags().GetBool("disabled")
	minTLS, _ := cmd.Flags().GetString("min-tls")
	ciphers, _ := cmd.Flags().GetStringSlice("cipher")

	if enabledSet && disabledSet {
		return fmt.Errorf("--enabled and --disabled are mutually exclusive")
	}
	anySet := cmd.Flags().Changed("enabled") || cmd.Flags().Changed("disabled") ||
		cmd.Flags().Changed("min-tls") || cmd.Flags().Changed("cipher")
	if !anySet {
		return fmt.Errorf("nothing to update: pass at least one of --enabled, --disabled, --min-tls, --cipher")
	}

	req := cosmoflare.UpdateBucketDomainRequest{}
	if cmd.Flags().Changed("enabled") || cmd.Flags().Changed("disabled") {
		v := enabledSet && !disabledSet
		req.Enabled = &v
	}
	if cmd.Flags().Changed("min-tls") {
		tls := minTLS
		req.MinTLS = &tls
	}
	if cmd.Flags().Changed("cipher") {
		req.Ciphers = ciphers
	}

	svc := getBucketDomainService()
	d, err := svc.Update(context.Background(), bucket, domain, req)
	if err != nil {
		return outErr("failed to update domain", err)
	}

	return outResult(d, func() {
		printSuccess("Domain '%s' updated", domain)
	})
}

func runBucketDomainDetach(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("bucket name and domain are required")
	}
	bucket, domain := args[0], args[1]
	force, _ := cmd.Flags().GetBool("force")

	if !force && !DryRun && !ux.Confirm(fmt.Sprintf("Detach domain '%s' from bucket '%s'?", domain, bucket)) {
		printInfo("Domain detach cancelled")
		return nil
	}

	svc := getBucketDomainService()
	if err := svc.Detach(context.Background(), bucket, domain); err != nil {
		return outErr("failed to detach domain", err)
	}

	return outPayload("Domain detached successfully", func() any {
		return map[string]string{"bucket": bucket, "domain": domain}
	}, func() {
		printSuccess("Domain '%s' detached from bucket '%s'", domain, bucket)
	})
}
