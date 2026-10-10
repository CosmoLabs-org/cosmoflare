---
ulid: 01M4E2BEHWYPCBNGJKZPDN5VXR
title: Baseline anomaly detection for Workers/DO usage (model B)
created: "2026-10-08T19:33:16.988086+04:00"
status: harvested
source: agent
origin:
    session: 67
    trigger: 'session 67 brainstorm: operator chose thresholds-now, baselines-later'
promoted_to: FEAT-p0QB1KX
---

# Baseline anomaly detection for Workers/DO usage (model B)

# Baseline anomaly detection for Workers/DO usage (model B)

Learn each script's rolling 7-day normal (requests, CPU, errors, DO invocations) and alert on deviation ('worker X at 8x its usual CPU'). Builds on wave-1 infrastructure: per-script analytics rows + evaluator window state become the time-series seed. Storage: local rolling history file or SQLite under ~/.cosmoflare. Deliverable: a 'deviation' condition type in alertConditionRegistry sitting beside threshold conditions.
