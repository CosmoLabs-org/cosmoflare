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
//
// Pairing lifecycle polish: on mount (after serviceWorker.ready) an
// existing push subscription flips the view into the paired state — "This
// device is paired" with the endpoint host as a muted detail, the Enable
// button becoming a secondary Re-pair beside an Unpair (browser
// unsubscribe + DELETE /api/subscribe with the { endpoint } body
// handleSubscribe validates). Enable renders a compact three-step
// checklist that ticks as it progresses, and failures map to human
// sentences (friendlyEnableError) while the raw message stays visible as
// a second muted detail line for diagnosis. Off-standalone (not installed
// to the home screen) shows a dismissible install hint remembered in
// localStorage under pager-install-hint-dismissed.

import { useEffect, useRef, useState } from "react";
import { fetchVapidPublicKey, urlBase64ToUint8Array } from "../../vapid";
import { PAIRING_COMMAND, PAIRING_GLYPH, PAIRING_STATUS_LEVEL } from "../../views";

type StatusLevel = keyof typeof PAIRING_STATUS_LEVEL;

/** The three ritual steps, in order — the numbers encode it. */
const STEPS: ReadonlyArray<{ num: number; label: string; key: "enable" | "test" | "copy" }> = [
  { num: 1, label: "Allow push from this phone", key: "enable" },
  { num: 2, label: "Prove it with a test", key: "test" },
  { num: 3, label: "Fallback — pair from the CLI", key: "copy" },
];

/** The Enable checklist: what a successful pairing actually does, in
 * order — rendered as muted rows that tick to ok as each step completes;
 * the failing step carries the raw error under its label. */
const ENABLE_STEPS: ReadonlyArray<{ num: 1 | 2 | 3; label: string }> = [
  { num: 1, label: "Allow notifications" },
  { num: 2, label: "Subscribe this device" },
  { num: 3, label: "Register with the Ops worker" },
];

/** StepState is one checklist row: pending while in flight, ok once done,
 * error with the raw message when its step failed. */
type StepState = { state: "pending" | "ok" | "error"; detail?: string };
type EnableSteps = { 1: StepState; 2: StepState; 3: StepState };

/** localStorage key remembering the install hint dismissal. */
const HINT_KEY = "pager-install-hint-dismissed";

/** rawMessage extracts the text for the always-present diagnosis line. */
function rawMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

/** endpointHost renders just the push-service host from an endpoint URL,
 * falling back to the raw string when URL parsing rejects it. */
function endpointHost(endpoint: string): string {
  try {
    return new URL(endpoint).host;
  } catch {
    return endpoint;
  }
}

/** isStandalone: running installed-to-home-screen — iOS reports
 * navigator.standalone, everywhere else display-mode: standalone. */
function isStandalone(): boolean {
  try {
    if (window.matchMedia?.("(display-mode: standalone)").matches) return true;
  } catch {
    // jsdom / old browsers: fall through to the iOS flag
  }
  return (navigator as Navigator & { standalone?: boolean }).standalone === true;
}

/** notAllowedError builds the DOMException-shaped error for a permission
 * result that came back denied without throwing (so both paths share the
 * same friendly text). */
function notAllowedError(): Error {
  const err = new Error("the permission prompt was dismissed or denied");
  err.name = "NotAllowedError";
  return err;
}

/** friendlyEnableError maps a failed Enable attempt to a sentence the
 * operator can act on, keyed by error name first, message second. The raw
 * message always rides along as a muted detail line in the caller. */
function friendlyEnableError(err: unknown): string {
  const name = err instanceof Error ? err.name : "";
  const msg = rawMessage(err).toLowerCase();
  if (name === "NotAllowedError") {
    return "Notifications are blocked. Enable them for this app in iOS Settings, then tap Enable again.";
  }
  if (name === "AbortError" || msg.includes("inactiveregistration") || msg.includes("user aborted")) {
    return "The app just updated. Close it fully and reopen, then tap Enable again.";
  }
  if (name === "LoginExpiredError" || msg.includes("login expired")) {
    return "Your login expired — close and reopen the app to sign in again.";
  }
  // fetchVapidPublicKey's two failure shapes (vapid.ts).
  if (msg.includes("key request failed") || msg.includes("vapid")) {
    return "The server has no push key configured.";
  }
  if (err instanceof TypeError || msg.includes("failed to fetch") || msg.includes("networkerror") || msg.includes("fetch failed") || msg.includes("load failed")) {
    return "Could not reach the Ops worker — check your connection.";
  }
  return `Subscription failed: ${rawMessage(err)}`;
}

export default function PairingView(): React.JSX.Element {
  const [status, setStatus] = useState<{ text: string; level: StatusLevel; detail?: string }>({
    text: "Notifications are not enabled yet.",
    level: "info",
  });
  // Which button is in flight (honest busy feedback: disabled + busy label
  // while in flight, restored after).
  const [busyKey, setBusyKey] = useState<string | null>(null);
  // Paired state: a live push subscription exists for this device (checked
  // on mount after serviceWorker.ready, kept fresh by Enable and Unpair).
  const [paired, setPaired] = useState(false);
  const [subscriptionJson, setSubscriptionJson] = useState<string | null>(null);
  const [showCmd, setShowCmd] = useState(false);
  // The Enable checklist: null until an Enable attempt starts, then one
  // state per step that ticks through pending -> ok / error.
  const [enableSteps, setEnableSteps] = useState<EnableSteps | null>(null);
  const [hintDismissed, setHintDismissed] = useState<boolean>(() => {
    try {
      return window.localStorage.getItem(HINT_KEY) === "1";
    } catch {
      return false; // private mode: show the hint, dismissal just won't stick
    }
  });
  // The VAPID key prefetch: starts once, before any tap (see below).
  const keyPromise = useRef<Promise<string> | null>(null);
  if (keyPromise.current === null) {
    keyPromise.current = fetchVapidPublicKey();
    keyPromise.current.catch(() => undefined); // surfaced on click
  }

  // On mount, after serviceWorker.ready, an existing push subscription
  // means this device is already paired — say so instead of pretending
  // the ritual has not run.
  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const registration = await navigator.serviceWorker.ready;
        const sub = await registration.pushManager.getSubscription();
        if (cancelled) return;
        if (sub !== null) {
          setSubscriptionJson(JSON.stringify(sub.toJSON()));
          setPaired(true);
          setStatus({ text: "This device is paired", level: "ok", detail: endpointHost(sub.endpoint) });
        } else {
          setPaired(false);
        }
      } catch {
        // No service worker (or the check failed): stay unpaired with the
        // default status — the ritual is still the way forward.
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // busy() runs an async action with honest button feedback.
  const run = (key: string, action: () => Promise<void>): void => {
    if (busyKey !== null) return;
    setBusyKey(key);
    void action().finally(() => setBusyKey(null));
  };

  // recheckPaired re-runs the mounted getSubscription check and paints the
  // paired status; returns whether the device is (still) paired. Unpair
  // uses it to settle the view back to the unpaired ritual.
  const recheckPaired = async (): Promise<boolean> => {
    try {
      const registration = await navigator.serviceWorker.ready;
      const sub = await registration.pushManager.getSubscription();
      if (sub !== null) {
        setSubscriptionJson(JSON.stringify(sub.toJSON()));
        setPaired(true);
        setStatus({ text: "This device is paired", level: "ok", detail: endpointHost(sub.endpoint) });
        return true;
      }
      setPaired(false);
      return false;
    } catch {
      return false; // no service worker: treat as unpaired, status stands
    }
  };

  const onEnable = (): void =>
    run("enable", async () => {
      // fail() pins the checklist row to error with the raw message and
      // lands the friendly sentence (raw message as muted detail) in the
      // status line.
      const fail = (step: 1 | 2 | 3, err: unknown): void => {
        const raw = rawMessage(err);
        setEnableSteps((prev) => (prev === null ? prev : { ...prev, [step]: { state: "error", detail: raw } }));
        setStatus({ text: friendlyEnableError(err), level: "error", detail: raw });
      };
      setEnableSteps({ 1: { state: "pending" }, 2: { state: "pending" }, 3: { state: "pending" } });
      try {
        // Step 1: permission.
        let permission: NotificationPermission;
        try {
          permission = await Notification.requestPermission();
        } catch (err) {
          fail(1, err);
          return;
        }
        if (permission !== "granted") {
          fail(1, notAllowedError());
          return;
        }
        setEnableSteps((prev) => (prev === null ? prev : { ...prev, 1: { state: "ok" } }));
        const registration = await navigator.serviceWorker.ready;
        // Safari and Chrome require the VAPID application server key; the
        // key must match the one the sender signs with (FEAT-052 fix). A
        // failed prefetch blocks subscribe, so it counts against step 2.
        let subscription: PushSubscription;
        try {
          const key = await keyPromise.current!;
          subscription = await registration.pushManager.subscribe({
            userVisibleOnly: true,
            applicationServerKey: urlBase64ToUint8Array(key),
          });
        } catch (err) {
          fail(2, err);
          return;
        }
        setEnableSteps((prev) => (prev === null ? prev : { ...prev, 2: { state: "ok" } }));
        const json = JSON.stringify(subscription.toJSON());
        setSubscriptionJson(json);
        // Step 3: register the device with the Worker so the cron can page
        // it; the CLI "Copy subscription" path below stays as the secondary
        // option.
        let res: Response;
        try {
          res = await fetch("/api/subscribe", {
            method: "POST",
            credentials: "same-origin",
            headers: { "Content-Type": "application/json" },
            body: json,
          });
        } catch (err) {
          fail(3, err);
          return;
        }
        if (res.ok) {
          setEnableSteps((prev) => (prev === null ? prev : { ...prev, 3: { state: "ok" } }));
          setPaired(true);
          setStatus({ text: "This device will receive alerts. Send a test to be sure.", level: "ok" });
        } else {
          setEnableSteps((prev) => (prev === null ? prev : { ...prev, 3: { state: "error", detail: `HTTP ${res.status}` } }));
          setStatus({
            text: `The Worker rejected the subscription (HTTP ${res.status}). Use "Copy subscription" instead.`,
            level: "warn",
          });
        }
      } catch (err) {
        // Anything outside the per-step guards (serviceWorker.ready): the
        // subscription phase is the honest bucket for it.
        fail(2, err);
      }
    });

  // onUnpair: drop the browser subscription and tell the Worker to forget
  // the endpoint — the DELETE body is the same { endpoint } shape
  // handleSubscribe validates — then re-check so the view lands back in
  // the unpaired ritual.
  const onUnpair = (): void =>
    run("unpair", async () => {
      try {
        const registration = await navigator.serviceWorker.ready;
        const sub = await registration.pushManager.getSubscription();
        if (sub !== null) {
          const endpoint = sub.endpoint;
          await sub.unsubscribe();
          await fetch("/api/subscribe", {
            method: "DELETE",
            credentials: "same-origin",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ endpoint }),
          });
        }
        setSubscriptionJson(null);
        const stillPaired = await recheckPaired();
        if (!stillPaired) setStatus({ text: "This device is unpaired.", level: "info" });
      } catch (err) {
        setStatus({ text: `Unpair failed: ${rawMessage(err)}`, level: "error" });
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

  const dismissHint = (): void => {
    try {
      window.localStorage.setItem(HINT_KEY, "1");
    } catch {
      // private mode: hide for this visit only
    }
    setHintDismissed(true);
  };

  type ActionKey = "enable" | "test" | "copy" | "unpair";

  const action = (key: ActionKey): (() => void) | undefined => {
    if (key === "enable") return onEnable;
    if (key === "test") return onTest;
    if (key === "unpair") return onUnpair;
    return onCopy;
  };

  const label = (key: ActionKey): string => {
    if (busyKey !== key) {
      if (key === "enable") return paired ? "Re-pair" : "Enable notifications";
      if (key === "test") return "Send test alert";
      if (key === "copy") return "Copy subscription";
      return "Unpair";
    }
    if (key === "enable") return paired ? "Re-pairing…" : "Enabling…";
    return key === "test" ? "Sending…" : key === "copy" ? "Copying…" : "Unpairing…";
  };

  const disabled = (key: ActionKey): boolean => {
    if (key === "enable") return busyKey === "enable"; // paired: Re-pair stays available
    if (key === "test") return busyKey === "test";
    if (key === "copy") return subscriptionJson === null || busyKey === "copy";
    return busyKey === "unpair";
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

      {/* Install hint: pairing only behaves fully from the installed app
          (service worker scope, iOS push permission), so point at Add to
          Home Screen first — dismissible, remembered in localStorage. */}
      {!isStandalone() && !hintDismissed ? (
        <div className="cf-pairing-hint cf-empty cf-empty-quiet" style={{ display: "flex", alignItems: "baseline", justifyContent: "center", gap: 12 }}>
          <span className="cf-row-detail">For alerts, install the app first: Share → Add to Home Screen, then open it from the home screen.</span>
          <button type="button" className="cf-btn" onClick={dismissHint}>
            Dismiss
          </button>
        </div>
      ) : null}

      {STEPS.map(({ num, label: stepLabel, key }) => (
        <div key={key} className="cf-pairing-step">
          <span className="cf-pairing-stepnum">{num}</span>
          <span className="cf-pairing-steplabel">{stepLabel}</span>
          <button
            type="button"
            className={`cf-btn${key === "enable" && !paired ? " cf-btn-primary" : ""}${busyKey === key ? " is-busy" : ""}`}
            disabled={disabled(key)}
            onClick={action(key)}
          >
            {label(key)}
          </button>
          {key === "enable" && paired ? (
            <button type="button" className={`cf-btn${busyKey === "unpair" ? " is-busy" : ""}`} disabled={disabled("unpair")} onClick={onUnpair}>
              {label("unpair")}
            </button>
          ) : null}
        </div>
      ))}

      {/* The Enable checklist: muted rows ticking as the attempt progresses;
          the failing row carries the raw error under its label. */}
      {enableSteps !== null ? (
        <div className="cf-pairing-check" role="list" aria-label="Enable progress">
          {ENABLE_STEPS.map(({ num, label: stepLabel }) => {
            const st = enableSteps[num];
            const cls = st?.state === "ok" ? "cf-ok-text" : st?.state === "error" ? "cf-level-critical-text" : "";
            return (
              <div key={num} role="listitem" className="cf-row-detail" style={{ display: "flex", gap: 8, alignItems: "baseline" }}>
                <span className={cls} aria-hidden="true">
                  {st?.state === "ok" ? "✓" : num}
                </span>
                <span>
                  <span className={cls}>{stepLabel}</span>
                  {st?.state === "error" && st.detail ? (
                    <span className="cf-level-critical-text" style={{ display: "block" }}>
                      {st.detail}
                    </span>
                  ) : null}
                </span>
              </div>
            );
          })}
        </div>
      ) : null}

      {showCmd ? <code className="cf-pairing-cmd">{PAIRING_COMMAND}</code> : null}

      <p className={`cf-pairing-status ${PAIRING_STATUS_LEVEL[status.level]}`} role="status">
        {status.text}
        {/* The diagnosis line: raw errors keep their message visible as a
            second muted detail so the friendly text never hides the cause. */}
        {status.detail ? (
          <span className="cf-row-detail" style={{ display: "block" }}>
            {status.detail}
          </span>
        ) : null}
      </p>
    </section>
  );
}
