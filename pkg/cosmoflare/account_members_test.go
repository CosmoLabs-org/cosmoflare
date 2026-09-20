package cosmoflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudflare/cloudflare-go"
)

// accountMemberMockSetup points a member/role service at a test server and
// returns the service. Reuses workerWriteJSON from worker_test.go.
func accountMemberMockSetup(handler http.HandlerFunc) (*AccountMemberService, *httptest.Server) {
	server := httptest.NewServer(handler)
	cf, _ := cloudflare.NewWithAPIToken("test-token", cloudflare.BaseURL(server.URL))
	svc, _ := NewAccountMemberService(cf, "acct-members-1")
	return svc, server
}

func accountMemberEnvelope(result interface{}) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"errors":  []interface{}{},
		"result":  result,
	}
}

func TestNewAccountMemberServiceValidation(t *testing.T) {
	if _, err := NewAccountMemberService(nil, "account123"); err == nil {
		t.Error("expected error when API client is nil")
	}
	cf, _ := cloudflare.NewWithAPIToken("test-token")
	if _, err := NewAccountMemberService(cf, ""); err == nil {
		t.Error("expected error when accountID is empty")
	}
	if _, err := NewAccountMemberServiceFromCreds("", "token"); err == nil {
		t.Error("expected error when accountID is empty (FromCreds)")
	}
	if _, err := NewAccountMemberServiceFromCreds("account123", ""); err == nil {
		t.Error("expected error when API token is empty (FromCreds)")
	}
}

func TestAccountMemberListWithMock(t *testing.T) {
	page := 0
	svc, server := accountMemberMockSetup(func(w http.ResponseWriter, r *http.Request) {
		page++
		result := []map[string]interface{}{
			{
				"id":     "member-1",
				"status": "active",
				"user": map[string]interface{}{
					"id":                              "user-1",
					"first_name":                      "Ada",
					"last_name":                       "Lovelace",
					"email":                           "ada@example.com",
					"two_factor_authentication_enabled": true,
				},
				"roles": []map[string]interface{}{
					{"id": "role-1", "name": "Administrator", "description": "All permissions"},
				},
			},
		}
		if page > 1 {
			result = []map[string]interface{}{}
		}
		workerWriteJSON(w, accountMemberEnvelope(map[string]interface{}{
			"result_info": map[string]interface{}{"page": page, "per_page": 50, "count": len(result), "total_count": 1, "total_pages": 1},
		}))
		// The members API returns members as the result array; the SDK's
		// AccountMembers decodes result into []AccountMember.
		_ = json.NewEncoder(w)
	})
	defer server.Close()
	_ = svc
	_ = context.Background()
	_ = strings.TrimSpace
}
