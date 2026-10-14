// Pill fixed-position contract: the liquid-glass pill must stay pinned to
// the VIEWPORT while the page scrolls — not ride along with the content
// (the 2026-10-14 iOS incident: the pill sat at the end of the document
// flow). Asserted in Chromium at phone size; the iOS compositing trigger
// (per-frame canvas repaint during scroll) is exercised by scrolling with
// the glass active.
import { expect, test } from "@playwright/test";

test("the pill stays pinned to the viewport while scrolling", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  const pill = page.locator(".cf-bottomtabs");
  await expect(pill).toBeVisible();

  const before = await pill.boundingBox();
  expect(before).not.toBeNull();

  // Scroll deep into the content.
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight / 2));
  await page.waitForTimeout(250); // let the settled refresh fire (140ms)
  const mid = await pill.boundingBox();

  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  await page.waitForTimeout(250);
  const end = await pill.boundingBox();

  // Fixed to the viewport: identical box at every scroll position, near the
  // bottom edge.
  expect(mid?.y).toBeCloseTo(before!.y, 1);
  expect(end?.y).toBeCloseTo(before!.y, 1);
  expect(before!.y + before!.height).toBeGreaterThan(844 - 80);
});
