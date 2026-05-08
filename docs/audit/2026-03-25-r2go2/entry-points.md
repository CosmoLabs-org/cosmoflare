# R2Go2 Entry Points

## Primary Entry Point

**`main.go`** -> `cmd.Execute()` -> `rootCmd.Execute()` (Cobra)

All execution flows through Cobra's command dispatch. The `PersistentPreRun` on rootCmd validates environment before any subcommand executes (this is a known bug for non-API commands).

## Command Execution Paths

### Bucket Operations
```
r2go2 list -> cmd/list.go:listCmd.RunE
  -> api.NewClientFromProfile("") -> config.GetCurrent()
  -> client.ListBuckets() [STUB: returns empty slice]
  -> printJSON(buckets) or formatted table

r2go2 create <name> -> cmd/create.go:createCmd.RunE
  -> validateBucketName()
  -> client.CreateBucket(name) [STUB: returns mock bucket]
  -> printSuccess or printSuccessJSON

r2go2 delete <name> -> cmd/delete.go:deleteCmd.RunE
  -> confirmation prompt
  -> client.DeleteBucket(name) [STUB: no-op]
```

### Upload Flow (most complete path)
```
r2go2 object upload <bucket> <file> -> cmd/object.go
  -> api.NewEnhancedClient(baseClient)
  -> ec.UploadFile(ctx, bucket, key, filePath, opts)
     -> os.Stat(filePath) to get size
     -> if size > chunkSize: multipartUpload()
        -> s3.CreateMultipartUpload
        -> loop: uploadPart() with seek + LimitReader
        -> s3.CompleteMultipartUpload
     -> else: singlePartUpload()
        -> s3.PutObject with progress reader
     -> UploadResult{Key, Bucket, Size, ETag, Speed, Parts}
```

### Auth Flow
```
r2go2 auth login -> cmd/auth.go:runAuthLogin
  -> interactive prompt for token + account ID
  -> api.NewClient(opts) -> client.TestConnection()
  -> config.SetProfile(profile) -> config.Save()

r2go2 auth status -> cmd/auth.go
  -> config.GetCurrent() -> display masked credentials
```

### TUI Dashboard
```
r2go2 dashboard -> cmd/dashboard.go
  -> tui.initialModel() -> InitializeStyles(darkTheme)
  -> tea.NewProgram(model).Run()
  -> Init: loadDataCmd() + 5s tick for realTimeUpdate
  -> Update: key events (j/k/h/l/tab/enter/q)
  -> View: renderHeader + renderSection(current) + renderStatusBar
```

### Setup / First Run
```
r2go2 setup -> cmd/setup.go
  -> interactive.RunFirstTimeSetup()
  -> interactive.RunSetupWizard()
  -> config.SetProfile() + config.Save()
```

## Background Processes

- **Real-time stats polling**: TUI dashboard ticks every 5 seconds via `tea.Tick`
- **Upload progress**: Goroutine in `UploadWithRealTimeProgress` for `rp.ShowSimpleProgress()`
- **No daemon/service**: CLI only, no background processes
