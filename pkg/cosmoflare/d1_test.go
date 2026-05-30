package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func d1MockSetup(handler http.HandlerFunc) (*D1Service, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewD1Service(cf, "account-test-123")
	return svc, server
}

func d1WriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func TestNewD1ServiceValidation(t *testing.T) {
	_, err := NewD1Service(nil, "account123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewD1Service(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}
}

func TestNewD1ServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewD1Service(cf, "account123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
}

func TestNewD1ServiceFromCredsValidation(t *testing.T) {
	_, err := NewD1ServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewD1ServiceFromCreds("account123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}
}

func TestNewD1ServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewD1ServiceFromCreds("account123", "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
}

func TestD1CreateValidation(t *testing.T) {
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Create(context.Background(), "")
	if err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestD1CreateSuccess(t *testing.T) {
	now := time.Now().UTC()
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		d1WriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"uuid":       "db-uuid-123",
				"name":       "my-database",
				"version":    "production",
				"num_tables": 0,
				"file_size":  0,
				"created_at": now.Format(time.RFC3339),
			},
		})
	})
	defer server.Close()

	db, err := svc.Create(context.Background(), "my-database")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if db.UUID != "db-uuid-123" {
		t.Errorf("expected UUID=db-uuid-123, got %s", db.UUID)
	}
	if db.Name != "my-database" {
		t.Errorf("expected Name=my-database, got %s", db.Name)
	}
}

func TestD1CreateError(t *testing.T) {
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		d1WriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "internal error"}},
		})
	})
	defer server.Close()

	_, err := svc.Create(context.Background(), "my-db")
	if err == nil {
		t.Error("expected error on server failure")
	}
}

func TestD1ListSuccess(t *testing.T) {
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		d1WriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"uuid": "db1", "name": "first", "version": "production", "num_tables": 3, "file_size": 1024},
				{"uuid": "db2", "name": "second", "version": "production", "num_tables": 1, "file_size": 512},
			},
			"result_info": map[string]interface{}{
				"page": 1, "per_page": 50, "total_count": 2, "count": 2,
			},
		})
	})
	defer server.Close()

	dbs, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dbs) != 2 {
		t.Fatalf("expected 2 databases, got %d", len(dbs))
	}
	if dbs[0].UUID != "db1" || dbs[0].Name != "first" {
		t.Errorf("first db mismatch: %+v", dbs[0])
	}
	if dbs[1].NumTables != 1 {
		t.Errorf("expected NumTables=1, got %d", dbs[1].NumTables)
	}
}

func TestD1ListError(t *testing.T) {
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		d1WriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "forbidden"}},
		})
	})
	defer server.Close()

	_, err := svc.List(context.Background())
	if err == nil {
		t.Error("expected error on forbidden")
	}
}

func TestD1GetValidation(t *testing.T) {
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Get(context.Background(), "")
	if err == nil {
		t.Error("expected error when databaseID is empty")
	}
}

func TestD1GetSuccess(t *testing.T) {
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		d1WriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"uuid": "db-get-123", "name": "test-db", "version": "production",
				"num_tables": 5, "file_size": 2048,
			},
		})
	})
	defer server.Close()

	db, err := svc.Get(context.Background(), "db-get-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if db.UUID != "db-get-123" {
		t.Errorf("expected UUID=db-get-123, got %s", db.UUID)
	}
	if db.FileSize != 2048 {
		t.Errorf("expected FileSize=2048, got %d", db.FileSize)
	}
}

func TestD1DeleteValidation(t *testing.T) {
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	err := svc.Delete(context.Background(), "")
	if err == nil {
		t.Error("expected error when databaseID is empty")
	}
}

func TestD1DeleteSuccess(t *testing.T) {
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		d1WriteJSON(w, map[string]interface{}{"success": true, "errors": []interface{}{}})
	})
	defer server.Close()

	err := svc.Delete(context.Background(), "db-delete-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestD1DeleteError(t *testing.T) {
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		d1WriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "not found"}},
		})
	})
	defer server.Close()

	err := svc.Delete(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error on not found")
	}
}

func TestD1QueryValidation(t *testing.T) {
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.Query(context.Background(), "", "SELECT 1")
	if err == nil {
		t.Error("expected error when databaseID is empty")
	}

	_, err = svc.Query(context.Background(), "db-123", "")
	if err == nil {
		t.Error("expected error when SQL is empty")
	}
}

func TestD1QuerySuccess(t *testing.T) {
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		success := true
		changedDB := false
		d1WriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{
					"results": []map[string]interface{}{
						{"id": 1, "name": "alice"},
						{"id": 2, "name": "bob"},
					},
					"success": &success,
					"meta": map[string]interface{}{
						"changed_db":   &changedDB,
						"changes":      0,
						"duration":     0.5,
						"last_row_id":  0,
						"rows_read":    2,
						"rows_written": 0,
						"size_after":   4096,
					},
				},
			},
		})
	})
	defer server.Close()

	results, err := svc.Query(context.Background(), "db-123", "SELECT * FROM users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result set, got %d", len(results))
	}
	if results[0].Meta.RowsRead != 2 {
		t.Errorf("expected RowsRead=2, got %d", results[0].Meta.RowsRead)
	}
}

func TestD1QueryError(t *testing.T) {
	svc, server := d1MockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		d1WriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "syntax error"}},
		})
	})
	defer server.Close()

	_, err := svc.Query(context.Background(), "db-123", "INVALID SQL")
	if err == nil {
		t.Error("expected error on bad SQL")
	}
}

func TestMapD1Database(t *testing.T) {
	now := time.Now().UTC()
	db := mapD1Database(cloudflare.D1Database{
		UUID:      "test-uuid",
		Name:      "test-db",
		Version:   "production",
		NumTables: 7,
		FileSize:  8192,
		CreatedAt: &now,
	})

	if db.UUID != "test-uuid" {
		t.Errorf("expected UUID=test-uuid, got %s", db.UUID)
	}
	if db.NumTables != 7 {
		t.Errorf("expected NumTables=7, got %d", db.NumTables)
	}
	if db.CreatedAt == nil || !db.CreatedAt.Equal(now) {
		t.Error("CreatedAt mismatch")
	}
}

func TestMapD1Result(t *testing.T) {
	success := true
	changedDB := true
	result := mapD1Result(cloudflare.D1Result{
		Success: &success,
		Results: []map[string]any{{"id": 1}},
		Meta: cloudflare.D1DatabaseMetadata{
			ChangedDB:   &changedDB,
			Changes:     5,
			Duration:    1.2,
			LastRowID:   10,
			RowsRead:    20,
			RowsWritten: 5,
			SizeAfter:   16384,
		},
	})

	if !result.Success {
		t.Error("expected Success=true")
	}
	if !result.Meta.ChangedDB {
		t.Error("expected ChangedDB=true")
	}
	if result.Meta.Changes != 5 {
		t.Errorf("expected Changes=5, got %d", result.Meta.Changes)
	}
	if result.Meta.RowsWritten != 5 {
		t.Errorf("expected RowsWritten=5, got %d", result.Meta.RowsWritten)
	}
}

func TestMapD1ResultNilPointers(t *testing.T) {
	result := mapD1Result(cloudflare.D1Result{
		Success: nil,
		Results: nil,
		Meta:    cloudflare.D1DatabaseMetadata{ChangedDB: nil},
	})

	if result.Success {
		t.Error("expected Success=false for nil pointer")
	}
	if result.Meta.ChangedDB {
		t.Error("expected ChangedDB=false for nil pointer")
	}
}
