# Architecture Map

## System Layers

```
┌─────────────────────────────────────────────┐
│  CLI Layer (cmd/)                            │
│  16 Cobra commands, global flags, output     │
│  helpers, PersistentPreRun validation        │
├─────────────────────────────────────────────┤
│  API Layer (internal/api/)                   │
│  Client (9 stub methods) + EnhancedClient    │
│  (real S3 SDK for uploads only)              │
├─────────────────────────────────────────────┤
│  TUI Layer (internal/tui/)                   │
│  Bubbletea Model/View/Update, 7 sections     │
│  (3 functional, 4 stubs)                     │
├─────────────────────────────────────────────┤
│  Interactive Layer (internal/interactive/)    │
│  18 files: setup, themes, animations,        │
│  accessibility, tutorials, input             │
├─────────────────────────────────────────────┤
│  CLI UX Layer (internal/cli/)                │
│  batch/, progress/, visual/, ux/, operations/│
├─────────────────────────────────────────────┤
│  Config Layer (internal/config/)             │
│  Multi-profile Viper config, validation      │
├─────────────────────────────────────────────┤
│  Disabled (5 packages)                       │
│  analytics, cicd, domain, migrate, api_old   │
└─────────────────────────────────────────────┘
```

## Module Boundaries
- `internal/` properly encapsulates all business logic (see agent-1-code-quality.md)
- Clean separation: cmd/ depends on internal/, never reverse
- **Issue**: cmd/ depends on concrete `*api.Client`, not interface (see agent-7-api-design.md)
- **Issue**: Two client construction patterns — legacy inline vs centralized getAPIClient() (see agent-1-code-quality.md)

## Data Flow: `r2go2 bucket list`
1. `main.go` → `cmd.Execute()` → Cobra routes to `runBucketList`
2. `runBucketList` → `getAPIClient()` → reads env/config → creates `*api.Client`
3. `client.ListBuckets()` → returns `[]*Bucket{}` (placeholder, empty)
4. Output formatted via `printJSON` or tabwriter depending on `--json` flag
5. **No HTTP request is ever made** (see agent-2-core-logic.md)

## Dependency Graph
- cmd/ → internal/api, internal/config, internal/cli/*, internal/utils
- internal/api → internal/config, internal/cli/visual, internal/utils, aws-sdk-go-v2
- internal/tui → internal/api, charmbracelet/bubbletea+lipgloss
- internal/interactive → internal/config, fatih/color
- **No circular dependencies detected**
