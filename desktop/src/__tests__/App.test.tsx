// Regression guard: App must provide a QueryClientProvider so the Dashboard's
// useQuery calls don't throw "No QueryClient set" at runtime. The Dashboard's
// own tests pass because they wrap in a provider; this exercises the real App
// tree end-to-end with the Tauri/fetch/EventSource seams stubbed.

import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, vi } from "vitest";
import App from "../App";

// vitest hoists vi.mock above the imports, so App's transitive
// @tauri-apps/api/core import is mocked. Resolve immediately so App stops
// polling and mounts the Dashboard.
vi.mock("@tauri-apps/api/core", () => ({
  invoke: vi.fn().mockResolvedValue({ url: "http://127.0.0.1:1", token: "t" }),
}));

class FakeEventSource {
  constructor(public url: string) {}
  addEventListener() {}
  removeEventListener() {}
  close() {}
}

describe("App", () => {
  beforeEach(() => {
    vi.stubGlobal("EventSource", FakeEventSource);
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue({ ok: true, json: async () => [] }));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("mounts the Dashboard view without a QueryClient error", async () => {
    // If the provider were missing, this render would throw when the
    // Dashboard's useQuery fires after the endpoint resolves.
    render(<App />);
    await waitFor(() => {
      // "Zones" is a ServiceCard label that only renders once the Dashboard
      // view mounts with a live client.
      expect(screen.getByText("Zones")).toBeInTheDocument();
    });
  });
});
