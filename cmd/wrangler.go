package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var wranglerCmd = &cobra.Command{
	Use:   "wrangler",
	Short: "Wrangler compatibility — import, diff, and validate wrangler.toml",
	Long: `Wrangler compatibility layer for migrating from Cloudflare Wrangler to Cosmoflare.

Read wrangler.toml files and translate them to .cosmoflare.yaml format.

Commands:
  import    Read wrangler.toml and generate .cosmoflare.yaml
  diff      Compare wrangler.toml against existing .cosmoflare.yaml
  validate  Validate a wrangler.toml file for correctness

The import maps these wrangler.toml fields:
  name               → workers.main.name
  main               → workers.main.script
  compatibility_date → workers.main.compatibility_date
  kv_namespaces      → kv.namespaces
  r2_buckets         → r2.buckets
  d1_databases       → d1.databases
  [env.*]            → profiles (multiple environments)

Examples:
  cosmoflare wrangler import                       # Import ./wrangler.toml
  cosmoflare wrangler import ./path/to/wrangler.toml
  cosmoflare wrangler diff                         # Compare against .cosmoflare.yaml
  cosmoflare wrangler validate                     # Validate wrangler.toml
  cosmoflare wrangler import --output custom.yaml  # Custom output path
  cosmoflare wrangler import --json                # Output as JSON`,
}

var (
	wranglerOutputPath string
	wranglerForce      bool
)

var wranglerImportCmd = &cobra.Command{
	Use:   "import [path]",
	Short: "Import wrangler.toml and generate .cosmoflare.yaml",
	Long: `Read a wrangler.toml file and generate the equivalent .cosmoflare.yaml configuration.

If no path is given, reads ./wrangler.toml from the current directory.
The output file defaults to .cosmoflare.yaml in the same directory as the input.

Mapped fields:
  name               → workers.main.name
  main               → workers.main.script
  compatibility_date → workers.main.compatibility_date
  kv_namespaces      → kv.namespaces + workers.main.bindings.kv
  r2_buckets         → r2.buckets + workers.main.bindings.r2
  d1_databases       → d1.databases + workers.main.bindings.d1
  [env.*]            → profiles.<env>.worker
  vars               → workers.main.vars

Examples:
  cosmoflare wrangler import
  cosmoflare wrangler import ./project/wrangler.toml
  cosmoflare wrangler import --output .cosmoflare.yaml --force
  cosmoflare wrangler import --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWranglerImport,
}

var wranglerDiffCmd = &cobra.Command{
	Use:   "diff [path]",
	Short: "Compare wrangler.toml against existing .cosmoflare.yaml",
	Long: `Compare a wrangler.toml file against an existing .cosmoflare.yaml to show
what would change if you re-imported.

If no path is given, reads ./wrangler.toml from the current directory.
The .cosmoflare.yaml is expected in the same directory as wrangler.toml.

Examples:
  cosmoflare wrangler diff
  cosmoflare wrangler diff ./project/wrangler.toml
  cosmoflare wrangler diff --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWranglerDiff,
}

var wranglerValidateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate a wrangler.toml file",
	Long: `Validate a wrangler.toml file for common issues and errors.

Checks for:
  - Missing required fields (name)
  - Missing KV namespace IDs
  - Missing R2 bucket names
  - Missing D1 database IDs
  - Duplicate binding names
  - Missing compatibility_date (warning)
  - Missing entry point (warning)

Examples:
  cosmoflare wrangler validate
  cosmoflare wrangler validate ./project/wrangler.toml
  cosmoflare wrangler validate --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWranglerValidate,
}

func init() {
	// Register subcommands
	wranglerCmd.AddCommand(wranglerImportCmd)
	wranglerCmd.AddCommand(wranglerDiffCmd)
	wranglerCmd.AddCommand(wranglerValidateCmd)

	// Flags
	wranglerImportCmd.Flags().StringVarP(&wranglerOutputPath, "output", "o", "", "Output path for .cosmoflare.yaml (default: same directory as input)")
	wranglerImportCmd.Flags().BoolVarP(&wranglerForce, "force", "f", false, "Overwrite existing .cosmoflare.yaml without prompting")

	// Register on root
	rootCmd.AddCommand(wranglerCmd)
}

func runWranglerImport(cmd *cobra.Command, args []string) error {
	inputPath := ""
	if len(args) > 0 {
		inputPath = args[0]
	}

	tomlPath, err := cosmoflare.FindWranglerToml(inputPath)
	if err != nil {
		return outErr("failed to find wrangler.toml", err)
	}

	ws := cosmoflare.NewWranglerService()

	// Parse
	wc, err := ws.ParseToml(tomlPath)
	if err != nil {
		return outErr(fmt.Sprintf("failed to parse %s", tomlPath), err)
	}

	// Validate
	results := ws.Validate(wc)
	if cosmoflare.WranglerHasErrors(results) {
		// JSON mode prints the shaped payload; plain mode prints the errors
		// and surfaces the validation failure as the command's error.
		var humanErr error
		if perr := outPayload("validation failed", func() any {
			return map[string]interface{}{
				"valid":   false,
				"results": results,
			}
		}, func() {
			printError("wrangler.toml has validation errors:")
			for _, r := range results {
				if r.Level == "error" {
					printError("  %s: %s", r.Field, r.Message)
				}
			}
			humanErr = fmt.Errorf("wrangler.toml validation failed; fix errors before importing")
		}); perr != nil {
			return perr
		}
		return humanErr
	}

	// Print warnings
	if !JSONOutput {
		for _, r := range results {
			if r.Level == "warning" {
				printWarning("%s: %s", r.Field, r.Message)
			}
		}
	}

	// Convert
	cc := ws.ConvertToConfig(wc)

	// Determine output path
	outputPath := wranglerOutputPath
	if outputPath == "" {
		outputPath = filepath.Join(filepath.Dir(tomlPath), ".cosmoflare.yaml")
	}

	// Check if output exists
	if !wranglerForce {
		if _, err := os.Stat(outputPath); err == nil {
			return outErrf("output file %s already exists. Use --force to overwrite", outputPath)
		}
	}

	if DryRun {
		return outPayload("dry run: would write .cosmoflare.yaml", func() any {
			data, _ := cosmoflare.MarshalWranglerImportYAML(cc)
			return map[string]interface{}{
				"input":   tomlPath,
				"output":  outputPath,
				"preview": string(data),
				"worker":  wc.Name,
			}
		}, func() {
			printSuccess("Would import %s → %s", getRelativePath(tomlPath), getRelativePath(outputPath))
			printInfo("Worker: %s", wc.Name)
			if wc.Main != "" {
				printInfo("Entry point: %s", wc.Main)
			}
		})
	}

	// Write output
	if err := cosmoflare.WriteWranglerImportYAML(cc, outputPath); err != nil {
		return outErr(fmt.Sprintf("failed to write %s", outputPath), err)
	}

	return outPayload("imported wrangler.toml to .cosmoflare.yaml", func() any {
		return map[string]interface{}{
			"input":         tomlPath,
			"output":        outputPath,
			"worker":        wc.Name,
			"kv_namespaces": len(wc.KVNamespaces),
			"r2_buckets":    len(wc.R2Buckets),
			"d1_databases":  len(wc.D1Databases),
			"environments":  len(wc.Env),
		}
	}, func() {
		printSuccess("Imported %s → %s", getRelativePath(tomlPath), getRelativePath(outputPath))
		printInfo("Worker: %s", wc.Name)
		if len(wc.KVNamespaces) > 0 {
			printInfo("KV namespaces: %d", len(wc.KVNamespaces))
		}
		if len(wc.R2Buckets) > 0 {
			printInfo("R2 buckets: %d", len(wc.R2Buckets))
		}
		if len(wc.D1Databases) > 0 {
			printInfo("D1 databases: %d", len(wc.D1Databases))
		}
		if len(wc.Env) > 0 {
			printInfo("Environments: %d", len(wc.Env))
		}
	})
}

func runWranglerDiff(cmd *cobra.Command, args []string) error {
	inputPath := ""
	if len(args) > 0 {
		inputPath = args[0]
	}

	tomlPath, err := cosmoflare.FindWranglerToml(inputPath)
	if err != nil {
		return outErr("failed to find wrangler.toml", err)
	}

	// Find existing .cosmoflare.yaml
	cosmoPath := filepath.Join(filepath.Dir(tomlPath), ".cosmoflare.yaml")
	if _, err := os.Stat(cosmoPath); os.IsNotExist(err) {
		return outErrf("no .cosmoflare.yaml found at %s. Run 'cosmoflare wrangler import' first", cosmoPath)
	}

	ws := cosmoflare.NewWranglerService()

	// Parse wrangler.toml
	wc, err := ws.ParseToml(tomlPath)
	if err != nil {
		return outErr("failed to parse wrangler.toml", err)
	}

	// Load existing .cosmoflare.yaml
	existing, err := cosmoflare.LoadWranglerImportYAML(cosmoPath)
	if err != nil {
		return outErr("failed to parse .cosmoflare.yaml", err)
	}

	// Diff
	diffs := ws.DiffWranglerConfigs(wc, existing)

	return outPayload("diff complete", func() any {
		return map[string]interface{}{
			"wrangler_path": tomlPath,
			"cosmo_path":    cosmoPath,
			"differences":   diffs,
			"diff_count":    len(diffs),
			"in_sync":       len(diffs) == 0,
		}
	}, func() {
		printInfo("Comparing %s ↔ %s", getRelativePath(tomlPath), getRelativePath(cosmoPath))
		fmt.Println()
		fmt.Print(cosmoflare.FormatWranglerDiffTable(diffs))
	})
}

func runWranglerValidate(cmd *cobra.Command, args []string) error {
	inputPath := ""
	if len(args) > 0 {
		inputPath = args[0]
	}

	tomlPath, err := cosmoflare.FindWranglerToml(inputPath)
	if err != nil {
		return outErr("failed to find wrangler.toml", err)
	}

	ws := cosmoflare.NewWranglerService()

	// Parse
	wc, err := ws.ParseToml(tomlPath)
	if err != nil {
		return outErr("failed to parse wrangler.toml", err)
	}

	// Validate
	results := ws.Validate(wc)

	// JSON mode prints the shaped payload; plain mode prints the results and
	// surfaces a validation failure as the command's error.
	var humanErr error
	if perr := outPayload("validation complete", func() any {
		return map[string]interface{}{
			"path":     tomlPath,
			"valid":    !cosmoflare.WranglerHasErrors(results),
			"results":  results,
			"errors":   countByLevel(results, "error"),
			"warnings": countByLevel(results, "warning"),
		}
	}, func() {
		printInfo("Validating %s", getRelativePath(tomlPath))
		fmt.Println()

		errors := 0
		warnings := 0
		for _, r := range results {
			switch r.Level {
			case "error":
				printError("%s: %s", r.Field, r.Message)
				errors++
			case "warning":
				printWarning("%s: %s", r.Field, r.Message)
				warnings++
			}
		}

		fmt.Println()
		if errors > 0 {
			printError("%d error(s), %d warning(s)", errors, warnings)
			humanErr = fmt.Errorf("validation failed with %d error(s)", errors)
			return
		}

		if warnings > 0 {
			printWarning("%d warning(s), 0 errors", warnings)
		} else {
			printSuccess("wrangler.toml is valid (0 errors, 0 warnings)")
		}
	}); perr != nil {
		return perr
	}
	return humanErr
}

// countByLevel counts validation results matching a level.
func countByLevel(results []cosmoflare.WranglerValidationResult, level string) int {
	n := 0
	for _, r := range results {
		if r.Level == level {
			n++
		}
	}
	return n
}
