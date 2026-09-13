# Agent 2: Core Logic — 5/10

**Sub-scores**:
| Dimension | Score | Notes |
|-----------|-------|-------|
| bug_detection | 5/10 | Upload/multipart core is defensively written (BUG-026/BUG-028 fixes visible), but the sync engine carries two data-destroying bugs and a silently ignored `--include` flag |
| performance | 5/10 | Sync execution is fully sequential; 30s shared HTTP timeout caps every transfer; existence checks list all buckets; no bounded worker pool anywhere in sync |
| edge_cases | 6/10 | Reader-shorter-than-size, unreadable-subtree deletion guard, and multipart ETag fallback are genuinely well handled; CopySource encoding, symlinks, duplicate KV titles, unprefixed keys are not |
| security | 6/10 | SQL splitter is a proper lexer (no injection); guardrails are upload-only — deletes/copies bypass them; machine config tokens written 0644 before chmod 0600 |
| data_integrity | 4/10 | `sync down --delete` no-ops locally AND can delete the wrong remote object; partial downloads truncate pre-existing files; aborted multiparts leak; `apply` deletes by omission |

**Scope**: 18 files read in depth — `pkg/cosmoflare/{client,options,upload,download,storage,sync,guardrails,watcher,diff,apply,retry,rest_client,config,d1_sql_splitter,multipart,cost}.go`, `cmd/sync.go`, `cmd/serve.go`, plus `cmd/installer_tui/main.go` and `cmd/watch.go` greps.

**Data flow traced**: `main.go` → `cmd/root.go` → command handlers → `pkg/cosmoflare` services. The "engine" is three subsystems: (1) the S3/R2 data path (`client.go` → AWS SDK v2 → upload/download/multipart), (2) the sync planner/executor (`sync.go` + `cmd/sync.go` backend + `watcher.go`), (3) the reconciliation pair (`diff.go` → `apply.go`) over `.cosmoflare.yaml`.

---

## Critical Findings

### 1. `sync down --delete` never deletes local files — and can delete the wrong REMOTE object

- **Severity**: high
- **File**: `pkg/cosmoflare/sync.go:344-357` (plan), `pkg/cosmoflare/sync.go:425-426` (execute)
- **Evidence**: `planDown` emits `SyncOpDelete` meaning "delete local file not in remote":

```go
// pkg/cosmoflare/sync.go:344-357
if input.Delete {
    for relPath, local := range localByRel {
        if _, exists := remoteByKey[relPath]; !exists {
            ops = append(ops, SyncOp{
                Action:    SyncOpDelete,
                Key:       relPath,                    // NOT prefixed
                LocalPath: filepath.Join(input.LocalDir, relPath),
                Reason:    "not in remote",
```

But `Execute` handles every `SyncOpDelete` identically, regardless of direction:

```go
// pkg/cosmoflare/sync.go:425-426
case SyncOpDelete:
    opErr = s.backend.DeleteRemoteObject(ctx, plan.Bucket, op.Key, ...)
```

Verified `os.Remove` appears nowhere in `cmd/sync.go`, `cmd/watch.go`, or `pkg/cosmoflare/sync.go` (grep). Two consequences:
1. The documented `--delete` behavior for `sync down` silently does nothing to local files — and the op is counted as `Succeeded` in `SyncResult`, so the CLI prints "Sync completed successfully!"
2. `DeleteRemoteObject` is called with the **unprefixed** relPath. With `cosmoflare sync down bucket/site/ ./out --delete`, a local-only file `extra.txt` triggers `DeleteObject(bucket, "extra.txt")` — deleting an unrelated object at the bucket **root** outside the synced prefix if one happens to share the name.

- **Fix**: In `Execute`, branch on `plan.Direction`: for `SyncDown`, `SyncOpDelete` must `os.Remove(op.LocalPath)` (never a remote call). Add a regression test that runs a down-sync with `Delete: true` and asserts zero `DeleteRemoteObject` calls plus actual file removal.

### 2. Shared 30s HTTP client timeout kills every large transfer

- **Severity**: high
- **File**: `pkg/cosmoflare/client.go:111-114` and `client.go:156-173`
- **Evidence**:

```go
// client.go:111-114
httpClient := cfg.httpClient
if httpClient == nil {
    httpClient = &http.Client{Timeout: cfg.timeout}   // default 30s (options.go:48, client.go:77)
}
...
// client.go:158 — same client handed to the S3 SDK:
awsconfig.WithHTTPClient(c.httpClient)
```

`http.Client.Timeout` covers the entire request-response cycle **including body transfer**. `Download` (download.go:59-80) and the sync backend's `DownloadFile` (cmd/sync.go:528-541) stream `result.Body` via `io.Copy` under this deadline. A 500MB object at 50 Mbps (~80s) fails mid-body with "Client.Timeout exceeded while reading body" — `sync down` of any sizable site is structurally broken, and single PUTs up to the 100MB threshold (upload.go:86) fail on slower links. API-call-scale defaults were applied to a data-transfer transport.

- **Fix**: Give the S3 transport its own client with no `Timeout` (deadline control belongs to the per-op `ctx` plus the SDK's built-in retries); keep the 30s client for the Cloudflare API and `restClient`. A library test with a slow body reader (`httptest` + throttled pipe) would pin this.

### 3. Multipart abort uses the already-canceled request context — orphaned uploads accumulate

- **Severity**: high
- **File**: `pkg/cosmoflare/upload.go:180-186`, called at `upload.go:209`, `246-247`, `266-268`; same pattern `pkg/cosmoflare/multipart.go:466-469`
- **Evidence**:

```go
// upload.go:180-186
abort := func() {
    c.s3Client().AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{   // ctx == request ctx
        Bucket: aws.String(bucket), ...
    })
}
...
// upload.go:246-247 — invoked right after a part failed:
if r.err != nil {
    abort()   // the most common failure cause is ctx cancellation itself
```

When the context is canceled (Ctrl-C mid-upload, timeout), part uploads fail, `abort()` runs **with the same dead context**, the abort request fails instantly, and the error is discarded. The incomplete multipart upload stays in R2 — billed as storage and invisible in object listings. Repeat interrupts = silent accumulation.

- **Fix**: `abortCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)` captured in the closure; log (at minimum) an abort failure. Same fix in `multipart.go`.

### 4. `sync up --delete` deletes remote objects that match the exclude patterns

- **Severity**: high (data loss)
- **File**: `pkg/cosmoflare/sync.go:273-289`
- **Evidence**: The local scan already drops excluded files (`cmd/sync.go:174` `isLocalPathExcluded`), so they are absent from `localByRel`. The delete loop then treats every remote key not in `localByRel` as garbage — without consulting `isExcluded`:

```go
// pkg/cosmoflare/sync.go:274-288
if input.Delete {
    for relKey, remote := range remoteByKey {
        if _, exists := localByRel[relKey]; !exists {   // no isExcluded(relKey, input.Exclude) check
            ops = append(ops, SyncOp{Action: SyncOpDelete, Key: ..., Reason: "not in local"})
```

With the default excludes (`cmd/sync.go:55`: `.git/`, `.env`, `.env.*`, `.DS_Store`), any remote `.env` (e.g. uploaded before excludes existed, or written by a deploy pipeline) is destroyed on the next `sync up --delete`. rsync's default semantics deliberately protect excluded files from `--delete`.

- **Fix**: `if isExcluded(relKey, input.Exclude) { continue }` at the top of the delete loop in both `planUp` and (symmetrically) `planDown`.

### 5. `--include` flag is a silent no-op

- **Severity**: medium
- **File**: `cmd/sync.go:120,127,278,353`; `pkg/cosmoflare/sync.go:104-115`
- **Evidence**: `SyncPlanInput.Include` is populated from the CLI flag but grep shows **zero non-test reads** of `.Include` anywhere in `pkg/cosmoflare` — neither `planUp` nor `planDown` consults it. `cosmoflare sync up ./site bucket --include "*.html"` uploads the entire tree (minus excludes) while the user believes the scope was narrowed — a surprise with billing implications and a `--help` contract violation.
- **Fix**: Apply include semantics after exclusion (a file is synced only if it matches at least one include pattern), or remove the flag until implemented.

### 6. `apply` destroys any account resource not declared in the config (delete-by-omission)

- **Severity**: medium
- **File**: `pkg/cosmoflare/diff.go:241-260`, `pkg/cosmoflare/apply.go:422-429`
- **Evidence**: `CompareR2` marks **every** live bucket absent from `.cosmoflare.yaml` as a deletion; `applyR2Changes` executes `DeleteBucket` on each. There is no `managed_by` marker or import model. A project config declaring one `assets` bucket in an account holding ten buckets (other projects, other tools) will attempt to delete the other nine. `cmd/apply.go` has a confirmation prompt, but `--yes`/`--json` CI flows skip it, and the prompt lists what may be dozens of resources. Non-empty buckets fail (accidental safety); empty ones die silently.
- **Fix**: Adopt an explicit-ownership model — only delete resources created by cosmoflare (tagged/tracked in state) or require `--delete-unmanaged` opt-in.

### 7. KV namespaces assumed unique by title

- **Severity**: medium
- **File**: `pkg/cosmoflare/diff.go:279-315`, `pkg/cosmoflare/apply.go:472-500`
- **Evidence**: Both the diff map and the delete-resolution loop key on `ns.Title`. Cloudflare permits duplicate namespace titles; with two "cache" namespaces, the diff collapses them into one entry (false drift) and `apply`'s title scan (`apply.go:482-487`) deletes the **first** match — possibly the wrong namespace.
- **Fix**: Track namespace IDs in config (or match by ID when present); refuse to act on ambiguous titles.

### 8. Machine config tokens written world-readable before chmod

- **Severity**: medium
- **File**: `pkg/cosmoflare/config.go:164-173`
- **Evidence**:

```go
if err := v.WriteConfigAs(configPath); err != nil { ... }   // creates file with umask default (0644)
return os.Chmod(configPath, 0600)                            // race window; skipped entirely on crash between the two
```

`~/.r2go2/config.yaml` holds `api_token`/`secret_key` for every profile. A crash (or kill) between write and chmod leaves credentials readable by every local user; even the happy path has a TOCTOU window.
- **Fix**: Marshal to bytes and write via `os.OpenFile(path, O_CREATE|O_WRONLY|O_TRUNC, 0600)`; keep the chmod as belt-and-braces.

### 9. CopySource not URL-encoded — copies fail on keys with special characters

- **Severity**: medium
- **File**: `pkg/cosmoflare/storage.go:249-253`
- **Evidence**: `copySource := fmt.Sprintf("%s/%s", srcBucket, srcKey)` is sent raw as `x-amz-copy-source`. The S3 spec requires the source to be URL-encoded; keys containing spaces, `+`, `%`, `?`, or non-ASCII bytes produce 404s/invalid copies. Affects `cosmoflare copy` and anything built on `CopyObject`.
- **Fix**: `copySource := srcBucket + "/" + url.PathEscape(srcKey)` (bucket names are restricted to safe chars already).

### 10. Interrupted downloads truncate pre-existing local files

- **Severity**: medium
- **File**: `pkg/cosmoflare/download.go:92-101`, `cmd/sync.go:534-541`
- **Evidence**: Both `writeToFile` and `DownloadFile` `os.Create`/`os.OpenFile` the final destination, then stream. Any mid-copy failure (timeout from Finding 2, network drop) leaves a truncated file that has already destroyed the previous version. The next `sync down` plan then sees a size mismatch and re-downloads — but local edits made to the previous copy are gone. `writeToFile` also ignores the `f.Close()` error (`defer f.Close()`), losing flush failures.
- **Fix**: Write to `path + ".part"` in the same directory, `f.Sync()`, check close, then `os.Rename` (atomic on POSIX).

### 11. Guardrails protect uploads only — deletes and copies bypass them entirely

- **Severity**: medium
- **File**: `pkg/cosmoflare/guardrails.go:71-82` (only `enforceUploadGuardrails`), `pkg/cosmoflare/storage.go:219-235` (`DeleteObject` unguarded)
- **Evidence**: `blocked_keys`/`allowed_buckets` are enforced in `Upload`/`MultipartUpload`/`ResumableMultipartUpload` (upload.go:76,140,395) but `DeleteObject` and `CopyObject` perform no guardrail check at all. A project guardrail meant to fence a bucket is trivially sidestepped by deleting its contents — arguably the more dangerous operation.
- **Fix**: Add `enforceDeleteGuardrails`/`enforceCopyGuardrails` mirroring the upload check.

### 12. Diff silently skips drift for services the config doesn't declare

- **Severity**: low
- **File**: `pkg/cosmoflare/diff.go:414-444` (`CompareAll` gates each service on non-empty local config)
- **Evidence**: A config with no `workers:` section produces no worker diff even if 40 workers are deployed. Combined with Finding 6 this is inconsistent: declared-but-empty reports nothing, declared-and-partially-populated reports deletions of everything else. The reconciliation engine under-reports by construction.
- **Fix**: Run comparisons for services explicitly enabled in config (e.g. `workers: managed: true`), and report skipped services as "unmanaged" in the summary.

### 13. Sync execution is strictly sequential — no concurrency

- **Severity**: low (performance)
- **File**: `pkg/cosmoflare/sync.go:404-463`
- **Evidence**: `Execute` processes `plan.Operations` one at a time; each file pays full RTT + upload time. A 5,000-file site at ~150ms/op = 12+ minutes where 8 workers would take ~2. `MultipartUpload` already proves the codebase can do bounded concurrency (sem channel, upload.go:201).
- **Fix**: Bounded worker pool (semaphore of N, default 4-8) over the op slice, preserving `result.Errors` ordering or sorting before report. Estimated impact: 4-8x wall-clock on large syncs.

### 14. Misc lower-severity items

- **Retry sleeps not context-aware** — `rest_client.go:263` `time.Sleep` between retries ignores Ctrl-C during a 30s backoff; `parseRetryAfter` (rest_client.go:106-124) accepts unbounded values (`Retry-After: 86400` → sleeps a day) — cap at `maxRetryBackoff`. (low)
- **`retry.go` is dead exported code** — "currently unwired" per rest_client.go:50-51; `retry.Do` has zero production callers. (low)
- **Raw `cmd.Start()` fire-and-forget** — `cmd/installer_tui/main.go:1280` violates the project spawn policy (ROAD-525 lint hit); URL is a constant so no injection, but child is untracked. (low)
- **Existence checks list every bucket** — `GetBucket`/`BucketExists` (storage.go:56-102) full-list + linear scan per call. (low)
- **Symlinks report the link's size** — `cmd/sync.go:154-194` lstat semantics: declared size ≠ content size; BUG-026 aborts the upload with a confusing error instead of following the link. Same in `watcher.go:136`. (low)
- **Library `Plan()` doesn't normalize the prefix** — `pkg/cosmoflare/sync.go:177-181` TrimPrefix mapping is only safe with a trailing-`/` prefix; CLI normalizes (cmd/sync.go:139-141), library callers can corrupt key mapping. (low)
- **`cmd/serve.go:117` http.Server has no Read/Write/Idle timeouts** — localhost + token auth limits exposure; hung local clients still hold goroutines. (low)

### Verified-solid areas

`readUploadPart` (upload.go:291-312, BUG-026 truncated-reader guard) and the watcher unreadable-subtree guard (watcher.go:136-147, BUG-028) are exactly right. `d1_sql_splitter.go` is a real lexer — no naive `Split(";")` breakage. `rest_client.go` retry policy (429-always, 502-504 idempotent-only, Retry-After honored) and `cmd/serve.go` shutdown/handshake are correct; only the sleep-loop details above need work.

---

## Recommendations

- [ ] Fix `Execute` to branch on `plan.Direction` for delete ops (local remove for down-sync); add regression test asserting zero remote deletes in a down-sync (effort: small)
- [ ] Give the S3 transport its own timeout-free `http.Client`; keep 30s for API clients (effort: small)
- [ ] Abort multiparts with `context.WithoutCancel(ctx)` and surface abort failures (effort: small)
- [ ] Skip excluded keys in the `--delete` loops of `planUp`/`planDown`; add test with remote `.env` + default excludes (effort: small)
- [ ] Implement or remove `--include` (effort: medium)
- [ ] Atomic temp-file+rename writes for all downloads; check `Close()` errors (effort: small)
- [ ] Extend guardrail enforcement to delete/copy paths (effort: medium)
- [ ] `url.PathEscape` the CopySource key segment (effort: small)
- [ ] Write machine config with `O_CREATE 0600` instead of write-then-chmod (effort: small)
- [ ] Bounded concurrency in `SyncService.Execute` (effort: medium)
- [ ] Explicit-ownership/opt-in model for apply deletions of unmanaged resources (effort: large)
- [ ] Resolve KV namespaces by ID; refuse ambiguous duplicate titles (effort: medium)

## Roadmap Suggestions

- **Sync engine correctness hardening** — Direction-aware deletes, exclude-protected deletions, working `--include`, atomic download writes; the sync engine is the product's flagship workflow and currently has two data-loss-class bugs (priority: high, effort: medium)
- **Large-transfer timeout architecture** — Separate control-plane (30s) and data-plane (context-driven) HTTP clients across client/restClient; add a slow-body integration test (priority: high, effort: small)
- **Multipart upload lifecycle hygiene** — Cancellation-safe aborts, periodic reaping of orphaned multipart uploads (`cosmoflare doctor` check) (priority: medium, effort: medium)
- **Guardrails v2: mutation-wide enforcement** — Extend project guardrails from upload-only to delete/copy/apply with a single enforcement chokepoint (priority: medium, effort: medium)
- **Apply safety model** — Managed-resource tracking (state file or tags) so `apply` never deletes resources cosmoflare did not create (priority: medium, effort: large)
- **Parallel sync executor** — Bounded worker pool with ordered error reporting (priority: low, effort: medium)

```json:audit-result
{
  "agent": "core-logic",
  "overall_score": 5,
  "sub_scores": {
    "bug_detection": 5,
    "performance": 5,
    "edge_cases": 6,
    "security": 6,
    "data_integrity": 4
  },
  "critical_findings": [{"title": "sync down --delete never deletes local files and can delete wrong remote object", "severity": "high", "file": "pkg/cosmoflare/sync.go:425", "fix": "Branch Execute on plan.Direction: for SyncDown delete ops call os.Remove(op.LocalPath), never DeleteRemoteObject; regression-test that down-sync emits zero remote deletes", "effort": "small"}, {"title": "Shared 30s http.Client.Timeout kills large S3 transfers mid-body", "severity": "high", "file": "pkg/cosmoflare/client.go:113", "fix": "Dedicated timeout-free HTTP client for the S3 transport (ctx + SDK retries govern); keep 30s client for API calls", "effort": "small"}, {"title": "Multipart abort uses canceled request context, orphaning billed uploads", "severity": "high", "file": "pkg/cosmoflare/upload.go:181", "fix": "Abort with context.WithoutCancel(ctx) + timeout; log abort failures (also multipart.go:467)", "effort": "small"}, {"title": "sync up --delete deletes remote objects matching exclude patterns", "severity": "high", "file": "pkg/cosmoflare/sync.go:275", "fix": "Add isExcluded(relKey, input.Exclude) check in the planUp/planDown delete loops to protect excluded remote files", "effort": "small"}, {"title": "--include flag is a silent no-op (SyncPlanInput.Include never read)", "severity": "medium", "file": "pkg/cosmoflare/sync.go:113", "fix": "Implement include filtering in planUp/planDown or remove the flag from cmd/sync.go", "effort": "medium"}, {"title": "apply deletes any account resource not declared in config (delete-by-omission)", "severity": "medium", "file": "pkg/cosmoflare/apply.go:423", "fix": "Only delete resources cosmoflare created (state-tracked/tagged) or require explicit --delete-unmanaged opt-in", "effort": "large"}, {"title": "KV namespaces assumed unique by title; duplicate titles delete wrong namespace", "severity": "medium", "file": "pkg/cosmoflare/diff.go:280", "fix": "Match namespaces by ID where available; refuse ambiguous duplicate-title operations", "effort": "medium"}, {"title": "Machine config tokens written 0644 before chmod 0600 (TOCTOU/crash window)", "severity": "medium", "file": "pkg/cosmoflare/config.go:169", "fix": "Write via os.OpenFile with 0600 perms instead of viper.WriteConfigAs + chmod", "effort": "small"}, {"title": "CopySource not URL-encoded; keys with spaces/+/ %/? break CopyObject", "severity": "medium", "file": "pkg/cosmoflare/storage.go:249", "fix": "copySource := srcBucket + \"/\" + url.PathEscape(srcKey)", "effort": "small"}, {"title": "Interrupted downloads truncate pre-existing local files (no atomic write)", "severity": "medium", "file": "pkg/cosmoflare/download.go:94", "fix": "Stream to temp file in target dir, sync+close with error check, then os.Rename (also cmd/sync.go:534)", "effort": "small"}, {"title": "Guardrails enforce uploads only; deletes and copies bypass them", "severity": "medium", "file": "pkg/cosmoflare/storage.go:227", "fix": "Add enforceDeleteGuardrails/enforceCopyGuardrails mirroring the upload chokepoint", "effort": "medium"}, {"title": "Sync executor is strictly sequential (no worker pool)", "severity": "low", "file": "pkg/cosmoflare/sync.go:408", "fix": "Bounded concurrent workers over plan.Operations with ordered error collection; est. 4-8x faster large syncs", "effort": "medium"}, {"title": "restClient retry sleep ignores ctx; Retry-After unbounded", "severity": "low", "file": "pkg/cosmoflare/rest_client.go:263", "fix": "Select on ctx.Done during delay; cap parseRetryAfter result at maxRetryBackoff", "effort": "small"}, {"title": "Diff skips drift for services absent from config (under-reporting)", "severity": "low", "file": "pkg/cosmoflare/diff.go:414", "fix": "Run comparisons for explicitly-enabled services and report skipped services as unmanaged", "effort": "medium"}, {"title": "Raw cmd.Start() fire-and-forget violates project spawn policy (ROAD-525)", "severity": "low", "file": "cmd/installer_tui/main.go:1280", "fix": "Use daemon.SpawnAsync or Start+Wait in a goroutine", "effort": "small"}],
  "recommendations": [{"action": "Direction-aware delete handling in SyncService.Execute with regression tests", "effort": "small", "priority": "high"}, {"action": "Split control-plane vs data-plane HTTP clients (30s API, timeout-free S3)", "effort": "small", "priority": "high"}, {"action": "Cancellation-safe multipart abort with context.WithoutCancel", "effort": "small", "priority": "high"}, {"action": "Exclude-protected --delete loops plus implemented --include in sync planner", "effort": "medium", "priority": "high"}, {"action": "Atomic temp-file+rename downloads with Close() error checking", "effort": "small", "priority": "medium"}, {"action": "Extend guardrail enforcement to delete/copy paths", "effort": "medium", "priority": "medium"}, {"action": "url.PathEscape CopySource key segment", "effort": "small", "priority": "medium"}, {"action": "0600-first write for machine config credentials", "effort": "small", "priority": "medium"}, {"action": "Bounded concurrency in SyncService.Execute", "effort": "medium", "priority": "low"}, {"action": "Managed-resource ownership model before apply deletions", "effort": "large", "priority": "medium"}],
  "roadmap_suggestions": [{"title": "Sync engine correctness hardening", "description": "Direction-aware deletes, exclude-protected deletions, working --include, atomic download writes \u2014 two data-loss-class bugs live in the flagship workflow", "priority": "high", "effort": "medium"}, {"title": "Large-transfer timeout architecture", "description": "Separate 30s control-plane client from context-driven data-plane client; slow-body integration test", "priority": "high", "effort": "small"}, {"title": "Multipart upload lifecycle hygiene", "description": "Cancellation-safe aborts plus a doctor check that reaps orphaned incomplete multipart uploads", "priority": "medium", "effort": "medium"}, {"title": "Guardrails v2: mutation-wide enforcement", "description": "Single guardrail chokepoint covering upload, delete, copy, and apply mutations", "priority": "medium", "effort": "medium"}, {"title": "Apply safety model", "description": "State/tag-tracked managed resources so apply never deletes resources cosmoflare did not create", "priority": "medium", "effort": "large"}, {"title": "Parallel sync executor", "description": "Bounded worker pool with ordered error reporting; est. 4-8x wall-clock improvement on large syncs", "priority": "low", "effort": "medium"}]
}
```
