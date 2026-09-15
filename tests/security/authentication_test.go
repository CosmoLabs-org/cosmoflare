/*
Package security provides comprehensive security testing for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/CosmoLabs-org/cosmoflare/tests/helpers"
)

// AuthenticationTestSuite provides comprehensive authentication testing
type AuthenticationTestSuite struct {
	suite.Suite
	server     *httptest.Server
	testConfig helpers.TestConfig
}

// SetupSuite sets up the authentication test suite
func (suite *AuthenticationTestSuite) SetupSuite() {
	// Create test configuration
	suite.testConfig = *helpers.SetupTest(suite.T())

	// Create test server that simulates authentication scenarios
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		suite.handleAuthRequest(w, r)
	}))
}

// TearDownSuite cleans up after authentication tests
func (suite *AuthenticationTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
}

// handleAuthRequest simulates various authentication scenarios
func (suite *AuthenticationTestSuite) handleAuthRequest(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	path := r.URL.Path

	switch path {
	case "/auth/validate":
		suite.handleTokenValidation(w, authHeader)
	case "/auth/refresh":
		suite.handleTokenRefresh(w, authHeader)
	case "/auth/revoke":
		suite.handleTokenRevocation(w, authHeader)
	case "/auth/permissions":
		suite.handlePermissionsCheck(w, authHeader)
	case "/auth/expired":
		suite.handleExpiredToken(w, authHeader)
	case "/auth/malformed":
		suite.handleMalformedToken(w, authHeader)
	case "/auth/weak":
		suite.handleWeakToken(w, authHeader)
	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error": "endpoint not found"}`)
	}
}

// handleTokenValidation validates authentication tokens
func (suite *AuthenticationTestSuite) handleTokenValidation(w http.ResponseWriter, authHeader string) {
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "missing authorization header"}`)
		return
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "invalid authorization format"}`)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	// Validate token format and content
	switch {
	case token == "valid-token-12345":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"valid": true,
			"account_id": "account-001",
			"expires_at": "2025-12-31T23:59:59Z",
			"permissions": ["read", "write", "delete"]
		}`)

	case token == "expired-token-12345":
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "token has expired"}`)

	case token == "revoked-token-12345":
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "token has been revoked"}`)

	case len(token) < 20:
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "token too short"}`)

	case strings.Contains(token, " "):
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "token contains invalid characters"}`)

	default:
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "invalid token"}`)
	}
}

// handleTokenRefresh handles token refresh operations
func (suite *AuthenticationTestSuite) handleTokenRefresh(w http.ResponseWriter, authHeader string) {
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "missing authorization header"}`)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	switch token {
	case "refreshable-token-12345":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"new_token": "new-refreshed-token-67890",
			"expires_at": "2025-12-31T23:59:59Z",
			"refresh_required": false
		}`)

	case "non-refreshable-token-12345":
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "token cannot be refreshed"}`)

	default:
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "invalid refresh token"}`)
	}
}

// handleTokenRevocation handles token revocation
func (suite *AuthenticationTestSuite) handleTokenRevocation(w http.ResponseWriter, authHeader string) {
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "missing authorization header"}`)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	switch token {
	case "revocable-token-12345":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"revoked": true,
			"revoked_at": "2025-01-15T10:30:00Z",
			"message": "token successfully revoked"
		}`)

	case "already-revoked-token-12345":
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error": "token already revoked"}`)

	default:
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "invalid token for revocation"}`)
	}
}

// handlePermissionsCheck checks token permissions
func (suite *AuthenticationTestSuite) handlePermissionsCheck(w http.ResponseWriter, authHeader string) {
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "missing authorization header"}`)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")

	switch token {
	case "admin-token-12345":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"permissions": ["read", "write", "delete", "admin", "manage_accounts"],
			"scope": "global",
			"role": "administrator"
		}`)

	case "read-only-token-12345":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"permissions": ["read"],
			"scope": "bucket:read-only-bucket",
			"role": "reader"
		}`)

	case "limited-token-12345":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"permissions": ["read", "write"],
			"scope": "bucket:limited-bucket",
			"role": "contributor"
		}`)

	default:
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "invalid token"}`)
	}
}

// handleExpiredToken handles expired token scenarios
func (suite *AuthenticationTestSuite) handleExpiredToken(w http.ResponseWriter, authHeader string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="Cloudflare R2", error="invalid_token", error_description="The access token expired"`)
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprint(w, `{"error": "token expired", "error_code": "TOKEN_EXPIRED"}`)
}

// handleMalformedToken handles malformed token scenarios
func (suite *AuthenticationTestSuite) handleMalformedToken(w http.ResponseWriter, authHeader string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="Cloudflare R2", error="invalid_token", error_description="The access token is invalid"`)
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprint(w, `{"error": "invalid token format", "error_code": "INVALID_TOKEN"}`)
}

// handleWeakToken handles weak token scenarios
func (suite *AuthenticationTestSuite) handleWeakToken(w http.ResponseWriter, authHeader string) {
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprint(w, `{"error": "weak token detected", "error_code": "WEAK_TOKEN", "suggestion": "use a stronger authentication method"}`)
}

// assertValidateStatus performs a GET /auth/validate request with the given
// Authorization header (omitted when empty) and asserts the response status.
func (suite *AuthenticationTestSuite) assertValidateStatus(authHeader string, wantStatus int, msg string) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", suite.server.URL+"/auth/validate", nil)
	require.NoError(suite.T(), err)

	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := client.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), wantStatus, resp.StatusCode, msg)
}

// TestTokenValidation tests various token validation scenarios
func (suite *AuthenticationTestSuite) TestTokenValidation() {
	suite.Run("Valid Token", func() {
		suite.assertValidateStatus("Bearer valid-token-12345", http.StatusOK, "Valid token should be accepted")
	})

	suite.Run("Missing Authorization Header", func() {
		suite.assertValidateStatus("", http.StatusUnauthorized, "Missing auth header should be rejected")
	})

	suite.Run("Invalid Authorization Format", func() {
		// Basic auth instead of Bearer
		suite.assertValidateStatus("Basic dGVzdDp0ZXN0", http.StatusUnauthorized, "Invalid auth format should be rejected")
	})

	suite.Run("Expired Token", func() {
		suite.assertValidateStatus("Bearer expired-token-12345", http.StatusUnauthorized, "Expired token should be rejected")
	})

	suite.Run("Revoked Token", func() {
		suite.assertValidateStatus("Bearer revoked-token-12345", http.StatusUnauthorized, "Revoked token should be rejected")
	})

	suite.Run("Short Token", func() {
		suite.assertValidateStatus("Bearer short", http.StatusUnauthorized, "Short token should be rejected")
	})

	suite.Run("Token with Spaces", func() {
		suite.assertValidateStatus("Bearer token with spaces", http.StatusUnauthorized, "Token with spaces should be rejected")
	})
}

// TestTokenRefresh tests token refresh functionality
func (suite *AuthenticationTestSuite) TestTokenRefresh() {
	suite.Run("Successful Token Refresh", func() {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("POST", suite.server.URL+"/auth/refresh", nil)
		require.NoError(suite.T(), err)

		req.Header.Set("Authorization", "Bearer refreshable-token-12345")

		resp, err := client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Refreshable token should be refreshed")
	})

	suite.Run("Non-Refreshable Token", func() {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("POST", suite.server.URL+"/auth/refresh", nil)
		require.NoError(suite.T(), err)

		req.Header.Set("Authorization", "Bearer non-refreshable-token-12345")

		resp, err := client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode, "Non-refreshable token should be rejected")
	})

	suite.Run("Invalid Refresh Token", func() {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("POST", suite.server.URL+"/auth/refresh", nil)
		require.NoError(suite.T(), err)

		req.Header.Set("Authorization", "Bearer invalid-token")

		resp, err := client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode, "Invalid refresh token should be rejected")
	})
}

// TestTokenRevocation tests token revocation functionality
func (suite *AuthenticationTestSuite) TestTokenRevocation() {
	suite.Run("Successful Token Revocation", func() {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("POST", suite.server.URL+"/auth/revoke", nil)
		require.NoError(suite.T(), err)

		req.Header.Set("Authorization", "Bearer revocable-token-12345")

		resp, err := client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Revocable token should be revoked")
	})

	suite.Run("Already Revoked Token", func() {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("POST", suite.server.URL+"/auth/revoke", nil)
		require.NoError(suite.T(), err)

		req.Header.Set("Authorization", "Bearer already-revoked-token-12345")

		resp, err := client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusBadRequest, resp.StatusCode, "Already revoked token should return bad request")
	})

	suite.Run("Invalid Token for Revocation", func() {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("POST", suite.server.URL+"/auth/revoke", nil)
		require.NoError(suite.T(), err)

		req.Header.Set("Authorization", "Bearer invalid-token")

		resp, err := client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode, "Invalid token should be rejected")
	})
}

// TestPermissionChecks tests permission validation
func (suite *AuthenticationTestSuite) TestPermissionChecks() {
	suite.Run("Admin Token Permissions", func() {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("GET", suite.server.URL+"/auth/permissions", nil)
		require.NoError(suite.T(), err)

		req.Header.Set("Authorization", "Bearer admin-token-12345")

		resp, err := client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Admin token should have permissions")
	})

	suite.Run("Read-Only Token Permissions", func() {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("GET", suite.server.URL+"/auth/permissions", nil)
		require.NoError(suite.T(), err)

		req.Header.Set("Authorization", "Bearer read-only-token-12345")

		resp, err := client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Read-only token should have limited permissions")
	})

	suite.Run("Limited Token Permissions", func() {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("GET", suite.server.URL+"/auth/permissions", nil)
		require.NoError(suite.T(), err)

		req.Header.Set("Authorization", "Bearer limited-token-12345")

		resp, err := client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusOK, resp.StatusCode, "Limited token should have specific permissions")
	})
}

// TestTokenSecurity tests token security properties
func (suite *AuthenticationTestSuite) TestTokenSecurity() {
	suite.Run("Expired Token Response Headers", func() {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("GET", suite.server.URL+"/auth/expired", nil)
		require.NoError(suite.T(), err)

		req.Header.Set("Authorization", "Bearer expired-token-12345")

		resp, err := client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode, "Expired token should return 401")
		wwwAuth := resp.Header.Get("WWW-Authenticate")
		assert.Contains(suite.T(), wwwAuth, "invalid_token", "Should contain invalid_token error")
		assert.Contains(suite.T(), wwwAuth, "expired", "Should mention token expiration")
	})

	suite.Run("Malformed Token Response Headers", func() {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequest("GET", suite.server.URL+"/auth/malformed", nil)
		require.NoError(suite.T(), err)

		req.Header.Set("Authorization", "Bearer malformed-token")

		resp, err := client.Do(req)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode, "Malformed token should return 401")
		wwwAuth := resp.Header.Get("WWW-Authenticate")
		assert.Contains(suite.T(), wwwAuth, "invalid_token", "Should contain invalid_token error")
	})
}

// TestTokenGeneration tests secure token generation
func (suite *AuthenticationTestSuite) TestTokenGeneration() {
	suite.Run("Generate Secure Random Token", func() {
		token := generateSecureToken(32)
		assert.Len(suite.T(), token, 44, "Base64 encoded 32-byte token should be 44 characters")
		_, err := base64.StdEncoding.DecodeString(token)
		assert.NoError(suite.T(), err, "Token should be valid base64")
	})

	suite.Run("Generate Tokens of Different Lengths", func() {
		lengths := []int{16, 24, 32, 64}
		for _, length := range lengths {
			token := generateSecureToken(length)
			expectedLen := base64.StdEncoding.EncodedLen(length)
			assert.Len(suite.T(), token, expectedLen,
				fmt.Sprintf("Token of length %d should encode to %d characters", length, expectedLen))
		}
	})

	suite.Run("Token Uniqueness", func() {
		tokens := make(map[string]bool)
		for i := 0; i < 100; i++ {
			token := generateSecureToken(32)
			assert.False(suite.T(), tokens[token], "Generated tokens should be unique")
			tokens[token] = true
		}
		assert.Len(suite.T(), tokens, 100, "Should generate 100 unique tokens")
	})
}

// TestAuthenticationSecurityTests runs authentication security tests
func TestAuthenticationSecurityTests(t *testing.T) {
	suite.Run(t, new(AuthenticationTestSuite))
}

// TestTokenSecurityValidation tests additional security validations
func TestTokenSecurityValidation(t *testing.T) {
	t.Run("Token Strength Validation", func(t *testing.T) {
		// Test weak token patterns
		weakTokens := []string{
			"12345678",
			"password",
			"abcdefgh",
			"token123",
			"secret",
			"key123456",
		}

		for _, token := range weakTokens {
			isWeak := isWeakToken(token)
			assert.True(t, isWeak, "Token should be identified as weak: %s", token)
		}

		// Test strong tokens
		strongToken := generateSecureToken(32)
		isWeak := isWeakToken(strongToken)
		assert.False(t, isWeak, "Generated secure token should not be weak")
	})

	t.Run("Token Format Validation", func(t *testing.T) {
		validTokens := []string{
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			"v2.local.abcdefghijklmnopqrstuvwxyz1234567890",
			"Bearer abcdef1234567890",
		}

		for _, token := range validTokens {
			isValid := isValidTokenFormat(token)
			assert.True(t, isValid, "Token format should be valid: %s", token)
		}

		invalidTokens := []string{
			"",
			" ",
			"\t\n",
			"null",
			"undefined",
			string([]byte{0x00, 0x01, 0x02}),
		}

		for _, token := range invalidTokens {
			isValid := isValidTokenFormat(token)
			assert.False(t, isValid, "Token format should be invalid: %q", token)
		}
	})
}

// Helper functions for security testing

// generateSecureToken generates a cryptographically secure random token
func generateSecureToken(length int) string {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(fmt.Sprintf("Failed to generate secure token: %v", err))
	}
	return base64.StdEncoding.EncodeToString(bytes)
}

// isWeakToken checks if a token exhibits weak patterns
func isWeakToken(token string) bool {
	weakPatterns := []string{
		"123456", "password", "secret", "token", "key",
		"abc", "def", "test", "demo", "sample",
	}

	lowerToken := strings.ToLower(token)
	for _, pattern := range weakPatterns {
		if strings.Contains(lowerToken, pattern) {
			return true
		}
	}

	// Check for repetitive characters
	if len(token) > 0 {
		firstChar := token[0]
		allSame := true
		for _, char := range token {
			if char != rune(firstChar) {
				allSame = false
				break
			}
		}
		if allSame {
			return true
		}
	}

	// Check for sequential characters
	if isSequential(token) {
		return true
	}

	return false
}

// isValidTokenFormat checks if token has valid format
func isValidTokenFormat(token string) bool {
	if len(token) == 0 {
		return false
	}

	// Check for control characters
	for _, r := range token {
		if r < 32 || r == 127 {
			return false
		}
	}

	// Reject common sentinel/placeholder strings
	trimmed := strings.TrimSpace(token)
	invalidTokens := []string{"null", "undefined", "none", "nil", "false", "0"}
	for _, invalid := range invalidTokens {
		if strings.EqualFold(trimmed, invalid) {
			return false
		}
	}

	// Basic length check
	if len(token) < 8 {
		return false
	}

	return true
}

// isSequential checks if token contains sequential characters
func isSequential(token string) bool {
	if len(token) < 3 {
		return false
	}

	for i := 2; i < len(token); i++ {
		if token[i]-token[i-1] == 1 && token[i-1]-token[i-2] == 1 {
			return true
		}
	}

	return false
}
