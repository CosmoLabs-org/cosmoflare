// Cosmoflare Pager entry point (FEAT-045): registers the push service
// worker and mounts the two views — alert list and pairing.

import "./styles.css";
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

function mount(): void {
  const root = document.getElementById("app");
  if (!root) return;

  const top = document.createElement("header");
  top.className = "cf-top";
  const brand = document.createElement("h1");
  brand.textContent = "Cosmoflare Pager";
  const tabs = document.createElement("nav");
  tabs.className = "cf-tabs";
  tabs.setAttribute("role", "tablist");
  const alertsTab = document.createElement("button");
  alertsTab.textContent = "Alerts";
  alertsTab.setAttribute("role", "tab");
  alertsTab.setAttribute("aria-selected", "true");
  const pairingTab = document.createElement("button");
  pairingTab.textContent = "Pairing";
  pairingTab.setAttribute("role", "tab");
  pairingTab.setAttribute("aria-selected", "false");
  tabs.append(alertsTab, pairingTab);
  top.append(brand, tabs);

  const main = document.createElement("main");
  main.className = "cf-main";
  const listHost = document.createElement("div");
  const pairingHost = document.createElement("div");
  pairingHost.hidden = true;
  main.append(listHost, pairingHost);

  root.replaceChildren(top, main);

  alertsTab.addEventListener("click", () => {
    alertsTab.setAttribute("aria-selected", "true");
    pairingTab.setAttribute("aria-selected", "false");
    pairingHost.hidden = true;
    listHost.hidden = false;
    renderAlertList(listHost);
  });
  pairingTab.addEventListener("click", () => {
    pairingTab.setAttribute("aria-selected", "true");
    alertsTab.setAttribute("aria-selected", "false");
    listHost.hidden = true;
    pairingHost.hidden = false;
    void renderPairing(pairingHost);
  });

  renderAlertList(listHost);
}

mount();
void registerServiceWorker();
