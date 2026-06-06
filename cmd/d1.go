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

var d1Cmd = &cobra.Command{
	Use:   "d1",
	Short: "Manage Cloudflare D1 databases",
	Long: `D1 database management for Cloudflare's serverless SQL databases.

Commands:
  create   Create a D1 database
  list     List D1 databases
  get      Get D1 database details
  delete   Delete a D1 database
  query    Execute SQL against a D1 database

Examples:
  cosmoflare d1 create my-database
  cosmoflare d1 list --json
  cosmoflare d1 get <database-id>
  cosmoflare d1 query <database-id> --sql="SELECT * FROM users"`,
}

var (
	d1Force bool
	d1SQL   string
	d1Params []string
)

var d1CreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a D1 database",
	Long: `Create a new D1 database.

The name must be unique within your account.

Examples:
  cosmoflare d1 create my-database
  cosmoflare d1 create production-db --json`,
	Args: cobra.ExactArgs(1),
	RunE: runD1Create,
}

var d1ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all D1 databases",
	Long: `List all D1 databases in the current account.

Examples:
  cosmoflare d1 list
  cosmoflare d1 list --json`,
	RunE: runD1List,
}

var d1GetCmd = &cobra.Command{
	Use:   "get [database-id]",
	Short: "Get D1 database details",
	Long: `Get details of a D1 database by its UUID.

Examples:
  cosmoflare d1 get 480f4f69-1a28-4fdd-9240-1ed29f0ac1df
  cosmoflare d1 get 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --json`,
	Args: cobra.ExactArgs(1),
	RunE: runD1Get,
}

var d1DeleteCmd = &cobra.Command{
	Use:   "delete [database-id]",
	Short: "Delete a D1 database",
	Long: `Delete a D1 database and all its data.

WARNING: This action is irreversible. All tables and data will be lost.

Examples:
  cosmoflare d1 delete 480f4f69-1a28-4fdd-9240-1ed29f0ac1df
  cosmoflare d1 delete 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --force`,
	Args: cobra.ExactArgs(1),
	RunE: runD1Delete,
}

var d1QueryCmd = &cobra.Command{
	Use:   "query [database-id]",
	Short: "Execute SQL against a D1 database",
	Long: `Execute a SQL query against a D1 database.

Supports SELECT, INSERT, UPDATE, DELETE, CREATE TABLE, and other SQL statements.
Use --params to bind positional parameters (?1, ?2, ...) for safe query execution.

Examples:
  cosmoflare d1 query <db-id> --sql="SELECT * FROM users"
  cosmoflare d1 query <db-id> --sql="SELECT * FROM users WHERE id = ?1" --param="42"
  cosmoflare d1 query <db-id> --sql="CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)" --json
  cosmoflare d1 query <db-id> --sql="INSERT INTO users (name) VALUES (?1)" --param="Alice" --json`,
	Args: cobra.ExactArgs(1),
	RunE: runD1Query,
}

func init() {
	rootCmd.AddCommand(d1Cmd)

	d1Cmd.AddCommand(d1CreateCmd)
	d1Cmd.AddCommand(d1ListCmd)
	d1Cmd.AddCommand(d1GetCmd)
	d1Cmd.AddCommand(d1DeleteCmd)
	d1Cmd.AddCommand(d1QueryCmd)

	d1DeleteCmd.Flags().BoolVar(&d1Force, "force", false, "Skip confirmation prompt")

	d1QueryCmd.Flags().StringVar(&d1SQL, "sql", "", "SQL query to execute")
	d1QueryCmd.Flags().StringArrayVar(&d1Params, "param", nil, "Positional query parameter (can be repeated)")
	_ = d1QueryCmd.MarkFlagRequired("sql")
}

func getD1Service() (*cosmoflare.D1Service, error) {
	return cosmoflare.NewD1ServiceFromCreds(AccountID, APIToken)
}

func runD1Create(cmd *cobra.Command, args []string) error {
	name := args[0]

	svc, err := getD1Service()
	if err != nil {
		return fmt.Errorf("failed to create D1 service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create database", map[string]string{"name": name})
		}
		printInfo("DRY RUN: Would create database '%s'", name)
		return nil
	}

	db, err := svc.Create(context.Background(), name)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create database: %v", err))
		}
		return fmt.Errorf("failed to create database: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Database created successfully", db)
	}
	printSuccess("Database '%s' created (ID: %s)", db.Name, db.UUID)
	return nil
}

func runD1List(cmd *cobra.Command, args []string) error {
	svc, err := getD1Service()
	if err != nil {
		return fmt.Errorf("failed to create D1 service: %w", err)
	}

	databases, err := svc.List(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list databases: %v", err))
		}
		return fmt.Errorf("failed to list databases: %w", err)
	}

	if JSONOutput {
		return printJSON(databases)
	}

	if len(databases) == 0 {
		printInfo("No D1 databases found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "UUID\tNAME\tTABLES\tSIZE\tVERSION")
	for _, db := range databases {
		size := formatBytes(db.FileSize)
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", db.UUID, db.Name, db.NumTables, size, db.Version)
	}
	w.Flush()
	printInfo("Total: %d database(s)", len(databases))
	return nil
}

func runD1Get(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	svc, err := getD1Service()
	if err != nil {
		return fmt.Errorf("failed to create D1 service: %w", err)
	}

	db, err := svc.Get(context.Background(), databaseID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get database: %v", err))
		}
		return fmt.Errorf("failed to get database: %w", err)
	}

	if JSONOutput {
		return printJSON(db)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "UUID:\t%s\n", db.UUID)
	fmt.Fprintf(w, "Name:\t%s\n", db.Name)
	fmt.Fprintf(w, "Version:\t%s\n", db.Version)
	fmt.Fprintf(w, "Tables:\t%d\n", db.NumTables)
	fmt.Fprintf(w, "Size:\t%s\n", formatBytes(db.FileSize))
	if db.CreatedAt != nil {
		fmt.Fprintf(w, "Created:\t%s\n", db.CreatedAt.Format("2006-01-02 15:04:05 UTC"))
	}
	w.Flush()
	return nil
}

func runD1Delete(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	if !d1Force && !DryRun {
		fmt.Printf("Are you sure you want to delete database '%s' and all its data? [y/N]: ", databaseID)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Database deletion cancelled")
			return nil
		}
	}

	svc, err := getD1Service()
	if err != nil {
		return fmt.Errorf("failed to create D1 service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete database", map[string]string{"id": databaseID})
		}
		printInfo("DRY RUN: Would delete database '%s'", databaseID)
		return nil
	}

	if err := svc.Delete(context.Background(), databaseID); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete database: %v", err))
		}
		return fmt.Errorf("failed to delete database: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Database deleted successfully", map[string]string{"id": databaseID})
	}
	printSuccess("Database '%s' deleted successfully!", databaseID)
	return nil
}

func runD1Query(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	if d1SQL == "" {
		return fmt.Errorf("--sql flag is required")
	}

	svc, err := getD1Service()
	if err != nil {
		return fmt.Errorf("failed to create D1 service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would execute query", map[string]string{
				"database_id": databaseID,
				"sql":         d1SQL,
			})
		}
		printInfo("DRY RUN: Would execute query on database '%s'", databaseID)
		return nil
	}

	results, err := svc.Query(context.Background(), databaseID, d1SQL, d1Params...)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to execute query: %v", err))
		}
		return fmt.Errorf("failed to execute query: %w", err)
	}

	if JSONOutput {
		return printJSON(results)
	}

	for i, result := range results {
		if i > 0 {
			fmt.Println()
		}

		if len(result.Rows) == 0 {
			printInfo("Query executed successfully (no rows returned)")
			printQueryMeta(result)
			continue
		}

		// Extract column names from the first row
		columns := extractColumns(result.Rows)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, strings.Join(columns, "\t"))
		for _, row := range result.Rows {
			vals := make([]string, 0, len(columns))
			for _, col := range columns {
				vals = append(vals, fmt.Sprintf("%v", row[col]))
			}
			fmt.Fprintln(w, strings.Join(vals, "\t"))
		}
		w.Flush()

		printQueryMeta(result)
	}
	return nil
}

// printQueryMeta prints metadata about a query result.
func printQueryMeta(result *cosmoflare.D1QueryResult) {
	printInfo("Rows read: %d | Rows written: %d | Changes: %d | Duration: %.2fms",
		result.Meta.RowsRead, result.Meta.RowsWritten, result.Meta.Changes, result.Meta.Duration)
}

// extractColumns returns column names from the first row of results in a stable order.
func extractColumns(rows []map[string]any) []string {
	if len(rows) == 0 {
		return nil
	}
	columns := make([]string, 0, len(rows[0]))
	for k := range rows[0] {
		columns = append(columns, k)
	}
	return columns
}

// formatBytes converts bytes to a human-readable string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
