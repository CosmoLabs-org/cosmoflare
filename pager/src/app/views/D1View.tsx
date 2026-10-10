// D1 as a real React view (P-05a, docs/planning-mode/
// 2026-10-10-pager-react-rebuild.md): replaces the imperative bridge to
// tables.ts's renderD1 — same table family, same classes, pure helpers
// imported verbatim (format.ts, usageSharePct, dashboard.ts levelForD1).
// The masthead owns the page title; the card keeps the "(24h)" subtitle.

import { useCallback, useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { api, LoginExpiredError, type Billing, type D1Row, type FetchResult, type Summary } from "../../api";
import { levelForD1 } from "../../dashboard";
import { formatCount } from "../../format";
import { usageSharePct } from "../../tables";
import DataTable, { type DataTableColumn } from "../components/DataTable";
import type { ViewFetcher } from "./WorkersView";

const D1_COLUMNS: DataTableColumn<D1Row>[] = [
  { key: "name", label: "Database", text: (d) => d.name },
  { key: "rowsRead", label: "Rows read", numeric: true, text: (d) => formatCount(d.rowsRead), num: (d) => d.rowsRead },
  { key: "rowsWritten", label: "Rows written", numeric: true, text: (d) => formatCount(d.rowsWritten), num: (d) => d.rowsWritten },
  { key: "queries", label: "Queries", numeric: true, text: (d) => formatCount(d.readQueries), num: (d) => d.readQueries },
  { key: "rowsPerQuery", label: "Rows/query", numeric: true, text: (d) => formatCount(d.rowsPerQuery), num: (d) => d.rowsPerQuery },
];

export default function D1View({ refreshSeq = 0, fetcher }: { refreshSeq?: number; fetcher?: ViewFetcher }): React.JSX.Element {
  const fetchJson: ViewFetcher = fetcher ?? ((endpoint, opts) => api.fetchJson(endpoint, opts));
  const [summary, setSummary] = useState<FetchResult<Summary> | null>(null);
  // Allowance for the graded bars: the D1 rows-read allowance from the
  // billing cache — one cached fetch, absent bars on failure.
  const [included, setIncluded] = useState(0);
  const [loadError, setLoadError] = useState<{ message: string; expired: boolean } | null>(null);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [retryTick, setRetryTick] = useState(0);
  const reduced = useReducedMotion();

  const repaint = useCallback((): void => {
    void fetchJson<Summary>("api/summary").then(
      (res) => {
        setSummary(res);
        setRefreshError(null);
      },
      () => undefined,
    );
  }, [fetchJson]);

  useEffect(() => {
    const refresh = refreshSeq > 0;
    let cancelled = false;
    void (async () => {
      try {
        const res = await fetchJson<Summary>("api/summary", { refresh, onRevalidate: repaint });
        if (cancelled) return;
        setSummary(res);
        setRefreshError(null);
        setLoadError(null);
      } catch (err) {
        if (cancelled) return;
        if (refresh) {
          setRefreshError(err instanceof Error ? err.message : String(err));
        } else {
          setLoadError({
            message: err instanceof Error ? err.message : String(err),
            expired: err instanceof LoginExpiredError,
          });
        }
      }
    })();
    // Billing feeds only the usage bars — its failure just drops them.
    void fetchJson<Billing>("api/billing").then(
      (b) => {
        if (cancelled) return;
        setIncluded(b.data.products.find((p) => p.id === "d1.rows_read")?.included ?? 0);
      },
      () => undefined,
    );
    return () => {
      cancelled = true;
    };
  }, [refreshSeq, retryTick, repaint, fetchJson]);

  if (loadError !== null) {
    if (loadError.expired) {
      return <p className="cf-empty cf-login-expired">{loadError.message}</p>;
    }
    return (
      <section className="cf-card cf-level-warning">
        <h2>Could not load the D1 list</h2>
        <p className="cf-row-detail">{loadError.message}</p>
        <button type="button" className="cf-btn" onClick={() => {
          setLoadError(null);
          setRetryTick((t) => t + 1);
        }}>
          Retry
        </button>
      </section>
    );
  }

  if (summary === null) {
    return (
      <div className="cf-loading">
        <div className="cf-skeleton" />
        <div className="cf-skeleton" />
        <div className="cf-skeleton" />
        <div className="cf-skeleton" />
      </div>
    );
  }

  return (
    <motion.div
      initial={reduced ? false : { opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={reduced ? { duration: 0 } : { duration: 0.2, ease: "easeOut" }}
    >
      {refreshError !== null ? (
        <p className="cf-empty cf-level-warning-text" role="alert">{`Refresh failed — ${refreshError}`}</p>
      ) : null}

      <section className="cf-card">
        <p className="cf-row-detail">(24h)</p>
        {summary.data.d1.length === 0 ? (
          <p className="cf-empty cf-empty-quiet">No D1 activity in the window.</p>
        ) : (
          <DataTable
            columns={D1_COLUMNS}
            rows={summary.data.d1}
            initial={{ key: "rowsRead", dir: "desc" }}
            sortPrefId="d1"
            rowKey={(d) => d.name}
            usageBar={included > 0 ? (d) => ({ pctOfAllowance: usageSharePct(d.rowsRead, included) }) : undefined}
            rowClass={(d) => {
              const lv = levelForD1(d.rowsRead);
              return lv === "critical" ? "cf-level-critical" : lv === "warning" ? "cf-level-warning" : "";
            }}
          />
        )}
      </section>
    </motion.div>
  );
}
