package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// Flags for `efficiency caching` (bound in init).
var (
	efficiencyCachingSource string
	efficiencyCachingJSON   bool
)

// efficiencyCachingScanFn is the scan seam (efficiencyCollectFn pattern):
// production walks the project source tree; tests stub it with a fixture
// inventory.
var efficiencyCachingScanFn = cosmoflare.ScanProjectCache

var efficiencyCachingCmd = &cobra.Command{
	Use:   "caching",
	Short: "Find caching pattern violations (P1-P5) in Worker source",
	Long: `Find caching pattern violations (P1-P5) in your Worker source.

Scans the project's TypeScript source — no Cloudflare API credentials
needed, this is a pure static analysis — and reports the caching
anti-patterns behind the P-numbered findings:

  p1-path-only-cache-key      path-only KV key collapses query variants
  p1-key-misses-query         keyFn omits query params the handler reads
  p2-auth-cache-leak          cached route behind auth — cross-user leak by construction
  p3-unmaterialized-aggregate whole-table aggregate per request instead of a cron snapshot
  p4-no-cache-control         cache middleware never emits Cache-Control
  p5-caches-non-200           cache middleware stores errors and 404s

Every finding cites its P-number, the exact file:line, and the fix.
Route paths are as written at the sub-router (mount prefixes from
index.ts app.route(...) are not resolved).

Flags:
  --source   Project source directory to scan (default ".")
  --json     Machine-readable envelope

Examples:
  cosmoflare efficiency caching --source ./workers/src
  cosmoflare efficiency caching --json | jq '.data.findings[] | select(.severity=="critical")'`,
	RunE: runEfficiencyCaching,
}

func init() {
	efficiencyCmd.AddCommand(efficiencyCachingCmd)
	efficiencyCachingCmd.Flags().StringVar(&efficiencyCachingSource, "source", ".", "Project source directory to scan")
	efficiencyCachingCmd.Flags().BoolVar(&efficiencyCachingJSON, "json", false, "Output JSON envelope")
}

func runEfficiencyCaching(cmd *cobra.Command, args []string) error {
	inv, err := efficiencyCachingScanFn(efficiencyCachingSource)
	if err != nil {
		return err
	}
	findings := efficiencyCachingFindings(inv)
	if efficiencyCachingJSON {
		enc := json.NewEncoder(efficiencyJSONOut)
		return enc.Encode(map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"source":           inv.Root,
				"findings":         findings,
				"cached_routes":    len(inv.Routes),
				"total_get_routes": inv.TotalGETRoutes,
			},
		})
	}
	return writeEfficiencyCachingTable(inv, findings)
}

// efficiencyCachingFindings maps the source inventory to P1-P5 findings.
// Resource convention: route:GET /path for route-level checks,
// middleware:<file> for middleware-level ones; Evidence carries file:line.
func efficiencyCachingFindings(inv *cosmoflare.ProjectCacheInventory) []cosmoflare.EfficiencyFinding {
	// Non-nil from the start: an empty slice marshals to [] so the JSON
	// envelope never emits null findings.
	findings := make([]cosmoflare.EfficiencyFinding, 0, len(inv.Routes)+len(inv.Aggregates))
	for _, r := range inv.Routes {
		ev := fmt.Sprintf("%s:%d", r.File, r.Line)
		switch {
		case r.HasAuth:
			findings = append(findings, cosmoflare.EfficiencyFinding{
				Resource: "route:" + r.Route,
				Check:    "p2-auth-cache-leak",
				Severity: "critical",
				Evidence: ev,
				Fix:      "cached route carries auth — cross-user leak by construction; remove the cache middleware from this route (P2)",
			})
		case !r.KeyFn && r.HandlerQuery:
			findings = append(findings, cosmoflare.EfficiencyFinding{
				Resource: "route:" + r.Route,
				Check:    "p1-path-only-cache-key",
				Severity: "warning",
				Evidence: ev,
				Fix:      "path-only KV key collapses query variants onto one entry — add a keyFn including every query param the handler reads (P1; cf. Churches-app meta.ts cities keyFn)",
			})
		case r.KeyFn && !r.QueryAware && r.HandlerQuery:
			findings = append(findings, cosmoflare.EfficiencyFinding{
				Resource: "route:" + r.Route,
				Check:    "p1-key-misses-query",
				Severity: "warning",
				Evidence: ev,
				Fix:      "handler reads query params the key does not include (P1)",
			})
		}
	}
	for _, agg := range inv.Aggregates {
		findings = append(findings, cosmoflare.EfficiencyFinding{
			Resource: "route:" + agg.Route,
			Check:    "p3-unmaterialized-aggregate",
			Severity: "warning",
			Evidence: fmt.Sprintf("%s:%d %s", agg.File, agg.Line, agg.SQL),
			Fix:      "whole-table aggregate per request — materialize by cron into KV and serve the snapshot (P3; 40-line reference: Churches-app jobs/stats-materialize.ts)",
		})
	}
	for _, mw := range inv.Middleware {
		if !mw.EmitsCacheControl {
			findings = append(findings, cosmoflare.EfficiencyFinding{
				Resource: "middleware:" + mw.File,
				Check:    "p4-no-cache-control",
				Severity: "warning",
				Evidence: fmt.Sprintf("%s: middleware body never emits Cache-Control", mw.File),
				Fix:      "cache HIT still costs an invocation + KV read — emit `Cache-Control: public, max-age=300` on cached responses (P4)",
			})
		}
		if !mw.OnlyStatus200 {
			findings = append(findings, cosmoflare.EfficiencyFinding{
				Resource: "middleware:" + mw.File,
				Check:    "p5-caches-non-200",
				Severity: "warning",
				Evidence: fmt.Sprintf("%s: middleware stores responses without a status===200 guard", mw.File),
				Fix:      "guard the put on status 200 — errors and 404s must bypass (P5)",
			})
		}
	}
	return findings
}

// writeEfficiencyCachingTable renders the text table plus the
// "X of Y GET routes cached" summary line.
func writeEfficiencyCachingTable(inv *cosmoflare.ProjectCacheInventory, findings []cosmoflare.EfficiencyFinding) error {
	w := efficiencyTextOut
	if len(findings) == 0 {
		fmt.Fprintf(w, "No caching pattern violations found — %d of %d GET routes cached in %s.\n",
			len(inv.Routes), inv.TotalGETRoutes, inv.Root)
		return nil
	}
	fmt.Fprintf(w, "Caching pattern audit — %s\n\n", inv.Root)
	fmt.Fprintf(w, "%-40s %-28s %-4s  %s\n", "RESOURCE", "CHECK", "SEV", "EVIDENCE")
	for _, f := range findings {
		sev := "⚠"
		switch f.Severity {
		case "critical":
			sev = "‼"
		case "info":
			sev = "ℹ"
		}
		fmt.Fprintf(w, "%-40s %-28s %-4s  %s\n      fix: %s\n", f.Resource, f.Check, sev, f.Evidence, f.Fix)
	}
	fmt.Fprintf(w, "\n%d of %d GET routes cached — %d finding(s).\n",
		len(inv.Routes), inv.TotalGETRoutes, len(findings))
	return nil
}
