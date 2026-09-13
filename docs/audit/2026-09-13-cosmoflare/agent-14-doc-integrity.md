# Agent 14: Documentation Integrity — 59/100 (DETERMINISTIC)

**Deterministic score (DD-3)**: `ccs audit doc-score --json` = **59/100** — reported verbatim, not re-derived.
Score decomposition (verified): 100 − 24 (4 broken chains × 6 pts) − 10 (100% un-enriched × 0.1, under 12-pt cap) − 7 (71% un-reviewed × 0.1, under 12-pt cap) − 0 (stale) ≈ 59.
Inputs: chains_total=102, broken_chains=4, unenriched_docs=102, reviewed_docs=28, unreviewed_docs=69, stale_docs=0, stale_prompts=0, vendored_excluded_docs=64.

**Sub-scores** (qualitative 1-10 reads; overall comes from the formula above):
| Dimension | Score | Notes |
|-----------|-------|-------|
| chain_integrity | 7/10 | 98/102 chains intact; 4 broken edges, all one-sided plan→brainstorm back-links from one Sept batch; zero dangling file references — no doc content lost |
| enrichment | 2/10 | 102/102 docs un-enriched per binary; 0/25 plans carry `covers_brainstorm_deliverables:`; 19/55 prompts lack `requires_reading:`; `deliverables:` lines exist in 49 docs but no machine-verifiable coverage chain closes |
| review_coverage | 3/10 | 28/97 reviewed (29%), 0 stale; single review sweep 2026-09-06 covering docs dated ≤2026-07-01; the entire post-2026-07-01 wave — the most current design work — has never been reviewed |
| prompt_freshness | 7/10 | 0 stale prompts (`ccs prompts stale` empty); debt is state-hygiene not staleness: 6 PENDING, 3 ABANDONED, 18 UNKNOWN (needs migration) |

## Evidence Base

Sources read: briefing `docs/audit/2026-09-13-cosmoflare/_ctx/brief-14-doc-integrity.md`; full chain verdicts `docs/audit/2026-09-13-cosmoflare/_ctx/doc-chain.json` (3189 lines, parsed via jq — all 102 docs, 409 link verdicts); review coverage `docs/audit/2026-09-13-cosmoflare/_ctx/doc-status.json` (161 doc entries); frontmatter verification greps across ~110 doc files (brainstorms/plans/prompts); `ccs prompts status -v` output. ~12 primary artifacts plus bulk frontmatter scans.

Link-verdict tally across the 102-doc universe (deterministic, from doc-chain.json):
- `ok` 164, `missing` 245, `orphaned` 4, `dangling` 0.
- Missing-link causes (top): 46 no origin/issue/issue_ref; 42 prompts without brainstorm in `requires_reading`; 42 prompts without plan in `requires_reading`; 23 plans without `covers_brainstorm_deliverables`; 20 `origin="migrated by ccs prompts migrate"` with no issue ID; 13 brainstorms without `plan_ref`; 13 plans never referenced by any prompt; 12 plans without `brainstorm_ref`; 9 brainstorms never referenced by any prompt; 6 README/USAGE index files failing frontmatter parse.

## Critical Findings

1. **Zero machine-verifiable enrichment across the whole chain universe (102/102 un-enriched)** — The `deliverables:` → `covers_brainstorm_deliverables:` coverage loop required by ADR-005 does not close anywhere in the project. Not one of the 25 plans carries `covers_brainstorm_deliverables:` (grep: 0 matches; chain tool flags 23 where the check applies). 19 of 55 prompts lack `requires_reading:`. 49 docs carry a bare `deliverables:` line, but without the coverage back-reference the binary cannot verify any deliverable as implemented. This is the second-largest score penalty (−10) and makes `ccs prompts load-context` and `verify` unable to gate anything.
   - **Severity**: high
   - **File**: all 25 `docs/planning-mode/*.md` (0 carry `covers_brainstorm_deliverables:`); e.g. `docs/planning-mode/2026-09-11-d1-depth.md`
   - **Fix**: `ccs prompts enrich <plan> --apply` for each plan (23 flagged by the chain tool), then re-derive prompts via `ccs prompts init --from-plan` so `requires_reading:`/`covers_*` fields are machine-generated, never hand-authored.

2. **Review coverage 29%, and the newest documentation is the least reviewed** — One review sweep on 2026-09-06 stamped 28 docs, all dated ≤2026-07-01. Every design doc created after that date is unreviewed: the FEAT-008 alert/SSE bridge plan, the entire 2026-09-09/10/11 wave (cf-api-knowledge-layer, domain-center-residuals, quota-limit-view, ratelimit-traffic-classes, cf-limits-awareness-layer, d1-depth), plus 2026-06-21-inprocess-event-bus. 69 unreviewed total: 54 prompts, 8 brainstorms, 7 plans (excluding 64 vendored `plugins/` docs). Overlap compounds the risk: the same 4 docs with broken chains are also unreviewed — no independent eyes have touched the September chain at all.
   - **Severity**: high
   - **File**: the 15 unreviewed design docs (listed in full below); `docs/audit/2026-09-13-cosmoflare/_ctx/doc-status.json` (all 28 `last_reviewed` values are 2026-09-06)
   - **Fix**: `/independent-review <the 15 design docs>` (per DD-6 contract this audit detects and points; the review fleet does the stamping).

3. **4 orphaned plan→brainstorm edges (the 2026-09-09/10 batch)** — Exactly 4 broken chains (matching `broken_chains: 4`). Each plan declares `brainstorm_ref:` but the paired brainstorm carries no `plan_ref:` back-link (verified: `grep -c plan_ref` = 0 in all 4 brainstorm files). The files exist; only the reverse edge is missing, so the chain is one-sided:
   1. `docs/planning-mode/2026-09-09-cf-api-knowledge-layer.md` → `docs/brainstorming/2026-09-09-cf-api-knowledge-layer.md` = **orphaned** (brainstorm has no `plan_ref`)
   2. `docs/planning-mode/2026-09-09-domain-center-residuals.md` → `docs/brainstorming/2026-09-09-domain-center-residuals.md` = **orphaned**
   3. `docs/planning-mode/2026-09-09-quota-limit-view.md` → `docs/brainstorming/2026-09-09-quota-limit-view.md` = **orphaned**
   4. `docs/planning-mode/2026-09-10-ratelimit-traffic-classes.md` → `docs/brainstorming/2026-09-10-ratelimit-traffic-classes.md` = **orphaned**
   - **Severity**: medium (largest per-item penalty, 6 pts each = −24)
   - **File**: the 4 planning-mode files above (each carries `brainstorm_ref:`; each target brainstorm lacks `plan_ref:`)
   - **Fix**: add `plan_ref:` to the 4 brainstorms' frontmatter via `ccs prompts` tooling (same field pattern as the working pairs, e.g. `docs/brainstorming/2026-06-14-domain-management-center.md`).

4. **Origin/issue traceability absent for 66 docs** — 46 docs have no origin/issue/issue_ref at all; 20 more carry `origin="migrated by ccs prompts migrate"` which contains no issue ID; 8 use non-identifier origins (`/brainplan`, `manual`, `/continuation-prompt`, `/session-end`, `/brainstorming`). Consequence: no doc-to-issue lineage for two-thirds of the universe — the chain tool reports `Origin → Issue: missing`, and audit/coverage tooling cannot tie design docs back to FEAT/ROAD/TASK items (e.g. FEAT-012 shows `status: unknown` because the linkage is one-sided).
   - **Severity**: medium
   - **File**: e.g. `docs/brainstorming/2026-03-28-r2go2-product-vision-and-integration.md` (no origin at all); 20 migrated docs across all three tiers
   - **Fix**: `ccs prompts enrich <doc> --apply` backfills issue linkage where the issue exists; for genuinely issue-less historical docs, accept and suppress via tooling config rather than fake IDs.

5. **18 docs in UNKNOWN status needing migration + 9 actionable prompt states** — `ccs prompts status -v`: 7 prompts, 8 plans, 3 session docs carry no YAML status metadata ("needs migration"). Separately (named from frontmatter): 6 PENDING prompts — `2026-06-21-master-session-next.md`, `2026-06-21-desktop-followups-and-event-bus.md`, `2026-09-08-post-v0.22.0-control-plane.md`, `2026-09-10-cf-limits-corpus-research-ingestion.md`, `2026-09-10-cosmoflare-continuation.md`, `2026-09-12-session-2026-continuation.md` — and 3 ABANDONED — `docs/prompts/2026-05-14-session-continuation.md`, `docs/prompts/2026-06-21-inprocess-event-bus.md`, `docs/planning-mode/2026-06-21-inprocess-event-bus.md`. Note: `ccs prompts stale` returns empty, so none are stale by the tool's definition — this is metadata debt, not freshness loss. Two of the PENDING prompts (2026-09-10-cosmoflare-continuation, 2026-09-12-session-2026-continuation) may describe work already done and should be verified against reality.
   - **Severity**: medium
   - **File**: listed above
   - **Fix**: `ccs prompts scan` to auto-complete finished work; `ccs prompts set-status` for the confirmed-dead ones; migration for the 18 UNKNOWN.

6. **Stale-detector blind spot: 28/28 reviewed docs modified after their review stamp yet stale_docs=0** — Every reviewed doc's `last_modified` (2026-09-06T20:15:38) is ~95s after its `last_reviewed` (2026-09-06T20:14:03) — consistent with the review sweep itself rewriting files to stamp them. The tool reports 0 stale, implying same-day (or coarser) comparison granularity. Real same-day post-review edits would go undetected. Low project impact today (the deltas are the stamp writes), but the "0 stale" figure should not be read as "reviews are current" — nothing has been re-reviewed in the 7 days since the single sweep.
   - **Severity**: low
   - **File**: `docs/audit/2026-09-13-cosmoflare/_ctx/doc-status.json` (all 28 reviewed entries)
   - **Fix**: report to ClaudeCodeSetup (`ccs feedback send`): compare stale at timestamp granularity, excluding the stamp-write itself.

7. **6 README/USAGE index files fail frontmatter parsing and pollute the chain universe** — `docs/{brainstorming,planning-mode,prompts}/{README,USAGE}.md` are counted as chain docs but have no YAML frontmatter ("Could not parse: no YAML frontmatter delimiters found"). They inflate `chains_total` and can never be enriched or reviewed, permanently capping the score.
   - **Severity**: low
   - **File**: the 6 files above
   - **Fix**: feedback to ClaudeCodeSetup to exclude README/USAGE from the doc-review universe (they are directory indexes, not ADR-005 chain docs).

## Recommendations

- [ ] Add `plan_ref:` to the 4 orphaned brainstorms (`2026-09-09-cf-api-knowledge-layer`, `2026-09-09-domain-center-residuals`, `2026-09-09-quota-limit-view`, `2026-09-10-ratelimit-traffic-classes`) via ccs prompts tooling — clears all 4 broken chains (effort: small)
- [ ] Run `ccs prompts enrich <doc> --apply` over the 23 flagged plans, prioritizing the 6 Sept plans that anchor current work (effort: medium)
- [ ] Dispatch `/independent-review` on the 15 unreviewed design docs — starts with the 6 Sept brainstorms/plans which are both unreviewed AND chain-broken (effort: medium)
- [ ] Triage prompt states: `ccs prompts scan` for the 6 PENDING (especially `2026-09-10-cosmoflare-continuation.md` and `2026-09-12-session-2026-continuation.md`, likely done), `ccs prompts set-status` for the 3 ABANDONED (effort: small)
- [ ] Migrate the 18 UNKNOWN-status docs (`ccs prompts migrate`) (effort: small)
- [ ] Send both tooling feedback items to ClaudeCodeSetup: stale-detector timestamp granularity; README/USAGE exclusion from the chain universe (effort: small)

## Roadmap Suggestions

- **Documentation spine cadence: enrich + review at session-end** — Every ADR-005 chain doc gets `ccs prompts enrich` at creation and `/independent-review` before its feature ships; prevents the "newest docs least reviewed" pattern from recurring (priority: high, effort: medium)
- **Prompt metadata migration completion** — Clear the 18 UNKNOWN + 9 PENDING/ABANDONED backlog and add a `ccs sync` step that fails on regressions (priority: medium, effort: small)

## Full Lists (deterministic)

### Unreviewed design docs (15) — `/independent-review` candidates

docs/brainstorming/2026-06-21-inprocess-event-bus.md, 2026-09-09-cf-api-knowledge-layer.md, 2026-09-09-domain-center-residuals.md, 2026-09-09-quota-limit-view.md, 2026-09-10-cf-limits-awareness-layer.md, 2026-09-10-ratelimit-traffic-classes.md, 2026-09-11-d1-depth.md, collaborative-template.md;
docs/planning-mode/2026-06-21-inprocess-event-bus.md, 2026-09-06-feat008-alert-sse-bridge.md, 2026-09-09-cf-api-knowledge-layer.md, 2026-09-09-domain-center-residuals.md, 2026-09-09-quota-limit-view.md, 2026-09-10-ratelimit-traffic-classes.md, 2026-09-11-d1-depth.md
(all under `docs/brainstorming/` / `docs/planning-mode/`; collaborative-template.md is a template — arguably exempt from review)

### Un-enriched docs (102/102 — the entire chain universe)

The doc-score input `unenriched_docs: 102` equals `chains_total: 102`: every doc in `docs/brainstorming/` (22), `docs/planning-mode/` (25), and `docs/prompts/` (55) lacks the machine-verifiable coverage fields. The full path list is in `agent-14-doc-integrity.json` → `unenriched_docs`. Highest-leverage subset: the 23 plans flagged "No covers_brainstorm_deliverables" and the 19 prompts without `requires_reading:` (42 prompt→plan/brainstorm gaps: 42 prompts reference neither plan nor brainstorm in `requires_reading`).

### Unreviewed docs (69, vendored `plugins/` excluded)

54 in `docs/prompts/` (full list in `agent-14-doc-integrity.json` → `unreviewed_docs`), plus the 15 design docs above. The 64 `plugins/internal/skills/**/skill.md` files are vendored and excluded from scoring.

### Stale prompts

None — `ccs prompts stale` returns empty; `stale_prompts: 0` in doc-score inputs.

```json:audit-result
{
  "agent": "doc-integrity",
  "overall_score": 59,
  "score_scale": 100,
  "score_source": "ccs audit doc-score --json (deterministic, DD-3) — reported verbatim",
  "sub_scores": {
    "chain_integrity": 7,
    "enrichment": 2,
    "review_coverage": 3,
    "prompt_freshness": 7
  },
  "critical_findings": [
    {
      "title": "Zero machine-verifiable enrichment (102/102 docs un-enriched)",
      "severity": "high",
      "file": "docs/planning-mode/2026-09-11-d1-depth.md (representative; 0/25 plans carry covers_brainstorm_deliverables)",
      "fix": "ccs prompts enrich <plan> --apply for the 23 flagged plans; re-derive prompts via ccs prompts init --from-plan so requires_reading/covers_* are machine-generated",
      "effort": "medium"
    },
    {
      "title": "Review coverage 29%; entire post-2026-07-01 documentation wave unreviewed",
      "severity": "high",
      "file": "docs/planning-mode/2026-09-06-feat008-alert-sse-bridge.md (plus 14 more design docs listed in unreviewed_docs)",
      "fix": "/independent-review on the 15 unreviewed design docs; prioritize the 6 Sept docs that are also chain-broken",
      "effort": "medium"
    },
    {
      "title": "4 orphaned plan-to-brainstorm edges (2026-09-09/10 batch lacks plan_ref back-links)",
      "severity": "medium",
      "file": "docs/planning-mode/2026-09-09-cf-api-knowledge-layer.md, docs/planning-mode/2026-09-09-domain-center-residuals.md, docs/planning-mode/2026-09-09-quota-limit-view.md, docs/planning-mode/2026-09-10-ratelimit-traffic-classes.md",
      "fix": "Add plan_ref: to the 4 paired brainstorm frontmatters via ccs prompts tooling",
      "effort": "small"
    },
    {
      "title": "Origin/issue traceability absent for 66 docs (46 no origin, 20 migrated-origin without issue ID)",
      "severity": "medium",
      "file": "docs/brainstorming/2026-03-28-r2go2-product-vision-and-integration.md (representative)",
      "fix": "ccs prompts enrich backfills issue linkage where the issue exists; suppress tooling noise for genuinely issue-less historical docs",
      "effort": "medium"
    },
    {
      "title": "18 UNKNOWN-status docs need migration; 6 PENDING + 3 ABANDONED prompts need triage",
      "severity": "medium",
      "file": "docs/prompts/2026-09-12-session-2026-continuation.md (PENDING, likely done)",
      "fix": "ccs prompts scan for PENDING; ccs prompts set-status for ABANDONED; ccs prompts migrate for UNKNOWN",
      "effort": "small"
    },
    {
      "title": "Stale-detector blind spot: 28/28 reviewed docs modified after review stamp yet stale_docs=0",
      "severity": "low",
      "file": "docs/audit/2026-09-13-cosmoflare/_ctx/doc-status.json (all 28 reviewed entries)",
      "fix": "ccs feedback send to ClaudeCodeSetup: compare at timestamp granularity excluding the stamp write",
      "effort": "small"
    },
    {
      "title": "6 README/USAGE index files fail frontmatter parse and pollute the chain universe",
      "severity": "low",
      "file": "docs/prompts/README.md (plus 5 sibling index files)",
      "fix": "ccs feedback send: exclude README/USAGE directory indexes from the doc-review universe",
      "effort": "small"
    }
  ],
  "recommendations": [
    { "action": "Add plan_ref to the 4 orphaned brainstorms (clears all 4 broken chains)", "effort": "small", "priority": "high" },
    { "action": "ccs prompts enrich --apply over the 23 flagged plans, Sept plans first", "effort": "medium", "priority": "high" },
    { "action": "/independent-review on the 15 unreviewed design docs", "effort": "medium", "priority": "high" },
    { "action": "ccs prompts scan + set-status triage for 6 PENDING / 3 ABANDONED prompts", "effort": "small", "priority": "medium" },
    { "action": "ccs prompts migrate for the 18 UNKNOWN-status docs", "effort": "small", "priority": "medium" },
    { "action": "Send stale-granularity and README/USAGE-exclusion feedback to ClaudeCodeSetup", "effort": "small", "priority": "low" }
  ],
  "roadmap_suggestions": [
    { "title": "Documentation spine cadence: enrich + review at session-end", "description": "Every ADR-005 chain doc gets enriched at creation and independently reviewed before its feature ships", "priority": "high", "effort": "medium" },
    { "title": "Prompt metadata migration completion", "description": "Clear the 18 UNKNOWN + 9 PENDING/ABANDONED backlog and gate regressions in ccs sync", "priority": "medium", "effort": "small" }
  ],
  "broken_chains": [
    { "doc": "docs/planning-mode/2026-09-09-cf-api-knowledge-layer.md", "broken_edge": "plan -> brainstorm orphaned: docs/brainstorming/2026-09-09-cf-api-knowledge-layer.md has no plan_ref back-link", "fix": "Add plan_ref: docs/planning-mode/2026-09-09-cf-api-knowledge-layer.md to the brainstorm frontmatter via ccs prompts tooling" },
    { "doc": "docs/planning-mode/2026-09-09-domain-center-residuals.md", "broken_edge": "plan -> brainstorm orphaned: docs/brainstorming/2026-09-09-domain-center-residuals.md has no plan_ref back-link", "fix": "Add plan_ref back-link to brainstorm frontmatter" },
    { "doc": "docs/planning-mode/2026-09-09-quota-limit-view.md", "broken_edge": "plan -> brainstorm orphaned: docs/brainstorming/2026-09-09-quota-limit-view.md has no plan_ref back-link", "fix": "Add plan_ref back-link to brainstorm frontmatter" },
    { "doc": "docs/planning-mode/2026-09-10-ratelimit-traffic-classes.md", "broken_edge": "plan -> brainstorm orphaned: docs/brainstorming/2026-09-10-ratelimit-traffic-classes.md has no plan_ref back-link", "fix": "Add plan_ref back-link to brainstorm frontmatter" }
  ],
  "unenriched_docs": ["docs/brainstorming/2026-03-28-r2go2-product-vision-and-integration.md","docs/brainstorming/2026-05-16-cors-transform-rules.md","docs/brainstorming/2026-05-16-domain-operations.md","docs/brainstorming/2026-05-18-service-interfaces.md","docs/brainstorming/2026-05-18-tui-command-palette.md","docs/brainstorming/2026-06-07-road002-tui-object-browser.md","docs/brainstorming/2026-06-07-road020-dashboard-tui.md","docs/brainstorming/2026-06-08-road007-s3-migration-resume.md","docs/brainstorming/2026-06-14-domain-management-center.md","docs/brainstorming/2026-06-20-cosmoflare-desktop.md","docs/brainstorming/2026-06-21-inprocess-event-bus.md","docs/brainstorming/2026-07-01-live-metrics-producer.md","docs/brainstorming/2026-09-09-cf-api-knowledge-layer.md","docs/brainstorming/2026-09-09-domain-center-residuals.md","docs/brainstorming/2026-09-09-quota-limit-view.md","docs/brainstorming/2026-09-10-cf-limits-awareness-layer.md","docs/brainstorming/2026-09-10-ratelimit-traffic-classes.md","docs/brainstorming/2026-09-11-d1-depth.md","docs/brainstorming/README.md","docs/brainstorming/USAGE.md","docs/brainstorming/collaborative-template.md","docs/brainstorming/feature-ideas.md","docs/planning-mode/2025-11-24-r2go2-initial-development.md","docs/planning-mode/2026-02-26-codebase-analysis-v0.2.0.md","docs/planning-mode/2026-03-03-consolidate-duplicated-code.md","docs/planning-mode/2026-05-07-r2go2-phase0-phase1-implementation.md","docs/planning-mode/2026-05-12-phase3-workers-kv.md","docs/planning-mode/2026-05-16-cosmoflare-module-rename.md","docs/planning-mode/2026-05-16-domain-operations.md","docs/planning-mode/2026-05-16-phase4-cloudflare-services.md","docs/planning-mode/2026-05-18-service-interfaces.md","docs/planning-mode/2026-05-18-tui-command-palette.md","docs/planning-mode/2026-06-07-road020-dashboard-tui.md","docs/planning-mode/2026-06-08-road002-tui-object-browser.md","docs/planning-mode/2026-06-08-road007-s3-migration-resume.md","docs/planning-mode/2026-06-14-domain-management-center.md","docs/planning-mode/2026-06-20-cosmoflare-desktop.md","docs/planning-mode/2026-06-21-inprocess-event-bus.md","docs/planning-mode/2026-07-01-live-metrics-producer.md","docs/planning-mode/2026-09-06-feat008-alert-sse-bridge.md","docs/planning-mode/2026-09-09-cf-api-knowledge-layer.md","docs/planning-mode/2026-09-09-domain-center-residuals.md","docs/planning-mode/2026-09-09-quota-limit-view.md","docs/planning-mode/2026-09-10-ratelimit-traffic-classes.md","docs/planning-mode/2026-09-11-d1-depth.md","docs/planning-mode/README.md","docs/planning-mode/USAGE.md","docs/prompts/2025-01-25-tui-installer-wizard-implementation.md","docs/prompts/2026-03-02-testing-strategy-progress.md","docs/prompts/2026-03-07-testing-quality-next-steps.md","docs/prompts/2026-03-25-project-audit-followup.md","docs/prompts/2026-03-28-r2go2-audit-followup.md","docs/prompts/2026-05-08-phase1-library-extraction.md","docs/prompts/2026-05-08-phase2-continuation.md","docs/prompts/2026-05-08-phase2-integration-and-hardening.md","docs/prompts/2026-05-12-phase2-roadmap-expansion.md","docs/prompts/2026-05-13-phase3-workers-kv-implementation.md","docs/prompts/2026-05-14-batch2-batch3-quick-wins.md","docs/prompts/2026-05-14-session-continuation.md","docs/prompts/2026-05-16-cosmoflare-phase4-rebrand.md","docs/prompts/2026-05-16-domain-operations.md","docs/prompts/2026-05-16-glm-parallel-coverage-wave2.md","docs/prompts/2026-05-19-post-audit-fixes.md","docs/prompts/2026-05-27-next-session.md","docs/prompts/2026-05-30-next-session.md","docs/prompts/2026-06-06-next-session.md","docs/prompts/2026-06-07-next-session.md","docs/prompts/2026-06-09-next-session.md","docs/prompts/2026-06-14-domain-management-center.md","docs/prompts/2026-06-20-cosmoflare-desktop.md","docs/prompts/2026-06-21-desktop-followups-and-event-bus.md","docs/prompts/2026-06-21-inprocess-event-bus.md","docs/prompts/2026-06-21-master-session-next.md","docs/prompts/2026-08-31-cosmoflare-audit-followup.md","docs/prompts/2026-09-05-next-session.md","docs/prompts/2026-09-06-launch-and-hardening.md","docs/prompts/2026-09-07-launch-gate-and-v0.21.0.md","docs/prompts/2026-09-08-post-v0.22.0-control-plane.md","docs/prompts/2026-09-09-cf-api-knowledge-layer.md","docs/prompts/2026-09-09-domain-center-residuals.md","docs/prompts/2026-09-09-quota-limit-view.md","docs/prompts/2026-09-10-cf-limits-corpus-research-ingestion.md","docs/prompts/2026-09-10-cosmoflare-continuation.md","docs/prompts/2026-09-10-cosmoflare-handoff.md","docs/prompts/2026-09-10-ratelimit-traffic-classes.md","docs/prompts/2026-09-11-d1-depth.md","docs/prompts/2026-09-12-session-2026-continuation.md","docs/prompts/CONTINUATION-session-008-cli-features.md","docs/prompts/CONTINUATION-session-008-tui-installer.md","docs/prompts/PARALLEL-SESSION-COORDINATION.md","docs/prompts/README.md","docs/prompts/USAGE.md","docs/prompts/cli-feature-commands.md","docs/prompts/continuation-prompt-v0.2.0.md","docs/prompts/enhanced-interactive-features.md","docs/prompts/session-007-continuation-completed.md","docs/prompts/session-007-continuation-prompt.md","docs/prompts/session-D-testing-continuation.md","docs/prompts/session-e-continuation-prompt.md","docs/prompts/testing-strategy-bulletproof-cli.md","docs/prompts/testing-validation-production.md","docs/prompts/tui-dashboard-implementation.md"],
  "unreviewed_docs": ["docs/brainstorming/2026-06-21-inprocess-event-bus.md","docs/brainstorming/2026-09-09-cf-api-knowledge-layer.md","docs/brainstorming/2026-09-09-domain-center-residuals.md","docs/brainstorming/2026-09-09-quota-limit-view.md","docs/brainstorming/2026-09-10-cf-limits-awareness-layer.md","docs/brainstorming/2026-09-10-ratelimit-traffic-classes.md","docs/brainstorming/2026-09-11-d1-depth.md","docs/brainstorming/collaborative-template.md","docs/planning-mode/2026-06-21-inprocess-event-bus.md","docs/planning-mode/2026-09-06-feat008-alert-sse-bridge.md","docs/planning-mode/2026-09-09-cf-api-knowledge-layer.md","docs/planning-mode/2026-09-09-domain-center-residuals.md","docs/planning-mode/2026-09-09-quota-limit-view.md","docs/planning-mode/2026-09-10-ratelimit-traffic-classes.md","docs/planning-mode/2026-09-11-d1-depth.md","docs/prompts/2025-01-25-tui-installer-wizard-implementation.md","docs/prompts/2026-03-02-testing-strategy-progress.md","docs/prompts/2026-03-07-testing-quality-next-steps.md","docs/prompts/2026-03-25-project-audit-followup.md","docs/prompts/2026-03-28-r2go2-audit-followup.md","docs/prompts/2026-05-08-phase1-library-extraction.md","docs/prompts/2026-05-08-phase2-continuation.md","docs/prompts/2026-05-08-phase2-integration-and-hardening.md","docs/prompts/2026-05-12-phase2-roadmap-expansion.md","docs/prompts/2026-05-13-phase3-workers-kv-implementation.md","docs/prompts/2026-05-14-batch2-batch3-quick-wins.md","docs/prompts/2026-05-14-session-continuation.md","docs/prompts/2026-05-16-cosmoflare-phase4-rebrand.md","docs/prompts/2026-05-16-domain-operations.md","docs/prompts/2026-05-16-glm-parallel-coverage-wave2.md","docs/prompts/2026-05-19-post-audit-fixes.md","docs/prompts/2026-05-27-next-session.md","docs/prompts/2026-05-30-next-session.md","docs/prompts/2026-06-06-next-session.md","docs/prompts/2026-06-07-next-session.md","docs/prompts/2026-06-09-next-session.md","docs/prompts/2026-06-14-domain-management-center.md","docs/prompts/2026-06-20-cosmoflare-desktop.md","docs/prompts/2026-06-21-desktop-followups-and-event-bus.md","docs/prompts/2026-06-21-inprocess-event-bus.md","docs/prompts/2026-06-21-master-session-next.md","docs/prompts/2026-08-31-cosmoflare-audit-followup.md","docs/prompts/2026-09-05-next-session.md","docs/prompts/2026-09-06-launch-and-hardening.md","docs/prompts/2026-09-07-launch-gate-and-v0.21.0.md","docs/prompts/2026-09-08-post-v0.22.0-control-plane.md","docs/prompts/2026-09-09-cf-api-knowledge-layer.md","docs/prompts/2026-09-09-domain-center-residuals.md","docs/prompts/2026-09-09-quota-limit-view.md","docs/prompts/2026-09-10-cf-limits-corpus-research-ingestion.md","docs/prompts/2026-09-10-cosmoflare-continuation.md","docs/prompts/2026-09-10-cosmoflare-handoff.md","docs/prompts/2026-09-10-ratelimit-traffic-classes.md","docs/prompts/2026-09-11-d1-depth.md","docs/prompts/2026-09-12-session-2026-continuation.md","docs/prompts/CONTINUATION-session-008-cli-features.md","docs/prompts/CONTINUATION-session-008-tui-installer.md","docs/prompts/PARALLEL-SESSION-COORDINATION.md","docs/prompts/cli-feature-commands.md","docs/prompts/continuation-prompt-v0.2.0.md","docs/prompts/enhanced-interactive-features.md","docs/prompts/session-007-continuation-completed.md","docs/prompts/session-007-continuation-prompt.md","docs/prompts/session-D-testing-continuation.md","docs/prompts/session-e-continuation-prompt.md","docs/prompts/testing-strategy-bulletproof-cli.md","docs/prompts/testing-validation-production.md","docs/prompts/tui-dashboard-implementation.md","docs/prompts/worktrees/2026-05-15-UnitTesting-session-2.md"],
  "stale_prompts": [],
  "pending_prompts": ["docs/prompts/2026-06-21-master-session-next.md","docs/prompts/2026-06-21-desktop-followups-and-event-bus.md","docs/prompts/2026-09-08-post-v0.22.0-control-plane.md","docs/prompts/2026-09-10-cf-limits-corpus-research-ingestion.md","docs/prompts/2026-09-10-cosmoflare-continuation.md","docs/prompts/2026-09-12-session-2026-continuation.md"],
  "abandoned_prompts": ["docs/prompts/2026-05-14-session-continuation.md","docs/prompts/2026-06-21-inprocess-event-bus.md","docs/planning-mode/2026-06-21-inprocess-event-bus.md"]
}
```
