# Codebase Patterns — cosmoflare v0.17.0

> Synthesized from 13 agent reports (2026-08-31 audit).

## Error Handling

**Strong at the library core** (agent-1, agent-7):
- `R2Error` hierarchy with `Unwrap()` (`pkg/cosmoflare/errors.go:14-24`): typed `notFound`/`auth`/`quota`/`accessDenied`/`validation` constructors used uniformly across all 23 services.
- `retry.go:123-156` consumes the taxonomy for retryability decisions.
- Tests assert error behavior via real `httptest` servers, not mocks (agent-1: "behavioral, not mock-theater").

**Broken at every transport boundary** — the taxonomy exists but no edge maps it (agent-7):
- REST daemon: every service error → `502 {"error": ...}` (`internal/server/rest.go:94`). Invalid profile, expired token, missing resource — indistinguishable.
- CLI: `printError` returns immediately when `JSONOutput` is true (`cmd/root.go:197-202`), so config failures in `--json` mode print *nothing* and exit 1 — breaking the project's own agent-first contract.
- Error content types mix `text/plain` and JSON across daemon and dev server (agent-7: `server.go:135`, `dev.go:161`).

**Silent-swallow anti-pattern in data paths** (agent-2):
- `io.ReadFull` short-read treated as success in both multipart loops (`upload.go:207`, `multipart.go:493`).
- Watcher walk errors returned as `nil` (`watcher.go:124-127`) → unreadable subtree → phantom deletions.

## State Management

- **Daemon**: hub-and-subscribe SSE; drop-on-full at buffer 32 (`sse.go:43-53`); delta suppression via `reflect.DeepEqual` on `any` payloads (agent-2: expensive, fragile; agent-1: same). No replay, no heartbeat — idle streams get proxy-dropped (agent-2, agent-7).
- **Desktop**: local `useState` only, no state library — diverges from mandated Zustand 5 profile (agent-17). React Query with retry capped at 1 (`App.tsx:28`).
- **CLI**: package-level globals (`JSONOutput` flag) drive 489 branch sites — the largest duplication mass in the codebase (agent-1: sample `cmd/email.go:356-359`).
- **Sync idempotency is broken**: download sets mtime=now, comparison requires `!local.ModTime.After(remote.LastModified)` → permanent re-download loop (`sync.go:368`, agent-2).

## Naming Conventions

- **Service template is consistent**: `*Service` struct + `New*Service(ctx, cfg)` + typed errors per service file (agent-1).
- **Rename debt is the dominant inconsistency** — `r2go2` survives in: error prefix (`errors.go:16-21`), package doc (`types.go:2` "Package r2go2"), interface doc (`client.go:14` "R2Client is ... R2Go2 operations"), config path (`~/.r2go2/`), ldflags paths, installers, Makefile, docs, version-registry description (agents 1, 4, 5, 6, 7, 9, 10, 12 — 8 of 13 agents independently hit this).
- Class vocabulary (`cf-*` BEM-ish) is consistent in desktop TSX but has zero backing CSS (agent-17).

## File Organization

- Layering follows the documented split (cmd / pkg / internal) (agent-1).
- **God files**: `cmd/object.go` 1183 lines (`runObjectPut` 230 lines × 3 duplicated paths), `cmd/email.go` 1063 lines (17 same-shape handlers), `cmd/installer_tui/main.go` 1299 lines single model (agent-1).
- **Dead structure**: 6 empty internal dirs, 4 `//go:build disabled` packages, 4 `.go.disabled` files (agent-1); root-level doc sprawl of pre-rename artifacts (agent-10).
- **Repo-as-attic**: 234MB GOrchestra recovery patches tracked (some 665K lines), 419MB git pack, committed binaries (agents 1, 5, 9).
- Tests live alongside sources; 193 test files vs 202 code files; coverage polarized — internals 88-99%, cmd 43.2%, `internal/cli/operations` 0%, `internal/migration` 10.4% (agent-1).

## Testing Patterns

- **The proven pattern**: `httptest` server fronting cloudflare-go, asserting real HTTP behavior (`cmd/kv_test.go`, `pkg/cosmoflare/kv_test.go:17-22`) (agent-1). Extending it to the top-10 commands is the cheapest coverage win.
- **Contract tests exist for the daemon**: 401-without-token and `?profile=` scoping asserted against a fake source (`internal/server/rest_test.go:66-96`) (agent-7).
- **Gap**: race-detector failures in `internal/interactive` (package-level shared state across 520 tests) block CI and Release `-race` gates (agent-5).

## Agent Cross-References

| Pattern | Source Agent | Evidence |
|---------|-------------|----------|
| Typed error taxonomy discarded at boundaries | agent-7-api-design.md | internal/server/rest.go:94 |
| 489 duplicated JSONOutput branches | agent-1-code-quality.md | cmd/email.go:356 |
| Silent short-read acceptance | agent-2-core-logic.md | pkg/cosmoflare/upload.go:207 |
| r2go2→cosmoflare rename debt (8 agents) | agents 1,4,5,6,7,9,10,12 | pkg/cosmoflare/errors.go:16 et al. |
| httptest behavioral testing (keep, extend) | agent-1-code-quality.md | cmd/kv_test.go |
| Drop-on-full SSE + no heartbeat | agent-2-core-logic.md, agent-7-api-design.md | internal/server/sse.go:43,109 |
| God-file triple duplication | agent-1-code-quality.md | cmd/object.go:483-713 |
