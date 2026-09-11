package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var (
	d1ExportOutput string
	d1ExportLocal  bool
	d1ExportRemote bool
)

var d1ExportCmd = &cobra.Command{
	Use:   "export [database-id]",
	Short: "Export a D1 database as a SQL dump",
	Long: `Export a D1 database as a streaming SQL dump.

By default the dump is written to stdout. Use --output to write it to a
file instead.

Examples:
  cosmoflare d1 export 480f4f69-1a28-4fdd-9240-1ed29f0ac1df > dump.sql
  cosmoflare d1 export 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --output dump.sql
  cosmoflare d1 export 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --output dump.sql --json`,
	Args: cobra.ExactArgs(1),
	RunE: runD1Export,
}

func init() {
	d1Cmd.AddCommand(d1ExportCmd)

	d1ExportCmd.Flags().StringVar(&d1ExportOutput, "output", "", "Write the SQL dump to this file instead of stdout")
	d1ExportCmd.Flags().BoolVar(&d1ExportLocal, "local", false, "Export the local D1 database (not yet supported)")
	d1ExportCmd.Flags().BoolVar(&d1ExportRemote, "remote", true, "Export the remote D1 database (default)")
}

func runD1Export(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	if d1ExportLocal {
		return fmt.Errorf("--local D1 execution is not yet supported; use --remote")
	}

	svc, err := getD1Service()
	if err != nil {
		return fmt.Errorf("failed to create D1 service: %w", err)
	}

	if DryRun {
		if JSONOutput {
			return printSuccessJSON("DRY RUN: Would export database", map[string]string{"database_id": databaseID, "output": d1ExportOutput})
		}
		printInfo("DRY RUN: Would export database '%s'", databaseID)
		return nil
	}

	rc, err := svc.Export(context.Background(), databaseID)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to export database: %v", err))
		}
		return fmt.Errorf("failed to export database: %w", err)
	}
	defer rc.Close()

	if d1ExportOutput == "" {
		if JSONOutput {
			return printErrorJSON("--json requires --output; streaming the dump to stdout is incompatible with JSON metadata output")
		}
		if _, err := io.Copy(os.Stdout, rc); err != nil {
			return fmt.Errorf("failed to write export dump to stdout: %w", err)
		}
		return nil
	}

	f, err := os.Create(d1ExportOutput)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to create output file: %v", err))
		}
		return fmt.Errorf("failed to create output file %q: %w", d1ExportOutput, err)
	}
	defer f.Close()

	written, err := io.Copy(f, rc)
	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to write export dump: %v", err))
		}
		return fmt.Errorf("failed to write export dump to %q: %w", d1ExportOutput, err)
	}

	if JSONOutput {
		return printSuccessJSON("Database exported", map[string]any{
			"database_id": databaseID,
			"output_path": d1ExportOutput,
			"bytes":       written,
		})
	}

	printSuccess("Database '%s' exported to '%s' (%d bytes)", databaseID, d1ExportOutput, written)
	return nil
}
