// InstallBanner (FEAT-pXNQ5GW phase 1, operator ask "I never got a nice
// button to say install"): a fixed, bottom-of-viewport teach-card that
// appears ONLY in install-eligible, non-standalone, not-yet-dismissed
// contexts. iOS Safari has no beforeinstallprompt, so the iOS branch is a
// teach-card (Share -> Add to Home Screen); Chromium gets a native
// "Install app" button wired to the captured BeforeInstallPromptEvent.
// Tailwind utilities only, mapped onto the @theme tokens (bg-surface-raised,
// border-border, text-muted, ...) so nothing leaks outside the design
// system; styles.css is left untouched. z-index sits UNDER the drawer (40)
// and at the backdrop level (30), above page content. Dismissal is
// remembered in localStorage, quota-safe via try/catch.
import { useCallback, useEffect, useState } from "react";
import { motion, useReducedMotion } from "framer-motion";

/** localStorage flag (quota-safe access only). */
const DISMISS_KEY = "pager-install-hint-dismissed";

/** The subset of BeforeInstallPromptEvent we need (Chrome/Edge only). */
type InstallPromptEvent = Event & { prompt: () => Promise<void> };

function readDismissed(): boolean {
  try {
    return window.localStorage.getItem(DISMISS_KEY) === "1";
  } catch {
    return false;
  }
}

/** Standalone = already installed (display-mode standalone, or the iOS
 *  Safari-only navigator.standalone flag). Installed users never see it. */
function isStandalone(): boolean {
  try {
    if (
      typeof window.matchMedia === "function" &&
      window.matchMedia("(display-mode: standalone)").matches
    ) {
      return true;
    }
  } catch {
    // matchMedia unavailable — fall through to the iOS flag.
  }
  return (navigator as { standalone?: boolean }).standalone === true;
}

function isIOS(): boolean {
  return /iPad|iPhone|iPod/.test(navigator.userAgent);
}

export default function InstallBanner(): React.JSX.Element | null {
  const reduced = useReducedMotion();
  const [standalone] = useState(isStandalone);
  const [dismissed, setDismissed] = useState(readDismissed);
  const [iosDevice] = useState(isIOS);
  // Chromium: the install prompt arrives asynchronously, after mount, so
  // capture it and only then surface the native-install branch.
  const [installPrompt, setInstallPrompt] = useState<InstallPromptEvent | null>(null);

  useEffect(() => {
    if (standalone || iosDevice) return undefined;
    const onPrompt = (e: Event): void => {
      e.preventDefault(); // keep Chrome's own mini-infobar out of the way
      setInstallPrompt(e as InstallPromptEvent);
    };
    window.addEventListener("beforeinstallprompt", onPrompt);
    return () => window.removeEventListener("beforeinstallprompt", onPrompt);
  }, [standalone, iosDevice]);

  const dismiss = useCallback(() => {
    setDismissed(true);
    try {
      window.localStorage.setItem(DISMISS_KEY, "1");
    } catch {
      // Quota/privacy-mode failure: dismissal still holds for this session.
    }
  }, []);

  const install = useCallback(() => {
    void installPrompt?.prompt();
  }, [installPrompt]);

  if (standalone || dismissed) return null;
  // iOS Safari: always eligible (no beforeinstallprompt exists there).
  // Chromium: only once the browser signals installability.
  if (!iosDevice && !installPrompt) return null;

  return (
    <motion.aside
      className="fixed inset-x-0 bottom-9 z-30 border-b border-border bg-surface-raised px-4 py-3 text-text"
      role="complementary"
      aria-label="Install app"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: reduced ? 0 : 0.2 }}
    >
      <div className="mx-auto flex max-w-3xl items-center justify-between gap-4">
        <div className="min-w-0">
          <p className="text-sm font-semibold">Use it like an app</p>
          <p className="text-xs text-muted">
            Share → Add to Home Screen, then open CL Ops from your home screen.
            Alerts need the installed app.
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {installPrompt ? (
            <button
              type="button"
              className="rounded bg-accent px-3 py-1.5 text-sm font-semibold text-bg hover:opacity-90 focus-visible:outline-2 focus-visible:outline-focus"
              onClick={install}
            >
              Install app
            </button>
          ) : null}
          <button
            type="button"
            className="rounded px-2 py-1 text-muted hover:bg-surface hover:text-text focus-visible:outline-2 focus-visible:outline-focus"
            aria-label="Dismiss"
            onClick={dismiss}
          >
            ×
          </button>
        </div>
      </div>
    </motion.aside>
  );
}
