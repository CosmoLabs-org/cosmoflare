# SPEC.md — Cosmoflare

> Last updated: 2026-10-10T01:30:00+04:00

## Overview

Cosmoflare is CosmoLabs' open-source (MIT) Go library and CLI managing the full Cloudflare developer platform — R2, Workers, KV, D1, DNS, and 20+ services — with two private commercial tiers built on the same core: a phone-first Ops PWA (billing, usage, domains, alerts) and a Tauri desktop dashboard. Library-first: every tier consumes `pkg/cosmoflare` or the Ops Worker's APIs; no separate implementations per tier. Users: the CosmoLabs operator (private tiers) and the open-source community (CLI).

## Tech Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| CLI + library | Go 1.26, cobra | Single binary, agent-first UX (`--json` everywhere, `cosmoflare search`, stable `error_code`s) |
| Ops backend | Cloudflare Worker (TypeScript, Bun) | Edge, KV-cached upstream reads, cron push |
| Ops pager | Vanilla TS PWA (Vite, no framework) | Install-size and perf on phones; hash-routed views |
| Desktop | Tauri 2 + React 18 + Tailwind v4 + shadcn/ui + lucide | Native shell, component-rich UI |
| Fonts | Space Grotesk (numbers, tabular) + Nunito (brand/nav) | Characterful display + round clean face |
| Deploy | Local releases only (no CI); wrangler for Ops | Operator policy: releases need explicit go |
| Testing | go test; vitest (bun); impeccable detect = 0; Playwright live-data harness | S334: prose is not evidence |

## Architecture Decisions

- **Real numbers only** (2026-10-10): every displayed metric must be API-authoritative; derived values carry explicit labels; datasets that misrepresent reality (KV rolling bytes) are excluded from allowance visuals, never priced.
- **shadcn/ui + framer motion land in the DESKTOP tier** (React). The pager stays vanilla TS — a React rebuild is an open ADR to decide at the design overhaul; until then the pager's component layer is `pager/src/components.ts` (dashBar/sectionCard/fieldRow) with the shared token system.
- **Design system**: dark-first, amber accent, no left borders, no tile outlines, 4/8/12/16/24/32 spacing, ease-out enters, reduced-motion respected. `/impeccable` is the enforcement gate.
- **Three-layer caching**: Worker KV (per-dataset TTL), client memory, sessionStorage — counts and views cost nothing after first paint.
- **Agent-CLI contract**: JSON default-able, deterministic exits, honest errors carrying HTTP status + API cause.

## Out of Scope

- CI/CD pipelines (releases are local by policy).
- Backend-for-frontend beyond the Ops Worker; no separate mobile app until the pager React ADR resolves.
- Showing storage/usage numbers without an authoritative API source.
