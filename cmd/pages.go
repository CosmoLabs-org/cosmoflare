package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var pagesCmd = &cobra.Command{
	Use:   "pages",
	Short: "Manage Cloudflare Pages projects",
	Long: `Cloudflare Pages project and deployment management.

Commands:
  create        Create a Pages project
  list          List Pages projects
  get           Get Pages project details
  delete        Delete a Pages project
  deployments   List deployments for a project

Examples:
  cosmoflare pages create my-site --branch main
  cosmoflare pages list --json
  cosmoflare pages get my-site
  cosmoflare pages deployments my-site --json`,
}

var (
	pagesForce  bool
	pagesBranch string
)

var pagesCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a Pages project",
	Long: `Create a new Cloudflare Pages project.

The name must be unique within your account. A production branch is required.

Examples:
  cosmoflare pages create my-site --branch main
  cosmoflare pages create my-blog --branch master --json`,
	Args: cobra.ExactArgs(1),
	RunE: runPagesCreate,
}

var pagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all Pages projects",
	Long: `List all Cloudflare Pages projects in the current account.

Examples:
  cosmoflare pages list
  cosmoflare pages list --json`,
	RunE: runPagesList,
}

var pagesGetCmd = &cobra.Command{
	Use:   "get [project-name]",
	Short: "Get Pages project details",
	Long: `Get details of a Pages project by name.

Examples:
  cosmoflare pages get my-site
  cosmoflare pages get my-site --json`,
	Args: cobra.ExactArgs(1),
	RunE: runPagesGet,
}

var pagesDeleteCmd = &cobra.Command{
	Use:   "delete [project-name]",
	Short: "Delete a Pages project",
	Long: `Delete a Pages project and all its deployments.

WARNING: This action is irreversible. All deployments and data will be lost.

Examples:
  cosmoflare pages delete my-site
  cosmoflare pages delete my-site --force`,
	Args: cobra.ExactArgs(1),
	RunE: runPagesDelete,
}

var pagesDeploymentsCmd = &cobra.Command{
	Use:   "deployments [project-name]",
	Short: "List deployments for a Pages project",
	Long: `List all deployments for a Cloudflare Pages project.

Examples:
  cosmoflare pages deployments my-site
  cosmoflare pages deployments my-site --json`,
	Args: cobra.ExactArgs(1),
	RunE: runPagesDeployments,
}

func init() {
	rootCmd.AddCommand(pagesCmd)

	pagesCmd.AddCommand(pagesCreateCmd)
	pagesCmd.AddCommand(pagesListCmd)
	pagesCmd.AddCommand(pagesGetCmd)
	pagesCmd.AddCommand(pagesDeleteCmd)
	pagesCmd.AddCommand(pagesDeploymentsCmd)

	pagesCreateCmd.Flags().StringVar(&pagesBranch, "branch", "", "Production branch name (required)")
	_ = pagesCreateCmd.MarkFlagRequired("branch")

	pagesDeleteCmd.Flags().BoolVar(&pagesForce, "force", false, "Skip confirmation prompt")
}

func getPagesService() (*cosmoflare.PagesService, error) {
	return cosmoflare.NewPagesServiceFromCreds(AccountID, APIToken)
}

func runPagesCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	svc, err := getPagesService()
	if err != nil {
		return fmt.Errorf("failed to create Pages service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would create project", map[string]string{
				"name":   name,
				"branch": pagesBranch,
			})
		}
		printInfo("DRY RUN: Would create project '%s' with production branch '%s'", name, pagesBranch)
		return nil
	}

	project, err := svc.Create(context.Background(), name, pagesBranch)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create project: %v", err))
		}
		return fmt.Errorf("failed to create project: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Project created successfully", project)
	}
	printSuccess("Project '%s' created (ID: %s, subdomain: %s)", project.Name, project.ID, project.SubDomain)
	return nil
}

func runPagesList(cmd *cobra.Command, args []string) error {
	svc, err := getPagesService()
	if err != nil {
		return fmt.Errorf("failed to create Pages service: %w", err)
	}

	projects, err := svc.List(context.Background())
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list projects: %v", err))
		}
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if JSONOutput {
		return printJSON(projects)
	}

	if len(projects) == 0 {
		printInfo("No Pages projects found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSUBDOMAIN\tBRANCH\tDOMAINS")
	for _, p := range projects {
		domains := strings.Join(p.Domains, ", ")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", p.ID, p.Name, p.SubDomain, p.ProductionBranch, domains)
	}
	w.Flush()
	printInfo("Total: %d project(s)", len(projects))
	return nil
}

func runPagesGet(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	svc, err := getPagesService()
	if err != nil {
		return fmt.Errorf("failed to create Pages service: %w", err)
	}

	project, err := svc.Get(context.Background(), projectName)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get project: %v", err))
		}
		return fmt.Errorf("failed to get project: %w", err)
	}

	if JSONOutput {
		return printJSON(project)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID:\t%s\n", project.ID)
	fmt.Fprintf(w, "Name:\t%s\n", project.Name)
	fmt.Fprintf(w, "Subdomain:\t%s\n", project.SubDomain)
	fmt.Fprintf(w, "Branch:\t%s\n", project.ProductionBranch)
	fmt.Fprintf(w, "Domains:\t%s\n", strings.Join(project.Domains, ", "))
	if project.CreatedOn != nil {
		fmt.Fprintf(w, "Created:\t%s\n", project.CreatedOn.Format("2006-01-02 15:04:05 UTC"))
	}
	w.Flush()
	return nil
}

func runPagesDelete(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	if !pagesForce && !DryRun {
		fmt.Printf("Are you sure you want to delete project '%s' and all its deployments? [y/N]: ", projectName)
		var response string
		fmt.Scanln(&response)
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			printInfo("Project deletion cancelled")
			return nil
		}
	}

	svc, err := getPagesService()
	if err != nil {
		return fmt.Errorf("failed to create Pages service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete project", map[string]string{"name": projectName})
		}
		printInfo("DRY RUN: Would delete project '%s'", projectName)
		return nil
	}

	if err := svc.Delete(context.Background(), projectName); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to delete project: %v", err))
		}
		return fmt.Errorf("failed to delete project: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Project deleted successfully", map[string]string{"name": projectName})
	}
	printSuccess("Project '%s' deleted successfully!", projectName)
	return nil
}

func runPagesDeployments(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	svc, err := getPagesService()
	if err != nil {
		return fmt.Errorf("failed to create Pages service: %w", err)
	}

	deployments, err := svc.ListDeployments(context.Background(), projectName)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to list deployments: %v", err))
		}
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	if JSONOutput {
		return printJSON(deployments)
	}

	if len(deployments) == 0 {
		printInfo("No deployments found for project '%s'", projectName)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tENVIRONMENT\tURL\tCREATED")
	for _, d := range deployments {
		created := ""
		if d.CreatedOn != nil {
			created = d.CreatedOn.Format("2006-01-02 15:04:05")
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", d.ID, d.Environment, d.URL, created)
	}
	w.Flush()
	printInfo("Total: %d deployment(s)", len(deployments))
	return nil
}
