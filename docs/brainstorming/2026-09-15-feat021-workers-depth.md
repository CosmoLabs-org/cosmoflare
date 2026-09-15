---
title: "FEAT-021 Workers Command Depth — Design"
created: 2026-09-15
status: COMPLETED
related_issue: FEAT-021
tags: [brainstorm, workers, cli]
deliverables:
  - id: BR-01
    title: "Library: secrets CRUD + bulk (worker_secrets.go)"
  - id: BR-02
    title: "Library: routes CRUD (worker_routes.go, zone-scoped)"
  - id: BR-03
    title: "Library: versions raw-REST client (worker_versions.go, injectable base URL)"
  - id: BR-04
    title: "Library: deployments list/view/rollback (worker_deployments.go, raw REST)"
  - id: BR-05
    title: "Library: custom domains attach/detach/list (worker_domains.go)"
  - id: BR-06
    title: "Library + cmd: subdomain get/set (worker_subdomain.go)"
  - id: BR-07
    title: "Library + cmd: cron triggers CRUD (worker_cron.go)"
  - id: BR-08
    title: "Commands: 29 new worker subcommands with --json + Presenter"
  - id: BR-09
    title: "worker types: full .d.ts Env-interface generator"
  - id: BR-10
    title: "worker tail: streaming command over existing TailLogs"
---

# FEAT-021 — Workers Command Depth

## Problem

`cmd/worker.go` ships 14 commands (deploy/list/get/delete/logs/settings) but
none of the daily-loop surface wrangler users need: versions, deployments,
rollback, secrets, routes, custom domains, workers.dev subdomain, cron
triggers, bindings, tail, types. The competitive audit (agent-4) named this
the top gap: "without these, the Workers majority of the Cloudflare market
cannot daily-drive cosmoflare."

## Source of truth

Corpus `docs/research/2026-09-10-cf-limits-corpus/qwen-results-commands.md`
§1.2 (lifecycle + deployments + versions), §1.3 (secrets), §2.4 (script
management API: routes/domains/subdomain/tails/cron/bindings).

## Decisions (Q&A with operator, 2026-09-15)

1. **`worker types` = full `.d.ts` generation** (operator choice over
   JSON-only / defer). Generator is a PURE function over
   `WorkerSettings`/bindings — no network beyond fetching settings —
   emitting a `worker-configuration.d.ts` with an `Env` interface typed per
   binding kind.
2. **Execute artifacts + dispatch wave 1 immediately** (operator choice);
   waves 2+ run via the continuation prompt.

## Architecture

### Library-first, one file per group

New `*WorkerService` methods live in NEW files beside `worker.go`
(`worker_secrets.go`, `worker_routes.go`, `worker_versions.go`,
`worker_deployments.go`, `worker_domains.go`, `worker_subdomain.go`,
`worker_cron.go`). `worker.go` stays untouched except where noted — keeps
funlen clean and merges conflict-free.

### Transport split (the key constraint)

cloudflare-go **v0.116.0** (pinned in go.mod) provides:
secrets (Set/Delete/List), routes (CRUD, zone-scoped rc), custom domains
(List/Attach/Detach/Get), subdomain (Get/Create), cron (List/Update), and
the tail API. It does **NOT** ship script versions or deployments.

Therefore:
- Groups with cloudflare-go coverage call it directly (existing pattern:
  `rc := cloudflare.AccountIdentifier(s.accountID)`).
- **Versions + deployments use a raw-REST helper ON WorkerService**:
  `s.rawRequest(ctx, method, path, body, out)` over
  `https://api.cloudflare.com/client/v4` with the bearer token captured at
  `NewWorkerServiceFromCreds` time. The base URL is an unexported field
  defaulting to the production host — **tests point it at an
  `httptest.NewServer`** (same contract-test style as `worker_test.go`).
  Secrets bulk = loop over SetWorkersSecret (no raw REST).

### Command layer

- 29 new subcommands across `cmd/worker_secrets.go`, `cmd/worker_routes.go`,
  `cmd/worker_versions.go`, `cmd/worker_deployments.go`,
  `cmd/worker_domains.go`, `cmd/worker_subdomain.go`, `cmd/worker_cron.go`,
  `cmd/worker_bindings.go`, `cmd/worker_tail.go`, `cmd/worker_types.go`.
- Each cmd file exposes `registerWorkerXCmds(parent *cobra.Command)` —
  **the one-line `workerCmd.AddCommand`-style wiring in `cmd/worker.go`
  stays Opus-side** so no two agents ever edit the same file (S313).
- Every command: `--json` via the Presenter (`outResult`/`outPayload`/
  `outErr`), rich `--help` with examples, agent-readable errors, funlen
  ≤80/50.
- `worker tail [name]`: wraps the EXISTING `WorkerService.TailLogs`
  channel — prints JSON lines in JSON mode, plain lines otherwise, until
  ctx cancel (ctrl-c).
- `worker rollback [deployment-id]`: deployments-level convenience =
  fetch deployments, resolve target, PUT deployment pointing at the
  previous version (raw REST).

### Rollback semantics (both corpus variants)

- `worker deployments list/view` + `worker rollback [id]` — deployment-level.
- `worker versions list/view/upload/deploy/delete/rollback` — version-level.
  `versions rollback <id>` = deploy that version (same PUT), reported
  distinctly.

### Testing

- Library: httptest contract tests per new file (mock the CF API JSON
  shapes; raw-REST tests hit the injected httptest server). Validation
  paths for empty names/IDs.
- cmd: offline validation-path tests per command file (the
  `cache_run_test.go` exemplar: globals snapshot, direct run-func calls,
  exact error substrings).

## Non-goals

- workerd local dev runtime (separate roadmap item).
- Gradual-deployment percentages/traffic splitting (version deploy is
  all-or-nothing pointer move).
- Static-asset upload paths.
- Secrets values ever printed (list = names only, per corpus redaction rule).

## Wave structure

- **Wave 1 (dispatched 2026-09-15):** secrets, routes, versions,
  deployments+rollback — the daily loop.
- **Wave 2:** domains, subdomain, cron, bindings, tail.
- **Wave 3:** types generator + integration wiring + acceptance gates
  (all 38 present with --json; rollback restores; tail streams; cron
  round-trip).
