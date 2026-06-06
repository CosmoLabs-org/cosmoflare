package interactive_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CosmoLabs-org/cosmoflare/internal/interactive"
	"github.com/CosmoLabs-org/cosmoflare/tests/helpers"
)

func TestValidateAccountID(t *testing.T) {
	helpers.SetupTest(t)

	// Test valid account IDs
	validAccountIDs := []string{
		"1234567890abcdef1234567890abcdef",
		"abcdef1234567890abcdef1234567890",
		"ABCDEF1234567890ABCDEF1234567890",
		"fedcba0987654321fedcba0987654321",
	}

	for _, accountID := range validAccountIDs {
		err := interactive.ValidateAccountID(accountID)
		assert.NoError(t, err, "Account ID %s should be valid", accountID)
	}

	// Test invalid account IDs by length
	invalidAccountIDs := []struct {
		accountID string
		expected  string
	}{
		{"", "account ID must be 32 characters long"},
		{"12345678", "account ID must be 32 characters long"},
		{"1234567890abcdef1234567890abcde", "account ID must be 32 characters long"},
		{"1234567890abcdef1234567890abcdef1", "account ID must be 32 characters long"},
	}

	for _, test := range invalidAccountIDs {
		err := interactive.ValidateAccountID(test.accountID)
		assert.Error(t, err, "Account ID %s should be invalid", test.accountID)
		assert.Contains(t, err.Error(), test.expected)
	}

	// Test with valid length but invalid characters
	invalidCharAccountIDs := []string{
		"1234567890abcdef1234567890abcdgf",
		"1234567890abcdef1234567890abcdef!",
	}

	for _, accountID := range invalidCharAccountIDs {
		err := interactive.ValidateAccountID(accountID)
		assert.Error(t, err, "Account ID %s should be invalid", accountID)
	}
}

func TestTokenInfoStruct(t *testing.T) {
	helpers.SetupTest(t)

	// Test TokenInfo struct creation
	tokenInfo := &interactive.TokenInfo{
		AccountID:   "1234567890abcdef1234567890abcdef",
		AccountName: "Test Account",
		TokenType:   "api_token",
		Valid:       true,
		Error:       "",
	}

	assert.Equal(t, "1234567890abcdef1234567890abcdef", tokenInfo.AccountID)
	assert.Equal(t, "Test Account", tokenInfo.AccountName)
	assert.Equal(t, "api_token", tokenInfo.TokenType)
	assert.True(t, tokenInfo.Valid)
	assert.Empty(t, tokenInfo.Error)

	// Test TokenInfo with error
	tokenInfoWithError := &interactive.TokenInfo{
		Valid: false,
		Error: "Invalid token format",
	}

	assert.False(t, tokenInfoWithError.Valid)
	assert.Equal(t, "Invalid token format", tokenInfoWithError.Error)
}

func TestValidateAccountIDEdgeCases(t *testing.T) {
	helpers.SetupTest(t)

	// Test edge cases for account ID validation

	// Valid hex digits only - create a 32 character string with valid hex chars
	validHex := "0123456789abcdefABCDEF"
	validAccountID := validHex + validHex + validHex + validHex // 64 chars, too long
	validAccountID = validAccountID[:32]                      // Take first 32

	err := interactive.ValidateAccountID(validAccountID)
	assert.NoError(t, err)

	// Test with mixed case
	mixedCase := "AbCdEf1234567890aBcDeF1234567890"
	err = interactive.ValidateAccountID(mixedCase)
	assert.NoError(t, err)

	// Test with zeros and ones
	zerosOnes := "00000000000000001111111111111111"
	err = interactive.ValidateAccountID(zerosOnes)
	assert.NoError(t, err)
}

func TestValidateAPITokenInfoFields(t *testing.T) {
	helpers.SetupTest(t)

	// Test with empty token
	tokenInfo, err := interactive.ValidateAPIToken("")
	assert.Error(t, err)
	assert.NotNil(t, tokenInfo)
	assert.False(t, tokenInfo.Valid)
	assert.Empty(t, tokenInfo.AccountID)
	assert.Empty(t, tokenInfo.AccountName)
	assert.Empty(t, tokenInfo.TokenType)
	assert.NotEmpty(t, tokenInfo.Error)
}

func TestValidateAccountIDEmpty(t *testing.T) {
	helpers.SetupTest(t)

	err := interactive.ValidateAccountID("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account ID must be 32 characters long")
}

func TestValidateAccountIDTooLong(t *testing.T) {
	helpers.SetupTest(t)

	// Test with 33 characters
	longAccountID := "1234567890abcdef1234567890abcdef1"
	err := interactive.ValidateAccountID(longAccountID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account ID must be 32 characters long")
}

func TestValidateAccountIDTooShort(t *testing.T) {
	helpers.SetupTest(t)

	// Test with 31 characters
	shortAccountID := "1234567890abcdef1234567890abcde"
	err := interactive.ValidateAccountID(shortAccountID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "account ID must be 32 characters long")
}

func TestValidateAccountIDLowercaseHex(t *testing.T) {
	helpers.SetupTest(t)

	// Test all valid lowercase hex characters
	lowercaseHex := "0123456789abcdef0123456789abcdef"
	err := interactive.ValidateAccountID(lowercaseHex)
	assert.NoError(t, err)
}

func TestValidateAccountIDUppercaseHex(t *testing.T) {
	helpers.SetupTest(t)

	// Test all valid uppercase hex characters
	uppercaseHex := "0123456789ABCDEF0123456789ABCDEF"
	err := interactive.ValidateAccountID(uppercaseHex)
	assert.NoError(t, err)
}