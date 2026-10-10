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
 * Pairing view: a centered three-step ritual — allow push, prove it with a
 * test alert, and the CLI copy fallback. Every button reports its own busy
 * state and lands its result in the status line (level-colored), so a tap
 * is never silent.
 */
export async function renderPairing(root: HTMLElement): Promise<void> {
  root.replaceChildren();
  const panel = el("section", "cf-pairing");

  // Hero: the push glyph, the title, one line of purpose.
  const hero = el("div", "cf-pairing-hero");
  const glyph = el("span", "cf-pairing-glyph");
  glyph.innerHTML = PAIRING_GLYPH;
  hero.append(glyph, el("h2", undefined, "Pair this device"), el("p", "cf-pairing-sub", "Over-limit alerts land on this phone."));

  // Load the VAPID key before any tap: Apple asks for subscribe() to follow
  // the user gesture directly, with no network round-trip in between.
  const keyPromise = fetchVapidPublicKey();
  keyPromise.catch(() => undefined); // surfaced on click

  const status = el("p", "cf-pairing-status", "Notifications are not enabled yet.");
  status.setAttribute("role", "status");
  const setStatus = (text: string, level: "info" | "ok" | "warn" | "error" = "info"): void => {
    status.textContent = text;
    status.className = `cf-pairing-status ${PAIRING_STATUS_LEVEL[level]}`;
  };

  // busy() runs an async action with honest button feedback: disabled +
  // busy label while in flight, restored label after.
  async function busy(btn: HTMLButtonElement, busyLabel: string, action: () => Promise<void>): Promise<void> {
    const idleLabel = btn.textContent ?? "";
    btn.disabled = true;
    btn.classList.add("is-busy");
    btn.textContent = busyLabel;
    try {
      await action();
    } finally {
      btn.classList.remove("is-busy");
      if (!btn.classList.contains("is-done")) btn.disabled = false;
      if (!btn.classList.contains("is-done")) btn.textContent = idleLabel;
    }
  }

  const enable = el("button", "cf-btn cf-btn-primary", "Enable notifications") as HTMLButtonElement;
  const test = el("button", "cf-btn", "Send test alert") as HTMLButtonElement;
  const copy = el("button", "cf-btn", "Copy subscription") as HTMLButtonElement;
  copy.disabled = true;
  let subscriptionJson: string | null = null;

  enable.addEventListener("click", () =>
    busy(enable, "Enabling…", async () => {
      try {
        const permission = await Notification.requestPermission();
        if (permission !== "granted") {
          setStatus("Permission denied — push cannot be received.", "error");
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
        subscriptionJson = JSON.stringify(subscription.toJSON());
        // Register the device with the Worker so the cron can page it; the
        // CLI "Copy subscription" path below stays as the secondary option.
        const res = await fetch("/api/subscribe", {
          method: "POST",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          body: subscriptionJson,
        });
        if (res.ok) {
          enable.classList.add("is-done");
          enable.textContent = "Enabled";
          copy.disabled = false;
          setStatus("This device will receive alerts. Send a test to be sure.", "ok");
        } else {
          setStatus(`The Worker rejected the subscription (HTTP ${res.status}). Use "Copy subscription" instead.`, "warn");
        }
      } catch (err) {
        setStatus(`Subscription failed: ${err instanceof Error ? err.message : String(err)}`, "error");
      }
    }),
  );

  test.addEventListener("click", () =>
    busy(test, "Sending…", async () => {
      try {
        // redirect:"manual" (BUG-057 defect 7): an expired Access session
        // answers with a redirect; without this the browser would follow it
        // and the JSON parse would fail with a confusing CORS/network error.
        const res = await fetch("/api/test-fire", { method: "POST", credentials: "same-origin", redirect: "manual" });
        // An opaque redirect or 403 means Access re-auth is needed.
        if (res.type === "opaqueredirect" || res.status === 403) {
          setStatus("Your login expired — close and reopen the app to sign in again.", "warn");
          return;
        }
        const body = (await res.json().catch(() => undefined)) as { sent?: number; issues?: string[] } | undefined;
        if (!res.ok) {
          setStatus(`Test failed (HTTP ${res.status})`, "error");
          return;
        }
        // Surface the Worker's per-delivery issues when it reports any.
        if (body?.issues?.length) {
          setStatus(`Sent ${body.sent ?? 0} — ${body.issues.join(" ")}`, "warn");
        } else if ((body?.sent ?? 0) > 0) {
          setStatus("Test sent — check your notification shade.", "ok");
        } else {
          setStatus("Nothing was sent — no device is subscribed yet. Enable first.", "warn");
        }
      } catch (err) {
        setStatus(`Test failed: ${err instanceof Error ? err.message : String(err)}`, "error");
      }
    }),
  );

  copy.addEventListener("click", () =>
    busy(copy, "Copying…", async () => {
      if (subscriptionJson === null) return;
      try {
        await navigator.clipboard.writeText(subscriptionJson);
        setStatus("Copied. On your machine, run:", "ok");
        panel.append(el("code", "cf-pairing-cmd", PAIRING_COMMAND));
      } catch (err) {
        setStatus(`Copy failed: ${err instanceof Error ? err.message : String(err)}`, "error");
      }
    }),
  );

  // Steps are a real sequence: push permission first, proof second, the
  // CLI fallback last — the numbers encode that order.
  const step = (num: number, label: string, btn: HTMLButtonElement): HTMLElement => {
    const row = el("div", "cf-pairing-step");
    row.append(el("span", "cf-pairing-stepnum", String(num)), el("span", "cf-pairing-steplabel", label));
    row.append(btn);
    return row;
  };

  panel.append(hero, step(1, "Allow push from this phone", enable), step(2, "Prove it with a test", test), step(3, "Fallback — pair from the CLI", copy), status);
  root.append(panel);
}

/** The pairing status line's level classes (existing severity text colors). */
const PAIRING_STATUS_LEVEL: Record<string, string> = {
  info: "",
  ok: "cf-ok-text",
  warn: "cf-level-warning-text",
  error: "cf-level-critical-text",
};

/** The pairing hero mark: a phone emitting a broadcast — the signal arcs
 *  carry .cf-wave and breathe in a staggered loop (see styles; frozen under
 *  prefers-reduced-motion). */
const PAIRING_GLYPH =
  '<svg width="44" height="44" viewBox="0 0 26 24" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><rect x="4" y="2.5" width="10" height="19" rx="2.6" stroke-width="1.7"/><path d="M7.6 18.6h2.8" stroke-width="1.7"/><path class="cf-wave" d="M16.2 8.6a4.6 4.6 0 0 1 0 6.8" stroke-width="1.7"/><path class="cf-wave" d="M19.3 5.6a8.8 8.8 0 0 1 0 12.8" stroke-width="1.7"/><path class="cf-wave" d="M22.4 2.7a12.9 12.9 0 0 1 0 18.6" stroke-width="1.7"/></svg>';
