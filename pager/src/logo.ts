/**
 * CosmoLabs Ops · Cosmoflare logo mark (FEAT-052).
 *
 * Hand-written SVG on a 32x32 grid. Two primitives:
 *  - a bold "C" for CosmoLabs: a 270° arc, center (16,16), radius 10,
 *    stroke-width 4.5 (gap spans the ±45° sector on the right);
 *  - an upward pulse caret in the C's open end: M 20.5 18.5 L 23.5 14.5 L 26.5 18.5,
 *    stroke-width 3 — the "ops signal" the C monitors.
 *
 * Color contract: both strokes are hard-coded — brand amber #f59e0b for the
 * C, brand sky #7dd3fc for the pulse — because `.cf-brand-mark` sets no CSS
 * color, so `currentColor` would inherit the heading text color instead of
 * the brand amber. The icon SVGs carry the same two hard-coded hex values.
 *
 * Legibility: the C's stroke (4.5 units) and the pulse (3 units) both clear
 * the 3-unit floor on the 32 grid; the pulse sits ≥1.2 units from the C's
 * rounded caps so amber and sky never merge, even at 16px.
 *
 * `idSuffix` appends to any <defs> ids; today's mark has none, but the
 * signature stays `logoMark(idSuffix = "")` for API stability.
 */
export function logoMark(idSuffix = ""): string {
  void idSuffix;
  return `<svg width="24" height="24" viewBox="0 0 32 32" aria-hidden="true" focusable="false">
  <path d="M 23.07 9.07 A 10 10 0 1 0 23.07 23.07" fill="none" stroke="#f59e0b" stroke-width="4.5" stroke-linecap="round"/>
  <path d="M 20.5 18.5 L 23.5 14.5 L 26.5 18.5" fill="none" stroke="#7dd3fc" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>
</svg>`;
}
