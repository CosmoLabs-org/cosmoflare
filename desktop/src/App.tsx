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
import { NotificationsPanel } from "./views/Notifications";

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

  // Live health from the SSE status channel.
  useDaemonSSE(endpoint, (e) => {
    if (e.channel !== "status") return;
    const s = e.data as StatusPayload;
    if (typeof s.systems_online === "boolean") setSystemsOnline(s.systems_online);
    if (typeof s.cloudflare_online === "boolean") setCloudflareOnline(s.cloudflare_online);
  });

  // Build the REST client once the endpoint is known.
  const client = useMemo(() => (endpoint ? new ApiClient(endpoint) : null), [endpoint]);

  // Hydrate the account switcher once the endpoint is live (read-only list).
  useEffect(() => {
    if (!endpoint) return;
    let cancelled = false;
    const load = async () => {
      try {
        const resp = await fetch(endpoint.url + "/accounts", {
          headers: { Authorization: `Bearer ${endpoint.token}` },
        });
        if (!resp.ok) return;
        const list = (await resp.json()) as Account[];
        if (cancelled) return;
        setAccounts(list);
        if (list.length && !selected) setSelected(list[0].name);
      } catch {
        // daemon not ready yet — will retry on next render cycle
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [endpoint]);

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
          <aside className="cf-sidebar">
            {SIDEBAR_ITEMS.map((item) => (
              <button
                key={item}
                className={`cf-nav-item ${view === item ? "is-active" : ""}`}
                onClick={() => setView(item)}
              >
                {item}
              </button>
            ))}
          </aside>
          <main className="cf-main">
            {view === "Dashboard" &&
              (client ? <Dashboard client={client} profile={selected} /> : <p>Connecting to daemon…</p>)}
            {view === "Notifications" && <NotificationsPanel endpoint={endpoint} />}
          </main>
        </div>
      </div>
    </QueryClientProvider>
  );
}
