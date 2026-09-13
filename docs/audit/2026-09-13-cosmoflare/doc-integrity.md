# Documentation Integrity Report — Dimension A

**Deterministic score: 59/100** (`ccs audit doc-score`, DD-3 — authoritative, reported verbatim). Prior: 47 → **+12**.

| Input | Count | Weight applied |
|-------|-------|----------------|
| Broken chains | 4 | −24pt (6pt each) |
| Un-enriched docs | 102 | −10pt (capped) |
| Unreviewed docs | 69 | −7pt (capped) |
| Stale docs | 0 | 0 |
| Stale prompts | 0 | 0 |

Decomposition verified by Agent 14 against the formula (see agent-14-doc-integrity.md).

## Findings (see agent-14-doc-integrity.md for full arrays)

1. **HIGH — Zero machine-verifiable enrichment (102/102 docs un-enriched).** No plan carries `covers_brainstorm_deliverables:`; zero deliverables are machine-verifiable. Fix: `ccs prompts enrich <plan> --apply` over the 23 flagged plans; re-derive prompts via `ccs prompts init --from-plan`.
2. **HIGH — Review coverage 29% (28/97).** One 2026-09-06 sweep covered docs ≤2026-07-01; the entire September design wave (15 docs incl. all 2026-09-06→09-11 plans) is unreviewed. Handoff: `/independent-review <15 docs>` (per DD-6 the audit detects and points; it never runs the review fleet).
3. **MED — 4 orphaned plan→brainstorm edges.** The 2026-09-09/10 batch (cf-api-knowledge-layer, domain-center-residuals, quota-limit-view, ratelimit-traffic-classes) — brainstorms lack `plan_ref` back-links. Clears all 4 broken chains with frontmatter edits.
4. **MED — Origin/issue traceability absent for 66 docs** (46 no origin, 20 migrated without issue ID).
5. **MED — 18 UNKNOWN-status prompts need migration; 6 PENDING + 3 ABANDONED need triage** (2026-09-12-session-2026-continuation likely done).
6. **LOW — Stale-detector blind spot:** 28/28 reviewed docs modified after stamp yet stale_docs=0 — timestamp comparison likely includes the stamp write. Report to ClaudeCodeSetup.
7. **LOW — 6 README/USAGE index files fail frontmatter parse** and pollute the chain universe.

## Next commands (detect-and-point, DD-6)

```bash
# broken chains (4): add plan_ref to the 4 brainstorms
# enrichment: ccs prompts enrich <plan> --apply   (23 plans, Sept first)
# reviews: /independent-review docs/brainstorming/2026-09-09-cf-api-knowledge-layer.md ... (15 docs)
# triage: ccs prompts scan; ccs prompts set-status; ccs prompts migrate
```
