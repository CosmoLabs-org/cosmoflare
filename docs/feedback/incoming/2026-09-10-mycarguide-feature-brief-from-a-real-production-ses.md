---
ulid: 01M26009ZMNH2ZQDAJNW3YTKXZ
title: 'FEATURE BRIEF from a real production session (MyCarGuide, 2026-09-10) — every item below is friction cosmoflare could have erased. Context: Cloudflare Pages + D1 (1.41M-row specs table, 0.94GB DB) + R2 + KV; all ops driven by an AI session agent.'
type: feature
status: converted
priority: medium
complexity: complex
from_project: MyCarGuide
from_path: /Users/gabstudio/PROJECTS/MyCarGuide
to_project: cosmoflare
to_target: project
created: "2026-09-10T19:46:55.604743+04:00"
updated: "2026-09-11T00:30:41.658305+04:00"
suggested_conversion: feature
converted_to: FEAT-018
related_issues: []
brainstorm_ref: null
session: 17
suggested_workflow:
    - brainstorming
    - plan-mode
    - implementation
response:
    acknowledged: null
    acknowledged_by: null
    started: null
    implemented: null
    rejected: null
    rejection_reason: null
    notes: ""
---

# FB-p3YTKXZ: FEATURE BRIEF from a real production session (MyCarGuide, 2026-09-10) — every item below is friction cosmoflare could have erased. Context: Cloudflare Pages + D1 (1.41M-row specs table, 0.94GB DB) + R2 + KV; all ops driven by an AI session agent.

WHAT HAPPENED (evidence, all today):
1. Remote D1 apply of a 35MB seed (84,437 INSERT...SELECT statements): `wrangler d1 execute --remote --file` cannot handle it. I hand-wrote a byte-capped batcher (infra/scripts/d1-push-epa-specs.mjs, ~110 lines, cloning the d1-push-all-tables.mjs pattern — this pattern now exists in TWO MyCarGuide scripts): 40KB multi-value INSERT OR IGNORE batches via --command, concurrency 2, SQLITE_TOOBIG split-retry. One batch died with "internal error code: 7500" — no auto-retry existed, needed a clean second pass.
2. DB size discovery: `wrangler d1 info mycarguide-db --remote` rejects --remote and dumps the ENTIRE help text (token burn for an agent); size only extractable via `--json` parsing.
3. Usage-vs-limits blindness (URGENT): since 2026-09-01 Cloudflare ENFORCES free-tier D1 daily limits by failing queries (changelog). There is no CLI way to see today's rows read/written vs the budget. Today's EPA push alone wrote ~100K rows — at the free-tier daily write cap. We cannot check from the terminal whether the next write will fail.
4. Deploy+verify loop (done 3× today): build with PUBLIC_BUILD_ID=$(git rev-parse --short HEAD) → wrangler pages deploy → 6+ manual curls to verify routes/islands live. No single command.
5. Local-vs-remote parity checks: manual sqlite3 counts + wrangler --json counts per table, done twice by hand (TASK-009 pattern).
6. WAF rate-limit rule for /catalog.json: deferred 5 sessions running — needs dashboard interaction and threshold design.
7. wrangler --json quirks: rows_written parsing needs fallbacks (error JSON rides inside exit-0 stdout).

WHY IT MATTERS: items 1-6 each cost 10-45 min of agent session time; the batch-push pattern has been re-implemented per-project at least twice; item 3 is production-outage class after the 2026-09-01 enforcement change.

PROPOSED FEATURES (prioritized):
P0 —
- `cosmoflare d1 push-sql FILE --db X [--remote] [--concurrency N]`: statement-aware SQL splitter (quote/comment/semicolon-aware — never the naive ';' split), ~40KB byte-capped batches, TOOBIG split-retry, retry-with-backoff on internal error 7500, progress lines, kill-safe resume (INSERT OR IGNORE idempotency).
- `cosmoflare d1 usage --db X --days 7` + `cosmoflare d1 status`: rows read/written per day vs plan budget (GraphQL analytics API), DB size vs storage limit, per-table row counts, warning threshold. This is the outage-prevention command.
- `cosmoflare pages deploy --verify /garage,/api/recalls?...`: stamp PUBLIC_BUILD_ID from git, deploy, poll until live, run the verify URL list, print a pass/fail table, non-zero exit on any failure.
P1 —
- `cosmoflare d1 parity --tables a,b,c`: local-vs-remote row-count diff in one shot.
- `cosmoflare waf ratelimit PATH --requests N --period SECONDS [--dry-run]`: close the deferred-decision loop entirely from the terminal.
- JSON-first, agent-friendly output everywhere: `--json` default, terse errors, no flag-trap help dumps (wrangler d1 info --remote).
P2 —
- `cosmoflare r2 du / put-bulk` (wrap the r2-bulk-put pattern with concurrency+retries), `cosmoflare env diff` (named-env binding/route inheritance gotchas), `cosmoflare pages deployments / rollback`.

DESIGN PRINCIPLES: idempotent-by-default; kill-safe resumability (our long jobs get OOM-killed — everything must survive re-run); cost-aware (print estimated rows read/written per command — D1 bills per row); machine-readable output for AI agents first, human pretty second.

PRIORITY JUSTIFICATION: every P0 item blocked or slowed concrete production work in this single session; usage-vs-limits became outage-risk nine days ago.

## Suggested Workflow

1. brainstorming
2. plan-mode
3. implementation

