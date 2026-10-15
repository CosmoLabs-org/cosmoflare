package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// Flags for `efficiency ttl` (bound in init).
var (
	efficiencyTTLSource   string
	efficiencyTTLDatabase string
	efficiencyTTLWindow   int
	efficiencyTTLJSON     bool
)

// efficiencyTTLScanFn is the scan seam (efficiencyCachingScanFn pattern):
// production walks the project source tree; tests stub it with a fixture
// inventory.
var efficiencyTTLScanFn = cosmoflare.ScanProjectCache

// efficiencyTTLCadenceFn is the cadence seam: production sums real
// D1UsageService.DailyUsage rows-written over the window (never modeled);
// tests stub it. Returns total rows written, not the per-day average —
// the caller divides by the window length.
var efficiencyTTLCadenceFn = func(databaseID string, days int) (int64, error) {
	if AccountID == "" || APIToken == "" {
		return 0, fmt.Errorf("account ID and API token are required (run 'cosmoflare account' to configure)")
	}
	svc, err := cosmoflare.NewD1UsageServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return 0, fmt.Errorf("failed to create D1 usage service: %w", err)
	}
	usage, err := svc.DailyUsage(context.Background(), databaseID, days)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, day := range usage {
		total += day.RowsWritten
	}
	return total, nil
}

// efficiencyTTLListFn is the database auto-resolve seam: production lists
// the account's D1 databases; tests stub it.
var efficiencyTTLListFn = func() ([]*cosmoflare.D1Database, error) {
	if AccountID == "" || APIToken == "" {
		return nil, fmt.Errorf("account ID and API token are required (run 'cosmoflare account' to configure)")
	}
	d1, err := cosmoflare.NewD1ServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create D1 service: %w", err)
	}
	return d1.List(context.Background())
}

var efficiencyTTLCmd = &cobra.Command{
	Use:   "ttl",
	Short: "Grade every cached route's TTL against real D1 write cadence",
	Long: `Grade every cached route's TTL against your database's real write cadence.

Scans the project's Worker source for kvCache'd routes (same scanner as
'efficiency caching'), measures how often the backing D1 database is
written over the look-back window from real Cloudflare analytics, and
grades each route:

  stale-risk        nothing purges this prefix and the TTL outlives the
                    data's freshness — readers can see day-old data
  no-invalidation   safe today only because the TTL is short — add
                    invalidation to unlock longer TTLs
  raise             rarely-changing data with invalidation covered can
                    safely take a longer TTL
  appropriate       coverage and cadence are consistent with this TTL

Findings use checks ttl-stale-risk (warning), ttl-raise and
ttl-no-invalidation (info). Every fix quotes the KV/Cache-Control
asymmetry: KV entries are purgeable from code, the edge layer is not.

Writes/day is measured database-wide — D1 analytics has no table
dimension, so per-route cadence is an upper bound, never a guess.
Real numbers only: AccountID + APIToken are required.

Flags:
  --source    Project source directory to scan (default ".")
  --database  D1 database ID (omitted: auto-resolve when the account has
              exactly one database)
  --window    Look-back window in days (default 7, 1-90)
  --json      Machine-readable envelope

Examples:
  cosmoflare efficiency ttl --source ./workers/src --database <uuid>
  cosmoflare efficiency ttl --json | jq '.data.verdicts[] | select(.verdict=="stale-risk")'`,
	RunE: runEfficiencyTTL,
}

func init() {
	efficiencyCmd.AddCommand(efficiencyTTLCmd)
	efficiencyTTLCmd.Flags().StringVar(&efficiencyTTLSource, "source", ".", "Project source directory to scan")
	efficiencyTTLCmd.Flags().StringVar(&efficiencyTTLDatabase, "database", "", "D1 database ID (auto-resolved when the account has exactly one)")
	efficiencyTTLCmd.Flags().IntVar(&efficiencyTTLWindow, "window", 7, "Look-back window in days (1-90)")
	efficiencyTTLCmd.Flags().BoolVar(&efficiencyTTLJSON, "json", false, "Output JSON envelope")
}

func runEfficiencyTTL(cmd *cobra.Command, args []string) error {
	// Real-numbers rule: grading against measured cadence is the whole
	// point, so refuse before any scan when credentials are absent.
	if AccountID == "" || APIToken == "" {
		return fmt.Errorf("account ID and API token are required (run 'cosmoflare account' to configure)")
	}
	if efficiencyTTLWindow < 1 || efficiencyTTLWindow > 90 {
		return fmt.Errorf("--window must be 1-90 days, got %d", efficiencyTTLWindow)
	}
	databaseID := efficiencyTTLDatabase
	databaseLabel := databaseID
	if databaseID == "" {
		databases, err := efficiencyTTLListFn()
		if err != nil {
			return fmt.Errorf("resolving database (pass --database): %w", err)
		}
		switch len(databases) {
		case 0:
			return fmt.Errorf("no D1 databases found on this account — pass --database once one exists")
		case 1:
			databaseID = databases[0].UUID
			databaseLabel = fmt.Sprintf("%s (%s)", databases[0].Name, databases[0].UUID)
		default:
			candidates := ""
			for i, db := range databases {
				if i > 0 {
					candidates += ", "
				}
				candidates += fmt.Sprintf("%s (%s)", db.Name, db.UUID)
			}
			return fmt.Errorf("account has %d D1 databases — pass --database explicitly; candidates: %s",
				len(databases), candidates)
		}
	}
	inv, err := efficiencyTTLScanFn(efficiencyTTLSource)
	if err != nil {
		return err
	}
	totalWrites, err := efficiencyTTLCadenceFn(databaseID, efficiencyTTLWindow)
	if err != nil {
		return err
	}
	writesPerDay := totalWrites / int64(efficiencyTTLWindow)
	verdicts := cosmoflare.GradeTTLs(inv, writesPerDay)
	findings := efficiencyTTLFindings(verdicts)
	if efficiencyTTLJSON {
		enc := json.NewEncoder(efficiencyJSONOut)
		return enc.Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"verdicts":       verdicts,
				"findings":       findings,
				"database":       databaseID,
				"writes_per_day": writesPerDay,
				"window_days":    efficiencyTTLWindow,
			},
		})
	}
	return writeEfficiencyTTLTable(inv, verdicts, findings, databaseLabel, writesPerDay)
}

// efficiencyTTLFindings maps verdicts to EfficiencyFindings: stale-risk is
// a warning; raise and no-invalidation are informational. Each fix quotes
// the verdict's upgrade path and the KV/Cache-Control asymmetry note.
func efficiencyTTLFindings(verdicts []cosmoflare.TTLVerdict) []cosmoflare.EfficiencyFinding {
	var findings []cosmoflare.EfficiencyFinding
	for _, v := range verdicts {
		switch v.Verdict {
		case "stale-risk":
			findings = append(findings, cosmoflare.EfficiencyFinding{
				Resource: "route:" + v.Route,
				Check:    "ttl-stale-risk",
				Severity: "warning",
				Evidence: fmt.Sprintf("key %q, TTL %ds, %d writes/day, no invalidation covers this prefix",
					v.KeyPrefix, v.TTLSeconds, v.WritesPerDay),
				Fix: "add invalidation (cache.delete on this prefix) in every write path — that is the upgrade path to serving longer TTLs safely. " + cosmoflare.TTLAsymmetryNote,
			})
		case "raise":
			findings = append(findings, cosmoflare.EfficiencyFinding{
				Resource: "route:" + v.Route,
				Check:    "ttl-raise",
				Severity: "info",
				Evidence: fmt.Sprintf("key %q, TTL %ds, %d writes/day (rare), invalidation covered",
					v.KeyPrefix, v.TTLSeconds, v.WritesPerDay),
				Fix: "raise the TTL to 6-24h — rare-change data with invalidation covered can safely take it. " + cosmoflare.TTLAsymmetryNote,
			})
		case "no-invalidation":
			findings = append(findings, cosmoflare.EfficiencyFinding{
				Resource: "route:" + v.Route,
				Check:    "ttl-no-invalidation",
				Severity: "info",
				Evidence: fmt.Sprintf("key %q, TTL %ds, %d writes/day, nothing purges this prefix",
					v.KeyPrefix, v.TTLSeconds, v.WritesPerDay),
				Fix: "this is safe only while the TTL stays short — adding invalidation is the upgrade path to longer TTLs. " + cosmoflare.TTLAsymmetryNote,
			})
		}
	}
	return findings
}

// writeEfficiencyTTLTable renders one line per cached route: route, TTL,
// cadence, coverage, verdict, reason.
func writeEfficiencyTTLTable(inv *cosmoflare.ProjectCacheInventory, verdicts []cosmoflare.TTLVerdict, findings []cosmoflare.EfficiencyFinding, databaseID string, writesPerDay int64) error {
	w := efficiencyTextOut
	fmt.Fprintf(w, "TTL audit — database %s — %d writes/day over %d day(s) — %s\n\n",
		databaseID, writesPerDay, efficiencyTTLWindow, inv.Root)
	fmt.Fprintf(w, "%-28s %-7s %-8s %-10s %-16s  %s\n", "ROUTE", "TTL(S)", "CADENCE", "COVERAGE", "VERDICT", "REASON")
	for _, v := range verdicts {
		coverage := "unknown"
		if v.CoverageKnown {
			coverage = "no"
			if v.Covered {
				coverage = "yes"
			}
		}
		fmt.Fprintf(w, "%-28s %-7d %-8s %-10s %-16s  %s\n",
			v.Route, v.TTLSeconds, v.Cadence, coverage, v.Verdict, v.Reason)
	}
	fmt.Fprintf(w, "\n%d finding(s):\n", len(findings))
	for _, f := range findings {
		sev := "⚠"
		switch f.Severity {
		case "critical":
			sev = "‼"
		case "info":
			sev = "ℹ"
		}
		fmt.Fprintf(w, "%-40s %-22s %-4s  %s\n      fix: %s\n", f.Resource, f.Check, sev, f.Evidence, f.Fix)
	}
	return nil
}
