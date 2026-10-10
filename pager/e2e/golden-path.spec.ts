// Golden-path E2E (one spec, three tests): the real rendered Ops app against
// the Vite dev server, where localhost endpoints with fixtures answer demo
// data. Assertions target stable chrome — brand, masthead, cards, stat rows —
// never fixture numbers.
import { expect, test } from "@playwright/test";

test.describe("Ops golden path", () => {
  test("overview: brand, quicknav and main paint", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator("h1 .cf-brand-name")).toHaveText("CosmoLabs Ops: Overview");
    await expect(page.locator(".cf-quicknav")).toBeVisible();
    // The overview paints cards/KPIs into .cf-main within 5s.
    await expect(page.locator(".cf-main .cf-card").first()).toBeVisible({ timeout: 5_000 });
    await expect(page.locator(".cf-main .cf-kpi").first()).toBeVisible({ timeout: 5_000 });
  });

  test("navigation: quicknav chip to Workers, row through to profile", async ({ page }) => {
    await page.goto("/");
    await page.locator(".cf-quicknav-link", { hasText: "Workers" }).click();
    await expect(page.locator(".cf-sectionhead-title")).toHaveText("Workers");
    // Tap the first script link in the workers table → the profile masthead
    // names the worker and the profile paints stat rows.
    const scriptLink = page.locator(".cf-link").first();
    const workerName = (await scriptLink.textContent()) ?? "";
    await scriptLink.click();
    await expect(page.locator(".cf-sectionhead-title")).toHaveText(workerName);
    await expect(page.locator(".cf-stat-value").first()).toBeVisible({ timeout: 5_000 });
  });

  test("rules: editor paints rule cards and the condition select lists options", async ({ page }) => {
    await page.goto("/#/rules");
    await expect(page.locator(".cf-sectionhead-title")).toHaveText("Rules");
    // Fixtures supply starter rules — at least one rule card paints.
    await expect(page.locator(".cf-rule").first()).toBeVisible({ timeout: 5_000 });
    // The condition select lists its options.
    const options = page.locator(".cf-select option");
    await expect(options.first()).toBeAttached();
    expect(await options.count()).toBeGreaterThan(0);
  });
});
