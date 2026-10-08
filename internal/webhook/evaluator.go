// Package webhook — alert evaluation: AlertRules × live metrics → TriggerAlert.
//
// The evaluator is the missing link between stored alert rules
// (pkg/cosmoflare.AlertService) and the alert bridge (Manager.TriggerAlert):
// it judges each enabled rule against metrics collected from the analytics
// API and fires through the manager with a per-rule cooldown so a
// persistently-broken condition does not re-fire every cycle.

package webhook

import (
	"context"
	"fmt"
	"log"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// EvalMetrics is the typed input the evaluator judges rules against.
type EvalMetrics struct {
	WorkersRequests uint64  // sum over the window (workersInvocationsAdaptive)
	WorkersErrors   uint64  // sum over the window
	CPUP99AvgMS     float64 // average of per-script cpuTimeP99 (0 when no scripts)
	R2StorageBytes  uint64  // total payloadSize across buckets
	R2ObjectCount   uint64  // total objectCount across buckets

	Scripts []cosmoflare.WorkersSummary // per-script rows, kept for stuck-work (script-scoped) conditions; flat fields above stay account sums

	DurableObjects []cosmoflare.DurableObjectSummary // per-namespace DO rows for do-scoped conditions (FEAT-047)

	Usage []cosmoflare.UsageDimension // monthly pacing rows for usage-pct conditions (FEAT-048); nil = no snapshot this cycle

	WorkersScriptCount uint64  // live script count (LimitsService)
	R2BucketCount      uint64  // live bucket count (LimitsService)
	DNSRecordQuotaPct  float64 // max percent across per-zone dns.records rows (0 = no rows)
}

// Evaluator evaluates AlertRules against EvalMetrics and fires the manager's
// TriggerAlert, with a per-rule cooldown so a persistently-broken condition
// does not re-fire every cycle.
type Evaluator struct {
	rules     *cosmoflare.AlertService
	manager   *Manager
	cooldown  time.Duration
	clock     func() time.Time     // injectable for tests
	lastFired map[string]time.Time // alert ID (rule name, or rule/scope for fan-out) → last fire time
	fires     map[string]int       // alert ID → fire count (drives Alert.Count)
}

// NewEvaluator creates an evaluator over the given rule service and alert
// manager. cooldown <= 0 defaults to 15 minutes.
func NewEvaluator(rules *cosmoflare.AlertService, manager *Manager, cooldown time.Duration) *Evaluator {
	if cooldown <= 0 {
		cooldown = 15 * time.Minute
	}
	return &Evaluator{
		rules:     rules,
		manager:   manager,
		cooldown:  cooldown,
		clock:     time.Now,
		lastFired: make(map[string]time.Time),
		fires:     make(map[string]int),
	}
}

// SetClock overrides the evaluator's clock. Used by tests to exercise the
// cooldown deterministically; harmless in production.
func (e *Evaluator) SetClock(fn func() time.Time) {
	if fn != nil {
		e.clock = fn
	}
}

// conditionValue computes the observed value for a rule condition and its
// display unit. The unit comes from the registry descriptor (FEAT-015: the
// registry is authoritative), so it cannot drift from the declared one.
// ok=false means the condition can never fire for these metrics
// (e.g. error-rate with zero requests, latency with no CPU samples).
func conditionValue(condition string, m EvalMetrics) (value float64, unit string, ok bool) {
	desc, registered := cosmoflare.LookupAlertCondition(condition)
	if !registered {
		return 0, "", false
	}
	switch condition {
	case "error-rate":
		if m.WorkersRequests == 0 {
			return 0, desc.Unit, false // no traffic → no rate, never divide by zero
		}
		return 100.0 * float64(m.WorkersErrors) / float64(m.WorkersRequests), desc.Unit, true
	case "storage-limit":
		return float64(m.R2StorageBytes), desc.Unit, true
	case "latency":
		if m.CPUP99AvgMS == 0 {
			return 0, desc.Unit, false // no CPU samples → no latency signal
		}
		return m.CPUP99AvgMS, desc.Unit, true
	case "failure-count":
		return float64(m.WorkersErrors), desc.Unit, true
	case "workers-script-count":
		return float64(m.WorkersScriptCount), desc.Unit, true
	case "r2-bucket-count":
		return float64(m.R2BucketCount), desc.Unit, true
	case "dns-record-quota":
		if m.DNSRecordQuotaPct == 0 {
			return 0, desc.Unit, false // no quota rows → nothing to judge
		}
		return m.DNSRecordQuotaPct, desc.Unit, true
	case "usage-pct":
		if v := usageMax(m.Usage, false); v > 0 {
			return v, desc.Unit, true
		}
		return 0, desc.Unit, false // no known-limit dims → nothing to judge
	case "usage-projected-pct":
		if v := usageMax(m.Usage, true); v > 0 {
			return v, desc.Unit, true
		}
		return 0, desc.Unit, false
	default:
		return 0, "", false // registered but unimplemented — the coverage test catches this
	}
}

// usageMax returns the highest percent (or projected percent) across usage
// dimensions with a known limit — one hot dimension fires the rule, the
// dns-record-quota pattern. 0 means no judgeable dimension.
func usageMax(dims []cosmoflare.UsageDimension, projected bool) float64 {
	var max float64
	for _, d := range dims {
		if d.Limit <= 0 {
			continue
		}
		v := d.Pct
		if projected {
			v = d.ProjectedPct
		}
		if v > max {
			max = v
		}
	}
	return max
}

// CollectUsageMetrics augments m with the monthly usage snapshot (FEAT-048).
// Callers own cadence: the watch caches ~15 min so the 60s loop does not
// re-query cycle telemetry every cycle.
func CollectUsageMetrics(m *EvalMetrics, snap *cosmoflare.UsageSnapshot) {
	if m == nil || snap == nil {
		return
	}
	m.Usage = snap.Dimensions
}

// scopedValue is one observable value of a condition: account-scoped
// conditions yield exactly one (ScopeID empty); script/do-scoped conditions
// yield one per script or Durable Object row (FEAT-047).
type scopedValue struct {
	ScopeID string // "" for account scope; script/DO name otherwise
	Value   float64
	Unit    string
}

// conditionValues fans a condition out over the metrics: one value per
// scope instance. An empty slice means the condition cannot fire this
// cycle (unknown condition, or no signal in the metrics).
func conditionValues(condition string, m EvalMetrics) []scopedValue {
	desc, registered := cosmoflare.LookupAlertCondition(condition)
	if !registered {
		return nil
	}
	switch desc.Scope {
	case "script":
		out := make([]scopedValue, 0, len(m.Scripts))
		for _, s := range m.Scripts {
			var v float64
			switch condition {
			case "worker-cpu":
				v = s.CPUP99
			case "worker-errors":
				v = float64(s.Errors)
			case "worker-requests":
				v = float64(s.Requests)
			case "worker-subrequests":
				v = float64(s.Subrequests)
			default:
				return nil // registered-but-unimplemented: the coverage test guards
			}
			// idle scripts (no traffic, no CPU samples) carry no stuck-work signal
			if s.Requests == 0 && s.Errors == 0 && s.Subrequests == 0 && s.CPUP99 == 0 {
				continue
			}
			out = append(out, scopedValue{ScopeID: s.Script, Value: v, Unit: desc.Unit})
		}
		return out
	case "do":
		out := make([]scopedValue, 0, len(m.DurableObjects))
		for _, d := range m.DurableObjects {
			var v float64
			switch condition {
			case "do-cpu":
				v = d.WallTimeMS
			case "do-requests":
				v = float64(d.Requests)
			default:
				return nil // registered-but-unimplemented: the coverage test guards
			}
			if d.Requests == 0 && d.Errors == 0 && d.WallTimeMS == 0 {
				continue // idle namespace carries no stuck-work signal
			}
			out = append(out, scopedValue{ScopeID: d.Script + "/" + d.Namespace, Value: v, Unit: desc.Unit})
		}
		return out
	default: // account scope
		v, unit, ok := conditionValue(condition, m)
		if !ok {
			return nil
		}
		return []scopedValue{{Value: v, Unit: unit}}
	}
}

// metricData builds the TriggerAlert data map from the raw metrics.
func metricData(m EvalMetrics) map[string]interface{} {
	return map[string]interface{}{
		"workers_requests":        m.WorkersRequests,
		"workers_errors":          m.WorkersErrors,
		"r2_storage_bytes":        m.R2StorageBytes,
		"r2_object_count":         m.R2ObjectCount,
		"cpu_p99_ms":              m.CPUP99AvgMS,
		"workers_script_count":    m.WorkersScriptCount,
		"r2_bucket_count":         m.R2BucketCount,
		"dns_record_quota_pct":    m.DNSRecordQuotaPct,
		"usage_pct_max":           usageMax(m.Usage, false),
		"usage_projected_pct_max": usageMax(m.Usage, true),
	}
}

// Evaluate runs every enabled rule against m and returns one rule name per
// fire (after cooldown filtering; a fan-out rule repeats per offender). Each fire calls
// manager.TriggerAlert with a converted Alert and a message that includes the
// observed value and threshold. A TriggerAlert error is logged but does not
// stop other rules; the rule still counts as fired.
func (e *Evaluator) Evaluate(m EvalMetrics) []string {
	if e.rules == nil {
		return nil
	}
	rules, err := e.rules.List()
	if err != nil {
		log.Printf("[alerts] list rules: %v", err)
		return nil
	}

	now := e.clock()
	fired := make([]string, 0, len(rules))
	for _, rule := range rules {
		if rule == nil || !rule.Enabled {
			continue
		}
		for _, sv := range conditionValues(rule.Condition, m) {
			if sv.Value < rule.Threshold {
				continue
			}
			// Per-script/do fires key the alert identity on the scope
			// instance so re-fires update the same per-script alert; the
			// NAME stays the rule name — cmd/alerts_watch.go resolves
			// severity/service through byName[Alert.Name]. Cooldown and
			// fire counts key on the same identity: two offenders breaching
			// one rule are two incidents, and both page.
			alertID := rule.Name
			if sv.ScopeID != "" {
				alertID = rule.Name + "/" + sv.ScopeID
			}
			if last, seen := e.lastFired[alertID]; seen && now.Sub(last) < e.cooldown {
				continue
			}

			e.lastFired[alertID] = now
			prev := e.fires[alertID]
			e.fires[alertID] = prev + 1
			fired = append(fired, rule.Name)

			updatedAt := rule.UpdatedAt
			if updatedAt.IsZero() {
				updatedAt = now
			}
			alert := &Alert{
				ID:        alertID, // deterministic: re-fires update the same alert
				Name:      rule.Name,
				Type:      AlertTypeThreshold,
				Threshold: rule.Threshold,
				Metric:    rule.Condition,
				Enabled:   true,
				CreatedAt: updatedAt,
				UpdatedAt: updatedAt,
				Count:     prev, // TriggerAlert increments, landing on prev+1
			}
			message := fmt.Sprintf("%s %s: observed %g meets threshold %g (%s)",
				rule.Condition, rule.Name, sv.Value, rule.Threshold, sv.Unit)
			if sv.ScopeID != "" {
				message = fmt.Sprintf("%s %s [%s]: observed %g meets threshold %g (%s)",
					rule.Condition, sv.ScopeID, rule.Name, sv.Value, rule.Threshold, sv.Unit)
			}

			if e.manager == nil {
				continue
			}
			if err := e.manager.TriggerAlert(alert, sv.Value, message, metricData(m)); err != nil {
				log.Printf("[alerts] trigger %q: %v", alertID, err)
			}
		}
	}
	return fired
}

// CollectEvalMetrics builds EvalMetrics from the analytics service over the
// window. Workers errors/requests/CPUP99 come from analytics.Workers; R2
// totals come from analytics.R2Storage. Any error is returned so callers can
// skip the cycle instead of evaluating against fabricated zeros.
func CollectEvalMetrics(ctx context.Context, analytics *cosmoflare.AnalyticsService, w cosmoflare.AnalyticsWindow) (EvalMetrics, error) {
	scripts, err := analytics.Workers(ctx, w)
	if err != nil {
		return EvalMetrics{}, fmt.Errorf("workers analytics: %w", err)
	}
	buckets, err := analytics.R2Storage(ctx, w)
	if err != nil {
		return EvalMetrics{}, fmt.Errorf("r2 storage analytics: %w", err)
	}
	// DO telemetry is additive signal, not load-bearing: an account without
	// DOs (or a dataset hiccup) must never break the watch cycle — the
	// do-scoped conditions simply see no rows and skip.
	dos, doErr := analytics.DurableObjects(ctx, w)
	if doErr != nil {
		log.Printf("[alerts] durable objects analytics unavailable, do-* conditions skip this cycle: %v", doErr)
		dos = nil
	}

	var m EvalMetrics
	var cpuSum float64
	var cpuCount int
	m.Scripts = scripts    // per-script rows for stuck-work conditions (FEAT-047)
	m.DurableObjects = dos // per-namespace rows for do-scoped conditions
	for _, s := range scripts {
		m.WorkersRequests += s.Requests
		m.WorkersErrors += s.Errors
		if s.CPUP99 > 0 { // scripts without CPU samples do not drag the mean down
			cpuSum += s.CPUP99
			cpuCount++
		}
	}
	if cpuCount > 0 {
		m.CPUP99AvgMS = cpuSum / float64(cpuCount)
	}
	for _, b := range buckets {
		m.R2StorageBytes += b.PayloadSize
		m.R2ObjectCount += b.ObjectCount
	}
	return m, nil
}

// CollectLimitMetrics augments m with limit-proximity metrics from the
// LimitsService. Only a total failure (zero rows collected) is an error —
// partial snapshots leave the untouched fields at zero and the evaluator
// skips conditions it cannot judge. DNSRecordQuotaPct is the MAXIMUM percent
// across per-zone rows so one hot zone fires the rule.
func CollectLimitMetrics(ctx context.Context, limits LimitsSource, m *EvalMetrics) error {
	snap, err := limits.Snapshot(ctx, "")
	if err != nil {
		return fmt.Errorf("limits snapshot: %w", err)
	}
	for _, row := range snap.Rows {
		switch row.Resource {
		case "workers.scripts":
			m.WorkersScriptCount = row.Used
		case "r2.buckets":
			m.R2BucketCount = row.Used
		case "dns.records":
			if row.Percent > m.DNSRecordQuotaPct {
				m.DNSRecordQuotaPct = row.Percent
			}
		}
	}
	return nil
}
