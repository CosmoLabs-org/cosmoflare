/*
Package interactive provides validation functions for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package interactive

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// cfAPIBaseURL is the Cloudflare API base URL, injectable for testing.
var cfAPIBaseURL = "https://api.cloudflare.com/client/v4"

// TokenInfo represents information extracted from a Cloudflare API token
type TokenInfo struct {
	AccountID   string `json:"account_id,omitempty"`
	AccountName string `json:"account_name,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
	Valid       bool   `json:"valid"`
	Error       string `json:"error,omitempty"`
}

// tokenVerifyResponse is the Cloudflare API response shape for token verification.
type tokenVerifyResponse struct {
	Success bool `json:"success"`
	Result  struct {
		ID        string    `json:"id"`
		Status    string    `json:"status"`
		Name      string    `json:"name"`
		ExpiresOn time.Time `json:"expires_on"`
		Policy    struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Effect      string `json:"effect"`
			Resources   struct {
				Computation []any `json:"computation"`
				AccountID   []any `json:"computation:cloudflare_account:account_id"`
			} `json:"resources"`
			PermissionGroups []struct {
				ID          string   `json:"id"`
				Name        string   `json:"name"`
				Permissions []string `json:"permissions"`
			} `json:"permission_groups"`
		} `json:"policy"`
	} `json:"result"`
}

// ValidateAPIToken validates a Cloudflare API token and extracts account info
func ValidateAPIToken(token string) (*TokenInfo, error) {
	info := &TokenInfo{}

	// Basic format validation
	if len(token) < 20 {
		info.Error = "Token is too short"
		return info, fmt.Errorf("token validation failed: %s", info.Error)
	}

	// Make a test API call to validate the token
	body, err := fetchTokenVerifyResponse(token, info)
	if err != nil {
		return info, err
	}

	// Parse the validation response
	validateResponse, err := parseTokenVerifyResponse(body, info)
	if err != nil {
		return info, err
	}

	// Check if token has R2 permissions
	if err := validateTokenPermissions(validateResponse, info); err != nil {
		return info, err
	}

	// Extract account ID from policy if available
	var accountID string
	if len(validateResponse.Result.Policy.Resources.AccountID) > 0 {
		if id, ok := validateResponse.Result.Policy.Resources.AccountID[0].(string); ok {
			accountID = id
		}
	}

	info.AccountID = accountID
	info.TokenType = "api_token"
	info.Valid = true
	return info, nil
}

// fetchTokenVerifyResponse performs the token verify API call and returns the
// raw response body. Failures are recorded on info and returned as errors.
func fetchTokenVerifyResponse(token string, info *TokenInfo) ([]byte, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", cfAPIBaseURL+"/user/tokens/verify", nil)
	if err != nil {
		info.Error = "Failed to create request"
		return nil, fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		info.Error = "Network error during validation"
		return nil, fmt.Errorf("network error during token validation: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		info.Error = "Failed to read response"
		return nil, fmt.Errorf("failed to read validation response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		info.Error = fmt.Sprintf("Token validation failed (HTTP %d)", resp.StatusCode)
		return nil, fmt.Errorf("token validation failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// parseTokenVerifyResponse unmarshals the verify response body and checks the
// API success flag. Failures are recorded on info and returned as errors.
func parseTokenVerifyResponse(body []byte, info *TokenInfo) (*tokenVerifyResponse, error) {
	var validateResponse tokenVerifyResponse

	if err := json.Unmarshal(body, &validateResponse); err != nil {
		info.Error = "Failed to parse validation response"
		return nil, fmt.Errorf("failed to parse token validation response: %w", err)
	}

	if !validateResponse.Success {
		info.Error = "Token validation API returned failure"
		return nil, fmt.Errorf("token validation API returned failure: %s", string(body))
	}

	return &validateResponse, nil
}

// validateTokenPermissions checks that the verify response policy grants R2
// permissions. Failure is recorded on info and returned as an error.
func validateTokenPermissions(validateResponse *tokenVerifyResponse, info *TokenInfo) error {
	hasR2Permissions := false
	for _, pg := range validateResponse.Result.Policy.PermissionGroups {
		for _, perm := range pg.Permissions {
			if strings.Contains(perm, "r2:") {
				hasR2Permissions = true
				break
			}
		}
		if hasR2Permissions {
			break
		}
	}

	if !hasR2Permissions {
		info.Error = "Token lacks R2 permissions"
		return fmt.Errorf("token validation failed: token lacks required R2 permissions")
	}

	return nil
}

// autoDetectAccountInfo attempts to auto-detect account information
func autoDetectAccountInfo(token string) (string, string) {
	// Try to validate the token first
	tokenInfo, err := ValidateAPIToken(token)
	if err != nil || !tokenInfo.Valid {
		return "", ""
	}

	// If we got an account ID from token validation, try to get account name
	if tokenInfo.AccountID != "" {
		accountName := getAccountName(token, tokenInfo.AccountID)
		return tokenInfo.AccountID, accountName
	}

	return "", ""
}

// getAccountName fetches the account name from Cloudflare API
func getAccountName(token, accountID string) string {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/accounts/%s", cfAPIBaseURL, accountID), nil)
	if err != nil {
		return ""
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var accountResponse struct {
		Success bool `json:"success"`
		Result  struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &accountResponse); err != nil {
		return ""
	}

	if accountResponse.Success {
		return accountResponse.Result.Name
	}

	return ""
}

// ValidateAccountID validates the format of a Cloudflare Account ID
func ValidateAccountID(accountID string) error {
	if len(accountID) != 32 {
		return fmt.Errorf("account ID must be 32 characters long, got %d", len(accountID))
	}

	// Check if all characters are hex digits
	for _, c := range accountID {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("account ID must contain only hex digits (0-9, a-f, A-F)")
		}
	}

	return nil
}

// TestConnection tests the connection to Cloudflare R2 API
func TestConnection(accountID, token string) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Try to list buckets as a connection test
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/accounts/%s/r2/buckets", cfAPIBaseURL, accountID), nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	defer resp.Body.Close()

	// Read and discard the body
	_, _ = io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound {
		// 200 = success, 404 = no buckets (still a valid connection)
		return nil
	}

	return fmt.Errorf("connection test failed: HTTP %d", resp.StatusCode)
}