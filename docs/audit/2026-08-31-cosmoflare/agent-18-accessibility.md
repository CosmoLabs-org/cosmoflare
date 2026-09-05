# Agent 18: Accessibility

> Audit: 2026-08-31, cosmoflare v0.17.0, fresh mode. Report saved verbatim from agent output.

All evidence gathered. The audit surface is fully read: 4 TSX components, styles.css (15 lines — the entire stylesheet), index.html, built assets, SSE/API clients, tests, and git history. No HTML is served from the Go side (grep over `cmd/` and `internal/` returned zero embedded-HTML hits); the Go Bubbletea TUIs are terminal apps outside WCAG scope.

---

## Accessibility (WCAG 2.2 AA): 5/10

**Static pass — runtime verification pending. Focus traps, live regions, tab order, and var()-composited contrast cannot be verified from source alone. A green score is NOT WCAG certification.**

Audited surface: `desktop/src/` (Tauri + React 18 app shell — `App.tsx`, `Header.tsx`, `Dashboard.tsx`, `Notifications.tsx`, `styles.css`, `index.html`). This is the only HTML/DOM surface in the repo. The Go CLI and Bubbletea TUIs (`cmd/installer_tui/`, `cmd/domains_tui.go`) are terminal programs — WCAG does not apply; they are excluded.

**Sub-scores**:

| Dimension | Score | Notes |
|-----------|-------|-------|
| Color contrast | 5/10 | Base text `#e6e6e6` on `#1a1a1a` = 13.95:1 (passes AAA, SC 1.4.3). But every state color (`is-online`/`is-offline`, `is-active`) is undefined — the `cf-*` stylesheet was never written — so indicator contrast (SC 1.4.11) cannot pass by construction. |
| Keyboard navigation | 8/10 | All interactives are native `<button>`/`<select>`; zero `div onClick`, zero positive `tabindex`, no `outline: none` anywhere, so native focus rings survive (SC 2.1.1, 2.4.7 pass). Docked: no focus-visible styling layer exists at all. |
| Screen reader compatibility | 3/10 | No `aria-live`/`role="status"` anywhere; SSE-driven notifications, unread badge, and daemon-connection state mutate the DOM silently (SC 4.1.3 fail). Health online/offline state is absent from the accessible name (SC 4.1.2 fail). |
| Semantic HTML | 6/10 | Landmarks (`header`, `aside`, `main`) and real `<ul>/<li>` lists are used. But the document has no `<h1>` — it opens at `<h2>`, and the brand is a `<div>` (SC 1.3.1). |
| Forms & labels | 6/10 | Single form control (account `<select>`) has `aria-label="Account"` (SC 3.3.2 satisfied programmatically) but no visible label. No other inputs exist; error-association patterns untested by absence. |

### Critical Findings

1. **No live region for any dynamic content — the app's core output is silent to screen readers** — Every real-time update arrives via SSE and mutates the DOM with no announcement: new notifications prepend to the list (`Notifications.tsx:63-74`), the unread count changes (`Notifications.tsx:47-49`), and the "Connecting to daemon…" status swaps to the Dashboard (`App.tsx:146`). There is no `aria-live="polite"`, `role="status"`, or `role="log"` anywhere in `desktop/src/`. For a monitoring dashboard whose entire value is real-time delivery, this is a total functional failure for assistive-technology users — a screen reader user cannot know a notification ever arrived.
   - **Severity**: high — **WCAG 4.1.3 Status Messages (Level AA)** (also touches 4.1.2 for the connection state)
   - **File**: `desktop/src/views/Notifications.tsx:44`, `desktop/src/App.tsx:146`
   - **Fix**: Wrap the notification list in `role="log" aria-live="polite"` (or add an `aria-live="polite"` status line for new messages); give the unread badge `role="status"`; put "Connecting to daemon…" inside a `role="status"` region. Effort: small.

2. **Health online/offline state is invisible to assistive tech — color + a tooltip that never computes** — In `HealthDot` (`Header.tsx:56-76`), the visible state carrier is `<span className="cf-dot-mark" aria-hidden />` (hidden from AT, and once styled will differ only by dot color), while the state text lives in `title={`${label}: online`}`. Accessible-name computation takes the element's text content ("Systems") *before* the `title` fallback, so the tooltip text is never exposed — the announced name is just "Systems". `data-online="true"` is a data attribute and does not enter the accessibility tree (the code comment claiming "a11y tooling can assert state machine-readably" is wrong for real AT). Net: online vs offline is conveyed by nothing a screen reader receives, and by color alone once styles exist.
   - **Severity**: high — **WCAG 1.4.1 Use of Color (A)** and **4.1.2 Name, Role, Value (A)**
   - **File**: `desktop/src/components/Header.tsx:66-74`
   - **Fix**: Include the state in the accessible name: `aria-label={`Systems: ${online ? "online" : "offline"}`}` on the outer span (or a visually-hidden status suffix), and pair the future dot color with a non-color cue (icon/text). Effort: small.

3. **The component stylesheet does not exist — 30+ referenced `cf-*` classes have no definitions** — `styles.css` is 15 lines (`:root` font/colors, `body` margin, `main` padding) and was committed once in the scaffold commit (`e7401cd`) and never expanded. Every layout/state class the TSX references (`cf-app`, `cf-sidebar`, `cf-nav-item`, `cf-health-dot`, `is-active`, `cf-card-grid`, `cf-unread-badge`, …) resolves to nothing; the built asset `dist/assets/index-BsRT4Cat.css` (167 bytes) confirms this ships. Accessibility impact: interactive controls render at UA-default sizing (typically under 24 CSS px tall) with no `min-height`, failing the WCAG 2.2 AA target minimum; focus-visible treatment and indicator colors (SC 1.4.11 non-text contrast, 3:1 for the health dots and active nav state) are absent by construction.
   - **Severity**: high — **WCAG 2.5.8 Target Size (Minimum) (Level AA)**; blocks verification of 1.4.11 and 1.4.3 on state colors
   - **File**: `desktop/src/styles.css:1-15` (entire file), consumers at `desktop/src/App.tsx:124-152`, `desktop/src/components/Header.tsx:25-52`
   - **Fix**: Write the `cf-*` stylesheet: `min-height: 24px` (prefer 44px) + padding on `.cf-nav-item` and `.cf-mark-seen`, `:focus-visible` outlines, and state colors chosen against `#1a1a1a` at ≥3:1 for indicators and ≥4.5:1 for any colored text. Effort: medium.

4. **Active view not programmatically indicated in the sidebar** — `App.tsx:134-142` marks the current view only with the `is-active` class, which no AT announces. A screen reader user navigating the two nav buttons cannot tell which view is active. The sidebar is also an `<aside>` rather than a `<nav>` landmark.
   - **Severity**: medium — **WCAG 1.3.1 Info and Relationships (A)** / 4.1.2
   - **File**: `desktop/src/App.tsx:133-143`
   - **Fix**: Add `aria-current={view === item ? "page" : undefined}` on the button and change `<aside>` to `<nav aria-label="Views">`. Effort: small.

5. **Unread badge announces a bare number** — `Notifications.tsx:47-49` renders `<span ...>{unread}</span>`, so AT announces just "3" with no context, and it updates without a live region (see finding 1).
   - **Severity**: medium — **WCAG 1.3.1 Info and Relationships (A)**, 4.1.3
   - **File**: `desktop/src/views/Notifications.tsx:47-49`
   - **Fix**: `aria-label={unread === 1 ? "1 unread notification" : `${unread} unread notifications`}` on the badge span. Effort: small.

6. **No `<h1>`; heading structure starts at `<h2>`** — The brand is `<div className="cf-brand">⚡ Cosmoflare</div>` (`Header.tsx:26`), not a heading; the first heading on screen is the `<h2>` view title (`Dashboard.tsx:59`, `Notifications.tsx:46`), then `<h3>` card labels. The document skips level 1 entirely.
   - **Severity**: medium — **WCAG 1.3.1 Info and Relationships (A)**
   - **File**: `desktop/src/components/Header.tsx:26`, `desktop/src/views/Dashboard.tsx:59`
   - **Fix**: Make the brand the `<h1>`; keep view titles at `<h2>` and card labels at `<h3>`. Effort: small.

7. **Account switcher relies on `aria-label` alone; control colors are UA-default against a forced dark theme** — `Header.tsx:29-41`: the `<select>` has `aria-label="Account"` (3.3.2 passes programmatically) but no visible label — a sighted user gets only account names with no control name. Separately, `:root` forces `color:#e6e6e6` for the dark theme (`styles.css:3-4`) while the select's background stays UA-default light on some engines — a light-on-light risk that is statically unverifiable but plausible.
   - **Severity**: medium — **WCAG 1.4.3 Contrast (Minimum) (AA)** risk; 3.3.2 best-practice gap
   - **File**: `desktop/src/components/Header.tsx:29-41`, `desktop/src/styles.css:1-5`
   - **Fix**: Add a visible `<label htmlFor>` (or persistent "Account" text label) and set explicit `color`/`background-color` on `.cf-account-switcher` once the stylesheet exists. Effort: small.

8. **Dashboard card error state is the bare word "Error"** — `Dashboard.tsx:34`: `else if (error) body = "Error"` — no message, cause, or recovery action; same for "Loading…" with no live region. AT users hear an unexplained "Error" per card.
   - **Severity**: low — **WCAG 4.1.3 Status Messages (AA)** (degenerate case)
   - **File**: `desktop/src/views/Dashboard.tsx:33-35`
   - **Fix**: Render the error message (`String(error)` or a mapped message) with `role="alert"` on the card body. Effort: small.

9. **Notification fallback dumps raw JSON into the accessible tree** — `Notifications.tsx:70`: `{it.message ?? JSON.stringify(it.raw)}` — events without a `message` field are read as unparsed JSON key/value soup.
   - **Severity**: low — **WCAG 1.3.1 Info and Relationships (A)** (degraded fallback)
   - **File**: `desktop/src/views/Notifications.tsx:70`
   - **Fix**: Fall back to a human string ("Event received") plus optional `<details>` for the raw payload. Effort: small.

### What already passes (worth preserving)

- `index.html:2-6`: `lang="en"`, charset, viewport, `<title>` — SC 3.1.1 pass.
- Native controls everywhere: `<button>` (`App.tsx:135`, `Notifications.tsx:50`), `<select>` (`Header.tsx:29`) — SC 2.1.1 pass; no `div onClick`, no positive `tabindex`, no `tabindex={-1}` on interactives.
- No `outline: none` in any source file — default focus indicators survive — SC 2.4.7 pass.
- `aria-hidden` on the decorative dot mark (`Header.tsx:72`); real `<ul>/<li>` for notifications (`Notifications.tsx:63`); landmarks `header`/`main` used (`Header.tsx:25`, `App.tsx:144`).
- Base text contrast 13.95:1 — SC 1.4.3 pass with AAA headroom.

### Recommendations

- [ ] Add `role="log"` + `aria-live="polite"` to the notification list, `role="status"` to the unread badge and the "Connecting to daemon…" pane (effort: small)
- [ ] Fix `HealthDot` accessible name to include the online/offline state via `aria-label` (effort: small)
- [ ] Write the missing `cf-*` stylesheet with ≥24px (target 44px) hit areas, `:focus-visible` styles, and ≥3:1 indicator colors on `#1a1a1a` (effort: medium)
- [ ] Add `aria-current="page"` to the active sidebar button and use a `<nav>` landmark (effort: small)
- [ ] Promote the brand `<div>` to `<h1>` so the heading hierarchy starts at level 1 (effort: small)
- [ ] Give the unread badge and account switcher contextual accessible labels (effort: small)
- [ ] Add an `axe-core` pass to the existing Vitest suite (`vitest-axe` or `@axe-core/react` in dev) so these regressions fail CI instead of re-audit (effort: small)

### Roadmap Suggestions

- **Screen-reader-first pass on the desktop app** — Live regions, accessible health state, labeled badge, and h1 hierarchy; makes the real-time dashboard usable with VoiceOver/NVDA (priority: high, effort: small)
- **Implement the desktop design system stylesheet** — The `cf-*` layer (layout, sizing, focus, state colors tuned to WCAG 1.4.3/1.4.11/2.5.8) was scaffolded but never written; the shipped app renders unstyled (priority: high, effort: medium)
- **Automated accessibility gate in CI** — Add axe-core to the Vitest run plus a Playwright/WDEV axe sweep of the Tauri webview at session-end (priority: medium, effort: small)
- **Runtime WCAG verification session** — Static pass cannot certify: run VoiceOver + keyboard-only + contrast instrumentation against the styled app once the stylesheet lands (priority: medium, effort: medium)
