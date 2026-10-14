---
ulid: 01M4M5Y0RK3B5G9A3A3W098734
title: WAF starter-rules guide in the Cosmoflare guides section
created: "2026-10-11T04:31:17.779115+04:00"
status: harvested
source: human
origin:
    session: 75
promoted_to: FEAT-pAGY842
---

# WAF starter-rules guide in the Cosmoflare guides section

# WAF starter-rules guide in the Cosmoflare guides section

# WAF starter-rules guide in the Cosmoflare guides section

# WAF starter-rules guide in the Cosmoflare guides section

Extend the planned Guides section (idea from the BUG-056 closeout, 2026-10-11) with a practical 'Cloudflare WAF starter rules' guide: the 6-10 highest-value custom rules for small teams (condition / action / why / rollout order), when to rely on managed rules vs custom rules, three rate-limiting starter patterns (login paths, API abuse, scrapers) with safe thresholds, and a log-first rollout sequence tied to security-event analytics. Agent 0399 (2026-10-11) is drafting the research skeleton against live Cloudflare docs; the guide lands as docs/guides/ content with every number citing developers.cloudflare.com.

---

RESEARCH 2026-10-11 (agents 0398/0399): every number in the guide must cite live doc URLs — the old limits pages 404 now (docs restructured; successors at /waf/custom-rules/, /waf/rate-limiting-rules/, /waf/tools/lists/); note legacy firewall rules are deprecated upstream and the guide should teach custom rules only, with one migration note. Agent 0399's starter-rules skeleton (6-10 rules with condition/action/why/rollout, managed-vs-custom guidance, 3 rate-limit patterns, log-first rollout) is the guide's core.

---

Draft skeleton landed at docs/research/2026-10-11-waf-starter-rules-guide-draft.md (e660b66) — 7 starter rules with condition/action/why/rollout, managed-vs-custom section, 3 rate-limit patterns, log-first rollout sequence; all numbers cited to live doc URLs.
