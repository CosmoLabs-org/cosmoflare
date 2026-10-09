import { afterEach, describe, expect, it, vi } from "vitest";
import { fadeSwap, refreshButton } from "./refresh";

// Node-environment DOM stubs: vitest here runs with environment "node" and no
// DOM library is installed, so refreshButton/fadeSwap are exercised against
// minimal fake elements (same approach as views.test.ts) that track classes,
// attributes, styles and captured click listeners.
type Handler = (ev?: unknown) => void;

interface FakeClassList {
  add(...names: string[]): void;
  remove(...names: string[]): void;
  contains(name: string): boolean;
}

function makeClassList(): FakeClassList {
  const set = new Set<string>();
  return {
    add: (...names) => names.forEach((n) => set.add(n)),
    remove: (...names) => names.forEach((n) => set.delete(n)),
    contains: (name) => set.has(name),
  };
}

interface FakeEl {
  tagName: string;
  className: string;
  textContent: string;
  innerHTML: string;
  title: string;
  disabled: boolean;
  type: string;
  style: Record<string, string>;
  offsetWidth: number;
  classList: FakeClassList;
  children: FakeEl[];
  attributes: Map<string, string>;
  append(...nodes: FakeEl[]): void;
  replaceChildren(...nodes: FakeEl[]): void;
  setAttribute(name: string, value: string): void;
  removeAttribute(name: string): void;
  getAttribute(name: string): string | null;
  addEventListener(type: string, fn: Handler): void;
  _click(): Promise<void>;
}

function fakeEl(tag: string): FakeEl {
  const handlers: Record<string, Handler[]> = {};
  const node: FakeEl = {
    tagName: tag.toUpperCase(),
    className: "",
    textContent: "",
    innerHTML: "",
    title: "",
    disabled: false,
    type: "",
    style: {},
    offsetWidth: 100,
    classList: makeClassList(),
    children: [],
    attributes: new Map(),
    append(...nodes) {
      node.children.push(...nodes);
    },
    replaceChildren(...nodes) {
      node.children = [...nodes];
    },
    setAttribute(name, value) {
      node.attributes.set(name, value);
    },
    removeAttribute(name) {
      node.attributes.delete(name);
    },
    getAttribute(name) {
      return node.attributes.get(name) ?? null;
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

// findIcon walks the button subtree for the .cf-refresh-icon span.
function findIcon(root: FakeEl): FakeEl {
  const stack = [...root.children];
  while (stack.length > 0) {
    const node = stack.shift()!;
    if (node.className === "cf-refresh-icon") return node;
    stack.unshift(...node.children);
  }
  throw new Error("icon span not found");
}

// stubDom installs the fake document + matchMedia globals refresh.ts uses.
function stubDom(reduced: boolean) {
  vi.stubGlobal("document", { createElement: (tag: string) => fakeEl(tag) as unknown });
  vi.stubGlobal("matchMedia", vi.fn(() => ({ matches: reduced })));
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

describe("refreshButton", () => {
  it("renders an icon-only button labelled for assistive tech", () => {
    stubDom(false);
    const btn = refreshButton(async () => {}) as unknown as FakeEl;
    expect(btn.className).toContain("cf-refresh-btn");
    expect(btn.getAttribute("aria-label")).toBe("Refresh data");
    expect(btn.title).toBe("Refresh");
    expect(btn.type).toBe("button");
    expect(btn.textContent).not.toContain("Refresh"); // icon-only, no text label
    const icon = findIcon(btn);
    expect(icon.innerHTML).toContain("polyline"); // the circular-arrow SVG
  });

  it("busy state: click disables the button and sets aria-busy until the refresh resolves", async () => {
    stubDom(false);
    vi.useFakeTimers();
    let resolveRefresh: () => void = () => {};
    const onRefresh = vi.fn(() => new Promise<void>((res) => { resolveRefresh = res; }));
    const btn = refreshButton(onRefresh) as unknown as FakeEl;
    await btn._click();
    expect(onRefresh).toHaveBeenCalledTimes(1);
    expect(btn.disabled).toBe(true);
    expect(btn.getAttribute("aria-busy")).toBe("true");
    expect(btn.classList.contains("cf-refresh-busy")).toBe(true);
    resolveRefresh();
    await Promise.resolve();
    await Promise.resolve();
    // The success check beat keeps the button busy until the arrow returns.
    expect(btn.getAttribute("aria-busy")).toBe("true");
    vi.advanceTimersByTime(900);
    expect(btn.getAttribute("aria-busy")).toBeNull();
    expect(btn.disabled).toBe(false);
    expect(btn.classList.contains("cf-refresh-busy")).toBe(false);
  });

  it("success path: the icon swaps to a check for 900ms, then back to the arrow", async () => {
    stubDom(false);
    vi.useFakeTimers();
    let resolveRefresh: () => void = () => {};
    const onRefresh = () => new Promise<void>((res) => { resolveRefresh = res; });
    const btn = refreshButton(onRefresh) as unknown as FakeEl;
    await btn._click();
    resolveRefresh();
    await Promise.resolve();
    await Promise.resolve();
    const icon = findIcon(btn);
    expect(icon.innerHTML).toContain('points="20 6 9 17 4 12"'); // check glyph
    vi.advanceTimersByTime(899);
    expect(icon.innerHTML).toContain('points="20 6 9 17 4 12"'); // still the check at 899ms
    vi.advanceTimersByTime(1);
    expect(icon.innerHTML).toContain('points="23 4 23 10 17 10"'); // arrow back
    expect(btn.disabled).toBe(false);
  });

  it("failure path: icon shakes, the tooltip carries the error, button re-arms after 240ms", async () => {
    stubDom(false);
    vi.useFakeTimers();
    let rejectRefresh: (err: unknown) => void = () => {};
    const onRefresh = () => new Promise<void>((_res, rej) => { rejectRefresh = rej; });
    const btn = refreshButton(onRefresh) as unknown as FakeEl;
    await btn._click();
    rejectRefresh(new Error("HTTP 503"));
    await Promise.resolve();
    await Promise.resolve();
    expect(btn.title).toBe("Refresh failed — HTTP 503");
    expect(btn.classList.contains("cf-refresh-shake")).toBe(true);
    vi.advanceTimersByTime(240);
    expect(btn.classList.contains("cf-refresh-shake")).toBe(false);
    expect(btn.disabled).toBe(false);
  });

  it("reduced motion failure: no shake class, tooltip carries the error, button re-arms", async () => {
    stubDom(true);
    vi.useFakeTimers();
    let rejectRefresh: (err: unknown) => void = () => {};
    const onRefresh = () => new Promise<void>((_res, rej) => { rejectRefresh = rej; });
    const btn = refreshButton(onRefresh) as unknown as FakeEl;
    await btn._click();
    rejectRefresh(new Error("HTTP 503"));
    await Promise.resolve();
    await Promise.resolve();
    expect(btn.title).toBe("Refresh failed — HTTP 503");
    expect(btn.classList.contains("cf-refresh-shake")).toBe(false);
    expect(btn.getAttribute("aria-busy")).toBeNull();
    expect(btn.disabled).toBe(false);
  });

  it("reduced motion busy: static ellipsis instead of spin, arrow restored after success", async () => {
    stubDom(true);
    vi.useFakeTimers();
    let resolveRefresh: () => void = () => {};
    const onRefresh = () => new Promise<void>((res) => { resolveRefresh = res; });
    const btn = refreshButton(onRefresh) as unknown as FakeEl;
    await btn._click();
    expect(findIcon(btn).textContent).toBe("…");
    expect(btn.classList.contains("cf-refresh-busy")).toBe(false);
    resolveRefresh();
    await Promise.resolve();
    await Promise.resolve();
    expect(findIcon(btn).textContent).toBe("");
    expect(findIcon(btn).innerHTML).toContain("polyline"); // arrow restored
    expect(btn.disabled).toBe(false);
  });
});

describe("fadeSwap", () => {
  it("swaps the content of the old node and fades the new node in", () => {
    stubDom(false);
    const hostFake = fakeEl("div");
    hostFake.append(fakeEl("div")); // the old child being replaced
    const host = hostFake as unknown as HTMLElement;
    const fresh = fakeEl("div") as unknown as HTMLElement;
    fadeSwap(host, fresh);
    expect(host.children).toEqual([fresh as unknown as FakeEl]);
    expect(fresh.classList.contains("cf-fade-swap")).toBe(true);
    expect(fresh.style.opacity).toBe("1");
  });

  it("removes the fade class and inline opacity after the transition", () => {
    stubDom(false);
    vi.useFakeTimers();
    const host = fakeEl("div") as unknown as HTMLElement;
    const fresh = fakeEl("div") as unknown as HTMLElement;
    fadeSwap(host, fresh);
    expect(fresh.classList.contains("cf-fade-swap")).toBe(true);
    vi.advanceTimersByTime(200);
    expect(fresh.classList.contains("cf-fade-swap")).toBe(false);
    expect(fresh.style.opacity).toBe("");
  });
});
