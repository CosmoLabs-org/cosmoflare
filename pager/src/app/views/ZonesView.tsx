// Zones as a real React view (P-05a, docs/planning-mode/
// 2026-10-10-pager-react-rebuild.md): replaces the imperative bridge to
// tables.ts's renderZones — same table family, same classes. The stacked
// byStatus bars and the shared legend reuse tables.ts's byStatusStack /
// byStatusLegend builders verbatim through a small DOM slot (the exported
// builders keep serving the vanilla renderer too). The masthead owns the
// page title; the card keeps the "(24h)" subtitle.

import { useCallback, useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { api, LoginExpiredError, type FetchResult, type Summary, type ZoneRow } from "../../api";
import { zoneAttentionLevel } from "../../attention";
import { formatCount, formatPct } from "../../format";
import { byStatusLegend, byStatusStack } from "../../tables";
import DataTable, { type DataTableColumn } from "../components/DataTable";
import type { ViewFetcher } from "./WorkersView";

/** DomSlot mounts an imperatively-built HTMLElement (tables.ts's exported
 *  byStatus builders) into the React tree — the builders run once per
 *  render and their node is removed on cleanup. */
function DomSlot({ build }: { build: () => HTMLElement }): React.JSX.Element {
  return (
    <span
      ref={(host) => {
        // No dep array: `build` is a fresh closure every render, so the
        // slot repaints with its row — matching the vanilla repaint.
        if (!host) return;
        host.replaceChildren();
        host.append(build());
      }}
    />
  );
}

const ZONE_COLUMNS: DataTableColumn<ZoneRow>[] = [
  { key: "zone", label: "Zone", text: (z) => z.zone },
  { key: "total", label: "Requests", numeric: true, text: (z) => formatCount(z.total), num: (z) => z.total },
  { key: "uncached", label: "Uncached", numeric: true, text: (z) => formatCount(z.uncached), num: (z) => z.uncached },
  { key: "missPct", label: "Miss %", numeric: true, text: (z) => (z.missPct === null ? "—" : formatPct(z.missPct)), num: (z) => z.missPct },
  {
    key: "byStatus",
    label: "By status",
    text: (z) => Object.entries(z.byStatus ?? {}).map(([k, v]) => `${k} ${v}`).join(" "),
    cell: (z) => (
      <div className="cf-zonestack">
        {z.byStatus ? <DomSlot build={() => byStatusStack(z.byStatus as Record<string, number>)} /> : null}
        {z.byStatus ? <DomSlot build={() => byStatusLegend(z.byStatus as Record<string, number>)} /> : null}
      </div>
    ),
  },
];

export default function ZonesView({ refreshSeq = 0, fetcher }: { refreshSeq?: number; fetcher?: ViewFetcher }): React.JSX.Element {
  const fetchJson: ViewFetcher = fetcher ?? ((endpoint, opts) => api.fetchJson(endpoint, opts));
  const [summary, setSummary] = useState<FetchResult<Summary> | null>(null);
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
        <h2>Could not load the zones list</h2>
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

  // Zero-traffic zones drop out (vanilla parity); the rest sort by
  // uncached requests first.
  const rows = summary.data.zones.filter((z) => z.total > 0);

  // One shared byStatus legend for phones: per-row legends are hidden in
  // compact rows, so aggregate every zone's statuses into one line.
  const agg: Record<string, number> = {};
  for (const z of summary.data.zones) {
    for (const [k, v] of Object.entries(z.byStatus ?? {})) agg[k] = (agg[k] ?? 0) + v;
  }
  const sharedLegend = (
    <DomSlot
      build={() => {
        const legend = byStatusLegend(agg);
        legend.classList.add("cf-shared-legend");
        return legend;
      }}
    />
  );

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
        {rows.length === 0 ? (
          <p className="cf-empty cf-empty-quiet">No zone traffic in the window.</p>
        ) : (
          <DataTable
            columns={ZONE_COLUMNS}
            rows={rows}
            initial={{ key: "uncached", dir: "desc" }}
            sortPrefId="zones"
            rowKey={(z) => z.zone}
            sharedLegend={sharedLegend}
            rowClass={(z) => {
              const lv = zoneAttentionLevel(z.uncached, z.missPct);
              return lv === "critical" ? "cf-level-critical" : lv === "warning" ? "cf-level-warning" : "";
            }}
          />
        )}
      </section>
    </motion.div>
  );
}
