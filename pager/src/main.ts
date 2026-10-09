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
import { renderWorkers, renderD1, renderZones } from "./tables";
import { renderAlertList, renderPairing } from "./views";
import { renderRules } from "./rules";
import { parseHash, hrefFor, ROUTES, type RouteId } from "./routes";

/** Nav labels per route. */
const NAV: Record<RouteId, { label: string }> = {
  overview: { label: "Overview" },
  billing: { label: "Billing" },
  workers: { label: "Workers" },
  d1: { label: "D1" },
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
  workers: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M13 2 4.5 13.5H11L9.5 22 19 10.5h-6.5z"/></svg>`,
  d1: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><ellipse cx="12" cy="5.5" rx="8" ry="3"/><path d="M4 5.5v13c0 1.7 3.6 3 8 3s8-1.3 8-3v-13"/><path d="M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3"/></svg>`,
  zones: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><path d="M12 3a13.5 13.5 0 0 1 0 18a13.5 13.5 0 0 1 0-18z"/></svg>`,
  alerts: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M18 9a6 6 0 1 0-12 0c0 6-2.5 7-2.5 7h17S18 15 18 9"/><path d="M10 20a2.2 2.2 0 0 0 4 0"/></svg>`,
  pairing: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><rect x="7" y="2.5" width="10" height="19" rx="2.5"/><path d="M10.5 18.5h3"/><path d="M16.5 7a4.5 4.5 0 0 1 2.6 4.1M19.8 4.2a8 8 0 0 1 1.6 4.9"/></svg>`,
  // Sliders — the Rules view edits rule rows (name, condition, threshold).
  rules: `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path d="M4 7h9M17 7h3M4 12h3M11 12h9M4 17h9M17 17h3"/><circle cx="15" cy="7" r="2"/><circle cx="9" cy="12" r="2"/><circle cx="15" cy="17" r="2"/></svg>`,
};

// Inline logo mark (FEAT-052): a monitoring pulse crossing an orbit ring —
// operations on Cloudflare's edge. Same geometry as /icon-master.svg, amber
// #f59e0b primary plus one cool secondary #7dd3fc satellite tick. The dark
// halos become transparent via a mask so the gap shows the real background
// on any surface. `idSuffix` keeps the mask id unique per lockup instance.
function logoMark(idSuffix: string): string {
  const maskId = `cf-ops-pulse-${idSuffix}`;
  return `<svg width="24" height="24" viewBox="0 0 512 512" aria-hidden="true" focusable="false">
  <defs><mask id="${maskId}">
    <rect width="512" height="512" fill="#fff"/>
    <path d="M 70 256 H 190 l 26 -80 l 48 160 l 26 -80 H 442" fill="none" stroke="#000" stroke-width="118" stroke-linecap="round" stroke-linejoin="round"/>
    <circle cx="150" cy="150" r="38" fill="#000"/>
  </mask></defs>
  <circle cx="256" cy="256" r="150" fill="none" stroke="#f59e0b" stroke-width="54" mask="url(#${maskId})"/>
  <path d="M 70 256 H 190 l 26 -80 l 48 160 l 26 -80 H 442" fill="none" stroke="#f59e0b" stroke-width="50" stroke-linecap="round" stroke-linejoin="round"/>
  <circle cx="150" cy="150" r="21" fill="#7dd3fc"/>
</svg>`;
}

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
  top.append(burger, brand);

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
  const linkTargets: { a: HTMLAnchorElement; route: RouteId }[] = [];
  for (const route of ROUTES) {
    for (const nav of [sidebar, drawer]) {
      const a = document.createElement("a");
      a.href = hrefFor(route);
      a.className = "cf-navlink";
      const glyph = el("span", "cf-navglyph");
      glyph.innerHTML = ICONS[route];
      a.append(
        glyph,
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
    workers: (h, o) => void renderWorkers(h, o ?? {}),
    d1: (h, o) => void renderD1(h, o ?? {}),
    zones: (h, o) => void renderZones(h, o ?? {}),
    alerts: (h) => renderAlertList(h),
    rules: (h) => void renderRules(h),
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

// Inline hamburger glyph (three bars, middle shorter) — spans so the open
// state can animate them into an X with plain CSS transforms.
const burgerBars = `<span class="cf-burger-box" aria-hidden="true">
  <span class="cf-burger-bar"></span>
  <span class="cf-burger-bar"></span>
  <span class="cf-burger-bar"></span>
</span>`;

mount();
void registerServiceWorker();
