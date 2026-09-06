---
branch: master
created: "2026-09-07T02:49:04+04:00"
goals_completed: 1
goals_total: 3
priority: high
related_prompts: []
requires_reading:
    - docs/sessions/sha-map-filter-repo-2026-09-07.txt
    - docs/changelog/unreleased.yaml
    - .goreleaser.yaml
schema_version: 1
status: PENDING
supersedes: "docs/prompts/2026-09-06-launch-and-hardening.md"
tags: []
title: Cosmoflare — launch gate, v0.21.0, CCS feedback watch
---

# Cosmoflare — Launch Gate, v0.21.0, CCS Feedback Watch

## Context

The 2026-09-06 launch-and-hardening prompt executed 10/11 goals. v0.20.0 is PUBLISHED with full binary distribution (5 binaries + checksums-sha256.txt). All hardening-tier audit items are closed: guardrails enforcement (item 13), daemon error contract v2 (item 14), desktop `cf-*` stylesheet + a11y pass (item 12). The only open goal was the launch gate, which was blocked on the history-purge decision — that decision is now DONE: 630 MB of dead blobs purged via `git filter-repo`, repo pack 485 MiB → 16.7 MiB, force-pushed. **All SHAs predating 2026-09-07 are dead** — the old↔new SHA map lives at `docs/sessions/sha-map-filter-repo-2026-09-07.txt`, and the pre-purge mirror is at `~/PROJECTS/cosmoflare-backups/` if anything needs recovering. The repo is one explicit user confirmation away from public, then the tap completes distribution.

## GLM Dispatch Rules

1. ALWAYS use `ccs glm-agent exec` for GLM agents (queue + retry)
2. NEVER Agent tool with `model:sonnet`/`model:haiku` (hook-blocked; bypasses queue)
3. Agent tool with `model:opus` is fine
4. Parallel work: `/glm-sprint` or `ccs glm-agent exec-batch`

## What Got Done (2026-09-06/07 sessions)

- v0.20.0 GitHub release published with full binary distribution (5 binaries + checksums — first release with assets)
- History purge executed: 630 MB of dead blobs (recovery patches, 12 old binaries) removed via `git filter-repo`; repo pack 485 MiB → 16.7 MiB; force-pushed; SHA map recorded at `docs/sessions/sha-map-filter-repo-2026-09-07.txt`; pre-purge mirror at `~/PROJECTS/cosmoflare-backups/`
- Hardening tier fully closed: guardrails enforcement wired into upload paths (audit item 13), daemon error contract v2 mapping typed R2 errors → 400/401/403/404/429/502 with `{error, code}` JSON bodies (audit item 14), desktop `cf-*` stylesheet + 6 ARIA fixes (audit item 12)
- 3 changelog entries staged in `docs/changelog/unreleased.yaml`: release distribution (added), error contract (changed), history purge (removed)
- Copyright notices swept to 2025-2026 range (chore(license) commit); the durable fix for the annual roll is FB-p97SDF1 in ClaudeCodeSetup
- Third FB filed to ClaudeCodeSetup: FB-p97SDF1 (license-year sweep automation), joining FB-pTBCHWG and FB-pS6PF06

## Goals

### [ ] 1. Launch gate — public flip + Homebrew tap (user decision)
**Model:** main-session operator (irreversible visibility change; do NOT execute without explicit user confirmation)
**Files:** `.goreleaser.yaml` (add `brew` pipe after tap repo exists)
The history purge is DONE — the blocker from the previous session's Goal 2 is cleared. Sequence:
1. Present the state to the user and get EXPLICIT confirmation to open-source (irreversible).
2. After confirmation: `gh repo edit CosmoLabs-org/cosmoflare --visibility public --accept-visibility-change-consequences`
3. Create the tap repo: `CosmoLabs-org/homebrew-cosmoflare`
4. Add the goreleaser `brew` pipe to `.goreleaser.yaml` (homebrew_cask/brews section — see goreleaser v2 docs for `repository.owner`/`repository.name`)
5. The NEXT release (v0.22.0+) publishes the formula automatically — v0.21.0 already shipped without the brew pipe (Goal 2 below, done).
**Acceptance:** `gh repo view CosmoLabs-org/cosmoflare --json visibility --jq .visibility` → `public` (only after user confirms); tap repo exists; `brew pipe` config present in `.goreleaser.yaml`.

### [x] 2. v0.21.0 minor release — DONE 2026-09-07 during session-end Phase 4: bump + tag + push landed; GitHub release published with 6 assets (5 binaries + checksums, hashes verified, marked Latest)
**Model:** main-session operator (needs local goreleaser + gh auth; not delegable)
**Files:** `docs/changelog/unreleased.yaml` → consumed by `ccs changelog finalize`
`release-check` suggests a minor bump; 3 changelog entries are staged (release distribution, error contract, history purge). Run the standard release flow (commands documented in the `.goreleaser.yaml` header):
1. `ccs changelog finalize 0.21.0 "<slug>"` (see `docs/changelog/unreleased.yaml` header for the exact invocation)
2. Commit the finalized changelog, then `git tag v0.21.0 && git push origin v0.21.0`
3. `goreleaser release --clean`
4. Assert `./dist/cosmoflare_darwin_arm64_v8.0/cosmoflare --version` prints `0.21.0` (NOT dev, NOT SNAPSHOT)
5. `gh release create v0.21.0 dist/checksums-sha256.txt $(jq -r '.[] | select(.type=="Binary" and (.name|startswith("cosmoflare-"))) | "\(.path)#\(.name)"' dist/artifacts.json) --repo CosmoLabs-org/cosmoflare --notes-file <release-notes.md> --latest`
   NOTE: `gh` requires the `--repo` flag when building from a clone (remote resolution changed after the purge/force-push), and `--notes-from-tag` is INCOMPATIBLE with `--repo` — use `--notes-file`.
**Acceptance:** `gh release view v0.21.0 --repo CosmoLabs-org/cosmoflare --json assets --jq '.assets | length'` → 6 (or 7 if the brew tap publish succeeded and adds nothing to assets — the formula lives in the tap repo, so still 6). If Goal 1's brew pipe is in place, also verify `CosmoLabs-org/homebrew-cosmoflare` contains `Formula/cosmoflare.rb` at v0.21.0.

### [ ] 3. CCS-side watch (do NOT implement here)
**Model:** main-session check only
**Files:** none in this repo
FB-pTBCHWG (recovery-patch lifecycle), FB-pS6PF06 (salvage-restore requirement), and FB-p97SDF1 (license-year sweep) are filed to ClaudeCodeSetup. Verify they receive canonical numbers via a ClaudeCodeSetup session (`ccs feedback ingest` runs during `ccs sync`). Implementation of those feedback items belongs to that repo, NOT cosmoflare.
**Acceptance:** canonical FB numbers confirmed for all three; roadmap/watch list updated to reference them.

## Carry-Over Notes

- **fb git hygiene:** after the history purge force-push, remote-tracking refs go stale — use `git ls-remote` for leases/comparisons instead of `git status` vs origin.
- **Copyright year** rolls to 2027 next January: the sweep pattern is documented in the chore(license) commit; the durable fix is FB-p97SDF1 (CCS-side).
- **Docs convention:** this repo documents commands in `docs/USAGE.md` ONLY. doc-audit "README gaps" are template false positives — do NOT create a READMEs/commands/ tree (adjudicated 2026-09-06).

## Where We're Headed

Distribution completes this cycle: user confirms → public → tap → v0.21.0 publishes the first Homebrew formula. With the hardening tier closed, the "agent trust" positioning (guardrails + error contract + audit trail) is now real, not claimed — the defensible niche against Cloudflare's official `cf` CLI + MCP (audit agent-4). The natural feature follow-up after launch is the alert evaluator feeding `TriggerAlert` via the `newServeAlertBridge` seam, once richer metric producers exist. Post-launch attention shifts to the audit's remaining transport-edge risks (storage engine silent-corruption paths: multipart short-read, unpaginated sync `--delete`, watcher delete-on-scan-error) — the next tier worth scheduling once distribution is stable.

## Priority Order
1. Goal 1 (launch gate — user confirmation is the only blocker; everything downstream waits)
2. Goal 2 (v0.21.0 — DONE, see tick)
3. Goal 3 (CCS watch — verification only, no code here)
