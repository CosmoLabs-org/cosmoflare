package interactive

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAPIToken_WithHTTPTest(t *testing.T) {
	t.Run("valid token with R2 permissions", testValidateAPITokenValidR2)
	t.Run("invalid token returns error", testValidateAPITokenInvalid)
	t.Run("token without R2 permissions", testValidateAPITokenNoR2)
	t.Run("short token rejected without HTTP call", testValidateAPITokenShort)
}

func testValidateAPITokenValidR2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token-valid-12345678", r.Header.Get("Authorization"))
		assert.Equal(t, "/client/v4/user/tokens/verify", r.URL.Path)

		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result": map[string]any{
				"id":     "token-id-123",
				"status": "active",
				"policy": map[string]any{
					"permission_groups": []map[string]any{
						{"id": "pg1", "name": "R2 Write", "permissions": []string{"r2:write", "r2:read"}},
					},
					"resources": map[string]any{
						"computation:cloudflare_account:account_id": []string{"abc123def456"},
					},
				},
			},
		})
	}))
	defer server.Close()

	origURL := cfAPIBaseURL
	cfAPIBaseURL = server.URL + "/client/v4"
	defer func() { cfAPIBaseURL = origURL }()

	info, err := ValidateAPIToken("test-token-valid-12345678")
	require.NoError(t, err)
	assert.True(t, info.Valid)
	assert.Equal(t, "api_token", info.TokenType)
}

func testValidateAPITokenInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": "Invalid API Token"}},
		})
	}))
	defer server.Close()

	origURL := cfAPIBaseURL
	cfAPIBaseURL = server.URL + "/client/v4"
	defer func() { cfAPIBaseURL = origURL }()

	info, err := ValidateAPIToken("invalid-token-but-long-enough")
	require.Error(t, err)
	assert.False(t, info.Valid)
	assert.Contains(t, err.Error(), "HTTP 401")
}

func testValidateAPITokenNoR2(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result": map[string]any{
				"id":     "token-id-456",
				"status": "active",
				"policy": map[string]any{
					"permission_groups": []map[string]any{
						{"id": "pg1", "name": "DNS Write", "permissions": []string{"dns:write"}},
					},
				},
			},
		})
	}))
	defer server.Close()

	origURL := cfAPIBaseURL
	cfAPIBaseURL = server.URL + "/client/v4"
	defer func() { cfAPIBaseURL = origURL }()

	info, err := ValidateAPIToken("valid-token-no-r2-permissions")
	require.Error(t, err)
	assert.False(t, info.Valid)
	assert.Contains(t, info.Error, "R2 permissions")
}

func testValidateAPITokenShort(t *testing.T) {
	info, err := ValidateAPIToken("short")
	require.Error(t, err)
	assert.Contains(t, info.Error, "too short")
}

func TestGetAccountName_WithHTTPTest(t *testing.T) {
	t.Run("returns account name on success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/client/v4/accounts/acc123", r.URL.Path)
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"result":  map[string]any{"id": "acc123", "name": "My Account"},
			})
		}))
		defer server.Close()

		origURL := cfAPIBaseURL
		cfAPIBaseURL = server.URL + "/client/v4"
		defer func() { cfAPIBaseURL = origURL }()

		name := getAccountName("token123", "acc123")
		assert.Equal(t, "My Account", name)
	})

	t.Run("returns empty on HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		origURL := cfAPIBaseURL
		cfAPIBaseURL = server.URL + "/client/v4"
		defer func() { cfAPIBaseURL = origURL }()

		name := getAccountName("bad-token", "acc123")
		assert.Equal(t, "", name)
	})
}

func TestTestConnection_WithHTTPTest(t *testing.T) {
	t.Run("succeeds on 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.Path, "/r2/buckets")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		origURL := cfAPIBaseURL
		cfAPIBaseURL = server.URL + "/client/v4"
		defer func() { cfAPIBaseURL = origURL }()

		err := TestConnection("abcdef0123456789abcdef0123456789", "valid-token")
		assert.NoError(t, err)
	})

	t.Run("succeeds on 404 (no buckets)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		origURL := cfAPIBaseURL
		cfAPIBaseURL = server.URL + "/client/v4"
		defer func() { cfAPIBaseURL = origURL }()

		err := TestConnection("abcdef0123456789abcdef0123456789", "valid-token")
		assert.NoError(t, err)
	})

	t.Run("fails on 403", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		origURL := cfAPIBaseURL
		cfAPIBaseURL = server.URL + "/client/v4"
		defer func() { cfAPIBaseURL = origURL }()

		err := TestConnection("abcdef0123456789abcdef0123456789", "bad-token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP 403")
	})
}

// ---------------------------------------------------------------------------
// autoDetectAccountInfo with httptest (42.9% coverage gap)
// ---------------------------------------------------------------------------

func TestAutoDetectAccountInfo_ValidTokenWithAccountID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "verify") {
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{
					"id":     "token-id",
					"status": "active",
					"policy": map[string]interface{}{
						"permission_groups": []map[string]interface{}{
							{"permissions": []string{"r2:read", "r2:write"}},
						},
						"resources": map[string]interface{}{
							"computation:cloudflare_account:account_id": []string{"acc-0123456789abcdef0123456789abcd"},
						},
					},
				},
			})
		} else if strings.Contains(r.URL.Path, "accounts/") {
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{
					"id":   "acc-0123456789abcdef0123456789abcd",
					"name": "Test Account",
				},
			})
		} else {
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	origURL := cfAPIBaseURL
	cfAPIBaseURL = server.URL
	defer func() { cfAPIBaseURL = origURL }()

	token := strings.Repeat("a", 25)
	accountID, accountName := autoDetectAccountInfo(token)
	assert.Equal(t, "acc-0123456789abcdef0123456789abcd", accountID)
	assert.Equal(t, "Test Account", accountName)
}

func TestAutoDetectAccountInfo_ValidTokenNoAccountID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"result": map[string]interface{}{
				"id":     "token-id",
				"status": "active",
				"policy": map[string]interface{}{
					"permission_groups": []map[string]interface{}{
						{"permissions": []string{"r2:read"}},
					},
					"resources": map[string]interface{}{},
				},
			},
		})
	}))
	defer server.Close()

	origURL := cfAPIBaseURL
	cfAPIBaseURL = server.URL
	defer func() { cfAPIBaseURL = origURL }()

	accountID, accountName := autoDetectAccountInfo(strings.Repeat("b", 25))
	assert.Equal(t, "", accountID)
	assert.Equal(t, "", accountName)
}

func TestAutoDetectAccountInfo_InvalidTokenResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"success": false}`))
	}))
	defer server.Close()

	origURL := cfAPIBaseURL
	cfAPIBaseURL = server.URL
	defer func() { cfAPIBaseURL = origURL }()

	accountID, accountName := autoDetectAccountInfo(strings.Repeat("c", 25))
	assert.Equal(t, "", accountID)
	assert.Equal(t, "", accountName)
}

func TestGetAccountName_NetworkError(t *testing.T) {
	origURL := cfAPIBaseURL
	cfAPIBaseURL = "http://127.0.0.1:1" // unreachable
	defer func() { cfAPIBaseURL = origURL }()

	name := getAccountName("token", "account-id")
	assert.Equal(t, "", name)
}

func TestGetAccountName_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	origURL := cfAPIBaseURL
	cfAPIBaseURL = server.URL
	defer func() { cfAPIBaseURL = origURL }()

	name := getAccountName("token", "account-id")
	assert.Equal(t, "", name)
}

func TestGetAccountName_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	origURL := cfAPIBaseURL
	cfAPIBaseURL = server.URL
	defer func() { cfAPIBaseURL = origURL }()

	name := getAccountName("token", "account-id")
	assert.Equal(t, "", name)
}

func TestGetAccountName_SuccessFalse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"result":  map[string]interface{}{},
		})
	}))
	defer server.Close()

	origURL := cfAPIBaseURL
	cfAPIBaseURL = server.URL
	defer func() { cfAPIBaseURL = origURL }()

	name := getAccountName("token", "account-id")
	assert.Equal(t, "", name)
}

// ---------------------------------------------------------------------------
// ValidateAccountID additional cases
// ---------------------------------------------------------------------------

func TestValidateAccountID_Valid(t *testing.T) {
	err := ValidateAccountID("abcdef0123456789abcdef01234567ab")
	assert.NoError(t, err)
}

func TestValidateAccountID_UpperCase(t *testing.T) {
	err := ValidateAccountID("ABCDEF0123456789ABCDEF01234567AB")
	assert.NoError(t, err)
}

func TestValidateAccountID_InvalidChar(t *testing.T) {
	err := ValidateAccountID("abcdef0123456789abcdef01234567xz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hex")
}

// ---------------------------------------------------------------------------
// TestConnection additional cases
// ---------------------------------------------------------------------------

func TestTestConnection_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	origURL := cfAPIBaseURL
	cfAPIBaseURL = server.URL
	defer func() { cfAPIBaseURL = origURL }()

	err := TestConnection("account-id", "token")
	assert.NoError(t, err)
}

func TestTestConnection_NetworkError(t *testing.T) {
	origURL := cfAPIBaseURL
	cfAPIBaseURL = "http://127.0.0.1:1"
	defer func() { cfAPIBaseURL = origURL }()

	err := TestConnection("account-id", "token")
	assert.Error(t, err)
}
