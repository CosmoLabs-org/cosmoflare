// Zone profile view (profiles program instance 4): one zone's metadata,
// SSL mode, and DNSSEC — the drill-in the Zones table's rows point at,
// mirroring WorkerProfileView. The 24h requests/uncached analytics ride
// the summary the ZonesView table already renders (no second analytics
// call); the configuration side-fetch (OverviewView's email pattern) must
// not gate the analytics card — a failed/absent side fetch silently leaves
// the section cards unrendered. Sections render only with data; an omitted
// section is data absent, never invented. The masthead owns the page
// title, so the cards drop name headings.

import { useCallback, useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { api, LoginExpiredError, type RulesQuotaPayload, type Summary, type ZoneProfilePayload, type ZoneRow } from "../../api";
import { formatCount, formatPct } from "../../format";
import type { ViewFetcher } from "./WorkersView";

/** One labeled field: label left, value in the data face (DomainsView's
 *  Field row shape). */
function Field({ label, value }: { label: string; value: string }): React.JSX.Element {
  return (
    <div className="cf-domain-field">
      <span className="cf-domain-field-label">{label}</span>
      <span className="cf-domain-field-value">{value}</span>
    </div>
  );
}

/** Display-only ISO date, "—" when absent. */
function createdDate(createdOn: string | null): string {
  return createdOn === null ? "—" : createdOn.slice(0, 10);
}

/** One stat row in the data face (WorkerProfileView's StatRow shape). */
function StatRow({ label, value, valueClass }: { label: string; value: string; valueClass?: string }): React.JSX.Element {
  return (
    <div className="cf-domain-field">
      <span className="cf-domain-field-label">{label}</span>
      <span className={`cf-domain-field-value cf-stat-value${valueClass ? ` ${valueClass}` : ""}`}>{value}</span>
    </div>
  );
}

const defaultFetchJson: ViewFetcher = (endpoint, opts) => api.fetchJson(endpoint, opts);

export default function ZoneProfileView({ name, refreshSeq = 0, fetcher }: {
  name: string;
  refreshSeq?: number;
  fetcher?: ViewFetcher;
}): React.JSX.Element {
  const fetchJson: ViewFetcher = fetcher ?? defaultFetchJson;
  const [zone, setZone] = useState<ZoneRow | null>(null);
  const [found, setFound] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [loadError, setLoadError] = useState<{ message: string; expired: boolean } | null>(null);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [retryTick, setRetryTick] = useState(0);
  const [profile, setProfile] = useState<ZoneProfilePayload | null>(null);
  const [quota, setQuota] = useState<RulesQuotaPayload | null>(null);
  const reduced = useReducedMotion();

  const repaint = useCallback((fresh: Summary): void => {
    const z = fresh.zones.find((row) => row.zone === name);
    setZone(z ?? null);
    setFound(z !== undefined);
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
  // must not gate the analytics card — a failed/absent zone-profile fetch
  // silently leaves the section cards unrendered.
  useEffect(() => {
    let cancelled = false;
    void fetchJson<ZoneProfilePayload>(`api/zone-profile?name=${encodeURIComponent(name)}`, { refresh: refreshSeq > 0 }).then(
      (res) => {
        if (!cancelled) setProfile(res.data);
      },
      () => undefined,
    );
    return () => {
      cancelled = true;
    };
  }, [name, refreshSeq, retryTick, fetchJson]);

  // Side fetch (same silent-failure contract): the rule-quota payload must
  // not gate the profile — an absent payload leaves the Rules card off.
  useEffect(() => {
    let cancelled = false;
    void fetchJson<RulesQuotaPayload>(`api/rules-quota?name=${encodeURIComponent(name)}`, { refresh: refreshSeq > 0 }).then(
      (res) => {
        if (!cancelled) setQuota(res.data);
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
        <h2>Could not load the zone profile</h2>
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

      <a className="cf-btn cf-btn-ghost cf-profile-back" href="#/zones">All zones</a>

      <p className="cf-row-detail">One zone's last 24h traffic plus its configuration. Use the browser back or Zones to return.</p>

      {found && zone !== null ? (
        <section className="cf-card">
          <p className="cf-row-detail">24h window · from the summary's zone analytics</p>
          <StatRow label="Requests" value={formatCount(zone.total)} />
          <StatRow label="Uncached" value={formatCount(zone.uncached)} />
          <StatRow label="Miss %" value={zone.missPct === null ? "—" : formatPct(zone.missPct, 2)} />
        </section>
      ) : (
        <section className="cf-card">
          {/* A data absence is not an error — the same quiet tone as the
              worker profile's not-found card. */}
          <h2>Zone not found in the 24h window</h2>
          <p className="cf-empty cf-empty-quiet">{`No 24h analytics row for ${name}.`}</p>
        </section>
      )}

      {profile?.metadata ? (
        <section className="cf-card">
          <h2>Zone metadata</h2>
          <Field label="Plan" value={profile.metadata.planName || "—"} />
          <Field label="Status" value={profile.metadata.status || "—"} />
          <Field label="Created" value={createdDate(profile.metadata.createdOn)} />
          <Field label="Type" value={profile.metadata.type || "—"} />
          <Field label="Paused" value={profile.metadata.paused ? "yes" : "no"} />
          <Field label="Nameservers" value={profile.metadata.nameServers.length > 0 ? profile.metadata.nameServers.join(", ") : "—"} />
        </section>
      ) : null}

      {profile?.sslMode ? (
        <section className="cf-card">
          <h2>TLS/SSL</h2>
          <Field label="SSL mode" value={profile.sslMode} />
        </section>
      ) : null}

      {profile?.dnssec ? (
        <section className="cf-card">
          <h2>DNSSEC</h2>
          <Field label="Status" value={profile.dnssec.status ?? "—"} />
          {profile.dnssec.algorithm !== null ? <Field label="Algorithm" value={profile.dnssec.algorithm} /> : null}
        </section>
      ) : null}

      {quota !== null && Array.isArray(quota.families) && quota.families.length > 0 ? (
        <section className="cf-card">
          <h2>Rules</h2>
          <p className="cf-row-detail">Rule inventory against the plan's free allowances.</p>
          {quota.families.map((f) => {
            const atCap = f.authorized && f.used >= f.allowance;
            return (
              <div className="cf-quota-family" key={f.family}>
                <Field label={f.label} value={f.authorized ? `${f.used} / ${f.allowance}` : "scope needed"} />
                {atCap ? (
                  <p className="cf-empty cf-level-warning-text">{`At the free allowance — the next ${f.label.replace(/s$/, "")} needs a plan upgrade or a slot freed.`}</p>
                ) : null}
                {!f.authorized ? (
                  <p className="cf-empty cf-empty-quiet">Needs the Zone Rulesets read scope on the API token.</p>
                ) : null}
                {f.rules.map((r) => (
                  <p className="cf-row-detail" key={r.name}>{r.name}</p>
                ))}
              </div>
            );
          })}
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
        <h2>About this zone</h2>
        <p className="cf-empty cf-empty-quiet">
          Sections render from the live API only — an absent section means no data, never an invented number.
        </p>
        <a
          className="cf-link"
          href={`https://dash.cloudflare.com/?to=/:account/${encodeURIComponent(name)}`}
          target="_blank"
          rel="noreferrer"
        >
          Open in Cloudflare →
        </a>
      </section>
    </motion.div>
  );
}
