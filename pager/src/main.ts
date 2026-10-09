// Cosmoflare Ops entry point (FEAT-045, FEAT-052): registers the push
// service worker and mounts the app shell — hash router
// (#/overview #/billing #/workers #/d1 #/zones #/alerts #/pairing), a
// persistent sidebar at ≥1024px, and below that a hamburger drawer with a
// backdrop, focus trap and scroll lock. Views render into per-route hosts
// that persist across route changes, so a background revalidation only ever
// repaints its own route.

import "./styles.css";
import { el } from "./dom";
import { renderOverview } from "./dashboard";
import { renderBilling } from "./billing";
import { renderWorkers, renderD1, renderZones } from "./tables";
import { renderAlertList, renderPairing } from "./views";
import { parseHash, hrefFor, ROUTES, type RouteId } from "./routes";

/** Nav labels; the icons are emoji-free glyphs so the drawer reads fast. */
const NAV: Record<RouteId, { label: string; glyph: string }> = {
  overview: { label: "Overview", glyph: "◈" },
  billing: { label: "Billing", glyph: "◆" },
  workers: { label: "Workers", glyph: "≡" },
  d1: { label: "D1", glyph: "▤" },
  zones: { label: "Zones", glyph: "◍" },
  alerts: { label: "Alerts", glyph: "◆" },
  pairing: { label: "Pairing", glyph: "⚙" },
};

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
  burger.innerHTML = burgerSvg;
  const brand = el("h1", undefined, "Cosmoflare Ops");
  top.append(burger, brand);

  // Skip link for keyboard users.
  const skip = el("a", "cf-skip-link", "Skip to content");
  skip.href = "#main";

  // Persistent sidebar (≥1024px).
  const sidebar = document.createElement("nav");
  sidebar.className = "cf-sidebar";
  sidebar.setAttribute("aria-label", "Sections");

  // Drawer (below 1024px).
  const backdrop = el("div", "cf-backdrop");
  backdrop.hidden = true;
  const drawer = document.createElement("nav");
  drawer.id = "cf-drawer";
  drawer.className = "cf-drawer";
  drawer.setAttribute("aria-label", "Sections");
  drawer.hidden = true;

  // Same links in both navs; route change updates aria-current on both.
  const linkTargets: { a: HTMLAnchorElement; route: RouteId }[] = [];
  for (const route of ROUTES) {
    for (const nav of [sidebar, drawer]) {
      const a = document.createElement("a");
      a.href = hrefFor(route);
      a.className = "cf-navlink";
      a.append(
        el("span", "cf-navglyph", NAV[route].glyph),
        el("span", "cf-navlabel", NAV[route].label),
      );
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

  const shell = el("div", "cf-shell");
  shell.append(sidebar, main);
  root.append(skip, top, shell, backdrop, drawer);

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
    workers: (h, o) => void renderWorkers(h, o ?? {}),
    d1: (h, o) => void renderD1(h, o ?? {}),
    zones: (h, o) => void renderZones(h, o ?? {}),
    alerts: (h) => renderAlertList(h),
    pairing: (h) => void renderPairing(h),
  };

  function showRoute(route: RouteId): void {
    for (const r of ROUTES) hosts[r].hidden = r !== route;
    for (const { a, route: r } of linkTargets) {
      if (r === route) a.setAttribute("aria-current", "page");
      else a.removeAttribute("aria-current");
    }
    RENDER[route](hosts[route]);
    window.scrollTo(0, 0);
  }

  let current: RouteId = initialRoute;
  function onHashChange(): void {
    const next = parseHash(window.location.hash);
    if (next !== current) {
      closeDrawer();
      current = next;
      showRoute(next);
    }
  }
  window.addEventListener("hashchange", onHashChange);
  showRoute(initialRoute);
  burger.addEventListener("click", () => (drawerOpen ? closeDrawer() : openDrawer()));
}

// Inline hamburger glyph (two bars) — no icon dependency.
const burgerSvg = `<svg width="22" height="22" viewBox="0 0 22 22" aria-hidden="true" focusable="false">
  <path d="M3 6.5h16M3 15.5h16" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
</svg>`;

mount();
void registerServiceWorker();
