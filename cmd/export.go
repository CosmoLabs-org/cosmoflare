package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var (
	exportServices string
	exportFormat   string
)

var exportCmd = &cobra.Command{
	Use:   "export [file]",
	Short: "Export Cloudflare account configuration to a file",
	Long: `Export all Cloudflare service configurations to a single YAML or JSON file
for disaster recovery, migration, or version control.

The export captures the current state of your account's services including
Workers, KV namespaces, R2 buckets, DNS records, and Zones.

Default output file: cosmoflare-export.yaml

Use --services to export only specific services (comma-separated).
Valid services: workers, kv, r2, dns, zones.

Examples:
  cosmoflare export                                     # Export all to cosmoflare-export.yaml
  cosmoflare export backup.yaml                         # Export all to backup.yaml
  cosmoflare export --services=r2,dns                   # Export only R2 and DNS
  cosmoflare export --format=json backup.json           # Export as JSON
  cosmoflare export --services=workers,kv --json        # Export workers+kv, output summary as JSON`,
	RunE: runExport,
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVar(&exportServices, "services", "", "Services to export (comma-separated: workers,kv,r2,dns,zones)")
	exportCmd.Flags().StringVar(&exportFormat, "format", "yaml", "Output format (yaml or json)")
}

func getExportService() (*cosmoflare.ExportService, error) {
	return cosmoflare.NewExportServiceFromCreds(AccountID, APIToken)
}

func runExport(cmd *cobra.Command, args []string) error {
	outputFile := "cosmoflare-export.yaml"
	if len(args) > 0 {
		outputFile = args[0]
	}

	// Parse format
	format := cosmoflare.ExportFormatYAML
	switch strings.ToLower(exportFormat) {
	case "yaml", "yml":
		format = cosmoflare.ExportFormatYAML
	case "json":
		format = cosmoflare.ExportFormatJSON
	default:
		return fmt.Errorf("unsupported format %q; use yaml or json", exportFormat)
	}

	// Parse service filter
	var opts []cosmoflare.ExportOption
	if exportServices != "" {
		services := strings.Split(exportServices, ",")
		for i := range services {
			services[i] = strings.TrimSpace(services[i])
		}
		if err := cosmoflare.ValidateServices(services); err != nil {
			return fmt.Errorf("invalid --services: %w", err)
		}
		opts = append(opts, cosmoflare.WithExportServices(services))
	}
	opts = append(opts, cosmoflare.WithExportFormat(format))

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("dry run: would export configuration", map[string]interface{}{
				"output_file": outputFile,
				"format":      string(format),
				"services":    exportServices,
			})
		}
		printInfo("DRY RUN: Would export configuration to %s (format: %s)", outputFile, format)
		return nil
	}

	svc, err := getExportService()
	if err != nil {
		return fmt.Errorf("failed to create export service: %w", err)
	}

	printInfo("Exporting Cloudflare configuration...")

	export, err := svc.Export(context.Background(), opts...)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("export failed: %v", err))
		}
		return fmt.Errorf("export failed: %w", err)
	}

	if err := svc.WriteFile(export, outputFile, format); err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to write file: %v", err))
		}
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Build summary
	summary := map[string]interface{}{
		"file":          outputFile,
		"format":        string(format),
		"workers":       len(export.Services.Workers),
		"kv_namespaces": len(export.Services.KVNamespaces),
		"r2_buckets":    len(export.Services.R2Buckets),
		"dns_records":   len(export.Services.DNSRecords),
		"zones":         len(export.Services.Zones),
	}

	if JSONOutput {
		return printSuccessJSON("configuration exported", summary)
	}

	printSuccess("Configuration exported to %s", outputFile)
	printInfo("  Workers:       %d", len(export.Services.Workers))
	printInfo("  KV Namespaces: %d", len(export.Services.KVNamespaces))
	printInfo("  R2 Buckets:    %d", len(export.Services.R2Buckets))
	printInfo("  DNS Records:   %d", len(export.Services.DNSRecords))
	printInfo("  Zones:         %d", len(export.Services.Zones))
	return nil
}
