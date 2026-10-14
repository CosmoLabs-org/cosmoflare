// Durable Objects namespace profile (profiles program instance 3): the
// React drill-in for one DO namespace — the 24h analytics row exactly as
// the Durable Objects table shows it. Mirrors WorkerProfileView's shape:
// the statRow visual rules (cf-stat-value, zero-errors green, error-rate
// tint), the back control, the quiet not-found state. The masthead owns
// the page title, so the stats card keeps the subtitle only. A failed side
// fetch degrades to quiet sections — data absent, never an invented number.

import { useCallback, useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { api, LoginExpiredError, type DoProfilePayload } from "../../api";
import { levelForErrorPct } from "../../dashboard";
import { formatCount, formatPct } from "../../format";
import type { ViewFetcher } from "./WorkersView";

/** One stat indicator row — the same classes as WorkerProfileView's
 *  StatRow (kept local so each view owns its copy, as the sibling does). */
function StatRow({ label, value, valueClass }: { label: string; value: string; valueClass?: string }): React.JSX.Element {
  return (
    <div className="cf-domain-field">
      <span className="cf-domain-field-label">{label}</span>
      <span className={`cf-domain-field-value cf-stat-value${valueClass ? ` ${valueClass}` : ""}`}>{value}</span>
    </div>
  );
}

const defaultFetchJson: ViewFetcher = (endpoint, opts) => api.fetchJson(endpoint, opts);

export default function DoProfileView({ name, refreshSeq = 0, fetcher }: {
  name: string;
  refreshSeq?: number;
  fetcher?: ViewFetcher;
}): React.JSX.Element {
  const fetchJson: ViewFetcher = fetcher ?? defaultFetchJson;
  const [profile, setProfile] = useState<DoProfilePayload | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [loadError, setLoadError] = useState<{ message: string; expired: boolean } | null>(null);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [retryTick, setRetryTick] = useState(0);
  const reduced = useReducedMotion();

  const repaint = useCallback((fresh: DoProfilePayload): void => {
    setProfile(fresh);
    setRefreshError(null);
  }, [name]);

  useEffect(() => {
    const refresh = refreshSeq > 0;
    let cancelled = false;
    void (async () => {
      try {
        const res = await fetchJson<DoProfilePayload>(`api/do-profile?name=${encodeURIComponent(name)}`, { refresh, onRevalidate: repaint });
        if (cancelled) return;
        setProfile(res.data);
        setLoaded(true);
        setLoadError(null);
        setRefreshError(null);
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
    return () => {
      cancelled = true;
    };
  }, [name, refreshSeq, retryTick, repaint, fetchJson]);

  if (loadError !== null) {
    if (loadError.expired) {
      return <p className="cf-empty cf-login-expired">{loadError.message}</p>;
    }
    return (
      <section className="cf-card cf-level-warning">
        <h2>Could not load the namespace profile</h2>
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

  if (!loaded || profile === null) {
    return (
      <div className="cf-loading">
        <div className="cf-skeleton" />
        <div className="cf-skeleton" />
        <div className="cf-skeleton" />
      </div>
    );
  }

  return (
    <motion.div
      className="cf-profile"
      initial={reduced ? false : { opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={reduced ? { duration: 0 } : { duration: 0.2, ease: "easeOut" }}
    >
      {refreshError !== null ? (
        <p className="cf-empty cf-level-warning-text" role="alert">{`Refresh failed — ${refreshError}`}</p>
      ) : null}

      <a className="cf-btn cf-btn-ghost cf-profile-back" href="#/durable-objects">All namespaces</a>

      <p className="cf-row-detail">One namespace's last 24h. Use the browser back or Durable Objects to return.</p>

      {profile.found ? (
        <section className="cf-card">
          <p className="cf-row-detail">24h window · from Durable Objects analytics</p>
          <StatRow label="Requests" value={formatCount(profile.requests)} />
          <StatRow
            label="Errors"
            value={formatCount(profile.errors)}
            valueClass={profile.errors === 0 ? "cf-ok-text" : "cf-level-critical-text"}
          />
          <StatRow
            label="Error rate"
            value={formatPct(profile.errorPct, 2)}
            valueClass={`cf-level-${levelForErrorPct(profile.errorPct)}-text`}
          />
          {profile.namespaceId !== null ? (
            <StatRow label="Namespace ID" value={profile.namespaceId} />
          ) : null}
        </section>
      ) : (
        <section className="cf-card">
          {/* A data absence is not an error — the same quiet tone. */}
          <h2>Namespace not found</h2>
          <p className="cf-empty cf-empty-quiet">{`No 24h analytics row for ${name}.`}</p>
        </section>
      )}

      {profile.collectionErrors?.length ? (
        <section className="cf-card">
          <h2>Collection notes</h2>
          {profile.collectionErrors.map((e) => (
            <p className="cf-empty cf-empty-quiet" key={e}>{e}</p>
          ))}
        </section>
      ) : null}

      <section className="cf-card">
        <h2>About this namespace</h2>
        <p className="cf-empty cf-empty-quiet">
          Sections render from the live API only — an absent section means no data, never an invented number.
        </p>
        <a
          className="cf-link"
          href="https://dash.cloudflare.com/?to=/:account/workers/durable-objects"
          target="_blank"
          rel="noreferrer"
        >
          Open in Cloudflare →
        </a>
      </section>
    </motion.div>
  );
}
