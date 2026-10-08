//go:build live

package cosmoflare

import (
	"context"
	"os"
	"testing"
	"time"
)

// liveEnv reads the repo's credential names, falling back to the CF_* names
// that `ccs credentials source` exports.
func liveEnv(primary, fallback string) string {
	if v := os.Getenv(primary); v != "" {
		return v
	}
	return os.Getenv(fallback)
}

// TestLiveAnalyticsQueries runs every analytics query cosmoflare's alerts
// depend on against the real Cloudflare GraphQL API, with the SAME windows
// production uses: 24h for the watch, the billing cycle for usage. Mocked
// tests cannot catch schema drift or window limits (BUG-055 shipped green).
// Opt-in only:
//
//	CLOUDFLARE_ACCOUNT_ID=… CLOUDFLARE_API_TOKEN=… go test -tags live ./pkg/cosmoflare/ -run '^TestLive' -v
func TestLiveAnalyticsQueries(t *testing.T) {
	acct := liveEnv("CLOUDFLARE_ACCOUNT_ID", "CF_ACCOUNT_ID")
	tok := liveEnv("CLOUDFLARE_API_TOKEN", "CF_API_TOKEN")
	if acct == "" || tok == "" {
		t.Skip("CLOUDFLARE_ACCOUNT_ID and CLOUDFLARE_API_TOKEN (or CF_*) are required for live tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	now := time.Now().UTC()
	w := AnalyticsWindow{Start: now.Add(-24 * time.Hour), End: now} // watch window
	cycleStart, _ := CycleWindow(0, now)
	cycle := AnalyticsWindow{Start: cycleStart, End: now} // usage window
	a := NewAnalyticsService(acct, tok)

	if _, err := a.Workers(ctx, w); err != nil {
		t.Errorf("Workers (24h): %v", err)
	}
	if _, err := a.D1RowsRead(ctx, w); err != nil {
		t.Errorf("D1RowsRead (24h): %v", err)
	}
	if _, err := a.D1RowsRead(ctx, cycle); err != nil {
		t.Errorf("D1RowsRead (billing cycle %s → now): %v", cycleStart.Format(time.RFC3339), err)
	}
	if _, err := CollectUsage(ctx, a, "paid", 0, now); err != nil {
		t.Errorf("CollectUsage: %v", err)
	}

	zs, err := NewZoneServiceFromCreds(acct, tok)
	if err != nil {
		t.Fatalf("zone service: %v", err)
	}
	zones, err := zs.List(ctx)
	if err != nil {
		t.Fatalf("zone list: %v", err)
	}
	if len(zones) == 0 {
		t.Skip("account has no zones; zone queries not exercised")
	}
	// One zone per distinct plan first (plan-specific field limits are the
	// risk), then fill to a full batch of 10.
	refs := make([]ZoneRef, 0, 10)
	seenPlan := map[string]bool{}
	picked := map[string]bool{}
	for _, z := range zones {
		if !seenPlan[z.Plan.Name] {
			seenPlan[z.Plan.Name] = true
			picked[z.ID] = true
			refs = append(refs, ZoneRef{ID: z.ID, Name: z.Name})
		}
	}
	for _, z := range zones {
		if len(refs) >= 10 {
			break
		}
		if !picked[z.ID] {
			refs = append(refs, ZoneRef{ID: z.ID, Name: z.Name})
		}
	}
	if len(refs) > 10 {
		refs = refs[:10]
	}
	t.Logf("live zones: %d across %d plans", len(refs), len(seenPlan))
	if _, err := a.ZoneHTTP(ctx, refs[0].ID, w); err != nil {
		t.Errorf("ZoneHTTP: %v", err)
	}
	got, err := a.ZoneCache(ctx, refs, w)
	if err != nil {
		t.Errorf("ZoneCache: %v", err)
	} else if len(got) != len(refs) {
		t.Errorf("ZoneCache returned %d zones, want %d", len(got), len(refs))
	}
}
