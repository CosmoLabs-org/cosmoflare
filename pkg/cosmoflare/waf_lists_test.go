package cosmoflare

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

// wafListsMockSetup creates a mock Cloudflare API server and WAFListService.
func wafListsMockSetup(handler http.HandlerFunc) (*WAFListService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewWAFListService(cf, "acct-waf-lists")
	return svc, server
}

// wafListsWriteJSON writes a JSON response envelope.
func wafListsWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func wafListsEnvelope(result interface{}) map[string]interface{} {
	return map[string]interface{}{
		"result":   result,
		"success":  true,
		"errors":   []interface{}{},
		"messages": []interface{}{},
	}
}

// wafListsRoute dispatches on method+path for list/item/entrypoint mocks.
func wafListsRoute(w http.ResponseWriter, r *http.Request, listItems []map[string]interface{}) {
	path := r.URL.Path
	switch {
	case r.Method == http.MethodGet && strings.HasSuffix(path, "/rules/lists"):
		wafListsWriteJSON(w, wafListsEnvelope([]map[string]interface{}{
			{"id": "list-1", "name": "blocked-ips", "kind": "ip", "num_items": 3, "description": "d1"},
			{"id": "list-2", "name": "bad-asns", "kind": "asn", "num_items": 1},
		}))
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/rules/lists"):
		wafListsWriteJSON(w, wafListsEnvelope(map[string]interface{}{
			"id": "list-new", "name": "blocked-ips", "kind": "ip", "num_items": 0,
		}))
	case r.Method == http.MethodPut && strings.Contains(path, "/rules/lists/"):
		wafListsWriteJSON(w, wafListsEnvelope(map[string]interface{}{
			"id": "list-1", "name": "blocked-ips", "kind": "ip", "description": "updated",
		}))
	case r.Method == http.MethodDelete && strings.Contains(path, "/rules/lists/"):
		wafListsWriteJSON(w, wafListsEnvelope(map[string]interface{}{"id": "list-1"}))
	case r.Method == http.MethodGet && strings.Contains(path, "/bulk_operations/"):
		wafListsWriteJSON(w, wafListsEnvelope(map[string]interface{}{"status": "completed"}))
	case (r.Method == http.MethodPost || r.Method == http.MethodPut) && strings.HasSuffix(path, "/items"):
		wafListsWriteJSON(w, wafListsEnvelope(map[string]interface{}{"operation_id": "op-1"}))
	case r.Method == http.MethodGet && strings.HasSuffix(path, "/items"):
		wafListsWriteJSON(w, wafListsEnvelope(listItems))
	default:
		wafListsWriteJSON(w, wafListsEnvelope(nil))
	}
}

// --- Constructor tests ---

func TestNewWAFListServiceValidation(t *testing.T) {
	_, err := NewWAFListService(nil, "acct123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewWAFListService(cf, "")
	if err == nil {
		t.Error("expected error when accountID is empty")
	}
}

func TestNewWAFListServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewWAFListService(cf, "acct123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
	if svc.cf == nil {
		t.Error("expected cf client to be set")
	}
}

func TestNewWAFListServiceFromCredsValidation(t *testing.T) {
	if _, err := NewWAFListServiceFromCreds("", "token"); err == nil {
		t.Error("expected error when accountID is empty")
	}
	if _, err := NewWAFListServiceFromCreds("acct123", ""); err == nil {
		t.Error("expected error when apiToken is empty")
	}
}

func TestNewWAFListServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewWAFListServiceFromCreds("acct123", "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.accountID != "acct123" {
		t.Errorf("expected accountID=acct123, got %s", svc.accountID)
	}
}

// --- ListLists ---

func TestWAFListsListListsSuccess(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/acct-waf-lists/rules/lists" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		wafListsWriteJSON(w, wafListsEnvelope([]map[string]interface{}{
			{"id": "list-1", "name": "blocked-ips", "kind": "ip", "num_items": 3, "description": "d1",
				"created_on": "2024-01-01T00:00:00Z", "modified_on": "2024-02-02T00:00:00Z"},
			{"id": "list-2", "name": "bad-asns", "kind": "asn", "num_items": 1},
		}))
	})
	defer server.Close()

	lists, err := svc.ListLists(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lists) != 2 {
		t.Fatalf("expected 2 lists, got %d", len(lists))
	}
	if lists[0].ID != "list-1" || lists[0].Name != "blocked-ips" || lists[0].Kind != "ip" || lists[0].NumItems != 3 {
		t.Errorf("first list mismatch: %+v", lists[0])
	}
	if lists[0].CreatedOn == "" || lists[0].ModifiedOn == "" {
		t.Errorf("expected timestamps parsed, got %+v", lists[0])
	}
}

func TestWAFListsListListsError(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		wafListsWriteJSON(w, map[string]interface{}{
			"result":  nil,
			"success": false,
			"errors":  []map[string]interface{}{{"code": 10000, "message": "boom"}},
		})
	})
	defer server.Close()

	if _, err := svc.ListLists(context.Background()); err == nil {
		t.Error("expected error for 500 response")
	}
}

// --- CreateList ---

func TestWAFListsCreateListValidation(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if _, err := svc.CreateList(context.Background(), "", "ip", ""); err == nil {
		t.Error("expected error for empty name")
	}
	if _, err := svc.CreateList(context.Background(), "name", "", ""); err == nil {
		t.Error("expected error for empty kind")
	}
	if _, err := svc.CreateList(context.Background(), "name", "bogus", ""); err == nil {
		t.Error("expected error for invalid kind")
	}
}

func TestWAFListsCreateListSuccess(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafListsRoute(w, r, nil)
	})
	defer server.Close()

	list, err := svc.CreateList(context.Background(), "blocked-ips", WAFListKindIP, "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list.ID != "list-new" || list.Name != "blocked-ips" || list.Kind != "ip" {
		t.Errorf("created list mismatch: %+v", list)
	}
}

func TestWAFListsCreateListError(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		wafListsWriteJSON(w, map[string]interface{}{
			"result":  nil,
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1001, "message": "name taken"}},
		})
	})
	defer server.Close()

	if _, err := svc.CreateList(context.Background(), "blocked-ips", "ip", ""); err == nil {
		t.Error("expected error for 400 response")
	}
}

// --- UpdateList ---

func TestWAFListsUpdateListValidation(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if _, err := svc.UpdateList(context.Background(), "", "", "d"); err == nil {
		t.Error("expected error for empty list ID")
	}
	_, err := svc.UpdateList(context.Background(), "list-1", "new-name", "d")
	if err == nil {
		t.Fatal("expected error when renaming a list")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("expected rename-unsupported error, got: %v", err)
	}
}

func TestWAFListsUpdateListSuccess(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafListsRoute(w, r, nil)
	})
	defer server.Close()

	list, err := svc.UpdateList(context.Background(), "list-1", "", "updated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list.Description != "updated" {
		t.Errorf("expected description updated, got %+v", list)
	}
}

// --- DeleteList ---

func TestWAFListsDeleteListValidation(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if err := svc.DeleteList(context.Background(), ""); err == nil {
		t.Error("expected error for empty list ID")
	}
}

func TestWAFListsDeleteListSuccess(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafListsRoute(w, r, nil)
	})
	defer server.Close()

	if err := svc.DeleteList(context.Background(), "list-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWAFListsDeleteListError(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		wafListsWriteJSON(w, map[string]interface{}{
			"result":  nil,
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "not found"}},
		})
	})
	defer server.Close()

	if err := svc.DeleteList(context.Background(), "list-1"); err == nil {
		t.Error("expected error for 404 response")
	}
}

// --- ListItems ---

func TestWAFListsListItemsValidation(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if _, err := svc.ListItems(context.Background(), ""); err == nil {
		t.Error("expected error for empty list ID")
	}
}

func TestWAFListsListItemsSuccess(t *testing.T) {
	asmn := uint32(13335)
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafListsRoute(w, r, []map[string]interface{}{
			{"id": "item-1", "ip": "1.2.3.4", "comment": "scanner"},
			{"id": "item-2", "asn": asmn},
		})
	})
	defer server.Close()

	items, err := svc.ListItems(context.Background(), "list-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].IP != "1.2.3.4" || items[0].Comment != "scanner" {
		t.Errorf("first item mismatch: %+v", items[0])
	}
	if items[1].ASN != 13335 {
		t.Errorf("expected ASN 13335, got %d", items[1].ASN)
	}
}

// --- AddItems / ReplaceItems ---

func TestWAFListsAddItemsValidation(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if _, err := svc.AddItems(context.Background(), "", []WAFListItem{{IP: "1.2.3.4"}}); err == nil {
		t.Error("expected error for empty list ID")
	}
	if _, err := svc.AddItems(context.Background(), "list-1", nil); err == nil {
		t.Error("expected error for empty items")
	}
}

func TestWAFListsAddItemsSuccess(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafListsRoute(w, r, []map[string]interface{}{
			{"id": "item-1", "ip": "1.2.3.4", "comment": "scanner"},
			{"id": "item-2", "asn": 13335},
		})
	})
	defer server.Close()

	items, err := svc.AddItems(context.Background(), "list-1", []WAFListItem{
		{IP: "1.2.3.4", Comment: "scanner"},
		{ASN: 13335},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestWAFListsReplaceItemsValidation(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if _, err := svc.ReplaceItems(context.Background(), "", []WAFListItem{{IP: "1.2.3.4"}}); err == nil {
		t.Error("expected error for empty list ID")
	}
	if _, err := svc.ReplaceItems(context.Background(), "list-1", nil); err == nil {
		t.Error("expected error for empty items")
	}
}

func TestWAFListsReplaceItemsSuccess(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafListsRoute(w, r, []map[string]interface{}{
			{"id": "item-9", "ip": "5.6.7.8"},
		})
	})
	defer server.Close()

	items, err := svc.ReplaceItems(context.Background(), "list-1", []WAFListItem{{IP: "5.6.7.8"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].IP != "5.6.7.8" {
		t.Fatalf("expected replaced item set, got %+v", items)
	}
}

// --- UpdateManagedRuleset ---

func TestWAFListsUpdateManagedRulesetValidation(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if _, err := svc.UpdateManagedRuleset(context.Background(), "", "on"); err == nil {
		t.Error("expected error for empty phase")
	}
	if _, err := svc.UpdateManagedRuleset(context.Background(), "http_request_firewall_managed", "maybe"); err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestWAFListsUpdateManagedRulesetNotFound(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		wafListsWriteJSON(w, map[string]interface{}{
			"result":  nil,
			"success": false,
			"errors":  []map[string]interface{}{{"code": 1000, "message": "not found"}},
		})
	})
	defer server.Close()

	_, err := svc.UpdateManagedRuleset(context.Background(), "http_request_firewall_managed", "on")
	if err == nil {
		t.Fatal("expected error when phase entrypoint is missing")
	}
	var valErr *R2ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("expected validation error for missing entrypoint, got %T: %v", err, err)
	}
}

func TestWAFListsUpdateManagedRulesetSuccess(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/entrypoint"):
			wafListsWriteJSON(w, wafListsEnvelope(map[string]interface{}{
				"id": "rs-1", "name": "default", "phase": "http_request_firewall_managed",
				"rules": []map[string]interface{}{
					{"id": "rule-1", "action": "execute", "expression": "true", "enabled": true},
				},
			}))
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/entrypoint"):
			wafListsWriteJSON(w, wafListsEnvelope(map[string]interface{}{
				"id": "rs-1", "name": "default", "phase": "http_request_firewall_managed",
				"rules": []map[string]interface{}{
					{"id": "rule-1", "action": "execute", "expression": "true", "enabled": false},
				},
			}))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			wafListsWriteJSON(w, wafListsEnvelope(nil))
		}
	})
	defer server.Close()

	result, err := svc.UpdateManagedRuleset(context.Background(), "http_request_firewall_managed", "off")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Phase != "http_request_firewall_managed" || result.Mode != "off" || result.NumRules != 1 {
		t.Errorf("unexpected result: %+v", result)
	}
	if result.RulesetID != "rs-1" {
		t.Errorf("expected ruleset ID rs-1, got %q", result.RulesetID)
	}
}

func TestWAFListsUpdateManagedRulesetEmptyEntrypoint(t *testing.T) {
	svc, server := wafListsMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafListsWriteJSON(w, wafListsEnvelope(map[string]interface{}{
			"id": "rs-1", "name": "default", "phase": "http_request_firewall_managed", "rules": []interface{}{},
		}))
	})
	defer server.Close()

	_, err := svc.UpdateManagedRuleset(context.Background(), "http_request_firewall_managed", "on")
	if err == nil {
		t.Fatal("expected error when entrypoint has no rules")
	}
	if !strings.Contains(err.Error(), "no rules") {
		t.Errorf("expected no-rules error, got: %v", err)
	}
}
