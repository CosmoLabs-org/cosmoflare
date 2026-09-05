# Risk Assessment — cosmoflare v0.17.0

> Synthesized from 13 agent reports (2026-08-31 audit). Overall project score 46.1/100 (Developing, High priority).

## Tech Debt Hotspots

| Hotspot | Why it hurts | Source |
|---------|--------------|--------|
| `cmd/object.go:483-713` (`runObjectPut`) | 230 lines × 3 duplicated upload paths — every output change must be made 3× | agent-1 |
| 489 `if JSONOutput` branches | Largest duplication mass; blocks any output-contract change | agent-1 |
| r2go2→cosmoflare rename debt | Dead module path in ldflags/installers breaks releases silently; public library errors carry wrong product name | agents 1,5,7,9 |
| `internal/cli/operations/copy.go` (0% tests, 442 lines) | Untested retry/recovery paths users depend on | agent-1 |
| `internal/migration` (10.4%) | Near-untested migration engine | agent-1 |
| 234MB GOrchestra patches + 419MB pack | Clone cost, CI checkout time, scanner noise (AWS example keys flagged as critical) | agents 1,5,9 |

## Fragile Areas

1. **The storage engine's error discipline** (agent-2): short-read acceptance (`upload.go:207`), `WithPartSize(0)` panic (`upload.go:184`), abort-with-canceled-context (`upload.go:175`). The R2 core — the product's origin — is where correctness is weakest.
2. **Sync at any real scale** (agent-2): >1000 objects breaks listing entirely; mtime heuristic guarantees perpetual re-download; ETag compare can never match multipart uploads.
3. **CI/Release pipeline** (agent-5, agent-9): `internal/interactive` races make `-race` gates fail on every run; 18/18 release runs failed; Docker unbuildable. Master CI has been red for months.
4. **The daemon's error contract** (agent-7): everything is 502 — the desktop app cannot distinguish user-fixable from transient.
5. **Zero-coverage CLI wiring** (agent-1): cmd/ at 43.2% while internals hit 93-100%.

## Security Surface

- **Credentials**: plaintext tokens/keys in `~/.r2go2/config.yaml` (0600 correct, plaintext on disk); `internal/keychain` exists, tested, unwired to profile storage (agent-9). Project memory forbids auto-triggering keychain probing — fix must be opt-in.
- **Daemon auth**: localhost + random token is a sound default (agent-7), but `==` compare, empty-token edge, `?token=` in query strings (shell-history leak), no `ReadHeaderTimeout` (agent-7, agent-2).
- **Guardrails theater** (agent-2): `.cosmoflare.yaml` guardrails (blocked_keys, allowed_buckets) parse and test but never run — `sync up . bucket` uploads `.env` and `.git/` by default with no default excludes. A false sense of protection is worse than none; this is advertised as an agent-safety feature (agent-4).
- **Installer supply chain** (agent-9): `curl | bash` downloads with no checksum verification (the release publishes checksums-sha256.txt — unused). Currently moot (dead repo URL = 404) but must be fixed before any launch.
- **No security scanning in CI**: no govulncheck, CodeQL, trivy, dependabot (agent-9).
- **Committed scan noise**: AWS example key (`AKIAIOSFODNN7EXAMPLE`) inside tracked recovery.patch files produces recurring critical-severity findings that mask real ones (agent-1). Legit test-fixture use is acceptable (`internal/interactive/backup_restore_test.go:47`).

## Chained Risks (compounding)

1. **The release death-spiral** (agents 5, 9, 6): races fail tests → CI red → release blocked → no GitHub Releases → installers 404 → zero distribution → paid tiers have no funnel. Each link is individually small; the chain means the product *cannot reach a single external user today*.
2. **Silent corruption + sync deletion** (agent-2): a file that changes during upload can store truncated (bug 1); a bucket >1000 objects combined with `sync down --delete` deletes local files that exist remotely (bug 2). A user hitting both destroys local and remote copies of the same data.
3. **Agent-safety claim vs. reality** (agents 2, 4): the competitive analysis identifies "agent trust tier" as the *only defensible niche* post-`cf` — while the audit found guardrails unenforced, `--json` errors silent, and MCP at 2% surface. The strategy and the implementation point in opposite directions.
4. **Rename debt × release tooling** (agents 5, 9): the dead `CosmoDev-R2Go2` path appears in ldflags (silent no-op), installers (404), and release notes (broken install command). Fixing only one site yields a green pipeline shipping binaries that lie about their version.

## Single Points of Failure

- **`ccs`-side pipeline discipline lives in docs no one re-reads**: README truth (4 agents found stale service table), roadmap (frozen 70 days), prompts (71-day-old "latest"). The tracking system is itself the drift source (agents 6, 10, 12, 14).
- **`MetricsProducer.poll`** fails closed — one permission-less source kills the entire desktop dashboard's live data (agent-2).
- **One open issue carries the desktop tier** (FEAT-008, 71 days at 0% plan execution) while its plan predates the code that shipped around it (agent-15).

## Agent Cross-References

| Risk | Severity | Source Agents | Evidence |
|------|----------|---------------|----------|
| Multipart silent truncation | CRITICAL | agent-2 | pkg/cosmoflare/upload.go:207 |
| sync --delete destroys local files (>1000 objects) | CRITICAL | agent-2 | cmd/sync.go:428 |
| Guardrails never enforced (.env uploadable) | HIGH | agent-2, agent-4 | pkg/cosmoflare/guardrails.go:29 |
| Release pipeline 18/18 failed, 0 releases | HIGH | agent-5, agent-6 | .github/workflows/release.yml:31 |
| CI red: internal/interactive races | HIGH | agent-5 | internal/interactive/ |
| Dead ldflags path — binaries report 'dev' | HIGH | agent-5, agent-9 | Makefile:19 |
| .dockerignore kills Docker build | HIGH | agent-9 | .dockerignore:29 |
| Daemon 502-everything error contract | MEDIUM | agent-7 | internal/server/rest.go:94 |
| --json silent exit on config errors | MEDIUM | agent-7 | cmd/root.go:99 |
| Desktop ships unstyled + a11y-invisible health state | HIGH | agent-17, agent-18 | desktop/src/styles.css:1 |
| Vendor shipped same product (cf CLI + MCP) | HIGH (strategic) | agent-4, agent-6 | docs/PRODUCT-VISION.md:25 |
