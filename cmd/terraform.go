package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var (
	tfServices        []string
	tfFormat          string
	tfProviderVersion string
)

var terraformCmd = &cobra.Command{
	Use:   "terraform",
	Short: "Generate Terraform configurations from live Cloudflare state",
	Long: `Terraform integration for Cosmoflare.

Export your live Cloudflare infrastructure as Terraform .tf files,
or generate import blocks to adopt existing resources into Terraform state.

Commands:
  export        Export all resources as Terraform HCL configs
  import-block  Generate terraform import blocks for existing resources

Examples:
  cosmoflare terraform export
  cosmoflare terraform export ./infra --services workers,dns
  cosmoflare terraform export --format hcl --provider-version "~> 5.0"
  cosmoflare terraform import-block --json`,
}

var terraformExportCmd = &cobra.Command{
	Use:   "export [dir]",
	Short: "Export Cloudflare resources as Terraform configuration files",
	Long: `Export live Cloudflare resources as Terraform .tf files.

Generates provider.tf, workers.tf, dns.tf, r2.tf, kv.tf, zones.tf,
and import.tf in the target directory (default: ./terraform/).

Use --services to filter which resource types to export.
Supported services: workers, dns, r2, kv, zones.

Examples:
  cosmoflare terraform export                            # Export all to ./terraform/
  cosmoflare terraform export ./infra                    # Export all to ./infra/
  cosmoflare terraform export --services workers,r2      # Workers and R2 only
  cosmoflare terraform export --provider-version "~> 5.0"
  cosmoflare terraform export --json                     # JSON output`,
	RunE: runTerraformExport,
}

var terraformImportBlockCmd = &cobra.Command{
	Use:   "import-block",
	Short: "Generate terraform import blocks for existing resources",
	Long: `Generate Terraform import blocks for all discovered Cloudflare resources.

These blocks can be added to your Terraform configuration to adopt
existing infrastructure into Terraform state management.

Use --services to filter which resource types to include.

Examples:
  cosmoflare terraform import-block
  cosmoflare terraform import-block --services workers,dns
  cosmoflare terraform import-block --json`,
	RunE: runTerraformImportBlock,
}

func init() {
	rootCmd.AddCommand(terraformCmd)

	terraformCmd.AddCommand(terraformExportCmd)
	terraformCmd.AddCommand(terraformImportBlockCmd)

	terraformExportCmd.Flags().StringSliceVar(&tfServices, "services", []string{}, "Filter services to export (workers,dns,r2,kv,zones)")
	terraformExportCmd.Flags().StringVar(&tfFormat, "format", "hcl", "Output format (hcl or json)")
	terraformExportCmd.Flags().StringVar(&tfProviderVersion, "provider-version", "~> 4.0", "Cloudflare provider version constraint")

	terraformImportBlockCmd.Flags().StringSliceVar(&tfServices, "services", []string{}, "Filter services to include (workers,dns,r2,kv,zones)")
}

func getTerraformExporter() (*cosmoflare.TerraformExporter, error) {
	return cosmoflare.NewTerraformExporter(AccountID, APIToken)
}

func runTerraformExport(cmd *cobra.Command, args []string) error {
	outDir := "./terraform"
	if len(args) > 0 {
		outDir = args[0]
	}

	exporter, err := getTerraformExporter()
	if err != nil {
		return fmt.Errorf("failed to create terraform exporter: %w", err)
	}

	var opts []cosmoflare.TerraformExportOption
	if len(tfServices) > 0 {
		opts = append(opts, cosmoflare.WithTerraformServices(tfServices))
	}
	opts = append(opts, cosmoflare.WithTerraformProviderVersion(tfProviderVersion))
	opts = append(opts, cosmoflare.WithTerraformFormat(tfFormat))

	ctx := context.Background()

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("dry run: would export terraform configs", map[string]interface{}{
				"output_dir": outDir,
				"services":   tfServices,
				"format":     tfFormat,
			})
		}
		printInfo("Would export Terraform configs to %s", outDir)
		return nil
	}

	result, err := exporter.Export(ctx, opts...)
	if err != nil {
		return outErr("terraform export failed", err)
	}

	if JSONOutput {
		return printSuccessJSON("terraform export complete", result)
	}

	// Write files to disk
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for filename, content := range result.Files {
		path := filepath.Join(outDir, filename)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
		printSuccess("Wrote %s", getRelativePath(path))
	}

	printSuccess("Exported %d resources to %s", result.Summary.Total, outDir)
	if result.Summary.Workers > 0 {
		printInfo("  Workers:       %d", result.Summary.Workers)
	}
	if result.Summary.DNSRecords > 0 {
		printInfo("  DNS Records:   %d", result.Summary.DNSRecords)
	}
	if result.Summary.R2Buckets > 0 {
		printInfo("  R2 Buckets:    %d", result.Summary.R2Buckets)
	}
	if result.Summary.KVSpaces > 0 {
		printInfo("  KV Namespaces: %d", result.Summary.KVSpaces)
	}
	if result.Summary.Zones > 0 {
		printInfo("  Zones:         %d", result.Summary.Zones)
	}

	return nil
}

func runTerraformImportBlock(cmd *cobra.Command, args []string) error {
	exporter, err := getTerraformExporter()
	if err != nil {
		return fmt.Errorf("failed to create terraform exporter: %w", err)
	}

	var opts []cosmoflare.TerraformExportOption
	if len(tfServices) > 0 {
		opts = append(opts, cosmoflare.WithTerraformServices(tfServices))
	}

	ctx := context.Background()

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("dry run: would generate import blocks", map[string]interface{}{
				"services": tfServices,
			})
		}
		printInfo("Would generate terraform import blocks")
		return nil
	}

	imports, err := exporter.GenerateImportBlocks(ctx, opts...)
	if err != nil {
		return outErr("failed to generate import blocks", err)
	}

	if JSONOutput {
		return printSuccessJSON("import blocks generated", map[string]interface{}{
			"imports": imports,
			"count":   len(imports),
		})
	}

	if len(imports) == 0 {
		printInfo("No resources found to import")
		return nil
	}

	hcl := cosmoflare.GenerateImportHCL(imports)
	fmt.Print(hcl)

	printSuccess("Generated %d import blocks", len(imports))
	// Show per-service breakdown
	counts := map[string]int{}
	for _, imp := range imports {
		parts := strings.SplitN(imp.To, ".", 2)
		if len(parts) > 0 {
			counts[parts[0]]++
		}
	}
	for resType, count := range counts {
		printInfo("  %s: %d", resType, count)
	}

	return nil
}
