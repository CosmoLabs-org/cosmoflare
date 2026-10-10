import { afterEach, describe, expect, it, vi } from "vitest";
import {
  formatCount,
  levelForD1,
  levelForMiss,
  levelForUncached,
  levelForUsage,
  pacingLabel,
  paintOverview,
  periodEndsLabel,
  ringPropsFor,
} from "./dashboard";
import type { Billing, BillingPeriod, FetchResult, ProductUsage, Summary } from "./api";

describe("dashboard formatting", () => {
  it("formats counts like the CLI (2.9B / 52.3M / 3.3k / 522)", () => {
    expect(formatCount(2_945_546_702)).toBe("2.9B");
    expect(formatCount(52_332_502)).toBe("52.3M");
    expect(formatCount(3289)).toBe("3.3k");
    expect(formatCount(522)).toBe("522");
  });
  it("maps values to the starter alert thresholds (O10)", () => {
    expect(levelForUncached(36_321)).toBe("critical");
    expect(levelForUncached(8_192)).toBe("warning"); // amber band at half the 10k paging threshold
    expect(levelForUncached(4_000)).toBe("ok");
    expect(levelForMiss(74)).toBe("warning");
    expect(levelForMiss(null)).toBe("ok");
    expect(levelForD1(2.9e9)).toBe("critical");
    expect(levelForD1(52e6)).toBe("ok");
    expect(levelForUsage(120)).toBe("critical");
    expect(levelForUsage(85)).toBe("warning");
    expect(levelForUsage(10)).toBe("ok");
  });
});

describe("pacingLabel", () => {
  it("states the metric, that it is a projection, and the allowance", () => {
    const d1: ProductUsage = {
      id: "d1.rows_read", product: "D1", metric: "Rows read", unit: "rows",
      included: 25e9, used: 640e6, projected: 27e9, unitPriceUsd: 0.001,
      priceUnit: 1e6, projectedOverageUsd: 2, topConsumers: [],
    };
    expect(pacingLabel(d1)).toBe("D1 rows read · projected % of 25.0B rows");
  });
});

describe("ringPropsFor", () => {
  const period: BillingPeriod = {
    start: "2026-09-23T00:00:00Z",
    end: "2026-10-23T00:00:00Z",
    day: 16,
    days: 30,
    source: "anchor",
  };
  const d1: ProductUsage = {
    id: "d1.rows_read", product: "D1", metric: "Rows read", unit: "rows",
    included: 25e9, used: 640e6, projected: 27e9, unitPriceUsd: 0.001,
    priceUnit: 1e6, projectedOverageUsd: 2, topConsumers: [],
  };
  it("maps usage into ring geometry: % of allowance, expected tick, label", () => {
    const props = ringPropsFor(d1, period);
    expect(props.usedPct).toBeCloseTo((640e6 / 25e9) * 100);
    expect(props.projectedPct).toBeCloseTo(108);
    expect(props.expectedPct).toBeCloseTo((16 / 30) * 100);
    expect(props.label).toBe("D1 rows read");
    expect(props.sublabel).toBe("25.0B rows");
  });
  it("a zero allowance never divides by zero", () => {
    const empty: ProductUsage = { ...d1, included: 0 };
    expect(ringPropsFor(empty, period).usedPct).toBe(0);
  });
  it("storage rings carry the size as the second line, not a repeated % (IMP-002)", () => {
    const kv: ProductUsage = {
      id: "kv.storage", product: "KV", metric: "Storage", unit: "GB",
      included: 1, used: 3.888, projected: 3.888, unitPriceUsd: 0.15,
      priceUnit: 1e6, projectedOverageUsd: 0.43, topConsumers: [],
    };
    const props = ringPropsFor(kv, period);
    expect(props.usedLineText).toBe("3.9 GB of 1.0 GB");
    // The sublabel would duplicate the allowance right under the size line.
    expect(props.sublabel).toBeUndefined();
  });
  it("non-storage rings keep the default used-% line", () => {
    expect(ringPropsFor(d1, period).usedLineText).toBeUndefined();
  });
});

describe("periodEndsLabel", () => {
  it("marks a calendar-sourced period as an assumption", () => {
    const label = periodEndsLabel("2026-11-01T00:00:00Z", "calendar");
    expect(label).toMatch(/Nov 1/);
    expect(label).toContain("(calendar month)");
  });
  it("leaves subscription-sourced periods unmarked", () => {
    const label = periodEndsLabel("2026-11-01T00:00:00Z", "subscription");
    expect(label).toMatch(/Nov 1/);
    expect(label).not.toContain("(calendar month)");
  });
});

// ---- paintOverview attention rows (UI-2) ----
// Node-environment DOM stubs mirroring the FakeNode pattern from
// views.test.ts: minimal fake elements that record children, attributes and
// click listeners, so the accordion behaviour is testable without a browser.
type DashHandler = () => unknown;
interface DashNode {
  tagName: string;
  className: string;
  textContent: string;
  hidden: boolean;
  type: string;
  href: string;
  style: { width: string; setProperty: (name: string, value: string) => void };
  attrs: Record<string, string>;
  classList: { add: (...c: string[]) => void; remove: (...c: string[]) => void; contains: (c: string) => boolean };
  children: DashNode[];
  setAttribute: (name: string, value: string) => void;
  getAttribute: (name: string) => string | null;
  append: (...nodes: DashNode[]) => void;
  insertBefore: (newNode: DashNode, ref: DashNode) => void;
  replaceChildren: (...nodes: DashNode[]) => void;
  addEventListener: (type: string, fn: DashHandler) => void;
  _click: () => Promise<void>;
}

function dashEl(tag: string): DashNode {
  const handlers: Record<string, DashHandler[]> = {};
  const classes = new Set<string>();
  const node: DashNode = {
    tagName: tag.toUpperCase(),
    className: "",
    textContent: "",
    hidden: false,
    type: "",
    href: "",
    style: { width: "", setProperty: () => undefined },
    attrs: {},
    classList: {
      add: (...c) => {
        c.forEach((x) => classes.add(x));
        node.className = [...classes].join(" ");
      },
      remove: (...c) => {
        c.forEach((x) => classes.delete(x));
        node.className = [...classes].join(" ");
      },
      contains: (c) => classes.has(c),
    },
    children: [],
    setAttribute(name, value) {
      node.attrs[name] = value;
    },
    getAttribute(name) {
      return node.attrs[name] ?? null;
    },
    append(...nodes) {
      node.children.push(...nodes);
    },
    insertBefore(newNode, ref) {
      const idx = node.children.indexOf(ref);
      if (idx === -1) node.children.push(newNode);
      else node.children.splice(idx, 0, newNode);
    },
    replaceChildren(...nodes) {
      node.children = [...nodes];
    },
    addEventListener(type, fn) {
      (handlers[type] ??= []).push(fn);
    },
    async _click() {
      for (const fn of handlers.click ?? []) await fn();
    },
  };
  return node;
}

// Depth-first collect of every descendant carrying a class token.
function dashFindAll(root: DashNode, className: string): DashNode[] {
  const found: DashNode[] = [];
  const stack = [...root.children];
  while (stack.length > 0) {
    const node = stack.shift()!;
    if (node.className.split(/\s+/).includes(className)) found.push(node);
    stack.unshift(...node.children);
  }
  return found;
}

function dashFirstChild(root: DashNode, className: string): DashNode {
  const hit = dashFindAll(root, className)[0];
  if (!hit) throw new Error(`element not found: .${className}`);
  return hit;
}

function dashSummary(): Summary {
  return {
    generatedAt: "2026-10-09T00:00:00Z",
    windowHours: 24,
    usage: [],
    d1: [],
    zones: [],
    workers: [],
    errors: [],
  };
}

function dashProduct(): ProductUsage {
  return {
    id: "workers.requests",
    product: "Workers",
    metric: "Requests",
    unit: "requests",
    included: 10_000_000,
    used: 12_000_000,
    projected: 12_000_000,
    unitPriceUsd: 0.3,
    priceUnit: 1e6,
    projectedOverageUsd: 0.6,
    topConsumers: [],
  };
}

function dashBilling(products: ProductUsage[]): Billing {
  return {
    generatedAt: "2026-10-09T00:00:00Z",
    period: { start: "2026-10-01T00:00:00Z", end: "2026-11-01T00:00:00Z", day: 9, days: 31, source: "calendar" },
    products,
    totalProjectedOverageUsd: products.reduce((s, p) => s + p.projectedOverageUsd, 0),
    projects: [],
    pricing: { verifiedOn: "2026-10-09", sources: [] },
    errors: [],
  } as unknown as Billing;
}

function stubPaintDoc(): void {
  vi.stubGlobal("document", {
    createElement: (tag: string) => dashEl(tag),
    createElementNS: (_ns: string, tag: string) => dashEl(tag),
    getElementById: () => null,
  });
}

describe("paintOverview attention rows (UI-2)", () => {
  const res = <T>(data: T): FetchResult<T> => ({ data, source: "network", ageSec: 0, demo: false });

  function paintWithOverage(): DashNode {
    stubPaintDoc();
    const root = dashEl("div");
    paintOverview(root as unknown as HTMLElement, res(dashSummary()), res(dashBilling([dashProduct()])));
    return root;
  }

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders each attention row with mark + title on the first line (detail no longer inlined)", () => {
    const root = paintWithOverage();
    const rows = dashFindAll(root, "cf-attention");
    expect(rows).toHaveLength(1);
    const toggle = dashFirstChild(rows[0], "cf-attention-toggle");
    expect(toggle.tagName).toBe("BUTTON");
    expect(dashFirstChild(toggle, "cf-attention-mark").textContent).toBe("!");
    expect(dashFirstChild(toggle, "cf-attention-title").textContent).toBe("Workers requests");
  });

  it("starts collapsed: the detail block is hidden and aria-expanded is false", () => {
    const root = paintWithOverage();
    const row = dashFindAll(root, "cf-attention")[0];
    const detail = dashFirstChild(row, "cf-attention-detail");
    const toggle = dashFirstChild(row, "cf-attention-toggle");
    expect(detail.hidden).toBe(true);
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
    // The FULL detail text lives in the expandable block, not the title line.
    const text = dashFirstChild(detail, "cf-attention-text").textContent;
    expect(text).toContain("of 10.0M requests included");
    expect(text).toContain("+$0.60 projected");
  });

  it("tapping the control expands the detail and flips aria-expanded; tapping again collapses", async () => {
    const root = paintWithOverage();
    const row = dashFindAll(root, "cf-attention")[0];
    const detail = dashFirstChild(row, "cf-attention-detail");
    const toggle = dashFirstChild(row, "cf-attention-toggle");
    await toggle._click();
    expect(detail.hidden).toBe(false);
    expect(toggle.getAttribute("aria-expanded")).toBe("true");
    await toggle._click();
    expect(detail.hidden).toBe(true);
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
  });

  it("keeps navigation available as a secondary 'Open section →' link inside the detail", () => {
    const root = paintWithOverage();
    const row = dashFindAll(root, "cf-attention")[0];
    const open = dashFirstChild(dashFirstChild(row, "cf-attention-detail"), "cf-attention-open");
    expect(open.textContent).toContain("Open section");
    expect(open.href).toBe("#/billing");
  });
});
