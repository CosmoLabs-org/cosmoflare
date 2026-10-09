// Sortable list views for Workers, D1 and zones (FEAT-052): shared
// sortable-table component (header tap or sort select, filter box, collapse
// after 10 rows), plus the three section renderers. Data: /api/summary v2
// via the shared ApiClient.

import { el, skeleton, statusLine } from "./dom";
import { formatCount, formatPct } from "./format";
import { api, LoginExpiredError, totalAgeSec, type FetchResult, type Summary } from "./api";
import { cmpRows, cmpText, type SortDir } from "./sort";
import { zoneAttentionLevel } from "./attention";
import { levelForErrorPct, levelForD1 } from "./dashboard";
import { fadeSwap, refreshButton } from "./refresh";

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
}): void {
  let sortKey = opts.initial.key;
  let sortDir = opts.initial.dir;
  let filterText = "";
  let showAll = false;

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
    render();
  });

  const filter = el("input", "cf-filter") as HTMLInputElement;
  filter.type = "search";
  filter.placeholder = "Filter…";
  filter.setAttribute("aria-label", "Filter rows");
  filter.addEventListener("input", () => {
    filterText = filter.value;
    showAll = false;
    render();
  });

  const filterBar = el("div", "cf-table-bar");
  filterBar.append(filter, sortSelect);

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
      for (const col of opts.columns) {
        const td = el("td");
        if (col.numeric) td.className = "cf-num";
        td.append(col.cell ? col.cell(row) : document.createTextNode(col.text(row)));
        tr.append(td);
      }
      tbody.append(tr);
      list.append(mobileRowItem(opts.columns, sortKey, opts.rowClass, row));
    }
    // Show all N toggle — wrapped in a full-width row and a list item so
    // both renderings can expand.
    if (!showAll && rows.length > COLLAPSE_AFTER) {
      const more = () => {
        showAll = true;
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
function mobileRowItem<T>(columns: Column<T>[], sortKey: string, rowClass: ((row: T) => string) | undefined, row: T): HTMLLIElement {
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
    const bar = el("div", "cf-dash-bar");
    const line = statusLine(totalAgeSec(res.ageSec, res.data.cache), { cached: Boolean(res.data.cache?.stale), demo: res.demo });
    const refresh = refreshButton(async () => renderWorkers(host, { refresh: true }));
    bar.append(line, refresh);

    const card = el("section", "cf-card");
    card.append(el("h2", undefined, "Workers (24h)"));
    if (res.data.workers.length === 0) {
      card.append(el("p", "cf-empty cf-empty-quiet", "No Worker traffic in the window."));
    } else {
      renderSortableTable(card, {
        columns: [
          { key: "script", label: "Script", text: (w) => w.script },
          { key: "requests", label: "Requests", numeric: true, text: (w) => formatCount(w.requests), num: (w) => w.requests },
          { key: "errors", label: "Errors", numeric: true, text: (w) => formatCount(w.errors), num: (w) => w.errors },
          { key: "errorPct", label: "Error %", numeric: true, text: (w) => formatPct(w.errorPct, 2), num: (w) => w.errorPct },
          { key: "cpuP50", label: "CPU p50", numeric: true, text: (w) => (w.cpuP50Ms === null ? "—" : `${w.cpuP50Ms.toFixed(1)}ms`), num: (w) => w.cpuP50Ms },
          { key: "cpuP99", label: "CPU p99", numeric: true, text: (w) => (w.cpuP99Ms === null ? "—" : `${w.cpuP99Ms.toFixed(1)}ms`), num: (w) => w.cpuP99Ms },
        ],
        rows: res.data.workers,
        initial: { key: "requests", dir: "desc" },
        rowClass: (w) => {
          const lv = levelForErrorPct(w.errorPct);
          return lv === "critical" ? "cf-level-critical" : lv === "warning" ? "cf-level-warning" : "";
        },
      });
    }
    // Repaint through fadeSwap so refreshed content fades in without a jump.
    const view = el("div");
    view.append(bar, card);
    fadeSwap(host, view);
  });
}

// ---- D1 ----

export function renderD1(root: HTMLElement, opts: { refresh?: boolean } = {}): Promise<void> {
  return fetchSummaryView(root, "the D1 list", opts, (host, res) => {
    const bar = el("div", "cf-dash-bar");
    const line = statusLine(totalAgeSec(res.ageSec, res.data.cache), { cached: Boolean(res.data.cache?.stale), demo: res.demo });
    const refresh = refreshButton(async () => renderD1(host, { refresh: true }));
    bar.append(line, refresh);

    const card = el("section", "cf-card");
    card.append(el("h2", undefined, "D1 databases (24h)"));
    if (res.data.d1.length === 0) {
      card.append(el("p", "cf-empty cf-empty-quiet", "No D1 activity in the window."));
    } else {
      renderSortableTable(card, {
        columns: [
          { key: "name", label: "Database", text: (d) => d.name },
          { key: "rowsRead", label: "Rows read", numeric: true, text: (d) => formatCount(d.rowsRead), num: (d) => d.rowsRead },
          { key: "rowsWritten", label: "Rows written", numeric: true, text: (w) => formatCount(w.rowsWritten), num: (w) => w.rowsWritten },
          { key: "queries", label: "Queries", numeric: true, text: (d) => formatCount(d.readQueries), num: (d) => d.readQueries },
          { key: "rowsPerQuery", label: "Rows/query", numeric: true, text: (d) => formatCount(d.rowsPerQuery), num: (d) => d.rowsPerQuery },
        ],
        rows: res.data.d1,
        initial: { key: "rowsRead", dir: "desc" },
        rowClass: (d) => {
          const lv = levelForD1(d.rowsRead);
          return lv === "critical" ? "cf-level-critical" : lv === "warning" ? "cf-level-warning" : "";
        },
      });
    }
    // Repaint through fadeSwap so refreshed content fades in without a jump.
    const view = el("div");
    view.append(bar, card);
    fadeSwap(host, view);
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
    const bar = el("div", "cf-dash-bar");
    const line = statusLine(totalAgeSec(res.ageSec, res.data.cache), { cached: Boolean(res.data.cache?.stale), demo: res.demo });
    const refresh = refreshButton(async () => renderZones(host, { refresh: true }));
    bar.append(line, refresh);

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
        initial: { key: "uncached", dir: "desc" },
        rowClass: (z) => {
          const lv = zoneAttentionLevel(z.uncached, z.missPct);
          return lv === "critical" ? "cf-level-critical" : lv === "warning" ? "cf-level-warning" : "";
        },
      });
    }
    // Repaint through fadeSwap so refreshed content fades in without a jump.
    const view = el("div");
    view.append(bar, card);
    fadeSwap(host, view);
  });
}
