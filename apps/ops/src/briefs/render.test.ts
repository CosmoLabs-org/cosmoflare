// Golden renderer tests (P-16): renderAgentBrief is PURE — no clock, no env,
// identical Briefs render byte-identical text. Each golden below asserts the
// FULL literal expected string (not a shape snapshot), built from a real
// compileBrief over the real registry so the whole pipeline is pinned
// byte-for-byte: one golden per kind, one mixed-zone brief, plus
// TestRenderDeterministic.
import { describe, expect, it } from "vitest";
import { compileBrief } from "./compile";
import { renderAgentBrief } from "./render";
import { TEMPLATES } from "./templates";
import type { ZoneFacts } from "./types";

function mkFacts(over: Partial<ZoneFacts> = {}): ZoneFacts {
  return { zone: "example.com", zoneId: "zone-1", window: "7d", firingAlerts: [], ...over };
}

const MISS_ALERT = {
  ts: "2026-10-15T00:00:00.000Z",
  ruleId: "miss pct/zone-1",
  condition: "zone-cache-miss-pct",
  title: "cache miss % · example.com",
  value: 63.6,
  threshold: 50,
  unit: "%",
  message: "cache miss rate above 50%",
  sent: true,
  context: "46 misses of 139 eligible requests",
};

describe("renderAgentBrief goldens (full literal strings)", () => {
  it("golden: low-cache-hit", () => {
    const b = compileBrief(mkFacts({ coverage: { requests: 50_000, cached: 200, hitRatePct: 0.4 } }), TEMPLATES);
    expect(renderAgentBrief(b)).toBe(
`# example.com — Cloudflare optimization directive

## Problem
- [critical] low-cache-hit: 7d cache hit rate 0.4% — 200 of 50000 requests served from edge cache

## Actions
1. Raise the edge cache hit rate above the 5% floor: send origin Cache-Control headers on cacheable responses, or add a zone Cache Rule — HTML needs a Cache Rule, s-maxage alone does nothing.

## Verify
- cosmoflare efficiency --json --window 7 | jq '.data.zone_coverage[] | select(.zone=="example.com")' — hit_rate_pct should climb above 5

## Context
Zone example.com (zone-1) — measured 7d by CosmoLabs Ops.`,
    );
  });

  it("golden: html-cache-trap", () => {
    const b = compileBrief(
      mkFacts({ probe: { contentType: "text/html; charset=utf-8", cacheControl: "public, s-maxage=3600", cfCacheStatus: "DYNAMIC" }, cacheRuleCount: 0 }),
      TEMPLATES,
    );
    expect(renderAgentBrief(b)).toBe(
`# example.com — Cloudflare optimization directive

## Problem
- [critical] html-cache-trap: origin "public, s-maxage=3600" on HTML; cf-cache-status DYNAMIC; no zone Cache Rule exists

## Actions
1. Origin s-maxage does not cache HTML at Cloudflare by default — add a zone Cache Rule covering the HTML paths (dash: Caching > Cache Rules), or emit the cache header at the edge (Worker fetch handler / Cache API).

## Verify
- curl -s -D - -o /dev/null https://example.com/ — repeat fetch should report cf-cache-status: HIT
- /api/rules-quota?name=example.com — cache-rules family used should count the new rule

## Context
Zone example.com (zone-1) — measured 7d by CosmoLabs Ops.`,
    );
  });

  it("golden: cache-rule-gap", () => {
    const b = compileBrief(
      mkFacts({ probe: { contentType: "text/html", cacheControl: "public, max-age=300", cfCacheStatus: "HIT" }, cacheRuleCount: 0 }),
      TEMPLATES,
    );
    expect(renderAgentBrief(b)).toBe(
`# example.com — Cloudflare optimization directive

## Problem
- [info] cache-rule-gap: origin "public, max-age=300" on HTML; cf-cache-status HIT; no zone Cache Rule exists

## Actions
1. Add a zone Cache Rule covering the HTML paths (dash: Caching > Cache Rules) so the cache behavior is explicit configuration — HTML is currently served on default cache eligibility alone.

## Verify
- /api/rules-quota?name=example.com — cache-rules family used should count the new rule

## Context
Zone example.com (zone-1) — measured 7d by CosmoLabs Ops.`,
    );
  });

  it("golden: alert-firing", () => {
    const b = compileBrief(mkFacts({ firingAlerts: [MISS_ALERT] }), TEMPLATES);
    expect(renderAgentBrief(b)).toBe(
`# example.com — Cloudflare optimization directive

## Problem
- [warning] alert-firing: cache miss % · example.com: 63.6% vs threshold 50% in 24h · 46 misses of 139 eligible requests

## Actions
1. Investigate each firing alert before changing configuration — every metric below breached its threshold in the last 24h; confirm the underlying metric, then fix the cause or tune the alert rule.

## Verify
- /api/alerts/feed — this zone's entries should clear once addressed

## Context
Zone example.com (zone-1) — measured 7d by CosmoLabs Ops.`,
    );
  });

  it("golden: quota-pressure", () => {
    const b = compileBrief(
      mkFacts({ quota: { families: [{ family: "page-rules", label: "Page Rules", used: 3, allowance: 3, authorized: true }] } }),
      TEMPLATES,
    );
    expect(renderAgentBrief(b)).toBe(
`# example.com — Cloudflare optimization directive

## Problem
- [critical] quota-pressure: Page Rules: 3/3 used (100% of allowance)

## Actions
1. Bring the pressured rule families back under allowance: inventory first, then prune or consolidate unused rules — delete only what is retired. — \`cosmoflare pagerules list zone-1\`

## Verify
- /api/rules-quota?name=example.com — used should drop below 80% of allowance

## Context
Zone example.com (zone-1) — measured 7d by CosmoLabs Ops.`,
    );
  });

  it("golden: mixed-zone brief — severity-ranked findings, deduped actions/verify", () => {
    const b = compileBrief(
      mkFacts({
        coverage: { requests: 100_000, cached: 400, hitRatePct: 0.4 },
        probe: { contentType: "text/html", cacheControl: "public, s-maxage=3600", cfCacheStatus: "DYNAMIC" },
        cacheRuleCount: 2,
        firingAlerts: [MISS_ALERT],
        quota: {
          families: [
            { family: "page-rules", label: "Page Rules", used: 3, allowance: 3, authorized: true },
            { family: "cache-rules", label: "Cache Rules", used: 9, allowance: 10, authorized: true },
          ],
        },
      }),
      TEMPLATES,
    );
    expect(renderAgentBrief(b)).toBe(
`# example.com — Cloudflare optimization directive

## Problem
- [critical] low-cache-hit: 7d cache hit rate 0.4% — 400 of 100000 requests served from edge cache
- [critical] quota-pressure: Page Rules: 3/3 used (100% of allowance)
- [warning] alert-firing: cache miss % · example.com: 63.6% vs threshold 50% in 24h · 46 misses of 139 eligible requests
- [warning] html-cache-trap: origin "public, s-maxage=3600" on HTML; cf-cache-status DYNAMIC; 2 zone Cache Rule(s) exist — check they cover HTML paths
- [warning] quota-pressure: Cache Rules: 9/10 used (90% of allowance)

## Actions
1. Raise the edge cache hit rate above the 5% floor: send origin Cache-Control headers on cacheable responses, or add a zone Cache Rule — HTML needs a Cache Rule, s-maxage alone does nothing.
2. Bring the pressured rule families back under allowance: inventory first, then prune or consolidate unused rules — delete only what is retired. — \`cosmoflare pagerules list zone-1\`
3. Investigate each firing alert before changing configuration — every metric below breached its threshold in the last 24h; confirm the underlying metric, then fix the cause or tune the alert rule.
4. Origin s-maxage does not cache HTML at Cloudflare by default — add a zone Cache Rule covering the HTML paths (dash: Caching > Cache Rules), or emit the cache header at the edge (Worker fetch handler / Cache API).

## Verify
- cosmoflare efficiency --json --window 7 | jq '.data.zone_coverage[] | select(.zone=="example.com")' — hit_rate_pct should climb above 5
- /api/rules-quota?name=example.com — used should drop below 80% of allowance
- /api/alerts/feed — this zone's entries should clear once addressed
- curl -s -D - -o /dev/null https://example.com/ — repeat fetch should report cf-cache-status: HIT
- /api/rules-quota?name=example.com — cache-rules family used should count the new rule

## Context
Zone example.com (zone-1) — measured 7d by CosmoLabs Ops.`,
    );
  });
});

describe("TestRenderDeterministic", () => {
  it("renders byte-identical text from identical Briefs, equal to the golden", () => {
    const facts = mkFacts({
      coverage: { requests: 100_000, cached: 400, hitRatePct: 0.4 },
      probe: { contentType: "text/html", cacheControl: "public, s-maxage=3600", cfCacheStatus: "DYNAMIC" },
      cacheRuleCount: 2,
      firingAlerts: [MISS_ALERT],
      quota: { families: [{ family: "page-rules", label: "Page Rules", used: 3, allowance: 3, authorized: true }] },
    });
    const a = renderAgentBrief(compileBrief(facts, TEMPLATES));
    const b = renderAgentBrief(compileBrief(facts, TEMPLATES));
    expect(a).toBe(b);
    expect(a).toMatch(/^# example\.com — Cloudflare optimization directive\n/);
    expect(a).toMatch(/\nZone example\.com \(zone-1\) — measured 7d by CosmoLabs Ops\.$/);
  });
});
