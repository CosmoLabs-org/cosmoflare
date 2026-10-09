import { describe, expect, it, vi } from "vitest";
import { collectDomains, fetchDomainsList, type DomainRecord } from "./domains";

const ZONES = [
  { id: "z1", name: "cosmolabs.org", status: "active", paused: false, expires_at: "2027-01-15" },
  { id: "z2", name: "external.example", status: "active", paused: false, expires_at: null },
  { id: "z3", name: "paused.example", status: "active", paused: true, expires_at: "2026-11-02" },
];

vi.mock("./summary", () => ({
  rest: vi.fn(async () => ZONES),
}));

describe("fetchDomainsList", () => {
  it("keeps every status, paused flag, and the registrar expiry", async () => {
    const got = await fetchDomainsList("tok", "acct");
    expect(got).toEqual<DomainRecord[]>([
      { id: "z1", name: "cosmolabs.org", status: "active", paused: false, expiresAt: "2027-01-15" },
      { id: "z2", name: "external.example", status: "active", paused: false, expiresAt: null },
      { id: "z3", name: "paused.example", status: "active", paused: true, expiresAt: "2026-11-02" },
    ]);
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
