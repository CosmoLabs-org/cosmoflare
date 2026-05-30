# Template Variables

This document is the **contract** between the `templates/claudedesign/` template files and the Go init code (`internal/claudedesign/init.go`). Every `{{.Field}}` directive used in the templates appears here with its expected type, the templates that consume it, and an example value drawn from the CountCountries pilot.

The Go init code owns the struct shape (`TemplateData`) that satisfies this contract. When a new variable is introduced in any template, it must be added here AND to the Go struct simultaneously.

Templates are rendered with Go's `text/template` engine (NOT `html/template` — these are markdown / yaml outputs, not HTML).

**v1 layout note**: all template output paths are under `ClaudeDesign/` (single root). The v0 two-folder split (`docs/claudedesign/` + `ClaudeDesign/`) is retired. Any path examples in this doc use `ClaudeDesign/` as the destination prefix.

---

## Top-level identity

| Variable | Type | Used in | Example |
|----------|------|---------|---------|
| `.ProjectName` | `string` | all templates | `"CountCountries"` |
| `.ProjectDescription` | `string` | `00-PROJECT-INSTRUCTIONS.md` | `"a mobile-first travel-tracking app whose flagship feature is Schengen 90/180 visa-compliance monitoring"` |
| `.ProjectType` | `string` (enum: `mobile` \| `web-landing` \| `web-app` \| `desktop` \| `fullstack`) | `manifest.yaml` | `"mobile"` |
| `.CreatedDate` | `string` (ISO date) | `manifest.yaml` | `"2026-04-26"` |
| `.PlatformDir` | `string` (one of `mobile` \| `web-landing` \| `web-app` \| `desktop`) | `README.md`, `runbook.md` | `"mobile"` |
| `.PlatformLabel` | `string` (human-readable) | `runbook.md` | `"Mobile"`, `"Web Landing"`, `"Web App"`, `"Desktop"` |

## Brand & visual identity

| Variable | Type | Used in | Example |
|----------|------|---------|---------|
| `.BrandColorFamily` | `string` | `00-PROJECT-INSTRUCTIONS.md`, `design-system.md`, all platform briefs | `"blue"` |
| `.BrandRecognitionPhrase` | `string` | `design-system.md` | `"ocean / Schengen blue"` |
| `.PlatformFeel` | `string` | `00-PROJECT-INSTRUCTIONS.md` | `"iOS / Android"` (mobile), `"web"`, `"native macOS / Windows"` |
| `.PlatformNativeRule` | `string` | `design-system.md` | `"iOS uses iOS system colors as the foundation; Android uses Material 3 baseline. The app must not look like an iOS app on Android or vice versa."` |
| `.DefaultPalette` | `Palette` (struct, see below) | `design-system.md` | see Palette schema |
| `.SemanticStatusColors` | `[]StatusColor` (optional, see below) | `design-system.md` | Schengen Clear/Warning/Violation rows |
| `.SpacingScale` | `string` | `design-system.md` | `"4 / 8 / 12 / 16 / 24 / 32"` |
| `.RadiusRhythm` | `string` | `design-system.md` | `"10 inputs / 12 cards & buttons / 12 banners"` |
| `.DarkBackgroundChoice` | `string` | `design-system.md`, web-app and desktop briefs | `"deep navy"` |
| `.TypographyDefaultText` | `string` (paragraph) | `design-system.md` | full paragraph describing default type face approach |
| `.IconographyDefaultText` | `string` (paragraph) | `design-system.md` | full paragraph describing default icon library |

### `Palette` struct schema

```go
type Palette struct {
  Light []PaletteToken
  Dark  []PaletteToken
}

type PaletteToken struct {
  Token string // e.g. "background", "primary"
  Value string // e.g. "#f8fafc", "rgba(96,165,250,0.08)"
  Usage string // e.g. "Page background"
}
```

### `StatusColor` struct schema (optional)

```go
type StatusColor struct {
  Status   string // "Clear" / "Warning" / "Violation"
  Meaning  string // "0–75 days used"
  Light    string // "#10b981"
  Dark     string // "#34d399"
  GlowDark string // "rgba(52,211,153,0.15)"
}
```

If `.SemanticStatusColors` is nil/empty, the section is skipped via `{{if}}`.

## Personas, JTBDs, features

| Variable | Type | Used in | Example |
|----------|------|---------|---------|
| `.PrimaryPersonas` | `[]Persona` | `constitution.md` | `[{Tier: "Primary", Name: "Maya", Role: "the digital nomad", Age: "35", Description: "Lives between Lisbon, Mexico City, and Bali..."}]` |
| `.SharedTraits` | `[]string` | `constitution.md` | `["They're nervous about getting it wrong.", "They open the app at unpredictable moments...", "They will not read documentation."]` |
| `.JTBDs` | `[]JTBD` | `constitution.md` | `[{ID: "JTBD-1", Job: "Tell me right now whether...", Trigger: "Booking a trip", DoneWhen: "User sees a clear status..."}]` |
| `.FlagshipFeature` | `string` | `00-PROJECT-INSTRUCTIONS.md`, `web-landing/01-landing-page-brief.md` | `"Schengen 90/180 tracker"` |
| `.NorthStar` | `string` (paragraph) | `00-PROJECT-INSTRUCTIONS.md` | full paragraph stating the product's central question(s) |
| `.Vision` | `string` (paragraph) | `constitution.md` | full vision paragraph |
| `.EmotionalContext` | `string` (paragraph) | `constitution.md` | full paragraph on user's emotional state when opening the app |
| `.EmotionalQualities` | `[]HeadingBody` | `constitution.md`, `00-PROJECT-INSTRUCTIONS.md` (QualityBar) | `[{Heading: "Calm.", Body: "Never use red unless..."}]` |
| `.NonNegotiablePrinciples` | `[]HeadingBody` | `constitution.md` | `[{Heading: "Offline-first.", Body: "Every screen must work..."}]` |
| `.NotList` | `[]string` | `constitution.md` | `["Not a flight booking app.", "Not a travel-blog..."]` |
| `.Features` | `[]Feature` | `feature-inventory.md` | structured below |
| `.NotInList` | `[]string` | `feature-inventory.md` | mirrors `NotList` for the inventory's "what's not here" section |
| `.VisibilityRanking` | `[]HeadingBody` | `feature-inventory.md` | `[{Heading: "Schengen status", Body: "every screen, top of every dashboard"}]` |

### Persona / JTBD / Feature struct schemas

```go
type Persona struct {
  Tier        string // "Primary" | "Secondary" | "Tertiary" | ""
  Name        string // "Maya"
  Role        string // "the digital nomad"
  Age         string // "35"
  Description string // multi-sentence paragraph about who they are and what they need
}

type JTBD struct {
  ID       string // "JTBD-1"
  Job      string // the user-voice statement
  Trigger  string // when the job arises
  DoneWhen string // the success state
}

type HeadingBody struct {
  Heading string // bolded lead phrase ending in a period
  Body    string // the explanation that follows
}

type Feature struct {
  Name              string   // "Schengen 90/180 compliance"
  IsFlagship        bool     // true for the flagship; appends " — the flagship" to heading
  UserWants         string   // single-sentence user-voice
  MathOrLogicLabel  string   // "math", "logic", "rule" — the noun for the explainer subsection
  MathOrLogic       string   // explanation of the rule/math; empty string skips the subsection
  Surfaces          string   // comma-list of surfaces this shows up on
  EdgeCases         []string // bulleted edge cases
}
```

## Screens

| Variable | Type | Used in | Example |
|----------|------|---------|---------|
| `.ScreenSections` | `[]ScreenSection` | `screen-list.md` | `[{Letter: "A", Title: "Foundation surfaces", ThreadHint: "Thread 1", Screens: [...]}]` |
| `.CrossCutting` | `[]HeadingBody` | `screen-list.md` | dark mode, reduced motion, offline, multi-X cross-cutting concerns |
| `.OfflineOrConnectivityHeading` | `string` | `screen-list.md` (fallback) | `"Offline"` or `"Connectivity"` |
| `.OfflineOrConnectivityBody` | `string` | `screen-list.md` (fallback) | full sentence |
| `.MultiplicityHeading` | `string` | `screen-list.md`, `00-PROJECT-INSTRUCTIONS.md` | `"Multi-traveler scaling"` |
| `.MultiplicityKind` | `string` | `screen-list.md` | `"traveler"` |
| `.MultiplicityCases` | `string` | `screen-list.md`, `00-PROJECT-INSTRUCTIONS.md` | `"1, 2, 3, 4, or 5 travelers"` |
| `.MultiplicityBody` | `string` | `00-PROJECT-INSTRUCTIONS.md` | full sentence describing the rule |

### `ScreenSection` / `Screen` struct schemas

```go
type ScreenSection struct {
  Letter     string   // "A" | "B" | "C"
  Title      string   // "Foundation surfaces"
  ThreadHint string   // "Thread 1" — appears parenthetically after title; "" omits
  Screens    []Screen
}

type Screen struct {
  ID          string // "A.1"
  Title       string // "App shell / tab bar"
  Who         string
  JTBD        string
  Success     string
  Failure     string // "" omits the line
  EdgeCases   string // "" omits the line
  Constraints string // "" omits the line
}
```

## Voice & tone

| Variable | Type | Used in | Example |
|----------|------|---------|---------|
| `.BrandPersonality` | `string` | `voice-and-tone.md` | `"the calm border-control friend — competent, factual, never alarmist, never gamified"` |
| `.BrandVoice` | `BrandVoice` (struct, see below) | `voice-and-tone.md` | aggregate voice spec |
| `.VerbExamples` | `string` | `voice-and-tone.md` | `"Save, Delete, Add traveler, Log entry"` |
| `.DomainNomenclatureRule` | `string` | `voice-and-tone.md` | `"Country names: official short form (\"United Kingdom\", not \"UK\"...)."` |
| `.VoiceRubric` | `string` (paragraph) | `voice-and-tone.md` | `"Read the message aloud as if you were a customs officer who likes their job. If a customs officer wouldn't say it that way, rewrite."` |

### `BrandVoice` struct schema

```go
type BrandVoice struct {
  Adjectives        []AdjectiveDef    // e.g. {Adjective: "Calm", Definition: "never urgent unless..."}
  AntiPatterns      []AntiPattern     // {Pattern, Example, Why} for the "what it never sounds like" table
  MicrocopyPatterns []MicrocopyGroup  // grouped microcopy examples
  ToneRotation      []ToneRow         // per-surface tone notes
}

type AdjectiveDef struct {
  Adjective  string // "Calm"
  Definition string // "never urgent unless..."
}

type AntiPattern struct {
  Pattern string // "Alarmist"
  Example string // "⚠️ DANGER!..."
  Why     string // "The user is already nervous..."
}

type MicrocopyGroup struct {
  Section     string         // "Status messages (Schengen)"
  Examples    []MicrocopyEx  // {Label, Copy, Note}
  PatternNote string         // optional trailing paragraph
}

type MicrocopyEx struct {
  Label string // "Clear"
  Copy  string // "You have 90 days available..."
  Note  string // optional parenthetical
}

type ToneRow struct {
  Surface string // "Schengen hero"
  Notes   string // "Most reassuring..."
}
```

## Animation & haptics

| Variable | Type | Used in | Example |
|----------|------|---------|---------|
| `.MotionReferences` | `string` | `animation-haptics.md` | `"Apple Health/Fitness (smooth data counters, ring fills), Airbnb (shared transitions, bottom sheets)"` |
| `.HasHaptics` | `bool` | `animation-haptics.md` | `true` for mobile, `false` for web/desktop (renders pointer-feedback section instead) |
| `.WarningHapticTrigger` | `string` | `animation-haptics.md` (only when `HasHaptics`) | `"Approaching Schengen limit, data-loss warning"` |
| `.ExtraPrimitives` | `[]Primitive` | `animation-haptics.md` | `[{Name: "RingFill", Description: "Animates the Schengen ring..."}]` |
| `.PrimaryNavMotionRule` | `string` | `animation-haptics.md` | `"Tab bar icons. NativeTabs uses platform chrome."` |
| `.ExtraPerformanceNotes` | `string` | `animation-haptics.md` | optional extra performance bullet |

### `Primitive` struct schema

```go
type Primitive struct {
  Name        string // "RingFill"
  Description string // "Animates the Schengen ring on mount..."
}
```

## Quality bar (PROJECT-INSTRUCTIONS)

| Variable | Type | Used in | Example |
|----------|------|---------|---------|
| `.QualityBar` | `[]HeadingBody` | `00-PROJECT-INSTRUCTIONS.md` | optional override; if empty falls through to default fallback |
| `.TouchTargetMinimum` | `string` | `00-PROJECT-INSTRUCTIONS.md`, `design-system.md`, `mobile/01-foundation-brief.md` | `"44pt"` (mobile), `"24px pointer / 44px touch"` (web) |
| `.EmotionalQualityHeading` | `string` | `00-PROJECT-INSTRUCTIONS.md` (fallback) | `"Anxiety-aware"` |
| `.EmotionalQualityBody` | `string` | `00-PROJECT-INSTRUCTIONS.md` (fallback) | full sentence |
| `.OfflineOrPerformanceHeading` | `string` | `00-PROJECT-INSTRUCTIONS.md` (fallback) | `"Offline-first reality"` (mobile) or `"Performance-first"` (web) |
| `.OfflineOrPerformanceBody` | `string` | `00-PROJECT-INSTRUCTIONS.md` (fallback) | full sentence |
| `.ExampleComponentNameOne` | `string` | `00-PROJECT-INSTRUCTIONS.md` | `"StatusRing"` |
| `.ExampleComponentNameTwo` | `string` | `00-PROJECT-INSTRUCTIONS.md` | `"DayCounter"` |
| `.ExampleComponentNameThree` | `string` | `00-PROJECT-INSTRUCTIONS.md` | `"EntryRow"` |

## Threads

| Variable | Type | Used in | Example |
|----------|------|---------|---------|
| `.ThreadCount` | `string` | `00-PROJECT-INSTRUCTIONS.md` | `"four"` |
| `.ThreadPlan` | `string` (rendered markdown table or empty) | `00-PROJECT-INSTRUCTIONS.md` | optional; if empty, the fallback table renders |
| `.SecondaryThreadOne` | `string` | `00-PROJECT-INSTRUCTIONS.md`, `runbook.md` | `"Schengen Tracker"` |
| `.SecondaryThreadOneGoal` | `string` | `00-PROJECT-INSTRUCTIONS.md` | full sentence |
| `.SecondaryThreadTwo` | `string` | `00-PROJECT-INSTRUCTIONS.md`, `runbook.md` | `"Entries CRUD"` |
| `.SecondaryThreadTwoGoal` | `string` | `00-PROJECT-INSTRUCTIONS.md` | full sentence |
| `.SecondaryThreadThree` | `string` | `00-PROJECT-INSTRUCTIONS.md`, `runbook.md` | `"Multi-traveler + Attachments + Auth"` |
| `.SecondaryThreadThreeGoal` | `string` | `00-PROJECT-INSTRUCTIONS.md` | full sentence |
| `.RepoDescription` | `string` | `00-PROJECT-INSTRUCTIONS.md`, `mobile/01-foundation-brief.md`, `web-landing/01-landing-page-brief.md` | `"an Expo / React Native repository"` |
| `.TargetStack` | `string` | `README.md` | `"Expo / React Native (apps/mobile/)"` |
| `.CodebaseTarget` | `string` | `runbook.md`, `outputs-README.md` | `"apps/mobile/"` |

## Tracks (README)

| Variable | Type | Used in | Example |
|----------|------|---------|---------|
| `.Tracks` | `[]Track` | `README.md` | optional list of parallel tracks; empty falls back to single-track display |
| `.PrimaryTrack` | `string` | `README.md` (fallback) | `"Mobile track"` |
| `.PrimaryTrackOutput` | `string` | `README.md` (fallback) | `"React Native screens via Claude Code in apps/mobile/"` |
| `.FoundationSlug` | `string` | `outputs-README.md` | `"foundation"` |
| `.SecondThreadSlug` | `string` | `outputs-README.md` | `"schengen"` |
| `.ThirdThreadSlug` | `string` | `outputs-README.md` | `"entries"` |

### `Track` struct schema

```go
type Track struct {
  Name        string // "Mobile track"
  Description string // "01- … 05-."
  Output      string // "React Native screens via Claude Code in apps/mobile/."
}
```

## Mobile foundation brief

| Variable | Type | Used in | Example |
|----------|------|---------|---------|
| `.TabCount` | `string` (word form) | `mobile/01-foundation-brief.md` | `"four"` |
| `.TabList` | `string` | `mobile/01-foundation-brief.md` | `"Home, Trips, Calendar, Settings"` |
| `.Tabs` | `[]Tab` | `mobile/01-foundation-brief.md` | per-tab spec rows |
| `.SecondaryTabName` | `string` | `mobile/01-foundation-brief.md` | `"Trips"` |
| `.TertiaryTabName` | `string` | `mobile/01-foundation-brief.md` | `"Calendar"` |
| `.DarkModeRationale` | `string` | `mobile/01-foundation-brief.md` | `"The app is used at airports, in seat-back screens at night..."` |
| `.ListRowExamples` | `string` | `mobile/01-foundation-brief.md` | `"trips, travelers, attachments, settings"` |
| `.StatusPillStates` | `string` | `mobile/01-foundation-brief.md` | `"clear / warning / violation"` |
| `.DomainPickers` | `[]NameDescription` | `mobile/01-foundation-brief.md` | `[{Name: "Country picker", Description: "searchable list with flag + name + Schengen-status indicator"}]` |
| `.OnboardingJTBDs` | `string` | `mobile/01-foundation-brief.md` | `"JTBD-1 / JTBD-2 / JTBD-4"` |
| `.MobileStack` | `string` | `mobile/01-foundation-brief.md` | `"Expo, NativeWind for styling, and HugeIcons for in-screen icons"` |
| `.MobileBaseComponentsPath` | `string` | `mobile/01-foundation-brief.md` | `"apps/mobile/components/native/"` |
| `.MobileSharedUIPath` | `string` | `mobile/01-foundation-brief.md` | `"apps/mobile/components/ui/"` |

### `Tab` / `NameDescription` struct schemas

```go
type Tab struct {
  Name          string // "Home"
  IOSSymbol     string // "house" / "house.fill"
  AndroidSymbol string // "home" / "home"
  Description   string // "Schengen hero + per-traveler dashboard"
}

type NameDescription struct {
  Name        string
  Description string
}
```

## Web-landing brief

| Variable | Type | Used in | Example |
|----------|------|---------|---------|
| `.MarketingDomain` | `string` | `web-landing/01-landing-page-brief.md` | `"countcountries.com"` |
| `.PrimaryCTAVerb` | `string` | `web-landing/01-landing-page-brief.md` | `"download"` |
| `.LandingCenterpiece` | `string` | `web-landing/01-landing-page-brief.md` | `"infinite slow flag scroll"` |
| `.LandingCenterpieceDescription` | `string` | `web-landing/01-landing-page-brief.md` | `"an infinite, slow horizontal scroll of every flag in the world"` |
| `.CenterpieceConstraints` | `[]string` | `web-landing/01-landing-page-brief.md` | bulleted constraints; empty falls back to defaults |
| `.CenterpieceThesis` | `string` | `web-landing/01-landing-page-brief.md` | `"\"look at the world. Now count what you've seen.\""` |
| `.PrimaryCTAOptions` | `string` | `web-landing/01-landing-page-brief.md` | `"App Store + Google Play badges, plus a secondary \"Learn more\" or smooth-scroll arrow"` |
| `.ExtraSections` | `[]HeadingBody` | `web-landing/01-landing-page-brief.md` | optional extra page sections |
| `.PrivacyValuesCopy` | `string` | `web-landing/01-landing-page-brief.md` | `"encryption at rest, biometric lock, no third-party tracking, your data lives on your phone."` |
| `.PricingHint` | `string` | `web-landing/01-landing-page-brief.md` | `"free? freemium? one-time? to be decided"` |
| `.HeadlineExamples` | `string` | `web-landing/01-landing-page-brief.md` | `"\"Count the countries you've actually visited.\" / \"Every border, every day. Counted.\""` |
| `.AssetPolicy` | `string` | `web-landing/01-landing-page-brief.md` | `"Flags + UI screenshots only. Anything else feels generic."` |
| `.WebStack` | `string` | `web-landing/01-landing-page-brief.md` | `"React 19 + Tailwind 4 + Radix primitives in an existing apps/web/ Vite codebase"` |

## Web-app brief

| Variable | Type | Used in | Example |
|----------|------|---------|---------|
| `.WebAppStack` | `string` | `web-app/01-app-shell-brief.md` | `"React 19 + Vite + Tailwind 4 + Radix primitives"` |
| `.WebAppPrimitives` | `string` | `web-app/01-app-shell-brief.md` | `"Radix UI"` |

(The web-app brief also reuses `.ProjectName`, `.BrandColorFamily`, `.DarkBackgroundChoice` from the shared identity / palette set above.)

## Desktop brief

| Variable | Type | Used in | Example |
|----------|------|---------|---------|
| `.DesktopStack` | `string` | `desktop/01-shell-brief.md` | `"Tauri 2 + React + Tailwind"` (or `"Electron"`, `"SwiftUI + AppKit"`, `"native Win32 / WinUI 3"`) |
| `.PrimaryFeatureMenuName` | `string` | `desktop/01-shell-brief.md` | `"Travel"` |

(The desktop brief also reuses `.ProjectName`, `.BrandColorFamily`, `.DarkBackgroundChoice` from the shared identity / palette set above.)

## Manifest

| Variable | Type | Used in | Example |
|----------|------|---------|---------|
| `.ProjectType` | string (enum, see top) | `manifest.yaml` | `"mobile"` |
| `.CreatedDate` | string (ISO date) | `manifest.yaml` | `"2026-04-26"` |

---

## Helper functions used in templates

The following template helpers must be registered on `template.FuncMap`:

| Helper | Signature | Purpose |
|--------|-----------|---------|
| `add` | `func(a, b int) int` | Used for `{{ add $i 1 }}` to produce 1-indexed list numbering |

(`text/template` provides `index`, `len`, `range`, `if`, `with`, etc. natively. No HTML helpers are needed — outputs are markdown / yaml.)

---

## Defaults & invariants

The Go init code MUST:

1. Provide a sensible default for every variable — running `ccs claudedesign init --type mobile --name X` should produce **renderable, reasonable** output even if every advanced field (palette tokens, full personas, tracks list) falls back to defaults. The CountCountries example values above are reference points, not hardcoded defaults.
2. Validate `ProjectType` against the enum (`mobile` / `web-landing` / `web-app` / `desktop` / `fullstack`) before scaffolding.
3. Render the project's chosen platform brief(s) only; don't drop irrelevant platform brief files into `briefs/` for a single-platform project. (Multi-platform / fullstack drops all four.)
4. Treat `.HasHaptics` as `true` only for `mobile`; `false` for `web-landing`, `web-app`, and `desktop`.
5. Never inline CountCountries-specific phrases (Schengen, Maya, Daniel, Priya, etc.) into the rendered output unless the user explicitly authors them into the project metadata.

The greenfield framing — `"You are designing {{.ProjectName}} from first principles. Do not mimic..."` — is **load-bearing per pilot decision D6** and must appear verbatim in every platform brief.
