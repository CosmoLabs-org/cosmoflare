// Shared view constants (FEAT-045): the alert-list relative-time formatter
// and the pairing view's command, glyph and status-level classes. The
// imperative vanilla renderers that used to live here were retired
// (TASK-pD575FN); pager/src/app/views/{AlertsView,PairingView}.tsx are the
// only render paths now. Visual tokens mirror desktop/src/styles.css
// verbatim (dark-first, severity ramp).

/** The exact command shown next to the copied subscription blob. */
export const PAIRING_COMMAND = "cosmoflare alerts push add";

/** Relative-time formatter: "just now", "4m ago", "2h ago", "3d ago". */
export function relativeTime(epochMs: number, nowMs: number = Date.now()): string {
  const seconds = Math.max(0, Math.floor((nowMs - epochMs) / 1000));
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

/** The pairing status line's level classes (existing severity text colors). */
export const PAIRING_STATUS_LEVEL: Record<string, string> = {
  info: "",
  ok: "cf-ok-text",
  warn: "cf-level-warning-text",
  error: "cf-level-critical-text",
};

/** The pairing hero mark: a phone emitting a broadcast — the signal arcs
 *  carry .cf-wave and breathe in a staggered loop (see styles; frozen under
 *  prefers-reduced-motion). */
export const PAIRING_GLYPH =
  '<svg width="44" height="44" viewBox="0 0 26 24" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><rect x="4" y="2.5" width="10" height="19" rx="2.6" stroke-width="1.7"/><path d="M7.6 18.6h2.8" stroke-width="1.7"/><path class="cf-wave" d="M16.2 8.6a4.6 4.6 0 0 1 0 6.8" stroke-width="1.7"/><path class="cf-wave" d="M19.3 5.6a8.8 8.8 0 0 1 0 12.8" stroke-width="1.7"/><path class="cf-wave" d="M22.4 2.7a12.9 12.9 0 0 1 0 18.6" stroke-width="1.7"/></svg>';
