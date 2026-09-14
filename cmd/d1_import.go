package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

var (
	d1ImportFile      string
	d1ImportLocal     bool
	d1ImportRemote    bool
	d1ImportForce     bool
	d1ImportBatchSize int
)

var d1ImportCmd = &cobra.Command{
	Use:   "import [database-id]",
	Short: "Import a SQL dump into a D1 database",
	Long: `Import a SQL file into a D1 database.

The file is split into individual SQL statements and grouped into
byte-bounded batches (see --batch-size) before being executed. A batch
that Cloudflare rejects as too large (TOOBIG / internal error 7500) is
automatically retried and, if needed, split into smaller batches.

If a previous import of the same file into the same database stopped
with errors, the command prints a resume hint (the last fully-applied
byte offset) before running.

Examples:
  cosmoflare d1 import 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --file dump.sql
  cosmoflare d1 import 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --file dump.sql --dry-run
  cosmoflare d1 import 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --file dump.sql --force
  cosmoflare d1 import 480f4f69-1a28-4fdd-9240-1ed29f0ac1df --file dump.sql --batch-size 20480 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runD1Import,
}

func init() {
	d1Cmd.AddCommand(d1ImportCmd)

	d1ImportCmd.Flags().StringVar(&d1ImportFile, "file", "", "SQL file to import (required)")
	d1ImportCmd.Flags().BoolVar(&d1ImportLocal, "local", false, "Import into the local D1 database (dev database, skips confirmation)")
	d1ImportCmd.Flags().BoolVar(&d1ImportRemote, "remote", true, "Import into the remote D1 database (default)")
	d1ImportCmd.Flags().BoolVar(&d1ImportForce, "force", false, "Skip confirmation prompt")
	d1ImportCmd.Flags().IntVar(&d1ImportBatchSize, "batch-size", 40960, "Maximum SQL bytes per batch")
	_ = d1ImportCmd.MarkFlagRequired("file")
}

// d1ImportProgressCacheFile mirrors the unexported cache file name written
// by (*cosmoflare.D1Service).Import (pkg/cosmoflare/d1_import.go) so the CLI
// can print a resume hint before running. There is no exported accessor —
// this is the documented, stable on-disk contract (relative to cwd, never
// under $HOME).
const d1ImportProgressCacheFile = ".cosmoflare-d1-import-progress.json"

// d1ImportProgressEntry mirrors the JSON shape of the library's progress
// cache entries; only the fields the CLI needs are declared.
type d1ImportProgressEntry struct {
	LastAppliedOffset int64 `json:"last_applied_offset"`
	HadErrors         bool  `json:"had_errors"`
}

// d1ImportResumeOffset looks up a prior failed import's last-applied byte
// offset for (databaseID, filePath), if the local progress cache records
// one. It returns ok=false when there is no cache, no entry, or the prior
// run completed without errors.
func d1ImportResumeOffset(databaseID, filePath string) (offset int64, ok bool) {
	data, err := os.ReadFile(d1ImportProgressCacheFile)
	if err != nil {
		return 0, false
	}
	var cache map[string]d1ImportProgressEntry
	if err := json.Unmarshal(data, &cache); err != nil {
		return 0, false
	}
	abs, err := filepath.Abs(filePath)
	if err != nil {
		abs = filePath
	}
	entry, found := cache[databaseID+"|"+abs]
	if !found || !entry.HadErrors {
		return 0, false
	}
	return entry.LastAppliedOffset, true
}

// formatCommaInt formats n with thousands separators, e.g. 1234 -> "1,234".
func formatCommaInt(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var groups []string
	for len(s) > 3 {
		groups = append([]string{s[len(s)-3:]}, groups...)
		s = s[:len(s)-3]
	}
	groups = append([]string{s}, groups...)
	out := strings.Join(groups, ",")
	if neg {
		out = "-" + out
	}
	return out
}

func runD1Import(cmd *cobra.Command, args []string) error {
	databaseID := args[0]

	if d1ImportFile == "" {
		return fmt.Errorf("--file flag is required")
	}
	if _, err := os.Stat(d1ImportFile); err != nil {
		return outErr(fmt.Sprintf("cannot read --file %q", d1ImportFile), err)
	}

	svc, err := getD1Service()
	if err != nil {
		return outErr("failed to create D1 service", err)
	}

	if offset, ok := d1ImportResumeOffset(databaseID, d1ImportFile); ok && !JSONOutput {
		printInfo("resuming hint: a previous import of this file stopped with errors after byte offset %d", offset)
	}

	if DryRun {
		result, err := svc.Import(context.Background(), databaseID, d1ImportFile, cosmoflare.ImportOptions{BatchSize: d1ImportBatchSize, DryRun: true})
		if err != nil {
			return outErr("failed to parse import file", err)
		}

		return outResult(result, func() {
			printInfo(
				"DRY RUN: would import %s statement(s) in %d batch(es) (%s total) into database '%s'",
				formatCommaInt(result.TotalStatements), result.TotalBatches, formatBytes(result.BytesProcessed), databaseID,
			)
		})
	}

	if !d1ImportForce && !d1ImportLocal {
		fmt.Printf("Type the database ID '%s' to confirm importing '%s': ", databaseID, d1ImportFile)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)
		if response != databaseID {
			printInfo("Import cancelled")
			return nil
		}
	}

	opts := cosmoflare.ImportOptions{BatchSize: d1ImportBatchSize}
	var statementsSoFar int
	if !JSONOutput {
		opts.Progress = func(batchIndex, totalBatches, batchStatements int, bytesProcessed int64) {
			statementsSoFar += batchStatements
			fmt.Printf("[batch %d/%d] %s statements | %s processed\n", batchIndex, totalBatches, formatCommaInt(statementsSoFar), formatBytes(bytesProcessed))
		}
	}

	result, err := svc.Import(context.Background(), databaseID, d1ImportFile, opts)
	if err != nil {
		return outErr("failed to import file", err)
	}

	var importErr error
	if err := outResult(result, func() {
		if !result.Success {
			printWarning("import completed with %d error(s); see errors above", len(result.Errors))
			for _, e := range result.Errors {
				printError("%s", e)
			}
			if result.ResumeHint != "" {
				printInfo("%s", result.ResumeHint)
			}
			importErr = fmt.Errorf("import completed with %d error(s)", len(result.Errors))
			return
		}

		printSuccess("Imported %s statement(s) in %d batch(es) into database '%s' (%s)", formatCommaInt(result.TotalStatements), result.BatchesApplied, databaseID, formatBytes(result.BytesProcessed))
	}); err != nil {
		return err
	}
	return importErr
}
