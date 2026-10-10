import { describe, expect, it, vi } from "vitest";
import { collectDomains, fetchDomainsList, type DomainRecord } from "./domains";

const ZONES = [
  { id: "z1", name: "cosmolabs.org", status: "active", paused: false, expires_at: "2027-01-15", type: "full", plan: { name: "Pro" }, development_mode: 0, created_on: "2026-05-15T00:00:00Z", modified_on: "2026-06-19T00:00:00Z", name_servers: ["sage.ns.cloudflare.com", "samara.ns.cloudflare.com"] },
  { id: "z2", name: "external.example", status: "active", paused: false, expires_at: null, type: "partial", development_mode: 1, created_on: "2026-01-01T00:00:00Z", modified_on: "2026-02-01T00:00:00Z" },
  { id: "z3", name: "paused.example", status: "active", paused: true, expires_at: "2026-11-02", type: "full", created_on: "2026-03-01T00:00:00Z", modified_on: "2026-03-01T00:00:00Z", name_servers: [] },
];

vi.mock("./summary", () => ({
  rest: vi.fn(async () => ZONES),
}));

describe("fetchDomainsList", () => {
  it("keeps status, expiry, and the detail-sheet fields", async () => {
    const got = await fetchDomainsList("tok", "acct");
    expect(got[0]).toEqual<DomainRecord>({
      id: "z1", name: "cosmolabs.org", status: "active", paused: false, expiresAt: "2027-01-15",
      type: "full", plan: "Pro", developmentMode: false,
      createdOn: "2026-05-15T00:00:00Z", modifiedOn: "2026-06-19T00:00:00Z",
      nameServers: ["sage.ns.cloudflare.com", "samara.ns.cloudflare.com"],
    });
    expect(got[1]?.type).toBe("partial");
    expect(got[1]?.developmentMode).toBe(true);
    expect(got[1]?.nameServers).toEqual([]);
    expect(got[2]?.plan).toBe("");
  });
});

describe("collectDomains", () => {
  it("wraps the list with generatedAt and no errors", async () => {
    const got = await collectDomains("acct", "tok", new Date("2026-10-10T00:00:00Z"));
    expect(got.errors).toEqual([]);
    expect(got.domains).toHaveLength(3);
    expect(got.generatedAt).toBe("2026-10-10T00:00:00.000Z");
  });
  it("a list failure becomes an errors entry, never a throw", async () => {
    const { rest } = await import("./summary");
    vi.mocked(rest).mockRejectedValueOnce(new Error("HTTP 500"));
    const got = await collectDomains("acct", "tok");
    expect(got.domains).toEqual([]);
    expect(got.errors).toEqual(["HTTP 500"]);
  });
});
