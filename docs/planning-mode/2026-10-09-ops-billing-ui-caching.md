---
title: Cosmoflare Ops — billing view, UI/responsive pass, upstream caching, cron push
created: "2026-10-09T14:00:00+04:00"
issue: FEAT-052
status: IN_PROGRESS
deliverables:
    - P-01: Upstream caching layer for the Ops Worker (per-dataset TTL, stale-while-revalidate, single-flight)
    - P-02: Summary v2 — per-Worker 24h rows, zone status breakdown, cache metadata
    - P-03: /api/billing collector — billing period, per-product allowance/usage/projection/overage, top consumers, project attribution, verified prices
    - P-04: Pager UI — responsive layout, hamburger drawer, overview/billing/workers/D1/zones views
    - P-05: App icon set (home-screen, maskable, favicon)
    - P-06: Cron push from the Worker (TS port of FEAT-049 rules, Web Push aes128gcm, /api/subscribe)
    - P-07: Integration (routes, wrangler bindings, KV namespace, deploy) + docs
---

# Goal

The operator opens https://ops.cosmolabs.org on an iPhone or a desktop and sees,
in one place: where the account is in its billing period, what each product
will cost beyond its allowance, which project causes it, and the 24h health of
Workers, D1 and zones. The layout works at 375px (hamburger drawer) and at
desktop width (sidebar). Upstream Cloudflare calls stay low and predictable
through caching. Later the Worker pages the phone itself on a Cron Trigger.

Operator request 2026-10-09: "improve the design, improve the responsive web
design and hamburger menu and more, and how much info we get to see. And
ensure we are also pulling this info from Cloudflare efficiently without
sacrificing requests, with caching where it makes sense."

# Execution model

Opus orchestrates only. Each workstream runs in its own git worktree (Agent
tool, `isolation: worktree`). Opus reads every diff and re-runs tests in the
worktree before `git merge --no-ff`.

| Wave | Agent | Model | Owns (only these files) |
|------|-------|-------|-------------------------|
| 1 | A — caching + summary v2 | sonnet | `apps/ops/src/cache.ts` (new), `apps/ops/src/cache.test.ts` (new), `apps/ops/src/summary.ts`, `apps/ops/src/summary.test.ts`, `apps/ops/src/index.ts`, `apps/ops/src/index.test.ts` |
| 1 | B — billing collector | sonnet | `apps/ops/src/billing.ts`, `apps/ops/src/pricing.ts`, `apps/ops/src/projects.ts` (+ `*.test.ts`) — all new |
| 1 | C — pager UI | sonnet | `pager/src/**` (not `pager/public/**`, not `pager/index.html`) |
| 1 | D — icon set | sonnet | `pager/public/icon-*`, `pager/public/favicon.svg`, `pager/public/apple-touch-icon.png`, `pager/public/manifest.webmanifest`, `pager/index.html` |
| 1 | E — cron push engine | sonnet | `apps/ops/src/rules.ts`, `apps/ops/src/webpush.ts`, `apps/ops/src/subscriptions.ts`, `apps/ops/src/scheduled.ts` (+ `*.test.ts`) — all new; `apps/ops/package.json` (dependency only) |
| 2 | F — integration | sonnet | `apps/ops/src/index.ts`, `apps/ops/wrangler.jsonc`, KV namespace, deploy |
| 2 | G — docs | haiku | `apps/ops/README.md`, `docs/USAGE.md` (Ops section) |

Merge order: A → B → E → D → C → F → G. Integration point: F wires B and E
into `index.ts` through A's cache.

# Shared contracts (every agent codes against these exact shapes)

## Cache (agent A produces, F consumes)

```ts
// apps/ops/src/cache.ts
export interface CacheEnv { OPS_KV?: KVNamespace }
/** Returns the cached value for key when younger than ttlSec. When older but
 *  younger than staleSec, returns the stale value and refreshes in the
 *  background via ctx.waitUntil (stale-while-revalidate). Concurrent misses
 *  for the same key share one in-flight load (single-flight). L1 = module
 *  memory, L2 = Cache API (caches.default) or KV when bound. */
export function cached<T>(
  ctx: ExecutionContext, env: CacheEnv, key: string,
  opts: { ttlSec: number; staleSec: number },
  load: () => Promise<T>,
): Promise<{ value: T; ageSec: number; stale: boolean }>;
```

TTLs (agent A documents the final values in `cache.ts`):

| Dataset | Fresh TTL | Stale window | Reason |
|---------|-----------|--------------|--------|
| Zone / D1 / Worker / bucket / namespace lists | 1 h | 24 h | change on a days timescale |
| 24h analytics (summary) | 5 min | 1 h | GraphQL data lags 1-5 min anyway |
| Month-to-date billing usage | 15 min | 6 h | projection moves slowly |
| Billing subscription (period dates) | 12 h | 7 d | changes once a month |
| Prices | constants in code | — | verified by hand, source URL + date |

`?refresh` bypasses L1/L2 for the summary and billing routes, but at most once
per 60 s per key (refresh-storm guard).

## GET /api/summary v2 (agent A produces, C consumes)

Additive to the current shape (`apps/ops/src/summary.ts` `Summary`):

```ts
interface Summary {
  generatedAt: string; windowHours: number;
  usage: UsageRow[]; d1: D1Row[]; zones: ZoneRow[]; errors: string[];
  // new in v2
  workers: { script: string; requests: number; errors: number; errorPct: number;
             cpuP50Ms: number | null; cpuP99Ms: number | null }[]; // sorted by requests desc
  cache: { ageSec: number; stale: boolean };   // filled by the route from cached()
}
// ZoneRow gains:  byStatus: Record<string, number>   (every cacheStatus count)
// D1Row gains:    rowsWritten: number; databaseId: string
```

## GET /api/billing (agent B produces `collectBilling`, C consumes, F routes)

```ts
// apps/ops/src/billing.ts
export function collectBilling(accountId: string, token: string, now?: Date,
  opts?: { anchorDay?: number; projectMap?: Record<string, string> }): Promise<Billing>;

interface Billing {
  generatedAt: string;
  period: { start: string; end: string; day: number; days: number;
            source: "subscription" | "anchor" | "calendar" };
  products: ProductUsage[];          // sorted by projectedOverageUsd desc, then product
  totalProjectedOverageUsd: number;
  projects: { project: string; projectedOverageUsd: number; drivers: string[] }[]; // drivers = ProductUsage ids
  pricing: { verifiedOn: string; sources: { product: string; url: string }[] };
  errors: string[];                  // one entry per dataset that failed; never a failed response
  cache?: { ageSec: number; stale: boolean };
}
interface ProductUsage {
  id: string;            // e.g. "workers.requests", "workers.cpu_ms", "d1.rows_read", "d1.rows_written",
                         // "d1.storage", "r2.storage", "r2.class_a", "r2.class_b", "kv.reads", "kv.writes",
                         // "kv.storage", "do.requests", "do.duration"
  product: string;       // "Workers" | "D1" | "R2" | "KV" | "Durable Objects"
  metric: string;        // human label, e.g. "Rows read"
  unit: string;          // "requests" | "rows" | "ms" | "GB" | "GB-s" | "ops"
  included: number;      // monthly allowance on Workers Paid
  used: number;          // period to date (storage: current or average, say which in metric)
  projected: number;     // linear to period end (storage: = used)
  unitPriceUsd: number;  // USD per priceUnit units beyond the allowance
  priceUnit: number;     // e.g. 1_000_000
  projectedOverageUsd: number;
  topConsumers: { name: string; project: string; used: number; share: number }[]; // top 5, share 0..1
}
```

## Cron push (agent E produces, F wires)

```ts
// apps/ops/src/scheduled.ts
export async function runScheduled(env: OpsEnv, now?: Date): Promise<{ fired: number; sent: number; pruned: number; gaps: string[] }>;
// apps/ops/src/subscriptions.ts
export async function handleSubscribe(request: Request, env: OpsEnv): Promise<Response>; // POST add, DELETE remove, GET count
// OpsEnv extends the current Env with OPS_KV: KVNamespace; VAPID_PRIVATE_KEY: string; VAPID_SUBJECT: string
```

# Constraints (all agents)

- Live facts only: every Cloudflare API field, GraphQL dataset, permission and
  price is verified live (introspection or a real call with the account token)
  or from a cited Cloudflare docs page with the date. Values from memory are forbidden.
- Credentials: `eval "$(ccs credentials source)"` gives `CF_ACCOUNT_ID` /
  `CF_API_TOKEN`. Never print, log or commit them. Never touch the macOS keychain.
- This repo is public: no account IDs, tokens, emails or KV IDs in committed
  files except where wrangler requires the KV namespace ID (agent F only).
- Read-only against Cloudflare. No deploys, no secret puts, no pushes, except agent F's deploy.
- TypeScript: `bunx tsc --noEmit` exit 0 and `bun run test` green in the package you touched.
  Pager also: `bun run build` exit 0. Bun only.
- Conventional commits, no AI attribution trailers, no `Co-Authored-By`.
- Workers best practices: no floating promises (use ctx.waitUntil), no
  request-scoped state in module globals except the cache L1.
</content>
</invoke>
