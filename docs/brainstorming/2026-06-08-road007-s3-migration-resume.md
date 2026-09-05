---
title: "ROAD-007: S3 to R2 Migration with Resume Support"
created: 2026-06-08T02:00:00-03:00
status: approved
roadmap: ROAD-007
origin: ROAD-007
plan_ref: docs/planning-mode/2026-06-08-road007-s3-migration-resume.md
last_reviewed: 2026-06-08T03:00:00-03:00
last_review_ref: docs/independent-reviews/2026-06-08T03-00-00-road007-s3-migration-resume.md
last_review_findings: 5
deliverables:
  - BR-01: Checkpoint persistence in ~/.cosmoflare/migrations/
  - BR-02: Worker pool with concurrent S3→R2 streaming transfers
  - BR-03: Per-object retry with exponential backoff (3 attempts)
  - BR-04: Resume from checkpoint (--resume flag)
  - BR-05: Graceful SIGINT shutdown with checkpoint save
  - BR-06: Periodic checkpoint flush (every 10 objects or 30s)
  - BR-07: ETag verification mode (--verify)
---

# ROAD-007: S3 to R2 Migration with Resume Support

## Summary

Replace the simulated transfer loop in `internal/migration/s3.go` with a real S3→R2 streaming transfer engine. Add checkpoint persistence for resume across interrupted runs, a worker pool for concurrent transfers, per-object retry with backoff, and graceful SIGINT handling. Builds on ROAD-019 (basic S3 migration command structure).

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Scope | Full implementation (real transfers + resume) | Resume is meaningless without real transfers — inseparable. |
| Checkpoint location | `~/.cosmoflare/migrations/{hash}.json` | Survives reboots, doesn't pollute project dirs, centralized. |
| Concurrency model | Worker pool (goroutines + channel) | Clean shutdown, natural checkpoint pairing, standard Go pattern. |
| Failure handling | Retry 3x + skip | Large migrations always have a few failures. Skip and log. `--resume` retries on next run. |

## Architecture

### Component Layout

Three components in `internal/migration/`:

| File | Responsibility |
|------|---------------|
| `checkpoint.go` | **New.** Load/save/delete checkpoint state. Hash generation for deterministic file names. Periodic flush logic. |
| `worker.go` | **New.** Worker pool: N goroutines pulling objects from channel. Single-object S3 GetObject → R2 Upload stream. Retry with backoff. Results channel. |
| `s3.go` | **Modified.** `Execute()` becomes orchestrator: load checkpoint → filter done → feed workers → collect results → update checkpoint → shutdown. |

### Transfer Flow

```
Execute(r2Client)
  ├─ listS3Objects() → full object list
  ├─ loadCheckpoint() → set of completed keys
  ├─ remaining = all objects - completed keys
  │   (failed keys from prior run ARE retried)
  ├─ if len(remaining) == 0 → print "nothing to do", delete checkpoint, return
  ├─ print resume summary if checkpoint existed
  ├─ spawn N workers (--concurrency flag, default 10)
  ├─ register SIGINT handler
  ├─ feed remaining objects into work channel
  ├─ collect results from results channel:
  │   ├─ success → mark completed in checkpoint
  │   └─ failure (after 3 retries) → add to failed list, continue
  ├─ flush checkpoint periodically (every 10 completions or 30s)
  ├─ on SIGINT:
  │   ├─ close work channel (no new work)
  │   ├─ workers finish current transfer
  │   ├─ save checkpoint
  │   ├─ print partial summary
  │   └─ exit code 1
  └─ on completion:
      ├─ all succeeded → delete checkpoint, print summary
      └─ some failed → save checkpoint, print summary with failure count
```

### Checkpoint File

Location: `~/.cosmoflare/migrations/{hash}.json`

Hash: SHA256 of `s3bucket + ":" + r2bucket + ":" + filter`, truncated to 16 hex chars. Deterministic — same inputs always find the same checkpoint.

```json
{
  "version": 1,
  "s3_bucket": "source-bucket",
  "r2_bucket": "dest-bucket",
  "filter": "images/*",
  "started_at": "2026-06-08T10:00:00-03:00",
  "updated_at": "2026-06-08T10:15:30-03:00",
  "total_objects": 10000,
  "total_size": 5368709120,
  "completed": {
    "images/a.jpg": 1048576,
    "images/b.png": 2097152
  },
  "failed": ["images/broken.dat"],
  "transferred_size": 3145728
}
```

The `completed` map stores key→size for fast lookup. On `--resume`, completed keys are skipped. Failed keys are retried (may have been transient).

### Checkpoint Flush Strategy

Writing after every object creates an I/O bottleneck for fast small-file migrations. Instead:
- Flush every 10 successful transfers, OR
- Flush every 30 seconds (whichever comes first)
- Always flush on SIGINT/shutdown
- Always flush on completion

A `sync.Mutex` protects the checkpoint map. The flush timer runs in its own goroutine.

### Worker Pool

```go
type transferResult struct {
    Key   string
    Size  int64
    Err   error
}
```

The worker function signature receives both clients: `func transferWorker(s3Client *s3.Client, r2Client cosmoflare.R2Client, s3Bucket, r2Bucket string, work <-chan *S3Object, results chan<- transferResult, wg *sync.WaitGroup)`.

Each worker goroutine:
1. Pull `*S3Object` from `work chan *S3Object`
2. `s3Client.GetObject(ctx, &s3.GetObjectInput{Bucket: s3Bucket, Key: key})` → `io.ReadCloser` body + content length
3. Stream body to `r2Client.Upload(ctx, r2Bucket, key, body, size)` — no temp file. For objects > 100MB, use `r2Client.MultipartUpload` instead (the library does NOT auto-switch).
4. On success → send `transferResult{Key, Size, nil}` to results channel
5. On failure → retry up to 3 times (backoff: 1s, 4s, 16s). If all fail → send `transferResult{Key, 0, err}` to results channel

Workers use a per-transfer context with a size-scaled timeout: `max(5 minutes, size / 1MB/s)` — ensures large objects (10GB+) have sufficient time. NOT the global shutdown context, so in-flight transfers complete on SIGINT.

**Checkpoint corruption safety:** `saveCheckpoint` writes to a temp file then renames (atomic on POSIX). If a crash occurs during write, the old checkpoint survives. `loadCheckpoint` returns an error on corrupt JSON — `Execute()` treats this as "no checkpoint" and starts fresh, logging a warning.

### Graceful Shutdown

Register SIGINT/SIGTERM via `os/signal.Notify`. On signal:
1. Set a `stopping` atomic bool → feeder stops sending new work
2. Close work channel → workers see channel closed after finishing current transfer
3. Wait for workers to exit (WaitGroup) — they complete their in-flight transfer, not abort mid-stream
4. Save checkpoint with current state
5. Print partial summary: `"Interrupted: X/Y transferred. Run with --resume to continue."`
6. Return error (cmd layer maps to exit code 1)

### --resume Flag

`S3Migration.Resume` already exists as a bool field. Behavior:
- `true` → `loadCheckpoint()`. If found, skip completed keys, print: `"Resuming: X/Y already transferred (Z bytes)"`. If no checkpoint found, start fresh (not an error).
- `false` (default) → start fresh. If a stale checkpoint exists, warn: `"Stale checkpoint found. Use --resume to continue or delete ~/.cosmoflare/migrations/{hash}.json to start over."`

### --verify Flag

`S3Migration.Verify` already exists. When true, after the transfer phase completes:
- For each transferred object, compare the S3 ETag with the R2 object's ETag (via `r2Client.HeadObject`)
- Mismatches logged as warnings: `"⚠ ETag mismatch: images/a.jpg (S3: abc, R2: def)"`
- Verification runs with the same worker pool concurrency
- Mismatch count included in the summary

### Progress Bar

Keep the existing `cheggaaa/pb/v3` progress bar. Update it from the results collector goroutine (not from workers — single writer). Show: objects completed / total, bytes transferred, elapsed time, ETA.

## File Changes

| File | Change |
|------|--------|
| `internal/migration/checkpoint.go` | **New.** `Checkpoint` struct, `loadCheckpoint`, `saveCheckpoint`, `deleteCheckpoint`, `checkpointPath`, hash generation, periodic flusher. |
| `internal/migration/checkpoint_test.go` | **New.** Tests for save/load/delete round-trip, hash determinism, completed key lookup. |
| `internal/migration/worker.go` | **New.** `transferWorker` function, `transferResult` type, retry with backoff. |
| `internal/migration/worker_test.go` | **New.** Tests for retry logic, result channel behavior. |
| `internal/migration/s3.go` | Replace simulated loop (lines 112-156) with real orchestrator: checkpoint load, worker pool dispatch, result collection, periodic flush, SIGINT handling, verify mode. |
| `internal/migration/s3_test.go` | **New.** Tests for Execute with mock clients. |
| `cmd/migrate.go` | No changes — already passes all flags to S3Migration. |

## Out of Scope

- Bandwidth throttling (`--bandwidth-limit`)
- Compression during transfer
- Delete source after transfer
- Incremental sync (changed objects only)
- Cross-region optimization
- Multipart transfer for large individual objects (R2 Upload handles this internally)

## Estimated Effort

~150 lines `checkpoint.go`, ~120 lines `worker.go`, ~100 lines modified in `s3.go`, ~200 lines tests. Moderate.
