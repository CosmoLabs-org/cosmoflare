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
  type AttachmentsPayload,
  type AuditEntry,
  type AuditPayload,
  type DomainDetailPayload,
  type DomainRecord,
  type DomainsPayload,
  type ZoneAttachment,
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

const defaultFetchJson: ViewFetcher = (endpoint, opts) => api.fetchJson(endpoint, opts);

// ---- Lifecycle mutations (FEAT-pEWA6N6 phases 2-3) ----

/** Injectable mutation hook (the mutator mirrors the ViewFetcher injection
 *  pattern so tests can stub the POSTs). */
export type ViewMutator = (endpoint: string, body: unknown) => Promise<unknown>;

// postLifecycle POSTs one lifecycle mutation. Contract (ops lifecycle.ts):
// 200 echoes {ok:true}; ANY failure carries {error} and it surfaces VERBATIM
// as a plain Error — no silent failures. An expired Access session answers
// opaqueredirect, or a 403 with no JSON error body, and surfaces as
// LoginExpiredError (a 403 WITH {error} is an API refusal, e.g. the
// protected-host gate, and stays a verbatim message).
async function postLifecycle(endpoint: string, body: unknown): Promise<void> {
  let res: Response;
  try {
    res = await fetch(`/${endpoint}`, {
      method: "POST",
      credentials: "same-origin",
      redirect: "manual",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch (err) {
    throw new Error(`Could not reach the Ops Worker: ${err instanceof Error ? err.message : String(err)}`);
  }
  if (res.type === "opaqueredirect") throw new LoginExpiredError();
  const data = (await res.json().catch(() => undefined)) as { error?: string } | undefined;
  if (res.status === 403 && !data?.error) throw new LoginExpiredError();
  if (!res.ok) throw new Error(data?.error ?? `HTTP ${res.status}`);
}

// Per-attachment-row UI state: attached → (Take offline) → confirm →
// detached → (Put back) → attached, plus the in-flight flag and the last
// verbatim error under the row.
type RowMode = "attached" | "confirm" | "detached";
interface RowUi {
  mode: RowMode;
  busy?: boolean;
  error?: string;
}

/** One attached domain row with the Take offline / Confirm detach / Put
 *  back lifecycle. Workers rows surface the Cloudflare note that removing
 *  the custom domain does NOT delete its advanced certificate. */
function AttachmentRow({ rowKey, hostname, badges, detached, ui, onDetach, onCancel, onAttach }: {
  rowKey: string;
  hostname: string;
  badges: React.ReactNode;
  detached?: React.ReactNode; // extra detached-state line (workers cert note)
  ui: RowUi;
  onDetach: () => void;
  onCancel: () => void;
  onAttach: () => void;
}): React.JSX.Element {
  return (
    <div className="cf-attachment-row" data-row={rowKey}>
      <span className="cf-attachment-hostname">{hostname}</span>
      <span className="cf-attachment-badges">
        {badges}
        {ui.mode === "detached" ? <span className="cf-badge is-muted">detached</span> : null}
      </span>
      {ui.mode === "attached" ? (
        <span className="cf-attachment-actions">
          <button type="button" className="cf-btn cf-btn-ghost" disabled={ui.busy === true} onClick={onDetach}>Take offline</button>
        </span>
      ) : ui.mode === "confirm" ? (
        <span className="cf-attachment-actions">
          <button type="button" className="cf-btn" disabled={ui.busy === true} onClick={onDetach}>Confirm detach</button>
          <button type="button" className="cf-btn cf-btn-ghost" disabled={ui.busy === true} onClick={onCancel}>Cancel</button>
        </span>
      ) : (
        <span className="cf-attachment-actions">
          <button type="button" className="cf-btn cf-btn-ghost" disabled={ui.busy === true} onClick={onAttach}>Put back</button>
        </span>
      )}
      {ui.mode === "detached" && detached}
      {ui.error ? <p className="cf-attachment-error cf-level-warning-text">{ui.error}</p> : null}
    </div>
  );
}

/** One zone row with the typed-confirmation Remove panel: the input must
 *  equal the zone name EXACTLY, a "this cannot be undone" warning, and
 *  protected zones show a lock line naming the protection with the submit
 *  disabled (removal requires an operator override server-side). */
function ZoneRemoveRow({ zone, mutate }: { zone: ZoneAttachment; mutate: ViewMutator }): React.JSX.Element {
  const [open, setOpen] = useState(false);
  const [typed, setTyped] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [removed, setRemoved] = useState(false);
  const isProtected = zone.protected === true;
  const canSubmit = !isProtected && typed === zone.name && !busy;

  const submit = (): void => {
    setBusy(true);
    setError(null);
    void mutate("api/zones/remove", { zoneId: zone.id, typedName: typed, override: false }).then(
      () => {
        setRemoved(true);
        setBusy(false);
        setOpen(false);
      },
      (err: unknown) => {
        setError(err instanceof Error ? err.message : String(err));
        setBusy(false);
      },
    );
  };

  return (
    <div className="cf-attachment-row cf-zone-row" data-row={`zone:${zone.id}`}>
      <span className="cf-attachment-hostname">{zone.name}</span>
      <span className="cf-attachment-badges">
        {removed ? <span className="cf-badge is-muted">removed</span> : null}
        {isProtected && !removed ? <span className="cf-badge is-muted">protected</span> : null}
        {zone.status ? <span className="cf-badge is-muted">{zone.status}</span> : null}
        {zone.recordCount !== null ? <span className="cf-badge is-muted">{`${zone.recordCount} records`}</span> : null}
      </span>
      {!removed ? (
        open ? (
          <div className="cf-zone-remove-panel">
            <p className="cf-zone-warning cf-level-warning-text">This cannot be undone — the zone and its DNS records are deleted.</p>
            {isProtected ? (
              <p className="cf-zone-lock" role="note">{`🔒 ${zone.name} is a protected host — removal requires an operator override`}</p>
            ) : (
              <input
                type="text"
                className="cf-input"
                value={typed}
                placeholder={zone.name}
                aria-label={`Type ${zone.name} to confirm removal`}
                onChange={(e) => setTyped(e.target.value)}
              />
            )}
            <span className="cf-attachment-actions">
              <button type="button" className="cf-btn" disabled={!canSubmit} onClick={submit}>Confirm removal</button>
              <button
                type="button"
                className="cf-btn cf-btn-ghost"
                disabled={busy}
                onClick={() => {
                  setOpen(false);
                  setTyped("");
                  setError(null);
                }}
              >
                Cancel
              </button>
            </span>
          </div>
        ) : (
          <span className="cf-attachment-actions">
            <button type="button" className="cf-btn cf-btn-ghost" onClick={() => setOpen(true)}>Remove</button>
          </span>
        )
      ) : null}
      {error ? <p className="cf-attachment-error cf-level-warning-text">{error}</p> : null}
    </div>
  );
}

/** The attachment inventory (FEAT-pEWA6N6 phases 1+2): one group per Pages
 *  project (project name row, one row per domain with muted status + the
 *  SSL state when present), then the Workers group, then the zone rows with
 *  their gated Remove panels. The detach/attach flow is row-local: Take
 *  offline → one inline confirm → the row flips to detached with Put back;
 *  API errors render verbatim under the row. Hidden entirely while the
 *  attachments fetch has not landed — its failure changes nothing in the
 *  domains error paths. */
function AttachmentsSection({ payload, mutate }: { payload: AttachmentsPayload | null; mutate: ViewMutator }): React.JSX.Element | null {
  const [ui, setUi] = useState<Record<string, RowUi>>({});
  if (payload === null) return null;
  // Defensive array guards: a payload of the wrong shape (never expected
  // from the worker) must blank the section, not crash the view.
  const pages = Array.isArray(payload.pages) ? payload.pages : [];
  const workers = Array.isArray(payload.workers) ? payload.workers : [];
  const zones = Array.isArray(payload.zones) ? payload.zones : [];
  const errors = Array.isArray(payload.errors) ? payload.errors : [];
  const rowOf = (key: string): RowUi => ui[key] ?? { mode: "attached" };
  const patch = (key: string, p: Partial<RowUi>): void =>
    setUi((prev) => ({ ...prev, [key]: { ...(prev[key] ?? { mode: "attached" as RowMode }), ...p } }));
  const run = (key: string, endpoint: string, body: Record<string, unknown>, done: RowMode): void => {
    patch(key, { busy: true, error: undefined });
    void mutate(endpoint, body).then(
      () => patch(key, { busy: false, mode: done }),
      (err: unknown) => patch(key, { busy: false, error: err instanceof Error ? err.message : String(err) }),
    );
  };

  return (
    <section className="cf-card cf-attachments">
      <p className="cf-row-detail">
        {errors.length > 0 ? `Custom domains by project — partial load — ${errors[0]}` : "Custom domains by project"}
      </p>
      {pages.map((p) => (
        <div className="cf-attachments-group" key={p.project}>
          <div className="cf-product-head">
            <strong className="cf-product-name">{p.project}</strong>
          </div>
          {p.domains.map((d) => {
            const key = `pages:${p.project}/${d.hostname}`;
            const uiRow = rowOf(key);
            return (
              <AttachmentRow
                key={d.hostname}
                rowKey={key}
                hostname={d.hostname}
                ui={uiRow}
                badges={
                  <>
                    {d.status ? <span className="cf-badge is-muted">{d.status}</span> : null}
                    {d.ssl ? <span className="cf-badge is-muted">{`ssl: ${d.ssl}`}</span> : null}
                  </>
                }
                onDetach={() => (uiRow.mode === "confirm" ? run(key, "api/domains/detach", { kind: "pages", project: p.project, name: d.hostname }, "detached") : patch(key, { mode: "confirm" }))}
                onCancel={() => patch(key, { mode: "attached" })}
                onAttach={() => run(key, "api/domains/attach", { kind: "pages", project: p.project, name: d.hostname }, "attached")}
              />
            );
          })}
        </div>
      ))}
      {workers.length > 0 ? (
        <div className="cf-attachments-group">
          <div className="cf-product-head">
            <strong className="cf-product-name">Workers</strong>
          </div>
          {workers.map((w) => {
            const key = `workers:${w.id ?? w.hostname}`;
            const uiRow = rowOf(key);
            return (
              <AttachmentRow
                key={key}
                rowKey={key}
                hostname={w.hostname}
                ui={uiRow}
                badges={<span className="cf-badge is-muted">{`${w.service} / ${w.environment}`}</span>}
                detached={
                  // Cloudflare docs (verified 2026-10-12): deleting a Workers
                  // custom domain does not auto-delete its advanced certificate.
                  <p className="cf-attachment-note">Note: removing a Workers custom domain does not delete its advanced certificate.</p>
                }
                onDetach={() => (uiRow.mode === "confirm" ? run(key, "api/domains/detach", { kind: "workers", name: w.hostname, domainId: w.id }, "detached") : patch(key, { mode: "confirm" }))}
                onCancel={() => patch(key, { mode: "attached" })}
                onAttach={() => run(key, "api/domains/attach", { kind: "workers", name: w.hostname, service: w.service }, "attached")}
              />
            );
          })}
        </div>
      ) : null}
      {zones.length > 0 ? (
        <div className="cf-attachments-group">
          <div className="cf-product-head">
            <strong className="cf-product-name">Zones</strong>
          </div>
          {zones.map((z) => (
            <ZoneRemoveRow key={z.id} zone={z} mutate={mutate} />
          ))}
        </div>
      ) : null}
    </section>
  );
}

/** The lifecycle audit trail (FEAT-pEWA6N6 phase 3): the newest-first audit
 *  rows from /api/audit — ts, action, target, operator, ok/error — with the
 *  same skeleton + verbatim-error patterns as the rest of the view. */
function AuditCard({ fetchJson }: { fetchJson: ViewFetcher }): React.JSX.Element {
  const [state, setState] = useState<{ status: "loading" } | { status: "ok"; entries: AuditEntry[] } | { status: "error"; message: string }>({ status: "loading" });
  useEffect(() => {
    let cancelled = false;
    void fetchJson<AuditPayload>("api/audit").then(
      (res) => {
        if (!cancelled) setState({ status: "ok", entries: Array.isArray(res.data?.audit) ? res.data.audit : [] });
      },
      (err: unknown) => {
        if (!cancelled) setState({ status: "error", message: err instanceof Error ? err.message : String(err) });
      },
    );
    return () => {
      cancelled = true;
    };
  }, [fetchJson]);
  return (
    <section className="cf-card cf-audit">
      <h3 className="cf-audit-title">Audit</h3>
      {state.status === "loading" ? (
        <div className="cf-loading">
          <div className="cf-skeleton" />
          <div className="cf-skeleton" />
        </div>
      ) : state.status === "error" ? (
        <p className="cf-empty cf-level-warning-text">{`Could not load the audit log — ${state.message}`}</p>
      ) : state.entries.length === 0 ? (
        <p className="cf-empty cf-empty-quiet">No lifecycle actions recorded yet.</p>
      ) : (
        state.entries.map((e, i) => (
          <div className="cf-audit-row" key={`${e.ts}-${i}`}>
            <span className="cf-audit-ts">{new Date(e.ts).toLocaleString()}</span>
            <span className="cf-audit-action">{e.action}</span>
            <span className="cf-audit-target">{e.target}</span>
            <span className="cf-audit-operator">{e.operator || "—"}</span>
            <span className={e.ok ? "cf-audit-ok" : "cf-audit-fail cf-level-warning-text"}>
              {e.ok ? "ok" : `error${e.error ? ` — ${e.error}` : ""}`}
            </span>
          </div>
        ))
      )}
    </section>
  );
}

export default function DomainsView({ refreshSeq = 0, fetcher, mutator }: { refreshSeq?: number; fetcher?: ViewFetcher; mutator?: ViewMutator }): React.JSX.Element {
  // The Retry re-render must go through the SAME fetcher (injected or the
  // shared client) — hence the closure rather than a bare method reference.
  const fetchJson: ViewFetcher = fetcher ?? defaultFetchJson;
  // Lifecycle mutations (detach/attach/zone-remove) go through the same
  // injection pattern: the injected mutator in tests, the shared POST
  // helper in production.
  const mutate: ViewMutator = mutator ?? postLifecycle;
  const [data, setData] = useState<FetchResult<DomainsPayload> | null>(null);
  // Attachment inventory (FEAT-pEWA6N6 phase 1): fetched independently of
  // api/domains — a failure here leaves the zones view and its error paths
  // untouched (the section just stays hidden).
  const [attachments, setAttachments] = useState<FetchResult<AttachmentsPayload> | null>(null);
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
    // Attachments ride along independently (soft): any failure is dropped.
    void fetchJson<AttachmentsPayload>("api/attachments").then(
      (res) => {
        if (!cancelled) setAttachments(res);
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
      ? `${expiring} expiring within 30 days`
      : null;

  return (
    <motion.div
      initial={reduced ? false : { opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={reduced ? { duration: 0 } : { duration: 0.2, ease: "easeOut" }}
    >
      {refreshError !== null ? (
        <p className="cf-empty cf-level-warning-text" role="alert">{`Refresh failed — ${refreshError}`}</p>
      ) : null}

      {subtitle !== null && <p className="cf-row-detail">{subtitle}</p>}
      <section className="cf-card">
        {sorted.length === 0 ? (
          <p className="cf-empty cf-empty-quiet">No domains found on this account.</p>
        ) : (
          sorted.map((d) => <DomainRow key={d.id} d={d} nowMs={nowMs} onOpen={() => setActiveId(d.id)} />)
        )}
      </section>

      <AttachmentsSection payload={attachments?.data ?? null} mutate={mutate} />
      <AuditCard fetchJson={fetchJson} />
    </motion.div>
  );
}
