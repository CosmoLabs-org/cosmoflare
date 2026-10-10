// Domains as a real React view (P-05b, docs/planning-mode/
// 2026-10-10-pager-react-rebuild.md): replaces the imperative bridge to
// domains.ts's renderDomains. Sorted list with expiry attention coloring
// and the "external" handling, and the master-detail drill-down INSIDE the
// component (the activeDomainId module variable becomes state) — profile
// sections Registration/DNS/Security/Renewal, enriched from
// /api/domains/detail when it lands (failures leave the base profile).
// The pure helpers (daysUntil, expiryLevel, sortDomains) and the
// DomainRecord/DomainDetailPayload types are imported verbatim from
// domains.ts. The section masthead owns the page title.

import { useCallback, useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { api, LoginExpiredError, type FetchResult, type Summary } from "../../api";
import { formatCount } from "../../format";
import {
  daysUntil,
  expiryLevel,
  sortDomains,
  type DomainDetailPayload,
  type DomainRecord,
  type DomainsPayload,
} from "../../domains";
import type { ViewFetcher } from "./WorkersView";

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

/** One field in a profile section grid — same classes as the vanilla
 *  domainProfile's field() builder. */
function Field({ label, value }: { label: string; value: string }): React.JSX.Element {
  return (
    <div className="cf-domain-field">
      <span className="cf-domain-field-label">{label}</span>
      <span className="cf-domain-field-value">{value}</span>
    </div>
  );
}

/** One profile section card: heading + a fields grid. */
function ProfileSection({ heading, children }: { heading: string; children: React.ReactNode }): React.JSX.Element {
  return (
    <section className="cf-card cf-profile-section">
      <h3 className="cf-profile-section-title">{heading}</h3>
      <div className="cf-profile-fields">{children}</div>
    </section>
  );
}

/** Enrichment rows from /api/domains/detail — each optional, appended into
 *  its section only when the fetch resolved it. */
interface DetailFields {
  originalNameServers?: string[];
  activatedOn?: string | null;
  ownerType?: string;
  dnssecStatus?: string | null;
  sslMode?: string | null;
}

/** The domain profile: full-page detail with the "All domains" back
 *  control. Paints from the list data immediately, then enriches from
 *  /api/domains/detail (original nameservers, DNSSEC, SSL mode, activation)
 *  and the cached summary (24h zone traffic) when they land — failures
 *  leave the base profile untouched. */
function DomainProfile({ domain, onBack, fetchJson }: {
  domain: DomainRecord;
  onBack: () => void;
  fetchJson: ViewFetcher;
}): React.JSX.Element {
  const [detail, setDetail] = useState<DetailFields | null>(null);
  const [traffic24h, setTraffic24h] = useState<number | null>(null);
  const reduced = useReducedMotion();

  useEffect(() => {
    let cancelled = false;
    // Enrichment: original NS, activation, DNSSEC, SSL — from the cached
    // per-zone endpoint; failures leave the base profile untouched.
    void fetchJson<DomainDetailPayload>(`api/domains/detail?id=${encodeURIComponent(domain.id)}`).then(
      (res) => {
        if (!cancelled) setDetail(res.data.zone);
      },
      () => undefined,
    );
    // Renewal traffic: the zone's 24h request total joined from the cached
    // summary by zone name (soft: absent when the summary has not loaded).
    void fetchJson<Summary>("api/summary").then(
      (res) => {
        if (cancelled) return;
        const z = res.data.zones.find((x) => x.zone === domain.name);
        if (z) setTraffic24h(z.total);
      },
      () => undefined,
    );
    return () => {
      cancelled = true;
    };
  }, [domain.id, domain.name, fetchJson]);

  const days = domain.expiresAt !== null ? daysUntil(domain.expiresAt) : null;
  const level = expiryLevel(days);

  return (
    <motion.div
      className="cf-profile"
      initial={reduced ? false : { opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={reduced ? { duration: 0 } : { duration: 0.2, ease: "easeOut" }}
    >
      <button type="button" className="cf-btn cf-btn-ghost cf-profile-back" onClick={onBack}>All domains</button>
      <p className="cf-profile-eyebrow">Domain profile</p>
      <h2 className="cf-profile-title">{domain.name}</h2>

      <section className={`cf-card cf-profile-status${level ? ` cf-level-${level}` : ""}`}>
        <p className="cf-profile-line">
          {domain.paused ? "Paused — Cloudflare is bypassed for this zone" : "Active — DNS resolves through Cloudflare"}
        </p>
        {days !== null ? (
          <p className={`cf-profile-line${level ? ` cf-level-${level}-text` : ""}`}>
            {days < 0 ? `Expired ${formatDate(domain.expiresAt!)}` : `Renews ${formatDate(domain.expiresAt!)} — ${days}d`}
          </p>
        ) : (
          <p className="cf-profile-line">Registered at an external registrar — renewal happens there</p>
        )}
      </section>

      {/* Same section order as the vanilla profile: status, Renewal,
          Registration, DNS, Security. */}
      <ProfileSection heading="Renewal">
        <Field label="Expiration" value={domain.expiresAt !== null ? formatDate(domain.expiresAt) : "External registrar"} />
        {days !== null ? (
          <Field
            label="Days away"
            value={days < 0 ? `${-days} days ago` : days === 0 ? "today" : `${days} days`}
          />
        ) : null}
        {traffic24h !== null ? <Field label="Traffic (24h)" value={formatCount(traffic24h)} /> : null}
      </ProfileSection>

      <ProfileSection heading="Registration">
        <Field label="Plan" value={domain.plan || "—"} />
        <Field label="Setup" value={domain.type === "partial" ? "Partial (CNAME)" : "Full (DNS on Cloudflare)"} />
        {domain.createdOn ? <Field label="Added to Cloudflare" value={formatDate(domain.createdOn)} /> : null}
        {detail?.activatedOn ? <Field label="Activated" value={formatDate(detail.activatedOn)} /> : null}
        {detail?.ownerType ? <Field label="Owner" value={detail.ownerType} /> : null}
      </ProfileSection>

      <ProfileSection heading="DNS">
        <Field
          label="Nameservers"
          value={domain.nameServers && domain.nameServers.length > 0 ? domain.nameServers.join("\n") : "—"}
        />
        {domain.modifiedOn ? <Field label="Last change" value={formatDate(domain.modifiedOn)} /> : null}
        {detail?.originalNameServers && detail.originalNameServers.length > 0 ? (
          // Historical: where DNS lived BEFORE the move to Cloudflare. Worded
          // so it never reads as current.
          <Field label="Previous nameservers (before Cloudflare)" value={detail.originalNameServers.join("\n")} />
        ) : null}
      </ProfileSection>

      <ProfileSection heading="Security">
        {domain.developmentMode ? <Field label="Development mode" value="on" /> : null}
        {detail?.dnssecStatus ? <Field label="DNSSEC" value={detail.dnssecStatus} /> : null}
        {detail?.sslMode ? <Field label="SSL mode" value={detail.sslMode} /> : null}
      </ProfileSection>
    </motion.div>
  );
}

/** One list row: name, expiry with attention coloring (or "external"),
 *  muted paused/status badges. */
function DomainRow({ d, nowMs, onOpen }: { d: DomainRecord; nowMs: number; onOpen: () => void }): React.JSX.Element {
  const days = d.expiresAt !== null ? daysUntil(d.expiresAt, nowMs) : null;
  const level = expiryLevel(days);
  return (
    <button type="button" className="cf-domain-row" onClick={onOpen}>
      <span className="cf-domain-name">{d.name}</span>
      <span className={`cf-domain-expiry${level ? ` cf-level-${level}-text` : ""}`}>
        {days === null
          ? "external"
          : days < 0
            ? `expired ${formatDate(d.expiresAt!)}`
            : `${days}d · ${formatDate(d.expiresAt!)}`}
      </span>
      <span className="cf-domain-badges">
        {d.paused ? <span className="cf-badge is-muted">paused</span> : null}
        {d.status !== "active" ? <span className="cf-badge is-muted">{d.status}</span> : null}
      </span>
    </button>
  );
}

export default function DomainsView({ refreshSeq = 0, fetcher }: { refreshSeq?: number; fetcher?: ViewFetcher }): React.JSX.Element {
  // The Retry re-render must go through the SAME fetcher (injected or the
  // shared client) — hence the closure rather than a bare method reference.
  const fetchJson: ViewFetcher = fetcher ?? ((endpoint, opts) => api.fetchJson(endpoint, opts));
  const [data, setData] = useState<FetchResult<DomainsPayload> | null>(null);
  // Master-detail drill-down INSIDE the component (the vanilla module's
  // activeDomainId, as React state — no route change).
  const [activeId, setActiveId] = useState<string | null>(null);
  const [loadError, setLoadError] = useState<{ message: string; expired: boolean } | null>(null);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [retryTick, setRetryTick] = useState(0);
  const reduced = useReducedMotion();

  // A background revalidation repaints the view with the fresh copy
  // (BUG-057 defect 2) instead of leaving hours-old data on screen.
  const repaint = useCallback((fresh: DomainsPayload): void => {
    setData({ data: fresh, source: "network", ageSec: 0, demo: false });
    setRefreshError(null);
  }, []);

  useEffect(() => {
    const refresh = refreshSeq > 0;
    let cancelled = false;
    void (async () => {
      try {
        const res = await fetchJson<DomainsPayload>("api/domains", { refresh, onRevalidate: repaint });
        if (cancelled) return;
        setData(res);
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
        <h2>Could not load domains</h2>
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

  const payload = data.data;
  const active = activeId !== null ? payload.domains.find((d) => d.id === activeId) ?? null : null;

  if (active !== null) {
    return (
      <>
        {refreshError !== null ? (
          <p className="cf-empty cf-level-warning-text" role="alert">{`Refresh failed — ${refreshError}`}</p>
        ) : null}
        <DomainProfile domain={active} onBack={() => setActiveId(null)} fetchJson={fetchJson} />
      </>
    );
  }

  // Soonest expiry first, no-expiry domains last — domains.ts's sort.
  const nowMs = Date.now();
  const sorted = sortDomains(payload.domains);
  const expiring = sorted.filter((x) => {
    const d = x.expiresAt !== null ? daysUntil(x.expiresAt, nowMs) : null;
    return d !== null && d <= 30;
  }).length;
  const subtitle = payload.errors.length > 0
    ? `Partial load — ${payload.errors[0]}`
    : expiring > 0
      ? `${sorted.length} domains · ${expiring} expiring within 30 days`
      : `${sorted.length} domains`;

  return (
    <motion.div
      initial={reduced ? false : { opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={reduced ? { duration: 0 } : { duration: 0.2, ease: "easeOut" }}
    >
      {refreshError !== null ? (
        <p className="cf-empty cf-level-warning-text" role="alert">{`Refresh failed — ${refreshError}`}</p>
      ) : null}

      <p className="cf-row-detail">{subtitle}</p>
      <section className="cf-card">
        {sorted.length === 0 ? (
          <p className="cf-empty cf-empty-quiet">No domains found on this account.</p>
        ) : (
          sorted.map((d) => <DomainRow key={d.id} d={d} nowMs={nowMs} onOpen={() => setActiveId(d.id)} />)
        )}
      </section>
    </motion.div>
  );
}
