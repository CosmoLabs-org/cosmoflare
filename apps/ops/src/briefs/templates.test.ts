// Template registry tests (P-16): every v1 kind has a playbook, the findings
// slot stays empty (compileBrief owns verdicts), the directives quote the
// FEAT-067/068/070 rulebook phrasing, and every verify/command string stays
// on the VERIFIED surface — a template that invents a CLI flag fails here.
import { describe, expect, it } from "vitest";
import { TEMPLATES } from "./templates";
import type { BriefKind, ZoneFacts } from "./types";

const ALL_KINDS: BriefKind[] = ["low-cache-hit", "html-cache-trap", "cache-rule-gap", "alert-firing", "quota-pressure"];

function zoneFacts(over: Partial<ZoneFacts> = {}): ZoneFacts {
  return { zone: "example.com", zoneId: "zone-1", window: "7d", firingAlerts: [], ...over };
}

describe("TEMPLATES registry shape", () => {
  it("covers exactly the v1 kinds, one entry each", () => {
    expect(Object.keys(TEMPLATES).sort()).toEqual([...ALL_KINDS].sort());
  });

  it("leaves the findings slot empty — compileBrief owns verdicts", () => {
    const f = zoneFacts();
    for (const kind of ALL_KINDS) {
      expect(TEMPLATES[kind](f).findings).toEqual([]);
    }
  });

  it("is deterministic: identical facts produce identical playbooks", () => {
    const f = zoneFacts({ zoneId: "abc123" });
    for (const kind of ALL_KINDS) {
      expect(TEMPLATES[kind](f)).toEqual(TEMPLATES[kind](f));
    }
  });
});

describe("TEMPLATES verified-surface guard", () => {
  // The only cosmoflare invocations that exist: efficiency --json and
  // pagerules list. There is NO zone cache-rules list command — a template
  // citing one must fail.
  const ALLOWED_VERIFY_PREFIXES = [
    "cosmoflare efficiency --json",
    "cosmoflare pagerules list ",
    "curl -s -D - -o /dev/null https://",
    "/api/rules-quota?name=",
    "/api/alerts/feed",
  ];

  function stringsOf(f: ZoneFacts): string[] {
    const out: string[] = [];
    for (const kind of ALL_KINDS) {
      const tpl = TEMPLATES[kind](f);
      for (const a of tpl.actions) {
        out.push(a.directive);
        if (a.command) out.push(a.command);
      }
      out.push(...tpl.verify);
    }
    return out;
  }

  it("every verify entry starts with a verified command shape", () => {
    for (const s of stringsOf(zoneFacts())) {
      const isVerifyLike = !s.startsWith("cosmoflare") || true; // directives are prose; check all strings
      if (!isVerifyLike) continue;
      const looksLikeCommand = /^(cosmoflare|curl|\/api\/)/.test(s);
      if (!looksLikeCommand) continue;
      expect(ALLOWED_VERIFY_PREFIXES.some((p) => s.startsWith(p))).toBe(true);
    }
  });

  it("never cites a zone cache-rules list command (none exists)", () => {
    for (const s of stringsOf(zoneFacts())) {
      expect(s).not.toMatch(/cosmoflare (zone|cache) /);
      expect(s).not.toContain("cache rules list");
      expect(s).not.toContain("cache-rules list");
    }
  });

  it("the only cosmoflare subcommands used are efficiency and pagerules", () => {
    for (const s of stringsOf(zoneFacts())) {
      for (const m of s.matchAll(/cosmoflare ([a-z]+)/g)) {
        expect(["efficiency", "pagerules"]).toContain(m[1]);
      }
    }
  });
});

describe("TEMPLATES per-kind playbooks", () => {
  it("low-cache-hit: CLI Fix directive + zone-scoped efficiency check", () => {
    const tpl = TEMPLATES["low-cache-hit"](zoneFacts());
    expect(tpl.actions).toEqual([
      {
        directive:
          "Raise the edge cache hit rate above the 5% floor: send origin Cache-Control headers on cacheable responses, or add a zone Cache Rule — HTML needs a Cache Rule, s-maxage alone does nothing.",
      },
    ]);
    expect(tpl.verify).toEqual([
      "cosmoflare efficiency --json --window 7 | jq '.data.zone_coverage[] | select(.zone==\"example.com\")' — hit_rate_pct should climb above 5",
    ]);
  });

  it("low-cache-hit: --window N follows the facts' own day window", () => {
    expect(TEMPLATES["low-cache-hit"](zoneFacts({ window: "30d" })).verify[0]).toContain("cosmoflare efficiency --json --window 30 |");
  });

  it("low-cache-hit: a non-day window label never fabricates the flag", () => {
    expect(TEMPLATES["low-cache-hit"](zoneFacts({ window: "24h" })).verify[0]).toBe(
      "cosmoflare efficiency --json | jq '.data.zone_coverage[] | select(.zone==\"example.com\")' — hit_rate_pct should climb above 5",
    );
  });

  it("html-cache-trap: quotes the caching.go Fix and verifies via probe + rules-quota", () => {
    const tpl = TEMPLATES["html-cache-trap"](zoneFacts());
    expect(tpl.actions).toEqual([
      {
        directive:
          "Origin s-maxage does not cache HTML at Cloudflare by default — add a zone Cache Rule covering the HTML paths (dash: Caching > Cache Rules), or emit the cache header at the edge (Worker fetch handler / Cache API).",
      },
    ]);
    expect(tpl.verify).toEqual([
      "curl -s -D - -o /dev/null https://example.com/ — repeat fetch should report cf-cache-status: HIT",
      "/api/rules-quota?name=example.com — cache-rules family used should count the new rule",
    ]);
  });

  it("cache-rule-gap: prescribes the explicit Cache Rule, inventories via rules-quota", () => {
    const tpl = TEMPLATES["cache-rule-gap"](zoneFacts());
    expect(tpl.actions).toEqual([
      {
        directive:
          "Add a zone Cache Rule covering the HTML paths (dash: Caching > Cache Rules) so the cache behavior is explicit configuration — HTML is currently served on default cache eligibility alone.",
      },
    ]);
    expect(tpl.verify).toEqual(["/api/rules-quota?name=example.com — cache-rules family used should count the new rule"]);
  });

  it("alert-firing: investigate-first directive, the feed is the acceptance surface", () => {
    const tpl = TEMPLATES["alert-firing"](zoneFacts());
    expect(tpl.actions).toEqual([
      {
        directive:
          "Investigate each firing alert before changing configuration — every metric below breached its threshold in the last 24h; confirm the underlying metric, then fix the cause or tune the alert rule.",
      },
    ]);
    expect(tpl.verify).toEqual(["/api/alerts/feed — this zone's entries should clear once addressed"]);
  });

  it("quota-pressure: pagerules list carries the real zone id, quota via rules-quota", () => {
    const tpl = TEMPLATES["quota-pressure"](zoneFacts({ zoneId: "abc123def" }));
    expect(tpl.actions).toEqual([
      {
        directive:
          "Bring the pressured rule families back under allowance: inventory first, then prune or consolidate unused rules — delete only what is retired.",
        command: "cosmoflare pagerules list abc123def",
      },
    ]);
    expect(tpl.verify).toEqual(["/api/rules-quota?name=example.com — used should drop below 80% of allowance"]);
  });
});
