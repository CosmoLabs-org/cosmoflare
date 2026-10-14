// Overview as a real React view (P-04, docs/planning-mode/
// 2026-10-10-pager-react-rebuild.md): replaces the imperative bridge to
// dashboard.ts's renderOverview. The DOM structure and CSS classes are the
// vanilla view's (cf-plan-line, cf-kpis, cf-ring*, cf-attention*) and the
// pure helpers are imported verbatim — gauges.ts, attention.ts and the
// dashboard pure functions — never re-implemented.

import { useCallback, useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { api, LoginExpiredError, totalAgeSec, type Billing, type FetchResult, type ProductUsage, type Summary } from "../../api";
import type { EmailRoutingPayload } from "../../email";
import type { RulesQuotaSummary } from "../../api";
import { ATTENTION_CAP, capAttention, collectAttention, type AttentionItem } from "../../attention";
import { isRollingStorage, projectedMonthlyTotal, WORKERS_PAID_BASE_USD, WORKERS_PAID_PLAN } from "../../billing";
import { formatCount, formatPct, formatUsd } from "../../format";
import { capTopRings, gaugeLevel, productLabel, sortByProjectedDesc, type Level } from "../../gauges";
import { hrefFor } from "../../routes";
import { updateStatusStrip } from "../../statusstrip";
import {
  pacingPct,
  pacingSublabel,
  ringOverageText,
  ringPropsFor,
  usageLevelFromPct,
} from "../../dashboard";
import RingGauge from "../components/RingGauge";

/** findProduct locates a billing product by id, for the pacing KPI tiles. */
const findProduct = (b: Billing | null, id: string): ProductUsage | null => b?.products.find((p) => p.id === id) ?? null;

/** One painted (summary, billing) pair — the React state equivalent of
 *  paintOverview's root argument. */
interface OverviewData {
  sumRes: FetchResult<Summary>;
  billRes: FetchResult<Billing>;
}

/** One tappable attention disclosure (UI-2): mark + title on the first
 *  line, the FULL detail text and the secondary "Open section →" link
 *  behind the tap. Independent toggles — no accordion state. */
function AttentionRow({ item }: { item: AttentionItem }): React.JSX.Element {
  const [expanded, setExpanded] = useState(false);
  return (
    <div className={`cf-attention cf-level-${item.level}`}>
      <button
        type="button"
        className="cf-attention-toggle"
        aria-expanded={expanded}
        onClick={() => setExpanded((e) => !e)}
      >
        <span className="cf-attention-mark">{item.level === "critical" ? "!" : "•"}</span>
        <span className="cf-attention-title">{item.title}</span>
        <span className="cf-attention-chevron">→</span>
      </button>
      <div className="cf-attention-detail" hidden={!expanded}>
        <p className="cf-attention-text">{item.detail}</p>
        <a className="cf-attention-open" href={hrefFor(item.route)}>Open section →</a>
      </div>
    </div>
  );
}

/** One ring tile: the gauge wrapped in the billing link, with the projected
 *  overage dollars under it when there is one (item-6 overview half). */
function RingLink({ p, period }: { p: ProductUsage; period: Billing["period"] }): React.JSX.Element {
  const props = ringPropsFor(p, period);
  const { level, overLimit } = gaugeLevel(props.usedPct, props.projectedPct);
  const overage = ringOverageText(p);
  return (
    <a className={`cf-ring cf-level-${level}${overLimit ? " cf-over" : ""}`} href={hrefFor("billing")}>
      <RingGauge {...props} />
      {overage !== null ? <span className="cf-ring-overage cf-level-critical-text">{overage}</span> : null}
    </a>
  );
}

/** One KPI tile (value + label + optional sublabel) with the level class. */
function KpiTile({ value, label, level = "ok", sublabel, children }: {
  value: string;
  label: string;
  level?: Level;
  sublabel?: string;
  children?: React.ReactNode;
}): React.JSX.Element {
  return (
    <div className={`cf-kpi cf-level-${level}`} role="group">
      <div className="cf-kpi-value">{value}</div>
      <div className="cf-kpi-label">{label}</div>
      {sublabel ? <div className="cf-kpi-sub">{sublabel}</div> : null}
      {children}
    </div>
  );
}

export default function OverviewView({ refreshSeq = 0 }: { refreshSeq?: number }): React.JSX.Element {
  const [data, setData] = useState<OverviewData | null>(null);
  const [loadError, setLoadError] = useState<{ message: string; expired: boolean } | null>(null);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [showAllRings, setShowAllRings] = useState(false);
  const [retryTick, setRetryTick] = useState(0);
  const [email, setEmail] = useState<EmailRoutingPayload | null>(null);
  const [quotaSummary, setQuotaSummary] = useState<RulesQuotaSummary | null>(null);
  const reduced = useReducedMotion();

  // A background revalidation repaints the view with the fresh data
  // (BUG-057 defect 2) — same shape as the vanilla repaint: fetch both
  // endpoints again (both then fresh in the client cache, so instant) and
  // keep the old state when it fails.
  const repaint = useCallback((): void => {
    void Promise.all([
      api.fetchJson<Summary>("api/summary"),
      api.fetchJson<Billing>("api/billing"),
    ]).then(
      ([s, b]) => {
        setData({ sumRes: s, billRes: b });
        setRefreshError(null);
      },
      () => undefined,
    );
  }, []);

  useEffect(() => {
    // refreshSeq > 0 means the shell's global Refresh re-invoked the view.
    const refresh = refreshSeq > 0;
    let cancelled = false;
    void (async () => {
      try {
        const [sumRes, billRes] = await Promise.all([
          api.fetchJson<Summary>("api/summary", { refresh, onRevalidate: repaint }),
          api.fetchJson<Billing>("api/billing", { refresh, onRevalidate: repaint }),
        ]);
        if (cancelled) return;
        setData({ sumRes, billRes });
        setRefreshError(null);
        setLoadError(null);
      } catch (err) {
        if (cancelled) return;
        // A failed refresh must NOT wipe good state (BUG-057 defect 2): the
        // last good data stays on screen and the error text surfaces in the
        // view. Only a first (skeleton) load shows the error/retry state.
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
  }, [refreshSeq, retryTick, repaint]);

  // Email Routing (FEAT-059): a side fetch that must NOT gate the first
  // paint or the error states — the collector is operator-gated (Email
  // Routing Read), so an errors-only payload or a failed fetch simply
  // leaves the card off the overview.
  useEffect(() => {
    let cancelled = false;
    void api.fetchJson<EmailRoutingPayload>("api/email", { refresh: refreshSeq > 0 }).then(
      (res) => {
        if (!cancelled) setEmail(res.data);
      },
      () => undefined,
    );
    return () => {
      cancelled = true;
    };
  }, [refreshSeq, retryTick]);

  // Rule-quota strip (ROAD-p9X66J9): one cached read from the cron's
  // rotating batch; a failed fetch or empty store leaves the card off.
  useEffect(() => {
    let cancelled = false;
    void api.fetchJson<RulesQuotaSummary>("api/rules-quota/summary", { refresh: refreshSeq > 0 }).then(
      (res) => {
        if (!cancelled) setQuotaSummary(res.data);
      },
      () => undefined,
    );
    return () => {
      cancelled = true;
    };
  }, [refreshSeq, retryTick]);

  // Bottom status strip (UI-1): the worst age across both endpoints (the
  // honest client copy age + server cache age) plus the billing-period day.
  useEffect(() => {
    if (!data) return;
    const { sumRes, billRes } = data;
    updateStatusStrip({
      ageSec: Math.max(
        totalAgeSec(sumRes.ageSec, sumRes.data.cache),
        totalAgeSec(billRes.ageSec, billRes.data.cache),
      ),
      day: billRes.data.period.day,
      days: billRes.data.period.days,
    });
  }, [data]);

  // First-load error states mirror the vanilla view.
  if (loadError !== null) {
    if (loadError.expired) {
      return <p className="cf-empty cf-login-expired">{loadError.message}</p>;
    }
    return (
      <section className="cf-card cf-level-warning">
        <h2>Could not load the overview</h2>
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

  // Skeleton placeholder while the first fetch is in flight.
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

  const summary = data.sumRes.data;
  const billing = data.billRes.data;
  const attention = collectAttention(summary, billing);
  const { shown: attentionShown, extra: attentionExtra } = capAttention(attention, ATTENTION_CAP);

  // KPI tiles — the period day lives in the status strip (critique P2) and
  // the overage dollars live in the plan line, so the tiles carry what the
  // rings don't: worker errors, pacing, attention count.
  const d1Prod = findProduct(billing, "d1.rows_read");
  const wProd = findProduct(billing, "workers.requests");
  const over = billing.totalProjectedOverageUsd;
  const workerErrors = summary.workers.reduce((n, w) => n + w.errors, 0);

  // Workers Paid allowances — the 8 rings closest to their limit, worst-first
  // (projected % desc), the rest behind a "Show all N" toggle.
  const sorted = sortByProjectedDesc(billing.products.filter((p) => !isRollingStorage(p.id)));
  const { shown, hiddenCount } = capTopRings(sorted);
  const rings = showAllRings ? sorted : shown;

  return (
    <motion.div
      initial={reduced ? false : { opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={reduced ? { duration: 0 } : { duration: 0.2, ease: "easeOut" }}
    >
      {refreshError !== null ? (
        <p className="cf-empty cf-level-warning-text" role="alert">{`Refresh failed — ${refreshError}`}</p>
      ) : null}

      {/* Plan line (operator 2026-10-13): the plan name, then the month on
          one line — the same cost lockup the billing card runs, at line
          scale. Gray base reads as the standard price; the total closes.
          Zero-overage collapses to the base alone (critique P2). */}
      <p className="cf-plan-line">
        <span className="cf-plan-day">{`day ${billing.period.day}/${billing.period.days}`}</span>
        <span className="cf-plan-name">{`${WORKERS_PAID_PLAN} ·`}</span>
        <span
          className="cf-cost-line"
          aria-label={
            over >= 0.01
              ? `Base ${formatUsd(WORKERS_PAID_BASE_USD)} plus projected overage ${formatUsd(over)}, ${formatUsd(projectedMonthlyTotal(billing.products))} this month`
              : `Base ${formatUsd(WORKERS_PAID_BASE_USD)}, no overage projected, this month`
          }
        >
          <span className="cf-cost-line-base">{formatUsd(WORKERS_PAID_BASE_USD)}</span>
          {over >= 0.01 ? (
            <>
              <span className="cf-cost-line-op" aria-hidden="true">+</span>
              <span className={`cf-cost-line-over${over >= 50 ? " cf-level-critical-text" : ""}`}>{formatUsd(over)}</span>
              <span className="cf-cost-line-total">{formatUsd(projectedMonthlyTotal(billing.products))}</span>
            </>
          ) : null}
          <span className="cf-cost-line-month">this month</span>
        </span>
      </p>

      <p className="cf-row-detail cf-view-intro">Live account health for the last 24h. Tap Workers, D1 or a zone for its profile.</p>

      <div className="cf-kpis">
        <KpiTile
          value={formatCount(workerErrors)}
          label={workerErrors === 1 ? "worker error · 24h" : "worker errors · 24h"}
          level={workerErrors > 0 ? "warning" : "ok"}
        />
        <KpiTile
          value={d1Prod ? formatPct(pacingPct(d1Prod), 0) : "—"}
          label={d1Prod ? productLabel(d1Prod.id, d1Prod.product, d1Prod.metric) : "D1 rows-read pacing"}
          level={d1Prod ? usageLevelFromPct(pacingPct(d1Prod)) : "ok"}
          sublabel={d1Prod ? pacingSublabel(d1Prod) : undefined}
        />
        <KpiTile
          value={wProd ? formatPct(pacingPct(wProd), 0) : "—"}
          label={wProd ? productLabel(wProd.id, wProd.product, wProd.metric) : "Workers requests pacing"}
          level={wProd ? usageLevelFromPct(pacingPct(wProd)) : "ok"}
          sublabel={wProd ? pacingSublabel(wProd) : undefined}
        />
        <KpiTile
          value={String(attention.length)}
          label={attention.length === 1 ? "item needs attention" : "items need attention"}
          level={attention.some((i) => i.level === "critical") ? "critical" : attention.length ? "warning" : "ok"}
        />
      </div>

      {/* Rule quotas (ROAD-p9X66J9): fleet strip from the cron's rotating
          batch — rendered only when the summary has tracked zones; an
          absent store (fresh deploy) keeps the overview unchanged. */}
      {quotaSummary !== null && quotaSummary.zonesTracked > 0 ? (
        <section className="cf-card" aria-label="Rule quotas">
          <h2>Rule slots</h2>
          <p className="cf-row-detail">{`Across ${quotaSummary.zonesTracked} tracked zones, refreshed in rotation.`}</p>
          {quotaSummary.families
            .filter((f) => f.zonesTracked > 0)
            .map((f) => (
              <p className={f.atCapZones > 0 ? "cf-empty cf-level-warning-text" : "cf-row-detail"} key={f.family}>
                {`${f.label}: ${f.totalUsed} in use${f.atCapZones > 0 ? ` · ${f.atCapZones} zone${f.atCapZones === 1 ? "" : "s"} at the free cap` : ""}`}
              </p>
            ))}
        </section>
      ) : null}

      {/* Email Routing (FEAT-059): rendered ONLY when the collector returned
          zones — an errors-only payload (token without Email Routing Read)
          keeps the overview unchanged, the same gate the AI view applies
          before painting gateway cards. */}
      {email !== null && email.zones.length > 0 ? (
        <section className="cf-card" aria-label="Email Routing">
          <h2>Email Routing</h2>
          <div className="cf-kpi cf-level-ok" role="group">
            <div className="cf-kpi-value">{`${email.zones.filter((z) => z.enabled).length}/${email.zones.length}`}</div>
            <div className="cf-kpi-label">zones with routing enabled</div>
          </div>
          <ul className="cf-consumers-list">
            {email.zones.map((z) => (
              <li className="cf-consumer" key={z.zone}>
                <span className="cf-consumer-who">
                  <span className="cf-consumer-name">{z.name ?? z.zone}</span>
                  {z.name && z.name !== z.zone ? <span className="cf-consumer-project">{z.zone}</span> : null}
                </span>
                <span className="cf-consumer-sharewrap">
                  <span className="cf-chip">{z.enabled ? "enabled" : "disabled"}</span>
                </span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      <section className="cf-card">
        <h2>Workers Paid allowances</h2>
        <div className="cf-rings">
          {rings.map((p) => (
            <RingLink key={p.id} p={p} period={billing.period} />
          ))}
        </div>
        {hiddenCount > 0 ? (
          <button type="button" className="cf-btn cf-show-all" onClick={() => setShowAllRings((s) => !s)}>
            {showAllRings ? "Show top 8" : `Show all ${sorted.length}`}
          </button>
        ) : null}
      </section>

      <section className="cf-card">
        <h2>Needs attention</h2>
        {attention.length === 0 ? (
          <p className="cf-empty cf-empty-quiet">All clear — nothing above threshold.</p>
        ) : (
          <>
            {attentionShown.map((item) => (
              <AttentionRow key={`${item.route}:${item.title}`} item={item} />
            ))}
            {attentionExtra > 0 ? (
              <a className="cf-attention cf-attention-more" href={hrefFor("billing")}>
                <span className="cf-attention-body">{`Show all ${attention.length}`}</span>
              </a>
            ) : null}
          </>
        )}
      </section>
    </motion.div>
  );
}
