# Agent 1: Code Quality

> Audit: 2026-08-31, cosmoflare v0.17.0, fresh mode. Report saved verbatim from agent output.

All investigation complete. Here is my full audit report.

## Code Quality Audit: cosmoflare

Verified live: `go build ./...` OK, `go vet` clean, full Go test suite passes (18 packages, exit 0), race detector clean on `internal/webhook`, desktop vitest 15/15 passes.

**Sub-scores**:
| Dimension | Score | Notes |
|-----------|-------|-------|
| architecture | 7/10 | Clean 3-layer split (cmd → pkg/cosmoflare → internal), library-first design holds; dragged down by 4 god files and dead package litter |
| code_patterns | 7/10 | Typed error hierarchy, functional options, generics used well; but 489 duplicated `if JSONOutput` branches and pervasive boilerplate handlers |
| tech_debt | 6/10 | Only 2 TODOs (both intentional template strings), deps current — but 234MB tracked session artifacts, broken golangci-lint, 4 disabled packages |
| test_coverage | 7/10 | 172 test files vs 175 source files, httptest-based behavioral tests, high internal coverage; library 65.3%, cmd 43.2%, two packages near zero |
| maintainability | 6/10 | Naming drift (r2go2 vs cosmoflare), two different project config filenames, config split confuses the "single config" contract |

### Architecture Evidence

The layering is real and disciplined. `pkg/cosmoflare/client.go:19` defines an `R2Client` interface with 18 methods; `internal/server/server.go:1-7` documents itself as "a thin transport over the existing per-service constructors... owns no Cloudflare logic of its own" — and the code matches the claim. Each Cloudflare service follows an identical template (`kv.go:66-95`: struct + `NewXService` with nil/empty guards + typed errors), so adding a service is a two-file operation. Good extensibility.

Weaknesses are concentrated in `cmd/`:
- `cmd/installer_tui/main.go` — 1299 lines, single `main` package, one model doing onboarding, quick-start guide rendering, and browser launching.
- `cmd/object.go` — 1183 lines; `runObjectPut` (line 483) runs ~230 lines through three near-identical upload paths (stdin at 496, resume at 555, normal) each repeating option-building, client creation, JSON-branch error handling, and success output.
- `cmd/email.go` — 1063 lines, 17 handlers of the same list/get/create/update/delete shape.
- Dead structure: 6 empty directories (`internal/types`, `internal/auth`, `internal/storage`, `internal/client`, `internal/cli/live`, `internal/api_disabled/tests`), 4 packages behind `//go:build disabled` tags (~57KB: `analytics_disabled`, `api_disabled`, `domain_disabled`, `migration_disabled`), and 4 `.go.disabled` files in `cmd/`.

### Critical Findings

1. **Dual project-config filenames split the "single config" contract** — `pkg/cosmoflare/config.go:67` loads `.r2go2.yaml` (profiles) plus `~/.r2go2/config.yaml` (machine config), while `pkg/cosmoflare/diff.go:131` loads `.cosmoflare.yaml` (declarative state). CLAUDE.md claims "All services configured via single .cosmoflare.yaml". Error prefixes still say `r2go2:` (`errors.go:16-21`, `client.go:112`). Users get two config files with two names from one tool.
   - **Severity**: medium
   - **File**: `pkg/cosmoflare/config.go:67`, `pkg/cosmoflare/diff.go:131`
   - **Fix**: Make `LoadProjectConfig` search `.cosmoflare.yaml` first with `.r2go2.yaml` as legacy fallback; rename error prefix to `cosmoflare:` behind one constant.

2. **`runObjectPut` god function with triple-duplicated upload paths** — 230 lines, three copies of option assembly + error/JSON handling + result printing. Any change to upload output must be made three times.
   - **Severity**: medium
   - **File**: `cmd/object.go:483-713`
   - **Fix**: Extract `buildUploadOpts()` and `reportUploadResult(*UploadResult)` helpers; collapse the three paths into one parameterized flow.

3. **489 `if JSONOutput {` branches — output-mode boilerplate at every site** — Every handler duplicates the dual print/error pattern, e.g. `cmd/email.go:356-359`: `if JSONOutput { return printErrorJSON(...) } return fmt.Errorf(...)`. This is the single largest duplication mass in the codebase.
   - **Severity**: medium
   - **File**: `cmd/` (489 occurrences; sample `cmd/object.go:525-528`)
   - **Fix**: Add `reportError(err, format, args...)` and `reportResult(msg, data)` helpers that branch once, mirroring the existing `printInfo`/`printSuccess` layer.

4. **Untested retry/batch logic package** — `internal/cli/operations/copy.go` (442 lines: `copyWithRetry`, `isRetryableError`, `verifyCopy`, `NewBatchCopy`) has zero test files. `internal/migration` sits at 10.4% coverage. These contain the failure-recovery paths users depend on.
   - **Severity**: medium
   - **File**: `internal/cli/operations/copy.go`, `internal/migration/`
   - **Fix**: Add table-driven tests for `isRetryableError` and the retry loop using a fault-injecting fake source; cover migration happy path + one failure.

5. **234MB of GOrchestra session artifacts tracked in git, 81 recovery.patch files** — Several patches exceed 665,000 lines. They bloat clones and the security scanner flags AWS example keys (`AKIAIOSFODNN7EXAMPLE`) embedded in transcript content inside `GOrchestra/sessions/agent-a1cdb2ac246339700/recovery.patch:104`, producing recurring critical-severity noise that masks real findings. (Source code itself uses that key only in test fixtures — `internal/interactive/backup_restore_test.go:47` — which is acceptable.)
   - **Severity**: medium
   - **File**: `GOrchestra/sessions/*/recovery.patch`
   - **Fix**: Add `GOrchestra/sessions/` to `.gitignore`, `git rm --cached` existing patches, keep only manifests; archive old sessions outside the repo.

6. **golangci-lint panics — lint gate absent** — `golangci-lint run` dies with `panic: file requires newer Go version go1.27 (application built with go1.26)`. Only `go vet` (which passes) and the spawn-check currently gate quality.
   - **Severity**: medium
   - **File**: toolchain (go.mod `go 1.26` vs installed golangci-lint binary)
   - **Fix**: Rebuild golangci-lint with the repo's Go toolchain or pin a compatible version; add it to CI so the skew is caught.

7. **Raw `cmd.Start()` fire-and-forget violates project spawn policy (ROAD-525)** — `_ = cmd.Start()` with no Wait leaks a zombie process per documentation open on some platforms.
   - **Severity**: low
   - **File**: `cmd/installer_tui/main.go:1280`
   - **Fix**: Use `exec.Command(...).Start()` followed by a goroutine `cmd.Wait()`, or the project's `daemon.SpawnAsync()`.

8. **Auth token compared non-constant-time; empty-token edge in library API** — `server.authorized` uses `==` for the bearer token; if `Config.Token` were empty, a bare `Authorization: Bearer ` header would match. The CLI mitigates (`cmd/serve.go:76-83` generates a random token when empty) but `server.New` does not validate.
   - **Severity**: low
   - **File**: `internal/server/server.go:147-152`
   - **Fix**: Reject empty `Config.Token` in `New()`; use `crypto/subtle.ConstantTimeCompare`.

9. **`MetricsSnapshot.Profile` never populated; delta detection via `reflect.DeepEqual` on `any`** — `internal/server/metrics.go:50` hardcodes `Profile: ""` forever, and the four data fields are untyped `any`, making DeepEqual both expensive and fragile across pointer-bearing payloads.
   - **Severity**: low
   - **File**: `internal/server/metrics.go:10-16,50,81`
   - **Fix**: Thread the profile name from the serve layer; type the snapshot fields or marshal-to-JSON compare if payload types vary.

10. **CLI wiring coverage is 43.2% and library 65.3%** — Measured: `cmd` 43.2%, `pkg/cosmoflare` 65.3%, `internal/config` 50.8%. Internals are strong (webhook 99.4%, interactive 93.1%, server 88.9%, utils 98.2%, tui components 95-100%), which shows the team can hit high coverage — the CLI layer just hasn't received it.
    - **Severity**: medium
    - **File**: `cmd/` overall
    - **Fix**: Extend the existing httptest-mock pattern (already proven in `cmd/kv_test.go`) to the top-10 most-used commands; target 60% for `cmd`.

### Positive Observations

- Error handling is genuinely idiomatic: `R2Error` hierarchy with `Unwrap()` (`pkg/cosmoflare/errors.go:14-24`), typed notFound/auth/quota/accessDenied/validation constructors used uniformly across all services.
- `ListResult[T any]` generics (`types.go:61`) avoid the usual `[]interface{}` pagination mess; constructors validate inputs consistently (`kv.go:72-79`).
- Tests are behavioral, not mock-theater: `kv_test.go:17-22` spins real `httptest` servers fronting `cloudflare-go` — they exercise actual HTTP handling.
- Security hygiene in the config layer: machine config written with `os.Chmod(configPath, 0600)` (`config.go:159`); serve generates a 128-bit random token (`cmd/serve.go:152`).
- Project management is exceptionally healthy: 99 roadmap items in `docs/roadmap/items/`, 36 tracked issues with 35 closed and 1 open (FEAT-008), 19 planning-mode docs, 17 brainstorms, active three-tier doc chain, USAGE.md as agent reference. The project knows what it owes.

### Recommendations

- [ ] Unify project config to `.cosmoflare.yaml` with `.r2go2.yaml` legacy fallback; single `cosmoflare:` error prefix constant (effort: medium)
- [ ] Extract `reportError`/`reportResult` output helpers and collapse the 489 JSON branches (effort: medium)
- [ ] Split `runObjectPut` into three helpers; break `installer_tui/main.go` and `email.go` into subpackages (effort: medium)
- [ ] Add tests for `internal/cli/operations` and raise `internal/migration` from 10.4% (effort: small)
- [ ] Remove `GOrchestra/sessions/` from git tracking; add to `.gitignore` (effort: small)
- [ ] Fix golangci-lint toolchain skew and wire it into CI (effort: small)
- [ ] Delete 6 empty dirs, archive 4 `_disabled` packages and 4 `.go.disabled` files to a branch (effort: small)
- [ ] Harden `server.New` token validation + constant-time compare (effort: small)

### Roadmap Suggestions

- **Config unification (r2go2 → cosmoflare)** — One filename, one error prefix, migration path for existing `.r2go2.yaml` users (priority: high, effort: medium)
- **CLI output helper refactor** — Kill the 489-site JSON/human dual-branch pattern with a shared reporting layer (priority: medium, effort: medium)
- **Coverage floor for cmd/ and internal/cli** — Bring cmd wiring to 60% using the existing httptest pattern; CI coverage gate (priority: medium, effort: medium)
- **Repo hygiene: purge GOrchestra session artifacts** — 234MB tracked patches hurt every clone and pollute security scans (priority: medium, effort: small)
- **God-file decomposition (object.go, email.go, installer_tui)** — Subpackage per command group under 500-line budget (priority: low, effort: large)
