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

  // Ground truth for the interior fit: the VALUE line and the interior
  // "USED" line must fit inside their ring's true inner opening
  // (r - stroke/2). The value is the hero — it gets measured too, not just
  // the small lines.
  const overflows = await page.evaluate(() => {
    const bad: string[] = [];
    for (const svg of Array.from(document.querySelectorAll<SVGSVGElement>(".cf-ring-svg"))) {
      const track = svg.querySelector(".cf-ring-track");
      if (!track) continue;
      const r = Number(track.getAttribute("r"));
      // Stroke width is computed in render px; getBBox is viewBox units —
      // normalize the stroke through the viewBox scale, then everything is
      // viewBox-space and exact.
      const vb = svg.viewBox.baseVal;
      const rect = svg.getBoundingClientRect();
      const scale = rect.width / vb.width;
      const swVB = (Number(getComputedStyle(track).strokeWidth.replace("px", "")) || 11) / scale;
      const opening = 2 * (r - swVB / 2);
      const cx = vb.width / 2;
      const cy = vb.width / 2; // ring center — the band grows below it
      for (const t of Array.from(svg.querySelectorAll<SVGTextElement>(".cf-ring-value, .cf-ring-usedpct"))) {
        const bb = (t as SVGGraphicsElement).getBBox();
        const dx = bb.x + bb.width / 2 - cx;
        const dy = bb.y + bb.height / 2 - cy;
        if (Math.hypot(dx, dy) > r) continue; // outside the circle → band line
        if (bb.width > opening + 1) bad.push(`${t.textContent} (${bb.width.toFixed(1)} > ${opening.toFixed(1)})`);
      }
    }
    return bad;
  });
  expect(overflows).toEqual([]);

  await page.screenshot({ path: "e2e-results/gauge-restyle.png", fullPage: true });
});
