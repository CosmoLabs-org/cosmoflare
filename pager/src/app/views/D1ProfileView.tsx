// D1 database profile view (the worker drill-in's sibling): the 24h
// rows-read/written stats the summary already carries (D1View's data
// source), plus the database's REST metadata as a side fetch
// (/api/d1-profile — a failed side fetch leaves the analytics card
// untouched). Every section renders only with data — an absent section is
// data absent, never invented. The masthead owns the page title, so the
// stats card keeps only the subtitle.

import { useCallback, useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { api, LoginExpiredError, type D1ProfilePayload, type D1Row, type Summary } from "../../api";
import { formatCount, formatDateShort } from "../../format";
import type { ViewFetcher } from "./WorkersView";

/** One stat indicator row (the WorkerProfileView StatRow rules: the value
 *  renders in the data face, larger and tabular; counts stay neutral). */
function StatRow({ label, value, valueClass }: { label: string; value: string; valueClass?: string }): React.JSX.Element {
  return (
    <div className="cf-domain-field">
      <span className="cf-domain-field-label">{label}</span>
      <span className={`cf-domain-field-value cf-stat-value${valueClass ? ` ${valueClass}` : ""}`}>{value}</span>
    </div>
  );
}

/** "12.3 MB" from the metadata's file_size bytes; "—" when absent. */
function fileSizeLabel(bytes: number | null): string {
  if (bytes === null) return "—";
  const units = ["B", "kB", "MB", "GB", "TB"];
  let v = bytes;
  let u = 0;
  while (v >= 1024 && u < units.length - 1) {
    v /= 1024;
    u += 1;
  }
  return u === 0 ? `${v} ${units[u]}` : `${v.toFixed(1)} ${units[u]}`;
}

/** "auto ×2" — replication mode with its part count when present. */
function replicationLabel(r: { mode: string; parts: number | null }): string {
  return r.parts === null ? r.mode : `${r.mode} ×${r.parts}`;
}

const defaultFetchJson: ViewFetcher = (endpoint, opts) => api.fetchJson(endpoint, opts);

export default function D1ProfileView({ name, refreshSeq = 0, fetcher }: {
  name: string;
  refreshSeq?: number;
  fetcher?: ViewFetcher;
}): React.JSX.Element {
  const fetchJson: ViewFetcher = fetcher ?? defaultFetchJson;
  const [row, setRow] = useState<D1Row | null>(null);
  const [found, setFound] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [loadError, setLoadError] = useState<{ message: string; expired: boolean } | null>(null);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [retryTick, setRetryTick] = useState(0);
  const [profile, setProfile] = useState<D1ProfilePayload | null>(null);
  const reduced = useReducedMotion();

  const repaint = useCallback((fresh: Summary): void => {
    const r = fresh.d1.find((x) => x.name === name);
    setRow(r ?? null);
    setFound(r !== undefined);
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
        // only a first load shows the error/retry state.
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

  // Side fetch (OverviewView's email pattern): the metadata payload must
  // not gate the analytics card — a failed/absent d1-profile fetch
  // silently leaves the metadata card unrendered.
  useEffect(() => {
    let cancelled = false;
    void fetchJson<D1ProfilePayload>(`api/d1-profile?name=${encodeURIComponent(name)}`, { refresh: refreshSeq > 0 }).then(
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
        <h2>Could not load the D1 profile</h2>
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

      <a className="cf-btn cf-btn-ghost cf-profile-back" href="#/d1">All databases</a>

      <p className="cf-row-detail">One database's last 24h. Use the browser back or D1 to return.</p>

      {found && row !== null ? (
        <section className="cf-card">
          <p className="cf-row-detail">24h window · from D1 analytics</p>
          <StatRow label="Rows read" value={formatCount(row.rowsRead)} />
          <StatRow label="Rows written" value={formatCount(row.rowsWritten)} />
          <StatRow label="Queries" value={formatCount(row.readQueries)} />
        </section>
      ) : (
        <section className="cf-card">
          {/* A data absence is not an error — the same quiet tone. */}
          <h2>Database not found in the 24h window</h2>
          <p className="cf-empty cf-empty-quiet">{`No 24h analytics row for ${name}.`}</p>
        </section>
      )}

      {profile?.metadata ? (
        <section className="cf-card">
          <h2>Metadata</h2>
          <StatRow label="Name" value={profile.metadata.name} />
          {profile.metadata.createdAt !== null ? (
            <StatRow label="Created" value={formatDateShort(profile.metadata.createdAt)} />
          ) : null}
          {profile.metadata.version !== null ? (
            <StatRow label="Version" value={profile.metadata.version} />
          ) : null}
          {profile.metadata.region !== null ? (
            <StatRow label="Region" value={profile.metadata.region} />
          ) : null}
          {profile.metadata.numTables !== null ? (
            <StatRow label="Tables" value={formatCount(profile.metadata.numTables)} />
          ) : null}
          {profile.metadata.fileSize !== null ? (
            <StatRow label="File size" value={fileSizeLabel(profile.metadata.fileSize)} />
          ) : null}
          {profile.metadata.readReplication !== null ? (
            <StatRow label="Replication" value={replicationLabel(profile.metadata.readReplication)} />
          ) : null}
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
    </motion.div>
  );
}
