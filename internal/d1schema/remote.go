package d1schema

import (
	"context"
	"fmt"
	"sort"
	"strings"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// D1Queryer is the slice of the D1 query API the remote fetch needs. It is
// an interface so tests can stub the remote side without credentials;
// *cosmoflare.D1Service satisfies it via Query (FEAT-051).
type D1Queryer interface {
	Query(ctx context.Context, databaseID, sql string, params ...string) ([]*cosmoflare.D1QueryResult, error)
}

// Diff statuses. ADDED-LOCAL-ONLY marks drift where local migrations know a
// table or column the remote database is missing (the FB-14 production
// incident shape); REMOTE-ONLY is the reverse drift; COLUMN-DRIFT is a
// table present on both sides with a differing column set.
const (
	DiffMatch       = "match"
	DiffLocalOnly   = "ADDED-LOCAL-ONLY"
	DiffRemoteOnly  = "REMOTE-ONLY"
	DiffColumnDrift = "COLUMN-DRIFT"
)

// TableDiff is one table's parity verdict.
type TableDiff struct {
	Table      string   `json:"table"`
	Status     string   `json:"status"`
	LocalOnly  []string `json:"local_only,omitempty"`
	RemoteOnly []string `json:"remote_only,omitempty"`
}

// RemoteSchema fetches the remote database's table and column set through
// the D1 query API: sqlite_master for tables (internal sqlite_% tables are
// excluded), then PRAGMA table_info per table for columns.
func RemoteSchema(ctx context.Context, db D1Queryer, databaseID string) (map[string][]string, error) {
	tables, err := remoteTables(ctx, db, databaseID)
	if err != nil {
		return nil, err
	}

	remote := make(map[string][]string, len(tables))
	for _, table := range tables {
		columns, err := remoteColumns(ctx, db, databaseID, table)
		if err != nil {
			return nil, err
		}
		remote[table] = columns
	}
	return remote, nil
}

// remoteTables lists user tables from sqlite_master.
func remoteTables(ctx context.Context, db D1Queryer, databaseID string) ([]string, error) {
	results, err := db.Query(ctx, databaseID, `SELECT name FROM sqlite_master WHERE type = 'table'`)
	if err != nil {
		return nil, fmt.Errorf("d1schema: listing remote tables: %w", err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("d1schema: sqlite_master query returned no result set")
	}

	var tables []string
	for _, row := range results[0].Rows {
		name, ok := row["name"].(string)
		if !ok || strings.HasPrefix(name, "sqlite_") {
			continue
		}
		tables = append(tables, name)
	}
	sort.Strings(tables)
	return tables, nil
}

// remoteColumns lists one table's columns via PRAGMA table_info.
func remoteColumns(ctx context.Context, db D1Queryer, databaseID, table string) ([]string, error) {
	results, err := db.Query(ctx, databaseID, fmt.Sprintf("PRAGMA table_info(%s)", quoteIdent(table)))
	if err != nil {
		return nil, fmt.Errorf("d1schema: table_info for %q: %w", table, err)
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("d1schema: table_info(%q) returned no result set", table)
	}

	var columns []string
	for _, row := range results[0].Rows {
		if name, ok := row["name"].(string); ok {
			columns = append(columns, name)
		}
	}
	return columns, nil
}

// quoteIdent renders an SQL identifier in double quotes with inner quotes
// doubled; never interpolates the name raw.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// Diff compares the local migration schema against the remote column map
// and returns one TableDiff per table, sorted by table name. Columns are
// compared as sets (order is not semantic for parity).
func Diff(local Schema, remote map[string][]string) []TableDiff {
	localSet := make(map[string]map[string]bool, len(local.Tables))
	for table, columns := range local.Tables {
		localSet[table] = columnSet(columns)
	}
	remoteSet := make(map[string]map[string]bool, len(remote))
	for table, columns := range remote {
		remoteSet[table] = columnSet(columns)
	}

	names := make([]string, 0, len(localSet)+len(remoteSet))
	for table := range localSet {
		names = append(names, table)
	}
	for table := range remoteSet {
		if _, ok := localSet[table]; !ok {
			names = append(names, table)
		}
	}
	sort.Strings(names)

	diffs := make([]TableDiff, 0, len(names))
	for _, table := range names {
		l, r := localSet[table], remoteSet[table]
		diff := TableDiff{Table: table}
		switch {
		case l == nil:
			diff.Status = DiffRemoteOnly
		case r == nil:
			diff.Status = DiffLocalOnly
			diff.LocalOnly = sortedKeys(l)
		default:
			diff.LocalOnly = sortedKeys(setMinus(l, r))
			diff.RemoteOnly = sortedKeys(setMinus(r, l))
			if len(diff.LocalOnly) > 0 || len(diff.RemoteOnly) > 0 {
				diff.Status = DiffColumnDrift
			} else {
				diff.Status = DiffMatch
			}
		}
		diffs = append(diffs, diff)
	}
	return diffs
}

func columnSet(columns []string) map[string]bool {
	set := make(map[string]bool, len(columns))
	for _, c := range columns {
		set[c] = true
	}
	return set
}

func setMinus(a, b map[string]bool) map[string]bool {
	out := make(map[string]bool)
	for k := range a {
		if !b[k] {
			out[k] = true
		}
	}
	return out
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
