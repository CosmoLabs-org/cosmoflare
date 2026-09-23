package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli/ux"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// Spectrum is zone-scoped; a zone ID is required for all operations.
// Spectrum permission pending FEAT-011 dataset.

var spectrumCmd = &cobra.Command{
	Use:   "spectrum",
	Short: "Manage Cloudflare Spectrum applications",
	Long: `Spectrum application management for Cloudflare zones.

Spectrum proxies arbitrary TCP/UDP/HTTP traffic through Cloudflare's edge,
protecting non-web origins with DDoS mitigation and TLS termination.

Commands:
  app       Manage Spectrum applications (create, list, get, update, delete)

NOTE: Spectrum is an Enterprise-gated product; TCP monitoring in
particular requires an Enterprise plan (dataset note).

Examples:
  cosmoflare spectrum app create ZONE_ID --name=ssh.example.com --protocol=tcp/22 --origin-port=22 --origin-direct=10.0.0.1:22
  cosmoflare spectrum app list ZONE_ID --json
  cosmoflare spectrum app get ZONE_ID APP_ID
  cosmoflare spectrum app update ZONE_ID APP_ID --origin-port=2222
  cosmoflare spectrum app delete ZONE_ID APP_ID --force`,
}

var spectrumAppCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage Spectrum applications",
	Long: `Manage Spectrum applications for a zone.

Commands:
  create    Create a Spectrum application
  list      List Spectrum applications
  get       Get a Spectrum application
  update    Update a Spectrum application
  delete    Delete a Spectrum application

NOTE: Spectrum is an Enterprise-gated product; TCP monitoring in
particular requires an Enterprise plan (dataset note).

Examples:
  cosmoflare spectrum app create ZONE_ID --name=ssh.example.com --protocol=tcp/22 --origin-port=22 --origin-direct=10.0.0.1:22`,
}

var (
	spName          string
	spProtocol      string
	spOriginPort    string
	spOriginDirect  []string
	spOriginDNS     string
	spDNSType       string
	spTrafficType   string
	spTLS           string
	spProxyProtocol string
	spArgo          bool
	spIPv4          bool
	spIPFirewall    bool
	spForce         bool
)

var spectrumAppCreateCmd = &cobra.Command{
	Use:   "create [zone-id]",
	Short: "Create a Spectrum application",
	Long: `Create a new Spectrum application in a Cloudflare zone.

Required: --name, --protocol, --origin-port, plus an origin (--origin-direct
or --origin-dns).

Sensible defaults when flags are omitted: --dns-type=CNAME,
--traffic-type=tcp (derived from the protocol), TLS termination uses the
API default ("full").

NOTE: Spectrum is an Enterprise-gated product; TCP monitoring in
particular requires an Enterprise plan.

Examples:
  cosmoflare spectrum app create ZONE_ID --name=ssh.example.com --protocol=tcp/22 --origin-port=22 --origin-direct=10.0.0.1:22
  cosmoflare spectrum app create ZONE_ID --name=minecraft.example.com --protocol=tcp/25565 --origin-port=25565 --origin-direct=10.0.0.5:25565 --ip-firewall
  cosmoflare spectrum app create ZONE_ID --name=ssh.example.com --protocol=tcp/22 --origin-port=22 --origin-dns=origin.internal --json`,
	Args: prefixedResourceArgs(cobra.MinimumNArgs(1)),
	RunE: runSpectrumAppCreate,
}

var spectrumAppListCmd = &cobra.Command{
	Use:   "list [zone-id]",
	Short: "List Spectrum applications",
	Long: `List all Spectrum applications in a zone.

Examples:
  cosmoflare spectrum app list ZONE_ID
  cosmoflare spectrum app list ZONE_ID --json`,
	Args: prefixedResourceArgs(cobra.MinimumNArgs(1)),
	RunE: runSpectrumAppList,
}

var spectrumAppGetCmd = &cobra.Command{
	Use:   "get [zone-id] [app-id]",
	Short: "Get a Spectrum application",
	Long: `Get details of a single Spectrum application by its ID.

Examples:
  cosmoflare spectrum app get ZONE_ID APP_ID
  cosmoflare spectrum app get ZONE_ID APP_ID --json`,
	Args: prefixedResourceArgs(cobra.MinimumNArgs(2)),
	RunE: runSpectrumAppGet,
}

var spectrumAppUpdateCmd = &cobra.Command{
	Use:   "update [zone-id] [app-id]",
	Short: "Update a Spectrum application",
	Long: `Update an existing Spectrum application.

Provide only the fields you want to change. Unspecified fields keep their
current values (the PUT replace is assembled from the live app first).

NOTE: Spectrum is an Enterprise-gated product; TCP monitoring in
particular requires an Enterprise plan.

Examples:
  cosmoflare spectrum app update ZONE_ID APP_ID --origin-port=2222
  cosmoflare spectrum app update ZONE_ID APP_ID --protocol=tcp/2022 --tls=strict
  cosmoflare spectrum app update ZONE_ID APP_ID --argo-smart-routing --json`,
	Args: prefixedResourceArgs(cobra.MinimumNArgs(2)),
	RunE: runSpectrumAppUpdate,
}

var spectrumAppDeleteCmd = &cobra.Command{
	Use:   "delete [zone-id] [app-id]",
	Short: "Delete a Spectrum application",
	Long: `Delete a Spectrum application from a zone.

WARNING: This action is irreversible. Traffic for the application's DNS
name stops being proxied through Spectrum immediately.

Examples:
  cosmoflare spectrum app delete ZONE_ID APP_ID
  cosmoflare spectrum app delete ZONE_ID APP_ID --force`,
	Args: prefixedResourceArgs(cobra.MinimumNArgs(2)),
	RunE: runSpectrumAppDelete,
}

func init() {
	rootCmd.AddCommand(spectrumCmd)
	spectrumCmd.AddCommand(spectrumAppCmd)

	spectrumAppCmd.AddCommand(spectrumAppCreateCmd)
	spectrumAppCmd.AddCommand(spectrumAppListCmd)
	spectrumAppCmd.AddCommand(spectrumAppGetCmd)
	spectrumAppCmd.AddCommand(spectrumAppUpdateCmd)
	spectrumAppCmd.AddCommand(spectrumAppDeleteCmd)

	// Create flags
	spectrumAppCreateCmd.Flags().StringVar(&spName, "name", "", "DNS name for the application (e.g., ssh.example.com)")
	spectrumAppCreateCmd.Flags().StringVar(&spProtocol, "protocol", "", "Edge protocol and port (e.g., tcp/22, udp/53, http/80) (required)")
	spectrumAppCreateCmd.Flags().StringVar(&spOriginPort, "origin-port", "", "Origin port or range (e.g., 22 or 8000-9000) (required)")
	spectrumAppCreateCmd.Flags().StringSliceVar(&spOriginDirect, "origin-direct", nil, "Direct origin address(es) (comma-separated or repeated)")
	spectrumAppCreateCmd.Flags().StringVar(&spOriginDNS, "origin-dns", "", "Origin DNS name (alternative to --origin-direct)")
	spectrumAppCreateCmd.Flags().StringVar(&spDNSType, "dns-type", "CNAME", "External DNS record type (default CNAME)")
	spectrumAppCreateCmd.Flags().StringVar(&spTrafficType, "traffic-type", "tcp", "Traffic type: tcp, udp, http (default tcp)")
	spectrumAppCreateCmd.Flags().StringVar(&spTLS, "tls", "", "TLS termination mode (e.g., full, strict)")
	spectrumAppCreateCmd.Flags().StringVar(&spProxyProtocol, "proxy-protocol", "", "Proxy Protocol v1/v2: off, v1, v2, simple")
	spectrumAppCreateCmd.Flags().BoolVar(&spArgo, "argo-smart-routing", false, "Enable Argo Smart Routing")
	spectrumAppCreateCmd.Flags().BoolVar(&spIPv4, "ipv4", false, "Enable IPv4 for the application")
	spectrumAppCreateCmd.Flags().BoolVar(&spIPFirewall, "ip-firewall", false, "Enable the IP firewall (recommended)")
	_ = spectrumAppCreateCmd.MarkFlagRequired("name")
	_ = spectrumAppCreateCmd.MarkFlagRequired("protocol")
	_ = spectrumAppCreateCmd.MarkFlagRequired("origin-port")

	// Update flags: every field is optional; at least one must change.
	spectrumAppUpdateCmd.Flags().StringVar(&spProtocol, "protocol", "", "Edge protocol and port (e.g., tcp/22)")
	spectrumAppUpdateCmd.Flags().StringSliceVar(&spOriginDirect, "origin-direct", nil, "Direct origin address(es)")
	spectrumAppUpdateCmd.Flags().StringVar(&spOriginDNS, "origin-dns", "", "Origin DNS name (empty string to clear)")
	spectrumAppUpdateCmd.Flags().StringVar(&spOriginPort, "origin-port", "", "Origin port or range (e.g., 22 or 8000-9000)")
	spectrumAppUpdateCmd.Flags().StringVar(&spDNSType, "dns-type", "", "External DNS record type")
	spectrumAppUpdateCmd.Flags().StringVar(&spTrafficType, "traffic-type", "", "Traffic type: tcp, udp, http")
	spectrumAppUpdateCmd.Flags().StringVar(&spTLS, "tls", "", "TLS termination mode")
	spectrumAppUpdateCmd.Flags().StringVar(&spProxyProtocol, "proxy-protocol", "", "Proxy Protocol: off, v1, v2, simple")
	spectrumAppUpdateCmd.Flags().BoolVar(&spArgo, "argo-smart-routing", false, "Enable Argo Smart Routing")
	spectrumAppUpdateCmd.Flags().BoolVar(&spIPv4, "ipv4", false, "Enable IPv4 for the application")
	spectrumAppUpdateCmd.Flags().BoolVar(&spIPFirewall, "ip-firewall", false, "Enable the IP firewall")

	// Delete flags
	spectrumAppDeleteCmd.Flags().BoolVar(&spForce, "force", false, "Skip confirmation prompt")
}

func getSpectrumService(zoneID string) (*cosmoflare.SpectrumService, error) {
	return cosmoflare.NewSpectrumServiceFromCreds(zoneID, APIToken)
}

func runSpectrumAppCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getSpectrumService(zoneID)
	if err != nil {
		return outErr("failed to create Spectrum service", err)
	}

	opts := cosmoflare.SpectrumAppCreate{
		Name:             spName,
		Protocol:         spProtocol,
		OriginPort:       spOriginPort,
		OriginDirect:     spOriginDirect,
		OriginDNSName:    spOriginDNS,
		DNSType:          spDNSType,
		TrafficType:      spTrafficType,
		TLS:              spTLS,
		ProxyProtocol:    spProxyProtocol,
		ArgoSmartRouting: spArgo,
		IPv4:             spIPv4,
		IPFirewall:       spIPFirewall,
	}

	if DryRun {
		return outPayload("DRY RUN: Would create Spectrum application", func() any {
			return map[string]interface{}{
				"zone_id":     zoneID,
				"name":        spName,
				"protocol":    spProtocol,
				"origin_port": spOriginPort,
			}
		}, func() {
			printInfo("DRY RUN: Would create Spectrum app '%s' (%s) in zone %s", spName, spProtocol, zoneID)
		})
	}

	app, err := svc.Create(context.Background(), opts)
	if err != nil {
		return outErr("failed to create Spectrum application", err)
	}

	return outPayload("Spectrum application created successfully", func() any {
		return app
	}, func() {
		printSuccess("Spectrum application created successfully!")
		printInfo("ID: %s", app.ID)
		printInfo("DNS: %s", app.DNSName)
		printInfo("Protocol: %s", app.Protocol)
		printInfo("Origin port: %s", app.OriginPort)
	})
}

func runSpectrumAppList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getSpectrumService(zoneID)
	if err != nil {
		return outErr("failed to create Spectrum service", err)
	}

	apps, err := svc.List(context.Background())
	if err != nil {
		return outErr("failed to list Spectrum applications", err)
	}

	return outResult(apps, func() {
		if len(apps) == 0 {
			printInfo("No Spectrum applications found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tDNS NAME\tPROTOCOL\tORIGIN PORT")
		for _, a := range apps {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.ID, a.DNSName, a.Protocol, a.OriginPort)
		}
		w.Flush()

		printInfo("Total: %d Spectrum application(s)", len(apps))
	})
}

func runSpectrumAppGet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and application ID are required")
	}
	zoneID, appID := args[0], args[1]

	svc, err := getSpectrumService(zoneID)
	if err != nil {
		return outErr("failed to create Spectrum service", err)
	}

	app, err := svc.Get(context.Background(), appID)
	if err != nil {
		return outErr("failed to get Spectrum application", err)
	}

	return outResult(app, func() {
		fmt.Printf("ID:                %s\n", app.ID)
		fmt.Printf("DNS name:          %s\n", app.DNSName)
		if app.DNSType != "" {
			fmt.Printf("DNS type:          %s\n", app.DNSType)
		}
		fmt.Printf("Protocol:          %s\n", app.Protocol)
		if app.TrafficType != "" {
			fmt.Printf("Traffic type:      %s\n", app.TrafficType)
		}
		if app.TLS != "" {
			fmt.Printf("TLS:               %s\n", app.TLS)
		}
		if app.ProxyProtocol != "" {
			fmt.Printf("Proxy protocol:    %s\n", app.ProxyProtocol)
		}
		if len(app.OriginDirect) > 0 {
			fmt.Printf("Origin direct:     %v\n", app.OriginDirect)
		}
		if app.OriginDNSName != "" {
			fmt.Printf("Origin DNS:        %s\n", app.OriginDNSName)
		}
		if app.OriginPort != "" {
			fmt.Printf("Origin port:       %s\n", app.OriginPort)
		}
		fmt.Printf("Argo Smart Routing: %v\n", app.ArgoSmartRouting)
		fmt.Printf("IPv4:              %v\n", app.IPv4)
		fmt.Printf("IP firewall:       %v\n", app.IPFirewall)
		fmt.Printf("Created:           %s\n", app.CreatedOn)
		fmt.Printf("Modified:          %s\n", app.ModifiedOn)
	})
}

func runSpectrumAppUpdate(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and application ID are required")
	}
	zoneID, appID := args[0], args[1]

	var opts cosmoflare.SpectrumAppUpdate
	changed := 0
	if cmd.Flags().Changed("protocol") {
		opts.Protocol = &spProtocol
		changed++
	}
	if cmd.Flags().Changed("origin-direct") {
		opts.OriginDirect = &spOriginDirect
		changed++
	}
	if cmd.Flags().Changed("origin-dns") {
		opts.OriginDNSName = &spOriginDNS
		changed++
	}
	if cmd.Flags().Changed("origin-port") {
		opts.OriginPort = &spOriginPort
		changed++
	}
	if cmd.Flags().Changed("dns-type") {
		opts.DNSType = &spDNSType
		changed++
	}
	if cmd.Flags().Changed("traffic-type") {
		opts.TrafficType = &spTrafficType
		changed++
	}
	if cmd.Flags().Changed("tls") {
		opts.TLS = &spTLS
		changed++
	}
	if cmd.Flags().Changed("proxy-protocol") {
		opts.ProxyProtocol = &spProxyProtocol
		changed++
	}
	if cmd.Flags().Changed("argo-smart-routing") {
		opts.ArgoSmartRouting = &spArgo
		changed++
	}
	if cmd.Flags().Changed("ipv4") {
		opts.IPv4 = &spIPv4
		changed++
	}
	if cmd.Flags().Changed("ip-firewall") {
		opts.IPFirewall = &spIPFirewall
		changed++
	}
	if changed == 0 {
		return fmt.Errorf("at least one update flag is required (--protocol, --origin-direct, --origin-dns, --origin-port, --dns-type, --traffic-type, --tls, --proxy-protocol, --argo-smart-routing, --ipv4, --ip-firewall)")
	}

	svc, err := getSpectrumService(zoneID)
	if err != nil {
		return outErr("failed to create Spectrum service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would update Spectrum application", func() any {
			return map[string]string{
				"zone_id": zoneID,
				"app_id":  appID,
			}
		}, func() {
			printInfo("DRY RUN: Would update Spectrum app '%s' in zone '%s'", appID, zoneID)
		})
	}

	app, err := svc.Update(context.Background(), appID, opts)
	if err != nil {
		return outErr("failed to update Spectrum application", err)
	}

	return outPayload("Spectrum application updated successfully", func() any {
		return app
	}, func() {
		printSuccess("Spectrum application '%s' updated successfully!", appID)
		printInfo("DNS: %s  Protocol: %s  Origin port: %s", app.DNSName, app.Protocol, app.OriginPort)
	})
}

func runSpectrumAppDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and application ID are required")
	}
	zoneID, appID := args[0], args[1]

	// Not registry-flagged destructive yet (registry pass follows): the
	// confirmation prompt stays live; an explicit --dry-run still
	// short-circuits before any service call.
	dry := destructiveDryRun(cliPathOfCmdOr(cmd, "spectrum app delete"), spForce)
	if dry {
		return outPayload("DRY RUN: Would delete Spectrum application", func() any {
			return map[string]string{
				"zone_id": zoneID,
				"app_id":  appID,
			}
		}, func() {
			printInfo("DRY RUN: Would delete Spectrum app '%s' from zone '%s'", appID, zoneID)
		})
	}
	if !spForce && !ux.Confirm(fmt.Sprintf("Delete Spectrum application '%s'?", appID)) {
		printInfo("Spectrum application deletion cancelled")
		return nil
	}

	svc, err := getSpectrumService(zoneID)
	if err != nil {
		return outErr("failed to create Spectrum service", err)
	}

	if err := svc.Delete(context.Background(), appID); err != nil {
		return outErr("failed to delete Spectrum application", err)
	}

	return outPayload("Spectrum application deleted successfully", func() any {
		return map[string]string{
			"zone_id": zoneID,
			"app_id":  appID,
		}
	}, func() {
		printSuccess("Spectrum application '%s' deleted successfully!", appID)
	})
}
