// Playwright E2E harness for the Ops app (golden-path spec in ./e2e).
// The webServer boots the same Vite dev server the operator uses — on
// localhost api.ts answers fixtures where they exist, so the tests assert
// on stable chrome (brand, masthead, cards, stat rows) rather than numbers.
import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  use: {
    baseURL: "http://localhost:5173",
  },
  webServer: {
    command: "bun run dev",
    url: "http://localhost:5173",
    reuseExistingServer: !process.env.CI,
  },
  projects: [
    {
      name: "chromium",
      // 960px keeps the quicknav row visible — the desktop breakpoint
      // (≥1024px) swaps it for the sidebar, and the golden path taps
      // quicknav chips.
      use: { ...devices["Desktop Chrome"], viewport: { width: 960, height: 800 } },
    },
  ],
});
