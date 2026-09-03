import "@testing-library/jest-dom";
import { expect } from "vitest";
import * as matchers from "vitest-axe/matchers";

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
