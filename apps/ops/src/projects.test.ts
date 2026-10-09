import { describe, expect, it } from "vitest";
import { projectFor } from "./projects";

describe("projectFor", () => {
  it("maps the documented resource names", () => {
    expect(projectFor("mycarguide-db")).toBe("mycarguide");
    expect(projectFor("mycar.guide")).toBe("mycarguide");
    expect(projectFor("churches.app")).toBe("churches");
    expect(projectFor("churches-api")).toBe("churches");
    expect(projectFor("noblecoffee-bucket")).toBe("noblecoffee");
    expect(projectFor("cosmoflare-portal")).toBe("portal");
    expect(projectFor("MYCARGUIDE-DB")).toBe("mycarguide");
  });

  it("lets an explicit map override win over the heuristics", () => {
    expect(projectFor("churches.app", { "churches.app": "acme" })).toBe("acme");
  });

  it("falls back to a normalised name when nothing matches", () => {
    expect(projectFor("mystery_worker", {})).toBe("mysteryworker");
  });
});
