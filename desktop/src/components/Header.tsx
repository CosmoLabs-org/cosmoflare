// App header: account switcher + the two health indicators. The health dots
// carry `data-online` (string "true"/"false") so tests and tooling can assert
// state machine-readably, plus an aria-label so screen readers announce the
// state (data-* attributes never enter the accessibility tree, and `title`
// never computes into the accessible name).

import { useEffect, useState } from "react";

export interface Account {
  name: string;
}

export interface HeaderProps {
  systemsOnline: boolean;
  cloudflareOnline: boolean;
  accounts: Account[];
  selectedAccount?: string;
  onAccountChange?: (name: string) => void;
}

/* --- Theme toggle (FEAT-041) ---------------------------------------------- */

export type Theme = "light" | "dark";

/** localStorage key persisting the user's explicit theme choice. */
export const THEME_STORAGE_KEY = "cosmoflare-theme";

/**
 * Initial theme resolution: an explicit stored choice wins; otherwise follow
 * the OS `prefers-color-scheme`, defaulting to dark (the app's roots).
 */
export function resolveInitialTheme(): Theme {
  try {
    const stored = localStorage.getItem(THEME_STORAGE_KEY);
    if (stored === "light" || stored === "dark") return stored;
  } catch {
    // localStorage can throw in hardened/embedded contexts — fall through.
  }
  return window.matchMedia?.("(prefers-color-scheme: light)").matches ? "light" : "dark";
}

/** Apply a theme to the document root; styles.css keys off [data-theme]. */
export function applyTheme(theme: Theme): void {
  document.documentElement.dataset.theme = theme;
}

function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>(resolveInitialTheme);

  // Apply on every change (and once on mount, picking up the stored/OS theme).
  useEffect(() => {
    applyTheme(theme);
  }, [theme]);

  const next: Theme = theme === "dark" ? "light" : "dark";
  const toggle = () => {
    setTheme(next);
    try {
      localStorage.setItem(THEME_STORAGE_KEY, next);
    } catch {
      // Persistence is best-effort; the in-memory theme still applies.
    }
  };

  return (
    <button
      type="button"
      data-testid="theme-toggle"
      className="cf-theme-toggle"
      onClick={toggle}
      aria-label={`Switch to ${next} theme`}
      aria-pressed={theme === "light"}
      title={`Switch to ${next} theme`}
    >
      <span aria-hidden>{theme === "dark" ? "☀" : "☾"}</span>
    </button>
  );
}

export function Header({
  systemsOnline,
  cloudflareOnline,
  accounts,
  selectedAccount,
  onAccountChange,
}: HeaderProps) {
  return (
    <header className="cf-header">
      <h1 className="cf-brand">⚡ Cosmoflare</h1>

      {accounts.length > 1 ? (
        <select
          data-testid="account-switcher"
          className="cf-account-switcher"
          value={selectedAccount ?? accounts[0]?.name ?? ""}
          onChange={(e) => onAccountChange?.(e.target.value)}
          aria-label="Account"
        >
          {accounts.map((a) => (
            <option key={a.name} value={a.name}>
              {a.name}
            </option>
          ))}
        </select>
      ) : accounts.length === 1 ? (
        <span data-testid="account-label" className="cf-account-label">
          {accounts[0].name}
        </span>
      ) : null}

      <div className="cf-health">
        <HealthDot testId="health-systems" label="Systems" online={systemsOnline} />
        <HealthDot testId="health-cloudflare" label="Cloudflare" online={cloudflareOnline} />
        <ThemeToggle />
      </div>
    </header>
  );
}

function HealthDot({
  testId,
  label,
  online,
}: {
  testId: string;
  label: string;
  online: boolean;
}) {
  const state = online ? "online" : "offline";
  return (
    <span
      className={`cf-health-dot ${online ? "is-online" : "is-offline"}`}
      data-testid={testId}
      data-online={String(online)}
      title={`${label}: ${state}`}
      aria-label={`${label}: ${state}`}
    >
      <span className="cf-dot-mark" aria-hidden />
      {label}
    </span>
  );
}
