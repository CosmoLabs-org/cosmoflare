// Hash route parser for the Ops shell (FEAT-052). Pure — tests hit it
// directly.

export const ROUTES = ["overview", "billing", "ai", "projects", "domains", "workers", "d1", "durable-objects", "zones", "alerts", "rules", "pairing"] as const;

export type RouteId = (typeof ROUTES)[number];

/** Unknown or missing hashes land on the overview. */
export function parseHash(hash: string): RouteId {
  const raw = hash.replace(/^#\/?/, "").split("?")[0];
  if ((ROUTES as readonly string[]).includes(raw)) return raw as RouteId;
  return "overview";
}

/** "#/billing" for a route id — the href the drawer and sidebar point at. */
export function hrefFor(id: RouteId): string {
  return `#/${id}`;
}

/** "#/worker/<name>" — a worker profile page (UI-3). Script names may carry
 *  spaces and punctuation, so the name is percent-encoded. */
export function workerHref(name: string): string {
  return `#/worker/${encodeURIComponent(name)}`;
}

/** The decoded worker name when `hash` targets a profile page
 *  ("#/worker/<name>"), else null. "#/worker/" (empty name) and the
 *  "#/workers" section hash are not profile targets. */
export function parseWorkerHash(hash: string): string | null {
  const m = hash.match(/^#\/worker\/(.+)$/);
  if (!m) return null;
  try {
    return decodeURIComponent(m[1]);
  } catch {
    // Malformed escape sequence: treat the raw text as the name.
    return m[1];
  }
}
