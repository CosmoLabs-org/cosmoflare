import { describe, expect, it } from "vitest";
import { parseHash, hrefFor, ROUTES } from "./routes";

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
