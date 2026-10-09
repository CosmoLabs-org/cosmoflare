import { describe, expect, it } from "vitest";
import { logoMark } from "./logo";

describe("logoMark", () => {
  it("returns an SVG string with the 32x32 viewBox", () => {
    expect(logoMark()).toContain('viewBox="0 0 32 32"');
  });

  it("contains no <text> node", () => {
    expect(logoMark()).not.toContain("<text");
  });

  it("defaults idSuffix to an empty string without changing output", () => {
    expect(logoMark()).toBe(logoMark(""));
  });

  it("accepts an idSuffix without changing the mark (no defs ids yet)", () => {
    expect(logoMark("brand")).toBe(logoMark());
  });

  it("hard-codes the brand colors: amber C, sky pulse", () => {
    const svg = logoMark();
    expect(svg).toContain('stroke="#f59e0b"');
    expect(svg).toContain('stroke="#7dd3fc"');
  });

  it("renders 24px square, decorative", () => {
    const svg = logoMark();
    expect(svg).toContain('width="24"');
    expect(svg).toContain('height="24"');
    expect(svg).toContain('aria-hidden="true"');
  });
});
