package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// costRunGlobals snapshots and restores the package-level variables that the
// cost run functions read, so tests cannot leak state.
func costRunGlobals(t *testing.T) {
	t.Helper()
	runGlobalsSnapshot(t)
	oldPeriod, oldFormat := costPeriod, costFormat
	t.Cleanup(func() { costPeriod, costFormat = oldPeriod, oldFormat })
}

// costRunDefaults installs the pristine flag state each subtest needs.
func costRunDefaults() {
	costPeriod = "30d"
	costFormat = "table"
	AccountID = "test-account"
	APIToken = "test-token-1234567890"
	JSONOutput = false
	DryRun = false
}

// costRunCapture runs fn with os.Stdout swapped for a pipe and returns what
// fn printed.
func costRunCapture(t *testing.T, fn func()) string {
	t.Helper()
	r, restore := captureStdout(t)
	fn()
	restore()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout failed: %v", err)
	}
	return string(out)
}

// costRunFns returns every cost run function so table-driven tests can cover
// the shared validation and service-construction branches uniformly.
func costRunFns() map[string]func() error {
	return map[string]func() error{
		"cost":    func() error { return runCost(costCmd, nil) },
		"r2":      func() error { return runCostR2(costR2Cmd, nil) },
		"workers": func() error { return runCostWorkers(costWorkersCmd, nil) },
		"kv":      func() error { return runCostKV(costKVCmd, nil) },
		"detail":  func() error { return runCostDetail(costDetailCmd, nil) },
	}
}

// costSeamUsage overrides collectCostUsage with a deterministic
// MeasuredCostUsage fixture (hermetic — no test may hit the network) and
// restores the seam via t.Cleanup. The closure rejects days outside the
// 7d/30d/90d flag mapping, so a wiring regression fails loudly. Mutate the
// returned fixture (e.g. ProjectedTo30d) before running the command.
func costSeamUsage(t *testing.T) *cosmoflare.MeasuredCostUsage {
	t.Helper()
	m := &cosmoflare.MeasuredCostUsage{
		R2:             cosmoflare.R2Usage{StorageGB: 100, ClassAOps: 1_000_000, ClassBOps: 10_000_000},
		Workers:        cosmoflare.WorkersUsage{Requests: 1_100_000, CPUTimeMs: 123_000},
		KV:             cosmoflare.KVUsage{Reads: 2_000_000, Writes: 1_000_000},
		WindowStart:    time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC),
		WindowEnd:      time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC),
		ProjectedTo30d: true,
	}
	prev := collectCostUsage
	collectCostUsage = func(ctx context.Context, svc *cosmoflare.CostService, days int) (*cosmoflare.MeasuredCostUsage, error) {
		switch days {
		case 7, 30, 90:
		default:
			t.Errorf("collectCostUsage days = %d, want 7/30/90", days)
		}
		return m, nil
	}
	t.Cleanup(func() { collectCostUsage = prev })
	return m
}

// TestRunCostAll_InvalidPeriod verifies the period guard rejects bad values with
// the exact message before any service is built.
func TestRunCost_InvalidPeriod(t *testing.T) {
	costRunGlobals(t)
	costRunDefaults()
	costPeriod = "1y"

	err := runCost(costCmd, nil)
	if err == nil || !strings.Contains(err.Error(), `invalid period "1y": must be 7d, 30d, or 90d`) {
		t.Fatalf("expected invalid period error, got %v", err)
	}
}

// TestRunCost_InvalidFormat verifies the format guard rejects bad values with
// the exact message.
func TestRunCost_InvalidFormat(t *testing.T) {
	costRunGlobals(t)
	costRunDefaults()
	costFormat = "xml"

	err := runCost(costCmd, nil)
	if err == nil || !strings.Contains(err.Error(), `invalid format "xml": must be table, json, or csv`) {
		t.Fatalf("expected invalid format error, got %v", err)
	}
}

// TestRunCostAll_InvalidPeriod verifies every cost subcommand enforces the
// period guard before touching credentials.
func TestRunCostAll_InvalidPeriod(t *testing.T) {
	fns := costRunFns()
	for _, name := range []string{"cost", "r2", "workers", "kv", "detail"} {
		t.Run(name, func(t *testing.T) {
			costRunGlobals(t)
			costRunDefaults()
			costPeriod = "365d"

			err := fns[name]()
			if err == nil || !strings.Contains(err.Error(), `invalid period "365d"`) {
				t.Fatalf("expected invalid period error, got %v", err)
			}
		})
	}
}

// TestRunCostAll_MissingAccountID verifies the service-construction error path
// for every cost command when no account ID is set.
func TestRunCostAll_MissingAccountID(t *testing.T) {
	fns := costRunFns()
	for _, name := range []string{"cost", "r2", "workers", "kv", "detail"} {
		t.Run(name, func(t *testing.T) {
			costRunGlobals(t)
			costRunDefaults()
			AccountID = ""

			err := fns[name]()
			if err == nil || !strings.Contains(err.Error(), "failed to create cost service") {
				t.Fatalf("expected service creation error, got %v", err)
			}
			if !strings.Contains(err.Error(), "accountID is required") {
				t.Fatalf("expected wrapped accountID error, got %v", err)
			}
		})
	}
}

// TestRunCostAll_MissingAPIToken verifies the token guard surfaces through
// service construction.
func TestRunCostAll_MissingAPIToken(t *testing.T) {
	fns := costRunFns()
	for _, name := range []string{"cost", "r2", "workers", "kv", "detail"} {
		t.Run(name, func(t *testing.T) {
			costRunGlobals(t)
			costRunDefaults()
			APIToken = ""

			err := fns[name]()
			if err == nil || !strings.Contains(err.Error(), "apiToken is required") {
				t.Fatalf("expected apiToken error, got %v", err)
			}
		})
	}
}

// TestRunCostAll_OutputFormats verifies every command renders each output
// format offline: the collectCostUsage seam is stubbed with a fixture
// MeasuredCostUsage (no network) and the measured numbers must appear in
// every format (JSON shape unchanged — usage values are now measured).
func TestRunCostAll_OutputFormats(t *testing.T) {
	fns := costRunFns()
	costSeamUsage(t) // seam is package-global: one fixture for the whole table
	// Distinctive measured numbers per command, valid in every format:
	// table/csv print them raw; subcommand JSON is indented ("k": v) and
	// cost/detail JSON is compact ("k":v).
	wantByCmd := map[string][]string{
		"cost":    {"$21.10/mo", "21.10", `"storage_gb":100`, `"class_a_ops": 1000000`},
		"r2":      {"1000000", `"class_a_ops": 1000000`},
		"workers": {"1100000", `"requests": 1100000`},
		"kv":      {"2000000", `"reads": 2000000`},
		"detail":  {"$21.10/mo", "21.10", `"storage_gb":100`, `"class_a_ops": 1000000`},
	}
	for _, name := range []string{"cost", "r2", "workers", "kv", "detail"} {
		for _, format := range []string{"table", "json", "csv"} {
			t.Run(name+"-"+format, func(t *testing.T) {
				costRunGlobals(t)
				costRunDefaults()
				costFormat = format

				out := costRunCapture(t, func() {
					if err := fns[name](); err != nil {
						t.Fatalf("runCost(%s, %s) failed: %v", name, format, err)
					}
				})
				if strings.TrimSpace(out) == "" {
					t.Fatalf("runCost(%s, %s) produced no output", name, format)
				}
				var matched int
				for _, want := range wantByCmd[name] {
					if strings.Contains(out, want) {
						matched++
					}
				}
				if matched == 0 {
					t.Errorf("runCost(%s, %s) output carries none of the measured numbers %v:\n%s", name, format, wantByCmd[name], out)
				}
			})
		}
	}
}

// TestRunCost_JSONFlagOverridesFormat verifies the global --json flag routes
// to JSON even when the format flag says table, and that the JSON carries the
// measured usage from the seam fixture.
func TestRunCost_JSONFlagOverridesFormat(t *testing.T) {
	costRunGlobals(t)
	costRunDefaults()
	costSeamUsage(t)
	JSONOutput = true

	out := costRunCapture(t, func() {
		if err := runCost(costCmd, nil); err != nil {
			t.Fatalf("runCost with --json failed: %v", err)
		}
	})
	if !strings.Contains(out, "\"period\": \"30d\"") && !strings.Contains(out, "\"period\":\"30d\"") {
		t.Fatalf("expected period field in JSON output, got: %s", out)
	}
	if !strings.Contains(out, `"class_a_ops":1000000`) {
		t.Fatalf("expected measured class_a_ops in JSON output, got: %s", out)
	}
}

// costRunSampleEstimates builds estimates from concrete usage so both the
// math and the printers can be asserted against exact numbers.
func costRunSampleEstimates(t *testing.T) (*cosmoflare.R2CostEstimate, *cosmoflare.WorkersCostEstimate, *cosmoflare.KVCostEstimate) {
	t.Helper()
	svc := newTestCostService(t)
	r2 := svc.EstimateR2Cost(cosmoflare.R2Usage{StorageGB: 100, ClassAOps: 1_000_000, ClassBOps: 10_000_000})
	workers := svc.EstimateWorkersCost(cosmoflare.WorkersUsage{Requests: 1_100_000})
	kv := svc.EstimateKVCost(cosmoflare.KVUsage{Reads: 2_000_000, Writes: 1_000_000, StorageGB: 10})
	return r2, workers, kv
}

// TestRunCost_EstimationMath asserts the exact offline cost math behind the
// printed tables: R2 9.60 + Workers 0.50 + KV 11.00 = 21.10.
func TestRunCost_EstimationMath(t *testing.T) {
	r2, workers, kv := costRunSampleEstimates(t)

	if got := fmt2(r2.StorageCost); got != "1.50" {
		t.Errorf("R2 storage cost = %s, want 1.50 (100GB x $0.015)", got)
	}
	if got := fmt2(r2.ClassACost); got != "4.50" {
		t.Errorf("R2 class A cost = %s, want 4.50 (1M x $4.50)", got)
	}
	if got := fmt2(r2.ClassBCost); got != "3.60" {
		t.Errorf("R2 class B cost = %s, want 3.60 (10M x $0.36)", got)
	}
	if got := fmt2(r2.TotalCost); got != "9.60" {
		t.Errorf("R2 total = %s, want 9.60", got)
	}
	if workers.BillableRequests != 1_000_000 {
		t.Errorf("billable requests = %d, want 1000000", workers.BillableRequests)
	}
	if got := fmt2(workers.TotalCost); got != "0.50" {
		t.Errorf("workers total = %s, want 0.50", got)
	}
	if got := fmt2(kv.ReadsCost); got != "1.00" {
		t.Errorf("KV reads cost = %s, want 1.00", got)
	}
	if got := fmt2(kv.WritesCost); got != "5.00" {
		t.Errorf("KV writes cost = %s, want 5.00", got)
	}
	if got := fmt2(kv.StorageCost); got != "5.00" {
		t.Errorf("KV storage cost = %s, want 5.00", got)
	}
	if got := fmt2(kv.TotalCost); got != "11.00" {
		t.Errorf("KV total = %s, want 11.00", got)
	}
	total := fmt2(r2.TotalCost + workers.TotalCost + kv.TotalCost)
	if total != "21.10" {
		t.Errorf("aggregate total = %s, want 21.10", total)
	}
}

// fmt2 formats a float to two decimals for exact string assertions.
func fmt2(v float64) string { return fmt.Sprintf("%.2f", v) }

// TestRunCost_WorkersBelowFreeTier verifies the free-tier floor clamps
// negative billable requests to zero.
func TestRunCost_WorkersBelowFreeTier(t *testing.T) {
	svc := newTestCostService(t)
	est := svc.EstimateWorkersCost(cosmoflare.WorkersUsage{Requests: 42_000})

	if est.BillableRequests != 0 {
		t.Errorf("billable requests = %d, want 0 below free tier", est.BillableRequests)
	}
	if got := fmt2(est.TotalCost); got != "0.00" {
		t.Errorf("workers total = %s, want 0.00 below free tier", got)
	}
}

// TestRunCost_TablesRenderNumbers verifies each table printer emits the
// concrete computed totals into the tabwriter output.
func TestRunCost_TablesRenderNumbers(t *testing.T) {
	r2, workers, kv := costRunSampleEstimates(t)
	svc := newTestCostService(t)
	total := svc.EstimateTotal(
		cosmoflare.R2Usage{StorageGB: 100, ClassAOps: 1_000_000, ClassBOps: 10_000_000},
		cosmoflare.WorkersUsage{Requests: 1_100_000},
		cosmoflare.KVUsage{Reads: 2_000_000, Writes: 1_000_000, StorageGB: 10},
	)
	total.Period = "7d"

	cases := []struct {
		name string
		fn   func() string
		want []string
	}{
		{"r2", func() string { return costRunCapture(t, func() { printR2CostTable(r2) }) },
			[]string{"R2 Storage Cost Estimate", "$1.50", "$4.50", "$3.60", "$9.60"}},
		{"workers", func() string { return costRunCapture(t, func() { printWorkersCostTable(workers) }) },
			[]string{"Workers Cost Estimate", "1000000", "$0.50"}},
		{"kv", func() string { return costRunCapture(t, func() { printKVCostTable(kv) }) },
			[]string{"KV Cost Estimate", "$1.00", "$5.00", "$11.00"}},
		{"summary", func() string { return costRunCapture(t, func() { printCostSummary(total, nil, nil) }) },
			[]string{"period: 7d", "$21.10/mo", "Estimates are approximate"}},
		{"detail", func() string { return costRunCapture(t, func() { printDetailTable(total, nil) }) },
			[]string{"Itemized Cost Breakdown (period: 7d)", "$21.10/mo"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := tc.fn()
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("table %s missing %q in output:\n%s", tc.name, want, out)
				}
			}
		})
	}
}

// TestRunCost_CSVExactOutput asserts the exact CSV rows for each printer.
func TestRunCost_CSVExactOutput(t *testing.T) {
	r2, workers, kv := costRunSampleEstimates(t)
	svc := newTestCostService(t)
	total := svc.EstimateTotal(
		cosmoflare.R2Usage{StorageGB: 100, ClassAOps: 1_000_000, ClassBOps: 10_000_000},
		cosmoflare.WorkersUsage{Requests: 1_100_000},
		cosmoflare.KVUsage{Reads: 2_000_000, Writes: 1_000_000, StorageGB: 10},
	)

	wantR2 := "item,usage,rate,cost\n" +
		"Storage,100.00 GB,$0.015/GB,1.50\n" +
		"Class A Ops,1000000,$4.50/M,4.50\n" +
		"Class B Ops,10000000,$0.36/M,3.60\n" +
		"TOTAL,,,9.60\n"
	if got := costRunCapture(t, func() { _ = printR2CostCSV(r2) }); got != wantR2 {
		t.Errorf("R2 CSV mismatch:\ngot:\n%s\nwant:\n%s", got, wantR2)
	}

	wantWorkers := "item,usage,rate,cost\n" +
		"Requests,1100000,$0.50/M,0.50\n" +
		"Free Tier,100000,,0.00\n" +
		"Billable,1000000,,\n" +
		"TOTAL,,,0.50\n"
	if got := costRunCapture(t, func() { _ = printWorkersCostCSV(workers) }); got != wantWorkers {
		t.Errorf("workers CSV mismatch:\ngot:\n%s\nwant:\n%s", got, wantWorkers)
	}

	wantKV := "item,usage,rate,cost\n" +
		"Reads,2000000,$0.50/M,1.00\n" +
		"Writes,1000000,$5.00/M,5.00\n" +
		"Storage,10.00 GB,$0.50/GB,5.00\n" +
		"TOTAL,,,11.00\n"
	if got := costRunCapture(t, func() { _ = printKVCostCSV(kv) }); got != wantKV {
		t.Errorf("KV CSV mismatch:\ngot:\n%s\nwant:\n%s", got, wantKV)
	}

	wantTotal := "service,estimated_cost\n" +
		"R2 Storage,9.60\nWorkers,0.50\nKV,11.00\nTOTAL,21.10\n"
	if got := costRunCapture(t, func() { _ = printCostCSV(total, nil) }); got != wantTotal {
		t.Errorf("summary CSV mismatch:\ngot:\n%s\nwant:\n%s", got, wantTotal)
	}

	wantDetail := "TOTAL,,,,21.10\n"
	if got := costRunCapture(t, func() { _ = printDetailCSV(total) }); !strings.HasSuffix(got, wantDetail) {
		t.Errorf("detail CSV must end with %q, got:\n%s", wantDetail, got)
	}
}

// TestRunCostAll_CollectErrorIsActionable verifies the seam-error path: when
// measured-usage collection fails, every cost command returns the actionable
// error naming the analytics permission, wrapping the collector's cause.
func TestRunCostAll_CollectErrorIsActionable(t *testing.T) {
	fns := costRunFns()
	for _, name := range []string{"cost", "r2", "workers", "kv", "detail"} {
		t.Run(name, func(t *testing.T) {
			costRunGlobals(t)
			costRunDefaults()
			prev := collectCostUsage
			collectCostUsage = func(ctx context.Context, svc *cosmoflare.CostService, days int) (*cosmoflare.MeasuredCostUsage, error) {
				return nil, fmt.Errorf("analytics dataset exploded")
			}
			t.Cleanup(func() { collectCostUsage = prev })

			err := fns[name]()
			if err == nil {
				t.Fatal("want collect error, got nil")
			}
			for _, want := range []string{
				"failed to collect measured usage",
				"analytics dataset exploded",
				"Account Analytics:Read",
			} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q missing %q", err, want)
				}
			}
		})
	}
}

// TestRunCost_MeasuredWindowLine verifies the table honesty line
// printCostSummary/printDetailTable gain after the header: the measured
// window in RFC3339, with ", projected to 30d" only when the trailing window
// was scaled — and that JSON/CSV shapes stay free of it.
func TestRunCost_MeasuredWindowLine(t *testing.T) {
	const wantProjected = "Usage measured 2026-10-08T00:00:00Z to 2026-10-15T00:00:00Z, projected to 30d"
	const wantExact = "Usage measured 2026-10-08T00:00:00Z to 2026-10-15T00:00:00Z"

	t.Run("summary-7d-projected", func(t *testing.T) {
		costRunGlobals(t)
		costRunDefaults()
		costPeriod = "7d"
		costSeamUsage(t)

		out := costRunCapture(t, func() {
			if err := runCost(costCmd, nil); err != nil {
				t.Fatalf("runCost: %v", err)
			}
		})
		if !strings.Contains(out, wantProjected) {
			t.Errorf("summary missing projected measured line:\n%s", out)
		}
	})

	t.Run("summary-30d-exact", func(t *testing.T) {
		costRunGlobals(t)
		costRunDefaults()
		costPeriod = "30d"
		costSeamUsage(t).ProjectedTo30d = false

		out := costRunCapture(t, func() {
			if err := runCost(costCmd, nil); err != nil {
				t.Fatalf("runCost: %v", err)
			}
		})
		if !strings.Contains(out, wantExact) {
			t.Errorf("summary missing measured line:\n%s", out)
		}
		if strings.Contains(out, "projected to 30d") {
			t.Errorf("30d summary must not claim projection:\n%s", out)
		}
	})

	t.Run("detail-7d-projected", func(t *testing.T) {
		costRunGlobals(t)
		costRunDefaults()
		costPeriod = "7d"
		costSeamUsage(t)

		out := costRunCapture(t, func() {
			if err := runCostDetail(costDetailCmd, nil); err != nil {
				t.Fatalf("runCostDetail: %v", err)
			}
		})
		if !strings.Contains(out, wantProjected) {
			t.Errorf("detail missing projected measured line:\n%s", out)
		}
	})

	t.Run("json-and-csv-stay-clean", func(t *testing.T) {
		costRunGlobals(t)
		costRunDefaults()
		costSeamUsage(t)

		costFormat = "json"
		out := costRunCapture(t, func() {
			if err := runCost(costCmd, nil); err != nil {
				t.Fatalf("runCost json: %v", err)
			}
		})
		if strings.Contains(out, "Usage measured") {
			t.Errorf("JSON shape must not gain the measured line:\n%s", out)
		}

		costFormat = "csv"
		out = costRunCapture(t, func() {
			if err := runCost(costCmd, nil); err != nil {
				t.Fatalf("runCost csv: %v", err)
			}
		})
		if strings.Contains(out, "Usage measured") {
			t.Errorf("CSV shape must not gain the measured line:\n%s", out)
		}
	})
}

// TestRunCostDaysWindowIsWired verifies runCost passes the flag-derived day
// count to the collector: overriding the seam captures the days argument for
// each period value.
func TestRunCostDaysWindowIsWired(t *testing.T) {
	cases := map[string]int{"7d": 7, "30d": 30, "90d": 90}
	for period, wantDays := range cases {
		t.Run(period, func(t *testing.T) {
			costRunGlobals(t)
			costRunDefaults()
			costPeriod = period
			costSeamUsage(t)

			got := 0
			prev := collectCostUsage
			collectCostUsage = func(ctx context.Context, svc *cosmoflare.CostService, days int) (*cosmoflare.MeasuredCostUsage, error) {
				got = days
				return prev(ctx, svc, days)
			}
			t.Cleanup(func() { collectCostUsage = prev })

			if err := runCost(costCmd, nil); err != nil {
				t.Fatalf("runCost: %v", err)
			}
			if got != wantDays {
				t.Errorf("collector days = %d, want %d", got, wantDays)
			}
		})
	}
}
