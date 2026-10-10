// Workers as a real React view (P-05a, docs/planning-mode/
// 2026-10-10-pager-react-rebuild.md): replaces the imperative bridge to
// tables.ts's renderWorkers. The DOM structure and CSS classes are the
// vanilla view's; the pure helpers (format.ts, sort prefs, usage-share
// math, dashboard.ts levelForErrorPct, worker_profile.ts cpuMs) are
// imported verbatim. The section masthead owns the page title — the view
// keeps only the "(24h)" card subtitle.

import { useCallback, useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { api, LoginExpiredError, type Billing, type FetchResult, type Summary, type WorkerRow } from "../../api";
import { levelForErrorPct } from "../../dashboard";
import { formatCount, formatPct } from "../../format";
import { usageSharePct } from "../../tables";
import { workerHref } from "../../routes";
import { cpuMs } from "../../worker_profile";
import DataTable, { type DataTableColumn } from "../components/DataTable";

/** Injectable fetch shape (tests pass a stub instead of the shared
 *  ApiClient singleton, exactly like worker_profile.test.ts). */
export type ViewFetcher = <T>(endpoint: string, opts?: { refresh?: boolean; onRevalidate?: (data: T) => void }) => Promise<FetchResult<T>>;

const WORKER_COLUMNS: DataTableColumn<WorkerRow>[] = [
  // UI-3: the script cell taps through to the worker's profile page.
  {
    key: "script",
    label: "Script",
    text: (w) => w.script,
    cell: (w) => <a className="cf-link" href={workerHref(w.script)}>{w.script}</a>,
  },
  { key: "requests", label: "Requests", numeric: true, text: (w) => formatCount(w.requests), num: (w) => w.requests },
  { key: "errors", label: "Errors", numeric: true, text: (w) => formatCount(w.errors), num: (w) => w.errors },
  { key: "errorPct", label: "Error %", numeric: true, text: (w) => formatPct(w.errorPct, 2), num: (w) => w.errorPct },
  { key: "cpuP50", label: "CPU p50", numeric: true, text: (w) => cpuMs(w.cpuP50Ms), num: (w) => w.cpuP50Ms },
  { key: "cpuP99", label: "CPU p99", numeric: true, text: (w) => cpuMs(w.cpuP99Ms), num: (w) => w.cpuP99Ms },
];

export default function WorkersView({ refreshSeq = 0, fetcher }: { refreshSeq?: number; fetcher?: ViewFetcher }): React.JSX.Element {
  // The Retry re-render must go through the SAME fetcher (injected or the
  // shared client) — hence the closure rather than a bare method reference.
  const fetchJson: ViewFetcher = fetcher ?? ((endpoint, opts) => api.fetchJson(endpoint, opts));
  const [summary, setSummary] = useState<FetchResult<Summary> | null>(null);
  // Allowance for the graded bars: Workers Paid request allowance from the
  // billing cache — one cached fetch, absent bars on failure.
  const [included, setIncluded] = useState(0);
  const [loadError, setLoadError] = useState<{ message: string; expired: boolean } | null>(null);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [retryTick, setRetryTick] = useState(0);
  const reduced = useReducedMotion();

  // A background revalidation repaints the view with the fresh copy
  // (BUG-057 defect 2) instead of leaving hours-old data on screen.
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
        // A failed refresh keeps the last good data (BUG-057 defects 2/3);
        // only a first (skeleton) load shows the error/retry state.
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
        setIncluded(b.data.products.find((p) => p.id === "workers.requests")?.included ?? 0);
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
        <h2>Could not load the Workers list</h2>
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
        {summary.data.workers.length === 0 ? (
          <p className="cf-empty cf-empty-quiet">No Worker traffic in the window.</p>
        ) : (
          <DataTable
            columns={WORKER_COLUMNS}
            rows={summary.data.workers}
            initial={{ key: "requests", dir: "desc" }}
            sortPrefId="workers"
            rowKey={(w) => w.script}
            usageBar={included > 0 ? (w) => ({ pctOfAllowance: usageSharePct(w.requests, included) }) : undefined}
            // Whole-row tap-through to the worker profile (UI-3).
            onRowClick={(w) => {
              window.location.hash = workerHref(w.script).slice(1);
            }}
            rowClass={(w) => {
              const lv = levelForErrorPct(w.errorPct);
              return lv === "critical" ? "cf-level-critical" : lv === "warning" ? "cf-level-warning" : "";
            }}
          />
        )}
      </section>
    </motion.div>
  );
}
