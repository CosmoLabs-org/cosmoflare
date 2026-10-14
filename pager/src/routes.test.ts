import { describe, expect, it } from "vitest";
import { parseHash, hrefFor, workerHref, parseWorkerHash, d1Href, parseD1Hash, doHref, parseDoHash, zoneHref, parseZoneHash, ROUTES } from "./routes";

describe("parseHash", () => {
  it("accepts every declared route", () => {
    for (const id of ROUTES) {
      expect(parseHash(`#/${id}`)).toBe(id);
      expect(parseHash(`#/${id}?x=1`)).toBe(id);
    }
  });

  it("treats a hash missing the # prefix as unknown", () => {
    // window.location.hash always carries the #, so "/billing" (no #) is
    // not a navigation target and lands on the overview.
    expect(parseHash("/billing")).toBe("overview");
  });

  it("lands unknown and missing hashes on the overview", () => {
    expect(parseHash("#/nope")).toBe("overview");
    expect(parseHash("")).toBe("overview");
    expect(parseHash("#/")).toBe("overview");
    expect(parseHash("garbage")).toBe("overview");
  });

  it("ignores the query part of a hash", () => {
    expect(parseHash("#/billing?refresh=1")).toBe("billing");
  });
});

describe("hrefFor", () => {
  it("builds the drawer and sidebar hrefs", () => {
    expect(hrefFor("overview")).toBe("#/overview");
    expect(hrefFor("billing")).toBe("#/billing");
  });
});

describe("workerHref / parseWorkerHash (UI-3)", () => {
  it("round-trips a plain name", () => {
    expect(workerHref("api-gateway")).toBe("#/worker/api-gateway");
    expect(parseWorkerHash(workerHref("api-gateway"))).toBe("api-gateway");
  });

  it("round-trips a name needing encoding (UI-3)", () => {
    const name = "my worker (prod)";
    expect(workerHref(name)).toBe("#/worker/my%20worker%20(prod)");
    expect(parseWorkerHash(workerHref(name))).toBe(name);
  });

  it("returns null for non-profile hashes", () => {
    expect(parseWorkerHash("#/workers")).toBeNull();
    expect(parseWorkerHash("#/worker/y")).toBe("y");
    expect(parseWorkerHash("#/overview")).toBeNull();
    expect(parseWorkerHash("")).toBeNull();
  });

  it("rejects an empty name", () => {
    expect(parseWorkerHash("#/worker/")).toBeNull();
  });
});

describe("entity drill-in parsers (profiles program)", () => {
  it("round-trips D1, DO and zone names through their hrefs", () => {
    expect(parseD1Hash(d1Href("sessions-db"))).toBe("sessions-db");
    expect(parseDoHash(doHref("ccs-state-worker"))).toBe("ccs-state-worker");
    expect(parseZoneHash(zoneHref("cosmolabs.org"))).toBe("cosmolabs.org");
  });

  it("round-trips names needing encoding", () => {
    const name = "my db (prod)";
    expect(parseD1Hash(d1Href(name))).toBe(name);
    expect(parseDoHash(doHref(name))).toBe(name);
    expect(parseZoneHash(zoneHref(name))).toBe(name);
  });

  it("returns null for section hashes and empty names", () => {
    for (const parse of [parseD1Hash, parseDoHash, parseZoneHash]) {
      expect(parse("#/d1")).toBeNull();
      expect(parse("#/durable-objects")).toBeNull();
      expect(parse("#/zones")).toBeNull();
      expect(parse("#/overview")).toBeNull();
      expect(parse("")).toBeNull();
    }
    expect(parseD1Hash("#/d1/")).toBeNull();
    expect(parseDoHash("#/do/")).toBeNull();
    expect(parseZoneHash("#/zone/")).toBeNull();
  });

  it("keeps the parsers disjoint — one prefix never matches another", () => {
    expect(parseD1Hash(doHref("x"))).toBeNull();
    expect(parseDoHash(zoneHref("x"))).toBeNull();
    expect(parseZoneHash(workerHref("x"))).toBeNull();
  });
});
