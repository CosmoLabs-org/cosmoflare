package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var (
	templatesForce bool
)

var templatesCmd = &cobra.Command{
	Use:   "templates",
	Short: "Scaffold Cloudflare projects from built-in templates",
	Long: `Template library for scaffolding complete Cloudflare projects.

All templates are embedded in the binary — no network access required.

Commands:
  list      List all available templates
  info      Show detailed information about a template
  create    Scaffold a new project from a template

Available templates:
  api        Worker API with KV bindings, CORS, error handling
  static     Pages static site with build pipeline
  fullstack  Worker API + Pages frontend + R2 storage + D1 database
  cron       Scheduled Worker with KV state persistence
  queue      Queue producer + consumer Workers

Examples:
  cosmoflare templates list
  cosmoflare templates list --json
  cosmoflare templates info api
  cosmoflare templates create api my-project
  cosmoflare templates create fullstack ./my-app --force`,
}

var templatesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available project templates",
	Long: `List all built-in project templates with their descriptions and services.

Examples:
  cosmoflare templates list
  cosmoflare templates list --json`,
	Args: cobra.NoArgs,
	RunE: runTemplatesList,
}

var templatesInfoCmd = &cobra.Command{
	Use:   "info [template-name]",
	Short: "Show detailed information about a template",
	Long: `Display full details of a template including description, services used,
and files that will be generated.

Examples:
  cosmoflare templates info api
  cosmoflare templates info fullstack
  cosmoflare templates info cron --json`,
	RunE: runTemplatesInfo,
}

var templatesCreateCmd = &cobra.Command{
	Use:   "create [template-name] [directory]",
	Short: "Scaffold a new project from a template",
	Long: `Create a new Cloudflare project by scaffolding files from a built-in template.

The directory defaults to the current directory if not specified.
The project name is derived from the directory name.

Use --force to overwrite existing files.

Examples:
  cosmoflare templates create api
  cosmoflare templates create api my-api-project
  cosmoflare templates create fullstack ./my-app
  cosmoflare templates create static . --force
  cosmoflare templates create cron my-cron --json`,
	RunE: runTemplatesCreate,
}

func init() {
	rootCmd.AddCommand(templatesCmd)

	templatesCmd.AddCommand(templatesListCmd)
	templatesCmd.AddCommand(templatesInfoCmd)
	templatesCmd.AddCommand(templatesCreateCmd)

	templatesCreateCmd.Flags().BoolVar(&templatesForce, "force", false, "Overwrite existing files")
}

func runTemplatesList(cmd *cobra.Command, args []string) error {
	svc := cosmoflare.NewTemplateService()
	templates := svc.ListTemplates()

	if JSONOutput {
		return printJSON(templates)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDESCRIPTION\tSERVICES")
	for _, t := range templates {
		fmt.Fprintf(w, "%s\t%s\t%s\n", t.Name, t.Description, strings.Join(t.Services, ", "))
	}
	w.Flush()

	fmt.Fprintf(cmd.OutOrStdout(), "\nUse 'cosmoflare templates info <name>' for details.\n")
	return nil
}

func runTemplatesInfo(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("template name is required\n\nUsage: cosmoflare templates info <name>\n\nAvailable: api, static, fullstack, cron, queue")
	}

	svc := cosmoflare.NewTemplateService()
	tmpl, err := svc.GetTemplate(args[0])
	if err != nil {
		return err
	}

	if JSONOutput {
		return printJSON(tmpl)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Template: %s\n", tmpl.Name)
	fmt.Fprintf(out, "Description: %s\n", tmpl.Description)
	fmt.Fprintf(out, "\nServices:\n")
	for _, s := range tmpl.Services {
		fmt.Fprintf(out, "  - %s\n", s)
	}
	fmt.Fprintf(out, "\nFiles generated:\n")
	for _, f := range tmpl.Files {
		fmt.Fprintf(out, "  - %s\n", f)
	}
	fmt.Fprintf(out, "\nUsage: cosmoflare templates create %s [directory]\n", tmpl.Name)
	return nil
}

func runTemplatesCreate(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("template name is required\n\nUsage: cosmoflare templates create <name> [directory]\n\nAvailable: api, static, fullstack, cron, queue")
	}

	templateName := args[0]

	// Determine target directory
	dir := "."
	if len(args) >= 2 {
		dir = args[1]
	}

	// Resolve to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	// Ensure target directory exists
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", absDir, err)
	}

	projectName := filepath.Base(absDir)

	svc := cosmoflare.NewTemplateService()
	result, err := svc.CreateFromTemplate(templateName, absDir, projectName, templatesForce)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create project: %v", err))
		}
		return err
	}

	if JSONOutput {
		return printJSON(result)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Project %q created from template %q\n", result.ProjectName, result.TemplateName)
	fmt.Fprintf(out, "Directory: %s\n", result.Directory)
	fmt.Fprintf(out, "\nFiles created:\n")
	for _, f := range result.FilesCreated {
		fmt.Fprintf(out, "  - %s\n", f)
	}
	fmt.Fprintf(out, "\nNext steps:\n")
	fmt.Fprintf(out, "  cd %s\n", result.Directory)
	fmt.Fprintf(out, "  cosmoflare doctor %s   # verify domain health\n", result.ProjectName)
	return nil
}
