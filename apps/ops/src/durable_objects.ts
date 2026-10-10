// Durable Objects collector (FEAT-p3KJAC2): namespaces from REST, usage
// from the invocations analytics dataset — per script over the last 24h.
// No storage numbers: no authoritative per-DO storage dataset exists
// (durableObjectsStorageAdaptiveGroups is not a dataset; verified live
// 2026-10-10), and the real-numbers rule forbids inventing one.

import { rest } from "./summary";
import { fetchRetry } from "./retry";

export interface DORow {
  namespaceId: string;
  script: string;
  requests: number;
  errors: number;
  errorPct: number;
}

export interface DurableObjectsPayload {
  generatedAt: string;
  namespaces: number;
  rows: DORow[];
  errors: string[];
}

export function errorPct(requests: number, errors: number): number {
  return requests > 0 ? (errors / requests) * 100 : 0;
}

/** Merge namespace IDs (REST) with per-script usage (GraphQL) into rows.
 *  Scripts without a namespace row keep an empty ID; namespaces without
 *  traffic still appear with zeros so the count is honest. Pure. */
export function mergeDoRows(
  namespaces: { id: string; script: string }[],
  usage: Record<string, { requests: number; errors: number }>,
): DORow[] {
  const nsByScript = new Map<string, string>();
  for (const ns of namespaces) {
    if (!nsByScript.has(ns.script)) nsByScript.set(ns.script, ns.id);
  }
  const scripts = new Set<string>([...nsByScript.keys(), ...Object.keys(usage)]);
  const rows: DORow[] = [];
  for (const script of scripts) {
    const u = usage[script] ?? { requests: 0, errors: 0 };
    rows.push({
      namespaceId: nsByScript.get(script) ?? "",
      script,
      requests: u.requests,
      errors: u.errors,
      errorPct: errorPct(u.requests, u.errors),
    });
  }
  return rows.sort((a, b) => b.requests - a.requests || a.script.localeCompare(b.script));
}

/** Collect DO namespaces + 24h invocations; each source soft-fails into
 *  errors so one permission gap never blanks the view. */
export async function collectDurableObjects(accountId: string, token: string, now = new Date()): Promise<DurableObjectsPayload> {
  const out: DurableObjectsPayload = { generatedAt: now.toISOString(), namespaces: 0, rows: [], errors: [] };

  const namespaces = await rest<{ id: string; script: string }>(token, `/accounts/${accountId}/workers/durable_objects/namespaces`)
    .catch((e: Error) => {
      out.errors.push(`namespaces: ${e.message}`);
      return [] as { id: string; script: string }[];
    });
  out.namespaces = namespaces.length;

  const end = now.toISOString();
  const start = new Date(now.getTime() - 24 * 3600 * 1000).toISOString();
  const query = `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){o: durableObjectsInvocationsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){dimensions{scriptName} sum{requests errors}}}}}`;
  const usage: Record<string, { requests: number; errors: number }> = {};
  try {
    const res = await fetchRetry("https://api.cloudflare.com/client/v4/graphql", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
      body: JSON.stringify({ query, variables: { a: accountId, s: start, e: end } }),
    });
    const body = (await res.json()) as {
      data?: { viewer?: { accounts?: { o?: { dimensions: { scriptName: string }; sum: { requests: number; errors: number } }[] }[] } };
      errors?: { message: string }[];
    };
    const rows = body.data?.viewer?.accounts?.[0]?.o ?? [];
    if (rows.length === 0 && body.errors?.length) {
      out.errors.push(`invocations: ${body.errors[0].message}`);
    }
    for (const r of rows) {
      const s = r.dimensions.scriptName;
      usage[s] = { requests: r.sum.requests, errors: r.sum.errors };
    }
  } catch (e) {
    out.errors.push(`invocations: ${e instanceof Error ? e.message : String(e)}`);
  }

  out.rows = mergeDoRows(namespaces, usage);
  return out;
}
