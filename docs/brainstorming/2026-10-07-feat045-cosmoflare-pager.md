---
title: "FEAT-045: Cosmoflare Pager — open-source mobile + desktop alerting companion"
created: "2026-10-07T16:45:00+04:00"
status: COMPLETED
issue: FEAT-045
schema_version: 1
deliverables:
  - id: BR-01
    title: "pkg/cosmoflare/alertspush — VAPID keygen/store, subscription store, web-push sender with expiry pruning"
  - id: BR-02
    title: "CLI surface: cosmoflare alerts watch (interval evaluator firing pushes) + alerts push keygen|add|list|remove"
  - id: BR-03
    title: "Push payload protocol — {id, severity, service, title, detail, fired_at}, FEAT-041 vocabulary, no credentials"
  - id: BR-04
    title: "Static PWA in pager/ — service worker push handler, severity-colored alert list, detail sheet, ack/snooze, IndexedDB history"
  - id: BR-05
    title: "Tauri notifications view gains ack/snooze parity (shared semantics with PWA)"
  - id: BR-06
    title: "Vision + roadmap bookkeeping — PRODUCT-VISION principle 6 pivot, ROAD-064 link, FEAT-029 noted as v2 auth path"
---

# FEAT-045: Cosmoflare Pager — open-source mobile + desktop alerting companion

Status: COMPLETED (design approved in-session 2026-10-07). Issue: FEAT-045.

## Why

Cosmoflare can already EVALUATE alerts (`cosmoflare alerts` rules, the
webhook evaluator, the desktop notifications view) but an alert that nobody
sees at 3am is worthless. The missing piece is delivery to the person. The
pager is an open-source, zero-infrastructure companion that gets
Cloudflare-monitoring alerts onto a phone (and onto the existing desktop
dashboard) with a pager's discipline: severity, acknowledge, history.

Product pivot (approved): PRODUCT-VISION principle 6 said "open-source
core, paid mobile." The pager is open source — adoption for the alerting
loop beats subscription revenue at this stage; paid tiers remain desktop
dashboard depth + managed conveniences.

## Decisions (Q&A, approved 2026-10-07)

| # | Question | Decision |
|---|----------|----------|
| 1 | Transport | **Web Push (VAPID), zero infra** — no relay we run, no app-store gate, PWA on Android + iOS 16.4+ |
| 2 | Client shape | **Static PWA at cosmolabs.org/pager** — app shell is static; no user data touches our servers; phone is a pure receiver with NO CF token |
| 3 | v1 scope | **Pure receiver** — push → alert card, ack/snooze, on-device history. Live-status views and one-tap actions are later phases (v2 pairs with FEAT-029 device-flow auth) |
| 4 | Layout | **Monorepo + Tauri parity** — pager/ dir here; Tauri app gains ack/snooze; library-first, no separate implementations |

Sender approach: **A — sender in the Go core** (`pkg/cosmoflare/alertspush`),
shared by the headless CLI (`alerts watch`) and the Tauri sidecar. B
(desktop-only) strands CLI-only users; C (Worker cron) is the rejected relay
model.

## Architecture

```
~/.cosmoflare/push.json ──┐
  (VAPID keypair,         │
   subscriptions)         ▼
              pkg/cosmoflare/alertspush
              (keygen · store · send · prune)
                          │
        ┌─────────────────┴──────────────────┐
        ▼                                    ▼
cosmoflare alerts watch             Tauri sidecar (same loop)
(interval: evaluate rules           (desktop app running =
 → fire pushes)                      watch running)
        │
        ▼  Web Push (VAPID, sender-side keys only)
browser push service (vendor)
        ▼
cosmolabs.org/pager (static PWA)
  service worker → notification + IndexedDB
  alert list / detail / ack / snooze  (local-only)
```

## Pairing (one paste, one time)

VAPID keys are sender-side ONLY — the phone never receives or needs them.
1. `cosmoflare alerts push keygen` — creates the keypair in
   `~/.cosmoflare/push.json` (0600).
2. Phone: install PWA → enable notifications → tap "Copy subscription"
   (copies the push-subscription blob: endpoint + p256dh + auth).
3. `cosmoflare alerts push add` — reads the blob from stdin/clipboard into
   the store.
QR pairing is v2 polish.

## Push payload (the protocol)

```json
{"id": "alert-1730...", "severity": "warning", "service": "r2",
 "title": "Class A ops at 92% of monthly free tier",
 "detail": "bucket media-cache …", "fired_at": "2026-10-07T16:20:05+04:00"}
```

- FEAT-041 severity vocabulary exactly (info | warning | critical) and the
  same color ramp tokens as the desktop (`--sev-*` values documented in
  desktop/src/styles.css).
- ≤ 2KB. **No credentials ever travel in a payload.**
- ack/snooze in v1 are LOCAL to the device (no backchannel); the watch loop
  does not track per-device ack state.

## PWA (pager/)

- Vanilla TypeScript + Vite. No framework — v1 is three views (list,
  detail, settings/pairing). Build output is fully static, deployable to
  cosmolabs.org/pager by the site's existing pipeline.
- Service worker: `push` event → show notification (severity-tagged) +
  append to IndexedDB; `notificationclick` → focus/open detail.
- Alert list mirrors desktop tokens (dark-first, `#1a1a1a` ground, amber
  accent, severity ramp) — visual kinship with the desktop dashboard.
- Ack = swipe/button marks the item acknowledged (local); snooze hides for
  N minutes (local). History capped (default 500 items, FIFO).
- Manifest: standalone display, monochrome icon, theme color `#1a1a1a`.

## Desktop parity

The existing Tauri notifications view (FEAT-041) gains the same
ack/snooze actions with identical semantics, so both surfaces behave the
same. No new desktop views in v1.

## Error handling

- Push respond 404/410 → prune subscription, log one line.
- VAPID signing failure → actionable error naming `alerts push keygen`.
- Watch-loop rule evaluation failures reuse the alerts engine's existing
  error envelope; one failing rule never stops the loop.

## Security

- VAPID private key: `~/.cosmoflare/push.json`, 0600, never leaves the machine.
- Phone holds zero Cloudflare credentials in v1.
- Subscription endpoints are treated as sensitive (they can push to the
  device) — stored locally, never logged.
- Payload integrity: Web Push TLS; v2 may add a symmetric HMAC if payload
  authenticity concerns arise (not v1 — payload carries no instructions).

## Testing

- Go: `alertspush` unit tests — keygen (persist/load, 0600), payload
  marshal (schema + size cap), subscription add/remove/list, prune on
  404/410 (stub sender), determinism where applicable. Sender tests use a
  stub HTTP transport; no network in tests.
- CLI: `alerts watch --test-fire` emits a canned info-severity push for
  end-to-end pairing verification (also the manual E2E step).
- PWA: vitest — payload parsing, IndexedDB store (fake-indexeddb), ack/
  snooze reducers.
- Tauri: extend the existing vitest suite for ack/snooze state.

## Out of scope (v1)

Live status views, one-tap actions, QR pairing, per-device ack backchannel,
on-call rotations/escalation, APNs/FCM native apps, i18n.

## v2 native note (decided 2026-10-07)

If native iOS/Android apps follow, they wrap `pkg/cosmoflare` via gomobile
bindings — same single Go core the CLI/desktop/MCP wrap. A Crux-style shared
Rust core was considered and declined: it would re-implement domain logic in
a second language and break PRODUCT-VISION principle 4 (no separate
implementations). Revisit only as an explicit, separately-brainstormed
platform pivot.
