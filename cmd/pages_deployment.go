package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var pagesDeploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "Manage Pages deployment operations",
	Long: `View, retry, and inspect logs for Pages deployments.

Commands:
  view          View a single deployment
  retry         Retry a deployment (re-run its build)
  logs          View a deployment's build log

Examples:
  cosmoflare pages deployment view my-site dep-abc123
  cosmoflare pages deployment retry my-site dep-abc123
  cosmoflare pages deployment logs my-site dep-abc123`,
}

var pagesDeploymentViewCmd = &cobra.Command{
	Use:   "view [project] [deployment-id]",
	Short: "View a Pages deployment",
	Long: `Show details of a single Pages deployment.

Examples:
  cosmoflare pages deployment view my-site dep-abc123
  cosmoflare pages deployment view my-site dep-abc123 --json`,
	Args: cobra.ExactArgs(2),
	RunE: runPagesDeploymentView,
}

var pagesDeploymentRetryCmd = &cobra.Command{
	Use:   "retry [project] [deployment-id]",
	Short: "Retry a Pages deployment",
	Long: `Retry a Pages deployment. This triggers a new build using the same
source as the given deployment; it does not modify or remove the
original deployment.

Examples:
  cosmoflare pages deployment retry my-site dep-abc123`,
	Args: cobra.ExactArgs(2),
	RunE: runPagesDeploymentRetry,
}

var pagesDeploymentLogsCmd = &cobra.Command{
	Use:   "logs [project] [deployment-id]",
	Short: "View a Pages deployment's build log",
	Long: `Show the build log for a Pages deployment.

Examples:
  cosmoflare pages deployment logs my-site dep-abc123
  cosmoflare pages deployment logs my-site dep-abc123 --json`,
	Args: cobra.ExactArgs(2),
	RunE: runPagesDeploymentLogs,
}

func init() {
	pagesCmd.AddCommand(pagesDeploymentCmd)
	pagesDeploymentCmd.AddCommand(pagesDeploymentViewCmd)
	pagesDeploymentCmd.AddCommand(pagesDeploymentRetryCmd)
	pagesDeploymentCmd.AddCommand(pagesDeploymentLogsCmd)
}

func runPagesDeploymentView(cmd *cobra.Command, args []string) error {
	project := args[0]
	deploymentID := args[1]

	svc, err := getPagesService()
	if err != nil {
		return fmt.Errorf("failed to create Pages service: %w", err)
	}

	deployment, err := svc.GetDeployment(context.Background(), project, deploymentID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get deployment: %v", err))
		}
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	if JSONOutput {
		return printJSON(deployment)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "ID:\t%s\n", deployment.ID)
	fmt.Fprintf(w, "Project:\t%s\n", deployment.ProjectName)
	fmt.Fprintf(w, "Environment:\t%s\n", deployment.Environment)
	fmt.Fprintf(w, "URL:\t%s\n", deployment.URL)
	if deployment.CreatedOn != nil {
		fmt.Fprintf(w, "Created:\t%s\n", deployment.CreatedOn.Format("2006-01-02 15:04:05 UTC"))
	}
	if deployment.ModifiedOn != nil {
		fmt.Fprintf(w, "Modified:\t%s\n", deployment.ModifiedOn.Format("2006-01-02 15:04:05 UTC"))
	}
	w.Flush()
	return nil
}

func runPagesDeploymentRetry(cmd *cobra.Command, args []string) error {
	project := args[0]
	deploymentID := args[1]

	svc, err := getPagesService()
	if err != nil {
		return fmt.Errorf("failed to create Pages service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would retry deployment", map[string]string{
				"project":       project,
				"deployment_id": deploymentID,
			})
		}
		printInfo("DRY RUN: Would retry deployment '%s' for project '%s' (re-runs the build using the same source)", deploymentID, project)
		return nil
	}

	printInfo("Retrying deployment '%s': this triggers a new build using the same source", deploymentID)

	newDeployment, err := svc.RetryDeployment(context.Background(), project, deploymentID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to retry deployment: %v", err))
		}
		return fmt.Errorf("failed to retry deployment: %w", err)
	}

	if JSONOutput {
		return printSuccessJSON("Deployment retry started", newDeployment)
	}
	printSuccess("Retry started: new deployment '%s' for project '%s'", newDeployment.ID, project)
	return nil
}

func runPagesDeploymentLogs(cmd *cobra.Command, args []string) error {
	project := args[0]
	deploymentID := args[1]

	svc, err := getPagesService()
	if err != nil {
		return fmt.Errorf("failed to create Pages service: %w", err)
	}

	logs, err := svc.GetDeploymentLogs(context.Background(), project, deploymentID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to get deployment logs: %v", err))
		}
		return fmt.Errorf("failed to get deployment logs: %w", err)
	}

	if JSONOutput {
		return printJSON(logs)
	}

	if len(logs.Data) == 0 {
		printInfo("No log lines found for deployment '%s'", deploymentID)
		return nil
	}

	for _, line := range logs.Data {
		if line.Timestamp != nil {
			fmt.Printf("[%s] %s\n", line.Timestamp.Format("2006-01-02 15:04:05"), line.Line)
		} else {
			fmt.Println(line.Line)
		}
	}
	printInfo("Total: %d log line(s)", logs.Total)
	return nil
}
