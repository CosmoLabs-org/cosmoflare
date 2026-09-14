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
		return fmt.Errorf("failed to create Pages service: %w", err)
	}

	domains, err := svc.ListDomains(context.Background(), project)
	if err != nil {
		return outErr("failed to list domains", err)
	}

	if JSONOutput {
		return printJSON(domains)
	}

	if len(domains) == 0 {
		printInfo("No custom domains found for project '%s'", project)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tVERIFICATION\tVALIDATION")
	for _, d := range domains {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", d.Name, d.Status, d.VerificationStatus, d.ValidationStatus)
	}
	w.Flush()
	printInfo("Total: %d domain(s)", len(domains))
	return nil
}

func runPagesDomainAttach(cmd *cobra.Command, args []string) error {
	project := args[0]
	domain := args[1]

	svc, err := getPagesService()
	if err != nil {
		return fmt.Errorf("failed to create Pages service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would attach domain", map[string]string{
				"project": project,
				"domain":  domain,
			})
		}
		printInfo("DRY RUN: Would attach domain '%s' to project '%s'", domain, project)
		return nil
	}

	result, err := svc.AttachDomain(context.Background(), project, domain)
	if err != nil {
		return outErr("failed to attach domain", err)
	}

	if JSONOutput {
		return printSuccessJSON("Domain attached successfully", result)
	}
	printSuccess("Domain '%s' attached to project '%s' (status: %s, validation: %s)", result.Name, project, result.Status, result.ValidationStatus)
	return nil
}

func runPagesDomainDetach(cmd *cobra.Command, args []string) error {
	project := args[0]
	domain := args[1]

	svc, err := getPagesService()
	if err != nil {
		return fmt.Errorf("failed to create Pages service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would detach domain", map[string]string{
				"project": project,
				"domain":  domain,
			})
		}
		printInfo("DRY RUN: Would detach domain '%s' from project '%s'", domain, project)
		return nil
	}

	if err := svc.DetachDomain(context.Background(), project, domain); err != nil {
		return outErr("failed to detach domain", err)
	}

	if JSONOutput {
		return printSuccessJSON("Domain detached successfully", map[string]string{
			"project": project,
			"domain":  domain,
		})
	}
	printSuccess("Domain '%s' detached from project '%s'", domain, project)
	return nil
}
