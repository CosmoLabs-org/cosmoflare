// Cosmoflare Ops entry point (FEAT-045, FEAT-052): registers the push
// service worker and mounts the app shell — hash router
// (#/overview #/billing #/workers #/d1 #/zones #/alerts #/rules #/pairing), a
// persistent sidebar at ≥1024px, and below that a hamburger drawer with a
// backdrop, focus trap and scroll lock. Views render into per-route hosts
// that persist across route changes, so a background revalidation only ever
// repaints its own route.

import "./styles.css";
import { el } from "./dom";
import { renderOverview } from "./dashboard";
import { renderBilling } from "./billing";
import { renderProjects } from "./projects";
import { renderDomains, type DomainsPayload } from "./domains";
import { renderDurableObjects, type DOPayload } from "./durable_objects";
import { renderWorkers, renderD1, renderZones } from "./tables";
import { renderAlertList, renderPairing } from "./views";
import { renderRules, type RulesPayload } from "./rules";
import { renderWorkerProfile } from "./worker_profile";
import { logoMark } from "./logo";
import { api, type Billing, type Summary } from "./api";
import { refreshButton } from "./refresh";
import { startStatusStripClock } from "./statusstrip";

/** Nav counters (operator ask 2026-10-10): right-aligned per-section counts
 *  in the sidebar and drawer — "Domains 43", "Workers 17". Data comes from
 *  the same cached endpoints the views use; a failed fetch leaves that
 *  counter hidden rather than showing a wrong number. */
function setNavCount(route: string, n: number | null): void {
  for (const slot of document.querySelectorAll<HTMLElement>(`.cf-navcount[data-count-for="${route}"]`)) {
    if (n === null || !Number.isFinite(n) || n < 0) {
      slot.hidden = true;
      slot.textContent = "";
    } else {
      slot.hidden = false;
      slot.textContent = String(n);
    }
  }
}

async function loadNavCounts(): Promise<void> {
  const [summary, domains, billing, rules] = await Promise.allSettled([
    api.fetchJson<Summary>("api/summary"),
    api.fetchJson<DomainsPayload>("api/domains"),
    api.fetchJson<Billing>("api/billing"),
    api.fetchJson<RulesPayload>("api/rules"),
  ]);
  if (summary.status === "fulfilled") {
    setNavCount("workers", summary.value.data.workers.length);
    setNavCount("d1", summary.value.data.d1.length);
    setNavCount("zones", summary.value.data.zones.length);
  }
  if (domains.status === "fulfilled") setNavCount("domains", domains.value.data.domains.length);
  const doRes = await Promise.allSettled([api.fetchJson<DOPayload>("api/durable-objects")]);
  if (doRes[0].status === "fulfilled") setNavCount("durable-objects", doRes[0].value.data.namespaces);
  if (billing.status === "fulfilled") setNavCount("projects", billing.value.data.projects.length);
  if (rules.status === "fulfilled") setNavCount("rules", rules.value.data.rules.length);
}
import { parseHash, parseWorkerHash, hrefFor, ROUTES, type RouteId } from "./routes";

/** Nav labels per route. */
const NAV: Record<RouteId, { label: string }> = {
  overview: { label: "Overview" },
  billing: { label: "Billing" },
  projects: { label: "Projects" },
  domains: { label: "Domains" },
  workers: { label: "Workers" },
  d1: { label: "D1" },
  "durable-objects": { label: "Durable Objects" },
  zones: { label: "Zones" },
  alerts: { label: "Alerts" },
  rules: { label: "Rules" },
  pairing: { label: "Pairing" },
};

// One distinct stroke icon per route (18px, currentColor, decorative). Both
// the sidebar and the drawer use them — replaces the unicode glyphs that
// repeated the same diamond for Billing and Alerts.
const ICONS: Record<RouteId, string> = {
  overview: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><rect x="3" y="3" width="7" height="7" rx="1.5"/><rect x="14" y="3" width="7" height="7" rx="1.5"/><rect x="3" y="14" width="7" height="7" rx="1.5"/><rect x="14" y="14" width="7" height="7" rx="1.5"/></svg>`,
  billing: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><rect x="2.5" y="5" width="19" height="14" rx="2"/><path d="M2.5 10h19"/></svg>`,
  // Folder tree — the Projects view groups consumption by project.
  projects: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M3.5 7a1.5 1.5 0 0 1 1.5-1.5h4l2 2.5h8A1.5 1.5 0 0 1 20.5 9.5v8A1.5 1.5 0 0 1 19 19H5a1.5 1.5 0 0 1-1.5-1.5z"/></svg>`,
  // Price tag — the Domains view tracks name ownership and renewal dates.
  domains: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M20.6 13.4 12 22l-8-8 8.6-8.6a2 2 0 0 1 1.4-.6H19a2 2 0 0 1 2 2v4.8a2 2 0 0 1-.4 1.8z"/><circle cx="16" cy="8" r="1.4"/></svg>`,
  workers: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M13 2 4.5 13.5H11L9.5 22 19 10.5h-6.5z"/></svg>`,
  d1: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><ellipse cx="12" cy="5.5" rx="8" ry="3"/><path d="M4 5.5v13c0 1.7 3.6 3 8 3s8-1.3 8-3v-13"/><path d="M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3"/></svg>`,
  // Hexagon pair — Durable Objects: stateful singletons living in cells.
  "durable-objects": `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M8.5 2.5h-4l-2 3.5 2 3.5h4l2-3.5z"/><path d="M19.5 14.5h-4l-2 3.5 2 3.5h4l2-3.5z"/><path d="M9.5 5.8h6.5M12 15.5c0-3.5 2-6 5-7.2"/></svg>`,
  zones: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><path d="M12 3a13.5 13.5 0 0 1 0 18a13.5 13.5 0 0 1 0-18z"/></svg>`,
  alerts: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M18 9a6 6 0 1 0-12 0c0 6-2.5 7-2.5 7h17S18 15 18 9"/><path d="M10 20a2.2 2.2 0 0 0 4 0"/></svg>`,
  pairing: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><rect x="7" y="2.5" width="10" height="19" rx="2.5"/><path d="M10.5 18.5h3"/><path d="M16.5 7a4.5 4.5 0 0 1 2.6 4.1M19.8 4.2a8 8 0 0 1 1.6 4.9"/></svg>`,
  // Sliders — the Rules view edits rule rows (name, condition, threshold).
  rules: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M4 7h9M17 7h3M4 12h3M11 12h9M4 17h9M17 17h3"/><circle cx="15" cy="7" r="2"/><circle cx="9" cy="12" r="2"/><circle cx="15" cy="17" r="2"/></svg>`,
};

// Logo mark lives in ./logo (one source for the inline lockup and the icon set).

/** Brand lockup: logo mark + "CosmoLabs Ops" (primary) with "Cosmoflare"
 *  beneath it as a muted secondary label. Shared by the top bar (h1),
 *  the drawer header and the desktop sidebar. */
function brandLockup(tag: "h1" | "div", idSuffix: string): HTMLElement {
  const lockup = el(tag, "cf-brand");
  lockup.innerHTML = `
    <span class="cf-brand-mark" aria-hidden="true">${logoMark(idSuffix)}</span>
    <span class="cf-brand-text">
      <span class="cf-brand-name">CosmoLabs Ops</span>
      <span class="cf-brand-sub">Cosmoflare</span>
    </span>`;
  return lockup;
}

async function registerServiceWorker(): Promise<void> {
  if (!("serviceWorker" in navigator)) {
    console.warn("pager: service workers unsupported; push delivery disabled");
    return;
  }
  try {
    await navigator.serviceWorker.register("/sw.js");
  } catch (err) {
    console.error("pager: service worker registration failed", err);
  }
}

function mount(): void {
  const root = document.getElementById("app");
  if (!root) return;

  // Top bar: hamburger (below 1024px) + brand + live demo badge slot.
  const top = document.createElement("header");
  top.className = "cf-top";
  const burger = document.createElement("button");
  burger.className = "cf-hamburger";
  burger.setAttribute("aria-label", "Open navigation");
  burger.setAttribute("aria-expanded", "false");
  burger.setAttribute("aria-controls", "cf-drawer");
  burger.innerHTML = burgerBars;
  const brand = brandLockup("h1", "top");
  // Global Refresh (UI-1, operator request 2026-10-10): the refresh button
  // lives in the top bar on every route — always available in the heading —
  // instead of a per-view status bar.
  const topActions = el("div", "cf-top-actions");
  topActions.append(refreshButton(() => refreshCurrentRoute()));
  top.append(burger, brand, topActions);

  // Skip link for keyboard users.
  const skip = el("a", "cf-skip-link", "Skip to content");
  skip.href = "#main";

  // Persistent sidebar (≥1024px).
  const sidebar = document.createElement("nav");
  sidebar.className = "cf-sidebar";
  sidebar.setAttribute("aria-label", "Sections");
  const sidebarBrand = el("div", "cf-sidebar-brand");
  sidebarBrand.append(brandLockup("div", "sidebar"));
  sidebar.append(sidebarBrand);

  // Drawer (below 1024px). The header carries the brand and the close
  // button so the drawer is a complete, closable surface on phones.
  const backdrop = el("div", "cf-backdrop");
  backdrop.hidden = true;
  const drawer = document.createElement("nav");
  drawer.id = "cf-drawer";
  drawer.className = "cf-drawer";
  drawer.setAttribute("aria-label", "Sections");
  drawer.hidden = true;
  const drawerHead = el("div", "cf-drawer-head");
  const drawerClose = el("button", "cf-drawer-close", "×");
  drawerClose.setAttribute("aria-label", "Close menu");
  drawerClose.addEventListener("click", () => closeDrawer());
  drawerHead.append(brandLockup("div", "drawer"), drawerClose);
  drawer.append(drawerHead);

  // Same links in both navs; route change updates aria-current on both.
  // Each link also carries a right-aligned count slot (populated by
  // loadNavCounts below): the sidebar answers "how many do I have?".
  const linkTargets: { a: HTMLAnchorElement; route: RouteId }[] = [];
  for (const route of ROUTES) {
    for (const nav of [sidebar, drawer]) {
      const a = document.createElement("a");
      a.href = hrefFor(route);
      a.className = "cf-navlink";
      const glyph = el("span", "cf-navglyph");
      glyph.innerHTML = ICONS[route];
      const count = el("span", "cf-navcount");
      count.dataset.countFor = route;
      count.hidden = true;
      a.append(glyph, el("span", "cf-navlabel", NAV[route].label), count);
      a.dataset.route = route;
      a.addEventListener("click", () => closeDrawer());
      nav.append(a);
      linkTargets.push({ a, route });
    }
  }

  // Main: one persistent host per route (background revalidations repaint
  // only their own route's host).
  const initialRoute = parseHash(window.location.hash);

  const main = document.createElement("main");
  main.id = "main";
  main.className = "cf-main";
  const hosts = {} as Record<RouteId, HTMLDivElement>;
  for (const route of ROUTES) {
    const host = document.createElement("div");
    host.className = "cf-view";
    host.hidden = route !== initialRoute;
    hosts[route] = host;
    main.append(host);
  }

  // Worker profile host (UI-3): one persistent view for "#/worker/<name>".
  // Like the route hosts it survives navigation, so a refresh only repaints
  // the profile; the global [hidden] CSS rule keeps it out of the layout
  // while a section view is showing.
  const workerHost = document.createElement("div");
  workerHost.className = "cf-view";
  workerHost.id = "cf-view-worker";
  workerHost.hidden = true;
  main.append(workerHost);

  const shell = el("div", "cf-shell");
  shell.append(sidebar, main);

  // Bottom status strip (UI-1): "updated X ago" + billing-period day, one
  // global live region fed by the views as they paint.
  const statusbar = document.createElement("footer");
  statusbar.className = "cf-statusbar";
  statusbar.setAttribute("role", "status");
  statusbar.setAttribute("aria-live", "polite");
  const statusAge = el("span", "cf-status-age");
  statusAge.id = "cf-status-age";
  const statusPeriod = el("span", "cf-status-period");
  statusPeriod.id = "cf-status-period";
  statusPeriod.hidden = true;
  statusbar.append(statusAge, statusPeriod);

  root.append(skip, top, shell, backdrop, drawer, statusbar);
  startStatusStripClock();

  // ---- Drawer behavior ----
  let drawerOpen = false;
  let lastFocused: HTMLElement | null = null;

  function openDrawer(): void {
    drawerOpen = true;
    lastFocused = document.activeElement as HTMLElement | null;
    backdrop.hidden = false;
    drawer.hidden = false;
    // Next frame so the transition plays instead of snapping.
    requestAnimationFrame(() => {
      drawer.classList.add("is-open");
      backdrop.classList.add("is-open");
    });
    burger.setAttribute("aria-expanded", "true");
    document.body.classList.add("cf-nav-open");
    const first = drawer.querySelector<HTMLElement>("a, button");
    first?.focus();
  }

  function closeDrawer(): void {
    if (!drawerOpen) return;
    drawerOpen = false;
    drawer.classList.remove("is-open");
    backdrop.classList.remove("is-open");
    burger.setAttribute("aria-expanded", "false");
    document.body.classList.remove("cf-nav-open");
    // Return focus to the element that opened the drawer (the hamburger),
    // so keyboard and screen-reader users are not left on a hidden element.
    const focusTarget = lastFocused ?? burger;
    focusTarget.focus();
    window.setTimeout(() => {
      if (!drawerOpen) {
        drawer.hidden = true;
        backdrop.hidden = true;
      }
    }, 230);
  }

  // Escape closes; Tab is trapped inside the drawer while open.
  drawer.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      closeDrawer();
      burger.focus();
      return;
    }
    if (e.key !== "Tab") return;
    const focusables = drawer.querySelectorAll<HTMLElement>("a[href], button:not([disabled])");
    if (focusables.length === 0) return;
    const first = focusables[0];
    const last = focusables[focusables.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  });
  backdrop.addEventListener("click", () => {
    closeDrawer();
    burger.focus();
  });

  // ---- Router ----
  const RENDER: Record<RouteId, (host: HTMLElement, opts?: { refresh?: boolean }) => void | Promise<void>> = {
    overview: (h, o) => void renderOverview(h, o),
    billing: (h, o) => void renderBilling(h, o),
    projects: (h, o) => void renderProjects(h, o ?? {}),
    domains: (h, o) => void renderDomains(h, o ?? {}),
    workers: (h, o) => void renderWorkers(h, o ?? {}),
    d1: (h, o) => void renderD1(h, o ?? {}),
    zones: (h, o) => void renderZones(h, o ?? {}),
    "durable-objects": (h, o) => void renderDurableObjects(h, o ?? {}),
    alerts: (h) => renderAlertList(h),
    rules: (h) => void renderRules(h),
    pairing: (h) => void renderPairing(h),
  };

  function showRoute(route: RouteId): void {
    for (const r of ROUTES) hosts[r].hidden = r !== route;
    // UI-3: the worker profile host only ever shows for a worker hash.
    workerHost.hidden = true;
    for (const { a, route: r } of linkTargets) {
      if (r === route) a.setAttribute("aria-current", "page");
      else a.removeAttribute("aria-current");
    }
    RENDER[route](hosts[route]);
    window.scrollTo(0, 0);
  }

  // showWorker renders one worker's profile into the worker host (UI-3).
  // No nav link is current on a profile page — the worker is not a section.
  function showWorker(name: string): void {
    for (const r of ROUTES) hosts[r].hidden = true;
    workerHost.hidden = false;
    for (const { a } of linkTargets) a.removeAttribute("aria-current");
    void renderWorkerProfile(workerHost, name);
    window.scrollTo(0, 0);
  }

  // refreshCurrentRoute re-renders the active route the way the top-bar
  // Refresh button expects: a Promise (void renderers resolve via
  // Promise.resolve), refresh:true so data views revalidate and rethrow on
  // failure — the button keeps its node, shakes and surfaces the error.
  function refreshCurrentRoute(): Promise<void> {
    if (currentKey.startsWith("worker/")) {
      return renderWorkerProfile(workerHost, currentWorkerName() ?? "", { refresh: true });
    }
    if (current === "alerts") {
      renderAlertList(hosts.alerts);
      return Promise.resolve();
    }
    return Promise.resolve(RENDER[current](hosts[current], { refresh: true }));
  }

  // The router's current target: a route id, or "worker/<name>" for a
  // profile page (UI-3). Tracking the raw worker name means navigating
  // worker→worker (a different name) re-renders the profile.
  const currentWorkerName = (): string | null => parseWorkerHash(window.location.hash);
  const targetKey = (): string => {
    const worker = currentWorkerName();
    return worker !== null ? `worker/${worker}` : parseHash(window.location.hash);
  };

  let current: RouteId = initialRoute;
  let currentKey: string = targetKey();
  function onHashChange(): void {
    const key = targetKey();
    if (key === currentKey) return;
    closeDrawer();
    currentKey = key;
    const worker = currentWorkerName();
    if (worker !== null) {
      showWorker(worker);
      return;
    }
    current = parseHash(window.location.hash);
    showRoute(current);
  }
  window.addEventListener("hashchange", onHashChange);
  if (currentKey.startsWith("worker/")) showWorker(currentWorkerName() ?? "");
  else showRoute(initialRoute);
  void loadNavCounts();
  window.setInterval(() => void loadNavCounts(), 5 * 60 * 1000);
  burger.addEventListener("click", () => (drawerOpen ? closeDrawer() : openDrawer()));
}

// Inline hamburger glyph (three bars, middle shorter) — spans so the open
// state can animate them into an X with plain CSS transforms.
const burgerBars = `<span class="cf-burger-box" aria-hidden="true">
  <span class="cf-burger-bar"></span>
  <span class="cf-burger-bar"></span>
  <span class="cf-burger-bar"></span>
</span>`;

mount();
void registerServiceWorker();
