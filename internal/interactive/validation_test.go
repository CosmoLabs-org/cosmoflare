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

// stubCFBaseURL points the package-level Cloudflare API base URL at url for
// the duration of the test, restoring the previous value via t.Cleanup.
//
// cfAPIBaseURL is shared package state, so tests using this helper must NOT
// call t.Parallel().
func stubCFBaseURL(t *testing.T, url string) {
	t.Helper()
	orig := cfAPIBaseURL
	cfAPIBaseURL = url
	t.Cleanup(func() { cfAPIBaseURL = orig })
}

// startCFTestServer spins up an httptest server backed by handler and points
// the package-level Cloudflare API base URL at it for the duration of the
// test. pathSuffix is appended to the server URL (usually "/client/v4").
// The server is closed and the base URL is restored via t.Cleanup.
func startCFTestServer(t *testing.T, pathSuffix string, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	stubCFBaseURL(t, server.URL+pathSuffix)
	return server
}

// TestValidateAPIToken_WithHTTPTest verifies ValidateAPIToken behavior, one t.Run subtest per...
func TestValidateAPIToken_WithHTTPTest(t *testing.T) {
	t.Run("valid token with R2 permissions", testValidateAPITokenValidR2)
	t.Run("invalid token returns error", testValidateAPITokenInvalid)
	t.Run("token without R2 permissions", testValidateAPITokenNoR2)
	t.Run("short token rejected without HTTP call", testValidateAPITokenShort)
}

// testValidateAPITokenValidR2 asserts that a verified token carrying R2 permission groups is reported as a valid api_token.
func testValidateAPITokenValidR2(t *testing.T) {
	server := startCFTestServer(t, "/client/v4", func(w http.ResponseWriter, r *http.Request) {
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
	})

	info, err := ValidateAPIToken("test-token-valid-12345678")
	require.NoError(t, err)
	assert.True(t, info.Valid)
	assert.Equal(t, "api_token", info.TokenType)
}

// testValidateAPITokenInvalid asserts that an HTTP 401 from the verify endpoint yields an invalid token and an error mentioning the status code.
func testValidateAPITokenInvalid(t *testing.T) {
	server := startCFTestServer(t, "/client/v4", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": "Invalid API Token"}},
		})
	})

	info, err := ValidateAPIToken("invalid-token-but-long-enough")
	require.Error(t, err)
	assert.False(t, info.Valid)
	assert.Contains(t, err.Error(), "HTTP 401")
}

// testValidateAPITokenNoR2 asserts that a verified token lacking R2 permissions is rejected with an error mentioning R2 permissions.
func testValidateAPITokenNoR2(t *testing.T) {
	server := startCFTestServer(t, "/client/v4", func(w http.ResponseWriter, r *http.Request) {
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
	})

	info, err := ValidateAPIToken("valid-token-no-r2-permissions")
	require.Error(t, err)
	assert.False(t, info.Valid)
	assert.Contains(t, info.Error, "R2 permissions")
}

// testValidateAPITokenShort asserts that a too-short token is rejected locally without any HTTP call.
func testValidateAPITokenShort(t *testing.T) {
	info, err := ValidateAPIToken("short")
	require.Error(t, err)
	assert.Contains(t, info.Error, "too short")
}

// TestGetAccountName_WithHTTPTest verifies getAccountName behavior, one t.Run subtest per scenario...
func TestGetAccountName_WithHTTPTest(t *testing.T) {
	t.Run("returns account name on success", func(t *testing.T) {
		server := startCFTestServer(t, "", func(w http.ResponseWriter, r *http.Request) {
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

// TestTestConnection_WithHTTPTest verifies TestConnection behavior, one t.Run subtest per scenario...
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

// TestAutoDetectAccountInfo_ValidTokenWithAccountID verifies that autoDetectAccountInfo handles...
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
	})

	token := strings.Repeat("a", 25)
	accountID, accountName := autoDetectAccountInfo(token)
	assert.Equal(t, "acc-0123456789abcdef0123456789abcd", accountID)
	assert.Equal(t, "Test Account", accountName)
}

// TestAutoDetectAccountInfo_ValidTokenNoAccountID verifies that autoDetectAccountInfo handles the...
func TestAutoDetectAccountInfo_ValidTokenNoAccountID(t *testing.T) {
	server := startCFTestServer(t, "", func(w http.ResponseWriter, r *http.Request) {
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
	})

	accountID, accountName := autoDetectAccountInfo(strings.Repeat("b", 25))
	assert.Equal(t, "", accountID)
	assert.Equal(t, "", accountName)
}

// TestAutoDetectAccountInfo_InvalidTokenResponse verifies that autoDetectAccountInfo handles the...
func TestAutoDetectAccountInfo_InvalidTokenResponse(t *testing.T) {
	server := startCFTestServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"success": false}`))
	})

	accountID, accountName := autoDetectAccountInfo(strings.Repeat("c", 25))
	assert.Equal(t, "", accountID)
	assert.Equal(t, "", accountName)
}

// TestGetAccountName_NetworkError verifies that getAccountName degrades gracefully when the API is...
func TestGetAccountName_NetworkError(t *testing.T) {
	stubCFBaseURL(t, "http://127.0.0.1:1") // unreachable

	name := getAccountName("token", "account-id")
	assert.Equal(t, "", name)
}

// TestGetAccountName_NonOKStatus verifies that getAccountName degrades gracefully on a non-200...
func TestGetAccountName_NonOKStatus(t *testing.T) {
	server := startCFTestServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})

	name := getAccountName("token", "account-id")
	assert.Equal(t, "", name)
}

// TestGetAccountName_InvalidJSON verifies that getAccountName degrades gracefully on malformed JSON.
func TestGetAccountName_InvalidJSON(t *testing.T) {
	server := startCFTestServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`invalid json`))
	})

	name := getAccountName("token", "account-id")
	assert.Equal(t, "", name)
}

// TestGetAccountName_SuccessFalse verifies that getAccountName degrades gracefully when...
func TestGetAccountName_SuccessFalse(t *testing.T) {
	server := startCFTestServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"result":  map[string]interface{}{},
		})
	})

	name := getAccountName("token", "account-id")
	assert.Equal(t, "", name)
}

// ---------------------------------------------------------------------------
// ValidateAccountID additional cases
// ---------------------------------------------------------------------------

// TestValidateAccountID_Valid verifies that ValidateAccountID accepts valid input.
func TestValidateAccountID_Valid(t *testing.T) {
	err := ValidateAccountID("abcdef0123456789abcdef01234567ab")
	assert.NoError(t, err)
}

// TestValidateAccountID_UpperCase verifies that ValidateAccountID accepts and normalizes uppercase...
func TestValidateAccountID_UpperCase(t *testing.T) {
	err := ValidateAccountID("ABCDEF0123456789ABCDEF01234567AB")
	assert.NoError(t, err)
}

// TestValidateAccountID_InvalidChar verifies that ValidateAccountID rejects non-hexadecimal...
func TestValidateAccountID_InvalidChar(t *testing.T) {
	err := ValidateAccountID("abcdef0123456789abcdef01234567xz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hex")
}

// ---------------------------------------------------------------------------
// TestConnection additional cases
// ---------------------------------------------------------------------------

// TestTestConnection_NotFound verifies that TestConnection handles the not found case.
func TestTestConnection_NotFound(t *testing.T) {
	server := startCFTestServer(t, "", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	err := TestConnection("account-id", "token")
	assert.NoError(t, err)
}

// TestTestConnection_NetworkError verifies that TestConnection degrades gracefully when the API is...
func TestTestConnection_NetworkError(t *testing.T) {
	stubCFBaseURL(t, "http://127.0.0.1:1")

	err := TestConnection("account-id", "token")
	assert.Error(t, err)
}
