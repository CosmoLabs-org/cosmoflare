// React port of gauges.ts ringGauge (P-04, docs/planning-mode/
// 2026-10-10-pager-react-rebuild.md): the SAME svg structure — track,
// dashed projected arc, solid used arc, critical-only glow filter, today
// dot, overflow notch, center value + second line, label band tspans, the
// same viewBox math — with the pure arc/label math imported verbatim from
// gauges.ts, never re-implemented. Only the DOM builder changed hands
// (document.createElementNS → JSX) and the module-level filter-id counter
// became a per-instance useId.

import { useId } from "react";
import {
  ariaLabelFor,
  arcLens,
  dashedPattern,
  gaugeLevel,
  labelLines,
  ringSecondLine,
  todayDotPos,
  type RingGaugeProps,
} from "../../gauges";

/** 24px mono value advance (0.6em + no tracking) in viewBox units — the one
 *  width constant the interior squeeze needs; lives beside its only user. */
const VALUE_CHAR_W = 14.4;

export default function RingGauge(props: RingGaugeProps): React.JSX.Element {
  // Ring sizing (operator 2026-10-10): 132 default — the 120 ring let the
  // center percentage crowd the circumference; the bigger radius plus the
  // wider center-text spacing below keeps the numbers clear of the arcs.
  const size = props.size ?? 132;
  const stroke = Math.max(6, Math.round(size / 12));
  const cx = size / 2;
  const cy = size / 2;
  const r = cx - stroke / 2 - 4; // 4px headroom so notch/tick strokes never clip
  const geom = arcLens(props.usedPct, props.projectedPct, r);
  const { level, overLimit } = gaugeLevel(props.usedPct, props.projectedPct);
  // cf-flow drives the traveling dash pattern; only critical/over-limit rings.
  const levelClass = overLimit
    ? "cf-level-critical cf-over cf-flow"
    : `cf-level-${level}${level === "critical" ? " cf-flow" : ""}`;
  // Unique filter id per gauge instance — duplicate ids across one document's
  // SVGs would make every url(#…) reference resolve to the first filter.
  // (gauges.ts uses a module counter; React gives one per instance via useId,
  // stripped of the colons so the url(#…) fragment reference stays valid.)
  const filterId = `cf-glow-${useId().replace(/[^a-zA-Z0-9_-]/g, "")}`;

  const usedLenDash = `${geom.usedLen} ${geom.circumference}`;
  const circleStyle = (extra: Record<string, string> = {}): React.CSSProperties =>
    ({ "--ring-c": String(geom.circumference), ...extra }) as React.CSSProperties;

  const lines = labelLines(props.label);
  const today = props.expectedPct !== undefined ? todayDotPos(props.expectedPct, cx, cy, r) : null;
  // The glow head rides the used arc's end angle — same polar helper as the
  // today dot (todayDotPos is generic pct→point math), driven by usedPct.
  // Zero usage renders no head: an empty arc must not paint a start dot.
  const head = props.usedPct > 0 ? todayDotPos(props.usedPct, cx, cy, r) : null;
  // Interior: the measured value with units plus one always-fitting "USED"
  // line; the used % and the allowance are band lines below the ring.
  const second = ringSecondLine(props.usedPct);
  const innerD = 2 * (r - stroke);
  // ONE width rule (mono advance ∝ font size): a value wider than the ring's
  // inner opening gets an exact SVG textLength squeeze. Size-aware — scales
  // with the ring, unlike the char-count threshold it replaces.
  const valueSqueeze = props.centerValueText.length * VALUE_CHAR_W > innerD;
  // Band geometry: label line(s) → used % line → allowance, 13-unit pitch;
  // the tail grows with the lines actually present.
  const bandStart = lines.length === 1 ? size + 15 : size + 12;
  const bandYs = lines.length === 1 ? [bandStart] : [bandStart, bandStart + 13];
  const capY = bandYs[bandYs.length - 1] + 13;
  const subY = capY + 13;
  const tail = (props.sublabel ? subY : capY) + 7;

  return (
    <svg
      className={`cf-ring-svg ${levelClass}`}
      role="img"
      aria-label={ariaLabelFor(props)}
      viewBox={`0 0 ${size} ${size + tail}`}
    >
      {/* Track (full circle, neutral). */}
      <circle
        cx={cx} cy={cy} r={r}
        transform={`rotate(-90 ${cx} ${cy})`}
        className="cf-ring-track"
        style={circleStyle()}
      />

      {/* Projected arc: dashed, lighter — continues from the used arc's end
          via its dashoffset (--proj-off). */}
      <circle
        cx={cx} cy={cy} r={r}
        transform={`rotate(-90 ${cx} ${cy})`}
        strokeDasharray={dashedPattern(Math.max(0, geom.projectedLen - geom.usedLen), geom.circumference)}
        className="cf-ring-projected"
        style={circleStyle({ "--proj-off": String(-geom.usedLen) })}
      />

      {/* Critical/over-limit only: blurred copy of the used arc underneath
          (the glow must paint BEFORE the used arc, as in the vanilla order). */}
      {level === "critical" ? (
        <circle
          cx={cx} cy={cy} r={r}
          transform={`rotate(-90 ${cx} ${cy})`}
          strokeDasharray={usedLenDash}
          className="cf-ring-glow"
          filter={`url(#${filterId})`}
          style={circleStyle({ "--arc-len": String(geom.usedLen) })}
        />
      ) : null}

      {/* Used arc: solid, fills from 0 via the dasharray keyframes. */}
      <circle
        cx={cx} cy={cy} r={r}
        transform={`rotate(-90 ${cx} ${cy})`}
        strokeDasharray={usedLenDash}
        className="cf-ring-used cf-anim-used"
        style={circleStyle({ "--arc-len": String(geom.usedLen) })}
      />

      {/* Glow head: the leading point of the used arc, lit. */}
      {head !== null ? <circle cx={head.x} cy={head.y} r="4" className="cf-ring-head" /> : null}

      {/* Glow filter definition (appended last, as the vanilla builder does). */}
      {level === "critical" ? (
        <defs>
          <filter id={filterId} x="-30%" y="-30%" width="160%" height="160%">
            <feGaussianBlur stdDeviation="3" />
          </filter>
        </defs>
      ) : null}

      {/* Expected-to-date today dot: a 3px dot ON the track at the angle. */}
      {today !== null ? (
        <circle cx={today.x} cy={today.y} r="3" className="cf-ring-today" />
      ) : null}

      {/* Overflow notch: a short red line crossing the ring at 12 o'clock. */}
      {geom.overflow ? (
        <line
          x1={cx}
          y1={cy - r - stroke / 2 - 2}
          x2={cx}
          y2={cy - r + stroke / 2 + 2}
          className="cf-ring-notch"
        />
      ) : null}

      {/* Interior: the measured value with units is the hero; one
          always-fitting "USED" line under it. */}
      <text
        x={cx}
        y={cy - 10}
        textAnchor="middle"
        dominantBaseline="central"
        className="cf-ring-value"
        textLength={valueSqueeze ? innerD : undefined}
        lengthAdjust={valueSqueeze ? "spacingAndGlyphs" : undefined}
      >
        {props.centerValueText}
      </text>
      <text x={cx} y={cy + 13} textAnchor="middle" dominantBaseline="central" className="cf-ring-usedpct">
        USED
      </text>

      {/* Label band below the ring: product label, used % so far, allowance —
          the band holds what the interior doesn't. */}
      <text x={cx} textAnchor="middle" className="cf-ring-label">
        {bandYs.map((y, i) => (
          <tspan key={i} x={cx} y={String(y)}>
            {lines[i]}
          </tspan>
        ))}
      </text>
      <text x={cx} y={String(capY)} textAnchor="middle" className="cf-ring-usedpct">
        {second}
      </text>
      {props.sublabel ? (
        <text x={cx} y={String(subY)} textAnchor="middle" className="cf-ring-sub">
          {`of ${props.sublabel}`}
        </text>
      ) : null}
    </svg>
  );
}
