package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/webhook"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/alertspush"
)

// watchFakeSender is a no-network alertspush.Sender capturing every envelope
// it is handed. status maps endpoint → HTTP status code (default 200) so the
// 404/410 pruning path can be exercised deterministically.
type watchFakeSender struct {
	mu        sync.Mutex
	status    map[string]int
	envelopes [][]byte
}

func (f *watchFakeSender) Send(_ context.Context, sub alertspush.Subscription, envelope []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.envelopes = append(f.envelopes, append([]byte(nil), envelope...))
	if code, ok := f.status[sub.Endpoint]; ok {
		return code, nil
	}
	return http.StatusOK, nil
}

// lastEnvelope returns the most recent envelope, failing the test if none
// was captured.
func (f *watchFakeSender) lastEnvelope(t *testing.T) []byte {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.envelopes) == 0 {
		t.Fatal("fake sender recorded no envelopes")
	}
	return f.envelopes[len(f.envelopes)-1]
}

// watchAlertEnv wires the seams evaluateOnce reads: getAlertServiceFn points
// at a temp rules store (the same seam the other alert tests use) and
// collectWatchMetricsFn returns canned metrics that trip a storage-limit
// rule with threshold < 100. Both are restored on cleanup. Returns the
// service and its rules path so tests can seed rules.
func watchAlertEnv(t *testing.T) (*cosmoflare.AlertService, string) {
	t.Helper()
	dir := t.TempDir()
	rulesPath := filepath.Join(dir, ".cosmoflare-alerts.yaml")
	historyPath := filepath.Join(dir, "alert-history.log")

	origSvc := getAlertServiceFn
	origMetrics := collectWatchMetricsFn
	t.Cleanup(func() {
		getAlertServiceFn = origSvc
		collectWatchMetricsFn = origMetrics
	})
	getAlertServiceFn = func() (*cosmoflare.AlertService, error) {
		return cosmoflare.NewAlertService(rulesPath, historyPath)
	}
	collectWatchMetricsFn = func(context.Context) (webhook.EvalMetrics, error) {
		return webhook.EvalMetrics{WorkersRequests: 1000, WorkersErrors: 5, R2StorageBytes: 100}, nil
	}

	svc, err := cosmoflare.NewAlertService(rulesPath, historyPath)
	if err != nil {
		t.Fatalf("NewAlertService: %v", err)
	}
	return svc, rulesPath
}

// watchWriteRule writes rules straight to the service's YAML store (same
// trick alerts_check_test.go uses) because that is the deterministic way to
// get an enabled rule with exact fields.
func watchWriteRule(t *testing.T, rulesPath, ruleYAML string) {
	t.Helper()
	if err := os.WriteFile(rulesPath, []byte(ruleYAML), 0o600); err != nil {
		t.Fatalf("write rules yaml: %v", err)
	}
}

const watchStorageRuleYAML = "rules:\n" +
	"  - name: r2-storage\n" +
	"    service: r2\n" +
	"    condition: storage-limit\n" +
	"    threshold: 80\n" +
	"    action: log\n" +
	"    target: /dev/null\n" +
	"    enabled: true\n"

// TestEvaluateOnceTrippedRulePushesEnvelope verifies one evaluation iteration
// pushes exactly one envelope for a tripped rule and that the envelope
// follows the FEAT-045 rule→payload mapping.
func TestEvaluateOnceTrippedRulePushesEnvelope(t *testing.T) {
	svc, rulesPath := watchAlertEnv(t)
	watchWriteRule(t, rulesPath, watchStorageRuleYAML)

	st := &alertspush.Store{Subscriptions: []alertspush.Subscription{
		{Endpoint: "https://push.example/e1", P256dh: "k", Auth: "a"},
	}}
	sender := &watchFakeSender{}

	sent, pruned, err := evaluateOnce(context.Background(), svc, st, sender)
	if err != nil {
		t.Fatalf("evaluateOnce: %v", err)
	}
	if sent != 1 || pruned != 0 {
		t.Fatalf("sent=%d pruned=%d, want 1/0", sent, pruned)
	}

	var p alertspush.Payload
	if err := json.Unmarshal(sender.lastEnvelope(t), &p); err != nil {
		t.Fatalf("envelope is not valid JSON: %v", err)
	}
	if p.ID != "r2-storage" {
		t.Errorf("payload ID = %q, want rule name %q", p.ID, "r2-storage")
	}
	if p.Severity != alertspush.SeverityInfo {
		t.Errorf("severity = %q, want default %q (rules carry no priority today)", p.Severity, alertspush.SeverityInfo)
	}
	if p.Service != "r2" {
		t.Errorf("service = %q, want the rule service %q", p.Service, "r2")
	}
	if p.Title != "r2-storage" {
		t.Errorf("title = %q, want rule name", p.Title)
	}
	if p.Detail == "" {
		t.Error("detail empty, want the check message")
	}
	if p.FiredAt.IsZero() || time.Since(p.FiredAt) > time.Minute {
		t.Errorf("fired_at = %v, want a fresh timestamp", p.FiredAt)
	}
	if err := p.Validate(); err != nil {
		t.Errorf("payload fails its own validation: %v", err)
	}
}

// TestEvaluateOnceDefaultsServiceToCloudflare verifies a rule with no service
// maps to the "cloudflare" default service.
func TestEvaluateOnceDefaultsServiceToCloudflare(t *testing.T) {
	svc, rulesPath := watchAlertEnv(t)
	watchWriteRule(t, rulesPath, "rules:\n"+
		"  - name: no-service\n"+
		"    condition: storage-limit\n"+
		"    threshold: 80\n"+
		"    action: log\n"+
		"    target: /dev/null\n"+
		"    enabled: true\n")

	st := &alertspush.Store{Subscriptions: []alertspush.Subscription{
		{Endpoint: "https://push.example/e1", P256dh: "k", Auth: "a"},
	}}
	sender := &watchFakeSender{}

	if _, _, err := evaluateOnce(context.Background(), svc, st, sender); err != nil {
		t.Fatalf("evaluateOnce: %v", err)
	}
	var p alertspush.Payload
	if err := json.Unmarshal(sender.lastEnvelope(t), &p); err != nil {
		t.Fatalf("envelope is not valid JSON: %v", err)
	}
	if p.Service != "cloudflare" {
		t.Errorf("service = %q, want default %q", p.Service, "cloudflare")
	}
}

// TestEvaluateOncePrunesExpiredSubscription verifies a 410 endpoint is pruned
// from the store instead of counted as sent, and delivery to healthy
// endpoints continues.
func TestEvaluateOncePrunesExpiredSubscription(t *testing.T) {
	svc, rulesPath := watchAlertEnv(t)
	watchWriteRule(t, rulesPath, watchStorageRuleYAML)

	st := &alertspush.Store{Subscriptions: []alertspush.Subscription{
		{Endpoint: "https://push.example/gone", P256dh: "k", Auth: "a"},
		{Endpoint: "https://push.example/alive", P256dh: "k", Auth: "a"},
	}}
	sender := &watchFakeSender{status: map[string]int{
		"https://push.example/gone": http.StatusGone,
	}}

	sent, pruned, err := evaluateOnce(context.Background(), svc, st, sender)
	if err != nil {
		t.Fatalf("evaluateOnce: %v", err)
	}
	if sent != 1 || pruned != 1 {
		t.Fatalf("sent=%d pruned=%d, want 1/1", sent, pruned)
	}
	for _, sub := range st.Subscriptions {
		if sub.Endpoint == "https://push.example/gone" {
			t.Error("expired subscription was not pruned from the store")
		}
	}
}

// TestEvaluateOnceNoEnabledRulesIsOffline verifies a rules store without
// enabled rules succeeds without collecting metrics (no credentials, no
// network) — the same contract runAlertsCheck honors.
func TestEvaluateOnceNoEnabledRulesIsOffline(t *testing.T) {
	svc, _ := watchAlertEnv(t)

	called := false
	orig := collectWatchMetricsFn
	collectWatchMetricsFn = func(context.Context) (webhook.EvalMetrics, error) {
		called = true
		return webhook.EvalMetrics{}, nil
	}
	defer func() { collectWatchMetricsFn = orig }()

	sent, pruned, err := evaluateOnce(context.Background(), svc, &alertspush.Store{}, &watchFakeSender{})
	if err != nil {
		t.Fatalf("evaluateOnce with no enabled rules: %v", err)
	}
	if sent != 0 || pruned != 0 {
		t.Fatalf("sent=%d pruned=%d, want 0/0", sent, pruned)
	}
	if called {
		t.Error("metrics collected despite no enabled rules")
	}
}

// TestAlertsWatchTestFirePushesCannedInfo drives the cobra command with
// --test-fire against a fake sender: the canned payload must carry severity
// info and the command must exit 0.
func TestAlertsWatchTestFirePushesCannedInfo(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	pushPath := filepath.Join(dir, ".cosmoflare", "push.json")
	st := &alertspush.Store{
		VAPIDPublicKey:  "pub",
		VAPIDPrivateKey: "priv",
		Subscriptions: []alertspush.Subscription{
			{Endpoint: "https://push.example/phone", P256dh: "k", Auth: "a"},
		},
	}
	if err := st.Save(pushPath); err != nil {
		t.Fatalf("seed push store: %v", err)
	}

	// Reset watch flag globals (pflag does not reset unset flags between
	// parses, and tests in this package share the command tree).
	alertsWatchInterval = watchCycleInterval
	alertsWatchTestFire = false
	origSvc := getAlertServiceFn
	origSender := newWebPushSenderFn
	t.Cleanup(func() {
		getAlertServiceFn = origSvc
		newWebPushSenderFn = origSender
		alertsWatchInterval = watchCycleInterval
		alertsWatchTestFire = false
	})
	getAlertServiceFn = func() (*cosmoflare.AlertService, error) {
		return cosmoflare.NewAlertService(filepath.Join(dir, ".cosmoflare-alerts.yaml"), filepath.Join(dir, "alert-history.log"))
	}
	sender := &watchFakeSender{}
	newWebPushSenderFn = func(string, string) alertspush.Sender { return sender }

	if _, err := executeAlertsCommand("alerts", "watch", "--test-fire"); err != nil {
		t.Fatalf("alerts watch --test-fire exited non-zero: %v", err)
	}

	var p alertspush.Payload
	if err := json.Unmarshal(sender.lastEnvelope(t), &p); err != nil {
		t.Fatalf("test-fire envelope is not valid JSON: %v", err)
	}
	if p.Severity != alertspush.SeverityInfo {
		t.Errorf("test-fire severity = %q, want %q", p.Severity, alertspush.SeverityInfo)
	}
	if p.ID != watchTestFireID {
		t.Errorf("test-fire ID = %q, want %q", p.ID, watchTestFireID)
	}
	if err := p.Validate(); err != nil {
		t.Errorf("canned payload fails validation: %v", err)
	}
}

// TestAlertsWatchIntervalBounds verifies the flag defaults to 60s and values
// below the 10s floor are rejected with an agent-readable error.
func TestAlertsWatchIntervalBounds(t *testing.T) {
	f := alertsWatchCmd.Flags().Lookup("interval")
	if f == nil {
		t.Fatal("--interval flag not registered")
	}
	def, err := time.ParseDuration(f.DefValue)
	if err != nil {
		t.Fatalf("parse default %q: %v", f.DefValue, err)
	}
	if def != 60*time.Second {
		t.Errorf("interval default = %s, want 60s", def)
	}

	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// Reset watch flag globals: --test-fire may be true from an earlier test.
	alertsWatchInterval = watchCycleInterval
	alertsWatchTestFire = false
	origSvc := getAlertServiceFn
	t.Cleanup(func() {
		getAlertServiceFn = origSvc
		alertsWatchInterval = watchCycleInterval
		alertsWatchTestFire = false
	})
	getAlertServiceFn = func() (*cosmoflare.AlertService, error) {
		return cosmoflare.NewAlertService(filepath.Join(dir, ".cosmoflare-alerts.yaml"), filepath.Join(dir, "alert-history.log"))
	}

	_, err = executeAlertsCommand("alerts", "watch", "--interval", "5s")
	if err == nil {
		t.Fatal("expected --interval 5s to be rejected")
	}
	if !strings.Contains(err.Error(), "10s") {
		t.Errorf("error %q does not name the 10s minimum", err)
	}
}

// TestCachedUsageSnapshotTTL pins the FEAT-048 slow loop: within the TTL
// the collector is not called again; past it, it is. A collection error
// keeps the last good snapshot rather than nil-ing the pacing conditions.
func TestCachedUsageSnapshotTTL(t *testing.T) {
	calls := 0
	base := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	clock := base
	prevCollect, prevNow := usageCollectWatchFn, usageNowFn
	prevCache, prevAt := usageSnapCache, usageSnapFetchedAt
	usageCollectWatchFn = func(ctx context.Context) (*cosmoflare.UsageSnapshot, error) {
		calls++
		return &cosmoflare.UsageSnapshot{DaysElapsed: float64(calls)}, nil
	}
	usageNowFn = func() time.Time { return clock }
	t.Cleanup(func() {
		usageCollectWatchFn, usageNowFn = prevCollect, prevNow
		usageSnapCache, usageSnapFetchedAt = prevCache, prevAt
	})

	first := cachedUsageSnapshot(context.Background())
	if first == nil || first.DaysElapsed != 1 {
		t.Fatalf("first snapshot = %+v, want DaysElapsed 1", first)
	}
	if again := cachedUsageSnapshot(context.Background()); again != first {
		t.Fatal("within TTL the cache must return the same snapshot without collecting")
	}
	if calls != 1 {
		t.Fatalf("collector calls = %d, want 1 within TTL", calls)
	}

	clock = base.Add(16 * time.Minute)
	second := cachedUsageSnapshot(context.Background())
	if second == first || second.DaysElapsed != 2 {
		t.Fatalf("past TTL snapshot = %+v, want a fresh collect", second)
	}

	usageCollectWatchFn = func(ctx context.Context) (*cosmoflare.UsageSnapshot, error) {
		return nil, fmt.Errorf("analytics down")
	}
	clock = base.Add(32 * time.Minute)
	if kept := cachedUsageSnapshot(context.Background()); kept != second {
		t.Fatal("collection error must keep the last good snapshot, not nil")
	}
}
