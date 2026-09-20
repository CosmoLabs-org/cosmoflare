package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// TestRegistrarService_List TestRegistrarService_List verifies that List
// issues a GET to the account's registrar domains endpoint and maps each
// response entry by domain name, decoding registrar name, transfer-lock
// state, and expiry into the returned info struct.
func TestRegistrarService_List(t *testing.T) {
	t.Parallel()
	const accountID = "account-test-123"
	// A clearly-future expiry so the assertion below is meaningful.
	futureExpiry := time.Now().UTC().Add(365 * 24 * time.Hour).Truncate(time.Second)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/"+accountID+"/registrar/domains"; got != want {
			t.Errorf("expected path %q, got %q", want, got)
		}
		w.Header().Set("Content-Type", "application/json")
		// Mirror the real Cloudflare Registrar list response shape. The domain
		// name is carried in the "id" field (see cloudflare-go registrar.go).
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": []map[string]interface{}{
				{
					"id":                "locked-example.com",
					"current_registrar": "Cloudflare",
					"expires_at":        futureExpiry.Format(time.RFC3339),
					"locked":            true,
				},
				{
					"id":                "open-example.com",
					"current_registrar": "Cloudflare",
					"expires_at":        futureExpiry.Format(time.RFC3339),
					"locked":            false,
				},
			},
			"result_info": map[string]interface{}{
				"page": 1, "per_page": 20, "count": 2, "total_count": 2,
			},
		})
	}))
	defer server.Close()

	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc := NewRegistrarService(cf, accountID)

	infos, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("expected 2 results, got %d", len(infos))
	}

	locked, ok := infos["locked-example.com"]
	if !ok {
		t.Fatalf("expected locked-example.com in results, got %+v", infos)
	}
	if !locked.TransferLock {
		t.Errorf("expected locked-example.com TransferLock=true, got false")
	}
	if locked.Registrar != "cloudflare" {
		t.Errorf("expected Registrar=cloudflare, got %q", locked.Registrar)
	}
	if locked.ExpiresAt == nil {
		t.Fatalf("expected locked-example.com ExpiresAt non-nil")
	}
	if !locked.ExpiresAt.After(time.Now()) {
		t.Errorf("expected ExpiresAt in the future, got %v", locked.ExpiresAt)
	}

	open, ok := infos["open-example.com"]
	if !ok {
		t.Fatalf("expected open-example.com in results, got %+v", infos)
	}
	if open.TransferLock {
		t.Errorf("expected open-example.com TransferLock=false, got true")
	}
	if open.Registrar != "cloudflare" {
		t.Errorf("expected Registrar=cloudflare, got %q", open.Registrar)
	}
	if open.ExpiresAt == nil {
		t.Fatalf("expected open-example.com ExpiresAt non-nil")
	}
	if !open.ExpiresAt.After(time.Now()) {
		t.Errorf("expected open-example.com ExpiresAt in the future, got %v", open.ExpiresAt)
	}
}

// TestRegistrarService_ListValidation TestRegistrarService_ListValidation
// verifies that List rejects an empty account ID instead of issuing a request
// with a malformed URL.
func TestRegistrarService_ListValidation(t *testing.T) {
	t.Parallel()
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc := NewRegistrarService(cf, "")
	_, err := svc.List(context.Background())
	if err == nil {
		t.Error("expected error when accountID is empty")
	}
}

// TestRegistrarService_ListEmpty TestRegistrarService_ListEmpty verifies that
// an empty result set from the API decodes to an empty collection without
// error.
func TestRegistrarService_ListEmpty(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"errors":      []interface{}{},
			"result":      []map[string]interface{}{},
			"result_info": map[string]interface{}{"page": 1, "per_page": 20, "count": 0, "total_count": 0},
		})
	}))
	defer server.Close()

	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc := NewRegistrarService(cf, "account-test-123")

	infos, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("expected 0 results, got %d", len(infos))
	}
}

// TestNewRegistrarServiceFromCreds TestNewRegistrarServiceFromCreds verifies
// constructor credential validation: an empty account ID or API token is
// rejected, while valid credentials yield a non-nil service.
func TestNewRegistrarServiceFromCreds(t *testing.T) {
	t.Parallel()
	if _, err := NewRegistrarServiceFromCreds("", "tok"); err == nil {
		t.Error("expected error for empty account ID")
	}
	if _, err := NewRegistrarServiceFromCreds("acct", ""); err == nil {
		t.Error("expected error for empty API token")
	}
	svc, err := NewRegistrarServiceFromCreds("acct", "tok")
	if err != nil || svc == nil {
		t.Fatalf("expected service, got svc=%v err=%v", svc, err)
	}
}

// ---------------------------------------------------------------------------
// FEAT-030 — operations depth tests.
// ---------------------------------------------------------------------------

// registrarStubServer spins up an httptest server for the Registrar API and
// records every request it served (method, path, decoded JSON body).
func registrarStubServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *cloudflare.API) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	cf, err := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	if err != nil {
		t.Fatalf("failed to build client: %v", err)
	}
	return server, cf
}

// registrarStubDomain builds a single-domain GET response body.
// registrarStubDomainArray writes the same domain wrapped in a result
// ARRAY — the shape TransferRegistrarDomain decodes.
func registrarStubDomainArray(w http.ResponseWriter, name string, locked bool) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"errors":  []interface{}{},
		"result": []map[string]interface{}{{
			"id": name, "available": false, "supported_tld": true,
			"can_register": false, "current_registrar": "Cloudflare",
			"expires_at": "2027-01-01T00:00:00Z", "locked": locked,
			"registry_statuses": "clientTransferProhibited",
		}},
	})
}

func registrarStubDomain(w http.ResponseWriter, name string, locked bool) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"errors":  []interface{}{},
		"result": map[string]interface{}{
			"id":                name,
			"available":         false,
			"supported_tld":     true,
			"can_register":      false,
			"current_registrar": "Cloudflare",
			"expires_at":        "2027-01-01T00:00:00Z",
			"locked":            locked,
			"registry_statuses": "clientTransferProhibited",
			"registrant_contact": map[string]interface{}{
				"first_name": "Ada",
				"last_name":  "Lovelace",
				"email":      "ada@example.com",
			},
		},
	})
}

// TestRegistrarService_Get verifies Get maps the SDK read model into the
// detail struct, including the registrant contact and nil-safe timestamps.
func TestRegistrarService_Get(t *testing.T) {
	t.Parallel()
	_, cf := registrarStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/acct-1/registrar/domains/example.com"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		registrarStubDomain(w, "example.com", true)
	})

	svc := NewRegistrarService(cf, "acct-1")
	detail, err := svc.Get(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Name != "example.com" {
		t.Errorf("Name = %q, want example.com", detail.Name)
	}
	if !detail.Locked {
		t.Error("expected Locked=true")
	}
	if detail.RegistryStatuses != "clientTransferProhibited" {
		t.Errorf("RegistryStatuses = %q", detail.RegistryStatuses)
	}
	if detail.Registrant == nil || detail.Registrant.Email != "ada@example.com" {
		t.Errorf("Registrant = %+v, want ada@example.com", detail.Registrant)
	}
	if detail.ExpiresAt == nil || detail.ExpiresAt.Year() != 2027 {
		t.Errorf("ExpiresAt = %v, want 2027", detail.ExpiresAt)
	}
}

// TestRegistrarService_GetValidation verifies Get rejects an empty domain
// or account before issuing a request.
func TestRegistrarService_GetValidation(t *testing.T) {
	t.Parallel()
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	if _, err := (NewRegistrarService(cf, "acct")).Get(context.Background(), ""); err == nil {
		t.Error("expected error for empty domain")
	}
	if _, err := (NewRegistrarService(cf, "")).Get(context.Background(), "example.com"); err == nil {
		t.Error("expected error for empty account ID")
	}
}

// TestRegistrarService_Register verifies Register POSTs the domain name to
// the collection endpoint and maps the response.
func TestRegistrarService_Register(t *testing.T) {
	t.Parallel()
	_, cf := registrarStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/acct-1/registrar/domains"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body["name"] != "newdomain.com" {
			t.Errorf("body name = %q, want newdomain.com", body["name"])
		}
		registrarStubDomain(w, "newdomain.com", false)
	})

	svc := NewRegistrarService(cf, "acct-1")
	detail, err := svc.Register(context.Background(), "newdomain.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Name != "newdomain.com" {
		t.Errorf("Name = %q, want newdomain.com", detail.Name)
	}
}

// TestRegistrarService_TransferPreflight_WarnsLocked verifies the
// unlock-before-transfer sequencing check fires on a locked domain and
// stays quiet for an unlocked one.
func TestRegistrarService_TransferPreflight_WarnsLocked(t *testing.T) {
	t.Parallel()
	_, cf := registrarStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		registrarStubDomain(w, "example.com", r.URL.Query().Get("locked") != "false")
	})

	svc := NewRegistrarService(cf, "acct-1")
	warnings, err := svc.TransferPreflight(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) == 0 {
		t.Fatal("expected unlock warning for locked domain, got none")
	}
	if !strings.Contains(warnings[0], "unlock") {
		t.Errorf("warning = %q, want it to mention unlock", warnings[0])
	}
}

// TestRegistrarService_TransferPreflight_UnlockedNoWarnings verifies a
// clean transfer-in checklist produces no warnings.
func TestRegistrarService_TransferPreflight_UnlockedNoWarnings(t *testing.T) {
	t.Parallel()
	_, cf := registrarStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"id": "example.com",
				"transfer_in": map[string]interface{}{
					"unlock_domain":   "complete",
					"disable_privacy": "complete",
				},
				"locked": false,
			},
		})
	})

	svc := NewRegistrarService(cf, "acct-1")
	warnings, err := svc.TransferPreflight(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings, got %v", warnings)
	}
}

// TestRegistrarService_TransferAndCancel verifies the typed SDK transfer
// and cancel-transfer endpoints are wired to the right paths.
func TestRegistrarService_TransferAndCancel(t *testing.T) {
	t.Parallel()
	_, cf := registrarStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		// TransferRegistrarDomain decodes result as an ARRAY of domains.
		registrarStubDomainArray(w, "example.com", false)
	})

	svc := NewRegistrarService(cf, "acct-1")
	detail, err := svc.Transfer(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if detail.Name != "example.com" {
		t.Errorf("Transfer Name = %q", detail.Name)
	}
	if _, err := svc.CancelTransfer(context.Background(), "example.com"); err != nil {
		t.Fatalf("CancelTransfer: %v", err)
	}
}

// TestRegistrarService_Renew verifies Renew POSTs to the renew endpoint
// and surfaces the API error verbatim when the window is closed.
func TestRegistrarService_Renew(t *testing.T) {
	t.Parallel()
	_, cf := registrarStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/accounts/acct-1/registrar/domains/example.com/renew"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		registrarStubDomain(w, "example.com", false)
	})

	svc := NewRegistrarService(cf, "acct-1")
	if _, err := svc.Renew(context.Background(), "example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestRegistrarService_RenewWindowError verifies renewal-window API errors
// pass through the service with their message intact.
func TestRegistrarService_RenewWindowError(t *testing.T) {
	t.Parallel()
	_, cf := registrarStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"errors": []map[string]interface{}{
				{"code": 1001, "message": "domain is not within the renewal window"},
			},
		})
	})

	svc := NewRegistrarService(cf, "acct-1")
	_, err := svc.Renew(context.Background(), "example.com")
	if err == nil {
		t.Fatal("expected error for closed renewal window")
	}
	if !strings.Contains(err.Error(), "renewal window") {
		t.Errorf("error = %q, want it to carry the API message verbatim", err)
	}
}

// TestRegistrarService_SetAutoRenewAndLock verifies the scoped PATCHes for
// auto-renew and transfer lock hit the domain resource with the right body.
func TestRegistrarService_SetAutoRenewAndLock(t *testing.T) {
	t.Parallel()
	_, cf := registrarStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if len(body) != 1 {
			t.Errorf("expected scoped single-key PATCH body, got %v", body)
		}
		registrarStubDomain(w, "example.com", body["locked"] == true)
	})

	svc := NewRegistrarService(cf, "acct-1")
	if _, err := svc.SetAutoRenew(context.Background(), "example.com", true); err != nil {
		t.Fatalf("SetAutoRenew: %v", err)
	}
	if _, err := svc.SetLock(context.Background(), "example.com", true); err != nil {
		t.Fatalf("SetLock: %v", err)
	}
}

// TestRegistrarService_ContactsUpdate verifies UpdateContacts PUTs the
// contact roles and decodes the updated set.
func TestRegistrarService_ContactsUpdate(t *testing.T) {
	t.Parallel()
	_, cf := registrarStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if got, want := r.URL.Path, "/accounts/acct-1/registrar/domains/example.com/contacts"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		var body map[string]map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if body["registrant"]["email"] != "ada@example.com" {
			t.Errorf("registrant email = %q", body["registrant"]["email"])
		}
		if _, hasBilling := body["billing"]; hasBilling {
			t.Error("expected nil roles omitted from payload")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"registrant": map[string]interface{}{"email": "ada@example.com"},
			},
		})
	})

	svc := NewRegistrarService(cf, "acct-1")
	updated, err := svc.UpdateContacts(context.Background(), "example.com", RegistrarContacts{
		Registrant: &RegistrarContact{FirstName: "Ada", Email: "ada@example.com"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Registrant == nil || updated.Registrant.Email != "ada@example.com" {
		t.Errorf("updated registrant = %+v", updated.Registrant)
	}
}

// TestRegistrarService_ContactsUpdateValidation verifies UpdateContacts
// rejects an all-nil contact set.
func TestRegistrarService_ContactsUpdateValidation(t *testing.T) {
	t.Parallel()
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	svc := NewRegistrarService(cf, "acct-1")
	if _, err := svc.UpdateContacts(context.Background(), "example.com", RegistrarContacts{}); err == nil {
		t.Error("expected error for empty contacts")
	}
}

// TestRegistrarService_DNSSEC verifies Get/Enable/Disable hit the dnssec
// endpoint with the expected status payloads and map DS records.
func TestRegistrarService_DNSSEC(t *testing.T) {
	t.Parallel()
	var lastStatus string
	_, cf := registrarStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/accounts/acct-1/registrar/domains/example.com/dnssec"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		var body map[string]string
		if r.Method == http.MethodPut {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode body: %v", err)
			}
			lastStatus = body["status"]
		}
		status := "disabled"
		var ds []interface{}
		if r.Method == http.MethodGet && lastStatus == "" || lastStatus == "active" {
			status = "active"
			ds = []interface{}{
				map[string]interface{}{"key_tag": 1234, "algorithm": 13, "digest_type": 2, "digest": "abcdef0123456789"},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{"status": status, "ds_records": ds},
		})
	})

	svc := NewRegistrarService(cf, "acct-1")
	current, err := svc.GetDNSSEC(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("GetDNSSEC: %v", err)
	}
	if current.Status != "active" || len(current.DSRecords) != 1 {
		t.Fatalf("GetDNSSEC = %+v, want active with 1 DS record", current)
	}
	if ds := current.DSRecords[0]; ds.KeyTag != 1234 || ds.Digest != "abcdef0123456789" {
		t.Errorf("DS record = %+v", ds)
	}
	if _, err := svc.DisableDNSSEC(context.Background(), "example.com"); err != nil {
		t.Fatalf("DisableDNSSEC: %v", err)
	}
	if lastStatus != "disabled" {
		t.Errorf("disable PUT status = %q, want disabled", lastStatus)
	}
}

// TestRegistrarService_DisableDNSSECPreflight_WarnsDSRecords verifies the
// DS-before-disable sequencing check warns while DS records are published
// at the parent and stays quiet once they are gone.
func TestRegistrarService_DisableDNSSECPreflight_WarnsDSRecords(t *testing.T) {
	t.Parallel()
	dsPublished := true
	_, cf := registrarStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		var ds []interface{}
		status := "disabled"
		if dsPublished {
			status = "active"
			ds = []interface{}{map[string]interface{}{"key_tag": 1, "algorithm": 13, "digest_type": 2, "digest": "aa"}}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result":  map[string]interface{}{"status": status, "ds_records": ds},
		})
	})

	svc := NewRegistrarService(cf, "acct-1")
	warnings, err := svc.DisableDNSSECPreflight(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) == 0 {
		t.Fatal("expected DS-record warning, got none")
	}
	if !strings.Contains(warnings[0], "DS record") {
		t.Errorf("warning = %q, want DS record mention", warnings[0])
	}

	dsPublished = false
	warnings, err = svc.DisableDNSSECPreflight(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error after DS removal: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings once DS records are gone, got %v", warnings)
	}
}

// TestRegistrarService_EnableDNSSECReturnsDS verifies enabling surfaces the
// DS records that must be added at the parent zone.
func TestRegistrarService_EnableDNSSECReturnsDS(t *testing.T) {
	t.Parallel()
	_, cf := registrarStubServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"errors":  []interface{}{},
			"result": map[string]interface{}{
				"status": "pending",
				"ds_records": []interface{}{
					map[string]interface{}{"key_tag": 9999, "algorithm": 8, "digest_type": 2, "digest": "feedbeef"},
				},
			},
		})
	})

	svc := NewRegistrarService(cf, "acct-1")
	result, err := svc.EnableDNSSEC(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "pending" {
		t.Errorf("Status = %q, want pending", result.Status)
	}
	if len(result.DSRecords) != 1 || result.DSRecords[0].KeyTag != 9999 {
		t.Errorf("DSRecords = %+v, want key_tag 9999", result.DSRecords)
	}
}
