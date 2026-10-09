// Cosmoflare Ops / Pager entry point (FEAT-045, FEAT-052): registers the
// push service worker and mounts three views — the live dashboard (served by
// the Ops Worker's /api/summary), the alert list, and device pairing.

import "./styles.css";
import { renderDashboard } from "./dashboard";
import { renderAlertList, renderPairing } from "./views";

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

interface Tab {
  label: string;
  render: (host: HTMLElement) => void | Promise<void>;
}

const TABS: Tab[] = [
  { label: "Dashboard", render: (host) => renderDashboard(host) },
  { label: "Alerts", render: (host) => renderAlertList(host) },
  { label: "Pairing", render: (host) => renderPairing(host) },
];

function mount(): void {
  const root = document.getElementById("app");
  if (!root) return;

  const top = document.createElement("header");
  top.className = "cf-top";
  const brand = document.createElement("h1");
  brand.textContent = "Cosmoflare Ops";
  const nav = document.createElement("nav");
  nav.className = "cf-tabs";
  nav.setAttribute("role", "tablist");
  top.append(brand, nav);

  const main = document.createElement("main");
  main.className = "cf-main";
  root.replaceChildren(top, main);

  const hosts = TABS.map(() => document.createElement("div"));
  main.append(...hosts);
  const buttons = TABS.map((tab, i) => {
    const button = document.createElement("button");
    button.textContent = tab.label;
    button.setAttribute("role", "tab");
    button.addEventListener("click", () => select(i));
    nav.append(button);
    return button;
  });

  function select(index: number): void {
    buttons.forEach((b, i) => b.setAttribute("aria-selected", String(i === index)));
    hosts.forEach((h, i) => (h.hidden = i !== index));
    void TABS[index].render(hosts[index]);
  }
  select(0);
}

mount();
void registerServiceWorker();
