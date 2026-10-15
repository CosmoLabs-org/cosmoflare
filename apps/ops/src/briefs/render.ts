// Agent-brief renderer (P-16, FEAT-pC1EV32): the pure Brief -> text pass.
// No clock, no env — generatedAt lives in the API envelope, never in the
// rendered text; that is what makes the golden tests byte-stable (identical
// Briefs render byte-identical strings). Format (brainstorm §3):
//
//   # <zone> — Cloudflare optimization directive
//
//   ## Problem            — one "- " bullet per finding, evidence inline
//   ## Actions            — numbered "1." lines (command backticked when set)
//   ## Verify             — "- " bulleted acceptance commands
//   ## Context            — Zone <name> (<id>) — measured <window> by CosmoLabs Ops.

import type { Brief } from "./types";

export function renderAgentBrief(b: Brief): string {
  const lines: string[] = [`# ${b.zone} — Cloudflare optimization directive`, "", "## Problem"];
  for (const f of b.findings) lines.push(`- [${f.severity}] ${f.kind}: ${f.evidence}`);
  lines.push("", "## Actions");
  b.actions.forEach((a, i) => lines.push(`${i + 1}. ${a.directive}${a.command ? ` — \`${a.command}\`` : ""}`));
  lines.push("", "## Verify");
  for (const v of b.verify) lines.push(`- ${v}`);
  lines.push("", "## Context", `Zone ${b.zone} (${b.zoneId}) — measured ${b.window} by CosmoLabs Ops.`);
  return lines.join("\n");
}
