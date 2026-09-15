---
title: "Session 1 Continuation Prompt"
created: 2026-09-15
status: PENDING
branch: master
goals_total: 4
goals_completed: 3
schema_version: 1
---

## Context

cmd/ renders all output through the Presenter — the 620-branch if-JSONOutput duplication audit finding is closed at 20 documented survivors (FEAT-040 complete). v0.28.0 is published with parallel sync (--concurrency), unified transport policy, deterministic JSON exit codes, corrected pkg.go.dev package doc, and (post-replacement) version-stamped binaries. make release TAG= now sanity-gates and stages releases end-to-end with the VERSION bug fixed at source. Idea backlog fully triaged: 4 promoted to tracked issues, only 2 deliberate seeds remain. Three worktree orphans reaped; roadmap ROAD-089 advanced to in_progress.

## Goals

### [x] 1. Launch program: ROAD-096 + TASK-010 product page + ROAD-089 Homebrew tap
Acceptance: vs-Wrangler comparison page and cosmolabs.org/cosmoflare product page exist with OG/JSON-LD; Show HN + r/Cloudflare drafts awaiting operator voice; tap formula installs cosmoflare
### [x] 2. TASK-009 funlen wave: split 45 god functions, adopt golangci funlen=80
Acceptance: golangci-lint run with funlen=80 exits 0; go test ./... green; zero functions over 80 lines
### [ ] 3. Verify pkg.go.dev re-indexed v0.28.0
Acceptance: pkg.go.dev/github.com/CosmoLabs-org/cosmoflare landing text reads 'Package cosmoflare provides' (not 'Package r2go2')
### [x] 4. Quality wave: FEAT-042 typed wire contract + cmd coverage 44.6%→60%
Acceptance: internal/server returns concrete structs; TS types generated drift-checked; go test ./cmd/ -cover ≥ 60%


## Carry-Over

v0.28.0 asset replacement completes this session (delete-7/upload-7 once make release finishes — approved). pkg.go.dev verification waits on indexing latency; check it first thing next session. FEAT-041 (desktop severity+light theme) and FEAT-042 (typed wire contract) are tracked but unscheduled — schedule FEAT-042 with the quality wave. Two idea seeds intentionally held: knowledge-transport (needs second pack), update-check (post-launch). Missing session summaries 2022-2025 and 26 unsynced memory files remain CCS-side housekeeping.

## Next Session Context

v0.28.0 is live with corrected assets. FEAT-040 closed. The next lever is distribution: ROAD-096 launch program (posts need the operator's voice — draft, don't publish), TASK-010 product page, ROAD-089 Homebrew tap (in_progress). TASK-009 (funlen splits) is ideal parallel GLM material — reuse .glm-prompts/species2/ spec pattern with exemplars and per-file isolation. Known blockers: memory writes blocked by cosmohooks cross-project-guard misfire (CCS feedback filed, FB-p26FGWZ stays open until fixed); pkg.go.dev re-index takes hours. Open CCS feedback from this session: 4 items (verify-worktree --all scope, commit --worktree, rebase-stale review SHA, memory-dir guard).

## File Scope

- 952eddc chore(ideas): triage 7 seeds — 4 promoted, 1 withered, 2 held
- 2abe46e chore(issues): create TASK-010
- 3cdf34c chore(issues): create TASK-009
- 20c366a chore(issues): create FEAT-042
- ff3630e chore(issues): create FEAT-041
- 39ff06b Merge branch '_glm-agent-0198-task-feat-016-codify'
- 3bfe223 chore(review): record session verdict for FEAT-016 merge
- 21c77af build(release): codify make release TAG= target — sanity gate, staging, operator-gated publish (FEAT-016)
- 0b8efa7 wip(salvage): auto-commit on died
- d8912f8 chore(housekeeping): housekeeping decisions — ROAD-089 in_progress, superseded prompt COMPLETED, FB triage
- 9fbfb67 chore(issues): close FEAT-040 — presenter sweep complete; prompt COMPLETED 2/2
- 46ed56f chore(release): v0.28.0
- 408dcf0 chore(session): gitignore .claude/ state, record CCS feedback copies, tick sweep prompt
- dfbfe03 Merge branch '_glm-agent-0197-compare'
- 69c6740 refactor(cli): species-2 presenter conversion for compare.go (FEAT-040)
- f093a36 Merge branch '_glm-agent-0196-decode'
- 09eedf8 refactor(cli): species-2 presenter conversion for decode.go (FEAT-040)
- 1e4f416 Merge branch '_glm-agent-0195-domains-redirects'
- 2ed362b refactor(cli): species-2 presenter conversion for domains_redirects.go (FEAT-040)
- 500066c Merge branch '_glm-agent-0194-domains-stats'
- 8f3caa4 refactor(cli): species-2 presenter conversion for domains_stats.go (FEAT-040)
- 336b35d Merge branch '_glm-agent-0193-init'
- def6739 refactor(cli): species-2 presenter conversion for init.go (FEAT-040)
- 26ad275 Merge branch '_glm-agent-0192-knowledge-cmd'
- cb7551f refactor(cli): presenter conversion for knowledge_cmd (FEAT-040)
- 5867412 Merge branch '_glm-agent-0191-limits'
- 041ad6e refactor(cli): species-2 presenter conversion for limits.go (FEAT-040)
- 8983482 refactor(cli): hand-convert 6 irregular species-2 stragglers (FEAT-040)
- 7220245 Merge branch '_glm-agent-0190-analytics'
- 2e94ef5 refactor(cli): species-2 presenter conversion for analytics (FEAT-040)
- cd88fa7 Merge branch '_glm-agent-0189-auth-permissions'
- bd1b024 refactor(cli): presenter conversion for auth_permissions.go (FEAT-040)

