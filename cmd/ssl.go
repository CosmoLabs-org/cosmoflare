package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var sslCmd = &cobra.Command{
	Use:   "ssl",
	Short: "Manage Cloudflare SSL/TLS settings",
	Long: `SSL/TLS certificate and settings management for Cloudflare zones.

Commands:
  status      View current SSL/TLS encryption mode
  settings    View SSL/TLS-related zone settings
  update      Update SSL/TLS settings
  verify      View certificate verification status

SSL/TLS settings are zone-scoped, so a zone ID is required for all operations.

Examples:
  cosmoflare ssl status ZONE_ID
  cosmoflare ssl settings ZONE_ID --json
  cosmoflare ssl update ZONE_ID --mode=full --min-tls=1.2 --always-https
  cosmoflare ssl verify ZONE_ID`,
}

var (
	sslMode         string
	sslMinTLS       string
	sslAlwaysHTTPS  bool
	sslAutoRewrites bool
)

var sslStatusCmd = &cobra.Command{
	Use:   "status [zone-id]",
	Short: "View current SSL/TLS encryption mode",
	Long: `View the current SSL/TLS encryption mode for a zone.

Possible values: off, flexible, full, strict (Full Strict).

Examples:
  cosmoflare ssl status ZONE_ID
  cosmoflare ssl status ZONE_ID --json`,
	RunE: runSSLStatus,
}

var sslSettingsCmd = &cobra.Command{
	Use:   "settings [zone-id]",
	Short: "View SSL/TLS-related zone settings",
	Long: `View SSL/TLS-related settings including minimum TLS version,
Always Use HTTPS, Automatic HTTPS Rewrites, and Universal SSL.

Examples:
  cosmoflare ssl settings ZONE_ID
  cosmoflare ssl settings ZONE_ID --json`,
	RunE: runSSLSettings,
}

var sslUpdateCmd = &cobra.Command{
	Use:   "update [zone-id]",
	Short: "Update SSL/TLS settings",
	Long: `Update SSL/TLS encryption mode and related settings.

SSL modes: off, flexible, full, strict (Full Strict)
Min TLS versions: 1.0, 1.1, 1.2, 1.3

Examples:
  cosmoflare ssl update ZONE_ID --mode=full
  cosmoflare ssl update ZONE_ID --min-tls=1.2
  cosmoflare ssl update ZONE_ID --always-https
  cosmoflare ssl update ZONE_ID --mode=strict --min-tls=1.2 --always-https --auto-rewrites
  cosmoflare ssl update ZONE_ID --mode=full --json`,
	RunE: runSSLUpdate,
}

var sslVerifyCmd = &cobra.Command{
	Use:   "verify [zone-id]",
	Short: "View certificate verification status",
	Long: `View Universal SSL certificate verification details for a zone.

Shows certificate status, verification type, and validation method for
each certificate pack.

Examples:
  cosmoflare ssl verify ZONE_ID
  cosmoflare ssl verify ZONE_ID --json`,
	RunE: runSSLVerify,
}

func init() {
	rootCmd.AddCommand(sslCmd)

	sslCmd.AddCommand(sslStatusCmd)
	sslCmd.AddCommand(sslSettingsCmd)
	sslCmd.AddCommand(sslUpdateCmd)
	sslCmd.AddCommand(sslVerifyCmd)

	sslUpdateCmd.Flags().StringVar(&sslMode, "mode", "", "SSL/TLS encryption mode (off, flexible, full, strict)")
	sslUpdateCmd.Flags().StringVar(&sslMinTLS, "min-tls", "", "Minimum TLS version (1.0, 1.1, 1.2, 1.3)")
	sslUpdateCmd.Flags().BoolVar(&sslAlwaysHTTPS, "always-https", false, "Enable Always Use HTTPS")
	sslUpdateCmd.Flags().BoolVar(&sslAutoRewrites, "auto-rewrites", false, "Enable Automatic HTTPS Rewrites")
}

func getSSLService(zoneID string) (*cosmoflare.SSLService, error) {
	return cosmoflare.NewSSLServiceFromCreds(zoneID, APIToken)
}

func runSSLStatus(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getSSLService(zoneID)
	if err != nil {
		return outErr("failed to create SSL service", err)
	}

	status, err := svc.GetSSL(context.Background())
	if err != nil {
		return outErr("failed to get SSL status", err)
	}

	return outResult(status, func() {
		fmt.Printf("SSL/TLS Encryption Mode: %s\n", status.Value)
		fmt.Printf("Certificate Status:      %s\n", status.CertificateStatus)
		fmt.Printf("Editable:                %v\n", status.Editable)
		if status.ModifiedOn != "" {
			fmt.Printf("Modified:                %s\n", status.ModifiedOn)
		}
	})
}

func runSSLSettings(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getSSLService(zoneID)
	if err != nil {
		return outErr("failed to create SSL service", err)
	}

	settings, err := svc.GetSettings(context.Background())
	if err != nil {
		return outErr("failed to get SSL settings", err)
	}

	return outResult(settings, func() {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SETTING\tVALUE")
		fmt.Fprintf(w, "Minimum TLS Version\t%s\n", settings.MinTLSVersion)
		fmt.Fprintf(w, "Always Use HTTPS\t%v\n", settings.AlwaysUseHTTPS)
		fmt.Fprintf(w, "Automatic HTTPS Rewrites\t%v\n", settings.AutomaticHTTPSRewrites)
		fmt.Fprintf(w, "Universal SSL\t%v\n", settings.UniversalSSL)
		w.Flush()
	})
}

func runSSLUpdate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	hasMode := sslMode != ""
	hasSettings := cmd.Flags().Changed("min-tls") || cmd.Flags().Changed("always-https") || cmd.Flags().Changed("auto-rewrites")

	if !hasMode && !hasSettings {
		return fmt.Errorf("at least one update flag is required (--mode, --min-tls, --always-https, --auto-rewrites)")
	}

	svc, err := getSSLService(zoneID)
	if err != nil {
		return outErr("failed to create SSL service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would update SSL settings", func() any {
			return map[string]string{"zone_id": zoneID}
		}, func() {
			printInfo("DRY RUN: Would update SSL settings for zone '%s'", zoneID)
		})
	}

	if hasMode {
		status, err := svc.UpdateSSL(context.Background(), sslMode)
		if err != nil {
			return outErr("failed to update SSL mode", err)
		}
		if err := outResult(status, func() {
			printSuccess("SSL/TLS mode updated to '%s'", status.Value)
		}); err != nil {
			return err
		}
	}

	if hasSettings {
		var opts []cosmoflare.SSLOption
		if cmd.Flags().Changed("min-tls") {
			opts = append(opts, cosmoflare.WithMinTLSVersion(sslMinTLS))
		}
		if cmd.Flags().Changed("always-https") {
			opts = append(opts, cosmoflare.WithAlwaysHTTPS(sslAlwaysHTTPS))
		}
		if cmd.Flags().Changed("auto-rewrites") {
			opts = append(opts, cosmoflare.WithAutoHTTPSRewrites(sslAutoRewrites))
		}

		if err := svc.UpdateSettings(context.Background(), opts...); err != nil {
			return outErr("failed to update SSL settings", err)
		}
		if !JSONOutput {
			printSuccess("SSL/TLS settings updated for zone '%s'", zoneID)
		}
	}

	return nil
}

func runSSLVerify(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getSSLService(zoneID)
	if err != nil {
		return outErr("failed to create SSL service", err)
	}

	verifications, err := svc.GetVerification(context.Background())
	if err != nil {
		return outErr("failed to get SSL verification", err)
	}

	return outResult(verifications, func() {
		if len(verifications) == 0 {
			printInfo("No SSL verification entries found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "CERT PACK\tSTATUS\tTYPE\tMETHOD\tVERIFIED\tBRAND CHECK")
		for _, v := range verifications {
			verified := "no"
			if v.VerificationStatus {
				verified = "yes"
			}
			brandCheck := "no"
			if v.BrandCheck {
				brandCheck = "yes"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				v.CertPackUUID,
				v.CertificateStatus,
				v.VerificationType,
				v.ValidationMethod,
				verified,
				brandCheck,
			)
		}
		w.Flush()

		printInfo("Total: %d verification(s)", len(verifications))
	})
}
