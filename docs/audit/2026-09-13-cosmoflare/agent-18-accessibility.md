# Agent 18: Accessibility Compliance (WCAG 2.2 AA) — Cosmoflare Desktop

**Scope**: `desktop/src/` Tauri app shell (React 19 + TypeScript, plain token-driven CSS) — App.tsx, Header.tsx, Dashboard.tsx, Notifications.tsx, styles.css, index.html, main.tsx, api/client.ts, api/sse.ts, plus 4 test files (14 files read total).

**STATIC PASS — runtime verification pending.** Static pass — runtime verification pending. Focus traps, live regions, tab order, and var()-composited contrast cannot be verified from source alone. A green score is NOT WCAG certification.

**Detector facts: PARTIAL** — Impeccable detector unavailable in this project; all findings below derive from direct code reading and hand-computed WCAG relative-luminance contrast math (all color tokens are literal hex in `styles.css`, so ratios are exact, not `var()`-composited).

**Prior art found**: a prior a11y pass (BUG-036/037/038) is visible in the code and pinned by an axe-core test gate (`desktop/src/__tests__/a11y.test.tsx`). This audit validates that pass and finds the gaps it left.

## Accessibility: 8/10

**Sub-scores**:
| Dimension | Score | Notes |
|-----------|-------|-------|
| color_contrast | 8/10 | All 11 text/state pairs verified by hand ≥ 4.5:1 (text) or 3:1 (non-text). One real failure: interactive control borders ~1.19–1.36:1 (SC 1.4.11). Disabled-state opacity exempt. |
| keyboard_nav | 9/10 | Native `<button>`/`<select>` only, zero `tabIndex` in JSX, no `outline:none`, `:focus-visible` ring (8.14:1) on all interactive elements (styles.css:74), 44px nav targets. Runtime tab order unverified. |
| screen_reader | 7/10 | Strong naming/roles (aria-label, aria-current, role=status/alert/log), but two SC 4.1.3 failures: notifications silent off-tab, health transitions never announced. |
| semantic_html | 9/10 | `<header>`/`<nav aria-label>`/`<main>` landmarks, sequential h1→h2→h3, real `<ul>/<li>`, `lang="en"`, page `<title>`. No div soup. |
| forms_labels | 9/10 | Single form control (account `<select>`) labeled via `aria-label="Account"` (Header.tsx:36) — accepted method. Read-only app: no error-association or required-field patterns exist yet. |

### Verified contrast pairs (hand-computed, WCAG relative luminance)

| Element | Foreground | Background | Ratio | Required | Verdict |
|---|---|---|---|---|---|
| Body text | #e6e6e6 | #1a1a1a bg | 13.94:1 | 4.5:1 | PASS |
| Text on controls | #e6e6e6 | #2e2e2e raised | 10.88:1 | 4.5:1 | PASS |
| Muted labels/empty state | #a8a8a8 | #1a1a1a | 7.32:1 | 4.5:1 | PASS |
| Muted on cards | #a8a8a8 | #242424 surface | 6.53:1 | 4.5:1 | PASS |
| Card error text | #f87171 | #242424 surface | 5.61:1 | 4.5:1 | PASS |
| Unread badge (12px bold) | #1a1a1a | #f59e0b accent | 8.10:1 | 4.5:1 | PASS |
| Success dot (non-text, 1.4.11) | #4ade80 | #242424 header | 8.91:1 | 3:1 | PASS |
| Danger dot (non-text, 1.4.11) | #f87171 | #242424 header | 5.61:1 | 3:1 | PASS |
| Focus ring (non-text, 1.4.11) | #7dd3fc | #2e2e2e raised | 8.14:1 | 3:1 | PASS |
| Active-nav accent bar (1.4.11) | #f59e0b | #2e2e2e raised | 6.32:1 | 3:1 | PASS |
| **Control borders (1.4.11)** | **#3a3a3a** | **#2e2e2e / #242424** | **1.19:1 / 1.36:1** | **3:1** | **FAIL** |

The CSS header comment (styles.css:5-9) documents these ratios; my independent math confirms every claimed value. The comment's claim that state colors "clear 3:1" is also correct — the failure it misses is the border token (Finding 4).

### Critical Findings

1. **Incoming notifications are completely silent while any other view is active** — SC 4.1.3 Status Messages (Level AA). The only live regions for notifications (`role="log"` wrapper, Notifications.tsx:68, and the unread badge `role="status"`, Notifications.tsx:47-54) are children of the `Notifications` view, which App.tsx mounts conditionally:
   ```tsx
   // App.tsx:152-154
   {view === "Notifications" && (
     <Notifications items={notifications} unread={unread} onSeen={markSeen} />
   )}
   ```
   While the Dashboard is showing, a daemon notification (poll error, quota alert, Cloudflare dropout — the app's core monitoring signal) updates `notifications`/`unread` state but renders into nothing: no live region exists in the DOM, so screen reader users receive no announcement. The sidebar button (App.tsx:134-143) also exposes no unread count, so there is not even a discoverable passive indicator.
   - **Severity**: high (escalate)
   - **WCAG**: 4.1.3 Status Messages (AA)
   - **File**: `desktop/src/App.tsx:152-154`, `desktop/src/views/Notifications.tsx:68`
   - **Fix**: Keep an always-mounted, visually-hidden `aria-live="polite"` region at App level that renders the latest notification message (state already lives at App level via `useNotifications`, App.tsx:94 — only the render target is missing). Additionally surface the count on the nav item (`aria-label={`Notifications${unread ? `, ${unread} unread` : ""}`}`).

2. **Health state transitions are never announced** — SC 4.1.3 Status Messages (Level AA). The two `HealthDot`s mutate silently: `systemsOnline`/`cloudflareOnline` flips (App.tsx:73-74, 85-86) change an `aria-label` string and a dot fill, but the span has no live-region semantics, so assistive tech never re-reads it. A screen reader user learns the daemon died only if a card query happens to fail (role=alert). Compounding fragility: `aria-label` sits on a role-less `<span>` (implicit `generic` role), where name-from-author is prohibited by ARIA and support is browser-dependent — some AT announces the label, some falls back to the visible text "Systems" and silently loses the online/offline state:
   ```tsx
   // Header.tsx:69-78
   <span
     className={`cf-health-dot ${online ? "is-online" : "is-offline"}`}
     data-online={String(online)}
     title={`${label}: ${state}`}
     aria-label={`${label}: ${state}`}
   >
     <span className="cf-dot-mark" aria-hidden />
     {label}
   </span>
   ```
   - **Severity**: medium
   - **WCAG**: 4.1.3 Status Messages (AA); ARIA name computation on generic role
   - **File**: `desktop/src/components/Header.tsx:69-78`
   - **Fix**: Add `role="status"` to the outer span — this both legitimizes `aria-label` (status is nameable from author) and makes online/offline transitions announce politely. Header.test.tsx:42-52 already asserts the label string, so the fix is regression-covered.

3. **Online/offline state is color-only for sighted users** — SC 1.4.1 Use of Color (Level A). The visible text beside each dot is the static label ("Systems" / "Cloudflare", Header.tsx:77); the words "online"/"offline" exist only in `aria-label` and the `title` tooltip (hover-only, mouse-only). The state carrier is the dot fill (#4ade80 green vs #f87171 red, styles.css:157-163). The two hues do differ in luminance (≈1.6:1), which partially mitigates protan/deutan confusion, but hue remains the primary discriminator and nothing textual or shaped differentiates the states. Partial mitigation exists (daemon loss also produces the "Connecting to daemon…" pane, App.tsx:150), which is why this is medium, not high.
   - **Severity**: medium
   - **WCAG**: 1.4.1 Use of Color (A)
   - **File**: `desktop/src/components/Header.tsx:67-78`, `desktop/src/styles.css:157-163`
   - **Fix**: Render the state as visible small text after the label (e.g., "Systems · Offline" in `--muted`/`--danger`) or differentiate shape as well as hue (filled vs hollow ring). Keep the `aria-label` in sync.

4. **Interactive control boundaries fail non-text contrast** — SC 1.4.11 Non-text Contrast (Level AA). The `<select>` (styles.css:120-126) and "Mark read" button (styles.css:285-293) are delineated by a 1px `--border: #3a3a3a` on their `--surface-raised: #2e2e2e` / `--surface: #242424` fills — 1.19:1 to 1.36:1, far below the 3:1 required to identify the component's extent. The internal text passes 1.4.3 (10.88:1) so the control is *identifiable*, but its *boundary* is effectively invisible at low vision; the adjacent card surfaces differ from the page by only 1.12:1 (#242424 on #1a1a1a), so background separation provides no compensation. Related (noted, not failed): the nav hover state (#2e2e2e on #242424 sidebar, 1.14:1) is likewise imperceptible, but hover indication is supplementary.
   - **Severity**: medium
   - **WCAG**: 1.4.11 Non-text Contrast (AA)
   - **File**: `desktop/src/styles.css:20` (`--border` token), `:120-126`, `:285-293`
   - **Fix**: Introduce `--border-strong` (≥ #7a7a7a; #8a8a8a measures 3.93:1 on raised surface) for interactive control borders; keep `--border` for purely decorative card edges. Add the pair to the contrast table in the CSS header comment.

5. **Unread badge announces a bare number on change** — SC 4.1.3 / 4.1.2. The badge's live-region announcement comes from its text content changing ("2" → "3"), and live-region change announcements read content, not the accessible name — so the `aria-label="3 unread notifications"` (Notifications.tsx:51) may never be spoken during the update, leaving users hearing just "three".
   - **Severity**: low
   - **WCAG**: 4.1.3 Status Messages (AA), 4.1.2 Name, Role, Value
   - **File**: `desktop/src/views/Notifications.tsx:47-54`
   - **Fix**: Put visually-hidden descriptive text inside the badge (`<span className="sr-only"> unread notifications</span>` after the count) so the content change is self-describing; keep or drop the aria-label accordingly. Needs an `.sr-only` utility class (none exists yet in styles.css).

6. **Brand h1 accessible name contains the ⚡ emoji** — SC 1.1.1 (adjacent). "⚡" (Header.tsx:28) is announced by screen readers as "high voltage"/"electric", so the page's level-one heading reads "high voltage Cosmoflare".
   - **Severity**: low
   - **WCAG**: 1.1.1 Non-text Content (adjacent; announcement noise in the name)
   - **File**: `desktop/src/components/Header.tsx:28`
   - **Fix**: `<h1 className="cf-brand"><span aria-hidden="true">⚡ </span>Cosmoflare</h1>`.

7. **Dashboard loading→count transitions are not announced** — SC 4.1.3. Cards swap "Loading…" for a count (Dashboard.tsx:42) with no live semantics; only the error path announces (`role="alert"`, Dashboard.tsx:38, correctly). Similarly, the "Connecting to daemon…" `role="status"` pane (App.tsx:150) is replaced by the Dashboard without a "connected" announcement.
   - **Severity**: low
   - **WCAG**: 4.1.3 Status Messages (AA)
   - **File**: `desktop/src/views/Dashboard.tsx:42`, `desktop/src/App.tsx:146-151`
   - **Fix**: `role="status"` on the count paragraph so hydration completes out loud for SR users; optionally announce "Connected" once on first mount.

8. **No forced-colors / high-contrast-mode handling** — SC 1.4.11 / 1.4.1 risk on Windows. styles.css contains zero `@media` rules: no `forced-colors` fallback for the state dots (fills would flatten to a single system color, erasing the online/offline distinction for Windows High Contrast users) and no `prefers-contrast` bump for the failing border token.
   - **Severity**: low
   - **WCAG**: 1.4.11 / 1.4.1 conformance under forced-colors rendering
   - **File**: `desktop/src/styles.css` (whole file — no media queries)
   - **Fix**: `@media (forced-colors: active)` — give `.cf-dot-mark` a distinguishing `outline`/`forced-color-adjust: auto` pattern and pair with visible state text (Finding 3's fix covers the root cause).

### What already passes (evidence of the BUG-036/037/038 pass)

- Keyboard: only native `<button>`/`<select>` interactive elements (grep-verified: 2 onClick handlers, both on `<button>`; zero `tabIndex`; zero `outline:none`); `:focus-visible` 2px ring with offset on every interactive selector including `[tabindex]` (styles.css:74-77).
- Landmarks and headings: `<header>` (Header.tsx:27), `<nav aria-label="Views">` (App.tsx:133), `<main>` (App.tsx:145); h1 brand (Header.tsx:28) → h2 view titles (Dashboard.tsx:63, Notifications.tsx:46) → h3 card labels (Dashboard.tsx:34) — sequential, no skips; `<html lang="en">` and `<title>` (index.html:2,6).
- `aria-current="page"` on the active nav item (App.tsx:138); native select labeled `aria-label="Account"` (Header.tsx:36); dot mark correctly `aria-hidden` (Header.tsx:76).
- Target size (SC 2.5.8, 24px AA minimum): nav items `min-height: 44px` (styles.css:183), select and "Mark read" 32px (styles.css:121,286) — all pass; no adjacent small targets.
- Errors rendered as real messages in `role="alert"` (Dashboard.tsx:38), not the bare "Error" string; empty state is text, not a blank (Notifications.tsx:70).
- Test gate: axe-core with structural assertions for live regions, nav landmark, aria-current, connecting status (`desktop/src/__tests__/a11y.test.tsx`), plus BUG-037 label tests (Header.test.tsx:42-52) and BUG-038 live-log tests (Notifications.test.tsx:59-87). Disabled "Mark read" (opacity 0.5) is exempt from 1.4.3 as an inactive control.
- Motion: no animations exist anywhere (animation_instances: 0) — SC 2.3.3 not applicable.

### Runtime-verification backlog (cannot be concluded statically)

- Tab order across header → sidebar → main in the real WKWebView/WebView2; focus never obscured (SC 2.4.11) when `.cf-main` scrolls.
- 200% text zoom (SC 1.4.4): sizes are rem-based and panes scroll, but Tauri webview zoom availability (menu/shortcuts) must be confirmed at runtime.
- Actual live-region behavior of `role="log"`/`role="status"` in the shipping webview; `aria-label`-on-span exposure per screen reader (Finding 2).

### Recommendations

- [ ] Always-mounted app-level `aria-live="polite"` region announcing new notifications + unread count on the nav button label (effort: small)
- [ ] `role="status"` on HealthDot outer span — fixes announcement + aria-label fragility in one attribute (effort: small)
- [ ] Visible state text ("· Offline") or shape differentiation beside each health dot (effort: small)
- [ ] `--border-strong` token (≥ #7a7a7a) for interactive control borders; update CSS contrast table (effort: small)
- [ ] Visually-hidden text inside the unread badge + add `.sr-only` utility class (effort: small)
- [ ] `aria-hidden` wrapper on the ⚡ emoji in the h1 (effort: small)
- [ ] `forced-colors`/`prefers-contrast` media blocks for dots and control borders (effort: medium)
- [ ] Runtime a11y pass in the shipped webview: axe scan, tab order, 200% zoom, VoiceOver/NVDA walkthrough of health flips and notification arrivals (effort: medium)
- [ ] Explicit `type="button"` on the nav buttons (App.tsx:135) for consistency with Notifications.tsx:56 and future form-safety (effort: small)

### Roadmap Suggestions

- **SSE announcement bus (4.1.3)** — One always-mounted live region at App level that narrates notification arrivals and health transitions, replacing per-view live regions that unmount (priority: high, effort: small)
- **Forced-colors and high-contrast support** — Windows High Contrast mode flattens the state dots and borders; needs media-query fallbacks plus the visible state text (priority: medium, effort: medium)
- **Runtime accessibility verification harness** — Extend the existing vitest-axe gate with a WebDriver-driven pass in the real Tauri webview covering tab order, zoom, and live-region announcements (priority: medium, effort: medium)
