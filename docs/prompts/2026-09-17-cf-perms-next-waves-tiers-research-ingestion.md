---
title: "Cross-AI Research Ingestion — CF permissions, next waves, paid tiers"
created: 2026-09-17
status: PENDING
branch: master
goals_total: 5
goals_completed: 0
requires_reading:
    - docs/research/2026-09-17-cf-perms-next-waves-tiers/qwen-pack.md
    - docs/research/2026-09-17-cf-perms-next-waves-tiers/grok-pack.md
    - docs/research/2026-09-17-cf-perms-next-waves-tiers/gemini-pack.md
    - docs/issues/FEAT-011.yaml
schema_version: 1
---

## Context

v0.29.0 shipped (service-depth wave). FEAT-026 part 2 merged; part 3
(sub-resource name surfaces) noted on the issue. Three research packs went
out to external AIs (Qwen/Grok/Gemini) on 2026-09-17 from
`docs/research/2026-09-17-cf-perms-next-waves-tiers/`. Their outputs land in
the same directory as `qwen-results.md`, `grok-results.md`,
`gemini-results.md` (clipboard/file/paste intake — see the
brainstorming-research skill). If a results file is missing, that goal is
blocked: tell the operator which file is absent and stop that goal — do not
fabricate or partially ingest.

## Goals

### [ ] 1. Ingest and validate Qwen results (FEAT-011 canonical permission dataset)
Acceptance: qwen-results.md parsed; permission catalog + error table
validated for YAML shape; unverified rows flagged; dataset structured into
the FEAT-011 catalog (criterion 4) as a committed data file + loader wiring
decision recorded on the issue
### [ ] 2. Ingest Grok results (changelog drift, sentiment, competitor gaps, limits drift)
Acceptance: grok-results.md summarized; every limits-layer drift row
verified against our shipped numbers in pkg/cosmoflare; CLI-impact items
filed as issues/ideas (deterministic IDs via /feature or /idea — never
hand-create); sentiment-backed differentiation gaps recorded
### [ ] 3. Ingest Gemini results (tier business model, licensing stack, wave matrix)
Acceptance: gemini-results.md summarized; recommended monetization pattern +
licensing picks captured as a decision brief (docs/brainstorming/); legal
findings surfaced to the operator verbatim with citations
### [ ] 4. Synthesize cross-model findings with provenance
Acceptance: one synthesis doc in the research dir merging all three results
(the permissions needed per wave from Qwen x the demand/coverage evidence
from Grok x the priority matrix from Gemini), every claim tagged
[qwen|grok|gemini]; conflicts between models called out explicitly
### [ ] 5. Write the wave decision and record it
Acceptance: FEAT-030/035/036/037 order decided and recorded (issue notes +
roadmap); FEAT-011 dataset outcome recorded (closed if criterion 4 met, else
precise remainder); changelog staged if any code shipped

## Carry-Over

None — this prompt exists solely to ingest the three results files.

## Next Session Context

The packs' return envelopes define the exact markdown shapes — parse against
them. Qwen rows carry `verified:` dates; anything `unverified` needs a spot-
check before it enters the catalog. Grok's limits drift table must be
reconciled with pkg/cosmoflare's limits layer (normalizePlanTier values
free/paid). Gemini's ASK 4 is labeled INFORMED JUDGMENT — treat as input to
the operator's decision, not as the decision. Session-end: run
`ccs prompts verify --mechanical-only` and confirm new CONFIRMED_COVERED
verdicts.
