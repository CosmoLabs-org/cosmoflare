package webhook

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// newEvalService creates a real AlertService over a temp dir, seeds rules via
// Create, and returns the service plus its rules path (tests that need to
// hand-write YAML, e.g. disabled rules, use the path).
func newEvalService(t *testing.T, rules ...*cosmoflare.AlertRule) (*cosmoflare.AlertService, string) {
	t.Helper()
	dir := t.TempDir()
	rulesPath := filepath.Join(dir, ".cosmoflare-alerts.yaml")
	svc, err := cosmoflare.NewAlertService(rulesPath, filepath.Join(dir, "alert-history.log"))
	if err != nil {
		t.Fatalf("NewAlertService: %v", err)
	}
	for _, r := range rules {
		if _, err := svc.Create(r); err != nil {
			t.Fatalf("Create(%s): %v", r.Name, err)
		}
	}
	return svc, rulesPath
}

// alertRecorder observes notifier payloads without any HTTP.
type alertRecorder struct {
	mu       sync.Mutex
	payloads []*NotificationPayload
}

func (r *alertRecorder) record(p *NotificationPayload) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.payloads = append(r.payloads, p)
}

func (r *alertRecorder) names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, 0, len(r.payloads))
	for _, p := range r.payloads {
		out = append(out, p.Alert.Name)
	}
	return out
}

func (r *alertRecorder) first() *NotificationPayload {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.payloads) == 0 {
		return nil
	}
	return r.payloads[0]
}

func newEvalManager(rec *alertRecorder) *Manager {
	m := NewManager(nil, "")
	m.SetNotifier(rec.record)
	return m
}

func evalRule(name, condition string, threshold float64) *cosmoflare.AlertRule {
	return &cosmoflare.AlertRule{
		Name:      name,
		Service:   "workers",
		Condition: condition,
		Threshold: threshold,
		Action:    "log",
		Target:    "/dev/null",
	}
}

func TestEvaluatorErrorRateFires(t *testing.T) {
	svc, _ := newEvalService(t,
		evalRule("err-over", "error-rate", 4),
		evalRule("err-under", "error-rate", 6),
	)
	rec := &alertRecorder{}
	eval := NewEvaluator(svc, newEvalManager(rec), time.Minute)

	fired := eval.Evaluate(EvalMetrics{WorkersRequests: 1000, WorkersErrors: 50})
	if len(fired) != 1 || fired[0] != "err-over" {
		t.Fatalf("fired = %v, want [err-over]", fired)
	}
	p := rec.first()
	if p == nil {
		t.Fatal("no notification recorded")
	}
	if !strings.Contains(p.Message, "5") || !strings.Contains(p.Message, "4") {
		t.Errorf("message %q missing observed value 5 or threshold 4", p.Message)
	}
	if !strings.Contains(p.Message, "%") {
		t.Errorf("message %q missing unit %%", p.Message)
	}
	if p.Value != 5 {
		t.Errorf("value = %v, want 5", p.Value)
	}
	if p.Threshold != 4 {
		t.Errorf("threshold = %v, want 4", p.Threshold)
	}
	if p.Alert.Type != AlertTypeThreshold {
		t.Errorf("alert type = %q, want %q", p.Alert.Type, AlertTypeThreshold)
	}
	if p.Alert.ID != "err-over" || p.Alert.Metric != "error-rate" {
		t.Errorf("alert ID/Metric = %q/%q, want err-over/error-rate", p.Alert.ID, p.Alert.Metric)
	}
	if p.Alert.Count != 1 {
		t.Errorf("count = %d, want 1", p.Alert.Count)
	}
	if got := rec.names(); len(got) != 1 {
		t.Errorf("notifications = %v, want one (err-under must not fire)", got)
	}
}

func TestEvaluatorZeroRequestsNeverFires(t *testing.T) {
	svc, _ := newEvalService(t, evalRule("err-zero", "error-rate", 1))
	rec := &alertRecorder{}
	eval := NewEvaluator(svc, newEvalManager(rec), time.Minute)

	fired := eval.Evaluate(EvalMetrics{WorkersErrors: 10})
	if len(fired) != 0 {
		t.Errorf("fired = %v, want none on zero requests", fired)
	}
	if len(rec.payloads) != 0 {
		t.Errorf("notifications = %d, want 0", len(rec.payloads))
	}
}

func TestEvaluatorStorageLimit(t *testing.T) {
	svc, _ := newEvalService(t,
		evalRule("store-over", "storage-limit", 1000),
		evalRule("store-under", "storage-limit", 1200),
	)
	rec := &alertRecorder{}
	eval := NewEvaluator(svc, newEvalManager(rec), time.Minute)

	fired := eval.Evaluate(EvalMetrics{R2StorageBytes: 1100})
	if len(fired) != 1 || fired[0] != "store-over" {
		t.Fatalf("fired = %v, want [store-over]", fired)
	}
	if msg := rec.first().Message; !strings.Contains(msg, "bytes") {
		t.Errorf("message %q missing unit bytes", msg)
	}
}

func TestEvaluatorLatencyAndFailureCount(t *testing.T) {
	svc, _ := newEvalService(t,
		evalRule("lat-over", "latency", 100),
		evalRule("fail-over", "failure-count", 5),
		evalRule("fail-under", "failure-count", 10),
	)
	rec := &alertRecorder{}
	eval := NewEvaluator(svc, newEvalManager(rec), time.Minute)

	fired := eval.Evaluate(EvalMetrics{CPUP99AvgMS: 120, WorkersErrors: 8})
	want := []string{"lat-over", "fail-over"}
	if strings.Join(fired, ",") != strings.Join(want, ",") {
		t.Fatalf("fired = %v, want %v", fired, want)
	}
	for _, p := range rec.payloads {
		if p.Alert.Name == "lat-over" && !strings.Contains(p.Message, "ms") {
			t.Errorf("latency message %q missing unit ms", p.Message)
		}
	}
}

func TestEvaluatorCooldown(t *testing.T) {
	svc, _ := newEvalService(t, evalRule("cool", "error-rate", 4))
	rec := &alertRecorder{}
	eval := NewEvaluator(svc, newEvalManager(rec), 15*time.Minute)

	base := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)
	now := base
	eval.SetClock(func() time.Time { return now })

	if fired := eval.Evaluate(EvalMetrics{WorkersRequests: 100, WorkersErrors: 10}); len(fired) != 1 {
		t.Fatalf("first evaluation fired = %v, want [cool]", fired)
	}

	now = base.Add(1 * time.Minute)
	if fired := eval.Evaluate(EvalMetrics{WorkersRequests: 100, WorkersErrors: 10}); len(fired) != 0 {
		t.Errorf("within cooldown fired = %v, want none", fired)
	}

	now = base.Add(16 * time.Minute)
	if fired := eval.Evaluate(EvalMetrics{WorkersRequests: 100, WorkersErrors: 10}); len(fired) != 1 {
		t.Errorf("after cooldown fired = %v, want [cool]", fired)
	}
	if len(rec.payloads) != 2 {
		t.Errorf("notifications = %d, want 2", len(rec.payloads))
	}
	if p := rec.payloads[1]; p.Alert.Count != 2 {
		t.Errorf("second fire count = %d, want 2", p.Alert.Count)
	}
}

func TestEvaluatorDisabledRuleSkipped(t *testing.T) {
	dir := t.TempDir()
	rulesPath := filepath.Join(dir, ".cosmoflare-alerts.yaml")
	svc, err := cosmoflare.NewAlertService(rulesPath, filepath.Join(dir, "history.log"))
	if err != nil {
		t.Fatalf("NewAlertService: %v", err)
	}
	yaml := "version: \"1\"\nrules:\n  - name: off-rule\n    service: workers\n    condition: error-rate\n    threshold: 1\n    action: log\n    target: /dev/null\n    enabled: false\n"
	if err := os.WriteFile(rulesPath, []byte(yaml), 0644); err != nil {
		t.Fatalf("write rules: %v", err)
	}

	rec := &alertRecorder{}
	eval := NewEvaluator(svc, newEvalManager(rec), time.Minute)
	if fired := eval.Evaluate(EvalMetrics{WorkersRequests: 100, WorkersErrors: 50}); len(fired) != 0 {
		t.Errorf("fired = %v, want none for disabled rule", fired)
	}
	if len(rec.payloads) != 0 {
		t.Errorf("notifications = %d, want 0", len(rec.payloads))
	}
}

func TestCollectEvalMetrics(t *testing.T) {
	workersResp := `{"data":{"viewer":{"accounts":[{"workersInvocationsAdaptive":[` +
		`{"sum":{"requests":1000,"errors":50},"quantiles":{"cpuTimeP50":10,"cpuTimeP99":120},"dimensions":{"scriptName":"a","status":"ok"}},` +
		`{"sum":{"requests":500,"errors":10},"quantiles":{"cpuTimeP50":5,"cpuTimeP99":80},"dimensions":{"scriptName":"b","status":"ok"}}` +
		`]}]}}}`
	storageResp := `{"data":{"viewer":{"accounts":[{"r2StorageAdaptiveGroups":[` +
		`{"max":{"objectCount":100,"payloadSize":1000},"dimensions":{"bucketName":"media"}},` +
		`{"max":{"objectCount":50,"payloadSize":500},"dimensions":{"bucketName":"backups"}}` +
		`]}]}}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(string(body), "workersInvocationsAdaptive") {
			w.Write([]byte(workersResp))
			return
		}
		w.Write([]byte(storageResp))
	}))
	defer srv.Close()

	analytics := cosmoflare.NewAnalyticsService("acct", "tok", cosmoflare.WithAnalyticsBaseURL(srv.URL))
	w := cosmoflare.AnalyticsWindow{Start: time.Now().Add(-24 * time.Hour), End: time.Now()}

	m, err := CollectEvalMetrics(context.Background(), analytics, w)
	if err != nil {
		t.Fatalf("CollectEvalMetrics: %v", err)
	}
	if m.WorkersRequests != 1500 || m.WorkersErrors != 60 {
		t.Errorf("workers requests/errors = %d/%d, want 1500/60", m.WorkersRequests, m.WorkersErrors)
	}
	if m.CPUP99AvgMS != 100 {
		t.Errorf("cpu p99 avg = %v, want 100", m.CPUP99AvgMS)
	}
	if m.R2StorageBytes != 1500 || m.R2ObjectCount != 150 {
		t.Errorf("r2 bytes/objects = %d/%d, want 1500/150", m.R2StorageBytes, m.R2ObjectCount)
	}
}

func TestCollectEvalMetricsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"errors":[{"message":"boom"}]}`))
	}))
	defer srv.Close()

	analytics := cosmoflare.NewAnalyticsService("acct", "tok", cosmoflare.WithAnalyticsBaseURL(srv.URL))
	w := cosmoflare.AnalyticsWindow{Start: time.Now().Add(-24 * time.Hour), End: time.Now()}

	if _, err := CollectEvalMetrics(context.Background(), analytics, w); err == nil {
		t.Fatal("CollectEvalMetrics should return an error on analytics failure, got nil")
	}
}

func TestConditionValueLimitMetrics(t *testing.T) {
	m := EvalMetrics{WorkersScriptCount: 90, R2BucketCount: 5, DNSRecordQuotaPct: 95}

	if v, unit, ok := conditionValue("workers-script-count", m); !ok || v != 90 || unit != "scripts" {
		t.Fatalf("workers-script-count = (%v, %q, %v)", v, unit, ok)
	}
	if v, unit, ok := conditionValue("r2-bucket-count", m); !ok || v != 5 || unit != "buckets" {
		t.Fatalf("r2-bucket-count = (%v, %q, %v)", v, unit, ok)
	}
	if v, unit, ok := conditionValue("dns-record-quota", m); !ok || v != 95 || unit != "%" {
		t.Fatalf("dns-record-quota = (%v, %q, %v)", v, unit, ok)
	}

	// Zero DNS percent means "no quota rows" — the condition must not fire.
	if _, _, ok := conditionValue("dns-record-quota", EvalMetrics{}); ok {
		t.Fatal("dns-record-quota must be !ok with no quota rows")
	}
}

func TestCollectLimitMetrics(t *testing.T) {
	// All sources failing must error; partial success must fill fields.
	// Fixture: reuse the package's existing fake/httptest patterns. A limits
	// service with no listers configured returns a snapshot with only
	// plan-unknown rows when DNS is absent — use a real httptest server for
	// subscriptions 403 + fake listers via exported options.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	limits := cosmoflare.NewLimitsService("acct", "tok",
		cosmoflare.WithLimitsBaseURL(srv.URL),
		cosmoflare.WithLimitsWorkers(fakeWorkers{n: 42}),
		cosmoflare.WithLimitsR2(fakeBuckets{n: 3}),
	)

	var m EvalMetrics
	if err := CollectLimitMetrics(context.Background(), limits, &m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.WorkersScriptCount != 42 || m.R2BucketCount != 3 {
		t.Fatalf("metrics = %+v, want 42 scripts / 3 buckets", m)
	}

	// All sources failing must error — zero rows + all sources errored is the
	// snapshot's hard-failure condition.
	failing := cosmoflare.NewLimitsService("acct", "tok",
		cosmoflare.WithLimitsBaseURL(srv.URL),
		cosmoflare.WithLimitsWorkers(fakeWorkers{err: errors.New("x")}),
		cosmoflare.WithLimitsR2(fakeBuckets{err: errors.New("x")}),
		cosmoflare.WithLimitsZones(fakeZones{err: errors.New("x")}),
		cosmoflare.WithLimitsAnalytics(fakeAnalytics{err: errors.New("x")}),
	)
	if err := CollectLimitMetrics(context.Background(), failing, &m); err == nil {
		t.Fatal("all limit sources failing must return an error")
	}
}

type fakeWorkers struct {
	n   int
	err error
}

func (f fakeWorkers) List(ctx context.Context) ([]*cosmoflare.Worker, error) {
	if f.err != nil {
		return nil, f.err
	}
	return make([]*cosmoflare.Worker, f.n), nil
}

type fakeBuckets struct {
	n   int
	err error
}

func (f fakeBuckets) ListBuckets(ctx context.Context) ([]*cosmoflare.Bucket, error) {
	if f.err != nil {
		return nil, f.err
	}
	return make([]*cosmoflare.Bucket, f.n), nil
}

type fakeZones struct{ err error }

func (f fakeZones) List(ctx context.Context) ([]*cosmoflare.Zone, error) {
	if f.err != nil {
		return nil, f.err
	}
	return nil, nil
}

type fakeAnalytics struct{ err error }

func (f fakeAnalytics) Workers(ctx context.Context, w cosmoflare.AnalyticsWindow) ([]cosmoflare.WorkersSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	return nil, nil
}

// TestConditionValueCoversRegistry pins the FEAT-015 contract across
// packages: every condition cosmoflare.AlertConditions() registers must be
// a case in conditionValue. The original bug: the limits feature extended
// the registry without the evaluator, and the CLI rejected valid conditions.
func TestConditionValueCoversRegistry(t *testing.T) {
	populated := EvalMetrics{
		WorkersRequests:    1000,
		WorkersErrors:      10,
		R2StorageBytes:     4096,
		CPUP99AvgMS:        12,
		WorkersScriptCount: 30,
		R2BucketCount:      7,
		DNSRecordQuotaPct:  55,
	}
	for _, c := range cosmoflare.AlertConditions() {
		value, unit, ok := conditionValue(c.Name, populated)
		if !ok {
			t.Errorf("conditionValue(%q) not ok with populated metrics — evaluator does not cover the registry", c.Name)
			continue
		}
		if unit != c.Unit {
			t.Errorf("conditionValue(%q) unit = %q, registry says %q", c.Name, unit, c.Unit)
		}
		if value == 0 && c.Name != "storage-limit" {
			t.Errorf("conditionValue(%q) = 0 with populated metrics, want non-zero", c.Name)
		}
	}
}
