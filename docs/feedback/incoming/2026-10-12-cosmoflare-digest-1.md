---
kind: digest
ulid: 01M4PK06HY12WRAQSF2T5X4R2Z
from: cosmoflare
to_project: cosmoflare
created: "2026-10-12T02:58:09.597997+04:00"
---

# Feedback digest — cosmoflare (2026-10-12)

| title | type | severity | one-line |
| --- | --- | --- | --- |
| Operator UI pass 2026-10-11 (wave 2): section color identity, heading counts, circular refresh | feature | medium | # Operator UI pass 2026-10-11 (wave 2): section color identity, heading counts, … |
| Pager domain detail cache-key collision + drill-down scroll (suspected) | bug | medium | # Pager domain detail cache-key collision + drill-down scroll (suspected) |

## 01M4M4PKPME6DJNVH3MC2YX6QM — Operator UI pass 2026-10-11 (wave 2): section color identity, heading counts, circular refresh

# Operator UI pass 2026-10-11 (wave 2): section color identity, heading counts, circular refresh

Operator requests mid-session 2037, all implemented same-session: (1) confirm every domain has its own profile page (DomainProfile exists since P-05b — verified live); (2) move section totals into the masthead heading as "Domains (43)" instead of a count line under the title; (3) per-section accent color + smooth breathing micro-animation on the selected item in the global nav (all variants), section-colored masthead icon; (4) circular refresh button with grow-shrink breathing while busy, spring press feedback. Taste constraints per ops-ui-taste memory hold: no left borders/tile outlines, impeccable detect must stay 0, reduced-motion respected.

## 01M4M7571W8KQVTCZFG5MMRAKE — Pager domain detail cache-key collision + drill-down scroll (suspected)

# Pager domain detail cache-key collision + drill-down scroll (suspected)

Suspected client-side defect found by code reading during the 2026-10-11 domain-tap investigation (repro harness built at /tmp/cfscratch, run pending operator word): the pager ApiClient stores every response under keyOf(endpoint) which DROPS the query string — so api/domains/detail?id=A and ?id=B share ONE cache key 'api/domains/detail'. Within the 60s fresh window, opening domain B right after domain A renders B's profile header (from list data) but A's ENRICHMENT fields (original nameservers, DNSSEC, SSL mode, activation). Fix direction: include the full endpoint including query in the memory/session cache key (or key detail fetches by zone id). Same investigation note: the master-detail drill-down does not scroll to top (ViewHost scrolls only on route change), so tapping a domain deep in a long list can look like nothing happened — likely the 'app gets stuck' report; fix by scrolling the profile into view on activeId change. Both need live confirmation via the harness before fixing.

