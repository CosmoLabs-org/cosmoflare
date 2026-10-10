/// <reference types="vitest" />
import { defineConfig } from "vitest/config";
import tailwindcss from "@tailwindcss/vite";

// Fully static PWA — no runtime backend; dist/ deploys to any static host.
// Tailwind v4 plugin (React rebuild, docs/planning-mode/2026-10-10-pager-react-rebuild.md):
// inert until the CSS entry adds `@import "tailwindcss"` at the P-03 shell
// swap — preflight must never land under the still-vanilla views.
export default defineConfig({
  base: "/",
  plugins: [tailwindcss()],
  build: {
    target: "esnext",
  },
  test: {
    // parsePayload is pure data validation; no DOM needed (Task 7's IndexedDB
    // store tests will switch to fake-indexeddb).
    environment: "node",
  },
});
