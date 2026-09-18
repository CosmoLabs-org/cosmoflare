# Unit Test Generation — cmd/part9 of 9 (risk tier: medium)

You are a Go test engineer. Your task is to write unit tests for the package `cmd`.

## Functions to Test

- `runD1QueryLocal` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/d1_local.go`
- `splitSQLStatements` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/d1_local.go`
- `runDomains` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/domains.go`
- `runDomainsDetail` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/domains.go`
- `runWorkerBindings` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/worker_bindings.go`
- `printBindingsTable` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/worker_bindings.go`
- `runTemplatesInfo` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/templates.go`
- `runTemplatesCreate` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/templates.go`
- `runDomainsGet` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/domains_get.go`
- `printDomainDetail` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/domains_get.go`
- `d1ImportResumeOffset` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/d1_import.go`
- `runD1Import` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/d1_import.go`
- `runQueueSend` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/queue_send.go`
- `runQueueSendBatch` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/queue_send.go`
- `runInit` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/init.go`
- `detectTemplate` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/init.go`
- `executeImport` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/import_cmd.go`
- `presentImportResult` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/import_cmd.go`
- `runWorkerTail` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/worker_tail.go`
- `streamTailEntries` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/worker_tail.go`
- `runDomainsTUI` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/domains_tui.go`
- `printBucketsTable` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/list.go`
- `completeBucketNames` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/completion.go`
- `runD1Export` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/d1_export.go`
- `createTheme` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/theme.go`
- `runWorkerTypes` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/worker_types.go`
- `runDomainsNS` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/domains_ns.go`
- `runDev` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/dev.go`
- `runMigrateFromS3` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/migrate.go`
- `runSwitch` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/switch.go`
- `runD1TimeTravelRestore` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/d1_timetravel.go`
- `runBackup` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/backup.go`
- `runDomainsRedirects` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/domains_redirects.go`
- `outSuccess` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/output.go`
- `runExport` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/export.go`
- `runAccountVerify` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/account.go`
- `runLimits` in `/Users/gabstudio/PROJECTS/cosmoflare/cmd/limits.go`

## Test Style Requirements

- Use t.Run subtests for all functions with multiple cases.
- Each test case must have a descriptive name.
- Add GoDoc comments above each test function explaining what it validates.
- Test files follow the `*_test.go` convention.
- Prohibit: testing unexported internals via reflection, mocking without understanding dependencies.

## Risk-Tier Instructions (medium)

This is a MEDIUM-risk package. Focus on algorithm correctness and edge cases.

- Algorithm correctness: verify outputs match expected results for representative inputs.
- Empty/nil/zero-value inputs must not panic or raise unhandled exceptions.
- Test at least one error path per exported function.

## Verification

After writing tests, verify they compile and pass:

```
go test ./... -v -count=1
```

Ensure `go test` exits 0 before submitting.
