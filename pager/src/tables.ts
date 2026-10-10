// Sortable list views for Workers, D1 and zones (FEAT-052): shared
// sortable-table component (header tap or sort select, filter box, collapse
// after 10 rows), plus the three section renderers. Data: /api/summary v2
// via the shared ApiClient.

import { el, skeleton } from "./dom";
import { formatCount, formatPct } from "./format";
import { api, LoginExpiredError, type Billing, type FetchResult, type Summary } from "./api";
import { cmpRows, cmpText, type SortDir } from "./sort";
import { zoneAttentionLevel } from "./attention";
import { levelForErrorPct, levelForD1 } from "./dashboard";
import { fadeSwap } from "./refresh";

export interface Column<T> {
  key: string;
  label: string;
  /** Right-align numeric columns. */
  numeric?: boolean;
  /** Cell text (used for sorting, filtering and the mobile copy). */
  text: (row: T) => string;
  /** Numeric sort value; omit for text columns. */
  num?: (row: T) => number | null;
  /** Rich cell content; default is a text node of `text`. */
  cell?: (row: T) => HTMLElement;
}

// COLLAPSE_AFTER: lists longer than this render 10 rows + "Show all N".
const COLLAPSE_AFTER = 10;

// Table interaction state (sort/filter/collapse), keyed by the PERSISTENT
// section host so a background-revalidation repaint (BUG-057 defect 2)
// re-applies the user's view instead of resetting it — the table host
// element itself is rebuilt on every paint and cannot carry state.
interface TableUiState {
  sortKey: string;
  sortDir: SortDir;
  filterText: string;
  showAll: boolean;
}
const tableState = new WeakMap<HTMLElement, TableUiState>();

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

// renderSortableTable renders `rows` under one local sort/filter/collapse
// state. State changes rebuild the tbody only — header buttons and the
// filter box keep their value and focus across re-renders.
export function renderSortableTable<T>(host: HTMLElement, opts: {
  columns: Column<T>[];
  rows: T[];
  initial: { key: string; dir: SortDir };
  rowKey?: (row: T) => string;
  rowClass?: (row: T) => string;
  // One legend for the whole table (e.g. the zones byStatus colors), shown
  // under the filter/sort controls on phones (<600px) only.
  sharedLegend?: HTMLElement;
  // Persistent section root whose WeakMap entry carries sort/filter/collapse
  // across repaints (background revalidation, refresh).
  persistKey?: HTMLElement;
  // Stable per-table id ("workers"/"d1"/"zones") — persists the chosen sort
  // across reloads via localStorage (FEAT-pRC6EDA).
  sortPrefId?: string;
  // Per-row graded usage bar (operator ask 2026-10-10): the row's share of
  // the product allowance, colored by the shared level thresholds. Null
  // hides the bar for that row.
  usageBar?: (row: T) => { pctOfAllowance: number } | null;
}): void {
  const stored = opts.sortPrefId ? readSortPref(opts.sortPrefId) : undefined;
  const saved = opts.persistKey ? tableState.get(opts.persistKey) : undefined;
  let sortKey = stored?.key ?? saved?.sortKey ?? opts.initial.key;
  let sortDir = stored?.dir ?? saved?.sortDir ?? opts.initial.dir;
  let filterText = saved?.filterText ?? "";
  let showAll = saved?.showAll ?? false;
  const persist = (): void => {
    if (opts.persistKey) tableState.set(opts.persistKey, { sortKey, sortDir, filterText, showAll });
    if (opts.sortPrefId) writeSortPref(opts.sortPrefId, sortKey, sortDir);
  };

  const table = el("table", "cf-table");
  const thead = el("thead");
  const headerRow = el("tr");
  const buttons = new Map<string, HTMLButtonElement>();
  // Stacked list companion for phones (<600px): same rows, same sort/filter
  // state, rendered as compact list items instead of clipped table columns.
  const list = el("ul", "cf-mobile-list");
  for (const col of opts.columns) {
    const th = el("th");
    const b = el("button", "cf-sort-btn", col.label);
    b.addEventListener("click", () => {
      if (sortKey === col.key) {
        sortDir = sortDir === "asc" ? "desc" : "asc";
      } else {
        sortKey = col.key;
        sortDir = col.numeric ? "desc" : "asc";
      }
      persist();
      render();
    });
    th.append(b);
    headerRow.append(th);
    buttons.set(col.key, b);
  }
  thead.append(headerRow);
  const tbody = el("tbody");
  table.append(thead, tbody);

  // Mobile sort select: mirrors the header buttons; both update one state.
  const sortSelect = el("select", "cf-sort-select") as HTMLSelectElement;
  sortSelect.setAttribute("aria-label", "Sort by");
  for (const col of opts.columns) {
    const o = el("option") as HTMLOptionElement;
    o.value = col.key;
    o.textContent = col.label;
    sortSelect.append(o);
  }
  sortSelect.value = sortKey;
  sortSelect.addEventListener("change", () => {
    const col = opts.columns.find((c) => c.key === sortSelect.value);
    if (col) {
      sortKey = col.key;
      sortDir = col.numeric ? "desc" : "asc";
    }
    paintDirToggle();
    persist();
    render();
  });

  // Direction toggle (FEAT-pRC6EDA): the standard ↑/↓ control beside the
  // sort-by select — one tap flips the order without re-finding the column.
  const dirToggle = el("button", "cf-btn cf-btn-ghost cf-sort-dir") as HTMLButtonElement;
  dirToggle.type = "button";
  const paintDirToggle = (): void => {
    dirToggle.textContent = sortDir === "asc" ? "↑" : "↓";
    dirToggle.setAttribute("aria-label", sortDir === "asc" ? "Sort ascending — tap for largest first" : "Sort descending — tap for smallest first");
    dirToggle.setAttribute("aria-pressed", String(sortDir === "desc"));
  };
  paintDirToggle();
  dirToggle.addEventListener("click", () => {
    sortDir = sortDir === "asc" ? "desc" : "asc";
    paintDirToggle();
    persist();
    render();
  });

  const sortCluster = el("div", "cf-sort-controls");
  sortCluster.append(sortSelect, dirToggle);

  const filter = el("input", "cf-filter") as HTMLInputElement;
  filter.type = "search";
  filter.placeholder = "Filter…";
  filter.value = filterText;
  filter.setAttribute("aria-label", "Filter rows");
  filter.addEventListener("input", () => {
    filterText = filter.value;
    showAll = false;
    persist();
    render();
  });

  const filterBar = el("div", "cf-table-bar");
  filterBar.append(filter, sortCluster);

  function render(): void {
    const col = opts.columns.find((c) => c.key === sortKey) ?? opts.columns[0];
    const cmp = col.num ? cmpRows(col.num, sortDir) : cmpText(col.text, sortDir);
    let rows = [...opts.rows].sort(cmp);
    // Filter: case-insensitive substring across every column's text.
    const q = filterText.trim().toLowerCase();
    if (q) rows = rows.filter((row) => opts.columns.some((c) => c.text(row).toLowerCase().includes(q)));

    // Sort-direction indicator on the active header + aria-sort.
    const activeIdx = opts.columns.findIndex((c) => c.key === sortKey);
    headerRow.querySelectorAll("th").forEach((th, i) => {
      th.removeAttribute("aria-sort");
      if (i === activeIdx) th.setAttribute("aria-sort", sortDir === "asc" ? "ascending" : "descending");
    });
    for (const [key, b] of buttons) {
      const colDef = opts.columns.find((c) => c.key === key)!;
      b.textContent = colDef.label + (key === sortKey ? (sortDir === "asc" ? " ↑" : " ↓") : "");
      b.setAttribute("aria-pressed", String(key === sortKey));
    }

    // Body: apply the collapse rule to the filtered rows.
    tbody.replaceChildren();
    list.replaceChildren();
    const visible = showAll ? rows : rows.slice(0, COLLAPSE_AFTER);
    for (const row of visible) {
      const tr = el("tr", opts.rowClass?.(row));
      const barInfo = opts.usageBar?.(row) ?? null;
      for (const col of opts.columns) {
        const td = el("td");
        if (col.numeric) td.className = "cf-num";
        td.append(col.cell ? col.cell(row) : document.createTextNode(col.text(row)));
        // Graded usage bar sits under the FIRST column's content: the row's
        // share of the product allowance at a glance.
        if (barInfo && col === opts.columns[0]) td.append(miniUsageBar(barInfo.pctOfAllowance));
        tr.append(td);
      }
      tbody.append(tr);
      list.append(mobileRowItem(opts.columns, sortKey, opts.rowClass, row, barInfo?.pctOfAllowance));
    }
    // Show all N toggle — wrapped in a full-width row and a list item so
    // both renderings can expand.
    if (!showAll && rows.length > COLLAPSE_AFTER) {
      const more = () => {
        showAll = true;
        persist();
        render();
      };
      const wrap = el("tr", "cf-showall-row");
      const cell = el("td");
      cell.colSpan = opts.columns.length;
      const moreBtn = el("button", "cf-btn cf-show-all", `Show all ${rows.length}`);
      moreBtn.addEventListener("click", more);
      cell.append(moreBtn);
      wrap.append(cell);
      tbody.append(wrap);
      const li = el("li", "cf-showall-row");
      const listBtn = el("button", "cf-btn cf-show-all", `Show all ${rows.length}`);
      listBtn.addEventListener("click", more);
      li.append(listBtn);
      list.append(li);
    }
  }
  render();

  host.append(filterBar);
  if (opts.sharedLegend) host.append(opts.sharedLegend);
  host.append(table, list);
}

// mobileRowItem renders one row as a compact stacked list item for phones
// (<600px): line 1 name + the active sort metric (right, tabular-nums),
// line 2 the byStatus bar when the table has one, line 3 the remaining
// numeric metrics in muted text. Mirrors the table's sort/filter/collapse
// state exactly.
function mobileRowItem<T>(columns: Column<T>[], sortKey: string, rowClass: ((row: T) => string) | undefined, row: T, usagePct?: number): HTMLLIElement {
  const nameCol = columns[0];
  const active = columns.find((c) => c.key === sortKey) ?? columns[0];
  const primaryCol = active.numeric ? active : columns.find((c) => c.numeric);
  const li = el("li", rowClass?.(row));
  const top = el("div", "cf-mrow-top");
  top.append(
    el("span", "cf-mrow-name", nameCol.text(row)),
    el("span", "cf-mrow-primary", primaryCol ? primaryCol.text(row) : ""),
  );
  li.append(top);
  if (usagePct !== undefined) li.append(miniUsageBar(usagePct));
  const statusCol = columns.find((c) => c.key === "byStatus" && c.cell);
  if (statusCol?.cell) {
    const bar = el("div", "cf-mrow-status");
    bar.append(statusCol.cell(row));
    li.append(bar);
  }
  const shortLabel = (label: string): string => label.replace(/%\s*$/, "").trim().toLowerCase();
  const meta = columns
    .filter((c) => c.numeric && c !== primaryCol)
    .map((c) => `${c.text(row)} ${shortLabel(c.label)}`)
    .join(" · ");
  if (meta) li.append(el("div", "cf-mrow-meta", meta));
  return li;
}

/** Share of an allowance as a percentage; 0 when the allowance is unknown. */
export function usageSharePct(used: number, included: number): number {
  return included > 0 ? (used / included) * 100 : 0;
}

/** miniUsageBar: a 4px graded bar — this row's share of the product
 *  allowance. Color follows the shared thresholds (ok → warning → critical
 *  via the level classes); width caps at 100% visually, the title carries
 *  the true number past it. */
export function miniUsageBar(pctOfAllowance: number): HTMLElement {
  const level = levelForUsageBar(pctOfAllowance);
  const wrap = el("div", `cf-mini-bar cf-level-${level}`);
  const fill = el("div", "cf-mini-fill");
  fill.style.width = `${Math.min(100, Math.max(0, pctOfAllowance))}%`;
  fill.title = `${pctOfAllowance.toFixed(pctOfAllowance >= 100 ? 0 : 1)}% of the allowance`;
  wrap.append(fill);
  wrap.setAttribute("role", "img");
  wrap.setAttribute("aria-label", fill.title);
  return wrap;
}

function levelForUsageBar(pct: number): "ok" | "warning" | "critical" {
  if (pct >= 80) return "critical";
  if (pct >= 50) return "warning";
  return "ok";
}

// ---- Section plumbing ----

// fetchSummaryView drives one summary-backed section: skeleton, fetch (with
// refresh), error/login-expired state with retry, then the paint callback.
// A background revalidation repaints with the fresh data (BUG-057 defect 2);
// a failed refresh rethrows (defect 3) so the old DOM — good data and the
// clicked Refresh button node — stays and the button surfaces the error.
async function fetchSummaryView(root: HTMLElement, label: string, opts: { refresh?: boolean },
  paint: (root: HTMLElement, data: FetchResult<Summary>) => void,
  refetch?: (root: HTMLElement) => void): Promise<void> {
  if (!opts.refresh) root.replaceChildren(skeleton());
  try {
    const res = await api.fetchJson<Summary>("api/summary", {
      refresh: opts.refresh,
      onRevalidate: (fresh) => paint(root, { data: fresh, source: "network", ageSec: 0, demo: false }),
    });
    paint(root, res);
  } catch (err) {
    if (opts.refresh) throw err;
    let box: HTMLElement;
    if (err instanceof LoginExpiredError) {
      box = el("section", "cf-card");
      box.append(el("p", "cf-empty", err.message));
    } else {
      box = el("section", "cf-card cf-level-warning");
      box.append(
        el("h2", undefined, "Could not load " + label),
        el("p", "cf-row-detail", err instanceof Error ? err.message : String(err)),
      );
      const retry = el("button", "cf-btn", "Retry");
      retry.addEventListener("click", () => void fetchSummaryView(root, label, {}, paint, refetch));
      box.append(retry);
    }
    root.replaceChildren(box);
  }
}

// ---- Workers ----

export function renderWorkers(root: HTMLElement, opts: { refresh?: boolean } = {}): Promise<void> {
  return fetchSummaryView(root, "the Workers list", opts, (host, res) => {
    void (async () => {
      // Allowance for the graded bars: Workers Paid request allowance from
      // the billing cache — one cached fetch, absent bars on failure.
      let included = 0;
      try {
        const b = await api.fetchJson<Billing>("api/billing");
        included = b.data.products.find((p) => p.id === "workers.requests")?.included ?? 0;
      } catch {
        included = 0;
      }
      const card = el("section", "cf-card");
      card.append(el("h2", undefined, "Workers (24h)"));
      if (res.data.workers.length === 0) {
        card.append(el("p", "cf-empty cf-empty-quiet", "No Worker traffic in the window."));
      } else {
        renderSortableTable(card, {
          persistKey: host,
          columns: [
            { key: "script", label: "Script", text: (w) => w.script },
            { key: "requests", label: "Requests", numeric: true, text: (w) => formatCount(w.requests), num: (w) => w.requests },
            { key: "errors", label: "Errors", numeric: true, text: (w) => formatCount(w.errors), num: (w) => w.errors },
            { key: "errorPct", label: "Error %", numeric: true, text: (w) => formatPct(w.errorPct, 2), num: (w) => w.errorPct },
            { key: "cpuP50", label: "CPU p50", numeric: true, text: (w) => (w.cpuP50Ms === null ? "—" : `${w.cpuP50Ms.toFixed(1)}ms`), num: (w) => w.cpuP50Ms },
            { key: "cpuP99", label: "CPU p99", numeric: true, text: (w) => (w.cpuP99Ms === null ? "—" : `${w.cpuP99Ms.toFixed(1)}ms`), num: (w) => w.cpuP99Ms },
          ],
          rows: res.data.workers,
          sortPrefId: "workers",
          initial: { key: "requests", dir: "desc" },
          usageBar: included > 0 ? (w) => ({ pctOfAllowance: usageSharePct(w.requests, included) }) : undefined,
          rowClass: (w) => {
            const lv = levelForErrorPct(w.errorPct);
            return lv === "critical" ? "cf-level-critical" : lv === "warning" ? "cf-level-warning" : "";
          },
        });
      }
      // Repaint through fadeSwap so refreshed content fades in without a jump.
      const view = el("div");
      view.append(card);
      fadeSwap(host, view);
    })();
  });
}

// ---- D1 ----

export function renderD1(root: HTMLElement, opts: { refresh?: boolean } = {}): Promise<void> {
  return fetchSummaryView(root, "the D1 list", opts, (host, res) => {
    void (async () => {
      // Allowance for the graded bars: the D1 rows-read allowance from the
      // billing cache — one cached fetch, absent bars on failure.
      let included = 0;
      try {
        const b = await api.fetchJson<Billing>("api/billing");
        included = b.data.products.find((p) => p.id === "d1.rows_read")?.included ?? 0;
      } catch {
        included = 0;
      }
      const card = el("section", "cf-card");
      card.append(el("h2", undefined, "D1 databases (24h)"));
      if (res.data.d1.length === 0) {
        card.append(el("p", "cf-empty cf-empty-quiet", "No D1 activity in the window."));
      } else {
        renderSortableTable(card, {
          persistKey: host,
          columns: [
            { key: "name", label: "Database", text: (d) => d.name },
            { key: "rowsRead", label: "Rows read", numeric: true, text: (d) => formatCount(d.rowsRead), num: (d) => d.rowsRead },
            { key: "rowsWritten", label: "Rows written", numeric: true, text: (w) => formatCount(w.rowsWritten), num: (w) => w.rowsWritten },
            { key: "queries", label: "Queries", numeric: true, text: (d) => formatCount(d.readQueries), num: (d) => d.readQueries },
            { key: "rowsPerQuery", label: "Rows/query", numeric: true, text: (d) => formatCount(d.rowsPerQuery), num: (d) => d.rowsPerQuery },
          ],
          rows: res.data.d1,
          sortPrefId: "d1",
          initial: { key: "rowsRead", dir: "desc" },
          usageBar: included > 0 ? (d) => ({ pctOfAllowance: usageSharePct(d.rowsRead, included) }) : undefined,
          rowClass: (d) => {
            const lv = levelForD1(d.rowsRead);
            return lv === "critical" ? "cf-level-critical" : lv === "warning" ? "cf-level-warning" : "";
          },
        });
      }
      // Repaint through fadeSwap so refreshed content fades in without a jump.
      const view = el("div");
      view.append(card);
      fadeSwap(host, view);
    })();
  });
}

// ---- Zones ----

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

export function renderZones(root: HTMLElement, opts: { refresh?: boolean } = {}): Promise<void> {
  return fetchSummaryView(root, "the zones list", opts, (host, res) => {
    const card = el("section", "cf-card");
    card.append(el("h2", undefined, "Zones (24h)"));
    const rows = res.data.zones.filter((z) => z.total > 0);
    if (rows.length === 0) {
      card.append(el("p", "cf-empty cf-empty-quiet", "No zone traffic in the window."));
    } else {
      // One shared byStatus legend for phones: per-row legends are hidden in
      // compact rows, so aggregate every zone's statuses into one line.
      const agg: Record<string, number> = {};
      for (const z of res.data.zones) {
        for (const [k, v] of Object.entries(z.byStatus ?? {})) agg[k] = (agg[k] ?? 0) + v;
      }
      const sharedLegend = byStatusLegend(agg);
      sharedLegend.classList.add("cf-shared-legend");
      renderSortableTable(card, {
        persistKey: host,
        columns: [
          { key: "zone", label: "Zone", text: (z) => z.zone },
          { key: "total", label: "Requests", numeric: true, text: (z) => formatCount(z.total), num: (z) => z.total },
          { key: "uncached", label: "Uncached", numeric: true, text: (z) => formatCount(z.uncached), num: (z) => z.uncached },
          { key: "missPct", label: "Miss %", numeric: true, text: (z) => (z.missPct === null ? "—" : formatPct(z.missPct)), num: (z) => z.missPct },
          { key: "byStatus", label: "By status",
            text: (z) => Object.entries(z.byStatus ?? {}).map(([k, v]) => `${k} ${v}`).join(" "),
            cell: (z) => {
              const wrap = el("div", "cf-zonestack");
              if (z.byStatus) wrap.append(byStatusStack(z.byStatus), byStatusLegend(z.byStatus));
              return wrap;
            } },
        ],
        rows,
        sharedLegend,
        sortPrefId: "zones",
        initial: { key: "uncached", dir: "desc" },
        rowClass: (z) => {
          const lv = zoneAttentionLevel(z.uncached, z.missPct);
          return lv === "critical" ? "cf-level-critical" : lv === "warning" ? "cf-level-warning" : "";
        },
      });
    }
    // Repaint through fadeSwap so refreshed content fades in without a jump.
    const view = el("div");
    view.append(card);
    fadeSwap(host, view);
  });
}
