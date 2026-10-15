// Ring gauge fit + restyle guards (2026-10-16): the glow head paints on
// used rings and every inner capacity line fits its ring's opening —
// measured in real browser geometry (getBBox), immune to fixture string
// length changes. Vision-model review drove two bloom iterations; this
// spec keeps the fit provable without eyes.
import { expect, test } from "@playwright/test";

test("overview rings: glow head present, capacity lines fit, luminous readout renders", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator(".cf-main .cf-card").first()).toBeVisible({ timeout: 5_000 });
  const rings = page.locator(".cf-ring-svg");
  expect(await rings.count()).toBeGreaterThan(0);
  await expect(page.locator(".cf-ring-value").first()).toBeVisible();
  const heads = page.locator(".cf-ring-head");
  // At least one ring has usage, so at least one glow head paints.
  expect(await heads.count()).toBeGreaterThan(0);

  // Ground truth for the inner-line fit: every capacity line's rendered
  // bbox must fit inside its ring's true inner opening (r - stroke/2).
  const overflows = await page.evaluate(() => {
    const bad: string[] = [];
    for (const svg of Array.from(document.querySelectorAll<SVGSVGElement>(".cf-ring-svg"))) {
      const track = svg.querySelector(".cf-ring-track");
      const line = svg.querySelector(".cf-ring-usedpct");
      if (!track || !line) continue;
      const r = Number(track.getAttribute("r"));
      const sw = Number(getComputedStyle(track).strokeWidth.replace("px", "")) || 11;
      // bbox + computed stroke are render-space; normalize r to the same
      // space via the viewBox ratio before comparing.
      const vb = svg.viewBox.baseVal;
      const rect = svg.getBoundingClientRect();
      const scale = rect.width / vb.width;
      const opening = 2 * (r * scale - sw / 2);
      const w = (line as SVGGraphicsElement).getBBox().width * scale;
      if (w > opening + 1) bad.push(`${line.textContent} (${w.toFixed(1)} > ${opening.toFixed(1)})`);
    }
    return bad;
  });
  expect(overflows).toEqual([]);

  await page.screenshot({ path: "e2e-results/gauge-restyle.png", fullPage: true });
});
