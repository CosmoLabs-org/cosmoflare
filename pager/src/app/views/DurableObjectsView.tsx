// Durable Objects as a real React view (P-05a, docs/planning-mode/
// 2026-10-10-pager-react-rebuild.md): replaces the imperative bridge to
// durable_objects.ts's renderDurableObjects — every namespace with its 24h
// requests, errors and error rate, graded usage bars against the DO request
// allowance (cached billing fetch, bars drop on failure). The masthead owns
// the page title; the card keeps the "(24h)" subtitle.

import { useCallback, useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { api, LoginExpiredError, type Billing, type FetchResult } from "../../api";
import { levelForErrorPct } from "../../dashboard";
import { formatCount, formatPct } from "../../format";
import { usageSharePct } from "../../tables";
import type { DOPayload, DORow } from "../../durable_objects";
import DataTable, { type DataTableColumn } from "../components/DataTable";
import type { ViewFetcher } from "./WorkersView";

// The table columns match durable_objects.ts's renderInto exactly (the
// vanilla Column shape carries an HTMLElement cell; this view needs none).
const DO_COLUMNS: DataTableColumn<DORow>[] = [
  { key: "script", label: "Script", text: (r) => r.script },
  { key: "requests", label: "Requests", numeric: true, text: (r) => formatCount(r.requests), num: (r) => r.requests },
  { key: "errors", label: "Errors", numeric: true, text: (r) => formatCount(r.errors), num: (r) => r.errors },
  { key: "errorPct", label: "Error %", numeric: true, text: (r) => formatPct(r.errorPct, 2), num: (r) => r.errorPct },
];

export default function DurableObjectsView({ refreshSeq = 0, fetcher }: { refreshSeq?: number; fetcher?: ViewFetcher }): React.JSX.Element {
  const fetchJson: ViewFetcher = fetcher ?? ((endpoint, opts) => api.fetchJson(endpoint, opts));
  const [data, setData] = useState<FetchResult<DOPayload> | null>(null);
  // Graded bars: each script's requests as a share of the DO request
  // allowance from the cached billing payload; bars drop out on failure.
  const [included, setIncluded] = useState(0);
  const [loadError, setLoadError] = useState<{ message: string; expired: boolean } | null>(null);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [retryTick, setRetryTick] = useState(0);
  const reduced = useReducedMotion();

  const repaint = useCallback((fresh: DOPayload): void => {
    setData({ data: fresh, source: "network", ageSec: 0, demo: false });
    setRefreshError(null);
  }, []);

  useEffect(() => {
    const refresh = refreshSeq > 0;
    let cancelled = false;
    void (async () => {
      try {
        const res = await fetchJson<DOPayload>("api/durable-objects", { refresh, onRevalidate: repaint });
        if (cancelled) return;
        setData(res);
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
        setIncluded(b.data.products.find((p) => p.id === "do.requests")?.included ?? 0);
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
        <h2>Could not load Durable Objects</h2>
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

  if (data === null) {
    return (
      <div className="cf-loading">
        <div className="cf-skeleton" />
        <div className="cf-skeleton" />
        <div className="cf-skeleton" />
        <div className="cf-skeleton" />
      </div>
    );
  }

  const d = data.data;

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
        <p className="cf-row-detail">
          {d.errors.length > 0
            ? `(24h) · Partial load — ${d.errors[0]}`
            : `(24h) · ${d.namespaces} ${d.namespaces === 1 ? "namespace" : "namespaces"} · requests, errors and error rate per script`}
        </p>
        {d.rows.length === 0 ? (
          <p className="cf-empty cf-empty-quiet">No namespaces or traffic found.</p>
        ) : (
          <DataTable
            columns={DO_COLUMNS}
            rows={d.rows}
            initial={{ key: "requests", dir: "desc" }}
            sortPrefId="durable-objects"
            rowKey={(r) => r.script}
            usageBar={included > 0 ? (r) => ({ pctOfAllowance: usageSharePct(r.requests, included) }) : undefined}
            rowClass={(r) => {
              const lv = levelForErrorPct(r.errorPct);
              return lv === "critical" ? "cf-level-critical" : lv === "warning" ? "cf-level-warning" : "";
            }}
          />
        )}
      </section>
    </motion.div>
  );
}
