// Cosmoflare Ops pricing table (FEAT-052): Workers Paid allowances and unit
// prices for every product the billing view projects. Every row verified
// against the live Cloudflare docs on 2026-10-09. Prices from memory are
// forbidden.

export interface PriceRow {
  id: string;
  product: string;
  metric: string;
  unit: string;
  included: number;
  unitPriceUsd: number;
  priceUnit: number;
  sourceUrl: string;
  verifiedOn: string;
}

// Compact rows: [id, metric, unit, included, unitPriceUsd, priceUnit]
// The page date each product's numbers were read from is noted inline.
const ROWS: [string, string, string, string, number, number, number][] = [
  // Workers pricing page (dateModified 2026-10-02): "10 million included per
  // month +$0.30 per additional million".
  ["workers.requests", "Workers", "Requests", "requests", 10_000_000, 0.3, 1_000_000],
  // Same page: "30 million CPU milliseconds included per month +$0.02 per
  // additional million CPU milliseconds".
  ["workers.cpu_ms", "Workers", "CPU time", "ms", 30_000_000, 0.02, 1_000_000],
  // D1 pricing page (last updated Apr 21, 2026): "First 25 billion / month
  // included + $0.001 / million rows".
  ["d1.rows_read", "D1", "Rows read", "rows", 25_000_000_000, 0.001, 1_000_000],
  // Same page: "First 50 million / month included + $1.00 / million rows".
  ["d1.rows_written", "D1", "Rows written", "rows", 50_000_000, 1.0, 1_000_000],
  // Same page: "First 5 GB included + $0.75 / GB-mo".
  ["d1.storage", "D1", "Storage", "GB", 5, 0.75, 1],
  // R2 pricing page (last updated Oct 1, 2026): "Storage 10 GB-month / month,
  // $0.015 / GB-month".
  ["r2.storage", "R2", "Storage", "GB", 10, 0.015, 1],
  // Same page: "Class A Operations 1 million requests / month, $4.50 /
  // million requests".
  ["r2.class_a", "R2", "Class A operations", "ops", 1_000_000, 4.5, 1_000_000],
  // Same page: "Class B Operations 10 million requests / month, $0.36 /
  // million requests".
  ["r2.class_b", "R2", "Class B operations", "ops", 10_000_000, 0.36, 1_000_000],
  // Workers pricing page (2026-10-02): "Keys read 10 million/month,
  // + $0.50/million".
  ["kv.reads", "KV", "Keys read", "requests", 10_000_000, 0.5, 1_000_000],
  // Same page: "Keys written 1 million/month, + $5.00/million".
  ["kv.writes", "KV", "Keys written", "requests", 1_000_000, 5.0, 1_000_000],
  // Same page: "Stored data 1 GB, + $0.50/GB-month".
  ["kv.storage", "KV", "Stored data", "GB", 1, 0.5, 1],
  // Workers pricing page (2026-10-02): DO "Requests 1 million / month,
  // + $0.15/million".
  ["do.requests", "Durable Objects", "Requests", "requests", 1_000_000, 0.15, 1_000_000],
  // Same page: "Duration 400,000 GB-s / month, + $12.50/million GB-s".
  ["do.duration", "Durable Objects", "Duration", "GB-s", 400_000, 12.5, 1_000_000],
];

const SOURCE: Record<string, string> = {
  workers: "https://developers.cloudflare.com/workers/platform/pricing/",
  d1: "https://developers.cloudflare.com/d1/platform/pricing/",
  r2: "https://developers.cloudflare.com/r2/pricing/",
  kv: "https://developers.cloudflare.com/kv/platform/pricing/",
  do: "https://developers.cloudflare.com/durable-objects/platform/pricing/",
};

export const PRICES: PriceRow[] = ROWS.map(([id, product, metric, unit, included, unitPriceUsd, priceUnit]) => ({
  id,
  product,
  metric,
  unit,
  included,
  unitPriceUsd,
  priceUnit,
  sourceUrl: SOURCE[id.split(".")[0]],
  verifiedOn: "2026-10-09",
}));

export const PRICING_SOURCES: { product: string; url: string }[] = Object.entries(SOURCE).map(([product, url]) => ({
  product,
  url,
}));

export function priceFor(id: string): PriceRow {
  const row = PRICES.find((p) => p.id === id);
  if (!row) throw new Error(`no verified price for ${id}`);
  return row;
}
