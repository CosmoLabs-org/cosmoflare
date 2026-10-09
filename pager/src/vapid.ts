// VAPID application server key decoding (FEAT-052 fix for FEAT-045):
// PushManager.subscribe needs the server's public key as bytes; Safari and
// Chrome reject a subscription without it.

/** Decodes an unpadded base64url string (a VAPID public key) to bytes. */
export function urlBase64ToUint8Array(base64url: string): Uint8Array<ArrayBuffer> {
  const padded = base64url + "=".repeat((4 - (base64url.length % 4)) % 4);
  const raw = atob(padded.replace(/-/g, "+").replace(/_/g, "/"));
  const out = new Uint8Array(new ArrayBuffer(raw.length)); // ArrayBuffer-backed: PushManager rejects SharedArrayBuffer views
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
  return out;
}

/** Fetches the VAPID public key the Ops Worker serves (behind Access). */
export async function fetchVapidPublicKey(): Promise<string> {
  const res = await fetch("/api/vapid-public-key", { credentials: "same-origin" });
  if (!res.ok) throw new Error(`key request failed (HTTP ${res.status})`);
  const { key } = (await res.json()) as { key?: string };
  if (!key) throw new Error("the server has no VAPID public key configured");
  return key;
}
