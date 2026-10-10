import { describe, expect, it } from "vitest";
import { parseHash, hrefFor, workerHref, parseWorkerHash, ROUTES } from "./routes";

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
