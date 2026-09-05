---
status: SUPERSEDED
type: audit-followup
priority: high
created: 2026-08-31T03:15:00-03:00
superseded_by: "docs/prompts/2026-09-05-next-session.md"
completed: "2026-09-05T20:25:51+04:00"
---
# Audit Followup: Cosmoflare 2026-08-31

## Start Here — Action Plan "Now" tier (from action-plan.md)

1. **Fix the multipart silent-corruption path** — uploads can store truncated objects with no error. `pkg/cosmoflare/upload.go:207` + `multipart.go:493`: abort on EOF before `uploadedBytes == size`; add short-reader regression test. (data-integrity)
2. **Paginate sync's remote listing** — `cmd/sync.go:428`: loop on `result.NextToken`; removes the `--delete` data-destruction path on >1000-object buckets. (data-integrity)
3. **Fix `internal/interactive` data races** — they fail the `-race` gate and block every release. Reproduce with `go test -race ./internal/interactive/`. (blocking releases)
4. **Sweep the dead `CosmoDev-R2Go2` module path** — Makefile:19, Dockerfile:30-32, release.yml:62-64+87, install.sh/ps1/menu.sh. Every binary currently reports `version dev`. (blocking releases)
5. **Decide FEAT-008** — re-scope to "bridge TriggerAlert → existing sseHub" (standalone bus obsolete since hub shipped), then `ccs prompts stale --cleanup`. (drift)

## Critical Bugs to Fix First

1. AUD-001: multipart truncated-object storage — pkg/cosmoflare/upload.go:207 — abort on short read
2. AUD-002: sync `--delete` destroys local files (>1000 objects) — cmd/sync.go:428 — paginate NextToken
3. AUD-004: WithPartSize(0) panic — pkg/cosmoflare/upload.go:184 — ValidatePartSize
4. AUD-005: watcher phantom deletions — pkg/cosmoflare/watcher.go:124 — never delete under failed reads
5. AUD-003: guardrails never enforced — pkg/cosmoflare/guardrails.go:29 — wire CheckUpload
6. AUD-006: WithTimeout never wired — pkg/cosmoflare/client.go:91 — pass httpClient to both SDKs
7. AUD-011: .dockerignore excludes go.mod/go.sum — Docker build broken — remove *.mod/*.sum patterns

## Quick Wins (small effort, high impact)

1. Add 7 `plan_ref` back-links to orphaned brainstorms — lifts doc-integrity 25 pts (see doc-integrity.md)
2. `ccs prompts stale --cleanup` + `ccs prompts scan` — clears 71-day prompt drift
3. Close duplicate roadmap stubs ROAD-074/081/082/083; regenerate index.yaml
4. Untrack GOrchestra/sessions + committed binaries (234MB); slim `make dist`
5. Regenerate README service table from CLAUDE.md matrix; fix kv put / worker deploy syntax
6. Desktop: `role="log"` + `aria-live` on notifications, `aria-label` health state, brand → h1 (6 ARIA edits, one session)

## Roadmap Items to Start

1. Restore the binary release channel — first GitHub Release ever (races + ldflags + installers), then v0.18.0 with smoke gate
2. Desktop design system implementation — write the cf-* stylesheet (app currently unstyled)
3. Agent trust tier — guardrail profiles + SIEM-ready audit export (the defensible post-`cf` niche)
4. Full-surface MCP server — auto-generate tools from the cobra tree (currently 8 of 366 commands)
5. Positioning pivot ADR — cf CLI + official MCP response

## Files to Read First

- docs/audit/latest/action-plan.md (prioritized copy-paste next-steps — START HERE)
- docs/audit/latest/brief.md (project context)
- docs/audit/latest/risk-map.md (where NOT to touch without tests)
- docs/audit/latest/surprises.md (non-obvious findings, incl. vacuous anti-slop 9/10)
