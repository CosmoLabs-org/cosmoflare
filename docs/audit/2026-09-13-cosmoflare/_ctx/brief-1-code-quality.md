export COSMOHOOKS_SUBAGENT=1

```
You are auditing the project "cosmoflare" for code quality. Your goal is to provide a comprehensive code quality assessment.

**Tech Stack**: CosmoLabs Tech Stack v2026.02 (updated 2026-02-23)

⚖️  Constitution:
  • Data Sovereignty: Users own their data — control, export, delete at any time
  • Mobile-First: Every screen matters — design for mobile first, scale up
  • Offline-First: The network is optional — local storage as primary data layer
  • Privacy by Default: No surveillance — minimal collection, opt-in analytics
  • Accessibility-First: Every user matters — WCAG 2.2 AA minimum
  • Open Standards: No vendor lock-in — interoperability and portability

Principles:
  • Type safety everywhere (TypeScript, Go, Rust, Swift)
  • Tailwind CSS for all web styling
  • shadcn/ui for React components
  • Bun over npm/pnpm/yarn
  • Go for CLI tools and backends
  • Vite for frontend builds
  • Native-first for platform apps (SwiftUI, Jetpack Compose, Tauri)
  • PostgreSQL for all relational databases

── android-native ──
  architecture        MVVM + Hilt
  data                Room
  dependency_manager  Gradle (Kotlin DSL)
  language            Kotlin 2.1
  local_storage       Room
  min_target          API 26 (Android 8.0)
  networking          Retrofit + OkHttp + Kotlin Coroutines
  sync                WorkManager for background sync
  testing             JUnit 5 + Espresso + Compose Test
  ui_framework        Jetpack Compose

── api-backend ──
  database    PostgreSQL
  docs        Swagger/OpenAPI
  framework   Gin 1.11 (simple) or Fiber 3 (fast)
  language    Go 1.25
  orm         GORM 1.31 or sqlx
  sync        Versioned sync endpoints + conflict resolution
  testing     go test + testify
  validation  go-playground/validator

── desktop-cross-platform ──
  alternative    Electron (when native Node.js access needed)
  animations     Framer Motion 12
  components     shadcn/ui
  e2e_testing    WebDriver
  framework      Tauri 2.10 (Rust + Web)
  frontend       React 19 + TypeScript 5 + Tailwind 4
  icons          Hugeicons + Lucide
  local_storage  SQLite (tauri-plugin-sql 2)
  state          Zustand 5
  sync           HTTP sync when online
  targets        macOS, Windows, Linux
  unit_testing   Vitest + jsdom

── go-cli ──
  cli_framework  Cobra 1.10
  config         Viper 1.21
  language       Go 1.25
  local_storage  SQLite (modernc.org/sqlite) or bbolt
  logging        zerolog
  sync           HTTP sync when online
  testing        go test + testify
  tui            Bubbletea 1.3 + Lipgloss 1.1

── ios-native ──
  architecture        MVVM
  data                SwiftData (or Core Data for complex needs)
  dependency_manager  Swift Package Manager
  fallback_ui         UIKit (legacy integration)
  language            Swift 6
  local_storage       SwiftData + CloudKit
  min_target          iOS 17
  networking          URLSession + async/await
  sync                CloudKit automatic sync or custom URLSession sync
  testing             XCTest + Swift Testing
  ui_framework        SwiftUI

── macos-native ──
  app_lifecycle  SwiftUI App
  data           SwiftData
  distribution   Mac App Store or notarized DMG
  language       Swift 6
  local_storage  SwiftData + CloudKit
  min_target     macOS 14
  sync           CloudKit automatic sync or custom URLSession sync
  testing        XCTest + Swift Testing
  ui_framework   SwiftUI

── mobile-cross-platform ──
  e2e_testing    Detox
  framework      React Native + Expo SDK 54
  language       TypeScript 5
  local_storage  WatermelonDB or MMKV (react-native-mmkv 3)
  navigation     Expo Router
  state          Zustand 5
  styling        NativeWind (Tailwind for RN)
  sync           Custom sync service + expo-background-fetch
  unit_testing   Jest + React Native Testing Library

── monorepo ──
  apps             apps/ (web, api, mobile, docs)
  ci               GitHub Actions with Turbo remote cache
  language         TypeScript 5
  package_manager  Bun Workspaces
  shared           packages/ (ui, config, tsconfig, utils)
  tool             Turborepo 2.8

── php-api ──
  database         PostgreSQL
  framework        Slim 4.15
  language         PHP 8.4
  orm              Doctrine or Eloquent (standalone)
  package_manager  Composer
  testing          Pest or PHPUnit

── php-fullstack ──
  database         PostgreSQL
  framework        Laravel 12
  frontend         Livewire + Alpine.js + Tailwind (TALL stack)
  language         PHP 8.4
  orm              Eloquent
  package_manager  Composer
  testing          Pest

── python-api ──
  async            uvicorn + asyncio
  database         PostgreSQL
  framework        FastAPI 0.129
  language         Python 3.13
  orm              SQLAlchemy 2.0
  package_manager  uv
  testing          pytest + httpx
  validation       Pydantic 2.12

── rust-cli ──
  async           tokio (if needed)
  cli_framework   clap 4.5
  error_handling  anyhow + thiserror
  language        Rust (stable)
  serialization   serde + serde_json
  testing         cargo test
  tui             ratatui 0.30

── web-frontend ──
  animations       Framer Motion 12
  build            Vite
  components       shadcn/ui
  data_fetching    TanStack Query 5
  e2e_testing      Playwright
  forms            React Hook Form + Zod 4
  framework        React 19
  icons            Hugeicons + Lucide
  language         TypeScript 5
  local_storage    IndexedDB (via Dexie.js 4)
  package_manager  Bun
  state            Zustand 5
  styling          Tailwind CSS 4
  sync             Service Workers + Background Sync API
  unit_testing     Vitest + jsdom

── web-fullstack ──
  animations       Framer Motion 12
  auth             NextAuth
  components       shadcn/ui
  data_fetching    TanStack Query 5
  database         PostgreSQL
  e2e_testing      Playwright
  forms            React Hook Form + Zod 4
  framework        Next.js 15 (App Router)
  icons            Hugeicons + Lucide
  language         TypeScript 5
  local_storage    IndexedDB (via Dexie.js 4)
  orm              Drizzle ORM
  package_manager  Bun
  state            Zustand 5
  styling          Tailwind CSS 4
  sync             Service Workers + Background Sync API
  unit_testing     Vitest + jsdom

── web-static ──
  components       shadcn/ui (via React islands)
  content          Astro Content Collections (MDX)
  e2e_testing      Playwright
  framework        Astro 5
  icons            Hugeicons + Lucide
  language         TypeScript 5
  package_manager  Bun
  styling          Tailwind CSS 4
  unit_testing     Vitest + jsdom
**File Tree**: cmd/object.go
cmd/limits.go
cmd/waf.go
cmd/bucket_notifications.go
cmd/queue_dlq.go
cmd/theme.go
cmd/installer_tui/main.go
cmd/installer_tui/main_test.go
cmd/decode_test.go
cmd/create.go
cmd/firewall.go
cmd/ratelimit_probe_test.go
cmd/cors.go
cmd/completion.go
cmd/dns.go
cmd/migrate_test.go
cmd/firewall_test.go
cmd/compare_test.go
cmd/pages_deployment_test.go
cmd/config.go
cmd/cors_test.go
cmd/mcp_test.go
cmd/pages_domain.go
cmd/list.go
cmd/switch_test.go
cmd/bucket.go
cmd/audit.go
cmd/d1_test.go
cmd/mcp_generate_test.go
cmd/redirects_test.go
cmd/queue_dlq_test.go
cmd/bucket_test.go
cmd/terraform.go
cmd/cache_run_test.go
cmd/backup.go
cmd/bucket_policy.go
cmd/auth_test.go
cmd/metrics.go
cmd/d1_migrations.go
cmd/auth.go
cmd/audit_test.go
cmd/cache_test.go
cmd/d1_import.go
cmd/terraform_test.go
cmd/d1.go
cmd/worker.go
cmd/domains_tui.go
cmd/d1_timetravel.go
cmd/images_test.go
cmd/sync.go
cmd/demo_test.go
cmd/queue.go
cmd/sync_guardrails_test.go
cmd/alerts.go
cmd/delete.go
cmd/pages_env.go
cmd/mcp_generate.go
cmd/kv.go
cmd/apply_test.go
cmd/switch.go
cmd/stream.go
cmd/wrangler_test.go
cmd/init_test.go
cmd/metrics_test.go
cmd/d1_export.go
cmd/export.go
cmd/queue_send_test.go
cmd/hyperdrive_test.go
cmd/ratelimit.go
cmd/create_test.go
cmd/pagerules.go
cmd/completion_test.go
cmd/serve.go
cmd/validate_test.go
cmd/email_test.go
cmd/domains_test.go
cmd/root_test.go
cmd/sync_test.go
cmd/copy.go
cmd/queue_send.go
cmd/status_test.go
cmd/cache.go
cmd/serve_test.go
cmd/queue_test.go
cmd/domains.go
cmd/waf_test.go
cmd/dev_test.go
cmd/analytics_test.go
cmd/templates_test.go
cmd/ai.go
cmd/domains_get.go
cmd/diff_test.go
cmd/bucket_domain.go
cmd/hyperdrive.go
cmd/alerts_test.go
cmd/domains_ns.go
cmd/pagerules_test.go
cmd/plugin_test.go
cmd/dns_test.go
cmd/diff.go
cmd/vectorize.go
cmd/pages_domain_test.go
cmd/plugin.go
cmd/wrangler.go
cmd/auth_permissions.go
cmd/pages_deployment.go
cmd/cost_test.go
cmd/images.go
cmd/zone.go
cmd/pages_test.go
cmd/email.go
cmd/domains_tui_test.go
cmd/mcp.go
cmd/domains_stats_test.go
cmd/export_test.go
cmd/watch_test.go
cmd/setup.go
cmd/redirects.go
cmd/auth_permissions_test.go
cmd/pages.go
**Entry Points**: main.go
cmd/root.go

## Your Mission

Audit the codebase for architecture quality, code patterns, tech debt, and test coverage. Score each dimension 1-10.

## Investigation Steps

1. **Architecture Review**
   - Read entry points and trace the main execution flow
   - Identify the architectural pattern (MVC, layered, monolithic, microservices, etc.)
   - Evaluate separation of concerns — are boundaries clean?
   - Check for god files (>500 lines) and god functions (>50 lines)
   - Assess extensibility — how hard is it to add a new feature?

2. **Code Patterns**
   - Check for consistent error handling patterns
   - Look for proper use of language idioms (Go interfaces, TS generics, etc.)
   - Identify code duplication (similar logic in multiple files)
   - Check naming conventions — are they consistent?
   - Look for magic numbers, hardcoded strings, missing constants

3. **Tech Debt**
   - Search for TODO, FIXME, HACK, XXX comments
   - Identify deprecated dependencies or patterns
   - Look for commented-out code blocks
   - Check for dead code (unreachable functions, unused exports)
   - Assess dependency health (outdated versions, known vulnerabilities)

4. **Test Coverage & Execution**
   - **Run actual tests** to get real pass/fail data (not just static analysis):
     - `bun test 2>&1 | tail -20` or `npm test 2>&1 | tail -20` (JS/TS)
     - `go test ./... 2>&1 | tail -20` (Go)
     - `pytest --tb=short 2>&1 | tail -20` (Python)
     - `flutter test 2>&1 | tail -20` (Flutter)
   - **Run type checking** if applicable:
     - `bun tsc --noEmit 2>&1 | tail -20` or `npx tsc --noEmit 2>&1 | tail -20` (TypeScript)
     - `go vet ./... 2>&1` (Go)
   - **Run linting** if configured:
     - `bun lint 2>&1 | tail -20` or `npx eslint . --max-warnings=0 2>&1 | tail -20`
     - `golangci-lint run 2>&1 | tail -20` (Go)
   - Find test files and assess what percentage of code has tests
   - Check test quality — are edge cases covered?
   - Look for integration tests vs only unit tests
   - Identify critical paths with no test coverage

5. **Project Management Health**
   - Does `docs/roadmap/` exist with active items?
   - Are issues tracked (`docs/issues/`)?
   - Is there a planning pipeline (`docs/planning-mode/`)?
   - Are there stale items (no activity in 60+ days)?
   - Score the project's self-awareness: does it know what needs fixing?

## Output Format

CRITICAL: Your output MUST follow the EXACT structure in output-format.md.
Do NOT use tables for Critical Findings — use numbered lists.
Do NOT skip the Sub-scores table.
Do NOT exceed 400 lines.

Include:
- Architecture score (1-10) with justification
- Code patterns score (1-10)
- Tech debt score (1-10) — higher means LESS debt
- Test coverage score (1-10)
- Overall Code Quality score (average)
- Critical findings with severity
- Actionable recommendations

MANDATORY: At the END of your output, emit a JSON block fenced with ```json:audit-result containing your structured results. Use sub_score keys: architecture, code_patterns, tech_debt, test_coverage, maintainability. See output-format.md for the full schema.
```

---

---

# Output Format Reference

# Agent Output Format

Each agent MUST structure their output in this exact format. This enables automated merging during synthesis.

---

## Per-Agent Output Template

```markdown
## [Agent Name]: [Overall Score]/10

**Sub-scores**:
| Dimension | Score | Notes |
|-----------|-------|-------|
| [Dim 1] | X/10 | [Brief justification] |
| [Dim 2] | X/10 | [Brief justification] |
| [Dim 3] | X/10 | [Brief justification] |

### Critical Findings

1. **[Finding Title]** — [Description of the issue, where it occurs, and why it matters]
   - **Severity**: high | medium | low
   - **File**: `path/to/file.ext:line`
   - **Fix**: [Suggested fix or approach]

2. **[Finding Title]** — [Description]
   - **Severity**: ...
   - **File**: ...
   - **Fix**: ...

### Recommendations

- [ ] [Actionable recommendation 1] (effort: small | medium | large)
- [ ] [Actionable recommendation 2] (effort: ...)
- [ ] [Actionable recommendation 3] (effort: ...)

### Roadmap Suggestions

Items the synthesis phase should create as roadmap entries:

- **[Title]** — [Brief description] (priority: high | medium | low, effort: small | medium | large)
- **[Title]** — [Brief description] (priority: ..., effort: ...)
```

## JSON Output (for synthesis parsing)

After writing the markdown report, each agent MUST also emit a structured JSON block at the very end of their output, fenced with ` ```json:audit-result ` and ` ``` `. This enables deterministic synthesis without LLM re-parsing:

```json
{
  "agent": "code-quality",
  "overall_score": 5,
  "sub_scores": {
    "architecture": 6,
    "code_patterns": 5,
    "tech_debt": 4,
    "test_coverage": 5
  },
  "critical_findings": [
    {
      "title": "No input validation on API endpoint",
      "severity": "high",
      "file": "src/api/handler.ts:45",
      "fix": "Add zod schema validation",
      "effort": "small"
    }
  ],
  "recommendations": [
    {
      "action": "Add unit tests for auth module",
      "effort": "medium",
      "priority": "high"
    }
  ],
  "roadmap_suggestions": [
    {
      "title": "Implement rate limiting",
      "description": "API has no rate limiting, vulnerable to abuse",
      "priority": "high",
      "effort": "medium"
    }
  ]
}
```

**Synthesis uses JSON for scoring, deduplication, and bug routing.** The markdown report is for human consumption; the JSON block is for the synthesis algorithm.

Agent-specific `sub_scores` keys:

| Agent | Sub-score Keys |
|-------|---------------|
| Code Quality | `architecture`, `code_patterns`, `tech_debt`, `test_coverage`, `maintainability` |
| Core Logic | `bug_detection`, `performance`, `edge_cases`, `security`, `data_integrity` |
| Design Taste | `visual_hierarchy`, `typography`, `color_system`, `spacing_layout`, `anti_slop`, `responsiveness_states` |
| Accessibility | `color_contrast`, `keyboard_nav`, `screen_reader`, `semantic_html`, `forms_labels` |
| Motion | `easing_curves`, `duration`, `property_choice`, `reduced_motion`, `motion_purpose` |
| Competitive | `feature_parity`, `differentiation`, `market_position`, `technical_moat`, `growth_potential` |
| Distribution | `ci_cd`, `packaging`, `deployment`, `multi_platform`, `developer_experience` |
| SEO/Content | `meta_tags`, `structured_data`, `technical_seo`, `content_strategy`, `discoverability`, `conversion_cta` |
| API Design | `api_design`, `versioning`, `auth_security`, `documentation` |
| Data/Schema | `schema_design`, `migrations`, `data_integrity`, `backup_recovery` |
| Infrastructure | `containerization`, `orchestration`, `iac_quality`, `security` |
| Documentation | `completeness`, `accuracy`, `onboarding`, `api_docs` |
| Document Analysis | `classification`, `structure`, `completeness`, `actionability` |

---

## Scoring Rubric

Scores must be **comparable across projects and audits**. Use the general guidelines first, then the per-dimension calibration tables to anchor your score.

### General Scale

| Score | Meaning | Calibration |
|-------|---------|-------------|
| 9-10 | Excellent | Industry best practices, CI/CD green, >80% coverage, no high-severity findings |
| 7-8 | Good | Solid foundation, minor gaps, would pass most code reviews without blocking issues |
| 5-6 | Adequate | Functional but has significant gaps a senior engineer would flag in review |
| 3-4 | Poor | Major issues — missing tests, broken builds, security holes, or UX blockers |
| 1-2 | Critical | Fundamental problems requiring overhaul — barely functional or dangerous to ship |
| 0 | Missing | Capability does not exist (no tests, no CI, no docs, etc.) |

### Per-Dimension Calibration

**Code Quality:**
| Score | What it looks like |
|-------|--------------------|
| 9-10 | Clean architecture, consistent patterns, <5 TODOs, >80% test coverage, all lints pass |
| 7-8 | Good separation of concerns, some tech debt but managed, 50-80% coverage |
| 5-6 | Works but has god files, inconsistent patterns, 20-50% coverage, unresolved TODOs |
| 3-4 | No architecture, copy-paste code, <20% coverage, failing lints |
| 1-2 | Single file monolith, no tests, no error handling |

**Core Logic:**
| Score | What it looks like |
|-------|--------------------|
| 9-10 | All edge cases handled, no known bugs, input validated, performance profiled |
| 7-8 | Core paths solid, some edge cases unhandled, minor performance concerns |
| 5-6 | Happy path works, error handling incomplete, some known bugs |
| 3-4 | Crashes on unexpected input, race conditions, data loss possible |
| 1-2 | Core functionality broken, security vulnerabilities, data corruption risks |

**Design (Taste/Accessibility/Motion):**
| Score | What it looks like |
|-------|--------------------|
| 9-10 | Polished UI, WCAG AA compliant, responsive, fast loads, clear CTAs |
| 7-8 | Good design, minor accessibility gaps, mostly responsive |
| 5-6 | Functional UI, inconsistent styling, some mobile issues, no loading states |
| 3-4 | Confusing navigation, poor accessibility, broken on mobile, no error states |
| 1-2 | Unusable without instructions, no responsive design, critical accessibility failures |

**Distribution:**
| Score | What it looks like |
|-------|--------------------|
| 9-10 | Automated CI/CD, multi-platform, one-click release, store-ready |
| 7-8 | CI exists, packaging works, release process documented but manual steps |
| 5-6 | CI partially set up, manual packaging, single platform |
| 3-4 | No CI, manual builds, no versioning strategy |
| 1-2 | Cannot build from fresh clone, no documented release process |

**SEO/Content:**
| Score | What it looks like |
|-------|--------------------|
| 9-10 | Full meta tags, structured data, sitemap, blog with keyword strategy |
| 7-8 | Good meta tags, some structured data, docs site exists |
| 5-6 | Basic meta tags, no structured data, README only |
| 3-4 | Missing meta tags, no Open Graph, poor README |
| 1-2 | No web presence, no documentation, no discoverability |

**Competitive Position:**
| Score | What it looks like |
|-------|--------------------|
| 9-10 | Clear moat, unique features competitors lack, strong community |
| 7-8 | Competitive feature set, some unique angles, growing adoption |
| 5-6 | Feature parity on basics, missing key differentiators |
| 3-4 | Behind competitors on core features, no clear value proposition |
| 1-2 | Competitors dominate, no differentiators, unclear why users would choose this |

---

## Severity Definitions

| Severity | Criteria |
|----------|----------|
| **high** | Bugs causing data loss, security vulnerabilities, blocking release, or broken core functionality |
| **medium** | UX issues, performance problems, missing common features, or inconsistent behavior |
| **low** | Style issues, minor improvements, nice-to-haves, or cosmetic problems |

## Rules

1. Every finding MUST have a severity level
2. Every finding MUST reference a specific file when possible
3. Scores MUST be justified — no score without explanation
4. Recommendations MUST be actionable (not "improve testing" but "add unit tests for X using Y framework")
5. Keep output concise — aim for 200-400 lines per agent, not 1000+
6. The JSON audit-result block MUST appear at the end of every agent output
7. Use the per-dimension calibration tables to anchor scores — a "7" should mean the same thing across different projects

## CRITICAL: Read Code, Don't Re-Discover Structure

The pre-computed context below already tells you the file tree, tech stack, metrics, and lint results.
Do NOT waste tokens re-discovering these via Glob/Grep. Instead:

1. **Spend 80% of your tokens READING actual source code** at specific file:line locations
2. **Read at least 10 files** and produce findings referencing at least 5 specific file:line locations
3. **Include code snippets** in your analysis — show the actual problematic code, not just describe it
4. **Trace execution paths** — follow function calls, check error handling, verify data flow
5. **Every claim must have evidence** — cite the exact file, line, and code that supports your finding

Your report should read like a senior engineer who actually READ the code, not a tool that scanned metadata.

## Pre-Computed Context (DO NOT re-discover — go straight to reading code)

Working directory: /Users/gabstudio/PROJECTS/cosmoflare

Project context (ccs audit context):
```json
{
  "project_name": "cosmoflare",
  "version": "0.26.0",
  "tech_stack": "CosmoLabs Tech Stack v2026.02 (updated 2026-02-23)\n\n⚖️  Constitution:\n  • Data Sovereignty: Users own their data — control, export, delete at any time\n  • Mobile-First: Every screen matters — design for mobile first, scale up\n  • Offline-First: The network is optional — local storage as primary data layer\n  • Privacy by Default: No surveillance — minimal collection, opt-in analytics\n  • Accessibility-First: Every user matters — WCAG 2.2 AA minimum\n  • Open Standards: No vendor lock-in — interoperability and portability\n\nPrinciples:\n  • Type safety everywhere (TypeScript, Go, Rust, Swift)\n  • Tailwind CSS for all web styling\n  • shadcn/ui for React components\n  • Bun over npm/pnpm/yarn\n  • Go for CLI tools and backends\n  • Vite for frontend builds\n  • Native-first for platform apps (SwiftUI, Jetpack Compose, Tauri)\n  • PostgreSQL for all relational databases\n\n── android-native ──\n  architecture        MVVM + Hilt\n  data                Room\n  dependency_manager  Gradle (Kotlin DSL)\n  language            Kotlin 2.1\n  local_storage       Room\n  min_target          API 26 (Android 8.0)\n  networking          Retrofit + OkHttp + Kotlin Coroutines\n  sync                WorkManager for background sync\n  testing             JUnit 5 + Espresso + Compose Test\n  ui_framework        Jetpack Compose\n\n── api-backend ──\n  database    PostgreSQL\n  docs        Swagger/OpenAPI\n  framework   Gin 1.11 (simple) or Fiber 3 (fast)\n  language    Go 1.25\n  orm         GORM 1.31 or sqlx\n  sync        Versioned sync endpoints + conflict resolution\n  testing     go test + testify\n  validation  go-playground/validator\n\n── desktop-cross-platform ──\n  alternative    Electron (when native Node.js access needed)\n  animations     Framer Motion 12\n  components     shadcn/ui\n  e2e_testing    WebDriver\n  framework      Tauri 2.10 (Rust + Web)\n  frontend       React 19 + TypeScript 5 + Tailwind 4\n  icons          Hugeicons + Lucide\n  local_storage  SQLite (tauri-plugin-sql 2)\n  state          Zustand 5\n  sync           HTTP sync when online\n  targets        macOS, Windows, Linux\n  unit_testing   Vitest + jsdom\n\n── go-cli ──\n  cli_framework  Cobra 1.10\n  config         Viper 1.21\n  language       Go 1.25\n  local_storage  SQLite (modernc.org/sqlite) or bbolt\n  logging        zerolog\n  sync           HTTP sync when online\n  testing        go test + testify\n  tui            Bubbletea 1.3 + Lipgloss 1.1\n\n── ios-native ──\n  architecture        MVVM\n  data                SwiftData (or Core Data for complex needs)\n  dependency_manager  Swift Package Manager\n  fallback_ui         UIKit (legacy integration)\n  language            Swift 6\n  local_storage       SwiftData + CloudKit\n  min_target          iOS 17\n  networking          URLSession + async/await\n  sync                CloudKit automatic sync or custom URLSession sync\n  testing             XCTest + Swift Testing\n  ui_framework        SwiftUI\n\n── macos-native ──\n  app_lifecycle  SwiftUI App\n  data           SwiftData\n  distribution   Mac App Store or notarized DMG\n  language       Swift 6\n  local_storage  SwiftData + CloudKit\n  min_target     macOS 14\n  sync           CloudKit automatic sync or custom URLSession sync\n  testing        XCTest + Swift Testing\n  ui_framework   SwiftUI\n\n── mobile-cross-platform ──\n  e2e_testing    Detox\n  framework      React Native + Expo SDK 54\n  language       TypeScript 5\n  local_storage  WatermelonDB or MMKV (react-native-mmkv 3)\n  navigation     Expo Router\n  state          Zustand 5\n  styling        NativeWind (Tailwind for RN)\n  sync           Custom sync service + expo-background-fetch\n  unit_testing   Jest + React Native Testing Library\n\n── monorepo ──\n  apps             apps/ (web, api, mobile, docs)\n  ci               GitHub Actions with Turbo remote cache\n  language         TypeScript 5\n  package_manager  Bun Workspaces\n  shared           packages/ (ui, config, tsconfig, utils)\n  tool             Turborepo 2.8\n\n── php-api ──\n  database         PostgreSQL\n  framework        Slim 4.15\n  language         PHP 8.4\n  orm              Doctrine or Eloquent (standalone)\n  package_manager  Composer\n  testing          Pest or PHPUnit\n\n── php-fullstack ──\n  database         PostgreSQL\n  framework        Laravel 12\n  frontend         Livewire + Alpine.js + Tailwind (TALL stack)\n  language         PHP 8.4\n  orm              Eloquent\n  package_manager  Composer\n  testing          Pest\n\n── python-api ──\n  async            uvicorn + asyncio\n  database         PostgreSQL\n  framework        FastAPI 0.129\n  language         Python 3.13\n  orm              SQLAlchemy 2.0\n  package_manager  uv\n  testing          pytest + httpx\n  validation       Pydantic 2.12\n\n── rust-cli ──\n  async           tokio (if needed)\n  cli_framework   clap 4.5\n  error_handling  anyhow + thiserror\n  language        Rust (stable)\n  serialization   serde + serde_json\n  testing         cargo test\n  tui             ratatui 0.30\n\n── web-frontend ──\n  animations       Framer Motion 12\n  build            Vite\n  components       shadcn/ui\n  data_fetching    TanStack Query 5\n  e2e_testing      Playwright\n  forms            React Hook Form + Zod 4\n  framework        React 19\n  icons            Hugeicons + Lucide\n  language         TypeScript 5\n  local_storage    IndexedDB (via Dexie.js 4)\n  package_manager  Bun\n  state            Zustand 5\n  styling          Tailwind CSS 4\n  sync             Service Workers + Background Sync API\n  unit_testing     Vitest + jsdom\n\n── web-fullstack ──\n  animations       Framer Motion 12\n  auth             NextAuth\n  components       shadcn/ui\n  data_fetching    TanStack Query 5\n  database         PostgreSQL\n  e2e_testing      Playwright\n  forms            React Hook Form + Zod 4\n  framework        Next.js 15 (App Router)\n  icons            Hugeicons + Lucide\n  language         TypeScript 5\n  local_storage    IndexedDB (via Dexie.js 4)\n  orm              Drizzle ORM\n  package_manager  Bun\n  state            Zustand 5\n  styling          Tailwind CSS 4\n  sync             Service Workers + Background Sync API\n  unit_testing     Vitest + jsdom\n\n── web-static ──\n  components       shadcn/ui (via React islands)\n  content          Astro Content Collections (MDX)\n  e2e_testing      Playwright\n  framework        Astro 5\n  icons            Hugeicons + Lucide\n  language         TypeScript 5\n  package_manager  Bun\n  styling          Tailwind CSS 4\n  unit_testing     Vitest + jsdom",
  "file_tree": [
    "cmd/object.go",
    "cmd/limits.go",
    "cmd/waf.go",
    "cmd/bucket_notifications.go",
    "cmd/queue_dlq.go",
    "cmd/theme.go",
    "cmd/installer_tui/main.go",
    "cmd/installer_tui/main_test.go",
    "cmd/decode_test.go",
    "cmd/create.go",
    "cmd/firewall.go",
    "cmd/ratelimit_probe_test.go",
    "cmd/cors.go",
    "cmd/completion.go",
    "cmd/dns.go",
    "cmd/migrate_test.go",
    "cmd/firewall_test.go",
    "cmd/compare_test.go",
    "cmd/pages_deployment_test.go",
    "cmd/config.go",
    "cmd/cors_test.go",
    "cmd/mcp_test.go",
    "cmd/pages_domain.go",
    "cmd/list.go",
    "cmd/switch_test.go",
    "cmd/bucket.go",
    "cmd/audit.go",
    "cmd/d1_test.go",
    "cmd/mcp_generate_test.go",
    "cmd/redirects_test.go",
    "cmd/queue_dlq_test.go",
    "cmd/bucket_test.go",
    "cmd/terraform.go",
    "cmd/cache_run_test.go",
    "cmd/backup.go",
    "cmd/bucket_policy.go",
    "cmd/auth_test.go",
    "cmd/metrics.go",
    "cmd/d1_migrations.go",
    "cmd/auth.go",
    "cmd/audit_test.go",
    "cmd/cache_test.go",
    "cmd/d1_import.go",
    "cmd/terraform_test.go",
    "cmd/d1.go",
    "cmd/worker.go",
    "cmd/domains_tui.go",
    "cmd/d1_timetravel.go",
    "cmd/images_test.go",
    "cmd/sync.go",
    "cmd/demo_test.go",
    "cmd/queue.go",
    "cmd/sync_guardrails_test.go",
    "cmd/alerts.go",
    "cmd/delete.go",
    "cmd/pages_env.go",
    "cmd/mcp_generate.go",
    "cmd/kv.go",
    "cmd/apply_test.go",
    "cmd/switch.go",
    "cmd/stream.go",
    "cmd/wrangler_test.go",
    "cmd/init_test.go",
    "cmd/metrics_test.go",
    "cmd/d1_export.go",
    "cmd/export.go",
    "cmd/queue_send_test.go",
    "cmd/hyperdrive_test.go",
    "cmd/ratelimit.go",
    "cmd/create_test.go",
    "cmd/pagerules.go",
    "cmd/completion_test.go",
    "cmd/serve.go",
    "cmd/validate_test.go",
    "cmd/email_test.go",
    "cmd/domains_test.go",
    "cmd/root_test.go",
    "cmd/sync_test.go",
    "cmd/copy.go",
    "cmd/queue_send.go",
    "cmd/status_test.go",
    "cmd/cache.go",
    "cmd/serve_test.go",
    "cmd/queue_test.go",
    "cmd/domains.go",
    "cmd/waf_test.go",
    "cmd/dev_test.go",
    "cmd/analytics_test.go",
    "cmd/templates_test.go",
    "cmd/ai.go",
    "cmd/domains_get.go",
    "cmd/diff_test.go",
    "cmd/bucket_domain.go",
    "cmd/hyperdrive.go",
    "cmd/alerts_test.go",
    "cmd/domains_ns.go",
    "cmd/pagerules_test.go",
    "cmd/plugin_test.go",
    "cmd/dns_test.go",
    "cmd/diff.go",
    "cmd/vectorize.go",
    "cmd/pages_domain_test.go",
    "cmd/plugin.go",
    "cmd/wrangler.go",
    "cmd/auth_permissions.go",
    "cmd/pages_deployment.go",
    "cmd/cost_test.go",
    "cmd/images.go",
    "cmd/zone.go",
    "cmd/pages_test.go",
    "cmd/email.go",
    "cmd/domains_tui_test.go",
    "cmd/mcp.go",
    "cmd/domains_stats_test.go",
    "cmd/export_test.go",
    "cmd/watch_test.go",
    "cmd/setup.go",
    "cmd/redirects.go",
    "cmd/auth_permissions_test.go",
    "cmd/pages.go"
  ],
  "entry_points": [
    "main.go",
    "cmd/root.go"
  ],
  "intel": "✓ intel OK",
  "has_ccs": true,
  "has_roadmap": true,
  "has_issues": true,
  "has_docs": true,
  "git_branch": "master",
  "file_count": 4828,
  "project_type": "structured",
  "capabilities": [
    "issues",
    "ideas",
    "roadmap",
    "feedback",
    "changelog"
  ],
  "design": {
    "frontend": {
      "detected": true,
      "component_files": 10,
      "dirs": [
        "desktop/src/",
        "desktop/src/__tests__/",
        "desktop/src/components/",
        "desktop/src/views/"
      ],
      "animation_instances": 0,
      "motion_evidence": false
    },
    "skipped_paths": 0
  }
}
```

Code metrics (ccs audit metrics):
```json
{
  "total_files": 2497,
  "total_lines": 738525,
  "lines_by_language": {
    "go": 165497,
    "javascript": 28,
    "rust": 445,
    "typescript": 1166
  },
  "files_by_type": {
    "code": 237,
    "other": 2021,
    "test": 239
  },
  "todo_count": 2,
  "fixme_count": 0,
  "hack_count": 0,
  "console_log_count": 1,
  "any_type_count": 0,
  "largest_files": [
    {
      "path": "cmd.test",
      "lines": 32297
    },
    {
      "path": "r2go2",
      "lines": 24548
    },
    {
      "path": "docs/conversation-transcripts/2026-05-16_024358_2cffd242.md",
      "lines": 22844
    },
    {
      "path": "r2go2-enhanced",
      "lines": 15843
    },
    {
      "path": "GOrchestra/glm-agents/0011-serve-server-root-keychain-tes/session.log",
      "lines": 11687
    },
    {
      "path": "GOrchestra/glm-agents/0011-serve-server-root-keychain-tes/output.log",
      "lines": 11687
    },
    {
      "path": "r2go2-tui-installer",
      "lines": 11534
    },
    {
      "path": "GOrchestra/glm-agents/0009-rest-server-tests/output.log",
      "lines": 11277
    },
    {
      "path": "GOrchestra/glm-agents/0009-rest-server-tests/session.log",
      "lines": 11277
    },
    {
      "path": "GOrchestra/glm-agents/0010-sse-tests/output.log",
      "lines": 10883
    }
  ]
}
```

Lint results (ccs audit lint):
```json
[
  {
    "linter": "go vet",
    "pass": true,
    "error_count": 0,
    "output": ""
  },
  {
    "linter": "spawn-check (ROAD-525)",
    "pass": false,
    "error_count": 1,
    "output": "Found 1 raw cmd.Start() calls that should use daemon.SpawnAsync() or daemon.Acquire():\n  cmd/installer_tui/main.go:1280: _ = cmd.Start() // Fire and forget"
  }
]
```

Version analysis: {
  "versions": [
    {
      "source": ".version-registry.json",
      "version": "0.26.0",
      "file": ".version-registry.json"
    }
  ],
  "has_conflict": false,
  "canonical": "0.26.0"
}

Readiness: {
  "context": "Readiness:\n  PR: ✓ No conflicts",
  "pr_ready": true,
  "deploy_ready": true,
  "tests_passing": true,
  "ci_passing": false,
  "has_conflicts": false,
  "needs_rebase": false,
  "analysis_ms": 0
}

Security scan summary (full 538-finding detail: docs/audit/2026-09-13-cosmoflare/_ctx/security.json):
```json
{
 "scanned_files": 7197,
 "total": 538,
 "critical": 13,
 "high": 221,
 "medium": 59,
 "low": 236,
 "top_rules": [
  [
   "hardcoded-ip",
   236
  ],
  [
   "go-sql-injection",
   143
  ],
  [
   "debug-true",
   50
  ],
  [
   "disabled-ssl-verify",
   24
  ],
  [
   "twilio-account-sid",
   23
  ],
  [
   "aws-access-key",
   13
  ],
  [
   "path-traversal",
   12
  ],
  [
   "command-injection",
   9
  ],
  [
   "generic-secret",
   9
  ],
  [
   "cors-wildcard",
   8
  ],
  [
   "todo-auth",
   5
  ],
  [
   "todo-security",
   4
  ]
 ],
 "critical_findings": [
  "- [aws-access-key] docs/audit/2026-08-31-cosmoflare/agent-1-code-quality.md:52 \u2014 AWS Access Key ID - never commit these",
  "- [aws-access-key] docs/audit/2026-08-31-cosmoflare/risk-map.md:31 \u2014 AWS Access Key ID - never commit these",
  "- [aws-access-key] docs/conversation-transcripts/2026-05-16_024358_2cffd242.md:12110 \u2014 AWS Access Key ID - never commit these",
  "- [aws-access-key] docs/conversation-transcripts/2026-05-16_024358_2cffd242.md:12163 \u2014 AWS Access Key ID - never commit these",
  "- [aws-access-key] docs/conversation-transcripts/2026-05-16_024358_2cffd242.md:12438 \u2014 AWS Access Key ID - never commit these",
  "- [aws-access-key] docs/conversation-transcripts/2026-08-31_023804_e6394177.md:1339 \u2014 AWS Access Key ID - never commit these",
  "- [aws-access-key] docs/conversation-transcripts/2026-09-06_201854_d6606927.md:1068 \u2014 AWS Access Key ID - never commit these",
  "- [aws-access-key] docs/feedback/outgoing/2026-09-06-claudecodesetup-gorchestra-recovery-patches-never-pruned.md:58 \u2014 AWS Access Key ID - never commit these",
  "- [aws-access-key] internal/interactive/backup_restore_test.go:47 \u2014 AWS Access Key ID - never commit these",
  "- [aws-access-key] internal/interactive/backup_restore_test.go:106 \u2014 AWS Access Key ID - never commit these",
  "- [aws-access-key] internal/interactive/backup_restore_test.go:378 \u2014 AWS Access Key ID - never commit these",
  "- [aws-access-key] tests/fixtures/testdata.go:17 \u2014 AWS Access Key ID - never commit these",
  "- [aws-access-key] tests/fixtures/testdata.go:27 \u2014 AWS Access Key ID - never commit these"
 ]
}
```
