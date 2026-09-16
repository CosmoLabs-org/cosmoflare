package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var sslCustomHostnameCmd = &cobra.Command{
	Use:   "custom-hostname",
	Short: "Manage SSL for SaaS custom hostnames",
	Long: `Custom hostname (SSL for SaaS) management for a Cloudflare zone.

Custom hostnames let customers point their own hostnames at your zone.
The get verb is the way to check verification status.

Commands:
  list      List custom hostnames
  create    Create a custom hostname
  get       Get a custom hostname and its verification status
  update    Update a custom hostname
  delete    Delete a custom hostname

Custom hostnames are zone-scoped, so a zone ID is required for all operations.

Examples:
  cosmoflare ssl custom-hostname list ZONE_ID
  cosmoflare ssl custom-hostname create ZONE_ID app.customer.com --origin origin.example.com
  cosmoflare ssl custom-hostname get ZONE_ID HOSTNAME_ID --json
  cosmoflare ssl custom-hostname update ZONE_ID HOSTNAME_ID --origin new-origin.example.com
  cosmoflare ssl custom-hostname delete ZONE_ID HOSTNAME_ID`,
}

var (
	chHostnameFilter string
	chOrigin         string
	chUpdateHostname string
)

var chListCmd = &cobra.Command{
	Use:   "list [zone-id]",
	Short: "List custom hostnames",
	Long: `List SSL for SaaS custom hostnames configured on a zone.

Use --hostname to filter results to hostnames containing a substring.

Examples:
  cosmoflare ssl custom-hostname list ZONE_ID
  cosmoflare ssl custom-hostname list ZONE_ID --hostname customer.com --json`,
	RunE: runCHList,
}

var chCreateCmd = &cobra.Command{
	Use:   "create [zone-id] [hostname]",
	Short: "Create a custom hostname",
	Long: `Create a new custom hostname on a zone.

Optionally point the hostname at a custom origin server with --origin.

Examples:
  cosmoflare ssl custom-hostname create ZONE_ID app.customer.com
  cosmoflare ssl custom-hostname create ZONE_ID app.customer.com --origin origin.example.com --json`,
	RunE: runCHCreate,
}

var chGetCmd = &cobra.Command{
	Use:   "get [zone-id] [hostname-id]",
	Short: "Get a custom hostname and its verification status",
	Long: `Get a single custom hostname including its full verification state.

Shows hostname status, verification status, verification type, SSL status,
and any verification errors.

Examples:
  cosmoflare ssl custom-hostname get ZONE_ID HOSTNAME_ID
  cosmoflare ssl custom-hostname get ZONE_ID HOSTNAME_ID --json`,
	RunE: runCHGet,
}

var chUpdateCmd = &cobra.Command{
	Use:   "update [zone-id] [hostname-id]",
	Short: "Update a custom hostname",
	Long: `Update a custom hostname's hostname or custom origin server.

At least one of --hostname or --origin is required.

Examples:
  cosmoflare ssl custom-hostname update ZONE_ID HOSTNAME_ID --origin new-origin.example.com
  cosmoflare ssl custom-hostname update ZONE_ID HOSTNAME_ID --hostname new.customer.com --json`,
	RunE: runCHUpdate,
}

var chDeleteCmd = &cobra.Command{
	Use:   "delete [zone-id] [hostname-id]",
	Short: "Delete a custom hostname",
	Long: `Delete a custom hostname from a zone.

Examples:
  cosmoflare ssl custom-hostname delete ZONE_ID HOSTNAME_ID
  cosmoflare ssl custom-hostname delete ZONE_ID HOSTNAME_ID --json`,
	RunE: runCHDelete,
}

func init() {
	sslCmd.AddCommand(sslCustomHostnameCmd)

	sslCustomHostnameCmd.AddCommand(chListCmd)
	sslCustomHostnameCmd.AddCommand(chCreateCmd)
	sslCustomHostnameCmd.AddCommand(chGetCmd)
	sslCustomHostnameCmd.AddCommand(chUpdateCmd)
	sslCustomHostnameCmd.AddCommand(chDeleteCmd)

	chListCmd.Flags().StringVar(&chHostnameFilter, "hostname", "", "Filter by hostname substring")
	chCreateCmd.Flags().StringVar(&chOrigin, "origin", "", "Custom origin server for the hostname")
	chUpdateCmd.Flags().StringVar(&chUpdateHostname, "hostname", "", "New hostname value")
	chUpdateCmd.Flags().StringVar(&chOrigin, "origin", "", "Custom origin server for the hostname")
}

func getSSLCustomHostnameService(zoneID string) (*cosmoflare.SSLCustomHostnameService, error) {
	return cosmoflare.NewSSLCustomHostnameServiceFromCreds(zoneID, APIToken)
}

func runCHList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}

	svc, err := getSSLCustomHostnameService(args[0])
	if err != nil {
		return outErr("failed to create SSL custom hostname service", err)
	}

	hostnames, err := svc.List(context.Background(), cosmoflare.CustomHostnameListOptions{Hostname: chHostnameFilter})
	if err != nil {
		return outErr("failed to list custom hostnames", err)
	}

	return outResult(hostnames, func() {
		if len(hostnames) == 0 {
			printInfo("No custom hostnames found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tHOSTNAME\tSTATUS\tVERIFICATION\tSSL")
		for _, h := range hostnames {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				h.ID,
				h.Hostname,
				h.Status,
				h.VerificationStatus,
				h.SSLStatus,
			)
		}
		w.Flush()

		printInfo("Total: %d custom hostname(s)", len(hostnames))
	})
}

func runCHCreate(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and hostname are required")
	}
	zoneID, hostname := args[0], args[1]

	svc, err := getSSLCustomHostnameService(zoneID)
	if err != nil {
		return outErr("failed to create SSL custom hostname service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would create custom hostname", func() any {
			return map[string]string{"zone_id": zoneID, "hostname": hostname, "custom_origin_server": chOrigin}
		}, func() {
			printInfo("DRY RUN: Would create custom hostname '%s'", hostname)
		})
	}

	created, err := svc.Create(context.Background(), hostname, chOrigin)
	if err != nil {
		return outErr("failed to create custom hostname", err)
	}

	return outResult(created, func() {
		printSuccess("Custom hostname '%s' created (status: %s)", created.Hostname, created.Status)
	})
}

func runCHGet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and hostname ID are required")
	}

	svc, err := getSSLCustomHostnameService(args[0])
	if err != nil {
		return outErr("failed to create SSL custom hostname service", err)
	}

	hostname, err := svc.Get(context.Background(), args[1])
	if err != nil {
		return outErr("failed to get custom hostname", err)
	}

	return outResult(hostname, func() {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "FIELD\tVALUE")
		fmt.Fprintf(w, "ID\t%s\n", hostname.ID)
		fmt.Fprintf(w, "Hostname\t%s\n", hostname.Hostname)
		fmt.Fprintf(w, "Status\t%s\n", hostname.Status)
		if hostname.CustomOriginServer != "" {
			fmt.Fprintf(w, "Custom Origin Server\t%s\n", hostname.CustomOriginServer)
		}
		fmt.Fprintf(w, "Verification Status\t%s\n", hostname.VerificationStatus)
		if hostname.VerificationType != "" {
			fmt.Fprintf(w, "Verification Type\t%s\n", hostname.VerificationType)
		}
		fmt.Fprintf(w, "SSL Status\t%s\n", hostname.SSLStatus)
		for _, ve := range hostname.VerificationErrors {
			fmt.Fprintf(w, "Verification Error\t%s\n", ve)
		}
		if hostname.CreatedOn != "" {
			fmt.Fprintf(w, "Created\t%s\n", hostname.CreatedOn)
		}
		w.Flush()
	})
}

func runCHUpdate(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and hostname ID are required")
	}

	changedHostname := cmd.Flags().Changed("hostname")
	changedOrigin := cmd.Flags().Changed("origin")
	if !changedHostname && !changedOrigin {
		return fmt.Errorf("at least one update flag is required (--hostname, --origin)")
	}

	svc, err := getSSLCustomHostnameService(args[0])
	if err != nil {
		return outErr("failed to create SSL custom hostname service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would update custom hostname", func() any {
			return map[string]string{"zone_id": args[0], "id": args[1]}
		}, func() {
			printInfo("DRY RUN: Would update custom hostname '%s'", args[1])
		})
	}

	opts := cosmoflare.CustomHostnameUpdateOptions{}
	if changedHostname {
		opts.Hostname = &chUpdateHostname
	}
	if changedOrigin {
		opts.CustomOriginServer = &chOrigin
	}

	updated, err := svc.Update(context.Background(), args[1], opts)
	if err != nil {
		return outErr("failed to update custom hostname", err)
	}

	return outResult(updated, func() {
		printSuccess("Custom hostname '%s' updated", updated.Hostname)
	})
}

func runCHDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and hostname ID are required")
	}

	svc, err := getSSLCustomHostnameService(args[0])
	if err != nil {
		return outErr("failed to create SSL custom hostname service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would delete custom hostname", func() any {
			return map[string]string{"zone_id": args[0], "id": args[1]}
		}, func() {
			printInfo("DRY RUN: Would delete custom hostname '%s'", args[1])
		})
	}

	if err := svc.Delete(context.Background(), args[1]); err != nil {
		return outErr("failed to delete custom hostname", err)
	}

	if !JSONOutput {
		printSuccess("Custom hostname '%s' deleted", args[1])
	}
	return nil
}
