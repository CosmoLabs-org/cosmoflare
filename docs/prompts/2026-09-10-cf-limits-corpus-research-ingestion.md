---
brainstorm_ref: docs/brainstorming/2026-09-10-cf-limits-awareness-layer.md
covers_brainstorm_deliverables:
    - BR-01
    - BR-02
created: "2026-09-10T19:32:07+04:00"
id: P-2026-09-10-cf-limits-corpus-research-ingestion
priority: high
requires_reading:
    - docs/brainstorming/2026-09-10-cf-limits-awareness-layer.md
    - docs/research/2026-09-10-cf-limits-corpus/grok-pack.md
    - docs/research/2026-09-10-cf-limits-corpus/gemini-pack.md
    - docs/research/2026-09-10-cf-limits-corpus/qwen-pack.md
schema_version: 1
status: PENDING
tags:
    - research
    - limits
    - cloudflare
title: CF limits corpus — research ingestion
---
# CF limits corpus — research ingestion

## BEFORE Starting — Required Reading

**You MUST read these files in full before writing any code. They are the gospel truth of what must be implemented.**

Read in order:

1. **`docs/brainstorming/2026-09-10-cf-limits-awareness-layer.md`** — design rationale and decision history.

Loading is enforced by `ccs prompts load-context` — the command will error if any file is missing.

## Context

The limits-awareness brainstorm (2026-09-10) produced three self-contained
research packs for external AIs (Grok, Gemini, Qwen). The user ran each pack
in its AI and pasted the responses back into the research directory as
results files. This session ingests those results and produces the consensus
catalog draft. Implementation deliverables (BR-03..BR-06) are NOT in scope
here — they get their own continuation prompt after the catalog lands.

## BEFORE Starting — Preflight

Verify these exist in `docs/research/2026-09-10-cf-limits-corpus/`; if any
is missing, stop and tell the user to paste it first:

- `grok-results.md` — must contain RECENT CHANGES + CONFLICTS + CATALOG JSON
- `gemini-results.md` — must contain INCLUDED QUOTAS + CONFLICTS + CATALOG JSON
- `qwen-results.md` — must contain EDGE CASES + CONFLICTS + CATALOG JSON

## Goals

### [ ] G-01 Ingest Grok results
Parse `grok-results.md` CATALOG JSON. Record every RECENT CHANGES item with
its source URL. Covers BR-02.

### [ ] G-02 Ingest Gemini results
Parse `gemini-results.md` CATALOG JSON. Elevate INCLUDED QUOTAS entries the
same way the pack defined. Covers BR-02.

### [ ] G-03 Ingest Qwen results
Parse `qwen-results.md` CATALOG JSON. Preserve `enforceability` and `soft`
fields (defined by the qwen pack's catalog schema — they are not part of
brainstorm D1) — they determine what phase-2 guards (brainstorm D4) can
actually enforce. Record EDGE CASES items. Covers BR-02.

### [ ] G-04 Synthesize cross-model findings
For every catalog id: compare values across the three sources. Agreement →
consensus value (provenance = majority source_url). Disagreement → conflict
row. Write `docs/research/2026-09-10-cf-limits-corpus/conflicts.md` with the
full register (id, values, sources, recommended resolution, confidence).
Covers BR-02.

### [ ] G-05 Draft the catalog
Write `docs/research/2026-09-10-cf-limits-corpus/catalog-draft.json` —
schema v1 per the brainstorm (D1), one entry per consensus limit, every
entry with `source_url` + `verified_on` (full ISO8601 with timezone per
schema v1 / IMP-032 — record the moment of verification). Unresolved
conflicts: keep the more-conservative documented value, mark the conflict in
`notes`, list them in conflicts.md. Do NOT write to
`pkg/cosmoflare/limitsdata/` — that is BR-03 implementation work.
Covers BR-01 (as the research-dir draft; BR-03 embeds it as
`pkg/cosmoflare/limitsdata/catalog.json`).

### [ ] G-06 Update the brainstorm with reality
Append a "## Corpus findings (2026-09-10 ingestion)" section to the
brainstorm doc: corrected values for the 6 currently-hardcoded resources
(workers.scripts, workers.daily_requests, r2.buckets,
r2.custom_domains_per_bucket, dns.records, zones.count), any limit that
changed recently (Grok's RECENT CHANGES), and any catalog entry count worth
noting. Flag anything that contradicts brainstorm assumptions.

## Related

- Brainstorm: `docs/brainstorming/2026-09-10-cf-limits-awareness-layer.md`
- Next step after this session: implementation continuation prompt covering
  BR-03..BR-06 (LimitsService v2, tracking expansion, catalog export,
  guards/alerts specs) via `ccs prompts init --from-brainstorm`.
