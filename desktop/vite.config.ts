/// <reference types="vitest" />
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

// Tauri expects a fixed port in dev. See: https://vitejs.dev/config/server-options
export default defineConfig({
  plugins: [react()],
  // Vitest needs jsdom for component tests (@testing-library/react).
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test-setup.ts"],
  },
  // Tauri-specific dev server options.
  clearScreen: false,
  server: {
    port: 1420,
    strictPort: true,
    watch: {
      // Don't watch the Rust source from the Vite dev server.
      ignored: ["**/src-tauri/**"],
    },
  },
  // Env vars starting with TAURI_ are exposed to the webview.
  envPrefix: ["VITE_", "TAURI_"],
  build: {
    target: "esnext",
  },
});
