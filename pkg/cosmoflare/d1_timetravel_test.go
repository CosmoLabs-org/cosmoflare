package cosmoflare

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// withTempCacheDir chdirs into a fresh temp directory for the duration of
// the test so the time-travel quota cache file never touches $HOME or the
// repo working tree.
func withTempCacheDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir into temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
}

func TestTimeTravelQuotaCheck_Empty(t *testing.T) {
	withTempCacheDir(t)
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	quota, err := svc.TimeTravelQuotaCheck("db-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quota.Used != 0 {
		t.Errorf("expected Used=0, got %d", quota.Used)
	}
	if quota.Limit != 10 {
		t.Errorf("expected Limit=10, got %d", quota.Limit)
	}
}

func TestTimeTravelQuotaCheck_Validation(t *testing.T) {
	withTempCacheDir(t)
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if _, err := svc.TimeTravelQuotaCheck(""); err == nil {
		t.Error("expected error when databaseID is empty")
	}
}

func TestTimeTravelQuotaCheck_CountsWithinWindow(t *testing.T) {
	withTempCacheDir(t)
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	now := time.Now().UTC()
	cache := timeTravelCache{
		"db-1": {
			{Timestamp: now, RestoredAt: now.Add(-1 * time.Minute)},
			{Timestamp: now, RestoredAt: now.Add(-2 * time.Minute)},
		},
	}
	if err := saveTimeTravelCache(cache); err != nil {
		t.Fatalf("failed to seed cache: %v", err)
	}

	quota, err := svc.TimeTravelQuotaCheck("db-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quota.Used != 2 {
		t.Errorf("expected Used=2, got %d", quota.Used)
	}
}

func TestTimeTravelQuotaCheck_WindowExpiry(t *testing.T) {
	withTempCacheDir(t)
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	now := time.Now().UTC()
	cache := timeTravelCache{
		"db-1": {
			{Timestamp: now, RestoredAt: now.Add(-15 * time.Minute)}, // outside window
			{Timestamp: now, RestoredAt: now.Add(-1 * time.Minute)},  // inside window
		},
	}
	if err := saveTimeTravelCache(cache); err != nil {
		t.Fatalf("failed to seed cache: %v", err)
	}

	quota, err := svc.TimeTravelQuotaCheck("db-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quota.Used != 1 {
		t.Errorf("expected Used=1 (expired entry excluded), got %d", quota.Used)
	}
}

func TestTimeTravelQuotaCheck_OtherDatabaseIsolated(t *testing.T) {
	withTempCacheDir(t)
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	now := time.Now().UTC()
	cache := timeTravelCache{
		"db-1": {{Timestamp: now, RestoredAt: now}},
	}
	if err := saveTimeTravelCache(cache); err != nil {
		t.Fatalf("failed to seed cache: %v", err)
	}

	quota, err := svc.TimeTravelQuotaCheck("db-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quota.Used != 0 {
		t.Errorf("expected Used=0 for unrelated database, got %d", quota.Used)
	}
}

func TestTimeTravelCache_RoundTrip(t *testing.T) {
	withTempCacheDir(t)

	now := time.Now().UTC().Truncate(time.Second)
	cache := timeTravelCache{
		"db-1": {{Timestamp: now, RestoredAt: now}},
	}
	if err := saveTimeTravelCache(cache); err != nil {
		t.Fatalf("failed to save cache: %v", err)
	}

	if _, err := os.Stat(timeTravelCacheFile); err != nil {
		t.Fatalf("expected cache file in cwd: %v", err)
	}

	loaded, err := loadTimeTravelCache()
	if err != nil {
		t.Fatalf("failed to load cache: %v", err)
	}
	if len(loaded["db-1"]) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded["db-1"]))
	}
	if !loaded["db-1"][0].RestoredAt.Equal(now) {
		t.Errorf("expected RestoredAt=%v, got %v", now, loaded["db-1"][0].RestoredAt)
	}
}

func TestLoadTimeTravelCache_MissingFile(t *testing.T) {
	withTempCacheDir(t)

	cache, err := loadTimeTravelCache()
	if err != nil {
		t.Fatalf("unexpected error for missing cache file: %v", err)
	}
	if len(cache) != 0 {
		t.Errorf("expected empty cache, got %d entries", len(cache))
	}
}

func TestTimeTravelRestore_Validation(t *testing.T) {
	withTempCacheDir(t)
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if _, err := svc.TimeTravelRestore(context.Background(), "", time.Now()); err == nil {
		t.Error("expected error when databaseID is empty")
	}
	if _, err := svc.TimeTravelRestore(context.Background(), "db-1", time.Time{}); err == nil {
		t.Error("expected error when timestamp is zero")
	}
}

func TestTimeTravelRestore_QuotaExceeded(t *testing.T) {
	withTempCacheDir(t)
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		t.Error("API should not be called when quota is exhausted")
	})
	defer server.Close()

	now := time.Now().UTC()
	var entries []timeTravelCacheEntry
	for i := 0; i < 10; i++ {
		entries = append(entries, timeTravelCacheEntry{Timestamp: now, RestoredAt: now.Add(-time.Duration(i) * time.Minute)})
	}
	cache := timeTravelCache{"db-1": entries}
	if err := saveTimeTravelCache(cache); err != nil {
		t.Fatalf("failed to seed cache: %v", err)
	}

	if _, err := svc.TimeTravelRestore(context.Background(), "db-1", time.Now()); err == nil {
		t.Error("expected quota exceeded error")
	}
}

func TestTimeTravelRestore_Success(t *testing.T) {
	withTempCacheDir(t)

	var gotBookmarkReq, gotRestoreReq bool
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "time_travel/bookmark"):
			gotBookmarkReq = true
			d1WriteJSON(w, map[string]any{
				"success": true,
				"result":  map[string]string{"bookmark": "bm-123"},
			})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/restore"):
			gotRestoreReq = true
			d1WriteJSON(w, map[string]any{
				"success": true,
				"result":  map[string]any{},
			})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	result, err := svc.TimeTravelRestore(context.Background(), "db-1", ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gotBookmarkReq {
		t.Error("expected bookmark endpoint to be called")
	}
	if !gotRestoreReq {
		t.Error("expected restore endpoint to be called")
	}
	if !result.Success {
		t.Error("expected Success=true")
	}
	if result.DatabaseID != "db-1" {
		t.Errorf("expected DatabaseID=db-1, got %s", result.DatabaseID)
	}

	quota, err := svc.TimeTravelQuotaCheck("db-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if quota.Used != 1 {
		t.Errorf("expected Used=1 after restore, got %d", quota.Used)
	}
}

func TestD1Export_Success(t *testing.T) {
	oldInterval := d1ExportPollInterval
	d1ExportPollInterval = time.Millisecond
	defer func() { d1ExportPollInterval = oldInterval }()

	dumpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "-- SQL dump\nCREATE TABLE t (id INTEGER);\n")
	}))
	defer dumpServer.Close()

	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		d1WriteJSON(w, map[string]any{
			"success": true,
			"result": map[string]any{
				"status":     "complete",
				"signed_url": dumpServer.URL,
			},
		})
	})
	defer server.Close()

	rc, err := svc.Export(context.Background(), "db-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read export body: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty export dump")
	}
}

func TestD1Export_Polling(t *testing.T) {
	oldInterval := d1ExportPollInterval
	d1ExportPollInterval = time.Millisecond
	defer func() { d1ExportPollInterval = oldInterval }()

	dumpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "-- dump\n")
	}))
	defer dumpServer.Close()

	calls := 0
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var req struct {
			CurrentBookmark string `json:"current_bookmark"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		if calls < 3 {
			d1WriteJSON(w, map[string]any{
				"success": true,
				"result": map[string]any{
					"status":      "in_progress",
					"at_bookmark": fmt.Sprintf("bm-%d", calls),
				},
			})
			return
		}
		d1WriteJSON(w, map[string]any{
			"success": true,
			"result": map[string]any{
				"status":     "complete",
				"signed_url": dumpServer.URL,
			},
		})
	})
	defer server.Close()

	rc, err := svc.Export(context.Background(), "db-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	if calls < 3 {
		t.Errorf("expected at least 3 polling calls, got %d", calls)
	}
}

func TestD1Export_Validation(t *testing.T) {
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if _, err := svc.Export(context.Background(), ""); err == nil {
		t.Error("expected error when databaseID is empty")
	}
}
