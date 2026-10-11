// @vitest-environment jsdom
// InstallBanner (FEAT-pXNQ5GW phase 1): visibility gates (standalone,
// dismissal), the Chromium beforeinstallprompt capture path, and the
// iOS teach-card branch. jsdom has no matchMedia, so every test stubs it
// BEFORE rendering — mirroring the shell tests' stub-then-render pattern.
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import InstallBanner from "./InstallBanner";

afterEach(cleanup);

type Matcher = (query: string) => { matches: boolean };

// jsdom (as vitest configures it) exposes no localStorage — install a Map
// stand-in, per the ZonesView.test.tsx pattern, so dismissal persistence is
// observable.
const bannerStore: { map: Map<string, string>; clear(): void } = (() => {
  const store = {
    map: new Map<string, string>(),
    clear(): void {
      store.map.clear();
    },
    getItem(key: string): string | null {
      return store.map.get(key) ?? null;
    },
    setItem(key: string, value: string): void {
      store.map.set(key, value);
    },
    removeItem(key: string): void {
      store.map.delete(key);
    },
  };
  Object.defineProperty(window, "localStorage", { value: store, configurable: true, writable: true });
  return store;
})();

/** jsdom lacks matchMedia entirely — install a stub before render. The
 *  returned object carries the listener API framer-motion's
 *  useReducedMotion expects. */
function stubBannerMatchMedia(matches: boolean): Matcher {
  const matcher = vi.fn((query: string) => ({
    matches,
    media: query,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    addListener: () => undefined,
    removeListener: () => undefined,
  })) as unknown as Matcher;
  (window as unknown as { matchMedia?: Matcher }).matchMedia = matcher as unknown as never;
  return matcher;
}

function setStandaloneFlag(value: boolean | undefined): void {
  Object.defineProperty(window.navigator, "standalone", {
    value,
    configurable: true,
  });
}

beforeEach(() => {
  bannerStore.clear();
});

describe("InstallBanner (FEAT-pXNQ5GW)", () => {
  it("renders nothing when standalone (installed users never see it)", () => {
    stubBannerMatchMedia(true);
    const { container } = render(<InstallBanner />);
    expect(container.textContent).toBe("");
  });

  it("renders nothing when the iOS navigator.standalone flag says installed", () => {
    stubBannerMatchMedia(false);
    setStandaloneFlag(true);
    const { container } = render(<InstallBanner />);
    expect(container.textContent).toBe("");
  });

  it("non-standalone Chromium: renders the heading + dismiss once beforeinstallprompt fires", async () => {
    stubBannerMatchMedia(false);
    setStandaloneFlag(undefined);
    const prompt = vi.fn().mockResolvedValue(undefined);
    const { container } = render(<InstallBanner />);
    // Before the browser signals installability the banner stays hidden.
    expect(container.textContent).toBe("");
    const evt = new Event("beforeinstallprompt");
    (evt as Event & { prompt: () => Promise<void> }).prompt = prompt;
    window.dispatchEvent(evt);
    expect(await screen.findByText("Use it like an app")).toBeTruthy();
    expect(screen.getByText(/Add to Home Screen/)).toBeTruthy();
    expect(screen.getByRole("button", { name: "Install app" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Dismiss" })).toBeTruthy();
  });

  it("install button invokes the captured prompt()", async () => {
    stubBannerMatchMedia(false);
    const prompt = vi.fn().mockResolvedValue(undefined);
    render(<InstallBanner />);
    const evt = new Event("beforeinstallprompt");
    (evt as Event & { prompt: () => Promise<void> }).prompt = prompt;
    window.dispatchEvent(evt);
    fireEvent.click(await screen.findByRole("button", { name: "Install app" }));
    await waitFor(() => expect(prompt).toHaveBeenCalledTimes(1));
  });

  it("dismiss click hides the banner and persists the localStorage flag", async () => {
    stubBannerMatchMedia(false);
    render(<InstallBanner />);
    window.dispatchEvent(new Event("beforeinstallprompt"));
    fireEvent.click(await screen.findByRole("button", { name: "Dismiss" }));
    expect(screen.queryByText("Use it like an app")).toBeNull();
    expect(window.localStorage.getItem("pager-install-hint-dismissed")).toBe("1");
    // A remount stays hidden — the flag is remembered.
    const { container } = render(<InstallBanner />);
    expect(container.textContent).toBe("");
  });

  it("iOS Safari: always eligible — the teach-card shows without beforeinstallprompt", () => {
    stubBannerMatchMedia(false);
    setStandaloneFlag(undefined);
    const originalUA = window.navigator.userAgent;
    Object.defineProperty(window.navigator, "userAgent", {
      value: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 Safari/604.1",
      configurable: true,
    });
    try {
      const { container } = render(<InstallBanner />);
      expect(screen.getByText("Use it like an app")).toBeTruthy();
      expect(screen.queryByRole("button", { name: "Install app" })).toBeNull();
      expect(screen.getByRole("button", { name: "Dismiss" })).toBeTruthy();
      expect(container.querySelector("aside")).toBeTruthy();
    } finally {
      Object.defineProperty(window.navigator, "userAgent", {
        value: originalUA,
        configurable: true,
      });
    }
  });
});
