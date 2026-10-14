// Durable Objects namespace profile collector (profiles program instance 3):
// one DO namespace's 24h analytics — requests, errors, error rate — wrapped
// from collectDurableObjects' proven per-script rows. DO namespaces have no
// cheap per-namespace REST metadata beyond analytics, so the profile is
// analytics-driven by design. Follows the house rule every collector obeys:
// a failure becomes a collectionErrors[] line and an omitted value, never a
// failed response and never an invented number.

import { collectDurableObjects } from "./durable_objects";

export interface DoProfilePayload {
  generatedAt: string;
  /** The requested namespace script name (percent-decoded by the caller). */
  name: string;
  /** True when the 24h analytics join produced a row for this script. */
  found: boolean;
  requests: number;
  /** Error count over the 24h window (the DORow field, not failure lines). */
  errors: number;
  errorPct: number;
  /** The REST namespace ID when the namespaces list carried one, else null. */
  namespaceId: string | null;
  /** One line per failed source — data absent, never invented. Named
   *  collectionErrors because `errors` is the analytics error count here. */
  collectionErrors: string[];
}

export async function collectDoProfile(accountId: string, token: string, name: string, now = new Date()): Promise<DoProfilePayload> {
  // One shared source of truth: the same namespaces REST list + 24h
  // invocations GraphQL query the DO view already uses, cached 5 min at the
  // route. No second query shape to drift.
  const payload = await collectDurableObjects(accountId, token, now);
  const row = payload.rows.find((r) => r.script === name);

  return {
    generatedAt: payload.generatedAt,
    name,
    found: row !== undefined,
    requests: row?.requests ?? 0,
    errors: row?.errors ?? 0,
    errorPct: row?.errorPct ?? 0,
    namespaceId: row && row.namespaceId !== "" ? row.namespaceId : null,
    collectionErrors: payload.errors,
  };
}
