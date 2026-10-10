// AI Gateway collector (BR-02, docs/brainstorming/2026-10-10-ai-gateway-monitoring.md):
// per-gateway × model × provider usage over 24h (+ MTD gateway totals) from
// aiGatewayRequestsAdaptiveGroups — the ONLY documented dataset (verified
// live 2026-10-10; dims model/provider/gateway/datetimeMinute, metric count).
// tokens/cost fields are NOT documented: a field probe decides once per day
// (KV-cached, bounding the undocumented GraphQL rate limits) whether the
// query may carry them; absent fields degrade the payload to counts with an
// explicit note — never invented numbers. Every dollar figure the API calls
// an estimate, so costBasis is permanently "gateway-estimate" (real-numbers
// rule). Push note: mapping gateways to projects is BR-04 (bootstrap) —
// this collector exposes names; unmapped rendering marks itself.

import { fetchRetry } from "./retry";

const API = "https://api.cloudflare.com/client/v4";
const GRAPHQL = `${API}/graphql`;

/** KV key caching the once-per-day undocumented-field probe result. */
export const PROBE_KEY = "ai-field-probe";
const PROBE_TTL_MS = 24 * 60 * 60 * 1000;

export interface AIModelRow {
  model: string;
  provider: string;
  requests: number;
  tokensIn?: number;
  tokensOut?: number;
  costUsd?: number;
}

export interface AIGatewayRow {
  id: string;
  name: string;
  requests24h: number;
  requestsMtd: number;
  models: AIModelRow[];
}

export interface AIGatewaysPayload {
  generatedAt: string;
  gateways: AIGatewayRow[];
  costBasis: "gateway-estimate";
  notes: string[];
  errors: string[];
}

interface KVLike {
  get(key: string): Promise<string | null>;
  put(key: string, value: string): Promise<void>;
}

interface ProbeState {
  tokens: boolean;
  cost: boolean;
  checkedAt: string;
}

interface GroupRow {
  count: number;
  sum?: { tokensIn?: number; tokensOut?: number; cost?: number };
  dimensions: { gateway: string; model: string; provider: string };
}

async function gql(token: string, query: string, variables: Record<string, unknown>): Promise<{ data?: { viewer?: { accounts?: { g?: GroupRow[] }[] } }; errors?: { message: string }[] }> {
  const res = await fetchRetry(GRAPHQL, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    body: JSON.stringify({ query, variables }),
  });
  return (await res.json()) as { data?: { viewer?: { accounts?: { g?: GroupRow[] }[] } }; errors?: { message: string }[] };
}

function analyticsQuery(withFields: boolean): string {
  const sums = withFields ? "count sum{tokensIn tokensOut cost}" : "count";
  return `query($a:String!,$s:Time!,$e:Time!){viewer{accounts(filter:{accountTag:$a}){g: aiGatewayRequestsAdaptiveGroups(limit:10000,filter:{datetime_geq:$s,datetime_leq:$e}){${sums} dimensions{gateway model provider}}}}}`;
}

function rowsOf(body: { data?: { viewer?: { accounts?: { g?: GroupRow[] }[] } } }): GroupRow[] {
  return body.data?.viewer?.accounts?.[0]?.g ?? [];
}

export async function collectAIGateways(accountId: string, token: string, now: Date, kv?: KVLike): Promise<AIGatewaysPayload> {
  const out: AIGatewaysPayload = { generatedAt: now.toISOString(), gateways: [], costBasis: "gateway-estimate", notes: [], errors: [] };

  // 1. Gateway list (names are the operator's project convention — BR-04).
  let gateways: { id: string; name?: string }[] = [];
  try {
    const res = await fetchRetry(`${API}/accounts/${accountId}/ai-gateway/gateways?per_page=50`, { headers: { Authorization: `Bearer ${token}` } });
    const body = (await res.json()) as { success?: boolean; result?: { id: string; name?: string }[] };
    if (!res.ok || body.success !== true) throw new Error(`gateway list HTTP ${res.status}`);
    gateways = body.result ?? [];
  } catch (err) {
    out.errors.push(`gateway list unavailable: ${err instanceof Error ? err.message : String(err)}`);
    return out;
  }
  const names = new Map(gateways.map((g) => [g.id, g.name || g.id]));

  // 2. Field probe: decide once per day whether the undocumented sum fields
  //    exist on the dataset (KV-cached; a GraphQL field error means no).
  let probe: ProbeState | null = null;
  if (kv) {
    try {
      const raw = await kv.get(PROBE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw) as ProbeState;
        if (now.getTime() - Date.parse(parsed.checkedAt) < PROBE_TTL_MS) probe = parsed;
      }
    } catch {
      probe = null; // unreadable cache: re-probe
    }
  }
  if (probe === null) {
    try {
      const body = await gql(token, analyticsQuery(true), { a: accountId, s: new Date(now.getTime() - 60_000).toISOString(), e: now.toISOString() });
      probe = { tokens: !body.errors, cost: !body.errors, checkedAt: now.toISOString() };
    } catch {
      probe = { tokens: false, cost: false, checkedAt: now.toISOString() };
    }
    if (kv) {
      try {
        await kv.put(PROBE_KEY, JSON.stringify(probe));
      } catch {
        // probe persistence is best-effort; the in-run decision stands
      }
    }
  }
  if (!probe.tokens) {
    out.notes.push(`token/cost fields unavailable on the analytics dataset — counts only (probe ${probe.checkedAt.slice(0, 10)})`);
  }

  // 3. Analytics: 24h detail + MTD gateway totals.
  const vars = { a: accountId };
  const [day, mtd] = await Promise.all([
    gql(token, analyticsQuery(probe.tokens), { ...vars, s: new Date(now.getTime() - 24 * 3600_000).toISOString(), e: now.toISOString() }),
    gql(token, analyticsQuery(false), { ...vars, s: new Date(now.getFullYear(), now.getMonth(), 1).toISOString(), e: now.toISOString() }),
  ]);
  if (day.errors && !day.data) {
    out.errors.push("ai analytics unavailable");
    return out;
  }

  const mtdTotals = new Map<string, number>();
  for (const r of rowsOf(mtd)) mtdTotals.set(r.dimensions.gateway, (mtdTotals.get(r.dimensions.gateway) ?? 0) + r.count);

  const byGateway = new Map<string, AIGatewayRow>();
  for (const r of rowsOf(day)) {
    const id = r.dimensions.gateway;
    let gw = byGateway.get(id);
    if (!gw) {
      gw = { id, name: names.get(id) ?? id, requests24h: 0, requestsMtd: mtdTotals.get(id) ?? 0, models: [] };
      byGateway.set(id, gw);
      out.gateways.push(gw);
    }
    gw.requests24h += r.count;
    gw.models.push({
      model: r.dimensions.model,
      provider: r.dimensions.provider,
      requests: r.count,
      tokensIn: probe.tokens ? r.sum?.tokensIn : undefined,
      tokensOut: probe.tokens ? r.sum?.tokensOut : undefined,
      costUsd: probe.cost ? r.sum?.cost : undefined,
    });
  }
  // Gateways with MTD traffic but none in 24h still surface (zeroed day).
  for (const [id, total] of mtdTotals) {
    if (!byGateway.has(id)) {
      out.gateways.push({ id, name: names.get(id) ?? id, requests24h: 0, requestsMtd: total, models: [] });
    }
  }
  out.gateways.sort((a, b) => b.requests24h - a.requests24h || a.name.localeCompare(b.name));
  for (const gw of out.gateways) gw.models.sort((a, b) => b.requests - a.requests);
  return out;
}
