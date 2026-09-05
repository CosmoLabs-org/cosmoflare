# Agent 2: Core Logic

> Audit: 2026-08-31, cosmoflare v0.17.0, fresh mode. Report saved verbatim from agent output.

I have completed a deep read of 24 source files across the library, CLI, and server layers. Compiling the final report.

## Core Logic Audit: 5/10

**Sub-scores**:
| Dimension | Score | Notes |
|-----------|-------|-------|
| bug_detection | 4/10 | Silent data corruption in multipart short-read path; pagination missing (data loss); 6 of 14 client options are silent no-ops |
| performance | 5/10 | `sync down` re-downloads everything on every run; checksum mode re-uploads all multipart files; O(n) per-delete API calls |
| edge_cases | 4/10 | `WithPartSize(0)` panics; >1000-object buckets break sync; clock skew mtime heuristic; zero-size multipart upload sends empty parts |
| security | 5/10 | Guardrails parsed but never enforced; `.env`/`.git` uploaded by default; non-constant-time token compare; localhost-only mitigates |
| data_integrity | 4/10 | Truncated uploads stored without error; `sync down --delete` can delete local files that exist remotely; watch `--delete` deletes on scan errors |

The happy paths work and tests exist (193 test files), but the core storage engine has two silent-corruption paths and the config-driven safety features (guardrails, audit, dry-run) are dead code at the library layer.

### Critical Findings

1. **Multipart upload silently stores truncated/corrupt objects** — In `MultipartUpload` and `ResumableMultipartUpload`, a short read from the reader is accepted as success:
   ```go
   n, err := io.ReadFull(reader, buf)
   if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {   // upload.go:209
       abort(); ...
   }
   buf = buf[:n]
   ```
   If the reader delivers fewer bytes than the claimed `size` (file changed after `stat()`, stale size passed by `cmd/sync.go:457` / `cmd/watch.go:218`), the loop keeps iterating: later parts get `io.EOF` with `n=0`, and **empty parts are uploaded and completed with no error returned**. The object is stored short. The correct behavior: any `EOF`/`ErrUnexpectedEOF` before `uploadedBytes == size` must abort with an error.
   - **Severity**: high
   - **File**: `pkg/cosmoflare/upload.go:207-213`, `pkg/cosmoflare/multipart.go:493-498`
   - **Fix**: Track total read; after each `ReadFull`, if `err == io.EOF || err == io.ErrUnexpectedEOF`, return `validationError(... "reader shorter than declared size (%d of %d bytes)")` and abort.

2. **`sync` lists only the first 1000 objects — no pagination** — `r2StorageBackend.ListRemoteObjects` makes a single `ListObjects` call with `maxKeys=1000` and discards `result.NextToken`:
   ```go
   result, err := b.client.ListObjects(ctx, bucket, prefix, "", 1000, "")  // cmd/sync.go:430
   ```
   For prefixes with more than 1000 objects (typical static site), `sync down --delete` treats objects 1001+ as absent from remote and **deletes the matching local files**; `sync up` re-uploads everything beyond 1000 on every run.
   - **Severity**: high
   - **File**: `cmd/sync.go:428-443`
   - **Fix**: Loop on `result.NextToken != ""`, passing it as `continuationToken`, accumulating items.

3. **Guardrails are parsed, tested, and never enforced** — `GuardrailChecker.CheckUpload` (bucket allowlist, `max_file_size`, `blocked_keys`) has **zero production callers** (verified by grep across `pkg/`, `cmd/`, `internal/`). Neither `client.Upload`, `apply`, `sync`, nor `watch` invoke it. A user with `guardrails: {blocked_keys: [".env"], allowed_buckets: ["prod"]}` in `.cosmoflare.yaml` uploads `.env` to any bucket unimpeded — a false sense of protection.
   - **Severity**: high
   - **File**: `pkg/cosmoflare/guardrails.go:29-65` (no call sites)
   - **Fix**: Call `CheckUpload` inside `client.Upload`/`MultipartUpload` (loaded from project config), or at minimum in `cmd/sync.go` and `cmd/watch.go` before each upload; fail the op when `!result.Allowed`.

4. **`sync down` is never idempotent — re-downloads all files every run** — The comparison heuristic is `local.Size == remote.Size && !local.ModTime.After(remote.LastModified)` (`pkg/cosmoflare/sync.go:368`). A downloaded file gets `mtime = now` (`os.Create` + `io.Copy`, `cmd/sync.go:473-479`), which is always *after* the remote `LastModified`. On the next `sync down`, every file compares as "modified" and is re-downloaded. Fix: set the local file's mtime to `remote.LastModified` after download (`os.Chtimes`).
   - **Severity**: high (functional correctness + bandwidth cost)
   - **File**: `pkg/cosmoflare/sync.go:361-369`, `cmd/sync.go:461-481`
   - **Fix**: After `io.Copy`, call `os.Chtimes(localPath, remote.LastModified, remote.LastModified)`.

5. **Checksum mode can never match multipart ETags** — In checksum mode, `isUnchanged` compares the local MD5 to the raw ETag (`sync.go:364-365`). R2/S3 ETags for multipart uploads are `"<md5>-<N>"`, never a bare MD5 — so every file above the 100MB multipart threshold re-uploads on every `--checksum` sync, the exact files where re-upload hurts most. Strip the `-N` suffix before comparing, or store and compare part-count-aware digests.
   - **Severity**: medium
   - **File**: `pkg/cosmoflare/sync.go:361-366`
   - **Fix**: `remoteHash := strings.TrimSuffix(strings.Trim(remote.ETag, "\""), regexpSuffix)` — split on `-` and use the left side when the right side is numeric.

6. **`WithPartSize(0)` crashes the process** — `MultipartUpload` computes `numParts := (size + partSize - 1) / partSize` (`upload.go:184`) with **no part-size validation**; a caller passing `WithPartSize(0)` gets an integer-divide-by-zero panic. `ResumableMultipartUpload` validates (`multipart.go:407`) — the base path does not. Also, parts below the 5MB S3 minimum are only rejected mid-transfer by the API.
   - **Severity**: high
   - **File**: `pkg/cosmoflare/upload.go:183-187`
   - **Fix**: Call `ValidatePartSize(cfg.partSize)` in `MultipartUpload` exactly as `multipart.go:407` does.

7. **Watcher swallows scan errors, then deletes remote objects** — `FileWatcher.Snapshot`'s walk callback returns `nil` on every error (`watcher.go:124-127`), so an unreadable subtree silently vanishes from the snapshot. `Diff` then reports those files as `ChangeDeleted`, and `cmd/watch.go:178-183` with `--delete` removes them from R2. A permissions change on one directory turns into remote data deletion with only a nil-cost error swallow.
   - **Severity**: high
   - **File**: `pkg/cosmoflare/watcher.go:124-127`, `cmd/watch.go:178-185`
   - **Fix**: Distinguish root failure (return error) from per-entry failure; at minimum track unreadable paths and never emit `ChangeDeleted` for files under a path that failed to read.

8. **6 of 14 exported client options are silent no-ops** — Verified against all production code:
   - `WithTimeout`/`WithHTTPClient` (`client.go:91-94`): the client with timeout is constructed and stored, but **never passed** to `cloudflare.NewWithAPIToken` (line 97) or the AWS config — no API call has a timeout; a hung connection blocks forever.
   - `WithProfile` (`options.go:28`): documented in `NewClient` ("options > env > profile", `client.go:76`) but `cfg.profile` is never read.
   - `WithAuditLog`, `WithDryRun`, `WithBucket`: set fields that nothing reads.
   - **Severity**: high (API contract violation; the timeout one is a hang risk)
   - **File**: `pkg/cosmoflare/client.go:66-116`, `pkg/cosmoflare/options.go`
   - **Fix**: Pass `cloudflare.HTTPClient(httpClient)` to `NewWithAPIToken` and `awsconfig.WithHTTPClient` to the S3 config; resolve the profile via `LoadMachineConfig` when `cfg.profile != ""`; delete or implement the other three.

9. **Aborted multipart uploads leak on R2 (billed storage)** — The `abort()` helper reuses the request `ctx` (`upload.go:175-179`, `multipart.go:465-471`). When part failures are caused by context cancellation (Ctrl+C — note `cmd/sync.go:225` uses `context.Background()` so SIGINT hard-kills instead), `AbortMultipartUpload` fails immediately, leaving orphaned parts on R2. Also `/tmp` state files from killed processes are never swept.
   - **Severity**: medium
   - **File**: `pkg/cosmoflare/upload.go:174-180`, `pkg/cosmoflare/multipart.go:464-472`
   - **Fix**: Abort with `context.WithTimeout(context.Background(), 10*time.Second)`; add a state-file GC keyed on `StartedAt` age.

10. **Content-Type is never auto-detected on upload** — `detectContentType` (`upload.go:315-319`) is dead code (no callers). Uploads without explicit `WithContentType` store no Content-Type, so R2 serves `application/octet-stream` — HTML/CSS/JS pushed by `sync up`/`watch` download instead of rendering. Meanwhile `defaultCachePolicy` *is* auto-applied, making the omission inconsistent.
    - **Severity**: medium
    - **File**: `pkg/cosmoflare/upload.go:315-319`, `upload.go:100-102`
    - **Fix**: In `Upload`/`MultipartUpload`, `if cfg.contentType == "" { cfg.contentType = detectContentType(key) }`.

11. **`--include` flag accepted, then ignored** — `cmd/sync.go:104` defines `--include`, passed into `SyncPlanInput.Include`, but `planUp`/`planDown` (`pkg/cosmoflare/sync.go:224-356`) never read `input.Include`. Users scoping an upload with `--include` upload everything. Related: `scanLocalDir` has no default excludes, so `cosmoflare sync up . my-bucket` uploads `.git/`, `.env`, and credentials files.
    - **Severity**: medium
    - **File**: `cmd/sync.go:103-104`, `pkg/cosmoflare/sync.go:103-113`
    - **Fix**: Implement include filtering in `planUp`/`planDown` symmetric to `isExcluded`; default-exclude `.git/`, `.env*`, `.DS_Store` unless overridden.

12. **DNS diff collapses duplicate records and is non-deterministic** — The diff keys live records by `{Type, Name, Content}` in a map (`diff.go:345-355`); legitimate round-robin duplicates (two identical A records) overwrite each other, so only one deletion is planned and reported. Deletions also iterate the map directly (`diff.go:371`) while every other service sorts — `--json` diff output ordering varies between runs, which breaks agent-consumable determinism (a stated project goal).
    - **Severity**: medium
    - **File**: `pkg/cosmoflare/diff.go:345-380`
    - **Fix**: Key by record ID for deletions or count occurrences; collect and `sort.Strings` the deletion entries as `CompareWorkers` does.

13. **One failing Cloudflare source silences all metrics** — `MetricsProducer.poll` returns on any single source error (`metrics.go:78-80`): a token without Workers permission permanently suppresses zones/R2/KV metrics too. Subscribers get nothing and no error frame — only stderr logs. This matches the open issue "desktop dashboard is not live." Also `snap.Profile` is hardcoded `""` (`metrics.go:50`) — the field is emitted but never populated, and `reflect.DeepEqual` on `any` payloads is walked by reflection every tick.
    - **Severity**: medium
    - **File**: `internal/server/metrics.go:49-86`
    - **Fix**: Publish partial snapshots with a per-source error map; populate `Profile` from the active account; compare marshaled JSON bytes for cheap, stable delta detection.

14. **SSE stream has no heartbeat** — With delta suppression active, an unchanged dashboard sends zero frames; `handleEvents` (`sse.go:109-123`) never sends keepalives. Proxies (nginx 60s read timeout, ALB) drop the connection and the desktop app reconnect-loops. Add a `: ping` comment frame every ~20s.
    - **Severity**: medium
    - **File**: `internal/server/sse.go:109-123`
    - **Fix**: `case <-time.After(20 * time.Second): fmt.Fprintf(w, ": ping\n\n"); flusher.Flush()`.

15. **Token comparison is not constant-time; empty-token config bypasses auth** — `authorized` uses `==` on the bearer token (`server.go:147-151`). If `Config.Token` were ever empty, `Authorization: Bearer ` matches `"Bearer " + ""` — full bypass. `cmd/serve.go:76-83` currently always generates a 128-bit token, so this is defense-in-depth, but `server.New` accepts an empty token without error. The `?token=` query fallback also leaks the token into shell history/logs (documented tradeoff).
    - **Severity**: low
    - **File**: `internal/server/server.go:147-152`
    - **Fix**: `crypto/subtle.ConstantTimeCompare`; reject empty `cfg.Token` in `New`.

16. **KV apply does N+1 API calls for deletions** — Each `ApplyDelete` op re-lists every namespace to find the ID (`apply.go:474-487`). Deleting 10 namespaces = 10 full list calls. List once before the loop and reuse (the DNS applier already does this correctly at `apply.go:532`).
    - **Severity**: low
    - **File**: `pkg/cosmoflare/apply.go:472-493`
    - **Fix**: Hoist `ListNamespaces` above the `for _, op := range ops` loop, build a title→ID map.

### Recommendations

- [ ] Fix the short-read acceptance in both multipart loops; add a regression test with a reader shorter than declared size (effort: small)
- [ ] Paginate `ListRemoteObjects` on `NextToken`; test with a fake backend returning 2 pages (effort: small)
- [ ] Wire `GuardrailChecker.CheckUpload` into `client.Upload` + sync/watch command paths (effort: medium)
- [ ] Set downloaded-file mtime from `remote.LastModified` and strip multipart `-N` ETag suffix in checksum compare (effort: small)
- [ ] Pass the configured `httpClient` into `cloudflare.NewWithAPIToken` and the AWS SDK config; add `ValidatePartSize` to `MultipartUpload` (effort: small)
- [ ] Remove or implement `WithProfile`, `WithAuditLog`, `WithDryRun`, `WithBucket`; auto-set `Content-Type` via the existing `detectContentType` (effort: medium)
- [ ] Never emit `ChangeDeleted` for paths under a failed walk entry; retry failed watch uploads instead of advancing `lastSnap` (effort: medium)
- [ ] Add SSE heartbeat, publish partial metrics with per-source errors, populate `Profile` (effort: small)
- [ ] Sort DNS deletion entries and handle duplicate records; implement `--include` and default excludes in sync (effort: medium)

### Roadmap Suggestions

- **Storage engine integrity hardening** — Short-read aborts, part-size validation, orphaned-multipart cleanup (abort with background context + `/tmp` state GC) (priority: high, effort: medium)
- **Make sync correct at scale** — Pagination, download mtime preservation, ETag normalization, deterministic operation ordering (priority: high, effort: medium)
- **Enforce declared guardrails** — Wire `.cosmoflare.yaml` guardrails/audit/dry-run into the actual mutation paths so config stops being advisory (priority: high, effort: medium)
- **Live-metrics reliability** — SSE keepalive, partial-snapshot publishing, delta detection on marshaled bytes (priority: medium, effort: small)
