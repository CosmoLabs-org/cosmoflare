package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var domainsGetCmd = &cobra.Command{
	Use:   "get [name]",
	Short: "Show full detail for a single domain",
	Long: `Resolve a domain by name and print its extended detail card.

Looks the domain up by name (exact or pattern), then fetches the full
DomainDetail — name servers, record types, SSL mode/expiry, redirects,
and registrar overlay (when available).

Examples:
  cosmoflare domains get example.com
  cosmoflare domains get example.com --json`,
	Args: cobra.ExactArgs(1),
	RunE: runDomainsGet,
}

func runDomainsGet(cmd *cobra.Command, args []string) error {
	name := args[0]

	svc, err := newDomainService(true)
	if err != nil {
		return outErr("failed to create domain service", err)
	}

	ctx := context.Background()

	domains, _, err := svc.List(ctx, cosmoflare.DomainListOptions{Name: name})
	if err != nil {
		return outErr("failed to list domains", err)
	}

	if len(domains) == 0 {
		return outErrf("no domain found matching %q", name)
	}

	target := domains[0]
	detail, err := svc.GetDetail(ctx, target.Zone.ID)
	if err != nil {
		return outErr("failed to get domain detail", err)
	}

	return outResult(detail, func() {
		printDomainDetail(detail)
	})
}

// printDomainDetail renders a readable summary of a DomainDetail for humans.
func printDomainDetail(d *cosmoflare.DomainDetail) {
	fmt.Printf("Domain: %s\n", d.Zone.Name)
	fmt.Printf("Zone ID: %s\n", d.Zone.ID)
	fmt.Printf("DNS Status: %s\n", d.DNSStatus)
	fmt.Printf("SSL Status: %s\n", d.SSLStatus)
	fmt.Printf("Health: %s\n", d.HealthStatus)
	fmt.Printf("NS Status: %s\n", d.NSStatus)
	fmt.Printf("Records: %d\n", d.RecordCount)
	if d.SSLMode != "" {
		fmt.Printf("SSL Mode: %s\n", d.SSLMode)
	}
	if d.SSLExpiry != "" {
		fmt.Printf("SSL Expiry: %s\n", d.SSLExpiry)
	}
	if d.ResponseTime != "" {
		fmt.Printf("Response Time: %s\n", d.ResponseTime)
	}
	if len(d.NameServers) > 0 {
		fmt.Printf("Name Servers: %s\n", strings.Join(d.NameServers, ", "))
	}
	if len(d.RecordTypes) > 0 {
		parts := make([]string, 0, len(d.RecordTypes))
		for t, n := range d.RecordTypes {
			parts = append(parts, fmt.Sprintf("%s=%d", t, n))
		}
		fmt.Printf("Record Types: %s\n", strings.Join(parts, " "))
	}
	if len(d.Redirects) > 0 {
		fmt.Printf("Redirects: %d\n", len(d.Redirects))
		for _, r := range d.Redirects {
			fmt.Printf("  - %s -> %s (%d)\n", r.When, r.Destination, r.StatusCode)
		}
	}
	if d.Registrar != nil {
		fmt.Printf("Registrar: %s (%s)\n", d.Registrar.RegistrarName, d.Registrar.Registrar)
	}
}
