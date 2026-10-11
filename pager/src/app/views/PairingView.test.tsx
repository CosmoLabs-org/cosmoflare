// @vitest-environment jsdom
// PairingView lifecycle polish: the mounted getSubscription check paints
// "This device is paired" (endpoint-host detail, Re-pair + Unpair
// buttons), Unpair unsubscribes and DELETEs the { endpoint } body then
// settles back into the unpaired ritual, the NotAllowedError path shows
// the friendly sentence beside the raw message, and the install hint
// shows off-standalone, hides when dismissed (localStorage-remembered)
// and never shows in standalone. navigator.serviceWorker,
// Notification and matchMedia are stubbed the way the DOM would provide
// them; fetch is a routed lookalike (jsdom has no Response).
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import PairingView from "./PairingView";

// hintStore: a Map-backed localStorage stand-in installed before any view
// mounts (the D1View.test.tsx pattern) — the install hint dismissal must
// survive a re-mount within a test but not across tests.
const hintStore: { map: Map<string, string>; clear(): void } = (() => {
  const store = {
    map: new Map<string, string>(),
    clear(): void {
      store.map.clear();
    },
    getItem(key: string): string | null {
      return store.map.get(key) ?? null;
    },
    setItem(key: string, value: string): void {
      store.map.set(key, value);
    },
    removeItem(key: string): void {
      store.map.delete(key);
    },
  };
  Object.defineProperty(window, "localStorage", { value: store, configurable: true, writable: true });
  return store;
})();

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  Reflect.deleteProperty(navigator, "serviceWorker");
});

beforeEach(() => {
  hintStore.clear();
});

const ENDPOINT = "https://push.example.com/sub/abc123";

type FetchCall = { url: string; init?: RequestInit };

/** Plain response lookalike — jsdom has no Response constructor. */
const jsonResp = (body: unknown, status = 200) => ({
  ok: status >= 200 && status < 300,
  status,
  type: "default",
  json: async () => body,
});

/** Fetch stub routing the VAPID prefetch and /api/subscribe; returns the
 *  recorded calls for method/body assertions. */
function stubFetch(opts: { subscribeStatus?: number } = {}): { calls: FetchCall[] } {
  const calls: FetchCall[] = [];
  vi.stubGlobal("fetch", async (input: unknown, init?: RequestInit) => {
    const url = String(input);
    calls.push({ url: url, init: init });
    if (url === "/api/vapid-public-key") return jsonResp({ key: "TEST-VAPID-KEY" });
    return jsonResp({ ok: true }, opts.subscribeStatus ?? 200);
  });
  return { calls: calls };
}

/** Subscription lookalike with the shape pushManager hands back. */
function makeSubscription(): { endpoint: string; toJSON: () => unknown; unsubscribe: ReturnType<typeof vi.fn> } {
  return {
    endpoint: ENDPOINT,
    toJSON: () => ({ endpoint: ENDPOINT, keys: { p256dh: "p256dh-key", auth: "auth-key" } }),
    unsubscribe: vi.fn(async () => true),
  };
}

/** navigator.serviceWorker stub: stateful like the real one — the mounted
 *  getSubscription probe returns the current subscription and unsubscribe
 *  clears it (so post-unpair re-checks land back on null). */
function stubServiceWorker(initial: { endpoint: string; toJSON: () => unknown; unsubscribe: ReturnType<typeof vi.fn> } | null) {
  const holder: { current: { endpoint: string; toJSON: () => unknown; unsubscribe: ReturnType<typeof vi.fn> } | null } = { current: initial };
  if (initial !== null) {
    initial.unsubscribe.mockImplementation(async () => {
      holder.current = null;
      return true;
    });
  }
  const pushManager = {
    getSubscription: vi.fn(async () => holder.current),
    subscribe: vi.fn(async () => {
      holder.current = makeSubscription();
      return holder.current;
    }),
  };
  Object.defineProperty(navigator, "serviceWorker", {
    value: { ready: Promise.resolve({ pushManager: pushManager }) },
    configurable: true,
  });
  return { pushManager: pushManager };
}

describe("PairingView", () => {
  it("a paired device shows the paired status with the endpoint host and Re-pair/Unpair buttons", async () => {
    stubServiceWorker(makeSubscription());
    stubFetch();
    render(<PairingView />);
    expect(await screen.findByText(/This device is paired/)).toBeTruthy();
    expect(screen.getByText("push.example.com")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Re-pair" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Unpair" })).toBeTruthy();
  });

  it("unpair unsubscribes, DELETEs the endpoint and returns to the unpaired ritual", async () => {
    const sub = makeSubscription();
    stubServiceWorker(sub);
    const { calls } = stubFetch();
    render(<PairingView />);
    await screen.findByText(/This device is paired/);
    fireEvent.click(screen.getByRole("button", { name: "Unpair" }));
    await waitFor(() => expect(sub.unsubscribe).toHaveBeenCalled());
    await screen.findByText(/This device is unpaired/);
    const del = calls.find((c) => c.url === "/api/subscribe" && c.init?.method === "DELETE");
    expect(del).toBeTruthy();
    expect(JSON.parse(String(del?.init?.body))).toEqual({ endpoint: ENDPOINT });
    expect(screen.queryByText(/This device is paired/)).toBeNull();
    expect(screen.getByRole("button", { name: "Enable notifications" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Unpair" })).toBeNull();
  });

  it("the NotAllowedError path shows the friendly sentence AND the raw message", async () => {
    stubServiceWorker(null);
    stubFetch();
    vi.stubGlobal("Notification", {
      requestPermission: async () => {
        const err = new Error("the user dismissed the permission prompt");
        err.name = "NotAllowedError";
        throw err;
      },
    });
    render(<PairingView />);
    fireEvent.click(await screen.findByRole("button", { name: "Enable notifications" }));
    expect(
      await screen.findByText("Notifications are blocked. Enable them for this app in iOS Settings, then tap Enable again."),
    ).toBeTruthy();
    // The raw message stays visible as a muted detail line (it appears in
    // both the checklist row and the status detail).
    expect(screen.getAllByText("the user dismissed the permission prompt").length).toBeGreaterThanOrEqual(1);
  });

  it("the install hint shows off-standalone and stays hidden once dismissed", async () => {
    stubServiceWorker(null);
    stubFetch();
    vi.stubGlobal("matchMedia", (query: string) => ({ matches: false, media: query }));
    render(<PairingView />);
    expect(await screen.findByText(/install the app first/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Dismiss" }));
    await waitFor(() => expect(screen.queryByText(/install the app first/)).toBeNull());
    expect(window.localStorage.getItem("pager-install-hint-dismissed")).toBe("1");
    // The dismissal is remembered across a re-mount.
    cleanup();
    render(<PairingView />);
    await screen.findByText("Notifications are not enabled yet.");
    expect(screen.queryByText(/install the app first/)).toBeNull();
  });

  it("the install hint never shows when running standalone", async () => {
    stubServiceWorker(null);
    stubFetch();
    vi.stubGlobal("matchMedia", (query: string) => ({ matches: query.includes("standalone"), media: query }));
    render(<PairingView />);
    await screen.findByText("Notifications are not enabled yet.");
    expect(screen.queryByText(/install the app first/)).toBeNull();
    expect(screen.queryByRole("button", { name: "Dismiss" })).toBeNull();
  });
});
