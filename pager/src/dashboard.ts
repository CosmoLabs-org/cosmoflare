// Dashboard view (FEAT-052): monthly pacing, D1 rows read and zone cache
// health from the Ops Worker's /api/summary. Levels mirror the starter alert
// thresholds (O10) so the screen and the pager agree on what is "red".

export type Level = "ok" | "warning" | "critical";

export function formatCount(v: number): string {
  if (v >= 1e9) return `${(v / 1e9).toFixed(1)}B`;
  if (v >= 1e6) return `${(v / 1e6).toFixed(1)}M`;
  if (v >= 1e3) return `${(v / 1e3).toFixed(1)}k`;
  return String(Math.round(v));
}

export const levelForUncached = (n: number): Level => (n >= 10_000 ? "critical" : n >= 5_000 ? "warning" : "ok");
export const levelForMiss = (pct: number | null): Level => (pct === null ? "ok" : pct >= 80 ? "critical" : pct >= 50 ? "warning" : "ok");
export const levelForD1 = (rows24h: number): Level => (rows24h >= 1e9 ? "critical" : rows24h >= 1e8 ? "warning" : "ok");
export const levelForUsage = (projectedPct: number | null): Level =>
  projectedPct === null ? "ok" : projectedPct >= 100 ? "critical" : projectedPct >= 80 ? "warning" : "ok";

interface Summary {
  generatedAt: string;
  usage: { name: string; used: number; limit: number; pct: number; projectedPct: number | null }[];
  d1: { name: string; rowsRead: number; readQueries: number; rowsPerQuery: number }[];
  zones: { zone: string; total: number; uncached: number; missPct: number | null }[];
  errors: string[];
}

function el(tag: string, className?: string, text?: string): HTMLElement {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

function row(label: string, value: string, level: Level, detail?: string): HTMLElement {
  const r = el("div", `cf-row cf-level-${level}`);
  const left = el("div", "cf-row-label", label);
  if (detail) left.append(el("div", "cf-row-detail", detail));
  r.append(left, el("div", "cf-row-value", value));
  return r;
}

export async function renderDashboard(root: HTMLElement, refresh = false): Promise<void> {
  root.replaceChildren(el("p", "cf-empty", "Loading…"));
  let data: Summary;
  try {
    const res = await fetch(`/api/summary${refresh ? "?refresh" : ""}`, { credentials: "same-origin", redirect: "manual" });
    // An expired Access session answers with a redirect to the login page.
    if (res.type === "opaqueredirect" || res.status === 403) throw new Error("your login expired — close and reopen the app to sign in again");
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    data = (await res.json()) as Summary;
  } catch (err) {
    root.replaceChildren(el("p", "cf-empty", `Could not load the summary: ${err instanceof Error ? err.message : String(err)}`));
    return;
  }

  const bar = el("div", "cf-dash-bar");
  bar.append(el("span", "cf-row-detail", `Updated ${new Date(data.generatedAt).toLocaleTimeString()}`));
  const reload = el("button", "cf-btn", "Refresh") as HTMLButtonElement;
  reload.addEventListener("click", () => void renderDashboard(root, true));
  bar.append(reload);

  const usage = el("section", "cf-card");
  usage.append(el("h2", undefined, "This month"));
  for (const u of data.usage) {
    usage.append(row(u.name, `${u.pct.toFixed(1)}%`, levelForUsage(u.projectedPct),
      `${formatCount(u.used)} of ${formatCount(u.limit)}${u.projectedPct !== null ? ` · projected ${u.projectedPct.toFixed(0)}%` : ""}`));
  }

  const d1 = el("section", "cf-card");
  d1.append(el("h2", undefined, "D1 rows read (24h)"));
  for (const d of data.d1.slice(0, 8)) {
    d1.append(row(d.name, formatCount(d.rowsRead), levelForD1(d.rowsRead), `${formatCount(d.readQueries)} queries · ${formatCount(d.rowsPerQuery)} rows/query`));
  }
  if (data.d1.length === 0) d1.append(el("p", "cf-empty", "No D1 activity."));

  const zones = el("section", "cf-card");
  zones.append(el("h2", undefined, "Zones (24h)"));
  for (const z of data.zones.filter((z) => z.total > 0).slice(0, 15)) {
    const level: Level = levelForUncached(z.uncached) === "critical" || levelForMiss(z.missPct) === "critical" ? "critical"
      : levelForUncached(z.uncached) === "warning" || levelForMiss(z.missPct) === "warning" ? "warning" : "ok";
    zones.append(row(z.zone, `${formatCount(z.uncached)} uncached`, level,
      `${formatCount(z.total)} requests${z.missPct !== null ? ` · ${z.missPct.toFixed(1)}% miss` : ""}`));
  }

  root.replaceChildren(bar, usage, d1, zones);
  if (data.errors.length) {
    const gaps = el("section", "cf-card cf-level-warning");
    gaps.append(el("h2", undefined, "Telemetry gaps"));
    for (const e of data.errors) gaps.append(el("p", "cf-row-detail", e));
    root.append(gaps);
  }
}
