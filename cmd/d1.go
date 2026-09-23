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
	d1Force  bool
	d1SQL    string
	d1Params []string
	d1Local  bool
	d1Remote bool
)

var d1CreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a D1 database",
	Long: `Create a new D1 database.

The name must be unique within your account.

Examples:
  cosmoflare d1 create my-database
  cosmoflare d1 create production-db --json`,
	Args: prefixedResourceArgs(cobra.ExactArgs(1)),
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
	Use:     "query [database-id]",
	Aliases: []string{"execute"},
	Short:   "Execute SQL against a D1 database",
	Long: `Execute a SQL query against a D1 database.

Supports SELECT, INSERT, UPDATE, DELETE, CREATE TABLE, and other SQL statements.
Use --params to bind positional parameters (?1, ?2, ...) for safe query execution.

By default queries run against the remote D1 database; pass --local to run
them against a local SQLite state file instead (the database argument is then
used as the local database key). "execute" is an alias for "query".

Examples:
  cosmoflare d1 query <db-id> --sql="SELECT * FROM users"
  cosmoflare d1 execute <db-id> --sql="SELECT * FROM users" --local
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
	d1QueryCmd.Flags().BoolVar(&d1Local, "local", false, "Run against a local SQLite state file instead of the remote database")
	d1QueryCmd.Flags().BoolVar(&d1Remote, "remote", false, "Run against the remote D1 database (default behavior)")
	_ = d1QueryCmd.MarkFlagRequired("sql")
}

func getD1Service() (*cosmoflare.D1Service, error) {
	return cosmoflare.NewD1ServiceFromCreds(AccountID, APIToken)
}

func runD1Create(cmd *cobra.Command, args []string) error {
	// TASK-011: args[0] is already profile-scoped by d1CreateCmd.Args
	// (prefixedResourceArgs) in cobra-driven runs. The wrap is kept because
	// direct runner calls (tests) rely on the runner itself scoping the
	// name; applyResourcePrefix is idempotent, so this is a no-op after
	// the Args-level prefix.
	name := applyResourcePrefix(args[0])

	svc, err := getD1Service()
	if err != nil {
		return outErr("failed to create D1 service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would create database", func() any {
			return map[string]string{"name": name}
		}, func() {
			printInfo("DRY RUN: Would create database '%s'", name)
		})
	}

	db, err := svc.Create(context.Background(), name)
	if err != nil {
		return outErr("failed to create database", err)
	}

	return outPayload("Database created successfully", func() any {
		return db
	}, func() {
		printSuccess("Database '%s' created (ID: %s)", db.Name, db.UUID)
	})
}

func runD1List(cmd *cobra.Command, args []string) error {
	svc, err := getD1Service()
	if err != nil {
		return outErr("failed to create D1 service", err)
	}

	databases, err := svc.List(context.Background())
	if err != nil {
		return outErr("failed to list databases", err)
	}
	databases = filterD1ByPrefix(databases)

	return outResult(databases, func() {
		if len(databases) == 0 {
			printInfo("No D1 databases found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "UUID\tNAME\tTABLES\tSIZE\tVERSION")
		for _, db := range databases {
			size := formatBytes(db.FileSize)
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", db.UUID, db.Name, db.NumTables, size, db.Version)
		}
		w.Flush()
		printInfo("Total: %d database(s)", len(databases))
	})
}

func runD1Get(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	svc, err := getD1Service()
	if err != nil {
		return outErr("failed to create D1 service", err)
	}

	db, err := svc.Get(context.Background(), databaseID)
	if err != nil {
		return outErr("failed to get database", err)
	}

	return outResult(db, func() {
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
	})
}

func runD1Delete(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	// Registry-flagged destructive: runs dry unless --force, so no prompt.
	dry := destructiveDryRun(cliPathOfCmdOr(cmd, "d1 delete"), d1Force)
	if dry {
		return outPayload("DRY RUN: Would delete database", func() any {
			return map[string]string{"id": databaseID}
		}, func() {
			printInfo("DRY RUN: Would delete database '%s'", databaseID)
		})
	}

	svc, err := getD1Service()
	if err != nil {
		return outErr("failed to create D1 service", err)
	}

	if err := svc.Delete(context.Background(), databaseID); err != nil {
		return outErr("failed to delete database", err)
	}

	return outPayload("Database deleted successfully", func() any {
		return map[string]string{"id": databaseID}
	}, func() {
		printSuccess("Database '%s' deleted successfully!", databaseID)
	})
}

func runD1Query(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	if d1SQL == "" {
		return fmt.Errorf("--sql flag is required")
	}
	if d1Local && d1Remote {
		return fmt.Errorf("use only one of --local or --remote")
	}
	if d1Local {
		return runD1QueryLocal(databaseID)
	}

	svc, err := getD1Service()
	if err != nil {
		return outErr("failed to create D1 service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would execute query", func() any {
			return map[string]string{
				"database_id": databaseID,
				"sql":         d1SQL,
			}
		}, func() {
			printInfo("DRY RUN: Would execute query on database '%s'", databaseID)
		})
	}

	results, err := svc.Query(context.Background(), databaseID, d1SQL, d1Params...)
	if err != nil {
		return outErr("failed to execute query", err)
	}

	return outResult(results, func() {
		printD1Results(results)
	})
}

// printD1Results renders D1 query results in the plain (non-JSON) format:
// columns + rows for statements that returned rows, metadata for all.
func printD1Results(results []*cosmoflare.D1QueryResult) {
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
}

// filterD1ByPrefix keeps only databases whose name carries the active
// profile's resource prefix. The D1 API has no server-side prefix filter,
// so scoping is applied client-side.
func filterD1ByPrefix(databases []*cosmoflare.D1Database) []*cosmoflare.D1Database {
	if ActiveProfile == nil || ActiveProfile.ResourcePrefix == "" {
		return databases
	}
	filtered := make([]*cosmoflare.D1Database, 0, len(databases))
	for _, db := range databases {
		if matchesResourcePrefix(db.Name) {
			filtered = append(filtered, db)
		}
	}
	return filtered
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
