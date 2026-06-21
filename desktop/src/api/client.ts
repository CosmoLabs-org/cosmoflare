// REST client for the cosmoflare daemon. Wraps fetch with the bearer-token
// auth the daemon requires (see internal/server authMiddleware). The endpoint
// {url, token} is resolved from the Rust `daemon_endpoint` command and injected
// so tests can pass a fake — no live daemon or Tauri runtime required.

export interface DaemonEndpoint {
  url: string;
  token: string;
}

export class ApiError extends Error {
  constructor(
    public path: string,
    public status: number,
    public body: string
  ) {
    super(`${path}: HTTP ${status}`);
    this.name = "ApiError";
  }
}

export class ApiClient {
  constructor(private endpoint: DaemonEndpoint) {}

  /** GET a JSON path, with optional query params (omitted values skipped). */
  async get<T>(path: string, params?: Record<string, string>): Promise<T> {
    const url = new URL(this.endpoint.url + path);
    if (params) {
      for (const [k, v] of Object.entries(params)) {
        if (v) url.searchParams.set(k, v);
      }
    }
    const resp = await fetch(url, {
      headers: { Authorization: `Bearer ${this.endpoint.token}` },
    });
    if (!resp.ok) {
      throw new ApiError(path, resp.status, await resp.text().catch(() => ""));
    }
    return resp.json() as Promise<T>;
  }
}

// Convenience endpoints keyed to the daemon's REST surface.
export interface Profile {
  name: string;
}

/**
 * Resolve the daemon endpoint from the Rust shell. Polls because the daemon
 * takes a moment to start; `daemon_endpoint` returns null until the handshake
 * completes. Returns null (not throws) when not running under Tauri (tests/dev
 * browser), so callers can degrade gracefully.
 */
export async function resolveEndpoint(): Promise<DaemonEndpoint | null> {
  try {
    const { invoke } = await import("@tauri-apps/api/core");
    const ep = await invoke<DaemonEndpoint | null>("daemon_endpoint");
    return ep ?? null;
  } catch {
    return null;
  }
}
