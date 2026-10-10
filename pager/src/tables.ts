// Sortable table helpers (FEAT-052): cross-reload sort preferences, the
// sort-direction glyph SVGs, the usage-share math and the zones byStatus
// bar/legend shared by the React DataTable and views. The shared sortable-
// table component and the three vanilla section renderers that used to live
// here were retired (TASK-pD575FN); pager/src/app/components/DataTable.tsx
// and the app views are the only render path now.

import { el } from "./dom";
import { formatCount } from "./format";
import type { SortDir } from "./sort";

/** Cross-reload sort preference (FEAT-pRC6EDA): "cf-sort:<table>" holds
 *  {"key","dir"} in localStorage. Pure helpers take the storage so tests
 *  can pass a Map stand-in. */
export function readSortPref(id: string, storage: Storage | Pick<Storage, "getItem"> = window.localStorage): { key: string; dir: SortDir } | undefined {
  try {
    const raw = storage.getItem(`cf-sort:${id}`);
    if (!raw) return undefined;
    const parsed = JSON.parse(raw) as { key?: string; dir?: string };
    if (typeof parsed.key === "string" && (parsed.dir === "asc" || parsed.dir === "desc")) {
      return { key: parsed.key, dir: parsed.dir };
    }
  } catch {
    // corrupt or unavailable storage: fall back to defaults
  }
  return undefined;
}

export function writeSortPref(id: string, key: string, dir: SortDir, storage: Storage | Pick<Storage, "setItem"> = window.localStorage): void {
  try {
    storage.setItem(`cf-sort:${id}`, JSON.stringify({ key, dir }));
  } catch {
    // private mode / quota: preference just doesn't persist
  }
}

// Sort-direction glyphs (operator 2026-10-10: proper icons, no text arrows,
// no emojis): module-scoped static SVG constants — same pattern as the nav
// icons; never any dynamic content.
// Exported (P-05a) so the React DataTable renders the exact same glyphs.
export const SORT_NEUTRAL_SVG =
  '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M8 9l4-4 4 4M8 15l4 4 4-4"/></svg>';
export const ARROW_UP_SVG =
  '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 19V5M5 12l7-7 7 7"/></svg>';
export const ARROW_DOWN_SVG =
  '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 5v14M19 12l-7 7-7-7"/></svg>';

/** Share of an allowance as a percentage; 0 when the allowance is unknown. */
export function usageSharePct(used: number, included: number): number {
  return included > 0 ? (used / included) * 100 : 0;
}

// Exported (P-05a) for the React DataTable's mini usage bars — the vanilla
// miniUsageBar this replaced kept the same thresholds.
export function levelForUsageBar(pct: number): "ok" | "warning" | "critical" {
  if (pct >= 80) return "critical";
  if (pct >= 50) return "warning";
  return "ok";
}

// ---- Zones byStatus ----

// ZONE_GROUPS: statuses fold into five visual groups. Unknown statuses land
// in "other".
const ZONE_GROUPS: { label: string; cls: string; keys: string[] }[] = [
  { label: "hit", cls: "cf-seg-hit", keys: ["hit"] },
  { label: "miss+expired", cls: "cf-seg-miss", keys: ["miss", "expired"] },
  { label: "dynamic", cls: "cf-seg-dynamic", keys: ["dynamic"] },
  { label: "bypass", cls: "cf-seg-bypass", keys: ["bypass"] },
  { label: "other", cls: "cf-seg-other", keys: [] },
];

function zoneSums(byStatus: Record<string, number>): { label: string; cls: string; sum: number }[] {
  const known = new Set(ZONE_GROUPS.flatMap((g) => g.keys));
  return ZONE_GROUPS.map((g) => ({
    label: g.label,
    cls: g.cls,
    sum: g.keys.length
      ? g.keys.reduce((s, k) => s + (byStatus[k] ?? 0), 0)
      : Object.entries(byStatus).reduce((s, [k, v]) => (known.has(k) ? s : s + v), 0),
  })).filter((g) => g.sum > 0);
}

/** Horizontal stacked bar of cache statuses (hit / miss+expired / dynamic / bypass / other). */
export function byStatusStack(byStatus: Record<string, number>): HTMLElement {
  const total = Object.values(byStatus).reduce((s, v) => s + v, 0);
  const stack = el("div", "cf-stack");
  stack.setAttribute("role", "img");
  if (total <= 0) return stack;
  for (const g of zoneSums(byStatus)) {
    const seg = el("div", `cf-seg ${g.cls}`);
    seg.style.width = `${(g.sum / total) * 100}%`;
    seg.title = `${g.label}: ${formatCount(g.sum)} (${((g.sum / total) * 100).toFixed(1)}%)`;
    stack.append(seg);
  }
  return stack;
}

/** Legend under the stacked bar: colored dot + muted label + count per group. */
export function byStatusLegend(byStatus: Record<string, number>): HTMLElement {
  const legend = el("div", "cf-stack-legend");
  for (const g of zoneSums(byStatus)) {
    // The color goes on the dot element only — never on the label span.
    const item = el("span", "cf-legend-item");
    item.append(
      el("span", `cf-legend-dot ${g.cls}`),
      document.createTextNode(`${g.label} ${formatCount(g.sum)}`),
    );
    legend.append(item);
  }
  return legend;
}
