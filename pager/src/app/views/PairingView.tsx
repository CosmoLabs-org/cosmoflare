// Pairing as a real React view (P-06, docs/planning-mode/
// 2026-10-10-pager-react-rebuild.md): replaces the imperative bridge to
// views.ts's renderPairing. The same centered three-step ritual — allow
// push (VAPID key prefetched before the tap so subscribe() follows the
// user gesture directly), prove it with a test alert (opaqueredirect/403
// handling), and the CLI copy fallback (PAIRING_COMMAND). Every button
// reports its own busy state and lands its result in the status line
// (level-colored), so a tap is never silent. fetchVapidPublicKey,
// urlBase64ToUint8Array, PAIRING_COMMAND, PAIRING_STATUS_LEVEL and the
// hero glyph are imported verbatim; the section masthead owns the page
// title (no h2 here).

import { useRef, useState } from "react";
import { fetchVapidPublicKey, urlBase64ToUint8Array } from "../../vapid";
import { PAIRING_COMMAND, PAIRING_GLYPH, PAIRING_STATUS_LEVEL } from "../../views";

type StatusLevel = keyof typeof PAIRING_STATUS_LEVEL;

/** The three ritual steps, in order — the numbers encode it. */
const STEPS: ReadonlyArray<{ num: number; label: string; key: "enable" | "test" | "copy" }> = [
  { num: 1, label: "Allow push from this phone", key: "enable" },
  { num: 2, label: "Prove it with a test", key: "test" },
  { num: 3, label: "Fallback — pair from the CLI", key: "copy" },
];

export default function PairingView(): React.JSX.Element {
  const [status, setStatus] = useState<{ text: string; level: StatusLevel }>({
    text: "Notifications are not enabled yet.",
    level: "info",
  });
  // Which button is in flight (honest busy feedback: disabled + busy label
  // while in flight, restored after).
  const [busyKey, setBusyKey] = useState<string | null>(null);
  const [enabled, setEnabled] = useState(false);
  const [subscriptionJson, setSubscriptionJson] = useState<string | null>(null);
  const [showCmd, setShowCmd] = useState(false);
  // The VAPID key prefetch: starts once, before any tap (see below).
  const keyPromise = useRef<Promise<string> | null>(null);
  if (keyPromise.current === null) {
    keyPromise.current = fetchVapidPublicKey();
    keyPromise.current.catch(() => undefined); // surfaced on click
  }

  // busy() runs an async action with honest button feedback.
  const run = (key: string, action: () => Promise<void>): void => {
    if (busyKey !== null) return;
    setBusyKey(key);
    void action().finally(() => setBusyKey(null));
  };

  const onEnable = (): void =>
    run("enable", async () => {
      try {
        const permission = await Notification.requestPermission();
        if (permission !== "granted") {
          setStatus({ text: "Permission denied — push cannot be received.", level: "error" });
          return;
        }
        const registration = await navigator.serviceWorker.ready;
        // Safari and Chrome require the VAPID application server key; the
        // key must match the one the sender signs with (FEAT-052 fix).
        const key = await keyPromise.current!;
        const subscription = await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: urlBase64ToUint8Array(key),
        });
        const json = JSON.stringify(subscription.toJSON());
        setSubscriptionJson(json);
        // Register the device with the Worker so the cron can page it; the
        // CLI "Copy subscription" path below stays as the secondary option.
        const res = await fetch("/api/subscribe", {
          method: "POST",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          body: json,
        });
        if (res.ok) {
          setEnabled(true);
          setStatus({ text: "This device will receive alerts. Send a test to be sure.", level: "ok" });
        } else {
          setStatus({
            text: `The Worker rejected the subscription (HTTP ${res.status}). Use "Copy subscription" instead.`,
            level: "warn",
          });
        }
      } catch (err) {
        setStatus({ text: `Subscription failed: ${err instanceof Error ? err.message : String(err)}`, level: "error" });
      }
    });

  const onTest = (): void =>
    run("test", async () => {
      try {
        // redirect:"manual" (BUG-057 defect 7): an expired Access session
        // answers with a redirect; without this the browser would follow it
        // and the JSON parse would fail with a confusing CORS/network error.
        const res = await fetch("/api/test-fire", { method: "POST", credentials: "same-origin", redirect: "manual" });
        // An opaque redirect or 403 means Access re-auth is needed.
        if (res.type === "opaqueredirect" || res.status === 403) {
          setStatus({ text: "Your login expired — close and reopen the app to sign in again.", level: "warn" });
          return;
        }
        const body = (await res.json().catch(() => undefined)) as { sent?: number; issues?: string[] } | undefined;
        if (!res.ok) {
          setStatus({ text: `Test failed (HTTP ${res.status})`, level: "error" });
          return;
        }
        // Surface the Worker's per-delivery issues when it reports any.
        if (body?.issues?.length) {
          setStatus({ text: `Sent ${body.sent ?? 0} — ${body.issues.join(" ")}`, level: "warn" });
        } else if ((body?.sent ?? 0) > 0) {
          setStatus({ text: "Test sent — check your notification shade.", level: "ok" });
        } else {
          setStatus({ text: "Nothing was sent — no device is subscribed yet. Enable first.", level: "warn" });
        }
      } catch (err) {
        setStatus({ text: `Test failed: ${err instanceof Error ? err.message : String(err)}`, level: "error" });
      }
    });

  const onCopy = (): void =>
    run("copy", async () => {
      if (subscriptionJson === null) return;
      try {
        await navigator.clipboard.writeText(subscriptionJson);
        setStatus({ text: "Copied. On your machine, run:", level: "ok" });
        setShowCmd(true);
      } catch (err) {
        setStatus({ text: `Copy failed: ${err instanceof Error ? err.message : String(err)}`, level: "error" });
      }
    });

  const action = (key: "enable" | "test" | "copy"): (() => void) | undefined => {
    if (key === "enable") return onEnable;
    if (key === "test") return onTest;
    return onCopy;
  };

  const label = (key: "enable" | "test" | "copy"): string => {
    if (busyKey !== key) {
      if (key === "enable" && enabled) return "Enabled";
      return key === "enable" ? "Enable notifications" : key === "test" ? "Send test alert" : "Copy subscription";
    }
    return key === "enable" ? "Enabling…" : key === "test" ? "Sending…" : "Copying…";
  };

  const disabled = (key: "enable" | "test" | "copy"): boolean => {
    if (key === "enable") return enabled || busyKey === "enable";
    if (key === "test") return busyKey === "test";
    return subscriptionJson === null || busyKey === "copy";
  };

  return (
    <section className="cf-pairing">
      {/* Hero: the push glyph, the title, one line of purpose. The section
          masthead owns the page title, so this is a lead-in, not an h2. */}
      <div className="cf-pairing-hero">
        <span className="cf-pairing-glyph" aria-hidden="true" dangerouslySetInnerHTML={{ __html: PAIRING_GLYPH }} />
        <h3 className="cf-pairing-title">Pair this device</h3>
        <p className="cf-pairing-sub">Over-limit alerts land on this phone.</p>
      </div>

      {STEPS.map(({ num, label: stepLabel, key }) => (
        <div key={key} className="cf-pairing-step">
          <span className="cf-pairing-stepnum">{num}</span>
          <span className="cf-pairing-steplabel">{stepLabel}</span>
          <button
            type="button"
            className={`cf-btn${key === "enable" ? " cf-btn-primary" : ""}${busyKey === key ? " is-busy" : ""}${key === "enable" && enabled ? " is-done" : ""}`}
            disabled={disabled(key)}
            onClick={action(key)}
          >
            {label(key)}
          </button>
        </div>
      ))}

      {showCmd ? <code className="cf-pairing-cmd">{PAIRING_COMMAND}</code> : null}

      <p className={`cf-pairing-status ${PAIRING_STATUS_LEVEL[status.level]}`} role="status">
        {status.text}
      </p>
    </section>
  );
}
