package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var pagesDomainCmd = &cobra.Command{
	Use:   "domain",
	Short: "Manage Pages custom domains",
	Long: `Manage custom domains attached to a Pages project.

Commands:
  list          List custom domains for a project
  attach        Attach a custom domain to a project
  detach        Detach a custom domain from a project

Examples:
  cosmoflare pages domain list my-site
  cosmoflare pages domain attach my-site example.com
  cosmoflare pages domain detach my-site example.com`,
}

var pagesDomainListCmd = &cobra.Command{
	Use:   "list [project]",
	Short: "List custom domains for a Pages project",
	Long: `List all custom domains attached to a Pages project.

Examples:
  cosmoflare pages domain list my-site
  cosmoflare pages domain list my-site --json`,
	Args: cobra.ExactArgs(1),
	RunE: runPagesDomainList,
}

var pagesDomainAttachCmd = &cobra.Command{
	Use:   "attach [project] [domain]",
	Short: "Attach a custom domain to a Pages project",
	Long: `Attach a custom domain to a Pages project. The domain must already be
present in your Cloudflare account as a zone. The returned status
reflects Cloudflare's ownership validation state; the domain may not
serve traffic until validation completes.

Examples:
  cosmoflare pages domain attach my-site example.com`,
	Args: cobra.ExactArgs(2),
	RunE: runPagesDomainAttach,
}

var pagesDomainDetachCmd = &cobra.Command{
	Use:   "detach [project] [domain]",
	Short: "Detach a custom domain from a Pages project",
	Long: `Detach a custom domain from a Pages project.

Examples:
  cosmoflare pages domain detach my-site example.com`,
	Args: cobra.ExactArgs(2),
	RunE: runPagesDomainDetach,
}

func init() {
	pagesCmd.AddCommand(pagesDomainCmd)
	pagesDomainCmd.AddCommand(pagesDomainListCmd)
	pagesDomainCmd.AddCommand(pagesDomainAttachCmd)
	pagesDomainCmd.AddCommand(pagesDomainDetachCmd)
}

func runPagesDomainList(cmd *cobra.Command, args []string) error {
	project := args[0]

	svc, err := getPagesService()
	if err != nil {
		return outErr("failed to create Pages service", err)
	}

	domains, err := svc.ListDomains(context.Background(), project)
	if err != nil {
		return outErr("failed to list domains", err)
	}

	return outResult(domains, func() {
		if len(domains) == 0 {
			printInfo("No custom domains found for project '%s'", project)
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tVERIFICATION\tVALIDATION")
		for _, d := range domains {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", d.Name, d.Status, d.VerificationStatus, d.ValidationStatus)
		}
		w.Flush()
		printInfo("Total: %d domain(s)", len(domains))
	})
}

func runPagesDomainAttach(cmd *cobra.Command, args []string) error {
	project := args[0]
	domain := args[1]

	svc, err := getPagesService()
	if err != nil {
		return outErr("failed to create Pages service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would attach domain", func() any {
			return map[string]string{
				"project": project,
				"domain":  domain,
			}
		}, func() {
			printInfo("DRY RUN: Would attach domain '%s' to project '%s'", domain, project)
		})
	}

	result, err := svc.AttachDomain(context.Background(), project, domain)
	if err != nil {
		return outErr("failed to attach domain", err)
	}

	return outPayload("Domain attached successfully", func() any {
		return result
	}, func() {
		printSuccess("Domain '%s' attached to project '%s' (status: %s, validation: %s)", result.Name, project, result.Status, result.ValidationStatus)
	})
}

func runPagesDomainDetach(cmd *cobra.Command, args []string) error {
	project := args[0]
	domain := args[1]

	svc, err := getPagesService()
	if err != nil {
		return outErr("failed to create Pages service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would detach domain", func() any {
			return map[string]string{
				"project": project,
				"domain":  domain,
			}
		}, func() {
			printInfo("DRY RUN: Would detach domain '%s' from project '%s'", domain, project)
		})
	}

	if err := svc.DetachDomain(context.Background(), project, domain); err != nil {
		return outErr("failed to detach domain", err)
	}

	return outPayload("Domain detached successfully", func() any {
		return map[string]string{
			"project": project,
			"domain":  domain,
		}
	}, func() {
		printSuccess("Domain '%s' detached from project '%s'", domain, project)
	})
}
