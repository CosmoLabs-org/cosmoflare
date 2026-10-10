// DataTable (P-05a, docs/planning-mode/2026-10-10-pager-react-rebuild.md):
// the sortable/filterable/collapsible table as one React component — the
// port of tables.ts's renderSortableTable. Same DOM, same CSS classes
// (cf-table, cf-sort-btn, cf-sort-icon, cf-mobile-list, cf-row-link,
// cf-mini-bar, cf-show-all), same sort/filter/collapse semantics, with the
// pure comparators (sort.ts), the cross-reload sort preference
// (readSortPref/writeSortPref) and the SVG sort glyph constants imported
// verbatim from tables.ts — never re-implemented.

import { useMemo, useState } from "react";
import { cmpRows, cmpText, type SortDir } from "../../sort";
import {
  ARROW_DOWN_SVG,
  ARROW_UP_SVG,
  SORT_NEUTRAL_SVG,
  levelForUsageBar,
  readSortPref,
  writeSortPref,
} from "../../tables";

/** One sortable column — the React mirror of tables.ts's Column, with the
 *  rich cell returning a ReactNode instead of an HTMLElement. */
export interface DataTableColumn<T> {
  key: string;
  label: string;
  /** Right-align numeric columns. */
  numeric?: boolean;
  /** Cell text (used for sorting, filtering and the mobile copy). */
  text: (row: T) => string;
  /** Numeric sort value; omit for text columns. */
  num?: (row: T) => number | null;
  /** Rich cell content; default is the text of `text`. */
  cell?: (row: T) => React.ReactNode;
}

export interface DataTableProps<T> {
  columns: DataTableColumn<T>[];
  rows: T[];
  /** Default sort until the user (or a stored preference) picks one. */
  initial: { key: string; dir: SortDir };
  /** Stable per-table id ("workers"/"d1"/"zones") — persists the chosen
   *  sort across reloads via localStorage (FEAT-pRC6EDA). */
  sortPrefId?: string;
  /** Per-row level class (cf-level-warning/critical row tinting). */
  rowClass?: (row: T) => string;
  /** Per-row graded usage bar; null/undefined hides it for that row. */
  usageBar?: (row: T) => { pctOfAllowance: number } | null;
  /** Whole-row tap-through (operator 2026-10-10): the row navigates like
   *  its primary link — a bigger tap target than the name cell alone. */
  onRowClick?: (row: T) => void;
  /** One legend for the whole table (e.g. the zones byStatus colors),
   *  shown under the filter/sort controls on phones (<600px) only. */
  sharedLegend?: React.ReactNode;
  /** Stable React key per row; defaults to the row's index. */
  rowKey?: (row: T) => string;
}

// COLLAPSE_AFTER: lists longer than this render 10 rows + "Show all N" —
// the same rule tables.ts applies.
const COLLAPSE_AFTER = 10;

/** MiniUsageBar: a 4px graded bar — the row's share of the product
 *  allowance. Color follows the shared thresholds via the level classes;
 *  width caps at 100% visually, the title carries the true number. */
function MiniUsageBar({ pctOfAllowance }: { pctOfAllowance: number }): React.JSX.Element {
  const level = levelForUsageBar(pctOfAllowance);
  const title = `${pctOfAllowance.toFixed(pctOfAllowance >= 100 ? 0 : 1)}% of the allowance`;
  return (
    <div className={`cf-mini-bar cf-level-${level}`} role="img" aria-label={title}>
      <div className="cf-mini-fill" style={{ width: `${Math.min(100, Math.max(0, pctOfAllowance))}%` }} title={title} />
    </div>
  );
}

export default function DataTable<T>({
  columns,
  rows,
  initial,
  sortPrefId,
  rowClass,
  usageBar,
  onRowClick,
  sharedLegend,
  rowKey,
}: DataTableProps<T>): React.JSX.Element {
  // Sort state seeds from the stored preference (FEAT-pRC6EDA), then stays
  // in component state — the React equivalent of tables.ts's WeakMap on the
  // persistent host, since the component survives re-renders on its own.
  const [sort, setSort] = useState<{ key: string; dir: SortDir }>(() => {
    const stored = sortPrefId ? readSortPref(sortPrefId) : undefined;
    return stored ?? initial;
  });
  const [filterText, setFilterText] = useState("");
  const [showAll, setShowAll] = useState(false);

  const applySort = (key: string, dir: SortDir): void => {
    setSort({ key, dir });
    if (sortPrefId) writeSortPref(sortPrefId, key, dir);
  };
  const onHeaderClick = (col: DataTableColumn<T>): void => {
    if (sort.key === col.key) {
      applySort(col.key, sort.dir === "asc" ? "desc" : "asc");
    } else {
      applySort(col.key, col.numeric ? "desc" : "asc");
    }
  };

  // Sort + filter, recomputed only when the inputs change. Filter:
  // case-insensitive substring across every column's text.
  const sorted = useMemo(() => {
    const col = columns.find((c) => c.key === sort.key) ?? columns[0];
    const cmp = col.num ? cmpRows(col.num, sort.dir) : cmpText(col.text, sort.dir);
    let out = [...rows].sort(cmp);
    const q = filterText.trim().toLowerCase();
    if (q) out = out.filter((row) => columns.some((c) => c.text(row).toLowerCase().includes(q)));
    return { col, out };
  }, [columns, rows, sort, filterText]);

  // Collapse rule applies to the filtered rows (vanilla parity).
  const visible = showAll ? sorted.out : sorted.out.slice(0, COLLAPSE_AFTER);
  const truncated = !showAll && sorted.out.length > COLLAPSE_AFTER;

  // Mobile list mirrors (tables.ts mobileRowItem): line 1 name + the active
  // sort metric, line 2 the usage bar, line 3 the byStatus bar, then the
  // remaining numeric metrics in muted text.
  const primaryCol = sorted.col.numeric ? sorted.col : columns.find((c) => c.numeric);
  const statusCol = columns.find((c) => c.key === "byStatus" && c.cell);
  const shortLabel = (label: string): string => label.replace(/%\s*$/, "").trim().toLowerCase();

  const showAllButton = (
    <button type="button" className="cf-btn cf-show-all" onClick={() => setShowAll(true)}>
      {`Show all ${sorted.out.length}`}
    </button>
  );

  return (
    <>
      <div className="cf-table-bar">
        <input
          type="search"
          className="cf-filter"
          placeholder="Filter…"
          aria-label="Filter rows"
          value={filterText}
          onChange={(e) => {
            setFilterText(e.target.value);
            setShowAll(false);
          }}
        />
        <div className="cf-sort-controls">
          <select
            className="cf-sort-select"
            aria-label="Sort by"
            value={sort.key}
            onChange={(e) => {
              const col = columns.find((c) => c.key === e.target.value);
              if (col) applySort(col.key, col.numeric ? "desc" : "asc");
            }}
          >
            {columns.map((col) => (
              <option key={col.key} value={col.key}>{col.label}</option>
            ))}
          </select>
          <button
            type="button"
            className="cf-btn cf-btn-ghost cf-sort-dir"
            aria-label={sort.dir === "asc" ? "Sort ascending — tap for largest first" : "Sort descending — tap for smallest first"}
            aria-pressed={sort.dir === "desc"}
            onClick={() => applySort(sort.key, sort.dir === "asc" ? "desc" : "asc")}
          >
            {sort.dir === "asc" ? "↑" : "↓"}
          </button>
        </div>
      </div>
      {sharedLegend ?? null}
      <table className="cf-table">
        <thead>
          <tr>
            {columns.map((col) => (
              <th
                key={col.key}
                aria-sort={col.key === sort.key ? (sort.dir === "asc" ? "ascending" : "descending") : undefined}
              >
                <button
                  type="button"
                  className="cf-sort-btn"
                  aria-pressed={col.key === sort.key}
                  onClick={() => onHeaderClick(col)}
                >
                  {col.label}
                  <span
                    className={`cf-sort-icon${col.key === sort.key ? " is-active" : ""}`}
                    aria-hidden="true"
                    dangerouslySetInnerHTML={{
                      __html: col.key === sort.key
                        ? (sort.dir === "asc" ? ARROW_UP_SVG : ARROW_DOWN_SVG)
                        : SORT_NEUTRAL_SVG,
                    }}
                  />
                </button>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {visible.map((row, i) => {
            const cls = rowClass?.(row) ?? "";
            const barInfo = usageBar?.(row) ?? null;
            return (
              <tr
                key={rowKey?.(row) ?? i}
                className={`${cls}${onRowClick ? " cf-row-link" : ""}`.trim() || undefined}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
              >
                {columns.map((col, ci) => (
                  <td key={col.key} className={col.numeric ? "cf-num" : undefined}>
                    {col.cell ? col.cell(row) : col.text(row)}
                    {barInfo && ci === 0 ? <MiniUsageBar pctOfAllowance={barInfo.pctOfAllowance} /> : null}
                  </td>
                ))}
              </tr>
            );
          })}
          {truncated ? (
            <tr className="cf-showall-row">
              <td colSpan={columns.length}>{showAllButton}</td>
            </tr>
          ) : null}
        </tbody>
      </table>
      <ul className="cf-mobile-list">
        {visible.map((row, i) => {
          const barInfo = usageBar?.(row) ?? null;
          const meta = columns
            .filter((c) => c.numeric && c !== primaryCol)
            .map((c) => `${c.text(row)} ${shortLabel(c.label)}`)
            .join(" · ");
          return (
            <li key={rowKey?.(row) ?? i} className={rowClass?.(row) || undefined}>
              <div className="cf-mrow-top">
                <span className="cf-mrow-name">{columns[0].text(row)}</span>
                <span className="cf-mrow-primary">{primaryCol ? primaryCol.text(row) : ""}</span>
              </div>
              {barInfo ? <MiniUsageBar pctOfAllowance={barInfo.pctOfAllowance} /> : null}
              {statusCol?.cell ? <div className="cf-mrow-status">{statusCol.cell(row)}</div> : null}
              {meta ? <div className="cf-mrow-meta">{meta}</div> : null}
            </li>
          );
        })}
        {truncated ? <li className="cf-showall-row">{showAllButton}</li> : null}
      </ul>
    </>
  );
}
