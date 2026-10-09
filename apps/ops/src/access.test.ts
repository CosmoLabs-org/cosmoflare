import { beforeAll, describe, expect, it } from "vitest";
import { verifyAccessJwt, resetJwksCache } from "./access";

const TEAM = "team.cloudflareaccess.com";
const AUD = "aud-123";
let priv: CryptoKey;
let jwk: JsonWebKey;

const b64url = (b: ArrayBuffer | Uint8Array) =>
  btoa(String.fromCharCode(...new Uint8Array(b instanceof Uint8Array ? b : new Uint8Array(b)))).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
const enc = (o: unknown) => b64url(new TextEncoder().encode(JSON.stringify(o)));

async function sign(claims: Record<string, unknown>, kid = "k1", key = priv): Promise<string> {
  const head = enc({ alg: "RS256", kid, typ: "JWT" });
  const body = enc(claims);
  const sig = await crypto.subtle.sign("RSASSA-PKCS1-v1_5", key, new TextEncoder().encode(`${head}.${body}`));
  return `${head}.${body}.${b64url(sig)}`;
}

const certsFetch = (async () => new Response(JSON.stringify({ keys: [{ ...jwk, kid: "k1", alg: "RS256", use: "sig" }] }))) as typeof fetch;
const now = () => Math.floor(Date.now() / 1000);
const good = () => ({ aud: [AUD], iss: `https://${TEAM}`, exp: now() + 600, email: "owner@example.com" });

beforeAll(async () => {
  const pair = (await crypto.subtle.generateKey(
    { name: "RSASSA-PKCS1-v1_5", modulusLength: 2048, publicExponent: new Uint8Array([1, 0, 1]), hash: "SHA-256" },
    true, ["sign", "verify"],
  )) as CryptoKeyPair;
  priv = pair.privateKey;
  jwk = (await crypto.subtle.exportKey("jwk", pair.publicKey)) as JsonWebKey;
});

describe("verifyAccessJwt", () => {
  it("accepts a token signed by the team key with the right aud, iss and exp", async () => {
    resetJwksCache();
    expect(await verifyAccessJwt(await sign(good()), TEAM, AUD, certsFetch)).toEqual({ ok: true, email: "owner@example.com" });
  });
  it("rejects a forged header value", async () => {
    resetJwksCache();
    expect((await verifyAccessJwt("x", TEAM, AUD, certsFetch)).ok).toBe(false);
  });
  it("rejects wrong audience, wrong issuer, expired, unknown kid", async () => {
    resetJwksCache();
    expect((await verifyAccessJwt(await sign({ ...good(), aud: ["other"] }), TEAM, AUD, certsFetch)).ok).toBe(false);
    expect((await verifyAccessJwt(await sign({ ...good(), iss: "https://evil.cloudflareaccess.com" }), TEAM, AUD, certsFetch)).ok).toBe(false);
    expect((await verifyAccessJwt(await sign({ ...good(), exp: now() - 5 }), TEAM, AUD, certsFetch)).ok).toBe(false);
    expect((await verifyAccessJwt(await sign(good(), "k9"), TEAM, AUD, certsFetch)).ok).toBe(false);
  });
  it("rejects a token signed by a different key", async () => {
    resetJwksCache();
    const other = (await crypto.subtle.generateKey(
      { name: "RSASSA-PKCS1-v1_5", modulusLength: 2048, publicExponent: new Uint8Array([1, 0, 1]), hash: "SHA-256" }, true, ["sign", "verify"],
    )) as CryptoKeyPair;
    expect((await verifyAccessJwt(await sign(good(), "k1", other.privateKey), TEAM, AUD, certsFetch)).ok).toBe(false);
  });
  it("fails closed when team domain or audience is not configured", async () => {
    resetJwksCache();
    const token = await sign(good());
    expect((await verifyAccessJwt(token, "", AUD, certsFetch)).ok).toBe(false);
    expect((await verifyAccessJwt(token, TEAM, "", certsFetch)).ok).toBe(false);
  });
});
