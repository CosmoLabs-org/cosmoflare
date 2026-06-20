package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var domainsNSCmd = &cobra.Command{
	Use:   "ns",
	Short: "Show nameserver status for every domain",
	Long: `List each domain alongside its NS status (cloudflare, external, mismatch).

Useful for spotting domains whose nameservers are not pointed at Cloudflare.

Examples:
  cosmoflare domains ns
  cosmoflare domains ns --json`,
	Args: cobra.NoArgs,
	RunE: runDomainsNS,
}

// domainNSEntry is the JSON shape for a single domain's NS status.
type domainNSEntry struct {
	Name     string `json:"name"`
	ZoneID   string `json:"zone_id"`
	NSStatus string `json:"ns_status"`
}

func runDomainsNS(cmd *cobra.Command, args []string) error {
	svc, err := newDomainService(false)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create domain service: %v", err))
		}
		return fmt.Errorf("failed to create domain service: %w", err)
	}

	ctx := context.Background()

	domains, _, err := svc.List(ctx, cosmoflare.DomainListOptions{})
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list domains: %v", err))
		}
		return fmt.Errorf("failed to list domains: %w", err)
	}

	if JSONOutput {
		entries := make([]domainNSEntry, 0, len(domains))
		for _, d := range domains {
			entries = append(entries, domainNSEntry{
				Name:     d.Zone.Name,
				ZoneID:   d.Zone.ID,
				NSStatus: d.NSStatus,
			})
		}
		return printJSON(entries)
	}

	if len(domains) == 0 {
		printInfo("No domains found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DOMAIN\tNS STATUS")
	for _, d := range domains {
		fmt.Fprintf(w, "%s\t%s\n", d.Zone.Name, d.NSStatus)
	}
	w.Flush()

	printInfo("Total: %d domain(s)", len(domains))
	return nil
}
