/*
Package cmd provides authentication commands for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/permdata"
	"github.com/spf13/cobra"
)

var (
	authPermissionsScope string
	authPermissionsGroup string
)

// authPermissionsCmd represents the auth permissions command
var authPermissionsCmd = &cobra.Command{
	Use:   "permissions",
	Short: "Manage Cloudflare API token permission manifest",
	Long: `View the Cloudflare API token permission manifest.

Commands:
  list      List permission families or a command group's least-privilege set`,
}

// authPermissionsListCmd represents the auth permissions list command
var authPermissionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Cloudflare API token permission families",
	Long: `List Cloudflare API token permission families from the embedded manifest.

Without flags, prints every permission family (account, zone, and user
scope) sorted by scope then name. Use --scope to filter to a single
scope, or --group to print the least-privilege permission set for a
command group instead of the family table.

Examples:
  cosmoflare auth permissions list
  cosmoflare auth permissions list --scope account
  cosmoflare auth permissions list --group deploy
  cosmoflare auth permissions list --scope zone --json`,
	RunE: runAuthPermissionsList,
}

func init() {
	authCmd.AddCommand(authPermissionsCmd)
	authPermissionsCmd.AddCommand(authPermissionsListCmd)

	authPermissionsListCmd.Flags().StringVar(&authPermissionsScope, "scope", "", "Filter by scope: account|zone|user")
	authPermissionsListCmd.Flags().StringVar(&authPermissionsGroup, "group", "", "Print the least-privilege permission set for this command group")
}

func runAuthPermissionsList(cmd *cobra.Command, args []string) error {
	if authPermissionsScope != "" &&
		authPermissionsScope != "account" &&
		authPermissionsScope != "zone" &&
		authPermissionsScope != "user" {
		return fmt.Errorf("invalid --scope %q: must be account, zone, or user", authPermissionsScope)
	}

	if authPermissionsGroup != "" {
		perms, ok := permdata.LeastPrivilege(authPermissionsGroup)
		if !ok {
			return fmt.Errorf("unknown command group %q: no least-privilege entry in the permission manifest", authPermissionsGroup)
		}

		if JSONOutput {
			return printSuccessJSON("Least-privilege permissions", map[string]interface{}{
				"command_group": authPermissionsGroup,
				"permissions":   perms,
			})
		}

		fmt.Printf("Least-privilege permissions for %q:\n", authPermissionsGroup)
		for _, p := range perms {
			fmt.Printf("  - %s\n", p)
		}
		return nil
	}

	families := permdata.Families(authPermissionsScope)
	sort.SliceStable(families, func(i, j int) bool {
		if families[i].Scope != families[j].Scope {
			return families[i].Scope < families[j].Scope
		}
		return families[i].Name < families[j].Name
	})

	if JSONOutput {
		return printSuccessJSON("Permission families", families)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSCOPE\tNAME\tREAD\tEDIT\tUSED BY")
	for _, f := range families {
		fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%v\t%s\n",
			f.ID, f.Scope, f.Name, f.Read, f.Edit, strings.Join(f.UsedBy, ", "))
	}
	w.Flush()

	return nil
}
