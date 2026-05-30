# Thread 1 — Foundation: Design System & Navigation (Mobile)

> **Read first:** `00-PROJECT-INSTRUCTIONS.md`. All Project Files in `00-context/` are loaded into this Project's context.

## Greenfield framing

You are designing **{{.ProjectName}}** mobile **from first principles**. Do not mimic existing apps in this category. Do not mimic any current {{.ProjectName}} UI. The constraints in `design-system.md` are starting points; the screen language is yours to invent.

The user explicitly does not want a polished version of what exists today. They want the best possible design for the goals in `constitution.md`.

## What this thread produces

This thread locks the **Automatic Design System** that subsequent feature threads will inherit. By the end, the Project should have:

1. A finalized color palette (light + dark) — proposed evolutions vs the starting tokens explained in chat.
2. A type scale.
3. A spacing rhythm.
4. The three surface archetypes (Overlay / Elevated / Recessed) rendered as named components.
5. The base component library (buttons, inputs, list rows, cards, badges, status pills, action sheets, banners).
6. The tab bar ({{.TabCount}} tabs: {{.TabList}}) — platform-native chrome.
7. Splash + first-run shell.
8. Empty / loading / error skeleton patterns.

## Goals (in priority order)

1. **Lock a palette that holds up in dark mode.** {{.DarkModeRationale}} Dark mode is not an afterthought.
2. **Lock the three surface archetypes.** Every future surface in later threads must map to one of three. No mixed variants.
3. **Establish a tab bar that feels native on both iOS and Android.** The flagship app feel starts here.
4. **Define base components that scale.** A button is a button across all surfaces. A list row works for {{.ListRowExamples}}.

## Hard constraints (do not break)

- **NativeTabs API limits the tab icons** to SF Symbols (iOS) + Ionicons / Material Symbols (Android). No custom React-component icons in the tab bar. (You may render SVG icons inside screens, just not in the tab bar.)
- **Five tabs maximum on Android** (BottomNavigationView limit). We're proposing {{.TabCount}}; don't exceed.
- **Status colors are semantic** (green/amber/red as defined in `design-system.md`). Don't remap.
- **WCAG AA contrast** on every text/background pair you propose.
- **{{.TouchTargetMinimum}} minimum touch target.**

## Soft constraints (open to evolution)

- Spacing scale.
- Border-radius rhythm.
- Specific hex values within the {{.BrandColorFamily}} / green / amber / red families.
- Type face (currently platform-native; you may propose Inter, Manrope, or similar if cross-platform and rationale is reading comfort).

## Tab bar — exactly {{.TabCount}} tabs

| Tab | SF Symbol (iOS) | Ionicons (Android) | What's there |
|---|---|---|---|
{{- range .Tabs }}
| {{ .Name }} | `{{ .IOSSymbol }}` | `{{ .AndroidSymbol }}` | {{ .Description }} |
{{- end }}

Propose alternative SF/Ionicons mappings if you have stronger ideas. Do not propose an additional tab — that's a different conversation.

## Base components needed (name them in the canvas)

Render each in light + dark, with all states (default / hovered / pressed / disabled):

- **Buttons**: `PrimaryButton` (filled), `SecondaryButton` (tinted), `TertiaryButton` (outlined), `GhostButton` (no chrome), `DestructiveButton`, `IconButton`. Sizes: small / medium (default) / large.
- **Inputs**: `TextInput`, `NumericInput`, `SearchInput`. With label + helper text + error state variants.
- **Switches**: native `Switch` styled per platform.
- **Selection**: `Chip` (selectable), `SegmentedControl` (≤3 options).
- **List rows**: `ListRow` with leading icon / leading avatar / leading badge, optional trailing chevron / trailing badge / trailing switch. Compact + comfortable density.
- **Cards**: `ElevatedCard`, `RecessedCard`. With + without header. Single + grouped.
- **Badges**: `LetterBadge` (circular initial), `CountBadge` (numeric, on tab/list), `StatusPill` ({{.StatusPillStates}}, with color + icon).
- **Action sheets / modals**: native `ActionSheet`, `FormSheet` (iOS form-sheet presentation), `Alert` (warning / error / info banners), `Toast`.
- **Date picker**: native single-date + date-range pickers.
{{- range .DomainPickers }}
- **{{ .Name }}**: {{ .Description }}.
{{- end }}
- **Attachment thumbnail**: small image with optional lock-icon overlay.

Each component should have a **named slot in the canvas** Claude Code will reference in the handoff.

## Screens to design in this thread

1. **Splash** — logo lockup, centered, brief.
2. **First-run intro** — 3 screens max, teaching what the app is for.
3. **Home tab shell** — empty (placeholder content for later feature threads to fill).
4. **{{.SecondaryTabName}} tab shell** — empty list state + the list row pattern.
5. **{{.TertiaryTabName}} tab shell** — empty state + the surface pattern for that tab.
6. **Settings tab shell** — settings list with grouped sections.
7. **Skeleton states** — loading variants for list, card, hero.

The flagship feature surfaces, the feature-specific components, and the deep flows are **not** in this thread — they live in later threads. Build the *containers* and the *navigation* here.

## Motion and haptics for this thread

- Tab switch: native chrome handles motion; haptic = `selection`.
- Splash → home: native screen transition; no extra animation.
- First-run swipe between intro screens: native pager; haptic = `selection`.
- Buttons: `ScalePress` to 0.97 with `snappy` spring; haptic = `tap` (`destructive` for the destructive button).
- List row press: `ScalePress` to 0.98; haptic = `tap`.
- Skeleton → content: 200ms cross-fade.
- Reduced-motion path: see `animation-haptics.md` § Reduced-motion path.

## Voice on this thread's screens

- Splash: no copy.
- First-run intro: ≤30 words per screen. Teach {{.OnboardingJTBDs}} in three slides.
- Empty states: see `voice-and-tone.md`.

## Deliverables for handoff

When the user approves this thread:

1. **Final canvas** with all components named, all screens in light + dark, all states represented.
2. **Design system summary** in chat: every token (color / type / spacing / radius), explicitly noting which were kept from the starting constraints and which were evolved (with rationale).
3. **Component inventory** with the canvas-name → intended-React-Native-component mapping.
4. **Handoff bundle** via the "Handoff to Claude Code" feature, packaged for {{.RepoDescription}}. The handoff should explicitly request:
   > *"Generate React Native components using {{.MobileStack}}. Tab bar uses NativeTabs from `expo-router/unstable-native-tabs`. Match the design system tokens to the existing repo's theme shape, evolving the values where the canvas evolved them. Build all base components in `{{.MobileBaseComponentsPath}}` and shared UI in `{{.MobileSharedUIPath}}`."*

## Done criteria

- Every base component is rendered, named, and stateful.
- The tab bar is platform-rendered on both iOS and Android (the canvas should show two side-by-side previews).
- Dark mode is rendered for every component and every screen.
- The Automatic Design System has captured the locked tokens.
- The user has approved the design and clicked "Handoff to Claude Code."

## What "approval" looks like

The user, in chat, says some version of:
> *"This is the system. Lock it and let's hand off to Claude Code."*

Until they say that, iterate. Don't ship a partial system to the handoff bundle.
