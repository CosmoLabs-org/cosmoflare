# Cloudflare `cf` CLI vs cosmoflare — gap analysis (ROAD-104)

Date: 2026-10-10. Method: live research (announcement + npm + GitHub verified
2026-10-10), out-of-tree clone analysis (`ccs analyze-clone`, FEAT-686) with
parallel read-only scouts, every claim re-verified by the orchestrator against
the clone at `~/.analysis-clones/cloudflare-cf` (clone: **`/Users/gabstudio/.analysis-clones/cloudflare-cf`**).

## What `cf` is (verified live)

- **Tool**: `cf` — "the agentic CLI for the entire Cloudflare API".
- **Repo**: github.com/cloudflare/cf (verified HTTP 200). **npm**: `cf@1.0.0-beta.14`,
  description "The Cloudflare CLI" (verified via `npm view cf`). TypeScript, pnpm
  monorepo (`packages/cli`), MIT/Apache dual license.
- **Announced**: 2026-09-28, Birthday Week 2026 — "Introducing cf: the agentic
  CLI for the entire Cloudflare API" (blog.cloudflare.com/cloudflare-cf-cli-launch/),
  docs at developers.cloudflare.com/cf/.
- **Positioning**: eventually replaces Wrangler (Wrangler's final major version
  will point users to cf; 18 months maintenance after beta). cf still delegates
  bundling (esbuild/Rust/Python Workers) to Wrangler internally.
- **Scale**: 165 generated top-level command roots, ~2,947 generated command
  leaves (`packages/cli/src/commands/_generated/_meta/commands.json`) + 52
  hand-written commands (`_meta/hand-written-commands.json`: auth, build, cli,
  complete, deploy, dev, init, migrate, schema, tools). Wrangler has ~280.
- **Context**: Cloudflare reports agent share of Wrangler usage grew 25%
  (2026-03) → 48% (2026-10).

## Command surface + output conventions (scout 1, verified)

| Aspect | cf | Source (clone) |
|---|---|---|
| Output | JSON only — no table renderer anywhere; `formatOutput` unwraps SDK envelopes, 2-space pretty JSON, TTY-only syntax highlight, null → silent stdout + `✓ label` on stderr | `packages/cli/src/lib/output.ts` |
| Errors | Human-readable boxed `APIError` with `[code]` + status + path; **no machine-readable error JSON mode** | `packages/cli/src/lib/errors.ts` |
| Exit codes | `CliExit` class thrown, mapped in `bin/cf`; other throws → exit 1 | `packages/cli/src/lib/cli-exit.ts` |
| Discovery | `cf cli search <query>` — MiniSearch fuzzy over the generated command index, top-5 compact JSON | `packages/cli/src/commands/cli/search.ts` |
| Agent help | When an agent harness is detected (12 harnesses via env: CLAUDECODE etc.), `AGENT_DISCOVERY_HELP` is prepended to `--help` | `packages/cli/src/lib/agent-context.ts`, `src/index.ts:143-158` |
| Schema help | `cf schema <cmd>` prints the API schema behind a command; 1h schema cache | `packages/cli/src/commands/api-schema-help.ts`, `src/lib/schema-cache.ts` |
| Reality check | README's "condensed for agents" output mode is **aspirational — not implemented**; actual mechanism is pretty JSON + `--quiet` + null suppression | `README.md:6` vs `src/lib/output.ts` |

## Auth + config (scout 2, verified)

- Token precedence: `CLOUDFLARE_API_TOKEN` env (global API key disabled) →
  stored OAuth token with refresh (`packages/cli/src/lib/auth-token.ts`).
- Account precedence: `CLOUDFLARE_ACCOUNT_ID` env → `cloudflare.config.ts`
  `accountId` → workers-auth account cache → picker (`src/lib/context.ts`).
- Profiles at `~/.config/cloudflare/config/<name>.json` (oauth/refresh/expiry/
  scopes); compliance region (`public`/`fedramp-high`) supported.
- `cloudflare.config.ts`: `defineConfig` from `cf/config`, discovered walking up
  from cwd (`src/lib/project-settings.ts`); types generated to
  `.cloudflare/types/index.d.ts` (`cf workers types`) — LSP-checked config.

## Agent integration (scout 3 area, orchestrator-verified)

- **No MCP server mode** in cf (grep for "mcp" hits only the unrelated
  `ZonesWebmcp*` SDK types — Cloudflare's WebMCP product config).
- `AGENTS.md` at the repo root defines how coding agents contribute to cf
  itself; `agent-context.ts` is how cf detects it runs INSIDE an agent.
- Declarative identity: JSON output + `cli search` + schema help = the entire
  agent story. No prompt-caching tricks, no structured tool protocol.

## How cf pulls account data (scout 4, verified) — THE key comparison

- Transport: **REST only** (`api.cloudflare.com/client/v4`) through a committed
  Fern-generated SDK (`packages/cli/src/sdk/`). **Zero** `graphql.cloudflare.com`
  references in the entire CLI source (verified by grep).
- Generation: the "Forge" pipeline (`scripts/sync-forge.ts`, vendored
  `@cloudflare/forge` tarballs in `vendor/`) turns Cloudflare OpenAPI specs into
  SDK + command code — ~4,130 generated files.
- Retries: SDK retry is **disabled** (`maxRetries: 0`) by cf's fetcher wiring.
- Pagination: **no auto-pagination** in command handlers; `cursor`/`per_page`
  are exposed as user-facing flags (single-page commands). SDK pagination
  primitives exist (`sdk/core/pagination/Page.ts`).
- Caching: only the help-time schema cache (1h TTL). No response caching.
- Analytics: the `analytics/query/{summary,timeseries,top-n}` overlay covers
  **Zero Trust datasets only** (access-logins, gateway-*). **No Workers request
  analytics, no D1 analytics** datasets anywhere (verified: zero hits for
  `workersInvocationsAdaptive`/`d1AnalyticsAdaptive`).
- Billing: `billing/usage/get-account-usage-v2` +
  `get-account-billable-metrics` REST endpoints exist as commands.

## Gap matrix

| Feature | cf | cosmoflare CLI | Ops | Adopt? | Effort |
|---|---|---|---|---|---|
| Command breadth (3,000+ ops) | generated via Forge | ~40 hand-written service commands | n/a | **Adopt selectively** — our value is curation, not breadth | large (per-service) |
| JSON-default output | JSON only, no tables | `--json` flag on every command | JSON API | **Skip** — our dual human/JSON mode is better for humans; keep `--json` mandatory | — |
| Fuzzy command discovery (`cli search`) | MiniSearch over command index, top-5 JSON | rich `--help` per command | n/a | **ADOPT** — small, high agent value | S |
| Agent-context detection | 12 harness env vars → agent help preamble | none | n/a | **ADOPT** — detect `CLAUDECODE`/`AGENT` envs, add "you are an agent" help section + compact examples | S |
| Machine-readable errors | boxed human errors, `[code]` | actionable text errors | JSON `errors[]` | **ADOPT (error codes)** — add stable `error_code` field to `--json` error output; cf lacks it — we can lead | M |
| Exit-code contract | CliExit class, 1 on failure | deterministic exit codes documented | n/a | **Skip** — already have it | — |
| Schema introspection (`cf schema`) | API schema behind each command | `cosmoflare validate` (config constraints) | n/a | **Consider later** — nice-to-have after search | M |
| MCP server mode | none | `cosmoflare mcp` exists | n/a | **Keep + market it** — cf has NO MCP; our MCP server is a differentiator | — |
| Config file w/ generated types | `cloudflare.config.ts` + `.cloudflare/types` | `.cosmoflare.yaml` | n/a | **ADOPT (types)** — ship `cosmoflare types` writing TS types for `.cosmoflare.yaml` | M |
| OAuth login flow | yes (workers-auth) | API token only | n/a | **ADOPT eventually** — `cosmoflare auth login` lowers onboarding friction | M |
| OpenAPI/codegen pipeline (Forge-style) | vendored forge, sync script | hand-written `pkg/cosmoflare/` services | n/a | **Skip for now** — curated Go SDK is our quality moat; revisit per-service | L |
| REST billing usage commands | `billing/usage` v2 commands | cost/export commands | `/api/billing` collector | **ADOPT endpoint** — verify `/accounts/{id}/billing/usage-v2` vs our GraphQL mix; may replace N calls with 1 | S |
| Workers/D1 GraphQL analytics | **MISSING** (REST-only, ZT datasets) | n/a | summary.ts GraphQL (workers 24h, D1, zones) | **Keep + extend** — we are AHEAD; richer data than cf | — |
| Upstream response caching | none (schema cache only) | n/a | KV L2 + TTLs + SWR + single-flight | **Keep** — cf refetches everything; our cache strategy is a genuine edge | — |
| Retry/backoff on 429 | SDK retries disabled | per-call retries | none documented | **ADOPT** — enable bounded retry+backoff in pkg/cosmoflare client + Ops fetcher | S |

## Data-pull strategies to adopt (with expected savings)

1. **Billing usage-v2 REST endpoint** (`billing/usage/get-account-usage-v2.ts`
   in cf): if `/accounts/{id}/billing/usage-v2` returns per-product MTD usage
   in one call, `/api/billing` can drop its per-product REST/GraphQL fan-out.
   Verify the response shape live with the account token before switching.
   Expected: several upstream calls → 1.
2. **Bounded retry + backoff on 429/5xx** in the shared client layer (cf
   disabled the SDK's — we should not): `pkg/cosmoflare` client gets
   maxRetries=2 with exponential backoff; Ops `cache.ts` load path inherits it.
   No call-count savings; reliability savings under rate limits.
3. **Keep GraphQL analytics where cf has nothing**: Workers invocations, D1
   rows, zone status — `apps/ops/src/summary.ts` stays GraphQL. Nothing to
   change; this is our differentiator vs cf.
4. **Do not chase breadth**: cf's 3,000 generated commands vs our curated 40 is
   a positioning choice, not a gap. Our roadmap: deepen the services agents
   actually use (R2, Workers, KV, D1, DNS — already implemented).

## Adopt/skip decisions (each cites a file in their repo)

1. **ADOPT fuzzy discovery** — `packages/cli/src/commands/cli/search.ts` +
   `_meta/commands.json` pattern → `cosmoflare search "..."` (or `--search`).
2. **ADOPT agent-context help** — `packages/cli/src/lib/agent-context.ts`
   pattern → detect harness envs, prepend agent guidance to help.
3. **ADOPT error codes in JSON errors** — improves on `packages/cli/src/lib/errors.ts`
   which is human-only.
4. **ADOPT `billing/usage-v2` endpoint** — command exists at
   `packages/cli/src/commands/_generated/billing/usage/get-account-usage-v2.ts`;
   mirror the call in our billing collector after live shape verification.
5. **ADOPT bounded retry** — counter-example: cf sets `maxRetries: 0`
   (`packages/cli/src/lib/raw-fetch.ts` area); we keep retries ON.
6. **ADOPT typed config** — `cloudflare.config.ts` + `cf workers types`
   (`packages/cli/src/commands/workers/types/generate.ts`) → `cosmoflare types`.
7. **SKIP JSON-only output** — cf's `lib/output.ts` has no human mode; our
   dual-mode is friendlier and already agent-complete via `--json`.
8. **SKIP Forge-style codegen** — `scripts/sync-forge.ts` + `vendor/` fits
   their 3,000-command goal; wrong tool for a curated Go library.
9. **KEEP MCP server** — zero MCP in cf (grep-verified); `cosmoflare mcp` is a
   marketable differentiator.
10. **KEEP KV-cached upstream + GraphQL analytics in Ops** — cf caches nothing
    and cannot pull Workers/D1 analytics at all.

## Follow-ups filed

Created 2026-10-10 under ROAD-102:

1. FEAT-pEKR6K6 — `cosmoflare search` fuzzy command discovery (adopt #1).
2. FEAT-pN68ZRM — agent-context detection + agent help preamble (adopt #2).
3. FEAT-p8KYM5K — stable error codes in `--json` error output (adopt #3).
4. FEAT-pDDEH5J — billing usage-v2 endpoint verification + Ops adoption (adopt #4).
5. FEAT-pC4N3QP — bounded retry/backoff in pkg/cosmoflare client (adopt #5).
