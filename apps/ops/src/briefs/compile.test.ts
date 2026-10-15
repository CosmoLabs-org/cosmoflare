// Brief compiler tests (P-15): compileBrief is PURE — fixtures in, Brief out,
// no env/fetch/Date.now. Every threshold case below asserts on the verdict
// (finding presence + severity), never re-derives the threshold math in the
// test body. Templates are stubs so actions/verify are traceable per kind.
import { describe, expect, it } from "vitest";
import { compileBrief } from "./compile";
import type { BriefFinding, TemplateMap, ZoneFacts } from "./types";

// Stub registry: each entry's actions/verify are unique strings so tests can
// assert which fired kinds contributed to the Brief.
const TEMPLATES: TemplateMap = {
  "low-cache-hit": () => ({
    findings: [],
    actions: [{ directive: "raise edge caching for the zone", command: "cosmoflare efficiency --json" }],
    verify: ["verify-low-cache-hit"],
  }),
  "html-cache-trap": () => ({
    findings: [],
    actions: [{ directive: "add a zone Cache Rule covering HTML paths" }],
    verify: ["verify-html-cache-trap"],
  }),
  "cache-rule-gap": () => ({ findings: [], actions: [{ directive: "gap-action" }], verify: ["verify-cache-rule-gap"] }),
  "alert-firing": () => ({ findings: [], actions: [{ directive: "inspect the firing alert" }], verify: ["verify-alert-firing"] }),
  "quota-pressure": () => ({ findings: [], actions: [{ directive: "prune or consolidate rules" }], verify: ["verify-quota-pressure"] }),
};

function facts(over: Partial<ZoneFacts> = {}): ZoneFacts {
  return { zone: "example.com", zoneId: "zone-1", window: "7d", firingAlerts: [], ...over };
}

function kindsOf(b: { findings: BriefFinding[] }): string[] {
  return b.findings.map((f) => f.kind);
}

describe("compileBrief low-cache-hit", () => {
  it("fires critical under 1% with enough requests", () => {
    const b = compileBrief(facts({ coverage: { requests: 50_000, cached: 200, hitRatePct: 0.4 } }), TEMPLATES);
    const f = b.findings.find((x) => x.kind === "low-cache-hit");
    expect(f?.severity).toBe("critical");
    expect(f?.evidence).toContain("0.4%");
    // template contributions for the fired kind land in the brief
    expect(b.actions).toEqual([{ directive: "raise edge caching for the zone", command: "cosmoflare efficiency --json" }]);
    expect(b.verify).toEqual(["verify-low-cache-hit"]);
  });

  it("fires warning between 1% and 5%", () => {
    const b = compileBrief(facts({ coverage: { requests: 50_000, cached: 1_250, hitRatePct: 2.5 } }), TEMPLATES);
    const f = b.findings.find((x) => x.kind === "low-cache-hit");
    expect(f?.severity).toBe("warning");
  });

  it("suppresses at exactly 5%", () => {
    const b = compileBrief(facts({ coverage: { requests: 50_000, cached: 2_500, hitRatePct: 5 } }), TEMPLATES);
    expect(kindsOf(b)).not.toContain("low-cache-hit");
  });

  it("suppresses under 1000 requests", () => {
    const b = compileBrief(facts({ coverage: { requests: 999, cached: 0, hitRatePct: 0 } }), TEMPLATES);
    expect(kindsOf(b)).not.toContain("low-cache-hit");
  });

  it("fires at exactly 1000 requests with 0% hit rate (critical)", () => {
    const b = compileBrief(facts({ coverage: { requests: 1_000, cached: 0, hitRatePct: 0 } }), TEMPLATES);
    const f = b.findings.find((x) => x.kind === "low-cache-hit");
    expect(f?.severity).toBe("critical");
  });

  it("emits nothing without coverage", () => {
    const b = compileBrief(facts(), TEMPLATES);
    expect(b.findings).toEqual([]);
    expect(b.actions).toEqual([]);
    expect(b.verify).toEqual([]);
  });
});

describe("compileBrief html-cache-trap (caching.go matrix)", () => {
  const probe = (cacheControl: string, cfCacheStatus: string, contentType = "text/html; charset=utf-8") => ({
    contentType,
    cacheControl,
    cfCacheStatus,
  });

  it("flags HTML + public s-maxage + DYNAMIC as warning when rules exist", () => {
    const b = compileBrief(facts({ probe: probe("public, s-maxage=3600", "DYNAMIC"), cacheRuleCount: 2 }), TEMPLATES);
    const f = b.findings.find((x) => x.kind === "html-cache-trap");
    expect(f?.severity).toBe("warning");
    expect(f?.evidence).toContain('origin "public, s-maxage=3600" on HTML');
    expect(f?.evidence).toContain("cf-cache-status DYNAMIC");
    expect(f?.evidence).toContain("2 zone Cache Rule(s) exist");
  });

  it("escalates to critical when no zone Cache Rule exists", () => {
    const b = compileBrief(facts({ probe: probe("public, s-maxage=3600", "DYNAMIC"), cacheRuleCount: 0 }), TEMPLATES);
    const f = b.findings.find((x) => x.kind === "html-cache-trap");
    expect(f?.severity).toBe("critical");
    expect(f?.evidence).toContain("no zone Cache Rule exists");
  });

  it("stays warning and says so when Cache Rules are not visible", () => {
    const b = compileBrief(facts({ probe: probe("public, s-maxage=3600", "DYNAMIC"), cacheRuleCount: undefined }), TEMPLATES);
    const f = b.findings.find((x) => x.kind === "html-cache-trap");
    expect(f?.severity).toBe("warning");
    expect(f?.evidence).toContain("not visible with this token");
  });

  it("flags a missing cf-cache-status header (empty status) as a trap", () => {
    const b = compileBrief(facts({ probe: probe("public, s-maxage=3600", ""), cacheRuleCount: 1 }), TEMPLATES);
    const f = b.findings.find((x) => x.kind === "html-cache-trap");
    expect(f?.severity).toBe("warning");
    expect(f?.evidence).toContain("no cf-cache-status header");
  });

  it("accepts max-age>=60 as the cacheable floor, case-insensitively", () => {
    const b = compileBrief(facts({ probe: probe("PUBLIC, MAX-AGE=60", "DYNAMIC"), cacheRuleCount: 1 }), TEMPLATES);
    expect(kindsOf(b)).toContain("html-cache-trap");
  });

  it("stays clean below the max-age floor (59s)", () => {
    const b = compileBrief(facts({ probe: probe("public, max-age=59", "DYNAMIC"), cacheRuleCount: 0 }), TEMPLATES);
    expect(kindsOf(b)).not.toContain("html-cache-trap");
  });

  it("stays clean when the edge serves from cache (HIT/STALE/REVALIDATED/UPDATING)", () => {
    for (const status of ["HIT", "STALE", "REVALIDATED", "UPDATING"]) {
      const b = compileBrief(facts({ probe: probe("public, s-maxage=3600", status), cacheRuleCount: 0 }), TEMPLATES);
      expect(kindsOf(b)).not.toContain("html-cache-trap");
    }
  });

  it("stays clean without a public directive even with s-maxage", () => {
    const b = compileBrief(facts({ probe: probe("s-maxage=3600", "DYNAMIC"), cacheRuleCount: 0 }), TEMPLATES);
    expect(kindsOf(b)).not.toContain("html-cache-trap");
  });

  it("stays clean for non-HTML content types", () => {
    const b = compileBrief(facts({ probe: probe("public, s-maxage=3600", "DYNAMIC", "application/json"), cacheRuleCount: 0 }), TEMPLATES);
    expect(kindsOf(b)).not.toContain("html-cache-trap");
  });

  it("stays clean with no probe at all", () => {
    const b = compileBrief(facts({ probe: undefined, cacheRuleCount: 0 }), TEMPLATES);
    expect(kindsOf(b)).not.toContain("html-cache-trap");
    expect(kindsOf(b)).not.toContain("cache-rule-gap");
  });
});

describe("compileBrief dedupe (rule gap vs trap)", () => {
  it("suppresses cache-rule-gap when html-cache-trap fired", () => {
    const b = compileBrief(facts({ probe: { contentType: "text/html", cacheControl: "public, s-maxage=3600", cfCacheStatus: "DYNAMIC" }, cacheRuleCount: 0 }), TEMPLATES);
    expect(kindsOf(b)).toEqual(["html-cache-trap"]);
  });

  it("reports cache-rule-gap (info) for cacheable HTML served from edge with zero rules", () => {
    const b = compileBrief(facts({ probe: { contentType: "text/html", cacheControl: "public, max-age=300", cfCacheStatus: "HIT" }, cacheRuleCount: 0 }), TEMPLATES);
    const f = b.findings.find((x) => x.kind === "cache-rule-gap");
    expect(f?.severity).toBe("info");
    expect(b.actions).toEqual([{ directive: "gap-action" }]);
    expect(b.verify).toEqual(["verify-cache-rule-gap"]);
  });

  it("reports no gap when rules exist", () => {
    const b = compileBrief(facts({ probe: { contentType: "text/html", cacheControl: "public, max-age=300", cfCacheStatus: "HIT" }, cacheRuleCount: 3 }), TEMPLATES);
    expect(b.findings).toEqual([]);
  });
});

describe("compileBrief alerts and quota", () => {
  it("emits one warning per firing alert with value/threshold/unit evidence", () => {
    const b = compileBrief(
      facts({
        firingAlerts: [
          { ts: "2026-10-15T00:00:00.000Z", ruleId: "miss pct/zone-1", condition: "zone-cache-miss-pct", title: "cache miss % · example.com", value: 63.6, threshold: 50, unit: "%", message: "m", sent: true, context: "46 misses of 139 eligible requests" },
          { ts: "2026-10-15T00:00:00.000Z", ruleId: "rows/zone-1", condition: "d1-rows-read", title: "d1 rows read · db", value: 2_000_000_000, threshold: 1_000_000_000, unit: "rows", message: "m", sent: false, context: "" },
        ],
      }),
      TEMPLATES,
    );
    expect(b.findings).toHaveLength(2);
    for (const f of b.findings) {
      expect(f.kind).toBe("alert-firing");
      expect(f.severity).toBe("warning");
    }
    expect(b.findings[0].evidence).toContain("cache miss % · example.com");
    expect(b.findings[0].evidence).toContain("63.6%");
    expect(b.findings[0].evidence).toContain("50%");
    expect(b.findings[0].evidence).toContain("46 misses of 139 eligible requests");
    expect(b.findings[1].evidence).toContain("2000000000 rows");
    expect(b.findings[1].evidence).toContain("1000000000 rows");
    // the alert template contributes once despite two fired alerts (dedupe)
    expect(b.actions).toEqual([{ directive: "inspect the firing alert" }]);
  });

  it("flags quota-pressure per family: >=95 critical, >=80 warning, below silent", () => {
    const b = compileBrief(
      facts({
        quota: {
          families: [
            { family: "page-rules", label: "Page Rules", used: 3, allowance: 3, authorized: true }, // 100 -> critical
            { family: "cache-rules", label: "Cache Rules", used: 9, allowance: 10, authorized: true }, // 90 -> warning
            { family: "redirect-rules", label: "Redirect Rules", used: 5, allowance: 10, authorized: true }, // 50 -> silent
            { family: "waf-custom-rules", label: "WAF Custom Rules", used: 5, allowance: 5, authorized: false }, // unauthorized -> silent
            { family: "config-rules", label: "Configuration Rules", used: 0, allowance: 0, authorized: true }, // no allowance -> silent
          ],
        },
      }),
      TEMPLATES,
    );
    const bySeverity = b.findings.filter((f) => f.kind === "quota-pressure");
    expect(bySeverity).toHaveLength(2);
    expect(bySeverity.find((f) => f.severity === "critical")?.evidence).toContain("Page Rules: 3/3");
    expect(bySeverity.find((f) => f.severity === "warning")?.evidence).toContain("Cache Rules: 9/10");
  });

  it("holds the 80% warning boundary exactly", () => {
    const b = compileBrief(
      facts({ quota: { families: [{ family: "cache-rules", label: "Cache Rules", used: 8, allowance: 10, authorized: true }] } }),
      TEMPLATES,
    );
    expect(b.findings).toHaveLength(1);
    expect(b.findings[0].severity).toBe("warning");
  });

  it("holds the 95% critical boundary exactly", () => {
    const b = compileBrief(
      facts({ quota: { families: [{ family: "cache-rules", label: "Cache Rules", used: 19, allowance: 20, authorized: true }] } }),
      TEMPLATES,
    );
    expect(b.findings).toHaveLength(1);
    expect(b.findings[0].severity).toBe("critical");
  });
});

describe("compileBrief ordering and envelope", () => {
  it("orders findings critical -> warning -> info, kind alphabetical within severity", () => {
    const b = compileBrief(
      facts({
        coverage: { requests: 100_000, cached: 400, hitRatePct: 0.4 }, // critical low-cache-hit
        probe: { contentType: "text/html", cacheControl: "public, max-age=300", cfCacheStatus: "HIT" },
        cacheRuleCount: 0, // info cache-rule-gap
        firingAlerts: [
          { ts: "2026-10-15T00:00:00.000Z", ruleId: "miss pct/zone-1", condition: "zone-cache-miss-pct", title: "miss", value: 63.6, threshold: 50, unit: "%", message: "m", sent: true, context: "" },
        ], // warning alert-firing
        quota: {
          families: [
            { family: "page-rules", label: "Page Rules", used: 3, allowance: 3, authorized: true }, // critical quota-pressure
            { family: "cache-rules", label: "Cache Rules", used: 9, allowance: 10, authorized: true }, // warning quota-pressure
          ],
        },
      }),
      TEMPLATES,
    );
    expect(kindsOf(b)).toEqual(["low-cache-hit", "quota-pressure", "alert-firing", "quota-pressure", "cache-rule-gap"]);
    expect(b.findings.map((f) => f.severity)).toEqual(["critical", "critical", "warning", "warning", "info"]);
  });

  it("is deterministic for identical facts and copies the envelope", () => {
    const f = facts({ coverage: { requests: 50_000, cached: 200, hitRatePct: 0.4 } });
    const a = compileBrief(f, TEMPLATES);
    const b = compileBrief(f, TEMPLATES);
    expect(a).toEqual(b);
    expect(a.scope).toBe("zone");
    expect(a.zone).toBe("example.com");
    expect(a.zoneId).toBe("zone-1");
    expect(a.window).toBe("7d");
  });
});
