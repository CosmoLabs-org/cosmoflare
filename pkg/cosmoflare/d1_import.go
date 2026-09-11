package cosmoflare

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// defaultD1ImportBatchSize is the default maximum number of SQL bytes
// grouped into a single batched query, matching wrangler's d1 execute
// convention.
const defaultD1ImportBatchSize = 40960

// d1ImportTOOBIGMaxRetries is the number of exponential-backoff retries
// attempted for an error coded 7500 before falling back to the
// split-and-retry path.
const d1ImportTOOBIGMaxRetries = 3

// ImportOptions configures a D1 Import operation.
type ImportOptions struct {
	// BatchSize is the maximum number of SQL bytes grouped into a single
	// batched query. Defaults to defaultD1ImportBatchSize when zero or
	// negative.
	BatchSize int

	// DryRun, when true, parses and batches the SQL file but does not
	// execute any statements against the database.
	DryRun bool

	// Progress, when set, is called after each batch attempt (including
	// dry-run batches) with the 1-indexed batch number, the total batch
	// count, the number of statements in that batch, and the cumulative
	// bytes processed so far across the whole import.
	Progress func(batchIndex, totalBatches, batchStatements int, bytesProcessed int64)
}

// D1ImportResult reports the outcome of a D1 Import operation.
type D1ImportResult struct {
	DatabaseID        string   `json:"database_id"`
	File              string   `json:"file"`
	TotalStatements   int      `json:"total_statements"`
	TotalBatches      int      `json:"total_batches"`
	BatchesApplied    int      `json:"batches_applied"`
	BytesProcessed    int64    `json:"bytes_processed"`
	LastAppliedOffset int64    `json:"last_applied_offset"`
	Errors            []string `json:"errors,omitempty"`
	Success           bool     `json:"success"`

	// ResumeHint is set when a previous Import of the same file against
	// the same database left errors behind: it names the byte offset the
	// prior run last fully applied through. There is no automatic resume
	// (no ledger, no --offset flag) — the hint plus the fact that D1
	// import batches are plain INSERT-friendly SQL is the v1 contract: a
	// human can trim the dump to that offset and re-run if they want to
	// skip already-applied statements.
	ResumeHint string `json:"resume_hint,omitempty"`
}

// d1ImportProgressCacheFile is the path (relative to the current working
// directory) of the local import progress cache, mirroring the
// time-travel quota cache convention: never written under $HOME.
const d1ImportProgressCacheFile = ".cosmoflare-d1-import-progress.json"

// d1ImportProgressEntry records the last fully-applied byte offset for one
// (database, file) pair, used only to print a resume hint on the next run —
// it is never read back automatically.
type d1ImportProgressEntry struct {
	LastAppliedOffset int64     `json:"last_applied_offset"`
	HadErrors         bool      `json:"had_errors"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// d1ImportProgressCache maps a "databaseID|absolute file path" key to its
// last recorded import progress.
type d1ImportProgressCache map[string]d1ImportProgressEntry

// d1ImportProgressKey builds the progress cache key for a (databaseID,
// filePath) pair, resolving filePath to an absolute path so the same file
// referenced by different relative paths still hits the same entry.
func d1ImportProgressKey(databaseID, filePath string) string {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		abs = filePath
	}
	return databaseID + "|" + abs
}

// loadD1ImportProgressCache reads the local import progress cache file. A
// missing file is treated as an empty cache, not an error.
func loadD1ImportProgressCache() (d1ImportProgressCache, error) {
	data, err := os.ReadFile(d1ImportProgressCacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return d1ImportProgressCache{}, nil
		}
		return nil, err
	}
	cache := d1ImportProgressCache{}
	if len(data) == 0 {
		return cache, nil
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	if cache == nil {
		cache = d1ImportProgressCache{}
	}
	return cache, nil
}

// saveD1ImportProgressCache writes the import progress cache file to the
// current working directory.
func saveD1ImportProgressCache(cache d1ImportProgressCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(d1ImportProgressCacheFile, data, 0o644)
}

// chunkStatementsByBytes groups statements into batches whose combined byte
// size (statement text plus one separating semicolon each) does not exceed
// maxBytes. A single statement larger than maxBytes still gets its own
// batch — TOOBIG/7500 handling in Import deals with that case, not
// chunking.
func chunkStatementsByBytes(statements []string, maxBytes int) [][]string {
	if maxBytes <= 0 {
		maxBytes = defaultD1ImportBatchSize
	}

	var batches [][]string
	var current []string
	var currentBytes int

	for _, stmt := range statements {
		stmtBytes := len(stmt) + 1 // +1 for the joining semicolon
		if len(current) > 0 && currentBytes+stmtBytes > maxBytes {
			batches = append(batches, current)
			current = nil
			currentBytes = 0
		}
		current = append(current, stmt)
		currentBytes += stmtBytes
	}
	if len(current) > 0 {
		batches = append(batches, current)
	}
	return batches
}

// batchBytes returns the combined byte size of a batch as computed by
// chunkStatementsByBytes (statement text plus one joining semicolon each),
// so progress/offset accounting matches the chunking decision.
func batchBytes(batch []string) int64 {
	var n int64
	for _, stmt := range batch {
		n += int64(len(stmt)) + 1
	}
	return n
}

// d1ErrorChainText concatenates the Error() text of err and everything it
// wraps. Our own error types (*R2Error and friends) format only their own
// operation/message and rely on Unwrap for the underlying cloudflare-go
// error, so a plain err.Error() would miss the CF error's message/code —
// walk the chain instead.
func d1ErrorChainText(err error) string {
	var sb strings.Builder
	for err != nil {
		sb.WriteString(err.Error())
		sb.WriteString(" ")
		err = errors.Unwrap(err)
	}
	return sb.String()
}

// d1IsTOOBIGError reports whether err is Cloudflare's "query too big to
// run" error: message containing "TOOBIG" (case-insensitive) or internal
// error code 7500, anywhere in err's wrap chain.
func d1IsTOOBIGError(err error) bool {
	if err == nil {
		return false
	}
	chain := d1ErrorChainText(err)
	return strings.Contains(strings.ToUpper(chain), "TOOBIG") || strings.Contains(chain, "7500")
}

// d1Is7500Error reports whether err carries Cloudflare's internal error
// code 7500 specifically (a stricter check than d1IsTOOBIGError, used to
// decide whether to attempt a backoff retry before falling back to split).
func d1Is7500Error(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(d1ErrorChainText(err), "7500")
}

// Import reads filePath, splits it into SQL statements, groups them into
// byte-bounded batches, and executes each batch against databaseID via
// Query. A batch that fails with a TOOBIG (or internal code 7500) error is
// retried: error 7500 first gets up to d1ImportTOOBIGMaxRetries
// exponential-backoff retries, then (for either error) the batch is halved
// and each half retried recursively until it succeeds or is a single
// statement that still fails, at which point the error is recorded and
// Import moves on to the next batch. Any other error is recorded
// immediately without retry.
//
// Import always returns a *D1ImportResult (even when some batches failed);
// the returned error is non-nil only for setup failures (validation,
// unreadable file). Per-batch failures are reported via
// D1ImportResult.Errors and Success=false.
func (s *D1Service) Import(ctx context.Context, databaseID string, filePath string, opts ImportOptions) (*D1ImportResult, error) {
	if databaseID == "" {
		return nil, validationError("D1Service.Import", "database ID is required")
	}
	if filePath == "" {
		return nil, validationError("D1Service.Import", "file path is required")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, newError("D1Service.Import", fmt.Sprintf("failed to read import file %q", filePath), err)
	}

	statements := SplitSQLStatements(string(data))
	batches := chunkStatementsByBytes(statements, opts.BatchSize)

	result := &D1ImportResult{
		DatabaseID:      databaseID,
		File:            filePath,
		TotalStatements: len(statements),
		TotalBatches:    len(batches),
	}

	if !opts.DryRun {
		if prev, ok, err := s.previousD1ImportProgress(databaseID, filePath); err == nil && ok && prev.HadErrors {
			result.ResumeHint = fmt.Sprintf(
				"a previous import of this file stopped with errors after byte offset %d; review the dump around that offset before re-running",
				prev.LastAppliedOffset,
			)
		}
	}

	sleepFn := s.sleepFn
	if sleepFn == nil {
		sleepFn = time.Sleep
	}

	var bytesProcessed int64
	for i, batch := range batches {
		bBytes := batchBytes(batch)

		if opts.DryRun {
			bytesProcessed += bBytes
			result.BatchesApplied++
			if opts.Progress != nil {
				opts.Progress(i+1, len(batches), len(batch), bytesProcessed)
			}
			continue
		}

		if errs := s.execD1ImportBatch(ctx, databaseID, batch, sleepFn); len(errs) > 0 {
			result.Errors = append(result.Errors, errs...)
		} else {
			result.BatchesApplied++
			result.LastAppliedOffset += bBytes
		}
		bytesProcessed += bBytes

		if opts.Progress != nil {
			opts.Progress(i+1, len(batches), len(batch), bytesProcessed)
		}
	}

	result.BytesProcessed = bytesProcessed
	result.Success = len(result.Errors) == 0

	if !opts.DryRun {
		if err := s.saveD1ImportProgress(databaseID, filePath, result); err != nil {
			return nil, newError("D1Service.Import", "failed to write import progress cache", err)
		}
	}

	return result, nil
}

// execD1ImportBatch executes one batch of statements, applying the
// TOOBIG/7500 retry-then-split policy described on Import. It returns the
// list of error messages recorded for any statement group that ultimately
// could not be applied (empty when the whole batch succeeded).
func (s *D1Service) execD1ImportBatch(ctx context.Context, databaseID string, batch []string, sleepFn func(time.Duration)) []string {
	if len(batch) == 0 {
		return nil
	}

	joined := strings.Join(batch, ";\n") + ";"
	_, err := s.Query(ctx, databaseID, joined)
	if err == nil {
		return nil
	}

	if d1Is7500Error(err) {
		for attempt := 1; attempt <= d1ImportTOOBIGMaxRetries; attempt++ {
			sleepFn(d1ImportBackoffDelay(attempt))
			_, err = s.Query(ctx, databaseID, joined)
			if err == nil {
				return nil
			}
			if !d1Is7500Error(err) {
				break
			}
		}
	}

	if !d1IsTOOBIGError(err) {
		return []string{d1ErrorChainText(err)}
	}

	if len(batch) == 1 {
		return []string{d1ErrorChainText(err)}
	}

	mid := len(batch) / 2
	var errs []string
	errs = append(errs, s.execD1ImportBatch(ctx, databaseID, batch[:mid], sleepFn)...)
	errs = append(errs, s.execD1ImportBatch(ctx, databaseID, batch[mid:], sleepFn)...)
	return errs
}

// d1ImportBackoffDelay computes the exponential backoff delay (1s, 2s, 4s,
// ...) used between error-7500 retries.
func d1ImportBackoffDelay(attempt int) time.Duration {
	shift := attempt - 1
	if shift < 0 {
		shift = 0
	}
	if shift > 10 {
		shift = 10
	}
	return time.Second * time.Duration(1<<uint(shift))
}

// previousD1ImportProgress looks up the progress cache entry for
// (databaseID, filePath), if any.
func (s *D1Service) previousD1ImportProgress(databaseID, filePath string) (d1ImportProgressEntry, bool, error) {
	cache, err := loadD1ImportProgressCache()
	if err != nil {
		return d1ImportProgressEntry{}, false, err
	}
	entry, ok := cache[d1ImportProgressKey(databaseID, filePath)]
	return entry, ok, nil
}

// saveD1ImportProgress records the outcome of an Import run in the local
// progress cache, keyed by (databaseID, filePath). A fully successful run
// clears any prior entry for that key.
func (s *D1Service) saveD1ImportProgress(databaseID, filePath string, result *D1ImportResult) error {
	cache, err := loadD1ImportProgressCache()
	if err != nil {
		return err
	}

	key := d1ImportProgressKey(databaseID, filePath)
	if result.Success {
		delete(cache, key)
	} else {
		cache[key] = d1ImportProgressEntry{
			LastAppliedOffset: result.LastAppliedOffset,
			HadErrors:         true,
			UpdatedAt:         time.Now().UTC(),
		}
	}

	return saveD1ImportProgressCache(cache)
}
