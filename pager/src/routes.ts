// Hash route parser for the Ops shell (FEAT-052). Pure — tests hit it
// directly.

export const ROUTES = ["overview", "billing", "ai", "projects", "domains", "workers", "d1", "durable-objects", "zones", "alerts", "rules", "pairing", "guides"] as const;

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

/** Entity drill-in hrefs (profiles program, ROAD-p7QJ0RX): D1 databases,
 *  Durable Objects namespaces and zones follow the worker contract. */
export function d1Href(name: string): string {
  return `#/d1/${encodeURIComponent(name)}`;
}

export function doHref(name: string): string {
  return `#/do/${encodeURIComponent(name)}`;
}

export function zoneHref(name: string): string {
  return `#/zone/${encodeURIComponent(name)}`;
}

/** "#/guides/<id>" — a guides article (FEAT-pDQEFQV). Ids are the static
 *  ids in guides.ts. */
export function guidesHref(id: string): string {
  return `#/guides/${encodeURIComponent(id)}`;
}

/** The decoded guide id when `hash` targets a guides article
 *  ("#/guides/<id>"), else null. "#/guides/" (empty id) and the
 *  "#/guides" section hash are not article targets. */
export function parseGuidesHash(hash: string): string | null {
  const m = hash.match(/^#\/guides\/(.+)$/);
  if (!m) return null;
  try {
    return decodeURIComponent(m[1]);
  } catch {
    // Malformed escape sequence: treat the raw text as the id.
    return m[1];
  }
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

/** Entity drill-in parsers — the worker contract, once per entity type. */
export function parseD1Hash(hash: string): string | null {
  const m = hash.match(/^#\/d1\/(.+)$/);
  if (!m) return null;
  try {
    return decodeURIComponent(m[1]);
  } catch {
    return m[1];
  }
}

export function parseDoHash(hash: string): string | null {
  const m = hash.match(/^#\/do\/(.+)$/);
  if (!m) return null;
  try {
    return decodeURIComponent(m[1]);
  } catch {
    return m[1];
  }
}

export function parseZoneHash(hash: string): string | null {
  const m = hash.match(/^#\/zone\/(.+)$/);
  if (!m) return null;
  try {
    return decodeURIComponent(m[1]);
  } catch {
    return m[1];
  }
}
