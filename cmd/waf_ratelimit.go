package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// wafRateLimitDefaultAction is what the contract fixes when --action is
// omitted. The legacy top-level `ratelimit create` hardcodes "block";
// `waf ratelimit` challenges instead so legit bursts get a JS challenge
// rather than a hard 403.
const wafRateLimitDefaultAction = "challenge"

// wafRateLimitMitigationTimeout is fixed at the Free-plan maximum (10s).
// Paid plans may raise it, but 10s is safe everywhere.
const wafRateLimitMitigationTimeout = 10

// wafRateLimitCharacteristics are the counting characteristics. cf.colo.id
// is mandatory (the service preflight rejects a payload without it);
// ip.src makes each source IP its own counter.
var wafRateLimitCharacteristics = []string{"cf.colo.id", "ip.src"}

var (
	wafRateLimitRequests int
	wafRateLimitPeriod   int
	wafRateLimitAction   string
)

var wafRateLimitCmd = &cobra.Command{
	Use:   "ratelimit [zone-id] [path]",
	Short: "Create a WAF rate-limiting rule for a URL path",
	Long: `Create a zone rate-limiting rule via the Rulesets http_ratelimit phase.

The rule matches requests whose URL path starts with PATH and counts them
per the characteristics (cf.colo.id + ip.src). Once a source exceeds
--requests within --period seconds, the action fires for the mitigation
timeout.

Rate limiting is governed by the Zone WAF Edit permission — there is no
separate "Rate Limiting" permission on modern accounts.

Actions:
  block      hard-block the request (403)
  challenge  issue a JS challenge (default when --action is omitted)

The zone argument is a zone ID, not a name.

Examples:
  cosmoflare waf ratelimit ZONE_ID /login --requests 30 --period 10
  cosmoflare waf ratelimit ZONE_ID /api/search --requests 100 --period 60 --action block
  cosmoflare waf ratelimit ZONE_ID /login --requests 30 --period 10 --action challenge --json
  cosmoflare waf ratelimit ZONE_ID /login --requests 30 --period 10 --dry-run`,
	Args: cobra.ExactArgs(2),
	RunE: runWAFRatelimit,
}

// validateWAFRatelimitSpec enforces the command contract before anything
// else runs, with agent-readable messages.
func validateWAFRatelimitSpec(path, action string, requests, period int) error {
	if len(path) == 0 || path[0] != '/' {
		return fmt.Errorf("path must be an absolute URL path starting with %q (got %q)", "/", path)
	}
	switch action {
	case "", "block", "challenge":
	default:
		return fmt.Errorf("invalid --action %q: must be %q or %q", action, "block", "challenge")
	}
	if requests <= 0 {
		return fmt.Errorf("--requests must be a positive integer (got %d)", requests)
	}
	if period <= 0 {
		return fmt.Errorf("--period must be a positive integer in seconds (got %d)", period)
	}
	return nil
}

// buildWAFRatelimitInput is the pure rule builder: PATH + flags in, the
// exact RateLimitCreateInput the service consumes out.
func buildWAFRatelimitInput(zoneID, path, action string, requests, period int) (cosmoflare.RateLimitCreateInput, error) {
	if err := validateWAFRatelimitSpec(path, action, requests, period); err != nil {
		return cosmoflare.RateLimitCreateInput{}, err
	}
	if action == "" {
		action = wafRateLimitDefaultAction
	}
	return cosmoflare.RateLimitCreateInput{
		ZoneID:            zoneID,
		Action:            action,
		Expression:        fmt.Sprintf("starts_with(http.request.uri.path, %q)", path),
		Description:       fmt.Sprintf("cosmoflare waf ratelimit: %d requests per %ds on %s", requests, period, path),
		RequestsPerPeriod: requests,
		Period:            period,
		MitigationTimeout: wafRateLimitMitigationTimeout,
		Characteristics:   wafRateLimitCharacteristics,
	}, nil
}

// wafRatelimitDryRunPayload is the full rule that would be created, in the
// stable shape both output modes print.
func wafRatelimitDryRunPayload(zoneID string, in cosmoflare.RateLimitCreateInput) map[string]any {
	return map[string]any{
		"zone_id":                    zoneID,
		"phase":                      "http_ratelimit",
		"action":                     in.Action,
		"expression":                 in.Expression,
		"requests_per_period":        in.RequestsPerPeriod,
		"period_seconds":             in.Period,
		"mitigation_timeout_seconds": in.MitigationTimeout,
		"characteristics":            in.Characteristics,
		"description":                in.Description,
	}
}

// runWAFRatelimit wires the builder to the existing RateLimitService.
func runWAFRatelimit(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("zone ID and path are required")
	}
	zoneID, path := args[0], args[1]

	in, err := buildWAFRatelimitInput(zoneID, path, wafRateLimitAction, wafRateLimitRequests, wafRateLimitPeriod)
	if err != nil {
		return err
	}

	if DryRun {
		payload := wafRatelimitDryRunPayload(zoneID, in)
		return outPayload("DRY RUN: Would create WAF rate-limiting rule", func() any {
			return payload
		}, func() {
			printInfo("DRY RUN: would create WAF rate-limiting rule on zone %s", zoneID)
			b, err := json.MarshalIndent(payload, "", "  ")
			if err != nil {
				printInfo("DRY RUN payload could not be marshalled: %v", err)
				return
			}
			fmt.Println(string(b))
		})
	}

	svc, err := cosmoflare.NewRateLimitServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return outErr("failed to create WAF rate-limit service", err)
	}
	rule, err := svc.Create(context.Background(), in)
	if err != nil {
		return outErr("failed to create WAF rate-limiting rule", err)
	}
	return outPayload("WAF rate-limiting rule created", func() any {
		return rule
	}, func() {
		printSuccess("Rate limit on %s: %d requests / %ds -> %s", path,
			rule.RequestsPerPeriod, rule.Period, rule.Action)
		printInfo("tune with --requests/--period; rule ID %s", rule.ID)
	})
}

func init() {
	wafCmd.AddCommand(wafRateLimitCmd)

	wafRateLimitCmd.Flags().IntVar(&wafRateLimitRequests, "requests", 10,
		"requests allowed per period")
	wafRateLimitCmd.Flags().IntVar(&wafRateLimitPeriod, "period", 10,
		"period in seconds (Free plan max 10)")
	wafRateLimitCmd.Flags().StringVar(&wafRateLimitAction, "action", "",
		`action when the threshold is exceeded: "block" or "challenge" (default "challenge")`)
}
