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
	lastFired map[string]time.Time // rule name → last fire time
	fires     map[string]int       // rule name → fire count (drives Alert.Count)
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
// display unit. ok=false means the condition can never fire for these metrics
// (e.g. error-rate with zero requests, latency with no CPU samples).
func conditionValue(condition string, m EvalMetrics) (value float64, unit string, ok bool) {
	switch condition {
	case "error-rate":
		if m.WorkersRequests == 0 {
			return 0, "%", false // no traffic → no rate, never divide by zero
		}
		return 100.0 * float64(m.WorkersErrors) / float64(m.WorkersRequests), "%", true
	case "storage-limit":
		return float64(m.R2StorageBytes), "bytes", true
	case "latency":
		if m.CPUP99AvgMS == 0 {
			return 0, "ms", false // no CPU samples → no latency signal
		}
		return m.CPUP99AvgMS, "ms", true
	case "failure-count":
		return float64(m.WorkersErrors), "errors", true
	case "workers-script-count":
		return float64(m.WorkersScriptCount), "scripts", true
	case "r2-bucket-count":
		return float64(m.R2BucketCount), "buckets", true
	case "dns-record-quota":
		if m.DNSRecordQuotaPct == 0 {
			return 0, "%", false // no quota rows → nothing to judge
		}
		return m.DNSRecordQuotaPct, "%", true
	default:
		return 0, "", false
	}
}

// metricData builds the TriggerAlert data map from the raw metrics.
func metricData(m EvalMetrics) map[string]interface{} {
	return map[string]interface{}{
		"workers_requests":     m.WorkersRequests,
		"workers_errors":       m.WorkersErrors,
		"r2_storage_bytes":     m.R2StorageBytes,
		"r2_object_count":      m.R2ObjectCount,
		"cpu_p99_ms":           m.CPUP99AvgMS,
		"workers_script_count": m.WorkersScriptCount,
		"r2_bucket_count":      m.R2BucketCount,
		"dns_record_quota_pct": m.DNSRecordQuotaPct,
	}
}

// Evaluate runs every enabled rule against m and returns the names of rules
// that fired (after cooldown filtering). Each firing rule calls
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
		value, unit, ok := conditionValue(rule.Condition, m)
		if !ok || value < rule.Threshold {
			continue
		}
		if last, seen := e.lastFired[rule.Name]; seen && now.Sub(last) < e.cooldown {
			continue
		}

		e.lastFired[rule.Name] = now
		prev := e.fires[rule.Name]
		e.fires[rule.Name] = prev + 1
		fired = append(fired, rule.Name)

		updatedAt := rule.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = now
		}
		alert := &Alert{
			ID:        rule.Name, // deterministic: re-fires update the same alert
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
			rule.Condition, rule.Name, value, rule.Threshold, unit)

		if e.manager == nil {
			continue
		}
		if err := e.manager.TriggerAlert(alert, value, message, metricData(m)); err != nil {
			log.Printf("[alerts] trigger %q: %v", rule.Name, err)
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

	var m EvalMetrics
	var cpuSum float64
	var cpuCount int
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
func CollectLimitMetrics(ctx context.Context, limits *cosmoflare.LimitsService, m *EvalMetrics) error {
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
