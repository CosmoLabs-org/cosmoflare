# Risk Assessment — cosmoflare v0.26.0

Synthesized from 13 audit agents (2026-09-13). Every claim cites its source agent report.

## Tech Debt Hotspots

| Hotspot | Risk | Source |
|---------|------|--------|
| `pkg/cosmoflare/sync.go` — plan/execute engine | 4 HIGH data-loss-class bugs; worst file in the codebase | agent-2-core-logic.md |
| 618 `if JSONOutput` branches across cmd/ | Every new command duplicates the pattern; unfixable by refactoring later without touching all 40+ commands | agent-1-code-quality.md |
| 45 god functions >80 lines (worst 238) | `internal/tui/update.go` handleKeyMsg, `runObjectPut`, `loadBuiltinThemes` — change-fragile | agent-1-code-quality.md |
| Untyped daemon↔desktop contract (`any` both ends) | Any daemon payload change silently breaks desktop at runtime; no compile-time check on the product's core boundary | agent-7-api-design.md |
| Three transports, three timeout/retry policies | 27 constructors → `http.DefaultClient` (no timeout); S3 path → 30s (kills large transfers); restClient → own policy | agent-7-api-design.md, agent-2-core-logic.md |

## Fragile Areas

- **Sync engine** (see agent-2-core-logic.md): direction-blind deletes, exclude-less delete loops, no-op `--include`, non-atomic downloads, sequential executor. Highest combined severity × likelihood in the product — this is the flagship workflow.
- **apply** (apply.go:423): deletes any account resource not declared in config — delete-by-omission on a *whole-account* scope (see agent-2-core-logic.md).
- **KV namespace matching by title** (diff.go:280): duplicate titles → wrong namespace deleted (see agent-2-core-logic.md).
- **cmd/ at 44.6% coverage with zero CI execution**: nothing prevents regressions from landing in the CLI layer; the sync bugs are exactly the class tests would have caught (see agent-1-code-quality.md).

## Security Surface

- **Release archives leak transcripts with AWS-key-pattern text** — `make dist` tars `../docs/` into public artifacts (Makefile:97). Scanner's 13 "critical" AWS-key hits are false positives (`AKIAIOSFODNN7EXAMPLE` docs placeholders) *in the repo*, but shipping them in release tarballs is unnecessary exposure (see agent-5-distribution.md, agent-9-infrastructure.md).
- Daemon: non-constant-time token compare with `?token=` query fallback (query strings land in logs/history) (internal/server/server.go:148) (see agent-9-infrastructure.md, agent-7-api-design.md).
- Machine config: tokens written 0644 then chmod'd — crash-window exposure (config.go:169) (see agent-2-core-logic.md).
- API token prompted with terminal echo (cmd/auth.go:417) (see agent-9-infrastructure.md).
- Tauri webview: `csp: null` (tauri.conf.json:19) (see agent-9-infrastructure.md).
- Plaintext keychain fallback without warning (internal/config/config.go:381) (see agent-9-infrastructure.md).

## Chained Risks

1. **sync down --delete × exclude-less deletes × non-atomic downloads** = a single interrupted or mis-flagged down-sync can delete an unrelated remote object AND corrupt the local copy it was meant to replace. Compounding, not independent (see agent-2-core-logic.md).
2. **Private repo × 4 unpublished releases × dead install URLs in GETTING_STARTED.md** = even if visibility flips today, first-time users hit: 404 on pkg.go.dev, v0.22.0 from install.sh, dead CosmoDev repo from the docs one-liner. Launch surface fails at every layer (see agent-6-seo-content.md, agent-5-distribution.md, agent-4-competitive.md).
3. **Docker toolchain drift × 1.4GB build context × dead CI** = the container path fails or wastes massively whenever anyone attempts it; it is also absent from goreleaser, so nobody would notice (see agent-9-infrastructure.md).
4. **Multipart abort-on-cancel × no abort-failure logging** = billed orphaned parts accumulate invisibly; R2 charges storage for incomplete uploads (see agent-2-core-logic.md).

## Single Points of Failure

- **The local release machine** — CI permanently disabled by policy; releases exist only when the maintainer runs goreleaser + gh release manually. The v0.23–v0.26 gap proves this fails silently (see agent-5-distribution.md).
- **`ccs issues` index** — docs/issues/index.yaml is empty (`issues: []`) while 81 issue files exist; anything reading the index sees zero issues (see agent-12-roadmap-health.md).
- **Maintainer attention** — 0 external users, 0 contributors; bus factor 1 across all three tiers (see agent-4-competitive.md).

## Agent Cross-References

| Risk | Severity | Source Agents | Evidence |
|------|----------|---------------|----------|
| Sync engine data-loss bugs | CRITICAL | agent-2 | pkg/cosmoflare/sync.go:275,425 |
| Release archives carry transcript secrets | HIGH | agent-5, agent-9 | Makefile:97 |
| Publish pipeline silent gap (4 releases) | HIGH | agent-5, agent-4 | .goreleaser.yaml:22 |
| auth --revoke-old revokes NEW token | HIGH | agent-9 | cmd/auth.go:227-239 |
| 30s timeout on S3 transfers | HIGH | agent-2 | pkg/cosmoflare/client.go:113 |
| Private repo blocks all distribution | HIGH | agent-6, agent-4 | README.md:10 |
| Daemon token compare non-constant-time | MEDIUM | agent-9, agent-7 | internal/server/server.go:148 |
| cmd/ 44.6% coverage, no CI | MEDIUM | agent-1 | cmd/ |
| Empty issues index | MEDIUM | agent-12 | docs/issues/index.yaml |
