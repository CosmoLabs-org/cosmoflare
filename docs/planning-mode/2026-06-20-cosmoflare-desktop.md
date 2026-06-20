---
created: "2026-06-20T17:44:43-03:00"
updated: "2026-06-20T17:44:43-03:00"
status: PLANNED
priority: high
origin: "/brainplan"
roadmap_ref: ROAD-063
brainstorm_ref: docs/brainstorming/2026-06-20-cosmoflare-desktop.md
title: "Cosmoflare Desktop (Tauri) — v1 Implementation Plan"
tags: [desktop, tauri, daemon, react, notifications]
deliverables:
  - id: P-01
    title: "internal/server + `cosmoflare serve`: HTTP server, token auth, stdout handshake, /healthz"
  - id: P-02
    title: "REST read endpoints (/accounts /zones /r2/buckets /workers /kv) over existing services"
  - id: P-03
    title: "SSE /events (metrics/notifications/status) + two-tier health (systems/cloudflare online)"
  - id: P-04
    title: "Tauri v2 scaffold in desktop/ + sidecar (externalBin) + build config"
  - id: P-05
    title: "Rust daemon lifecycle: spawn, handshake parse, health-poll state machine, kill, watchdog"
  - id: P-06
    title: "React+Vite+TS app shell + REST/SSE client hooks + two health indicators"
  - id: P-07
    title: "Multi-account read-only dashboard (zones/R2/Workers/KV cards)"
  - id: P-08
    title: "Real-time notifications panel (SSE notifications channel)"
  - id: P-09
    title: "Cross-platform packaging: per-triple Go sidecar + Tauri bundles"
requires_reading:
  - docs/brainstorming/2026-06-20-cosmoflare-desktop.md
  - internal/tui/datasource.go          # DataSource shape the daemon reuses
  - cmd/mcp.go                           # existing local-server command pattern
  - internal/keychain/keychain.go        # COSMOFLARE_NO_KEYCHAIN gate (hard constraint)
  - cmd/root.go                          # global AccountID/APIToken + printJSON helpers
---

# Cosmoflare Desktop (Tauri) — v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a cross-platform Cosmoflare desktop app — a Tauri shell that launches and supervises a local `cosmoflare serve` daemon and renders a read-only multi-account dashboard with real-time notifications.

**Architecture:** A new `cosmoflare serve` Go daemon exposes a localhost HTTP+SSE API over the existing service layer (zero new Cloudflare logic). The Tauri Rust shell spawns the daemon as a bundled per-OS sidecar, supervises its health, and serves a React+Vite+TS webview that talks to the daemon over REST (reads) and SSE (live metrics, notifications, health).

**Tech Stack:** Go 1.26 (`net/http`, `httptest`) · Tauri v2 (Rust) · React 18 + Vite + TypeScript + React Query · SSE (`text/event-stream`) · Vitest.

**Three waves:** Wave 1 (P-01..P-03) builds the daemon + fixes the API contract — it must land first. Wave 2 (P-04/P-05, Rust lifecycle) and Wave 3 (P-06..P-08, React UI) depend only on that contract and can run in parallel. Wave 4 (P-09) packages.

---

## File Structure

**Create (Go daemon):**
- `cmd/serve.go` — the `serve` cobra command (flags, handshake, graceful shutdown)
- `internal/server/server.go` — `Server` struct, router, token-auth middleware, `/healthz`
- `internal/server/server_test.go`
- `internal/server/rest.go` — REST read handlers delegating to services
- `internal/server/rest_test.go`
- `internal/server/sse.go` — SSE hub + `/events` handler + health tracking
- `internal/server/sse_test.go`

**Create (Tauri app, `desktop/`):**
- `desktop/src-tauri/tauri.conf.json`, `desktop/src-tauri/Cargo.toml`
- `desktop/src-tauri/src/main.rs`, `desktop/src-tauri/src/daemon.rs` (lifecycle)
- `desktop/src/main.tsx`, `desktop/src/App.tsx`, `desktop/src/api/client.ts`, `desktop/src/api/sse.ts`
- `desktop/src/components/Header.tsx` (account switcher + health dots), `Sidebar.tsx`
- `desktop/src/views/Dashboard.tsx`, `desktop/src/views/Notifications.tsx`
- `desktop/src/**/__tests__/*.test.tsx`
- `desktop/package.json`, `desktop/vite.config.ts`, `desktop/scripts/build-sidecar.sh`

**Verify before coding** (`requires_reading` above): the exact `DataSource` method names in `internal/tui/datasource.go`, how `cmd/mcp.go` builds its local server + reads creds, and the `printJSON`/global-creds helpers in `cmd/root.go`. Mirror those patterns.

---

# Wave 1 — Go daemon (API contract foundation)

## Task 1: server skeleton — HTTP, token auth, handshake, /healthz (P-01)

**Files:** Create `internal/server/server.go`, `internal/server/server_test.go`, `cmd/serve.go`

- [ ] **Step 1: Write the failing test** (`internal/server/server_test.go`)
```go
package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz_RequiresToken(t *testing.T) {
	s := New(Config{Token: "secret"})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	// no token → 401
	resp, _ := http.Get(srv.URL + "/healthz")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: got %d, want 401", resp.StatusCode)
	}

	// with token → 200 + systems_online true
	req, _ := http.NewRequest("GET", srv.URL+"/healthz", nil)
	req.Header.Set("Authorization", "Bearer secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("with token: %v, status %d", err, resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run test to verify it fails** — `GOWORK=off go test ./internal/server/ -run TestHealthz_RequiresToken -v` → FAIL (`undefined: New`).

- [ ] **Step 3: Implement** (`internal/server/server.go`)
```go
package server

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

type Config struct {
	Token    string
	Version  string
}

type Server struct {
	cfg       Config
	cfOnline  atomic.Bool // set by REST handlers on CF call success/failure
}

func New(cfg Config) *Server { return &Server{cfg: cfg} }

// SetCloudflareOnline records the result of the most recent Cloudflare call.
func (s *Server) SetCloudflareOnline(ok bool) { s.cfOnline.Store(ok) }

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	return s.authMiddleware(mux)
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+s.cfg.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"systems_online":   true,
		"cloudflare_online": s.cfOnline.Load(),
		"version":          s.cfg.Version,
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
```

- [ ] **Step 4: Run test to verify it passes** — same command → PASS.

- [ ] **Step 5: Implement `cmd/serve.go`** — a cobra command that: generates a random token (or reads `--token`), binds `net.Listen("tcp", "127.0.0.1:<port>")` (port from `--addr`, default `:0`), prints the handshake `{"addr":"127.0.0.1:<actual>","token":"<token>"}` as one JSON line to stdout, sets `COSMOFLARE_NO_KEYCHAIN=1` in its own env path, then `http.Serve(ln, s.Handler())`. Handle SIGINT/SIGTERM for graceful shutdown. Mirror `cmd/mcp.go`'s structure for cobra wiring and credential resolution. (No keychain — read creds from config/globals only.)

- [ ] **Step 6: Build + smoke** — `GOWORK=off go build ./... && go vet ./internal/server/`. Manual: `cosmoflare serve --addr 127.0.0.1:0 --token test &` then `curl -H 'Authorization: Bearer test' 127.0.0.1:<port>/healthz` → JSON.

- [ ] **Step 7: Commit** — `feat(serve): add cosmoflare serve daemon skeleton with token auth + /healthz (P-01)`

## Task 2: REST read endpoints (P-02)

**Files:** Create `internal/server/rest.go`, `internal/server/rest_test.go`; modify `internal/server/server.go` (register routes).

The endpoints delegate to the existing services / `DataSource`. **Inject the data source via an interface** so tests need no live credentials:

- [ ] **Step 1: Define the seam + failing test** (`internal/server/rest_test.go`)
```go
package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeData struct{ zones []map[string]any }

func (f fakeData) Zones() ([]map[string]any, error)     { return f.zones, nil }
func (f fakeData) Accounts() ([]map[string]any, error)  { return nil, nil }
func (f fakeData) R2Buckets() ([]map[string]any, error) { return nil, nil }
func (f fakeData) Workers() ([]map[string]any, error)   { return nil, nil }
func (f fakeData) KV() ([]map[string]any, error)        { return nil, nil }

func TestZonesEndpoint(t *testing.T) {
	s := New(Config{Token: "t"})
	s.SetData(fakeData{zones: []map[string]any{{"name": "example.com"}}})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/zones", nil)
	req.Header.Set("Authorization", "Bearer t")
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var out []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if len(out) != 1 || out[0]["name"] != "example.com" {
		t.Fatalf("unexpected body: %v", out)
	}
}
```

- [ ] **Step 2: Run → FAIL** (`undefined: SetData`, no `/zones` route).

- [ ] **Step 3: Implement** — add a `DataSource` interface (`Accounts/Zones/R2Buckets/Workers/KV`) to `rest.go`, a `SetData(DataSource)` setter on `Server`, register `/accounts /zones /r2/buckets /workers /kv` in `Handler()`, each calling the corresponding method, returning JSON. On error → 502 + `s.SetCloudflareOnline(false)`; on success → `s.SetCloudflareOnline(true)`.

- [ ] **Step 4: Run → PASS.**

- [ ] **Step 5: Wire the real adapter in `cmd/serve.go`** — implement the `DataSource` interface backed by the existing services (`NewZoneServiceFromCreds(AccountID, APIToken)`, the R2/Workers/KV services, and/or the `internal/tui` `DataSource`). Verify exact method names against `internal/tui/datasource.go` first.

- [ ] **Step 6: Build + commit** — `feat(serve): add REST read endpoints over existing services (P-02)`

## Task 3: SSE /events + two-tier health (P-03)

**Files:** Create `internal/server/sse.go`, `internal/server/sse_test.go`; register `/events`.

- [ ] **Step 1: Failing test** — assert `/events` sets `Content-Type: text/event-stream`, and that a published notification is written to a subscribed client as a `event: notifications\ndata: {...}\n\n` frame. Use `httptest` + a context with cancel to bound the read.
```go
func TestSSE_NotificationFrame(t *testing.T) {
	s := New(Config{Token: "t"})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/events", nil)
	req.Header.Set("Authorization", "Bearer t")
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q", ct)
	}
	go s.Publish("notifications", map[string]any{"msg": "hello"})
	buf := make([]byte, 256)
	n, _ := resp.Body.Read(buf)
	if !strings.Contains(string(buf[:n]), "notifications") || !strings.Contains(string(buf[:n]), "hello") {
		t.Fatalf("frame missing: %q", buf[:n])
	}
}
```

- [ ] **Step 2: Run → FAIL.**

- [ ] **Step 3: Implement** — an SSE hub (`map[chan event]struct{}` guarded by a mutex), a `Publish(channel string, data any)` method that fans out, and a `/events` handler that registers a client channel, sets the SSE headers, flushes per event (`http.Flusher`), and cleans up on `r.Context().Done()`. Add a `status` heartbeat that emits `{systems_online, cloudflare_online}` on change. Wire REST handlers to `Publish("status", ...)` when `cloudflare_online` flips.

- [ ] **Step 4: Run → PASS.**

- [ ] **Step 5: Build + commit** — `feat(serve): add SSE /events (metrics/notifications/status) + health (P-03)`

**Wave 1 checkpoint:** `GOWORK=off go test ./internal/server/ -v && go build ./...` all green. The API contract (handshake + REST shapes + SSE channels) is now fixed — Waves 2 and 3 can proceed in parallel.

---

# Wave 2 — Tauri Rust shell (lifecycle)

> Depends on: the `cosmoflare` binary producing the stdout handshake (Task 1, Step 5).

## Task 4: Tauri v2 scaffold + sidecar config (P-04)

**Files:** Create `desktop/` Tauri v2 project (`npm create tauri-app@latest -- --template react-ts` then prune), `desktop/src-tauri/tauri.conf.json`.

- [ ] **Step 1:** Scaffold Tauri v2 + React-TS template into `desktop/`. Set product name `Cosmoflare`, identifier `org.cosmolabs.cosmoflare`.
- [ ] **Step 2:** Declare the daemon as an `externalBin` sidecar in `tauri.conf.json` → `"externalBin": ["binaries/cosmoflare"]` (Tauri appends the target-triple). Add the `shell`/`process` capability scoped to the sidecar only.
- [ ] **Step 3:** `cd desktop && npm install && npm run tauri build --debug` builds (with a placeholder sidecar binary). Acceptance: `desktop/src-tauri/target/debug/` produces an app bundle. 
- [ ] **Step 4: Commit** — `feat(desktop): scaffold Tauri v2 app with sidecar config (P-04)`

## Task 5: Rust daemon lifecycle (P-05)

**Files:** Create `desktop/src-tauri/src/daemon.rs`; modify `main.rs`.

- [ ] **Step 1: Rust unit test for handshake parse** (`daemon.rs` `#[cfg(test)]`)
```rust
#[test]
fn parses_handshake_line() {
    let line = r#"{"addr":"127.0.0.1:54123","token":"abc"}"#;
    let h = parse_handshake(line).unwrap();
    assert_eq!(h.addr, "127.0.0.1:54123");
    assert_eq!(h.token, "abc");
}
```
- [ ] **Step 2: Run → FAIL** (`cargo test` in `src-tauri`).
- [ ] **Step 3: Implement** — `parse_handshake(&str) -> Result<Handshake>` (serde_json). Then a `DaemonManager` that: spawns the sidecar via Tauri's shell API with args `serve --addr 127.0.0.1:0 --token <random> --no-keychain` and env `COSMOFLARE_NO_KEYCHAIN=1`; reads stdout for the handshake line; stores `addr`+`token`; polls `GET /healthz` until 200 (timeout 10s, 200ms interval); on app `WindowEvent::CloseRequested` / exit, kills the child; a watchdog re-spawns on unexpected exit (3× backoff). Expose a `#[tauri::command] fn daemon_endpoint(state) -> {url, token}` for the webview.
- [ ] **Step 4: Run → PASS.** Manual smoke: `npm run tauri dev` → app launches, daemon spawns, health goes green.
- [ ] **Step 5: Commit** — `feat(desktop): supervise cosmoflare daemon lifecycle from Rust (P-05)`

---

# Wave 3 — React frontend

> Depends on: the Wave 1 API contract (can develop against a mock matching the contract; `daemon_endpoint` command from P-05 supplies the real URL+token at runtime).

## Task 6: app shell + API/SSE clients + health indicators (P-06)

**Files:** Create `desktop/src/api/client.ts`, `desktop/src/api/sse.ts`, `desktop/src/components/Header.tsx`, `desktop/src/App.tsx`, `__tests__/Header.test.tsx`.

- [ ] **Step 1: Failing test** (`Header.test.tsx`, Vitest + @testing-library/react)
```tsx
import { render, screen } from "@testing-library/react";
import { Header } from "../components/Header";

test("renders both health indicators", () => {
  render(<Header systemsOnline={true} cloudflareOnline={false} accounts={[]} />);
  expect(screen.getByTestId("health-systems")).toHaveAttribute("data-online", "true");
  expect(screen.getByTestId("health-cloudflare")).toHaveAttribute("data-online", "false");
});
```
- [ ] **Step 2: Run → FAIL** (`npx vitest run` in `desktop/`).
- [ ] **Step 3: Implement** — `client.ts` (fetch wrapper injecting `Authorization: Bearer <token>` from the `daemon_endpoint` Tauri command), `sse.ts` (an `EventSource`-like SSE hook subscribing to `metrics`/`notifications`/`status`), `Header.tsx` (account `<select>` + two indicator dots with `data-testid`/`data-online`), `App.tsx` (shell: Header + Sidebar + routed main pane; wires the `status` SSE channel to the header).
- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** — `feat(desktop): app shell, API/SSE clients, health indicators (P-06)`

## Task 7: multi-account dashboard (P-07)

**Files:** Create `desktop/src/views/Dashboard.tsx`, `__tests__/Dashboard.test.tsx`.

- [ ] **Step 1: Failing test** — render `<Dashboard>` with a mocked client returning 2 zones + 1 R2 bucket; assert cards show the counts.
- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement** — service cards (Zones, R2, Workers, KV) hydrated via React Query against `/zones` etc., then updated from the `metrics` SSE channel. Re-scope on account-switch (query key includes account id).
- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** — `feat(desktop): multi-account read-only dashboard (P-07)`

## Task 8: notifications panel (P-08)

**Files:** Create `desktop/src/views/Notifications.tsx`, `__tests__/Notifications.test.tsx`.

- [ ] **Step 1: Failing test** — mount the panel, push a `notifications` SSE event through the mock, assert the item renders and the unread badge increments.
- [ ] **Step 2: Run → FAIL.**
- [ ] **Step 3: Implement** — a panel subscribed to the `notifications` SSE channel with a scrollable history (capped, e.g. last 200) and an unread badge cleared on view.
- [ ] **Step 4: Run → PASS.**
- [ ] **Step 5: Commit** — `feat(desktop): real-time notifications panel (P-08)`

---

# Wave 4 — Packaging

## Task 9: cross-platform sidecar build + bundles (P-09)

**Files:** Create `desktop/scripts/build-sidecar.sh`; modify `tauri.conf.json` build hooks + repo `Makefile`.

- [ ] **Step 1:** `build-sidecar.sh` cross-compiles the Go daemon per target and names it for Tauri: `GOOS/GOARCH go build -o desktop/src-tauri/binaries/cosmoflare-<target-triple>[.exe] .` for `aarch64-apple-darwin`, `x86_64-apple-darwin`, `x86_64-pc-windows-msvc`, `x86_64-unknown-linux-gnu`.
- [ ] **Step 2:** Wire `build-sidecar.sh` as Tauri's `beforeBuildCommand` so the sidecar is present before bundling.
- [ ] **Step 3: Acceptance** — `cd desktop && npm run tauri build` produces a bundle for the host OS with the sidecar embedded; launching it brings the daemon online (manual smoke). `go build ./...` still green (the daemon is unchanged Go code).
- [ ] **Step 4: Commit** — `build(desktop): cross-platform sidecar build + Tauri bundles (P-09)`

**Final verification:** `GOWORK=off go test ./internal/server/ && go build ./...`; `cd desktop && npx vitest run && cargo test --manifest-path src-tauri/Cargo.toml`; manual `npm run tauri dev` → app launches, both health dots green, dashboard populates, a test notification appears. Update `docs/USAGE.md` with the `cosmoflare serve` command. Close the linked FEAT issue; mark ROAD-063 in progress/done.

---

## Self-Review

- **Spec coverage:** BR-01→Task1-3, BR-02→Task5, BR-03→Task1/3, BR-04→Task1(Step5)/Task5 (`--no-keychain`), BR-05→Task6, BR-06→Task7, BR-07→Task8, BR-08→Task4/9. All 8 brainstorm deliverables covered.
- **Placeholder scan:** Go daemon tasks carry real test+impl code. Rust/React tasks specify exact files, the key test, and concrete implementation requirements (Tauri/React boilerplate is generated by the scaffold, not hand-written line-by-line) — acceptance criteria are machine-checkable (cargo test / vitest / go test + manual smoke).
- **Type consistency:** `DataSource` interface (`Accounts/Zones/R2Buckets/Workers/KV`) is consistent across Task 2 test + impl + the `cmd/serve.go` adapter. Handshake shape `{addr, token}` is consistent between Task 1 (Go emits) and Task 5 (Rust parses). `daemon_endpoint` command (P-05) feeds `client.ts` (P-06).
- **Open risk:** exact `internal/tui` `DataSource` method names must be confirmed at implementation time (listed in `requires_reading`); the adapter in `cmd/serve.go` Step 5 maps to whatever those are.
