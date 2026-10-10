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
  ringCenterValue,
  ringSecondLine,
  todayDotPos,
  type RingGaugeProps,
} from "../../gauges";

export default function RingGauge(props: RingGaugeProps): React.JSX.Element {
  const size = props.size ?? 120;
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
  const ys = lines.length === 1 ? [size + 12] : [size + 8, size + 20];
  const today = props.expectedPct !== undefined ? todayDotPos(props.expectedPct, cx, cy, r) : null;

  return (
    <svg
      className={`cf-ring-svg ${levelClass}`}
      role="img"
      aria-label={ariaLabelFor(props)}
      viewBox={`0 0 ${size} ${size + (props.sublabel ? 46 : 32)}`}
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

      {/* Center: the projected % big and level-colored, one muted line under
          it (used %, or the size for storage metrics). */}
      <text x={cx} y={cy - 6} textAnchor="middle" dominantBaseline="central" className="cf-ring-value">
        {ringCenterValue(props.projectedPct)}
      </text>
      <text x={cx} y={cy + 12} textAnchor="middle" dominantBaseline="central" className="cf-ring-usedpct">
        {ringSecondLine(props.usedPct, props.usedLineText)}
      </text>

      {/* Label band below the ring: the (up to two-line) product label, then
          the allowance as its own muted line. */}
      <text x={cx} textAnchor="middle" className="cf-ring-label">
        {lines.map((line, i) => (
          <tspan key={i} x={cx} y={String(ys[i])}>
            {line}
          </tspan>
        ))}
      </text>
      {props.sublabel ? (
        <text
          x={cx}
          y={String((lines.length === 1 ? size + 12 : size + 20) + 12)}
          textAnchor="middle"
          className="cf-ring-sub"
        >
          {`of ${props.sublabel}`}
        </text>
      ) : null}
    </svg>
  );
}
