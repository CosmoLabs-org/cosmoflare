# Motion & Haptics — Starting Constraints

## Philosophy

**Premium and subtle.** Every animation serves a purpose — confirming an action, guiding attention, communicating state. Nothing decorative. Reference points: {{.MotionReferences}}.

Three principles:
1. Every interaction gets a response — visual + (where applicable) haptic, within 16ms.
2. Animation communicates, never decorates. If you can't explain why it moves, remove it.
3. The motion vocabulary is small. Three springs cover everything.

## Spring vocabulary (use these names in your designs)

| Preset | Damping | Stiffness | Mass | Use when |
|---|---|---|---|---|
| **snappy** | 15 | 300 | 1 | Interactive elements the user is touching/clicking. Must feel instant. (Button press, toggle, chip select.) |
| **gentle** | 20 | 120 | 1 | Content appearing on screen. Should feel natural, not rushed. (Card mount, modal enter, fade-in.) |
| **bouncy** | 12 | 180 | 1 | Positive outcomes. Slight overshoot communicates delight. (Save success, milestone reached.) |

For non-spring timings: fade-in 200–300ms ease-out, fade-out 150ms ease-in, slide-enter 250ms ease-out, stagger 50ms between items, number count-up 800ms ease-out.

{{- if .HasHaptics }}

## Haptic ladder (one action = one haptic)

| Interaction | Haptic | When |
|---|---|---|
| Tap (primary) | Light impact | Button press, card tap |
| Tap (destructive) | Medium impact | Delete, logout |
| Toggle | Light impact | Switch flip, checkbox |
| Selection | Selection | Tab switch, picker scroll, chip select |
| Success | Success notification | Save complete, sync done |
| Error | Error notification | Validation fail, network error |
| Warning | Warning notification | {{.WarningHapticTrigger}} |
| Threshold | Medium impact | Pull-to-refresh release |

Rules:
- One action = one haptic. Don't combine `tap` + `success` on a save button — fire `success` after the async completes.
- No haptics on passive events (scroll, swipe-in-progress, auto-refresh).
- The system "Reduce Haptics" setting is respected automatically.
{{- else }}

## Pointer feedback (desktop / web equivalents to haptics)

Without haptic hardware, communicate the same meaning through visual + audio cues:

| Interaction | Visual | Audio (optional, off by default) |
|---|---|---|
| Hover | 6px lift + soft shadow + 100ms ease | none |
| Click (primary) | scale 0.97 → spring back | none |
| Click (destructive) | scale 0.97 + brief tint flash | subtle "thud" |
| Toggle | colour state change + 150ms transition | none |
| Selection | selection ring + 100ms ease | none |
| Success | green check fade-in over the affected control | subtle ascending tone |
| Error | red border pulse (1×, 200ms) + inline error text | subtle descending tone |

Audio is **off by default**; expose a setting to enable it. Many users will never want it.
{{- end }}

## Animation primitives (these names are the design contract)

When you compose a screen, name the wrapping behaviors using these primitives so Claude Code can map them 1:1 in implementation:

| Primitive | What it does |
|---|---|
| `FadeIn` | Opacity 0 → 1 on mount. Optional delay. |
| `SlideUp` | TranslateY 20 → 0 + opacity fade on mount. |
| `ScalePress` | Wraps a Pressable; scales to 0.97 on press-in, springs back on release{{ if .HasHaptics }}, fires the assigned haptic{{ end }}. |
| `AnimatedNumber` | Smoothly interpolates a numeric value. |
| `Skeleton` | Shimmer placeholder during loading; cross-fades to content. |
| `StaggerList` | Renders children with sequential 50ms delays. |
{{- range .ExtraPrimitives }}
| `{{ .Name }}` | {{ .Description }} |
{{- end }}

If a screen needs motion the primitives can't express, propose a new primitive — name it, describe it, justify it. Don't invent one-off inline animation.

## What never moves

- Status colors (green/amber/red). They don't fade or pulse to draw attention. They're the truth.
- Critical numbers in the dashboard during scroll. They tick when data changes; they don't bounce.
- Primary navigation icons. {{.PrimaryNavMotionRule}}

## Reduced-motion path

When the user has "Reduce Motion" enabled in system settings (or `prefers-reduced-motion: reduce` on web):
- All `SlideUp` collapses to a 100ms `FadeIn` (no translation).
- `ScalePress` becomes a 50ms opacity dip (no scale).
- `AnimatedNumber` snaps to the final value (no interpolation).
- `Skeleton` shimmer becomes a static muted block.

Design as if both paths exist. The reduced-motion fallback should still feel intentional, not broken.

## Performance budget

- 60fps target, 55fps floor. Below 55fps, glass surfaces auto-degrade to flat tinted backgrounds (already wired in code; the visual fallback should be designed).
- Max 8 concurrent animations on screen.
- Stagger lists cap at 10 visible items; beyond that batch-appear instantly.
{{- if .ExtraPerformanceNotes }}
- {{ .ExtraPerformanceNotes }}
{{- end }}

## How to call out motion in your designs

In the canvas notes for any screen, list:
1. Which primitive(s) compose the motion.
2. Which {{ if .HasHaptics }}haptic{{ else }}feedback cue{{ end }} fires (and when).
3. Whether the motion is on mount, on press, on data change, or on transition.

Example:
> *"Hero ring fill: `gentle` spring on mount; `AnimatedNumber` for the counter at 800ms ease-out. {{ if .HasHaptics }}Haptic: none on mount, `success` if status crossed from warning → clear since last open.{{ else }}Audio: none. Visual: green check fade-in if status crossed warning → clear.{{ end }}"*
