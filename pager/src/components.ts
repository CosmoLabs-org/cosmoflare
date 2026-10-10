// Shared view components (IMP-pEY98JH wave 1, shipped with the Durable
// Objects section): the patterns every view repeats — section cards,
// key/value field rows — extracted once so the whole app stays visually
// identical by construction. The per-view status bar + refresh (dashBar)
// was removed by UI-1: the refresh button moved to the top bar and the
// age line to the global bottom status strip (pager/src/statusstrip.ts).

import { el } from "./dom";

/** A section card with its heading (and optional one-line subtitle) — the
 *  app's standard content grouping. */
export function sectionCard(title: string, subtitle: string | undefined, ...children: HTMLElement[]): HTMLElement {
  const card = el("section", "cf-card");
  card.append(el("h2", undefined, title));
  if (subtitle) card.append(el("p", "cf-row-detail", subtitle));
  card.append(...children);
  return card;
}

/** A key/value row — label muted left, value right. `value` renders with
 *  pre-line so multi-line strings (nameservers) stack. */
export function fieldRow(label: string, value: string): HTMLElement {
  const f = el("div", "cf-domain-field");
  f.append(el("span", "cf-domain-field-label", label), el("span", "cf-domain-field-value", value));
  return f;
}
