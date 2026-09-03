// App header: account switcher + the two health indicators. The health dots
// carry `data-online` (string "true"/"false") so tests and tooling can assert
// state machine-readably, plus an aria-label so screen readers announce the
// state (data-* attributes never enter the accessibility tree, and `title`
// never computes into the accessible name).

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
