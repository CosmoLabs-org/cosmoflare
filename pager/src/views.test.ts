import { afterEach, describe, expect, it, vi } from "vitest";
import { renderPairing } from "./views";

// Node-environment DOM stubs: vitest here runs with environment "node" and no
// DOM library is installed, so renderPairing is exercised against minimal fake
// elements that record text/children and capture click listeners.
type Handler = () => unknown;
interface FakeNode {
  tagName: string;
  className: string;
  textContent: string;
  dataset: Record<string, string>;
  disabled: boolean;
  children: FakeNode[];
  classList: { add: (...c: string[]) => void; remove: (...c: string[]) => void; contains: (c: string) => boolean };
  setAttribute: (name: string, value: string) => void;
  append: (...nodes: FakeNode[]) => void;
  replaceChildren: () => void;
  addEventListener: (type: string, fn: Handler) => void;
  _click: () => Promise<void>;
}

function fakeEl(tag: string): FakeNode {
  const handlers: Record<string, Handler[]> = {};
  const classes = new Set<string>();
  const node: FakeNode = {
    tagName: tag.toUpperCase(),
    className: "",
    textContent: "",
    dataset: {},
    disabled: false,
    children: [],
    classList: {
      add: (...c) => c.forEach((x) => classes.add(x)),
      remove: (...c) => c.forEach((x) => classes.delete(x)),
      contains: (c) => classes.has(c),
    },
    setAttribute: () => undefined,
    append(...nodes) {
      node.children.push(...nodes);
    },
    replaceChildren() {
      node.children = [];
    },
    addEventListener(type, fn) {
      (handlers[type] ??= []).push(fn);
    },
    async _click() {
      for (const fn of handlers.click ?? []) await fn();
    },
  };
  return node;
}

function findButton(root: FakeNode, text: string): FakeNode {
  const stack = [...root.children];
  while (stack.length > 0) {
    const node = stack.shift()!;
    if (node.tagName === "BUTTON" && node.textContent === text) return node;
    stack.unshift(...node.children);
  }
  throw new Error(`button not found: ${text}`);
}

function findByClass(root: FakeNode, className: string): FakeNode {
  const stack = [...root.children];
  while (stack.length > 0) {
    const node = stack.shift()!;
    // Token match: elements may carry additional state classes (e.g. the
    // pairing status line appends a severity class).
    if (node.className === className || node.className.split(/\s+/).includes(className)) return node;
    stack.unshift(...node.children);
  }
  throw new Error(`element not found: .${className}`);
}

const SUB = { endpoint: "https://push.example/dev1", keys: { p256dh: "p256dh-key", auth: "auth-key" } };

function stubEnv() {
  const calls: string[] = [];
  const bodies: string[] = [];
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    calls.push(url);
    if (init?.body) bodies.push(String(init.body));
    if (url.endsWith("/api/vapid-public-key")) return new Response(JSON.stringify({ key: "TESTVAPIDKEY" }), { status: 200 });
    if (url.endsWith("/api/subscribe")) return new Response(JSON.stringify({ ok: true, count: 1 }), { status: 201 });
    if (url.endsWith("/api/test-fire")) return new Response(JSON.stringify({ sent: 2, pruned: 0, issues: [] }), { status: 200 });
    return new Response("not found", { status: 404 });
  });
  vi.stubGlobal("fetch", fetchMock);
  vi.stubGlobal("Notification", { requestPermission: vi.fn(async () => "granted") });
  const subscribe = vi.fn(async () => ({ toJSON: () => SUB }));
  vi.stubGlobal("navigator", {
    serviceWorker: { ready: Promise.resolve({ pushManager: { subscribe } }) },
  });
  vi.stubGlobal("document", { createElement: (tag: string) => fakeEl(tag) });
  return { calls, bodies, fetchMock, subscribe };
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("renderPairing", () => {
  it("prefetches the VAPID key before any tap (Apple gesture rule)", async () => {
    const { calls } = stubEnv();
    const root = fakeEl("div");
    await renderPairing(root as unknown as HTMLElement);
    expect(calls).toEqual(["/api/vapid-public-key"]);
  });

  it("enable: subscribes, POSTs the subscription to /api/subscribe, and confirms the Worker path", async () => {
    const { calls, bodies } = stubEnv();
    const root = fakeEl("div");
    await renderPairing(root as unknown as HTMLElement);
    const enable = findButton(root, "Enable notifications");
    await enable._click();
    expect(calls).toEqual(["/api/vapid-public-key", "/api/subscribe"]);
    expect(bodies).toEqual([JSON.stringify(SUB)]);
    const status = findByClass(root, "cf-pairing-status");
    expect(status.textContent).toBe("This device will receive alerts. Send a test to be sure.");
  });

  it("shows the Worker rejection reason when /api/subscribe fails", async () => {
    const { calls, fetchMock } = stubEnv();
    fetchMock.mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input);
      calls.push(url);
      if (url.endsWith("/api/vapid-public-key")) return new Response(JSON.stringify({ key: "TESTVAPIDKEY" }), { status: 200 });
      if (url.endsWith("/api/subscribe")) return new Response(JSON.stringify({ error: "subscription limit reached" }), { status: 409 });
      return new Response("not found", { status: 404 });
    });
    const root = fakeEl("div");
    await renderPairing(root as unknown as HTMLElement);
    await findButton(root, "Enable notifications")._click();
    const status = findByClass(root, "cf-pairing-status");
    expect(status.textContent).toContain("HTTP 409");
  });

  it("Send test alert: POSTs /api/test-fire and reports sent N", async () => {
    const { calls } = stubEnv();
    const root = fakeEl("div");
    await renderPairing(root as unknown as HTMLElement);
    await findButton(root, "Send test alert")._click();
    expect(calls).toContain("/api/test-fire");
    const status = findByClass(root, "cf-pairing-status");
    expect(status.textContent).toBe("Test sent — check your notification shade.");
  });
});
