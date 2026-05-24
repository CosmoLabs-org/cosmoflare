# Architecture Analysis

**Source**: Synthesized from agent-1-code-quality.md, agent-2-core-logic.md, agent-3-api-design.md

## System Layers

CosmoDev-R2Go2 follows a **3-layer architecture**: public library, CLI shell, and internal packages. The design principle is library-first -- the CLI is a thin shell over the library, and future GUI/mobile apps will consume the same library.

```
                    +---------------------------+
                    |        main.go            |
                    +---------------------------+
                               |
                    +---------------------------+
                    |   cmd/ (Cobra CLI)        |
                    |   33 command files         |
                    |   root.go orchestrates     |
                    +---------------------------+
                         |              |
              +----------+----------+   +------------------+
              |  pkg/r2go2/         |   | internal/        |
              |  Public Library     |   | config/ tui/     |
              |  30 source files    |   | cli/ utils/      |
              |  12 service structs |   | interactive/     |
              +---------------------+   +------------------+
                         |
              +---------------------+
              |  Cloudflare API     |
              |  (REST + S3)        |
              +---------------------+
```

## Layer 1: Public Library (`pkg/r2go2/`)

The stable API surface. Importable by any Go project as:
```go
import r2go2 "github.com/CosmoLabs-org/CosmoDev-R2Go2/pkg/r2go2"
```

### Core Client

`client.go` defines `R2Client` with functional options pattern (see agent-1-code-quality.md, Strength #1):

- `NewClient(opts ...ClientOption) (*R2Client, error)`
- Options: `WithAccountID`, `WithAPIToken`, `WithEndpoint`, `WithHTTPClient`
- Handles S3 client initialization for R2 storage operations
- Validates required fields (account ID, API token) at construction time

### Service Structs (12 total)

Each service follows an identical pattern (see agent-1-code-quality.md, Strength #4):

| Service | File | Scope | API |
|---------|------|-------|-----|
| R2 Storage | `storage.go`, `upload.go`, `download.go` | Account | S3-compatible |
| Workers | `worker.go` | Account | Cloudflare REST |
| KV | `kv.go` | Account | Cloudflare REST |
| DNS | `dns.go` | Zone | Cloudflare REST |
| Zones | `zone.go` | Account | Cloudflare REST |
| SSL/TLS | `ssl.go` | Zone | Cloudflare REST |
| Cache | `cloudflare_cache.go` | Zone | Cloudflare REST |
| Healthchecks | `healthcheck.go` | Zone | Cloudflare REST |
| Doctor | `doctor.go` | Zone | stdlib probes |
| Domains | `domains.go` | Account | Cloudflare REST |
| CORS | `cors.go` | Bucket | S3-compatible |
| Page Rules | `pagerules.go` | Zone | Cloudflare REST |

**Structural pattern** for each service:
```
type XxxService struct { ... }
func NewXxxService(accountID, apiToken string) *XxxService
func NewXxxServiceFromCreds(creds Credentials) *XxxService
func (s *XxxService) Create(...) error
func (s *XxxService) List(...) ([]Xxx, error)
func (s *XxxService) Get(...) (*Xxx, error)
func (s *XxxService) Update(...) error
func (s *XxxService) Delete(...) error
```

**Key architectural weakness** (see agent-1-code-quality.md, Weakness #2): Services are siloed. There is no `client.DNS()` or `client.Workers()` accessor. Each service must be constructed independently, which prevents shared configuration, middleware, or connection pooling.

### Error Types

`errors.go` defines the typed error hierarchy (see agent-1-code-quality.md, Strength #2):
- `R2NotFoundError` -- resource does not exist
- `R2ValidationError` -- invalid input
- `R2AuthError` -- authentication/authorization failure
- `R2RateLimitError` -- rate limited
- `R2Error` -- generic Cloudflare API error

**Weakness** (see agent-2-core-logic.md): Some errors are misclassified -- network errors wrapped as validation errors, transient failures reported as permanent.

## Layer 2: CLI Shell (`cmd/`)

33 Cobra command files. `root.go` defines the root command, persistent flags (`--json`, `--profile`, `--account-id`), and the `printError`/`printJSON` helpers.

### Command Tree

```
r2go2 (root)
  |-- bucket (create, list, get, delete, update)
  |-- object (put, get, delete, copy, head, presign, batch)
  |-- worker (deploy, list, get, delete, logs, settings)
  |-- kv (namespace create/list/delete, put, get, delete, list)
  |-- dns (create, list, get, update, delete)
  |-- zone (create, list, get, settings, delete)
  |-- ssl (status, settings, update, verify)
  |-- cache (purge, settings)
  |-- domains (--detail, --enrich, --json)
  |-- doctor (--fix, --all, --json)
  |-- config (init, set, list, show, switch, export)
  |-- dashboard (launches TUI)
  |-- analytics
  |-- compare
  |-- completion
```

### Data Flow: CLI to API

```
User input → Cobra flag parsing → cmd/*.go handler
  → internal/config (load profile, credentials)
  → pkg/r2go2 NewXxxService(accountID, apiToken)
  → Service method (validates input)
  → HTTP request to Cloudflare API or S3
  → Response parsing → typed error or result
  → cmd/*.go formats output (table or --json)
```

**Weakness** (see agent-3-api-design.md): Global mutable CLI state -- flags are stored in package-level variables, making commands non-reentrant.

## Layer 3: Internal Packages (`internal/`)

### `internal/config/`

Viper-based profile management. Supports multiple named profiles with credentials, account IDs, and per-profile settings.

- Config file: `.cosmoflare.yaml` (legacy: `.r2go2.yaml`)
- Profile struct requires `json`, `yaml`, and `mapstructure` tags
- **Critical gap**: Zero test coverage for 332 lines of config code (see agent-1-code-quality.md)
- **Security concern**: Config file created with default permissions before chmod (see agent-2-core-logic.md)

### `internal/tui/`

Bubble Tea dashboard. Model/Update/View architecture.

- `model.go` -- tea.Model implementation, state management
- `update.go` -- input handling, key bindings
- `view.go` -- rendering, layout
- `components/` -- reusable UI components (navigation, panels)
- **Weakness**: Uses simulated/hardcoded data, not live API (see agent-1-code-quality.md, Weakness #5)

### `internal/interactive/`

Setup wizard for first-time configuration.

### `internal/utils/`

Shared utility functions.

## Module Boundaries

| Boundary | Direction | What Crosses |
|----------|-----------|--------------|
| `cmd/` -> `pkg/r2go2/` | Import | Service constructors, method calls, types |
| `cmd/` -> `internal/config/` | Import | Profile loading, credential retrieval |
| `cmd/` -> `internal/tui/` | Import | Dashboard launch (via `cmd/dashboard.go`) |
| `pkg/r2go2/` -> `internal/` | NONE | Library is self-contained, no internal deps |
| `internal/tui/` -> `pkg/r2go2/` | Potential | Should import for live data, currently does not |

The clean separation of `pkg/r2go2/` from `internal/` is a strong architectural property -- it ensures the library can be imported by external Go projects without pulling in CLI or TUI dependencies.

## Dual API Protocols

R2Go2 communicates with Cloudflare through two distinct protocols:

1. **S3-compatible API** (via AWS SDK v2): R2 storage operations (buckets, objects, uploads, downloads, presigned URLs)
2. **Cloudflare REST API** (via `net/http`): All other services (Workers, KV, DNS, Zones, SSL, Cache, etc.)

The `R2Client` struct manages the S3 client. Individual services manage their own HTTP clients for the REST API. This dual-protocol design is a natural consequence of Cloudflare R2 being S3-compatible while other services use Cloudflare's proprietary API.
