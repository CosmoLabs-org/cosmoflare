package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// ttlFixture builds a ProjectCacheInventory exercising the verdict rows a
// single database-wide cadence can produce: stale-risk (uncovered, long
// TTL, non-rare cadence), no-invalidation (uncovered, short TTL,
// non-rare), appropriate (covered, non-rare), and unknown coverage
// (function key). The raise verdict needs a rare cadence and is asserted
// separately in TestEfficiencyTTLRaiseVerdict.
func ttlFixture() *cosmoflare.ProjectCacheInventory {
	return &cosmoflare.ProjectCacheInventory{
		Root: "/proj",
		Routes: []cosmoflare.CachedRoute{
			// Uncovered (shares only the depth-1 "api" bucket with the
			// invalidation — never coverage per InvalidationCovered),
			// TTL 6h+, non-rare cadence → stale-risk (warning).
			{File: "routes/meta.ts", Line: 10, Route: "GET /meta/stats",
				KeyFn: true, QueryAware: true, TTL: 21600, KeyPrefix: "api:stats:overview"},
			// Uncovered, short TTL, non-rare cadence → no-invalidation (info).
			{File: "routes/page.ts", Line: 20, Route: "GET /page/home",
				TTL: 300, KeyPrefix: "page:/church/home"},
			// Covered at depth 2 (api:meta + cities), TTL exactly 6h →
			// appropriate under a non-rare cadence, raise under rare.
			{File: "routes/cities.ts", Line: 30, Route: "GET /meta/cities",
				KeyFn: true, QueryAware: true, TTL: 21600, KeyPrefix: "api:meta:cities"},
			// Function-derived key → coverage unknown, stays appropriate.
			{File: "routes/church.ts", Line: 40, Route: "GET /church/x",
				KeyFn: true, TTL: 600},
		},
		Invalidation: []cosmoflare.InvalidationCall{
			{File: "middleware/cache.ts", Line: 86, KeyPattern: "api:meta:cities:us"},
		},
		TotalGETRoutes: 4,
		FilesScanned:   4,
	}
}

// withTTLSeam stubs the scan, cadence, and list seams plus the text and
// JSON writers for the duration of one test. A nil inv leaves the scan
// seam untouched. cadence is the total rows written; daysUsed is the
// complete-day count the seam reports (today's partial day excluded).
func withTTLSeam(t *testing.T, inv *cosmoflare.ProjectCacheInventory, cadence int64, daysUsed int, cadenceErr error, databases []*cosmoflare.D1Database, listErr error) (*bytes.Buffer, *[]string) {
	t.Helper()
	if inv != nil {
		prevScan := efficiencyTTLScanFn
		efficiencyTTLScanFn = func(root string) (*cosmoflare.ProjectCacheInventory, error) {
			return inv, nil
		}
		t.Cleanup(func() { efficiencyTTLScanFn = prevScan })
	}
	cadenceCalls := &[]string{}
	prevCadence := efficiencyTTLCadenceFn
	efficiencyTTLCadenceFn = func(databaseID string, days int) (int64, int, error) {
		*cadenceCalls = append(*cadenceCalls, databaseID)
		return cadence, daysUsed, cadenceErr
	}
	t.Cleanup(func() { efficiencyTTLCadenceFn = prevCadence })
	prevList := efficiencyTTLListFn
	efficiencyTTLListFn = func() ([]*cosmoflare.D1Database, error) {
		if listErr != nil {
			return nil, listErr
		}
		return databases, nil
	}
	t.Cleanup(func() { efficiencyTTLListFn = prevList })
	buf := &bytes.Buffer{}
	prevText, prevJSON := efficiencyTextOut, efficiencyJSONOut
	efficiencyTextOut, efficiencyJSONOut = buf, buf
	t.Cleanup(func() {
		efficiencyTextOut, efficiencyJSONOut = prevText, prevJSON
	})
	return buf, cadenceCalls
}

// withTTLFlags pins the flag values for one test and restores them after.
func withTTLFlags(t *testing.T, source, database string, window int, jsonOut bool) {
	t.Helper()
	prevSource, prevDatabase, prevWindow, prevJSON :=
		efficiencyTTLSource, efficiencyTTLDatabase, efficiencyTTLWindow, efficiencyTTLJSON
	efficiencyTTLSource, efficiencyTTLDatabase, efficiencyTTLWindow, efficiencyTTLJSON =
		source, database, window, jsonOut
	t.Cleanup(func() {
		efficiencyTTLSource, efficiencyTTLDatabase, efficiencyTTLWindow, efficiencyTTLJSON =
			prevSource, prevDatabase, prevWindow, prevJSON
	})
}

// withTTLCreds pins non-empty credentials for one test and restores them.
func withTTLCreds(t *testing.T) {
	t.Helper()
	prevAccount, prevToken := AccountID, APIToken
	AccountID, APIToken = "acct-test", "token-test"
	t.Cleanup(func() { AccountID, APIToken = prevAccount, prevToken })
}

func execEfficiencyTTL(t *testing.T) error {
	t.Helper()
	return efficiencyTTLCmd.RunE(efficiencyTTLCmd, nil)
}

func TestEfficiencyTTLCmdRegistered(t *testing.T) {
	var found *cobra.Command
	for _, c := range efficiencyCmd.Commands() {
		if c.Name() == "ttl" {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("efficiency ttl subcommand is not registered")
	}
}

func TestEfficiencyTTLVerdictsJSON(t *testing.T) {
	withTTLCreds(t)
	withTTLFlags(t, "/proj", "db-1", 7, true)
	buf, cadenceCalls := withTTLSeam(t, ttlFixture(), 3500, 7, nil, nil, nil)
	if err := execEfficiencyTTL(t); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if len(*cadenceCalls) != 1 || (*cadenceCalls)[0] != "db-1" {
		t.Fatalf("cadence called with %v, want [db-1]", *cadenceCalls)
	}

	var env struct {
		Status string `json:"status"`
		Data   struct {
			Verdicts     []cosmoflare.TTLVerdict        `json:"verdicts"`
			Findings     []cosmoflare.EfficiencyFinding `json:"findings"`
			Database     string                         `json:"database"`
			WritesPerDay int64                          `json:"writes_per_day"`
			WindowDays   int                            `json:"window_days"`
		} `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("envelope JSON: %v\nraw: %s", err, buf.String())
	}
	if env.Status != "success" {
		t.Errorf("status = %q, want success", env.Status)
	}
	if env.Data.Database != "db-1" {
		t.Errorf("database = %q, want db-1", env.Data.Database)
	}
	// 3500 total writes over 7 days = 500/day ("daily" bucket).
	if env.Data.WritesPerDay != 500 {
		t.Errorf("writes_per_day = %d, want 500", env.Data.WritesPerDay)
	}
	if env.Data.WindowDays != 7 {
		t.Errorf("window_days = %d, want 7", env.Data.WindowDays)
	}
	if len(env.Data.Verdicts) != 4 {
		t.Fatalf("got %d verdicts, want 4", len(env.Data.Verdicts))
	}
	byRoute := map[string]cosmoflare.TTLVerdict{}
	for _, v := range env.Data.Verdicts {
		byRoute[v.Route] = v
	}
	stale := byRoute["GET /meta/stats"]
	if stale.Verdict != "stale-risk" || stale.Cadence != "daily" || stale.CoverageKnown != true || stale.Covered != false {
		t.Errorf("stats verdict = %+v, want stale-risk/daily/covered-known-false", stale)
	}
	if stale.TTLSeconds != 21600 || stale.WritesPerDay != 500 || stale.KeyPrefix != "api:stats:overview" {
		t.Errorf("stats fields = %+v, want ttl 21600, 500 writes/day, prefix api:stats:overview", stale)
	}
	if got := byRoute["GET /page/home"].Verdict; got != "no-invalidation" {
		t.Errorf("home verdict = %q, want no-invalidation", got)
	}
	// Covered at depth 2 by the api:meta:cities invalidation, but cadence is
	// daily, not rare → coverage consistent, verdict appropriate.
	cities := byRoute["GET /meta/cities"]
	if cities.Verdict != "appropriate" || cities.Covered != true {
		t.Errorf("cities verdict = %+v, want appropriate with covered=true", cities)
	}
	unknown := byRoute["GET /church/x"]
	if unknown.CoverageKnown != false || unknown.Verdict != "appropriate" {
		t.Errorf("church verdict = %+v, want appropriate with coverage unknown", unknown)
	}

	// Findings: stale-risk → warning; no-invalidation → info. The raise
	// verdict needs a rare cadence and cannot coexist with stale-risk at
	// one database-wide cadence, so no ttl-raise finding here.
	sevByCheck := map[string]string{}
	for _, f := range env.Data.Findings {
		sevByCheck[f.Check] = f.Severity
	}
	if sevByCheck["ttl-stale-risk"] != "warning" {
		t.Errorf("ttl-stale-risk severity = %q, want warning", sevByCheck["ttl-stale-risk"])
	}
	if sevByCheck["ttl-no-invalidation"] != "info" {
		t.Errorf("ttl-no-invalidation severity = %q, want info", sevByCheck["ttl-no-invalidation"])
	}
	if _, ok := sevByCheck["ttl-raise"]; ok {
		t.Errorf("ttl-raise finding present, want none at daily cadence")
	}
	// Every fix quotes the verdict's upgrade path (the asymmetry note).
	for _, f := range env.Data.Findings {
		if !strings.Contains(f.Fix, cosmoflare.TTLAsymmetryNote) {
			t.Errorf("%s fix missing asymmetry note: %s", f.Check, f.Fix)
		}
	}
}

// TestEfficiencyTTLRaiseVerdict pins the raise verdict: covered prefix,
// rare cadence, TTL under 6h → ttl-raise info finding with the upgrade
// path in its fix.
func TestEfficiencyTTLRaiseVerdict(t *testing.T) {
	withTTLCreds(t)
	withTTLFlags(t, "/proj", "db-1", 7, true)
	inv := ttlFixture()
	inv.Routes = []cosmoflare.CachedRoute{
		{File: "routes/cities.ts", Line: 30, Route: "GET /meta/cities",
			KeyFn: true, QueryAware: true, TTL: 3600, KeyPrefix: "api:meta:cities"},
	}
	buf, _ := withTTLSeam(t, inv, 0, 1, nil, nil, nil) // 0 writes/day → rare
	if err := execEfficiencyTTL(t); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var env struct {
		Data struct {
			Verdicts []cosmoflare.TTLVerdict        `json:"verdicts"`
			Findings []cosmoflare.EfficiencyFinding `json:"findings"`
		} `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("envelope JSON: %v", err)
	}
	if len(env.Data.Verdicts) != 1 || env.Data.Verdicts[0].Verdict != "raise" ||
		env.Data.Verdicts[0].Cadence != "rare" || !env.Data.Verdicts[0].Covered {
		t.Fatalf("verdicts = %+v, want single raise verdict at rare cadence, covered", env.Data.Verdicts)
	}
	if len(env.Data.Findings) != 1 || env.Data.Findings[0].Check != "ttl-raise" ||
		env.Data.Findings[0].Severity != "info" {
		t.Fatalf("findings = %+v, want single ttl-raise info finding", env.Data.Findings)
	}
	if !strings.Contains(env.Data.Findings[0].Fix, "6-24h") {
		t.Errorf("raise fix = %q, want 6-24h upgrade path", env.Data.Findings[0].Fix)
	}
}

func TestEfficiencyTTLWindowValidation(t *testing.T) {
	withTTLCreds(t)
	withTTLSeam(t, ttlFixture(), 0, 1, nil, nil, nil)
	for _, window := range []int{0, 91} {
		withTTLFlags(t, ".", "", window, false)
		err := execEfficiencyTTL(t)
		if err == nil {
			t.Errorf("window %d accepted, want rejection", window)
			continue
		}
		if !strings.Contains(err.Error(), "--window must be 1-90") {
			t.Errorf("window %d error = %v, want 1-90 message", window, err)
		}
	}
}

func TestEfficiencyTTLDatabaseAutoResolve(t *testing.T) {
	withTTLCreds(t)
	withTTLFlags(t, "/proj", "", 7, false)

	// Single database: auto-picked and passed to the cadence seam.
	buf, cadenceCalls := withTTLSeam(t, ttlFixture(), 70, 7, nil,
		[]*cosmoflare.D1Database{{UUID: "db-only", Name: "churches"}}, nil)
	if err := execEfficiencyTTL(t); err != nil {
		t.Fatalf("single-db RunE: %v", err)
	}
	if len(*cadenceCalls) != 1 || (*cadenceCalls)[0] != "db-only" {
		t.Fatalf("cadence called with %v, want [db-only]", *cadenceCalls)
	}
	if !strings.Contains(buf.String(), "db-only") || !strings.Contains(buf.String(), "churches") {
		t.Errorf("text output should name the resolved database, got:\n%s", buf.String())
	}

	// Two databases: refuse, naming both candidates.
	buf2, _ := withTTLSeam(t, ttlFixture(), 0, 1, nil,
		[]*cosmoflare.D1Database{
			{UUID: "db-a", Name: "churches"},
			{UUID: "db-b", Name: "analytics"},
		}, nil)
	withTTLFlags(t, "/proj", "", 7, false)
	err := execEfficiencyTTL(t)
	if err == nil {
		t.Fatal("two databases accepted without --database, want error")
	}
	for _, want := range []string{"churches", "db-a", "analytics", "db-b"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name candidate %q", err, want)
		}
	}
	if strings.Contains(buf2.String(), "GET /meta/stats") {
		t.Errorf("no output expected when resolution fails, got:\n%s", buf2.String())
	}
}

func TestEfficiencyTTLRequiresCreds(t *testing.T) {
	// Leave AccountID/APIToken empty: the real-numbers rule forbids modeled
	// cadence, so the command must refuse before any scan or API call.
	prevAccount, prevToken := AccountID, APIToken
	AccountID, APIToken = "", ""
	t.Cleanup(func() { AccountID, APIToken = prevAccount, prevToken })
	buf, cadenceCalls := withTTLSeam(t, ttlFixture(), 0, 1, nil, nil, nil)
	withTTLFlags(t, ".", "", 7, false)
	err := execEfficiencyTTL(t)
	if err == nil {
		t.Fatal("empty creds accepted, want error")
	}
	if !strings.Contains(err.Error(), "account ID and API token are required") {
		t.Errorf("error = %v, want creds message", err)
	}
	if len(*cadenceCalls) != 0 {
		t.Errorf("cadence seam called %v despite missing creds", *cadenceCalls)
	}
	if strings.Contains(buf.String(), "GET /") {
		t.Errorf("no verdict output expected without creds, got:\n%s", buf.String())
	}
}

// TestEfficiencyTTLCadenceError surfaces a cadence fetch failure verbatim.
func TestEfficiencyTTLCadenceError(t *testing.T) {
	withTTLCreds(t)
	withTTLFlags(t, "/proj", "db-1", 7, false)
	withTTLSeam(t, ttlFixture(), 0, 1, errors.New("analytics down"), nil, nil)
	err := execEfficiencyTTL(t)
	if err == nil || !strings.Contains(err.Error(), "analytics down") {
		t.Errorf("err = %v, want cadence error surfaced", err)
	}
}

// TestEfficiencyTTLCompleteDaysDivisor pins the cadence seam contract: the
// seam reports the complete-day count (today's partial UTC day excluded), and
// runEfficiencyTTL divides the total by THAT count — not by the raw window.
func TestEfficiencyTTLCompleteDaysDivisor(t *testing.T) {
	withTTLCreds(t)
	withTTLFlags(t, "/proj", "db-1", 7, true)
	// 3000 rows over 6 complete days (7-day window minus today) = 500/day.
	// Dividing by the raw window would yield 428.
	buf, _ := withTTLSeam(t, ttlFixture(), 3000, 6, nil, nil, nil)
	if err := execEfficiencyTTL(t); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	var env struct {
		Data struct {
			WritesPerDay int64 `json:"writes_per_day"`
			WindowDays   int   `json:"window_days"`
		} `json:"data"`
	}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("envelope JSON: %v\nraw: %s", err, buf.String())
	}
	if env.Data.WindowDays != 7 {
		t.Errorf("window_days = %d, want 7", env.Data.WindowDays)
	}
	if env.Data.WritesPerDay != 500 {
		t.Errorf("writes_per_day = %d, want 500 (3000 over 6 complete days, not 428 over 7)", env.Data.WritesPerDay)
	}
}

// TestEfficiencyTTLJSONEmptyFindings pins the non-nil findings slice: with
// no verdicts, the JSON envelope must emit findings [] — not null — so
// jq iteration works on the happy path.
func TestEfficiencyTTLJSONEmptyFindings(t *testing.T) {
	withTTLCreds(t)
	withTTLFlags(t, "/proj", "db-1", 7, true)
	buf, _ := withTTLSeam(t, &cosmoflare.ProjectCacheInventory{Root: "/proj"}, 0, 1, nil, nil, nil)
	if err := execEfficiencyTTL(t); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	raw := buf.String()
	if !strings.Contains(raw, `"findings":[]`) {
		t.Errorf("findings emitted as null, want []:\n%s", raw)
	}
	if !strings.Contains(raw, `"verdicts":[]`) {
		t.Errorf("verdicts emitted as null, want []:\n%s", raw)
	}
}
