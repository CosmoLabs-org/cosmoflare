# Codebase Patterns — cosmoflare v0.26.0

Synthesized from 13 audit agents (2026-09-13). Every claim cites its source agent report.

## Error Handling

**Two coexisting idioms in cmd/** — legacy `Run` + `printErrorAndExit` (os.Exit inside handler, cmd/create.go:30) vs modern `RunE` returning errors. The legacy path breaks deferred cleanup and makes testing harder (see agent-1-code-quality.md).

**Library layer is disciplined**: `R2Error` wrapping with operation context, consistent error mapping verified accurate by API Design (retry defaults in docs match rest_client.go:25-26 — Documentation agent verified every sampled claim) (see agent-7-api-design.md, agent-10-documentation.md).

**Weak spots**:
- `isNotFound` classifies errors by substring matching — fragile against API message changes (pkg/cosmoflare/config.go, see agent-7-api-design.md)
- Multipart abort failures not logged — orphaned billed uploads silently leak (upload.go:181, multipart.go:467, see agent-2-core-logic.md)
- Download `Close()` errors unchecked; interrupted downloads truncate pre-existing local files (download.go:94, see agent-2-core-logic.md)
- Daemon error paths mix `text/plain` 401/503 with `application/json` responses (internal/server/server.go:134, see agent-7-api-design.md)

## State Management

- **CLI**: stateless per-invocation; config from `.cosmoflare.yaml` + machine config + env vars. `CLOUDFLARE_EMAIL` env var used in code but undocumented anywhere (see agent-10-documentation.md).
- **Daemon**: in-memory pollers per source, 30s limits-snapshot cadence, SSE push. No caching layer — FEAT-014's cadence concern remains live (see agent-15-work-completion.md).
- **Desktop**: React state via hooks, SSE-fed; notifications panel uses per-view `role="log"` live regions that unmount on tab switch (see agent-18-accessibility.md).
- **Guardrails**: sync guardrail tests exist (`sync_guardrails_test.go`) but enforcement covers uploads only — deletes and copies bypass the chokepoint entirely (storage.go:227, see agent-2-core-logic.md).

## Naming Conventions

**Split product identity is systemic** — three brand generations coexist:
- `pkg/cosmoflare/types.go:2` package doc says "Package r2go2" (this is the pkg.go.dev landing text)
- `docs/README.md:1` titled "CosmoDev-R2Go2"; GETTING_STARTED.md install URL points at dead CosmoDev-R2Go2 repo
- `.version-registry.json:4` description still "R2Go2 R2-only CLI"
- Makefile release path r2go2-branded with different binary/asset names than goreleaser
(see agent-6-seo-content.md, agent-4-competitive.md, agent-5-distribution.md, agent-7-api-design.md — 4 agents, independent confirmations)

**Go code follows consistent patterns**: `*Service` per Cloudflare service, `NewXServiceFromCreds` constructor family, options structs. 45 functions exceed 80 lines (worst 238: `internal/tui/update.go` `handleKeyMsg`) (see agent-1-code-quality.md).

## File Organization

- Clean `cmd/` + `pkg/cosmoflare/` + `internal/` separation per CLAUDE.md conventions; tests live alongside sources (239 test files, see metrics.json).
- **Repo pollution is the organizational anomaly**: 93 conversation-transcripts (8MB) + 83 raw JSONL session dumps (95MB) + GOrchestra logs committed — 570K+ lines of non-code, confirmed by 4 agents (see agent-1-code-quality.md, agent-9-infrastructure.md, agent-10-documentation.md; ROAD-085 notes GOrchestra tracked size already reduced 234MB→7.3MB per agent-12-roadmap-health.md).
- Tracked build binaries in repo root: `cmd.test`, `r2go2`, `r2go2-enhanced`, `r2go2-tui-installer` (~105MB stale) (see metrics.json largest_files, agent-9-infrastructure.md).
- `docs/architecture/` exists as unfilled template — zero ADRs (see agent-10-documentation.md).

## Testing Patterns

- 35/35 Go packages + 27/27 TS test runs green this audit; near-zero TODO debt (2 TODOs in 165K Go lines) (see agent-1-code-quality.md).
- **cmd/ layer at 44.6% coverage** — library well-tested, CLI handlers under-tested; no CI executes any suite (policy: local releases) (see agent-1-code-quality.md).
- Desktop has vitest-axe a11y gate (a11y.test.tsx) — unusual discipline for a 10-component app (see agent-17-design-taste.md).
- Regression tests exist for sync guardrails but not for direction-aware deletes — the exact class that is broken (see agent-2-core-logic.md).

## Agent Cross-References

| Pattern | Source Agent | Evidence |
|---------|-------------|----------|
| Run vs RunE split | agent-1-code-quality.md | cmd/create.go:30 |
| Triple-brand fragmentation | agent-6, agent-4, agent-5, agent-7 | types.go:2, docs/README.md:1, Makefile:14 |
| Uploads-only guardrails | agent-2-core-logic.md | pkg/cosmoflare/storage.go:227 |
| Transcript/log pollution | agent-1, agent-9, agent-10 | docs/conversation-transcripts/, GOrchestra/ |
| Untyped wire contract | agent-7-api-design.md | internal/server/rest.go:24 |
| Per-view live regions unmount | agent-18-accessibility.md | desktop/src/App.tsx:152 |
