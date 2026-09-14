package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	knowledge "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare/knowledge"
)

var rateLimitCmd = &cobra.Command{
	Use:   "ratelimit",
	Short: "Manage zone rate-limiting rules (Rulesets http_ratelimit phase)",
	Long: `Manage zone rate-limiting rules via the Rulesets http_ratelimit phase.

Create runs client-side preflight first: plan caps (Free: 1 rule/zone, 10s
window, 10s mitigation, IP-only counting) and the mandatory cf.colo.id
characteristic — violations stop before any API call.`,
}

var rateLimitListCmd = &cobra.Command{
	Use:   "list <zone-id-or-name>",
	Short: "List a zone's rate-limiting rules",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, zoneID, err := ratelimitServiceAndZone(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		rules, err := svc.List(cmd.Context(), zoneID)
		if err != nil {
			return outErr("failed to list rate-limiting rules", err)
		}
		return outPayload("rate-limiting rules listed", func() any {
			return rules
		}, func() {
			if len(rules) == 0 {
				fmt.Println("no rate-limiting rules (fresh zones have no entrypoint — this counts as zero)")
				return
			}
			for _, r := range rules {
				fmt.Printf("%s  %s  %d req/%ds block %ds  characteristics=%s\n",
					r.ID, r.Expression, r.RequestsPerPeriod, r.Period, r.MitigationTimeout,
					strings.Join(r.Characteristics, ","))
			}
		})
	},
}

var (
	rateLimitRequests int
	rateLimitPeriod   int
	rateLimitTimeout  int
	rateLimitChars    string
	rateLimitDesc     string
)

var rateLimitCreateCmd = &cobra.Command{
	Use:   "create <zone-id-or-name>",
	Short: "Create a rate-limiting rule (preflight-validated)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, zoneID, err := ratelimitServiceAndZone(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		var chars []string
		for _, c := range strings.Split(rateLimitChars, ",") {
			if c = strings.TrimSpace(c); c != "" {
				chars = append(chars, c)
			}
		}
		rule, err := svc.Create(cmd.Context(), cosmoflare.RateLimitCreateInput{
			ZoneID:            zoneID,
			Expression:        ratelimitExpression,
			Description:       rateLimitDesc,
			RequestsPerPeriod: rateLimitRequests,
			Period:            rateLimitPeriod,
			MitigationTimeout: rateLimitTimeout,
			Characteristics:   chars,
		})
		if err != nil {
			return outErr("failed to create rate-limiting rule", err)
		}
		if err := outPayload("rate-limiting rule created", func() any {
			return rule
		}, func() {
			fmt.Printf("created %s  %s  %d req/%ds block %ds\n",
				rule.ID, rule.Expression, rule.RequestsPerPeriod, rule.Period, rule.MitigationTimeout)
			if adv := probeAdvisory(); adv != "" {
				fmt.Println(adv)
			}
		}); err != nil {
			return err
		}
		if probeOnCreate {
			path := cosmoflare.ExpressionPath(rule.Expression)
			if path == "" {
				fmt.Fprintln(os.Stderr, "probe skipped: expression has no literal path — run `cosmoflare ratelimit probe <zone> --path <p>`")
				return nil
			}
			host, err := zoneHostname(cmd.Context(), args[0], zoneID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "probe skipped: %v\n", err)
				return nil
			}
			burst := cosmoflare.DefaultBurst(*rule)
			res := cosmoflare.NewRateLimitProber().Probe(cmd.Context(), "https://"+host+path, burst)
			if err := reportProbe(res); err != nil {
				return err
			}
			os.Exit(probeExitCode(res.Verdict))
		}
		return nil
	},
}

// reportProbe prints one probe result in the active output mode — the
// single rendering shared by `ratelimit probe` and `create --probe`.
func reportProbe(res cosmoflare.RateLimitProbeResult) error {
	return outResult(res, func() {
		fmt.Printf("probe   %s\nsent    %d requests\nverdict %s\n        %s\n",
			res.URL, res.Requests, res.Verdict, res.Explanation)
	})
}

var ratelimitExpression string

// ratelimitServiceAndZone resolves the zone argument and builds the service.
// REUSE the existing resolveZoneID(ctx, domain) from cmd/bucket_domain.go:145
// (same `cmd` package — no import, no new helper). It delegates to
// ZoneService.ResolveIDForDomain and accepts a zone ID or name.
func ratelimitServiceAndZone(ctx context.Context, target string) (*cosmoflare.RateLimitService, string, error) {
	svc, err := cosmoflare.NewRateLimitServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return nil, "", outErr("failed to create rate-limit service", err)
	}
	zoneID, err := resolveZoneID(ctx, target)
	if err != nil {
		return nil, "", err
	}
	return svc, zoneID, nil
}

func init() {
	rootCmd.AddCommand(rateLimitCmd)
	rateLimitCmd.AddCommand(rateLimitListCmd)
	rateLimitCmd.AddCommand(rateLimitCreateCmd)
	rateLimitCmd.AddCommand(rateLimitProbeCmd)

	rateLimitCreateCmd.Flags().StringVar(&ratelimitExpression, "expression", "",
		"traffic expression, e.g. 'path eq \"/catalog.json\"' (required)")
	rateLimitCreateCmd.Flags().IntVar(&rateLimitRequests, "requests", 10, "requests per period")
	rateLimitCreateCmd.Flags().IntVar(&rateLimitPeriod, "period", 10, "period seconds (Free plan max 10)")
	rateLimitCreateCmd.Flags().IntVar(&rateLimitTimeout, "timeout", 10, "mitigation timeout seconds (Free plan max 10)")
	rateLimitCreateCmd.Flags().StringVar(&rateLimitChars, "characteristics", "cf.colo.id,ip.src",
		"comma-separated counting characteristics (must include cf.colo.id)")
	rateLimitCreateCmd.Flags().StringVar(&rateLimitDesc, "description", "", "rule description")
	_ = rateLimitCreateCmd.MarkFlagRequired("expression")

	rateLimitProbeCmd.Flags().StringVar(&probePath, "path", "", "URL path the rule matches (required)")
	rateLimitProbeCmd.Flags().IntVar(&probeRequests, "requests", 0, "burst size (default: 2x the matching rule's requests per period, capped at 60)")
	rateLimitCreateCmd.Flags().BoolVar(&probeOnCreate, "probe", false, "run a live trip probe against the created rule immediately after creation")
}

// probeExitCode maps a probe verdict to a scriptable exit code:
// 0 tripped, 2 not-counted, 3 inconclusive.
func probeExitCode(verdict string) int {
	switch verdict {
	case cosmoflare.VerdictTripped:
		return 0
	case cosmoflare.VerdictNotCounted:
		return 2
	default:
		return 3
	}
}

// probeAdvisory renders the post-create advisory from pack data — never
// hardcoded. Empty when the pack declares no skipped classes.
func probeAdvisory() string {
	classes := knowledge.SkippedClassesSummary(cosmoflare.KnowledgeProductRateLimit)
	if classes == "" {
		return ""
	}
	return "note: WAF rate limiting does not count: " + classes +
		" — run `cosmoflare ratelimit probe` to verify this rule sees its traffic"
}

// zoneHostname resolves the probe target host: the argument itself when it
// looks like a hostname, else the zone's registered name.
func zoneHostname(ctx context.Context, target, zoneID string) (string, error) {
	if strings.Contains(target, ".") {
		return target, nil
	}
	zones, err := getZoneService()
	if err != nil {
		return "", err
	}
	zone, err := zones.Get(ctx, zoneID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch zone name for probe target: %w", err)
	}
	return zone.Name, nil
}

// probeBurstDefault derives the burst size from the live rule matching the
// path (RuleMatchesPath: literal-path equality first, substring fallback;
// first match wins — the Free plan caps at one rule anyway). Zero matches
// is an error.
func probeBurstDefault(ctx context.Context, svc *cosmoflare.RateLimitService, zoneID, path string) (int, error) {
	rules, err := svc.List(ctx, zoneID)
	if err != nil {
		return 0, err
	}
	for _, r := range rules {
		if cosmoflare.RuleMatchesPath(r, path) {
			return cosmoflare.DefaultBurst(r), nil
		}
	}
	return 0, fmt.Errorf("no rate-limiting rule matches path %q on this zone — create one first or pass --requests", path)
}

var (
	probePath     string
	probeRequests int
	probeOnCreate bool
)

var rateLimitProbeCmd = &cobra.Command{
	Use:   "probe <zone-id-or-name>",
	Short: "Burst-probe whether a rate-limiting rule actually sees the zone's traffic",
	Long: `Send a bounded burst of live GETs (opt-in, cap 60, half cache-busted) at the
zone and report whether the rule trips.

Verdicts and exit codes:
  tripped        exit 0 — the rule sees this traffic
  not-counted    exit 2 — rule live but the traffic class is likely skipped
  inconclusive   exit 3 — network errors dominated

This command sends live traffic to the target origin. It never runs unless
explicitly invoked.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if probePath == "" {
			return fmt.Errorf("--path is required (the URL path the rule matches)")
		}
		svc, zoneID, err := ratelimitServiceAndZone(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		burst := probeRequests
		if burst == 0 {
			burst, err = probeBurstDefault(cmd.Context(), svc, zoneID, probePath)
			if err != nil {
				return err
			}
		}
		if burst > cosmoflare.MaxProbeRequests {
			fmt.Fprintf(os.Stderr, "note: --requests clamped to %d\n", cosmoflare.MaxProbeRequests)
			burst = cosmoflare.MaxProbeRequests
		}
		host, err := zoneHostname(cmd.Context(), args[0], zoneID)
		if err != nil {
			return err
		}
		res := cosmoflare.NewRateLimitProber().Probe(cmd.Context(), "https://"+host+probePath, burst)
		if err := reportProbe(res); err != nil {
			return err
		}
		os.Exit(probeExitCode(res.Verdict))
		return nil
	},
}
