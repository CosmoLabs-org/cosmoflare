package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/CosmoLabs-org/cosmoflare/internal/tui"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var domainsTUICmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive domain browser",
	Long: `Open a full-screen, split-pane browser over the account's domains.

Left pane lists every domain with its nameserver status; the right pane shows
detail for the selected domain (nameservers, SSL, registrar overlay, and
redirect rules). Press 'a' to quick-add a redirect on the selected domain,
arrow keys (or j/k) to navigate, and 'q' to quit.

Examples:
  cosmoflare domains tui`,
	Args: cobra.NoArgs,
	RunE: runDomainsTUI,
}

func runDomainsTUI(cmd *cobra.Command, args []string) error {
	svc, err := newDomainService(true)
	if err != nil {
		return err
	}

	domains, _, err := svc.List(context.Background(), cosmoflare.DomainListOptions{PerPage: 1000})
	if err != nil {
		return err
	}

	// Quick-add creates a catch-all redirect (status 301) to the destination.
	// Fine-grained match expressions are available via `cosmoflare redirects create`.
	createRedirect := func(zoneID, destination string) error {
		rs, err := cosmoflare.NewRedirectServiceFromCreds(AccountID, APIToken)
		if err != nil {
			return err
		}
		_, err = rs.Create(context.Background(), cosmoflare.RedirectRuleInput{
			ZoneID:      zoneID,
			When:        "true",
			Destination: destination,
			StatusCode:  301,
		})
		return err
	}

	return tui.RunDomainBrowser(domains, createRedirect)
}

func init() {
	domainsCmd.AddCommand(domainsTUICmd)
}
