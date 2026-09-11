package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var (
	d1MigrationsDir    string
	d1MigrationsLocal  bool
	d1MigrationsRemote bool
	d1MigrationsForce  bool
)

var d1MigrationsCmd = &cobra.Command{
	Use:   "migrations",
	Short: "Manage D1 schema migrations",
	Long: `Manage schema migrations for a D1 database, tracked in the
d1_migrations table.

Commands:
  create   Create a new migration file
  list     List migrations and their applied state
  apply    Apply pending migrations

Examples:
  cosmoflare d1 migrations create <database-id> add_users_table
  cosmoflare d1 migrations list <database-id>
  cosmoflare d1 migrations apply <database-id> --dry-run`,
}

var d1MigrationsCreateCmd = &cobra.Command{
	Use:   "create [database-id] [migration-name]",
	Short: "Create a new migration file",
	Long: `Create a new migration file in the migrations directory, named
NNNN_<name>.sql where NNNN is the next sequence number.

Examples:
  cosmoflare d1 migrations create 480f4f69-1a28-4fdd-9240-1ed29f0ac1df add_users_table
  cosmoflare d1 migrations create 480f4f69-1a28-4fdd-9240-1ed29f0ac1df add_users_table --migrations-dir db/migrations`,
	Args: cobra.ExactArgs(2),
	RunE: runD1MigrationsCreate,
}

var d1MigrationsListCmd = &cobra.Command{
	Use:   "list [database-id]",
	Short: "List migrations and their applied state",
	Long: `List all migration files and whether each has been applied to
the database.

Examples:
  cosmoflare d1 migrations list 480f4f69-1a28-4fdd-9240-1ed29f0ac1df
  cosmoflare d1 migrations list 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --json`,
	Args: cobra.ExactArgs(1),
	RunE: runD1MigrationsList,
}

var d1MigrationsApplyCmd = &cobra.Command{
	Use:   "apply [database-id]",
	Short: "Apply pending migrations",
	Long: `Apply all pending migrations to the database, in sequence order.
Already-applied migrations are skipped.

Migrations containing destructive SQL (DROP TABLE, DROP INDEX,
DROP DATABASE, TRUNCATE, or DELETE without WHERE) print a warning before
execution. Use --force to skip the confirmation prompt.

Examples:
  cosmoflare d1 migrations apply 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --dry-run
  cosmoflare d1 migrations apply 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --force`,
	Args: cobra.ExactArgs(1),
	RunE: runD1MigrationsApply,
}

func init() {
	d1Cmd.AddCommand(d1MigrationsCmd)

	d1MigrationsCmd.AddCommand(d1MigrationsCreateCmd)
	d1MigrationsCmd.AddCommand(d1MigrationsListCmd)
	d1MigrationsCmd.AddCommand(d1MigrationsApplyCmd)

	d1MigrationsCmd.PersistentFlags().StringVar(&d1MigrationsDir, "migrations-dir", "migrations", "Directory containing migration files")
	d1MigrationsCmd.PersistentFlags().BoolVar(&d1MigrationsLocal, "local", false, "Operate on the local D1 database (not yet supported)")
	d1MigrationsCmd.PersistentFlags().BoolVar(&d1MigrationsRemote, "remote", true, "Operate on the remote D1 database (default)")

	d1MigrationsApplyCmd.Flags().BoolVar(&d1MigrationsForce, "force", false, "Skip confirmation prompt")
}

// runD1MigrationsCreate creates a migration file locally. The database
// argument (args[0]) is accepted for command symmetry with list/apply but
// is not required by MigrationsCreate, which is filesystem-only.
func runD1MigrationsCreate(cmd *cobra.Command, args []string) error {
	name := args[1]

	svc, err := getD1Service()
	if err != nil {
		return fmt.Errorf("failed to create D1 service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create migration", map[string]string{"name": name, "migrations_dir": d1MigrationsDir})
		}
		printInfo("DRY RUN: Would create migration '%s' in '%s'", name, d1MigrationsDir)
		return nil
	}

	path, err := svc.MigrationsCreate(name, d1MigrationsDir)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create migration: %v", err))
		}
		return fmt.Errorf("failed to create migration: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Migration created successfully", map[string]string{"file_path": path})
	}
	printSuccess("Migration created: %s", path)
	return nil
}

func runD1MigrationsList(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	if d1MigrationsLocal {
		return fmt.Errorf("--local D1 execution is not yet supported; use --remote")
	}

	svc, err := getD1Service()
	if err != nil {
		return fmt.Errorf("failed to create D1 service: %w", err)
	}

	migrations, err := svc.MigrationsList(context.Background(), databaseID, d1MigrationsDir)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list migrations: %v", err))
		}
		return fmt.Errorf("failed to list migrations: %w", err)
	}

	if JSONOutput {
		return printJSON(migrations)
	}

	if len(migrations) == 0 {
		printInfo("No migrations found in '%s'", d1MigrationsDir)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tAPPLIED AT")
	for _, m := range migrations {
		status := "pending"
		appliedAt := "-"
		if m.AppliedAt != nil {
			status = "applied"
			appliedAt = m.AppliedAt.Format("2006-01-02 15:04:05 UTC")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", m.Name, status, appliedAt)
	}
	w.Flush()
	return nil
}

func runD1MigrationsApply(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	if d1MigrationsLocal {
		return fmt.Errorf("--local D1 execution is not yet supported; use --remote")
	}

	svc, err := getD1Service()
	if err != nil {
		return fmt.Errorf("failed to create D1 service: %w", err)
	}

	if !d1MigrationsForce && !DryRun {
		pending, err := svc.MigrationsList(context.Background(), databaseID, d1MigrationsDir)
		if err != nil {
			return fmt.Errorf("failed to list migrations: %w", err)
		}

		var pendingCount int
		for _, m := range pending {
			if m.AppliedAt != nil {
				continue
			}
			pendingCount++
			sqlBytes, err := os.ReadFile(m.FilePath)
			if err == nil && cosmoflare.ContainsDestructiveSQL(string(sqlBytes)) {
				printWarning("migration '%s' contains destructive SQL (DROP/TRUNCATE/DELETE without WHERE)", m.Name)
			}
		}

		if pendingCount == 0 {
			printInfo("No pending migrations")
			return nil
		}

		fmt.Printf("Type the database ID '%s' to confirm applying %d migration(s): ", databaseID, pendingCount)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)
		if response != databaseID {
			printInfo("Migration apply cancelled")
			return nil
		}
	}

	results, err := svc.MigrationsApply(context.Background(), databaseID, d1MigrationsDir, DryRun)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to apply migrations: %v", err))
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	if JSONOutput {
		return printJSON(results)
	}

	failed := 0
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tERROR")
	for _, r := range results {
		if r.Status == "failed" {
			failed++
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.Name, r.Status, r.Error)
	}
	w.Flush()

	if failed > 0 {
		return fmt.Errorf("%d migration(s) failed", failed)
	}

	printSuccess("Migrations applied")
	return nil
}
