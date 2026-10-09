// Hash route parser for the Ops shell (FEAT-052). Pure — tests hit it
// directly.

export const ROUTES = ["overview", "billing", "workers", "d1", "zones", "alerts", "pairing"] as const;

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
