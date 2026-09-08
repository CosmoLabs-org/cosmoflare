package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	"github.com/CosmoLabs-org/cosmoflare/internal/server"
	"github.com/CosmoLabs-org/cosmoflare/internal/webhook"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var (
	serveAddr            string        // --addr, default 127.0.0.1:0 (ephemeral port)
	serveToken           string        // --token, optional; generated if empty
	serveMetricsInterval time.Duration // --metrics-interval
)

// alertEvalInterval is how often the daemon re-evaluates alert rules against
// live analytics. Values below 30s are clamped to 30s in the loop.
const alertEvalInterval = 5 * time.Minute

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the local HTTP+SSE daemon for the Cosmoflare desktop app",
	Long: `Start the local daemon the Cosmoflare desktop app connects to.

The daemon binds a localhost address (default 127.0.0.1:0 — the OS picks an
ephemeral port) and protects every endpoint with a bearer token. On startup it
prints a single JSON handshake line to stdout:

  {"addr":"127.0.0.1:54123","token":"<random>"}

The desktop shell (Tauri) parses this line for the address + token, then drives
the webview against the daemon's REST + SSE endpoints:

  GET /healthz                   two-tier health (systems + Cloudflare online)
  GET /accounts|/zones|/r2/buckets|/workers|/kv
  GET /events                    SSE: metrics / notifications / status

Credentials are resolved lazily per request from the config profile selected
via ?profile=<name>. The daemon starts with NO valid credentials and reports
cloudflare_online=false so the first-run setup screen can run. It never probes
the OS keychain (COSMOFLARE_NO_KEYCHAIN=1 is set in-process).

Examples:
  cosmoflare serve                                # ephemeral port, random token
  cosmoflare serve --addr 127.0.0.1:8421          # fixed port
  cosmoflare serve --token $(openssl rand -hex 16) # caller-supplied token`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", "127.0.0.1:0", "address to bind (host:port; port 0 = ephemeral)")
	serveCmd.Flags().StringVar(&serveToken, "token", "", "bearer auth token (random if empty)")
	serveCmd.Flags().DurationVar(&serveMetricsInterval, "metrics-interval", 30*time.Second, "how often to poll Cloudflare for live metrics (0 to disable)")
	rootCmd.AddCommand(serveCmd)
}

// runServe starts the daemon. It must run without valid Cloudflare credentials
// (cmd/root.go's PersistentPreRun skips validation for "serve"), so it is
// registered there. The daemon resolves creds per-request from the selected
// config profile.
func runServe(cmd *cobra.Command, args []string) error {
	// Hard keychain gate (BR-04): never probe the OS keychain. The desktop
	// shell also sets this in the child env; setting it in-process is
	// belt-and-suspenders and makes `cosmoflare serve` safe to run directly.
	os.Setenv("COSMOFLARE_NO_KEYCHAIN", "1")

	token := serveToken
	if token == "" {
		generated, err := randomToken()
		if err != nil {
			return fmt.Errorf("generate token: %w", err)
		}
		token = generated
	}

	ln, err := net.Listen("tcp", serveAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", serveAddr, err)
	}

	srv := server.New(server.Config{Token: token, Version: AppVersion})

	// FEAT-008: wire the alert→SSE bridge. The evaluator loop below is the
	// alert producer: every cycle it judges the stored alert rules against
	// live analytics and fires TriggerAlert through this bridge onto the SSE
	// hub. The bridge itself is covered by
	// TestServeAlertBridge_PublishesAlertToNotificationsChannel.
	alertMgr := newServeAlertBridge(srv)

	// Wire the real data source: per-request credential resolution from the
	// local config profiles, delegating to the existing per-service
	// constructors. NewConfigManager is safe on first run (no config file →
	// empty profile list → CF endpoints report cloudflare_online=false, which
	// drives the first-run setup screen rather than crashing).
	cm, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	adapter := &serveAdapter{cm: cm}
	srv.SetData(adapter)

	httpServer := &http.Server{Handler: srv.Handler()}

	// Handshake: ONE JSON line on stdout. Everything else (logging, errors)
	// goes to stderr so the shell can parse stdout cleanly.
	handshake := map[string]string{
		"addr":  ln.Addr().String(),
		"token": token,
	}
	if err := json.NewEncoder(os.Stdout).Encode(handshake); err != nil {
		return fmt.Errorf("write handshake: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if serveMetricsInterval > 0 {
		mp := server.NewMetricsProducer(srv, adapter, serveMetricsInterval)
		mp.Start(ctx)
	}

	// Alert evaluation loop: rules × live analytics → TriggerAlert through
	// the bridge above. Runs in its own goroutine and respects ctx
	// cancellation the same way MetricsProducer does.
	go runServeAlertEvalLoop(ctx, cm, alertMgr)

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		srv.Close()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = httpServer.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	}
}

// randomToken returns 32 hex chars (128 bits) of cryptographic randomness.
func randomToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// runServeAlertEvalLoop periodically evaluates the stored alert rules against
// live analytics and fires TriggerAlert through the given manager. The rule
// service and evaluator are built once so the per-rule cooldown state survives
// across cycles; rule changes are still picked up because AlertService re-reads
// .cosmoflare-alerts.yaml on every List. Any per-cycle error is logged and the
// cycle is skipped — the evaluator never fires on missing data.
func runServeAlertEvalLoop(ctx context.Context, cm *config.ConfigManager, alertMgr *webhook.Manager) {
	interval := alertEvalInterval
	if interval < 30*time.Second {
		interval = 30 * time.Second
	}

	rules, err := getAlertService()
	if err != nil {
		log.Printf("[alerts] rules service unavailable: %v", err)
		return
	}
	eval := webhook.NewEvaluator(rules, alertMgr, 0) // 0 → library default cooldown (15m)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runServeAlertEvalCycle(ctx, cm, eval, rules)
		}
	}
}

// runServeAlertEvalCycle runs one evaluation pass: list rules, resolve the
// current profile's credentials, collect 24h analytics, evaluate. Any failure
// short-circuits with a log line — no alerts fire on fabricated zeros.
func runServeAlertEvalCycle(ctx context.Context, cm *config.ConfigManager, eval *webhook.Evaluator, rules *cosmoflare.AlertService) {
	listed, err := rules.List()
	if err != nil {
		log.Printf("[alerts] list rules: %v", err)
		return
	}
	if len(listed) == 0 {
		return // nothing configured — skip the analytics round-trip entirely
	}

	p, err := cm.GetCurrent()
	if err != nil {
		log.Printf("[alerts] resolve profile: %v", err)
		return
	}
	analytics := cosmoflare.NewAnalyticsService(p.AccountID, p.APIToken)

	now := time.Now()
	w := cosmoflare.AnalyticsWindow{Start: now.Add(-24 * time.Hour), End: now}
	metrics, err := webhook.CollectEvalMetrics(ctx, analytics, w)
	if err != nil {
		log.Printf("[alerts] collect metrics: %v", err)
		return
	}

	limits := cosmoflare.NewLimitsServiceFromCreds(p.AccountID, p.APIToken)
	if err := webhook.CollectLimitMetrics(ctx, limits, &metrics); err != nil {
		// Limit collection failing must NOT skip the analytics-based rules.
		log.Printf("[alerts] collect limits: %v", err)
	}

	for _, name := range eval.Evaluate(metrics) {
		log.Printf("[alerts] rule %q fired", name)
	}
}

// newServeAlertBridge constructs the daemon's alert manager and bridges every
// TriggerAlert call onto the SSE hub's notifications channel (FEAT-008).
// Webhook delivery is unchanged — the notifier only adds in-process fan-out
// to connected /events clients.
func newServeAlertBridge(srv *server.Server) *webhook.Manager {
	m := webhook.NewManager(nil, "")
	m.SetNotifier(func(p *webhook.NotificationPayload) {
		srv.Publish(server.ChannelNotifications, p)
	})
	return m
}

// serveAdapter implements server.ServeSource over the existing per-service
// constructors. It owns NO Cloudflare logic — each method constructs a service
// from the resolved profile's credentials and calls its list method. Per-request
// account selection is read-only: the daemon re-resolves creds from the named
// profile (empty = current) on every call; there is no write-side "switch".
type serveAdapter struct {
	cm *config.ConfigManager
}

// resolveProfile returns the named profile, or the current profile when name is
// empty. A missing/invalid profile yields an error so the handler records
// cloudflare_online=false and surfaces the first-run setup screen.
func (a *serveAdapter) resolveProfile(name string) (*config.Profile, error) {
	if name == "" {
		return a.cm.GetCurrent()
	}
	return a.cm.GetProfile(name)
}

// CurrentProfileName returns the profile the daemon serves by default —
// the name resolveProfile("") resolves to, or "" when none is configured.
func (a *serveAdapter) CurrentProfileName() string {
	p, err := a.cm.GetCurrent()
	if err != nil || p == nil {
		return ""
	}
	return p.Name
}

// Accounts returns the local config profiles (NOT a Cloudflare API call).
func (a *serveAdapter) Accounts(_ context.Context) (any, error) {
	names := a.cm.ListProfiles()
	out := make([]map[string]string, 0, len(names))
	for _, n := range names {
		out = append(out, map[string]string{"name": n})
	}
	return out, nil
}

func (a *serveAdapter) Zones(ctx context.Context, profile string) (any, error) {
	p, err := a.resolveProfile(profile)
	if err != nil {
		return nil, err
	}
	svc, err := cosmoflare.NewZoneServiceFromCreds(p.AccountID, p.APIToken)
	if err != nil {
		return nil, err
	}
	return svc.List(ctx)
}

func (a *serveAdapter) R2Buckets(ctx context.Context, profile string) (any, error) {
	p, err := a.resolveProfile(profile)
	if err != nil {
		return nil, err
	}
	// R2 has no NewR2ServiceFromCreds — it uses the NewClient option pattern.
	client, err := cosmoflare.NewClient(
		cosmoflare.WithAccountID(p.AccountID),
		cosmoflare.WithAPIToken(p.APIToken),
	)
	if err != nil {
		return nil, err
	}
	return client.ListBuckets(ctx)
}

func (a *serveAdapter) Workers(ctx context.Context, profile string) (any, error) {
	p, err := a.resolveProfile(profile)
	if err != nil {
		return nil, err
	}
	svc, err := cosmoflare.NewWorkerServiceFromCreds(p.AccountID, p.APIToken)
	if err != nil {
		return nil, err
	}
	return svc.List(ctx)
}

func (a *serveAdapter) KV(ctx context.Context, profile string) (any, error) {
	p, err := a.resolveProfile(profile)
	if err != nil {
		return nil, err
	}
	svc, err := cosmoflare.NewKVServiceFromCreds(p.AccountID, p.APIToken)
	if err != nil {
		return nil, err
	}
	return svc.ListNamespaces(ctx)
}
