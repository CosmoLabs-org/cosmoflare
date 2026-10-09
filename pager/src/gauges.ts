// Usage gauges for the Workers Paid allowances (FEAT-pDQ8JET): the level
// bands, arc math and sorting here are pure and DOM-free (tested); the SVG
// builders below need a DOM and are exercised in the browser only. No gauge
// libraries — hand-built SVG keeps the geometry ours to test.

// ---- Levels ----

export type Level = "ok" | "warning" | "critical";

/** Projected % of allowance bands: warning at 75%, critical at 90%. */
export function levelFor(pctOfAllowance: number): Level {
  if (pctOfAllowance >= 90) return "critical";
  if (pctOfAllowance >= 75) return "warning";
  return "ok";
}

/** "Over limit" once used or projected passes 100%, else "Near limit" from warning on. */
export function limitLabel(maxPct: number): string {
  return maxPct > 100 ? "Over limit" : "Near limit";
}

/**
 * The one place a gauge decides how red it is: the level bands run on the
 * projected % of allowance; passing 100% anywhere (used or projected) is
 * critical + "Over limit" regardless of the projection.
 */
export function gaugeLevel(
  usedPct: number,
  projectedPct: number,
): { level: Level; overLimit: boolean; limitText: string | null } {
  const overLimit = usedPct > 100 || projectedPct > 100;
  const level: Level = overLimit ? "critical" : levelFor(projectedPct);
  return {
    level,
    overLimit,
    limitText: level === "ok" ? null : limitLabel(Math.max(usedPct, projectedPct)),
  };
}

// ---- Arc math (pure, tested) ----

export interface ArcGeom {
  /** Full ring circumference at radius r. */
  circumference: number;
  /** Used-arc length, capped at the full ring. */
  usedLen: number;
  /** Projected-arc length, capped at the full ring. */
  projectedLen: number;
  /** True when used or projected passes 100% — the ring shows an overflow notch. */
  overflow: boolean;
}

/** Arc lengths for a ring of radius r; arcs cap at 100% with an overflow flag. */
export function arcLens(usedPct: number, projectedPct: number, r: number): ArcGeom {
  const circumference = 2 * Math.PI * r;
  const cap = (pct: number): number => Math.min(100, Math.max(0, pct));
  return {
    circumference,
    usedLen: (circumference * cap(usedPct)) / 100,
    projectedLen: (circumference * cap(projectedPct)) / 100,
    overflow: usedPct > 100 || projectedPct > 100,
  };
}

/** Clamp a percentage into 0..100 (track positions, ticks, markers). */
export function clampPct(pct: number): number {
  return Math.min(100, Math.max(0, pct));
}

/**
 * Dash pattern covering exactly `len` of arc in dash/gap steps (dashes first),
 * then one gap for the rest of the ring — the dashes stop where the arc does.
 */
export function dashedPattern(len: number, circumference: number, dash = 3, gap = 5): string {
  if (len <= 0) return `0 ${circumference}`;
  const parts: string[] = [];
  let rem = len;
  while (rem > 0) {
    const d = Math.min(dash, rem);
    parts.push(String(d));
    rem -= d;
    if (rem > 0) {
      const g = Math.min(gap, rem);
      parts.push(String(g));
      rem -= g;
    }
  }
  parts.push(String(Math.max(0, circumference - len)));
  return parts.join(" ");
}

/**
 * "4" / "7.2" / "185" — one decimal below 100, integer at 100+ so the
 * center value never wraps. The % sign is added by the callers.
 */
export function fmtGaugePct(v: number): string {
  const oneDec = Math.round(v * 10) / 10;
  const n = v >= 100 ? Math.round(oneDec) : oneDec;
  return String(n);
}

/** "Workers requests: 4% used, projected 7% of 10M requests" (the task's example). */
export function ariaLabelFor(p: { label: string; usedPct: number; projectedPct: number; sublabel?: string }): string {
  const tail = p.sublabel ? ` of ${p.sublabel}` : " of allowance";
  return `${p.label}: ${fmtGaugePct(p.usedPct)}% used, projected ${fmtGaugePct(p.projectedPct)}%${tail}`;
}

/**
 * Proper product names for the ring/bar labels — the raw "product + metric"
 * join reads "Workers requests" for CPU time ("Workers cpu time"), so the
 * billing ids map to the names Cloudflare's own docs use. Unknown ids fall
 * back to "product metric" so a new Workers Paid line item never renders "".
 */
export function productLabel(id: string, product: string, metric: string): string {
  const known = PRODUCT_LABELS[id];
  return known ?? `${product} ${metric.toLowerCase()}`;
}

const PRODUCT_LABELS: Record<string, string> = {
  "workers.requests": "Workers requests",
  "workers.cpu_time": "Workers CPU time",
  "r2.class_a": "R2 Class A ops",
  "r2.class_b": "R2 Class B ops",
  "durable_objects.requests": "Durable Objects requests",
  "durable_objects.duration": "Durable Objects duration",
  "kv.reads": "KV reads",
  "kv.writes": "KV writes",
  "kv.storage": "KV storage",
  "d1.rows_read": "D1 rows read",
  "d1.rows_written": "D1 rows written",
  "d1.storage": "D1 storage",
  "r2.storage": "R2 storage",
};

/**
 * Rings show only the 8 closest to their limit (worst-first); the rest sit
 * behind a "Show all N" toggle. The cap applies to the already-sorted list,
 * so the top-8 cut is deterministic.
 */
export function capTopRings<T>(sorted: T[], cap = 8): { shown: T[]; hiddenCount: number } {
  return {
    shown: sorted.slice(0, cap),
    hiddenCount: Math.max(0, sorted.length - cap),
  };
}

/** "106%" / "7%" — the ring's big center number is the projected share. */
export function ringCenterValue(projectedPct: number): string {
  return `${fmtGaugePct(projectedPct)}%`;
}

/** "30.7% used so far" — the muted second line under the projected value. */
export function ringUsedLine(usedPct: number): string {
  return `${fmtGaugePct(usedPct)}% used so far`;
}

/**
 * Ring labels never truncate: split on word boundaries into at most two
 * balanced lines (the shorter of the two wins), so "Durable Objects
 * duration" renders on two centered rows inside the label band.
 */
export function labelLines(label: string, maxChars = 16): string[] {
  if (label.length <= maxChars) return [label];
  const words = label.split(" ");
  if (words.length === 1) return [label];
  let best: string[] = [label];
  let bestLen = Infinity;
  for (let i = 1; i < words.length; i++) {
    const a = words.slice(0, i).join(" ");
    const b = words.slice(i).join(" ");
    const len = Math.max(a.length, b.length);
    if (len < bestLen) {
      best = [a, b];
      bestLen = len;
    }
  }
  return best;
}

/**
 * Today-dot position: ON the track (not a line crossing it), at the
 * expected-to-date angle — 0% at 12 o'clock, clockwise, matching the arcs.
 */
export function todayDotPos(expectedPct: number, cx: number, cy: number, r: number): { x: number; y: number } {
  const angle = (clampPct(expectedPct) / 100) * 360 - 90;
  const rad = (angle * Math.PI) / 180;
  return { x: cx + r * Math.cos(rad), y: cy + r * Math.sin(rad) };
}

/**
 * Rings sort worst-first: projected % of allowance desc, ties broken by id
 * so the order is deterministic when two products pace identically.
 */
export function sortByProjectedDesc<T extends { id: string; projected: number; included: number }>(items: T[]): T[] {
  const pct = (p: { projected: number; included: number }): number => (p.included > 0 ? p.projected / p.included : 0);
  return [...items].sort((a, b) => pct(b) - pct(a) || a.id.localeCompare(b.id));
}

// ---- SVG/DOM builders (need a DOM; tests stay DOM-free on the helpers) ----

const SVG_NS = "http://www.w3.org/2000/svg";

// Unique filter id per gauge instance — duplicate ids across one document's
// SVGs would make every url(#…) reference resolve to the first filter.
let filterSeq = 0;

export interface RingGaugeProps {
  /** Used % of the allowance (uncapped — may exceed 100). */
  usedPct: number;
  /** Projected % of the allowance (uncapped). */
  projectedPct: number;
  /** "Workers requests" — below the ring and inside the aria-label. */
  label: string;
  /** "10M requests" — under the center value and inside the aria-label. */
  sublabel?: string;
  /** Square ring size in px; default 120. */
  size?: number;
  /** Expected-to-date position 0..100 (period.day/period.days) — 3px dot on the ring. */
  expectedPct?: number;
}

export function ringGauge(props: RingGaugeProps): SVGSVGElement {
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

  const svg = document.createElementNS(SVG_NS, "svg");
  // 26px band below the ring holds the (up to two-line) label text.
  svg.setAttribute("viewBox", `0 0 ${size} ${size + 26}`);
  svg.classList.add("cf-ring-svg", ...levelClass.split(" "));
  svg.setAttribute("role", "img");
  svg.setAttribute("aria-label", ariaLabelFor(props));

  const circle = (cls: string, dasharray: string | null): SVGCircleElement => {
    const c = document.createElementNS(SVG_NS, "circle");
    c.setAttribute("cx", String(cx));
    c.setAttribute("cy", String(cy));
    c.setAttribute("r", String(r));
    c.setAttribute("transform", `rotate(-90 ${cx} ${cy})`); // 0% at 12 o'clock
    if (dasharray) c.setAttribute("stroke-dasharray", dasharray);
    // Static final state; the fill-in animation (CSS) interpolates the
    // dasharray from 0 via these inline custom properties.
    c.style.setProperty("--ring-c", String(geom.circumference));
    c.classList.add(cls);
    return c;
  };

  // Track (full circle, neutral).
  svg.append(circle("cf-ring-track", null));

  // Projected arc: dashed, lighter — continues from the used arc's end via
  // its dashoffset (kept in a custom property so the CSS flow animation can
  // travel the dash pattern without losing the start position). When the
  // used arc caps at 100% the projected arc collapses (len 0) — correct.
  const projected = circle("cf-ring-projected", dashedPattern(Math.max(0, geom.projectedLen - geom.usedLen), geom.circumference));
  projected.style.setProperty("--proj-off", String(-geom.usedLen));
  svg.append(projected);

  // Used arc: solid, fills from 0 via the dasharray keyframes.
  const used = circle("cf-ring-used", `${geom.usedLen} ${geom.circumference}`);
  used.style.setProperty("--arc-len", String(geom.usedLen));
  used.classList.add("cf-anim-used");
  svg.append(used);

  // Critical/over-limit only: blurred copy of the used arc underneath, the
  // soft pulsing red glow. ok/warning rings carry no filter at all.
  if (level === "critical") {
    const glow = circle("cf-ring-glow", `${geom.usedLen} ${geom.circumference}`);
    svg.insertBefore(glow, used);
    filterSeq += 1;
    const filterId = `cf-glow-${filterSeq}`;
    const blur = document.createElementNS(SVG_NS, "feGaussianBlur");
    blur.setAttribute("stdDeviation", "3");
    const filter = document.createElementNS(SVG_NS, "filter");
    filter.setAttribute("id", filterId);
    filter.setAttribute("x", "-30%");
    filter.setAttribute("y", "-30%");
    filter.setAttribute("width", "160%");
    filter.setAttribute("height", "160%");
    filter.append(blur);
    const defs = document.createElementNS(SVG_NS, "defs");
    defs.append(filter);
    svg.append(defs);
    glow.setAttribute("filter", `url(#${filterId})`);
  }

  // Expected-to-date today dot (period.day/period.days around the ring):
  // a 3px dot ON the track at the angle — not a line crossing the ring.
  if (props.expectedPct !== undefined) {
    const { x, y } = todayDotPos(props.expectedPct, cx, cy, r);
    const dot = document.createElementNS(SVG_NS, "circle");
    dot.setAttribute("cx", String(x));
    dot.setAttribute("cy", String(y));
    dot.setAttribute("r", "3");
    dot.classList.add("cf-ring-today");
    svg.append(dot);
  }

  // Overflow notch: a short red line crossing the ring at the 100% position
  // (12 o'clock) — the arc caps at a full circle and the notch says "past".
  if (geom.overflow) {
    const notch = document.createElementNS(SVG_NS, "line");
    notch.setAttribute("x1", String(cx));
    notch.setAttribute("y1", String(cy - r - stroke / 2 - 2));
    notch.setAttribute("x2", String(cx));
    notch.setAttribute("y2", String(cy - r + stroke / 2 + 2));
    notch.classList.add("cf-ring-notch");
    svg.append(notch);
  }

  // Center: projected % (big, level-colored), "N% used so far" and the
  // allowance ("of 25.0B rows") muted under it — the projection is the
  // headline; the used share is context, the allowance is the reference.
  const value = document.createElementNS(SVG_NS, "text");
  value.setAttribute("x", String(cx));
  value.setAttribute("y", String(cy - 12));
  value.setAttribute("text-anchor", "middle");
  value.setAttribute("dominant-baseline", "central");
  value.classList.add("cf-ring-value");
  value.textContent = ringCenterValue(props.projectedPct);
  svg.append(value);

  const usedLine = document.createElementNS(SVG_NS, "text");
  usedLine.setAttribute("x", String(cx));
  usedLine.setAttribute("y", String(cy + 4));
  usedLine.setAttribute("text-anchor", "middle");
  usedLine.setAttribute("dominant-baseline", "central");
  usedLine.classList.add("cf-ring-usedpct");
  usedLine.textContent = ringUsedLine(props.usedPct);
  svg.append(usedLine);

  if (props.sublabel) {
    const sub = document.createElementNS(SVG_NS, "text");
    sub.setAttribute("x", String(cx));
    sub.setAttribute("y", String(cy + 18));
    sub.setAttribute("text-anchor", "middle");
    sub.setAttribute("dominant-baseline", "central");
    sub.classList.add("cf-ring-sub");
    sub.textContent = `of ${props.sublabel}`;
    svg.append(sub);
  }

  // Label below the ring — up to two centered lines, never truncated.
  const lines = labelLines(props.label);
  const label = document.createElementNS(SVG_NS, "text");
  label.setAttribute("x", String(cx));
  label.setAttribute("text-anchor", "middle");
  label.classList.add("cf-ring-label");
  const ys = lines.length === 1 ? [size + 13] : [size + 9, size + 21];
  lines.forEach((line, i) => {
    const tspan = document.createElementNS(SVG_NS, "tspan");
    tspan.setAttribute("x", String(cx));
    tspan.setAttribute("y", String(ys[i]));
    tspan.textContent = line;
    label.append(tspan);
  });
  svg.append(label);

  return svg;
}

// ---- Bar gauge ----

export interface BarGaugeProps {
  /** Used % of the allowance (uncapped) — drives level/flow/badge. */
  usedPct: number;
  /** Projected % of the allowance (uncapped) — drives level/flow/badge. */
  projectedPct: number;
  /** Used fill width, % of the fitted track scale. */
  usedScalePct: number;
  /** Projected fill width, % of the fitted track scale. */
  projectedScalePct: number;
  /** Allowance marker position, % of the fitted track scale. */
  includedScalePct: number;
  /** Today marker position, % of the fitted track scale. */
  expectedPct?: number;
  /** "Workers — Requests" */
  label: string;
  /** Text row under the bar: "408k of 10M requests · projected 720k (7%)". */
  detailText: string;
}

/**
 * Horizontal gauge: solid used fill, lighter projected fill, allowance
 * marker line, today marker, level badge, text row. Bar percentages are
 * relative to a track scale fitted to max(used, projected, included) so an
 * over-allowance projection stays on the track with the allowance marker
 * showing where 100% sits.
 */
export function scaleFitPct(v: number, scale: number): number {
  return scale > 0 ? (v / scale) * 100 : 0;
}

export function barGauge(props: BarGaugeProps): HTMLElement {
  const { level, overLimit, limitText } = gaugeLevel(props.usedPct, props.projectedPct);
  const isFlow = level === "critical";
  const card = document.createElement("div");
  // cf-flow carries the stripe flow + pulse; only critical/over-limit bars.
  card.className = `cf-gauge cf-level-${level}${overLimit ? " cf-over" : ""}${isFlow ? " cf-flow" : ""}`;

  const head = document.createElement("div");
  head.className = "cf-gauge-head";
  if (props.label) {
    const name = document.createElement("span");
    name.className = "cf-gauge-name";
    name.textContent = props.label;
    head.append(name);
  }
  if (limitText) {
    const badge = document.createElement("span");
    badge.className = `cf-gauge-badge ${level === "critical" ? "cf-level-critical-text" : "cf-level-warning-text"}`;
    badge.textContent = limitText;
    head.append(badge);
  }
  // An empty head (no label, no badge) would only add its own margin — the
  // caller appends it only when it has content (guarded at the final append).

  // Track + today dot live in a wrapper: the track clips its fills
  // (overflow:hidden), the today dot floats ABOVE the bar so it is never
  // clipped and never reads as a line crossing the fill.
  const bar = document.createElement("div");
  bar.className = "cf-gauge-bar";
  const track = document.createElement("div");
  track.className = "cf-gauge-track";
  const usedFill = document.createElement("div");
  usedFill.className = "cf-gauge-fill";
  usedFill.style.width = `${clampPct(props.usedScalePct)}%`;
  track.append(usedFill);
  if (props.projectedScalePct > props.usedScalePct) {
    const projFill = document.createElement("div");
    projFill.className = "cf-gauge-projected";
    projFill.style.width = `${clampPct(props.projectedScalePct - props.usedScalePct)}%`;
    projFill.style.insetInlineStart = `${clampPct(props.usedScalePct)}%`;
    track.append(projFill);
  }
  const marker = document.createElement("span");
  marker.className = "cf-gauge-marker";
  marker.style.insetInlineStart = `${clampPct(props.includedScalePct)}%`;
  track.append(marker);
  bar.append(track);
  if (props.expectedPct !== undefined) {
    const today = document.createElement("span");
    today.className = "cf-gauge-today";
    today.style.insetInlineStart = `${clampPct(props.expectedPct)}%`;
    bar.append(today);
  }
  const detail = document.createElement("p");
  detail.className = "cf-gauge-detail cf-row-detail";
  detail.textContent = props.detailText;
  if (head.childElementCount > 0) card.append(head);
  card.append(bar, detail);
  return card;
}
