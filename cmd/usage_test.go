package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// stubUsageCollect swaps the collection seam for tests (the
// collectWatchMetricsFn pattern from alerts watch).
func stubUsageCollect(t *testing.T, snap *cosmoflare.UsageSnapshot) {
	t.Helper()
	prev := usageCollectFn
	usageCollectFn = func(ctx context.Context, plan string, anchor int, now time.Time) (*cosmoflare.UsageSnapshot, error) {
		return snap, nil
	}
	t.Cleanup(func() { usageCollectFn = prev })
}

func sampleUsageSnapshot() *cosmoflare.UsageSnapshot {
	return &cosmoflare.UsageSnapshot{
		CycleStart:  time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		CycleEnd:    time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		DaysElapsed: 15,
		DaysTotal:   31,
		Dimensions: []cosmoflare.UsageDimension{
			{ID: "workers.requests_monthly", Name: "Monthly request allowance", Unit: "requests_per_month", Used: 8e6, Limit: 1e7, Pct: 80, ProjectedPct: 165.33},
			{ID: "r2.storage_gb_monthly", Name: "Monthly R2 storage allowance", Unit: "gb_month_per_month", Used: 3, Limit: 0},
		},
	}
}

// TestUsageCommandJSON pins the --json envelope: success status wrapping
// the snapshot verbatim.
func TestUsageCommandJSON(t *testing.T) {
	stubUsageCollect(t, sampleUsageSnapshot())
	var out bytes.Buffer
	usageJSONOut = &out
	prevJSON := usageJSON
	usageJSON = true
	t.Cleanup(func() { usageJSON = prevJSON })

	if err := runUsage(nil, nil); err != nil {
		t.Fatalf("runUsage: %v", err)
	}
	var env struct {
		Status string                   `json:"status"`
		Data   cosmoflare.UsageSnapshot `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("envelope not JSON: %v\n%s", err, out.String())
	}
	if env.Status != "success" {
		t.Errorf("status = %q, want success", env.Status)
	}
	if len(env.Data.Dimensions) != 2 || env.Data.Dimensions[0].ID != "workers.requests_monthly" {
		t.Errorf("dimensions = %+v", env.Data.Dimensions)
	}
	if env.Data.DaysElapsed != 15 {
		t.Errorf("days elapsed = %v", env.Data.DaysElapsed)
	}
}

// TestUsageCommandTable pins the human table: dimension names, projected
// overage marking, and the unknown-limit rendering (never NaN).
func TestUsageCommandTable(t *testing.T) {
	stubUsageCollect(t, sampleUsageSnapshot())
	var out bytes.Buffer
	usageTextOut = &out

	if err := runUsage(nil, nil); err != nil {
		t.Fatalf("runUsage: %v", err)
	}
	s := out.String()
	for _, want := range []string{
		"Monthly request allowance",
		"80.0%",
		"165.3%",
		"projected-over",
		"Monthly R2 storage allowance",
		"limit unknown",
		"Oct 01",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("table missing %q:\n%s", want, s)
		}
	}
	if strings.Contains(s, "NaN") {
		t.Error("table must never render NaN")
	}
}
