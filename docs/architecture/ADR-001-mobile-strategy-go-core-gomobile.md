# ADR-001: Mobile strategy — one Go core, PWA-first pager, gomobile for native

- **Status**: Accepted
- **Date**: 2026-10-07
- **Deciders**: Operator (GΛB) + session design review
- **Related**: FEAT-045 (pager), FEAT-046 (shader pass), ROAD-064 (mobile app),
  docs/brainstorming/2026-10-07-feat045-cosmoflare-pager.md,
  docs/PRODUCT-VISION.md (principle 4 — library-first, no separate implementations)

## Context

Cosmoflare's founding principle is one Go core (`pkg/cosmoflare`) wrapped by
every tier: CLI, MCP server, desktop (Tauri). The pager (FEAT-045) extends
the chain to phones. Three questions arose together:

1. Is pager v1 a native iOS app? 
2. If native apps come later, how do they share logic?
3. Should we adopt Crux (shared Rust core) for mobile?

## Decision

1. **Pager v1 is an installable PWA** (cosmolabs.org/pager) receiving Web
   Push (VAPID). No App Store gate, no relay infrastructure, works on
   Android and iOS 16.4+. Native-only capabilities (lock-screen widgets,
   Watch complications) are the trigger to revisit, not a launch requirement.
2. **Future native apps wrap the Go core via gomobile** — `gomobile bind`
   produces an iOS `.xcframework` and an Android `.aar` from a curated
   facade package exposing only phone-appropriate operations. Swift/Kotlin
   or React Native shells consume the same library every other tier wraps.
3. **Crux (shared Rust core) is declined.** It would re-implement domain
   logic in a second language and break PRODUCT-VISION principle 4. Rust
   becomes an option only as an explicit, separately-brainstormed platform
   pivot — never smuggled in through a feature.

## Options considered

| Option | Verdict | Why |
|--------|---------|-----|
| PWA + Web Push (v1) | **Chosen for v1** | Zero infra, zero store gate, both platforms today; receiver has no business logic worth sharing |
| Crux Rust core | Declined | Second brain; violates single-implementation principle; v1 has no shared-core need |
| gomobile Go bindings | **Chosen for native (v2+)** | One core preserved; pure-Go codebase is the right shape; costs: ~10–15 MB/arch binary size, curated facade needed |
| Full native rewrites (Swift/Kotlin) | Declined | N implementations, N drift surfaces |
| Our hosted push relay (APNs/FCM) | Declined | Central infra, our cost/uptime, trust question for an open-source tool |
| Self-hosted CF Worker relay | Deferred | Viable community option later; adds deploy + credentials steps v1 doesn't need |

## Consequences

- The pager PWA ships without any server-side runtime; static hosting only.
- A `pkg/cosmoflare` mobile facade (e.g. `pkg/cosmoflare/mobile`) is the
  designated seam for gomobile when native work starts — keep exported
  types phone-friendly from the start (no io.Writer/Reader in the facade).
- PRODUCT-VISION principle 6 updates: open-source core + open-source pager;
  paid tiers are desktop dashboard depth and managed conveniences.
- Web Push payload stays a dumb, credential-free JSON envelope — any future
  transport (native push, relay) can carry the same schema.
