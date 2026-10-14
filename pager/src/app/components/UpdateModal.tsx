// UpdateModal (operator 2026-10-12): a new service worker taking control
// (sw.js skipWaiting + clients.claim on every deploy) fires controllerchange
// in every open window — this surfaces that as a centered, smooth dialog so
// the operator can reload into the new version instead of running stale JS.
// Event-driven by design: zero version polling, zero extra requests.
import { useEffect, useRef, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";
import { trapTab } from "./focus";

/** True when a service worker took control of this page AFTER it loaded
 *  (a fresh first install also fires controllerchange — that is not an
 *  update, so only pages that were already controlled count). */
export function useSwUpdateReady(): boolean {
  const [ready, setReady] = useState(false);
  useEffect(() => {
    if (!("serviceWorker" in navigator)) return;
    // Captured at effect time, not in the handler: what matters is whether
    // THIS page load was already controlled when it started.
    const wasControlled = navigator.serviceWorker.controller !== null;
    const onControllerChange = (): void => {
      if (wasControlled) setReady(true);
    };
    navigator.serviceWorker.addEventListener("controllerchange", onControllerChange);
    return () => navigator.serviceWorker.removeEventListener("controllerchange", onControllerChange);
  }, []);
  return ready;
}

export default function UpdateModal({ onReload, onDismiss }: { onReload: () => void; onDismiss: () => void }): React.JSX.Element | null {
  const reduced = useReducedMotion();
  // The button becomes the progress (operator 2026-10-14): tapping Update
  // starts a fill that sweeps the button while the reload is issued — the
  // update is an event you can see, not a silent white flash.
  const [updating, setUpdating] = useState(false);
  const reloadTimer = useRef<number | null>(null);
  useEffect(() => () => {
    if (reloadTimer.current !== null) window.clearTimeout(reloadTimer.current);
  }, []);
  const start = (): void => {
    if (updating) return;
    setUpdating(true);
    reloadTimer.current = window.setTimeout(() => onReload(), reduced ? 150 : 950);
  };
  const dialogRef = useRef<HTMLDivElement | null>(null);
  useEffect(() => {
    const onKey = (event: KeyboardEvent): void => {
      if (event.key === "Escape" && !updating) {
        onDismiss();
        return;
      }
      if (event.key === "Tab" && dialogRef.current) trapTab(dialogRef.current, event);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onDismiss, updating]);

  return (
    <motion.div
      className="cf-modal-backdrop"
      initial={reduced ? false : { opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={reduced ? { duration: 0 } : { duration: 0.15, ease: "easeOut" }}
      onClick={onDismiss}
    >
      <motion.div
        role="dialog"
        aria-modal="true"
        aria-labelledby="cf-update-title"
        ref={dialogRef}
        className="cf-modal"
        initial={reduced ? false : { opacity: 0, scale: 0.96, y: 8 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        transition={reduced ? { duration: 0 } : { duration: 0.2, ease: "easeOut" }}
        onClick={(event) => event.stopPropagation()}
      >
        <p className="cf-modal-eyebrow">CosmoLabs Ops</p>
        <h2 id="cf-update-title" className="cf-modal-title">Update available</h2>
        <p className="cf-modal-body">A new version of the app is ready. Update to get the latest fixes.</p>
        <div className="cf-modal-actions">
          <button
            type="button"
            className="cf-btn cf-btn-primary cf-update-btn"
            autoFocus
            disabled={updating}
            onClick={start}
          >
            {updating ? (
              <>
                <span className="cf-update-progress" aria-hidden="true">
                  <span className="cf-update-progress-fill" />
                </span>
                <span className="cf-update-label">Updating…</span>
              </>
            ) : (
              "Update now"
            )}
          </button>
          {updating ? null : (
            <button type="button" className="cf-btn cf-btn-ghost" onClick={onDismiss}>
              Later
            </button>
          )}
        </div>
      </motion.div>
    </motion.div>
  );
}
