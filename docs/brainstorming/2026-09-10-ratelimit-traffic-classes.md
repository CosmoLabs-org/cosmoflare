---
title: 'Rate-limit traffic classes — counted/skipped matrix + active trip probe'
created: "2026-09-10T16:27:12+04:00"
status: planned
tags: [brainstorm, knowledge, ratelimit, probe, feat-013]
issue: FEAT-013
deliverables:
  - id: BR-01
    title: "TrafficClass pack schema + SkippedTrafficClasses lookup + seed matrix with evidence links"
  - id: BR-02
    title: "RateLimitProber — bounded live burst with classifyProbe verdict engine"
  - id: BR-03
    title: "CLI — ratelimit probe command + --probe on create + pack-driven advisory"
  - id: BR-04
    title: "docs/USAGE.md — traffic-class matrix and probe documentation"
---

# Rate-limit traffic classes — counted/skipped matrix + active trip probe

## Problem

FEAT-013 (origin: MyCarGuide feedback FB-7, 2026-09-09): a rate-limiting rule
verified `enabled=true` via the API produced **zero 429s** across three burst
tests — 45+ requests per 10s window, including cache-busted unique-query
variants — against a Cloudflare Pages static asset served on the zone's custom
domain (`mycar.guide/catalog.json`). WAF rate limiting silently does not count
certain traffic classes. The API accepts the rule, reports enabled, and the
operator believes they are protected when they are not. No Cloudflare surface
answers "does this rule see my traffic" at creation time.

## Decisions (Q&A with user, 2026-09-10)

| # | Question | Decision |
|---|----------|----------|
| 1 | Matrix source of truth — docs, probe, or both? | **Docs-derived matrix with evidence links** — the pack encodes CF-documented counted/skipped classes, each entry carries a source URL; probe verdicts reference the matrix as explanation. CF behavior can drift; sources make drift auditable. |
| 2 | Probe surface? | **Command + flag** — new `cosmoflare ratelimit probe` (standalone, works on existing rules) AND `--probe` on `ratelimit create` (runs immediately after creation). Create without `--probe` prints a one-line suggestion. |
| 3 | Live-traffic safety? | **Flag = opt-in, capped burst** — explicit invocation is the consent; no extra y/N (agents script this). Burst defaults 2× the rule's `requests_per_period`, hard cap 60, `--requests` clamps to the cap. Half the requests carry a unique `cfprobe` query param (cache-buster) so cached vs origin classes are distinguishable. Never runs on default paths. |
| 4 | Where the static matrix surfaces? | **Pack-driven create advisory** — `ratelimit create` prints an advisory listing the pack's skipped traffic classes + the probe suggestion, sourced from pack data (new packs get it free); `knowledge list` shows traffic-class counts. |

## Design

### Schema extension (knowledge pack)

```go
// TrafficClass documents whether WAF rate limiting counts one traffic class.
type TrafficClass struct {
    Class   string `json:"class"`
    Counted bool   `json:"counted"`
    Note    string `json:"note,omitempty"`
    Source  string `json:"source,omitempty"` // evidence URL
}
```

`Pack` gains `TrafficClasses []TrafficClass \`json:"traffic_classes,omitempty"\``
— absent in old packs, no behavior change. New pure lookup
`SkippedTrafficClasses(product string) []TrafficClass`.

Seed matrix in `packs/ratelimit.json` (v1 entries, each source-linked):

| Class | Counted | Source |
|-------|---------|--------|
| `cache-hit-static-asset` | no | CF rate-limiting docs (counting happens pre-cache) + FB-7 evidence |
| `pages-custom-domain-asset` | no | FB-7 evidence (rule d65876b4…) |
| `origin-miss` | yes | CF rate-limiting docs |
| `cache-bypass` | yes | CF rate-limiting docs |

### RateLimitProber (RedirectProber pattern)

`pkg/cosmoflare/ratelimitprobe.go` — stdlib only, no CF credentials:

```go
type RateLimitProbeResult struct {
    URL          string         `json:"url"`
    Requests     int            `json:"requests"`            // actually sent
    Statuses     map[int]int    `json:"statuses"`            // status -> count
    Tripped      bool           `json:"tripped"`
    Verdict      string         `json:"verdict"`             // tripped | not-counted | inconclusive
    Explanation  string         `json:"explanation,omitempty"` // cites the matrix when not-counted
}
```

Options: `WithProbeHTTPClient`, `WithProbeTimeout`, `WithProbeConcurrency`.
`Probe(ctx, url, requests)` sends GETs — even indices plain, odd indices with
`?cfprobe=<unique>` — records the histogram, classifies. Verdict logic is a
pure extracted function:

```go
func classifyProbe(statuses map[int]int, sent int) (verdict, explanation string)
```

- any 429 (or block-status) → `tripped`
- all 2xx/3xx with sent == requested → `not-counted`, explanation composed
  from `SkippedTrafficClasses("ratelimit")`
- transport errors dominate (or sent == 0) → `inconclusive`

`expressionPath(expr string) string` — pure helper extracting the path from a
`path eq "/x"` expression (`""` when it cannot).

### CLI surface

- `cosmoflare ratelimit probe <zone-id-or-name> --path <p> [--requests N]` —
  resolves the zone (existing `resolveZoneID`), reads the live rule's
  `requests_per_period` for the burst default, probes
  `https://<zone-name><path>`.
- `cosmoflare ratelimit create ... --probe` — after a successful create,
  probes the created rule's expression path.
- Every successful `create` prints the pack-driven advisory (skipped classes
  + probe suggestion) — data from `SkippedTrafficClasses`, never hardcoded.
- Exit codes: `0` tripped, `2` not-counted, `3` inconclusive (scriptable,
  agent-readable; documented in --help and USAGE.md).
- `knowledge list` gains traffic-class counts in its line.

## Error Handling

| Failure | Behavior |
|---------|----------|
| Probe transport errors | Verdict `inconclusive` — never an error; the result carries counts |
| Probe after create | Create result stands; probe output is advisory, printed after |
| Unknown product in matrix lookup | Empty slice — advisory section omitted (advisory when absent) |
| `--requests` above cap | Clamped to 60 with a printed notice |
| Expression without a parseable path | `--probe` on create reports `inconclusive` with the reason; standalone probe requires `--path` |
| Rate limit not yet propagated (CF eventual consistency) | Verdict `not-counted` may be premature — result prints elapsed time and a caveat line |

## Testing (TDD, no live network)

- `classifyProbe` table tests: 429 → tripped; all-200 → not-counted + matrix
  explanation; error-dominated → inconclusive; mixed 429+200 → tripped.
- `SkippedTrafficClasses`: ratelimit pack returns the skipped entries;
  unknown product returns empty.
- Prober httptest servers: always-429, always-200, half-flaky, fully-dead.
- `expressionPath` table: `path eq "/x"`, `http.request.uri.path eq "/x"`,
  non-path expression → "".
- CLI: probe/create-flag registration, exit-code mapping, advisory rendering
  from pack data, --requests clamping.

## Non-Goals (deferred)

- No automatic probing on any default path (issue constraint: opt-in only).
- No new traffic-class research beyond the four v1 entries — the Qwen
  coverage pass (FEAT-011 companion research) may extend the matrix later as
  pack data.
- No probe result persistence/history — results print (and `--json`) only.
- No changes to `limits.go` (SSOT boundary decision, FEAT-012 brainstorm).

## Deliverables

- BR-01: TrafficClass schema + SkippedTrafficClasses + seed matrix (evidence-linked)
- BR-02: RateLimitProber with classifyProbe verdict engine
- BR-03: CLI probe command + --probe flag + pack-driven advisory
- BR-04: docs/USAGE.md traffic-class and probe documentation
