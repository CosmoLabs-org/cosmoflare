package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// stubEfficiency swaps the collection seam for tests (the usageCollectFn
// pattern from cmd/usage.go).
func stubEfficiency(t *testing.T, rep *cosmoflare.EfficiencyReport) {
	t.Helper()
	prev := efficiencyCollectFn
	efficiencyCollectFn = func(windowDays int, now time.Time) (*cosmoflare.EfficiencyReport, error) {
		return rep, nil
	}
	t.Cleanup(func() { efficiencyCollectFn = prev })
}

func effSampleReport() *cosmoflare.EfficiencyReport {
	return &cosmoflare.EfficiencyReport{
		WindowStart:         time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC),
		WindowEnd:           time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC),
		WorkersChecked:      5,
		KVNamespacesChecked: 3,
		BucketsChecked:      4,
		ZonesChecked:        2,
		ZoneCoverage: []cosmoflare.ZoneCacheCoverageSummary{
			{ZoneID: "z-slow", Zone: "slow.example", Requests: 2000, CachedRequests: 5, HitRatePct: 0.25},
		},
		Findings: []cosmoflare.EfficiencyFinding{
			{Resource: "worker:ghost", Check: "idle-worker", Severity: "warning",
				Evidence: "0 requests between 2026-10-07 and 2026-10-14",
				Fix:      "delete it (cosmoflare worker delete ghost)"},
			{Resource: "kv:ns-old", Check: "dead-kv-namespace", Severity: "warning",
				Evidence: "0 reads, 0 writes between 2026-10-07 and 2026-10-14",
				Fix:      "export anything worth keeping, then delete it (cosmoflare kv namespace delete ns-old)"},
			{Resource: "r2:void", Check: "empty-r2-bucket", Severity: "info",
				Evidence: "0 objects at 2026-10-14", Fix: "cosmoflare bucket delete void"},
			{Resource: "zone:slow.example", Check: "low-cache-hit", Severity: "info",
				Evidence: "2000 requests, 0.2% edge-cache hit over 2026-10-07 → 2026-10-14",
				Fix:      "origin Cache-Control headers on cacheable responses, or a zone Cache Rule (HTML needs a Cache Rule — s-maxage alone does nothing)"},
		},
	}
}

func TestEfficiencyIdleWorkers(t *testing.T) {
	stubEfficiency(t, effSampleReport())
	var out bytes.Buffer
	efficiencyTextOut = &out
	t.Cleanup(func() { efficiencyTextOut = os.Stdout })
	if err := runEfficiency(nil, nil); err != nil {
		t.Fatalf("runEfficiency: %v", err)
	}
	if !strings.Contains(out.String(), "worker:ghost") || !strings.Contains(out.String(), "idle-worker") {
		t.Fatalf("table lacks idle worker finding:\n%s", out.String())
	}
}

func TestEfficiencyEmptyKV(t *testing.T) {
	stubEfficiency(t, effSampleReport())
	var out bytes.Buffer
	efficiencyTextOut = &out
	t.Cleanup(func() { efficiencyTextOut = os.Stdout })
	if err := runEfficiency(nil, nil); err != nil {
		t.Fatalf("runEfficiency: %v", err)
	}
	if !strings.Contains(out.String(), "kv:ns-old") || !strings.Contains(out.String(), "dead-kv-namespace") {
		t.Fatalf("table lacks dead KV finding:\n%s", out.String())
	}
}

func TestEfficiencyJSONOutput(t *testing.T) {
	stubEfficiency(t, effSampleReport())
	var out bytes.Buffer
	efficiencyJSONOut = &out
	prevJSON := efficiencyJSON
	efficiencyJSON = true
	t.Cleanup(func() { efficiencyJSONOut = os.Stdout; efficiencyJSON = prevJSON })
	if err := runEfficiency(nil, nil); err != nil {
		t.Fatalf("runEfficiency: %v", err)
	}
	var env struct {
		Status string                      `json:"status"`
		Data   cosmoflare.EfficiencyReport `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("envelope not JSON: %v\n%s", err, out.String())
	}
	if env.Status != "success" || len(env.Data.Findings) != 4 {
		t.Fatalf("envelope = %+v", env)
	}
}

func TestEfficiencyEmptyAccount(t *testing.T) {
	stubEfficiency(t, &cosmoflare.EfficiencyReport{
		WindowStart:         time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC),
		WindowEnd:           time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC),
		WorkersChecked:      2,
		KVNamespacesChecked: 1,
		BucketsChecked:      1,
	})
	var out bytes.Buffer
	efficiencyTextOut = &out
	t.Cleanup(func() { efficiencyTextOut = os.Stdout })
	if err := runEfficiency(nil, nil); err != nil {
		t.Fatalf("runEfficiency: %v", err)
	}
	if !strings.Contains(out.String(), "No waste found") {
		t.Fatalf("empty account must print a clean bill:\n%s", out.String())
	}
}

// TestEfficiencyZoneCoverageOutput: the wired zone section shows in both
// output surfaces — the table row carries the zone finding, the JSON
// envelope carries zones_checked + per-zone coverage rows.
func TestEfficiencyZoneCoverageOutput(t *testing.T) {
	stubEfficiency(t, effSampleReport())

	var out bytes.Buffer
	efficiencyTextOut = &out
	prevJSON := efficiencyJSON
	efficiencyJSON = false
	t.Cleanup(func() { efficiencyTextOut = os.Stdout; efficiencyJSON = prevJSON })
	if err := runEfficiency(nil, nil); err != nil {
		t.Fatalf("runEfficiency: %v", err)
	}
	if !strings.Contains(out.String(), "zone:slow.example") || !strings.Contains(out.String(), "low-cache-hit") {
		t.Fatalf("table lacks zone finding:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "2 zones") {
		t.Fatalf("table lacks zone count:\n%s", out.String())
	}

	var jout bytes.Buffer
	efficiencyJSONOut = &jout
	efficiencyJSON = true
	t.Cleanup(func() { efficiencyJSONOut = os.Stdout; efficiencyJSON = prevJSON })
	if err := runEfficiency(nil, nil); err != nil {
		t.Fatalf("runEfficiency (json): %v", err)
	}
	var env struct {
		Status string                      `json:"status"`
		Data   cosmoflare.EfficiencyReport `json:"data"`
	}
	if err := json.Unmarshal(jout.Bytes(), &env); err != nil {
		t.Fatalf("envelope not JSON: %v\n%s", err, jout.String())
	}
	if env.Status != "success" || env.Data.ZonesChecked != 2 || len(env.Data.ZoneCoverage) != 1 {
		t.Fatalf("envelope = checked %d, coverage %+v", env.Data.ZonesChecked, env.Data.ZoneCoverage)
	}
	if env.Data.ZoneCoverage[0].HitRatePct != 0.25 {
		t.Fatalf("coverage hit rate = %v, want 0.25", env.Data.ZoneCoverage[0].HitRatePct)
	}
}
