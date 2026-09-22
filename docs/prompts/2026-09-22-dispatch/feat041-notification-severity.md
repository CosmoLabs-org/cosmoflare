# FEAT-041 / ROAD-100 — Notification severity system + light theme (desktop)

Repo: /Users/gabstudio/PROJECTS/cosmoflare, desktop tier (Tauri + React) under `desktop/src/`. Audit evidence: docs/audit/2026-09-13-cosmoflare/agent-17-design-taste.md — every notification uses the same amber border (no severity encoding, styles.css:327) and the app is dark-only; weak for a paid tier.

## Files you own

- `desktop/src/styles.css` — severity tokens, light theme
- `desktop/src/types.ts` (or wherever the notification model lives — grep first; likely `api/client.ts` or a types file) — severity field
- `desktop/src/views/Notifications.tsx` + `desktop/src/components/` notification rendering — severity → border color mapping
- `desktop/src/App.tsx` — theme toggle wiring ONLY if a theme mechanism already exists; otherwise CSS `prefers-color-scheme` + a `data-theme` attribute toggle is acceptable
- test files under `desktop/src/__tests__/`

Do NOT touch api/sse.ts transport logic or src-tauri/ (no Rust changes).

## Work items

1. **Severity field**: add `severity: "info" | "warning" | "critical"` to the notification model with a safe default ("info") for older payloads (parse defensively — SSE payloads may omit it).
2. **Severity → visual mapping**: left-border color per severity (info: current blue/neutral, warning: amber, critical: red), plus a small severity dot/label in the notification header. Adjust the existing border rule at styles.css:~327.
3. **Light theme**: add a `[data-theme="light"]` token set overriding the `:root` custom properties (bg, surface, border, text, muted — mirror every token defined at styles.css:~15-25 including `--border`). Map severities to accessible shades in BOTH themes (contrast ≥ 4.5:1 for text-on-surface).
4. **Toggle**: a small sun/moon button in Header.tsx persisting `cosmoflare-theme` in localStorage, setting `document.documentElement.dataset.theme`; default follows `prefers-color-scheme` when unset. Keep it minimal — no settings modal.
5. **Tests**: extend `desktop/src/__tests__/` — severity default on missing field, severity class mapping, theme toggle flips data-theme + persists.

## Verify

`cd desktop && bun install && bun run build` (or `tsc --noEmit` + vite build per its package.json — READ desktop/package.json scripts and run the typecheck + build + test scripts it defines; typecheck MUST exit 0). All existing a11y tests stay green.

## Commit

`feat(desktop): notification severity levels + light theme (FEAT-041)` — conventional, no AI attribution.
