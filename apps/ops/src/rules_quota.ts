// Rule-quota collector (ROAD-p9X66J9): per-zone rule inventory against the
// plan's free allowances — real API numbers only. Five families; Page Rules
// come from the legacy REST endpoint, the other four from the Ruleset Engine
// entrypoints (needs Zone Rulesets Read on the token — every rulesets family
// degrades to authorized:false on 403, the Page Rules row still renders).
// Budget: one cached zone-list lookup (the shared "cron:zones" key, written
// only by scheduled.ts's paginated active-only list — BUG-pP1YH9W) + up to 5
// GETs per zone, KV-cached 5 min per zone — never per request.

import { fetchRetry } from "./retry";
import { activeZones, type OpsEnv } from "./scheduled";

const API = "https://api.cloudflare.com/client/v4";

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

interface CachedQuota {
  at: number;
  value: RulesQuotaPayload;
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
    const zones = await activeZones(env);
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
  /** Per-zone collected-at (ISO) — the store's validated `at` exposed so
   *  staleness is visible per zone instead of masked by generatedAt. */
  entries: Record<string, string>;
}

interface SummaryStore {
  at: number;
  zones: Record<string, RulesQuotaZoneEntry>;
}

/** Collect the next batch and merge into the store. Failures per zone are
 *  skipped (that zone keeps its previous entry); the cursor still advances. */
export async function refreshRulesQuotaBatch(env: OpsEnv, batchSize = QUOTA_BATCH_SIZE, now = new Date()): Promise<RulesQuotaSummary> {
  const zones = await activeZones(env);
  if (zones.length === 0) return { generatedAt: now.toISOString(), zonesTracked: 0, families: [], entries: {} };

  // Guarded like every other OPS_KV access (BUG-p9GM11M): a corrupt cursor
  // value must not throw the cycle — it just restarts the rotation.
  let cursorRaw: number | null = null;
  try {
    cursorRaw = (await env.OPS_KV.get<number>(CURSOR_KEY, "json")) ?? 0;
  } catch {
    cursorRaw = 0;
  }
  const cursor = (((cursorRaw as number) % zones.length) + zones.length) % zones.length;

  let store: SummaryStore = { at: Date.now(), zones: {} };
  try {
    const cached = readStore(await env.OPS_KV.get<SummaryStore>(SUMMARY_KEY, "json"));
    if (cached) store = cached;
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
  // Overlap safety (BUG-p9GM11M): the store write strictly precedes the
  // cursor advance. A cycle killed between the two writes leaves the cursor
  // unmoved, so the next cycle re-processes the batch and re-merges it —
  // the merge is idempotent, so nothing is lost. The old all-or-nothing
  // order (cursor advancing alongside the store) is preserved but now the
  // cursor only moves after the merged store is durably in KV.
  try {
    await env.OPS_KV.put(SUMMARY_KEY, JSON.stringify(store));
  } catch {
    // Cache-write failure is non-fatal; the aggregate below still returns.
  }
  try {
    await env.OPS_KV.put(CURSOR_KEY, JSON.stringify((cursor + n) % zones.length));
  } catch {
    // Cursor-advance failure is non-fatal: the next cycle re-merges.
  }
  return aggregateSummary(store, now);
}

/** Read the store and aggregate — the strip's read path (no fetches). */
export async function readRulesQuotaSummary(env: OpsEnv, now = new Date()): Promise<RulesQuotaSummary> {
  try {
    const cached = readStore(await env.OPS_KV.get<SummaryStore>(SUMMARY_KEY, "json"));
    if (cached) return aggregateSummary(cached, now);
  } catch {
    // KV read failure: an empty summary, never a throw.
  }
  return { generatedAt: now.toISOString(), zonesTracked: 0, families: [], entries: {} };
}

/** Deep-validate one store entry (BUG-p9GM11M): entries are untrusted KV
 *  JSON, and a malformed one used to throw every cron cycle behind a
 *  swallowed catch. Returns null when the entry is unusable. */
function validEntry(entry: unknown): entry is RulesQuotaZoneEntry {
  if (!entry || typeof entry !== "object") return false;
  const e = entry as Partial<RulesQuotaZoneEntry>;
  if (typeof e.at !== "string" || Number.isNaN(Date.parse(e.at))) return false;
  if (!e.families || typeof e.families !== "object") return false;
  for (const f of Object.values(e.families)) {
    if (!f || typeof f !== "object") return false;
    if (typeof f.used !== "number" || typeof f.allowance !== "number" || typeof f.authorized !== "boolean") return false;
  }
  return true;
}

/** Deep-validate a raw KV store value, dropping corrupt entries per-zone
 *  (skip-and-log) instead of throwing the whole cycle. Returns null when the
 *  wrapper itself is unusable. */
function readStore(raw: unknown): SummaryStore | null {
  if (!raw || typeof raw !== "object") return null;
  const { at, zones } = raw as { at?: unknown; zones?: unknown };
  if (typeof at !== "number" || !zones || typeof zones !== "object") return null;
  const clean: Record<string, RulesQuotaZoneEntry> = {};
  let skipped = 0;
  for (const [name, entry] of Object.entries(zones as Record<string, unknown>)) {
    if (validEntry(entry)) clean[name] = entry;
    else skipped += 1;
  }
  if (skipped > 0) console.warn(`rules-quota summary: skipped ${skipped} corrupt zone ${skipped === 1 ? "entry" : "entries"}`);
  return { at, zones: clean };
}

function aggregateSummary(store: SummaryStore, now: Date): RulesQuotaSummary {
  const zones = Object.entries(store.zones);
  const families: RulesQuotaFamilySummary[] = [];
  for (const family of Object.keys(ALLOWANCE)) {
    let zonesTracked = 0;
    let totalUsed = 0;
    let atCapZones = 0;
    for (const [, entry] of zones) {
      const f = entry.families[family];
      if (!f || !f.authorized) continue;
      zonesTracked += 1;
      totalUsed += f.used;
      if (f.used >= f.allowance) atCapZones += 1;
    }
    families.push({ family, label: LABEL[family], zonesTracked, totalUsed, atCapZones });
  }
  // generatedAt is the NEWEST entry's collected-at (BUG-p9GM11M): stamping
  // read-time now presented up-to-48h-old data as current. Per-entry ages
  // are exposed so the client can render staleness per zone.
  const entries: Record<string, string> = {};
  let newest = 0;
  for (const [name, entry] of zones) {
    entries[name] = entry.at;
    const t = Date.parse(entry.at);
    if (!Number.isNaN(t) && t > newest) newest = t;
  }
  return {
    generatedAt: (newest > 0 ? new Date(newest) : now).toISOString(),
    zonesTracked: Math.max(...families.map((f) => f.zonesTracked), 0),
    families,
    entries,
  };
}
