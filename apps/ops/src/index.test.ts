import { describe, expect, it } from "vitest";
import worker from "./index";

const env = { CF_ACCOUNT_ID: "acct", CF_API_TOKEN: "tok", VAPID_PUBLIC_KEY: "BPUBLICKEY", ASSETS: { fetch: async () => new Response("asset") } };
const ctx = { waitUntil() {}, passThroughOnException() {} } as unknown as ExecutionContext;

describe("worker routing and access guard", () => {
  it("rejects /api/* without the Cloudflare Access assertion header (defense in depth)", async () => {
    const res = await worker.fetch(new Request("https://ops.example/api/vapid-public-key"), env as never, ctx);
    expect(res.status).toBe(403);
  });
  it("serves the VAPID public key behind Access", async () => {
    const res = await worker.fetch(new Request("https://ops.example/api/vapid-public-key", { headers: { "Cf-Access-Jwt-Assertion": "x" } }), env as never, ctx);
    expect(res.status).toBe(200);
    expect(await res.json()).toEqual({ key: "BPUBLICKEY" });
  });
  it("serves static assets for non-API paths", async () => {
    const res = await worker.fetch(new Request("https://ops.example/"), env as never, ctx);
    expect(await res.text()).toBe("asset");
  });
  it("404s unknown API routes", async () => {
    const res = await worker.fetch(new Request("https://ops.example/api/nope", { headers: { "Cf-Access-Jwt-Assertion": "x" } }), env as never, ctx);
    expect(res.status).toBe(404);
  });
});
