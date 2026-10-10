// Durable Objects view (FEAT-p3KJAC2): every namespace with its 24h usage —
// requests, errors, error rate — as a sortable table with graded usage bars
// against the DO request allowance. Built on the shared components.

import { el, skeleton } from "./dom";
import { formatCount, formatPct } from "./format";
import { api, LoginExpiredError, type Billing, type FetchResult } from "./api";
import { renderSortableTable, usageSharePct, type Column } from "./tables";
import { levelForErrorPct } from "./dashboard";
import { dashBar, sectionCard } from "./components";

export interface DORow {
  namespaceId: string;
  script: string;
  requests: number;
  errors: number;
  errorPct: number;
}

export interface DOPayload {
  generatedAt: string;
  namespaces: number;
  rows: DORow[];
  errors: string[];
  cache?: { ageSec: number; stale: boolean };
}

function errorState(root: HTMLElement, err: unknown): void {
  if (err instanceof LoginExpiredError) {
    root.replaceChildren(el("p", "cf-empty cf-login-expired", err.message));
    return;
  }
  const retry = el("button", "cf-btn", "Retry");
  retry.addEventListener("click", () => void renderDurableObjects(root));
  const box = el("section", "cf-card cf-level-warning");
  box.append(el("h2", undefined, "Could not load Durable Objects"), el("p", "cf-row-detail", err instanceof Error ? err.message : String(err)), retry);
  root.replaceChildren(box);
}

export async function renderDurableObjects(root: HTMLElement, opts: { refresh?: boolean } = {}): Promise<void> {
  if (!opts.refresh) root.replaceChildren(skeleton());
  try {
    const res = await api.fetchJson<DOPayload>("api/durable-objects", {
      refresh: opts.refresh,
      onRevalidate: (fresh) => renderInto(root, { data: fresh, source: "network", ageSec: 0, demo: false }),
    });
    renderInto(root, res);
  } catch (err) {
    if (opts.refresh) throw err;
    errorState(root, err);
  }
}

function renderInto(root: HTMLElement, res: FetchResult<DOPayload>): void {
  const d = res.data;
  void (async () => {
    // Graded bars: this script's requests as a share of the DO request
    // allowance from the cached billing payload; bars drop out on failure.
    let included = 0;
    try {
      const b = await api.fetchJson<Billing>("api/billing");
      included = b.data.products.find((p) => p.id === "durable_objects.requests")?.included ?? 0;
    } catch {
      included = 0;
    }

    root.replaceChildren(
      dashBar(res, d.cache, () => renderDurableObjects(root, { refresh: true })),
      sectionCard("Durable Objects (24h)",
        d.errors.length > 0
          ? `Partial load — ${d.errors[0]}`
          : `${d.namespaces} ${d.namespaces === 1 ? "namespace" : "namespaces"} · requests, errors and error rate per script`),
    );

    if (d.rows.length === 0) {
      root.append(sectionCard("No Durable Objects", undefined, el("p", "cf-empty cf-empty-quiet", "No namespaces or traffic found.")));
      return;
    }

    const columns: Column<DORow>[] = [
      { key: "script", label: "Script", text: (r) => r.script },
      { key: "requests", label: "Requests", numeric: true, text: (r) => formatCount(r.requests), num: (r) => r.requests },
      { key: "errors", label: "Errors", numeric: true, text: (r) => formatCount(r.errors), num: (r) => r.errors },
      { key: "errorPct", label: "Error %", numeric: true, text: (r) => formatPct(r.errorPct, 2), num: (r) => r.errorPct },
    ];
    const card = el("section", "cf-card");
    renderSortableTable(card, {
      columns,
      rows: d.rows,
      persistKey: root,
      sortPrefId: "durable-objects",
      initial: { key: "requests", dir: "desc" },
      usageBar: included > 0 ? (r) => ({ pctOfAllowance: usageSharePct(r.requests, included) }) : undefined,
      rowClass: (r) => {
        const lv = levelForErrorPct(r.errorPct);
        return lv === "critical" ? "cf-level-critical" : lv === "warning" ? "cf-level-warning" : "";
      },
    });
    root.append(card);
  })();
}
