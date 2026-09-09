---
ulid: 01M21S2EKEEB8F1B0PDAKME0R5
title: Alert condition descriptor table — one registry for names, units, help
created: "2026-09-09T04:28:48.110506+04:00"
status: harvested
source: agent
origin:
    session: 45
    trigger: session-end altitude review of limits diff
promoted_to: FEAT-015
---

# Alert condition descriptor table — one registry for names, units, help

# Alert condition descriptor table — one registry for names, units, help

The alert condition enum lives in 5+ places: validAlertConditions (pkg/cosmoflare/alerts.go), two error-message lists, the AlertRule comment, cmd/alerts.go help strings (x3), and the evaluator switch (internal/webhook/evaluator.go conditionValue). The limits feature extended exactly one of them and the session-end review caught the CLI rejecting the new conditions. A descriptor table (name -> unit, availability rule, data key, service tag) next to AlertRule would derive the registry, error lists, and help text from one source. Same pattern applies to limits resource names (workers.scripts etc. appear in tables, rows, and the evaluator switch — a rename silently drops the metric).
