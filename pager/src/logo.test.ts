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

  it("renders full-bleed (100% width/height, meet), decorative", () => {
    const svg = logoMark();
    expect(svg).toContain('width="100%"');
    expect(svg).toContain('height="100%"');
    expect(svg).toContain('preserveAspectRatio="xMidYMid meet"');
    expect(svg).toContain('aria-hidden="true"');
    expect(svg).not.toMatch(/width="\d+px"/);
  });

  it("strokes at v2 weights: 5-unit C, 2.75-unit pulse (>= 2.6 floor)", () => {
    const svg = logoMark();
    expect(svg).toContain('stroke-width="5"');
    expect(svg).toContain('stroke-width="2.75"');
  });

  it("keeps the v2 geometry: wide-gap C arc and heartbeat pulse path", () => {
    const svg = logoMark();
    expect(svg).toContain('d="M 22.43 8.34 A 10 10 0 1 0 22.43 23.66"');
    expect(svg).toContain(
      'd="M 17.5 16 L 21.5 16 L 25 12.75 L 26 19.75 L 27.5 15.5 L 28 15.5"',
    );
  });
});
