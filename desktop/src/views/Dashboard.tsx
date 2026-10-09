// Multi-account read-only dashboard. One shadcn Card per service (Zones, R2,
// Workers, KV), each hydrated via React Query against the daemon's REST
// endpoints. The selected profile is passed as ?profile= on every call so the
// dashboard re-scopes when the header switcher changes — and the profile is in
// the query key, so switching refetches. (BR-06 / read-only: no write "switch".)
//
// `client` is a structural interface so tests pass a stub; the real ApiClient
// satisfies it.

import { useQuery } from "@tanstack/react-query";
import { Archive, Database, Globe, Server, type LucideIcon } from "lucide-react";
import type { ApiClient } from "../api/client";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";

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
  icon: LucideIcon;
  data: unknown[] | undefined;
  isLoading: boolean;
  error: unknown;
}

/** How many names a card lists inline before scrolling kicks in. */
const INLINE_NAMES = 6;

/** Best-effort display name for an API resource ({name} first, then {id}). */
function resourceName(item: unknown): string {
  if (item && typeof item === "object") {
    const rec = item as Record<string, unknown>;
    if (typeof rec.name === "string") return rec.name;
    if (typeof rec.id === "string") return rec.id;
  }
  return String(item);
}

function ServiceCard({ testId, label, icon: Icon, data, isLoading, error }: CardProps) {
  const names = Array.isArray(data) ? data.map(resourceName) : [];
  const count = names.length;

  return (
    <Card data-testid={testId}>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Icon aria-hidden className="size-4 shrink-0" />
          <span>{label}</span>
          {!isLoading && !error && (
            <Badge variant="secondary" className="ml-auto">
              {count}
            </Badge>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-1 text-sm font-medium">
        {error ? (
          // Real failure reason, announced assertively (role=alert) — the bare
          // word "Error" told the user nothing about what broke.
          <p className="m-0 wrap-anywhere text-sm font-medium text-danger" role="alert">
            {error instanceof Error ? error.message : String(error)}
          </p>
        ) : isLoading ? (
          // Skeleton loading state (animate-pulse, no text spinners). The
          // visually-hidden "Loading" keeps the loading state legible to
          // screen readers and text-based assertions.
          <div className="flex flex-col gap-1.5" aria-busy="true">
            <span className="sr-only">Loading</span>
            <div className="h-4 w-3/4 animate-pulse rounded bg-raised" />
            <div className="h-3 w-1/2 animate-pulse rounded bg-raised" />
          </div>
        ) : names.length === 0 ? (
          <p className="m-0 text-sm text-muted-foreground">No resources.</p>
        ) : names.length > INLINE_NAMES ? (
          <ScrollArea className="-mr-2 max-h-40 pr-2">
            <NameList names={names} />
          </ScrollArea>
        ) : (
          <NameList names={names} />
        )}
      </CardContent>
    </Card>
  );
}

function NameList({ names }: { names: string[] }) {
  return (
    <ul className="m-0 flex list-none flex-col gap-1 p-0 text-sm text-muted-foreground">
      {names.map((name, i) => (
        <li key={`${name}-${i}`} className="truncate">
          {name}
        </li>
      ))}
    </ul>
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
    <div className="flex flex-col gap-4">
      <h2 className="cf-view-title">Dashboard{profile ? ` · ${profile}` : ""}</h2>
      {/* auto-fit collapses to a single column on narrow viewports and fills
          the row on wide ones — no breakpoints needed. */}
      <div className="grid gap-4 [grid-template-columns:repeat(auto-fit,minmax(min(100%,240px),1fr))]">
        <ServiceCard
          testId="card-zones"
          label="Zones"
          icon={Globe}
          data={zones.data}
          isLoading={zones.isLoading}
          error={zones.error}
        />
        <ServiceCard
          testId="card-r2"
          label="R2 Buckets"
          icon={Archive}
          data={r2.data}
          isLoading={r2.isLoading}
          error={r2.error}
        />
        <ServiceCard
          testId="card-workers"
          label="Workers"
          icon={Server}
          data={workers.data}
          isLoading={workers.isLoading}
          error={workers.error}
        />
        <ServiceCard
          testId="card-kv"
          label="KV Namespaces"
          icon={Database}
          data={kv.data}
          isLoading={kv.isLoading}
          error={kv.error}
        />
      </div>
    </div>
  );
}
