# Agent 14: Documentation Integrity

> Audit: 2026-08-31, cosmoflare v0.17.0, fresh mode. Report saved verbatim from agent output.
> Dimension score is DETERMINISTIC (DD-3): `ccs audit doc-score` = 47/100. The agent reports it verbatim.

All evidence is collected. Here is my complete audit report.

## Documentation Integrity: 47/100

**Deterministic score (DD-3)**: `ccs audit doc-score --json` returned `score: 47`. This is the dimension score. I report it verbatim. Formula verified: 100 − 25 (broken-chain cap) − 10 (unenriched) − 9.55 (unreviewed) − 6 (stale docs) − 2 (stale prompts) = 47.45 → 47.

**Score inputs (from the tool, not re-derived)**:

| Input | Value |
|-------|-------|
| chains_total | 76 |
| broken_chains | 7 |
| unenriched_docs | 76 |
| reviewed_docs | 0 |
| unreviewed_docs | 127 |
| stale_docs | 6 |
| stale_prompts | 2 |

**Sub-scores**:
| Dimension | Score | Notes |
|-----------|-------|-------|
| chain_integrity | 6/10 | 7 of 76 chains orphaned (9%). No dangling file references. Every referenced file exists. Orphans are one-way links, not dead files. 201 of 278 link checks report "missing" frontmatter fields. |
| enrichment | 6/10 | `deliverables:` blocks exist on 31 of 35 design docs (14/16 brainstorms, 17/19 plans). Only README/USAGE files lack them. But linkage frontmatter is incomplete on 100% of chain docs, and 3 origin fields are malformed. |
| review_coverage | 1/10 | Zero review stamps across all 133 tracked docs. 127 never reviewed. 6 stale. The independent-review layer does not exist in this project. |
| prompt_freshness | 6/10 | 2 of 40 prompts abandoned. 2 PENDING prompts are 71 days old. One prompt (desktop, 10/10 goals) closed correctly. |

### Critical Findings

1. **Seven plan→brainstorm chains are orphaned** — Each plan carries `brainstorm_ref:`. The paired brainstorm carries no `plan_ref:` back-link. The verifier cannot walk the chain in reverse. This defect consumed the full 25-point broken-chain penalty. All seven pairs:
   - `docs/planning-mode/2026-06-07-road020-dashboard-tui.md` → brainstorm lacks `plan_ref`
   - `docs/planning-mode/2026-06-08-road002-tui-object-browser.md` → same
   - `docs/planning-mode/2026-06-08-road007-s3-migration-resume.md` → same
   - `docs/planning-mode/2026-06-14-domain-management-center.md` → same
   - `docs/planning-mode/2026-06-20-cosmoflare-desktop.md` → same
   - `docs/planning-mode/2026-06-21-inprocess-event-bus.md` → same
   - `docs/planning-mode/2026-07-01-live-metrics-producer.md` → same

   Verified example: the plan at `docs/planning-mode/2026-07-01-live-metrics-producer.md` line 6 declares `brainstorm_ref: docs/brainstorming/2026-07-01-live-metrics-producer.md`. The brainstorm frontmatter (lines 1-10) has `id`, `created`, `updated`, `title`, `status`, `tags`, `related_issues`, `deliverables` — no `plan_ref`.
   - **Severity**: high
   - **File**: the seven `docs/brainstorming/2026-*.md` files above
   - **Fix**: Add one `plan_ref:` line to each brainstorm's frontmatter. Seven one-line edits.

2. **Review coverage is zero — 0 of 133 docs reviewed** — `ccs doc-review status` reports `reviewed: 0, unreviewed: 127, stale: 6, total: 133`. No document in this project carries an independent review stamp. The unreviewed set breaks down as: 9 brainstorming docs, 16 planning-mode docs, 40 prompts, 62 `plugins/internal` docs. The 24 project design docs (8 brainstorms + 16 plans, excluding templates) are the review candidates that matter.
   - **Severity**: high
   - **File**: `docs/brainstorming/` and `docs/planning-mode/` (24 design docs)
   - **Fix**: Run `/independent-review` on the 24 unreviewed design docs. The event-bus and live-metrics chains deserve review first — they are the newest.

3. **Event-bus three-tier chain documents a feature that does not exist** — The full ADR-005 chain was scaffolded on 2026-06-21: brainstorm, plan, prompt. The prompt declares `covers_brainstorm_deliverables: [BR-01..BR-05]` and `covers_plan_deliverables: [P-01..P-04]` (lines 4-14 of `docs/prompts/2026-06-21-inprocess-event-bus.md`). I searched all Go source: zero matches for `EventBus` or `event bus` in `cmd/`, `internal/`, `pkg/`. The prompt sits at 0/4 goals. The plan sits at 0/19 goals. Both are flagged abandoned. Meanwhile goal G-04 of the still-PENDING `docs/prompts/2026-06-21-desktop-followups-and-event-bus.md` (line 62, ROAD-080 / FEAT-008) names this same feature as open work. Three documents promise one feature. No code delivers it.
   - **Severity**: high
   - **File**: `docs/prompts/2026-06-21-inprocess-event-bus.md`, `docs/planning-mode/2026-06-21-inprocess-event-bus.md`, `docs/brainstorming/2026-06-21-inprocess-event-bus.md`
   - **Fix**: Decide the feature's fate. If deferred, mark ROAD-080/FEAT-008 as planned and run `ccs prompts stale --cleanup` on the pair. If dead, close the roadmap items and archive the chain.

4. **Six docs are stale — modified after their last review stamp** — Five brainstorms plus one plan. Example: `docs/brainstorming/2026-06-07-road002-tui-object-browser.md` was reviewed 2026-06-08T00:30:00-03:00 and modified 2026-06-08T05:19:40-03:00. The review verdict no longer covers the current text.
   - **Severity**: medium
   - **File**: the six paths listed in the stale section above
   - **Fix**: Re-review the six stale docs after the 24 unreviewed ones.

5. **Two PENDING prompts are 71 days old** — `docs/prompts/2026-06-21-desktop-followups-and-event-bus.md` (created 2026-06-21T08:10:00-03:00, status PENDING, 3/6 goals done) is still the project's "latest" prompt. G-01/G-02/G-03 (BUG-022/023/024) are checked complete — consistent with the modified BUG yaml files in git status. G-04 through G-06 remain open. `docs/prompts/2026-06-21-master-session-next.md` (created 2026-06-21T01:10:08-03:00, status PENDING) is also open. Both predate the entire IDEA-037 metrics work (commits 4a7bdd2..be8637b). The "latest" prompt no longer describes the actual frontier.
   - **Severity**: medium
   - **File**: `docs/prompts/2026-06-21-desktop-followups-and-event-bus.md:1`, `docs/prompts/2026-06-21-master-session-next.md:1`
   - **Fix**: Run `ccs prompts scan` to reconcile goal state. Then author a fresh continuation prompt covering the real remaining work.

6. **Three date-only timestamps violate the ISO8601 constitutional rule (IMP-032)** — `docs/brainstorming/2026-06-20-cosmoflare-desktop.md:4` has `last_reviewed: "2026-06-20"`. `docs/planning-mode/2026-06-20-cosmoflare-desktop.md:4` has the same. `docs/prompts/2026-06-21-inprocess-event-bus.md:15` has `created: "2026-06-21"`. Date-only stamps lose intra-day ordering.
   - **Severity**: low
   - **File**: the three locations above
   - **Fix**: Rewrite each stamp to full ISO8601 with timezone. Use `ccs timestamps backfill`.

7. **Three brainstorms carry malformed `origin:` values** — `docs/brainstorming/2026-06-07-road002-tui-object-browser.md:6` has `origin: ROAD-002`. The verifier rejects it: "ROAD-002 doesn't contain a valid issue ID". `docs/brainstorming/2026-05-16-domain-operations.md:5` has `origin: "/brainplan"`. `docs/brainstorming/2026-05-16-cors-transform-rules.md:5` has `origin: "/brainstorming"`. Origin-to-issue traceability is broken for these docs.
   - **Severity**: low
   - **File**: the three locations above
   - **Fix**: Point each `origin:` at a real issue ID (e.g. `FEAT-008`) or a ROADMAP ref the parser accepts.

**Enrichment reconciliation (important nuance)**: The score input `unenriched_docs: 76` equals the full chain-doc count. I verified this does NOT mean missing `deliverables:` blocks — 14/16 brainstorms and 17/19 plans have them (only README/USAGE files lack blocks, and those are directory docs). The 76 count reflects incomplete linkage frontmatter: 201 "missing" link fields across all chain docs (absent `plan_ref`, absent prompt references, absent valid issue origins). The ADR-005 deliverables layer on recent chains is healthy — the live-metrics chain carries BR-*/P-* IDs and the prompt carries `covers_*` fields.

### Recommendations

- [ ] Add `plan_ref:` back-links to the 7 orphaned brainstorms — lifts the 25-point cap penalty (effort: small)
- [ ] Run `/independent-review` on the 24 unreviewed design docs, newest chains first (effort: large)
- [ ] Decide the event-bus feature fate: implement ROAD-080/FEAT-008 or close the chain and mark the roadmap items (effort: small decision, large if implemented)
- [ ] Run `ccs prompts scan` to reconcile the 71-day-old PENDING prompts, then `ccs prompts stale --cleanup` for the abandoned pair (effort: small)
- [ ] Fix the 3 date-only timestamps and 3 malformed origins with `ccs timestamps backfill` + frontmatter edits (effort: small)
- [ ] Adopt a session-end rule: every new brainstorm gets `plan_ref` at plan-creation time, so chains never go one-way again (effort: small)

### Roadmap Suggestions

- **Doc-chain repair pass** — Fix 7 orphaned back-links, 3 malformed origins, 3 date-only stamps in one sweep; restores ~28 score points (priority: high, effort: small)
- **Independent review backlog burn-down** — 24 design docs with zero review stamps; batch through `/independent-review` weekly (priority: medium, effort: large)
- **Event-bus disposition (ROAD-080/FEAT-008)** — Three docs promise the feature, zero code exists; either schedule implementation or close the chain (priority: high, effort: small)

### Data Appendix (for action-plan.md)

**broken_chains** (7): plans for road020-dashboard-tui, road002-tui-object-browser, road007-s3-migration-resume, domain-management-center, cosmoflare-desktop, inprocess-event-bus, live-metrics-producer — each brainstorm lacks `plan_ref`. Fix = add `plan_ref` to each brainstorm frontmatter.

**unenriched_docs** (12 actionable): 2026-03-28-r2go2-product-vision-and-integration, 2026-05-16-cors-transform-rules, 2026-05-16-domain-operations, 2026-05-18-service-interfaces, 2026-05-18-tui-command-palette, 2026-06-07-road002-tui-object-browser, 2026-06-07-road020-dashboard-tui, 2026-06-08-road007-s3-migration-resume, 2026-06-14-domain-management-center, 2026-06-20-cosmoflare-desktop, 2026-06-21-inprocess-event-bus, 2026-07-01-live-metrics-producer (all in docs/brainstorming/). Tool counts 76 docs as unenriched (201 missing link fields). Only README/USAGE files literally lack deliverables blocks; 31/35 design docs have them.

**unreviewed_docs** (24 design-doc candidates for `/independent-review`): the 8 brainstorms above + feature-ideas.md; 16 planning-mode docs from 2025-11-24-r2go2-initial-development through 2026-07-01-live-metrics-producer. Remaining 103 unreviewed: 40 docs/prompts/*.md + 62 plugins/internal docs + template/README/USAGE files. 6 further docs are stale (modified after review): road002/road020/road007/domain-management-center brainstorms + both cosmoflare-desktop docs.

**stale_prompts**: docs/prompts/2026-06-21-inprocess-event-bus.md and docs/planning-mode/2026-06-21-inprocess-event-bus.md (both abandoned, 0/4 and 0/19 goals). Additionally 2 non-flagged PENDING prompts 71 days old: docs/prompts/2026-06-21-desktop-followups-and-event-bus.md (3/6 goals) and docs/prompts/2026-06-21-master-session-next.md — candidates for `ccs prompts scan`.
