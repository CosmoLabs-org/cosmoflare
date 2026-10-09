/**
 * Cosmoflare logo mark (FEAT-055), adapted from the pager PWA mark
 * (pager/src/logo.ts, FEAT-052) as an inline SVG React component.
 *
 * Two primitives on a 32x32 grid:
 *  - a bold "C" for CosmoLabs: a 270° arc, center (16,16), radius 10,
 *    stroke-width 4.5 (gap spans the ±45° sector on the right);
 *  - an upward pulse caret in the C's open end — the "ops signal".
 *
 * Color contract is unchanged from the pager mark: both strokes hard-coded
 * (brand amber for the C, brand sky for the pulse) because `currentColor`
 * would inherit the heading text color instead of the brand hues.
 */

interface LogoProps {
  /** Rendered box size in px (the viewBox stays 32x32). */
  size?: number;
  className?: string;
}

export function Logo({ size = 24, className }: LogoProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      aria-hidden="true"
      focusable="false"
      className={className}
    >
      <path
        d="M 23.07 9.07 A 10 10 0 1 0 23.07 23.07"
        fill="none"
        stroke="#f59e0b"
        strokeWidth={4.5}
        strokeLinecap="round"
      />
      <path
        d="M 20.5 18.5 L 23.5 14.5 L 26.5 18.5"
        fill="none"
        stroke="#7dd3fc"
        strokeWidth={3}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
