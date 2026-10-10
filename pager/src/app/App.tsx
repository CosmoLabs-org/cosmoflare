// React shell (P-03, docs/planning-mode/2026-10-10-pager-react-rebuild.md):
// the app frame — top bar with the global Refresh, quick nav, drawer,
// persistent per-route view hosts, bottom status strip. Views themselves are
// still the vanilla modules during the port: each ViewHost hands its div to
// the module's render function and re-calls it for refreshes (the imperative
// bridge). They are replaced by real React views tier by tier (P-04..P-06)
// without the shell changing.

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { hrefFor, parseHash, parseWorkerHash, ROUTES, type RouteId } from "../routes";
import { logoMark } from "../logo";
import { api, type Billing, type Summary } from "../api";
import type { DomainsPayload } from "../domains";
import type { RulesPayload } from "../rules";
import type { DOPayload } from "../durable_objects";
import { renderOverview } from "../dashboard";
import { renderBilling } from "../billing";
import { renderProjects } from "../projects";
import { renderDomains } from "../domains";
import { renderWorkers, renderD1, renderZones } from "../tables";
import { renderDurableObjects } from "../durable_objects";
import { renderWorkerProfile } from "../worker_profile";
import { renderAlertList, renderPairing } from "../views";
import { renderRules } from "../rules";
import { startStatusStripClock } from "../statusstrip";

/** Nav labels per route (mirror of the old main.ts NAV). */
const NAV_LABEL: Record<RouteId, string> = {
  overview: "Overview",
  billing: "Billing",
  projects: "Projects",
  domains: "Domains",
  workers: "Workers",
  d1: "D1",
  "durable-objects": "Durable Objects",
  zones: "Zones",
  alerts: "Alerts",
  rules: "Rules",
  pairing: "Pairing",
};

type ViewKind = { kind: "route"; route: RouteId } | { kind: "worker"; name: string };

/** One entry per renderable target: which vanilla module paints the host. */
interface ViewEntry {
  id: string;
  render: (host: HTMLElement, opts: { refresh?: boolean }) => void | Promise<void>;
}

function viewEntry(v: ViewKind): ViewEntry {
  if (v.kind === "worker") {
    return { id: `worker/${v.name}`, render: (h, o) => renderWorkerProfile(h, v.name, o) };
  }
  const map: Record<RouteId, ViewEntry["render"]> = {
    overview: (h, o) => renderOverview(h, o),
    billing: (h, o) => renderBilling(h, o),
    projects: (h, o) => renderProjects(h, o),
    domains: (h, o) => renderDomains(h, o),
    workers: (h, o) => renderWorkers(h, o),
    d1: (h, o) => renderD1(h, o),
    zones: (h, o) => renderZones(h, o),
    "durable-objects": (h, o) => renderDurableObjects(h, o),
    alerts: (h) => renderAlertList(h),
    rules: (h) => void renderRules(h),
    pairing: (h) => void renderPairing(h),
  };
  return { id: v.route, render: map[v.route] };
}

/** The current hash target: a section route or a worker profile. */
function readHash(): ViewKind {
  const worker = parseWorkerHash(window.location.hash);
  if (worker !== null) return { kind: "worker", name: worker };
  return { kind: "route", route: parseHash(window.location.hash) };
}

function useHashView(): ViewKind {
  const [view, setView] = useState<ViewKind>(readHash);
  useEffect(() => {
    const onHash = (): void => setView(readHash());
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);
  return view;
}

/** Nav counters (operator ask 2026-10-10): cached-endpoint counts, refreshed
 *  on mount and every 5 minutes; a failed fetch hides that counter. */
function useNavCounts(): Partial<Record<RouteId, number>> {
  const [counts, setCounts] = useState<Partial<Record<RouteId, number>>>({});
  useEffect(() => {
    let cancelled = false;
    const load = async (): Promise<void> => {
      const [summary, domains, billing, rules, doRes] = await Promise.allSettled([
        api.fetchJson<Summary>("api/summary"),
        api.fetchJson<DomainsPayload>("api/domains"),
        api.fetchJson<Billing>("api/billing"),
        api.fetchJson<RulesPayload>("api/rules"),
        api.fetchJson<DOPayload>("api/durable-objects"),
      ]);
      if (cancelled) return;
      const next: Partial<Record<RouteId, number>> = {};
      if (summary.status === "fulfilled") {
        next.workers = summary.value.data.workers.length;
        next.d1 = summary.value.data.d1.length;
        next.zones = summary.value.data.zones.length;
      }
      if (domains.status === "fulfilled") next.domains = domains.value.data.domains.length;
      if (billing.status === "fulfilled") next.projects = billing.value.data.projects.length;
      if (doRes.status === "fulfilled") next["durable-objects"] = doRes.value.data.namespaces;
      if (rules.status === "fulfilled") next.rules = rules.value.data.rules.length;
      setCounts(next);
    };
    void load();
    const t = window.setInterval(() => void load(), 5 * 60 * 1000);
    return () => {
      cancelled = true;
      window.clearInterval(t);
    };
  }, []);
  return counts;
}

/** The global Refresh: re-invokes the active view with refresh:true — the
 *  vanilla modules rethrow on failure, which we surface on the button. */
function RefreshControl({ onRefresh }: { onRefresh: () => Promise<void> }): React.JSX.Element {
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);
  const reduced = useReducedMotion();
  const run = async (): Promise<void> => {
    if (busy) return;
    setBusy(true);
    setFailed(null);
    try {
      await onRefresh();
      if (!reduced) {
        await new Promise((r) => setTimeout(r, 450)); // success beat
      }
    } catch (err) {
      setFailed(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };
  return (
    <motion.button
      type="button"
      className="cf-btn cf-refresh-btn"
      aria-label={failed ? `Refresh failed — ${failed}` : "Refresh data"}
      aria-busy={busy}
      disabled={busy}
      title={failed ? `Refresh failed — ${failed}` : "Refresh"}
      animate={failed && !reduced ? { x: [0, -4, 4, -3, 3, 0] } : {}}
      transition={{ duration: 0.24 }}
      onClick={() => void run()}
    >
      <motion.span
        className="cf-refresh-icon"
        aria-hidden="true"
        animate={busy && !reduced ? { rotate: 360 } : { rotate: 0 }}
        transition={busy && !reduced ? { repeat: Infinity, duration: 0.9, ease: "linear" } : {}}
        style={{ display: "inline-flex" }}
      >
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round">
          <polyline points="23 4 23 10 17 10" />
          <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10" />
        </svg>
      </motion.span>
    </motion.button>
  );
}

/** Brand lockup (operator 2026-10-10): "CosmoLabs Ops: Section" — the
 *  current section names itself in the top bar — with the bigger mark and
 *  the larger Cosmoflare subline. The sidebar/drawer lockups stay bare. */
function Brand({ tag, section }: { tag: "h1" | "div"; section?: string }): React.JSX.Element {
  const Tag = tag;
  return (
    <Tag className="cf-brand">
      <span className="cf-brand-mark" aria-hidden="true" dangerouslySetInnerHTML={{ __html: logoMark("react") }} />
      <span className="cf-brand-text">
        <span className="cf-brand-name">CosmoLabs Ops{section ? `: ${section}` : ""}</span>
        <span className="cf-brand-sub">Cosmoflare</span>
      </span>
    </Tag>
  );
}

/** NavLink row used by sidebar, drawer and quicknav. */
function NavLinks({
  variant,
  active,
  counts,
  onNavigate,
}: {
  variant: "sidebar" | "drawer" | "quick";
  active: string | null;
  counts: Partial<Record<RouteId, number>>;
  onNavigate?: () => void;
}): React.JSX.Element {
  return (
    <>
      {ROUTES.filter((r) => variant !== "quick" || r !== "pairing").map((r) => (
        <a
          key={`${variant}-${r}`}
          href={hrefFor(r)}
          className={variant === "quick" ? "cf-quicknav-link" : "cf-navlink"}
          aria-current={active === r ? "page" : undefined}
          onClick={onNavigate}
        >
          {variant === "quick" ? null : <span className="cf-navglyph" aria-hidden="true" />}
          <span className={variant === "quick" ? "cf-quicknav-label" : "cf-navlabel"}>{NAV_LABEL[r]}</span>
          <span className="cf-navcount" hidden={!Number.isFinite(counts[r] ?? NaN)}>
            {counts[r] ?? ""}
          </span>
        </a>
      ))}
    </>
  );
}

/** ViewHost: the imperative bridge. Mounts a div, hands it to the vanilla
 *  render on every view change, and reports its refresh function upward so
 *  the top-bar Refresh re-invokes the active view. */
function ViewHost({ entry, onReady }: { entry: ViewEntry; onReady: (fn: () => void) => void }): React.JSX.Element {
  const ref = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    const host = ref.current;
    if (!host) return;
    // The views own their error states (they paint them); the bridge's job
    // is only to keep their async rejections from escaping unhandled.
    void Promise.resolve(entry.render(host, {})).catch(() => undefined);
    onReady(() => void Promise.resolve(entry.render(host, { refresh: true })).catch(() => undefined));
    window.scrollTo(0, 0);
  }, [entry, onReady]);
  return <div ref={ref} className="cf-view" />;
}

export default function App(): React.JSX.Element {
  const view = useHashView();
  const counts = useNavCounts();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const reduced = useReducedMotion();

  useEffect(() => {
    startStatusStripClock();
  }, []);

  // The active view: one mounted host (the old shell kept every route
  // mounted; with the bridge, one active host is enough — route changes
  // repaint, matching the old showRoute behavior).
  const entry = useMemo(() => viewEntry(view), [view]);
  const activeKey = view.kind === "worker" ? `worker/${view.name}` : view.route;
  const [refreshFn, setRefreshFn] = useState<() => void>(() => () => undefined);
  const onReady = useCallback((fn: () => void) => setRefreshFn(() => fn), []);
  const onRefresh = useCallback(async () => {
    refreshFn();
  }, [refreshFn]);

  const closeDrawer = useCallback(() => setDrawerOpen(false), []);

  return (
    <>
      <a className="cf-skip-link" href="#main">Skip to content</a>
      <header className="cf-top">
        <button
          type="button"
          className="cf-hamburger"
          aria-label="Open navigation"
          aria-expanded={drawerOpen}
          aria-controls="cf-drawer"
          onClick={() => setDrawerOpen((o) => !o)}
        >
          <span className="cf-burger-box" aria-hidden="true">
            <span className="cf-burger-bar" />
            <span className="cf-burger-bar" />
            <span className="cf-burger-bar" />
          </span>
        </button>
        <Brand tag="h1" section={view.kind === "route" ? NAV_LABEL[view.route] : view.name} />
        <div className="cf-top-actions">
          <RefreshControl onRefresh={onRefresh} />
        </div>
      </header>

      <nav className="cf-quicknav" aria-label="Sections">
        <NavLinks variant="quick" active={view.kind === "route" ? view.route : null} counts={counts} />
      </nav>

      <div className="cf-shell">
        <nav className="cf-sidebar" aria-label="Sections">
          <div className="cf-sidebar-brand">
            <Brand tag="div" />
          </div>
          <NavLinks variant="sidebar" active={view.kind === "route" ? view.route : null} counts={counts} />
        </nav>
        <main id="main" className="cf-main">
          <ViewHost key={activeKey} entry={entry} onReady={onReady} />
        </main>
      </div>

      <AnimatePresence>
        {drawerOpen ? (
          <>
            <motion.div
              key="backdrop"
              className={`cf-backdrop${drawerOpen ? " is-open" : ""}`}
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: reduced ? 0 : 0.2 }}
              onClick={closeDrawer}
            />
            <motion.nav
              key="drawer"
              id="cf-drawer"
              className="cf-drawer is-open"
              aria-label="Sections"
              initial={{ x: "-100%" }}
              animate={{ x: 0 }}
              exit={{ x: "-100%" }}
              transition={reduced ? { duration: 0 } : { type: "spring", stiffness: 380, damping: 36 }}
            >
              <div className="cf-drawer-head">
                <Brand tag="div" />
                <button type="button" className="cf-drawer-close" aria-label="Close menu" onClick={closeDrawer}>×</button>
              </div>
              <NavLinks variant="drawer" active={view.kind === "route" ? view.route : null} counts={counts} onNavigate={closeDrawer} />
            </motion.nav>
          </>
        ) : null}
      </AnimatePresence>

      <footer className="cf-statusbar" role="status" aria-live="polite">
        <span id="cf-status-age" />
        <span id="cf-status-period" className="cf-status-period" hidden />
      </footer>
    </>
  );
}
