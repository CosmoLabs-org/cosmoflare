package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var (
	validateStrict bool
	validateFix    bool
)

var validateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate .cosmoflare.yaml against Cloudflare API constraints",
	Long: `Validate a .cosmoflare.yaml configuration file before applying it.

Checks the config against Cloudflare API constraints and naming rules:
  • Required fields (name, type)
  • Worker script paths exist on disk
  • KV namespace names are valid (alphanumeric + hyphens)
  • DNS record types are valid (A, AAAA, CNAME, MX, TXT, etc.)
  • R2 bucket names follow S3 naming rules (3-63 chars, lowercase, no dots)
  • Zone IDs are 32-char hex strings
  • SSL modes are valid (off, flexible, full, strict)
  • No duplicate resource names within a service
  • TTL values in valid range (1=auto, or 60-86400)

Output lists errors, warnings, and info messages with field locations.
Exit code 0 if valid, 1 if errors found.

Flags:
  --strict    Treat warnings as errors (exit 1 on any finding)
  --fix       Show auto-fix hints for fixable issues
  --json      Structured JSON output

Examples:
  cosmoflare validate                           # Validate .cosmoflare.yaml in cwd
  cosmoflare validate path/to/config.yaml       # Validate a specific file
  cosmoflare validate --strict                  # Fail on warnings too
  cosmoflare validate --fix                     # Show fix suggestions
  cosmoflare validate --json                    # Machine-readable output
  cosmoflare validate --json --strict           # JSON + strict mode`,
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().BoolVar(&validateStrict, "strict", false, "Treat warnings as errors (exit 1 on any warning)")
	validateCmd.Flags().BoolVar(&validateFix, "fix", false, "Show auto-fix hints for fixable issues")
}

func runValidate(cmd *cobra.Command, args []string) error {
	var cfg *cosmoflare.CosmoflareConfig
	var baseDir string
	var err error

	if len(args) > 0 {
		// Load specific file — resolve directory for path checks
		filePath := args[0]
		info, statErr := os.Stat(filePath)
		if statErr != nil {
			if JSONOutput {
				return printErrorJSON(fmt.Sprintf("config file not found: %s", filePath))
			}
			return fmt.Errorf("config file not found: %s", filePath)
		}
		if info.IsDir() {
			baseDir = filePath
		} else {
			baseDir = fileDir(filePath)
		}
		cfg, err = cosmoflare.LoadCosmoflareConfig(baseDir)
	} else {
		// Default: search from cwd
		baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		cfg, err = cosmoflare.LoadCosmoflareConfig(baseDir)
	}

	if err != nil {
		if JSONOutput {
			return printErrorJSON(fmt.Sprintf("failed to load config: %v", err))
		}
		return fmt.Errorf("failed to load config: %w\n\nRun 'cosmoflare init' to create a .cosmoflare.yaml", err)
	}

	validator := cosmoflare.NewConfigValidator(baseDir)
	result := validator.Validate(cfg)

	if JSONOutput {
		return printJSON(result)
	}

	return printValidationResult(result)
}

// printValidationResult renders the validation result for human consumption.
func printValidationResult(result *cosmoflare.ValidationResult) error {
	if result.TotalFindings() == 0 {
		printSuccess("Configuration is valid — no issues found")
		return nil
	}

	// Print errors
	for _, f := range result.Errors {
		printError("[%s] %s", f.Field, f.Message)
		if validateFix && f.Fixable {
			printInfo("  Fix: %s", f.FixHint)
		}
	}

	// Print warnings
	for _, f := range result.Warnings {
		printWarning("[%s] %s", f.Field, f.Message)
		if validateFix && f.Fixable {
			printInfo("  Fix: %s", f.FixHint)
		}
	}

	// Print info
	for _, f := range result.Info {
		printInfo("[%s] %s", f.Field, f.Message)
	}

	// Summary line
	fmt.Printf("\n%d error(s), %d warning(s), %d info\n",
		len(result.Errors), len(result.Warnings), len(result.Info))

	// Determine exit status
	if result.HasErrors() {
		return fmt.Errorf("validation failed with %d error(s)", len(result.Errors))
	}

	if validateStrict && result.HasWarnings() {
		return fmt.Errorf("validation failed in strict mode: %d warning(s)", len(result.Warnings))
	}

	printSuccess("Configuration is valid")
	return nil
}

// fileDir returns the directory component of a file path.
func fileDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
