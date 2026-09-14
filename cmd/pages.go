package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
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
  env           Manage environment variables and secrets
  domain        Manage custom domains
  deployment    View, retry, and inspect logs for deployments

Examples:
  cosmoflare pages create my-site --branch main
  cosmoflare pages list --json
  cosmoflare pages get my-site
  cosmoflare pages deployments my-site --json
  cosmoflare pages env list my-site --env production
  cosmoflare pages domain attach my-site example.com
  cosmoflare pages deployment logs my-site dep-abc123`,
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
		return outErr("failed to create Pages service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would create project", func() any {
			return map[string]string{
				"name":   name,
				"branch": pagesBranch,
			}
		}, func() {
			printInfo("DRY RUN: Would create project '%s' with production branch '%s'", name, pagesBranch)
		})
	}

	project, err := svc.Create(context.Background(), name, pagesBranch)
	if err != nil {
		return outErr("failed to create project", err)
	}

	return outPayload("Project created successfully", func() any {
		return project
	}, func() {
		printSuccess("Project '%s' created (ID: %s, subdomain: %s)", project.Name, project.ID, project.SubDomain)
	})
}

func runPagesList(cmd *cobra.Command, args []string) error {
	svc, err := getPagesService()
	if err != nil {
		return outErr("failed to create Pages service", err)
	}

	projects, err := svc.List(context.Background())
	if err != nil {
		return outErr("failed to list projects", err)
	}

	return outResult(projects, func() {
		if len(projects) == 0 {
			printInfo("No Pages projects found")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tSUBDOMAIN\tBRANCH\tDOMAINS")
		for _, p := range projects {
			domains := strings.Join(p.Domains, ", ")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", p.ID, p.Name, p.SubDomain, p.ProductionBranch, domains)
		}
		w.Flush()
		printInfo("Total: %d project(s)", len(projects))
	})
}

func runPagesGet(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	svc, err := getPagesService()
	if err != nil {
		return outErr("failed to create Pages service", err)
	}

	project, err := svc.Get(context.Background(), projectName)
	if err != nil {
		return outErr("failed to get project", err)
	}

	return outResult(project, func() {
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
	})
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
		return outErr("failed to create Pages service", err)
	}

	if DryRun {
		return outPayload("DRY RUN: Would delete project", func() any {
			return map[string]string{"name": projectName}
		}, func() {
			printInfo("DRY RUN: Would delete project '%s'", projectName)
		})
	}

	if err := svc.Delete(context.Background(), projectName); err != nil {
		return outErr("failed to delete project", err)
	}

	return outPayload("Project deleted successfully", func() any {
		return map[string]string{"name": projectName}
	}, func() {
		printSuccess("Project '%s' deleted successfully!", projectName)
	})
}

func runPagesDeployments(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	svc, err := getPagesService()
	if err != nil {
		return outErr("failed to create Pages service", err)
	}

	deployments, err := svc.ListDeployments(context.Background(), projectName)
	if err != nil {
		return outErr("failed to list deployments", err)
	}

	return outResult(deployments, func() {
		if len(deployments) == 0 {
			printInfo("No deployments found for project '%s'", projectName)
			return
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
	})
}
