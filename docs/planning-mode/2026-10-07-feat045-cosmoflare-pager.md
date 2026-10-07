---
title: "Cosmoflare Pager v1 Implementation Plan"
created: "2026-10-07T17:10:00+04:00"
status: PENDING
branch: master
issue: FEAT-045
brainstorm_ref: docs/brainstorming/2026-10-07-feat045-cosmoflare-pager.md
schema_version: 1
deliverables:
  - id: P-01
    title: "alertspush Store — VAPID keypair + subscriptions, push.json 0600"
  - id: P-02
    title: "alertspush Payload protocol — severity vocabulary, 2KB envelope"
  - id: P-03
    title: "alertspush Sender + Dispatch with 404/410 pruning"
  - id: P-04
    title: "CLI alerts push keygen|add|list|remove with --json + USAGE.md section"
  - id: P-05
    title: "CLI alerts watch interval evaluator + --test-fire"
  - id: P-06
    title: "PWA scaffold — manifest, service worker push handler, payload parser"
  - id: P-07
    title: "PWA views — list/detail/ack/snooze/pairing, IndexedDB store"
  - id: P-08
    title: "Desktop notifications ack/snooze parity"
  - id: P-09
    title: "PRODUCT-VISION principle-6 pivot + ROAD-064 link"
---

# Cosmoflare Pager v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the pager v1 — Web Push (VAPID) alert delivery from the Go core to an installable static PWA, plus ack/snooze parity on the desktop notifications view.

**Architecture:** A new `pkg/cosmoflare/alertspush` Go package owns VAPID keys, push subscriptions (`~/.cosmoflare/push.json`) and sending; `cosmoflare alerts watch` evaluates existing alert rules on an interval and fires pushes; a vanilla-TS static PWA in `pager/` receives pushes via a service worker and keeps history in IndexedDB; the Tauri notifications view gains the same ack/snooze semantics. Zero runtime infrastructure; the phone holds no Cloudflare credentials. Spec: `docs/brainstorming/2026-10-07-feat045-cosmoflare-pager.md` (approved). ADR: `docs/architecture/ADR-001-mobile-strategy-go-core-gomobile.md`.

**Tech Stack:** Go 1.26 (pure Go, `CGO_ENABLED=0`), cobra, `github.com/SherLockHolmes/webpush-go` (VAPID Web Push); pager: Vite + vanilla TypeScript + vitest + fake-indexeddb; desktop: React 18 + vitest (existing suite).

**Spec:** `docs/brainstorming/2026-10-07-feat045-cosmoflare-pager.md`

## Global Constraints

- Pure Go only — the release cross-compiles with `CGO_ENABLED=0`; no cgo dependencies.
- Severity vocabulary is exactly `info | warning | critical` (FEAT-041 ramp).
- Push payload is a JSON envelope `{id, severity, service, title, detail, fired_at}`, ≤ 2048 bytes marshaled; **no credentials ever in a payload**.
- `~/.cosmoflare/push.json` is created with permissions `0600`; VAPID private key never leaves the machine.
- Every new CLI command: rich `--help` with examples, `--json` output with the standard envelope, deterministic exit codes, agent-readable errors (what/why/how-to-fix).
- JS toolchain: bun only (`bun install`, `bun run test` → vitest). Never npm/pnpm.
- Tests: Go tests alongside source (`*_test.go`); no network in tests (fake senders / fake-indexeddb only).
- PWA is fully static — no runtime backend, no user data through our servers.

## File Structure

```
pkg/cosmoflare/alertspush/          # Go core package (Tasks 1-3)
  store.go        — Store: VAPID keys + subscriptions, push.json IO
  payload.go      — Payload type, validation, marshaling
  sender.go       — Sender interface, webpush impl, Dispatch + pruning
  *_test.go
cmd/alerts_push.go                  # cobra: alerts push keygen|add|list|remove (Task 4)
cmd/alerts_watch.go                 # cobra: alerts watch (+ --test-fire) (Task 5)
pager/                              # static PWA (Tasks 6-7)
  package.json, vite.config.ts, tsconfig.json, index.html
  src/main.ts, src/store.ts, src/payload.ts, src/views.ts, src/styles.css
  public/manifest.webmanifest, public/sw.js, public/icon-192.png, public/icon-512.png
  src/*.test.ts
desktop/src/views/Notifications.tsx  # ack/snooze parity (Task 8)
docs/USAGE.md, docs/PRODUCT-VISION.md, docs/roadmap/ (Task 9)
```

---

### Task 1: alertspush — Store (VAPID keys + subscriptions)

**Files:**
- Create: `pkg/cosmoflare/alertspush/store.go`
- Test: `pkg/cosmoflare/alertspush/store_test.go`

**Interfaces:**
- Consumes: nothing (first task in the package).
- Produces:
  - `type Subscription struct { Endpoint string; P256dh string; Auth string }`
  - `type Store struct { VAPIDPublicKey string; VAPIDPrivateKey string; Subscriptions []Subscription }`
  - `func LoadStore(path string) (*Store, error)` (missing file → empty Store, nil error)
  - `func (s *Store) Save(path string) error` (file mode 0600, mkdir -p parent)
  - `func (s *Store) Keygen(path string) error` (generate VAPID keypair if absent, persist, keep subscriptions)
  - `func (s *Store) AddSubscription(sub Subscription) error` (dedupe by Endpoint)
  - `func (s *Store) RemoveSubscription(endpoint string) bool`
  - `func (s *Store) HasVAPIDKeys() bool`

- [ ] **Step 1: Write failing tests**

```go
package alertspush

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadStore_MissingFileIsEmpty(t *testing.T) {
	s, err := LoadStore(filepath.Join(t.TempDir(), "push.json"))
	if err != nil { t.Fatalf("LoadStore: %v", err) }
	if s == nil || len(s.Subscriptions) != 0 || s.HasVAPIDKeys() { t.Fatal("want empty store") }
}

func TestKeygen_PersistsKeysAndKeepsSubscriptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "push.json")
	s, _ := LoadStore(path)
	if err := s.Keygen(path); err != nil { t.Fatalf("Keygen: %v", err) }
	if !s.HasVAPIDKeys() { t.Fatal("keys not generated") }
	reloaded, _ := LoadStore(path)
	if reloaded.VAPIDPublicKey != s.VAPIDPublicKey { t.Fatal("keys not persisted") }
	reloaded.AddSubscription(Subscription{Endpoint: "https://push.example/e1", P256dh: "k", Auth: "a"})
	if err := reloaded.Save(path); err != nil { t.Fatal(err) }
	if err := reloaded.Keygen(path); err != nil { t.Fatal(err) }
	if reloaded.VAPIDPublicKey != s.VAPIDPublicKey { t.Fatal("Keygen must not rotate existing keys") }
	if len(reloaded.Subscriptions) != 1 { t.Fatal("Keygen dropped subscriptions") }
}

func TestSave_FileMode0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "push.json")
	s, _ := LoadStore(path)
	s.AddSubscription(Subscription{Endpoint: "e", P256dh: "k", Auth: "a"})
	if err := s.Save(path); err != nil { t.Fatal(err) }
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 { t.Errorf("mode = %v, want 0600", info.Mode().Perm()) }
}

func TestAddSubscription_DedupesByEndpoint(t *testing.T) {
	s := &Store{}
	s.AddSubscription(Subscription{Endpoint: "e1", P256dh: "a", Auth: "a"})
	s.AddSubscription(Subscription{Endpoint: "e1", P256dh: "b", Auth: "b"})
	if len(s.Subscriptions) != 1 || s.Subscriptions[0].P256dh != "b" {
		t.Fatal("dedupe/replace by endpoint failed")
	}
	if !s.RemoveSubscription("e1") || s.RemoveSubscription("e1") { t.Fatal("remove semantics wrong") }
}
```

- [ ] **Step 2: Run, expect FAIL** — `go test ./pkg/cosmoflare/alertspush/` → undefined: LoadStore etc.
- [ ] **Step 3: Implement `store.go`** — JSON file `{"vapid_public_key":..., "vapid_private_key":..., "subscriptions":[{endpoint,p256dh,auth}]}`. Keygen generates via `webpush.GenerateVAPIDKeys()` from `github.com/SherLockHolmes/webpush-go` (run `go get github.com/SherLockHolmes/webpush-go@latest` first; `go doc github.com/SherLockHolmes/webpush-go` to confirm the exported names — if `GenerateVAPIDKeys` differs, adapt to the documented constructor, keep Store's API unchanged). Save writes with `os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)` after `os.MkdirAll(filepath.Dir(path), 0o700)`.
- [ ] **Step 4: Run, expect PASS** — `go test ./pkg/cosmoflare/alertspush/`
- [ ] **Step 5: Commit** — `feat(alertspush): VAPID keypair + subscription store (FEAT-045)`

### Task 2: alertspush — Payload protocol

**Files:**
- Create: `pkg/cosmoflare/alertspush/payload.go`
- Test: `pkg/cosmoflare/alertspush/payload_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces:
  - `type Severity string` with consts `SeverityInfo Severity = "info"`, `SeverityWarning = "warning"`, `SeverityCritical = "critical"`
  - `type Payload struct { ID string; Severity Severity; Service string; Title string; Detail string; FiredAt time.Time }`
  - `func (p Payload) Validate() error` (non-empty ID/Title/Service, valid severity)
  - `func (p Payload) MarshalEnvelope() ([]byte, error)` (JSON keys id/severity/service/title/detail/fired_at RFC3339; error if > 2048 bytes)

- [ ] **Step 1: Failing tests** — valid payload marshals with exact keys (`fired_at` RFC3339); `Severity("bogus")` → Validate error; a 2,100-byte Detail → MarshalEnvelope error `payload exceeds 2048 bytes`; ID/Title/Service empty → errors.
- [ ] **Step 2: Run → FAIL.** **Step 3: Implement** with `encoding/json` (struct tags `json:"id"` etc., `FiredAt time.Time` marshaled via its RFC3339 default). **Step 4: PASS.**
- [ ] **Step 5: Commit** — `feat(alertspush): push payload protocol, 2KB cap, severity vocabulary (FEAT-045)`

### Task 3: alertspush — Sender + Dispatch with pruning

**Files:**
- Create: `pkg/cosmoflare/alertspush/sender.go`
- Test: `pkg/cosmoflare/alertspush/sender_test.go`

**Interfaces:**
- Consumes: Store (Task 1), Payload (Task 2).
- Produces:
  - `type Sender interface { Send(ctx context.Context, sub Subscription, envelope []byte) (status int, err error) }`
  - `type WebPushSender struct{...}` — `func NewWebPushSender(publicKey, privateKey string) *WebPushSender` (wraps webpush-go; construct `webpush.Subscription{Endpoint, Keys: {P256dh, Auth}}` — confirm field names via `go doc` and adapt)
  - `func Dispatch(ctx context.Context, st *Store, sender Sender, p Payload) (sent, pruned int, err error)` — marshals, sends to every subscription; on status 404/410 removes the subscription and counts pruned; one failing subscription never aborts the rest; returns first non-prune error.

- [ ] **Step 1: Failing tests** with `type fakeSender struct{ statuses map[string]int; calls []string }` implementing Sender: two subs, one 404 → sent=1, pruned=1, sub removed from store; all 200 → sent=2 pruned=0; 500 → err non-nil, other subs still attempted.
- [ ] **Step 2: FAIL → Step 3: implement → Step 4: PASS** (`go test ./pkg/cosmoflare/alertspush/`).
- [ ] **Step 5: Commit** — `feat(alertspush): dispatch with expiry pruning, sender interface (FEAT-045)`

### Task 4: CLI — `alerts push keygen|add|list|remove`

**Files:**
- Create: `cmd/alerts_push.go`
- Test: `cmd/alerts_push_test.go`

**Interfaces:**
- Consumes: alertspush Store (Task 1); the repo's standard output envelope helpers (see an existing simple command, e.g. `cmd/limits.go`, for the exact helper names — reuse them; do not invent new envelope code).
- Produces: cobra commands registered under the existing `alertsCmd` in `cmd/alerts.go` (`alertsCmd.AddCommand(...)` in this file's init).

Behavior: `keygen` (idempotent, prints public key + one-time pairing instructions), `add` (reads subscription JSON `{endpoint,p256dh,auth}` from `--stdin` flag or clipboard arg; validates fields), `list` (--json), `remove ENDPOINT`. Store path: reuse the `~/.cosmoflare` dir resolution pattern from `internal/config/config.go` (~line 84-92) — extract or reuse its helper rather than re-deriving HOME.

- [ ] **Step 1: Failing tests** — golden JSON shape for `list` (empty + one sub); `add` with missing p256dh → agent-readable error naming the expected shape; `keygen` twice → same key. Use the existing cmd test patterns in `cmd/` for cobra invocation.
- [ ] **Step 2-4: red → implement → green.** `go test ./cmd/ -run TestAlertsPush`
- [ ] **Step 5: USAGE.md** — add an "alerts push / alerts watch" section with examples (agent-reference style, match existing section format).
- [ ] **Step 6: Commit** — `feat(cmd): alerts push keygen/add/list/remove (FEAT-045)`

### Task 5: CLI — `alerts watch` loop + `--test-fire`

**Files:**
- Create: `cmd/alerts_watch.go`
- Test: `cmd/alerts_watch_test.go`

**Interfaces:**
- Consumes: `cosmoflare.AlertService` (existing, via the `getAlertServiceFn` seam in `cmd/alerts.go` — reuse that seam, it exists for testability), alertspush Dispatch (Task 3).
- Produces: `func evaluateOnce(ctx context.Context, svc *cosmoflare.AlertService, st *alertspush.Store, sender alertspush.Sender) (sent, pruned int, err error)` — package-level so the test drives ONE iteration without the loop; the cobra command wraps it with `--interval` (default 60s, min 10s) and `--test-fire` (sends a canned info payload, exits 0).

Rule→payload mapping: each alert the service reports (use the existing check path `runAlertsCheck` calls — read `cmd/alerts.go:239-296` for the exact returned shape) becomes `Payload{ID: <rule id>, Severity: <from rule priority — map existing levels to info/warning/critical, defaulting info>, Service: <rule service or "cloudflare">, Title: <rule name/status>, Detail: <check message>, FiredAt: now}`.

- [ ] **Step 1: Failing tests** — fake AlertService (via getAlertServiceFn) returning one tripped rule + fake sender → sent=1 and the payload marshals with severity warning when the rule's level maps to warning; `--test-fire` path returns a payload with Severity info. If AlertService is a concrete type that can't be faked, test `evaluateOnce` against its interface seam as `runAlertsCheck` does — follow that existing pattern exactly.
- [ ] **Step 2-4: red → implement → green** (`go test ./cmd/ -run TestAlertsWatch`).
- [ ] **Step 5: Commit** — `feat(cmd): alerts watch — interval evaluator firing web pushes + --test-fire (FEAT-045)`

### Task 6: PWA scaffold — manifest, service worker, push handler

**Files:**
- Create: `pager/package.json`, `pager/vite.config.ts`, `pager/tsconfig.json`, `pager/index.html`, `pager/public/manifest.webmanifest`, `pager/public/sw.js`, `pager/public/icon-192.png`, `pager/public/icon-512.png`, `pager/src/main.ts`, `pager/src/payload.ts`
- Test: `pager/src/payload.test.ts`

**Interfaces:**
- Consumes: Payload JSON schema (Task 2 — mirror exactly).
- Produces: `parsePayload(data: unknown): Payload | null` (payload.ts) — same field names, severity union type; `public/sw.js` with `push` → `self.registration.showNotification` + IndexedDB append (store name `alerts`, keyPath `id`) and `notificationclick` → focus/open.

- [ ] **Step 1: Scaffold** — `bun create vite@latest pager -- --template vanilla-ts` inside a temp dir then move contents (or hand-write the four config files to avoid interactive prompts; hand-writing is preferred — deterministic). Manifest: `display: "standalone"`, `start_url: "/"`, `theme_color: "#1a1a1a"`, `background_color: "#1a1a1a"`. Icons: generate placeholder monochrome "⚡" PNGs (192/512) with any local tool — mark TODO(design) for final art.
- [ ] **Step 2: Failing test** — vitest for parsePayload: valid envelope parses; severity "bogus" → null; missing id → null. `bun run test` (add script `"test": "vitest run"`, devDeps vitest only).
- [ ] **Step 3-4: red → implement → green.**
- [ ] **Step 5: Service worker** — push handler reads `event.data.json()`, validates via the same rules (inline in sw.js — no imports), appends to IndexedDB, shows notification with `tag: severity` and vibrancy by severity (critical: requireInteraction).
- [ ] **Step 6: Build gate** — `bun run build` exits 0; dist/ is fully static.
- [ ] **Step 7: Commit** — `feat(pager): PWA scaffold — manifest, service worker push handler, payload parser (FEAT-045)`

### Task 7: PWA views — list, detail, ack/snooze, settings

**Files:**
- Create: `pager/src/store.ts` (IndexedDB wrapper), `pager/src/views.ts`, `pager/src/styles.css`
- Test: `pager/src/store.test.ts`

**Interfaces:**
- Consumes: parsePayload (Task 6), IndexedDB schema (sw.js writes; store.ts reads — same DB `cosmoflare-pager`, store `alerts`).
- Produces: `listAlerts(): Promise<AlertRecord[]>` (newest first, capped 500 FIFO on write), `acknowledge(id)`, `snooze(id, minutes)` — snoozed records get `snoozed_until` timestamp and are filtered from list until it passes.

Views (vanilla TS DOM, no framework): settings/pairing view with "Enable notifications" (Permission + pushManager.subscribe with `userVisibleOnly: true`; on success show "Copy subscription" — `navigator.clipboard.writeText(JSON.stringify(subscription.toJSON()))` plus the exact `cosmoflare alerts push add` command to paste it into); alert list (severity left-borders, relative time); detail sheet. styles.css: dark-first tokens mirroring `desktop/src/styles.css` values verbatim (`--bg:#1a1a1a`, `--sev-info:#7dd3fc`, `--sev-warning:#f59e0b`, `--sev-critical:#f87171`, text `#e6e6e6`, muted `#a8a8a8`, radius 8px).

- [ ] **Step 1: Failing tests** — store.ts against `fake-indexeddb` (bun add -d fake-indexeddb): put 3, ack 1 → ack flag set; snooze(1 min) → filtered from listAlerts; cap at 500 (write 502 → 500 remain, oldest evicted).
- [ ] **Step 2-4: red → implement → green** (`bun run test` in pager/).
- [ ] **Step 5: Build + manual smoke** — `bun run build`; `bun run preview` + browser devtools: subscribe flow shows subscription JSON; skip live-push verification (needs real keys — that's the E2E checklist in the continuation prompt).
- [ ] **Step 6: Commit** — `feat(pager): alert list, detail, ack/snooze, pairing view (FEAT-045)`

### Task 8: Desktop — ack/snooze parity on Notifications view

**Files:**
- Modify: `desktop/src/views/Notifications.tsx`
- Test: extend `desktop/src/__tests__/` (new `Notifications.ack.test.tsx`)

**Interfaces:**
- Consumes: the existing severity extraction (`severityOf`) and item shape in Notifications.tsx.
- Produces: per-item local state `acked`/`snoozedUntil` with "Acknowledge" and "Snooze 10m" buttons; acked items dim + strike; snoozed hidden until expiry; both purely local (no API).

- [ ] **Step 1: Failing test** — render an item, click Acknowledge → has `data-acked="true"` and dimmed class; Snooze → item hidden, a "Snoozed (n)" summary row appears. Follow the existing Header.test.tsx patterns (testing-library).
- [ ] **Step 2-4: red → implement → green** — `(cd desktop && bun run test)`.
- [ ] **Step 5: tsc gate** — `(cd desktop && bunx tsc --noEmit)` exit 0 (mandatory, BUG-881).
- [ ] **Step 6: Commit** — `feat(desktop): ack/snooze parity on notifications view (FEAT-045)`

### Task 9: Vision + roadmap bookkeeping

**Files:**
- Modify: `docs/PRODUCT-VISION.md` (principle 6 + tier table mobile row), `docs/roadmap/items/ROAD-064.yaml` (link FEAT-045)

- [ ] **Step 1:** principle 6 becomes: "**Open-source core, open-source pager** — The CLI is free (MIT). The pager (mobile + desktop alerting) is open source too. Paid tiers are desktop dashboard depth and managed conveniences." Tier table Mobile row: `React Native app → Installable PWA (Web Push); native shells via gomobile later | Open source | Pager v1 shipping`. 
- [ ] **Step 2:** `ccs roadmap link-issue ROAD-064 FEAT-045`.
- [ ] **Step 3: Commit** — `docs(vision): product-vision principle 6 pivot — open-source pager (FEAT-045, ADR-001)`

## Self-Review (done)

- Spec coverage: transport (T1-3), CLI (T4-5), PWA (T6-7), desktop parity (T8), vision/bookkeeping (T9), payload protocol (T2), pairing (T4+T7), pruning (T3), security constraints (Global). Live-status/actions/native = explicitly out of scope. ✓
- No placeholders: every step carries code or exact field/command specs; the two `go doc` confirm points are verification instructions, not missing content. ✓
- Type consistency: Store/Subscription/Payload/Severity/Dispatch/Sender names identical across tasks. ✓
