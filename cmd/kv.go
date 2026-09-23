package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var kvCmd = &cobra.Command{
	Use:   "kv",
	Short: "Manage Cloudflare Workers KV",
	Long: `KV namespace and key-value management for Cloudflare Workers KV.

Commands:
  namespace create   Create a KV namespace
  namespace list     List KV namespaces
  namespace delete   Delete a KV namespace
  put                Write a key-value pair
  get                Read a key-value pair
  delete             Delete a key
  list               List keys in a namespace

Examples:
  cosmoflare kv namespace create my-cache
  cosmoflare kv namespace list --json
  cosmoflare kv put ns-abc123 my-key --value="hello world"
  cosmoflare kv get ns-abc123 my-key
  cosmoflare kv list ns-abc123 --prefix=cache/`,
}

// namespace subcommands
var kvNamespaceCmd = &cobra.Command{
	Use:   "namespace",
	Short: "Manage KV namespaces",
	Long: `KV namespace management operations.

Commands:
  create   Create a new KV namespace
  list     List all KV namespaces
  delete   Delete a KV namespace`,
}

var (
	kvValue  string
	kvFile   string
	kvTTL    int64
	kvPrefix string
	kvLimit  int
	kvForce  bool
)

var kvNamespaceCreateCmd = &cobra.Command{
	Use:   "create [title]",
	Short: "Create a KV namespace",
	Long: `Create a new KV namespace.

A 400 error is returned if a namespace with this title already exists.

Examples:
  cosmoflare kv namespace create my-cache
  cosmoflare kv namespace create production-data --json`,
	// TASK-011: args[0] is the namespace TITLE (a resource name), so it is
	// profile-scoped at the cobra.Args level. cobra.ArbitraryArgs preserves
	// the previous nil-Args behavior; the runner still enforces presence.
	Args: prefixedResourceArgs(cobra.ArbitraryArgs),
	RunE: runKVNamespaceCreate,
}

var kvNamespaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all KV namespaces",
	Long: `List all KV namespaces in the current account.

Examples:
  cosmoflare kv namespace list
  cosmoflare kv namespace list --json`,
	RunE: runKVNamespaceList,
}

var kvNamespaceDeleteCmd = &cobra.Command{
	Use:   "delete [namespace-id]",
	Short: "Delete a KV namespace",
	Long: `Delete a KV namespace and all its keys.

WARNING: This action is irreversible.

Examples:
  cosmoflare kv namespace delete ns-abc123
  cosmoflare kv namespace delete ns-abc123 --force`,
	RunE: runKVNamespaceDelete,
}

var kvPutCmd = &cobra.Command{
	Use:   "put [namespace-id] [key]",
	Short: "Write a key-value pair",
	Long: `Write a key-value pair to a KV namespace.

Provide the value via --value flag or --file flag (file contents).

Examples:
  cosmoflare kv put ns-abc123 my-key --value="hello world"
  cosmoflare kv put ns-abc123 config.json --file=config.json
  cosmoflare kv put ns-abc123 session-123 --value="data" --ttl=3600`,
	RunE: runKVPut,
}

var kvGetCmd = &cobra.Command{
	Use:   "get [namespace-id] [key]",
	Short: "Read a key-value pair",
	Long: `Read a value from a KV namespace.

Examples:
  cosmoflare kv get ns-abc123 my-key
  cosmoflare kv get ns-abc123 my-key --json`,
	RunE: runKVGet,
}

var kvDeleteCmd = &cobra.Command{
	Use:   "delete [namespace-id] [key]",
	Short: "Delete a key",
	Long: `Delete a key from a KV namespace.

Examples:
  cosmoflare kv delete ns-abc123 my-key`,
	RunE: runKVDelete,
}

var kvListCmd = &cobra.Command{
	Use:   "list [namespace-id]",
	Short: "List keys in a namespace",
	Long: `List keys in a KV namespace with optional prefix filtering.

Examples:
  cosmoflare kv list ns-abc123
  cosmoflare kv list ns-abc123 --prefix=cache/
  cosmoflare kv list ns-abc123 --limit=100 --json`,
	RunE: runKVList,
}

func init() {
	rootCmd.AddCommand(kvCmd)

	kvCmd.AddCommand(kvNamespaceCmd)
	kvCmd.AddCommand(kvPutCmd)
	kvCmd.AddCommand(kvGetCmd)
	kvCmd.AddCommand(kvDeleteCmd)
	kvCmd.AddCommand(kvListCmd)

	kvNamespaceCmd.AddCommand(kvNamespaceCreateCmd)
	kvNamespaceCmd.AddCommand(kvNamespaceListCmd)
	kvNamespaceCmd.AddCommand(kvNamespaceDeleteCmd)

	kvNamespaceDeleteCmd.Flags().BoolVar(&kvForce, "force", false, "Skip confirmation prompt")

	kvPutCmd.Flags().StringVar(&kvValue, "value", "", "Value to write")
	kvPutCmd.Flags().StringVar(&kvFile, "file", "", "Read value from file")
	kvPutCmd.Flags().Int64Var(&kvTTL, "ttl", 0, "Expiration TTL in seconds")

	kvListCmd.Flags().StringVar(&kvPrefix, "prefix", "", "Filter keys by prefix")
	kvListCmd.Flags().IntVar(&kvLimit, "limit", 1000, "Maximum number of keys to return")
}

func getKVService() (*cosmoflare.KVService, error) {
	return cosmoflare.NewKVServiceFromCreds(AccountID, APIToken)
}

// filterKVNamespacesByProfile keeps only namespaces whose title carries the
// active profile's resource prefix (FEAT-026). The human title is scoped, not
// the server-assigned numeric ID. A no-op with no active prefix.
func filterKVNamespacesByProfile(namespaces []*cosmoflare.KVNamespace) []*cosmoflare.KVNamespace {
	filtered := make([]*cosmoflare.KVNamespace, 0, len(namespaces))
	for _, ns := range namespaces {
		if matchesResourcePrefix(ns.Title) {
			filtered = append(filtered, ns)
		}
	}
	return filtered
}

func runKVNamespaceCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("namespace title is required")
	}
	// TASK-011: args[0] is already profile-scoped by kvNamespaceCreateCmd.Args
	// (prefixedResourceArgs) in cobra-driven runs. The wrap is kept because
	// direct runner calls (tests) rely on the runner itself scoping the
	// title; applyResourcePrefix is idempotent, so this is a no-op after
	// the Args-level prefix.
	title := applyResourcePrefix(args[0])

	svc, err := getKVService()
	if err != nil {
		return outErr("failed to create KV service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would create namespace", func() any {
			return map[string]string{"title": title}
		}, func() {
			printInfo("DRY RUN: Would create namespace '%s'", title)
		})
	}

	ns, err := svc.CreateNamespace(context.Background(), title)
	if err != nil {
		return outErr("failed to create namespace", err)
	}

	return outPayload("Namespace created successfully", func() any {
		return ns
	}, func() {
		printSuccess("Namespace '%s' created (ID: %s)", ns.Title, ns.ID)
	})
}

func runKVNamespaceList(cmd *cobra.Command, args []string) error {
	svc, err := getKVService()
	if err != nil {
		return outErr("failed to create KV service", err)
	}

	namespaces, err := svc.ListNamespaces(context.Background())
	if err != nil {
		return outErr("failed to list namespaces", err)
	}
	namespaces = filterKVNamespacesByProfile(namespaces)

	return outResult(namespaces, func() {
		if len(namespaces) == 0 {
			printInfo("No KV namespaces found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tTITLE")
		for _, ns := range namespaces {
			fmt.Fprintf(w, "%s\t%s\n", ns.ID, ns.Title)
		}
		w.Flush()
		printInfo("Total: %d namespace(s)", len(namespaces))
	})
}

func runKVNamespaceDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("namespace ID is required")
	}
	id := args[0]

	force, _ := cmd.Flags().GetBool("force")
	// Registry-flagged destructive: runs dry unless --force, so no prompt.
	dry := destructiveDryRun(cliPathOfCmdOr(cmd, "kv namespace delete"), force)
	if dry {
		return outPayload("DRY RUN: Would delete namespace", func() any {
			return map[string]string{"id": id}
		}, func() {
			printInfo("DRY RUN: Would delete namespace '%s'", id)
		})
	}

	svc, err := getKVService()
	if err != nil {
		return outErr("failed to create KV service", err)
	}

	if err := svc.DeleteNamespace(context.Background(), id); err != nil {
		return outErr("failed to delete namespace", err)
	}

	return outPayload("Namespace deleted successfully", func() any {
		return map[string]string{"id": id}
	}, func() {
		printSuccess("Namespace '%s' deleted successfully!", id)
	})
}

func runKVPut(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("namespace ID and key are required")
	}
	namespaceID, key := args[0], args[1]

	if kvValue == "" && kvFile == "" {
		return fmt.Errorf("value is required (--value or --file)")
	}

	svc, err := getKVService()
	if err != nil {
		return outErr("failed to create KV service", err)
	}

	var valueReader *strings.Reader
	if kvFile != "" {
		data, err := os.ReadFile(kvFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		valueReader = strings.NewReader(string(data))
	} else {
		valueReader = strings.NewReader(kvValue)
	}

	var opts []cosmoflare.KVOption
	if kvTTL > 0 {
		opts = append(opts, cosmoflare.WithKVTTL(kvTTL))
	}

	if DryRun {
		return outPayload("DRY RUN: Would write key", func() any {
			return map[string]string{"namespace": namespaceID, "key": key}
		}, func() {
			printInfo("DRY RUN: Would write key '%s' to namespace '%s'", key, namespaceID)
		})
	}

	if err := svc.Put(context.Background(), namespaceID, key, valueReader, opts...); err != nil {
		return outErr("failed to write key", err)
	}

	return outPayload("Key written successfully", func() any {
		return map[string]string{"namespace": namespaceID, "key": key}
	}, func() {
		printSuccess("Key '%s' written to namespace '%s'", key, namespaceID)
	})
}

func runKVGet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("namespace ID and key are required")
	}
	namespaceID, key := args[0], args[1]

	svc, err := getKVService()
	if err != nil {
		return outErr("failed to create KV service", err)
	}

	data, err := svc.Get(context.Background(), namespaceID, key)
	if err != nil {
		return outErr("failed to get key", err)
	}

	return outResult(map[string]interface{}{
		"namespace": namespaceID,
		"key":       key,
		"value":     string(data),
		"size":      len(data),
	}, func() {
		fmt.Print(string(data))
		if len(data) > 0 && data[len(data)-1] != '\n' {
			fmt.Println()
		}
	})
}

func runKVDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("namespace ID and key are required")
	}
	namespaceID, key := args[0], args[1]

	svc, err := getKVService()
	if err != nil {
		return outErr("failed to create KV service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would delete key", func() any {
			return map[string]string{"namespace": namespaceID, "key": key}
		}, func() {
			printInfo("DRY RUN: Would delete key '%s' from namespace '%s'", key, namespaceID)
		})
	}

	if err := svc.Delete(context.Background(), namespaceID, key); err != nil {
		return outErr("failed to delete key", err)
	}

	return outPayload("Key deleted successfully", func() any {
		return map[string]string{"namespace": namespaceID, "key": key}
	}, func() {
		printSuccess("Key '%s' deleted from namespace '%s'", key, namespaceID)
	})
}

func runKVList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("namespace ID is required")
	}
	namespaceID := args[0]

	svc, err := getKVService()
	if err != nil {
		return outErr("failed to create KV service", err)
	}

	var opts []cosmoflare.KVListOption
	if kvPrefix != "" {
		opts = append(opts, cosmoflare.WithKVPrefix(kvPrefix))
	}
	if kvLimit > 0 {
		opts = append(opts, cosmoflare.WithKVLimit(kvLimit))
	}

	result, err := svc.ListKeys(context.Background(), namespaceID, opts...)
	if err != nil {
		return outErr("failed to list keys", err)
	}

	return outResult(result, func() {
		if len(result.Items) == 0 {
			printInfo("No keys found in namespace '%s'", namespaceID)
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "KEY\tEXPIRATION")
		for _, k := range result.Items {
			exp := "never"
			if k.Expiration > 0 {
				exp = fmt.Sprintf("%d", k.Expiration)
			}
			fmt.Fprintf(w, "%s\t%s\n", k.Key, exp)
		}
		w.Flush()
		printInfo("Total: %d key(s)", len(result.Items))
	})
}
