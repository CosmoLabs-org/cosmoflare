import "@testing-library/jest-dom";
import { expect } from "vitest";
import * as matchers from "vitest-axe/matchers";

// jsdom in this vitest version ships no localStorage; the theme toggle
// (FEAT-041) persists through it, so stub the Web Storage API surface the
// app uses. In-memory, no persistence across reloads — tests reset state
// via removeItem in their own afterEach hooks.
if (typeof globalThis.localStorage === "undefined") {
  const store = new Map<string, string>();
  const storage: Storage = {
    get length() {
      return store.size;
    },
    clear: () => store.clear(),
    getItem: (k) => (store.has(k) ? store.get(k)! : null),
    key: (i) => Array.from(store.keys())[i] ?? null,
    removeItem: (k) => void store.delete(k),
    setItem: (k, v) => void store.set(k, String(v)),
  };
  Object.defineProperty(globalThis, "localStorage", { value: storage });
  Object.defineProperty(globalThis, "sessionStorage", { value: storage });
}

// axe-core gate (BUG-036/037/038 audit). vitest-axe 0.1.0's prebuilt
// `extend-expect` entry is empty and its shipped augmentation targets the
// pre-1.0 `Vi` namespace, so the jest-axe-style matcher is registered manually
// (run `axe(el)`, assert `toHaveNoViolations()` on the results) and typed for
// vitest 2 here. Note: expect.extend requires the matchers object, not the
// bare function (unlike jest).
expect.extend(matchers);

declare module "vitest" {
  interface Assertion<T = any> {
    toHaveNoViolations(): T;
  }
  interface AsymmetricMatchersContaining {
    toHaveNoViolations(): void;
  }
}
