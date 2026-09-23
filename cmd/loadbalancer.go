package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var (
	lbPoolName        string
	lbPoolDescription string
	lbPoolMonitor     string
	lbPoolOriginsJSON string
	lbPoolEnabled     bool
	lbPoolDisable     bool
	lbPoolForce       bool
	lbPoolHealth      bool

	lbMonitorType          string
	lbMonitorDescription   string
	lbMonitorPath          string
	lbMonitorPort          uint16
	lbMonitorExpectedCodes string
	lbMonitorInterval      int
	lbMonitorRetries       int
	lbMonitorTimeout       int
	lbMonitorForce         bool
)

var loadbalancerCmd = &cobra.Command{
	Use:   "loadbalancer",
	Short: "Manage Load Balancer pools and monitors",
	Long: `Cloudflare Load Balancer management — create, inspect, update, and delete
account load balancer pools and their health-check monitors.

Commands:
  pool create       Create a pool with one or more origins
  pool list         List all pools in the account
  pool get          Get a pool's details (optionally with per-PoP health)
  pool update       Update a pool (partial; unspecified fields are kept)
  pool delete       Delete a pool (irreversible; dry-run by default)
  monitor create    Create a health-check monitor
  monitor list      List all monitors in the account
  monitor get       Get a monitor's details
  monitor update    Update a monitor (partial; unspecified fields are kept)
  monitor delete    Delete a monitor (irreversible; dry-run by default)

A pool groups the origins traffic fails over between; a monitor is the probe
Cloudflare runs against those origins to decide what is healthy. Attach a
monitor to a pool at create time with --monitor <monitor-id>, or later with
'pool update <pool-id> --monitor <monitor-id>'.

All commands here are account-scoped and require an API token with the
account-level "Load Balancing: Monitors and Pools" permission for every
operation, reads included.

Wrangler has no Load Balancer equivalent — this group is the CLI's
differentiation surface.

Examples:
  cosmoflare loadbalancer pool list --json
  cosmoflare loadbalancer pool create --name primary --origins '[{"name":"web-1","address":"10.0.0.1:80"},{"name":"web-2","address":"10.0.0.2:80","enabled":false}]'
  cosmoflare loadbalancer monitor create --type https --path /health --expected-codes 2xx
  cosmoflare loadbalancer pool get <pool-id> --health`,
}

var lbPoolCmd = &cobra.Command{
	Use:   "pool",
	Short: "Manage Load Balancer pools",
	Long: `Create, list, get, update, and delete load balancer pools.

A pool is an ordered set of origins (name + address + enabled). New pools
start enabled; origins whose "enabled" key is omitted in --origins also
start enabled.`,
}

var lbPoolCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Load Balancer pool",
	Long: `Create a load balancer pool with one or more origins.

--origins takes a JSON array of objects with "name", "address" and an
optional "enabled" (default true):

  [{"name":"web-1","address":"10.0.0.1:80"},
   {"name":"web-2","address":"10.0.0.2:8080","enabled":false}]

Addresses are IP or hostname with an optional port, exactly as Cloudflare's
API accepts them. An optional --monitor attaches an existing monitor
(see 'loadbalancer monitor list') so the pool is health-checked from the
start.

Examples:
  cosmoflare loadbalancer pool create --name primary --origins '[{"name":"web-1","address":"10.0.0.1:80"},{"name":"web-2","address":"10.0.0.2:80"}]'
  cosmoflare loadbalancer pool create --name edge --origins '[{"name":"app","address":"app.internal:8443"}]' --monitor 17f2b8d1b0be4f4e --json`,
	RunE: runLBPoolCreate,
}

var lbPoolListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Load Balancer pools",
	Long: `List every load balancer pool in the account.

Examples:
  cosmoflare loadbalancer pool list
  cosmoflare loadbalancer pool list --json`,
	RunE: runLBPoolList,
}

var lbPoolGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a Load Balancer pool's details",
	Long: `Get the details of one load balancer pool by its ID (see
'loadbalancer pool list' for IDs).

Pass --health to also fetch the pool's per-PoP health detail: which
Cloudflare points of presence see the pool (and each origin) as healthy.

Examples:
  cosmoflare loadbalancer pool get 1b6fbc15e6c34a8d
  cosmoflare loadbalancer pool get 1b6fbc15e6c34a8d --health --json`,
	Args: cobra.ExactArgs(1),
	RunE: runLBPoolGet,
}

var lbPoolUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a Load Balancer pool",
	Long: `Update a load balancer pool with partial changes.

Only the flags you pass change the pool; everything else keeps its current
value. The Cloudflare update is a full replace under the hood, so this
command fetches the current pool first and merges your changes on top.

Use --disable to stop routing traffic to the pool (the pool and its config
are kept) and --enabled to resume. Replacing --origins swaps the whole
origin set.

Examples:
  cosmoflare loadbalancer pool update 1b6fbc15e6c34a8d --name primary-eu
  cosmoflare loadbalancer pool update 1b6fbc15e6c34a8d --origins '[{"name":"web-1","address":"10.0.1.1:80"}]'
  cosmoflare loadbalancer pool update 1b6fbc15e6c34a8d --disable`,
	Args: cobra.ExactArgs(1),
	RunE: runLBPoolUpdate,
}

var lbPoolDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a Load Balancer pool",
	Long: `Delete a load balancer pool by its ID.

WARNING: This action is irreversible. Load balancers steering traffic to the
pool stop routing to it once it is gone.

This command is registry-flagged destructive: it runs as a dry-run unless
--force is passed.

Examples:
  cosmoflare loadbalancer pool delete 1b6fbc15e6c34a8d
  cosmoflare loadbalancer pool delete 1b6fbc15e6c34a8d --force`,
	Args: cobra.ExactArgs(1),
	RunE: runLBPoolDelete,
}

var lbMonitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Manage Load Balancer monitors",
	Long: `Create, list, get, update, and delete load balancer monitors.

A monitor is the health-check probe Cloudflare runs against every origin of
the pools referencing it: a type (http/https), a path, expected status
codes, and timing knobs.`,
}

var lbMonitorCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Load Balancer monitor",
	Long: `Create a load balancer monitor (health-check probe).

Defaults, all overridable:
  --type http            probe scheme (http or https)
  --path /               path probed on each origin
  --port 0               scheme default (80 for http, 443 for https)
  --expected-codes 2xx   status codes treated as healthy
  --interval 60          seconds between probes
  --retries 2            failed probes before an origin is marked down
  --timeout 5            seconds before a probe counts as failed

Examples:
  cosmoflare loadbalancer monitor create --type https --path /healthz --expected-codes 2xx
  cosmoflare loadbalancer monitor create --type http --path /ping --interval 30 --retries 3 --timeout 2 --json`,
	RunE: runLBMonitorCreate,
}

var lbMonitorListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Load Balancer monitors",
	Long: `List every load balancer monitor in the account.

Examples:
  cosmoflare loadbalancer monitor list
  cosmoflare loadbalancer monitor list --json`,
	RunE: runLBMonitorList,
}

var lbMonitorGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get a Load Balancer monitor's details",
	Long: `Get the details of one load balancer monitor by its ID (see
'loadbalancer monitor list' for IDs).

Examples:
  cosmoflare loadbalancer monitor get 3d5f1a2b9c8e7d6f
  cosmoflare loadbalancer monitor get 3d5f1a2b9c8e7d6f --json`,
	Args: cobra.ExactArgs(1),
	RunE: runLBMonitorGet,
}

var lbMonitorUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a Load Balancer monitor",
	Long: `Update a load balancer monitor with partial changes.

Only the flags you pass change the monitor; everything else keeps its
current value. The Cloudflare update is a full replace under the hood, so
this command fetches the current monitor first and merges your changes on
top. Flags here have NO defaults: omit one and it stays unchanged.

Examples:
  cosmoflare loadbalancer monitor update 3d5f1a2b9c8e7d6f --path /healthz
  cosmoflare loadbalancer monitor update 3d5f1a2b9c8e7d6f --interval 30 --timeout 2
  cosmoflare loadbalancer monitor update 3d5f1a2b9c8e7d6f --port 8443 --expected-codes 200`,
	Args: cobra.ExactArgs(1),
	RunE: runLBMonitorUpdate,
}

var lbMonitorDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a Load Balancer monitor",
	Long: `Delete a load balancer monitor by its ID.

WARNING: This action is irreversible. Pools still referencing the monitor
keep their last known health state until they are re-pointed at another
monitor.

This command is registry-flagged destructive: it runs as a dry-run unless
--force is passed.

Examples:
  cosmoflare loadbalancer monitor delete 3d5f1a2b9c8e7d6f
  cosmoflare loadbalancer monitor delete 3d5f1a2b9c8e7d6f --force`,
	Args: cobra.ExactArgs(1),
	RunE: runLBMonitorDelete,
}

func init() {
	rootCmd.AddCommand(loadbalancerCmd)

	loadbalancerCmd.AddCommand(lbPoolCmd, lbMonitorCmd)
	lbPoolCmd.AddCommand(
		lbPoolCreateCmd,
		lbPoolListCmd,
		lbPoolGetCmd,
		lbPoolUpdateCmd,
		lbPoolDeleteCmd,
	)
	lbMonitorCmd.AddCommand(
		lbMonitorCreateCmd,
		lbMonitorListCmd,
		lbMonitorGetCmd,
		lbMonitorUpdateCmd,
		lbMonitorDeleteCmd,
	)

	lbPoolCreateCmd.Flags().StringVar(&lbPoolName, "name", "", "Pool name (required)")
	lbPoolCreateCmd.Flags().StringVar(&lbPoolOriginsJSON, "origins", "", "JSON array of origins: [{\"name\":\"web-1\",\"address\":\"10.0.0.1:80\",\"enabled\":true}] (required)")
	lbPoolCreateCmd.Flags().StringVar(&lbPoolMonitor, "monitor", "", "Monitor ID to attach for health checks")

	lbPoolGetCmd.Flags().BoolVar(&lbPoolHealth, "health", false, "Also fetch per-PoP health detail")

	lbPoolUpdateCmd.Flags().StringVar(&lbPoolName, "name", "", "New pool name")
	lbPoolUpdateCmd.Flags().StringVar(&lbPoolOriginsJSON, "origins", "", "New origin set (JSON array; replaces the whole set)")
	lbPoolUpdateCmd.Flags().StringVar(&lbPoolMonitor, "monitor", "", "New monitor ID to attach")
	lbPoolUpdateCmd.Flags().BoolVar(&lbPoolEnabled, "enabled", false, "Resume routing traffic to the pool")
	lbPoolUpdateCmd.Flags().BoolVar(&lbPoolDisable, "disable", false, "Stop routing traffic to the pool")

	lbPoolDeleteCmd.Flags().BoolVar(&lbPoolForce, "force", false, "Skip the destructive dry-run default")

	lbMonitorCreateCmd.Flags().StringVar(&lbMonitorType, "type", "http", "Probe type: http or https")
	lbMonitorCreateCmd.Flags().StringVar(&lbMonitorPath, "path", "/", "Path probed on each origin")
	lbMonitorCreateCmd.Flags().Uint16Var(&lbMonitorPort, "port", 0, "Probe port (0 = scheme default: 80/443)")
	lbMonitorCreateCmd.Flags().StringVar(&lbMonitorExpectedCodes, "expected-codes", "2xx", "Status codes treated as healthy (e.g. 2xx, 200, 200-299)")
	lbMonitorCreateCmd.Flags().IntVar(&lbMonitorInterval, "interval", 60, "Seconds between probes")
	lbMonitorCreateCmd.Flags().IntVar(&lbMonitorRetries, "retries", 2, "Failed probes before an origin is marked down")
	lbMonitorCreateCmd.Flags().IntVar(&lbMonitorTimeout, "timeout", 5, "Seconds before a probe counts as failed")

	lbMonitorUpdateCmd.Flags().StringVar(&lbMonitorType, "type", "", "New probe type (http or https)")
	lbMonitorUpdateCmd.Flags().StringVar(&lbMonitorPath, "path", "", "New probe path")
	lbMonitorUpdateCmd.Flags().Uint16Var(&lbMonitorPort, "port", 0, "New probe port (omit to keep current)")
	lbMonitorUpdateCmd.Flags().StringVar(&lbMonitorExpectedCodes, "expected-codes", "", "New healthy status codes")
	lbMonitorUpdateCmd.Flags().IntVar(&lbMonitorInterval, "interval", 0, "New seconds between probes (omit to keep current)")
	lbMonitorUpdateCmd.Flags().IntVar(&lbMonitorRetries, "retries", 0, "New failed-probe threshold (omit to keep current)")
	lbMonitorUpdateCmd.Flags().IntVar(&lbMonitorTimeout, "timeout", 0, "New probe timeout in seconds (omit to keep current)")

	lbMonitorDeleteCmd.Flags().BoolVar(&lbMonitorForce, "force", false, "Skip the destructive dry-run default")
}

func getLoadBalancerService() (*cosmoflare.LoadBalancerService, error) {
	return cosmoflare.NewLoadBalancerServiceFromCreds(AccountID, APIToken)
}

// lbResetFlags returns the loadbalancer flag variables to their zero values
// so each test subtest starts from a pristine command state.
func lbResetFlags() {
	lbPoolName = ""
	lbPoolMonitor = ""
	lbPoolOriginsJSON = ""
	lbPoolEnabled = false
	lbPoolDisable = false
	lbPoolForce = false
	lbPoolHealth = false
	lbMonitorType = ""
	lbMonitorPath = ""
	lbMonitorPort = 0
	lbMonitorExpectedCodes = ""
	lbMonitorInterval = 0
	lbMonitorRetries = 0
	lbMonitorTimeout = 0
	lbMonitorForce = false
}

// lbOriginFlag mirrors the --origins JSON shape. Enabled is a pointer so an
// omitted key can default to true rather than silently disabling the origin.
type lbOriginFlag struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Enabled *bool  `json:"enabled"`
}

// lbParseOrigins decodes the --origins flag into service origins, defaulting
// an omitted "enabled" key to true.
func lbParseOrigins(raw string) ([]cosmoflare.LoadBalancerOrigin, error) {
	if raw == "" {
		return nil, fmt.Errorf("--origins is required (JSON array of {name,address,enabled}, e.g. '[{\"name\":\"web-1\",\"address\":\"10.0.0.1:80\"}]')")
	}
	var flags []lbOriginFlag
	if err := json.Unmarshal([]byte(raw), &flags); err != nil {
		return nil, fmt.Errorf("--origins is not valid JSON (expected an array of {name,address,enabled} objects): %v", err)
	}
	if len(flags) == 0 {
		return nil, fmt.Errorf("--origins must contain at least one origin")
	}
	origins := make([]cosmoflare.LoadBalancerOrigin, 0, len(flags))
	for _, f := range flags {
		enabled := true
		if f.Enabled != nil {
			enabled = *f.Enabled
		}
		origins = append(origins, cosmoflare.LoadBalancerOrigin{
			Name:    f.Name,
			Address: f.Address,
			Enabled: enabled,
		})
	}
	return origins, nil
}

func runLBPoolCreate(cmd *cobra.Command, args []string) error {
	if lbPoolName == "" {
		return outErrf("--name is required (a short pool name, e.g. primary)")
	}
	origins, err := lbParseOrigins(lbPoolOriginsJSON)
	if err != nil {
		return outErr("failed to parse origins", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would create load balancer pool", func() any {
			return map[string]any{
				"name":    lbPoolName,
				"origins": origins,
				"monitor": lbPoolMonitor,
			}
		}, func() {
			printInfo("DRY RUN: Would create load balancer pool %q with %d origin(s) (monitor=%q)", lbPoolName, len(origins), lbPoolMonitor)
		})
	}

	svc, err := getLoadBalancerService()
	if err != nil {
		return outErr("failed to create load balancer service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	pool, err := svc.Create(context.Background(), cosmoflare.LoadBalancerPoolCreate{
		Name:    lbPoolName,
		Monitor: lbPoolMonitor,
		Origins: origins,
	})
	if err != nil {
		return outErr("failed to create load balancer pool", err)
	}

	return outPayload("Load balancer pool created successfully", func() any {
		return pool
	}, func() {
		printSuccess("Load balancer pool '%s' created (ID: %s)", pool.Name, pool.ID)
		printInfo("Attach a monitor with: cosmoflare loadbalancer pool update %s --monitor <monitor-id>", pool.ID)
	})
}

func runLBPoolList(cmd *cobra.Command, args []string) error {
	svc, err := getLoadBalancerService()
	if err != nil {
		return outErr("failed to create load balancer service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	pools, err := svc.List(context.Background())
	if err != nil {
		return outErr("failed to list load balancer pools", err)
	}

	return outResult(pools, func() {
		if len(pools) == 0 {
			printInfo("No load balancer pools found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tENABLED\tORIGINS\tMONITOR\tHEALTHY")
		for _, p := range pools {
			fmt.Fprintf(w, "%s\t%s\t%v\t%d\t%s\t%s\n",
				p.ID, p.Name, p.Enabled, len(p.Origins), lbOrDash(p.Monitor), lbHealthString(p.Healthy))
		}
		w.Flush()
		printInfo("Total: %d pool(s)", len(pools))
	})
}

func runLBPoolGet(cmd *cobra.Command, args []string) error {
	poolID := args[0]

	svc, err := getLoadBalancerService()
	if err != nil {
		return outErr("failed to create load balancer service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	pool, err := svc.Get(context.Background(), poolID)
	if err != nil {
		return outErr(fmt.Sprintf("failed to get load balancer pool %s", poolID), err)
	}

	var health *cosmoflare.LoadBalancerPoolHealth
	if lbPoolHealth {
		health, err = svc.Health(context.Background(), poolID)
		if err != nil {
			return outErr(fmt.Sprintf("failed to get health for load balancer pool %s", poolID), err)
		}
	}

	return outResult(map[string]any{"pool": pool, "health": health}, func() {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID:\t%s\n", pool.ID)
		fmt.Fprintf(w, "Name:\t%s\n", pool.Name)
		if pool.Description != "" {
			fmt.Fprintf(w, "Description:\t%s\n", pool.Description)
		}
		fmt.Fprintf(w, "Enabled:\t%v\n", pool.Enabled)
		fmt.Fprintf(w, "Monitor:\t%s\n", lbOrDash(pool.Monitor))
		fmt.Fprintf(w, "Healthy:\t%s\n", lbHealthString(pool.Healthy))
		fmt.Fprintf(w, "Origins:\t%d\n", len(pool.Origins))
		for _, o := range pool.Origins {
			fmt.Fprintf(w, "  %s\t%s\tenabled=%v\n", o.Name, o.Address, o.Enabled)
		}
		if pool.CreatedOn != "" {
			fmt.Fprintf(w, "Created:\t%s\n", pool.CreatedOn)
		}
		if pool.ModifiedOn != "" {
			fmt.Fprintf(w, "Modified:\t%s\n", pool.ModifiedOn)
		}
		w.Flush()
		if health != nil {
			printInfo("Per-PoP health (%d PoP(s)):", len(health.Pop))
			hw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(hw, "POP\tHEALTHY\tORIGIN HEALTH")
			for pop, ph := range health.Pop {
				fmt.Fprintf(hw, "%s\t%v\t%d probe result(s)\n", pop, ph.Healthy, len(ph.Origins))
			}
			hw.Flush()
		}
	})
}

func runLBPoolUpdate(cmd *cobra.Command, args []string) error {
	poolID := args[0]
	if lbPoolEnabled && lbPoolDisable {
		return outErrf("--enabled and --disable are mutually exclusive")
	}

	var origins []cosmoflare.LoadBalancerOrigin
	if lbPoolOriginsJSON != "" {
		parsed, err := lbParseOrigins(lbPoolOriginsJSON)
		if err != nil {
			return outErr("failed to parse origins", err)
		}
		origins = parsed
	}

	var enabled *bool
	if lbPoolEnabled {
		t := true
		enabled = &t
	} else if lbPoolDisable {
		f := false
		enabled = &f
	}

	if DryRun {
		return outPayload("DRY RUN: Would update load balancer pool", func() any {
			return map[string]any{
				"pool_id": poolID,
				"name":    lbPoolName,
				"origins": origins,
				"monitor": lbPoolMonitor,
				"enabled": enabled,
			}
		}, func() {
			printInfo("DRY RUN: Would update load balancer pool %s (name=%q origins=%d monitor=%q enabled=%v)",
				poolID, lbPoolName, len(origins), lbPoolMonitor, enabled)
		})
	}

	svc, err := getLoadBalancerService()
	if err != nil {
		return outErr("failed to create load balancer service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	pool, err := svc.Update(context.Background(), poolID, cosmoflare.LoadBalancerPoolUpdate{
		Name:    lbPoolName,
		Monitor: lbPoolMonitor,
		Enabled: enabled,
		Origins: origins,
	})
	if err != nil {
		return outErr(fmt.Sprintf("failed to update load balancer pool %s", poolID), err)
	}

	return outPayload("Load balancer pool updated successfully", func() any {
		return pool
	}, func() {
		printSuccess("Load balancer pool '%s' updated (enabled=%v)", pool.Name, pool.Enabled)
	})
}

func runLBPoolDelete(cmd *cobra.Command, args []string) error {
	poolID := args[0]

	cliPath := cliPathOfCmdOr(cmd, "loadbalancer pool delete")
	// Registry-flagged destructive: runs dry unless --force, so no prompt.
	dry := destructiveDryRun(cliPath, lbPoolForce)
	if dry {
		return outPayload("DRY RUN: Would delete load balancer pool", func() any {
			return map[string]any{"pool_id": poolID}
		}, func() {
			printInfo("DRY RUN: Would delete load balancer pool %s (irreversible; pass --force to execute)", poolID)
		})
	}

	svc, err := getLoadBalancerService()
	if err != nil {
		return outErr("failed to create load balancer service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	if err := svc.Delete(context.Background(), poolID); err != nil {
		auditMutation(cliPath, poolID, false)
		return outErr("failed to delete load balancer pool", err)
	}
	auditMutation(cliPath, poolID, true)

	return outPayload("Load balancer pool deleted successfully", func() any {
		return map[string]any{"pool_id": poolID}
	}, func() {
		printSuccess("Load balancer pool %s deleted successfully!", poolID)
	})
}

func runLBMonitorCreate(cmd *cobra.Command, args []string) error {
	if lbMonitorType != "http" && lbMonitorType != "https" {
		return outErrf("--type must be http or https, got %q", lbMonitorType)
	}

	if DryRun {
		return outPayload("DRY RUN: Would create load balancer monitor", func() any {
			return map[string]any{
				"type":           lbMonitorType,
				"path":           lbMonitorPath,
				"port":           lbMonitorPort,
				"expected_codes": lbMonitorExpectedCodes,
				"interval":       lbMonitorInterval,
				"retries":        lbMonitorRetries,
				"timeout":        lbMonitorTimeout,
			}
		}, func() {
			printInfo("DRY RUN: Would create %s monitor probing %s (codes=%q interval=%ds retries=%d timeout=%ds)",
				lbMonitorType, lbMonitorPath, lbMonitorExpectedCodes, lbMonitorInterval, lbMonitorRetries, lbMonitorTimeout)
		})
	}

	svc, err := getLoadBalancerService()
	if err != nil {
		return outErr("failed to create load balancer service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	monitor, err := svc.CreateMonitor(context.Background(), cosmoflare.LoadBalancerMonitorCreate{
		Type:          lbMonitorType,
		Path:          lbMonitorPath,
		Port:          lbMonitorPort,
		ExpectedCodes: lbMonitorExpectedCodes,
		Interval:      lbMonitorInterval,
		Retries:       lbMonitorRetries,
		Timeout:       lbMonitorTimeout,
	})
	if err != nil {
		return outErr("failed to create load balancer monitor", err)
	}

	return outPayload("Load balancer monitor created successfully", func() any {
		return monitor
	}, func() {
		printSuccess("Load balancer monitor %s created (%s %s, codes %q)", monitor.ID, monitor.Type, monitor.Path, monitor.ExpectedCodes)
		printInfo("Attach it to a pool with: cosmoflare loadbalancer pool update <pool-id> --monitor %s", monitor.ID)
	})
}

func runLBMonitorList(cmd *cobra.Command, args []string) error {
	svc, err := getLoadBalancerService()
	if err != nil {
		return outErr("failed to create load balancer service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	monitors, err := svc.ListMonitors(context.Background())
	if err != nil {
		return outErr("failed to list load balancer monitors", err)
	}

	return outResult(monitors, func() {
		if len(monitors) == 0 {
			printInfo("No load balancer monitors found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTYPE\tPATH\tCODES\tINTERVAL\tRETRIES\tTIMEOUT")
		for _, m := range monitors {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%ds\t%d\t%ds\n",
				m.ID, m.Type, m.Path, lbOrDash(m.ExpectedCodes), m.Interval, m.Retries, m.Timeout)
		}
		w.Flush()
		printInfo("Total: %d monitor(s)", len(monitors))
	})
}

func runLBMonitorGet(cmd *cobra.Command, args []string) error {
	monitorID := args[0]

	svc, err := getLoadBalancerService()
	if err != nil {
		return outErr("failed to create load balancer service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	monitor, err := svc.GetMonitor(context.Background(), monitorID)
	if err != nil {
		return outErr(fmt.Sprintf("failed to get load balancer monitor %s", monitorID), err)
	}

	return outResult(monitor, func() {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID:\t%s\n", monitor.ID)
		fmt.Fprintf(w, "Type:\t%s\n", monitor.Type)
		fmt.Fprintf(w, "Path:\t%s\n", monitor.Path)
		if monitor.Port != 0 {
			fmt.Fprintf(w, "Port:\t%d\n", monitor.Port)
		}
		fmt.Fprintf(w, "Expected codes:\t%s\n", lbOrDash(monitor.ExpectedCodes))
		fmt.Fprintf(w, "Interval:\t%ds\n", monitor.Interval)
		fmt.Fprintf(w, "Retries:\t%d\n", monitor.Retries)
		fmt.Fprintf(w, "Timeout:\t%ds\n", monitor.Timeout)
		if monitor.CreatedOn != "" {
			fmt.Fprintf(w, "Created:\t%s\n", monitor.CreatedOn)
		}
		if monitor.ModifiedOn != "" {
			fmt.Fprintf(w, "Modified:\t%s\n", monitor.ModifiedOn)
		}
		w.Flush()
	})
}

func runLBMonitorUpdate(cmd *cobra.Command, args []string) error {
	monitorID := args[0]

	if DryRun {
		return outPayload("DRY RUN: Would update load balancer monitor", func() any {
			return map[string]any{
				"monitor_id":     monitorID,
				"type":           lbMonitorType,
				"path":           lbMonitorPath,
				"port":           lbMonitorPort,
				"expected_codes": lbMonitorExpectedCodes,
				"interval":       lbMonitorInterval,
				"retries":        lbMonitorRetries,
				"timeout":        lbMonitorTimeout,
			}
		}, func() {
			printInfo("DRY RUN: Would update load balancer monitor %s (type=%q path=%q port=%d codes=%q interval=%d retries=%d timeout=%d)",
				monitorID, lbMonitorType, lbMonitorPath, lbMonitorPort, lbMonitorExpectedCodes, lbMonitorInterval, lbMonitorRetries, lbMonitorTimeout)
		})
	}

	svc, err := getLoadBalancerService()
	if err != nil {
		return outErr("failed to create load balancer service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	monitor, err := svc.UpdateMonitor(context.Background(), monitorID, cosmoflare.LoadBalancerMonitorUpdate{
		Type:          lbMonitorType,
		Path:          lbMonitorPath,
		Port:          lbMonitorPort,
		ExpectedCodes: lbMonitorExpectedCodes,
		Interval:      lbMonitorInterval,
		Retries:       lbMonitorRetries,
		Timeout:       lbMonitorTimeout,
	})
	if err != nil {
		return outErr(fmt.Sprintf("failed to update load balancer monitor %s", monitorID), err)
	}

	return outPayload("Load balancer monitor updated successfully", func() any {
		return monitor
	}, func() {
		printSuccess("Load balancer monitor %s updated (%s %s)", monitor.ID, monitor.Type, monitor.Path)
	})
}

func runLBMonitorDelete(cmd *cobra.Command, args []string) error {
	monitorID := args[0]

	cliPath := cliPathOfCmdOr(cmd, "loadbalancer monitor delete")
	// Registry-flagged destructive: runs dry unless --force, so no prompt.
	dry := destructiveDryRun(cliPath, lbMonitorForce)
	if dry {
		return outPayload("DRY RUN: Would delete load balancer monitor", func() any {
			return map[string]any{"monitor_id": monitorID}
		}, func() {
			printInfo("DRY RUN: Would delete load balancer monitor %s (irreversible; pass --force to execute)", monitorID)
		})
	}

	svc, err := getLoadBalancerService()
	if err != nil {
		return outErr("failed to create load balancer service (check CLOUDFLARE_ACCOUNT_ID / CLOUDFLARE_API_TOKEN)", err)
	}

	if err := svc.DeleteMonitor(context.Background(), monitorID); err != nil {
		auditMutation(cliPath, monitorID, false)
		return outErr("failed to delete load balancer monitor", err)
	}
	auditMutation(cliPath, monitorID, true)

	return outPayload("Load balancer monitor deleted successfully", func() any {
		return map[string]any{"monitor_id": monitorID}
	}, func() {
		printSuccess("Load balancer monitor %s deleted successfully!", monitorID)
	})
}

// lbOrDash renders optional strings as a dash when the API reports none.
func lbOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// lbHealthString renders the API's optional pool health verdict.
func lbHealthString(h *bool) string {
	if h == nil {
		return "-"
	}
	if *h {
		return "healthy"
	}
	return "unhealthy"
}
