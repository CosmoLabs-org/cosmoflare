package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/cosmoflare"
)

var domainsCmd = &cobra.Command{
	Use:   "domains",
	Short: "List and inspect all domains in your Cloudflare account",
	Long: `Domain overview with health indicators for all zones.

Shows a compact table of all domains with DNS, SSL, and health status.
Use --detail for per-domain cards with extended information.
Use --enrich to run live health probes (HTTP, SSL) on each domain.

This command is designed for both human operators and AI agents:
- Default output is a compact, scannable table
- --json provides machine-readable output with pagination metadata
- --detail shows per-domain cards useful for debugging specific domains

Examples:
  cosmoflare domains                        # Overview table of all zones
  cosmoflare domains --detail               # Per-domain cards with full info
  cosmoflare domains --filter=active        # Only active zones
  cosmoflare domains --name="*.com"         # Filter by domain pattern
  cosmoflare domains --page=2 --per-page=25 # Pagination
  cosmoflare domains --sort=status          # Sort by zone status
  cosmoflare domains --enrich               # Add live health probes
  cosmoflare domains --json                 # Machine-readable output
  cosmoflare domains --json | jq '.domains[].zone.name'  # Extract domain names`,
	RunE: runDomains,
}

var (
	domainsFilter  string
	domainsName    string
	domainsPage    int
	domainsPerPage int
	domainsSort    string
	domainsDetail  bool
	domainsEnrich  bool
)

func init() {
	rootCmd.AddCommand(domainsCmd)

	domainsCmd.Flags().StringVar(&domainsFilter, "filter", "", "Filter by status: active, paused")
	domainsCmd.Flags().StringVar(&domainsName, "name", "", "Filter by domain name pattern (e.g., *.com)")
	domainsCmd.Flags().IntVar(&domainsPage, "page", 1, "Page number (default 1)")
	domainsCmd.Flags().IntVar(&domainsPerPage, "per-page", 50, "Results per page (default 50)")
	domainsCmd.Flags().StringVar(&domainsSort, "sort", "name", "Sort by: name, status, records")
	domainsCmd.Flags().BoolVar(&domainsDetail, "detail", false, "Show detailed per-domain cards")
	domainsCmd.Flags().BoolVar(&domainsEnrich, "enrich", false, "Run live health probes (slower)")
}

// domainsResponse is the JSON envelope for the domains command output.
type domainsResponse struct {
	Domains    []*cosmoflare.DomainStatus `json:"domains"`
	Pagination *cosmoflare.Pagination     `json:"pagination"`
}

func runDomains(cmd *cobra.Command, args []string) error {
	zoneSvc, err := getZoneService()
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create zone service: %v", err))
		}
		return fmt.Errorf("failed to create zone service: %w", err)
	}

	var doctor *cosmoflare.DoctorService
	if domainsEnrich {
		doctor = cosmoflare.NewDoctorService(10 * time.Second)
	}

	domainSvc, err := cosmoflare.NewDomainService(zoneSvc, nil, nil, doctor)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create domain service: %v", err))
		}
		return fmt.Errorf("failed to create domain service: %w", err)
	}

	opts := cosmoflare.DomainListOptions{
		Page:    domainsPage,
		PerPage: domainsPerPage,
		Filter:  domainsFilter,
		Name:    domainsName,
		Sort:    domainsSort,
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would list domains", opts)
		}
		printInfo("DRY RUN: Would list domains (page=%d, per-page=%d, filter=%q, name=%q, sort=%q)",
			opts.Page, opts.PerPage, opts.Filter, opts.Name, opts.Sort)
		return nil
	}

	ctx := context.Background()

	domains, pagination, err := domainSvc.List(ctx, opts)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list domains: %v", err))
		}
		return fmt.Errorf("failed to list domains: %w", err)
	}

	if domainsEnrich {
		domainSvc.EnrichWithHealth(ctx, domains)
	}

	// --detail mode: per-domain cards with extended info
	if domainsDetail {
		return runDomainsDetail(ctx, domainSvc, domains, pagination)
	}

	// --json mode: machine-readable output
	if JSONOutput {
		return printJSON(domainsResponse{
			Domains:    domains,
			Pagination: pagination,
		})
	}

	// Default: compact table output
	if len(domains) == 0 {
		printInfo("No domains found")
		return nil
	}

	fmt.Print(cosmoflare.FormatDomainTable(domains))
	printInfo("Page %d/%d (%d total)", pagination.Page, pagination.TotalPages, pagination.Total)

	return nil
}

func runDomainsDetail(ctx context.Context, svc *cosmoflare.DomainService, domains []*cosmoflare.DomainStatus, pagination *cosmoflare.Pagination) error {
	if len(domains) == 0 {
		if JSONOutput {
			return printJSON(domainsResponse{
				Domains:    domains,
				Pagination: pagination,
			})
		}
		printInfo("No domains found")
		return nil
	}

	details := make([]*cosmoflare.DomainDetail, 0, len(domains))
	for _, d := range domains {
		detail, err := svc.GetDetail(ctx, d.Zone.ID)
		if err != nil {
			// On detail failure, construct a minimal detail from the list data
			detail = &cosmoflare.DomainDetail{
				DomainStatus: *d,
				NameServers:  d.Zone.NameServers,
				RecordTypes:  make(map[string]int),
			}
		}
		details = append(details, detail)
	}

	if JSONOutput {
		type detailResponse struct {
			Domains    []*cosmoflare.DomainDetail `json:"domains"`
			Pagination *cosmoflare.Pagination     `json:"pagination"`
		}
		return printJSON(detailResponse{
			Domains:    details,
			Pagination: pagination,
		})
	}

	for i, detail := range details {
		if i > 0 {
			fmt.Println()
		}
		fmt.Print(cosmoflare.FormatDomainDetail(detail))
	}
	printInfo("Page %d/%d (%d total)", pagination.Page, pagination.TotalPages, pagination.Total)

	return nil
}
