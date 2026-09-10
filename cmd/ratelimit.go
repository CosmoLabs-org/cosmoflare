package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
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
			if JSONOutput {
				return printErrorJSON(err.Error())
			}
			return err
		}
		if JSONOutput {
			return printSuccessJSON("rate-limiting rules listed", rules)
		}
		if len(rules) == 0 {
			fmt.Println("no rate-limiting rules (fresh zones have no entrypoint — this counts as zero)")
			return nil
		}
		for _, r := range rules {
			fmt.Printf("%s  %s  %d req/%ds block %ds  characteristics=%s\n",
				r.ID, r.Expression, r.RequestsPerPeriod, r.Period, r.MitigationTimeout,
				strings.Join(r.Characteristics, ","))
		}
		return nil
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
			if JSONOutput {
				return printErrorJSON(err.Error())
			}
			return err
		}
		if JSONOutput {
			return printSuccessJSON("rate-limiting rule created", rule)
		}
		fmt.Printf("created %s  %s  %d req/%ds block %ds\n",
			rule.ID, rule.Expression, rule.RequestsPerPeriod, rule.Period, rule.MitigationTimeout)
		return nil
	},
}

var ratelimitExpression string

// ratelimitServiceAndZone resolves the zone argument and builds the service.
// REUSE the existing resolveZoneID(ctx, domain) from cmd/bucket_domain.go:145
// (same `cmd` package — no import, no new helper). It delegates to
// ZoneService.ResolveIDForDomain and accepts a zone ID or name.
func ratelimitServiceAndZone(ctx context.Context, target string) (*cosmoflare.RateLimitService, string, error) {
	svc, err := cosmoflare.NewRateLimitServiceFromCreds(AccountID, APIToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create rate-limit service: %w", err)
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

	rateLimitCreateCmd.Flags().StringVar(&ratelimitExpression, "expression", "",
		"traffic expression, e.g. 'path eq \"/catalog.json\"' (required)")
	rateLimitCreateCmd.Flags().IntVar(&rateLimitRequests, "requests", 10, "requests per period")
	rateLimitCreateCmd.Flags().IntVar(&rateLimitPeriod, "period", 10, "period seconds (Free plan max 10)")
	rateLimitCreateCmd.Flags().IntVar(&rateLimitTimeout, "timeout", 10, "mitigation timeout seconds (Free plan max 10)")
	rateLimitCreateCmd.Flags().StringVar(&rateLimitChars, "characteristics", "cf.colo.id,ip.src",
		"comma-separated counting characteristics (must include cf.colo.id)")
	rateLimitCreateCmd.Flags().StringVar(&rateLimitDesc, "description", "", "rule description")
	_ = rateLimitCreateCmd.MarkFlagRequired("expression")
}
