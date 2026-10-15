// Template registry (P-16, FEAT-pC1EV32): one playbook per BriefKind. The
// findings slot stays empty — compileBrief owns the verdict/threshold logic
// and orders findings itself (types.ts reserves the slot for later).
//
// Directives derive from the FEAT-067/068/070 rulebook — the CLI's Fix texts
// are the phrasing source (pkg/cosmoflare efficiency.go / caching.go),
// adapted to imperative agent direction. Verify commands cite the VERIFIED
// surface only — flags are never invented:
//   - `cosmoflare efficiency --json` emits the {status,data} envelope with
//     data.zone_coverage[].zone (flags: --window N, default 7);
//   - `cosmoflare pagerules list <zone-id>` inventories Page Rules — there
//     is NO zone cache-rules list command, so templates never cite one;
//   - the Ops /api/rules-quota?name=<zone> endpoint for rule-family
//     inventories, and /api/alerts/feed for the fired-alert history.

import type { TemplateMap, ZoneFacts } from "./types";

/** The zone-scoped acceptance check over the real efficiency envelope.
 *  `--window N` comes from the facts' own "7d"-style label and is omitted
 *  when the label is not whole days — the flag is never fabricated. */
function efficiencyVerify(f: ZoneFacts): string {
  const days = /^\d+(?=d$)/.exec(f.window)?.[0];
  return `cosmoflare efficiency --json${days ? ` --window ${days}` : ""} | jq '.data.zone_coverage[] | select(.zone=="${f.zone}")'`;
}

export const TEMPLATES: TemplateMap = {
  // efficiency.go low-cache-hit Fix, imperative.
  "low-cache-hit": (f) => ({
    findings: [],
    actions: [
      {
        directive:
          "Raise the edge cache hit rate above the 5% floor: send origin Cache-Control headers on cacheable responses, or add a zone Cache Rule — HTML needs a Cache Rule, s-maxage alone does nothing.",
      },
    ],
    verify: [`${efficiencyVerify(f)} — hit_rate_pct should climb above 5`],
  }),

  // caching.go assessHTMLCacheTrap Fix, imperative.
  "html-cache-trap": (f) => ({
    findings: [],
    actions: [
      {
        directive:
          "Origin s-maxage does not cache HTML at Cloudflare by default — add a zone Cache Rule covering the HTML paths (dash: Caching > Cache Rules), or emit the cache header at the edge (Worker fetch handler / Cache API).",
      },
    ],
    verify: [
      // The trap is proven the way caching.go's probe sees it: a GET of the
      // apex with response headers dumped; after the fix a repeat fetch
      // reports cf-cache-status: HIT.
      `curl -s -D - -o /dev/null https://${f.zone}/ — repeat fetch should report cf-cache-status: HIT`,
      `/api/rules-quota?name=${f.zone} — cache-rules family used should count the new rule`,
    ],
  }),

  // cache-rule-gap: cacheable HTML already edge-served — make it explicit.
  "cache-rule-gap": (f) => ({
    findings: [],
    actions: [
      {
        directive:
          "Add a zone Cache Rule covering the HTML paths (dash: Caching > Cache Rules) so the cache behavior is explicit configuration — HTML is currently served on default cache eligibility alone.",
      },
    ],
    verify: [`/api/rules-quota?name=${f.zone} — cache-rules family used should count the new rule`],
  }),

  // alert-firing: investigate first — the feed is the acceptance surface.
  "alert-firing": () => ({
    findings: [],
    actions: [
      {
        directive:
          "Investigate each firing alert before changing configuration — every metric below breached its threshold in the last 24h; confirm the underlying metric, then fix the cause or tune the alert rule.",
      },
    ],
    verify: ["/api/alerts/feed — this zone's entries should clear once addressed"],
  }),

  // quota-pressure: prune/consolidate; pagerules list is the only CLI rule
  // inventory (Page Rules) — other families via the Ops rules-quota endpoint.
  "quota-pressure": (f) => ({
    findings: [],
    actions: [
      {
        directive:
          "Bring the pressured rule families back under allowance: inventory first, then prune or consolidate unused rules — delete only what is retired.",
        command: `cosmoflare pagerules list ${f.zoneId}`,
      },
    ],
    verify: [`/api/rules-quota?name=${f.zone} — used should drop below 80% of allowance`],
  }),
};
