package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// migrationFileRe matches migration file names in the NNNN_<name>.sql
// convention used by MigrationsCreate/MigrationsList/MigrationsApply.
var migrationFileRe = regexp.MustCompile(`^(\d{4})_.+\.sql$`)

// D1Database represents a Cloudflare D1 database.
type D1Database struct {
	UUID      string     `json:"uuid"`
	Name      string     `json:"name"`
	Version   string     `json:"version"`
	NumTables int        `json:"num_tables"`
	FileSize  int64      `json:"file_size"`
	CreatedAt *time.Time `json:"created_at"`
}

// D1QueryResult represents the result of a D1 SQL query.
type D1QueryResult struct {
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	Meta    D1QueryMeta      `json:"meta"`
	Success bool             `json:"success"`
}

// D1QueryMeta contains metadata about a query execution.
type D1QueryMeta struct {
	ChangedDB   bool    `json:"changed_db"`
	Changes     int     `json:"changes"`
	Duration    float64 `json:"duration"`
	LastRowID   int     `json:"last_row_id"`
	RowsRead    int     `json:"rows_read"`
	RowsWritten int     `json:"rows_written"`
	SizeAfter   int     `json:"size_after"`
}

// D1Service implements D1 database operations.
type D1Service struct {
	cf        *cloudflare.API
	accountID string

	// sleepFn is a test-only seam for the Import backoff retry loop
	// (error 7500). Defaults to time.Sleep; set directly by tests in this
	// package, mirroring restClient.sleepFn in rest_client.go.
	sleepFn func(time.Duration)
}

// NewD1Service creates a new D1 service client.
func NewD1Service(api *cloudflare.API, accountID string) (*D1Service, error) {
	if api == nil {
		return nil, validationError("NewD1Service", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewD1Service", "account ID is required")
	}
	return &D1Service{cf: api, accountID: accountID}, nil
}

// NewD1ServiceFromCreds creates a D1Service from account ID and API token.
func NewD1ServiceFromCreds(accountID, apiToken string) (*D1Service, error) {
	if accountID == "" {
		return nil, validationError("NewD1Service", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewD1Service", "API token is required")
	}
	cf, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, authError("NewD1Service", "failed to create Cloudflare API client", err)
	}
	return &D1Service{cf: cf, accountID: accountID}, nil
}

// Create creates a new D1 database.
func (s *D1Service) Create(ctx context.Context, name string) (*D1Database, error) {
	if name == "" {
		return nil, validationError("D1Service.Create", "database name is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.CreateD1Database(ctx, rc, cloudflare.CreateD1DatabaseParams{Name: name})
	if err != nil {
		return nil, newError("D1Service.Create", fmt.Sprintf("failed to create database %q", name), err)
	}

	return mapD1Database(result), nil
}

// List returns all D1 databases in the account.
func (s *D1Service) List(ctx context.Context) ([]*D1Database, error) {
	rc := cloudflare.AccountIdentifier(s.accountID)
	results, _, err := s.cf.ListD1Databases(ctx, rc, cloudflare.ListD1DatabasesParams{})
	if err != nil {
		return nil, newError("D1Service.List", "failed to list databases", err)
	}

	databases := make([]*D1Database, 0, len(results))
	for _, db := range results {
		databases = append(databases, mapD1Database(db))
	}
	return databases, nil
}

// Get retrieves a single D1 database by ID.
func (s *D1Service) Get(ctx context.Context, databaseID string) (*D1Database, error) {
	if databaseID == "" {
		return nil, validationError("D1Service.Get", "database ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	result, err := s.cf.GetD1Database(ctx, rc, databaseID)
	if err != nil {
		return nil, newError("D1Service.Get", fmt.Sprintf("failed to get database %q", databaseID), err)
	}

	return mapD1Database(result), nil
}

// Delete deletes a D1 database.
func (s *D1Service) Delete(ctx context.Context, databaseID string) error {
	if databaseID == "" {
		return validationError("D1Service.Delete", "database ID is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	err := s.cf.DeleteD1Database(ctx, rc, databaseID)
	if err != nil {
		return newError("D1Service.Delete", fmt.Sprintf("failed to delete database %q", databaseID), err)
	}
	return nil
}

// Query executes a SQL query against a D1 database.
func (s *D1Service) Query(ctx context.Context, databaseID, sql string, params ...string) ([]*D1QueryResult, error) {
	if databaseID == "" {
		return nil, validationError("D1Service.Query", "database ID is required")
	}
	if sql == "" {
		return nil, validationError("D1Service.Query", "SQL query is required")
	}

	rc := cloudflare.AccountIdentifier(s.accountID)
	results, err := s.cf.QueryD1Database(ctx, rc, cloudflare.QueryD1DatabaseParams{
		DatabaseID: databaseID,
		SQL:        sql,
		Parameters: params,
	})
	if err != nil {
		return nil, newError("D1Service.Query", fmt.Sprintf("failed to query database %q", databaseID), err)
	}

	queryResults := make([]*D1QueryResult, 0, len(results))
	for _, r := range results {
		queryResults = append(queryResults, mapD1Result(r))
	}
	return queryResults, nil
}

// D1Migration represents a single migration file and its applied state.
type D1Migration struct {
	Name      string     `json:"name"`
	AppliedAt *time.Time `json:"applied_at"`
	FilePath  string     `json:"file_path"`
}

// D1MigrationResult represents the outcome of attempting to apply one migration.
type D1MigrationResult struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "applied", "skipped", "would_apply", "failed"
	Error  string `json:"error,omitempty"`
}

// MigrationsCreate creates a new migration file in migrationsDir named
// NNNN_<name>.sql, where NNNN is the next sequence number. It does not
// touch the remote database.
func (s *D1Service) MigrationsCreate(name string, migrationsDir string) (string, error) {
	if name == "" {
		return "", validationError("D1Service.MigrationsCreate", "migration name is required")
	}
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}

	if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
		return "", newError("D1Service.MigrationsCreate", fmt.Sprintf("failed to create migrations directory %q", migrationsDir), err)
	}

	next, err := nextMigrationSequence(migrationsDir)
	if err != nil {
		return "", newError("D1Service.MigrationsCreate", "failed to scan existing migrations", err)
	}

	fileName := fmt.Sprintf("%04d_%s.sql", next, slugifyMigrationName(name))
	filePath := filepath.Join(migrationsDir, fileName)

	content := fmt.Sprintf("-- Migration: %s\n-- Created: %s\n\n", name, time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return "", newError("D1Service.MigrationsCreate", fmt.Sprintf("failed to write migration file %q", filePath), err)
	}

	return filePath, nil
}

// MigrationsList scans migrationsDir for migration files and cross-references
// them against the applied state recorded in the database's d1_migrations
// table. Pending migrations have a nil AppliedAt.
func (s *D1Service) MigrationsList(ctx context.Context, databaseID string, migrationsDir string) ([]D1Migration, error) {
	if databaseID == "" {
		return nil, validationError("D1Service.MigrationsList", "database ID is required")
	}
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}

	files, err := listMigrationFiles(migrationsDir)
	if err != nil {
		return nil, newError("D1Service.MigrationsList", fmt.Sprintf("failed to read migrations directory %q", migrationsDir), err)
	}
	if len(files) == 0 {
		return []D1Migration{}, nil
	}

	if err := s.ensureMigrationsTable(ctx, databaseID); err != nil {
		return nil, err
	}

	applied, err := s.appliedMigrations(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	migrations := make([]D1Migration, 0, len(files))
	for _, name := range files {
		var appliedAt *time.Time
		if t, ok := applied[name]; ok {
			t := t
			appliedAt = &t
		}
		migrations = append(migrations, D1Migration{
			Name:      name,
			AppliedAt: appliedAt,
			FilePath:  filepath.Join(migrationsDir, name),
		})
	}
	return migrations, nil
}

// MigrationsApply applies pending migrations in order, recording each
// success in the d1_migrations table. Already-applied migrations are
// skipped. When dryRun is true, no SQL is executed.
func (s *D1Service) MigrationsApply(ctx context.Context, databaseID string, migrationsDir string, dryRun bool) ([]D1MigrationResult, error) {
	migrations, err := s.MigrationsList(ctx, databaseID, migrationsDir)
	if err != nil {
		return nil, err
	}

	results := make([]D1MigrationResult, 0, len(migrations))
	for _, m := range migrations {
		if m.AppliedAt != nil {
			results = append(results, D1MigrationResult{Name: m.Name, Status: "skipped"})
			continue
		}

		if dryRun {
			results = append(results, D1MigrationResult{Name: m.Name, Status: "would_apply"})
			continue
		}

		sqlBytes, err := os.ReadFile(m.FilePath)
		if err != nil {
			results = append(results, D1MigrationResult{Name: m.Name, Status: "failed", Error: err.Error()})
			continue
		}

		if _, err := s.Query(ctx, databaseID, string(sqlBytes)); err != nil {
			results = append(results, D1MigrationResult{Name: m.Name, Status: "failed", Error: err.Error()})
			continue
		}

		appliedAt := time.Now().UTC().Format(time.RFC3339)
		if _, err := s.Query(ctx, databaseID, "INSERT INTO d1_migrations (name, applied_at) VALUES (?1, ?2)", m.Name, appliedAt); err != nil {
			results = append(results, D1MigrationResult{Name: m.Name, Status: "failed", Error: err.Error()})
			continue
		}

		results = append(results, D1MigrationResult{Name: m.Name, Status: "applied"})
	}

	return results, nil
}

// ContainsDestructiveSQL reports whether sql contains a statement that
// could cause irreversible data loss: DROP TABLE, DROP INDEX,
// DROP DATABASE, TRUNCATE, or DELETE without a WHERE clause. Matching is
// case-insensitive.
func ContainsDestructiveSQL(sql string) bool {
	upper := strings.ToUpper(sql)

	for _, pattern := range []string{"DROP TABLE", "DROP INDEX", "DROP DATABASE", "TRUNCATE"} {
		if strings.Contains(upper, pattern) {
			return true
		}
	}

	for _, stmt := range strings.Split(upper, ";") {
		stmt = strings.TrimSpace(stmt)
		if strings.HasPrefix(stmt, "DELETE") && !strings.Contains(stmt, "WHERE") {
			return true
		}
	}

	return false
}

// D1TimeTravelResult represents the outcome of a time-travel restore.
type D1TimeTravelResult struct {
	DatabaseID string    `json:"database_id"`
	Timestamp  time.Time `json:"timestamp"`
	Success    bool      `json:"success"`
}

// D1TimeTravelQuota represents the current time-travel restore quota usage
// for a database within the rolling window.
type D1TimeTravelQuota struct {
	Used           int       `json:"used"`
	Limit          int       `json:"limit"`
	WindowResetsAt time.Time `json:"window_resets_at"`
}

// timeTravelCacheFile is the path (relative to the current working
// directory) of the local time-travel restore quota cache. It is never
// written under $HOME.
const timeTravelCacheFile = ".cosmoflare-time-travel-cache.json"

// timeTravelWindow is the rolling window Cloudflare enforces for D1
// time-travel restores.
const timeTravelWindow = 10 * time.Minute

// timeTravelLimit is the maximum number of restores allowed per database
// within timeTravelWindow.
const timeTravelLimit = 10

// timeTravelCacheEntry records a single restore for quota tracking.
type timeTravelCacheEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	RestoredAt time.Time `json:"restored_at"`
}

// timeTravelCache maps a database ID to its recorded restores.
type timeTravelCache map[string][]timeTravelCacheEntry

// loadTimeTravelCache reads the local quota cache file. A missing file is
// treated as an empty cache, not an error.
func loadTimeTravelCache() (timeTravelCache, error) {
	data, err := os.ReadFile(timeTravelCacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return timeTravelCache{}, nil
		}
		return nil, err
	}
	cache := timeTravelCache{}
	if len(data) == 0 {
		return cache, nil
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	if cache == nil {
		cache = timeTravelCache{}
	}
	return cache, nil
}

// saveTimeTravelCache writes the quota cache file to the current working
// directory.
func saveTimeTravelCache(cache timeTravelCache) error {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(timeTravelCacheFile, data, 0o644)
}

// timeTravelWindowEntries returns the entries from entries that fall
// within timeTravelWindow of now.
func timeTravelWindowEntries(entries []timeTravelCacheEntry, now time.Time) []timeTravelCacheEntry {
	var kept []timeTravelCacheEntry
	for _, e := range entries {
		if now.Sub(e.RestoredAt) < timeTravelWindow {
			kept = append(kept, e)
		}
	}
	return kept
}

// TimeTravelQuotaCheck reports the current time-travel restore quota usage
// for databaseID within the rolling 10-minute window.
func (s *D1Service) TimeTravelQuotaCheck(databaseID string) (*D1TimeTravelQuota, error) {
	if databaseID == "" {
		return nil, validationError("D1Service.TimeTravelQuotaCheck", "database ID is required")
	}

	cache, err := loadTimeTravelCache()
	if err != nil {
		return nil, newError("D1Service.TimeTravelQuotaCheck", "failed to read time-travel quota cache", err)
	}

	now := time.Now().UTC()
	entries := timeTravelWindowEntries(cache[databaseID], now)

	quota := &D1TimeTravelQuota{
		Used:           len(entries),
		Limit:          timeTravelLimit,
		WindowResetsAt: now,
	}
	if len(entries) > 0 {
		oldest := entries[0].RestoredAt
		for _, e := range entries[1:] {
			if e.RestoredAt.Before(oldest) {
				oldest = e.RestoredAt
			}
		}
		quota.WindowResetsAt = oldest.Add(timeTravelWindow)
	}
	return quota, nil
}

// recordTimeTravelRestore appends a restore entry to the local quota
// cache for databaseID.
func (s *D1Service) recordTimeTravelRestore(databaseID string, timestamp, restoredAt time.Time) error {
	cache, err := loadTimeTravelCache()
	if err != nil {
		return newError("D1Service.TimeTravelRestore", "failed to read time-travel quota cache", err)
	}

	entries := timeTravelWindowEntries(cache[databaseID], restoredAt)
	entries = append(entries, timeTravelCacheEntry{Timestamp: timestamp, RestoredAt: restoredAt})
	cache[databaseID] = entries

	if err := saveTimeTravelCache(cache); err != nil {
		return newError("D1Service.TimeTravelRestore", "failed to write time-travel quota cache", err)
	}
	return nil
}

// TimeTravelRestore restores databaseID to the state it had at timestamp.
// It performs a quota pre-flight check and refuses to call the API when
// the rolling 10-restore/10-minute window is exhausted.
//
// The restore is a two-step Cloudflare flow: resolve timestamp to a
// bookmark via the time_travel/bookmark endpoint, then restore the
// database to that bookmark.
func (s *D1Service) TimeTravelRestore(ctx context.Context, databaseID string, timestamp time.Time) (*D1TimeTravelResult, error) {
	if databaseID == "" {
		return nil, validationError("D1Service.TimeTravelRestore", "database ID is required")
	}
	if timestamp.IsZero() {
		return nil, validationError("D1Service.TimeTravelRestore", "timestamp is required")
	}

	quota, err := s.TimeTravelQuotaCheck(databaseID)
	if err != nil {
		return nil, err
	}
	if quota.Used >= quota.Limit {
		return nil, quotaError("D1Service.TimeTravelRestore", fmt.Sprintf("time-travel restore quota exceeded (%d/%d), resets at %s", quota.Used, quota.Limit, quota.WindowResetsAt.Format(time.RFC3339)), nil)
	}

	bookmarkURI := fmt.Sprintf("/accounts/%s/d1/database/%s/time_travel/bookmark?timestamp=%s", s.accountID, databaseID, timestamp.UTC().Format(time.RFC3339))
	bookmarkResp, err := s.cf.Raw(ctx, http.MethodGet, bookmarkURI, nil, nil)
	if err != nil {
		return nil, newError("D1Service.TimeTravelRestore", "failed to resolve time-travel bookmark", err)
	}

	var bookmarkResult struct {
		Bookmark string `json:"bookmark"`
	}
	if err := json.Unmarshal(bookmarkResp.Result, &bookmarkResult); err != nil {
		return nil, newError("D1Service.TimeTravelRestore", "failed to parse time-travel bookmark response", err)
	}
	if bookmarkResult.Bookmark == "" {
		return nil, newError("D1Service.TimeTravelRestore", "time-travel bookmark response did not include a bookmark", nil)
	}

	restoreURI := fmt.Sprintf("/accounts/%s/d1/database/%s/restore", s.accountID, databaseID)
	if _, err := s.cf.Raw(ctx, http.MethodPost, restoreURI, map[string]string{"bookmark": bookmarkResult.Bookmark}, nil); err != nil {
		return nil, newError("D1Service.TimeTravelRestore", fmt.Sprintf("failed to restore database %q", databaseID), err)
	}

	restoredAt := time.Now().UTC()
	if err := s.recordTimeTravelRestore(databaseID, timestamp.UTC(), restoredAt); err != nil {
		return nil, err
	}

	return &D1TimeTravelResult{
		DatabaseID: databaseID,
		Timestamp:  timestamp.UTC(),
		Success:    true,
	}, nil
}

// d1ExportPollInterval controls the delay between export status polls.
// It is a var so tests can shrink it.
var d1ExportPollInterval = 500 * time.Millisecond

// d1ExportMaxPolls bounds the number of export status polls before giving
// up, so a stuck job cannot hang the caller forever.
const d1ExportMaxPolls = 60

// Export streams a SQL dump of databaseID. Cloudflare's export endpoint is
// bookmark-based: an initial call kicks off the export job, and subsequent
// polls (each echoing back the last bookmark) return either the next
// bookmark or, once ready, a signed URL to download the dump from. Export
// streams that download's body back to the caller, who is responsible for
// closing it.
func (s *D1Service) Export(ctx context.Context, databaseID string) (io.ReadCloser, error) {
	if databaseID == "" {
		return nil, validationError("D1Service.Export", "database ID is required")
	}

	exportURI := fmt.Sprintf("/accounts/%s/d1/database/%s/export", s.accountID, databaseID)

	var bookmark string
	for attempt := 0; attempt < d1ExportMaxPolls; attempt++ {
		body := map[string]any{"output_format": "polling"}
		if bookmark != "" {
			body["current_bookmark"] = bookmark
		}

		resp, err := s.cf.Raw(ctx, http.MethodPost, exportURI, body, nil)
		if err != nil {
			return nil, newError("D1Service.Export", fmt.Sprintf("failed to export database %q", databaseID), err)
		}

		var result struct {
			Status     string `json:"status"`
			AtBookmark string `json:"at_bookmark"`
			SignedURL  string `json:"signed_url"`
		}
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			return nil, newError("D1Service.Export", "failed to parse export response", err)
		}

		if result.SignedURL != "" {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, result.SignedURL, nil)
			if err != nil {
				return nil, newError("D1Service.Export", "failed to build export download request", err)
			}
			httpResp, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, newError("D1Service.Export", "failed to download export dump", err)
			}
			if httpResp.StatusCode != http.StatusOK {
				httpResp.Body.Close()
				return nil, newError("D1Service.Export", fmt.Sprintf("export download returned status %d", httpResp.StatusCode), nil)
			}
			return httpResp.Body, nil
		}

		if result.AtBookmark != "" {
			bookmark = result.AtBookmark
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(d1ExportPollInterval):
		}
	}

	return nil, newError("D1Service.Export", fmt.Sprintf("export of database %q did not complete after %d polls", databaseID, d1ExportMaxPolls), nil)
}

// ensureMigrationsTable creates the d1_migrations tracking table if it
// does not already exist.
func (s *D1Service) ensureMigrationsTable(ctx context.Context, databaseID string) error {
	const createTableSQL = "CREATE TABLE IF NOT EXISTS d1_migrations (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE, applied_at TEXT NOT NULL)"
	if _, err := s.Query(ctx, databaseID, createTableSQL); err != nil {
		return newError("D1Service.MigrationsList", "failed to ensure d1_migrations table exists", err)
	}
	return nil
}

// appliedMigrations returns the set of migration names already recorded
// in the d1_migrations table, keyed by name.
func (s *D1Service) appliedMigrations(ctx context.Context, databaseID string) (map[string]time.Time, error) {
	results, err := s.Query(ctx, databaseID, "SELECT name, applied_at FROM d1_migrations ORDER BY id")
	if err != nil {
		return nil, newError("D1Service.MigrationsList", "failed to query applied migrations", err)
	}

	applied := make(map[string]time.Time)
	for _, r := range results {
		for _, row := range r.Rows {
			name, _ := row["name"].(string)
			if name == "" {
				continue
			}
			appliedAtStr, _ := row["applied_at"].(string)
			t, err := time.Parse(time.RFC3339, appliedAtStr)
			if err != nil {
				t = time.Time{}
			}
			applied[name] = t
		}
	}
	return applied, nil
}

// listMigrationFiles returns the sorted names of files in migrationsDir
// matching the NNNN_<name>.sql convention. A missing directory yields an
// empty (nil error) result.
func listMigrationFiles(migrationsDir string) ([]string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if migrationFileRe.MatchString(e.Name()) {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

// nextMigrationSequence scans migrationsDir for existing migration files
// and returns the next sequence number to use.
func nextMigrationSequence(migrationsDir string) (int, error) {
	files, err := listMigrationFiles(migrationsDir)
	if err != nil {
		return 0, err
	}

	max := 0
	for _, name := range files {
		m := migrationFileRe.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return max + 1, nil
}

// slugifyMigrationName normalizes a migration name into a filesystem-safe,
// lowercase, underscore-separated slug.
func slugifyMigrationName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-', r == '_':
			b.WriteRune('_')
		}
	}
	slug := b.String()
	if slug == "" {
		slug = "migration"
	}
	return slug
}

// mapD1Database converts a cloudflare.D1Database to our D1Database type.
func mapD1Database(db cloudflare.D1Database) *D1Database {
	return &D1Database{
		UUID:      db.UUID,
		Name:      db.Name,
		Version:   db.Version,
		NumTables: db.NumTables,
		FileSize:  db.FileSize,
		CreatedAt: db.CreatedAt,
	}
}

// mapD1Result converts a cloudflare.D1Result to our D1QueryResult type.
func mapD1Result(r cloudflare.D1Result) *D1QueryResult {
	success := false
	if r.Success != nil {
		success = *r.Success
	}

	changedDB := false
	if r.Meta.ChangedDB != nil {
		changedDB = *r.Meta.ChangedDB
	}

	return &D1QueryResult{
		Rows:    r.Results,
		Success: success,
		Meta: D1QueryMeta{
			ChangedDB:   changedDB,
			Changes:     r.Meta.Changes,
			Duration:    r.Meta.Duration,
			LastRowID:   r.Meta.LastRowID,
			RowsRead:    r.Meta.RowsRead,
			RowsWritten: r.Meta.RowsWritten,
			SizeAfter:   r.Meta.SizeAfter,
		},
	}
}
