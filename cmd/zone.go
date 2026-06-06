package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var zoneCmd = &cobra.Command{
	Use:   "zone",
	Short: "Manage Cloudflare Zones",
	Long: `Zone management operations for Cloudflare DNS zones.

Commands:
  create    Create a new zone
  list      List all zones
  get       Get zone details
  settings  View zone settings
  delete    Delete a zone

Examples:
  cosmoflare zone create example.com
  cosmoflare zone list --json
  cosmoflare zone get abc123
  cosmoflare zone settings abc123
  cosmoflare zone delete abc123 --force`,
}

var (
	zoneType  string
	zoneForce bool
)

var zoneCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new zone",
	Long: `Add a new zone to your Cloudflare account.

The zone type can be "full" (default) for Cloudflare-managed DNS,
or "partial" (CNAME setup) for zones where DNS is managed elsewhere.

Examples:
  cosmoflare zone create example.com
  cosmoflare zone create example.com --type=partial
  cosmoflare zone create example.com --json`,
	RunE: runZoneCreate,
}

var zoneListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all zones",
	Long: `List all zones in the current account.

Examples:
  cosmoflare zone list
  cosmoflare zone list --json`,
	RunE: runZoneList,
}

var zoneGetCmd = &cobra.Command{
	Use:   "get [zone-id]",
	Short: "Get zone details",
	Long: `Get detailed information about a specific zone.

Examples:
  cosmoflare zone get abc123
  cosmoflare zone get abc123 --json`,
	RunE: runZoneGet,
}

var zoneSettingsCmd = &cobra.Command{
	Use:   "settings [zone-id]",
	Short: "View zone settings",
	Long: `View all configuration settings for a zone.

Displays settings like SSL mode, minification, caching level,
security level, and more.

Examples:
  cosmoflare zone settings abc123
  cosmoflare zone settings abc123 --json`,
	RunE: runZoneSettings,
}

var zoneDeleteCmd = &cobra.Command{
	Use:   "delete [zone-id]",
	Short: "Delete a zone",
	Long: `Delete a zone from your Cloudflare account.

WARNING: This action is irreversible. All DNS records, settings,
and associated configuration will be permanently removed.

Examples:
  cosmoflare zone delete abc123
  cosmoflare zone delete abc123 --force`,
	RunE: runZoneDelete,
}

func init() {
	rootCmd.AddCommand(zoneCmd)

	zoneCmd.AddCommand(zoneCreateCmd)
	zoneCmd.AddCommand(zoneListCmd)
	zoneCmd.AddCommand(zoneGetCmd)
	zoneCmd.AddCommand(zoneSettingsCmd)
	zoneCmd.AddCommand(zoneDeleteCmd)

	zoneCreateCmd.Flags().StringVar(&zoneType, "type", "full", "Zone type: full (default) or partial")

	zoneDeleteCmd.Flags().BoolVar(&zoneForce, "force", false, "Skip confirmation prompt")
}

func getZoneService() (*cosmoflare.ZoneService, error) {
	return cosmoflare.NewZoneServiceFromCreds(AccountID, APIToken)
}

func runZoneCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone name is required")
	}
	name := args[0]

	svc, err := getZoneService()
	if err != nil {
		return fmt.Errorf("failed to create zone service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create zone", map[string]string{"name": name, "type": zoneType})
		}
		printInfo("DRY RUN: Would create zone '%s' (type: %s)", name, zoneType)
		return nil
	}

	zone, err := svc.Create(context.Background(), name, zoneType)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create zone: %v", err))
		}
		return fmt.Errorf("failed to create zone: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Zone created successfully", zone)
	}

	printSuccess("Zone '%s' created (ID: %s)", zone.Name, zone.ID)
	printInfo("Status: %s", zone.Status)
	if len(zone.NameServers) > 0 {
		printInfo("Name servers: %s", strings.Join(zone.NameServers, ", "))
	}
	return nil
}

func runZoneList(cmd *cobra.Command, args []string) error {
	svc, err := getZoneService()
	if err != nil {
		return fmt.Errorf("failed to create zone service: %w", err)
	}

	zones, err := svc.List(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list zones: %v", err))
		}
		return fmt.Errorf("failed to list zones: %w", err)
	}

	if JSONOutput {
		return printJSON(zones)
	}

	if len(zones) == 0 {
		printInfo("No zones found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tTYPE\tPLAN")
	for _, z := range zones {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			z.ID,
			z.Name,
			z.Status,
			z.Type,
			z.Plan.Name,
		)
	}
	w.Flush()

	printInfo("Total: %d zone(s)", len(zones))
	return nil
}

func runZoneGet(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getZoneService()
	if err != nil {
		return fmt.Errorf("failed to create zone service: %w", err)
	}

	zone, err := svc.Get(context.Background(), zoneID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get zone: %v", err))
		}
		return fmt.Errorf("failed to get zone: %w", err)
	}

	if JSONOutput {
		return printJSON(zone)
	}

	fmt.Printf("Zone: %s\n", zone.Name)
	fmt.Printf("ID: %s\n", zone.ID)
	fmt.Printf("Status: %s\n", zone.Status)
	fmt.Printf("Type: %s\n", zone.Type)
	fmt.Printf("Paused: %t\n", zone.Paused)
	fmt.Printf("Plan: %s\n", zone.Plan.Name)
	if len(zone.NameServers) > 0 {
		fmt.Printf("Name Servers: %s\n", strings.Join(zone.NameServers, ", "))
	}
	if len(zone.OriginalNS) > 0 {
		fmt.Printf("Original NS: %s\n", strings.Join(zone.OriginalNS, ", "))
	}
	fmt.Printf("Created: %s\n", zone.CreatedOn.Format("2006-01-02 15:04:05"))
	fmt.Printf("Modified: %s\n", zone.ModifiedOn.Format("2006-01-02 15:04:05"))
	return nil
}

func runZoneSettings(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getZoneService()
	if err != nil {
		return fmt.Errorf("failed to create zone service: %w", err)
	}

	settings, err := svc.GetSettings(context.Background(), zoneID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get zone settings: %v", err))
		}
		return fmt.Errorf("failed to get zone settings: %w", err)
	}

	if JSONOutput {
		return printJSON(settings)
	}

	if len(settings) == 0 {
		printInfo("No settings found for zone '%s'", zoneID)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tVALUE\tEDITABLE\tMODIFIED")
	for _, s := range settings {
		editable := "no"
		if s.Editable {
			editable = "yes"
		}
		modified := "-"
		if s.ModifiedOn != "" {
			modified = s.ModifiedOn
		}
		fmt.Fprintf(w, "%s\t%v\t%s\t%s\n",
			s.ID,
			s.Value,
			editable,
			modified,
		)
	}
	w.Flush()

	printInfo("Total: %d setting(s)", len(settings))
	return nil
}

func runZoneDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	if !zoneForce && !DryRun {
		fmt.Printf("Are you sure you want to delete zone '%s'? This removes all DNS records and settings. [y/N]: ", zoneID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Zone deletion cancelled")
			return nil
		}
	}

	svc, err := getZoneService()
	if err != nil {
		return fmt.Errorf("failed to create zone service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete zone", map[string]string{"id": zoneID})
		}
		printInfo("DRY RUN: Would delete zone '%s'", zoneID)
		return nil
	}

	if err := svc.Delete(context.Background(), zoneID); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete zone: %v", err))
		}
		return fmt.Errorf("failed to delete zone: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Zone deleted successfully", map[string]string{"id": zoneID})
	}
	printSuccess("Zone '%s' deleted successfully!", zoneID)
	return nil
}
