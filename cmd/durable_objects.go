package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var doCmd = &cobra.Command{
	Use:   "do",
	Short: "Inspect Durable Objects live state",
	Long: `Inspect Durable Objects namespaces and object instances.

wrangler only manages class deployments and bindings; this group covers
live-state inspection that has no CLI coverage elsewhere: namespace
discovery, object listing, and single-object lookup.

Commands:
  namespaces  List Durable Objects namespaces
  objects     List object instances tracked under a namespace
  inspect     Look up a single object instance by ID

Examples:
  cosmoflare do namespaces
  cosmoflare do objects <namespace-id> --limit 50
  cosmoflare do inspect <namespace-id> <object-id>`,
}

var (
	doObjectsLimit  int
	doObjectsCursor string
)

var doNamespacesCmd = &cobra.Command{
	Use:   "namespaces",
	Short: "List Durable Objects namespaces",
	Long: `List all Durable Objects namespaces in the current account.

Examples:
  cosmoflare do namespaces
  cosmoflare do namespaces --json`,
	RunE: runDoNamespaces,
}

var doObjectsCmd = &cobra.Command{
	Use:   "objects [namespace-id]",
	Short: "List object instances tracked under a namespace",
	Long: `List Durable Object instances tracked under a namespace, paginated
via cursor.

Examples:
  cosmoflare do objects 480f4f69-1a28-4fdd-9240-1ed29f0ac1df
  cosmoflare do objects 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --limit 50
  cosmoflare do objects 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --cursor abc123 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runDoObjects,
}

var doInspectCmd = &cobra.Command{
	Use:   "inspect [namespace-id] [object-id]",
	Short: "Look up a single object instance by ID",
	Long: `Look up a single Durable Object instance by ID within a namespace.

The Durable Objects API has no dedicated per-object detail endpoint, so
this pages through the objects listing until it finds a match.

Examples:
  cosmoflare do inspect 480f4f69-1a28-4fdd-9240-1ed29f0ac1df 3b2e9d1c4a5f
  cosmoflare do inspect 480f4f69-1a28-4fdd-9240-1ed29f0ac1df 3b2e9d1c4a5f --json`,
	Args: cobra.ExactArgs(2),
	RunE: runDoInspect,
}

func init() {
	rootCmd.AddCommand(doCmd)

	doCmd.AddCommand(doNamespacesCmd)
	doCmd.AddCommand(doObjectsCmd)
	doCmd.AddCommand(doInspectCmd)

	doObjectsCmd.Flags().IntVar(&doObjectsLimit, "limit", 0, "Maximum number of objects to return (API default when unset)")
	doObjectsCmd.Flags().StringVar(&doObjectsCursor, "cursor", "", "Pagination cursor from a previous response")
}

func getDurableObjectsService() (*cosmoflare.DurableObjectsService, error) {
	return cosmoflare.NewDurableObjectsServiceFromCreds(AccountID, APIToken)
}

func runDoNamespaces(cmd *cobra.Command, args []string) error {
	svc, err := getDurableObjectsService()
	if err != nil {
		return fmt.Errorf("failed to create Durable Objects service: %w", err)
	}

	namespaces, err := svc.ListNamespaces(context.Background())
	if err != nil {
		return outErr("failed to list namespaces", err)
	}

	return outResult(namespaces, func() {
		if len(namespaces) == 0 {
			printInfo("No Durable Objects namespaces found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tSCRIPT")
		for _, ns := range namespaces {
			fmt.Fprintf(w, "%s\t%s\t%s\n", ns.ID, ns.Name, ns.Script)
		}
		w.Flush()
		printInfo("Total: %d namespace(s)", len(namespaces))
	})
}

func runDoObjects(cmd *cobra.Command, args []string) error {
	namespaceID := args[0]

	svc, err := getDurableObjectsService()
	if err != nil {
		return fmt.Errorf("failed to create Durable Objects service: %w", err)
	}

	result, err := svc.ListObjects(context.Background(), namespaceID, cosmoflare.ListObjectsOptions{
		Limit:  doObjectsLimit,
		Cursor: doObjectsCursor,
	})
	if err != nil {
		return outErr("failed to list objects", err)
	}

	return outResult(result, func() {
		if len(result.Objects) == 0 {
			printInfo("No objects found in namespace '%s'", namespaceID)
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tHAS STORED DATA")
		for _, obj := range result.Objects {
			fmt.Fprintf(w, "%s\t%t\n", obj.ID, obj.HasStoredData)
		}
		w.Flush()
		printInfo("Total: %d object(s)", len(result.Objects))
		if result.NextCursor != "" {
			printInfo("Next cursor: %s", result.NextCursor)
		}
	})
}

func runDoInspect(cmd *cobra.Command, args []string) error {
	namespaceID := args[0]
	objectID := args[1]

	svc, err := getDurableObjectsService()
	if err != nil {
		return fmt.Errorf("failed to create Durable Objects service: %w", err)
	}

	detail, err := svc.GetObject(context.Background(), namespaceID, objectID)
	if err != nil {
		return outErr("failed to inspect object", err)
	}

	return outResult(detail, func() {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "Namespace:\t%s\n", detail.NamespaceID)
		fmt.Fprintf(w, "Object ID:\t%s\n", detail.ID)
		fmt.Fprintf(w, "Has Stored Data:\t%t\n", detail.HasStoredData)
		w.Flush()
	})
}
