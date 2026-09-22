package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

// FEAT-018 P1: local-vs-remote row-count parity for D1 tables. The command
// lives in its own file (own init) so cmd/d1.go stays untouched.

var (
	d1ParityLocalPath string
	d1ParityTables    string
	d1ParityNoFail    bool
)

var d1ParityCmd = &cobra.Command{
	Use:   "parity [database-id]",
	Short: "Compare local and remote D1 table row counts",
	Long: `Compare row counts between a local SQLite file and the remote D1 database.

For each table listed in --tables, the command runs SELECT COUNT(*) on both
sides and reports whether the counts match exactly. Rows are printed as a
table (human mode) or a JSON array of {table, local, remote, match} objects
(--json). A per-side failure is reported in the row's error field instead of
aborting the run.

The command exits non-zero when any table mismatches (usable from agents and
scripts); pass --no-fail to always exit zero.

Examples:
  cosmoflare d1 parity <database-id> --local .wrangler/state/v3/d1/db.sqlite --tables users,orders
  cosmoflare d1 parity <database-id> --local ./local.sqlite --tables users --json
  cosmoflare d1 parity <database-id> --local ./local.sqlite --tables users,orders --no-fail`,
	Args: cobra.ExactArgs(1),
	RunE: runD1Parity,
}

func init() {
	d1Cmd.AddCommand(d1ParityCmd)

	d1ParityCmd.Flags().StringVar(&d1ParityLocalPath, "local", "", "Path to the local SQLite state file to compare against")
	d1ParityCmd.Flags().StringVar(&d1ParityTables, "tables", "", "Comma-separated list of tables to compare")
	d1ParityCmd.Flags().BoolVar(&d1ParityNoFail, "no-fail", false, "Exit zero even when tables mismatch")
	_ = d1ParityCmd.MarkFlagRequired("local")
	_ = d1ParityCmd.MarkFlagRequired("tables")
}

// d1ParityRow is one table's comparison result. Local/Remote are pointers so
// an unreachable side stays absent from the JSON instead of reading as zero.
type d1ParityRow struct {
	Table  string `json:"table"`
	Local  *int64 `json:"local,omitempty"`
	Remote *int64 `json:"remote,omitempty"`
	Match  bool   `json:"match"`
	Error  string `json:"error,omitempty"`
}

// parseD1ParityTables splits the --tables flag value on commas, trimming
// whitespace and dropping empty entries.
func parseD1ParityTables(flagValue string) []string {
	var tables []string
	for _, t := range strings.Split(flagValue, ",") {
		if t = strings.TrimSpace(t); t != "" {
			tables = append(tables, t)
		}
	}
	return tables
}

// buildD1ParityRow assembles one row. Match requires both sides to have been
// counted without error and the counts to be exactly equal; any side error is
// flattened into the row's error field so the run never crashes.
func buildD1ParityRow(table string, localCount, remoteCount int64, localErr, remoteErr error) d1ParityRow {
	row := d1ParityRow{Table: table}
	if localErr == nil {
		row.Local = &localCount
	}
	if remoteErr == nil {
		row.Remote = &remoteCount
	}
	switch {
	case localErr != nil && remoteErr != nil:
		row.Error = fmt.Sprintf("local: %v; remote: %v", localErr, remoteErr)
	case localErr != nil:
		row.Error = fmt.Sprintf("local: %v", localErr)
	case remoteErr != nil:
		row.Error = fmt.Sprintf("remote: %v", remoteErr)
	default:
		row.Match = localCount == remoteCount
	}
	return row
}

// d1ParityMismatchCount returns how many rows failed to match (mismatched
// counts or a side error).
func d1ParityMismatchCount(rows []d1ParityRow) int {
	n := 0
	for _, r := range rows {
		if !r.Match {
			n++
		}
	}
	return n
}

// runD1Parity compares per-table row counts between the --local SQLite file
// and the remote D1 database identified by the positional argument.
func runD1Parity(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	tables := parseD1ParityTables(d1ParityTables)
	if len(tables) == 0 {
		return fmt.Errorf("--tables must list at least one table")
	}
	if d1ParityLocalPath == "" {
		return fmt.Errorf("--local is required")
	}
	if fi, err := os.Stat(d1ParityLocalPath); err != nil || fi.IsDir() {
		return fmt.Errorf("--local file %q not found", d1ParityLocalPath)
	}

	// Remote side is unreachable-safe: a missing credentials/service error is
	// recorded per row, not fatal, so the local side still reports.
	svc, svcErr := getD1Service()

	rows := make([]d1ParityRow, 0, len(tables))
	for _, table := range tables {
		localCount, localErr := d1ParityLocalCount(d1ParityLocalPath, table)

		var remoteCount int64
		var remoteErr error
		if svcErr != nil {
			remoteErr = svcErr
		} else {
			remoteCount, remoteErr = d1ParityRemoteCount(context.Background(), svc, databaseID, table)
		}

		rows = append(rows, buildD1ParityRow(table, localCount, remoteCount, localErr, remoteErr))
	}

	err := outResult(rows, func() { printD1ParityRows(rows) })
	if err != nil {
		return err
	}

	if mismatches := d1ParityMismatchCount(rows); mismatches > 0 && !d1ParityNoFail {
		return outErrf("d1 parity failed: %d of %d table(s) mismatch", mismatches, len(rows))
	}
	return nil
}

// printD1ParityRows renders the human table plus a summary line.
func printD1ParityRows(rows []d1ParityRow) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TABLE\tLOCAL\tREMOTE\tMATCH")
	for _, r := range rows {
		local, remote := "-", "-"
		if r.Local != nil {
			local = fmt.Sprintf("%d", *r.Local)
		}
		if r.Remote != nil {
			remote = fmt.Sprintf("%d", *r.Remote)
		}
		match := "yes"
		if !r.Match {
			match = "NO"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Table, local, remote, match)
		if r.Error != "" {
			fmt.Fprintf(w, "  error: %s\n", r.Error)
		}
	}
	w.Flush()
	printInfo("Matched: %d of %d table(s)", len(rows)-d1ParityMismatchCount(rows), len(rows))
}

// d1ParityLocalCount runs SELECT COUNT(*) against the local SQLite file
// using the same CGO-free driver the --local query path uses.
func d1ParityLocalCount(path, table string) (int64, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	var count int64
	// %q yields a double-quoted SQL identifier; the table flag is never
	// interpolated raw.
	if err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %q", table)).Scan(&count); err != nil {
		return 0, fmt.Errorf("table %q: %w", table, err)
	}
	return count, nil
}

// d1ParityRemoteCount runs SELECT COUNT(*) through the remote D1 query path.
func d1ParityRemoteCount(ctx context.Context, svc *cosmoflare.D1Service, databaseID, table string) (int64, error) {
	results, err := svc.Query(ctx, databaseID, fmt.Sprintf("SELECT COUNT(*) AS cnt FROM %q", table))
	if err != nil {
		return 0, fmt.Errorf("table %q: %w", table, err)
	}
	if len(results) == 0 || len(results[0].Rows) == 0 {
		return 0, fmt.Errorf("table %q: remote query returned no rows", table)
	}
	return d1ParityToInt(results[0].Rows[0]["cnt"])
}

// d1ParityToInt coerces a D1 cell value (json-decoded) into an int64.
func d1ParityToInt(v any) (int64, error) {
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case float64:
		return int64(n), nil
	case string:
		var out int64
		if _, err := fmt.Sscanf(n, "%d", &out); err != nil {
			return 0, fmt.Errorf("count %q is not numeric", n)
		}
		return out, nil
	default:
		return 0, fmt.Errorf("unexpected count value %#v", v)
	}
}
