package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// cachingFixture builds a ProjectCacheInventory hitting every row of the
// W3-C finding-mapping table exactly once.
func cachingFixture() *cosmoflare.ProjectCacheInventory {
	return &cosmoflare.ProjectCacheInventory{
		Root: "/proj",
		Routes: []cosmoflare.CachedRoute{
			// P1a: path-only key (no keyFn) but handler reads query.
			{File: "routes/legacy.ts", Line: 10, Route: "GET /list",
				KeyFn: false, QueryAware: false, HandlerQuery: true, TTL: 300,
				KeyPrefix: "api:list"},
			// P1b: keyFn present but the key ignores the query.
			{File: "routes/legacy.ts", Line: 20, Route: "GET /item",
				KeyFn: true, QueryAware: false, HandlerQuery: true, TTL: 600,
				KeyPrefix: "api:item"},
			// P2: cached route behind auth.
			{File: "routes/user.ts", Line: 30, Route: "GET /me",
				HasAuth: true, TTL: 300, KeyPrefix: "api:me"},
			// Clean route — must produce no finding.
			{File: "routes/meta.ts", Line: 40, Route: "GET /stats",
				KeyFn: true, QueryAware: true, TTL: 21600,
				KeyPrefix: "api:meta:stats"},
		},
		Aggregates: []cosmoflare.AggregateQuery{
			// P3: whole-table aggregate in an uncached handler.
			{File: "routes/stats.ts", Line: 50, Route: "GET /totals",
				SQL: "SELECT COUNT(*) FROM churches GROUP BY country"},
		},
		Middleware: []cosmoflare.MiddlewareBehavior{
			// P4 + P5: middleware emits neither Cache-Control nor a 200 guard.
			{File: "middleware/cache.ts", EmitsCacheControl: false, OnlyStatus200: false},
		},
		TotalGETRoutes: 5,
		FilesScanned:   4,
	}
}

// withCachingSeam stubs the scan seam, the text writer, and the --source
// flag for the duration of one test.
func withCachingSeam(t *testing.T, inv *cosmoflare.ProjectCacheInventory, err error) *bytes.Buffer {
	t.Helper()
	prevScan := efficiencyCachingScanFn
	efficiencyCachingScanFn = func(root string) (*cosmoflare.ProjectCacheInventory, error) {
		if err != nil {
			return nil, err
		}
		return inv, nil
	}
	buf := &bytes.Buffer{}
	prevText := efficiencyTextOut
	efficiencyTextOut = buf
	t.Cleanup(func() {
		efficiencyCachingScanFn = prevScan
		efficiencyTextOut = prevText
	})
	return buf
}

// runCaching executes the subcommand with a --source value.
func runCaching(t *testing.T) error {
	t.Helper()
	prev := efficiencyCachingSource
	efficiencyCachingSource = "."
	t.Cleanup(func() { efficiencyCachingSource = prev })
	return efficiencyCachingCmd.RunE(efficiencyCachingCmd, nil)
}

func TestEfficiencyCachingCmdRegistered(t *testing.T) {
	var found *cobra.Command
	for _, c := range efficiencyCmd.Commands() {
		if c.Name() == "caching" {
			found = c
			break
		}
	}
	if found == nil {
		t.Fatal("efficiency caching subcommand is not registered")
	}
}

func TestEfficiencyCachingFindings(t *testing.T) {
	buf := withCachingSeam(t, cachingFixture(), nil)
	if err := runCaching(t); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	out := buf.String()

	type expect struct {
		check    string
		severity string
		evidence string
	}
	expects := []expect{
		{"p1-path-only-cache-key", "warning", "routes/legacy.ts:10"},
		{"p1-key-misses-query", "warning", "routes/legacy.ts:20"},
		{"p2-auth-cache-leak", "critical", "routes/user.ts:30"},
		{"p3-unmaterialized-aggregate", "warning", "routes/stats.ts:50"},
		{"p4-no-cache-control", "warning", "middleware/cache.ts"},
		{"p5-caches-non-200", "warning", "middleware/cache.ts"},
	}
	for _, e := range expects {
		if !strings.Contains(out, e.check) {
			t.Errorf("output missing check %q\ngot:\n%s", e.check, out)
			continue
		}
		// Severity words render as glyphs in the text table; the word form
		// is asserted in TestEfficiencyCachingJSONEnvelope. Here the ‼ glyph
		// (critical) is checked below.
		if !strings.Contains(out, e.evidence) {
			t.Errorf("output for %q missing evidence %q", e.check, e.evidence)
		}
	}

	// Critical renders as the ‼ glyph in the text table.
	if !strings.Contains(out, "‼") {
		t.Error("critical severity did not render the ‼ glyph")
	}
	// The clean route (routes/meta.ts) must produce no finding line.
	if strings.Count(out, "routes/meta.ts") > 1 {
		t.Errorf("clean route leaked into findings:\n%s", out)
	}
}

func TestEfficiencyCachingJSONEnvelope(t *testing.T) {
	inv := cachingFixture()
	inv.Middleware[0].EmitsCacheControl = true
	inv.Middleware[0].OnlyStatus200 = true

	withCachingSeam(t, inv, nil)
	prevJSONFlag, prevJSONOut := efficiencyCachingJSON, efficiencyJSONOut
	efficiencyCachingJSON = true
	jsonBuf := &bytes.Buffer{}
	efficiencyJSONOut = jsonBuf
	t.Cleanup(func() {
		efficiencyCachingJSON, efficiencyJSONOut = prevJSONFlag, prevJSONOut
	})

	if err := runCaching(t); err != nil {
		t.Fatalf("RunE: %v", err)
	}

	var env struct {
		Status string `json:"status"`
		Data   struct {
			Findings []cosmoflare.EfficiencyFinding `json:"findings"`
		} `json:"data"`
	}
	if err := json.Unmarshal(jsonBuf.Bytes(), &env); err != nil {
		t.Fatalf("bad JSON: %v\n%s", err, jsonBuf.String())
	}
	if env.Status != "success" {
		t.Errorf("status = %q, want success", env.Status)
	}
	var got []string
	for _, f := range env.Data.Findings {
		got = append(got, f.Check+"|"+f.Severity)
	}
	want := []string{
		"p1-path-only-cache-key|warning",
		"p1-key-misses-query|warning",
		"p2-auth-cache-leak|critical",
		"p3-unmaterialized-aggregate|warning",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("findings = %v, want %v", got, want)
	}
}

func TestEfficiencyCachingMissingSource(t *testing.T) {
	buf := withCachingSeam(t, nil, &notFoundError{msg: "source directory /nope does not exist"})
	if err := runCaching(t); err == nil {
		t.Fatal("RunE returned nil error for a failed scan")
	} else if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error %q does not mention the missing source", err.Error())
	}
	if buf.String() != "" {
		t.Errorf("wrote output on error: %q", buf.String())
	}
}

// notFoundError is a tiny error stand-in for the scan failure case.
type notFoundError struct{ msg string }

func (e *notFoundError) Error() string { return e.msg }

func TestEfficiencyCachingSummary(t *testing.T) {
	buf := withCachingSeam(t, cachingFixture(), nil)
	if err := runCaching(t); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(buf.String(), "4 of 5 GET routes cached") {
		t.Errorf("summary line missing from output:\n%s", buf.String())
	}
}

// TestEfficiencyCachingJSONEmptyFindings pins the non-nil findings slice: a
// clean inventory must emit findings [] — not null — so jq iteration works
// on the happy path.
func TestEfficiencyCachingJSONEmptyFindings(t *testing.T) {
	inv := &cosmoflare.ProjectCacheInventory{
		Root:           "/proj",
		TotalGETRoutes: 3,
	}
	buf := withCachingSeam(t, inv, nil)
	prevJSONFlag, prevJSONOut := efficiencyCachingJSON, efficiencyJSONOut
	efficiencyCachingJSON = true
	jsonBuf := &bytes.Buffer{}
	efficiencyJSONOut = jsonBuf
	t.Cleanup(func() {
		efficiencyCachingJSON, efficiencyJSONOut = prevJSONFlag, prevJSONOut
	})
	if err := runCaching(t); err != nil {
		t.Fatalf("RunE: %v", err)
	}
	if !strings.Contains(jsonBuf.String(), `"findings":[]`) {
		t.Errorf("findings emitted as null, want []:\n%s", jsonBuf.String())
	}
	_ = buf
}
