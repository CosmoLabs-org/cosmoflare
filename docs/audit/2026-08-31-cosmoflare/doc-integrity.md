# Documentation Integrity — 47/100 (DETERMINISTIC)

> Dimension A of the Audit Supercharge. Score from `ccs audit doc-score` (DD-3), reported verbatim. Full agent analysis: agent-14-doc-integrity.md.

## Score Inputs

| Input | Value | Weight effect |
|-------|-------|---------------|
| broken_chains | 7 of 76 | −25 (cap) |
| unenriched_docs | 76 | −10 |
| unreviewed_docs | 127 of 133 | −9.55 |
| stale_docs | 6 | −6 |
| stale_prompts | 2 | −2 |
| **Total** | | **47.45 → 47** |

Formula verified by the agent: 100 − 25 − 10 − 9.55 − 6 − 2 = 47.45 → 47.

## Broken Chains (7) — the 25-point penalty

All one pattern: plans carry `brainstorm_ref:`, the paired brainstorms lack `plan_ref:` back-links. The verifier cannot walk the chain in reverse. Seven one-line edits lift the full cap:

1. `docs/brainstorming/2026-06-07-road020-dashboard-tui.md`
2. `docs/brainstorming/2026-06-08-road002-tui-object-browser.md`
3. `docs/brainstorming/2026-06-08-road007-s3-migration-resume.md`
4. `docs/brainstorming/2026-06-14-domain-management-center.md`
5. `docs/brainstorming/2026-06-20-cosmoflare-desktop.md`
6. `docs/brainstorming/2026-06-21-inprocess-event-bus.md`
7. `docs/brainstorming/2026-07-01-live-metrics-producer.md`

## Review Coverage: 0 of 133 (the independent-review layer does not exist)

`reviewed: 0, unreviewed: 127, stale: 6, total: 133`. The 24 project design docs (8 brainstorms + 16 plans) are the candidates that matter; the rest is 40 historical prompts + 62 plugins/internal docs.

**Per the DD-6 contract this audit detects and points — it does not run the review fleet.** The action plan emits `/independent-review <docs>` as the command. Newest chains first (event-bus, live-metrics).

## Stale Docs (6)

Modified after their last review stamp — the verdict no longer covers the text: road002 / road020 / road007 / domain-management-center brainstorms + both cosmoflare-desktop docs. Re-review after the unreviewed backlog.

## Prompt Freshness

- **2 abandoned** (0/4 and 0/19 goals, 71 days): `docs/prompts/2026-06-21-inprocess-event-bus.md`, `docs/planning-mode/2026-06-21-inprocess-event-bus.md` — never marked ABANDONED.
- **2 PENDING at 71 days**: `2026-06-21-desktop-followups-and-event-bus.md` (3/6 goals — bug half done, event-bus half open), `2026-06-21-master-session-next.md`. Both predate the IDEA-037 metrics work; the "latest" prompt no longer describes the frontier.

## The Enrichment Nuance (important)

`unenriched_docs: 76` equals the full chain-doc count, but agent 14 verified this does **not** mean missing `deliverables:` blocks — 31/35 design docs have them (only README/USAGE directory docs lack blocks). The 76 reflects incomplete *linkage* frontmatter: 201 missing link fields (absent `plan_ref`, absent prompt references, absent valid issue origins). The ADR-005 deliverables layer on recent chains is healthy. The 12 actionable enrichment targets are listed in agent-14-doc-integrity.json.

## Minor Integrity Violations

- 3 date-only timestamps violate ISO8601 constitutional rule (IMP-032): cosmoflare-desktop brainstorm/plan `last_reviewed: "2026-06-20"`, event-bus prompt `created: "2026-06-21"`.
- 3 malformed `origin:` values: `origin: ROAD-002` (verifier rejects — ROADMAP ids not accepted), `origin: "/brainplan"`, `origin: "/brainstorming"`.

## The Feature-That-Isn't (cross-ref work-completion.md)

The event-bus three-tier chain (brainstorm → plan → prompt with `covers_*` deliverable lists) documents a feature with **zero implementation** — agent 14 grepped all Go source: no `EventBus` anywhere. Agent 15 confirmed independently: `internal/events` does not exist; `TriggerAlert` publishes to no bus. Three documents promise one feature. See work-completion.md §FEAT-008.

## Repair Sequence (cheapest points first)

1. Add 7 `plan_ref` back-links → +~25 pts
2. `ccs prompts stale --cleanup` + `ccs prompts scan` → clears abandoned/pending drift
3. Fix 3 origins + 3 timestamps → `ccs timestamps backfill`
4. `/independent-review` burn-down of 24 docs (the long tail; the only path above ~70)
