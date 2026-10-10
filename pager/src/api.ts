// One fetch module for every Ops data view (FEAT-052): in-memory +
// sessionStorage copies of the last good response per endpoint, instant
// cached render then background revalidate, skeleton loaders in the views,
// Access-expiry handling, and a localhost fixtures fallback so `bun run dev`
// shows the UI without the Ops Worker.

import { makeFixtures } from "./fixtures";

/** Server-side cache metadata filled by the Ops Worker's cached() wrapper. */
export interface CacheInfo {
  ageSec: number;
  stale: boolean;
}

// ---- /api/summary v2 contract (docs/planning-mode/2026-10-09-ops-billing-ui-caching.md) ----

export interface UsageRow {
  name: string;
  used: number;
  limit: number;
  pct: number;
  projectedPct: number | null;
}

export interface D1Row {
  name: string;
  databaseId?: string;
  rowsRead: number;
  rowsWritten: number;
  readQueries: number;
  rowsPerQuery: number;
}

export interface ZoneRow {
  zone: string;
  total: number;
  uncached: number;
  missPct: number | null;
  byStatus?: Record<string, number>;
}

export interface WorkerRow {
  script: string;
  requests: number;
  errors: number;
  errorPct: number;
  cpuP50Ms: number | null;
  cpuP99Ms: number | null;
}

export interface Summary {
  generatedAt: string;
  windowHours: number;
  usage: UsageRow[];
  d1: D1Row[];
  zones: ZoneRow[];
  workers: WorkerRow[];
  errors: string[];
  cache?: CacheInfo;
}

// ---- /api/billing contract (agent B produces, C consumes) ----

export interface BillingPeriod {
  start: string;
  end: string;
  day: number;
  days: number;
  source: "subscription" | "anchor" | "calendar";
}

export interface TopConsumer {
  name: string;
  project: string;
  used: number;
  share: number; // 0..1
}

export interface ProductUsage {
  id: string;
  product: string;
  metric: string;
  unit: string;
  included: number;
  used: number;
  projected: number;
  unitPriceUsd: number;
  priceUnit: number;
  projectedOverageUsd: number;
  topConsumers: TopConsumer[];
}

export interface ProjectCost {
  project: string;
  projectedOverageUsd: number;
  drivers: string[]; // ProductUsage ids
}

export interface Billing {
  generatedAt: string;
  period: BillingPeriod;
  products: ProductUsage[];
  totalProjectedOverageUsd: number;
  projects: ProjectCost[];
  pricing: { verifiedOn: string; sources: { product: string; url: string }[] };
  errors: string[];
  cache?: CacheInfo;
}

// ---- Fetch client ----

export type DataSource = "network" | "memory" | "session" | "fixture";

export interface FetchResult<T> {
  data: T;
  source: DataSource;
  /** Age of the rendered data in seconds (local copy age, or 0 when fresh). */
  ageSec: number;
  /** True when served from fixtures (localhost demo mode). */
  demo: boolean;
}

export interface ApiClientOptions {
  fetchFn?: typeof fetch;
  storage?: Pick<Storage, "getItem" | "setItem" | "removeItem"> | null;
  now?: () => number;
  /** When true, a failed /api/* fetch falls back to fixtures (demo mode). */
  allowFixtures?: boolean;
}

// Data older than FRESH_MS is revalidated on route entry; younger copies are
// reused verbatim so switching tabs never refetches (task: no refetch when
// data is under 60s old).
const FRESH_MS = 60_000;

const ssKey = (endpoint: string): string => `pager-cache:${endpoint}`;

export class ApiClient {
  private readonly fetchFn: typeof fetch;
  private readonly storage: Pick<Storage, "getItem" | "setItem" | "removeItem"> | null;
  private readonly now: () => number;
  private readonly allowFixtures: boolean;
  private readonly memory = new Map<string, { data: unknown; fetchedAt: number }>();

  constructor(opts: ApiClientOptions = {}) {
    // Never store the bare global: calling it as this.fetchFn(...) binds
    // `this` to the client and browsers throw "Illegal invocation" before any
    // request leaves (production showed "Could not load" on every view).
    this.fetchFn = opts.fetchFn ?? ((input, init) => fetch(input, init));
    this.storage = opts.storage === undefined ? defaultStorage() : opts.storage;
    this.now = opts.now ?? Date.now;
    this.allowFixtures = opts.allowFixtures ?? isLocalhost();
  }

  /**
   * Resolve the last good response for `endpoint` (memory, then sessionStorage),
   * revalidate against the network when older than 60s, and fall back to
   * fixtures when nothing is cached and the network fails on localhost.
   * `refresh` bypasses all caches and appends ?refresh (the Ops Worker's
   * refresh-storm guard is per 60s per key).
   */
  async fetchJson<T>(endpoint: string, opts: { refresh?: boolean; onRevalidate?: (data: T) => void } = {}): Promise<FetchResult<T>> {
    const key = endpoint.split("?")[0];
    if (opts.refresh) return this.fromNetwork<T>(appendRefresh(endpoint), opts.onRevalidate);

    const local = this.lookup(key);
    if (local) {
      const ageSec = Math.floor((this.now() - local.fetchedAt) / 1000);
      if (ageSec < FRESH_MS / 1000) {
        return { data: local.data as T, source: local.source, ageSec, demo: false };
      }
      // Stale local copy: render cached instantly, then revalidate in the background.
      void this.fromNetwork<T>(endpoint, opts.onRevalidate).catch(() => undefined);
      return { data: local.data as T, source: local.source, ageSec, demo: false };
    }
    return this.fromNetwork<T>(endpoint, opts.onRevalidate);
  }
  // Network fetch with Access-expiry handling, cache store, and the
  // localhost fixtures fallback. On success the fresh copy is stored under
  // the bare endpoint key (query string dropped).
  private async fromNetwork<T>(endpoint: string, onRevalidate?: (data: T) => void): Promise<FetchResult<T>> {
    try {
      const res = await this.fetchFn(`/${endpoint}`, { credentials: "same-origin", redirect: "manual" });
      // An expired Access session answers with a redirect to the login page.
      if (res.type === "opaqueredirect" || res.status === 403) throw new LoginExpiredError();
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = (await res.json()) as T;
      this.store(keyOf(endpoint), data);
      onRevalidate?.(data);
      return { data, source: "network", ageSec: 0, demo: false };
    } catch (err) {
      if (err instanceof LoginExpiredError) throw err;
      // Demo mode: localhost dev without the Ops Worker falls back to
      // fixtures so the UI is explorable; production surfaces the error.
      // Only the endpoints with fixtures are answered — anything else
      // throws, exactly like a failed network fetch, so views paint their
      // own error states instead of decoding the wrong payload shape.
      if (this.allowFixtures) {
        const fixtures = makeFixtures(new Date(this.now()));
        const dataset = endpoint.startsWith("api/billing")
          ? fixtures.billing
          : endpoint.startsWith("api/rules")
            ? fixtures.rules
            : endpoint.startsWith("api/summary")
              ? fixtures.summary
              : null;
        if (dataset === null) throw err;
        const data = dataset as unknown as T;
        return { data, source: "fixture", ageSec: 0, demo: true };
      }
      throw err;
    }
  }

  /**
   * Update the cached copy for `endpoint` after a client-side mutation
   * (BUG-057 defect 1: PUT /api/rules must refresh the cache so the view
   * renders the saved rules immediately and a later Save cannot PUT the
   * stale pre-save rules back). Writes through memory + sessionStorage like
   * a network response would.
   */
  updateCache<T>(endpoint: string, data: T): void {
    this.store(keyOf(endpoint), data);
  }

  // lookup finds the last good copy: memory first, then sessionStorage.
  // sessionStorage keeps a reload's render instant; memory covers route
  // switches within the session.
  private lookup(key: string): { data: unknown; fetchedAt: number; source: DataSource } | null {
    const hit = this.memory.get(key);
    if (hit) return { ...hit, source: "memory" };
    if (!this.storage) return null;
    try {
      const raw = this.storage.getItem(ssKey(key));
      if (raw === null) return null;
      const parsed = JSON.parse(raw) as { data: unknown; fetchedAt: number };
      return { ...parsed, source: "session" };
    } catch {
      return null;
    }
  }

  // store writes the fresh copy into memory and sessionStorage. A storage
  // failure (quota, privacy mode) is non-fatal — memory still holds it.
  private store(key: string, data: unknown): void {
    const fetchedAt = this.now();
    this.memory.set(key, { data, fetchedAt });
    if (!this.storage) return;
    try {
      this.storage.setItem(ssKey(key), JSON.stringify({ data, fetchedAt }));
    } catch {
      // quota or serialization failure: memory copy remains
    }
  }
}

/** Thrown when Cloudflare Access has expired the session (redirect or 403). */
export class LoginExpiredError extends Error {
  constructor() {
    super("your login expired — close and reopen the app to sign in again");
    this.name = "LoginExpiredError";
  }
}

/**
 * Honest "Updated X ago" age (BUG-057 defect 2): the client-side copy age
 * (res.ageSec — time since this client fetched/stored it) plus the server
 * cache age (cache.ageSec — how long the Ops Worker had cached the
 * response). Without the server half, hours-old sessionStorage data reads
 * as fresh.
 */
export function totalAgeSec(ageSec: number, cache?: CacheInfo): number {
  return ageSec + (cache?.ageSec ?? 0);
}

/** "api/summary?refresh" — the refresh form of an endpoint. */
export function appendRefresh(endpoint: string): string {
  return endpoint.includes("?") ? `${endpoint}&refresh` : `${endpoint}?refresh`;
}

function keyOf(endpoint: string): string {
  return endpoint.split("?")[0];
}

// defaultStorage picks sessionStorage when reachable (vitest's node env has
// none; the guard keeps construction from throwing).
function defaultStorage(): Pick<Storage, "getItem" | "setItem" | "removeItem"> | null {
  try {
    return typeof sessionStorage === "undefined" ? null : sessionStorage;
  } catch {
    return null;
  }
}

// isLocalhost gates demo mode to dev machines — never to a deployed origin.
export function isLocalhost(): boolean {
  try {
    return typeof location !== "undefined" && /^(localhost|127\.0\.0\.1|\[::1\])$/.test(location.hostname);
  } catch {
    return false;
  }
}

/** Shared client the views use; tests build their own ApiClient instances. */
export const api = new ApiClient();
