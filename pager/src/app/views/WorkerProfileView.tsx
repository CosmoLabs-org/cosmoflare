// Worker profile as a real React view (P-05a, docs/planning-mode/
// 2026-10-10-pager-react-rebuild.md): replaces the imperative bridge to
// worker_profile.ts's renderWorkerProfile. The 24h analytics row from
// /api/summary, exactly as the Workers table project shows it, plus the
// Cloudflare dashboard link out — no invented numbers. The worker-profile
// expansion (2026-10-13) adds the script's configuration as a side fetch
// (OverviewView's email pattern): bindings, cron triggers, routes, and
// recent versions — each section renders only with data (an omitted section
// is data absent, never invented), and a failed/absent side fetch leaves
// the analytics card untouched. The statRow visual rules (cf-stat-value,
// zero-errors green, error-rate tint, the "All workers" back control, the
// not-found quiet state) carry over; the masthead owns the page title, so
// the stats card drops its name heading and keeps the subtitle.

import { useCallback, useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { api, LoginExpiredError, type Summary, type WorkerProfilePayload, type WorkerRow } from "../../api";
import { levelForErrorPct } from "../../dashboard";
import { formatCount, formatPct } from "../../format";
import { cpuMs } from "../../worker_profile";
import type { ViewFetcher } from "./WorkersView";

/** One stat indicator row (operator 2026-10-10): the value renders in the
 *  data face, larger and tabular, with an optional level tint — zero
 *  errors reads green ("clean"), the error rate carries its threshold
 *  color, counts stay neutral. Same classes as worker_profile.ts statRow. */
function StatRow({ label, value, valueClass }: { label: string; value: string; valueClass?: string }): React.JSX.Element {
  return (
    <div className="cf-domain-field">
      <span className="cf-domain-field-label">{label}</span>
      <span className={`cf-domain-field-value cf-stat-value${valueClass ? ` ${valueClass}` : ""}`}>{value}</span>
    </div>
  );
}

/** Display-only ISO date from a version's created_on, "—" when absent. */
function versionDate(createdOn: string | null): string {
  return createdOn === null ? "—" : createdOn.slice(0, 10);
}

const defaultFetchJson: ViewFetcher = (endpoint, opts) => api.fetchJson(endpoint, opts);

export default function WorkerProfileView({ name, refreshSeq = 0, fetcher }: {
  name: string;
  refreshSeq?: number;
  fetcher?: ViewFetcher;
}): React.JSX.Element {
  const fetchJson: ViewFetcher = fetcher ?? defaultFetchJson;
  const [worker, setWorker] = useState<WorkerRow | null>(null);
  const [found, setFound] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [loadError, setLoadError] = useState<{ message: string; expired: boolean } | null>(null);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [retryTick, setRetryTick] = useState(0);
  const [profile, setProfile] = useState<WorkerProfilePayload | null>(null);
  const reduced = useReducedMotion();

  const repaint = useCallback((fresh: Summary): void => {
    const w = fresh.workers.find((row) => row.script === name);
    setWorker(w ?? null);
    setFound(w !== undefined);
    setRefreshError(null);
  }, [name]);

  useEffect(() => {
    const refresh = refreshSeq > 0;
    let cancelled = false;
    void (async () => {
      try {
        const res = await fetchJson<Summary>("api/summary", { refresh, onRevalidate: repaint });
        if (cancelled) return;
        repaint(res.data);
        setLoaded(true);
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
    return () => {
      cancelled = true;
    };
  }, [name, refreshSeq, retryTick, repaint, fetchJson]);

  // Side fetch (OverviewView's email pattern): the configuration payload
  // must not gate the analytics card — a failed/absent worker-profile fetch
  // silently leaves the section cards unrendered.
  useEffect(() => {
    let cancelled = false;
    void fetchJson<WorkerProfilePayload>(`api/worker-profile?name=${encodeURIComponent(name)}`, { refresh: refreshSeq > 0 }).then(
      (res) => {
        if (!cancelled) setProfile(res.data);
      },
      () => undefined,
    );
    return () => {
      cancelled = true;
    };
  }, [name, refreshSeq, retryTick, fetchJson]);

  if (loadError !== null) {
    if (loadError.expired) {
      return <p className="cf-empty cf-login-expired">{loadError.message}</p>;
    }
    return (
      <section className="cf-card cf-level-warning">
        <h2>Could not load the worker profile</h2>
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

  if (!loaded) {
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

      <a className="cf-btn cf-btn-ghost cf-profile-back" href="#/workers">All workers</a>

      <p className="cf-row-detail">One script's last 24h. Use the browser back or Workers to return.</p>

      {found && worker !== null ? (
        <section className="cf-card">
          <p className="cf-row-detail">24h window · from Workers Analytics</p>
          <StatRow label="Requests" value={formatCount(worker.requests)} />
          <StatRow
            label="Errors"
            value={formatCount(worker.errors)}
            valueClass={worker.errors === 0 ? "cf-ok-text" : "cf-level-critical-text"}
          />
          <StatRow
            label="Error rate"
            value={formatPct(worker.errorPct, 2)}
            valueClass={`cf-level-${levelForErrorPct(worker.errorPct)}-text`}
          />
          <StatRow label="CPU p50" value={cpuMs(worker.cpuP50Ms)} />
          <StatRow label="CPU p99" value={cpuMs(worker.cpuP99Ms)} />
        </section>
      ) : (
        <section className="cf-card">
          {/* A data absence is not an error — the same quiet tone. */}
          <h2>Worker not found</h2>
          <p className="cf-empty cf-empty-quiet">{`No 24h analytics row for ${name}.`}</p>
        </section>
      )}

      {profile?.bindings?.length ? (
        <section className="cf-card">
          <h2>Bindings</h2>
          {profile.bindings.map((b) => (
            <div className="cf-domain-field" key={`binding-${b.name}`}>
              <span className="cf-domain-field-label">{b.name}</span>
              <span className="cf-domain-field-value cf-stat-value">{b.type}</span>
            </div>
          ))}
        </section>
      ) : null}

      {profile?.cronTriggers?.length ? (
        <section className="cf-card">
          <h2>Cron triggers</h2>
          {profile.cronTriggers.map((c) => (
            <div className="cf-domain-field" key={`cron-${c}`}>
              <span className="cf-domain-field-label">Cron</span>
              <span className="cf-domain-field-value cf-stat-value">{c}</span>
            </div>
          ))}
        </section>
      ) : null}

      {profile?.routes?.length ? (
        <section className="cf-card">
          <h2>Routes</h2>
          {profile.routes.map((r) => (
            <div className="cf-domain-field" key={`route-${r.zone}-${r.pattern}`}>
              <span className="cf-domain-field-label">{r.zone}</span>
              <span className="cf-domain-field-value cf-stat-value">{r.pattern}</span>
            </div>
          ))}
        </section>
      ) : null}

      {profile?.versions?.length ? (
        <section className="cf-card">
          <h2>Versions</h2>
          {profile.versions.map((v) => (
            <div className="cf-domain-field" key={`version-${v.number}`}>
              <span className="cf-domain-field-label">{`v${v.number}`}</span>
              <span className="cf-domain-field-value cf-stat-value">{versionDate(v.createdOn)}</span>
            </div>
          ))}
        </section>
      ) : null}

      {profile?.errors?.length ? (
        <section className="cf-card">
          <h2>Collection notes</h2>
          {profile.errors.map((e) => (
            <p className="cf-empty cf-empty-quiet" key={e}>{e}</p>
          ))}
        </section>
      ) : null}

      <section className="cf-card">
        <h2>About this worker</h2>
        <p className="cf-empty cf-empty-quiet">
          Sections render from the live API only — an absent section means no data, never an invented number.
        </p>
        <a
          className="cf-link"
          href={`https://dash.cloudflare.com/?to=/:account/workers/services/view/${encodeURIComponent(name)}`}
          target="_blank"
          rel="noreferrer"
        >
          Open in Cloudflare →
        </a>
      </section>
    </motion.div>
  );
}
