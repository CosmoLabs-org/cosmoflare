# CosmoDev-R2Go2 — ClaudeDesign Operator Guide

This file is your operating manual for running the `/claudedesign` workflow on this project. Read once, refer to it whenever you start a new thread or drop a new bundle.

> **Authoritative SOP**: `docs/standards/claudedesign-sop.md` (in the ClaudeCodeSetup repo). This file is a project-local operator quickstart — when in doubt, the SOP wins.

---

## What lives where

```
CosmoDev-R2Go2/
└── ClaudeDesign/                         # SINGLE ROOT — version-controlled
    ├── manifest.yaml                     # bundle registry (CCS-managed)
    ├── INSTRUCTIONS.md                   # this file
    ├── README.md
    ├── runbook.md                        # click-by-click pilot runbook
    ├── briefs/
    │   ├── 00-PROJECT-INSTRUCTIONS.md   # paste into Claude Design Project Instructions
    │   ├── 00-context/                  # shared across ALL platforms
    │   │   ├── constitution.md
    │   │   ├── design-system.md
    │   │   ├── animation-haptics.md
    │   │   ├── voice-and-tone.md
    │   │   ├── feature-inventory.md
    │   │   └── screen-list.md
    │   └── mobile/            # platform-specific brief overlays
    │       └── 01-*.md
    └── outputs/DESIGN-NNN-{slug}/        # one folder per dropped bundle
```

**Everything** (briefs, runbook, INSTRUCTIONS, outputs) lives under `ClaudeDesign/`. The v1 single-root layout matches `/claudedesktop` for mental-model consistency.

---

## Setup (one-time)

You'll do this once per project, then never again.

1. **Hand-author the brief pack.** Open `ClaudeDesign/briefs/` and fill in the context files. The shared `00-context/*.md` files are your project's design constitution — vision, personas, JTBDs, voice, design system, animations, features, screens. Take your time; these files inform every thread.
2. **Open Claude Design.** Either surface — `claude.ai/design` (web) or the Claude Desktop app's Design tab. Same product, two access points.
3. **Create a Project.** Name it `CosmoDev-R2Go2 Mobile` (e.g. "CosmoDev-R2Go2 Mobile" or "CosmoDev-R2Go2 Web"). One Project per platform track for fullstack; one Project total for single-platform projects.
4. **Paste Project Instructions.** Copy the entire `ClaudeDesign/briefs/00-PROJECT-INSTRUCTIONS.md` into the Project's Instructions field.
5. **Upload Project Files.** Upload all six `00-context/*.md` files via the Files sidebar. **Known quirk**: if Claude Design refuses `.md` (only accepts DOCX/PPTX/XLSX), append each file's content into the Instructions field after a `--- FILE: <name> ---` separator instead.
6. **Verify context loaded.** In a scratch chat, ask: *"What's our project's North Star and who are the three primary personas?"* If the canvas answers from the constitution, context is live. If it doesn't, re-check Files / Instructions.

---

## Per-thread workflow

For **each** thread (Thread 1 = foundation, Threads 2..N = feature surfaces):

1. **Author the brief.** For Thread 1, the brief is the platform-specific file at `ClaudeDesign/briefs/mobile/01-*.md`. For Thread N, you'll author `ClaudeDesign/briefs/mobile/NN-{slug}.md` AFTER Thread 1's bundle merges (the locked tokens become Thread N's starting constraints — never the other way around).
2. **Greenfield framing check.** Every brief MUST open with a variant of *"You are designing CosmoDev-R2Go2 from first principles. Do not mimic existing apps. Tokens are starting constraints; the screen language is yours to invent."* If you delete this clause, the SOP fails. Non-negotiable.
3. **New conversation in Claude Design.** Inside the Project, click "New conversation". Name it `NN — {short title}` (the chat name appears in handoff bundle metadata).
4. **Paste the brief.** Send it as your first message.
5. **Iterate.** Push back, ask for variants, request alternatives. Quality > speed; the bundle that ships defines a surface for months.
6. **Approve.** When the design system + screens are right, type the approval signal: *"This is the system. Lock it and let's hand off to Claude Code."*
7. **Click Handoff.** Click "Handoff to Claude Code" (top of canvas or in chat). Download the bundle ZIP.

### Sequencing rule

- **Thread 1** (foundation) is **sequential**. Run it first, alone, until its bundle is `merged`.
- **Threads 2..N** (feature surfaces) are **parallel-safe** — open as many tabs as you have threads, run them concurrently. They inherit the locked Automatic Design System from Thread 1.

Authoring a Thread N brief BEFORE Thread 1 ships means you reference starting-constraint tokens that may have evolved in Thread 1 — your downstream brief will be stale by the time it runs.

---

## Handoff bundle workflow

After downloading a bundle ZIP from Claude Design:

1. **Extract** into `ClaudeDesign/outputs/DESIGN-NNN-{slug}/`. Slug matches the thread name. First foundation bundle = `DESIGN-001-foundation/`. Re-runs of the same thread bump letter: `DESIGN-001b-foundation/`.
2. **Verify structure.** Bundle should contain `handoff-prompt.md`, `design-fetch-url.txt`, `README.md`. Optionally `raw-web/` (React/web reference — gitignored, never imported into native repos).
3. **Update manifest.** v0 is manual — append to `ClaudeDesign/manifest.yaml` under `bundles:`:
   ```yaml
   bundles:
     - bundle_id: DESIGN-001-foundation
       thread: foundation
       platform: mobile
       source_brief: ClaudeDesign/briefs/mobile/01-foundation-brief.md
       dropped_at: 2026-04-26T15:00:00Z
       status: dropped
       codebase_targets: []
       notes: ""
   ```
   v1 (FEAT-457) automates this with `ccs claudedesign ingest DESIGN-NNN`.
4. **Translate in Claude Code.** In a Claude Code session in this repo, prompt: *"Read `ClaudeDesign/outputs/DESIGN-NNN-{slug}/handoff-prompt.md` and the bundle's `README.md`. Implement the {scope} in `{path}` following our existing repo conventions ({framework}, {component-library}, theme shape). Update tokens if the bundle evolved them, but keep the API shape intact."*
5. **Review the diff** against the bundle's screen specs. Do tokens match (palette, spacing, typography)? Component boundaries respected? Motion / haptics match the brief? Iterate if drift is detected.
6. **Smoke test** on the target platform (iOS Simulator, Android emulator, browser, native binary).
7. **Commit + merge.** Update the manifest entry's `status: merged` and `codebase_targets: [{path: ..., kind: ...}, ...]`.

---

## Greenfield framing — non-negotiable

Every brief opens with the greenfield clause. This is the most-violated rule on first attempt because users default to "polish what exists". The brief templates emit it verbatim. If you delete it, Claude Design produces "CosmoDev-R2Go2 v1.1" instead of "CosmoDev-R2Go2 reimagined" — the entire pilot's value collapses.

**Common violations to grep for and delete:**
- "match the current vibe"
- "like Notion / Linear / Stripe"
- "use these screenshots as reference"
- "polish the existing screens"

Negative constraints ("do NOT mimic") work as well or better than positive ones ("do invent").

---

## Common pitfalls

| Pitfall | Why it bites | Fix |
|---------|-------------|-----|
| Briefs drift from real codebase after 6+ months | Brief pack is a frozen snapshot; live `theme.ts` / components evolve | Until Anthropic ships GitHub integration to Claude Design, manually attach key live files (`theme.ts`, representative components, types) to Project Files alongside the brief pack — see SOP § Live-codebase context for per-platform file lists |
| Files sidebar refuses `.md` uploads | Known Claude Design quirk | Append into Project Instructions after a `---` separator |
| Threads 2..N briefs authored too early | Reference stale starting-constraint tokens | Defer Thread N briefs until Thread 1's bundle is `merged` |
| Bundle imports raw-web/ TSX into native repos | Bundle's `raw-web/` is reference only; not Expo / native code | Use it to verify visual intent during diff review; never import |
| Multiple parallel Thread 1s (different platforms) running concurrently before any locks | Each thread evolves its own design system; reconciliation explodes | Run mobile Thread 1 alone first, let it lock, THEN open web-landing Thread 1 in the same Project; the Automatic Design System inherits |
| Project Instructions wording is approximate | Thread chats don't inherit Instructions field's exact wording — only its gist | Repeat greenfield framing verbatim in every brief, not just Project Instructions |
| Thread chat names are inconsistent | Bundle metadata uses chat name as identifier | Name every thread `{NN} — {short title}` |

---

## Quick command reference

```bash
# Status: bundles + next action
ccs claudedesign
ccs claudedesign status

# Re-scaffold (force overwrite — careful, blows away your hand-authored briefs)
ccs claudedesign init --type mobile --name CosmoDev-R2Go2 --force

# v1 commands (FEAT-457)
ccs claudedesign brief --thread N
ccs claudedesign ingest DESIGN-NNN
ccs claudedesign review DESIGN-NNN
ccs claudedesign integrate DESIGN-NNN
```

---

## GitHub repo connection (future)

When Anthropic ships GitHub integration to Claude Design (anticipated 2026 Q2-Q3, tracked in FEAT-460 / ROAD-571), this section will be replaced by an auto-generated "GitHub Repository Access" stanza in `ClaudeDesign/briefs/00-PROJECT-INSTRUCTIONS.md` pointing the canvas at live `theme.ts` / `types/` / `components/` paths. Until then, see the **Live-codebase context** section in the SOP for the manual workaround (attach files via Project Files, refresh per Thread N).

---

## Where to learn more

- `docs/standards/claudedesign-sop.md` — authoritative SOP (in the ClaudeCodeSetup repo)
- `ClaudeDesign/runbook.md` — click-by-click pilot runbook
- `plugins/internal/skills/claudedesign/skill.md` — skill quick reference
- `plugins/internal/sops/claudedesign-workflow/sop.md` — step-by-step workflow

When the SOP and this file disagree, the SOP wins — it's the canonical reference.
