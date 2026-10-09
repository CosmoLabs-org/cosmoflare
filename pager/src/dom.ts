// Small DOM helpers shared by the Ops views (FEAT-052).

import { formatAge } from "./format";

export function el<K extends keyof HTMLElementTagNameMap>(tag: K, className?: string, text?: string): HTMLElementTagNameMap[K] {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

/** Skeleton placeholder block shown while a view waits on its first fetch. */
export function skeleton(): HTMLElement {
  const host = el("div", "cf-loading");
  for (let i = 0; i < 4; i++) host.append(el("div", "cf-skeleton"));
  return host;
}

/** "Updated 3m ago · cached" / "Updated just now" / "demo data" strip. */
export function statusLine(ageSec: number, opts: { cached?: boolean; demo?: boolean; source?: string } = {}): HTMLElement {
  const line = el("p", "cf-status");
  const label = ageSec < 60 ? "Updated just now" : `Updated ${formatAge(ageSec)}`;
  const bits = [label];
  if (opts.cached) bits.push("cached");
  if (opts.demo) bits.push("demo data");
  line.textContent = bits.join(" · ");
  return line;
}
