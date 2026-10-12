// UpdateModal (operator 2026-10-12): a new service worker taking control
// (sw.js skipWaiting + clients.claim on every deploy) fires controllerchange
// in every open window — this surfaces that as a centered, smooth dialog so
// the operator can reload into the new version instead of running stale JS.
// Event-driven by design: zero version polling, zero extra requests.
import { useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";

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
  useEffect(() => {
    const onKey = (event: KeyboardEvent): void => {
      if (event.key === "Escape") onDismiss();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onDismiss]);

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
          <button type="button" className="cf-btn cf-btn-primary" autoFocus onClick={onReload}>
            Update now
          </button>
          <button type="button" className="cf-btn cf-btn-ghost" onClick={onDismiss}>
            Later
          </button>
        </div>
      </motion.div>
    </motion.div>
  );
}
