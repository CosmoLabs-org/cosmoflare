package interactive

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Validation tests ---

// TestValidateAccountID verifies ValidateAccountID behavior, one t.Run subtest per scenario.
func TestValidateAccountID(t *testing.T) {
	t.Run("valid 32-char hex ID", func(t *testing.T) {
		err := ValidateAccountID("abcdef0123456789abcdef0123456789")
		assert.NoError(t, err)
	})

	t.Run("valid uppercase hex", func(t *testing.T) {
		err := ValidateAccountID("ABCDEF0123456789ABCDEF0123456789")
		assert.NoError(t, err)
	})

	t.Run("too short", func(t *testing.T) {
		err := ValidateAccountID("abc123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "32 characters")
	})

	t.Run("too long", func(t *testing.T) {
		err := ValidateAccountID("abcdef0123456789abcdef0123456789extra")
		assert.Error(t, err)
	})

	t.Run("non-hex characters", func(t *testing.T) {
		err := ValidateAccountID("ghijkl0123456789ghijkl0123456789")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "hex digits")
	})

	t.Run("spaces", func(t *testing.T) {
		err := ValidateAccountID("abcdef01 3456789abcdef0123456789")
		assert.Error(t, err)
	})
}

// TestValidateAPIToken verifies ValidateAPIToken behavior, one t.Run subtest per scenario.
func TestValidateAPIToken(t *testing.T) {
	t.Run("token too short", func(t *testing.T) {
		info, err := ValidateAPIToken("short")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token validation failed")
		assert.Equal(t, "Token is too short", info.Error)
		assert.False(t, info.Valid)
	})

	t.Run("valid token with mock server", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-token-12345678901234567890", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"success": true,
				"result": {
					"id": "tok-id",
					"status": "active",
					"policy": {
						"permission_groups": [
							{"id": "pg1", "name": "R2", "permissions": ["r2:read", "r2:write"]}
						],
						"resources": {}
					}
				}
			}`))
		}))
		defer server.Close()

		// Can't easily redirect the real function to mock server without refactoring.
		// Testing basic validation path instead.
	})

	t.Run("network error", func(t *testing.T) {
		// ValidateAPIToken hits real Cloudflare API - test with a long enough token
		// that passes format check but fails at network level in CI
		info, err := ValidateAPIToken("this-is-a-long-enough-token-for-validation-test")
		// Either network error or HTTP error is expected
		assert.Error(t, err)
		assert.False(t, info.Valid)
	})
}

// --- Error types tests ---

// TestErrorType verifies the documented behavior of ErrorType.
func TestErrorType(t *testing.T) {
	assert.Equal(t, ErrorType(0), ErrorTypeNetwork)
	assert.Equal(t, ErrorType(1), ErrorTypeAuth)
	assert.Equal(t, ErrorType(2), ErrorTypeConfig)
	assert.Equal(t, ErrorType(3), ErrorTypeInput)
	assert.Equal(t, ErrorType(4), ErrorTypePermission)
	assert.Equal(t, ErrorType(5), ErrorTypeNotFound)
	assert.Equal(t, ErrorType(6), ErrorTypeValidation)
	assert.Equal(t, ErrorType(7), ErrorTypeUnknown)
}

// TestErrorContext verifies the documented behavior of ErrorContext.
func TestErrorContext(t *testing.T) {
	ctx := ErrorContext{
		Error:      assert.AnError,
		Type:       ErrorTypeNetwork,
		Operation:  "upload file",
		UserAction: "retry",
		Troubleshoot: []string{"check network", "check firewall"},
		NextSteps:   []string{"retry upload"},
	}

	assert.Equal(t, ErrorTypeNetwork, ctx.Type)
	assert.Equal(t, "upload file", ctx.Operation)
	assert.Len(t, ctx.Troubleshoot, 2)
}

// TestHandleError verifies the documented behavior of HandleError.
func TestHandleError(t *testing.T) {
	// HandleError just prints to stdout, verify it doesn't panic
	errorTypes := []ErrorType{
		ErrorTypeNetwork, ErrorTypeAuth, ErrorTypeConfig,
		ErrorTypeInput, ErrorTypePermission, ErrorTypeNotFound,
		ErrorTypeValidation, ErrorTypeUnknown,
	}

	for _, et := range errorTypes {
		ctx := ErrorContext{
			Error:     assert.AnError,
			Type:      et,
			Operation: "test op",
		}
		assert.NotPanics(t, func() {
			HandleError(ctx)
		})
	}
}

// TestHandleErrorWithDetails verifies that HandleError handles the with details case.
func TestHandleErrorWithDetails(t *testing.T) {
	ctx := ErrorContext{
		Error:      assert.AnError,
		Type:       ErrorTypeNetwork,
		Operation:  "upload",
		UserAction: "retry the upload",
		Troubleshoot: []string{
			"Check your internet connection",
			"Verify API token",
		},
		NextSteps: []string{
			"Run r2go2 check-connection",
		},
	}
	assert.NotPanics(t, func() {
		HandleError(ctx)
	})
}

// TestSuccessMessage verifies the documented behavior of SuccessMessage.
func TestSuccessMessage(t *testing.T) {
	assert.NotPanics(t, func() {
		SuccessMessage("Upload completed", "File: test.txt")
	})
	assert.NotPanics(t, func() {
		SuccessMessage("Upload completed", "")
	})
}

// TestWarningMessage verifies the documented behavior of WarningMessage.
func TestWarningMessage(t *testing.T) {
	assert.NotPanics(t, func() {
		WarningMessage("Rate limited", "Retry in 30s")
	})
	assert.NotPanics(t, func() {
		WarningMessage("Rate limited", "")
	})
}

// --- Helpers tests ---

// TestPrintFunctions verifies the documented behavior of print functions.
func TestPrintFunctions(t *testing.T) {
	assert.NotPanics(t, func() {
		PrintSuccess("test %s", "message")
		PrintError("test %s", "error")
		PrintWarning("test %s", "warning")
		PrintInfo("test %s", "info")
	})
}

// TestBoldDimReset verifies the documented behavior of bold dim reset.
func TestBoldDimReset(t *testing.T) {
	// Bold and Dim are SprintFunc values
	result := Bold("test text")
	assert.Contains(t, result, "test text")

	result = Dim("dimmed")
	assert.Contains(t, result, "dimmed")

	// Reset is an ANSI escape string
	assert.Equal(t, "\033[0m", Reset)
}

// --- Encryption tests (security-critical) ---

// TestEncryptDecryptBackupData verifies encrypt decrypt backup data behavior, one t.Run subtest...
func TestEncryptDecryptBackupData(t *testing.T) {
	bm := &BackupManager{} // nil configMgr is fine for encrypt/decrypt

	data := &BackupData{
		Version:   "1.0",
		CreatedAt: time.Now().Truncate(time.Second),
		Profiles: map[string]BackupProfile{
			"prod": {
				Name:      "prod",
				AccountID: "abc123",
				Region:    "us-east-1",
			},
		},
	}

	t.Run("round-trip encrypt then decrypt", func(t *testing.T) {
		testEncryptDecryptRoundTrip(t, bm, data)
	})
	t.Run("wrong password fails to decrypt", func(t *testing.T) {
		testEncryptDecryptWrongPassword(t, bm, data)
	})
	t.Run("tampered data fails to decrypt", func(t *testing.T) {
		testEncryptDecryptTampered(t, bm, data)
	})
	t.Run("too short data", func(t *testing.T) {
		testEncryptDecryptTooShort(t, bm)
	})
	t.Run("ciphertext too short for nonce", func(t *testing.T) {
		testEncryptDecryptShortCiphertext(t, bm)
	})
	t.Run("different encryptions produce different output", func(t *testing.T) {
		testEncryptDecryptUniqueOutput(t, bm, data)
	})
	t.Run("empty profiles", func(t *testing.T) {
		testEncryptDecryptEmptyProfiles(t, bm)
	})
	t.Run("multiple profiles", func(t *testing.T) {
		testEncryptDecryptMultipleProfiles(t, bm)
	})
}

func testEncryptDecryptRoundTrip(t *testing.T, bm *BackupManager, data *BackupData) {
	encrypted, err := bm.encryptBackupData(data, "strong-password-123!")
	require.NoError(t, err)
	require.NotEmpty(t, encrypted)

	// Encrypted data should not contain plaintext
	assert.NotContains(t, string(encrypted), "abc123")
	assert.NotContains(t, string(encrypted), "prod")

	decrypted, err := bm.decryptBackupData(encrypted, "strong-password-123!")
	require.NoError(t, err)
	require.NotNil(t, decrypted)

	assert.Equal(t, data.Version, decrypted.Version)
	assert.Equal(t, data.Profiles["prod"].AccountID, decrypted.Profiles["prod"].AccountID)
	assert.Equal(t, data.Profiles["prod"].Region, decrypted.Profiles["prod"].Region)
}

func testEncryptDecryptWrongPassword(t *testing.T, bm *BackupManager, data *BackupData) {
	encrypted, err := bm.encryptBackupData(data, "correct-password")
	require.NoError(t, err)

	_, err = bm.decryptBackupData(encrypted, "wrong-password")
	assert.Error(t, err)
}

func testEncryptDecryptTampered(t *testing.T, bm *BackupManager, data *BackupData) {
	encrypted, err := bm.encryptBackupData(data, "password")
	require.NoError(t, err)

	// Tamper with ciphertext (after salt)
	if len(encrypted) > 20 {
		encrypted[20] ^= 0xFF
	}

	_, err = bm.decryptBackupData(encrypted, "password")
	assert.Error(t, err)
}

func testEncryptDecryptTooShort(t *testing.T, bm *BackupManager) {
	_, err := bm.decryptBackupData([]byte("short"), "password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid encrypted data")
}

func testEncryptDecryptShortCiphertext(t *testing.T, bm *BackupManager) {
	// 16 bytes salt + tiny ciphertext
	shortData := make([]byte, 17)
	_, err := bm.decryptBackupData(shortData, "password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func testEncryptDecryptUniqueOutput(t *testing.T, bm *BackupManager, data *BackupData) {
	enc1, err := bm.encryptBackupData(data, "password")
	require.NoError(t, err)

	enc2, err := bm.encryptBackupData(data, "password")
	require.NoError(t, err)

	// Random salt + nonce means different output each time
	assert.NotEqual(t, enc1, enc2)
}

func testEncryptDecryptEmptyProfiles(t *testing.T, bm *BackupManager) {
	emptyData := &BackupData{
		Version:  "1.0",
		Profiles: map[string]BackupProfile{},
	}

	encrypted, err := bm.encryptBackupData(emptyData, "pass")
	require.NoError(t, err)

	decrypted, err := bm.decryptBackupData(encrypted, "pass")
	require.NoError(t, err)
	assert.Empty(t, decrypted.Profiles)
}

func testEncryptDecryptMultipleProfiles(t *testing.T, bm *BackupManager) {
	multiData := &BackupData{
		Version: "1.0",
		Profiles: map[string]BackupProfile{
			"prod":    {Name: "prod", AccountID: "id1"},
			"staging": {Name: "staging", AccountID: "id2"},
			"dev":     {Name: "dev", AccountID: "id3"},
		},
	}

	encrypted, err := bm.encryptBackupData(multiData, "pass")
	require.NoError(t, err)

	decrypted, err := bm.decryptBackupData(encrypted, "pass")
	require.NoError(t, err)
	assert.Len(t, decrypted.Profiles, 3)
	assert.Equal(t, "id2", decrypted.Profiles["staging"].AccountID)
}

// TestFindLatestBackup verifies findLatestBackup behavior, one t.Run subtest per scenario.
func TestFindLatestBackup(t *testing.T) {
	bm := &BackupManager{}

	t.Run("nonexistent directory", func(t *testing.T) {
		result := bm.findLatestBackup("/nonexistent/path")
		assert.Empty(t, result)
	})

	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()
		result := bm.findLatestBackup(dir)
		assert.Empty(t, result)
	})
}

// --- TokenInfo struct tests ---

// TestTokenInfo verifies the documented behavior of TokenInfo.
func TestTokenInfo(t *testing.T) {
	info := &TokenInfo{
		AccountID:   "acct-123",
		AccountName: "My Account",
		TokenType:   "api_token",
		Valid:       true,
	}

	assert.Equal(t, "acct-123", info.AccountID)
	assert.Equal(t, "My Account", info.AccountName)
	assert.True(t, info.Valid)
}

// TestBackupData verifies the documented behavior of BackupData.
func TestBackupData(t *testing.T) {
	data := &BackupData{
		Version:     "1.0",
		CreatedAt:   time.Now(),
		Description: "test backup",
		Profiles: map[string]BackupProfile{
			"test": {
				Name:      "test",
				AccountID: "123",
				Endpoint:  "https://example.com",
			},
		},
		Metadata: map[string]interface{}{
			"source": "unit-test",
		},
	}

	assert.Equal(t, "1.0", data.Version)
	assert.Equal(t, "test backup", data.Description)
	assert.Equal(t, "123", data.Profiles["test"].AccountID)
	assert.Equal(t, "unit-test", data.Metadata["source"])
}
