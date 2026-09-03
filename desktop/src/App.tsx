// Cosmoflare desktop app shell. Polls the Rust shell for the daemon endpoint
// (it takes a moment to start), then renders the header (account switcher +
// two health dots) wired to the SSE `status` channel, a service sidebar, and a
// main pane that G-07 (dashboard) and G-08 (notifications) populate.

import { useEffect, useMemo, useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Header, type Account } from "./components/Header";
import { ApiClient, type DaemonEndpoint, resolveEndpoint } from "./api/client";
import { useDaemonSSE, type StatusPayload } from "./api/sse";
import { Dashboard } from "./views/Dashboard";
import { Notifications, useNotifications } from "./views/Notifications";

interface MetricsPayload {
  profile: string;
  zones: unknown[];
  r2_buckets: unknown[];
  workers: unknown[];
  kv_namespaces: unknown[];
}

// Single QueryClient for the app (React Query recommendation: create once at
// module scope so it persists across renders). Without this provider, the
// Dashboard's useQuery calls throw "No QueryClient set" at runtime.
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

const SIDEBAR_ITEMS = ["Dashboard", "Notifications"] as const;

export default function App() {
  const [endpoint, setEndpoint] = useState<DaemonEndpoint | null>(null);
  const [systemsOnline, setSystemsOnline] = useState(false);
  const [cloudflareOnline, setCloudflareOnline] = useState(false);
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [selected, setSelected] = useState<string>("");
  const [view, setView] = useState<(typeof SIDEBAR_ITEMS)[number]>("Dashboard");

  // Resolve the daemon endpoint (poll until the handshake completes).
  useEffect(() => {
    let cancelled = false;
    const poll = async () => {
      while (!cancelled) {
        const ep = await resolveEndpoint();
        if (ep) {
          setEndpoint(ep);
          return;
        }
        await new Promise((r) => setTimeout(r, 500));
      }
    };
    void poll();
    return () => {
      cancelled = true;
    };
  }, []);

  // Live health from the SSE status channel. The connection callback drives the
  // "systems" tier directly: an open stream means the daemon is up and
  // answering, so the dot reads online immediately instead of waiting for the
  // first status transition; an error means it is unreachable, so the dot can't
  // falsely read "online" after the daemon dies.
  useDaemonSSE(
    endpoint,
    (e) => {
      if (e.channel === "status") {
        const s = e.data as StatusPayload;
        if (typeof s.systems_online === "boolean") setSystemsOnline(s.systems_online);
        if (typeof s.cloudflare_online === "boolean") setCloudflareOnline(s.cloudflare_online);
      } else if (e.channel === "metrics") {
        const m = e.data as MetricsPayload;
        const profile = m.profile || selected;
        if (m.zones) queryClient.setQueryData(["/zones", profile], m.zones);
        if (m.r2_buckets) queryClient.setQueryData(["/r2/buckets", profile], m.r2_buckets);
        if (m.workers) queryClient.setQueryData(["/workers", profile], m.workers);
        if (m.kv_namespaces) queryClient.setQueryData(["/kv", profile], m.kv_namespaces);
      }
    },
    (connected) => {
      setSystemsOnline(connected);
      if (!connected) setCloudflareOnline(false);
    }
  );

  // Notifications accumulate at the app level (NOT inside the panel) so the
  // history and unread badge survive switching away from the Notifications
  // view — a panel-local hook would unmount and lose all state on every tab
  // change.
  const { items: notifications, unread, markSeen } = useNotifications(endpoint);

  // Build the REST client once the endpoint is known.
  const client = useMemo(() => (endpoint ? new ApiClient(endpoint) : null), [endpoint]);

  // Hydrate the account switcher once the REST client is live (read-only list).
  // Goes through ApiClient so a bad token / unreachable daemon surfaces as an
  // ApiError we log, instead of silently rendering an empty account list.
  useEffect(() => {
    if (!client) return;
    let cancelled = false;
    client
      .get<Account[]>("/accounts")
      .then((list) => {
        if (cancelled) return;
        setAccounts(list);
        // Functional update: auto-select the first account only when none is
        // chosen yet — avoids capturing a stale `selected`.
        setSelected((cur) => cur || (list[0]?.name ?? ""));
      })
      .catch((err) => {
        if (!cancelled) console.error("cosmoflare: failed to load accounts:", err);
      });
    return () => {
      cancelled = true;
    };
  }, [client]);

  return (
    <QueryClientProvider client={queryClient}>
      <div className="cf-app">
        <Header
          systemsOnline={systemsOnline}
          cloudflareOnline={cloudflareOnline}
          accounts={accounts}
          selectedAccount={selected}
          onAccountChange={setSelected}
        />
        <div className="cf-body">
          <nav className="cf-sidebar" aria-label="Views">
            {SIDEBAR_ITEMS.map((item) => (
              <button
                key={item}
                className={`cf-nav-item ${view === item ? "is-active" : ""}`}
                aria-current={view === item ? "page" : undefined}
                onClick={() => setView(item)}
              >
                {item}
              </button>
            ))}
          </nav>
          <main className="cf-main">
            {view === "Dashboard" &&
              (client ? (
                <Dashboard client={client} profile={selected} />
              ) : (
                <p role="status">Connecting to daemon…</p>
              ))}
            {view === "Notifications" && (
              <Notifications items={notifications} unread={unread} onSeen={markSeen} />
            )}
          </main>
        </div>
      </div>
    </QueryClientProvider>
  );
}
