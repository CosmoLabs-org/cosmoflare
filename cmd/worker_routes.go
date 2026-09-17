package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	workerRoutePattern string
	workerRouteScript  string
	workerRouteForce   bool
)

var workerRouteCmd = &cobra.Command{
	Use:   "route",
	Short: "Manage zone Worker routes",
	Long: `Manage the Worker routes of a Cloudflare zone.

Worker routes map URL patterns to Worker scripts so Cloudflare runs the
script on matching requests to the zone (for example "example.com/api/*"
served by the "api-gateway" Worker).

Commands:
  list    List all routes in a zone
  create  Map a URL pattern to a Worker script
  update  Change a route's pattern and/or script
  delete  Remove a route from a zone

Examples:
  cosmoflare worker route list my-zone-id
  cosmoflare worker route create my-zone-id --pattern "example.com/api/*" --script api-gateway
  cosmoflare worker route update my-zone-id route-id --pattern "example.com/v2/*" --script api-gateway
  cosmoflare worker route delete my-zone-id route-id --force`,
}

var workerRouteListCmd = &cobra.Command{
	Use:   "list [zone-id]",
	Short: "List all Worker routes in a zone",
	Long: `List every Worker route configured on a zone.

Each route shows its ID, the URL pattern it matches, and the Worker
script it invokes. Use the ID with 'worker route update' and
'worker route delete'.

Examples:
  cosmoflare worker route list my-zone-id
  cosmoflare worker route list my-zone-id --json`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkerRouteList,
}

var workerRouteCreateCmd = &cobra.Command{
	Use:   "create [zone-id]",
	Short: "Create a Worker route",
	Long: `Map a URL pattern to a Worker script in a zone.

The pattern supports wildcards (for example "example.com/api/*") and must
not overlap an existing route in the same zone. The script must already
be deployed (see 'worker deploy').

Examples:
  cosmoflare worker route create my-zone-id --pattern "example.com/api/*" --script api-gateway
  cosmoflare worker route create my-zone-id --pattern "example.com/*" --script edge-render --json`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkerRouteCreate,
}

var workerRouteUpdateCmd = &cobra.Command{
	Use:   "update [zone-id] [route-id]",
	Short: "Update a Worker route",
	Long: `Change the pattern and/or script of an existing Worker route.

Both --pattern and --script are applied on every update; omitting a flag
re-submits its default (empty) value, so always pass both.

Find a route's ID with 'worker route list'.

Examples:
  cosmoflare worker route update my-zone-id route-id --pattern "example.com/v2/*" --script api-gateway
  cosmoflare worker route update my-zone-id route-id --pattern "example.com/api/*" --script api-gateway --json`,
	Args: cobra.ExactArgs(2),
	RunE: runWorkerRouteUpdate,
}

var workerRouteDeleteCmd = &cobra.Command{
	Use:   "delete [zone-id] [route-id]",
	Short: "Delete a Worker route",
	Long: `Remove a Worker route from a zone.

After deletion, matching traffic is no longer served by the Worker.
Confirmation is required unless --force is given.

Examples:
  cosmoflare worker route delete my-zone-id route-id
  cosmoflare worker route delete my-zone-id route-id --force
  cosmoflare worker route delete my-zone-id route-id --force --json`,
	Args: cobra.ExactArgs(2),
	RunE: runWorkerRouteDelete,
}

// registerWorkerRouteCmds attaches the 'worker route' command group to the
// given parent (the 'worker' command).
func registerWorkerRouteCmds(parent *cobra.Command) {
	parent.AddCommand(workerRouteCmd)
	workerRouteCmd.AddCommand(workerRouteListCmd)
	workerRouteCmd.AddCommand(workerRouteCreateCmd)
	workerRouteCmd.AddCommand(workerRouteUpdateCmd)
	workerRouteCmd.AddCommand(workerRouteDeleteCmd)

	if workerRouteCreateCmd.Flags().Lookup("pattern") == nil {
		workerRouteCreateCmd.Flags().StringVar(&workerRoutePattern, "pattern", "", "URL pattern the route matches (e.g. example.com/api/*)")
	}
	if workerRouteCreateCmd.Flags().Lookup("script") == nil {
		workerRouteCreateCmd.Flags().StringVar(&workerRouteScript, "script", "", "Worker script name the route invokes")
	}

	if workerRouteUpdateCmd.Flags().Lookup("pattern") == nil {
		workerRouteUpdateCmd.Flags().StringVar(&workerRoutePattern, "pattern", "", "New URL pattern for the route")
	}
	if workerRouteUpdateCmd.Flags().Lookup("script") == nil {
		workerRouteUpdateCmd.Flags().StringVar(&workerRouteScript, "script", "", "New Worker script name for the route")
	}

	if workerRouteDeleteCmd.Flags().Lookup("force") == nil {
		workerRouteDeleteCmd.Flags().BoolVar(&workerRouteForce, "force", false, "Skip confirmation prompt")
	}
}

func runWorkerRouteList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	routes, err := svc.RouteList(context.Background(), zoneID)
	if err != nil {
		return outErr("failed to list worker routes", err)
	}

	return outResult(routes, func() {
		if len(routes) == 0 {
			printInfo("No worker routes found in zone %s", zoneID)
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tPATTERN\tSCRIPT")
		for _, r := range routes {
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.ID, r.Pattern, r.Script)
		}
		w.Flush()

		printInfo("Total: %d route(s)", len(routes))
	})
}

func runWorkerRouteCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]

	if workerRoutePattern == "" {
		return fmt.Errorf("pattern is required (--pattern)")
	}
	if workerRouteScript == "" {
		return fmt.Errorf("script is required (--script)")
	}
	script := applyResourcePrefix(workerRouteScript)

	if DryRun {
		return outPayload("DRY RUN: Would create worker route", func() any {
			return map[string]string{
				"zone_id": zoneID,
				"pattern": workerRoutePattern,
				"script":  script,
			}
		}, func() {
			printInfo("DRY RUN: Would create route %q -> %q in zone %s", workerRoutePattern, script, zoneID)
		})
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	route, err := svc.RouteCreate(context.Background(), zoneID, workerRoutePattern, script)
	if err != nil {
		return outErr("failed to create worker route", err)
	}

	return outPayload("Worker route created successfully", func() any {
		return route
	}, func() {
		printSuccess("Worker route created successfully!")
		printInfo("ID: %s", route.ID)
		printInfo("Pattern: %s", route.Pattern)
		printInfo("Script: %s", route.Script)
	})
}

func runWorkerRouteUpdate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]
	if len(args) < 2 {
		return fmt.Errorf("route ID is required")
	}
	routeID := args[1]

	if workerRoutePattern == "" {
		return fmt.Errorf("pattern is required (--pattern)")
	}
	if workerRouteScript == "" {
		return fmt.Errorf("script is required (--script)")
	}
	script := applyResourcePrefix(workerRouteScript)

	if DryRun {
		return outPayload("DRY RUN: Would update worker route", func() any {
			return map[string]string{
				"zone_id":  zoneID,
				"route_id": routeID,
				"pattern":  workerRoutePattern,
				"script":   script,
			}
		}, func() {
			printInfo("DRY RUN: Would update route %s in zone %s to %q -> %q", routeID, zoneID, workerRoutePattern, script)
		})
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	route, err := svc.RouteUpdate(context.Background(), zoneID, routeID, workerRoutePattern, script)
	if err != nil {
		return outErr("failed to update worker route", err)
	}

	return outPayload("Worker route updated successfully", func() any {
		return route
	}, func() {
		printSuccess("Worker route updated successfully!")
		printInfo("ID: %s", route.ID)
		printInfo("Pattern: %s", route.Pattern)
		printInfo("Script: %s", route.Script)
	})
}

func runWorkerRouteDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	zoneID := args[0]
	if len(args) < 2 {
		return fmt.Errorf("route ID is required")
	}
	routeID := args[1]

	if !workerRouteForce && !DryRun {
		fmt.Printf("Are you sure you want to delete worker route '%s' in zone %s? [y/N]: ", routeID, zoneID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Worker route deletion cancelled")
			return nil
		}
	}

	if DryRun {
		return outPayload("DRY RUN: Would delete worker route", func() any {
			return map[string]string{"zone_id": zoneID, "route_id": routeID}
		}, func() {
			printInfo("DRY RUN: Would delete route %s in zone %s", routeID, zoneID)
		})
	}

	svc, err := getWorkerService()
	if err != nil {
		return outErr("failed to create worker service", err)
	}

	if err := svc.RouteDelete(context.Background(), zoneID, routeID); err != nil {
		return outErr("failed to delete worker route", err)
	}

	return outPayload("Worker route deleted successfully", func() any {
		return map[string]string{"zone_id": zoneID, "route_id": routeID}
	}, func() {
		printSuccess("Worker route '%s' deleted successfully!", routeID)
	})
}
