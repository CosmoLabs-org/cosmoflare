// Cosmoflare Ops rules API (FEAT-052): the /api/rules endpoint — GET returns
// the effective rule set (KV "rules", else starters) plus the condition
// catalog and a starter flag; PUT replaces the whole rule set after strict
// validation. NOTE: index.ts verifies the Cloudflare Access JWT on every
// /api/* request BEFORE calling the handler (same gate as /api/subscribe), so
// the handler performs no authentication of its own.

import { CONDITIONS, type Rule } from "./rules";
import { getRules, RULES_KEY, type OpsEnv } from "./scheduled";
import { json } from "./subscriptions";

// MAX_RULES / MAX_EXCLUDES cap the rule set: this is a single-operator pager,
// and 20 rules × 50 excludes is far beyond what one operator watches.
export const MAX_RULES = 20;
export const MAX_EXCLUDES = 50;

// validateRule checks one client rule and returns its normalized form
// (exclude entries trimmed, empties dropped). Returns a reason on failure.
function validateRule(raw: unknown, seenNames: Set<string>): { rule: Rule } | { error: string } {
  if (typeof raw !== "object" || raw === null) return { error: "rule must be a JSON object" };
  const r = raw as { name?: unknown; condition?: unknown; threshold?: unknown; enabled?: unknown; exclude?: unknown };
  if (typeof r.name !== "string" || r.name.trim() === "") return { error: "name is required and must be a non-empty string" };
  const name = r.name.trim();
  if (seenNames.has(name.toLowerCase())) return { error: `duplicate rule name "${name}"` };
  seenNames.add(name.toLowerCase());
  if (typeof r.condition !== "string" || !(r.condition in CONDITIONS)) {
    return { error: `unknown condition "${String(r.condition)}" (see the conditions list)` };
  }
  if (typeof r.threshold !== "number" || !Number.isFinite(r.threshold) || r.threshold < 0) {
    return { error: "threshold must be a finite number ≥ 0" };
  }
  if (typeof r.enabled !== "boolean") return { error: "enabled must be a boolean" };
  if (r.exclude === undefined) {
    return { rule: { name, condition: r.condition, threshold: r.threshold, enabled: r.enabled } };
  }
  if (!Array.isArray(r.exclude) || r.exclude.some((e) => typeof e !== "string")) {
    return { error: "exclude must be an array of strings" };
  }
  const exclude = (r.exclude as string[]).map((e) => e.trim()).filter((e) => e !== "");
  if (exclude.length > MAX_EXCLUDES) return { error: `exclude list exceeds ${MAX_EXCLUDES} entries` };
  return { rule: { name, condition: r.condition, threshold: r.threshold, enabled: r.enabled, exclude } };
}

// handleRules serves the /api/rules route: GET returns the effective rule set
// (KV "rules", else starters) with the condition catalog and starter flag;
// PUT replaces the entire set (max 20 rules) after per-rule validation and
// returns the normalized set; other methods → 405.
export async function handleRules(request: Request, env: OpsEnv): Promise<Response> {
  switch (request.method) {
    case "GET": {
      const rules = await getRules(env);
      const stored = await env.OPS_KV.get(RULES_KEY, "json");
      // starter mirrors getRules' own fallback condition: the starters are
      // what's served when the key is absent OR holds an empty array.
      const starter = !Array.isArray(stored) || stored.length === 0;
      return json({ rules, conditions: Object.keys(CONDITIONS), starter });
    }
    case "PUT": {
      let raw: unknown;
      try {
        raw = await request.json();
      } catch {
        return json({ error: "body must be valid JSON", index: -1 }, 400);
      }
      const rulesRaw = (raw as { rules?: unknown } | null)?.rules;
      if (!Array.isArray(rulesRaw)) return json({ error: "body must be { rules: Rule[] }", index: -1 }, 400);
      if (rulesRaw.length > MAX_RULES) {
        return json({ error: `too many rules (max ${MAX_RULES})`, index: -1 }, 400);
      }
      const seenNames = new Set<string>();
      const rules: Rule[] = [];
      for (let i = 0; i < rulesRaw.length; i++) {
        const checked = validateRule(rulesRaw[i], seenNames);
        if ("error" in checked) return json({ error: checked.error, index: i }, 400);
        rules.push(checked.rule);
      }
      await env.OPS_KV.put(RULES_KEY, JSON.stringify(rules));
      return json({ rules }, 200);
    }
    default:
      return json({ error: "method not allowed" }, 405);
  }
}
