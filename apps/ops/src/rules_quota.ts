// Rule-quota collector (ROAD-p9X66J9): per-zone rule inventory against the
// plan's free allowances — real API numbers only. Five families; Page Rules
// come from the legacy REST endpoint, the other four from the Ruleset Engine
// entrypoints (needs Zone Rulesets Read on the token — every rulesets family
// degrades to authorized:false on 403, the Page Rules row still renders).
// Budget: one cached zone-list lookup (shared "cron:zones" key) + up to 5
// GETs per zone, KV-cached 5 min per zone — never per request.

import { fetchRetry } from "./retry";
import type { OpsEnv } from "./scheduled";

const API = "https://api.cloudflare.com/client/v4";

const ZONES_CACHE_KEY = "cron:zones";
const ZONES_TTL_MS = 60 * 60 * 1000;
const ZONES_KV_TTL_SEC = 86400;
const QUOTA_KV_PREFIX = "ops:rules-quota:";
const QUOTA_KV_TTL_SEC = 300;

/** Published free-plan allowances per zone. Allowances are plan limits (not
 *  account usage), so the "real numbers only" rule reads them from the
 *  published plans page; bump verified_on when reconfirming.
 *  https://developers.cloudflare.com/rules/ (last verified 2026-10-14) */
const ALLOWANCE: Record<string, number> = {
  "page-rules": 3,
  "redirect-rules": 10,
  "waf-custom-rules": 5,
  "cache-rules": 10,
  "config-rules": 10,
};

const LABEL: Record<string, string> = {
  "page-rules": "Page Rules",
  "redirect-rules": "Redirect Rules",
  "waf-custom-rules": "WAF Custom Rules",
  "cache-rules": "Cache Rules",
  "config-rules": "Configuration Rules",
};

/** Ruleset-Engine phase feeding each rulesets family. */
const PHASE: Record<string, string> = {
  "redirect-rules": "http_request_dynamic_redirect",
  "waf-custom-rules": "http_request_firewall_custom",
  "cache-rules": "http_request_cache_settings",
  "config-rules": "http_request_config",
};

export interface QuotaRuleRow {
  name: string;
  detail: string;
}

export interface RuleFamilyQuota {
  family: string;
  label: string;
  used: number;
  allowance: number;
  /** false when the token lacks the family's read scope — used/allowance
   *  still render (0/n) but the UI shows an "unauthorized" note instead of
   *  a false "no rules". */
  authorized: boolean;
  rules: QuotaRuleRow[];
}

export interface RulesQuotaPayload {
  zone: string;
  generatedAt: string;
  families: RuleFamilyQuota[];
  errors: string[];
}

interface CachedZones {
  at: number;
  value: { id: string; name: string }[];
}

interface CachedQuota {
  at: number;
  value: RulesQuotaPayload;
}

async function zonesFromCacheOrFetch(env: OpsEnv): Promise<{ id: string; name: string }[]> {
  try {
    const cached = await env.OPS_KV.get<CachedZones>(ZONES_CACHE_KEY, "json");
    if (cached && typeof cached.at === "number" && Date.now() - cached.at < ZONES_TTL_MS) return cached.value;
  } catch {
    // KV read failure falls through to a live fetch.
  }
  const res = await fetchRetry(`${API}/zones?account.id=${env.CF_ACCOUNT_ID}`, {
    headers: { Authorization: `Bearer ${env.CF_API_TOKEN}` },
  });
  const body = (await res.json()) as { success?: boolean; result?: { id: string; name: string }[] };
  if (!res.ok || body.success !== true) throw new Error(`zone list: HTTP ${res.status}`);
  const zones = body.result ?? [];
  try {
    await env.OPS_KV.put(ZONES_CACHE_KEY, JSON.stringify({ at: Date.now(), value: zones }), { expirationTtl: ZONES_KV_TTL_SEC });
  } catch {
    // Cache-write failure is non-fatal.
  }
  return zones;
}

async function restGet<T>(token: string, path: string): Promise<T> {
  const res = await fetchRetry(`${API}${path}`, { headers: { Authorization: `Bearer ${token}` } });
  const body = (await res.json()) as { success?: boolean; result?: T; errors?: { message: string }[] };
  if (!res.ok || body.success !== true) {
    const err = new Error(body.errors?.[0]?.message ?? `HTTP ${res.status}`);
    (err as Error & { status?: number }).status = res.status;
    throw err;
  }
  return body.result as T;
}

/** Page Rules: [{id, status, targets:[{constraint:{value}}], actions}] —
 *  the forwarding target names the rule for the inventory list. */
async function collectPageRules(token: string, zoneID: string): Promise<RuleFamilyQuota> {
  const family = "page-rules";
  try {
    const rules = await restGet<
      { status?: string; targets?: { constraint?: { value?: string } }[]; actions?: { id?: string; value?: { url?: string; status_code?: number } }[] }[]
    >(token, `/zones/${zoneID}/pagerules`);
    const rows: QuotaRuleRow[] = (rules ?? []).map((r) => {
      const target = r.targets?.[0]?.constraint?.value ?? "";
      const fwd = r.actions?.find((a) => a.id === "forwarding_url")?.value;
      return {
        name: target,
        detail: fwd ? `forward ${target} to ${fwd.url ?? ""} (${fwd.status_code ?? 301})` : (r.actions ?? []).map((a) => a.id).join(", "),
      };
    });
    return { family, label: LABEL[family], used: rows.length, allowance: ALLOWANCE[family], authorized: true, rules: rows };
  } catch {
    // Page Rules are legacy and always readable with the current token; a
    // failure still degrades rather than failing the whole payload.
    return { family, label: LABEL[family], used: 0, allowance: ALLOWANCE[family], authorized: false, rules: [] };
  }
}

/** Rulesets families: the phase entrypoint's rules array is the inventory. */
async function collectRulesetsFamily(token: string, zoneID: string, family: string): Promise<RuleFamilyQuota> {
  try {
    const rs = await restGet<{ rules?: { description?: string; expression?: string; action?: string }[] }>(
      token,
      `/zones/${zoneID}/rulesets/phases/${PHASE[family]}/entrypoint`,
    );
    const rows: QuotaRuleRow[] = (rs.rules ?? []).map((r) => ({
      name: r.description || r.expression?.slice(0, 60) || "(unnamed rule)",
      detail: `${r.action ?? "execute"} · ${r.expression?.slice(0, 90) ?? ""}`,
    }));
    return { family, label: LABEL[family], used: rows.length, allowance: ALLOWANCE[family], authorized: true, rules: rows };
  } catch {
    // 403 (missing Zone Rulesets Read) or an empty phase: degrade, never fail.
    return { family, label: LABEL[family], used: 0, allowance: ALLOWANCE[family], authorized: false, rules: [] };
  }
}

export async function collectRulesQuota(env: OpsEnv, zoneName: string, now = new Date()): Promise<RulesQuotaPayload> {
  const cacheKey = QUOTA_KV_PREFIX + zoneName;
  try {
    const cached = await env.OPS_KV.get<CachedQuota>(cacheKey, "json");
    if (cached && typeof cached.at === "number" && Date.now() - cached.at < QUOTA_KV_TTL_SEC * 1000) return cached.value;
  } catch {
    // KV read failure falls through to live collection.
  }

  const out: RulesQuotaPayload = { zone: zoneName, generatedAt: now.toISOString(), families: [], errors: [] };
  try {
    const zones = await zonesFromCacheOrFetch(env);
    const zone = zones.find((z) => z.name === zoneName);
    if (!zone) throw new Error(`zone ${zoneName} not found`);

    out.families.push(await collectPageRules(env.CF_API_TOKEN, zone.id));
    for (const family of Object.keys(PHASE)) {
      out.families.push(await collectRulesetsFamily(env.CF_API_TOKEN, zone.id, family));
    }
  } catch (err) {
    out.errors.push(err instanceof Error ? err.message : String(err));
  }

  try {
    await env.OPS_KV.put(cacheKey, JSON.stringify({ at: Date.now(), value: out }), { expirationTtl: QUOTA_KV_TTL_SEC });
  } catch {
    // Cache-write failure is non-fatal.
  }
  return out;
}

/** Test hook: the published allowances, so tests pin the numbers they assert. */
export const RULE_ALLOWANCES = ALLOWANCE;

// --- Overview strip: rotating batch summary (ROAD-p9X66J9) ---
//
// The Overview reads ONE KV row, never a fan-out. Freshness comes from the
// cron: each cycle collects the next batch of zones (their 5-min per-zone
// caches serve when fresh), merges into the store, advances the cursor. A
// 45-zone fleet rotates fully in ~6 five-minute cycles (~30 min).

const SUMMARY_KEY = "ops:rules-quota-summary";
const CURSOR_KEY = "ops:rules-quota-cursor";
/** Zones per cron cycle — the enforced budget: at most this × 5 REST GETs. */
export const QUOTA_BATCH_SIZE = 8;
/** A zone's entry older than this drops from the summary (bounded store). */
const ENTRY_MAX_AGE_MS = 48 * 60 * 60 * 1000;

/** One zone's quota entry in the fleet store (family → counters). */
export interface RulesQuotaZoneEntry {
  at: string;
  families: Record<string, { used: number; allowance: number; authorized: boolean }>;
}

/** The aggregated family rows the Overview strip renders. */
export interface RulesQuotaFamilySummary {
  family: string;
  label: string;
  zonesTracked: number;
  totalUsed: number;
  atCapZones: number;
}

export interface RulesQuotaSummary {
  generatedAt: string;
  zonesTracked: number;
  families: RulesQuotaFamilySummary[];
}

interface SummaryStore {
  at: number;
  zones: Record<string, RulesQuotaZoneEntry>;
}

/** Collect the next batch and merge into the store. Failures per zone are
 *  skipped (that zone keeps its previous entry); the cursor still advances. */
export async function refreshRulesQuotaBatch(env: OpsEnv, batchSize = QUOTA_BATCH_SIZE, now = new Date()): Promise<RulesQuotaSummary> {
  const zones = await zonesFromCacheOrFetch(env);
  if (zones.length === 0) return { generatedAt: now.toISOString(), zonesTracked: 0, families: [] };

  const cursorRaw = (await env.OPS_KV.get<number>(CURSOR_KEY, "json")) ?? 0;
  const cursor = ((cursorRaw % zones.length) + zones.length) % zones.length;

  let store: SummaryStore = { at: Date.now(), zones: {} };
  try {
    const cached = await env.OPS_KV.get<SummaryStore>(SUMMARY_KEY, "json");
    if (cached && typeof cached.at === "number" && cached.zones) store = cached;
  } catch {
    // KV read failure starts a fresh store.
  }

  const n = Math.min(batchSize, zones.length);
  for (let i = 0; i < n; i++) {
    const zone = zones[(cursor + i) % zones.length];
    const payload = await collectRulesQuota(env, zone.name, now);
    if (payload.families.length === 0) continue; // zone vanished or errored
    const entry: RulesQuotaZoneEntry = { at: payload.generatedAt, families: {} };
    for (const f of payload.families) {
      entry.families[f.family] = { used: f.used, allowance: f.allowance, authorized: f.authorized };
    }
    store.zones[zone.name] = entry;
  }

  // Drop stale entries so the store stays bounded by the rotation window.
  for (const [name, entry] of Object.entries(store.zones)) {
    if (now.getTime() - Date.parse(entry.at) > ENTRY_MAX_AGE_MS) delete store.zones[name];
  }
  store.at = Date.now();
  try {
    await env.OPS_KV.put(SUMMARY_KEY, JSON.stringify(store));
    await env.OPS_KV.put(CURSOR_KEY, JSON.stringify((cursor + n) % zones.length));
  } catch {
    // Cache-write failure is non-fatal; the aggregate below still returns.
  }
  return aggregateSummary(store, now);
}

/** Read the store and aggregate — the strip's read path (no fetches). */
export async function readRulesQuotaSummary(env: OpsEnv, now = new Date()): Promise<RulesQuotaSummary> {
  try {
    const cached = await env.OPS_KV.get<SummaryStore>(SUMMARY_KEY, "json");
    if (cached && typeof cached.at === "number" && cached.zones) return aggregateSummary(cached, now);
  } catch {
    // KV read failure: an empty summary, never a throw.
  }
  return { generatedAt: now.toISOString(), zonesTracked: 0, families: [] };
}

function aggregateSummary(store: SummaryStore, now: Date): RulesQuotaSummary {
  const zones = Object.values(store.zones);
  const families: RulesQuotaFamilySummary[] = [];
  for (const family of Object.keys(ALLOWANCE)) {
    let zonesTracked = 0;
    let totalUsed = 0;
    let atCapZones = 0;
    for (const entry of zones) {
      const f = entry.families[family];
      if (!f || !f.authorized) continue;
      zonesTracked += 1;
      totalUsed += f.used;
      if (f.used >= f.allowance) atCapZones += 1;
    }
    families.push({ family, label: LABEL[family], zonesTracked, totalUsed, atCapZones });
  }
  return {
    generatedAt: now.toISOString(),
    zonesTracked: Math.max(...families.map((f) => f.zonesTracked), 0),
    families,
  };
}
