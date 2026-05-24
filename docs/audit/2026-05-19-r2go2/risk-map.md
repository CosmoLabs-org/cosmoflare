# Risk Map

**Source**: Synthesized from all 5 agent reports

## Risk Severity Definitions

| Severity | Meaning |
|----------|---------|
| CRITICAL | Data corruption, security vulnerability, or total feature failure |
| HIGH | Incorrect behavior that affects users, broken delivery pipeline |
| MEDIUM | Quality degradation, maintenance burden, user confusion |
| LOW | Cosmetic issues, minor inefficiencies, nice-to-haves |

---

## CRITICAL Risks

### CRIT-1: Multipart Upload Sort Bug (upload.go:253)

**Source**: agent-2-core-logic.md

Multipart upload reassembles parts in incorrect order. For files exceeding the single-upload threshold, the completed object is silently corrupted. The file uploads "successfully" but the content is wrong.

**Impact**: Data corruption for large file uploads. No error is raised. Users discover corruption only when downloading and using the file.

**Blast radius**: Any user uploading files larger than the multipart threshold (~5MB). This affects `r2go2 object put` for large files and any library consumer calling the upload API.

**Mitigation**: Fix the sort comparator in `upload.go:253` to sort by part number numerically. Add a test that uploads a multi-part file and verifies byte-for-byte integrity after download.

### CRIT-2: All S3 Errors Misclassified as NotFound (storage.go:158, download.go:59)

**Source**: agent-2-core-logic.md

In `GetObject`, `HeadObject`, and `Download`, any error returned by the S3 client is wrapped as `R2NotFoundError`. This includes authentication failures (403), network errors, server errors (500), and rate limiting (429).

**Impact**: Callers who check `errors.As(&r2NotFoundErr)` get false positives for every non-404 failure. A caller that deletes-then-recreates on "not found" would do so on auth errors. Retry logic that skips "not found" silently skips transient failures.

**Blast radius**: Every S3 operation that fails for any reason other than 404.

**Mitigation**: Inspect the underlying S3 error code. Only wrap as `R2NotFoundError` for `NoSuchKey`/HTTP 404. Map other codes to appropriate error types.

### CRIT-3: Config File Permissions Race (internal/config/config.go:114-120)

**Source**: agent-2-core-logic.md

Config file is written via `WriteConfigAs` (default permissions 0644, world-readable) and then `os.Chmod` to 0600. Between these two calls, the file containing API tokens is readable by any user on the system (TOCTOU vulnerability).

**Impact**: Credential exposure on multi-user systems. The race window is small but exploitable.

**Blast radius**: Any user running `r2go2 config init` or first-time setup on a shared system.

**Mitigation**: Use `os.OpenFile` with explicit mode `0600` at creation time, or write to a temp file and `os.Rename` into place.

### CRIT-4: Bucket Update No-Op Claiming Success (cmd/bucket.go:353-380)

**Source**: agent-3-api-design.md

The `bucket update` command accepts flags, validates input, prints "Bucket updated successfully", but makes no API call. Users and agents believe the update was applied when nothing changed.

**Impact**: Users have false confidence in bucket configuration. Agents proceed to next step assuming configuration is applied, causing downstream failures.

**Blast radius**: Any user or agent invoking `r2go2 bucket update`.

**Mitigation**: Implement the actual update logic or remove the command with a clear "not implemented" error.

### CRIT-5: printError Swallows Errors in JSON Mode (root.go:196-200)

**Source**: agent-3-api-design.md

When `--json` is active, `printError()` suppresses error output entirely. No JSON error object, no indication of failure in some paths.

**Impact**: Agents and scripts using `--json` receive no output and cannot detect failures. Directly undermines agent-first design.

**Blast radius**: Every error path when `--json` is active.

**Mitigation**: Emit `{"error": {"code": "...", "message": "..."}}` in JSON mode with non-zero exit code.

### CRIT-6: Double defer Close (object.go:389,391)

**Source**: agent-3-api-design.md

`obj.Content.Close()` is deferred twice in succession. While most `io.ReadCloser` implementations handle double-close, the interface contract does not guarantee it.

**Impact**: Potential panic at runtime if the underlying reader does not handle double-close.

**Blast radius**: The `object get` command's download path.

**Mitigation**: Remove the duplicate defer at line 391.

### CRIT-7: CI Pipeline Never Exercised

**Source**: agent-4-infrastructure.md

The GitHub Actions CI workflow exists but has never been triggered with the current codebase state. Known `go vet` errors would cause the pipeline to fail immediately. This means every release has been shipped without CI validation.

**Impact**: No automated quality gate. Bugs that `go vet`, `go test`, or linting would catch are shipped to users.

**Blast radius**: Every release.

**Mitigation**: Fix `go vet` errors, trigger the CI pipeline, make it required for merges.

---

## HIGH Risks

### HIGH-1: Download Range Bug (download.go:53)

**Source**: agent-1-code-quality.md

`rangeStart >= 0` is always true because the default value is `0`. Every download request includes a `Range` header, even when the caller did not request a range download. Server behavior with `Range: bytes=0-` is implementation-dependent.

**Impact**: Potential partial downloads or unexpected behavior depending on server-side interpretation. Most S3-compatible servers treat `Range: bytes=0-` as equivalent to no range, but this is not guaranteed.

**Mitigation**: Introduce a `rangeSet bool` field or use `-1` as the sentinel default.

### HIGH-2: install.sh Broken Binary Name

**Source**: agent-4-infrastructure.md

The install script references an uppercase binary name that does not match the actual build output. Users who follow the installation instructions get a "file not found" error.

**Impact**: First-time installation fails. Users cannot install the tool via the documented method.

**Mitigation**: Fix the binary name reference in `install.sh` to match the actual output name (`r2go2`).

### HIGH-3: ExportProfile Shell Injection Risk

**Source**: agent-2-core-logic.md

`ExportProfile` generates shell `export` statements by interpolating profile values directly into the output string. If a profile value contains shell metacharacters (backticks, `$(...)`, semicolons), the exported string could execute arbitrary commands when sourced.

**Impact**: Arbitrary command execution when a user sources the exported profile.

**Mitigation**: Sanitize or single-quote all values in the generated export statements.

### HIGH-4: DNSService.Update No Fetch-Then-Patch

**Source**: agent-2-core-logic.md

The `Update` method sends caller-provided fields without fetching the existing record. Omitted fields may be reset to defaults rather than preserved.

**Impact**: Users updating one field (e.g., TTL) may inadvertently reset other fields (e.g., content, proxied status).

**Mitigation**: Implement fetch-then-patch: retrieve current record, merge caller changes, send merged result.

---

## MEDIUM Risks

### MED-1: Release Asset Path Mismatch (.goreleaser.yml)

**Source**: agent-4-infrastructure.md

The goreleaser configuration references an asset path that does not match the actual build output directory. Release artifacts may be missing or incorrectly packaged.

**Impact**: Broken release artifacts. Users downloading from GitHub releases may get incomplete packages.

**Mitigation**: Align goreleaser asset paths with actual build output.

### MED-2: 3,929 Lines Dead Code

**Source**: agent-1-code-quality.md

`internal/migration/` and `cmd_disabled/` contain ~3,929 lines of dead code. Not compiled, not tested, not referenced.

**Impact**: Repository noise, search pollution, risk of accidental re-enablement in broken state.

**Mitigation**: Delete both directories. Code is recoverable from git history.

### MED-3: Missing Pagination on List Commands

**Source**: agent-3-api-design.md

Some list commands (e.g., `kv list`, `dns list`) do not implement pagination. For large result sets, only the first page is returned.

**Impact**: Users with many resources see incomplete results without warning.

**Mitigation**: Implement cursor-based pagination following the Cloudflare API pagination model.

### MED-4: TUI Simulated Data

**Source**: agent-1-code-quality.md

The dashboard shows hardcoded/simulated data rather than live API data.

**Impact**: Users see plausible but fake numbers. Dashboard is a visual prototype, not a functional tool.

**Mitigation**: Wire to live API with caching. Show "no data" state when credentials are not configured.

### MED-5: DNS Update Missing Flags

**Source**: agent-3-api-design.md

The `dns update` command does not expose all updateable fields as CLI flags. Some DNS record properties can only be modified through the library API, not through the CLI.

**Impact**: CLI users cannot fully manage DNS records without falling back to the library or direct API calls.

**Mitigation**: Add missing flags for all updateable DNS record fields.

### MED-6: README Documentation Drift

**Source**: agent-5-docs-roadmap.md

README lists 6 implemented services (DNS, Zones, SSL, Cache, Healthchecks, Doctor) as "Planned". ROAD-035 title does not match its content.

**Impact**: Users and contributors see inaccurate project state. May avoid the project thinking features they need are not yet available.

**Mitigation**: Update README service table to reflect actual implementation status.

### MED-7: 33.6MB Unstripped Binary

**Source**: agent-4-infrastructure.md

The release binary is ~33.6MB because it includes debug symbols. Comparable Go CLI tools strip to ~15-20MB.

**Impact**: Larger downloads, slower CI, more disk usage.

**Mitigation**: Add `-ldflags="-s -w"` to build flags or configure goreleaser to strip.

---

## LOW Risks

### LOW-1: No CONTRIBUTING.md

**Source**: agent-5-docs-roadmap.md

No contributor guidelines despite MIT license and open-source positioning.

### LOW-2: Inconsistent Pointer Helpers

**Source**: agent-1-code-quality.md

Mix of `StringPtr()` helpers and inline `&value` expressions.

### LOW-3: os.Exit in Library Code

**Source**: agent-1-code-quality.md

Some library code calls `os.Exit()` directly, preventing graceful error handling by callers.

### LOW-4: ROAD-035 Title Mismatch

**Source**: agent-5-docs-roadmap.md

Roadmap item ROAD-035 has a title that does not match its content description.

---

## Risk Heat Map

```
              Low Impact    Medium Impact    High Impact
            +-------------+----------------+---------------+
 Likely     | LOW-1,2,3   | MED-2,4,7      | CRIT-4,5,7    |
            +-------------+----------------+---------------+
 Possible   | LOW-4       | MED-1,3,5,6    | CRIT-1,2,6    |
            +-------------+----------------+---------------+
 Unlikely   |             | HIGH-1         | CRIT-3,HIGH-3 |
            +-------------+----------------+---------------+
```
