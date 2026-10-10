// Shared view components (IMP-pEY98JH wave 1, shipped with the Durable
// Objects section): the patterns every view repeats — status bar + refresh,
// section cards, key/value field rows — extracted once so the whole app
// stays visually identical by construction.

import { el, statusLine } from "./dom";
import { totalAgeSec, type FetchResult } from "./api";
import { refreshButton } from "./refresh";

/** The per-view status bar: honest age (client + server cache) on the left,
 *  the refresh icon button on the right. `onRefresh` should re-render the
 *  view with refresh:true (the failed-refresh rethrow keeps good DOM). */
export function dashBar<T>(res: FetchResult<T>, cache: { ageSec: number; stale: boolean } | undefined, onRefresh: () => void | Promise<void>): HTMLElement {
  const bar = el("div", "cf-dash-bar");
  bar.append(
    statusLine(totalAgeSec(res.ageSec, cache), { cached: Boolean(cache?.stale), demo: res.demo }),
    refreshButton(async () => {
      await onRefresh();
    }),
  );
  return bar;
}

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
