# ClaudeDesign/outputs/ — Drop Zone

This directory is where Claude Design **handoff bundles** land after each thread for **CosmoDev-R2Go2**.

## Workflow

1. You finish a thread in Claude Design (foundation, feature, or any platform-specific brief).
2. You click the **"Handoff to Claude Code"** button in Claude Design.
3. You download the bundle ZIP.
4. You drop it into the matching subfolder here:

```
outputs/
├── DESIGN-001-/          ← Thread 1 (foundation, lock-in pass)
├── DESIGN-002-/        ← Thread 2
├── DESIGN-003-/         ← Thread 3
└── DESIGN-NNN-{slug}/                       ← additional threads
```

5. You ping Claude Code in this repo: *"Read DESIGN-001- and implement in ."*

## What a bundle looks like

Per Claude Design's docs, each handoff bundle contains:

- `handoff-prompt.md` — copy-ready prompt for Claude Code
- `design-fetch-url.txt` — URL Claude Code uses to read screen specs
- `README.md` — tokens, component boundaries, screen specs, motion/haptic notes
- (Optional) `raw-web/` — ZIP of the React/web reference code (useful as visual reference even when the target stack is native)

## Rules

- **Never edit a bundle in place.** The bundle is the contract Claude Code reads. Edits go in ``, not here.
- **Keep raw-web/ as reference only** when targeting a non-web stack. Do not import the React/web TSX into a native app — it's there to verify Claude Code's re-implementation matches the visual intent.
- **One thread = one DESIGN-XXX folder.** If you re-run a thread (e.g. after iteration), increment: `DESIGN-001b-/`.
- **`.gitignore` rule:** the bundle ZIPs and `raw-web/` are large + AI-generated; the default `init` scaffolds a `.gitignore` that excludes them, while keeping `manifest.yaml`, `handoff-prompt.md`, and per-bundle `README.md` tracked. Adjust to taste.

## Manifest

`manifest.yaml` (sibling to this README) tracks each bundle: drop date, source thread, status (`dropped` / `read` / `implemented` / `merged`), and where the resulting code landed in ``.

## After Claude Code implements a bundle

The full `/claudedesign` SOP (in CCS) provides:

- `ccs claudedesign ingest DESIGN-XXX` — register the bundle in `manifest.yaml`, scan files
- `ccs claudedesign review DESIGN-XXX` — Opus agents validate the re-implementation matches the bundle's design intent
- `ccs claudedesign integrate DESIGN-XXX` — record where the bundle's components landed in ``

In v0 of the SOP, those steps happen by hand. The commands are stubbed and ship in a later release.
