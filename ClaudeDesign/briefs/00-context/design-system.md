# {{.ProjectName}} Design System — Starting Constraints

These are the design-system tokens currently in use (or proposed for v0). Treat them as **starting constraints**, not a cage. Evolve where it serves the user; preserve where evolution would break recognition.

## Hard constraints (do not break)

- **Brand color family**: {{.BrandColorFamily}}. The app is recognizably "{{.BrandRecognitionPhrase}}." Specific shades are yours; the family is not.
- **Platform native feel**: {{.PlatformNativeRule}}.
- **Status color semantics**: green = success/clear, amber = warning, red = error/violation. These cannot be remapped (color blindness rules + cultural convention).
- **WCAG AA contrast** on all foreground/background pairs.
- **{{.TouchTargetMinimum}} minimum touch target** on every interactive element (mobile/tablet); pointer targets ≥ 24px on desktop.

## Soft constraints (evolve with rationale)

- Specific hex values (you may propose better shades; show old + new).
- Spacing scale (currently {{.SpacingScale}}; tighten or loosen as the screen demands).
- Border-radius rhythm (currently {{.RadiusRhythm}}; you may unify or vary).
- Type scale (you choose; meet body 14–16, label 12–13 floors on mobile, body 15–17 on desktop).

## Current palette — light

| Token | Value | Usage |
|---|---|---|
{{- range .DefaultPalette.Light }}
| `{{ .Token }}` | `{{ .Value }}` | {{ .Usage }} |
{{- end }}

## Current palette — dark

| Token | Value | Usage |
|---|---|---|
{{- range .DefaultPalette.Dark }}
| `{{ .Token }}` | `{{ .Value }}` | {{ .Usage }} |
{{- end }}

{{- if .SemanticStatusColors }}

## Semantic status colors (do not remap)

| Status | Meaning | Light | Dark | Glow (dark) |
|---|---|---|---|---|
{{- range .SemanticStatusColors }}
| {{ .Status }} | {{ .Meaning }} | `{{ .Light }}` | `{{ .Dark }}` | `{{ .GlowDark }}` |
{{- end }}

These states are the heartbeat of the app. The flagship feature thread should give them their own visual treatment (rings, gauges, halos — your call) but the color semantics are locked.
{{- end }}

## Surface archetypes — three only

| Archetype | Where | How it looks |
|---|---|---|
| **Overlay** | Tab bar, nav header, modals, sheets, command palette | Translucent / liquid glass — content scrolls behind |
| **Elevated** | List cards, stat cards, settings rows | Solid `surface1` + 1px `border` + radius from the rhythm |
| **Recessed** | Section backgrounds, muted groupings | Solid `surface3`, deeper tone |

No mixed variants ("tinted-accent bordered card with subtle shadow") — every surface picks one archetype. You may introduce a fourth archetype if and only if it has a distinct rendering rule and a distinct semantic role.

## Glass policy

Glass surfaces are reserved for **overlays that float over content the user is actually looking through** — scrolling UI or imagery. Nothing else. Glass over a flat opaque background is noise; it has nothing to blur.

Allowed: tab bar, nav header, modals/sheets, notification overlays, badges over imagery.

Not allowed: stat cards, form containers, dashboard cards on a flat background.

## Spacing

Current scale: `{{.SpacingScale}}`. Card padding 16. Card radius from the rhythm above. Input radius 10. Button radius 12.

You may propose a different scale (8-point grid? 4-point grid? per-screen?) but the rhythm should be consistent across the app.

## Dark-mode rules (non-negotiable)

- No pure black `#000000` anywhere — use {{.DarkBackgroundChoice}}.
- No pure white text — use warm off-white (`#f0f4ff` or your equivalent).
- No drop shadows in dark mode (invisible on dark) — use tinted alpha borders for elevation cues.
- Borders in dark mode are `rgba` with brand tint, not solid grays.

## Typography (open)

{{.TypographyDefaultText}}

Floors: body text ≥ 14px (16 preferred on mobile, 16–17 on desktop), labels ≥ 12px, never below 12 for user-readable content.

## Iconography

{{.IconographyDefaultText}}

## Motion floor

See `animation-haptics.md` for the full motion vocabulary. Floor: every interaction has a sub-16ms visual + (where applicable) haptic response. Spring presets: snappy / gentle / bouncy.

## What "evolve with rationale" looks like in practice

Acceptable proposal:
> *"I'd like to shift the warning amber from `#f59e0b` to `#e8a317` — the current shade clashes with the dark-mode primary when both appear on screen, which happens on the dashboard. Old: f59e0b. Proposed: e8a317. Reason: 1.4× contrast against the primary while preserving the semantic warmth."*

Unacceptable proposal:
> *"I'd like to use a different blue because it looks more modern."*
