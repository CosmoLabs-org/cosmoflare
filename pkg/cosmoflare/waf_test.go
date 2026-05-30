package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

func wafMockSetup(handler http.HandlerFunc) (*WAFService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewWAFService(cf, "zone-waf-123")
	return svc, server
}

func wafWriteJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func TestNewWAFServiceValidation(t *testing.T) {
	_, err := NewWAFService(nil, "zone123")
	if err == nil {
		t.Error("expected error when API client is nil")
	}

	cf, _ := cloudflare.NewWithAPIToken("test-token")
	_, err = NewWAFService(cf, "")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}
}

func TestNewWAFServiceSuccess(t *testing.T) {
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc, err := NewWAFService(cf, "zone123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.zoneID != "zone123" {
		t.Errorf("expected zoneID=zone123, got %s", svc.zoneID)
	}
}

func TestNewWAFServiceFromCredsValidation(t *testing.T) {
	_, err := NewWAFServiceFromCreds("", "token")
	if err == nil {
		t.Error("expected error when zoneID is empty")
	}

	_, err = NewWAFServiceFromCreds("zone123", "")
	if err == nil {
		t.Error("expected error when apiToken is empty")
	}
}

func TestNewWAFServiceFromCredsSuccess(t *testing.T) {
	svc, err := NewWAFServiceFromCreds("zone123", "valid-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.zoneID != "zone123" {
		t.Errorf("expected zoneID=zone123, got %s", svc.zoneID)
	}
}

func TestWAFListPackagesSuccess(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "pkg1", "name": "OWASP", "description": "OWASP Core Rules", "detection_mode": "anomaly", "sensitivity": "high", "action_mode": "challenge"},
				{"id": "pkg2", "name": "Cloudflare", "description": "Cloudflare Managed Rules", "detection_mode": "traditional", "sensitivity": "medium", "action_mode": "block"},
			},
			"result_info": map[string]interface{}{"page": 1, "per_page": 50, "total_count": 2, "count": 2, "total_pages": 1},
		})
	})
	defer server.Close()

	pkgs, err := svc.ListPackages(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(pkgs))
	}
	if pkgs[0].ID != "pkg1" || pkgs[0].Name != "OWASP" {
		t.Errorf("first package mismatch: %+v", pkgs[0])
	}
	if pkgs[1].DetectionMode != "traditional" {
		t.Errorf("expected DetectionMode=traditional, got %s", pkgs[1].DetectionMode)
	}
}

func TestWAFListPackagesError(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		wafWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "forbidden"}},
		})
	})
	defer server.Close()

	_, err := svc.ListPackages(context.Background())
	if err == nil {
		t.Error("expected error on forbidden")
	}
}

func TestWAFGetPackageValidation(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.GetPackage(context.Background(), "")
	if err == nil {
		t.Error("expected error when packageID is empty")
	}
}

func TestWAFGetPackageSuccess(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id": "pkg1", "name": "OWASP", "description": "Core Rules",
				"detection_mode": "anomaly", "sensitivity": "high", "action_mode": "challenge",
			},
		})
	})
	defer server.Close()

	pkg, err := svc.GetPackage(context.Background(), "pkg1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkg.ID != "pkg1" {
		t.Errorf("expected ID=pkg1, got %s", pkg.ID)
	}
	if pkg.Sensitivity != "high" {
		t.Errorf("expected Sensitivity=high, got %s", pkg.Sensitivity)
	}
}

func TestWAFListRulesValidation(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.ListRules(context.Background(), "")
	if err == nil {
		t.Error("expected error when packageID is empty")
	}
}

func TestWAFListRulesSuccess(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{
					"id": "rule1", "description": "SQL Injection", "priority": "5",
					"package_id": "pkg1", "mode": "block", "default_mode": "block",
					"allowed_modes": []string{"block", "simulate", "disable"},
					"group": map[string]interface{}{"id": "grp1", "name": "SQLi"},
				},
			},
			"result_info": map[string]interface{}{"page": 1, "per_page": 50, "total_count": 1, "count": 1, "total_pages": 1},
		})
	})
	defer server.Close()

	rules, err := svc.ListRules(context.Background(), "pkg1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != "rule1" {
		t.Errorf("expected ID=rule1, got %s", rules[0].ID)
	}
	if rules[0].Group.Name != "SQLi" {
		t.Errorf("expected Group.Name=SQLi, got %s", rules[0].Group.Name)
	}
}

func TestWAFGetRuleValidation(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.GetRule(context.Background(), "", "rule1")
	if err == nil {
		t.Error("expected error when packageID is empty")
	}

	_, err = svc.GetRule(context.Background(), "pkg1", "")
	if err == nil {
		t.Error("expected error when ruleID is empty")
	}
}

func TestWAFGetRuleSuccess(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id": "rule1", "description": "XSS Attack", "priority": "3",
				"package_id": "pkg1", "mode": "simulate", "default_mode": "block",
				"allowed_modes": []string{"block", "simulate", "disable"},
				"group": map[string]interface{}{"id": "grp2", "name": "XSS"},
			},
		})
	})
	defer server.Close()

	rule, err := svc.GetRule(context.Background(), "pkg1", "rule1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.Mode != "simulate" {
		t.Errorf("expected Mode=simulate, got %s", rule.Mode)
	}
	if rule.Group.ID != "grp2" {
		t.Errorf("expected Group.ID=grp2, got %s", rule.Group.ID)
	}
}

func TestWAFUpdateRuleValidation(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.UpdateRule(context.Background(), "", "rule1", "block")
	if err == nil {
		t.Error("expected error when packageID is empty")
	}

	_, err = svc.UpdateRule(context.Background(), "pkg1", "", "block")
	if err == nil {
		t.Error("expected error when ruleID is empty")
	}

	_, err = svc.UpdateRule(context.Background(), "pkg1", "rule1", "")
	if err == nil {
		t.Error("expected error when mode is empty")
	}
}

func TestWAFUpdateRuleSuccess(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id": "rule1", "description": "SQL Injection", "priority": "5",
				"package_id": "pkg1", "mode": "block", "default_mode": "block",
				"allowed_modes": []string{"block", "simulate", "disable"},
			},
		})
	})
	defer server.Close()

	rule, err := svc.UpdateRule(context.Background(), "pkg1", "rule1", "block")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.Mode != "block" {
		t.Errorf("expected Mode=block, got %s", rule.Mode)
	}
}

func TestWAFListAccessRulesSuccess(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{"id": "ar1", "mode": "block", "notes": "bad actor", "configuration": map[string]interface{}{"target": "ip", "value": "1.2.3.4"}},
				{"id": "ar2", "mode": "challenge", "notes": "suspicious", "configuration": map[string]interface{}{"target": "ip_range", "value": "10.0.0.0/24"}},
			},
			"result_info": map[string]interface{}{"page": 1, "per_page": 50, "total_count": 2, "count": 2, "total_pages": 1},
		})
	})
	defer server.Close()

	rules, err := svc.ListAccessRules(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].Configuration.Target != "ip" || rules[0].Configuration.Value != "1.2.3.4" {
		t.Errorf("first rule config mismatch: %+v", rules[0])
	}
}

func TestWAFCreateAccessRuleValidation(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	_, err := svc.CreateAccessRule(context.Background(), "", "1.2.3.4", "block", "")
	if err == nil {
		t.Error("expected error when target is empty")
	}

	_, err = svc.CreateAccessRule(context.Background(), "ip", "", "block", "")
	if err == nil {
		t.Error("expected error when value is empty")
	}

	_, err = svc.CreateAccessRule(context.Background(), "ip", "1.2.3.4", "", "")
	if err == nil {
		t.Error("expected error when mode is empty")
	}
}

func TestWAFCreateAccessRuleSuccess(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id": "ar-new", "mode": "block", "notes": "blocked ip",
				"configuration": map[string]interface{}{"target": "ip", "value": "5.6.7.8"},
			},
		})
	})
	defer server.Close()

	rule, err := svc.CreateAccessRule(context.Background(), "ip", "5.6.7.8", "block", "blocked ip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.ID != "ar-new" {
		t.Errorf("expected ID=ar-new, got %s", rule.ID)
	}
	if rule.Configuration.Value != "5.6.7.8" {
		t.Errorf("expected Value=5.6.7.8, got %s", rule.Configuration.Value)
	}
}

func TestWAFDeleteAccessRuleValidation(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	err := svc.DeleteAccessRule(context.Background(), "")
	if err == nil {
		t.Error("expected error when ruleID is empty")
	}
}

func TestWAFDeleteAccessRuleSuccess(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {
		wafWriteJSON(w, map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"id": "ar-del"},
		})
	})
	defer server.Close()

	err := svc.DeleteAccessRule(context.Background(), "ar-del")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWAFDeleteAccessRuleError(t *testing.T) {
	svc, server := wafMockSetup(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		wafWriteJSON(w, map[string]interface{}{
			"success": false,
			"errors":  []interface{}{map[string]interface{}{"message": "not found"}},
		})
	})
	defer server.Close()

	err := svc.DeleteAccessRule(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error on not found")
	}
}
