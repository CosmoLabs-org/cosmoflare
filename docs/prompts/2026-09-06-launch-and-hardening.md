---
branch: master
created: "2026-09-06T23:10:19+04:00"
goals_completed: 9
goals_total: 11
priority: high
related_prompts: []
requires_reading:
    - docs/audit/latest/brief.md
    - docs/audit/latest/action-plan.md
schema_version: 1
status: PENDING
tags: []
title: Cosmoflare — v0.20.0 publish, launch gate, hardening
supersedes: "docs/prompts/2026-09-05-next-session.md"
---

# Cosmoflare — v0.20.0 Publish, Launch Gate, Hardening

## Context

The 2026-09-05 continuation executed 5/6 goals: FEAT-008 shipped as the alert→SSE bridge, BUG-035 verified (MCP full surface, 140+ tools), README matches reality, local GoReleaser pipeline proven, the 24-doc review backlog cleared (Reviewed 28 / Stale 0), and the 234MB attic is untracked with patches deleted. v0.20.0 is TAGGED and PUSHED — but the GitHub release has no binaries yet. The repo is one decision away from public: whether to purge the attic blobs from git history. Momentum is strong; the next unlock is distribution (public + tap), then the audit's hardening tier.

## GLM Dispatch Rules

1. ALWAYS use `ccs glm-agent exec` for GLM agents (queue + retry)
2. NEVER Agent tool with `model:sonnet`/`model:haiku` (hook-blocked; bypasses queue)
3. Agent tool with `model:opus` is fine
4. Parallel work: `/glm-sprint` or `ccs glm-agent exec-batch`

## What Got Done (2026-09-06 session)

- FEAT-008 SHIPPED: `webhook.Manager.SetNotifier` + `newServeAlertBridge` (cmd/serve.go) → sseHub notifications channel; SSE-level test; issue + ROAD-080 closed; plan 100% covered (docs/planning-mode/2026-09-06-feat008-alert-sse-bridge.md)
- 4-agent quality pass on the bridge: notifier now fires BEFORE webhook retries; `Channel{Metrics,Notifications,Status}` constants exported; tests green (cmd/, internal/server/, internal/webhook/)
- README regenerated; repo metadata set; `.goreleaser.yaml` verified end-to-end (5 binaries + checksums, version stamp correct)
- Attic untracked (555 files / 3.6M lines) + 84 recovery.patch deleted (233MB → 4.1MB); chat transcripts untouched
- G6: 22 docs reviewed via 4 GLM agents through the merge gate; 2 event-bus docs SUPERSEDED; doc-review Reviewed 28 / Stale 0
- ROAD-080/085/086 closed; BASE audit honest-no-closure; ROAD-084/087 orphan false-positives adjudicated
- v0.20.0 tagged + pushed (minor) — changelog finalized
- FBs filed to ClaudeCodeSetup: FB-pTBCHWG (recovery-patch lifecycle) + FB-pS6PF06 (salvage-restore requirement)

## Goals

### [x] 1. Publish the v0.20.0 GitHub release with binaries
**Model:** main-session operator (needs local goreleaser + gh auth; not delegable)
**Files:** none (artifacts to dist/)
The tag `v0.20.0` exists and is pushed; `gh release list` shows only v0.19.0. Run exactly (commands documented in `.goreleaser.yaml` header):
1. `goreleaser release --clean`
2. Assert `./dist/cosmoflare_darwin_arm64_v8.0/cosmoflare --version` prints `0.20.0` (NOT dev, NOT SNAPSHOT)
3. `gh release create v0.20.0 dist/checksums-sha256.txt $(jq -r '.[] | select(.type=="Binary" and (.name|startswith("cosmoflare-"))) | "\(.path)#\(.name)"' dist/artifacts.json) --notes-from-tag --latest`
**Acceptance:** `gh release view v0.20.0 --json assets --jq '.assets | length'` → 6.

### [ ] 2. History-purge decision → public flip → Homebrew tap
**Model:** main-session + user decision (irreversible; do NOT execute without explicit user confirmation)
**Files:** none in-repo
The attic blobs (234MB, incl. AWS example-key patterns inside recovery.patch files) remain in git history though untracked. Present the user the two options: (a) `git filter-repo --path GOrchestra/sessions --invert-paths` + force-push (rewrites SHAs — coordinate first), or (b) accept history as-is. After the user picks and (a) if chosen is done: `gh repo edit --visibility public --accept-visibility-change-consequences`, then create `CosmoLabs-org/homebrew-cosmoflare` tap and add the `brew` pipe config to `.goreleaser.yaml`.
**Acceptance:** `gh repo view --json visibility --jq .visibility` → `public` (only after user confirms).

### [x] 3. Guardrails enforcement (audit item 13 — the defensible-niche gap)
**Model:** `sonnet` | **Files:** `pkg/cosmoflare/guardrails.go`, `cmd/sync.go`, `cmd/object.go`
**Reason:** discovery — enforcement must hook the upload paths (`sync up`, `object put`, watcher) whose call shapes need tracing first.
`.cosmoflare.yaml` guardrails (blocked_keys, allowed_buckets) parse and test but never run — `sync up . bucket` uploads `.env` and `.git/` by default. Trace the upload call paths, wire guardrail checks at the library boundary, add default excludes, and TDD the enforcement (red test: uploading `.env` with guardrails configured must fail with an actionable error).
**Acceptance:** `go test ./pkg/cosmoflare/ ./cmd/ -run Guardrail` passes with new enforcement tests; audit risk-map row "Guardrails never enforced" is closed with a file:line citation.

### [ ] 4. Daemon error contract v2 (audit item 14)
**Model:** `sonnet` | **Files:** `internal/server/rest.go`, `pkg/cosmoflare/errors.go`
**Reason:** cross-file judgment — mapping typed R2Error/API errors to HTTP statuses touches the error hierarchy and all REST handlers.
Everything currently returns 502 (rest.go:94). Map typed errors → 400/401/404/429/502 with `{error, code}` JSON bodies so the desktop app can distinguish user-fixable from transient. Follow the typed-error hierarchy in `pkg/cosmoflare/errors.go`.
**Acceptance:** `go test ./internal/server/ -run ErrorContract` passes; a table-driven test covers each status mapping.

### [x] 5. Desktop `cf-*` stylesheet + a11y pass (audit item 12)
**Model:** `sonnet+worktree` | **Files:** `desktop/src/styles.css`, `desktop/src/App.tsx` (verify exact component paths in the worktree)
30+ `cf-*` classes referenced, 15-line CSS shipped; 3 a11y HIGHs (health state invisible to screen readers). Write the stylesheet, fix the 6 ARIA issues, verify in the Tauri dev shell.
**Acceptance:** `grep -c "cf-" desktop/src/styles.css` > 30; design-quality audit HIGHs closed.

### [x] G-06 First GitHub Release — DONE 2026-09-05 (v0.19.0, carried from G-01)
### [x] G-07 BUG-035 — DONE verified+closed 2026-09-06 (140+ MCP tools, tests re-run pass; carried from G-02)
### [x] G-08 Launch readiness — remainder carried into Goals 1+2 above (README/metadata/GoReleaser done 2026-09-06; from G-03)
### [x] G-09 FEAT-008 — DONE shipped 2026-09-06 as the sseHub bridge (bc18c4c + df2fc3c; from G-04)
### [x] G-10 Housekeeping — DONE 2026-09-06 (ROAD-085/086 closed, attic untracked+pruned; from G-05)
### [x] G-11 /independent-review backlog — DONE 2026-09-06 (Reviewed 28 / Stale 0; from G-06)
## Carry-Over Tasks

- [x] serve/mcp command docs — RESOLVED 2026-09-06: both are documented in `docs/USAGE.md` (serve at line 73+ with examples, mcp 5 references). The doc-audit README gaps are template false positives — this repo documents commands in the consolidated USAGE.md, not READMEs/commands/. Do NOT create a READMEs/ tree for this.

## Carry-Overs

1. **CCS-side watch** — FB-pTBCHWG + FB-pS6PF06 (recovery-patch lifecycle + salvage-restore) filed 2026-09-06: verify `ccs feedback inbox` assigned canonical numbers. Implementation happens in ClaudeCodeSetup sessions, NOT this repo.

## Where We're Headed

Distribution is the unlock: v0.20.0 binaries → history decision → public → tap. After launch, the audit's hardening tier (guardrails, error contract, desktop styling) converts the "agent trust" positioning from claim to reality — that is the only defensible niche against Cloudflare's official `cf` CLI + MCP (audit agent-4). The alert evaluator (feeding `TriggerAlert` via the `newServeAlertBridge` seam) is the natural feature follow-up once richer metric producers exist.

## Priority Order
1. Goal 1 (release publish — completes v0.20.0, 30 min)
2. Goal 2 (launch gate — user decision, everything downstream waits)
3. Goal 3 (guardrails — competitive-niche gap)
4. Goal 4 (error contract)
5. Goal 5 (desktop styling)
