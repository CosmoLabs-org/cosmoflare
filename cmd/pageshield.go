package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/cli/ux"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// Page Shield is zone-scoped; a zone ID is required for all operations.
// Token-UI permission: zone "Client-side security" (formerly "Page Shield"
// — rename caught by the Qwen dataset); the command name stays page-shield.

var pageShieldCmd = &cobra.Command{
	Use:   "page-shield",
	Short: "Inspect Page Shield detections and manage its policies",
	Long: `Page Shield (Client-side security) inspections and policy management.

Page Shield monitors third-party JavaScript loaded by your zone and lets
you enforce policies on detected connections and scripts.

Commands:
  connections  List detected outbound connections
  scripts      List detected scripts
  policy       Manage Page Shield policies (create, list, get, update, delete)

Examples:
  cosmoflare page-shield connections ZONE_ID --json
  cosmoflare page-shield scripts ZONE_ID
  cosmoflare page-shield policy create ZONE_ID --expression="http.request.uri.path matches \"^/checkout\"" --action=block --description=guard
  cosmoflare page-shield policy list ZONE_ID`,
}

var (
	psPolicyExpression string
	psPolicyAction     string
	psPolicyValue      string
	psPolicyDesc       string
	psPolicyEnabled    bool
	psPolicyForce      bool
)

var pageShieldPolicyDeleteCmd *cobra.Command

var pageShieldPolicyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Manage Page Shield policies",
	Long: `Manage Page Shield policies for a zone.

Commands:
  create    Create a Page Shield policy
  list      List Page Shield policies
  get       Get a Page Shield policy
  update    Update a Page Shield policy
  delete    Delete a Page Shield policy

Examples:
  cosmoflare page-shield policy create ZONE_ID --expression="..." --action=log --description=observe
  cosmoflare page-shield policy list ZONE_ID --json`,
}

var (
	getPageShieldService = func() (*cosmoflare.PageShieldService, error) {
		return cosmoflare.NewPageShieldServiceFromCreds(APIToken2ZoneID(), APIToken)
	}
)

// APIToken2ZoneID keeps the zone ID dependency explicit for the seam var
// above without new global state.
func APIToken2ZoneID() string { return pageShieldZoneID }

var pageShieldZoneID string

func runPageShieldConnections(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	pageShieldZoneID = args[0]
	svc, err := getPageShieldService()
	if err != nil {
		return outErr("failed to create Page Shield service", err)
	}
	conns, err := svc.ListConnections(context.Background())
	if err != nil {
		return outErr("failed to list Page Shield connections", err)
	}
	return outResult(conns, func() {
		if len(conns) == 0 {
			printInfo("No Page Shield connections detected")
			return
		}
		for _, c := range conns {
			printInfo("%+v", c)
		}
		printInfo("Total: %d connection(s)", len(conns))
	})
}

func runPageShieldScripts(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	pageShieldZoneID = args[0]
	svc, err := getPageShieldService()
	if err != nil {
		return outErr("failed to create Page Shield service", err)
	}
	scripts, err := svc.ListScripts(context.Background())
	if err != nil {
		return outErr("failed to list Page Shield scripts", err)
	}
	return outResult(scripts, func() {
		if len(scripts) == 0 {
			printInfo("No Page Shield scripts detected")
			return
		}
		for _, s := range scripts {
			printInfo("%+v", s)
		}
		printInfo("Total: %d script(s)", len(scripts))
	})
}

func psPolicyOptions() []cosmoflare.PageShieldPolicyOption {
	var opts []cosmoflare.PageShieldPolicyOption
	if cmdChanged("expression") {
		opts = append(opts, cosmoflare.WithPageShieldPolicyExpression(psPolicyExpression))
	}
	if cmdChanged("action") {
		opts = append(opts, cosmoflare.WithPageShieldPolicyAction(psPolicyAction))
	}
	if cmdChanged("value") {
		opts = append(opts, cosmoflare.WithPageShieldPolicyValue(psPolicyValue))
	}
	if cmdChanged("description") {
		opts = append(opts, cosmoflare.WithPageShieldPolicyDescription(psPolicyDesc))
	}
	if cmdChanged("enabled") {
		opts = append(opts, cosmoflare.WithPageShieldPolicyEnabled(psPolicyEnabled))
	}
	return opts
}

// cmdChanged reports whether the named flag was set on the invoked
// command; a nil cmd (direct runner calls) reports true so tests observe
// the flag variables directly.
func cmdChanged(name string) bool {
	c := pageShieldInvokedCmd
	if c == nil {
		return true
	}
	return c.Flags().Changed(name)
}

var pageShieldInvokedCmd *cobra.Command

func runPageShieldPolicyCreate(cmd *cobra.Command, args []string) error {
	pageShieldInvokedCmd = cmd
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	pageShieldZoneID = args[0]
	opts := psPolicyOptions()
	if len(opts) == 0 {
		return fmt.Errorf("at least one policy option is required (--expression, --action, --value, --description, --enabled)")
	}
	svc, err := getPageShieldService()
	if err != nil {
		return outErr("failed to create Page Shield service", err)
	}
	policy, err := svc.CreatePolicy(context.Background(), opts...)
	if err != nil {
		return outErr("failed to create Page Shield policy", err)
	}
	return outPayload("Page Shield policy created", func() any { return policy }, func() {
		printSuccess("Page Shield policy %s created", policy.ID)
	})
}

func runPageShieldPolicyList(cmd *cobra.Command, args []string) error {
	pageShieldInvokedCmd = cmd
	if len(args) == 0 {
		return fmt.Errorf("zone ID is required")
	}
	pageShieldZoneID = args[0]
	svc, err := getPageShieldService()
	if err != nil {
		return outErr("failed to create Page Shield service", err)
	}
	policies, err := svc.ListPolicies(context.Background())
	if err != nil {
		return outErr("failed to list Page Shield policies", err)
	}
	return outResult(policies, func() {
		if len(policies) == 0 {
			printInfo("No Page Shield policies")
			return
		}
		for _, p := range policies {
			printInfo("%+v", p)
		}
		printInfo("Total: %d policy(ies)", len(policies))
	})
}

func runPageShieldPolicyGet(cmd *cobra.Command, args []string) error {
	pageShieldInvokedCmd = cmd
	if len(args) < 2 {
		return fmt.Errorf("zone ID and policy ID are required")
	}
	pageShieldZoneID = args[0]
	svc, err := getPageShieldService()
	if err != nil {
		return outErr("failed to create Page Shield service", err)
	}
	policy, err := svc.GetPolicy(context.Background(), args[1])
	if err != nil {
		return outErr("failed to get Page Shield policy", err)
	}
	return outResult(policy, func() { printInfo("%+v", policy) })
}

func runPageShieldPolicyUpdate(cmd *cobra.Command, args []string) error {
	pageShieldInvokedCmd = cmd
	if len(args) < 2 {
		return fmt.Errorf("zone ID and policy ID are required")
	}
	pageShieldZoneID = args[0]
	opts := psPolicyOptions()
	if len(opts) == 0 {
		return fmt.Errorf("at least one policy option is required (--expression, --action, --value, --description, --enabled)")
	}
	svc, err := getPageShieldService()
	if err != nil {
		return outErr("failed to create Page Shield service", err)
	}
	policy, err := svc.UpdatePolicy(context.Background(), args[1], opts...)
	if err != nil {
		return outErr("failed to update Page Shield policy", err)
	}
	return outPayload("Page Shield policy updated", func() any { return policy }, func() {
		printSuccess("Page Shield policy %s updated", policy.ID)
	})
}

func runPageShieldPolicyDelete(cmd *cobra.Command, args []string) error {
	pageShieldInvokedCmd = cmd
	if len(args) < 2 {
		return fmt.Errorf("zone ID and policy ID are required")
	}
	zoneID, policyID := args[0], args[1]

	// Not registry-flagged destructive yet (registry pass follows): the
	// confirmation prompt stays live; an explicit --dry-run still
	// short-circuits before any service call.
	dry := destructiveDryRun(cliPathOfCmdOr(cmd, "page-shield policy delete"), psPolicyForce)
	if dry {
		return outPayload("DRY RUN: Would delete Page Shield policy", func() any {
			return map[string]string{"zone_id": zoneID, "policy_id": policyID}
		}, func() {
			printInfo("DRY RUN: Would delete Page Shield policy '%s'", policyID)
		})
	}
	if !psPolicyForce && !ux.Confirm(fmt.Sprintf("Delete Page Shield policy '%s'?", policyID)) {
		printInfo("Page Shield policy deletion cancelled")
		return nil
	}

	pageShieldZoneID = zoneID
	svc, err := getPageShieldService()
	if err != nil {
		return outErr("failed to create Page Shield service", err)
	}
	if err := svc.DeletePolicy(context.Background(), policyID); err != nil {
		return outErr("failed to delete Page Shield policy", err)
	}
	return outPayload("Page Shield policy deleted", func() any {
		return map[string]string{"policy_id": policyID}
	}, func() { printSuccess("Page Shield policy %s deleted", policyID) })
}

func init() {
	rootCmd.AddCommand(pageShieldCmd)
	pageShieldCmd.AddCommand(pageShieldPolicyCmd)

	pageShieldConnectionsCmd := &cobra.Command{
		Use:   "connections [zone-id]",
		Short: "List detected outbound connections",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runPageShieldConnections,
	}
	pageShieldScriptsCmd := &cobra.Command{
		Use:   "scripts [zone-id]",
		Short: "List detected scripts",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runPageShieldScripts,
	}
	pageShieldPolicyCreateCmd := &cobra.Command{
		Use:   "create [zone-id]",
		Short: "Create a Page Shield policy",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runPageShieldPolicyCreate,
	}
	pageShieldPolicyListCmd := &cobra.Command{
		Use:   "list [zone-id]",
		Short: "List Page Shield policies",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runPageShieldPolicyList,
	}
	pageShieldPolicyGetCmd := &cobra.Command{
		Use:   "get [zone-id] [policy-id]",
		Short: "Get a Page Shield policy",
		Args:  cobra.MinimumNArgs(2),
		RunE:  runPageShieldPolicyGet,
	}
	pageShieldPolicyUpdateCmd := &cobra.Command{
		Use:   "update [zone-id] [policy-id]",
		Short: "Update a Page Shield policy",
		Args:  cobra.MinimumNArgs(2),
		RunE:  runPageShieldPolicyUpdate,
	}
	pageShieldPolicyDeleteCmd = &cobra.Command{
		Use:   "delete [zone-id] [policy-id]",
		Short: "Delete a Page Shield policy",
		Args:  cobra.MinimumNArgs(2),
		RunE:  runPageShieldPolicyDelete,
	}
	pageShieldPolicyDeleteCmd.Flags().BoolVar(&psPolicyForce, "force", false, "Skip confirmation prompt")

	pageShieldPolicyCmd.AddCommand(pageShieldPolicyCreateCmd, pageShieldPolicyListCmd, pageShieldPolicyGetCmd, pageShieldPolicyUpdateCmd, pageShieldPolicyDeleteCmd)
	pageShieldCmd.AddCommand(pageShieldConnectionsCmd, pageShieldScriptsCmd)

	for _, c := range []*cobra.Command{pageShieldPolicyCreateCmd, pageShieldPolicyUpdateCmd} {
		c.Flags().StringVar(&psPolicyExpression, "expression", "", "Policy expression")
		c.Flags().StringVar(&psPolicyAction, "action", "", "Policy action (log, block, allow)")
		c.Flags().StringVar(&psPolicyValue, "value", "", "Policy value")
		c.Flags().StringVar(&psPolicyDesc, "description", "", "Policy description")
		c.Flags().BoolVar(&psPolicyEnabled, "enabled", true, "Enable the policy")
	}
}
