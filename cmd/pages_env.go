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

var pagesEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage Pages environment variables and secrets",
	Long: `Manage environment variables and secrets for a Pages project.

Commands:
  list          List environment variables for an environment
  set           Set one or more environment variables
  delete        Delete an environment variable

Examples:
  cosmoflare pages env list my-site --env production
  cosmoflare pages env set my-site API_URL=https://example.com --env production
  cosmoflare pages env delete my-site API_URL --env production`,
}

var pagesEnv string

var pagesEnvListCmd = &cobra.Command{
	Use:   "list [project]",
	Short: "List environment variables for a Pages project",
	Long: `List environment variables configured for a Pages project's production
or preview deployment environment. Secret values are never returned by
the Cloudflare API.

Examples:
  cosmoflare pages env list my-site --env production
  cosmoflare pages env list my-site --env preview --json`,
	Args: cobra.ExactArgs(1),
	RunE: runPagesEnvList,
}

var pagesEnvSet bool

var pagesEnvSetCmd = &cobra.Command{
	Use:   "set [project] KEY=VALUE...",
	Short: "Set one or more Pages environment variables",
	Long: `Set one or more environment variables for a Pages project's production
or preview deployment environment. Existing variables not listed are
left unchanged.

Examples:
  cosmoflare pages env set my-site API_URL=https://example.com --env production
  cosmoflare pages env set my-site TOKEN=abc123 --env production --secret`,
	Args: cobra.MinimumNArgs(2),
	RunE: runPagesEnvSet,
}

var pagesEnvDeleteCmd = &cobra.Command{
	Use:   "delete [project] KEY",
	Short: "Delete a Pages environment variable",
	Long: `Delete an environment variable from a Pages project's production or
preview deployment environment.

Examples:
  cosmoflare pages env delete my-site API_URL --env production`,
	Args: cobra.ExactArgs(2),
	RunE: runPagesEnvDelete,
}

func init() {
	pagesCmd.AddCommand(pagesEnvCmd)
	pagesEnvCmd.AddCommand(pagesEnvListCmd)
	pagesEnvCmd.AddCommand(pagesEnvSetCmd)
	pagesEnvCmd.AddCommand(pagesEnvDeleteCmd)

	pagesEnvListCmd.Flags().StringVar(&pagesEnv, "env", "production", "Deployment environment: production or preview")
	pagesEnvSetCmd.Flags().StringVar(&pagesEnv, "env", "production", "Deployment environment: production or preview")
	pagesEnvSetCmd.Flags().BoolVar(&pagesEnvSet, "secret", false, "Store the variable(s) as secrets (write-only)")
	pagesEnvDeleteCmd.Flags().StringVar(&pagesEnv, "env", "production", "Deployment environment: production or preview")
}

func validatePagesEnvFlag(env string) error {
	if env != "production" && env != "preview" {
		return fmt.Errorf("--env must be %q or %q, got %q", "production", "preview", env)
	}
	return nil
}

func runPagesEnvList(cmd *cobra.Command, args []string) error {
	project := args[0]

	if err := validatePagesEnvFlag(pagesEnv); err != nil {
		if JSONOutput {
			return printErrorJSON(err.Error())
		}
		return err
	}

	svc, err := getPagesService()
	if err != nil {
		return fmt.Errorf("failed to create Pages service: %w", err)
	}

	vars, err := svc.ListEnvVars(context.Background(), project, pagesEnv)
	if err != nil {
		return outErr("failed to list env vars", err)
	}

	if JSONOutput {
		return printJSON(vars)
	}

	if len(vars) == 0 {
		printInfo("No environment variables found for project '%s' (%s)", project, pagesEnv)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tTYPE\tVALUE")
	for _, v := range vars {
		value := v.Value
		if v.Type == "secret" {
			value = "(secret)"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", v.Key, v.Type, value)
	}
	w.Flush()
	printInfo("Total: %d variable(s)", len(vars))
	return nil
}

func runPagesEnvSet(cmd *cobra.Command, args []string) error {
	project := args[0]
	pairs := args[1:]

	if err := validatePagesEnvFlag(pagesEnv); err != nil {
		if JSONOutput {
			return printErrorJSON(err.Error())
		}
		return err
	}

	varType := "plain"
	if pagesEnvSet {
		varType = "secret"
	}

	vars := make([]cosmoflare.PagesEnvVar, 0, len(pairs))
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			err := fmt.Errorf("invalid KEY=VALUE pair: %q", pair)
			if JSONOutput {
				return printErrorJSON(err.Error())
			}
			return err
		}
		vars = append(vars, cosmoflare.PagesEnvVar{
			Key:   parts[0],
			Value: parts[1],
			Type:  varType,
		})
	}

	svc, err := getPagesService()
	if err != nil {
		return fmt.Errorf("failed to create Pages service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would set env vars", map[string]interface{}{
				"project": project,
				"env":     pagesEnv,
				"vars":    vars,
			})
		}
		printInfo("DRY RUN: Would set %d env var(s) for project '%s' (%s)", len(vars), project, pagesEnv)
		return nil
	}

	if err := svc.SetEnvVars(context.Background(), project, pagesEnv, vars); err != nil {
		return outErr("failed to set env vars", err)
	}

	if JSONOutput {
		return printSuccessJSON("Environment variables set successfully", map[string]interface{}{
			"project": project,
			"env":     pagesEnv,
			"count":   len(vars),
		})
	}
	printSuccess("Set %d environment variable(s) for project '%s' (%s)", len(vars), project, pagesEnv)
	return nil
}

func runPagesEnvDelete(cmd *cobra.Command, args []string) error {
	project := args[0]
	key := args[1]

	if err := validatePagesEnvFlag(pagesEnv); err != nil {
		if JSONOutput {
			return printErrorJSON(err.Error())
		}
		return err
	}

	svc, err := getPagesService()
	if err != nil {
		return fmt.Errorf("failed to create Pages service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would delete env var", map[string]string{
				"project": project,
				"env":     pagesEnv,
				"key":     key,
			})
		}
		printInfo("DRY RUN: Would delete env var '%s' for project '%s' (%s)", key, project, pagesEnv)
		return nil
	}

	if err := svc.DeleteEnvVar(context.Background(), project, pagesEnv, key); err != nil {
		return outErr("failed to delete env var", err)
	}

	if JSONOutput {
		return printSuccessJSON("Environment variable deleted successfully", map[string]string{
			"project": project,
			"env":     pagesEnv,
			"key":     key,
		})
	}
	printSuccess("Deleted environment variable '%s' for project '%s' (%s)", key, project, pagesEnv)
	return nil
}
