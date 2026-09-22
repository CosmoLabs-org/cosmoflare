package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

// FEAT-018 P0: batched remote SQL push. Production evidence (MyCarGuide)
// showed 84k-statement seed files pushed by hand; wrangler cannot handle
// them. `d1 push-sql` splits a SQL file into byte-capped batches, pushes
// them over the remote D1 query path with bounded concurrency, retries
// transient failures with backoff, splits TOOBIG batches recursively, and
// supports kill-safe resume via --from-batch (batches are idempotent when
// the caller's SQL uses INSERT OR IGNORE).

const (
	d1PushDefaultBatchBytes  = 40960 // ~40KB default batch cap
	d1PushDefaultConcurrency = 2
	d1PushMaxConcurrency     = 8
	d1PushMaxAttempts        = 3 // transient retry budget (7500 / 5xx / network)
	d1PushMaxSplitDepth      = 4 // recursive TOOBIG split depth cap
)

var (
	d1PushDatabase    string
	d1PushConcurrency int
	d1PushBatchBytes  int
	d1PushFromBatch   int

	// d1PushQueryBatch is the single remote-execution seam: it sends one
	// joined batch through the same (*D1Service).Query call that
	// `d1 query --remote` uses. It is a var so tests can stub the network.
	d1PushQueryBatch = func(ctx context.Context, svc *cosmoflare.D1Service, databaseID, sql string) error {
		_, err := svc.Query(ctx, databaseID, sql)
		return err
	}

	// d1PushSleep backs off between transient retries; var for tests.
	d1PushSleep = time.Sleep
)

var d1PushSQLCmd = &cobra.Command{
	Use:   "push-sql FILE --database ID",
	Short: "Push a SQL file to a remote D1 database in batches",
	Long: `Push a SQL file to a remote D1 database, batch by batch.

The file is split into individual statements (quote/comment/semicolon
aware) and packed into byte-capped batches (--batch-bytes, default 40KB).
Batches execute over the remote D1 query path with bounded concurrency
(--concurrency, default 2, max 8). Transient failures (internal error
code 7500, 5xx, network) are retried with backoff; SQLITE_TOOBIG-style
failures split the batch in half and retry each half recursively.

Statements should be idempotent (e.g. INSERT OR IGNORE) so an interrupted
run can resume: on failure the command prints which batch to resume from.

Examples:
  cosmoflare d1 push-sql seed.sql --database 480f4f69-1a28-4fdd-9240-1ed29f0ac1df
  cosmoflare d1 push-sql seed.sql --database <id> --concurrency 4 --batch-bytes 65536
  cosmoflare d1 push-sql seed.sql --database <id> --from-batch 13 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runD1PushSQL,
}

func init() {
	d1Cmd.AddCommand(d1PushSQLCmd)

	d1PushSQLCmd.Flags().StringVar(&d1PushDatabase, "database", "", "D1 database ID to push to (required)")
	d1PushSQLCmd.Flags().IntVar(&d1PushConcurrency, "concurrency", d1PushDefaultConcurrency, "Parallel batch executions (1-8)")
	d1PushSQLCmd.Flags().IntVar(&d1PushBatchBytes, "batch-bytes", d1PushDefaultBatchBytes, "Maximum SQL bytes per batch")
	d1PushSQLCmd.Flags().IntVar(&d1PushFromBatch, "from-batch", 1, "Skip batches before this 1-based index (resume after an interrupted run)")
	_ = d1PushSQLCmd.MarkFlagRequired("database")
}

// d1PushResult is the final envelope printed on completion (one JSON
// object in --json mode, a summary line otherwise).
type d1PushResult struct {
	DatabaseID      string `json:"database_id"`
	File            string `json:"file"`
	TotalStatements int    `json:"total_statements"`
	TotalBatches    int    `json:"total_batches"`
	BatchesExecuted int    `json:"batches_executed"`
	BatchesFailed   int    `json:"batches_failed"`
	ResumedFrom     int    `json:"resumed_from,omitempty"`
	Success         bool   `json:"success"`
	Error           string `json:"error,omitempty"`
}

func runD1PushSQL(cmd *cobra.Command, args []string) error {
	file := args[0]

	if d1PushDatabase == "" {
		return fmt.Errorf("--database flag is required")
	}
	if d1PushBatchBytes <= 0 {
		return fmt.Errorf("--batch-bytes must be positive")
	}
	if d1PushConcurrency < 1 {
		d1PushConcurrency = 1
	}
	if d1PushConcurrency > d1PushMaxConcurrency {
		d1PushConcurrency = d1PushMaxConcurrency
	}
	if d1PushFromBatch < 1 {
		d1PushFromBatch = 1
	}

	// The database argument is a server UUID, not a profile-scoped resource
	// name — no prefix is applied (TASK-011 rule).
	databaseID := d1PushDatabase

	data, err := os.ReadFile(file)
	if err != nil {
		return outErrf("cannot read file %q: %v", file, err)
	}

	svc, err := getD1Service()
	if err != nil {
		return outErr("failed to create D1 service", err)
	}

	statements := splitSQLStatements(string(data))
	batches := packSQLBatches(statements, d1PushBatchBytes)

	if DryRun {
		return outPayload("DRY RUN: Would push SQL file", func() any {
			return &d1PushResult{
				DatabaseID:      databaseID,
				File:            file,
				TotalStatements: len(statements),
				TotalBatches:    len(batches),
				BatchesExecuted: len(batches[max(0, d1PushFromBatch-1):]),
				ResumedFrom:     d1PushFromBatch,
				Success:         true,
			}
		}, func() {
			printInfo("DRY RUN: Would push %d statement(s) in %d batch(es) to database '%s'", len(statements), len(batches), databaseID)
		})
	}

	res := &d1PushResult{
		DatabaseID:      databaseID,
		File:            file,
		TotalStatements: len(statements),
		TotalBatches:    len(batches),
		ResumedFrom:     d1PushFromBatch,
	}

	executed, failedIdx, failedErr := d1PushAll(context.Background(), svc, databaseID, batches, d1PushFromBatch, d1PushConcurrency)
	res.BatchesExecuted = executed
	if failedIdx > 0 {
		res.BatchesFailed = len(batches) + 1 - d1PushFromBatch - executed
		res.Error = fmt.Sprintf("batch %d/%d failed: %v", failedIdx, len(batches), failedErr)
		res.Success = false
	} else {
		res.Success = true
	}

	if res.BatchesFailed > 0 {
		var pushErr error
		if err := outResult(res, func() {
			printError("batch %d/%d failed: %v", failedIdx, res.TotalBatches, failedErr)
			printInfo("resume with --from-batch %d", failedIdx)
			pushErr = fmt.Errorf("push failed on batch %d: %w — resume with --from-batch %d", failedIdx, failedErr, failedIdx)
		}); err != nil {
			return err
		}
		return pushErr
	}

	return outResult(res, func() {
		printSuccess("Pushed %d statement(s) in %d batch(es) to database '%s' (%d executed)", res.TotalStatements, res.TotalBatches, res.DatabaseID, res.BatchesExecuted)
	})
}

// d1PushAll executes batches[fromBatch-1:] over a bounded worker pool and
// returns the number of batches that succeeded plus the 1-based index and
// error of the first batch that failed (0 / nil when all succeeded).
// Progress lines go to stderr in human mode only.
func d1PushAll(ctx context.Context, svc *cosmoflare.D1Service, databaseID string, batches [][]string, fromBatch, concurrency int) (executed, failedIdx int, failedErr error) {
	if fromBatch < 1 {
		fromBatch = 1
	}

	jobs := make(chan int)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				err := d1PushExecBatch(ctx, svc, databaseID, batches[i], 0)
				mu.Lock()
				if err != nil {
					if failedIdx == 0 || i+1 < failedIdx {
						failedIdx, failedErr = i+1, err
					}
				} else {
					executed++
				}
				mu.Unlock()
				if err == nil && !JSONOutput {
					fmt.Fprintf(os.Stderr, "batch %d/%d ok (%d statements)\n", i+1, len(batches), len(batches[i]))
				}
			}
		}()
	}

	for i := fromBatch - 1; i < len(batches); i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	return executed, failedIdx, failedErr
}

// d1PushExecBatch pushes one batch: transient errors (7500 / 5xx / network)
// get up to d1PushMaxAttempts attempts with exponential backoff; TOOBIG
// errors split the batch in half and retry each half recursively up to
// d1PushMaxSplitDepth levels. Anything else fails immediately.
func d1PushExecBatch(ctx context.Context, svc *cosmoflare.D1Service, databaseID string, batch []string, depth int) error {
	if len(batch) == 0 {
		return nil
	}

	joined := d1PushJoinBatch(batch)
	var err error
	for attempt := 1; attempt <= d1PushMaxAttempts; attempt++ {
		if attempt > 1 {
			d1PushSleep(d1PushBackoff(attempt))
		}
		if err = d1PushQueryBatch(ctx, svc, databaseID, joined); err == nil {
			return nil
		}
		if d1PushIsTooBig(err) {
			break
		}
		if !d1PushIsTransient(err) {
			return err
		}
	}

	if err != nil && d1PushIsTooBig(err) {
		if len(batch) == 1 || depth >= d1PushMaxSplitDepth {
			return err
		}
		mid := len(batch) / 2
		if err := d1PushExecBatch(ctx, svc, databaseID, batch[:mid], depth+1); err != nil {
			return err
		}
		return d1PushExecBatch(ctx, svc, databaseID, batch[mid:], depth+1)
	}

	return err
}

// d1PushJoinBatch renders a batch as the SQL sent over the wire: statements
// joined with ";\n" and a trailing semicolon, matching the import path.
func d1PushJoinBatch(batch []string) string {
	return strings.Join(batch, ";\n") + ";"
}

// packSQLBatches accumulates whole statements into batches whose joined
// size stays at or below cap bytes. A single statement larger than the cap
// becomes its own batch rather than being dropped or truncated.
func packSQLBatches(statements []string, cap int) [][]string {
	var batches [][]string
	var cur []string
	var size int
	for _, stmt := range statements {
		n := len(stmt) + 2 // statement + ";\n" separator
		if len(cur) > 0 && size+n > cap {
			batches = append(batches, cur)
			cur, size = nil, 0
		}
		cur = append(cur, stmt)
		size += n
	}
	if len(cur) > 0 {
		batches = append(batches, cur)
	}
	return batches
}

// d1PushBackoff returns the exponential backoff between transient retries
// (250ms, 500ms, ...). Kept short: the remote errors this covers are
// typically immediate capacity rejections, not sustained outages.
func d1PushBackoff(attempt int) time.Duration {
	shift := attempt - 1
	if shift < 0 {
		shift = 0
	}
	if shift > 4 {
		shift = 4
	}
	return 250 * time.Millisecond * time.Duration(1<<uint(shift))
}

// d1PushErrText flattens an error's wrap chain into one lowercase string
// for substring classification.
func d1PushErrText(err error) string {
	var b strings.Builder
	for err != nil {
		b.WriteString(err.Error())
		b.WriteString("; ")
		err = errors.Unwrap(err)
	}
	return strings.ToLower(b.String())
}

// d1PushIsTransient reports whether err looks transient: Cloudflare
// internal error code 7500, gateway 5xx wording, or common network
// failures. These are retried with backoff.
func d1PushIsTransient(err error) bool {
	if err == nil {
		return false
	}
	text := d1PushErrText(err)
	for _, marker := range []string{
		"7500",
		"internal error",
		"bad gateway",
		"service unavailable",
		"gateway timeout",
		"502 bad gateway",
		"http status: 50",
		"timeout",
		"connection reset",
		"connection refused",
		"broken pipe",
		"unexpected eof",
		"network",
		"temporarily unavailable",
	} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

// d1PushIsTooBig reports whether err is an SQLITE_TOOBIG-style failure
// ("TOOBIG", "too big", "too large"). These split the batch instead of
// retrying it whole.
func d1PushIsTooBig(err error) bool {
	if err == nil {
		return false
	}
	text := d1PushErrText(err)
	return strings.Contains(text, "toobig") ||
		strings.Contains(text, "too big") ||
		strings.Contains(text, "too large")
}
