// React port of gauges.ts barGauge (P-04): the same div structure —
// cf-gauge with level/over/flow classes, cf-gauge-head (name + limit badge),
// cf-gauge-bar with the clipping track (solid used fill, lighter projected
// extension, allowance marker), the floating today dot, and the detail text
// row — with the level/scale math imported verbatim from gauges.ts.

import { clampPct, gaugeLevel, type BarGaugeProps } from "../../gauges";

export default function BarGauge(props: BarGaugeProps): React.JSX.Element {
  const { level, overLimit, limitText } = gaugeLevel(props.usedPct, props.projectedPct);
  const isFlow = level === "critical";
  return (
    <div className={`cf-gauge cf-level-${level}${overLimit ? " cf-over" : ""}${isFlow ? " cf-flow" : ""}`}>
      {props.label || limitText ? (
        <div className="cf-gauge-head">
          {props.label ? <span className="cf-gauge-name">{props.label}</span> : null}
          {limitText ? (
            <span className={`cf-gauge-badge ${level === "critical" ? "cf-level-critical-text" : "cf-level-warning-text"}`}>
              {limitText}
            </span>
          ) : null}
        </div>
      ) : null}

      {/* The track clips its fills (overflow:hidden); the today dot floats
          ABOVE the bar so it is never clipped and never reads as a line
          crossing the fill. */}
      <div className="cf-gauge-bar">
        <div className="cf-gauge-track">
          <div className="cf-gauge-fill" style={{ width: `${clampPct(props.usedScalePct)}%` }} />
          {props.projectedScalePct > props.usedScalePct ? (
            <div
              className="cf-gauge-projected"
              style={{
                width: `${clampPct(props.projectedScalePct - props.usedScalePct)}%`,
                insetInlineStart: `${clampPct(props.usedScalePct)}%`,
              }}
            />
          ) : null}
          <span className="cf-gauge-marker" style={{ insetInlineStart: `${clampPct(props.includedScalePct)}%` }} />
        </div>
        {props.expectedPct !== undefined ? (
          <span className="cf-gauge-today" style={{ insetInlineStart: `${clampPct(props.expectedPct)}%` }} />
        ) : null}
      </div>

      <p className="cf-gauge-detail cf-row-detail">{props.detailText}</p>
    </div>
  );
}
