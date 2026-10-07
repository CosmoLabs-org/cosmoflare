// Cosmoflare Pager entry point (FEAT-045).
//
// Views (alert list, detail, ack/snooze, pairing) land in a later task; for
// now this registers the push service worker and renders a placeholder.

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

function main(): void {
  const root = document.getElementById("app");
  if (root) {
    root.textContent = "Cosmoflare Pager";
  }
  void registerServiceWorker();
}

main();
