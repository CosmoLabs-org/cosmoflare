import { describe, expect, it } from "vitest";
import { collectAttachments } from "./attachments";

// Full upstream stub: REST lists (pages projects, workers domains, zones)
// go through rest() (paginated, per_page=50); the per-project domain and
// per-zone dns_records probes go through fetchRetry directly. All 200 OK,
// so the retry policy never sleeps in these tests.
function stubFetch(routes: { match: (url: string) => boolean; body: unknown; status?: number }[]): void {
  globalThis.fetch = (async (input: RequestInfo | URL) => {
    const url = String(input);
    const route = routes.find((r) => r.match(url));
    if (!route) throw new Error(`unexpected fetch: ${url}`);
    return new Response(JSON.stringify(route.body), { status: route.status ?? 200 });
  }) as typeof fetch;
}

const ok = (result: unknown, resultInfo?: Record<string, number>) => ({
  success: true,
  result,
  ...(resultInfo ? { result_info: resultInfo } : {}),
  ...(resultInfo ? {} : { result_info: { total_pages: 1 } }),
});

describe("collectAttachments (FEAT-pEWA6N6 phase 1)", () => {
  it("collects Pages project domains, Workers custom domains, and zone record counts", async () => {
    const urls: string[] = [];
    const real = globalThis.fetch;
    stubFetch([
      { match: (u) => { urls.push(u); return u.includes("/pages/projects/noble-web/domains"); }, body: ok([
        { name: "staging.noble.coffee", status: "active", created_on: "2026-01-05T00:00:00Z", validation_data: { status: "Active", ssl: { status: "Active" } } },
        { name: "noble.coffee", status: "pending" },
      ]) },
      { match: (u) => u.includes("/pages/projects?"), body: ok([{ name: "noble-web" }]) },
      { match: (u) => u.includes("/workers/domains"), body: ok([{ hostname: "api.noble.coffee", service: "noble-api", environment: "production" }]) },
      { match: (u) => u.includes("/zones?account.id="), body: ok([{ id: "z1", name: "noble.coffee", status: "active" }]) },
      { match: (u) => u.includes("/dns_records"), body: ok([{}], { total_count: 42 }) },
    ]);
    try {
      const got = await collectAttachments("acct", "tok", new Date("2026-10-10T00:00:00Z"));
      expect(got.generatedAt).toBe("2026-10-10T00:00:00.000Z");
      expect(got.errors).toEqual([]);
      expect(got.pages).toEqual([{
        project: "noble-web",
        domains: [
          { hostname: "staging.noble.coffee", status: "active", ssl: "Active", verification: "Active" },
          { hostname: "noble.coffee", status: "pending", ssl: null, verification: null },
        ],
      }]);
      expect(got.workers).toEqual([{ hostname: "api.noble.coffee", service: "noble-api", environment: "production" }]);
      expect(got.zones).toEqual([{ id: "z1", name: "noble.coffee", status: "active", recordCount: 42 }]);
      // The record-count probe asks for one record only; the total comes
      // from result_info, not the (truncated) result array.
      expect(urls.filter((u) => u.includes("/dns_records"))).toEqual(["https://api.cloudflare.com/client/v4/zones/z1/dns_records?per_page=1"]);
    } finally {
      globalThis.fetch = real;
    }
  });

  it("soft-fails each sub-fetch into errors without blanking the rest", async () => {
    const real = globalThis.fetch;
    stubFetch([
      // Pages list itself fails (success:false, HTTP 200 → no retry sleeps).
      { match: (u) => u.includes("/pages/projects"), body: { success: false, errors: [{ message: "pages.access_denied" }] } },
      { match: (u) => u.includes("/workers/domains"), body: { success: false, errors: [{ message: "workers.api.domain_not_found" }] } },
      { match: (u) => u.includes("/zones?account.id="), body: ok([{ id: "z1", name: "noble.coffee", status: "active" }]) },
      { match: (u) => u.includes("/dns_records"), body: { success: false, errors: [{ message: "dns records denied" }] } },
    ]);
    try {
      const got = await collectAttachments("acct", "tok");
      expect(got.pages).toEqual([]);
      expect(got.workers).toEqual([]);
      // The zone survives its failed count probe with a null count.
      expect(got.zones).toEqual([{ id: "z1", name: "noble.coffee", status: "active", recordCount: null }]);
      expect(got.errors).toEqual([
        "pages projects: pages.access_denied",
        "workers domains: workers.api.domain_not_found",
        "zone noble.coffee: dns records denied",
      ]);
    } finally {
      globalThis.fetch = real;
    }
  });

  it("a per-project domains failure still lists the project with an empty domain list", async () => {
    const real = globalThis.fetch;
    stubFetch([
      { match: (u) => u.includes("/pages/projects/noble-web/domains"), body: { success: false, errors: [{ message: "forbidden" }] } },
      { match: (u) => u.includes("/pages/projects?"), body: ok([{ name: "noble-web" }, { name: "baristabase" }]) },
      { match: (u) => u.includes("/pages/projects/baristabase/domains"), body: ok([{ name: "baristabase.dev", status: "active", validation_data: { status: "Active" } }]) },
      { match: (u) => u.includes("/workers/domains"), body: ok([]) },
      { match: (u) => u.includes("/zones?account.id="), body: ok([]) },
    ]);
    try {
      const got = await collectAttachments("acct", "tok");
      expect(got.pages).toEqual([
        { project: "noble-web", domains: [] },
        { project: "baristabase", domains: [{ hostname: "baristabase.dev", status: "active", ssl: null, verification: "Active" }] },
      ]);
      expect(got.errors).toEqual(["pages/noble-web: forbidden"]);
    } finally {
      globalThis.fetch = real;
    }
  });
});
