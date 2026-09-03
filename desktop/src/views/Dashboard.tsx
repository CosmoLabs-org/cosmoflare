// Multi-account read-only dashboard. One card per service (Zones, R2, Workers,
// KV), each hydrated via React Query against the daemon's REST endpoints. The
// selected profile is passed as ?profile= on every call so the dashboard
// re-scopes when the header switcher changes — and the profile is in the query
// key, so switching refetches. (BR-06 / read-only: no write "switch".)
//
// `client` is a structural interface so tests pass a stub; the real ApiClient
// satisfies it.

import { useQuery } from "@tanstack/react-query";
import type { ApiClient } from "../api/client";

export interface DashboardClient {
  get<T>(path: string, params?: Record<string, string>): Promise<T>;
}

interface DashboardProps {
  client: ApiClient | DashboardClient;
  profile: string;
}

interface CardProps {
  testId: string;
  label: string;
  data: unknown[] | undefined;
  isLoading: boolean;
  error: unknown;
}

function ServiceCard({ testId, label, data, isLoading, error }: CardProps) {
  const count = Array.isArray(data) ? data.length : 0;
  return (
    <section className="cf-card" data-testid={testId}>
      <h3 className="cf-card-label">{label}</h3>
      {error ? (
        // Real failure reason, announced assertively (role=alert) — the bare
        // word "Error" told the user nothing about what broke.
        <p className="cf-card-count cf-card-error" role="alert">
          {error instanceof Error ? error.message : String(error)}
        </p>
      ) : (
        <p className="cf-card-count">{isLoading ? "Loading…" : String(count)}</p>
      )}
    </section>
  );
}

function useServiceCount(client: DashboardClient, profile: string, path: string) {
  return useQuery({
    queryKey: [path, profile],
    queryFn: () => client.get<unknown[]>(path, { profile }),
  });
}

export function Dashboard({ client, profile }: DashboardProps) {
  const zones = useServiceCount(client, profile, "/zones");
  const r2 = useServiceCount(client, profile, "/r2/buckets");
  const workers = useServiceCount(client, profile, "/workers");
  const kv = useServiceCount(client, profile, "/kv");

  return (
    <div className="cf-dashboard">
      <h2 className="cf-view-title">
        Dashboard{profile ? ` · ${profile}` : ""}
      </h2>
      <div className="cf-card-grid">
        <ServiceCard testId="card-zones" label="Zones" data={zones.data} isLoading={zones.isLoading} error={zones.error} />
        <ServiceCard testId="card-r2" label="R2 Buckets" data={r2.data} isLoading={r2.isLoading} error={r2.error} />
        <ServiceCard testId="card-workers" label="Workers" data={workers.data} isLoading={workers.isLoading} error={workers.error} />
        <ServiceCard testId="card-kv" label="KV Namespaces" data={kv.data} isLoading={kv.isLoading} error={kv.error} />
      </div>
    </div>
  );
}
