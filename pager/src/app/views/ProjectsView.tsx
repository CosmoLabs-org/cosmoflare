// Projects as a real React view (P-05b, docs/planning-mode/
// 2026-10-10-pager-react-rebuild.md): replaces the imperative bridge to
// projects.ts's renderProjects. One card per project — the exact head the
// vanilla view paints (name + overage pair, no grade derivation), driver
// chips, and the expandable ranked top consumers with share bars. The pure
// helpers (sortProjectsByOverage, projectConsumers) are imported verbatim
// from projects.ts; formatting comes from format.ts / gauges.ts. The
// section masthead owns the page title — the view keeps only the card
// subtitle.

import { useCallback, useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { api, LoginExpiredError, type Billing, type FetchResult, type ProjectCost } from "../../api";
import { formatUsd } from "../../format";
import { productLabel } from "../../gauges";
import { projectConsumers, sortProjectsByOverage } from "../../projects";
import type { ViewFetcher } from "./WorkersView";

/** One project's card: the vanilla head (name + overage pair), the driver
 *  products as muted chips, then the expandable ranked consumers — the
 *  same markup the billing view's top-consumers use, so both read
 *  identically. */
function ProjectCard({ b, p }: { b: Billing; p: ProjectCost }): React.JSX.Element {
  const consumers = projectConsumers(b, p.project);
  return (
    <section className={`cf-card cf-project${p.projectedOverageUsd > 0 ? " cf-over" : ""}`}>
      <div className="cf-project-head">
        <strong className="cf-project-name">{p.project || "(no project)"}</strong>
        <span className={`cf-project-overage ${p.projectedOverageUsd > 0 ? "cf-level-critical-text" : "cf-ok-text"}`}>
          {p.projectedOverageUsd > 0 ? `+${formatUsd(p.projectedOverageUsd)} overage` : "no overage"}
        </span>
      </div>
      {/* Driver products as muted chips: what this project's cost comes from. */}
      {p.drivers.length > 0 ? (
        <div className="cf-chips">
          {p.drivers.map((id) => {
            const prod = b.products.find((x) => x.id === id);
            return <span className="cf-chip" key={id}>{prod ? productLabel(prod.id, prod.product, prod.metric) : id}</span>;
          })}
        </div>
      ) : null}
      {consumers.length > 0 ? (
        <details className="cf-consumers">
          <summary className="cf-consumers-summary">{`Top consumers (${consumers.length})`}</summary>
          <ol className="cf-consumers-list cf-consumers-ranked">
            {consumers.map((c, i) => (
              <li className="cf-consumer" key={`${c.productName}/${c.name}`}>
                <span className="cf-consumer-rank">{String(i + 1)}</span>
                <span className="cf-consumer-who">
                  <span className="cf-consumer-name">{c.name}</span>
                  <span className="cf-consumer-project">{c.productName}</span>
                </span>
                <span className="cf-consumer-sharewrap">
                  <span className="cf-consumer-sharebar">
                    <span
                      className="cf-consumer-sharefill"
                      style={{ width: `${Math.min(100, Math.max(0, c.share * 100))}%` }}
                    />
                  </span>
                  <span className="cf-consumer-share">{`${Math.round(c.share * 100)}%`}</span>
                </span>
              </li>
            ))}
          </ol>
        </details>
      ) : null}
    </section>
  );
}

export default function ProjectsView({ refreshSeq = 0, fetcher }: { refreshSeq?: number; fetcher?: ViewFetcher }): React.JSX.Element {
  // The Retry re-render must go through the SAME fetcher (injected or the
  // shared client) — hence the closure rather than a bare method reference.
  const fetchJson: ViewFetcher = fetcher ?? ((endpoint, opts) => api.fetchJson(endpoint, opts));
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
        const res = await fetchJson<Billing>("api/billing", { refresh, onRevalidate: repaint });
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
        <h2>Could not load projects</h2>
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
  // Worst payer first — projects.ts's sort, verbatim.
  const projects = sortProjectsByOverage(b.projects);

  return (
    <motion.div
      initial={reduced ? false : { opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={reduced ? { duration: 0 } : { duration: 0.2, ease: "easeOut" }}
    >
      {refreshError !== null ? (
        <p className="cf-empty cf-level-warning-text" role="alert">{`Refresh failed — ${refreshError}`}</p>
      ) : null}

      {/* Card subtitle stays; the section masthead owns the page title. */}
      <p className="cf-row-detail">Projected cost beyond allowances, by project, across Cloudflare products.</p>
      <div className="cf-projects-list">
        {projects.length === 0 ? (
          <p className="cf-empty cf-empty-quiet">No projects found on this account.</p>
        ) : (
          projects.map((p) => <ProjectCard key={p.project} b={b} p={p} />)
        )}
      </div>
    </motion.div>
  );
}
