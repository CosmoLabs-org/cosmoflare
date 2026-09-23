package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var (
	importYes   bool
	importMerge bool
)

var importCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import Cloudflare configuration from an export file",
	Long: `Restore Cloudflare service configurations from a previously exported file.

This command reads a YAML or JSON export file (created by 'cosmoflare export')
and applies the configuration to your account. It can create R2 buckets,
KV namespaces, and DNS records.

Use --dry-run to preview changes without applying them.
Use --merge to skip resources that already exist (instead of reporting conflicts).
Use --yes to skip the confirmation prompt.

Default input file: cosmoflare-export.yaml

Examples:
  cosmoflare import                                    # Import from cosmoflare-export.yaml
  cosmoflare import backup.yaml                        # Import from backup.yaml
  cosmoflare import backup.yaml --dry-run              # Preview what would change
  cosmoflare import backup.yaml --merge                # Merge with existing config
  cosmoflare import backup.yaml --merge --yes          # Merge without confirmation
  cosmoflare import backup.yaml --dry-run --json       # Preview as JSON`,
	RunE: runImport,
}

func init() {
	rootCmd.AddCommand(importCmd)

	importCmd.Flags().BoolVar(&importYes, "yes", false, "Skip confirmation prompt")
	importCmd.Flags().BoolVar(&importMerge, "merge", false, "Merge with existing configuration (skip existing resources)")
}

func runImport(cmd *cobra.Command, args []string) error {
	inputFile := "cosmoflare-export.yaml"
	if len(args) > 0 {
		inputFile = args[0]
	}

	// Check file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return outErrf("file not found: %s", inputFile)
	}

	// Parse the export file
	exportCfg, err := cosmoflare.ParseFile(inputFile)
	if err != nil {
		return outErr(fmt.Sprintf("failed to parse %s", inputFile), err)
	}

	if !DryRun && !importYes {
		if err := confirmImport(inputFile, exportCfg); err != nil {
			return err
		}
	}

	result, err := executeImport(inputFile, exportCfg)
	if err != nil {
		return err
	}

	return presentImportResult(result)
}

// confirmImport shows what will be imported and aborts unless the user
// passes --yes or --dry-run.
func confirmImport(inputFile string, exportCfg *cosmoflare.ExportConfig) error {
	totalResources := len(exportCfg.Services.Workers) +
		len(exportCfg.Services.KVNamespaces) +
		len(exportCfg.Services.R2Buckets) +
		len(exportCfg.Services.DNSRecords) +
		len(exportCfg.Services.Zones)

	printWarning("About to import configuration from %s", inputFile)
	printInfo("  Exported at: %s", exportCfg.ExportedAt)
	printInfo("  Account ID:  %s", exportCfg.AccountID)
	printInfo("  Resources:   %d total", totalResources)
	if importMerge {
		printInfo("  Mode:        merge (skip existing)")
	} else {
		printInfo("  Mode:        create")
	}
	printInfo("")
	printError("This will modify your Cloudflare account. Use --dry-run to preview first.")
	return fmt.Errorf("import aborted: use --yes to confirm or --dry-run to preview")
}

// executeImport builds import options from flags and runs the import.
func executeImport(inputFile string, exportCfg *cosmoflare.ExportConfig) (*cosmoflare.ImportResult, error) {
	svc, err := getExportService()
	if err != nil {
		return nil, outErr("failed to create export service", err)
	}

	var opts []cosmoflare.ImportOption
	if DryRun {
		opts = append(opts, cosmoflare.WithImportDryRun(true))
	}
	if importMerge {
		opts = append(opts, cosmoflare.WithImportMerge(true))
	}

	if !DryRun {
		printInfo("Importing configuration from %s...", inputFile)
	} else {
		printInfo("DRY RUN: Previewing import from %s...", inputFile)
	}

	result, err := svc.Import(context.Background(), exportCfg, opts...)
	if err != nil {
		return nil, outErr("import failed", err)
	}
	return result, nil
}

// presentImportResult renders the actions table, errors, and summary.
func presentImportResult(result *cosmoflare.ImportResult) error {
	msg := "import complete"
	if DryRun {
		msg = "import preview"
	}

	return outPayload(msg, func() any {
		return result
	}, func() {
		// Print results table
		if len(result.Actions) > 0 {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "SERVICE\tACTION\tRESOURCE\tDETAIL")
			for _, a := range result.Actions {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Service, a.Action, a.Resource, a.Detail)
			}
			w.Flush()
			fmt.Println()
		}

		// Print errors
		if len(result.Errors) > 0 {
			printWarning("%d errors occurred:", len(result.Errors))
			for _, e := range result.Errors {
				printError("  %s", e)
			}
		}

		// Summary
		created := 0
		skipped := 0
		for _, a := range result.Actions {
			switch a.Action {
			case "create":
				created++
			case "skip":
				skipped++
			}
		}

		if DryRun {
			printSuccess("DRY RUN: %d would be created, %d would be skipped", created, skipped)
		} else {
			printSuccess("Import complete: %d created, %d skipped, %d errors", created, skipped, len(result.Errors))
		}
	})
}
