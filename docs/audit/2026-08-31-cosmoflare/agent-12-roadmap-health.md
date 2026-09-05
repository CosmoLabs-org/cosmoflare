# Agent 12: Roadmap Health

> Audit: 2026-08-31, cosmoflare v0.17.0, fresh mode. Report saved verbatim from agent output.

I have all the evidence needed. Compiling the report.

## Roadmap Health: 5/10

**Sub-scores**:
| Dimension | Score | Notes |
|-----------|-------|-------|
| infrastructure | 6/10 | Real structured system (99 items, index.yaml, ccs tooling) but index drifted from item files and 5 duplicate item pairs exist |
| coverage | 6/10 | 78 completed items cover shipped history well; post-freeze work (IDEA-037 metrics) and Domain Center Waves B–D untracked |
| linkage | 5/10 | 13/36 issues backlinked; 19 of 20 open roadmap items are thin (no issue, no brainstorm, no plan) |
| staleness | 3/10 | Roadmap frozen 70 days; zero in_progress items; all 20 captured items 70–115 days old with no progression |
| hydration_needed | 3/10 | 19/20 open items thin; 4 strategic themes missing entirely; needs substantial hydration |

### Critical Findings

1. **Roadmap frozen for 70 days — no grooming cadence** — The last roadmap update was 2026-06-21 (`docs/roadmap/index.yaml` `last_updated: 2026-06-21T08:07:38`). Today is 2026-08-31. Zero items touched in that window even though 7 commits landed on 2026-07-01 (IDEA-037 metrics + BUG-022/023/024 fixes). All 20 captured items are 70–115 days old (`ROAD-029` oldest at 115 days). Status distribution is 78 completed / 20 captured / 1 archived / **0 in_progress / 0 planned** — the pipeline is binary: work either completes instantly or sits captured forever.
   - **Severity**: high
   - **File**: `docs/roadmap/index.yaml:3` (last_updated), `docs/roadmap/items/ROAD-029.yaml` (2026-05-07, untouched since)
   - **Fix**: Run a grooming pass (`ccs roadmap lint` + `ccs roadmap health`): promote ROAD-080 (plan exists, ready to build), park or archive stale BASE-xxx foundation items, and set a recurring cadence. Fold into `/triage` step 9.

2. **Five duplicate roadmap item pairs from a double batch import** — On 2026-06-20 the "expand with 6 items surfaced this session" commit (8d4076c) wrote items ROAD-073..078 at 21:01:03, then the same six titles were written again as ROAD-079..083 at 21:01:18 — 15 seconds apart. Confirmed duplicate pairs: ROAD-073/079 ("Local API daemon", both completed), ROAD-074/080 ("In-process event bus"), ROAD-076/081 ("Desktop distribution"), ROAD-077/082 ("Paid-tier licensing"), ROAD-078/083 ("cmd/ DI testability rollout"). The stub copies (074, 076, 077, 078) have priority 0, no category, no links and pollute health reporting.
   - **Severity**: medium
   - **File**: `docs/roadmap/items/ROAD-074.yaml` (duplicate of ROAD-080), `docs/roadmap/items/ROAD-076.yaml` through `ROAD-078.yaml`
   - **Fix**: Archive ROAD-074, ROAD-076, ROAD-077, ROAD-078 (the priority-0 stubs); keep the richer 079–083 versions. If `ccs roadmap` lacks an archive flag, move them to status `archived` like ROAD-023.

3. **index.yaml has drifted from item files** — The rendered index contradicts the source files. ROAD-035: file title is "DNS record management — CRUD for A, AAAA, CNAME, MX, TXT, SRV records" but index title is "Email Routing — create rules, list destinations, catch-all settings" — a wrong-title mapping. ROAD-025 through ROAD-032 have `priority` in their files (60–85) but no priority in the index, so `ccs roadmap list` reports them as priority 0. The index has not been regenerated since 2026-06-21.
   - **Severity**: medium
   - **File**: `docs/roadmap/index.yaml` (ROAD-035 entry, ROAD-025..032 entries) vs `docs/roadmap/items/ROAD-035.yaml`
   - **Fix**: Regenerate the index from item files (re-run the roadmap indexing command or `ccs roadmap lint --fix`), and resolve the ROAD-035 title collision (two different items mapped to one ID).

4. **Post-freeze work bypassed the roadmap entirely** — The live-metrics feature shipped on 2026-07-01 through the ideas pipeline only: IDEA-037 → brainstorm (4d2b2f9) → plan (c5c33ed) → 3 feat commits (MetricsProducer, `--metrics-interval` flag, SSE→React Query bridge). No roadmap item was ever created and the roadmap was not updated. This is the exact "prose/pipeline work not reflected in structured roadmap" failure the audit looks for — 100% of post-freeze shipped work (1 of 1 feature) is absent from the roadmap.
   - **Severity**: medium
   - **File**: `docs/ideas/2026-06-21-desktop-dashboard-is-not-live-metrics-sse-channel-has-no.md` (harvested), `docs/planning-mode/` IDEA-037 plan
   - **Fix**: Create a retroactive completed roadmap item for live metrics (completed_in: v0.17.0), and make `/idea promote`/`ccs idea harvest` create or update a roadmap item automatically.

5. **Domain Management Center Waves B–D tracked nowhere** — FEAT-006 is closed with a description covering only Waves 1–2 (library+CLI, TUI), but project memory records "Wave A (Redirect/Registrar services) landed; Waves B–D remaining". The remaining waves exist only in memory and the original brainplan. No roadmap item, no open issue. The only open issue in the entire project is FEAT-008.
   - **Severity**: medium
   - **File**: `docs/issues/FEAT-006.yaml` (status: closed; description mentions two waves only)
   - **Fix**: Create a roadmap item "Domain Center completion — Waves B–D" linking FEAT-006 and the 2026-06-14 brainplan/plan docs.

6. **README feature table contradicts the roadmap** — README.md lines 35–43 list Page Rules, WAF/Firewall, D1 Database, Pages, Email Routing, Images, Stream, and Workers AI as **"Planned"**, but the roadmap marks them completed and CLAUDE.md lists all as "Implemented". A reader assessing the project from README gets a roadmap picture months out of date.
   - **Severity**: medium
   - **File**: `README.md:35-43`
   - **Fix**: Update the README status column to match `docs/issues`/roadmap completed state, or generate the table from the roadmap.

7. **19 of 20 open roadmap items are thin** — `ccs roadmap health` reports `healthy: false`, 19 thin items: only ROAD-080 has a linked issue and a plan. The other 19 (all BASE-00x foundation items, ROAD-029, ROAD-064, ROAD-074–078 stubs) have no issue, no brainstorm, no plan — they are titles only. Several BASE items are plausibly already satisfied (BASE-002 "Tech stack defined" — a full tech-stack exists; BASE-010 "Linting & formatting enforced" — go vet passes) but were never closed.
   - **Severity**: medium
   - **File**: `docs/roadmap/items/BASE-002.yaml`, `docs/roadmap/items/BASE-010.yaml` (98 days captured, likely done)
   - **Fix**: Audit the 9 BASE items against reality; close satisfied ones. For the rest, either link an issue or archive.

8. **Stale prose roadmap artifacts mislead** — Three prose roadmap docs predate the structured system and were never reconciled: `docs/roadmap/PLANNED_FEATURES.md` (last touched 2026-02-26, still titled "R2Go2 Planned Feature Roadmap" — old branding), `docs/roadmap/roadmap.md` (2025-11-24), `docs/roadmap/GUI-INTEGRATION-STRATEGY.md` (2025-11-24, 721 lines). CLAUDE.md also claims "Roadmap items (ROAD-001 through ROAD-072)" — items actually run to ROAD-083 plus BASE-001–015.
   - **Severity**: low
   - **File**: `docs/roadmap/PLANNED_FEATURES.md:1`, `docs/roadmap/roadmap.md`, `CLAUDE.md` (roadmap range claim)
   - **Fix**: Delete or clearly deprecate the three stale prose docs (their content is superseded by item YAMLs); fix the CLAUDE.md range to "ROAD-000 through ROAD-083 + BASE-001–015".

9. **Continuation prompt PENDING 70 days after its scope completed** — `docs/prompts/2026-06-21-desktop-followups-and-event-bus.md` is `status: PENDING, priority: high`, but its bug scope (BUG-022/023/024) was closed on 2026-07-01. Only the ROAD-080 event bus half remains open — and `internal/webhook/` still has no subscribe API (grep for Subscribe/pubsub in `internal/webhook/manager.go`: no matches), so the highest-priority open infra item (priority 70, plan ready since 2026-06-21) has sat unbuilt.
   - **Severity**: low
   - **File**: `docs/prompts/2026-06-21-desktop-followups-and-event-bus.md:4`
   - **Fix**: Update the prompt status to reflect partial completion, or close it and let ROAD-080 carry the remaining scope.

10. **Issue lifecycle vocabulary inconsistent, FEAT-005 abandoned in "implemented"** — Issue statuses use five different values (resolved/closed/done/implemented/open). FEAT-005 (config encryption/keychain) has sat at `implemented` — never closed — since ~May, and conflicts with project memory that keychain probing is now banned ("never trigger macOS keychain probing" — caused a reset-dialog flood 2026-06-20).
    - **Severity**: low
    - **File**: `docs/issues/FEAT-005.yaml` (status: implemented), `docs/issues/index.yaml`
    - **Fix**: Normalize statuses to open/closed; resolve FEAT-005 as closed-with-caveat noting the keychain reversal.

### Recommendations

- [ ] Run a roadmap grooming session: regenerate index.yaml from item files, archive ROAD-074/076/077/078 duplicates, resolve ROAD-035 title collision (effort: small)
- [ ] Promote ROAD-080 to in_progress — the plan (`docs/planning-mode/2026-06-21-inprocess-event-bus.md`, 644 lines) has been ready for 70 days (effort: small to start, medium to build)
- [ ] Audit the 9 BASE foundation items for ones already satisfied; close them (effort: small)
- [ ] Wire the ideas pipeline to the roadmap: harvested ideas that ship must create/update roadmap items (retroactively create one for IDEA-037 live metrics) (effort: medium)
- [ ] Update README.md feature table and CLAUDE.md roadmap range to match reality (effort: small)
- [ ] Delete or deprecate PLANNED_FEATURES.md, roadmap.md, GUI-INTEGRATION-STRATEGY.md (effort: small)
- [ ] Add roadmap grooming to `/triage` so a 70-day freeze cannot recur silently (effort: small)

### Roadmap Suggestions

- **Desktop v1.1 reliability & live data** — Track the shipped IDEA-037 metrics lineage plus the still-seed watchdog idea (IDEA-038: Rust daemon watchdog only covers initial spawn) as a coherent v1.1 hardening theme (priority: high, effort: medium)
- **Domain Center completion — Waves B–D** — Remaining Domain Management Center waves currently tracked only in project memory; link FEAT-006 and the 2026-06-14 brainplan (priority: medium, effort: large)
- **Roadmap hygiene — dedupe and regenerate** — One-time cleanup item for the 5 duplicate pairs, index regeneration, and BASE item audit (priority: high, effort: small)
- **Secrets hygiene — purge committed scan findings** — Security scan reports critical AWS-key patterns in `GOrchestra/sessions/*/recovery.patch` files; no roadmap or issue tracks the purge (priority: high, effort: medium)

### Roadmap Hydration Plan (5 items)

1. **[HIGH] Desktop v1.1 reliability & live data (metrics + daemon watchdog)** — linked to FEAT-008. Track the v1.1 hardening theme: IDEA-037 live metrics shipped 2026-07-01 (MetricsProducer, --metrics-interval, SSE bridge) but absent from the roadmap; IDEA-038 (seed, 70 days) covers the Rust daemon watchdog gap where CommandEvent::Terminated is only logged, not acted on. Create the item, mark the metrics half completed_in v0.17.0, and promote the watchdog half.
2. **[MED] Domain Center completion — Waves B–D** — linked to FEAT-006. Domain Management Center Waves B–D (registrar visibility, redirect governance, dashboard depth per the 2026-06-14 brainplan) are tracked only in project memory. FEAT-006 was closed after Waves 1–2. Create a roadmap item linking FEAT-006 and docs/brainstorming/2026-06-14-domain-management-center.md.
3. **[HIGH] Secrets hygiene — purge scan findings from committed GOrchestra artifacts** — ccs security flags critical AWS-access-key patterns in GOrchestra/sessions/agent-a1cdb2ac246339700/recovery.patch (lines 104, 157, 432+) and hardcoded IPs across .glm-agent-history.yaml. No roadmap or issue covers purging these multi-hundred-KB recovery patches from the repo. Also covers the 3.3MB cmd.test and 24K-line r2go2 binary tracked in-repo.
4. **[HIGH] Roadmap hygiene — dedupe double-imported items and regenerate index** — Archive duplicate stub items ROAD-074, ROAD-076, ROAD-077, ROAD-078 (created 15 seconds after their real counterparts ROAD-080..083 in the 2026-06-20 batch import), regenerate docs/roadmap/index.yaml to fix the ROAD-035 title mismatch and restore dropped priorities for ROAD-025..032, and audit the 9 unclosed BASE foundation items for ones already satisfied (BASE-002 tech stack, BASE-010 linting).
5. **[MED] Process-spawning hardening — close the last raw cmd.Start() (ROAD-525 class)** — spawn-check lint fails on cmd/installer_tui/main.go:1280 (`_ = cmd.Start() // Fire and forget`). The ROAD-525 daemon.SpawnAsync rollout has exactly one remaining violation, untracked in this project's roadmap. Small, bounded hardening item.
