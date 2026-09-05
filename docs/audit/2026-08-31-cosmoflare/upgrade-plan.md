# Upgrade Plan — cosmoflare v0.17.0

> Phased recommendations from the 2026-08-31 audit (13 agents). Effort labels: small (< 1 day), medium (1-3 days), large (1+ week).

## Phase 0 — Critical Bugs (must-fix, blocks everything)

**Data integrity** (agent-2):
1. Fix multipart short-read acceptance in BOTH loops; abort with error when EOF before `uploadedBytes == size`. Add regression test: reader shorter than declared size. — `pkg/cosmoflare/upload.go:207`, `multipart.go:493` (small)
2. Paginate `ListRemoteObjects` on `NextToken` — removes the `--delete` data-destruction path. — `cmd/sync.go:428` (small)
3. `ValidatePartSize` in `MultipartUpload` (kills div-by-zero panic). — `upload.go:184` (small)
4. Watcher: never emit `ChangeDeleted` for paths under failed reads. — `watcher.go:124` (medium)

**Release pipeline unblock** (agents 5, 9):
5. Fix `internal/interactive` data races (package-level shared state; mutex or per-test instances) — makes CI green, unblocks Release. (medium)
6. Rename-drift sweep: `CosmoDev-R2Go2` → `cosmoflare` in ldflags (Makefile:19, Dockerfile:30-32, release.yml:62-64), release-note install command (release.yml:87), installer scripts (install.sh:16-20, install.ps1, install-menu.sh). Define the ldflags string ONCE. (small)
7. Fix `.dockerignore` `*.mod`/`*.sum` entries. (small)

**Agent contract** (agents 2, 7):
8. Emit JSON error envelope in `PersistentPreRun` before `os.Exit(1)`. — `cmd/root.go:99` (small)
9. Wire `GuardrailChecker.CheckUpload` into `client.Upload` + sync/watch paths; default-exclude `.git/`, `.env*`, `.DS_Store`. — `guardrails.go:29` (medium)

## Phase 1 — Foundation (testing, linting, CI, hygiene)

1. Ship v0.18.0: after Phase-0 fixes, tag with a release smoke gate (download assets, verify checksums + `--version` output). — first-ever GitHub Release (agents 5, 6) (medium)
2. Pin golangci-lint v2 (Go 1.26 support), commit `.golangci.yml`, add govulncheck + trivy security job + dependabot. (agents 5, 9) (medium)
3. Add tests: `internal/cli/operations` (0%), `internal/migration` (10.4%); extend httptest pattern to top-10 commands; target 60% for cmd/. (agent-1) (medium)
4. Repo hygiene: gitignore + untrack `GOrchestra/sessions/`, untrack `simple-setup`/`test-setup`/`build/` binaries; slim `make dist` to README/LICENSE/USAGE.md. (agents 1, 5, 9) (small)
5. Wire `httpClient`/timeout into cloudflare-go + AWS SDK; implement or delete the 4 dead options (`WithProfile`, `WithAuditLog`, `WithDryRun`, `WithBucket`). (agent-2) (medium)
6. Sync correctness: `os.Chtimes` on downloads; strip multipart `-N` ETag suffix in checksum compare. (agent-2) (small)
7. Delete/archive dead code: 4 disabled packages, 4 `.go.disabled` files, 6 empty dirs, 994-line `cicd.go.disabled`. (agents 1, 4) (small)

## Phase 2 — Quality (architecture, contracts, desktop)

1. **Positioning pivot** (strategic): write the `cf`/MCP competitive-response ADR; re-anchor on neutrality, agent safety (guardrails/audit), pure-Go embeddability, ops bundle. — vendor shipped `cf` CLI + 2,500-endpoint MCP server in April 2026 (agent-4) (small doc, large consequence)
2. Daemon error contract v2: map typed errors → 400/401/404/429/502 with `{error, code}` bodies; GET-only method guards; JSON content types everywhere; constant-time token compare + empty-token rejection; SSE heartbeat + last-frame replay. (agent-7) (medium)
3. Desktop design system: write the `cf-*` stylesheet (two-pane layout, card grid, ≥44px targets, `:focus-visible`, ≥3:1 indicators, semantic tokens). (agents 17, 18) (medium)
4. Desktop screen-reader pass: `role="log"`+`aria-live` on notifications, accessible health state, `aria-current="page"`, brand → `<h1>`, labeled badge; axe-core in Vitest. (agent-18) (small)
5. MCP parity program: auto-generate MCP tools from the cobra tree, mutations gated by guardrails. (agents 4, 7) (large)
6. Config unification: `.cosmoflare.yaml` first, `.r2go2.yaml` legacy fallback, `cosmoflare:` error prefix. (agents 1, 7) (medium)
7. Metrics reliability: partial snapshots with per-source errors; populate `Profile`; byte-based delta detection; decide FEAT-008 (bridge `TriggerAlert` → sseHub notifications — the standalone bus design likely collapses into a thin bridge now that the hub exists). (agents 2, 15) (medium)
8. Keychain opt-in: `--secure` flag on profile save storing `keychain://` refs (never auto-trigger — project memory). (agent-9) (medium)

## Phase 3 — Growth (distribution, content, tiers)

1. Make repo public; description, topics, homepage, social preview; verify `go install` from a clean machine. (agent-6) (small)
2. GoReleaser: archives, checksums, SBOM, Homebrew tap + Scoop manifest — package-manager distribution without Go toolchain. (agents 4, 5) (medium)
3. README regeneration from `cmd/` reality (13 services marked "Planned" are shipped); archive pre-rename root docs; CONTRIBUTING.md + templates; USAGE.md install section + TOC + env-var table. (agents 6, 10) (small-medium)
4. Library godoc: `pkg/cosmoflare/doc.go`, package comments, `example_test.go`, fix "R2Go2 operations" in `client.go:14`. (agents 6, 10) (medium)
5. Docs site (Astro 5): 53 USAGE sections → 53 indexable pages; comparison page targeting `wrangler alternative`. (agent-6) (large)
6. Desktop release pipeline: CI-built sidecars for all 3 targets (never committed), signing, updater feed, version sync with `.version-registry.json`. (agent-5) (large)
7. Declarative parity: extend apply/diff beyond workers/dns/kv/r2; publish coverage matrix. (agent-4) (large)
8. Domain Center Waves B–D (currently only in project memory — FEAT-006 closed prematurely). (agent-12) (large)

## Doc/Process track (runs alongside, from doc-integrity.md + work-completion.md)

- Add `plan_ref:` back-links to 7 orphaned brainstorms (lifts 25-pt score penalty) — small
- `ccs prompts stale --cleanup` for the abandoned event-bus pair; `ccs prompts scan` for the 71-day PENDING prompts — small
- Roadmap grooming: archive duplicate stubs ROAD-074/076/077/078, regenerate index.yaml (ROAD-035 title mismatch), audit 9 BASE items — small
- `/independent-review` on the 24 unreviewed design docs — large, batched
