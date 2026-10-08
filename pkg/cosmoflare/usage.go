// Usage pacing (FEAT-048): cycle-to-date usage per dimension against the
// verified monthly allowances in the limits catalog, with linear projection
// to cycle end. The billing anchor defaults to the calendar month (UTC);
// anniversary billing passes an explicit anchor day.

package cosmoflare

import (
	"context"
	"sort"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/limitsdata"
)

// UsageDimension is one monthly allowance dimension with its pacing state.
// Limit 0 means the plan tier has no value (e.g. enterprise custom pricing):
// Pct and ProjectedPct stay 0 — never NaN — while Used still reports.
type UsageDimension struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Unit         string  `json:"unit"`
	Used         float64 `json:"used"`
	Limit        float64 `json:"limit"`
	Pct          float64 `json:"pct"`
	ProjectedPct float64 `json:"projected_pct"`
}

// UsageSnapshot is the monthly usage view: every catalog monthly row with
// cycle-to-date usage and linear pacing.
type UsageSnapshot struct {
	CycleStart  time.Time        `json:"cycle_start"`
	CycleEnd    time.Time        `json:"cycle_end"`
	DaysElapsed float64          `json:"days_elapsed"`
	DaysTotal   float64          `json:"days_total"`
	Dimensions  []UsageDimension `json:"dimensions"`
}

// doBilledMemoryGB is the memory Cloudflare bills Durable Objects at
// regardless of actual usage ("memory billed at 128 MB") — the multiplier
// converting wall time to billable GB-seconds.
const doBilledMemoryGB = 0.125

// r2ClassA / r2ClassB classify R2 operation action names per the pricing
// page's verbatim lists (verified 2026-10-08). Free operations (Delete*,
// AbortMultipartUpload) count toward neither.
var r2ClassA = map[string]bool{
	"ListBuckets": true, "PutBucket": true, "ListObjects": true,
	"PutObject": true, "CopyObject": true, "CompleteMultipartUpload": true,
	"CreateMultipartUpload": true, "LifecycleStorageTierTransition": true,
	"ListMultipartUploads": true, "UploadPart": true, "UploadPartCopy": true,
	"ListParts": true, "PutBucketEncryption": true, "PutBucketCors": true,
	"PutBucketLifecycleConfiguration": true,
}

var r2ClassB = map[string]bool{
	"HeadBucket": true, "HeadObject": true, "GetObject": true,
	"UsageSummary": true, "GetBucketEncryption": true,
	"GetBucketLocation": true, "GetBucketCors": true,
	"GetBucketLifecycleConfiguration": true,
}

// CycleWindow anchors the billing cycle containing now. anchor 0 or 1 means
// calendar month (UTC). An anniversary anchor N starts on day N of the
// current month when now is on/after it, else day N of the previous month;
// N beyond a month's length clamps to its last day.
func CycleWindow(anchor int, now time.Time) (start, end time.Time) {
	if anchor <= 1 {
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return first, first.AddDate(0, 1, 0)
	}
	prev := now.AddDate(0, 0, -now.Day()) // last day of previous month
	clamp := func(y int, m time.Month) time.Time {
		last := time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
		day := anchor
		if day > last {
			day = last
		}
		return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
	}
	if now.Day() >= anchor {
		s := clamp(now.Year(), now.Month())
		return s, clamp(now.Year(), now.Month()).AddDate(0, 1, 0)
	}
	s := clamp(prev.Year(), prev.Month())
	return s, clamp(now.Year(), now.Month())
}

// CollectUsage assembles the monthly usage snapshot. Cumulative dimensions
// read analytics over [CycleStart, now); R2 storage is a gauge read over a
// trailing day. plan routes tier resolution ("free"|"paid"|"enterprise";
// anything else resolves as paid, matching limitsdata's default).
func CollectUsage(ctx context.Context, analytics *AnalyticsService, plan string, anchor int, now time.Time) (*UsageSnapshot, error) {
	if analytics == nil {
		return nil, validationError("CollectUsage", "analytics service is required")
	}
	start, end := CycleWindow(anchor, now)

	cycleWin := AnalyticsWindow{Start: start, End: now}
	scripts, err := analytics.Workers(ctx, cycleWin)
	if err != nil {
		return nil, err
	}
	dos, err := analytics.DurableObjects(ctx, cycleWin)
	if err != nil {
		return nil, err
	}
	gaugeWin := AnalyticsWindow{Start: now.Add(-24 * time.Hour), End: now}
	buckets, err := analytics.R2Storage(ctx, gaugeWin)
	if err != nil {
		return nil, err
	}
	ops, err := analytics.R2Operations(ctx, cycleWin)
	if err != nil {
		return nil, err
	}

	// D1 is additive (design D15): a D1 analytics failure must not drop the
	// Workers/DO/R2 pacing rows, so the dimension is simply omitted.
	var d1Rows uint64
	d1Summaries, d1Err := analytics.D1RowsRead(ctx, cycleWin)
	for _, r := range d1Summaries {
		d1Rows += r.RowsRead
	}

	var reqs, cpuMS uint64
	for _, s := range scripts {
		reqs += s.Requests
		cpuMS += uint64(s.CPUTimeMS)
	}
	var doReqs uint64
	var doWallMS float64
	for _, d := range dos {
		doReqs += d.Requests
		doWallMS += d.WallTimeMS
	}
	var storageBytes uint64
	for _, b := range buckets {
		storageBytes += b.PayloadSize
	}
	var classA, classB uint64
	for _, o := range ops {
		switch {
		case r2ClassA[o.Action]:
			classA += o.Requests
		case r2ClassB[o.Action]:
			classB += o.Requests
		}
	}

	doGBs := doWallMS / 1000.0 * doBilledMemoryGB // ms → s × 0.125 GB billed memory

	used := map[string]float64{
		"workers.requests_monthly": float64(reqs),
		"workers.cpu_ms_monthly":   float64(cpuMS),
		"do.requests_monthly":      float64(doReqs),
		"do.duration_gb_s_monthly": doGBs,
		"r2.storage_gb_monthly":    float64(storageBytes) / 1e9,
		"r2.class_a_monthly":       float64(classA),
		"r2.class_b_monthly":       float64(classB),
	}
	if d1Err == nil {
		used["d1.rows_read_monthly"] = float64(d1Rows)
	}

	ids := make([]string, 0, len(used))
	for id := range used {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	daysTotal := end.Sub(start).Hours() / 24
	daysElapsed := now.Sub(start).Hours() / 24

	snap := &UsageSnapshot{
		CycleStart:  start,
		CycleEnd:    end,
		DaysElapsed: daysElapsed,
		DaysTotal:   daysTotal,
		Dimensions:  make([]UsageDimension, 0, len(ids)),
	}
	for _, id := range ids {
		e, ok := limitsdata.Lookup(id)
		if !ok {
			continue
		}
		d := UsageDimension{ID: id, Name: e.Name, Unit: e.Unit, Used: used[id]}
		if limit, ok := limitFor(id, plan); ok {
			d.Limit = float64(limit)
		}
		if d.Limit > 0 {
			d.Pct = d.Used / d.Limit * 100
			if daysElapsed > 0 {
				d.ProjectedPct = d.Pct * daysTotal / daysElapsed
			}
		}
		snap.Dimensions = append(snap.Dimensions, d)
	}
	return snap, nil
}
