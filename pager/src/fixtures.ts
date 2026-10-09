// Fixture data for localhost demo mode (FEAT-052): shapes match the
// /api/summary v2 and /api/billing contracts in
// docs/planning-mode/2026-10-09-ops-billing-ui-caching.md exactly, so the
// same rendering code runs against live data and fixtures.

import type { Billing, ProductUsage, Summary } from "./api";
import type { RulesPayload } from "./rules";

// makeFixtures builds a coherent demo dataset around `now`: the billing
// period spans the current calendar month, pacing follows the elapsed share
// of the month, and the overage story reads like a real account (R2 storage
// running hot, D1 reads slightly ahead, everything else inside its
// allowance).
export function makeFixtures(now: Date): { summary: Summary; billing: Billing; rules: RulesPayload } {
  const year = now.getUTCFullYear();
  const month = now.getUTCMonth();
  const start = new Date(Date.UTC(year, month, 1));
  const end = new Date(Date.UTC(year, month + 1, 1));
  const days = Math.round((end.getTime() - start.getTime()) / 86_400_000);
  const day = now.getUTCDate();
  const share = day / days;

  const summary: Summary = {
    generatedAt: now.toISOString(),
    windowHours: 24,
    usage: [
      { name: "Workers requests", used: 18_400_000, limit: 10_000_000, pct: 184, projectedPct: Math.round(18_400_000 / 10_000_000 / share) },
      { name: "D1 rows read", used: 640e6, limit: 25e9, pct: 2.6, projectedPct: null },
      { name: "KV reads", used: 1_950_000, limit: 10_000_000, pct: 19.5, projectedPct: null },
    ],
    d1: [
      { name: "cf-state", databaseId: "fit-d1-01", rowsRead: 412e6, rowsWritten: 2.1e6, readQueries: 1.2e6, rowsPerQuery: 343 },
      { name: "domains", databaseId: "fit-d1-02", rowsRead: 180e6, rowsWritten: 890e3, readQueries: 620e3, rowsPerQuery: 290 },
      { name: "usage-ledger", databaseId: "fit-d1-03", rowsRead: 47e6, rowsWritten: 130e3, readQueries: 210e3, rowsPerQuery: 224 },
      { name: "cosmokit-registry", databaseId: "fit-d1-04", rowsRead: 1.1e6, rowsWritten: 45e3, readQueries: 12e3, rowsPerQuery: 92 },
    ],
    zones: [
      { zone: "cosmolabs.org", total: 412_000, uncached: 36_321, missPct: 8.8, byStatus: { hit: 330_000, miss: 41_000, expired: 4_000, dynamic: 33_000, bypass: 3_000, unknown: 1_000 } },
      { zone: "cosmoflare.dev", total: 95_000, uncached: 2_100, missPct: 2.2, byStatus: { hit: 88_000, miss: 6_000, expired: 500, dynamic: 400, bypass: 100 } },
    ],
    workers: [
      { script: "api-gateway", requests: 1_240_000, errors: 620, errorPct: 0.05, cpuP50Ms: 3.1, cpuP99Ms: 42 },
      { script: "ccs-state", requests: 402_000, errors: 1_800, errorPct: 0.45, cpuP50Ms: 2.2, cpuP99Ms: 18 },
      { script: "cdn", requests: 210_000, errors: 0, errorPct: 0, cpuP50Ms: 1.1, cpuP99Ms: 6 },
    ],
    errors: [],
  };

  const billingProducts = [
    {
      id: "workers.requests", product: "Workers", metric: "Requests", unit: "requests",
      included: 10_000_000, used: 18_400_000, projected: Math.round(18_400_000 / share),
      unitPriceUsd: 0.30, priceUnit: 1_000_000, projectedOverageUsd: 0,
      topConsumers: [
        { name: "api-gateway", project: "ops", used: 9_100_000, share: 0.49 },
        { name: "ccs-state", project: "ccs", used: 5_400_000, share: 0.29 },
        { name: "cdn", project: "web", used: 2_300_000, share: 0.13 },
      ],
    },
    {
      id: "d1.rows_read", product: "D1", metric: "Rows read", unit: "rows",
      included: 25e9, used: 640e6, projected: Math.round(640e6 / share),
      unitPriceUsd: 0.001, priceUnit: 1_000_000, projectedOverageUsd: 0,
      topConsumers: [
        { name: "cf-state", project: "ccs", used: 12e9, share: 0.55 },
        { name: "domains", project: "cosmoflare", used: 5.5e9, share: 0.25 },
      ],
    },
    {
      id: "r2.storage", product: "R2", metric: "Storage (average)", unit: "GB",
      included: 10, used: 214, projected: 214,
      unitPriceUsd: 0.015, priceUnit: 1, projectedOverageUsd: 0,
      topConsumers: [
        { name: "media-bucket", project: "web", used: 180, share: 0.84 },
        { name: "backups", project: "ops", used: 31, share: 0.14 },
      ],
    },
    {
      id: "kv.reads", product: "KV", metric: "Reads", unit: "ops",
      included: 10_000_000, used: 1_950_000, projected: Math.round(1_950_000 / share),
      unitPriceUsd: 0.50, priceUnit: 1_000_000, projectedOverageUsd: 0,
      topConsumers: [{ name: "session-cache", project: "ops", used: 1.6e6, share: 0.82 }],
    },
  ];

  for (const p of billingProducts) {
    p.projectedOverageUsd = Math.max(0, (p.projected - p.included) / p.priceUnit) * p.unitPriceUsd;
  }
  billingProducts.sort((a, b) => b.projectedOverageUsd - a.projectedOverageUsd);

  const billing: Billing = {
    generatedAt: now.toISOString(),
    period: {
      start: start.toISOString(),
      end: end.toISOString(),
      day,
      days,
      source: "calendar",
    },
    products: billingProducts,
    totalProjectedOverageUsd: billingProducts.reduce((sum, p) => sum + p.projectedOverageUsd, 0),
    projects: buildProjects(billingProducts),
    pricing: {
      verifiedOn: "2026-10-09",
      sources: [
        { product: "Workers", url: "https://developers.cloudflare.com/workers/platform/pricing/" },
        { product: "R2", url: "https://developers.cloudflare.com/r2/pricing/" },
      ],
    },
    errors: ["d1 storage: Cloudflare GraphQL timed out — usage is from the last successful pull"],
  };

  // Rules demo set: one rule per shape the view edits — a D1 rows-read rule
  // (threshold formats as "1.0B" in the hint), a Worker error-rate rule with
  // excludes, and a disabled rule. starter: false shows the customised path.
  const rules: RulesPayload = {
    conditions: ["d1-rows-read", "worker-error-pct", "r2-storage-gb", "kv-reads", "zone-miss-pct"],
    starter: false,
    rules: [
      {
        name: "D1 reads runaway",
        condition: "d1-rows-read",
        threshold: 1e9,
        exclude: ["cosmokit-registry"],
        enabled: true,
      },
      {
        name: "API gateway errors",
        condition: "worker-error-pct",
        threshold: 1,
        exclude: ["cdn", "ccs-state"],
        enabled: true,
      },
      {
        name: "Zone cache misses",
        condition: "zone-miss-pct",
        threshold: 15,
        enabled: false,
      },
    ],
  };
  return { summary, billing, rules };
}

// buildProjects aggregates product overages into per-project rows: a project
// pays for the share of each product's overage its consumers carry.
function buildProjects(products: ProductUsage[]): ProjectCostLike[] {
  const byProject = new Map<string, { overage: number; drivers: Set<string> }>();
  for (const p of products) {
    if (p.projectedOverageUsd <= 0) continue;
    for (const c of p.topConsumers) {
      const entry = byProject.get(c.project) ?? { overage: 0, drivers: new Set<string>() };
      entry.overage += p.projectedOverageUsd * c.share;
      entry.drivers.add(p.id);
      byProject.set(c.project, entry);
    }
    // Products with no consumer attribution still show under "unattributed".
    if (p.topConsumers.length === 0) {
      const entry = byProject.get("unattributed") ?? { overage: 0, drivers: new Set<string>() };
      entry.overage += p.projectedOverageUsd;
      entry.drivers.add(p.id);
      byProject.set("unattributed", entry);
    }
  }
  return [...byProject.entries()]
    .map(([project, entry]) => ({ project, projectedOverageUsd: entry.overage, drivers: [...entry.drivers] }))
    .sort((a, b) => b.projectedOverageUsd - a.projectedOverageUsd);
}

type ProjectCostLike = { project: string; projectedOverageUsd: number; drivers: string[] };
