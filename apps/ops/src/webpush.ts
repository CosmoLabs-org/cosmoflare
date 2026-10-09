// Cosmoflare Ops Web Push delivery (FEAT-052): sends pager payloads with
// @block65/webcrypto-web-push v2 (aes128gcm + VAPID, the only encoding Apple
// accepts — v1.x's legacy aesgcm is rejected). Mirrors the Go sender in
// pkg/cosmoflare/alertspush/sender.go: TTL 1h, prune expired endpoints
// (404/410), one failing subscription never aborts the rest.

import { buildPushPayload, type PushMessage, type PushSubscription as LibSubscription } from "@block65/webcrypto-web-push";
import type { StoredSubscription } from "./subscriptions";

// How long push services may keep an undelivered alert (1 hour, matching the
// Go pushTTL). Alerts are time-sensitive; a stale wake-up has little value.
const PUSH_TTL = 3600;

// Urgency for alert pages. "normal" keeps iOS from treating every cron page
// as a break-glass interrupt while still waking the screen when possible.
const PUSH_URGENCY = "normal" as const;

export interface PushOutcome {
  sent: number;
  pruned: number;
  /** Endpoints the push service reported gone (404/410): the caller removes them from KV. */
  prunedEndpoints: string[];
  /** One line per failed delivery: status + push-service reason text, or a transport error. */
  issues: string[];
}

// VapidConfig is the slice of OpsEnv sendPushes needs; the worker Env
// satisfies it structurally.
export interface VapidConfig {
  VAPID_PUBLIC_KEY: string;
  VAPID_PRIVATE_KEY: string;
  VAPID_SUBJECT: string;
}

/**
 * vapidSubject normalizes the VAPID subject (env VAPID_SUBJECT). This library
 * puts the value into the JWT sub claim verbatim — unlike webpush-go, which
 * adds the "mailto:" prefix itself. A bare address or a double prefix both
 * get rejected, so: absent prefix → add one; more than one "mailto:" → the
 * configured value is already broken, keep it and let sendPushes report it.
 * Returns the normalized subject plus an error text when unusable.
 */
export function vapidSubject(raw: string): { subject: string; error?: string } {
  const trimmed = raw.trim();
  if (!trimmed) return { subject: "", error: "VAPID_SUBJECT is empty; push services require a mailto: contact" };
  const occurrences = trimmed.split("mailto:").length - 1;
  if (occurrences === 0) return { subject: `mailto:${trimmed}` };
  if (occurrences > 1) return { subject: trimmed, error: `VAPID_SUBJECT must contain "mailto:" exactly once (got ${occurrences})` };
  return { subject: trimmed };
}

/**
 * sendPushes delivers the pager payload to every subscription. Expired
 * endpoints (HTTP 404/410 from the push service) count as pruned — the caller
 * removes them from KV. Other non-2xx statuses land in issues with the
 * status and the push service's reason text (Apple: BadJwtToken,
 * BadVapidPublicKey, VapidPkHashMismatch). Transport failures are issues too;
 * none of them abort the remaining subscriptions.
 */
export async function sendPushes(
  vapid: VapidConfig,
  subs: StoredSubscription[],
  payload: unknown,
): Promise<PushOutcome> {
  const outcome: PushOutcome = { sent: 0, pruned: 0, prunedEndpoints: [], issues: [] };
  const subj = vapidSubject(vapid.VAPID_SUBJECT);
  if (subj.error) {
    outcome.issues.push(subj.error);
    return outcome;
  }
  const envelope = JSON.stringify(payload);
  const message: PushMessage = { data: envelope, options: { ttl: PUSH_TTL, urgency: PUSH_URGENCY } };
  for (const sub of subs) {
    const libSub: LibSubscription = { endpoint: sub.endpoint, expirationTime: null, keys: { p256dh: sub.p256dh, auth: sub.auth } };
    try {
      const request = await buildPushPayload(message, libSub, { subject: subj.subject, publicKey: vapid.VAPID_PUBLIC_KEY, privateKey: vapid.VAPID_PRIVATE_KEY });
      const res = await fetch(sub.endpoint, request);
      if (res.status === 404 || res.status === 410) {
        outcome.pruned++; // endpoint gone: caller prunes it from KV
        outcome.prunedEndpoints.push(sub.endpoint);
        continue;
      }
      if (res.status >= 200 && res.status < 300) {
        outcome.sent++;
        continue;
      }
      outcome.issues.push(`web push ${shortEndpoint(sub.endpoint)}: unexpected status ${res.status} ${await reasonText(res)}`);
    } catch (err) {
      outcome.issues.push(`web push ${shortEndpoint(sub.endpoint)}: ${(err as Error).message}`);
    }
  }
  return outcome;
}

/** reasonText extracts the push service's reason (Apple sends {"reason":"BadJwtToken"}). */
async function reasonText(res: Response): Promise<string> {
  try {
    const body = await res.json() as { reason?: string; message?: string };
    return body.reason ?? body.message ?? "";
  } catch {
    return ""; // non-JSON body: the status alone still identifies the failure class
  }
}

/** shortEndpoint keeps issue lines readable: scheme host + tail path. */
function shortEndpoint(endpoint: string): string {
  try {
    const url = new URL(endpoint);
    return url.host + "…";
  } catch {
    return endpoint.slice(0, 24);
  }
}
