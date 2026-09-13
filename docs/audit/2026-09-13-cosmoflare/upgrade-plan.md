# Upgrade Plan — cosmoflare v0.26.0

Phased action plan synthesized from 13 audit agents. Effort estimates from source agents.

## Phase 0 — Critical bugs (must-fix before public launch)

| # | Fix | File:Line | Effort | Source |
|---|-----|-----------|--------|--------|
| 0.1 | Direction-aware deletes in `SyncService.Execute` (down-sync deletes local, never remote) + regression test asserting down-sync emits zero remote deletes | pkg/cosmoflare/sync.go:425 | small | agent-2 |
| 0.2 | Exclude-protect delete loops in planUp/planDown (.env/.git must never be delete-eligible) | pkg/cosmoflare/sync.go:275 | small | agent-2 |
| 0.3 | Split control-plane vs data-plane HTTP clients: 30s API client stays, S3 gets timeout-free client | pkg/cosmoflare/client.go:113 | small | agent-2 |
| 0.4 | Cancellation-safe multipart abort (`context.WithoutCancel` + timeout) + log abort failures | pkg/cosmoflare/upload.go:181, multipart.go:467 | small | agent-2 |
| 0.5 | Implement or remove `auth rotate`; fix `--revoke-old` capturing old token BEFORE overwrite | cmd/auth.go:227-239, 479-491 | medium | agent-9 |
| 0.6 | Remove `../docs/` from `make dist` archives (transcript secrets in public artifacts) | Makefile:97 | small | agent-5 |
| 0.7 | Atomic temp+rename downloads with Close() error checks | pkg/cosmoflare/download.go:94, cmd/sync.go:534 | small | agent-2 |
| 0.8 | Fix Dockerfile toolchain: golang:1.26-alpine, pinned alpine runtime, -o cosmoflare | Dockerfile:5,36 | small | agent-9 |

## Phase 1 — Launch gating (distribution + hygiene)

| # | Fix | File:Line | Effort | Source |
|---|-----|-----------|--------|--------|
| 1.1 | Purge internal artifacts from tracked tree: transcripts, session JSONL, GOrchestra logs; gitignore; history purge or private mirror | docs/conversation-transcripts/, docs/sessions/transcripts/, GOrchestra/ | medium | agent-10, agent-1, agent-9 |
| 1.2 | Publish v0.26.0 (dist/upload/ is staged); backfill v0.23–v0.25 or document; add post-release `gh release view` guard | .goreleaser.yaml:22 | small | agent-5 |
| 1.3 | Flip repo public (AFTER 1.1); verify unauthenticated `go install`; trigger pkg.go.dev | README.md:10 | small | agent-6 |
| 1.4 | Fix install paths: GETTING_STARTED.md → CosmoLabs-org/cosmoflare; drop install.sh armv7 or build it | GETTING_STARTED.md:15, install.sh:95 | small | agent-5 |
| 1.5 | Brand unification sweep: types.go package doc, docs/README.md title, .version-registry.json description, Makefile binary names, SECURITY.md URLs | pkg/cosmoflare/types.go:2 + 5 sites | small | agent-6, agent-7, agent-5, agent-4 |
| 1.6 | Delete dead CI workflows (policy: local releases); add govulncheck to release-prepare | .github/workflows/ | small | agent-9, agent-5 |
| 1.7 | `cosmoflare dev`: implement proxy or deprecate + correct USAGE.md | pkg/cosmoflare/dev.go:161 | medium | agent-7 |

## Phase 2 — Quality (architecture, patterns)

| # | Fix | Effort | Source |
|---|-----|--------|--------|
| 2.1 | Output presenter interface collapsing the 618 JSONOutput branches (do per-command-group, not big-bang) | large | agent-1 |
| 2.2 | cmd/ coverage 44.6% → 60%: output-mode tests for 10 highest-traffic commands | medium | agent-1 |
| 2.3 | Migrate legacy Run+printErrorAndExit commands to RunE | medium | agent-1 |
| 2.4 | Typed daemon wire contract (concrete Go structs) + TS type codegen for desktop/src/api | medium | agent-7 |
| 2.5 | Thread timeout-bearing http.Client through the 27 FromCreds constructors | small | agent-7 |
| 2.6 | Split 3 worst god functions; adopt funlen=80 lint | medium | agent-1 |
| 2.7 | Guardrails v2: extend enforcement to delete/copy paths | medium | agent-2 |
| 2.8 | Security polish: constant-time token compare, writeJSONStatus everywhere, silent token prompt, Tauri CSP, 0600-first config writes, url.PathEscape CopySource | small (each) | agent-9, agent-7, agent-2 |
| 2.9 | Implement `--include` in sync planner (or remove flag) | medium | agent-2 |
| 2.10 | Managed-resource ownership model before apply deletes unmanaged resources | large | agent-2 |

## Phase 3 — Growth (features, market)

| # | Item | Effort | Source |
|---|------|--------|--------|
| 3.1 | Launch program: Show HN, r/Cloudflare, Cloudflare Community, awesome-list PRs, vs-Wrangler comparison page | small | agent-4, agent-6 |
| 3.2 | Homebrew tap + Scoop/winget manifests driven by release assets | medium | agent-5 |
| 3.3 | Remote MCP transport (streamable HTTP, token-authed) + tools/list pagination — neutralizes official cloudflare/mcp | medium | agent-4 |
| 3.4 | Workers lifecycle parity: secrets, versions, rollback | medium | agent-4 |
| 3.5 | cosmolabs.org/cosmoflare product page with OG/JSON-LD | medium | agent-6 |
| 3.6 | cosign-signed checksums + SBOM per release | medium | agent-9, agent-5 |
| 3.7 | Opt-in env-gated update check | small | agent-4 |
| 3.8 | Desktop: light theme, notification severity system, SSE announcement bus (WCAG 4.1.3), linux/arm64 sidecar | small-medium each | agent-17, agent-18, agent-5 |
| 3.9 | Parallel sync executor (bounded workers, 4-8x large syncs) | medium | agent-2 |

## Sequencing logic

Phase 0 before Phase 1.3 (public flip) — the sync bugs and the dist-archive leak are the only findings that turn a launch into an incident. Phase 1 is small efforts with outsized strategic value (visibility + publish + brand). Phase 2 can overlap Phase 3's early items; 2.1/2.2 lower the cost of everything after.
