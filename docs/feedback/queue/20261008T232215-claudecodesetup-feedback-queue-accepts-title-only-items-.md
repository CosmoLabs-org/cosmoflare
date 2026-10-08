---
ulid: 01M4EFEQ4PH1879K77EQQ1ZZ14
from: cosmoflare
type: bug
severity: medium
title: feedback queue accepts title-only items — body silently empty at receiver
created: "2026-10-08T23:22:15.574218+04:00"
---

# feedback queue accepts title-only items — body silently empty at receiver

**What happened:** `ccs feedback queue` accepted a high-severity feature item with a title and no body. Churches-app queued ULID 01M4E5W5T97WA3KWGKB1RKHS9B ("CF efficiency audit dimension — caching, D1 rows-read, edge headers") on 2026-10-08. The queue file `Churches-app/docs/feedback/queue/sent/20261008T203450-cosmoflare-cf-efficiency-audit-dimension--caching-d.md` holds only frontmatter and an H1. Cosmoflare received an empty item (now FB-11).

**Why it matters:** The real spec (header sweep, repeat-probe, D1 rows-read, KV hit rate, unused bindings) lived only in `Churches-app/docs/audits/2026-10-08-cf-efficiency-audit.md`. The receiver had to search a foreign repo to recover it. A receiver without that access would convert an empty item or reject it as junk. The send-time quality bar (what/why/proposed-fix/priority, never shorter than 5 lines) is not enforced on the queue path.

**Proposed fix:** Make `ccs feedback queue` refuse an item with no `--body-file` (or an empty one) unless `--bare` is given, matching `ccs feedback send`. Optionally accept `--ref <path>` and inline the referenced doc's path into the body so the receiver knows where the spec lives.

**Priority:** medium — silent data loss on the default feedback path, but recoverable by hand.
