// React shell (P-03, docs/planning-mode/2026-10-10-pager-react-rebuild.md):
// the app frame — top bar with the global Refresh, quick nav, drawer,
// persistent per-route view hosts, bottom status strip. Since P-04 the
// Overview and Billing views, since P-05a the table family (Workers,
// worker profiles, D1, Zones, Durable Objects), since P-05b Projects and
// Domains, and since P-06 Rules, Alerts and Pairing are real React
// components mounted into their host divs — the imperative bridge to the
// vanilla modules is retired; ViewHost renders every view through a
// React root.

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { LiquidGlass } from "apple-liquid-glass-webgl";
import { trapTab } from "./components/focus";

/** The build stamp (stamp-sw.mjs writes it into index.html): identifies
 *  the running build on-device — the drawer shows it. */
const BUILD_ID = document.querySelector('meta[name="pager-build"]')?.getAttribute("content") ?? "";
import {
  Bell,
  BookOpen,
  Boxes,
  CreditCard,
  Database,
  FolderOpen,
  Globe,
  LayoutDashboard,
  Plus,
  SlidersHorizontal,
  Sparkles,
  Tag,
  Zap,
  type LucideIcon,
} from "lucide-react";
import { hrefFor, parseHash, parseWorkerHash, ROUTES, type RouteId } from "../routes";
import { logoMark } from "../logo";
import { api, type Billing, type Summary } from "../api";
import type { DomainsPayload } from "../domains";
import type { RulesPayload } from "../rules";
import type { DOPayload } from "../durable_objects";
import { startStatusStripClock } from "../statusstrip";
import InstallBanner from "./InstallBanner";
import OverviewView from "./views/OverviewView";
import BillingView from "./views/BillingView";
import ProjectsView from "./views/ProjectsView";
import DomainsView from "./views/DomainsView";
import WorkersView from "./views/WorkersView";
import WorkerProfileView from "./views/WorkerProfileView";
import D1View from "./views/D1View";
import ZonesView from "./views/ZonesView";
import DurableObjectsView from "./views/DurableObjectsView";
import AIView from "./views/AIView";
import AlertsView from "./views/AlertsView";
import RulesView from "./views/RulesView";
import PairingView from "./views/PairingView";
import GuidesView from "./views/GuidesView";
import UpdateModal, { useSwUpdateReady } from "./components/UpdateModal";

/** Nav labels per route (mirror of the old main.ts NAV). */
/** Section icons (operator 2026-10-10: icons left of every nav title, above
 *  the quick-nav labels, and pronounced in the section masthead) — lucide,
 *  one distinct glyph per route. Worker profiles borrow the Workers bolt. */
const NAV_ICON: Record<RouteId, LucideIcon> = {
  overview: LayoutDashboard,
  billing: CreditCard,
  ai: Sparkles,
  projects: FolderOpen,
  domains: Tag,
  workers: Zap,
  d1: Database,
  "durable-objects": Boxes,
  zones: Globe,
  alerts: Bell,
  rules: SlidersHorizontal,
  pairing: Bell,
  guides: BookOpen,
};

const NAV_LABEL: Record<RouteId, string> = {
  overview: "Overview",
  billing: "Billing",
  ai: "AI",
  projects: "Projects",
  domains: "Domains",
  workers: "Workers",
  d1: "D1",
  "durable-objects": "Durable Objects",
  zones: "Zones",
  alerts: "Alerts",
  rules: "Rules",
  pairing: "Pairing",
  guides: "Guides",
};

/** Bottom tab bar set (operator 2026-10-12): the five primary sections +
 *  a More tab that opens the drawer for the full list. Phone widths only —
 *  the gating is pure CSS (max-width: 640px), so the component renders
 *  unconditionally and the media query decides visibility. */
const BOTTOM_TABS: RouteId[] = ["overview", "billing", "workers", "domains", "alerts"];

type ViewKind = { kind: "route"; route: RouteId } | { kind: "worker"; name: string };

/** One entry per renderable target: a real React view component (P-04
 *  tier 1: Overview + Billing; P-05a tier 2a: the table family and worker
 *  profiles; P-05b tier 2b: Projects + Domains; P-06 tier 3: Rules,
 *  Alerts + Pairing). Since P-06 every view renders through a React root
 *  — the imperative bridge to the vanilla modules is retired. */
interface ViewEntry {
  id: string;
  Component?: React.ComponentType<{ refreshSeq?: number }>;
}

function viewEntry(v: ViewKind): ViewEntry {
  if (v.kind === "worker") {
    return {
      id: `worker/${v.name}`,
      Component: ({ refreshSeq }: { refreshSeq?: number }) => (
        <WorkerProfileView name={v.name} refreshSeq={refreshSeq} />
      ),
    };
  }
  const map: Record<RouteId, ViewEntry> = {
    overview: { id: "overview", Component: OverviewView },
    billing: { id: "billing", Component: BillingView },
    ai: { id: "ai", Component: AIView },
    projects: { id: "projects", Component: ProjectsView },
    domains: { id: "domains", Component: DomainsView },
    workers: { id: "workers", Component: WorkersView },
    d1: { id: "d1", Component: D1View },
    zones: { id: "zones", Component: ZonesView },
    "durable-objects": { id: "durable-objects", Component: DurableObjectsView },
    alerts: { id: "alerts", Component: AlertsView },
    rules: { id: "rules", Component: RulesView },
    pairing: { id: "pairing", Component: PairingView },
    guides: { id: "guides", Component: GuidesView },
  };
  return map[v.route];
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
    <span className="cf-refresh-wrap">
      {/* The honest data age (operator 2026-10-14): the bottom status
          strip is gone — the pill owns the bottom — so the age lives with
          the control that refreshes, painted imperatively by
          statusstrip.ts. */}
      <span id="cf-status-age" className="cf-top-age" role="status" aria-live="polite" />
      <motion.button
      type="button"
      className="cf-btn cf-refresh-btn"
      aria-label={failed ? `Refresh failed — ${failed}` : "Refresh data"}
      aria-busy={busy}
      disabled={busy}
      title={failed ? `Refresh failed — ${failed}` : "Refresh"}
      animate={
        failed && !reduced
          ? { x: [0, -4, 4, -3, 3, 0], scale: 1 }
          : busy && !reduced
            ? { scale: [1, 1.14, 1] }
            : { scale: 1 }
      }
      transition={
        failed && !reduced
          ? { duration: 0.24 }
          : busy && !reduced
            ? { repeat: Infinity, duration: 1.1, ease: "easeInOut" }
            : { type: "spring", stiffness: 400, damping: 22 }
      }
      whileHover={reduced ? undefined : { scale: 1.06 }}
      whileTap={reduced ? undefined : { scale: 0.9 }}
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
    </span>
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
  if (variant === "quick") {
    return (
      <ul className="cf-quicknav-list">
        {ROUTES.filter((r) => r !== "pairing" && r !== "guides").map((r) => {
          const Icon = NAV_ICON[r];
          return (
            <li key={`quick-${r}`}>
              <a
                href={hrefFor(r)}
                className="cf-quicknav-link"
                data-route={r}
                aria-current={active === r ? "page" : undefined}
                onClick={onNavigate}
              >
                <span className="cf-quicknav-icon" aria-hidden="true">
                  <Icon size={20} strokeWidth={1.8} />
                </span>
                <span className="cf-quicknav-label">{NAV_LABEL[r]}</span>
              </a>
            </li>
          );
        })}
      </ul>
    );
  }
  return (
    <>
      {ROUTES.map((r) => {
        const Icon = NAV_ICON[r];
        return (
          <a
            key={`${variant}-${r}`}
            href={hrefFor(r)}
            className="cf-navlink"
            data-route={r}
            aria-current={active === r ? "page" : undefined}
            onClick={onNavigate}
          >
            <span className="cf-navglyph" aria-hidden="true">
              <Icon size={18} strokeWidth={1.8} />
            </span>
            <span className="cf-navlabel">{NAV_LABEL[r]}</span>
            <span className="cf-navcount" hidden={!Number.isFinite(counts[r] ?? NaN)}>
              {counts[r] ?? ""}
            </span>
          </a>
        );
      })}
    </>
  );
}

/** Mobile bottom tab bar (operator 2026-10-12): the tab at the bottom that
 *  shows the section we're in. Five primary tabs (NAV_ICON at 22px, 10px
 *  labels) + a More tab that toggles the existing drawer for the FULL
 *  section list. Pure CSS visibility gating (max-width: 640px). */
function BottomTabs({
  active,
  moreOpen,
  onMore,
}: {
  active: string | null;
  moreOpen: boolean;
  onMore: () => void;
}): React.JSX.Element {
  // The More tab carries the active mark ONLY when the current section
  // lives in the drawer (D1, DO, AI, Rules, Pairing, Projects) — landing on
  // those used to leave the tab bar with nothing highlighted. Opening the
  // drawer over a PRIMARY section must NOT double-highlight (operator
  // 2026-10-12): the section keeps its mark; More keeps its pressed state.
  const moreActive = active !== null && !BOTTOM_TABS.includes(active as (typeof BOTTOM_TABS)[number]);

  // Liquid glass phase 2 (operator 2026-10-13): real refraction on the tab
  // pill where WebGL2 exists — the CSS ::before material stays as the
  // fallback (fallback: "none" keeps the library from injecting its own),
  // and reduced transparency is respected on both layers independently.
  // One refractive element per the brief's performance guardrail.
  const navRef = useRef<HTMLElement | null>(null);
  useEffect(() => {
    if (!navRef.current || !LiquidGlass.isSupported()) return;
    const glass = new LiquidGlass(navRef.current, {
      shape: "auto",
      // Glass balance (operator 2026-10-14, round seven — "more blur
      // too"): Apple's material is BOTH — heavily frosted content that
      // still bends at the rim. Frost returns on top of the maxed
      // refraction; the lensing reads through the blur's gradients.
      tint: 0.2,
      tintTone: "dark",
      frost: 0.42,
      material: { refraction: 185, edgeReach: 1.1, dispersion: 3.6 },
      fallback: "none",
      maxDpr: 2,
      respectReducedTransparency: true,
    });
    // The backdrop is a DOM capture — iOS lets it go stale (the "burned
    // image"). Two refresh paths, neither mid-scroll (per-frame canvas
    // repaint is the WebKit fixed-tracking breaker, 2026-10-12 class):
    // 1. scroll events settle-debounced at 140ms;
    // 2. a position watch — iOS fires NO scroll event when the page snaps
    //    back from an overscroll rubber-band (operator 2026-10-14: the pill
    //    burnt at the last frame after pulling past the top/bottom), so a
    //    500ms integer compare catches every settle the events miss. At
    //    rest it is one number compare; it only refreshes when the page
    //    actually moved.
    let settle = 0;
    let lastY = window.scrollY;
    const doRefresh = (): void => {
      glass.refresh();
      lastY = window.scrollY;
    };
    const refreshSettled = (): void => {
      window.clearTimeout(settle);
      settle = window.setTimeout(doRefresh, 140);
    };
    const refreshIfVisible = (): void => {
      if (document.visibilityState === "visible") doRefresh();
    };
    const watch = window.setInterval((): void => {
      if (window.scrollY !== lastY) doRefresh();
    }, 500);
    window.addEventListener("scroll", refreshSettled, { passive: true });
    document.addEventListener("visibilitychange", refreshIfVisible);
    return () => {
      window.clearTimeout(settle);
      window.clearInterval(watch);
      window.removeEventListener("scroll", refreshSettled);
      document.removeEventListener("visibilitychange", refreshIfVisible);
      glass.destroy();
    };
  }, []);

  return (
    <nav ref={navRef} className="cf-bottomtabs" aria-label="Sections">
      {BOTTOM_TABS.map((r) => {
        const Icon = NAV_ICON[r];
        return (
          <a
            key={`tab-${r}`}
            href={hrefFor(r)}
            className="cf-bottomtab"
            data-route={r}
            aria-current={active === r ? "page" : undefined}
          >
            <span className="cf-bottomtab-icon" aria-hidden="true">
              <Icon size={22} strokeWidth={1.8} />
            </span>
            <span className="cf-bottomtab-label">{NAV_LABEL[r]}</span>
          </a>
        );
      })}
      {/* The overflow tab (operator 2026-10-14): a compact icon-only Plus
          at the pill's end opens the drawer — the labeled More tab is gone,
          the pill stays tight. */}
      <button
        type="button"
        className="cf-bottomtab cf-bottomtab-more"
        data-route="more"
        aria-label="More sections"
        aria-expanded={moreOpen}
        aria-controls="cf-drawer"
        aria-current={moreActive ? "page" : undefined}
        onClick={onMore}
      >
        <span className="cf-bottomtab-icon" aria-hidden="true">
          <Plus size={22} strokeWidth={1.8} />
        </span>
      </button>
    </nav>
  );
}

/** ViewHost: mounts a div and paints the active view into it. React views
 *  (tier 1: Overview + Billing) get their own small React root created in
 *  the effect and collapsed on unmount; the refresh callback re-renders
 *  with a bumped refreshSeq so every global Refresh click re-fetches. The
 *  remaining views keep the imperative bridge: the div is handed to the
 *  vanilla render on every view change and re-called for refreshes; the
 *  views own their error states, and the bridge only keeps their async
 *  rejections from escaping unhandled. */
function ViewHost({ entry, onReady }: { entry: ViewEntry; onReady: (fn: () => void) => void }): React.JSX.Element {
  const ref = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    const host = ref.current;
    if (!host) return;
    if (entry.Component) {
      const View = entry.Component;
      const root = createRoot(host);
      let seq = 0;
      root.render(<View />);
      onReady(() => {
        seq += 1;
        root.render(<View refreshSeq={seq} />);
      });
      window.scrollTo(0, 0);
      return () => root.unmount();
    }
    return;
  }, [entry, onReady]);
  return <div ref={ref} className="cf-view" />;
}

export default function App(): React.JSX.Element {
  const view = useHashView();
  const counts = useNavCounts();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const reduced = useReducedMotion();
  // A deploy lands while the app is open → the new service worker takes
  // control → the centered UpdateModal offers a one-tap reload (operator
  // 2026-10-12).
  const updateReady = useSwUpdateReady();
  const [updateDismissed, setUpdateDismissed] = useState(false);

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

  // Update checks on open AND on foreground (operator 2026-10-13; hardened
  // 2026-10-14 after "not sure I see the latest"): iOS only re-fetches
  // sw.js on its own schedule; forcing reg.update() whenever the app
  // becomes visible means a deploy that landed while it was closed or
  // backgrounded still installs now — and the UpdateModal
  // (controllerchange) offers the reload.
  useEffect(() => {
    if (!("serviceWorker" in navigator)) return;
    const check = (): void => {
      if (document.visibilityState !== "visible") return;
      void navigator.serviceWorker.getRegistration().then((reg) => reg?.update()).catch(() => undefined);
    };
    check();
    document.addEventListener("visibilitychange", check);
    return () => document.removeEventListener("visibilitychange", check);
  }, []);

  // Live alert appearance (operator 2026-10-12): when the service worker
  // lands a push, it messages every open window — re-run the active view's
  // refresh so a fire shows up while the app is open, not on next open.
  useEffect(() => {
    if (!("serviceWorker" in navigator)) return;
    const onMessage = (event: MessageEvent): void => {
      if ((event.data as { type?: string } | null)?.type === "alerts-updated") refreshFn();
    };
    navigator.serviceWorker.addEventListener("message", onMessage);
    return () => navigator.serviceWorker.removeEventListener("message", onMessage);
  }, [refreshFn]);

  const closeDrawer = useCallback(() => setDrawerOpen(false), []);

  // Drawer keyboard hardening (critique P1): Escape closes, Tab stays
  // inside (the trap), focus moves in on open (the close button — the
  // drawer's first control) and restores to whatever had focus before the
  // drawer opened — the same contract the UpdateModal keeps.
  const drawerCloseRef = useRef<HTMLButtonElement | null>(null);
  const drawerNavRef = useRef<HTMLElement | null>(null);
  useEffect(() => {
    if (!drawerOpen) return;
    const restoreTo = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    drawerCloseRef.current?.focus();
    const onKey = (event: KeyboardEvent): void => {
      if (event.key === "Escape") {
        setDrawerOpen(false);
        return;
      }
      if (event.key === "Tab" && drawerNavRef.current) trapTab(drawerNavRef.current, event);
    };
    window.addEventListener("keydown", onKey);
    return () => {
      window.removeEventListener("keydown", onKey);
      restoreTo?.focus();
    };
  }, [drawerOpen]);

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
          {/* Section masthead (operator 2026-10-10): large heading with the
              section's icon pronounced at its left — one place, every view. */}
          <div
            className="cf-sectionhead"
            data-route={view.kind === "route" ? view.route : "workers"}
          >
            <span className="cf-sectionhead-icon" aria-hidden="true">
              <motion.span
                key={activeKey}
                initial={reduced ? false : { scale: 0.6, opacity: 0 }}
                animate={{ scale: 1, opacity: 1 }}
                transition={
                  reduced
                    ? { duration: 0 }
                    : { type: "spring", stiffness: 420, damping: 24 }
                }
                style={{ display: "inline-flex" }}
              >
                {view.kind === "route"
                  ? (() => {
                      const Icon = NAV_ICON[view.route];
                      return <Icon size={22} strokeWidth={1.9} />;
                    })()
                  : (() => {
                      const Icon = NAV_ICON.workers;
                      return <Icon size={22} strokeWidth={1.9} />;
                    })()}
              </motion.span>
            </span>
            <h2 className="cf-sectionhead-title">
              {view.kind === "route" ? NAV_LABEL[view.route] : view.name}
              {view.kind === "route" && Number.isFinite(counts[view.route]) ? (
                <span className="cf-sectionhead-count"> ({counts[view.route]})</span>
              ) : null}
            </h2>
          </div>
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
              ref={drawerNavRef}
              className="cf-drawer is-open"
              aria-label="Sections"
              initial={{ x: "-100%" }}
              animate={{ x: 0 }}
              exit={{ x: "-100%" }}
              transition={reduced ? { duration: 0 } : { type: "spring", stiffness: 380, damping: 36 }}
            >
              <div className="cf-drawer-head">
                <Brand tag="div" />
                <button type="button" className="cf-drawer-close" aria-label="Close menu" onClick={closeDrawer} ref={drawerCloseRef}>×</button>
              </div>
              <NavLinks variant="drawer" active={view.kind === "route" ? view.route : null} counts={counts} onNavigate={closeDrawer} />
              {/* The build marker (operator 2026-10-14): stamped into
                  index.html at build time — open the drawer, read the
                  build, compare with the deploy. "Am I on the latest?"
                  stays answerable on-device. */}
              {BUILD_ID ? <p className="cf-drawer-build">{`build ${BUILD_ID}`}</p> : null}
            </motion.nav>
          </>
        ) : null}
      </AnimatePresence>

      {/* Install guidance (FEAT-pXNQ5GW): self-gating — renders nothing in
          standalone, dismissed or not-yet-eligible contexts. */}
      <InstallBanner />

      {/* Mobile bottom tabs (operator 2026-10-12): stacked under the
          statusbar at the viewport bottom — the CSS media query decides
          whether it shows (phone widths only). */}
      <BottomTabs
        active={view.kind === "route" ? view.route : null}
        moreOpen={drawerOpen}
        onMore={() => setDrawerOpen(true)}
      />

      {/* New version ready (operator 2026-10-12): centered dialog the moment
          a deploy's service worker takes control while the app is open. */}
      {updateReady && !updateDismissed ? (
        <UpdateModal onReload={() => window.location.reload()} onDismiss={() => setUpdateDismissed(true)} />
      ) : null}
    </>
  );
}
