package r2go2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

func kvMockSetup(handler http.HandlerFunc) (*KVService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewKVService(cf, "acct-kv-123")
	return svc, server
}

func kvWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func TestNewKVServiceValidation(t *testing.T) {
	_, err := NewKVService(nil, "account123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewKVService(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}

	_, err = NewKVService(nil, "")
	if err == nil {
		t.Error("expected error when both are empty")
	}
}

func TestNewKVServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewKVService(cf, "account123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "account123" {
		t.Errorf("expected accountID=account123, got %s", svc.accountID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

func TestKVOptions(t *testing.T) {
	cfg := &kvConfig{}
	WithKVTTL(3600)(cfg)
	if cfg.ttl != 3600 {
		t.Errorf("expected ttl=3600, got %d", cfg.ttl)
	}

	meta := map[string]string{"env": "prod", "version": "2"}
	WithKVMetadata(meta)(cfg)
	if cfg.metadata["env"] != "prod" {
		t.Errorf("expected metadata env=prod, got %s", cfg.metadata["env"])
	}
}

func TestKVListOptions(t *testing.T) {
	cfg := &kvListConfig{}
	WithKVPrefix("cache/")(cfg)
	if cfg.prefix != "cache/" {
		t.Errorf("expected prefix=cache/, got %s", cfg.prefix)
	}

	WithKVLimit(100)(cfg)
	if cfg.limit != 100 {
		t.Errorf("expected limit=100, got %d", cfg.limit)
	}

	WithKVCursor("abc123")(cfg)
	if cfg.cursor != "abc123" {
		t.Errorf("expected cursor=abc123, got %s", cfg.cursor)
	}
}

func TestKVNamespaceType(t *testing.T) {
	ns := &KVNamespace{ID: "ns-abc", Title: "my-namespace"}
	if ns.ID != "ns-abc" {
		t.Errorf("unexpected ID: %s", ns.ID)
	}
	if ns.Title != "my-namespace" {
		t.Errorf("unexpected title: %s", ns.Title)
	}
}

func TestKVKeyType(t *testing.T) {
	k := &KVKey{Key: "user:123", Expiration: 1735689600}
	if k.Key != "user:123" {
		t.Errorf("unexpected key: %s", k.Key)
	}
	if k.Expiration != 1735689600 {
		t.Errorf("unexpected expiration: %d", k.Expiration)
	}
}

func TestKVCreateNamespaceValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewKVService(cf, "account123")

	_, err := svc.CreateNamespace(nil, "")
	if err == nil {
		t.Error("expected error when title is empty")
	}
	if !strings.Contains(err.Error(), "namespace title is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestKVGetNamespaceValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewKVService(cf, "account123")

	_, err := svc.GetNamespace(nil, "")
	if err == nil {
		t.Error("expected error when ID is empty")
	}
}

func TestKVDeleteNamespaceValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewKVService(cf, "account123")

	err := svc.DeleteNamespace(nil, "")
	if err == nil {
		t.Error("expected error when ID is empty")
	}
}

func TestKVPutValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewKVService(cf, "account123")

	err := svc.Put(nil, "", "key", strings.NewReader("val"))
	if err == nil {
		t.Error("expected error when namespaceID is empty")
	}

	err = svc.Put(nil, "ns-1", "", strings.NewReader("val"))
	if err == nil {
		t.Error("expected error when key is empty")
	}

	err = svc.Put(nil, "ns-1", "key", nil)
	if err == nil {
		t.Error("expected error when value is nil")
	}
}

func TestKVGetValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewKVService(cf, "account123")

	_, err := svc.Get(nil, "", "key")
	if err == nil {
		t.Error("expected error when namespaceID is empty")
	}

	_, err = svc.Get(nil, "ns-1", "")
	if err == nil {
		t.Error("expected error when key is empty")
	}
}

func TestKVDeleteKeyValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewKVService(cf, "account123")

	err := svc.Delete(nil, "", "key")
	if err == nil {
		t.Error("expected error when namespaceID is empty")
	}

	err = svc.Delete(nil, "ns-1", "")
	if err == nil {
		t.Error("expected error when key is empty")
	}
}

func TestKVListKeysValidation(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, _ := NewKVService(cf, "account123")

	_, err := svc.ListKeys(nil, "")
	if err == nil {
		t.Error("expected error when namespaceID is empty")
	}
}

// --- httptest-based API mock tests ---

func TestKVCreateNamespaceWithMock(t *testing.T) {
	svc, server := kvMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		kvWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id":    "ns-new-001",
				"title": "my-namespace",
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	ns, err := svc.CreateNamespace(ctx, "my-namespace")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ns.ID != "ns-new-001" {
		t.Errorf("expected ID=ns-new-001, got %s", ns.ID)
	}
	if ns.Title != "my-namespace" {
		t.Errorf("expected Title=my-namespace, got %s", ns.Title)
	}
}

func TestKVListNamespacesWithMock(t *testing.T) {
	svc, server := kvMockSetup(func(w http.ResponseWriter, r *http.Request) {
		kvWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "ns-001", "title": "cache"},
				{"id": "ns-002", "title": "sessions"},
			},
			"result_info": map[string]interface{}{"page": 1, "total_pages": 1, "count": 2},
		})
	})
	defer server.Close()

	ctx := context.Background()
	namespaces, err := svc.ListNamespaces(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(namespaces) != 2 {
		t.Fatalf("expected 2 namespaces, got %d", len(namespaces))
	}
	if namespaces[0].ID != "ns-001" {
		t.Errorf("expected first ID=ns-001, got %s", namespaces[0].ID)
	}
	if namespaces[1].Title != "sessions" {
		t.Errorf("expected second Title=sessions, got %s", namespaces[1].Title)
	}
}

func TestKVGetNamespaceWithMock(t *testing.T) {
	svc, server := kvMockSetup(func(w http.ResponseWriter, r *http.Request) {
		kvWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "ns-001", "title": "cache"},
				{"id": "ns-target", "title": "target-ns"},
			},
			"result_info": map[string]interface{}{"page": 1, "total_pages": 1, "count": 2},
		})
	})
	defer server.Close()

	ctx := context.Background()
	ns, err := svc.GetNamespace(ctx, "ns-target")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ns.ID != "ns-target" {
		t.Errorf("expected ID=ns-target, got %s", ns.ID)
	}
	if ns.Title != "target-ns" {
		t.Errorf("expected Title=target-ns, got %s", ns.Title)
	}
}

func TestKVGetNamespaceNotFound(t *testing.T) {
	svc, server := kvMockSetup(func(w http.ResponseWriter, r *http.Request) {
		kvWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "ns-001", "title": "cache"},
			},
			"result_info": map[string]interface{}{"page": 1, "total_pages": 1, "count": 1},
		})
	})
	defer server.Close()

	ctx := context.Background()
	_, err := svc.GetNamespace(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestKVDeleteNamespaceWithMock(t *testing.T) {
	svc, server := kvMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		kvWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"id": "ns-del-001"},
		})
	})
	defer server.Close()

	ctx := context.Background()
	err := svc.DeleteNamespace(ctx, "ns-del-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKVPutWithMock(t *testing.T) {
	svc, server := kvMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != "test-value" {
			t.Errorf("expected body 'test-value', got %q", string(body))
		}
		kvWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{},
		})
	})
	defer server.Close()

	ctx := context.Background()
	err := svc.Put(ctx, "ns-001", "my-key", strings.NewReader("test-value"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKVGetWithMock(t *testing.T) {
	svc, server := kvMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("retrieved-value"))
	})
	defer server.Close()

	ctx := context.Background()
	val, err := svc.Get(ctx, "ns-001", "my-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(val) != "retrieved-value" {
		t.Errorf("expected 'retrieved-value', got %q", string(val))
	}
}

func TestKVDeleteKeyWithMock(t *testing.T) {
	svc, server := kvMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		kvWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{},
		})
	})
	defer server.Close()

	ctx := context.Background()
	err := svc.Delete(ctx, "ns-001", "my-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKVListKeysWithMock(t *testing.T) {
	svc, server := kvMockSetup(func(w http.ResponseWriter, r *http.Request) {
		kvWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"name": "key1", "expiration": 1735689600},
				{"name": "key2", "expiration": 0},
			},
			"result_info": map[string]interface{}{
				"cursors": map[string]interface{}{"before": "", "after": "cursor-next"},
			},
		})
	})
	defer server.Close()

	ctx := context.Background()
	result, err := svc.ListKeys(ctx, "ns-001", WithKVPrefix("cache/"), WithKVLimit(10))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(result.Items))
	}
	if result.Items[0].Key != "key1" {
		t.Errorf("expected first key=key1, got %s", result.Items[0].Key)
	}
	if !result.IsTruncated {
		t.Error("expected IsTruncated=true")
	}
	if result.NextToken != "cursor-next" {
		t.Errorf("expected NextToken=cursor-next, got %s", result.NextToken)
	}
}

func TestKVCreateNamespaceAPIError(t *testing.T) {
	svc, server := kvMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		kvWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1001, "message": "invalid request"}},
		})
	})
	defer server.Close()

	ctx := context.Background()
	_, err := svc.CreateNamespace(ctx, "bad-ns")
	if err == nil {
		t.Fatal("expected error from API")
	}
	if _, ok := err.(*R2Error); !ok {
		t.Errorf("expected *R2Error, got %T", err)
	}
}

func TestKVNamespaceJSONMarshal(t *testing.T) {
	ns := &KVNamespace{ID: "ns-json-001", Title: "json-ns"}
	data, err := json.Marshal(ns)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded KVNamespace
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.ID != "ns-json-001" {
		t.Errorf("ID mismatch: got %q", decoded.ID)
	}
	if decoded.Title != "json-ns" {
		t.Errorf("Title mismatch: got %q", decoded.Title)
	}
}

func TestKVKeyJSONMarshal(t *testing.T) {
	k := &KVKey{Key: "user:123", Expiration: 1735689600}
	data, err := json.Marshal(k)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var decoded KVKey
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if decoded.Key != "user:123" {
		t.Errorf("Key mismatch: got %q", decoded.Key)
	}
	if decoded.Expiration != 1735689600 {
		t.Errorf("Expiration mismatch: got %d", decoded.Expiration)
	}
}

func TestKVNewFromCredsSuccess(t *testing.T) {
	svc, err := NewKVServiceFromCreds("acct123", fmt.Sprintf("test-token-%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
}

func TestKVPutWithMetadata(t *testing.T) {
	var receivedBody []byte
	svc, server := kvMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/accounts/acct-kv-123/storage/kv/namespaces/ns-001/bulk" {
			t.Errorf("expected bulk endpoint, got %s", r.URL.Path)
		}
		receivedBody, _ = io.ReadAll(r.Body)
		kvWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{},
		})
	})
	defer server.Close()

	ctx := context.Background()
	err := svc.Put(ctx, "ns-001", "my-key", strings.NewReader("test-value"),
		WithKVMetadata(map[string]string{"env": "prod"}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var pairs []map[string]interface{}
	if err := json.Unmarshal(receivedBody, &pairs); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("expected 1 pair, got %d", len(pairs))
	}
	if pairs[0]["key"] != "my-key" {
		t.Errorf("expected key=my-key, got %v", pairs[0]["key"])
	}
	meta, ok := pairs[0]["metadata"].(map[string]interface{})
	if !ok {
		t.Fatal("metadata not found in request body")
	}
	if meta["env"] != "prod" {
		t.Errorf("expected metadata env=prod, got %v", meta["env"])
	}
}

func TestKVPutWithTTL(t *testing.T) {
	var receivedBody []byte
	svc, server := kvMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/acct-kv-123/storage/kv/namespaces/ns-001/bulk" {
			t.Errorf("expected bulk endpoint for TTL, got %s", r.URL.Path)
		}
		receivedBody, _ = io.ReadAll(r.Body)
		kvWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{},
		})
	})
	defer server.Close()

	ctx := context.Background()
	err := svc.Put(ctx, "ns-001", "ttl-key", strings.NewReader("data"), WithKVTTL(3600))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var pairs []map[string]interface{}
	if err := json.Unmarshal(receivedBody, &pairs); err != nil {
		t.Fatalf("failed to parse body: %v", err)
	}
	if pairs[0]["expiration_ttl"] != float64(3600) {
		t.Errorf("expected expiration_ttl=3600, got %v", pairs[0]["expiration_ttl"])
	}
}

func TestKVPutWithoutOptsUsesSingleEndpoint(t *testing.T) {
	svc, server := kvMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/accounts/acct-kv-123/storage/kv/namespaces/ns-001/bulk" {
			t.Error("should use single-entry endpoint when no metadata/TTL, got bulk")
		}
		kvWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{},
		})
	})
	defer server.Close()

	ctx := context.Background()
	err := svc.Put(ctx, "ns-001", "plain-key", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
