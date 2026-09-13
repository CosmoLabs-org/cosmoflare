# Cosmoflare Brief (2026-09-13 audit)

Token-efficient context for future sessions. Load this instead of re-auditing. Full detail: sibling files.

## What this is

Go library + CLI (MIT) managing the full Cloudflare platform (R2, Workers, KV, DNS, SSL, D1, Pages, Queues, Images, Hyperize, Vectorize, AI, Stream, +20 more). 3-tier CosmoLabs product: free CLI → paid Tauri desktop → paid mobile. v0.26.0, Go 1.26, no CI (local releases by policy), 0 external users (repo private).

## Architecture (30 seconds)

- `cmd/` — 40+ cobra commands (thick: 618 JSONOutput branches, two error idioms)
- `pkg/cosmoflare/` — public library, one `*Service` per CF service; importable
- `internal/server` — serve daemon (REST+SSE, token auth) feeding `desktop/` (Tauri/React; untyped `any` wire contract)
- Transports: 3 (R2/S3 SDK, restClient, cloudflare-go via 27 FromCreds constructors)

## Scores that matter

Overall 59.8 Developing (+13.7). Strong: desktop design/a11y (75/80), architecture (70), tests-green. Weak: SEO 20 (private repo), sync-engine correctness (4 HIGH data-loss bugs), launch mechanics (4 unpublished releases).

## Known landmines (do not touch without tests)

- `pkg/cosmoflare/sync.go` — direction-blind deletes (:425), exclude-less delete loops (:275), no-op --include (:113)
- `pkg/cosmoflare/client.go:113` — shared 30s timeout kills large S3 transfers
- `pkg/cosmoflare/upload.go:181` — multipart abort with canceled ctx
- `cmd/auth.go:227-239` — revoke-old revokes NEW token
- `pkg/cosmoflare/apply.go:423` — delete-by-omission on whole account
- `Makefile:97` — dist tars docs/ (transcripts) into archives

## Strengths to preserve

Token-driven desktop design system with verified contrast table; vitest-axe gate; Retry-After-aware restClient; per-service library pattern; 2 TODOs in 165K lines.

## Sequenced next steps

1. Fix sync engine ×4 + auth rotate + Makefile:97 (action-plan.md 🔴)
2. Publish v0.26.0; purge transcripts/GOrchestra from tree; THEN flip repo public
3. Brand sweep (r2go2/CosmoDev remnants in types.go:2 pkg.go.dev text, docs/README, Makefile, registry)
4. Close FEAT-025/FEAT-019 (merged: 562a602, 4f512ef); rebuild empty docs/issues/index.yaml
5. Quality wave: presenter interface, cmd coverage 44.6%→60%, typed wire contract

## Entry points

`cmd/root.go` (CLI) · `cmd/serve.go` (daemon) · `desktop/src/App.tsx` (desktop) · `pkg/cosmoflare/dev.go:161` (dev stub — documented as working, returns 502)

## Gotchas

- Scanner "critical" AWS keys = AKIAIOSFODNN7EXAMPLE placeholders (false positives)
- ROAD-084 auto-fix "mark completed" is WRONG — watchdog half unfinished
- FEAT-014 premise outdated (limits snapshot wired at cmd/serve.go:238)
- `cosmoflare dev` is a stub; USAGE.md claims otherwise
- Desktop sidecar version 0.16.0 vs CLI 0.26.0
