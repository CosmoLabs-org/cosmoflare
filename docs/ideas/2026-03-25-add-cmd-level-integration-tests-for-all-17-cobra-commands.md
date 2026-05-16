---
id: IDEA-MN5GE5AY
title: Add cmd-level integration tests for all 17 Cobra commands
created: "2026-03-25T03:57:48.634487+01:00"
status: harvested
source: agent
origin:
    session: 17
    trigger: audit-code-quality-test-coverage
tags:
    - audit
    - testing
---


# Add cmd-level integration tests for all 17 Cobra commands

The cmd/ package has 17 command files but zero test files. Commands are not tested end-to-end. Create a test harness that executes commands with mock API client and verifies output format (human and JSON), flag parsing, and error handling.
