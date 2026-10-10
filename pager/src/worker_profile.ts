// Worker profile page (UI-3, operator request 2026-10-10): tapping a worker
// in the Workers table opens its own view — the 24h analytics row from
// /api/summary, exactly as the table shows it, plus a link out to the
// Cloudflare dashboard. No new endpoints, no invented numbers.

import { el, skeleton } from "./dom";
import { formatCount, formatPct } from "./format";
import { sectionCard, fieldRow } from "./components";
import { api, LoginExpiredError, type FetchResult, type Summary, type WorkerRow } from "./api";
import { levelForErrorPct } from "./dashboard";

/** Fetch shape for the profile's one summary call — injectable so tests can
 *  pass a stub instead of the shared ApiClient singleton. */
export type SummaryFetcher = (endpoint: string, opts?: { refresh?: boolean }) => Promise<FetchResult<Summary>>;

/** "12.5ms" from a CPU percentile, "—" when the analytics row has no data. */
function cpuMs(v: number | null): string {
  return v === null ? "—" : `${v.toFixed(1)}ms`;
}

// The one outbound link a profile carries: the worker's own page in the
// Cloudflare dashboard (same deep-link shape the rest of the ops tooling uses).
function dashboardLink(name: string): HTMLElement {
  const a = el("a", "cf-link", "Open in Cloudflare →");
  a.href = `https://dash.cloudflare.com/?to=/:account/workers/services/view/${encodeURIComponent(name)}`;
  a.target = "_blank";
  a.rel = "noreferrer";
  return a;
}

// paintProfile builds the two cards for a found (or missing) worker — split
// out so the fetch/error plumbing below stays readable.
function paintProfile(name: string, w: WorkerRow | undefined): HTMLElement {
  const stats = w
    ? sectionCard(name, "24h window · from Workers Analytics",
      statRow("Requests", formatCount(w.requests), ""),
      statRow("Errors", formatCount(w.errors), w.errors === 0 ? "cf-ok-text" : "cf-level-critical-text"),
      statRow("Error rate", formatPct(w.errorPct, 2), `cf-level-${levelForErrorPct(w.errorPct)}-text`),
      statRow("CPU p50", cpuMs(w.cpuP50Ms), ""),
      statRow("CPU p99", cpuMs(w.cpuP99Ms), ""),
    )
    : sectionCard("Worker not found", undefined,
      el("p", "cf-empty cf-empty-quiet", `No 24h analytics row for ${name}.`),
    );
  // A data absence is not an error — the same quiet tone either way.
  const about = sectionCard("About this worker", undefined,
    el("p", "cf-empty cf-empty-quiet", "Profiles currently surface the 24h analytics row — the same numbers the Workers table shows."),
    dashboardLink(name),
  );
  const back = el("a", "cf-btn cf-btn-ghost cf-profile-back", "All workers");
  back.href = "#/workers";
  const view = el("div", "cf-profile");
  view.append(back, stats, about);
  return view;
}

// statRow is the profile's visual indicator row (operator 2026-10-10: better
// indicators for the numbers): the value renders in the data face, larger and
// tabular, with an optional level tint — zero errors reads green ("clean"),
// the error rate carries its threshold color, counts stay neutral.
function statRow(label: string, value: string, valueClass: string): HTMLElement {
  const row = fieldRow(label, value);
  const v = row.children[row.children.length - 1] as HTMLElement;
  v.classList.add("cf-stat-value");
  if (valueClass) v.classList.add(valueClass);
  return row;
}

export async function renderWorkerProfile(root: HTMLElement, name: string, opts: { refresh?: boolean; fetcher?: SummaryFetcher } = {}): Promise<void> {
  // The Retry re-render must go through the SAME fetcher (injected or the
  // shared client) — hence the closure rather than a bare recursive call.
  const fetchSummary: SummaryFetcher = opts.fetcher ?? ((endpoint, o) => api.fetchJson<Summary>(endpoint, o));
  const load = async (loadOpts: { refresh?: boolean } = {}): Promise<void> => {
    if (!loadOpts.refresh) root.replaceChildren(skeleton());
    try {
      const res = await fetchSummary("api/summary", { refresh: loadOpts.refresh });
      const w = res.data.workers.find((row) => row.script === name);
      root.replaceChildren(paintProfile(name, w));
    } catch (err) {
      // A failed refresh rethrows so the clicked Refresh button keeps its node
      // and surfaces the error itself (the shared UI-1 convention); only a
      // first (skeleton) load replaces the view with an error card.
      if (loadOpts.refresh) throw err;
      if (err instanceof LoginExpiredError) {
        root.replaceChildren(el("p", "cf-empty cf-login-expired", err.message));
        return;
      }
      const box = el("section", "cf-card cf-level-warning");
      box.append(
        el("h2", undefined, "Could not load the worker profile"),
        el("p", "cf-row-detail", err instanceof Error ? err.message : String(err)),
      );
      const retry = el("button", "cf-btn", "Retry");
      retry.addEventListener("click", () => void load());
      box.append(retry);
      root.replaceChildren(box);
    }
  };
  await load({ refresh: opts.refresh });
}
