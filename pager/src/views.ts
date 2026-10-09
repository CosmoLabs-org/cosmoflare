// Cosmoflare Pager views (FEAT-045 Task 7): alert list, detail sheet, and
// the pairing/settings view. Vanilla DOM — no framework. Visual tokens
// mirror desktop/src/styles.css verbatim (dark-first, severity ramp).

import { acknowledge, listAlerts, snooze, type AlertRecord } from "./store";
import { fetchVapidPublicKey, urlBase64ToUint8Array } from "./vapid";

/** The exact command shown next to the copied subscription blob. */
export const PAIRING_COMMAND = "cosmoflare alerts push add";

/** Relative-time formatter: "just now", "4m ago", "2h ago", "3d ago". */
export function relativeTime(epochMs: number, nowMs: number = Date.now()): string {
  const seconds = Math.max(0, Math.floor((nowMs - epochMs) / 1000));
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

function el(tag: string, className?: string, text?: string): HTMLElement {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

function severityClass(severity: string): string {
  return severity === "warning" || severity === "critical" ? severity : "info";
}

function alertCard(record: AlertRecord, onRefresh: () => void): HTMLElement {
  const card = el("article", `cf-alert is-${severityClass(record.severity)}`);
  card.dataset.id = record.id;
  if (record.acked) card.dataset.acked = "true";

  const header = el("div", "cf-alert-header");
  header.append(
    el("span", "cf-severity-mark", record.severity),
    el("strong", "cf-alert-title", record.title),
  );

  const meta = el("div", "cf-alert-meta");
  meta.append(
    el("span", "cf-alert-service", record.service),
    el("time", "cf-alert-time", relativeTime(record.received_at)),
  );

  const actions = el("div", "cf-alert-actions");
  const ack = el("button", "cf-btn", record.acked ? "Acknowledged" : "Acknowledge");
  ack.addEventListener("click", async () => {
    await acknowledge(record.id);
    onRefresh();
  });
  const snz = el("button", "cf-btn cf-btn-ghost", "Snooze 10m");
  snz.addEventListener("click", async () => {
    await snooze(record.id, 10);
    onRefresh();
  });
  actions.append(ack, snz);

  const detail = el("p", "cf-alert-detail", record.detail);
  card.append(header, meta, detail, actions);
  return card;
}

/** Render the alert list into `root`; empty state when nothing visible. */
export function renderAlertList(root: HTMLElement): void {
  root.replaceChildren();
  void listAlerts().then((records) => {
    if (records.length === 0) {
      root.append(el("p", "cf-empty", "No alerts. Your daemon will page you here."));
      return;
    }
    const list = el("div", "cf-alert-list");
    for (const record of records) list.append(alertCard(record, () => renderAlertList(root)));
    root.append(list);
  });
}

/**
 * Pairing view: enable notifications (userVisibleOnly subscription), then
 * copy the subscription blob with the exact CLI command that consumes it.
 */
export async function renderPairing(root: HTMLElement): Promise<void> {
  root.replaceChildren();
  const panel = el("section", "cf-pairing");
  panel.append(el("h2", undefined, "Pair this device"));

  // Load the VAPID key before any tap: Apple asks for subscribe() to follow
  // the user gesture directly, with no network round-trip in between.
  const keyPromise = fetchVapidPublicKey();
  keyPromise.catch(() => undefined); // surfaced on click
  const status = el("p", "cf-pairing-status", "Notifications are not enabled yet.");
  const enable = document.createElement("button");
  enable.className = "cf-btn cf-btn-primary";
  enable.textContent = "Enable notifications";
  const copy = document.createElement("button");
  copy.className = "cf-btn";
  copy.textContent = "Copy subscription";
  copy.disabled = true;
  const test = document.createElement("button");
  test.className = "cf-btn";
  test.textContent = "Send test alert";

  enable.addEventListener("click", async () => {
    try {
      const permission = await Notification.requestPermission();
      if (permission !== "granted") {
        status.textContent = "Permission denied — push cannot be received.";
        return;
      }
      const registration = await navigator.serviceWorker.ready;
      // Safari and Chrome require the VAPID application server key; the
      // key must match the one the sender signs with (FEAT-052 fix).
      const key = await keyPromise;
      const subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(key),
      });
      // Register the device with the Worker so the cron can page it; the
      // CLI "Copy subscription" path below stays as the secondary option.
      const res = await fetch("/api/subscribe", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(subscription.toJSON()),
      });
      if (res.ok) {
        status.textContent = "This device will receive alerts from the Worker.";
      } else {
        status.textContent = `The Worker rejected the subscription (HTTP ${res.status}). Use "Copy subscription" instead.`;
      }
      copy.disabled = false;
      copy.addEventListener("click", async () => {
        await navigator.clipboard.writeText(JSON.stringify(subscription.toJSON()));
        status.textContent = "Copied. On your machine, run:";
        panel.append(el("code", "cf-pairing-cmd", PAIRING_COMMAND));
      });
    } catch (err) {
      status.textContent = `Subscription failed: ${err instanceof Error ? err.message : String(err)}`;
    }
  });

  test.addEventListener("click", async () => {
    test.disabled = true;
    try {
      const res = await fetch("/api/test-fire", { method: "POST", credentials: "same-origin" });
      const body = (await res.json()) as { sent?: number };
      status.textContent = res.ok ? `Test push: sent ${body.sent ?? 0}` : `Test failed (HTTP ${res.status})`;
    } catch (err) {
      status.textContent = `Test failed: ${err instanceof Error ? err.message : String(err)}`;
    } finally {
      test.disabled = false;
    }
  });

  panel.append(status, enable, copy, test);
  root.append(panel);
}
