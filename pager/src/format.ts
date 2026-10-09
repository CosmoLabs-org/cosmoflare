// Pure formatting and math helpers for the Ops dashboard views (FEAT-052).
// No DOM here — everything is testable without jsdom.

/** "3.3k" / "52.3M" / "2.9B" / "59" — same shape the CLI prints. */
export function formatCount(v: number): string {
  if (v >= 1e9) return `${(v / 1e9).toFixed(1)}B`;
  if (v >= 1e6) return `${(v / 1e6).toFixed(1)}M`;
  if (v >= 1e3) return `${(v / 1e3).toFixed(1)}k`;
  return String(Math.round(v));
}

/** "$4.20" — always two decimals, thousands separated. */
export function formatUsd(v: number): string {
  return `$${v.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

/** "3.9 GB" / "412k ops" — GB-style units keep one decimal, counts formatCount. */
export function formatAmount(v: number, unit: string): string {
  if (/^GB/i.test(unit)) return `${v.toFixed(1)} ${unit}`;
  return `${formatCount(v)} ${unit}`;
}

/** "74.2%" — one decimal by default. */
export function formatPct(v: number, digits = 1): string {
  return `${v.toFixed(digits)}%`;
}

/** "just now" / "3m ago" / "2h ago" / "3d ago" from an age in seconds. */
export function formatAge(ageSec: number): string {
  if (ageSec < 60) return "just now";
  const minutes = Math.floor(ageSec / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

/** "Oct 31" from an RFC3339 date — the billing period ends label. */
export function formatDateShort(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

// Period progress: the billing period carries day N of M (1-based, both
// inclusive); the progress bar and the "ends <date>" label derive from it.
export interface PeriodProgress {
  /** 0..100, clamped. */
  elapsedPct: number;
  /** Full days remaining after today. */
  daysLeft: number;
}

export function periodProgress(day: number, days: number): PeriodProgress {
  const safeDay = Math.max(0, day);
  const safeDays = Math.max(1, days);
  return {
    elapsedPct: Math.min(100, (safeDay / safeDays) * 100),
    daysLeft: Math.max(0, safeDays - safeDay),
  };
}

/** Walks back from `end` (exclusive) the seconds left in the period. */
export function secondsUntil(endIso: string, nowMs: number): number {
  const end = Date.parse(endIso);
  if (Number.isNaN(end)) return 0;
  return Math.max(0, Math.floor((end - nowMs) / 1000));
}

