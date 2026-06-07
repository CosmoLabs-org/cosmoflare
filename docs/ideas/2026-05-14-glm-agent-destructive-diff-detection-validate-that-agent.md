---
id: IDEA-019
title: GLM agent destructive diff detection — validate that agent diffs don't remove existing features (JSON output, progress bars, error handling)
created: "2026-05-14T23:56:27.716592-03:00"
status: harvested
source: human
origin:
    session: 2027
---





# GLM agent destructive diff detection — validate that agent diffs don't remove existing features (JSON output, progress bars, error handling)

Session 2026-05-14: 4/4 GLM agents produced destructive diffs that removed working features. The ccs glm-agent validation should detect net-negative diffs or removal of known patterns (printJSON, printSuccessJSON, etc.).
