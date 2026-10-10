// Rules helpers (FEAT-052, agent I): the pure layer behind the React rules
// view — draft validation, exclude-chip math, the threshold hint, the save
// error label and the RulesHttpError contract. The imperative vanilla
// renderer that used to live here was retired (TASK-pD575FN);
// pager/src/app/views/RulesView.tsx is the only render path now.

import { formatCount } from "./format";
import type { CacheInfo } from "./api";

// ---- /api/rules contract (built by agent B; consumed here) ----

/** One alert rule as the Ops Worker stores and evaluates it. */
export interface AlertRule {
  name: string;
  condition: string;
  threshold: number;
  exclude?: string[];
  enabled: boolean;
}

export interface RulesPayload {
  rules: AlertRule[];
  /** Condition ids the server understands — the condition <select> options. */
  conditions: string[];
  /** True when the account still runs the uncustomised starter set. */
  starter: boolean;
  /** Server-side cache metadata when the Ops Worker served a cached copy. */
  cache?: CacheInfo;
}

/** The D1 rows-read condition — its thresholds are formatted as counts. */
export const D1_ROWS_READ = "d1-rows-read";

/** The KV writes/day condition (FB-29) — thresholds are counts too. */
export const KV_WRITES = "kv-writes";

/** Count-valued conditions: their thresholds render with the count hint. */
const COUNT_CONDITIONS: ReadonlySet<string> = new Set([D1_ROWS_READ, KV_WRITES]);

/**
 * Validate one draft rule against the server's condition list. Returns the
 * error messages, empty when the draft is valid. The trimmed-name check
 * comes first so a whitespace-only name reads as missing.
 */
export function validateRuleDraft(rule: AlertRule, conditions: readonly string[]): string[] {
  const errors: string[] = [];
  if (rule.name.trim() === "") errors.push("Name is required.");
  if (!conditions.includes(rule.condition)) errors.push("Choose a condition from the list.");
  if (!Number.isFinite(rule.threshold) || rule.threshold < 0) errors.push("Threshold must be a number of 0 or more.");
  return errors;
}

/**
 * Add one exclude value (trimmed) unless it is empty or already on the
 * list. Match is exact after trimming — "cf-state" and "CF-State" are
 * distinct names on Cloudflare. Returns a new array; the input is untouched.
 */
export function addExclude(list: readonly string[], raw: string): string[] {
  const value = raw.trim();
  if (value === "") return [...list];
  if (list.includes(value)) return [...list];
  return [...list, value];
}

/** Remove every exact match of `value` (trimmed). Returns a new array. */
export function removeExclude(list: readonly string[], value: string): string[] {
  const needle = value.trim();
  return list.filter((v) => v !== needle);
}

/**
 * Human hint under the threshold input. Only the D1 rows-read condition has
 * one today: "1.0B" for 1e9, so the phone user does not count zeros.
 */
export function thresholdHint(condition: string, threshold: number): string | null {
  if (!COUNT_CONDITIONS.has(condition)) return null;
  if (!Number.isFinite(threshold) || threshold <= 0) return null;
  return formatCount(threshold);
}

/**
 * saveErrorLabel formats a save error for the action bar: rule-scoped
 * ("Rule 3: …") when the server named a rule index, request-level (the bare
 * message) otherwise. BUG-057 defect 5: an index below 0 must not render as
 * "Rule 0".
 */
export function saveErrorLabel(message: string, index?: number): string {
  return index === undefined || index < 0 ? message : `Rule ${index + 1}: ${message}`;
}

/** Thrown when the server answers 400 — carries the offending rule index. */
export class RulesHttpError extends Error {
  readonly index?: number;
  constructor(message: string, index?: number) {
    super(message);
    this.name = "RulesHttpError";
    this.index = index;
  }
}
