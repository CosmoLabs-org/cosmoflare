// Commands-to-AI-Agent E2E (P-18/P-19, B5): the real rendered Ops app on the
// Vite dev server, with the advisory envelope served at the network boundary
// in the endpoint's exact byte format. Proves the full dispatch ritual in a
// real browser: feed row → expand → button → modal → the agentBrief text →
// clipboard copy is byte-identical to the served brief → Esc closes.
// (The real-account half of the live pass — two real runs rendering a
// byte-identical brief — is apps/ops/src/briefs/advisory.live.test.ts.)
import { expect, test } from "@playwright/test";

const AGENT_BRIEF = `# demo.example — Cloudflare optimization directive

## Problem
- [critical] low-cache-hit: 7d cache hit rate 0.4% — 40 of 10000 requests served from edge cache

## Actions
1. Raise the edge cache hit rate above the 5% floor: send origin Cache-Control headers on cacheable responses, or add a zone Cache Rule — HTML needs a Cache Rule, s-maxage alone does nothing.

## Verify
- cosmoflare efficiency --json --window 7 | jq '.data.zone_coverage[] | select(.zone=="demo.example")' — hit_rate_pct should climb above 5

## Context
Zone demo.example (z-demo) — measured 7d by CosmoLabs Ops.`;

const ADVISORY_ENVELOPE = {
  zone: "demo.example",
  zoneId: "z-demo",
  generatedAt: "2026-10-15T22:40:00.000Z",
  facts: { zone: "demo.example", zoneId: "z-demo", window: "7d", firingAlerts: [] },
  brief: { findings: [{ kind: "low-cache-hit", severity: "critical", evidence: "7d cache hit rate 0.4%" }] },
  rendered: { agentBrief: AGENT_BRIEF },
};

test.describe("Commands to AI Agent", () => {
  test("feed card to modal: brief renders, Copy lands byte-identical, Esc closes", async ({ page, context }) => {
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);

    // The feed that carries the firing entry (zone = ruleId scope segment),
    // and the advisory envelope the modal fetches on open.
    await page.route("**/api/alerts/feed*", (route) =>
      route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({
          feed: [
            {
              ruleId: "hot rule/demo.example",
              condition: "zone-cache-miss-pct",
              title: "Cache miss share high",
              value: 96,
              threshold: 60,
              unit: "%",
              message: "96% of cache-eligible requests missed the edge",
              context: "demo feed entry for the advisory dispatch ritual",
              ts: new Date().toISOString(),
              sent: true,
            },
          ],
        }),
      }),
    );
    await page.route("**/api/advisory*", (route) =>
      route.fulfill({ contentType: "application/json", body: JSON.stringify(ADVISORY_ENVELOPE) }),
    );

    await page.goto("/#/alerts");
    await expect(page.locator(".cf-sectionhead-title")).toHaveText("Alerts");

    // Expand the firing row, then open the dispatch modal from its detail.
    const row = page.locator(".cf-feed-item").first();
    await row.locator("button").first().click();
    await page.getByRole("button", { name: "Commands to AI Agent" }).click();

    const dialog = page.locator('[role="dialog"]');
    await expect(dialog).toBeVisible();
    await expect(dialog.locator("#cf-agent-title")).toHaveText("Commands to AI Agent · demo.example");
    await expect(dialog.locator(".cf-modal-body")).toHaveText("measured 7d · CosmoLabs Ops");

    // The mono block carries the served brief, byte for byte.
    const brief = dialog.locator(".cf-agent-brief");
    await expect(brief).toBeVisible();
    expect(await brief.textContent()).toBe(AGENT_BRIEF);

    // Copy puts the exact served text on the clipboard. The checkmark face
    // is aria-hidden (decoration), so the state is asserted on its class.
    const copyBtn = dialog.locator(".cf-agent-copy");
    await copyBtn.click();
    await expect(copyBtn.locator(".cf-agent-check")).toHaveClass(/is-on/);
    const clipboard = await page.evaluate(() => navigator.clipboard.readText());
    expect(clipboard).toBe(AGENT_BRIEF);

    await page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();

    await page.screenshot({ path: "e2e-results/advisory-modal.png", fullPage: true });
  });
});
