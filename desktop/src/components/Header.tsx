// App header: account switcher + the two health indicators. The health dots
// carry `data-online` (string "true"/"false") so tests and tooling can assert
// state machine-readably, plus an aria-label so screen readers announce the
// state (data-* attributes never enter the accessibility tree, and `title`
// never computes into the accessible name).

import { useEffect, useState } from "react";
import { Logo } from "./Logo";

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
      className="inline-flex h-8 min-w-8 items-center justify-center rounded-md border border-border px-2 text-base leading-none transition-colors duration-150 ease-out hover:border-accent/60"
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
    // NOTE: the account switcher stays a native <select> (styled with the
    // Ops tokens) rather than the Radix-based ui/select.tsx: the test
    // contract reads `.options` off an HTMLSelectElement, and a native
    // select is the stronger a11y primitive in a plain webview anyway.
    <header className="flex flex-none items-center gap-4 border-b border-border bg-surface px-4 py-2">
      <h1 className="m-0 flex items-center gap-2 text-xl font-bold tracking-wide">
        <Logo size={24} />
        <span>Cosmoflare</span>
      </h1>

      {accounts.length > 1 ? (
        <select
          data-testid="account-switcher"
          className="h-8 rounded-md border border-border bg-raised px-2 py-1 text-sm"
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
        <span data-testid="account-label" className="text-sm text-muted-foreground">
          {accounts[0].name}
        </span>
      ) : null}

      <div className="ml-auto flex items-center gap-4">
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
      className={`inline-flex items-center gap-2 text-sm text-foreground ${
        online ? "is-online" : "is-offline"
      }`}
      data-testid={testId}
      data-online={String(online)}
      title={`${label}: ${state}`}
      aria-label={`${label}: ${state}`}
    >
      <span
        aria-hidden
        className={`size-2.5 shrink-0 rounded-full ${
          online ? "bg-success" : "bg-danger"
        }`}
      />
      {label}
    </span>
  );
}
