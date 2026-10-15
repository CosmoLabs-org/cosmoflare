// LIVE advisory determinism check (P-19, B5) — the Go suite's
// analytics_live_test.go pattern: runs ONLY when real credentials are in the
// environment (CLOUDFLARE_ACCOUNT_ID + CLOUDFLARE_API_TOKEN, the same vars
// the CLI reads); `bun run test` without them skips this file entirely.
//
// What it proves against the REAL account: two advisory runs for a real zone
// produce a byte-identical agentBrief — the determinism contract behind the
// endpoint's "same facts in, same brief out" guarantee. The second run rides
// the collectors' 1h cache exactly like a second GET /api/advisory does;
// the pure-render half is re-derived from a deep clone so purity is also
// exercised, not just the cache.
import { describe, expect, it } from "vitest";
import { activeZones, type OpsEnv } from "../scheduled";
import { collectZoneFacts } from "./collect";
import { compileBrief } from "./compile";
import { renderAgentBrief } from "./render";
import { TEMPLATES } from "./templates";

// The workers tsconfig has no node types; the vitest/bun runtime provides
// the real `process` this gate reads.
declare const process: { env: Record<string, string | undefined> };

const liveAccount = process.env.CLOUDFLARE_ACCOUNT_ID ?? process.env.CF_ACCOUNT_ID;
const liveToken = process.env.CLOUDFLARE_API_TOKEN ?? process.env.CF_API_TOKEN;
const live = Boolean(liveAccount && liveToken);

describe.skipIf(!live)("advisory live determinism (real account)", () => {
  it("two runs for a real zone render a byte-identical agentBrief", { timeout: 90_000 }, async () => {
    // scheduled.ts's kvCache requires OPS_KV (unlike cache.ts's cached(),
    // which degrades to memory): an in-memory namespace stands in — every
    // upstream read still hits the real account.
    const mem = new Map<string, string>();
    const kv = {
      get: async (key: string) => {
        const v = mem.get(key);
        return v === undefined ? null : (JSON.parse(v) as unknown);
      },
      put: async (key: string, value: string) => {
        mem.set(key, value);
      },
    } as unknown as OpsEnv["OPS_KV"];
    const env = { CF_ACCOUNT_ID: liveAccount!, CF_API_TOKEN: liveToken!, OPS_KV: kv } as unknown as OpsEnv;

    const zones = await activeZones(env);
    expect(zones.length).toBeGreaterThan(0);
    const zone = zones[0].name;

    const facts1 = await collectZoneFacts(env, zone);
    const facts2 = await collectZoneFacts(env, zone);

    const brief1 = compileBrief(facts1, TEMPLATES);
    const brief2 = compileBrief(facts2, TEMPLATES);
    const rendered1 = renderAgentBrief(brief1);
    // Purity: re-render from a deep clone of the first facts, never the
    // cache-backed second pass alone.
    const renderedClone = renderAgentBrief(compileBrief(structuredClone(facts1), TEMPLATES));
    const rendered2 = renderAgentBrief(brief2);

    expect(rendered1).toBe(rendered2);
    expect(rendered1).toBe(renderedClone);

    if (brief1.findings.length > 0) {
      expect(rendered1.startsWith(`# ${zone} — Cloudflare optimization directive`)).toBe(true);
      expect(rendered1.endsWith(`Zone ${zone} (${facts1.zoneId}) — measured ${facts1.window} by CosmoLabs Ops.`)).toBe(true);
    } else {
      // Zero findings is a valid live outcome: the endpoint answers
      // brief:null with an empty agentBrief, never a fabricated one.
      expect(rendered1).toBe("");
    }
  });
});
