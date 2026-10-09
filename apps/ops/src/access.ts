// Cloudflare Access JWT verification (FEAT-052). Access already blocks
// unauthenticated requests at the edge; this verifies the signed assertion
// itself so the API stays closed if the Access app is ever removed or
// misconfigured — a bare header-presence check can be forged by any caller.
// RS256 against the team's published keys; aud, iss and exp checked; fails
// closed when the team domain or audience is not configured.

export type AccessResult = { ok: true; email: string } | { ok: false; reason: string };

interface Jwks {
  keys: (JsonWebKey & { kid?: string })[];
}

const JWKS_TTL_MS = 60 * 60 * 1000;
let jwksCache: { team: string; at: number; jwks: Jwks } | null = null;

/** Test hook: forget cached team keys. */
export function resetJwksCache(): void {
  jwksCache = null;
}

function b64urlDecode(s: string): Uint8Array<ArrayBuffer> {
  const padded = s + "=".repeat((4 - (s.length % 4)) % 4);
  const raw = atob(padded.replace(/-/g, "+").replace(/_/g, "/"));
  const out = new Uint8Array(new ArrayBuffer(raw.length));
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
  return out;
}

async function teamKeys(team: string, fetcher: typeof fetch): Promise<Jwks> {
  if (jwksCache && jwksCache.team === team && Date.now() - jwksCache.at < JWKS_TTL_MS) return jwksCache.jwks;
  const res = await fetcher(`https://${team}/cdn-cgi/access/certs`);
  if (!res.ok) throw new Error(`certs HTTP ${res.status}`);
  const jwks = (await res.json()) as Jwks;
  jwksCache = { team, at: Date.now(), jwks };
  return jwks;
}

export async function verifyAccessJwt(token: string, team: string, aud: string, fetcher: typeof fetch = fetch): Promise<AccessResult> {
  if (!team || !aud) return { ok: false, reason: "access not configured" };
  const parts = token.split(".");
  if (parts.length !== 3) return { ok: false, reason: "malformed token" };
  try {
    const header = JSON.parse(new TextDecoder().decode(b64urlDecode(parts[0]))) as { alg?: string; kid?: string };
    const claims = JSON.parse(new TextDecoder().decode(b64urlDecode(parts[1]))) as { aud?: string | string[]; iss?: string; exp?: number; email?: string };
    if (header.alg !== "RS256") return { ok: false, reason: "unexpected alg" };
    const jwk = (await teamKeys(team, fetcher)).keys.find((k) => k.kid === header.kid);
    if (!jwk) return { ok: false, reason: "unknown key" };
    const key = await crypto.subtle.importKey("jwk", jwk, { name: "RSASSA-PKCS1-v1_5", hash: "SHA-256" }, false, ["verify"]);
    const valid = await crypto.subtle.verify("RSASSA-PKCS1-v1_5", key, b64urlDecode(parts[2]), new TextEncoder().encode(`${parts[0]}.${parts[1]}`));
    if (!valid) return { ok: false, reason: "bad signature" };
    const auds = Array.isArray(claims.aud) ? claims.aud : [claims.aud];
    if (!auds.includes(aud)) return { ok: false, reason: "wrong audience" };
    if (claims.iss !== `https://${team}`) return { ok: false, reason: "wrong issuer" };
    if (typeof claims.exp !== "number" || claims.exp * 1000 <= Date.now()) return { ok: false, reason: "expired" };
    return { ok: true, email: claims.email ?? "" };
  } catch {
    return { ok: false, reason: "unverifiable token" };
  }
}
