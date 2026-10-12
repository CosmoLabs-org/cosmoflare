package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	d1schema "github.com/CosmoLabs-org/cosmoflare/internal/d1schema"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// FEAT-051 slice: local-migrations vs remote D1 schema parity. Detection
// matrix classes 2-3 — the Churches-app drift (prod missing tables/columns
// local migrations had applied, and the reverse) found only by manual
// audit. Informational only: exit 0 unless the flags are wrong, so it can
// run anywhere a diff answer is wanted without paging anyone.

var (
	d1SchemaDiffDatabase   string
	d1SchemaDiffMigrations string
	// Test seam: when non-nil it REPLACES the remote read entirely (the
	// service argument is then unused); nil in production, where the runner
	// builds the real D1 service and reads through the query API.
	d1SchemaDiffFetchRemote func(ctx context.Context, svc *cosmoflare.D1Service, database string) (map[string][]string, error)
)

var d1SchemaDiffCmd = &cobra.Command{
	Use:   "schema-diff",
	Short: "Compare local migrations against a remote D1 database's schema",
	Long: `Compare local migrations against a remote D1 database's schema.

Reads every *.sql file in --migrations (sorted by filename — the usual
migration ordering), extracts CREATE TABLE / ALTER TABLE ADD COLUMN
statements into a local schema, fetches the remote database's tables and
columns through the D1 query API (sqlite_master + PRAGMA table_info), and
prints the parity verdict per table:

  ADDED-LOCAL-ONLY  table exists locally but not remotely (migration never applied)
  REMOTE-ONLY       table exists remotely but no migration creates it (reverse drift)
  COLUMN-DRIFT      table exists on both sides; columns differ (listed)
  MATCH             identical column sets

Informational only — the command exits 0 even when drift is found (and when
the remote read fails, the error is printed, not returned). Use --json for
the machine envelope.`,
	RunE: runD1SchemaDiff,
}

func init() {
	d1SchemaDiffCmd.Flags().StringVar(&d1SchemaDiffDatabase, "database", "", "remote D1 database name or ID (required)")
	d1SchemaDiffCmd.Flags().StringVar(&d1SchemaDiffMigrations, "migrations", "", "local migrations directory (required)")
	_ = d1SchemaDiffCmd.MarkFlagRequired("database")
	_ = d1SchemaDiffCmd.MarkFlagRequired("migrations")
	d1Cmd.AddCommand(d1SchemaDiffCmd)
}

func runD1SchemaDiff(cmd *cobra.Command, args []string) error {
	if d1SchemaDiffDatabase == "" {
		return outErrf("--database is required (the remote D1 database name or ID)")
	}
	if d1SchemaDiffMigrations == "" {
		return outErrf("--migrations is required (the local migrations directory)")
	}
	if info, err := os.Stat(d1SchemaDiffMigrations); err != nil || !info.IsDir() {
		return outErrf("migrations directory %q not found", d1SchemaDiffMigrations)
	}

	local, err := d1schema.ParseMigrationsDir(d1SchemaDiffMigrations)
	if err != nil {
		return outErr("failed to parse migrations", err)
	}

	// The remote read can fail for reasons this command must not turn into
	// an error (missing creds, missing D1 Read scope, network, wrong name) —
	// the failure is part of the answer. A stubbed seam (tests) never needs
	// credentials; only the real path builds the service.
	fetch := d1SchemaDiffFetchRemote
	if fetch == nil {
		svc, svcErr := getD1Service()
		if svcErr != nil {
			return outResult(map[string]any{"database": d1SchemaDiffDatabase, "error": svcErr.Error()}, func() {
				fmt.Fprintf(os.Stdout, "d1 schema-diff %s — remote read error: %v\n", d1SchemaDiffDatabase, svcErr)
				fmt.Fprintln(os.Stdout, "Check the database name and the token's D1 Read scope, then retry.")
			})
		}
		fetch = func(ctx context.Context, _ *cosmoflare.D1Service, database string) (map[string][]string, error) {
			return d1schema.RemoteSchema(ctx, svc, database)
		}
	}
	remote, fetchErr := fetch(context.Background(), nil, d1SchemaDiffDatabase)
	if fetchErr != nil {
		return outResult(map[string]any{"database": d1SchemaDiffDatabase, "error": fetchErr.Error()}, func() {
			fmt.Fprintf(os.Stdout, "d1 schema-diff %s — remote read error: %v\n", d1SchemaDiffDatabase, fetchErr)
			fmt.Fprintln(os.Stdout, "Check the database name and the token's D1 Read scope, then retry.")
		})
	}

	diffs := d1schema.Diff(local, remote)
	return outResult(
		struct {
			Database string               `json:"database"`
			Tables   []d1schema.TableDiff `json:"tables"`
			Legend   map[string]string    `json:"legend"`
		}{d1SchemaDiffDatabase, diffs, map[string]string{
			d1schema.DiffLocalOnly:   "table exists in local migrations but not remotely — migration never applied",
			d1schema.DiffRemoteOnly:  "table exists remotely but no migration creates it",
			d1schema.DiffColumnDrift: "table exists on both sides; listed columns differ",
			d1schema.DiffMatch:       "identical column sets",
		}},
		func() {
			fmt.Fprintf(os.Stdout, "d1 schema-diff — %s vs migrations in %s\n\n", d1SchemaDiffDatabase, d1SchemaDiffMigrations)
			for _, d := range diffs {
				fmt.Fprintf(os.Stdout, "%-22s %s\n", d.Table, d.Status)
				for _, col := range d.LocalOnly {
					fmt.Fprintf(os.Stdout, "    column ADDED-LOCAL-ONLY: %s\n", col)
				}
				for _, col := range d.RemoteOnly {
					fmt.Fprintf(os.Stdout, "    column remote-only: %s\n", col)
				}
			}
		},
	)
}
