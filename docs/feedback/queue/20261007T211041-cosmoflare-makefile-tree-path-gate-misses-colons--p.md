---
ulid: 01M4BNH2KC8MAKQPEQHK9090Q5
from: cosmoflare
type: bug
severity: medium
title: Makefile tree-path gate misses colons — proxy.golang.org zip poisoning class
created: "2026-10-07T21:10:41.00484+04:00"
---

# Makefile tree-path gate misses colons — proxy.golang.org zip poisoning class

What happened: v0.32.0's module zip was rejected by proxy.golang.org because a tracked filename contained colons (+04:00 timestamp in a feedback digest filename); the release tree-path gate (BUG-052, Makefile release target) scans for non-ASCII paths but not ':' characters.

Why it matters: any release cut with such a file ships a broken `go install @latest` path until a re-tag — v0.32.0 needed a filename rename plus tag move (delete-then-push) inside the release window.

Proposed fix: extend the gate's ls-files scan to reject non-ASCII OR ':' in tracked paths (colons are invalid in module zips per proxy zip rules); additionally normalize ccs digest filenames to colon-free timestamps at creation time.

Evidence: fix commit a76932c; proxy error text "create zip: ... malformed file path"; tag re-pushed 2026-10-07; proxy confirmed serving @latest v0.32.0 after.
