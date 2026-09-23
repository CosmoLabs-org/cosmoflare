package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Manage Cloudflare Tunnels",
	Long: `Cloudflare Tunnel management — create, inspect, and tear down cfd_tunnel resources.

Commands:
  create       Create a tunnel
  list         List all tunnels
  get          Get tunnel details (by ID or name)
  delete       Delete a tunnel
  token        Get the connector token
  connections  List active connectors
  cleanup      Force-disconnect connectors and delete the tunnel

Cloudflare Tunnels expose private origin services on the public internet via
outbound-only connections from cloudflared. Managing them requires an API
token with the account-level "Cloudflare Tunnel: Read" permission (writes
need "Cloudflare Tunnel: Edit").

Examples:
  cosmoflare tunnel create my-tunnel
  cosmoflare tunnel list --json
  cosmoflare tunnel get my-tunnel
  cosmoflare tunnel token my-tunnel
  cosmoflare tunnel connections my-tunnel
  cosmoflare tunnel delete <tunnel-id> --force --cascade
  cosmoflare tunnel cleanup <tunnel-id> --force`,
}

var (
	tunnelConfigSrc string
	tunnelForce     bool
	tunnelCascade   bool
)

var tunnelCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a Cloudflare Tunnel",
	Long: `Create a new Cloudflare Tunnel.

The tunnel token needed to run cloudflared is available after creation via
'cosmoflare tunnel token <name>'.

Use --config-src=cloudflare for a remotely-managed tunnel whose ingress
configuration lives in the Cloudflare dashboard (the default, an empty
value, is a locally-managed tunnel configured in cloudflared's config file).

Examples:
  cosmoflare tunnel create my-tunnel
  cosmoflare tunnel create dashboard-managed --config-src=cloudflare --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTunnelCreate,
}

var tunnelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Cloudflare Tunnels",
	Long: `List all Cloudflare Tunnels in the current account.

Examples:
  cosmoflare tunnel list
  cosmoflare tunnel list --json`,
	RunE: runTunnelList,
}

var tunnelGetCmd = &cobra.Command{
	Use:   "get [id-or-name]",
	Short: "Get Cloudflare Tunnel details",
	Long: `Get details of a single Cloudflare Tunnel by its ID or name.

Names are resolved against the account's tunnels; if several tunnels share
the name, pass the ID from 'cosmoflare tunnel list' instead.

Examples:
  cosmoflare tunnel get my-tunnel
  cosmoflare tunnel get 1234abcd-... --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTunnelGet,
}

var tunnelDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a Cloudflare Tunnel",
	Long: `Delete a Cloudflare Tunnel by its ID.

WARNING: This action is irreversible. Public hostnames routed through the
tunnel stop resolving once it is gone.

Use --cascade to tear down active connector connections before the delete,
avoiding races with running cloudflared instances.

Examples:
  cosmoflare tunnel delete <tunnel-id>
  cosmoflare tunnel delete <tunnel-id> --force --cascade`,
	Args: cobra.ExactArgs(1),
	RunE: runTunnelDelete,
}

var tunnelTokenCmd = &cobra.Command{
	Use:   "token [id]",
	Short: "Get a Cloudflare Tunnel connector token",
	Long: `Get the connector token for a Cloudflare Tunnel.

The token lets a cloudflared instance connect without credentials material
on disk:
  cloudflared tunnel run --token <token>

Treat the token as a secret: anyone holding it can connect a connector to
your tunnel.

Examples:
  cosmoflare tunnel token <tunnel-id>`,
	Args: cobra.ExactArgs(1),
	RunE: runTunnelToken,
}

var tunnelConnectionsCmd = &cobra.Command{
	Use:   "connections [id]",
	Short: "List active Cloudflare Tunnel connectors",
	Long: `List the cloudflared connectors currently attached to a Cloudflare Tunnel,
including their edge connections (colo, origin IP, client version).

Examples:
  cosmoflare tunnel connections <tunnel-id>
  cosmoflare tunnel connections <tunnel-id> --json`,
	Args: cobra.ExactArgs(1),
	RunE: runTunnelConnections,
}

var tunnelCleanupCmd = &cobra.Command{
	Use:   "cleanup [id]",
	Short: "Disconnect connectors and delete a Cloudflare Tunnel",
	Long: `Tear down a Cloudflare Tunnel completely: force-disconnect its active
connectors first, then delete the tunnel.

WARNING: This action is irreversible.

Examples:
  cosmoflare tunnel cleanup <tunnel-id>
  cosmoflare tunnel cleanup <tunnel-id> --force`,
	Args: cobra.ExactArgs(1),
	RunE: runTunnelCleanup,
}

func init() {
	rootCmd.AddCommand(tunnelCmd)

	tunnelCmd.AddCommand(tunnelCreateCmd)
	tunnelCmd.AddCommand(tunnelListCmd)
	tunnelCmd.AddCommand(tunnelGetCmd)
	tunnelCmd.AddCommand(tunnelDeleteCmd)
	tunnelCmd.AddCommand(tunnelTokenCmd)
	tunnelCmd.AddCommand(tunnelConnectionsCmd)
	tunnelCmd.AddCommand(tunnelCleanupCmd)

	tunnelCreateCmd.Flags().StringVar(&tunnelConfigSrc, "config-src", "", "Configuration source: cloudflare for remotely-managed tunnels, empty for locally-managed")

	tunnelDeleteCmd.Flags().BoolVar(&tunnelForce, "force", false, "Skip the destructive dry-run default")
	tunnelDeleteCmd.Flags().BoolVar(&tunnelCascade, "cascade", false, "Tear down active connector connections before deleting")

	tunnelCleanupCmd.Flags().BoolVar(&tunnelForce, "force", false, "Skip the destructive dry-run default")
}

func getTunnelService() (*cosmoflare.TunnelService, error) {
	return cosmoflare.NewTunnelServiceFromCreds(AccountID, APIToken)
}

func runTunnelCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	if DryRun {
		return outPayload("DRY RUN: Would create tunnel", func() any {
			return map[string]string{
				"name":       name,
				"config_src": tunnelConfigSrc,
			}
		}, func() {
			printInfo("DRY RUN: Would create tunnel '%s' (config_src=%q)", name, tunnelConfigSrc)
		})
	}

	svc, err := getTunnelService()
	if err != nil {
		return outErr("failed to create tunnel service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	t, err := svc.Create(context.Background(), name, tunnelConfigSrc)
	if err != nil {
		return outErr("failed to create tunnel", err)
	}

	return outPayload("Tunnel created successfully", func() any {
		return t
	}, func() {
		printSuccess("Tunnel '%s' created (ID: %s)", t.Name, t.ID)
		printInfo("Fetch the connector token with: cosmoflare tunnel token %s", t.ID)
	})
}

func runTunnelList(cmd *cobra.Command, args []string) error {
	svc, err := getTunnelService()
	if err != nil {
		return outErr("failed to create tunnel service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	tunnels, err := svc.List(context.Background())
	if err != nil {
		return outErr("failed to list tunnels", err)
	}

	return outResult(tunnels, func() {
		if len(tunnels) == 0 {
			printInfo("No tunnels found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tSTATUS\tCONNECTIONS\tCREATED")
		for _, t := range tunnels {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
				t.ID, t.Name, tunnelStatusString(t), len(t.Connections), tunnelTimeString(t.CreatedAt))
		}
		w.Flush()
		printInfo("Total: %d tunnel(s)", len(tunnels))
	})
}

func runTunnelGet(cmd *cobra.Command, args []string) error {
	svc, err := getTunnelService()
	if err != nil {
		return outErr("failed to create tunnel service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	t, err := svc.Resolve(context.Background(), args[0])
	if err != nil {
		return outErr(fmt.Sprintf("failed to resolve tunnel %q", args[0]), err)
	}

	return outResult(t, func() {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID:\t%s\n", t.ID)
		fmt.Fprintf(w, "Name:\t%s\n", t.Name)
		fmt.Fprintf(w, "Status:\t%s\n", tunnelStatusString(t))
		fmt.Fprintf(w, "Created:\t%s\n", tunnelTimeString(t.CreatedAt))
		if t.TunnelType != "" {
			fmt.Fprintf(w, "Type:\t%s\n", t.TunnelType)
		}
		fmt.Fprintf(w, "Remote-managed config:\t%v\n", t.RemoteConfig)
		if len(t.Connections) > 0 {
			fmt.Fprintf(w, "Connections:\t%d\n", len(t.Connections))
		}
		w.Flush()
	})
}

func runTunnelDelete(cmd *cobra.Command, args []string) error {
	tunnelID := args[0]

	cliPath := cliPathOfCmdOr(cmd, "tunnel delete")
	// Registry-flagged destructive: runs dry unless --force, so no prompt.
	dry := destructiveDryRun(cliPath, tunnelForce)
	if dry {
		return outPayload("DRY RUN: Would delete tunnel", func() any {
			return map[string]any{
				"tunnel_id": tunnelID,
				"cascade":   tunnelCascade,
			}
		}, func() {
			printInfo("DRY RUN: Would delete tunnel '%s' (cascade=%v)", tunnelID, tunnelCascade)
		})
	}

	svc, err := getTunnelService()
	if err != nil {
		return outErr("failed to create tunnel service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	if err := svc.Delete(context.Background(), tunnelID, tunnelCascade); err != nil {
		auditMutation(cliPath, tunnelID, false)
		return outErr("failed to delete tunnel", err)
	}
	auditMutation(cliPath, tunnelID, true)

	return outPayload("Tunnel deleted successfully", func() any {
		return map[string]any{
			"tunnel_id": tunnelID,
			"cascade":   tunnelCascade,
		}
	}, func() {
		printSuccess("Tunnel '%s' deleted successfully!", tunnelID)
	})
}

func runTunnelToken(cmd *cobra.Command, args []string) error {
	tunnelID := args[0]

	svc, err := getTunnelService()
	if err != nil {
		return outErr("failed to create tunnel service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	token, err := svc.Token(context.Background(), tunnelID)
	if err != nil {
		return outErr("failed to get tunnel token", err)
	}

	return outPayload("Tunnel token retrieved", func() any {
		return map[string]string{"token": token}
	}, func() {
		printSuccess("Tunnel token for '%s':", tunnelID)
		fmt.Println(token)
		printInfo("Run a connector with: cloudflared tunnel run --token <token>")
	})
}

func runTunnelConnections(cmd *cobra.Command, args []string) error {
	tunnelID := args[0]

	svc, err := getTunnelService()
	if err != nil {
		return outErr("failed to create tunnel service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	connectors, err := svc.Connections(context.Background(), tunnelID)
	if err != nil {
		return outErr("failed to list tunnel connections", err)
	}

	return outResult(connectors, func() {
		if len(connectors) == 0 {
			printInfo("No active connectors for tunnel '%s'", tunnelID)
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "CONNECTOR\tVERSION\tARCH\tEDGE CONNECTIONS")
		for _, c := range connectors {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", c.ID, c.Version, c.Arch, len(c.Connections))
			for _, conn := range c.Connections {
				fmt.Fprintf(w, "  %s\t%s\t%s\tcolo=%s origin=%s\n",
					conn.ID, conn.ClientVersion, "", conn.ColoName, conn.OriginIP)
			}
		}
		w.Flush()
	})
}

func runTunnelCleanup(cmd *cobra.Command, args []string) error {
	tunnelID := args[0]

	cliPath := cliPathOfCmdOr(cmd, "tunnel cleanup")
	// Registry-flagged destructive: runs dry unless --force, so no prompt.
	dry := destructiveDryRun(cliPath, tunnelForce)
	if dry {
		return outPayload("DRY RUN: Would clean up tunnel", func() any {
			return map[string]string{"tunnel_id": tunnelID}
		}, func() {
			printInfo("DRY RUN: Would disconnect connectors and delete tunnel '%s'", tunnelID)
		})
	}

	svc, err := getTunnelService()
	if err != nil {
		return outErr("failed to create tunnel service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	if err := svc.Cleanup(context.Background(), tunnelID); err != nil {
		auditMutation(cliPath, tunnelID, false)
		return outErr("failed to clean up tunnel", err)
	}
	auditMutation(cliPath, tunnelID, true)

	return outPayload("Tunnel cleaned up successfully", func() any {
		return map[string]string{"tunnel_id": tunnelID}
	}, func() {
		printSuccess("Tunnel '%s' connectors disconnected and tunnel deleted!", tunnelID)
	})
}

// tunnelStatusString renders a tunnel's status, defaulting to a dash when the
// API reports none (common for locally-managed tunnels).
func tunnelStatusString(t *cosmoflare.Tunnel) string {
	if t.Status != "" {
		return t.Status
	}
	if t.DeletedAt != nil {
		return "deleted"
	}
	return "-"
}

// tunnelTimeString renders an optional timestamp in RFC 3339 form.
func tunnelTimeString(ts *time.Time) string {
	if ts == nil {
		return "-"
	}
	return ts.Format(time.RFC3339)
}
