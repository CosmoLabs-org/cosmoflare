import { describe, expect, it } from "vitest";
import worker from "./index";

const env = { CF_ACCOUNT_ID: "acct", CF_API_TOKEN: "tok", VAPID_PUBLIC_KEY: "BPUBLICKEY", ACCESS_TEAM_DOMAIN: "", ACCESS_AUD: "", ASSETS: { fetch: async () => new Response("asset") } };

describe("worker routing and access guard", () => {
  it("rejects /api/* without the Cloudflare Access assertion header (defense in depth)", async () => {
    const res = await worker.fetch(new Request("https://ops.example/api/vapid-public-key"), env as never);
    expect(res.status).toBe(403);
  });
  it("rejects a forged assertion header (fails closed without verifiable Access config)", async () => {
    const res = await worker.fetch(new Request("https://ops.example/api/vapid-public-key", { headers: { "Cf-Access-Jwt-Assertion": "x" } }), env as never);
    expect(res.status).toBe(403);
  });
  it("serves static assets for non-API paths", async () => {
    const res = await worker.fetch(new Request("https://ops.example/"), env as never);
    expect(await res.text()).toBe("asset");
  });
});
