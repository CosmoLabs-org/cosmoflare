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
