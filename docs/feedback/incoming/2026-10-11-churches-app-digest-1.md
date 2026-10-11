---
kind: digest
ulid: 01M4M76VYZM2H02JD48VX92B8X
from: Churches-app
to_project: cosmoflare
created: "2026-10-11T04:53:36.348305+04:00"
---

# Feedback digest — cosmoflare (2026-10-11)

| title | type | severity | one-line |
| --- | --- | --- | --- |
| Standard: scanner-block WAF rule for every proxied zone (from Churches-app TASK-065) | feature | medium | # Standard: scanner-block WAF rule for every proxied zone (from Churches-app TAS… |

## 01M4M5STWWBR6W1DWVPY6NV5PT — Standard: scanner-block WAF rule for every proxied zone (from Churches-app TASK-065)

# Standard: scanner-block WAF rule for every proxied zone (from Churches-app TASK-065)

Proposal from Churches-app deployment 2026-10-11 (TASK-065): promote the
scanner-block WAF rule to a CosmoFlare standard for every proxied zone.

What shipped on churches.app (zone d27a79b528f037bb60e9eba75fd45912):
ONE custom rule, action block, expression:
(http.request.uri.path contains ".php") or (http.request.uri.path contains "wp-")
or (http.request.uri.path contains ".env") or (http.request.uri.path contains ".git")

Effect: scanner probes (the /txets.php, /wordpress/, /wp/ traffic the
2026-10-08 CF efficiency audit logged) die at the edge instead of reaching
the Worker — each was a billed invocation returning 404. Rule created via
the Rulesets API with the project's env token (zone WAF scope suffices).

Standard shape for the audit dimension:
1. Every proxied zone carries the scanner-block rule (Free tier: contains
   matchers only, no regex).
2. BEFORE applying: grep the project's route/slug space for collisions with
   the four patterns (Churches-app checked 60k slugs — zero hits; a
   WordPress-adjacent project WOULD collide on wp-).
3. Limits to respect: 5 rules/zone on Free, 20 Pro, 100 Business.
4. Companion (optional, needs Cache Rules scope the standard token lacks):
   edge-cache rule for static /geojson/*-style .json paths — Cloudflare's
   default cache-extension list excludes .json, so origin TTL headers alone
   leave them DYNAMIC at the edge.

Suggest CosmoFlare adds this to the audit's remediation playbook and the
project-setup checklist.

