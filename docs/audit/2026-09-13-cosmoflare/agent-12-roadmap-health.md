# Agent 12 — Roadmap Health Audit

Project: cosmoflare @ v0.26.0 | Audit date: 2026-09-13 | Auditor: agent-12 (roadmap-health)

## Roadmap Health: 5.4/10

**Sub-scores**:
| Dimension | Score | Notes |
|-----------|-------|-------|
| infrastructure | 7/10 | 109 structured YAML items (15 BASE + ROAD-000..093) + derived index.yaml (109 entries, `next_id: 94`, `last_updated` current). 76/82 completed items carry `completed_in` versions. Deductions: no `in_progress` status in vocabulary, duplicate items survive (ROAD-047/048, ROAD-073/079), index carries `has_plan` metadata absent from item files. |
| coverage | 6/10 | CLAUDE.md 3-tier vision and README in-progress list ARE tracked (ROAD-062/063/064/075/081/082/089). But 4 shipped strategic features have zero roadmap record (FEAT-005, FEAT-013, FEAT-028, FEAT-033) and the near-term queue (16 of 18 open FEATs) floats outside the roadmap. Legacy prose roadmap.md promises items with no structured counterpart. |
| linkage | 4/10 | Only 19/109 items (17%) have `linked_issues`. 16/18 open FEAT issues link to no roadmap item. ROAD-084's only link (FEAT-008) is closed; ROAD-085's only link (IDEA-046) is withered. Linkage is the weakest dimension. |
| staleness | 5/10 | Active September cadence (11 items updated 09-03..09-12, ideas triage round 09-12, ROAD-090 has dated progress notes) — but the captured tail is stagnant: 13/22 captured items idle 85–129 days with zero progression. Two items drifted hard against 70+ recent commits. |
| hydration_needed | 5/10 | 17 thin items (9 BASE + 8 ROAD) need links/brainstorms/plans; ROAD-084/085 need re-linking and re-scoping; 5 umbrella gaps deserve new items (see roadmap_hydration below). ~22 of 109 items need attention. |

### Critical Findings

1. **ROAD-084 is orphaned AND drifted — the suggested auto-fix would be wrong**
   - `docs/roadmap/items/ROAD-084.yaml:6` still says `status: captured` while its only linked issue FEAT-008 is `closed` (also completed via ROAD-080, which links the same FEAT-008 — a double-link). The health check suggests "update status to completed", but that would misrepresent reality: the item's second half (Rust daemon watchdog for mid-session crashes, IDEA-038) is genuinely unfinished — `desktop/src-tauri/src/daemon.rs` has spawn-time backoff (line 44) and child replacement (line 118) but grep finds no `Terminated`/mid-session-crash handling.
   - **Severity**: high
   - **File**: `docs/roadmap/items/ROAD-084.yaml:6,10`
   - **Fix**: Drop the stale FEAT-008 link, add `IDEA-038` (docs/ideas/2026-06-21-rust-daemon-watchdog-only-covers-initial-spawn-not-mid.md) as linked issue, add a progress note recording that the metrics half shipped and the watchdog half remains.

2. **ROAD-085 (secrets hygiene) is severely drifted — the purge already happened**
   - `ROAD-085.yaml:5` claims "234MB of tracked recovery patches (some 665K lines) bloat clones". Reality today: `git ls-files GOrchestra/` = 144 files, 7.3MB on disk. 70+ commits since 2026-08-31 match secrets/purge/artifact/untrack keywords — the described work substantially landed, but the item was never updated. Its only link, IDEA-046, has status `withered`. What REMAINS untracked by the stale description: the 2026-09-13 security scan still reports 13 critical AWS-access-key findings in docs/conversation-transcripts (4 files) and test fixtures (`internal/interactive/backup_restore_test.go:47,106,378`, `tests/fixtures/testdata.go:17,27`).
   - **Severity**: high
   - **File**: `docs/roadmap/items/ROAD-085.yaml:5,10`
   - **Fix**: Re-scope the description to the actual residue (transcript secrets, test-fixture keys), mark the GOrchestra purge half completed with `completed_in`, keep the item open only for the residue.

3. **docs/issues/index.yaml is empty and misnamed**
   - The file is 3 lines: header says "# R2Go2 Issues Index" (dead pre-rebrand name) and body is `issues: []` — while 81 issue files exist (37 FEAT, 39 BUG, 5 TASK; 18 FEAT open). Last touched 2026-02-26 (commit 836b045). Commit 51a062d "chore(tracking): issues index rebuild" did not regenerate it. Any consumer reading the derived index sees zero issues.
   - **Severity**: high
   - **File**: `docs/issues/index.yaml:1-3`
   - **Fix**: Rebuild the index with the issues tooling (`ccs issues` rebuild path — check `--help` first) and verify the header reflects "cosmoflare".

4. **Linkage desert: strategy execution floats outside the roadmap**
   - Only 19/109 roadmap items carry `linked_issues` (ROAD-000..023 generation, ROAD-063, 080, 084–087, 091, 092). 16 of 18 open FEAT issues — including the entire near-term queue (FEAT-011, 014–019, 025, 026, 029–032, 035–037) — have no roadmap home. Only FEAT-020/FEAT-021 point at ROAD-091. The roadmap describes direction; the issues describe execution; nothing connects them for 2026-09 work.
   - **Severity**: medium
   - **File**: `docs/roadmap/items/` (aggregate); e.g. `ROAD-089.yaml` (launch theme) vs unlinked FEAT-016 (release Makefile)
   - **Fix**: During the next `/triage` round, link each open FEAT to an existing item or spawn one; ROAD-089 should absorb FEAT-016; ROAD-091 should absorb FEAT-035/036/037 (same agentic-surface completeness theme).

5. **Stale prose roadmap trio contradicts the structured roadmap**
   - `docs/roadmap/roadmap.md` (269 lines) last committed 2025-11-24: describes v0.1.0 as "Current Development" (project is at v0.26.0), targets Q1–Q3 2025 dates long past, links the dead repo `github.com/CosmoLabs-org/CosmoDev-R2Go2` (line 268), and shows unchecked boxes for work that SHIPPED (multi-account, audit logging, terraform, REST API server, plugin architecture, migrate from-s3). `VISION-AND-ARCHITECTURE.md` and `GUI-INTEGRATION-STRATEGY.md` (721 lines) are same-vintage R2Go2 docs with elapsed "Next 3 Months" phases. `PLANNED_FEATURES.md` (2026-02-26) still opens "# R2Go2 Planned Feature Roadmap". A newcomer reading docs/roadmap/ top-level files gets a 10-month-old picture.
   - **Severity**: medium
   - **File**: `docs/roadmap/roadmap.md:16,268,270`; `docs/roadmap/PLANNED_FEATURES.md:1`; `docs/roadmap/VISION-AND-ARCHITECTURE.md:1`; `docs/roadmap/GUI-INTEGRATION-STRATEGY.md:1`
   - **Fix**: Replace the three 2025-11-24 files with short pointers to index.yaml + PRODUCT-VISION.md; either archive PLANNED_FEATURES.md or checklist its shipped items and port the genuinely-unplanned remainder (CI/CD template toolkit — no `cicd` command exists in cmd/) into structured items or an explicit wont-do note.

6. **Duplicate roadmap items inflate the completed count**
   - ROAD-047 and ROAD-048 are the same "Vectorize — vector storage and indexing for AI workloads" item (identical titles; descriptions differ by one sentence) — both `completed`. ROAD-073 and ROAD-079 are both "Local API daemon (cosmoflare serve)" — both `completed`. ROAD-076/081, 077/082, 078/083 are archived/captured re-import pairs of the same titles. ROAD-086 ("dedupe double-imported items") is completed but did not finish the job.
   - **Severity**: low
   - **File**: `docs/roadmap/items/ROAD-048.yaml:4`; `docs/roadmap/items/ROAD-079.yaml:4`
   - **Fix**: Merge 048→047 and 079→073 (or annotate "duplicate of"); add `superseded_by` refs to 076/077/078.

7. **Captured-tail stagnation with no in-progress visibility**
   - 13 of 22 captured items have sat untouched 85–129 days: 9 BASE items (112 days, all created 2026-05-24), ROAD-029 (129 days — "CCS r2 subcommand integration (deferred)"), ROAD-064 (120 days — the React Native mobile tier, priority 80!), ROAD-075/081/082/083 (85 days). The status vocabulary has no `in_progress`, so a captured item is indistinguishable from an active one until an audit runs. Note ROAD-064 is the paid mobile tier from the product vision — the most strategic unstarted item is also one of the most neglected.
   - **Severity**: medium
   - **File**: `docs/roadmap/items/ROAD-029.yaml:6`; `docs/roadmap/items/ROAD-064.yaml:6`
   - **Fix**: Monthly grooming pass over `status: captured` items: link, split, or archive. For BASE items (tooling checklists for a CLI-side project — e.g. BASE-012 "Database initialized" barely applies), mark wont-do/archived rather than leaving them perpetually captured.

8. **Shipped strategic work with no roadmap record**
   - Reverse-linkage sweep found closed strategic features referenced by no roadmap item in either direction: FEAT-005 (config encryption/keychain — the no-keychain memory note makes this strategically relevant), FEAT-013 (WAF rate-limit traffic-class encoding), FEAT-028 (R2 bucket policy get/set), FEAT-033 (domain fleet status matrix, shipped 2026-09-12). The roadmap's completed set understates delivered scope; coverage percentage of shipped work is therefore unverifiable from the index alone.
   - **Severity**: low
   - **File**: `docs/issues/FEAT-033.yaml` (status: done, no ROAD ref anywhere in docs/roadmap/items/)
   - **Fix**: Add a `completed_in`-style backfill: create a "2026-09 shipped residuals" completed item linking FEAT-013/028/033, and link FEAT-005 under the security posture area.

### Recommendations

- [ ] Rebuild `docs/issues/index.yaml` (empty since 2026-02-26, misnamed "R2Go2") via the ccs issues tooling; verify 81 entries appear (effort: small)
- [ ] Fix ROAD-084: remove closed FEAT-008 link, add IDEA-038, add progress note — do NOT auto-complete (effort: small)
- [ ] Re-scope ROAD-085 to the remaining secrets residue (transcripts + test fixtures); mark GOrchestra purge completed (effort: small)
- [ ] Link the 16 unlinked open FEAT issues to existing/new items in the next /triage round; ROAD-089←FEAT-016, ROAD-091←FEAT-035/036/037 (effort: medium)
- [ ] Replace roadmap.md / VISION-AND-ARCHITECTURE.md / GUI-INTEGRATION-STRATEGY.md (all 2025-11-24) with pointers to the structured index (effort: small)
- [ ] Dedupe ROAD-048→047, ROAD-079→073; annotate 076/077/078 as superseded (effort: small)
- [ ] Adopt a monthly grooming cadence for `status: captured` items >60 days; decide wont-do for inapplicable BASE items (effort: small)
- [ ] Backfill roadmap records for shipped-but-untracked FEAT-005/013/028/033 (effort: small)

### Roadmap Suggestions

- **Roadmap hygiene sweep #2** — link-or-close the open FEAT queue, dedupe 047/048 + 073/079, refresh ROAD-084/085, rebuild issues index (priority: high, effort: medium)
- **Agent-first onboarding tier** — first-run setup wizard (FEAT-017) + permission manifest/token doctor (FEAT-019) as one tracked theme; this is the CLI's stated differentiator and is currently untracked (priority: high, effort: large)
- **API client resilience & rate-limit awareness** — FEAT-025 shared client (Retry-After/backoff); partially landed via commit 74718b2 — needs a roadmap home to track the remainder (priority: high, effort: medium)
- **Production-evidence reliability loop** — FEAT-018 (MyCarGuide friction), FEAT-011 (token permission drift), FEAT-014 (snapshot cadence) grouped as one reliability theme (priority: medium, effort: medium)
- **Cloudflare surface completion wave 2** — FEAT-026 (env profiles), FEAT-029 (auth modernization), FEAT-030/031/032 (registrar/SSL-SaaS/WAF lists), FEAT-035/036/037 (tunnels/account/long-tail zones) — extend ROAD-091/093 or create an umbrella (priority: medium, effort: large)
- **CI/CD template toolkit** — the only still-relevant unshipped promise from PLANNED_FEATURES.md (`cicd template` for GH Actions/GitLab); no cmd/ exists — capture explicitly or record wont-do (priority: low, effort: medium)

---

## Structured Result

```json:audit-result
{
  "agent": "roadmap-health",
  "overall_score": 5.4,
  "sub_scores": {
    "infrastructure": 7,
    "coverage": 6,
    "linkage": 4,
    "staleness": 5,
    "hydration_needed": 5
  },
  "critical_findings": [
    {
      "title": "ROAD-084 orphaned and drifted; auto-fix 'mark completed' would be wrong",
      "severity": "high",
      "file": "docs/roadmap/items/ROAD-084.yaml:6",
      "fix": "Replace closed FEAT-008 link with IDEA-038 (daemon watchdog, unfinished); add progress note; keep captured",
      "effort": "small"
    },
    {
      "title": "ROAD-085 secrets-hygiene drifted: purge already landed (GOrchestra 234MB->7.3MB tracked, 70+ commits), description stale, linked IDEA-046 withered",
      "severity": "high",
      "file": "docs/roadmap/items/ROAD-085.yaml:5",
      "fix": "Mark purge half completed_in; re-scope to remaining residue (transcript AWS keys, test-fixture keys in internal/interactive/backup_restore_test.go and tests/fixtures/testdata.go)",
      "effort": "small"
    },
    {
      "title": "docs/issues/index.yaml is empty (issues: []) and titled 'R2Go2 Issues Index' despite 81 issue files",
      "severity": "high",
      "file": "docs/issues/index.yaml:3",
      "fix": "Rebuild index via ccs issues tooling; verify entry count and project name",
      "effort": "small"
    },
    {
      "title": "Linkage desert: 19/109 items carry linked_issues; 16/18 open FEAT issues have no roadmap home",
      "severity": "medium",
      "file": "docs/roadmap/items/ (aggregate)",
      "fix": "Link open FEATs during next triage; ROAD-089<-FEAT-016, ROAD-091<-FEAT-035/036/037",
      "effort": "medium"
    },
    {
      "title": "Stale prose roadmap trio (roadmap.md, VISION-AND-ARCHITECTURE.md, GUI-INTEGRATION-STRATEGY.md; all 2025-11-24) contradicts structured roadmap with dead repo links and unchecked shipped features",
      "severity": "medium",
      "file": "docs/roadmap/roadmap.md:16",
      "fix": "Replace with pointers to index.yaml + PRODUCT-VISION.md; port or wont-do the CI/CD template promise",
      "effort": "small"
    },
    {
      "title": "13/22 captured items idle 85-129 days including ROAD-064 (mobile paid tier, priority 80); no in_progress status exists",
      "severity": "medium",
      "file": "docs/roadmap/items/ROAD-029.yaml:6",
      "fix": "Monthly grooming pass over captured>60d items; archive inapplicable BASE items",
      "effort": "small"
    },
    {
      "title": "Duplicate completed items: ROAD-047/048 (Vectorize) and ROAD-073/079 (serve daemon); archived/captured re-import pairs 076/081, 077/082, 078/083",
      "severity": "low",
      "file": "docs/roadmap/items/ROAD-048.yaml:4",
      "fix": "Merge duplicates with cross-references; add superseded_by to archived pairs",
      "effort": "small"
    },
    {
      "title": "Shipped strategic work untracked: FEAT-005 (keychain), FEAT-013 (traffic classes), FEAT-028 (bucket policy), FEAT-033 (domain fleet status) closed with no roadmap record",
      "severity": "low",
      "file": "docs/issues/FEAT-033.yaml:1",
      "fix": "Backfill a completed umbrella item linking the four; link FEAT-005 under security posture",
      "effort": "small"
    }
  ],
  "recommendations": [
    {"action": "Rebuild docs/issues/index.yaml via ccs issues tooling (empty since 2026-02-26)", "effort": "small", "priority": "high"},
    {"action": "Fix ROAD-084 links (drop closed FEAT-008, add IDEA-038) and ROAD-085 scope (purge landed; residue remains)", "effort": "small", "priority": "high"},
    {"action": "Link 16 unlinked open FEAT issues to roadmap items in next /triage round", "effort": "medium", "priority": "high"},
    {"action": "Replace three 2025-11-24 prose roadmap docs with pointers to structured index", "effort": "small", "priority": "medium"},
    {"action": "Dedupe ROAD-048/047 and ROAD-079/073; annotate superseded archived pairs", "effort": "small", "priority": "low"},
    {"action": "Monthly grooming cadence for captured items older than 60 days", "effort": "small", "priority": "medium"},
    {"action": "Backfill roadmap records for shipped FEAT-005/013/028/033", "effort": "small", "priority": "low"}
  ],
  "roadmap_suggestions": [
    {"title": "Roadmap hygiene sweep #2", "description": "Link-or-close open FEAT queue, dedupe 047/048 and 073/079, refresh ROAD-084/085, rebuild issues index", "priority": "high", "effort": "medium"},
    {"title": "Agent-first onboarding tier", "description": "First-run setup wizard (FEAT-017) + permission manifest/token doctor (FEAT-019) — the CLI differentiator, currently untracked", "priority": "high", "effort": "large"},
    {"title": "API client resilience & rate-limit awareness", "description": "FEAT-025 shared client (Retry-After, backoff); commit 74718b2 landed part — track remainder", "priority": "high", "effort": "medium"},
    {"title": "Production-evidence reliability loop", "description": "Group FEAT-018 (MyCarGuide friction), FEAT-011 (token permission drift), FEAT-014 (snapshot cadence)", "priority": "medium", "effort": "medium"},
    {"title": "Cloudflare surface completion wave 2", "description": "FEAT-026/029/030/031/032/035/036/037 umbrella or ROAD-091/093 extension", "priority": "medium", "effort": "large"},
    {"title": "CI/CD template toolkit", "description": "Only relevant unshipped promise from PLANNED_FEATURES.md; no cmd/ exists — capture or record wont-do", "priority": "low", "effort": "medium"}
  ],
  "roadmap_hydration": [
    {"title": "Roadmap hygiene sweep #2 — linkage, dedupe, drift fixes", "description": "Rebuild empty docs/issues/index.yaml; link 16 open FEATs (ROAD-089<-FEAT-016, ROAD-091<-FEAT-035/036/037); fix ROAD-084 (relink IDEA-038, watchdog half unfinished) and ROAD-085 (purge landed, re-scope to transcript/fixture secret residue); dedupe ROAD-047/048 and ROAD-073/079; backfill shipped FEAT-005/013/028/033", "linked_issues": ["FEAT-016", "FEAT-035", "FEAT-036", "FEAT-037", "FEAT-033"], "priority": "high"},
    {"title": "Agent-first onboarding — first-run wizard + least-privilege token doctor", "description": "One token, whole platform: setup wizard (FEAT-017) plus permission manifest and token doctor (FEAT-019). The agent-first UX is cosmoflare's stated differentiator per CLAUDE.md yet has no roadmap item", "linked_issues": ["FEAT-017", "FEAT-019"], "priority": "high"},
    {"title": "Rate-limit-aware shared client", "description": "Retry-After parsing, exponential backoff, retry flags across all services (FEAT-025). Partially landed via commit 74718b2 (REST client retries) — needs a tracked home for the remaining surface and tests", "linked_issues": ["FEAT-025"], "priority": "high"},
    {"title": "Production-evidence reliability loop", "description": "Convert MyCarGuide production friction into tracked reliability work: D1 batch-push, usage-vs-limits, deploy-verify, parity, WAF ratelimit (FEAT-018), token permission-name drift (FEAT-011), serve limits-snapshot cadence (FEAT-014)", "linked_issues": ["FEAT-011", "FEAT-014", "FEAT-018"], "priority": "medium"},
    {"title": "Cloudflare surface completion wave 2", "description": "Umbrella for the open depth/coverage FEATs: named env profiles (FEAT-026), auth modernization/device-flow (FEAT-029), registrar ops (FEAT-030), SSL SaaS hostnames (FEAT-031), WAF lists (FEAT-032), Tunnels (FEAT-035), account mgmt + audit logs (FEAT-036), zone long-tail batch (FEAT-037). Alternatively extend ROAD-091/ROAD-093 instead of creating new", "linked_issues": ["FEAT-026", "FEAT-029", "FEAT-030", "FEAT-031", "FEAT-032", "FEAT-035", "FEAT-036", "FEAT-037"], "priority": "medium"},
    {"title": "CI/CD integration templates — decide or drop", "description": "PLANNED_FEATURES.md promises `cicd template` workflows (GitHub Actions, GitLab CI); no cmd/cicd exists and no structured item tracks it. Either capture as a real item or record an explicit wont-do so the prose promise stops dangling", "linked_issues": [], "priority": "low"}
  ]
}
```

## Evidence Appendix (key data points)

- Item inventory: 109 items = 15 BASE + ROAD-000..093; status distribution: 82 completed / 22 captured / 5 archived; no other statuses exist.
- Linkage: 19 items with `linked_issues` (ROAD-000,001,002,007,008,009,010,014,015,019,022,023,063,080,084,085,086,087,091,092); 76 items carry `completed_in`; 80 carry `priority`; 11 reference brainstorms.
- Issues: 81 files — 37 FEAT (18 open), 39 BUG (0 open), 5 TASK (0 open); statuses closed 31 / open 18 / done 17 / resolved 13 / implemented 2. Issues index: empty.
- Captured-tail ages (days idle): ROAD-029 129, ROAD-064 120, 9x BASE 112, ROAD-075/081/082/083 85, ROAD-084/085/088 13, ROAD-089 6, ROAD-093 2, ROAD-090 4, ROAD-091 1.
- Drift evidence: `git log --since=2026-08-31 | grep -icE 'secret|purge|gorchestra|artifact|untrack|bloat'` = 70; GOrchestra tracked = 144 files / 7.3MB (ROAD-085 claims 234MB).
- Prose doc staleness (git last-commit): roadmap.md / VISION-AND-ARCHITECTURE.md / GUI-INTEGRATION-STRATEGY.md = 2025-11-24 (368c05c); PLANNED_FEATURES.md = 2026-02-26 (836b045); README.md = 2026-09-06 (current).
- Positive exemplar: ROAD-090 (created 2026-09-08) carries a dated progress note, plan ref (docs/planning-mode/2026-09-09-quota-limit-view.md), and PRODUCT-VISION ref — the hydration standard the older tail should match.
- Files read directly: 14 (briefing, index.yaml, ROAD-029/047/048/064/073/079/084/085/090/093 via targeted reads/diffs, roadmap.md, PLANNED_FEATURES.md, README.md, VISION-AND-ARCHITECTURE.md, issues index, FEAT-006/008/012/022/023/033/034 + 7 orphan-issue YAMLs); aggregate analysis via grep across all 109 roadmap items and 81 issue files.
