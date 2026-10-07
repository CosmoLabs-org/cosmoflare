// Wire protocol for Cosmoflare Pager push envelopes (FEAT-045).
//
// This mirrors the Go schema in pkg/cosmoflare/alertspush/payload.go exactly:
// the JSON envelope keys are exactly id, severity, service, title, detail,
// fired_at (RFC3339), severity is exactly info | warning | critical, and no
// credentials may ever appear in a payload.

/** Alert severity vocabulary — exactly info | warning | critical (FEAT-041 ramp). */
export type Severity = "info" | "warning" | "critical";

export const SEVERITIES: readonly Severity[] = ["info", "warning", "critical"];

/**
 * Payload is the JSON envelope delivered to pager clients. The field names
 * mirror the Go alertspush.Payload struct's JSON tags on the wire
 * (snake_case, e.g. fired_at).
 */
export interface Payload {
  id: string;
  severity: Severity;
  service: string;
  title: string;
  /** Optional in the Go schema — normalized to "" when absent. */
  detail: string;
  /** RFC3339 timestamp, exactly as the Go sender emits it. */
  fired_at: string;
}

function isSeverity(value: unknown): value is Severity {
  return typeof value === "string" && (SEVERITIES as readonly string[]).includes(value);
}

function isNonEmptyString(value: unknown): value is string {
  return typeof value === "string" && value.length > 0;
}

/**
 * parsePayload validates an untrusted push envelope against the Go schema and
 * returns a typed Payload, or null when the envelope is malformed.
 *
 * Mirrors alertspush.Payload.Validate: non-empty id, service and title, and a
 * severity from the known vocabulary. detail is optional and normalized to
 * "". As a client-side hardening the envelope must also carry a parseable
 * RFC3339 fired_at (the Go sender always emits one; history ordering needs
 * it).
 */
export function parsePayload(data: unknown): Payload | null {
  if (typeof data !== "object" || data === null || Array.isArray(data)) {
    return null;
  }
  const raw = data as Record<string, unknown>;

  if (!isNonEmptyString(raw.id)) return null;
  if (!isSeverity(raw.severity)) return null;
  if (!isNonEmptyString(raw.service)) return null;
  if (!isNonEmptyString(raw.title)) return null;
  if (raw.detail !== undefined && typeof raw.detail !== "string") return null;
  if (typeof raw.fired_at !== "string" || Number.isNaN(Date.parse(raw.fired_at))) {
    return null;
  }

  return {
    id: raw.id,
    severity: raw.severity,
    service: raw.service,
    title: raw.title,
    detail: raw.detail ?? "",
    fired_at: raw.fired_at,
  };
}
