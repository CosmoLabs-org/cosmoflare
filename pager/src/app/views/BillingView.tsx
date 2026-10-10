// Billing as a real React view (P-04, docs/planning-mode/
// 2026-10-10-pager-react-rebuild.md): replaces the imperative bridge to
// billing.ts's renderBilling. Period card + legend, product cards with the
// grade/overage pair head and bar gauges, ranked top consumers, the
// kv.storage rolling-bytes exclusion note, projects table, pricing footnote
// and telemetry gaps — the same DOM and CSS classes as the vanilla view,
// with the pure helpers imported verbatim.

import { Fragment, useCallback, useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { api, LoginExpiredError, totalAgeSec, type Billing, type BillingPeriod, type FetchResult, type ProductUsage } from "../../api";
import { formatAmount, formatPct, formatUsd, formatDateShort, periodProgress } from "../../format";
import { productLabel, scaleFitPct } from "../../gauges";
import { isRollingStorage } from "../../billing";
import { updateStatusStrip } from "../../statusstrip";
import BarGauge from "../components/BarGauge";

/**
 * The overage half of the card-head pair (operator 2026-10-10): the
 * projected cost beyond the allowance when there is one, "no overage" when
 * there is not — imported from the vanilla module's pure helper.
 */
import { gradeLabel, overageHeadText } from "../../billing";

/** One product's bar-gauge card: name + the grade/overage status pair in
 *  the head, the gauge, then the expandable ranked top consumers. */
function ProductCard({ p, period }: { p: ProductUsage; period: BillingPeriod }): React.JSX.Element {
  const usedPct = p.included > 0 ? (p.used / p.included) * 100 : 0;
  const projectedPct = p.included > 0 ? (p.projected / p.included) * 100 : 0;
  const grade = gradeLabel(usedPct, projectedPct);
  // Bar percentages are of the fitted track scale (max of used/projected/
  // included) so an over-allowance projection stays on the track; the level
  // and badge run on the % of allowance. The today marker sits at the
  // expected-to-date use (allowance × period elapsed).
  const scale = Math.max(p.used, p.projected, p.included, 1e-9);
  return (
    <section className={`cf-card cf-product ${p.projectedOverageUsd > 0 ? "cf-over" : ""}`}>
      <div className="cf-product-head">
        <strong className="cf-product-name">{productLabel(p.id, p.product, p.metric)}</strong>
        {/* Grade + overage, always both, same size (operator 2026-10-10). */}
        <span className="cf-product-status">
          <span className={`cf-product-grade cf-level-${grade.level}-text`}>{grade.text}</span>
          <span className={`cf-product-overage ${p.projectedOverageUsd > 0 ? "cf-level-critical-text" : "cf-ok-text"}`}>
            {overageHeadText(p)}
          </span>
        </span>
      </div>
      <BarGauge
        usedPct={usedPct}
        projectedPct={projectedPct}
        usedScalePct={scaleFitPct(p.used, scale)}
        projectedScalePct={scaleFitPct(p.projected, scale)}
        includedScalePct={scaleFitPct(p.included, scale)}
        expectedPct={scaleFitPct(p.included * (period.day / period.days), scale)}
        label=""
        detailText={`${formatAmount(p.used, p.unit)} of ${formatAmount(p.included, p.unit)} used · projected ${formatAmount(p.projected, p.unit)} (${formatPct(projectedPct, 0)} of allowance)`}
      />
      <details className="cf-consumers">
        <summary className="cf-consumers-summary">{`Top consumers (${p.topConsumers.length})`}</summary>
        <ol className="cf-consumers-list cf-consumers-ranked">
          {p.topConsumers.map((c, i) => (
            <li className="cf-consumer" key={c.name}>
              <span className="cf-consumer-rank">{String(i + 1)}</span>
              <span className="cf-consumer-who">
                <span className="cf-consumer-name">{c.name}</span>
                <span className="cf-consumer-project">{c.project}</span>
              </span>
              <span className="cf-consumer-sharewrap">
                <span className="cf-consumer-sharebar">
                  <span
                    className="cf-consumer-sharefill"
                    style={{ width: `${Math.min(100, Math.max(0, c.share * 100))}%` }}
                  />
                </span>
                <span className="cf-consumer-share">{formatPct(c.share * 100, 0)}</span>
              </span>
            </li>
          ))}
        </ol>
      </details>
    </section>
  );
}

export default function BillingView({ refreshSeq = 0 }: { refreshSeq?: number }): React.JSX.Element {
  const [data, setData] = useState<FetchResult<Billing> | null>(null);
  const [loadError, setLoadError] = useState<{ message: string; expired: boolean } | null>(null);
  const [refreshError, setRefreshError] = useState<string | null>(null);
  const [retryTick, setRetryTick] = useState(0);
  const reduced = useReducedMotion();

  // A background revalidation repaints the view with the fresh copy
  // (BUG-057 defect 2) instead of leaving hours-old data on screen.
  const repaint = useCallback((fresh: Billing): void => {
    setData({ data: fresh, source: "network", ageSec: 0, demo: false });
    setRefreshError(null);
  }, []);

  useEffect(() => {
    const refresh = refreshSeq > 0;
    let cancelled = false;
    void (async () => {
      try {
        const res = await api.fetchJson<Billing>("api/billing", { refresh, onRevalidate: repaint });
        if (cancelled) return;
        setData(res);
        setRefreshError(null);
        setLoadError(null);
      } catch (err) {
        if (cancelled) return;
        // A failed refresh keeps the last good data (BUG-057 defects 2/3)
        // and surfaces the error text in the view; only a first load shows
        // the error/retry state.
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

  // Bottom status strip (UI-1): honest age (client copy age + server cache
  // age) plus the billing-period day — painted on every render.
  useEffect(() => {
    if (!data) return;
    updateStatusStrip({
      ageSec: totalAgeSec(data.ageSec, data.data.cache),
      day: data.data.period.day,
      days: data.data.period.days,
    });
  }, [data]);

  if (loadError !== null) {
    if (loadError.expired) {
      return <p className="cf-empty cf-login-expired">{loadError.message}</p>;
    }
    return (
      <section className="cf-card cf-level-warning">
        <h2>Could not load billing</h2>
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

  const b = data.data;
  const periodSource = b.period.source === "calendar" ? "Calendar month" : b.period.source === "anchor" ? "Anchor day" : "Subscription";

  return (
    <motion.div
      initial={reduced ? false : { opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={reduced ? { duration: 0 } : { duration: 0.2, ease: "easeOut" }}
    >
      {refreshError !== null ? (
        <p className="cf-empty cf-level-warning-text" role="alert">{`Refresh failed — ${refreshError}`}</p>
      ) : null}

      {/* Period progress */}
      <section className="cf-card">
        <h2>{`Billing period — day ${b.period.day} of ${b.period.days}`}</h2>
        <div className="cf-progress">
          <div
            className="cf-progress-fill"
            style={{ width: `${periodProgress(b.period.day, b.period.days).elapsedPct}%` }}
          />
        </div>
        <p className="cf-row-detail">
          {`${periodSource} · ends ${formatDateShort(b.period.end)} · ${b.period.days - b.period.day} full days left`}
        </p>
      </section>

      {/* Legend: solid = used, hatched = projected, line = allowance, dot =
          today — the encodings the bar gauges repeat. */}
      <div className="cf-legend" role="list" aria-label="Gauge legend: solid used, hatched projected, line allowance, dot today">
        <span className="cf-legend-item">
          <span className="cf-legend-swatch cf-legend-used" />used
        </span>
        <span className="cf-legend-item">
          <span className="cf-legend-swatch cf-legend-projected" />projected
        </span>
        <span className="cf-legend-item">
          <span className="cf-legend-allowance" />allowance
        </span>
        <span className="cf-legend-item">
          <span className="cf-legend-dot" />today
        </span>
      </div>

      {/* Products (sorted by projected overage desc — the server sends them
          so). Rolling bytes (kv.storage) get no allowance card. */}
      <div className="cf-products">
        {b.products.map((p) => (isRollingStorage(p.id) ? null : <ProductCard key={p.id} p={p} period={b.period} />))}
        <p className="cf-empty cf-empty-quiet">
          KV storage hidden: Cloudflare's dataset reports rolling bytes, not live size (BUG-pFAKDN3) — no overage is priced on it.
        </p>
      </div>

      {/* Telemetry gaps notice */}
      {b.errors.length > 0 ? (
        <section className="cf-card cf-level-warning">
          <h2>Telemetry gaps</h2>
          {b.errors.map((e, i) => (
            <p className="cf-row-detail" key={i}>{e}</p>
          ))}
        </section>
      ) : null}

      {/* Projects table — "who to optimize" */}
      <section className="cf-card">
        <h2>Projects by projected overage</h2>
        {b.projects.length === 0 ? (
          <p className="cf-empty cf-empty-quiet">No project is projected past its allowance.</p>
        ) : (
          <table className="cf-table">
            <thead>
              <tr>
                {["Project", "Projected overage", "Drivers"].map((label) => (
                  <th key={label}>{label}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {b.projects.map((proj) => (
                <tr key={proj.project}>
                  <td>{proj.project}</td>
                  <td className="cf-num">{formatUsd(proj.projectedOverageUsd)}</td>
                  <td className="cf-row-detail">{proj.drivers.join(", ")}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {/* Pricing footnote */}
      <p className="cf-pricing-note">
        {`Prices verified ${b.pricing.verifiedOn} · `}
        {b.pricing.sources.map((s, i) => (
          <Fragment key={s.url}>
            <a className="cf-link" href={s.url} target="_blank" rel="noreferrer">
              {s.product}
            </a>
            {i < b.pricing.sources.length - 1 ? ", " : null}
          </Fragment>
        ))}
      </p>
    </motion.div>
  );
}
