// Projects view (FEAT-p0QKZT0): one section for the account's projects and
// what each consumes across Cloudflare. Pure presentation over the existing
// /api/billing payload — projects[] carries each project's projected
// overage and driver product ids; the products' topConsumers carry the
// per-project attribution detail. Zero new upstream calls.

import { el, skeleton, statusLine } from "./dom";
import { formatUsd } from "./format";
import { api, LoginExpiredError, totalAgeSec, type Billing, type FetchResult } from "./api";
import { productLabel } from "./gauges";
import { refreshButton } from "./refresh";

/** Worst payer first: overage desc, then name for a stable order. */
export function sortProjectsByOverage<T extends { project: string; projectedOverageUsd: number }>(projects: T[]): T[] {
  return [...projects].sort((a, b) => b.projectedOverageUsd - a.projectedOverageUsd || a.project.localeCompare(b.project));
}

/** One project's consumers across products: name, what it uses, its share. */
export interface ProjectConsumer {
  productName: string;
  name: string;
  used: number;
  unit: string;
  share: number;
}

export function projectConsumers(b: Billing, project: string): ProjectConsumer[] {
  const out: ProjectConsumer[] = [];
  for (const p of b.products) {
    for (const c of p.topConsumers) {
      if (c.project === project) {
        out.push({ productName: productLabel(p.id, p.product, p.metric), name: c.name, used: c.used, unit: p.unit, share: c.share });
      }
    }
  }
  // Biggest share first; ties by consumer name.
  return out.sort((a, b) => b.share - a.share || a.name.localeCompare(b.name));
}

function errorState(root: HTMLElement, err: unknown): void {
  if (err instanceof LoginExpiredError) {
    root.replaceChildren(el("p", "cf-empty cf-login-expired", err.message));
    return;
  }
  const box = el("section", "cf-card cf-level-warning");
  box.append(
    el("h2", undefined, "Could not load projects"),
    el("p", "cf-row-detail", err instanceof Error ? err.message : String(err)),
  );
  const retry = el("button", "cf-btn", "Retry");
  retry.addEventListener("click", () => void renderProjects(root));
  box.append(retry);
  root.replaceChildren(box);
}

function projectCard(b: Billing, p: Billing["projects"][number]): HTMLElement {
  const card = el("section", `cf-card cf-project ${p.projectedOverageUsd > 0 ? "cf-over" : ""}`);
  const head = el("div", "cf-project-head");
  head.append(
    el("strong", "cf-project-name", p.project || "(no project)"),
    el("span", `cf-project-overage ${p.projectedOverageUsd > 0 ? "cf-level-critical-text" : "cf-ok-text"}`,
      p.projectedOverageUsd > 0 ? `+${formatUsd(p.projectedOverageUsd)} overage` : "no overage"),
  );
  card.append(head);

  // Driver products as muted chips: what this project's cost comes from.
  if (p.drivers.length > 0) {
    const chips = el("div", "cf-chips");
    for (const id of p.drivers) {
      const prod = b.products.find((x) => x.id === id);
      chips.append(el("span", "cf-chip", prod ? productLabel(prod.id, prod.product, prod.metric) : id));
    }
    card.append(chips);
  }

  const consumers = projectConsumers(b, p.project);
  if (consumers.length > 0) {
    const details = el("details", "cf-consumers");
    details.append(el("summary", "cf-consumers-summary", `Top consumers (${consumers.length})`));
    const list = el("ul", "cf-consumers-list");
    for (const c of consumers) {
      const li = el("li", "cf-consumer");
      li.append(
        el("span", "cf-consumer-name", c.name),
        el("span", "cf-consumer-project", c.productName),
        el("span", "cf-consumer-share", `${Math.round(c.share * 100)}%`),
      );
      list.append(li);
    }
    details.append(list);
    card.append(details);
  }
  return card;
}

export async function renderProjects(root: HTMLElement, opts: { refresh?: boolean } = {}): Promise<void> {
  if (!opts.refresh) root.replaceChildren(skeleton());
  try {
    const res = await api.fetchJson<Billing>("api/billing", {
      refresh: opts.refresh,
      // Background revalidation repaints with the fresh copy (BUG-057
      // defect 2) — this view has no unsaved-edit state to protect.
      onRevalidate: (fresh) => renderProjectsInto(root, { data: fresh, source: "network", ageSec: 0, demo: false }),
    });
    renderProjectsInto(root, res);
  } catch (err) {
    if (opts.refresh) throw err;
    errorState(root, err);
  }
}

function renderProjectsInto(root: HTMLElement, res: FetchResult<Billing>): void {
  const b = res.data;
  const bar = el("div", "cf-dash-bar");
  const line = statusLine(totalAgeSec(res.ageSec, b.cache), { cached: Boolean(b.cache?.stale), demo: res.demo });
  const refresh = refreshButton(() => renderProjects(root, { refresh: true }));
  bar.append(line, refresh);

  const heading = el("h2", undefined, "Projects");
  const sub = el("p", "cf-row-detail", "Projected cost beyond allowances, by project, across Cloudflare products.");

  const projects = sortProjectsByOverage(b.projects);
  const list = el("div", "cf-projects-list");
  if (projects.length === 0) {
    list.append(el("p", "cf-empty cf-empty-quiet", "No projects found on this account."));
  } else {
    for (const p of projects) list.append(projectCard(b, p));
  }
  root.replaceChildren(bar, heading, sub, list);
}
