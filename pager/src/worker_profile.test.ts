import { describe, expect, it, vi } from "vitest";
import { renderWorkerProfile, type SummaryFetcher } from "./worker_profile";
import type { FetchResult, Summary, WorkerRow } from "./api";
import { LoginExpiredError } from "./api";

// The views render through el() → document.createElement; stub it with the
// fake element factory for the whole file (no DOM library in this env).
// profileFakeEl hoists (function declaration), so this is safe at module top.
vi.stubGlobal("document", { createElement: (tag: string) => profileFakeEl(tag) });

// Node-environment DOM stubs (the views.test.ts FakeNode pattern): no DOM
// library is installed here, so renderWorkerProfile is exercised against
// minimal fake elements that record text/children/classes. The summary fetch
// is injected (the `fetcher` param) rather than stubbing the shared ApiClient.

type Handler = () => unknown;
interface FakeNode {
  tagName: string;
  className: string;
  textContent: string;
  href: string;
  target: string;
  rel: string;
  children: FakeNode[];
  classList: { add: (...c: string[]) => void; remove: (...c: string[]) => void; contains: (c: string) => boolean };
  setAttribute: (name: string, value: string) => void;
  append: (...nodes: FakeNode[]) => void;
  replaceChildren: (...nodes: FakeNode[]) => void;
  addEventListener: (type: string, fn: Handler) => void;
  _click: () => Promise<void>;
}

function profileFakeEl(tag: string): FakeNode {
  const handlers: Record<string, Handler[]> = {};
  const classes = new Set<string>();
  const node: FakeNode = {
    tagName: tag.toUpperCase(),
    className: "",
    textContent: "",
    href: "",
    target: "",
    rel: "",
    children: [],
    classList: {
      add: (...c) => c.forEach((x) => classes.add(x)),
      remove: (...c) => c.forEach((x) => classes.delete(x)),
      contains: (c) => classes.has(c),
    },
    setAttribute: (name, value) => {
      if (name === "class") node.className = value;
    },
    append(...nodes) {
      node.children.push(...nodes);
    },
    replaceChildren(...nodes) {
      node.children = [...nodes];
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

function findByText(root: FakeNode, text: string): FakeNode {
  const stack = [...root.children];
  while (stack.length > 0) {
    const node = stack.shift()!;
    if (node.textContent === text) return node;
    stack.unshift(...node.children);
  }
  throw new Error(`element not found with text: ${text}`);
}

function findByTag(root: FakeNode, tag: string): FakeNode {
  const stack = [...root.children];
  while (stack.length > 0) {
    const node = stack.shift()!;
    if (node.tagName === tag.toUpperCase()) return node;
    stack.unshift(...node.children);
  }
  throw new Error(`element not found: <${tag}>`);
}

const ROW: WorkerRow = {
  script: "api-gateway",
  requests: 52_332_502,
  errors: 1234,
  errorPct: 5.5,
  cpuP50Ms: 3.42,
  cpuP99Ms: null,
};

function fetcherReturning(summary: Summary, opts: { throwErr?: Error } = {}): SummaryFetcher {
  return async () => {
    if (opts.throwErr) throw opts.throwErr;
    const res: FetchResult<Summary> = { data: summary, source: "network", ageSec: 0, demo: false };
    return res;
  };
}

const SUMMARY = (workers: WorkerRow[]): Summary => ({
  generatedAt: "2026-10-10T00:00:00Z",
  windowHours: 24,
  usage: [],
  d1: [],
  zones: [],
  workers,
  errors: [],
});

describe("renderWorkerProfile", () => {
  it("renders the 24h stats for a known worker", async () => {
    const root = profileFakeEl("div");
    await renderWorkerProfile(root as unknown as HTMLElement, "api-gateway", {
      fetcher: fetcherReturning(SUMMARY([ROW])),
    });
    const heading = findByTag(root, "H2");
    expect(heading.textContent).toBe("api-gateway");
    expect(findByText(root, "52.3M")).toBeDefined(); // Requests
    expect(findByText(root, "1.2k")).toBeDefined(); // Errors
    const rate = findByText(root, "5.50%"); // Error rate, 2 decimals
    expect(rate.classList.contains("cf-level-critical")).toBe(true); // 5.5% ≥ 5
    expect(findByText(root, "3.4ms")).toBeDefined(); // CPU p50
    expect(findByText(root, "—")).toBeDefined(); // CPU p99 (null)
    expect(findByText(root, "24h window · from Workers Analytics")).toBeDefined();
    const link = findByText(root, "Open in Cloudflare →");
    expect(link.href).toBe("https://dash.cloudflare.com/?to=/:account/workers/services/view/api-gateway");
    expect(link.target).toBe("_blank");
    expect(link.rel).toBe("noreferrer");
  });

  it("shows a quiet empty state for an unknown worker — never an error", async () => {
    const root = profileFakeEl("div");
    await renderWorkerProfile(root as unknown as HTMLElement, "ghost", {
      fetcher: fetcherReturning(SUMMARY([ROW])),
    });
    expect(findByText(root, "Worker not found")).toBeDefined();
    expect(findByText(root, "No 24h analytics row for ghost.")).toBeDefined();
  });

  it("shows the login-expired message on LoginExpiredError", async () => {
    const root = profileFakeEl("div");
    await renderWorkerProfile(root as unknown as HTMLElement, "api-gateway", {
      fetcher: fetcherReturning(SUMMARY([]), { throwErr: new LoginExpiredError() }),
    });
    expect(findByText(root, "your login expired — close and reopen the app to sign in again")).toBeDefined();
  });

  it("offers a Retry after a first-load failure and recovers on click", async () => {
    const root = profileFakeEl("div");
    let fail = true;
    const flaky: SummaryFetcher = async () => {
      if (fail) throw new Error("HTTP 503");
      return { data: SUMMARY([ROW]), source: "network", ageSec: 0, demo: false };
    };
    await renderWorkerProfile(root as unknown as HTMLElement, "api-gateway", { fetcher: flaky });
    expect(findByText(root, "HTTP 503")).toBeDefined();
    const retry = findByText(root, "Retry");
    fail = false;
    await retry._click();
    // The click handler is fire-and-forget (void renderWorkerProfile) — let
    // the re-render's fetch resolve before asserting.
    await new Promise((r) => setTimeout(r, 0));
    // The retry repaints the profile with the good data.
    expect(findByText(root, "52.3M")).toBeDefined();
  });
});
