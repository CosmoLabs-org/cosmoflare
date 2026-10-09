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
// window ending now, credentials from the configured account. rules lets it
// fetch zone/d1 telemetry only when an enabled rule needs it (FEAT-049).
var collectWatchMetricsFn = func(ctx context.Context, rules []*cosmoflare.AlertRule) (webhook.EvalMetrics, error) {
	if AccountID == "" || APIToken == "" {
		return webhook.EvalMetrics{}, fmt.Errorf("account ID and API token are required for alert evaluation (run 'cosmoflare account' to configure)")
	}
	analytics := cosmoflare.NewAnalyticsService(AccountID, APIToken)
	now := time.Now()
	w := cosmoflare.AnalyticsWindow{Start: now.Add(-24 * time.Hour), End: now}
	m, err := webhook.CollectEvalMetrics(ctx, analytics, w)
	if err != nil {
		return m, err
	}
	// Monthly pacing rides the watch on the slow loop (FEAT-048 D3): the
	// snapshot is cached ~15 min so the 60s cycle never re-queries cycle
	// telemetry. A nil snapshot leaves the usage-* conditions skipped.
	webhook.CollectUsageMetrics(&m, cachedUsageSnapshot(ctx))
	// Zone cache + D1 rows-read telemetry is additive (FEAT-049 D15):
	// failures become telemetry-gap pages, never a skipped cycle.
	wantZones, wantD1 := webhook.RulesUseScope(rules, "zone"), webhook.RulesUseScope(rules, "d1")
	if wantZones || wantD1 {
		webhook.CollectTelemetryMetrics(ctx, analytics, wantZones, wantD1, cachedTelemetryRefs(ctx, wantZones, wantD1), w, &m)
	}
	return m, nil
}

// Watch cooldown memory (FEAT-049 O3). The watch builds a fresh evaluator
// each cycle; sharing one FireState keeps zone/d1 cooldowns across cycles.
// It is in memory only: a restart re-pages every current zone/d1 breach
// once. Scopes without an entry use cooldown 0 — they keep re-paging every
// cycle, the watch's long-standing policy.
var (
	watchFireState      = webhook.NewFireState()
	watchScopeCooldowns = map[string]time.Duration{"zone": time.Hour, "d1": time.Hour}
)

// Zone/D1 name-list cache + seams. The lists change on a days timescale, so
// each list is fetched at most every 15 min; a failed fetch is retried after
// 5 min, never every cycle (a token without D1:Read fails forever); a failed
// refresh keeps the last good list (stale names beat a gap page).
var (
	telemetryRefsCache    webhook.TelemetryRefs
	telemetryZonesAttempt time.Time // last zone-list fetch attempt
	telemetryDBAttempt    time.Time // last D1-list fetch attempt
	telemetryRefsTTL      = 15 * time.Minute
	telemetryRefsRetry    = 5 * time.Minute
	telemetryNowFn        = time.Now
	telemetryRefsFn       = defaultTelemetryRefs
)

// defaultTelemetryRefs lists active zones and D1 databases for the wanted
// scopes. Each list's error is carried in the refs, never returned.
func defaultTelemetryRefs(ctx context.Context, wantZones, wantD1 bool) webhook.TelemetryRefs {
	var refs webhook.TelemetryRefs
	if wantZones {
		zs, err := cosmoflare.NewZoneServiceFromCreds(AccountID, APIToken)
		var zones []*cosmoflare.Zone
		if err == nil {
			zones, err = zs.List(ctx)
		}
		if err != nil {
			refs.ZonesErr = fmt.Errorf("zone list: %w", err)
		}
		for _, z := range zones {
			if z != nil && z.Status == "active" {
				refs.Zones = append(refs.Zones, cosmoflare.ZoneRef{ID: z.ID, Name: z.Name})
			}
		}
	}
	if wantD1 {
		ds, err := cosmoflare.NewD1ServiceFromCreds(AccountID, APIToken)
		var dbs []*cosmoflare.D1Database
		if err == nil {
			dbs, err = ds.List(ctx)
		}
		if err != nil {
			refs.DBNamesErr = fmt.Errorf("d1 list: %w", err)
		}
		refs.DBNames = make(map[string]string, len(dbs))
		for _, d := range dbs {
			if d != nil {
				refs.DBNames[d.UUID] = d.Name
			}
		}
	}
	return refs
}

// cachedTelemetryRefs returns the wanted lists, fetching only those that are
// due: never attempted, older than the TTL, or failed more than the retry
// interval ago. A failed refresh keeps the last good list for that scope;
// with no last good list the error surfaces (a zone-list error becomes a gap
// page in CollectTelemetryMetrics; a D1-list error names databases by ID).
func cachedTelemetryRefs(ctx context.Context, wantZones, wantD1 bool) webhook.TelemetryRefs {
	now := telemetryNowFn()
	due := func(attempt time.Time, err error) bool {
		if attempt.IsZero() {
			return true
		}
		wait := telemetryRefsTTL
		if err != nil {
			wait = telemetryRefsRetry
		}
		return now.Sub(attempt) >= wait
	}
	needZones := wantZones && due(telemetryZonesAttempt, telemetryRefsCache.ZonesErr)
	needD1 := wantD1 && due(telemetryDBAttempt, telemetryRefsCache.DBNamesErr)
	if needZones || needD1 {
		fresh := telemetryRefsFn(ctx, needZones, needD1)
		if needZones {
			telemetryZonesAttempt = now
			if fresh.ZonesErr != nil && len(telemetryRefsCache.Zones) > 0 {
				printWarning("zone list refresh failed, using last known zones: %v", fresh.ZonesErr)
				telemetryRefsCache.ZonesErr = nil
			} else {
				telemetryRefsCache.Zones, telemetryRefsCache.ZonesErr = fresh.Zones, fresh.ZonesErr
			}
		}
		if needD1 {
			telemetryDBAttempt = now
			switch {
			case fresh.DBNamesErr != nil && len(telemetryRefsCache.DBNames) > 0:
				printWarning("D1 list refresh failed, using last known database names: %v", fresh.DBNamesErr)
				telemetryRefsCache.DBNamesErr = nil
			case fresh.DBNamesErr != nil:
				printWarning("D1 list unavailable, d1 alerts name databases by ID (retry in %s): %v", telemetryRefsRetry, fresh.DBNamesErr)
				telemetryRefsCache.DBNames, telemetryRefsCache.DBNamesErr = fresh.DBNames, fresh.DBNamesErr
			default:
				telemetryRefsCache.DBNames, telemetryRefsCache.DBNamesErr = fresh.DBNames, nil
			}
		}
	}
	out := telemetryRefsCache
	if !wantZones {
		out.Zones, out.ZonesErr = nil, nil
	}
	if !wantD1 {
		out.DBNames, out.DBNamesErr = nil, nil
	}
	return out
}

// Usage snapshot cache + seams (tests drive the clock and the collector).
var (
	usageSnapCache      *cosmoflare.UsageSnapshot
	usageSnapFetchedAt  time.Time
	usageCacheTTL       = 15 * time.Minute
	usageNowFn          = time.Now
	usageCollectWatchFn = defaultUsageCollectWatch
)

func defaultUsageCollectWatch(ctx context.Context) (*cosmoflare.UsageSnapshot, error) {
	analytics := cosmoflare.NewAnalyticsService(AccountID, APIToken)
	return cosmoflare.CollectUsage(ctx, analytics, "paid", 0, usageNowFn().UTC())
}

// cachedUsageSnapshot returns the cached snapshot within the TTL, else
// collects a fresh one. Collection errors keep the last good snapshot
// (stale pacing beats no pacing); nil when never collected successfully.
func cachedUsageSnapshot(ctx context.Context) *cosmoflare.UsageSnapshot {
	if usageSnapCache != nil && usageNowFn().Sub(usageSnapFetchedAt) < usageCacheTTL {
		return usageSnapCache
	}
	snap, err := usageCollectWatchFn(ctx)
	if err != nil {
		printWarning("usage snapshot unavailable, usage-* conditions use last known: %v", err)
		return usageSnapCache
	}
	usageSnapCache = snap
	usageSnapFetchedAt = usageNowFn()
	return snap
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
errors, Durable Objects, R2 storage, monthly usage pacing) and judges every
enabled rule — the same evaluation 'cosmoflare alerts check' runs once.
When a zone or d1 rule is enabled the cycle also collects zone cache
status per zone and D1 rows read per database (FEAT-049).

Re-paging: account, script and do rules re-page every cycle while
tripped. zone and d1 rules page at most once an hour per zone or
database, unless the value doubles since the last page; a recovery and
re-breach inside the hour stays quiet. If zone or D1 telemetry cannot be
collected, a telemetry gap page fires once an hour so silent rules are
never mistaken for healthy ones. Cooldown memory is in-process: a
restart re-pages current zone/d1 breaches once. Zones need at least 100
requests in the window before their cache ratios are judged.

Fired alerts are pushed to every
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
		cycleCtx, cancel := context.WithTimeout(ctx, watchCycleTimeout(alertsWatchInterval))
		sent, pruned, err := evaluateOnce(cycleCtx, svc, st, sender)
		cancel()
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

// watchCycleTimeout bounds one watch cycle: max(interval, 60s). A cycle now
// chains many sequential API calls (analytics, usage, zone/D1 lists, zone
// batches, D1); without a deadline a degraded API could stall every pager
// rule for minutes. Telemetry hit by the deadline becomes a gap page.
func watchCycleTimeout(interval time.Duration) time.Duration {
	if interval < 60*time.Second {
		return 60 * time.Second
	}
	return interval
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

	metrics, err := collectWatchMetricsFn(ctx, rules)
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

	// Fresh evaluator per cycle over shared cooldown memory. Scopes without
	// a cooldown entry (account, script, do) re-fire every interval
	// (deliberate — a broken service keeps buzzing the pager until it is
	// fixed or disabled, mirroring 'alerts check' semantics). zone/d1 rules
	// over rolling 24h windows stay tripped for hours after a fix, so they
	// page at most hourly per zone/database unless the value doubles
	// (FEAT-049 O3/O4).
	eval := webhook.NewEvaluator(svc, mgr, 0)
	eval.UseState(watchFireState)
	eval.SetCooldownPolicy(0, watchScopeCooldowns)
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
