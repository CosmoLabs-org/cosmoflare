package cmd

import (
	"context"
	"fmt"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var domainsRedirectsCmd = &cobra.Command{
	Use:   "redirects",
	Short: "Show domains that have redirect rules configured",
	Long: `Fetch detail for every domain and list those that have one or more
Redirect Rules (Rulesets API, dynamic redirect phase) configured.

Examples:
  cosmoflare domains redirects
  cosmoflare domains redirects --json`,
	Args: cobra.NoArgs,
	RunE: runDomainsRedirects,
}

// domainRedirectsEntry is the JSON shape for a domain's redirect rules.
type domainRedirectsEntry struct {
	Name      string                    `json:"name"`
	ZoneID    string                    `json:"zone_id"`
	Redirects []cosmoflare.RedirectRule `json:"redirects"`
}

func runDomainsRedirects(cmd *cobra.Command, args []string) error {
	svc, err := newDomainService(true)
	if err != nil {
		return outErr("failed to create domain service", err)
	}

	ctx := context.Background()

	domains, _, err := svc.List(ctx, cosmoflare.DomainListOptions{})
	if err != nil {
		return outErr("failed to list domains", err)
	}

	entries := make([]domainRedirectsEntry, 0)
	for _, d := range domains {
		detail, err := svc.GetDetail(ctx, d.Zone.ID)
		if err != nil {
			// Skip domains we can't fetch detail for; redirects are unknown.
			continue
		}
		if len(detail.Redirects) == 0 {
			continue
		}
		entries = append(entries, domainRedirectsEntry{
			Name:      d.Zone.Name,
			ZoneID:    d.Zone.ID,
			Redirects: detail.Redirects,
		})
	}

	return outResult(entries, func() {
		if len(entries) == 0 {
			printInfo("No domains with redirect rules found")
			return
		}

		for i, e := range entries {
			if i > 0 {
				fmt.Println()
			}
			fmt.Printf("%s (%d redirect rule(s)):\n", e.Name, len(e.Redirects))
			for _, r := range e.Redirects {
				state := "enabled"
				if !r.Enabled {
					state = "disabled"
				}
				fmt.Printf("  - %s -> %s (%d, %s)\n", r.When, r.Destination, r.StatusCode, state)
			}
		}

		printInfo("Total: %d domain(s) with redirects", len(entries))
	})
}
