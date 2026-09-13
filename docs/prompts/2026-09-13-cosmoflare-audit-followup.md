---
status: PENDING
type: audit-followup
priority: high
---

# Audit Followup: Cosmoflare 2026-09-13

## Start Here — Action Plan "Now" tier (from action-plan.md)

1. **Fix the 4 sync-engine data-loss bugs** — direction-aware deletes (`pkg/cosmoflare/sync.go:425`), exclude-protected delete loops (`:275`), split 30s API client from timeout-free S3 client (`pkg/cosmoflare/client.go:113`), `context.WithoutCancel` multipart aborts (`pkg/cosmoflare/upload.go:181`) — red-first TDD each (work-completion of the flagship workflow; agent-2-core-logic.md)
2. **Stop `make dist` from taring docs/ into release archives** — `Makefile:97` (public-artifact secret exposure; agent-5-distribution.md)
3. **Fix `auth rotate --revoke-old` revoking the NEW token** — `cmd/auth.go:227-239` (agent-9-infrastructure.md)

## Critical Bugs to Fix First

1. sync down --delete deletes wrong remote object — pkg/cosmoflare/sync.go:425 — branch Execute on plan.Direction
2. sync up --delete deletes excluded remote objects — pkg/cosmoflare/sync.go:275 — isExcluded check in delete loops
3. 30s shared HTTP timeout kills large S3 transfers — pkg/cosmoflare/client.go:113 — dedicated data-plane client
4. Multipart abort orphans billed uploads — pkg/cosmoflare/upload.go:181 — WithoutCancel + log failures
5. auth --revoke-old revokes NEW token — cmd/auth.go:227-239 — capture old token before overwrite

## Quick Wins (small effort, high impact)

1. Rebuild empty issues index — `ccs issues index-rebuild` (docs/issues/index.yaml is `issues: []`)
2. Publish v0.26.0 — `gh release create v0.26.0 dist/upload/* --notes-from-tag` (assets already staged)
3. Fix 4 broken doc chains — add `plan_ref:` to the 2026-09-09/10 brainstorms
4. Add `plan_ref` + close FEAT-025/FEAT-019 (merges 562a602/4f512ef verified) — after human scope check
5. Fix GETTING_STARTED.md install URL (points at dead CosmoDev repo)
6. Rewrite `pkg/cosmoflare/types.go:2` package doc (pkg.go.dev landing text says "Package r2go2")
7. role=status on HealthDot span + app-level aria-live region (desktop/src/App.tsx:152)

## Roadmap Items to Start

1. Roadmap hygiene sweep #2 (link 16 open FEATs, dedupe ROAD-047/048 + 073/079, fix ROAD-084/085 — do NOT auto-complete 084)
2. Agent-first onboarding umbrella (FEAT-017 wizard + FEAT-019 token doctor — the latter already merged)
3. Remote MCP transport (streamable HTTP + pagination) — official cloudflare/mcp is closing the window
4. Launch & distribution program (purge → publish → public → Show HN/awesome-lists) — after Phase 0 fixes

## Files to Read First

- docs/audit/latest/action-plan.md (prioritized copy-paste next-steps — START HERE)
- docs/audit/latest/brief.md (project context)
- docs/audit/latest/risk-map.md (where NOT to touch without tests)
- docs/audit/latest/surprises.md (non-obvious findings — private-repo trap, ROAD-084 auto-fix trap)
