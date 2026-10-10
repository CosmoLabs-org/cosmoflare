// Projects helpers (FEAT-p0QKZT0): the pure layer behind the React projects
// view — worst-payer-first sorting and per-project consumer attribution over
// the existing /api/billing payload. The imperative vanilla renderer that
// used to live here was retired (TASK-pD575FN);
// pager/src/app/views/ProjectsView.tsx is the only render path now.

import type { Billing } from "./api";
import { productLabel } from "./gauges";

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
