# BR-03 — LimitsService v2: embedded catalog, hardcoded maps deleted

Spec (authoritative): docs/brainstorming/2026-09-10-cf-limits-awareness-layer.md — sections D1 (catalog SSOT), D2 (per-service plan resolution), D3 (freshness). Corpus findings section (bottom of that doc) names two resources the catalog does NOT carry — read it.

Data source: docs/research/2026-09-10-cf-limits-corpus/catalog-draft.json (66 entries, schema: {id, name, kind, unit, scope, tiers:{free,paid,enterprise}, enforceability, soft, trackable, source_url, verified_on, notes}; values may be numbers OR null).

Read first: pkg/cosmoflare/limits.go (480 lines — limitFor:39, workerPlanLimits:17, staticLimits:25, dnsRecordsStaticLimit:52), the brainstorm sections above, cmd/limits.go.

## Exact changes

### 1. pkg/cosmoflare/limitsdata/ (new package)

- `catalog.json` — verbatim copy of docs/research/2026-09-10-cf-limits-corpus/catalog-draft.json
- `limitsdata.go`:
  - `//go:embed catalog.json` + `type Entry struct` mirroring the schema (Tiers as `*Tiers` with `*uint64` fields or json.Number — tiers values are numbers or null; soft is bool; trackable bool)
  - `func Load() ([]Entry, error)` (parse once, sync.OnceValues)
  - `func Lookup(id string) (Entry, bool)`
  - `func Stale(before time.Time) []string` — ids whose verified_on < the cutoff (feeds the 90-day warning)
  - Two LOCAL-ONLY constants as explicit entries appended at load time (NOT in catalog.json — they are unverified): `r2.buckets_per_account` and `workers.scripts_per_account`, each constructed as an Entry with `verified_on: nil`, `notes: "local constant — corpus docs-silent; flagged unverified"`, and the current limits.go values for their tiers. `Lookup` must return them like any other entry.

### 2. pkg/cosmoflare/limits.go rewiring

- DELETE `workerPlanLimits` and `staticLimits` maps entirely.
- `limitFor(resource, plan string)` becomes a catalog lookup: `Lookup(resource)`, pick the tier from plan (free|paid|enterprise → tiers field; unknown plan → paid); `ok=false` when the entry or the tier value is null/absent.
- `dnsRecordsStaticLimit` keeps its zone-object path (D2: zones carry their plan on the zone object) but reads its numbers from the catalog entry `dns.records_zone` when values exist; fall back to its current constants ONLY behind a named `unverifiedZoneFallback` constant with a comment — do not silently hardcode.
- Plan resolution (D2) — a `planFor(service string) (string, error)` chain: subscriptions API → per-service `--plan` override (already threaded via options if present; otherwise add `WithLimitsPlanOverrides(map[string]string)`) → `.cosmoflare.yaml` `plans:` map if the config is attached → `"unknown"`. Workers/R2/D1 from one subscriptions call; reuse whatever limits.go already fetches today.
- Freshness (D3): the `cosmoflare limits` output path gains a stale check — entries older than 90 days produce ONE warning line naming the ids (human output) and each row carries `verified_on` (JSON output).

### 3. cmd/limits.go

- New subcommand `cosmoflare limits catalog` — prints the catalog (`--json` = full entries; text = id/free/paid/verified table). Rich help + example.

### 4. Tests (write FIRST, red)

- limitsdata: Load parses 66+2 entries; Lookup hit/miss; Stale returns exactly the ids older than cutoff; the two local-only entries carry nil verified_on.
- limits: limitFor returns catalog values for `workers.requests_daily` (free=100000) and `workers.subrequests` (paid=10000); returns ok=false for null tiers; plan resolution precedence test (override beats config beats API stub); stale warning fires at >90d and not at fresh.
- Existing limits tests keep passing EXCEPT assertions on the deleted maps — update those to expect catalog values (do not delete tests).

## Verification (in order)

```bash
go test ./pkg/cosmoflare/limitsdata/ ./pkg/cosmoflare/ -run 'TestLimits|TestCatalog|TestStale|TestPlan' -v   # red first, then green
go test ./pkg/cosmoflare/ ./cmd/    # packages green
go vet ./pkg/cosmoflare/... && gofmt -l pkg/cosmoflare/ | grep -v _test.go; echo CLEAN
```

## Commit

`feat(limits): LimitsService v2 — embedded catalog SSOT, hardcoded maps deleted (BR-03)` with body: root cause (hardcoded maps drifted from docs; catalog with provenance landed via research ingestion), what changed (embed + lookup + plan chain + stale warning + limits catalog export + 2 explicitly-unverified local entries), spec + data refs.

## Out of scope

D4 guards (pre-flight enforcement), pricing fields, the pager, any cmdmanifest changes (limits ids unchanged).
