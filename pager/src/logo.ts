/**
 * CosmoLabs Ops · Cosmoflare logo mark (FEAT-052, v2).
 *
 * Hand-written SVG on a 32x32 grid. Two primitives:
 *  - a bold "C" for CosmoLabs: a ~260° arc, center (16,16), radius 10,
 *    stroke-width 5 (gap spans the ±50° sector on the right so the signal
 *    breathes — endpoints (22.43,8.34) and (22.43,23.66));
 *  - a heartbeat-style pulse spanning the C's opening vertically:
 *    M 17.5 16 L 21.5 16 L 25 12.75 L 26 19.75 L 27.5 15.5 L 28 15.5,
 *    stroke-width 2.75 — the "ops signal" the C monitors. The pulse is the
 *    hero: its spike and trough ride the opening so it reads "monitoring"
 *    even at 16px.
 *
 * Color contract: both strokes are hard-coded — brand amber #f59e0b for the
 * C, brand sky #7dd3fc for the pulse — because `.cf-brand-mark` sets no CSS
 * color, so `currentColor` would inherit the heading text color instead of
 * the brand amber. The icon SVGs carry the same two hard-coded hex values.
 *
 * Legibility: the C's stroke (5 units) and the pulse (2.75 units) both clear
 * the 2.6-unit floor on the 32 grid; the pulse's spike (25,12.75) and trough
 * (26,19.75) sit ≥1.2 units from the C's round caps (cap disc r 2.5 + pulse
 * half-stroke 1.375) so amber and sky never merge, even at 16px.
 *
 * Sizing: the svg element is full-bleed (width/height 100%,
 * preserveAspectRatio="xMidYMid meet") and `.cf-brand-mark` in styles.css
 * gives it its box (28px square, aspect-ratio 1), so the mark fills any
 * container that sizes the wrapper.
 *
 * `idSuffix` appends to any <defs> ids; today's mark has none, but the
 * signature stays `logoMark(idSuffix = "")` for API stability.
 */
export function logoMark(idSuffix = ""): string {
  void idSuffix;
  return `<svg width="100%" height="100%" viewBox="0 0 32 32" preserveAspectRatio="xMidYMid meet" aria-hidden="true" focusable="false">
  <path d="M 22.43 8.34 A 10 10 0 1 0 22.43 23.66" fill="none" stroke="#f59e0b" stroke-width="5" stroke-linecap="round"/>
  <path d="M 17.5 16 L 21.5 16 L 25 12.75 L 26 19.75 L 27.5 15.5 L 28 15.5" fill="none" stroke="#7dd3fc" stroke-width="2.75" stroke-linecap="round" stroke-linejoin="round"/>
</svg>`;
}
