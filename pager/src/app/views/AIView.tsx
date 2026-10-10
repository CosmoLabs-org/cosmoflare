// AI Gateway usage as a real React view (BR-03, ROAD-112 phase 2): one card
// per gateway (sorted as delivered by the worker) with the 24h request count
// in the head and the MTD total as a muted subtitle, the gateway's models as
// ranked rows (rank badge, model + provider, requests right; token in/out as
// a second muted line when the daily field probe found them). Every dollar
// figure carries a muted "est." suffix — costBasis is gateway-estimate, so
// these are never presented as real bill numbers (real-numbers rule). The
// masthead owns the page title.

import { useCallback, useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { api, LoginExpiredError, type FetchResult } from "../../api";
import { formatCount, formatUsd } from "../../format";
import type { AIGatewaysPayload, AIGatewayRow, AIModelRow } from "../../ai";
import type { ViewFetcher } from "./WorkersView";

/** One ranked model row: rank badge, model (+ provider chip when the gateway
 *  mixes providers), the provider line, the optional token line, and the
 *  data face right — requests count, with the estimate-labeled cost above it
 *  when the probe found cost fields. */
function ModelRow({ m, rank, mixed }: { m: AIModelRow; rank: number; mixed: boolean }): React.JSX.Element {
  return (
    <li className="cf-consumer">
      <span className="cf-consumer-rank">{String(rank)}</span>
      <span className="cf-consumer-who">
        <span className="cf-consumer-name">
          {m.model}
          {mixed ? <span className="cf-chip">{m.provider}</span> : null}
        </span>
        <span className="cf-consumer-project">{m.provider}</span>
        {m.tokensIn !== undefined || m.tokensOut !== undefined ? (
          <span className="cf-consumer-project">{`${formatCount(m.tokensIn ?? 0)} in / ${formatCount(m.tokensOut ?? 0)} out`}</span>
        ) : null}
      </span>
      <span className="cf-consumer-sharewrap">
        {m.costUsd !== undefined ? (
          <span className="cf-row-detail">
            {formatUsd(m.costUsd)} <span className="cf-row-detail">est.</span>
          </span>
        ) : null}
        <span className="cf-consumer-share">{formatCount(m.requests)}</span>
      </span>
    </li>
  );
}

/** One gateway card: name + 24h requests in the head, MTD total as the muted
 *  subtitle, then the model rows ranked as delivered (already sorted by
 *  requests desc server-side). */
function GatewayCard({ gw }: { gw: AIGatewayRow }): React.JSX.Element {
  const mixed = new Set(gw.models.map((m) => m.provider)).size > 1;
  return (
    <section className="cf-card">
      <div className="cf-product-head">
        <strong className="cf-product-name">{gw.name}</strong>
        <span className="cf-product-status" title="Requests, 24h">{formatCount(gw.requests24h)}</span>
      </div>
      <p className="cf-row-detail">{`MTD ${formatCount(gw.requestsMtd)} requests`}</p>
      {gw.models.length === 0 ? (
        <p className="cf-empty cf-empty-quiet">No model traffic in the last 24h.</p>
      ) : (
        <ol className="cf-consumers-list cf-consumers-ranked">
          {gw.models.map((m, i) => (
            <ModelRow key={`${m.provider}/${m.model}`} m={m} rank={i + 1} mixed={mixed} />
          ))}
        </ol>
      )}
    </section>
  );
}

export default function AIView({ refreshSeq = 0, fetcher }: { refreshSeq?: number; fetcher?: ViewFetcher }): React.JSX.Element {
  const fetchJson: ViewFetcher = fetcher ?? ((endpoint, opts) => api.fetchJson(endpoint, opts));
  const [data, setData] = useState<FetchResult<AIGatewaysPayload> | null>(null);
  const [loadError, setLoadError] = useState<{ message: string; expired: boolean } | null>(null);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [retryTick, setRetryTick] = useState(0);
  const reduced = useReducedMotion();

  // A background revalidation repaints the view with the fresh copy instead
  // of leaving hours-old data on screen (the BUG-057 pattern).
  const repaint = useCallback((fresh: AIGatewaysPayload): void => {
    setData({ data: fresh, source: "network", ageSec: 0, demo: false });
    setRefreshError(null);
  }, []);

  useEffect(() => {
    const refresh = refreshSeq > 0;
    let cancelled = false;
    void (async () => {
      try {
        const res = await fetchJson<AIGatewaysPayload>("api/ai", { refresh, onRevalidate: repaint });
        if (cancelled) return;
        setData(res);
        setRefreshError(null);
        setLoadError(null);
      } catch (err) {
        if (cancelled) return;
        // A failed refresh keeps the last good data and surfaces the error
        // text; only a first load shows the error/retry state.
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
        <h2>Could not load AI Gateway usage</h2>
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

      {d.gateways.length === 0 ? (
        <section className="cf-card">
          <p className="cf-empty cf-empty-quiet">
            No AI Gateways on this account yet — create one per project (R2 → Manage R2 API Tokens flow is the analog: AI Gateway → Create Gateway).
          </p>
        </section>
      ) : (
        d.gateways.map((gw) => <GatewayCard key={gw.id} gw={gw} />)
      )}

      {/* Probe notes: the token/cost honesty line(s) from the collector. */}
      {d.notes.map((n, i) => (
        <p className="cf-empty cf-empty-quiet" key={i}>{n}</p>
      ))}

      {/* Partial-load warnings from the collector (the Telemetry-gaps pattern). */}
      {d.errors.length > 0 ? (
        <section className="cf-card cf-level-warning">
          <h2>Partial data</h2>
          {d.errors.map((e, i) => (
            <p className="cf-row-detail" key={i}>{e}</p>
          ))}
        </section>
      ) : null}
    </motion.div>
  );
}
