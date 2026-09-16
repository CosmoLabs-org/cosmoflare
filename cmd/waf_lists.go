package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var wafListCmd = &cobra.Command{
	Use:   "list",
	Short: "Manage account-level WAF lists",
	Long: `Manage Cloudflare WAF rules lists (IP, ASN, redirect, hostname).

Lists are account-scoped, so CLOUDFLARE_ACCOUNT_ID (or --account-id) is required.

Commands:
  ls        List WAF lists
  create    Create a WAF list
  update    Update a WAF list description
  delete    Delete a WAF list
  item ls   List items in a WAF list
  item add  Append items to a WAF list
  item replace  Replace all items in a WAF list

Examples:
  cosmoflare waf list ls
  cosmoflare waf list create blocked-ips --kind=ip --json
  cosmoflare waf list item add LIST_ID --ip=1.2.3.4`,
}

var wafListItemCmd = &cobra.Command{
	Use:   "item",
	Short: "Manage items in a WAF list",
}

var wafManagedRulesetCmd = &cobra.Command{
	Use:   "managed-ruleset",
	Short: "Manage account-level managed rulesets",
	Long: `Enable or disable account-level managed rulesets by phase.

Examples:
  cosmoflare waf managed-ruleset update http_request_firewall_managed --mode=off`,
}

var (
	wafListKind        string
	wafListDescription string
	wafListName        string
	wafListForce       bool
	wafListItemIP      string
	wafListItemASN     uint32
	wafListItemComment string
	wafListItemFile    string
	wafManagedMode     string
)

var wafListLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List WAF lists",
	Long: `List all WAF lists on the account.

Examples:
  cosmoflare waf list ls
  cosmoflare waf list ls --json`,
	RunE: runWAFListLs,
}

var wafListCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a WAF list",
	Long: `Create a new WAF list.

Valid kinds: ip, asn, redirect, hostname

Examples:
  cosmoflare waf list create blocked-ips --kind=ip
  cosmoflare waf list create bad-asns --kind=asn --description="Known bad ASNs" --json`,
	RunE: runWAFListCreate,
}

var wafListUpdateCmd = &cobra.Command{
	Use:   "update [list-id]",
	Short: "Update a WAF list description",
	Long: `Update a WAF list's description.

The Cloudflare API does not support renaming lists, so only --description can be changed.

Examples:
  cosmoflare waf list update LIST_ID --description="Updated description"
  cosmoflare waf list update LIST_ID --description="Updated" --json`,
	RunE: runWAFListUpdate,
}

var wafListDeleteCmd = &cobra.Command{
	Use:   "delete [list-id]",
	Short: "Delete a WAF list",
	Long: `Delete a WAF list and all of its items.

WARNING: This action is irreversible.

Examples:
  cosmoflare waf list delete LIST_ID
  cosmoflare waf list delete LIST_ID --force`,
	RunE: runWAFListDelete,
}

var wafListItemLsCmd = &cobra.Command{
	Use:   "ls [list-id]",
	Short: "List items in a WAF list",
	Long: `List all items in a WAF list.

Examples:
  cosmoflare waf list item ls LIST_ID
  cosmoflare waf list item ls LIST_ID --json`,
	RunE: runWAFListItemLs,
}

var wafListItemAddCmd = &cobra.Command{
	Use:   "add [list-id]",
	Short: "Append items to a WAF list",
	Long: `Append one or more items to a WAF list.

Provide at least one of --ip or --asn; --comment applies to all added items.

Examples:
  cosmoflare waf list item add LIST_ID --ip=1.2.3.4
  cosmoflare waf list item add LIST_ID --ip=192.168.0.0/24 --comment="Office range"
  cosmoflare waf list item add LIST_ID --asn=13335 --json`,
	RunE: runWAFListItemAdd,
}

var wafListItemReplaceCmd = &cobra.Command{
	Use:   "replace [list-id]",
	Short: "Replace all items in a WAF list",
	Long: `Replace the entire item set of a WAF list with the contents of a JSON file.

The file must contain a JSON array of items, e.g.:
  [
    {"ip": "1.2.3.4", "comment": "scanner"},
    {"asn": 13335}
  ]

Examples:
  cosmoflare waf list item replace LIST_ID --items-file=items.json
  cosmoflare waf list item replace LIST_ID --items-file=items.json --json`,
	RunE: runWAFListItemReplace,
}

var wafManagedRulesetUpdateCmd = &cobra.Command{
	Use:   "update [phase]",
	Short: "Enable or disable a managed ruleset phase",
	Long: `Turn the managed ruleset in an account phase on or off.

The phase must already contain a managed ruleset entrypoint.

Examples:
  cosmoflare waf managed-ruleset update http_request_firewall_managed --mode=off
  cosmoflare waf managed-ruleset update http_request_firewall_managed --mode=on --json`,
	RunE: runWAFManagedRulesetUpdate,
}

func init() {
	wafCmd.AddCommand(wafListCmd)
	wafCmd.AddCommand(wafManagedRulesetCmd)

	wafListCmd.AddCommand(wafListLsCmd)
	wafListCmd.AddCommand(wafListCreateCmd)
	wafListCmd.AddCommand(wafListUpdateCmd)
	wafListCmd.AddCommand(wafListDeleteCmd)
	wafListCmd.AddCommand(wafListItemCmd)

	wafListItemCmd.AddCommand(wafListItemLsCmd)
	wafListItemCmd.AddCommand(wafListItemAddCmd)
	wafListItemCmd.AddCommand(wafListItemReplaceCmd)

	wafManagedRulesetCmd.AddCommand(wafManagedRulesetUpdateCmd)

	wafListCreateCmd.Flags().StringVar(&wafListKind, "kind", "", "List kind (ip, asn, redirect, hostname)")
	wafListCreateCmd.Flags().StringVar(&wafListDescription, "description", "", "Description for the list")
	wafListUpdateCmd.Flags().StringVar(&wafListName, "name", "", "New name (not supported by the Cloudflare API)")
	wafListUpdateCmd.Flags().StringVar(&wafListDescription, "description", "", "New description for the list")
	wafListDeleteCmd.Flags().BoolVar(&wafListForce, "force", false, "Skip confirmation prompt")

	wafListItemAddCmd.Flags().StringVar(&wafListItemIP, "ip", "", "IP address or CIDR range")
	wafListItemAddCmd.Flags().Uint32Var(&wafListItemASN, "asn", 0, "Autonomous System Number")
	wafListItemAddCmd.Flags().StringVar(&wafListItemComment, "comment", "", "Comment for the added items")
	wafListItemReplaceCmd.Flags().StringVar(&wafListItemFile, "items-file", "", "JSON file with the full item set")

	wafManagedRulesetUpdateCmd.Flags().StringVar(&wafManagedMode, "mode", "", "Managed ruleset mode (on, off)")
}

func getWAFListService() (*cosmoflare.WAFListService, error) {
	return cosmoflare.NewWAFListServiceFromCreds(AccountID, APIToken)
}

func runWAFListLs(cmd *cobra.Command, args []string) error {
	svc, err := getWAFListService()
	if err != nil {
		return outErr("failed to create WAF list service", err)
	}
	lists, err := svc.ListLists(context.Background())
	if err != nil {
		return outErr("failed to list WAF lists", err)
	}
	return outResult(lists, func() {
		if len(lists) == 0 {
			printInfo("No WAF lists found")
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tKIND\tITEMS\tDESCRIPTION")
		for _, l := range lists {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", l.ID, l.Name, l.Kind, l.NumItems, l.Description)
		}
		w.Flush()
		printInfo("Total: %d list(s)", len(lists))
	})
}

func runWAFListCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("list name is required")
	}
	if wafListKind == "" {
		return fmt.Errorf("--kind is required (ip, asn, redirect, hostname)")
	}
	name := args[0]
	svc, err := getWAFListService()
	if err != nil {
		return outErr("failed to create WAF list service", err)
	}
	if DryRun {
		return outPayload("DRY RUN: Would create WAF list", func() any {
			return map[string]string{"name": name, "kind": wafListKind, "description": wafListDescription}
		}, func() {
			printInfo("DRY RUN: Would create WAF list '%s' kind=%s", name, wafListKind)
		})
	}
	list, err := svc.CreateList(context.Background(), name, wafListKind, wafListDescription)
	if err != nil {
		return outErr("failed to create WAF list", err)
	}
	return outPayload("WAF list created", func() any {
		return list
	}, func() {
		printSuccess("WAF list '%s' created (%s)", list.ID, list.Kind)
	})
}

func runWAFListUpdate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("list ID is required")
	}
	listID := args[0]
	if wafListName != "" {
		return fmt.Errorf("renaming a WAF list is not supported by the Cloudflare API; only --description can be updated")
	}
	if wafListDescription == "" {
		return fmt.Errorf("--description is required")
	}
	svc, err := getWAFListService()
	if err != nil {
		return outErr("failed to create WAF list service", err)
	}
	if DryRun {
		return outPayload("DRY RUN: Would update WAF list", func() any {
			return map[string]string{"list_id": listID, "description": wafListDescription}
		}, func() {
			printInfo("DRY RUN: Would update description of WAF list '%s'", listID)
		})
	}
	list, err := svc.UpdateList(context.Background(), listID, "", wafListDescription)
	if err != nil {
		return outErr("failed to update WAF list", err)
	}
	return outPayload("WAF list updated", func() any {
		return list
	}, func() {
		printSuccess("WAF list '%s' description updated", listID)
	})
}

func runWAFListDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("list ID is required")
	}
	listID := args[0]

	if !wafListForce && !DryRun {
		fmt.Printf("Are you sure you want to delete WAF list '%s'? [y/N]: ", listID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("WAF list deletion cancelled")
			return nil
		}
	}
	svc, err := getWAFListService()
	if err != nil {
		return outErr("failed to create WAF list service", err)
	}
	if DryRun {
		return outPayload("DRY RUN: Would delete WAF list", func() any {
			return map[string]string{"list_id": listID}
		}, func() {
			printInfo("DRY RUN: Would delete WAF list '%s'", listID)
		})
	}
	if err := svc.DeleteList(context.Background(), listID); err != nil {
		return outErr("failed to delete WAF list", err)
	}
	return outPayload("WAF list deleted", func() any {
		return map[string]string{"list_id": listID}
	}, func() {
		printSuccess("WAF list '%s' deleted", listID)
	})
}

func runWAFListItemLs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("list ID is required")
	}
	svc, err := getWAFListService()
	if err != nil {
		return outErr("failed to create WAF list service", err)
	}
	items, err := svc.ListItems(context.Background(), args[0])
	if err != nil {
		return outErr("failed to list WAF list items", err)
	}
	return outResult(items, func() {
		if len(items) == 0 {
			printInfo("No items found in list '%s'", args[0])
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tIP\tASN\tCOMMENT")
		for _, it := range items {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", it.ID, it.IP, it.ASN, it.Comment)
		}
		w.Flush()
		printInfo("Total: %d item(s)", len(items))
	})
}

func runWAFListItemAdd(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("list ID is required")
	}
	if wafListItemIP == "" && wafListItemASN == 0 {
		return fmt.Errorf("at least one of --ip or --asn is required")
	}
	listID := args[0]

	items := make([]cosmoflare.WAFListItem, 0, 2)
	if wafListItemIP != "" {
		items = append(items, cosmoflare.WAFListItem{IP: wafListItemIP, Comment: wafListItemComment})
	}
	if wafListItemASN != 0 {
		items = append(items, cosmoflare.WAFListItem{ASN: wafListItemASN, Comment: wafListItemComment})
	}

	svc, err := getWAFListService()
	if err != nil {
		return outErr("failed to create WAF list service", err)
	}
	if DryRun {
		return outPayload("DRY RUN: Would add items to WAF list", func() any {
			return map[string]any{"list_id": listID, "items": items}
		}, func() {
			printInfo("DRY RUN: Would add %d item(s) to WAF list '%s'", len(items), listID)
		})
	}
	result, err := svc.AddItems(context.Background(), listID, items)
	if err != nil {
		return outErr("failed to add items to WAF list", err)
	}
	return outPayload("Items added to WAF list", func() any {
		return result
	}, func() {
		printSuccess("%d item(s) added to WAF list '%s'", len(items), listID)
	})
}

func runWAFListItemReplace(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("list ID is required")
	}
	if wafListItemFile == "" {
		return fmt.Errorf("--items-file is required")
	}
	listID := args[0]

	data, err := os.ReadFile(wafListItemFile)
	if err != nil {
		return fmt.Errorf("failed to read items file: %w", err)
	}
	var items []cosmoflare.WAFListItem
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("failed to parse items file as JSON array of items: %w", err)
	}
	if len(items) == 0 {
		return fmt.Errorf("items file contains no items")
	}

	svc, err := getWAFListService()
	if err != nil {
		return outErr("failed to create WAF list service", err)
	}
	if DryRun {
		return outPayload("DRY RUN: Would replace items in WAF list", func() any {
			return map[string]any{"list_id": listID, "num_items": len(items)}
		}, func() {
			printInfo("DRY RUN: Would replace all items in WAF list '%s' with %d item(s)", listID, len(items))
		})
	}
	result, err := svc.ReplaceItems(context.Background(), listID, items)
	if err != nil {
		return outErr("failed to replace items in WAF list", err)
	}
	return outPayload("WAF list items replaced", func() any {
		return result
	}, func() {
		printSuccess("WAF list '%s' now contains %d item(s)", listID, len(result))
	})
}

func runWAFManagedRulesetUpdate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("ruleset phase is required")
	}
	if wafManagedMode != "on" && wafManagedMode != "off" {
		return fmt.Errorf("--mode is required and must be on or off")
	}
	phase := args[0]

	svc, err := getWAFListService()
	if err != nil {
		return outErr("failed to create WAF list service", err)
	}
	if DryRun {
		return outPayload("DRY RUN: Would update managed ruleset", func() any {
			return map[string]string{"phase": phase, "mode": wafManagedMode}
		}, func() {
			printInfo("DRY RUN: Would set managed ruleset phase '%s' to mode '%s'", phase, wafManagedMode)
		})
	}
	result, err := svc.UpdateManagedRuleset(context.Background(), phase, wafManagedMode)
	if err != nil {
		return outErr("failed to update managed ruleset", err)
	}
	return outPayload("Managed ruleset updated", func() any {
		return result
	}, func() {
		printSuccess("Managed ruleset phase '%s' set to '%s' (%d rule(s))", phase, wafManagedMode, result.NumRules)
	})
}
