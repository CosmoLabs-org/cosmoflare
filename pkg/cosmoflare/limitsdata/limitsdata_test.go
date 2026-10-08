package limitsdata

import (
	"testing"
	"time"
)

func TestLoadEntryCount(t *testing.T) {
	entries, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// 74 embedded corpus entries + 2 appended local constants.
	if len(entries) != 76 {
		t.Fatalf("Load returned %d entries, want 76 (74 catalog + 2 local)", len(entries))
	}
}

func TestLookupHitAndMiss(t *testing.T) {
	e, ok := Lookup("workers.subrequests")
	if !ok {
		t.Fatal("Lookup(workers.subrequests) miss, want hit")
	}
	if e.Tiers == nil || e.Tiers.Paid == nil || e.Tiers.Paid.String() != "10000" {
		t.Fatalf("workers.subrequests paid tier = %+v, want 10000", e.Tiers)
	}
	if e.SourceURL == "" || e.VerifiedOn == nil {
		t.Fatalf("provenance missing on catalog entry: %+v", e)
	}
	if _, ok := Lookup("nope.not_a_resource"); ok {
		t.Fatal("Lookup(unknown id) must miss")
	}
}

// TestLookupLocalOnlyEntries pins the two corpus docs-silent constants: they
// are returned like any other entry but carry nil verified_on so the stale
// check names them.
func TestLookupLocalOnlyEntries(t *testing.T) {
	scripts, ok := Lookup("workers.scripts_per_account")
	if !ok {
		t.Fatal("Lookup(workers.scripts_per_account) miss, want hit")
	}
	if scripts.VerifiedOn != nil {
		t.Fatalf("local entry verified_on = %v, want nil (unverified)", scripts.VerifiedOn)
	}
	if scripts.Tiers == nil || scripts.Tiers.Free == nil || scripts.Tiers.Free.String() != "100" ||
		scripts.Tiers.Paid == nil || scripts.Tiers.Paid.String() != "500" {
		t.Fatalf("local workers.scripts_per_account tiers = %+v, want free=100 paid=500", scripts.Tiers)
	}
	if scripts.Notes == "" {
		t.Fatal("local entry must carry a flagging note")
	}

	buckets, ok := Lookup("r2.buckets_per_account")
	if !ok {
		t.Fatal("Lookup(r2.buckets_per_account) miss, want hit")
	}
	if buckets.VerifiedOn != nil {
		t.Fatalf("local entry verified_on = %v, want nil (unverified)", buckets.VerifiedOn)
	}
	if buckets.Tiers == nil || buckets.Tiers.Free == nil || buckets.Tiers.Free.String() != "1000000" {
		t.Fatalf("local r2.buckets_per_account free tier = %+v, want 1000000", buckets.Tiers)
	}
}

// TestStale pins the freshness semantics (D3): ids strictly older than the
// cutoff, plus every nil-verified_on entry (the local constants), are stale.
func TestStale(t *testing.T) {
	entries, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Fresh cutoff (before every verification date, all 2026-09-10): only
	// the two unverified locals are stale.
	fresh := Stale(time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC))
	if len(fresh) != 2 || fresh[0] != "workers.scripts_per_account" || fresh[1] != "r2.buckets_per_account" {
		t.Fatalf("Stale(fresh cutoff) = %v, want exactly the 2 local constants", fresh)
	}

	// Cutoff after every verification date (last batch: FEAT-048 monthly rows
	// verified 2026-10-08): everything is stale.
	old := Stale(time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC))
	if len(old) != len(entries) {
		t.Fatalf("Stale(post-verification cutoff) = %d ids, want all %d", len(old), len(entries))
	}

	// Boundary: strictly-before semantics. At cutoff 10:30 the entries
	// verified at 10:00 (e.g. workers.requests_daily) are stale; the ones
	// verified at 11:40 (the last batch, e.g. api.token_quota) are not.
	mid := Stale(time.Date(2026, 9, 10, 10, 30, 0, 0, time.UTC))
	has := func(ids []string, id string) bool {
		for _, v := range ids {
			if v == id {
				return true
			}
		}
		return false
	}
	if !has(mid, "workers.requests_daily") {
		t.Fatalf("Stale(10:30 cutoff) missing workers.requests_daily (verified 10:00): %v", mid)
	}
	last, ok := Lookup("api.token_quota")
	if !ok || last.VerifiedOn == nil {
		t.Fatalf("api.token_quota entry not loadable: %+v", last)
	}
	if has(mid, "api.token_quota") {
		t.Fatalf("Stale(10:30 cutoff) must not include api.token_quota (verified %v)", *last.VerifiedOn)
	}
}

// TestMonthlyUsageRowsPresent pins the FEAT-048 pacing inputs: the monthly
// usage dimensions the pacing engine resolves limits for. Values verified
// against live Cloudflare docs on 2026-10-08 (workers + DO + R2 pricing
// pages); free monthly aggregates are null where docs state only daily or
// per-invocation free limits — never derived.
func TestMonthlyUsageRowsPresent(t *testing.T) {
	expected := map[string]float64{
		"workers.requests_monthly": 10_000_000,
		"workers.cpu_ms_monthly":   30_000_000,
		"do.requests_monthly":      1_000_000,
		"do.duration_gb_s_monthly": 400_000,
		"r2.storage_gb_monthly":    10,
		"r2.class_a_monthly":       1_000_000,
		"r2.class_b_monthly":       10_000_000,
		"d1.rows_read_monthly":     25_000_000_000,
	}
	for id, wantPaid := range expected {
		e, ok := Lookup(id)
		if !ok {
			t.Errorf("catalog row %s missing", id)
			continue
		}
		if e.Tiers.Paid == nil {
			t.Errorf("%s: paid tier null", id)
			continue
		}
		if got, _ := e.Tiers.Paid.Float64(); got != wantPaid {
			t.Errorf("%s: paid = %v, want %v", id, got, wantPaid)
		}
		if e.SourceURL == "" || e.VerifiedOn == nil {
			t.Errorf("%s: provenance incomplete (source_url/verified_on)", id)
		}
	}
}

func TestD1RowsReadMonthlyFreeTierNull(t *testing.T) {
	e, ok := Lookup("d1.rows_read_monthly")
	if !ok {
		t.Fatal("catalog row d1.rows_read_monthly missing")
	}
	if e.Tiers.Free != nil {
		t.Errorf("free tier = %v, want null (Free is a daily 5M allowance, not monthly)", *e.Tiers.Free)
	}
}
