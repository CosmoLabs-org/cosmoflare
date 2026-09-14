package cosmoflare

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SyncDirection indicates whether the sync is uploading or downloading.
type SyncDirection int

const (
	// SyncUp uploads local files to R2.
	SyncUp SyncDirection = iota
	// SyncDown downloads R2 objects to local.
	SyncDown
)

// String returns the string representation of a SyncDirection.
func (d SyncDirection) String() string {
	switch d {
	case SyncUp:
		return "up"
	case SyncDown:
		return "down"
	default:
		return "unknown"
	}
}

// SyncOpAction describes what a single sync operation will do.
type SyncOpAction int

const (
	SyncOpUpload   SyncOpAction = iota // Upload local file to R2
	SyncOpDownload                     // Download R2 object to local
	SyncOpDelete                       // Delete file at destination
	SyncOpSkip                         // File is unchanged, skip
)

// String returns the string representation of a SyncOpAction.
func (a SyncOpAction) String() string {
	switch a {
	case SyncOpUpload:
		return "upload"
	case SyncOpDownload:
		return "download"
	case SyncOpDelete:
		return "delete"
	case SyncOpSkip:
		return "skip"
	default:
		return "unknown"
	}
}

// ObjectInfo describes a remote object for sync comparison.
type ObjectInfo struct {
	Key          string    `json:"key"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	ETag         string    `json:"etag"`
}

// LocalFileInfo describes a local file for sync comparison.
type LocalFileInfo struct {
	RelPath  string    `json:"rel_path"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
	Checksum string    `json:"checksum,omitempty"`
}

// SyncOp represents a single planned sync operation.
type SyncOp struct {
	Action        SyncOpAction `json:"action"`
	Key           string       `json:"key"`
	LocalPath     string       `json:"local_path,omitempty"`
	Size          int64        `json:"size"`
	Reason        string       `json:"reason,omitempty"`
	RemoteModTime time.Time    `json:"remote_mod_time,omitempty"`
}

// SyncPlanSummary holds aggregate counts for a sync plan.
type SyncPlanSummary struct {
	Uploads   int   `json:"uploads"`
	Downloads int   `json:"downloads"`
	Deletes   int   `json:"deletes"`
	Skips     int   `json:"skips"`
	TotalSize int64 `json:"total_size"`
}

// SyncPlan contains all operations to be performed during a sync.
type SyncPlan struct {
	Bucket     string        `json:"bucket"`
	Prefix     string        `json:"prefix,omitempty"`
	Direction  SyncDirection `json:"direction"`
	LocalDir   string        `json:"local_dir"`
	Operations []SyncOp      `json:"operations"`
	Summary    SyncPlanSummary `json:"summary"`
}

// SyncPlanInput provides all parameters needed to generate a sync plan.
type SyncPlanInput struct {
	Direction  SyncDirection
	Bucket     string
	Prefix     string
	LocalDir   string
	LocalFiles []LocalFileInfo
	Delete     bool
	Exclude    []string
	Include    []string
	Checksum   bool
}

// SyncResult holds the outcome of executing a sync plan.
type SyncResult struct {
	Succeeded int      `json:"succeeded"`
	Failed    int      `json:"failed"`
	Skipped   int      `json:"skipped"`
	Errors    []string `json:"errors,omitempty"`
}

// SyncProgress reports progress of a single operation during execution.
type SyncProgress struct {
	Current   int          `json:"current"`
	Total     int          `json:"total"`
	Action    SyncOpAction `json:"action"`
	Key       string       `json:"key"`
	Size      int64        `json:"size"`
	Error     string       `json:"error,omitempty"`
}

// StorageBackend abstracts R2 operations so SyncService is testable without
// live credentials.
type StorageBackend interface {
	ListRemoteObjects(ctx context.Context, bucket, prefix string) ([]ObjectInfo, error)
	UploadFile(ctx context.Context, bucket, key, localPath string) error
	DownloadFile(ctx context.Context, bucket, key, localPath string) error
	DeleteRemoteObject(ctx context.Context, bucket, key string) error
}

// SyncService synchronizes a local directory with an R2 bucket.
type SyncService struct {
	backend    StorageBackend
	OnProgress func(SyncProgress)
	// Concurrency caps the number of plan operations executed in parallel
	// (FEAT-038). Zero or 1 keeps execution strictly sequential — the
	// historical behavior and the default for library callers who want
	// zero surprise. Values >1 run a bounded worker pool; progress
	// callbacks fire in completion order with Current = completed count.
	Concurrency int
}

// NewSyncService creates a new SyncService with the given storage backend.
func NewSyncService(backend StorageBackend) *SyncService {
	return &SyncService{
		backend: backend,
	}
}

// Plan compares local filesystem state against R2 object listing and produces
// a SyncPlan describing all operations needed to synchronize them.
func (s *SyncService) Plan(ctx context.Context, input SyncPlanInput) (*SyncPlan, error) {
	if input.Bucket == "" {
		return nil, validationError("SyncService.Plan", "bucket is required")
	}
	if input.LocalDir == "" {
		return nil, validationError("SyncService.Plan", "local directory is required")
	}

	// List remote objects under the prefix
	remoteObjects, err := s.backend.ListRemoteObjects(ctx, input.Bucket, input.Prefix)
	if err != nil {
		return nil, newError("SyncService.Plan", "failed to list remote objects", err)
	}

	// Build lookup maps
	remoteByKey := make(map[string]ObjectInfo, len(remoteObjects))
	for _, obj := range remoteObjects {
		// Strip prefix to get the relative key for comparison
		relKey := obj.Key
		if input.Prefix != "" {
			relKey = strings.TrimPrefix(obj.Key, input.Prefix)
		}
		remoteByKey[relKey] = obj
	}

	localByRel := make(map[string]LocalFileInfo, len(input.LocalFiles))
	for _, f := range input.LocalFiles {
		localByRel[f.RelPath] = f
	}

	var ops []SyncOp

	switch input.Direction {
	case SyncUp:
		ops = s.planUp(input, localByRel, remoteByKey)
	case SyncDown:
		ops = s.planDown(input, localByRel, remoteByKey)
	}

	// Compute summary
	summary := SyncPlanSummary{}
	for _, op := range ops {
		switch op.Action {
		case SyncOpUpload:
			summary.Uploads++
			summary.TotalSize += op.Size
		case SyncOpDownload:
			summary.Downloads++
			summary.TotalSize += op.Size
		case SyncOpDelete:
			summary.Deletes++
		case SyncOpSkip:
			summary.Skips++
		}
	}

	return &SyncPlan{
		Bucket:     input.Bucket,
		Prefix:     input.Prefix,
		Direction:  input.Direction,
		LocalDir:   input.LocalDir,
		Operations: ops,
		Summary:    summary,
	}, nil
}

// planUp produces operations for uploading local files to R2.
func (s *SyncService) planUp(input SyncPlanInput, localByRel map[string]LocalFileInfo, remoteByKey map[string]ObjectInfo) []SyncOp {
	var ops []SyncOp

	// Check each local file
	for relPath, local := range localByRel {
		if isExcluded(relPath, input.Exclude) {
			continue
		}
		if !isIncluded(relPath, input.Include) {
			continue
		}

		remoteKey := relPath
		if input.Prefix != "" {
			remoteKey = input.Prefix + relPath
		}
		localPath := filepath.Join(input.LocalDir, relPath)

		remote, exists := remoteByKey[relPath]
		if !exists {
			// New file — upload
			ops = append(ops, SyncOp{
				Action:    SyncOpUpload,
				Key:       remoteKey,
				LocalPath: localPath,
				Size:      local.Size,
				Reason:    "new file",
			})
			continue
		}

		// File exists remotely — check if changed
		if s.isUnchanged(local, remote, input.Checksum) {
			ops = append(ops, SyncOp{
				Action: SyncOpSkip,
				Key:    remoteKey,
				Size:   local.Size,
				Reason: "unchanged",
			})
		} else {
			ops = append(ops, SyncOp{
				Action:    SyncOpUpload,
				Key:       remoteKey,
				LocalPath: localPath,
				Size:      local.Size,
				Reason:    "modified",
			})
		}
	}

	// Check for remote objects that don't exist locally (delete if --delete)
	if input.Delete {
		for relKey, remote := range remoteByKey {
			// Excluded paths are outside the sync universe: never delete-eligible
			// (BUG-041 — .env/.git/ patterns must survive --delete).
			if isExcluded(relKey, input.Exclude) {
				continue
			}
			// Outside the include universe: not managed, never delete-eligible
			// (BUG-051).
			if !isIncluded(relKey, input.Include) {
				continue
			}
			if _, exists := localByRel[relKey]; !exists {
				remoteKey := relKey
				if input.Prefix != "" {
					remoteKey = input.Prefix + relKey
				}
				ops = append(ops, SyncOp{
					Action: SyncOpDelete,
					Key:    remoteKey,
					Size:   remote.Size,
					Reason: "not in local",
				})
			}
		}
	}

	return ops
}

// planDown produces operations for downloading R2 objects to local.
func (s *SyncService) planDown(input SyncPlanInput, localByRel map[string]LocalFileInfo, remoteByKey map[string]ObjectInfo) []SyncOp {
	var ops []SyncOp

	// Check each remote object
	for relKey, remote := range remoteByKey {
		if isExcluded(relKey, input.Exclude) {
			continue
		}
		if !isIncluded(relKey, input.Include) {
			continue
		}

		localPath := filepath.Join(input.LocalDir, relKey)
		remoteKey := relKey
		if input.Prefix != "" {
			remoteKey = input.Prefix + relKey
		}

		local, exists := localByRel[relKey]
		if !exists {
			// New file — download
			ops = append(ops, SyncOp{
				Action:        SyncOpDownload,
				Key:           remoteKey,
				LocalPath:     localPath,
				Size:          remote.Size,
				Reason:        "new file",
				RemoteModTime: remote.LastModified,
			})
			continue
		}

		// File exists locally — check if changed
		if s.isUnchanged(local, remote, input.Checksum) {
			ops = append(ops, SyncOp{
				Action: SyncOpSkip,
				Key:    remoteKey,
				Size:   remote.Size,
				Reason: "unchanged",
			})
		} else {
			ops = append(ops, SyncOp{
				Action:        SyncOpDownload,
				Key:           remoteKey,
				LocalPath:     localPath,
				Size:          remote.Size,
				Reason:        "modified",
				RemoteModTime: remote.LastModified,
			})
		}
	}

	// Check for local files that don't exist remotely (delete if --delete)
	if input.Delete {
		for relPath, local := range localByRel {
			// Excluded paths are outside the sync universe: never delete-eligible
			// (BUG-041 — a down-sync --delete must not remove excluded files).
			if isExcluded(relPath, input.Exclude) {
				continue
			}
			// Outside the include universe: not managed, never delete-eligible
			// (BUG-051).
			if !isIncluded(relPath, input.Include) {
				continue
			}
			if _, exists := remoteByKey[relPath]; !exists {
				ops = append(ops, SyncOp{
					Action:    SyncOpDelete,
					Key:       relPath,
					LocalPath: filepath.Join(input.LocalDir, relPath),
					Size:      local.Size,
					Reason:    "not in remote",
				})
			}
		}
	}

	return ops
}

// isUnchanged determines whether a local file and remote object are identical.
// In checksum mode it compares ETag vs local checksum; otherwise it uses
// size + modification time heuristic.
func (s *SyncService) isUnchanged(local LocalFileInfo, remote ObjectInfo, checksum bool) bool {
	if checksum {
		// Compare checksums: R2 ETags are MD5 wrapped in quotes.
		remoteHash := strings.Trim(remote.ETag, "\"")
		// Multipart uploads append a "-N" part-count suffix, and the digest is
		// MD5-of-part-MD5s, not the content MD5 — a local checksum can never
		// match. Fall back to the size/mtime heuristic instead of forcing a
		// perpetual re-transfer.
		if !strings.Contains(remoteHash, "-") {
			return local.Checksum == remoteHash
		}
	}
	// Default: same size and local is not newer than remote
	return local.Size == remote.Size && !local.ModTime.After(remote.LastModified)
}

// isExcluded checks if a relative path matches any exclude glob patterns.
// A pattern ending in "/" is a directory prefix: it excludes every path
// nested under that directory, at any depth (e.g. ".git/" matches
// ".git/objects/ab/cdef").
// isIncluded reports whether a relative path falls inside the include
// filter's sync universe (BUG-051: SyncPlanInput.Include was previously
// never read — --include was a silent no-op). An empty filter includes
// everything. Pattern semantics mirror isExcluded: a trailing "/" is a
// directory prefix; otherwise a glob matched against the full relative path
// and the basename. Paths outside the universe are neither synced nor
// delete-eligible.
func isIncluded(relPath string, includes []string) bool {
	if len(includes) == 0 {
		return true
	}
	for _, pattern := range includes {
		if strings.HasSuffix(pattern, "/") && strings.HasPrefix(relPath, pattern) {
			return true
		}
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
		if matched, _ := filepath.Match(pattern, filepath.Base(relPath)); matched {
			return true
		}
	}
	return false
}

func isExcluded(relPath string, excludes []string) bool {
	for _, pattern := range excludes {
		// Directory prefix match (pattern ends with "/")
		if strings.HasSuffix(pattern, "/") && strings.HasPrefix(relPath, pattern) {
			return true
		}
		// Match against the full relative path
		if matched, _ := filepath.Match(pattern, relPath); matched {
			return true
		}
		// Also match against just the filename
		if matched, _ := filepath.Match(pattern, filepath.Base(relPath)); matched {
			return true
		}
	}
	return false
}

// Execute runs all operations in a SyncPlan against the storage backend.
func (s *SyncService) Execute(ctx context.Context, plan *SyncPlan) (*SyncResult, error) {
	if s.Concurrency > 1 && len(plan.Operations) > 1 {
		return s.executeConcurrent(ctx, plan)
	}
	return s.executeSequential(ctx, plan)
}

// executeSequential is the historical single-worker loop.
func (s *SyncService) executeSequential(ctx context.Context, plan *SyncPlan) (*SyncResult, error) {
	result := &SyncResult{}
	total := len(plan.Operations)

	for i, op := range plan.Operations {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		var opErr error

		opErr = s.runOperation(ctx, plan, op)
		switch op.Action {
		case SyncOpSkip:
			result.Skipped++
			s.reportProgress(SyncProgress{
				Current: i + 1,
				Total:   total,
				Action:  op.Action,
				Key:     op.Key,
				Size:    op.Size,
			})
			continue
		}

		if opErr != nil {
			result.Failed++
			result.Errors = append(result.Errors, opErr.Error())
			s.reportProgress(SyncProgress{
				Current: i + 1,
				Total:   total,
				Action:  op.Action,
				Key:     op.Key,
				Size:    op.Size,
				Error:   opErr.Error(),
			})
		} else {
			result.Succeeded++
			s.reportProgress(SyncProgress{
				Current: i + 1,
				Total:   total,
				Action:  op.Action,
				Key:     op.Key,
				Size:    op.Size,
			})
		}
	}

	return result, nil
}

// runOperation executes a single non-skip plan operation and returns its
// error. Shared by the sequential and concurrent executors so both paths
// have identical semantics.
func (s *SyncService) runOperation(ctx context.Context, plan *SyncPlan, op SyncOp) error {
	switch op.Action {
	case SyncOpUpload:
		return s.backend.UploadFile(ctx, plan.Bucket, op.Key, op.LocalPath)
	case SyncOpDownload:
		if err := s.backend.DownloadFile(ctx, plan.Bucket, op.Key, op.LocalPath); err != nil {
			return err
		}
		if !op.RemoteModTime.IsZero() {
			// Restore the remote timestamp so the next plan's mtime compare
			// treats the file as unchanged instead of perpetually newer.
			_ = os.Chtimes(op.LocalPath, op.RemoteModTime, op.RemoteModTime)
		}
		return nil
	case SyncOpDelete:
		// Delete is direction-aware: a down-sync delete removes the stale
		// LOCAL file; an up-sync delete removes the remote object. Calling
		// DeleteRemoteObject for a down-sync op would target an unprefixed
		// key and could destroy an unrelated bucket-root object (BUG-040).
		if plan.Direction == SyncDown {
			return os.Remove(op.LocalPath)
		}
		return s.backend.DeleteRemoteObject(ctx, plan.Bucket, op.Key)
	default: // SyncOpSkip — callers count it, no work to do
		return nil
	}
}

// executeConcurrent runs plan operations on a bounded worker pool
// (FEAT-038). Result semantics match the sequential executor: every op is
// attempted (no early exit on op error), errors are collected, and
// cancellation aborts remaining work with ctx's error. Progress callbacks
// fire in completion order with Current = completed-op count.
func (s *SyncService) executeConcurrent(ctx context.Context, plan *SyncPlan) (*SyncResult, error) {
	total := len(plan.Operations)
	workers := s.Concurrency
	if workers > total {
		workers = total
	}

	jobs := make(chan int, total)
	for i := range plan.Operations {
		jobs <- i
	}
	close(jobs)

	var (
		mu       sync.Mutex
		completed int64
		wg       sync.WaitGroup
	)
	result := &SyncResult{}

	report := func(op SyncOp, done int64, errMsg string) {
		s.reportProgress(SyncProgress{
			Current: int(done),
			Total:   total,
			Action:  op.Action,
			Key:     op.Key,
			Size:    op.Size,
			Error:   errMsg,
		})
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				if ctx.Err() != nil {
					return
				}
				op := plan.Operations[i]

				if op.Action == SyncOpSkip {
					mu.Lock()
					result.Skipped++
					done := atomic.AddInt64(&completed, 1)
					report(op, done, "")
					mu.Unlock()
					continue
				}

				opErr := s.runOperation(ctx, plan, op)

				mu.Lock()
				if opErr != nil {
					result.Failed++
					result.Errors = append(result.Errors, opErr.Error())
				} else {
					result.Succeeded++
				}
				done := atomic.AddInt64(&completed, 1)
				errMsg := ""
				if opErr != nil {
					errMsg = opErr.Error()
				}
				report(op, done, errMsg)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if ctx.Err() != nil {
		return result, ctx.Err()
	}
	return result, nil
}

// reportProgress sends a progress event if a callback is registered.
func (s *SyncService) reportProgress(p SyncProgress) {
	if s.OnProgress != nil {
		s.OnProgress(p)
	}
}
