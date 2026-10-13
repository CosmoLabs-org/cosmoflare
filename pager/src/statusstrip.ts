// Honest data age (UI-1, operator request 2026-10-10; relocated
// 2026-10-14): one global "updated X ago" line, painted into the span
// beside the masthead's Refresh control (the bottom status strip is gone —
// the liquid-glass pill owns the bottom). The billing-period day counter
// moved to the Overview plan line; Billing keeps its dial. Pure
// formatting helpers are split from the DOM writer so they test in the
// node vitest environment.

/** formatAgeText renders a data age honestly: "just now" under a minute,
 *  then Xm, Xh, Xd. Floor, never round up — "59m ago" must not read as an
 *  hour. */
export function formatAgeText(ageSec: number): string {
  if (ageSec < 60) return "just now";
  if (ageSec < 3600) return `${Math.floor(ageSec / 60)}m ago`;
  if (ageSec < 86400) return `${Math.floor(ageSec / 3600)}h ago`;
  return `${Math.floor(ageSec / 86400)}d ago`;
}

// Last painted data age and the wall-clock time it was painted, so the
// 60s clock can age the line forward between data updates.
let lastAgeSec: number | null = null;
let lastSetAtMs = 0;

/** updateStatusStrip paints one update into the age line (#cf-status-age,
 *  empty when ageSec is null — no data, no claim). Views still pass
 *  day/days; the period counter now lives in the Overview plan line and
 *  Billing's dial, so those fields are accepted and ignored — no view
 *  changes needed. */
export function updateStatusStrip(info: { ageSec: number | null; day?: number; days?: number }): void {
  lastAgeSec = info.ageSec;
  lastSetAtMs = Date.now();
  const age = document.getElementById("cf-status-age");
  if (age) age.textContent = info.ageSec === null ? "" : `Updated ${formatAgeText(info.ageSec)}`;
}

/** startStatusStripClock re-renders the age line every 60s by adding the
 *  elapsed wall time since the last update — the line stays honest between
 *  data refreshes without refetching anything. */
export function startStatusStripClock(): void {
  window.setInterval(() => {
    if (lastAgeSec === null) return;
    const elapsedSec = Math.floor((Date.now() - lastSetAtMs) / 1000);
    const age = document.getElementById("cf-status-age");
    if (age) age.textContent = `Updated ${formatAgeText(lastAgeSec + elapsedSec)}`;
  }, 60_000);
}
