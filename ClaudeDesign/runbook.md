# Claude Design — Runbook

Click-by-click instructions for running a Claude Design thread for **CosmoDev-R2Go2**.

## Prerequisites

- A Claude.ai paid account (Pro / Max / Team / Enterprise — Claude Design is not in the free tier).
- The brief pack in `ClaudeDesign/briefs/` (already scaffolded — fill in what's still placeholder).
- The drop zone at `ClaudeDesign/outputs/` (already scaffolded by `ccs claudedesign init`).

## Step 1 — Create the Project (once per platform)

1. Open **Claude Design**. Two equivalent entry points:
   - Web: `claude.ai/design`
   - Desktop app: open Claude → click the Design tab in the sidebar
2. Click **"New project"** in the sidebar.
3. Name it: **`CosmoDev-R2Go2 — Mobile`** (e.g. `CosmoDev-R2Go2 Mobile`, `CosmoDev-R2Go2 Web`, `CosmoDev-R2Go2 Desktop`).
4. In the project's **"Instructions"** field, paste the entire content of `ClaudeDesign/briefs/00-PROJECT-INSTRUCTIONS.md`.
5. In the project's **"Files"** sidebar, upload the six shared-context files from `ClaudeDesign/briefs/00-context/`:
   - `constitution.md`
   - `design-system.md`
   - `animation-haptics.md`
   - `feature-inventory.md`
   - `screen-list.md`
   - `voice-and-tone.md`

   *(If Claude Design only accepts DOCX/PPTX/XLSX uploads in the Files sidebar, append each file's content to the Project Instructions field after a `---` separator. Markdown renders in chat regardless.)*

6. Save the project.

## Step 2 — Run Thread 1 alone (sequential, foundation)

Thread 1 locks the design system that every later thread inherits. Run it solo first.

1. Inside the project, click **"New conversation"**.
2. Name the conversation: **`01 — Foundation`** (or the title of the platform's `01-*-brief.md`).
3. Paste the entire content of `ClaudeDesign/briefs/mobile/01-*-brief.md` as your first message.
4. Iterate with Claude Design until you approve the design system and base components. Use chat for structural changes, inline canvas comments for component tweaks.
5. **Approval signal:** type *"This is the system. Lock it and let's hand off to Claude Code."*
6. Click the **"Handoff to Claude Code"** button (top of canvas or in the chat).
7. Download the handoff bundle ZIP.
8. Drop it into `ClaudeDesign/outputs/DESIGN-001-foundation/` (create the subfolder if needed; extract the ZIP into it).

## Step 3 — Hand off DESIGN-001 to Claude Code

1. Open this repo in Claude Code (or `cd` into it from the terminal and run `claude`).
2. Tell Claude Code: *"Read `ClaudeDesign/outputs/DESIGN-001-foundation/handoff-prompt.md` and the bundle's README. Implement the design system + base components in `` following our existing repo conventions. Update tokens to match the canvas, but keep the API shape intact."*
3. Claude Code generates code, you review the diff, you merge to master.
4. Update `ClaudeDesign/manifest.yaml` with the bundle entry: `dropped_at`, `status: implemented`, `codebase_targets`.

## Step 4 — Run subsequent threads in parallel

Threads 2+ inherit the locked Automatic Design System from Thread 1, so they're safe to run concurrently. Open multiple browser tabs (or Desktop app windows) for parallel work.

| Tab | Thread name | Brief |
|---|---|---|
| 1 | `02 — Feature surface A` | `ClaudeDesign/briefs/mobile/02-*-brief.md` |
| 2 | `03 — Feature surface B` | `ClaudeDesign/briefs/mobile/03-*-brief.md` |
| 3 | `04 — Feature surface C` | `ClaudeDesign/briefs/mobile/04-*-brief.md` |

Iterate each thread independently. When approved, hand off → drop bundle into `outputs/DESIGN-002-...`, `…003…`, `…004…`.

> **Why we author later-thread briefs *after* Thread 1 lands:** Thread 1 may evolve the design system (e.g. shift a status hue, change the spacing scale). Authoring 2/3/4 briefs after 1 ships means they reference the *real* locked tokens, not the starting constraints. This keeps Claude Code's re-implementation faithful.

## Step 5 — Hand off DESIGN-002, -003, -004 to Claude Code

Same as Step 3, repeated per bundle. Each bundle's re-implementation is its own session in Claude Code. Merge each before starting the next to keep diffs reviewable.

## Step 6 — Optional reconciliation thread

After all platform threads are merged, optionally run a **Reconciliation Audit** thread:

1. Inside the same project, **"New conversation"** named `Reconciliation Audit`.
2. Paste a brief asking Claude Design to audit visual drift across the surfaces. Outputs are *delta notes* not a new bundle.
3. Apply deltas in Claude Code as targeted PRs.

## Multi-platform projects

If your project ships on multiple platforms (e.g. mobile + web-landing + web-app + desktop), create a **separate Claude Design Project per platform** but share the `00-context/` files across all of them. The Automatic Design System then inherits *within* a platform; cross-platform consistency is enforced by the shared context.

Optionally, run a **Cross-Platform Reconciliation** thread once all platforms have shipped at least Thread 1 to align tokens, motion vocabulary, and voice across surfaces.

## Quota management

Claude Design's iterations burn a **weekly quota** separate from Chat and Code. Tips:

- **Iterate per thread, not per atom.** Don't ask "make the button rounder" 12 times — bundle micro-feedback into one message.
- **Approve in batches** rather than nit-picking. The handoff bundle captures the canvas state at approval; small post-approval tweaks are cheaper to do in Claude Code.
- **Don't link the whole repo.** Claude Design supports linking codebases as input — for the brief-pack workflow we're using text briefs. If you do link code in the future, point at the relevant subdirectory only.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| Inline comment vanishes before Claude responds | Known Claude Design bug | Repeat the comment in chat. |
| "Compact layout mode" save error | Known bug | Save as standard layout. |
| Browser tab gets sluggish | Long thread | Start a sibling thread; reference the locked design system. |
| "Chat upstream error" | Transient | Retry. If persistent, file feedback to Anthropic. |
| Handoff bundle is 404 / missing files | Claude Design quirk | Re-run handoff; download immediately. |
| Desktop app and web show different state | Sync delay | Wait 30s or refresh. |

## What success looks like

By the end of the project's design phase:

- One DESIGN-XXX bundle per thread sits in `ClaudeDesign/outputs/` with the corresponding implementation merged in ``.
- CosmoDev-R2Go2's look-and-feel materially improves over the pre-Claude-Design state.
- A clear narrative is captured (in `docs/brainstorming/`) of what worked, what didn't, what the next pass needs.
- The shared `00-context/` files become living documents — updated as the product evolves so future threads start from the current truth.
