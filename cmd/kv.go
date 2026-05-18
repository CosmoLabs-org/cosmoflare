package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"
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
  r2go2 kv namespace create my-cache
  r2go2 kv namespace list --json
  r2go2 kv put ns-abc123 my-key --value="hello world"
  r2go2 kv get ns-abc123 my-key
  r2go2 kv list ns-abc123 --prefix=cache/`,
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
  r2go2 kv namespace create my-cache
  r2go2 kv namespace create production-data --json`,
	RunE: runKVNamespaceCreate,
}

var kvNamespaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all KV namespaces",
	Long: `List all KV namespaces in the current account.

Examples:
  r2go2 kv namespace list
  r2go2 kv namespace list --json`,
	RunE: runKVNamespaceList,
}

var kvNamespaceDeleteCmd = &cobra.Command{
	Use:   "delete [namespace-id]",
	Short: "Delete a KV namespace",
	Long: `Delete a KV namespace and all its keys.

WARNING: This action is irreversible.

Examples:
  r2go2 kv namespace delete ns-abc123
  r2go2 kv namespace delete ns-abc123 --force`,
	RunE: runKVNamespaceDelete,
}

var kvPutCmd = &cobra.Command{
	Use:   "put [namespace-id] [key]",
	Short: "Write a key-value pair",
	Long: `Write a key-value pair to a KV namespace.

Provide the value via --value flag or --file flag (file contents).

Examples:
  r2go2 kv put ns-abc123 my-key --value="hello world"
  r2go2 kv put ns-abc123 config.json --file=config.json
  r2go2 kv put ns-abc123 session-123 --value="data" --ttl=3600`,
	RunE: runKVPut,
}

var kvGetCmd = &cobra.Command{
	Use:   "get [namespace-id] [key]",
	Short: "Read a key-value pair",
	Long: `Read a value from a KV namespace.

Examples:
  r2go2 kv get ns-abc123 my-key
  r2go2 kv get ns-abc123 my-key --json`,
	RunE: runKVGet,
}

var kvDeleteCmd = &cobra.Command{
	Use:   "delete [namespace-id] [key]",
	Short: "Delete a key",
	Long: `Delete a key from a KV namespace.

Examples:
  r2go2 kv delete ns-abc123 my-key`,
	RunE: runKVDelete,
}

var kvListCmd = &cobra.Command{
	Use:   "list [namespace-id]",
	Short: "List keys in a namespace",
	Long: `List keys in a KV namespace with optional prefix filtering.

Examples:
  r2go2 kv list ns-abc123
  r2go2 kv list ns-abc123 --prefix=cache/
  r2go2 kv list ns-abc123 --limit=100 --json`,
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

func getKVService() (*r2go2.KVService, error) {
	return r2go2.NewKVServiceFromCreds(AccountID, APIToken)
}

func runKVNamespaceCreate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("namespace title is required")
	}
	title := args[0]

	svc, err := getKVService()
	if err != nil {
		return fmt.Errorf("failed to create KV service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create namespace", map[string]string{"title": title})
		}
		printInfo("DRY RUN: Would create namespace '%s'", title)
		return nil
	}

	ns, err := svc.CreateNamespace(context.Background(), title)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create namespace: %v", err))
		}
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Namespace created successfully", ns)
	}
	printSuccess("Namespace '%s' created (ID: %s)", ns.Title, ns.ID)
	return nil
}

func runKVNamespaceList(cmd *cobra.Command, args []string) error {
	svc, err := getKVService()
	if err != nil {
		return fmt.Errorf("failed to create KV service: %w", err)
	}

	namespaces, err := svc.ListNamespaces(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list namespaces: %v", err))
		}
		return fmt.Errorf("failed to list namespaces: %w", err)
	}

	if JSONOutput {
		return printJSON(namespaces)
	}

	if len(namespaces) == 0 {
		printInfo("No KV namespaces found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE")
	for _, ns := range namespaces {
		fmt.Fprintf(w, "%s\t%s\n", ns.ID, ns.Title)
	}
	w.Flush()
	printInfo("Total: %d namespace(s)", len(namespaces))
	return nil
}

func runKVNamespaceDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("namespace ID is required")
	}
	id := args[0]

	force, _ := cmd.Flags().GetBool("force")
	if !force && !DryRun {
		fmt.Printf("Are you sure you want to delete namespace '%s' and all its keys? [y/N]: ", id)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Namespace deletion cancelled")
			return nil
		}
	}

	svc, err := getKVService()
	if err != nil {
		return fmt.Errorf("failed to create KV service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete namespace", map[string]string{"id": id})
		}
		printInfo("DRY RUN: Would delete namespace '%s'", id)
		return nil
	}

	if err := svc.DeleteNamespace(context.Background(), id); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete namespace: %v", err))
		}
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Namespace deleted successfully", map[string]string{"id": id})
	}
	printSuccess("Namespace '%s' deleted successfully!", id)
	return nil
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
		return fmt.Errorf("failed to create KV service: %w", err)
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

	var opts []r2go2.KVOption
	if kvTTL > 0 {
		opts = append(opts, r2go2.WithKVTTL(kvTTL))
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would write key", map[string]string{"namespace": namespaceID, "key": key})
		}
		printInfo("DRY RUN: Would write key '%s' to namespace '%s'", key, namespaceID)
		return nil
	}

	if err := svc.Put(context.Background(), namespaceID, key, valueReader, opts...); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to write key: %v", err))
		}
		return fmt.Errorf("failed to write key: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Key written successfully", map[string]string{"namespace": namespaceID, "key": key})
	}
	printSuccess("Key '%s' written to namespace '%s'", key, namespaceID)
	return nil
}

func runKVGet(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("namespace ID and key are required")
	}
	namespaceID, key := args[0], args[1]

	svc, err := getKVService()
	if err != nil {
		return fmt.Errorf("failed to create KV service: %w", err)
	}

	data, err := svc.Get(context.Background(), namespaceID, key)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get key: %v", err))
		}
		return fmt.Errorf("failed to get key: %w", err)
	}

	if JSONOutput {
		return printJSON(map[string]interface{}{
			"namespace": namespaceID,
			"key":       key,
			"value":     string(data),
			"size":      len(data),
		})
	}

	fmt.Print(string(data))
	if len(data) > 0 && data[len(data)-1] != '\n' {
		fmt.Println()
	}
	return nil
}

func runKVDelete(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("namespace ID and key are required")
	}
	namespaceID, key := args[0], args[1]

	svc, err := getKVService()
	if err != nil {
		return fmt.Errorf("failed to create KV service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete key", map[string]string{"namespace": namespaceID, "key": key})
		}
		printInfo("DRY RUN: Would delete key '%s' from namespace '%s'", key, namespaceID)
		return nil
	}

	if err := svc.Delete(context.Background(), namespaceID, key); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete key: %v", err))
		}
		return fmt.Errorf("failed to delete key: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Key deleted successfully", map[string]string{"namespace": namespaceID, "key": key})
	}
	printSuccess("Key '%s' deleted from namespace '%s'", key, namespaceID)
	return nil
}

func runKVList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("namespace ID is required")
	}
	namespaceID := args[0]

	svc, err := getKVService()
	if err != nil {
		return fmt.Errorf("failed to create KV service: %w", err)
	}

	var opts []r2go2.KVListOption
	if kvPrefix != "" {
		opts = append(opts, r2go2.WithKVPrefix(kvPrefix))
	}
	if kvLimit > 0 {
		opts = append(opts, r2go2.WithKVLimit(kvLimit))
	}

	result, err := svc.ListKeys(context.Background(), namespaceID, opts...)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list keys: %v", err))
		}
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if JSONOutput {
		return printJSON(result)
	}

	if len(result.Items) == 0 {
		printInfo("No keys found in namespace '%s'", namespaceID)
		return nil
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
	return nil
}
