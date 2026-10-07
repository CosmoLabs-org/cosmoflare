/// <reference types="vitest" />
import { defineConfig } from "vitest/config";

// Fully static PWA — no runtime backend; dist/ deploys to any static host.
export default defineConfig({
  base: "/",
  build: {
    target: "esnext",
  },
  test: {
    // parsePayload is pure data validation; no DOM needed (Task 7's IndexedDB
    // store tests will switch to fake-indexeddb).
    environment: "node",
  },
});
