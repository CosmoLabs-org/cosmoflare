// Cosmoflare Ops cron alert engine (FEAT-052): a TS port of the FEAT-049
// zone/D1 alert semantics from internal/webhook/evaluator.go, so the Worker
// can page the phone with the Mac off. Pure module: no fetch, no KV — the
// caller (scheduled.ts) owns I/O and passes observations in.

// Conditions this engine implements. Scope and unit mirror the Go registry
// (pkg/cosmoflare/alerts.go); dataset names the upstream telemetry source the
// condition depends on, for telemetry-gap paging.
export interface ConditionDesc {
  scope: "zone" | "d1" | "kv";
  unit: string;
  dataset: "zone" | "d1" | "kv";
}

export const CONDITIONS: Record<string, ConditionDesc> = {
  "zone-uncached-requests": { scope: "zone", unit: "requests", dataset: "zone" },
  "zone-cache-miss-pct": { scope: "zone", unit: "%", dataset: "zone" },
  "d1-rows-read": { scope: "d1", unit: "rows", dataset: "d1" },
  // FB-29: KV writes/day — the one alert class neither the Go registry nor
  // this port had. Per-namespace write volume over the 24h window.
  "kv-writes": { scope: "kv", unit: "writes", dataset: "kv" },
};

// Rule is one stored alert rule (KV key "rules"). Excludes match
// case-insensitively and exactly against the zone or database name, or its
// ID when the row carries one.
export interface Rule {
  name: string;
  condition: string;
  threshold: number;
  exclude?: string[];
  enabled: boolean;
}

// Observation rows. ZoneRow/D1Row from summary.ts are structurally
// compatible (they carry extra display fields; rows fall back to IDs as
// names when a name list is down, Go design D8).
export interface ZoneObs {
  zone: string;
  id?: string;
  uncached: number;
  eligible: number;
  missPct: number | null; // null under the 100-eligible floor (design D6)
}

export interface D1Obs {
  name: string;
  id?: string;
  rowsRead: number;
}

/** KV namespace observation: writes over the window (kvOperationsAdaptiveGroups, actionType "write"). */
export interface KVObs {
  name: string;
  id?: string;
  writes: number;
}

// FireState is the cooldown memory persisted in KV key "fire-state"
// (Go FireState, epoch-milliseconds form). One read per run; written only
// when the run changed something.
export interface FireState {
  lastFired: Record<string, number>; // alert ID → last fire (epoch ms)
  lastValue: Record<string, number>; // alert ID → value of the last page (drives 2× escalation)
  fires: Record<string, number>; // alert ID → fire count
}

// STATE_TTL_MS bounds the cooldown memory: entries whose lastFired is older
// than 24h can no longer affect a run (cooldowns are 1h) and are pruned so
// the KV blob does not grow forever with retired alert IDs.
export const STATE_TTL_MS = 24 * 60 * 60 * 1000;

// pruneState drops lastFired/lastValue/fires entries whose lastFired is older
// than 24h, mutating state in place. Call before the changed-check so a prune
// alone still persists the smaller blob.
export function pruneState(state: FireState, now: number): void {
  for (const [alertId, at] of Object.entries(state.lastFired)) {
    if (now - at <= STATE_TTL_MS) continue; // "older than 24h" is strict
    delete state.lastFired[alertId];
    delete state.lastValue[alertId];
    delete state.fires[alertId];
  }
}

// Cooldown is per rule/scope, 1h (operator decision O5, watch parity). While
// cooling, a value at least 2× the last paged value escalates: pages again.
export const COOLDOWN_MS = 60 * 60 * 1000;

// ZONE_CACHE_FLOOR is the minimum eligible requests a zone needs in the
// window before its miss ratio is judged (design D6): a zone with 3 requests
// and 2 misses must not page at 67%.
export const ZONE_CACHE_FLOOR = 100;

const ESCALATION_PREFIX = "escalated (≥2× last page): ";

// humanCount renders row counts the way the Go watch does (format.go
// HumanCount): 2.9B / 52.3M / 3.3k on a phone, never exponent form.
export function humanCount(v: number): string {
  const abs = Math.abs(v);
  if (abs >= 1e9) return (v / 1e9).toFixed(1) + "B";
  if (abs >= 1e6) return (v / 1e6).toFixed(1) + "M";
  if (abs >= 1e3) return (v / 1e3).toFixed(1) + "k";
  return String(v);
}

// formatValue renders an observed value or threshold for a page: rows human,
// percents one decimal, others plain — matching Go formatValue (evaluator.go).
export function formatValue(v: number, unit: string): string {
  if (unit === "rows") return humanCount(v);
  if (unit === "%") return v.toFixed(1);
  return String(v);
}

// scopedValue is one observable value: a rule fires per offender (zone or
// database), and the stable identity keys cooldowns, excludes and payload IDs.
export interface ScopedValue {
  scopeName: string; // display name: zone or database name (ID when the name list was down)
  key: string; // stable identity for cooldown/exclude/payload ID
  value: number;
  unit: string;
}

// conditionValues fans a condition out over the observations: one value per
// zone, database, or KV namespace. Empty = the condition cannot fire this cycle.
export function conditionValues(condition: string, zones: ZoneObs[], d1: D1Obs[], kv: KVObs[] = []): ScopedValue[] {
  const desc = CONDITIONS[condition];
  if (!desc) return [];
  if (desc.scope === "zone") {
    const out: ScopedValue[] = [];
    for (const z of zones) {
      if (condition === "zone-cache-miss-pct") {
        if (z.missPct === null) continue; // under the floor: not judgeable (D6)
        out.push({ scopeName: z.zone, key: z.id || z.zone, value: z.missPct, unit: desc.unit });
      } else {
        // zone-uncached-requests: absolute volume, no floor.
        out.push({ scopeName: z.zone, key: z.id || z.zone, value: z.uncached, unit: desc.unit });
      }
    }
    return out;
  }
  if (desc.scope === "kv") {
    const out: ScopedValue[] = [];
    for (const n of kv) {
      out.push({ scopeName: n.name, key: n.id || n.name, value: n.writes, unit: desc.unit });
    }
    return out;
  }
  const out: ScopedValue[] = [];
  for (const d of d1) {
    out.push({ scopeName: d.name, key: d.id || d.name, value: d.rowsRead, unit: desc.unit });
  }
  return out;
}

// excludeMatches reports whether the observation's name or ID is excluded by
// the rule: case-insensitive exact match (Go AlertRule.Excludes).
export function excludeMatches(rule: Rule, sv: ScopedValue): boolean {
  const list = rule.exclude ?? [];
  const lowered = [sv.scopeName, sv.key].filter(Boolean).map((s) => s.toLowerCase());
  return list.some((e) => lowered.includes(e.toLowerCase()));
}

// Fire is one page-worthy event: enough to build the pager payload
// (id = alertId, title = rule name, detail = message) and to key state.
export interface Fire {
  alertId: string;
  ruleName: string;
  scopeName: string; // "" for telemetry-gap fires
  message: string;
  dataset: "zone" | "d1" | "kv" | "";
}

// evaluate runs every enabled rule against the observations. Mutates state in
// place (cooldowns, last values, fire counts). A fire that lands inside its
// alert ID's cooldown is suppressed unless the value is ≥2× the last paged
// value (escalation). Gaps page once per hour per dataset, only when an
// enabled rule depends on the dataset (Go fireGaps, design O5).
export function evaluate(
  rules: Rule[],
  zones: ZoneObs[],
  d1: D1Obs[],
  gaps: Record<string, string>,
  state: FireState,
  now: number,
  kv: KVObs[] = [],
): Fire[] {
  const fires: Fire[] = [];
  const enabled = rules.filter((r) => r.enabled);

  for (const rule of enabled) {
    const desc = CONDITIONS[rule.condition];
    if (!desc) continue; // unknown condition: skip, like Go's registered-but-unimplemented
    for (const sv of conditionValues(rule.condition, zones, d1, kv)) {
      if (sv.value < rule.threshold) continue;
      if (excludeMatches(rule, sv)) continue; // operator-excluded zone/database
      const alertId = `${rule.name}/${sv.key}`;
      const last = state.lastFired[alertId];
      let escalated = false;
      if (last !== undefined && now - last < COOLDOWN_MS) {
        const prevValue = state.lastValue[alertId] ?? 0;
        if (prevValue <= 0 || sv.value < 2 * prevValue) continue;
        escalated = true;
      }
      state.lastFired[alertId] = now;
      state.lastValue[alertId] = sv.value;
      state.fires[alertId] = (state.fires[alertId] ?? 0) + 1;
      const observed = formatValue(sv.value, desc.unit);
      const threshold = formatValue(rule.threshold, desc.unit);
      let message = `${rule.condition} ${sv.scopeName} [${rule.name}]: observed ${observed} meets threshold ${threshold} (${desc.unit})`;
      if (escalated) message = ESCALATION_PREFIX + message;
      fires.push({ alertId, ruleName: rule.name, scopeName: sv.scopeName, message, dataset: desc.dataset });
    }
  }

  // Telemetry gaps: sort datasets for deterministic output, page only when an
  // enabled rule depends on the dataset, 1h cooldown per dataset.
  for (const dataset of Object.keys(gaps).sort()) {
    const depends = enabled.some((r) => CONDITIONS[r.condition]?.dataset === dataset);
    if (!depends) continue;
    const alertId = `telemetry-gap/${dataset}`;
    const last = state.lastFired[alertId];
    if (last !== undefined && now - last < COOLDOWN_MS) continue;
    state.lastFired[alertId] = now;
    state.fires[alertId] = (state.fires[alertId] ?? 0) + 1;
    const message = `telemetry gap: ${dataset} analytics unavailable, ${dataset} rules cannot fire: ${gaps[dataset]}`;
    fires.push({ alertId, ruleName: "telemetry-gap", scopeName: "", message, dataset: dataset as Fire["dataset"] });
  }
  return fires;
}
