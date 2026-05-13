---
id: IDEA-016
title: Fix download progress bar -- ProgressWriter never renders to terminal
created: "2026-05-13T15:55:33.638196-03:00"
status: seed
source: human
origin:
    session: 2027
---

# Fix download progress bar -- ProgressWriter never renders to terminal

The download path in cmd/object.go wraps the file with ProgressWriter which tracks bytes, but FormatBar() is never called during io.Copy. The user sees 'Downloading: 42MB' then a blank line until completion. Need a ticker goroutine or reader wrapper that periodically prints the progress bar, similar to how the upload path uses WithProgressCallback. Found during code review of the Phase 2 session. Affects cmd/object.go:367-377.
