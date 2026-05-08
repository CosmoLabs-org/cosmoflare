# Risk Assessment

## Tech Debt Hotspots

| File | Lines | Issue | Severity |
|------|-------|-------|----------|
| internal/api/client.go | 225 | 9 placeholder stubs — the #1 blocker | CRITICAL |
| cmd/root.go | 216 | PersistentPreRun blocks everything + global state | CRITICAL |
| cmd/object.go | 790+ | Largest cmd file, fake upload, fake search | HIGH |
| cmd/bucket.go | 680+ | YAML uses JSON, update is no-op | HIGH |
| internal/api/enhanced_client.go | 431 | Speed calc bug, off-by-one in multipart | HIGH |
| cmd/list.go | 143 | printErrorAndExit doesn't exit | HIGH |
| internal/tui/update.go | 243 | Simulated stats, broken search mode | MEDIUM |

## Fragile Areas (high coupling, low tests)
- **cmd/ package**: 30+ global vars, no unit tests for command handlers, depends on concrete Client (see agent-1-code-quality.md, agent-7-api-design.md)
- **Legacy commands** (create, list, delete): Use different client pattern than newer commands, `APIToken` global never populated (see agent-1-code-quality.md)
- **Batch worker stats**: Non-atomic updates across goroutines — data race under load (see agent-2-core-logic.md)

## Security Surface
- **Plaintext credentials**: ~/.r2go2/config.yaml stores API tokens as raw strings (see agent-8-infrastructure.md)
- **No HTTP timeout**: Client can hang indefinitely (see agent-2-core-logic.md, agent-8-infrastructure.md)
- **Token in CLI flags**: --token visible in shell history and /proc (see agent-8-infrastructure.md)
- **Config dir 0755**: Should be 0700 (see agent-8-infrastructure.md)
- **ExportProfile**: Writes raw secrets to stdout (see agent-8-infrastructure.md)

## Chained Risks
- **Placeholder API + Aspirational docs**: Users discover features don't work after investing setup time → trust destruction (see surprises.md)
- **PersistentPreRun + No API**: Can't setup → can't use → can't evaluate the product at all
- **No interface + No tests + Placeholder**: Can't safely replace stubs without interface first → chicken-and-egg problem (see surprises.md)
- **Tracked binaries + No .gitignore enforcement**: Repo bloats over time, clone times increase

## Single Points of Failure
- **ROAD-000 (real API)**: ~60% of roadmap and all core commands are blocked by this one item
- **PersistentPreRun**: A single function controls access to all 16 commands
- **getAPIClient()**: Single function creates all API clients with no DI escape hatch
