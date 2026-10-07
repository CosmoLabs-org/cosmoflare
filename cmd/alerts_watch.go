package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/webhook"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/alertspush"
)

// minWatchInterval floors --interval. Pushing faster than this burns
// push-service quota (and risks 429s) with no alerting benefit.
const minWatchInterval = 10 * time.Second

// watchTestFireID is the payload ID of a --test-fire push. It is
// deterministic so pager clients dedupe repeated tests.
const watchTestFireID = "test-fire"

// watchCycleSeconds is the default evaluation cadence (60s per FEAT-045).
const watchCycleInterval = 60 * time.Second

// Flag variables for 'alerts watch'.
var (
	alertsWatchInterval time.Duration
	alertsWatchTestFire bool
)

// newWebPushSenderFn is the factory for the production Sender. Tests override
// it to capture envelopes without touching the network (same seam pattern as
// getAlertServiceFn).
var newWebPushSenderFn = func(publicKey, privateKey string) alertspush.Sender {
	return alertspush.NewWebPushSender(publicKey, privateKey)
}

// collectWatchMetricsFn collects the metrics one watch cycle evaluates
// against. It is a package-level var (same seam pattern as getAlertServiceFn)
// so tests drive one iteration against canned metrics instead of the live
// analytics API. The production implementation mirrors runAlertsCheck: 24h
// window ending now, credentials from the configured account.
var collectWatchMetricsFn = func(ctx context.Context) (webhook.EvalMetrics, error) {
	if AccountID == "" || APIToken == "" {
		return webhook.EvalMetrics{}, fmt.Errorf("account ID and API token are required for alert evaluation (run 'cosmoflare account' to configure)")
	}
	analytics := cosmoflare.NewAnalyticsService(AccountID, APIToken)
	now := time.Now()
	return webhook.CollectEvalMetrics(ctx, analytics, cosmoflare.AnalyticsWindow{
		Start: now.Add(-24 * time.Hour),
		End:   now,
	})
}

// pushStorePath resolves ~/.cosmoflare/push.json, using the same
// ~/.cosmoflare directory resolution as internal/config.
func pushStorePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".cosmoflare", "push.json"), nil
}

var alertsWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Evaluate alert rules on an interval and push fired alerts to pager devices",
	Long: `Continuously evaluate every enabled alert rule against live usage
analytics and deliver fired alerts as Web Push notifications to paired
pager devices (FEAT-045).

Each cycle collects a rolling 24h metrics window (Workers invocations and
errors, R2 storage) and judges every enabled rule — the same evaluation
'cosmoflare alerts check' runs once. Fired alerts are pushed to every
device subscribed via 'cosmoflare alerts push add'; expired device
endpoints (404/410 from the push service) are pruned automatically.

Pair a device first:
  cosmoflare alerts push keygen
  cosmoflare alerts push add '<subscription-json-from-your-pager>'

Examples:
  cosmoflare alerts watch
  cosmoflare alerts watch --interval 5m
  cosmoflare alerts watch --test-fire
  cosmoflare alerts watch --json`,
	RunE: runAlertsWatch,
}

func init() {
	alertsCmd.AddCommand(alertsWatchCmd)
	alertsWatchCmd.Flags().DurationVar(&alertsWatchInterval, "interval", watchCycleInterval, "Evaluation interval (minimum 10s)")
	alertsWatchCmd.Flags().BoolVar(&alertsWatchTestFire, "test-fire", false, "Send one canned info payload to every subscribed pager device and exit")
}

// runAlertsWatch implements 'cosmoflare alerts watch'. With --test-fire it
// dispatches a canned info payload once and exits 0; otherwise it loops
// evaluateOnce on the --interval cadence until interrupted.
func runAlertsWatch(cmd *cobra.Command, args []string) error {
	if !alertsWatchTestFire && alertsWatchInterval < minWatchInterval {
		return outErrf("--interval %s is below the %s minimum; raise it (e.g. --interval 60s)", alertsWatchInterval, minWatchInterval)
	}

	path, err := pushStorePath()
	if err != nil {
		return outErr("failed to resolve push store path", err)
	}
	st, err := alertspush.LoadStore(path)
	if err != nil {
		return outErr("failed to load push store", err)
	}

	if alertsWatchTestFire {
		return runAlertsWatchTestFire(cmd.Context(), st)
	}

	if len(st.Subscriptions) == 0 {
		return outErrf("no pager devices are subscribed; pair one first (run 'cosmoflare alerts push keygen', then 'cosmoflare alerts push add')")
	}
	if !st.HasVAPIDKeys() {
		return outErrf("push store has no VAPID keys; run 'cosmoflare alerts push keygen' first")
	}

	svc, err := getAlertService()
	if err != nil {
		return outErr("failed to create alert service", err)
	}
	sender := newWebPushSenderFn(st.VAPIDPublicKey, st.VAPIDPrivateKey)

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	printInfo("Watching alert rules every %s (%d pager device(s) subscribed, Ctrl+C to stop)", alertsWatchInterval, len(st.Subscriptions))
	for {
		sent, pruned, err := evaluateOnce(ctx, svc, st, sender)
		if err != nil {
			// A failed cycle (transient network, missing credentials) must
			// not kill the watch; the next cycle retries.
			printWarning("watch cycle failed: %v", err)
		} else if sent > 0 || pruned > 0 {
			printInfo("pushed %d alert(s), pruned %d device(s)", sent, pruned)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(alertsWatchInterval):
		}
	}
}

// runAlertsWatchTestFire dispatches one canned info payload and exits 0, so
// users can verify pairing end to end without waiting for a real alert.
func runAlertsWatchTestFire(ctx context.Context, st *alertspush.Store) error {
	if len(st.Subscriptions) > 0 && !st.HasVAPIDKeys() {
		return outErrf("push store has subscriptions but no VAPID keys; run 'cosmoflare alerts push keygen' first")
	}

	payload := alertspush.Payload{
		ID:       watchTestFireID,
		Severity: alertspush.SeverityInfo,
		Service:  "cloudflare",
		Title:    "Cosmoflare pager test",
		Detail:   "Test fire from 'cosmoflare alerts watch --test-fire'. Seeing this on your pager means pairing works.",
		FiredAt:  time.Now(),
	}

	// With zero subscriptions Dispatch has nothing to deliver and the nil
	// sender is never called; exit stays 0 with guidance in the output.
	var sender alertspush.Sender
	if len(st.Subscriptions) > 0 {
		sender = newWebPushSenderFn(st.VAPIDPublicKey, st.VAPIDPrivateKey)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	sent, pruned, err := alertspush.Dispatch(ctx, st, sender, payload)
	if err != nil {
		return outErr("test fire partially failed", err)
	}

	return outResult(map[string]any{"sent": sent, "pruned": pruned, "payload_id": watchTestFireID}, func() {
		if sent == 0 && pruned == 0 {
			printInfo("No pager devices are subscribed yet; pair one with 'cosmoflare alerts push add'.")
			return
		}
		printInfo("Test push delivered to %d device(s), %d pruned", sent, pruned)
	})
}

// evaluateOnce runs ONE watch cycle: evaluate every enabled rule against the
// cycle's metrics and push a pager payload for each fired alert. It is
// package-level so tests drive a single iteration without the loop. sent
// counts successful deliveries, pruned counts expired endpoints removed from
// the store. The first non-prune delivery error is returned after all fired
// alerts have been attempted.
func evaluateOnce(ctx context.Context, svc *cosmoflare.AlertService, st *alertspush.Store, sender alertspush.Sender) (sent, pruned int, err error) {
	rules, err := svc.List()
	if err != nil {
		return 0, 0, fmt.Errorf("list alert rules: %w", err)
	}
	byName := make(map[string]*cosmoflare.AlertRule, len(rules))
	evaluated := 0
	for _, r := range rules {
		if r == nil {
			continue
		}
		byName[r.Name] = r
		if r.Enabled {
			evaluated++
		}
	}
	if evaluated == 0 {
		// Nothing enabled: stay offline (no credentials, no network), the
		// same contract runAlertsCheck honors.
		return 0, 0, nil
	}

	metrics, err := collectWatchMetricsFn(ctx)
	if err != nil {
		return 0, 0, err
	}

	var firstErr error
	mgr := webhook.NewManager(nil, "")
	mgr.SetNotifier(func(p *webhook.NotificationPayload) {
		if p == nil || p.Alert == nil || sender == nil || st == nil {
			return
		}
		rule := byName[p.Alert.Name]
		payload := alertspush.Payload{
			ID:       p.Alert.ID,
			Severity: ruleSeverity(rule),
			Service:  ruleService(rule),
			Title:    p.Alert.Name,
			Detail:   p.Message,
			FiredAt:  p.Timestamp,
		}
		if payload.FiredAt.IsZero() {
			payload.FiredAt = time.Now()
		}
		s, pr, dispatchErr := alertspush.Dispatch(ctx, st, sender, payload)
		sent += s
		pruned += pr
		if dispatchErr != nil && firstErr == nil {
			firstErr = dispatchErr
		}
	})

	// One-shot evaluator per cycle: cooldown state dies with the evaluator,
	// so a persistently-tripped rule re-fires every interval (deliberate —
	// a broken service keeps buzzing the pager until it is fixed or
	// disabled, mirroring 'alerts check' semantics).
	eval := webhook.NewEvaluator(svc, mgr, 0)
	_ = eval.Evaluate(metrics)
	return sent, pruned, firstErr
}

// ruleSeverity maps an alert rule to the pager severity vocabulary
// (info | warning | critical). Alert rules carry no priority level today, so
// per the FEAT-045 plan ("map existing levels to info/warning/critical,
// defaulting info") every rule currently maps to SeverityInfo; when a
// priority field lands on AlertRule this is the single mapping point.
func ruleSeverity(_ *cosmoflare.AlertRule) alertspush.Severity {
	return alertspush.SeverityInfo
}

// ruleService returns the rule's service, defaulting to "cloudflare" when
// unset so every payload passes validation.
func ruleService(r *cosmoflare.AlertRule) string {
	if r != nil && r.Service != "" {
		return r.Service
	}
	return "cloudflare"
}
