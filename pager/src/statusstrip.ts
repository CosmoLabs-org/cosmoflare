// Bottom status strip (UI-1, operator request 2026-10-10): one global
// "updated X ago" line plus the billing-period day ("day eighteen out of
// thirty"), pinned to the bottom of the shell. The per-view status bars are
// gone — this strip replaces them. Pure formatting helpers are split from
// the DOM writers so they test in the node vitest environment.

/** formatAgeText renders a data age honestly: "just now" under a minute,
 *  then Xm, Xh, Xd. Floor, never round up — "59m ago" must not read as an
 *  hour. */
export function formatAgeText(ageSec: number): string {
  if (ageSec < 60) return "just now";
  if (ageSec < 3600) return `${Math.floor(ageSec / 60)}m ago`;
  if (ageSec < 86400) return `${Math.floor(ageSec / 3600)}h ago`;
  return `${Math.floor(ageSec / 86400)}d ago`;
}

/** periodText renders the billing-period progress — "Day 3 of 30" — or null
 *  when the period is unknown (the span stays hidden rather than guessing). */
export function periodText(day?: number, days?: number): string | null {
  if (day === undefined || days === undefined) return null;
  return `Day ${day} of ${days}`;
}

// Last painted data age and the wall-clock time it was painted, so the
// 60s clock can age the line forward between data updates.
let lastAgeSec: number | null = null;
let lastSetAtMs = 0;

/** updateStatusStrip paints one update into the strip: the age line
 *  (#cf-status-age, empty when ageSec is null — no data, no claim) and the
 *  period span (#cf-status-period, hidden when the period is unknown). */
export function updateStatusStrip(info: { ageSec: number | null; day?: number; days?: number }): void {
  lastAgeSec = info.ageSec;
  lastSetAtMs = Date.now();
  const age = document.getElementById("cf-status-age");
  if (age) age.textContent = info.ageSec === null ? "" : `Updated ${formatAgeText(info.ageSec)}`;
  const period = document.getElementById("cf-status-period");
  if (period) {
    const text = periodText(info.day, info.days);
    period.textContent = text ?? "";
    period.hidden = text === null;
  }
}

/** startStatusStripClock re-renders the age line every 60s by adding the
 *  elapsed wall time since the last update — the strip stays honest between
 *  data refreshes without refetching anything. */
export function startStatusStripClock(): void {
  window.setInterval(() => {
    if (lastAgeSec === null) return;
    const elapsedSec = Math.floor((Date.now() - lastSetAtMs) / 1000);
    const age = document.getElementById("cf-status-age");
    if (age) age.textContent = `Updated ${formatAgeText(lastAgeSec + elapsedSec)}`;
  }, 60_000);
}
