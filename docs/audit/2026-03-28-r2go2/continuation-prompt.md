---
status: PENDING
type: audit-followup
priority: high
---
# Audit Followup: R2Go2 2026-03-28

## Critical Bugs to Fix First
1. BUG-001: Speed calc Inf/NaN — enhanced_client.go:198,277 — change time.Since(time.Now()) to time.Since(startTime)
2. BUG-002: PersistentPreRun blocks setup — root.go:63 — add command exclusion list
3. BUG-004: Keyboard off-by-one — menu_selection.go:170 — change int(num-'1')-1 to int(num-'1')
4. printErrorAndExit doesn't exit — list.go:130 — add os.Exit(1)
5. Missing LICENSE file — repo root — create MIT license

## Quick Wins (small effort, high impact)
1. Fix install.sh binary name mismatch — install.sh:131 — effort: small
2. Fix CI Go version 1.26 → 1.25 — test-suite.yml:9 — effort: small
3. Add HTTP client timeout (30s) — client.go:72 — effort: small
4. Fix YAML output using json.Marshal — bucket.go:427 — effort: small
5. Add ETag bounds check — object.go:318 — effort: small

## Foundation Work (ROAD-000)
1. Extract Client interface from concrete struct (TASK-002)
2. Re-enable S3 implementation from api_disabled/client.go
3. Connect real S3 calls to existing command handlers
4. Update tests from placeholder assertions to real behavior

## Roadmap Items to Start
1. ROAD-000: Wire up real S3 API client (priority 95)
2. Link BUG-007 to ROAD-000 (duplicate tracking)
3. Run roadmap grooming: promote top 5 to planned, consolidate duplicates

## Files to Read First
- docs/audit/2026-03-28-r2go2/brief.md (project context)
- docs/audit/2026-03-28-r2go2/risk-map.md (where NOT to touch without tests)
- docs/audit/2026-03-28-r2go2/surprises.md (non-obvious findings)
