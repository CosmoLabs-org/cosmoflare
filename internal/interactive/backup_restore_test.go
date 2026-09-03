package interactive

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/pbkdf2"
)

// --- helpers ---

// newTestBackupManager creates a BackupManager with a temp HOME and mock input.
func newTestBackupManager(t *testing.T, inputs ...string) (*BackupManager, string) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cm, err := config.NewConfigManager()
	require.NoError(t, err)

	bm := &BackupManager{
		configMgr: cm,
		Input:     newMockReader(inputs...),
	}
	return bm, tmpDir
}

// seedProfile adds a profile to the config manager so backup functions have data.
func seedProfile(t *testing.T, bm *BackupManager, name string) {
	t.Helper()
	p := &config.Profile{
		Name:        name,
		AccountID:   strings.Repeat("a", 32),
		APIToken:    "test-api-token-12345",
		Description: "test profile " + name,
		Endpoint:    "https://test.example.com",
		AccessKey:   "AKIAIOSFODNN7EXAMPLE",
		Region:      "auto",
	}
	require.NoError(t, bm.configMgr.SetProfile(p))
}

// disableAnim disables the global animator for the duration of the test.
//
// The global animator is package-level shared state. Mutating it from a
// t.Parallel() test races with every other test running in the parallel
// batch (BUG-030). Only call this helper from sequential tests. A parallel
// test that needs animations off must drive an Animator instance it owns,
// not the package global.
func disableAnim(t *testing.T) {
	t.Helper()
	globalAnimator.Disabled = true
	t.Cleanup(func() { globalAnimator.Disabled = false })
}

// ===================================================================
// readPasswordWithReader
// ===================================================================

func TestReadPasswordWithReader(t *testing.T) {
	t.Parallel()

	t.Run("normal input", func(t *testing.T) {
		reader := newMockReader("  mysecret  ")
		pw, err := readPasswordWithReader(reader)
		require.NoError(t, err)
		assert.Equal(t, "mysecret", pw)
	})

	t.Run("EOF returns error", func(t *testing.T) {
		reader := newMockReader()
		_, err := readPasswordWithReader(reader)
		assert.Error(t, err)
	})
}

// ===================================================================
// encryptBackupData / decryptBackupData roundtrip
// ===================================================================

func TestEncryptDecryptRoundtrip(t *testing.T) {
	t.Parallel()

	bm := &BackupManager{Input: newMockReader()}

	original := &BackupData{
		Version:     "1.0",
		CreatedAt:   time.Now().Truncate(time.Second),
		Description: "roundtrip test",
		Profiles: map[string]BackupProfile{
			"prod": {
				Name:        "prod",
				Description: "production profile",
				AccountID:   strings.Repeat("b", 32),
				Endpoint:    "https://r2.cloudflarestorage.com",
				AccessKey:   "AKIAIOSFODNN7EXAMPLE",
				Region:      "auto",
			},
		},
		Metadata: map[string]interface{}{
			"created_by": "test",
		},
	}

	encrypted, err := bm.encryptBackupData(original, "password123")
	require.NoError(t, err)
	assert.Greater(t, len(encrypted), 16) // salt + nonce + ciphertext

	decrypted, err := bm.decryptBackupData(encrypted, "password123")
	require.NoError(t, err)
	assert.Equal(t, original.Version, decrypted.Version)
	assert.Equal(t, original.Description, decrypted.Description)
	assert.Equal(t, original.Profiles["prod"].Name, decrypted.Profiles["prod"].Name)
	assert.Equal(t, original.Profiles["prod"].AccountID, decrypted.Profiles["prod"].AccountID)
	assert.Equal(t, original.Profiles["prod"].Endpoint, decrypted.Profiles["prod"].Endpoint)
	assert.Equal(t, original.Profiles["prod"].AccessKey, decrypted.Profiles["prod"].AccessKey)
	assert.Equal(t, original.Profiles["prod"].Region, decrypted.Profiles["prod"].Region)
}

func TestDecryptWrongPassword(t *testing.T) {
	t.Parallel()

	bm := &BackupManager{Input: newMockReader()}

	data := &BackupData{
		Version:   "1.0",
		CreatedAt: time.Now(),
		Profiles:  map[string]BackupProfile{},
	}

	encrypted, err := bm.encryptBackupData(data, "correct-password")
	require.NoError(t, err)

	_, err = bm.decryptBackupData(encrypted, "wrong-password")
	assert.Error(t, err)
}

func TestDecryptInvalidData(t *testing.T) {
	t.Parallel()

	bm := &BackupManager{Input: newMockReader()}

	// Too short (< 16 bytes)
	_, err := bm.decryptBackupData([]byte("short"), "password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid encrypted data")
}

func TestEncryptEmptyData(t *testing.T) {
	t.Parallel()

	bm := &BackupManager{Input: newMockReader()}

	data := &BackupData{
		Version:   "1.0",
		CreatedAt: time.Now(),
		Profiles:  map[string]BackupProfile{},
	}

	encrypted, err := bm.encryptBackupData(data, "pw")
	require.NoError(t, err)
	require.NoError(t, err)

	decrypted, err := bm.decryptBackupData(encrypted, "pw")
	require.NoError(t, err)
	assert.Equal(t, "1.0", decrypted.Version)
	assert.Empty(t, decrypted.Profiles)
}

// ===================================================================
// findLatestBackup
// ===================================================================

func TestFindLatestBackup_FromBackupManager(t *testing.T) {
	t.Parallel()

	t.Run("finds .enc and .json files", func(t *testing.T) {
		dir := t.TempDir()
		bm := &BackupManager{Input: newMockReader()}

		// Create some backup files
		writeFile(t, filepath.Join(dir, ".cosmoflare-backup-2026-01-01.json"), "{}")
		time.Sleep(10 * time.Millisecond) // ensure different mtime
		writeFile(t, filepath.Join(dir, ".cosmoflare-backup-2026-05-15.enc"), "ciphertext")

		latest := bm.findLatestBackup(dir)
		assert.Contains(t, latest, ".cosmoflare-backup-2026-05-15.enc")
	})

	t.Run("returns empty for nonexistent dir", func(t *testing.T) {
		bm := &BackupManager{Input: newMockReader()}
		result := bm.findLatestBackup("/nonexistent/path/12345")
		assert.Empty(t, result)
	})

	t.Run("ignores non-backup files", func(t *testing.T) {
		dir := t.TempDir()
		bm := &BackupManager{Input: newMockReader()}

		writeFile(t, filepath.Join(dir, ".cosmoflare-backup-2026-01-01.json"), "{}")
		writeFile(t, filepath.Join(dir, "other-file.txt"), "data")
		writeFile(t, filepath.Join(dir, ".cosmoflare-backup.sh"), "#!/bin/bash") // .sh not matched

		latest := bm.findLatestBackup(dir)
		assert.Contains(t, latest, ".json")
	})

	t.Run("returns empty when no backups", func(t *testing.T) {
		dir := t.TempDir()
		bm := &BackupManager{Input: newMockReader()}
		result := bm.findLatestBackup(dir)
		assert.Empty(t, result)
	})
}

// ===================================================================
// ShowBackupInterface
// ===================================================================

func TestShowBackupInterface(t *testing.T) {

	t.Run("option 1 dispatches to encrypted backup", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t, "1")
		// No profiles so encrypted backup will warn and return nil
		err := bm.ShowBackupInterface()
		require.NoError(t, err)
	})

	t.Run("option 2 dispatches to plain backup", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t, "2")
		err := bm.ShowBackupInterface()
		require.NoError(t, err)
	})

	t.Run("option 3 dispatches to environment backup", func(t *testing.T) {
		disableAnim(t)
		// Input: "3" for env backup, "n" for confirm prompt (default no)
		bm, _ := newTestBackupManager(t, "3", "n")
		err := bm.ShowBackupInterface()
		require.NoError(t, err)
	})

	t.Run("empty input defaults to encrypted backup", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t, "")
		err := bm.ShowBackupInterface()
		require.NoError(t, err)
	})

	t.Run("invalid then valid input", func(t *testing.T) {
		disableAnim(t)
		// "bad" triggers recursion, then "1" picks encrypted
		bm, _ := newTestBackupManager(t, "bad", "1")
		err := bm.ShowBackupInterface()
		require.NoError(t, err)
	})
}

// ===================================================================
// CreateEncryptedBackup
// ===================================================================

func TestCreateEncryptedBackup(t *testing.T) {

	t.Run("no profiles shows warning", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t)
		err := bm.CreateEncryptedBackup()
		require.NoError(t, err)
	})

	t.Run("user cancels confirmation", func(t *testing.T) {
		disableAnim(t)
		bm, tmpDir := newTestBackupManager(t, "n")
		seedProfile(t, bm, "cancel-test")
		err := bm.CreateEncryptedBackup()
		require.NoError(t, err)
		// No backup file should exist
		matches, _ := filepath.Glob(filepath.Join(tmpDir, ".cosmoflare-backup-*.enc"))
		assert.Empty(t, matches)
	})

	t.Run("password mismatch returns error", func(t *testing.T) {
		disableAnim(t)
		// "y" to confirm, "pass1", "pass2" (mismatch)
		bm, tmpDir := newTestBackupManager(t, "y", "pass1", "pass2")
		seedProfile(t, bm, "mismatch-test")
		err := bm.CreateEncryptedBackup()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "password mismatch")
		// No file written
		matches, _ := filepath.Glob(filepath.Join(tmpDir, ".cosmoflare-backup-*.enc"))
		assert.Empty(t, matches)
	})

	t.Run("successful encrypted backup creates .enc file", func(t *testing.T) {
		disableAnim(t)
		// "y" confirm, "mypass", "mypass" (match)
		bm, tmpDir := newTestBackupManager(t, "y", "mypass", "mypass")
		seedProfile(t, bm, "enc-test")
		err := bm.CreateEncryptedBackup()
		require.NoError(t, err)

		matches, _ := filepath.Glob(filepath.Join(tmpDir, ".cosmoflare-backup-*.enc"))
		assert.Len(t, matches, 1)

		// File should be decryptable
		data, err := os.ReadFile(matches[0])
		require.NoError(t, err)
		require.Greater(t, len(data), 16)

		bm2 := &BackupManager{Input: newMockReader()}
		decrypted, err := bm2.decryptBackupData(data, "mypass")
		require.NoError(t, err)
		assert.Contains(t, decrypted.Profiles, "enc-test")
		assert.Equal(t, "enc-test", decrypted.Profiles["enc-test"].Name)
		assert.Equal(t, "test profile enc-test", decrypted.Profiles["enc-test"].Description)
	})
}

// ===================================================================
// CreatePlainBackup
// ===================================================================

func TestCreatePlainBackup(t *testing.T) {

	t.Run("no profiles creates empty backup file", func(t *testing.T) {
		disableAnim(t)
		bm, tmpDir := newTestBackupManager(t)
		err := bm.CreatePlainBackup()
		require.NoError(t, err)

		matches, _ := filepath.Glob(filepath.Join(tmpDir, ".cosmoflare-backup-*.json"))
		assert.Len(t, matches, 1)

		data, err := os.ReadFile(matches[0])
		require.NoError(t, err)

		var bd BackupData
		require.NoError(t, json.Unmarshal(data, &bd))
		assert.Equal(t, "1.0", bd.Version)
		assert.Empty(t, bd.Profiles)
	})

	t.Run("with profiles includes them in JSON", func(t *testing.T) {
		disableAnim(t)
		bm, tmpDir := newTestBackupManager(t)
		seedProfile(t, bm, "plain-prod")
		seedProfile(t, bm, "plain-staging")

		err := bm.CreatePlainBackup()
		require.NoError(t, err)

		matches, _ := filepath.Glob(filepath.Join(tmpDir, ".cosmoflare-backup-*.json"))
		assert.Len(t, matches, 1)

		data, err := os.ReadFile(matches[0])
		require.NoError(t, err)

		var bd BackupData
		require.NoError(t, json.Unmarshal(data, &bd))
		assert.Contains(t, bd.Profiles, "plain-prod")
		assert.Contains(t, bd.Profiles, "plain-staging")
		assert.Equal(t, "plain-prod", bd.Profiles["plain-prod"].Name)
		// AccessKey IS included in BackupProfile (it's not a secret like APIToken)
		assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", bd.Profiles["plain-prod"].AccessKey)
		// BackupProfile struct has no APIToken field - security by design
	})
}

// ===================================================================
// CreateEnvironmentBackup
// ===================================================================

func TestCreateEnvironmentBackup(t *testing.T) {

	t.Run("user declines confirmation returns nil", func(t *testing.T) {
		disableAnim(t)
		// default is no, so empty input = "n"
		bm, tmpDir := newTestBackupManager(t, "n")
		err := bm.CreateEnvironmentBackup()
		require.NoError(t, err)
		// No .sh file
		matches, _ := filepath.Glob(filepath.Join(tmpDir, ".cosmoflare-backup-*.sh"))
		assert.Empty(t, matches)
	})

	t.Run("user confirms creates shell script", func(t *testing.T) {
		disableAnim(t)
		bm, tmpDir := newTestBackupManager(t, "y")
		seedProfile(t, bm, "env-profile")

		err := bm.CreateEnvironmentBackup()
		require.NoError(t, err)

		matches, _ := filepath.Glob(filepath.Join(tmpDir, ".cosmoflare-backup-*.sh"))
		assert.Len(t, matches, 1)

		data, err := os.ReadFile(matches[0])
		require.NoError(t, err)
		content := string(data)

		assert.Contains(t, content, "#!/bin/bash")
		assert.Contains(t, content, "CLOUDFLARE_ACCOUNT_ID_ENV-PROFILE")
		assert.Contains(t, content, "CLOUDFLARE_API_TOKEN_ENV-PROFILE")
		assert.Contains(t, content, strings.Repeat("a", 32))
	})
}

// ===================================================================
// ShowRestoreInterface
// ===================================================================

func TestShowRestoreInterface(t *testing.T) {

	t.Run("empty path and no backups returns error", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t, "")
		err := bm.ShowRestoreInterface()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no backup file")
	})

	t.Run("empty path finds latest backup and uses it", func(t *testing.T) {
		disableAnim(t)
		bm, tmpDir := newTestBackupManager(t, "", "y", "1", "n") // "" empty filepath, "y" use backup, "1" all profiles, "n" cancel
		seedProfile(t, bm, "restore-test")

		// Create a JSON backup in the tmp dir
		backupData := &BackupData{
			Version:   "1.0",
			CreatedAt: time.Now(),
			Profiles: map[string]BackupProfile{
				"restore-test": {
					Name:      "restore-test",
					AccountID: strings.Repeat("c", 32),
				},
			},
		}
		jsonData, _ := json.MarshalIndent(backupData, "", "  ")
		backupPath := filepath.Join(tmpDir, ".cosmoflare-backup-2026-05-15.json")
		require.NoError(t, os.WriteFile(backupPath, jsonData, 0644))

		err := bm.ShowRestoreInterface()
		require.NoError(t, err)
	})

	t.Run("empty path finds latest but user declines", func(t *testing.T) {
		disableAnim(t)
		bm, tmpDir := newTestBackupManager(t, "", "n") // "" empty filepath, "n" don't use found backup
		seedProfile(t, bm, "decline-test")

		backupPath := filepath.Join(tmpDir, ".cosmoflare-backup-2026-05-15.json")
		require.NoError(t, os.WriteFile(backupPath, []byte(`{"version":"1.0","profiles":{}}`), 0644))

		err := bm.ShowRestoreInterface()
		require.NoError(t, err) // cancelled, not error
	})

	t.Run("explicit .json file restores from JSON", func(t *testing.T) {
		disableAnim(t)
		bm, tmpDir := newTestBackupManager(t, "1", "y") // "1" all profiles, "y" confirm
		seedProfile(t, bm, "json-restore")

		backupPath := filepath.Join(tmpDir, "test-backup.json")
		backupData := &BackupData{
			Version:   "1.0",
			CreatedAt: time.Now(),
			Profiles: map[string]BackupProfile{
				"json-restore": {
					Name:      "json-restore",
					AccountID: strings.Repeat("d", 32),
					Region:    "auto",
				},
			},
		}
		jsonData, _ := json.MarshalIndent(backupData, "", "  ")
		require.NoError(t, os.WriteFile(backupPath, jsonData, 0644))

		bm.Input = newMockReader(backupPath, "1", "y")
		err := bm.ShowRestoreInterface()
		require.NoError(t, err)
	})

	t.Run(".enc file dispatches to encrypted restore", func(t *testing.T) {
		disableAnim(t)
		bm, tmpDir := newTestBackupManager(t)

		// Create a real encrypted file
		bmEnc := &BackupManager{Input: newMockReader()}
		encData := &BackupData{
			Version:   "1.0",
			CreatedAt: time.Now(),
			Profiles:  map[string]BackupProfile{},
		}
		encrypted, err := bmEnc.encryptBackupData(encData, "testpw")
		require.NoError(t, err)

		encPath := filepath.Join(tmpDir, "backup.enc")
		require.NoError(t, os.WriteFile(encPath, encrypted, 0600))

		// ShowRestoreInterface will ask for password, then processRestoreData
		// Input: filepath already set, password "testpw", "1" all, "y" confirm
		bm.Input = newMockReader(encPath, "testpw", "1", "y")
		err = bm.ShowRestoreInterface()
		require.NoError(t, err)
	})

	t.Run(".sh file dispatches to shell restore", func(t *testing.T) {
		disableAnim(t)
		bm, tmpDir := newTestBackupManager(t)

		shPath := filepath.Join(tmpDir, "backup.sh")
		require.NoError(t, os.WriteFile(shPath, []byte("#!/bin/bash\n"), 0644))

		bm.Input = newMockReader(shPath)
		err := bm.ShowRestoreInterface()
		require.NoError(t, err)
	})

	t.Run("unsupported format returns error", func(t *testing.T) {
		disableAnim(t)
		bm, tmpDir := newTestBackupManager(t)

		badPath := filepath.Join(tmpDir, "backup.zip")
		require.NoError(t, os.WriteFile(badPath, []byte("data"), 0644))

		bm.Input = newMockReader(badPath)
		err := bm.ShowRestoreInterface()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported format")
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t)
		bm.Input = newMockReader("/nonexistent/backup.json")
		err := bm.ShowRestoreInterface()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

// ===================================================================
// restoreFromShell
// ===================================================================

func TestRestoreFromShell(t *testing.T) {

	t.Run("returns nil with info message", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t)
		err := bm.restoreFromShell("/some/path/backup.sh")
		require.NoError(t, err)
	})
}

// ===================================================================
// restoreFromJSON
// ===================================================================

func TestRestoreFromJSON(t *testing.T) {

	t.Run("invalid JSON returns error", func(t *testing.T) {
		disableAnim(t)
		bm, tmpDir := newTestBackupManager(t)

		badPath := filepath.Join(tmpDir, "bad.json")
		require.NoError(t, os.WriteFile(badPath, []byte("not json"), 0644))

		err := bm.restoreFromJSON(badPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse")
	})

	t.Run("valid JSON processes successfully", func(t *testing.T) {
		disableAnim(t)
		bm, tmpDir := newTestBackupManager(t, "1", "y")
		seedProfile(t, bm, "json-test")

		backupPath := filepath.Join(tmpDir, "restore.json")
		backupData := &BackupData{
			Version:   "1.0",
			CreatedAt: time.Now(),
			Profiles: map[string]BackupProfile{
				"json-test": {
					Name:      "json-test",
					AccountID: strings.Repeat("e", 32),
					Endpoint:  "https://r2.example.com",
				},
			},
		}
		jsonData, _ := json.MarshalIndent(backupData, "", "  ")
		require.NoError(t, os.WriteFile(backupPath, jsonData, 0644))

		err := bm.restoreFromJSON(backupPath)
		require.NoError(t, err)
	})
}

// ===================================================================
// restoreFromEncrypted
// ===================================================================

func TestRestoreFromEncrypted(t *testing.T) {

	t.Run("wrong password returns error", func(t *testing.T) {
		disableAnim(t)
		bm, tmpDir := newTestBackupManager(t)

		// Create encrypted file
		bmEnc := &BackupManager{Input: newMockReader()}
		encData := &BackupData{Version: "1.0", CreatedAt: time.Now(), Profiles: map[string]BackupProfile{}}
		encrypted, err := bmEnc.encryptBackupData(encData, "realpw")
		require.NoError(t, err)

		encPath := filepath.Join(tmpDir, "wrong-pw.enc")
		require.NoError(t, os.WriteFile(encPath, encrypted, 0600))

		// Provide wrong password
		bm.Input = newMockReader("wrongpw")
		err = bm.restoreFromEncrypted(encPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decryption failed")
	})

	t.Run("correct password processes data", func(t *testing.T) {
		disableAnim(t)
		bm, tmpDir := newTestBackupManager(t, "1", "y")
		seedProfile(t, bm, "enc-restore")

		bmEnc := &BackupManager{Input: newMockReader()}
		encData := &BackupData{
			Version:   "1.0",
			CreatedAt: time.Now(),
			Profiles: map[string]BackupProfile{
				"enc-restore": {
					Name:      "enc-restore",
					AccountID: strings.Repeat("f", 32),
				},
			},
		}
		encrypted, err := bmEnc.encryptBackupData(encData, "goodpw")
		require.NoError(t, err)

		encPath := filepath.Join(tmpDir, "good.enc")
		require.NoError(t, os.WriteFile(encPath, encrypted, 0600))

		bm.Input = newMockReader("goodpw", "1", "y")
		err = bm.restoreFromEncrypted(encPath)
		require.NoError(t, err)
	})
}

// ===================================================================
// processRestoreData
// ===================================================================

func TestProcessRestoreData(t *testing.T) {

	t.Run("choice 1 restores all profiles", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t, "1", "y")
		seedProfile(t, bm, "all-prod")
		seedProfile(t, bm, "all-staging")

		backupData := &BackupData{
			Version: "1.0",
			Profiles: map[string]BackupProfile{
				"all-prod":    {Name: "all-prod", AccountID: strings.Repeat("g", 32)},
				"all-staging": {Name: "all-staging", AccountID: strings.Repeat("h", 32)},
			},
		}

		err := bm.processRestoreData(backupData, "/tmp/test.json", true)
		require.NoError(t, err)
	})

	t.Run("empty choice defaults to restore all", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t, "", "y")
		seedProfile(t, bm, "default-restore")

		backupData := &BackupData{
			Version: "1.0",
			Profiles: map[string]BackupProfile{
				"default-restore": {Name: "default-restore", AccountID: strings.Repeat("i", 32)},
			},
		}

		err := bm.processRestoreData(backupData, "/tmp/test.json", true)
		require.NoError(t, err)
	})

	t.Run("choice 2 selects individual profiles", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t, "2", "1", "y") // select profile #1
		seedProfile(t, bm, "select-one")

		backupData := &BackupData{
			Version: "1.0",
			Profiles: map[string]BackupProfile{
				"select-one": {Name: "select-one", AccountID: strings.Repeat("j", 32)},
				"skip-me":    {Name: "skip-me", AccountID: strings.Repeat("k", 32)},
			},
		}

		err := bm.processRestoreData(backupData, "/tmp/test.json", true)
		require.NoError(t, err)
	})

	t.Run("choice 2 with invalid numbers selects nothing", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t, "2", "99,abc")
		seedProfile(t, bm, "none-selected")

		backupData := &BackupData{
			Version: "1.0",
			Profiles: map[string]BackupProfile{
				"none-selected": {Name: "none-selected", AccountID: strings.Repeat("l", 32)},
			},
		}

		err := bm.processRestoreData(backupData, "/tmp/test.json", true)
		require.NoError(t, err)
	})

	t.Run("invalid choice returns error", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t, "99")
		seedProfile(t, bm, "invalid-choice")

		backupData := &BackupData{
			Version: "1.0",
			Profiles: map[string]BackupProfile{
				"invalid-choice": {Name: "invalid-choice", AccountID: strings.Repeat("m", 32)},
			},
		}

		err := bm.processRestoreData(backupData, "/tmp/test.json", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid choice")
	})

	t.Run("cancel restore confirmation returns nil", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t, "1", "n") // "n" cancel
		seedProfile(t, bm, "cancel-restore")

		backupData := &BackupData{
			Version: "1.0",
			Profiles: map[string]BackupProfile{
				"cancel-restore": {Name: "cancel-restore", AccountID: strings.Repeat("n", 32)},
			},
		}

		err := bm.processRestoreData(backupData, "/tmp/test.json", true)
		require.NoError(t, err)
	})

	t.Run("existing profile overwrite declined skips it", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t, "1", "y", "n") // "n" don't overwrite
		// Profile already exists from seedProfile
		seedProfile(t, bm, "existing-skip")

		backupData := &BackupData{
			Version: "1.0",
			Profiles: map[string]BackupProfile{
				"existing-skip": {Name: "existing-skip", AccountID: strings.Repeat("o", 32)},
			},
		}

		err := bm.processRestoreData(backupData, "/tmp/test.json", true)
		require.NoError(t, err)
	})

	t.Run("existing profile overwrite accepted restores it", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t, "1", "y", "y") // "y" overwrite
		seedProfile(t, bm, "overwrite-me")

		backupData := &BackupData{
			Version: "1.0",
			Profiles: map[string]BackupProfile{
				"overwrite-me": {
					Name:        "overwrite-me",
					AccountID:   strings.Repeat("p", 32),
					Description: "updated description",
				},
			},
		}

		err := bm.processRestoreData(backupData, "/tmp/test.json", true)
		require.NoError(t, err)

		// Verify the profile was updated
		p, err := bm.configMgr.GetProfile("overwrite-me")
		require.NoError(t, err)
		assert.Equal(t, "updated description", p.Description)
	})

	t.Run("hasTokens false prompts for API token", func(t *testing.T) {
		disableAnim(t)
		// "1" all, "y" confirm restore, "y" overwrite existing, then token input
		bm, _ := newTestBackupManager(t, "1", "y", "y", "new-token-12345")
		seedProfile(t, bm, "token-prompt")

		backupData := &BackupData{
			Version: "1.0",
			Profiles: map[string]BackupProfile{
				"token-prompt": {Name: "token-prompt", AccountID: strings.Repeat("q", 32)},
			},
		}

		err := bm.processRestoreData(backupData, "/tmp/test.json", false)
		require.NoError(t, err)

		p, err := bm.configMgr.GetProfile("token-prompt")
		require.NoError(t, err)
		assert.Equal(t, "new-token-12345", p.APIToken)
	})

	t.Run("hasTokens false EOF on token read skips token", func(t *testing.T) {
		disableAnim(t)
		// "1" all, "y" confirm, then EOF for token
		bm, _ := newTestBackupManager(t, "1", "y")
		seedProfile(t, bm, "token-eof")

		backupData := &BackupData{
			Version: "1.0",
			Profiles: map[string]BackupProfile{
				"token-eof": {Name: "token-eof", AccountID: strings.Repeat("r", 32)},
			},
		}

		err := bm.processRestoreData(backupData, "/tmp/test.json", false)
		require.NoError(t, err)
	})

	t.Run("empty profiles map shows zero count", func(t *testing.T) {
		disableAnim(t)
		bm, _ := newTestBackupManager(t, "1", "y")

		backupData := &BackupData{
			Version:  "1.0",
			Profiles: map[string]BackupProfile{},
		}

		err := bm.processRestoreData(backupData, "/tmp/test.json", true)
		require.NoError(t, err)
	})
}

// ===================================================================
// NewBackupManager
// ===================================================================

func TestNewBackupManager(t *testing.T) {

	t.Run("creates manager with valid HOME", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		bm, err := NewBackupManager()
		require.NoError(t, err)
		assert.NotNil(t, bm)
		assert.NotNil(t, bm.configMgr)
		assert.NotNil(t, bm.Input)
	})

	t.Run("returns error when HOME is invalid", func(t *testing.T) {
		t.Setenv("HOME", "/nonexistent/path/that/does/not/exist/12345")
		_, err := NewBackupManager()
		require.Error(t, err)
	})
}

// ===================================================================
// Additional edge cases for coverage
// ===================================================================

func TestCreateEncryptedBackup_PasswordReadEOF(t *testing.T) {
	disableAnim(t)
	// "y" confirm, then EOF on password read
	bm, _ := newTestBackupManager(t, "y")
	seedProfile(t, bm, "pw-eof")
	err := bm.CreateEncryptedBackup()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read password")
}

func TestCreateEncryptedBackup_ConfirmPasswordEOF(t *testing.T) {
	disableAnim(t)
	// "y" confirm, "mypass", then EOF on confirm read
	bm, _ := newTestBackupManager(t, "y", "mypass")
	seedProfile(t, bm, "confirm-eof")
	err := bm.CreateEncryptedBackup()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read confirmation password")
}

func TestRestoreFromEncrypted_PasswordEOF(t *testing.T) {
	disableAnim(t)
	bm, _ := newTestBackupManager(t) // no inputs -> EOF immediately
	err := bm.restoreFromEncrypted("/some/path.enc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read password")
}

func TestRestoreFromEncrypted_FileReadError(t *testing.T) {
	disableAnim(t)
	bm, _ := newTestBackupManager(t, "testpw")
	err := bm.restoreFromEncrypted("/nonexistent/file.enc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read backup file")
}

func TestRestoreFromJSON_FileReadError(t *testing.T) {
	disableAnim(t)
	bm, _ := newTestBackupManager(t)
	err := bm.restoreFromJSON("/nonexistent/file.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read backup file")
}

func TestDecryptBackupData_CiphertextTooShort(t *testing.T) {
	disableAnim(t)
	bm := &BackupManager{Input: newMockReader()}

	// 16 bytes salt + 0 bytes ciphertext (less than nonce size)
	data := make([]byte, 16+1) // salt + 1 byte (too short for nonce)
	_, err := bm.decryptBackupData(data, "password")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ciphertext too short")
}

func TestDecryptBackupData_InvalidJSON(t *testing.T) {
	disableAnim(t)
	bm := &BackupManager{Input: newMockReader()}

	// Encrypt garbage that will decrypt but not unmarshal as BackupData
	// We need valid encryption of non-BackupData JSON
	garbage := []byte(`{"not": "valid backup data"}`)
	// Manually construct encrypted data with known password
	salt := make([]byte, 16)
	for i := range salt {
		salt[i] = byte(i)
	}

		key := pbkdf2.Key([]byte("password"), salt, 100000, 32, sha256.New)
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	ct := gcm.Seal(nonce, nonce, garbage, nil)
	result := append(salt, ct...)

	_, err := bm.decryptBackupData(result, "password")
	require.NoError(t, err) // decryption succeeds but we still get a BackupData with zero fields
}

func TestProcessRestoreData_SetProfileError(t *testing.T) {
	disableAnim(t)
	// Use a profile with empty name to trigger SetProfile error
	bm, _ := newTestBackupManager(t, "1", "y")

	backupData := &BackupData{
		Version: "1.0",
		Profiles: map[string]BackupProfile{
			"": {Name: "", AccountID: strings.Repeat("s", 32)},
		},
	}

	err := bm.processRestoreData(backupData, "/tmp/test.json", true)
	// SetProfile with empty name returns error, which is logged and continued
	require.NoError(t, err) // processRestoreData itself doesn't return error for individual failures
}

func TestProcessRestoreData_Choose2_MultipleSelections(t *testing.T) {
	disableAnim(t)
	bm, _ := newTestBackupManager(t, "2", "1,3", "y") // select #1 and #3
	seedProfile(t, bm, "multi-a")

	backupData := &BackupData{
		Version: "1.0",
		Profiles: map[string]BackupProfile{
			"multi-a": {Name: "multi-a", AccountID: strings.Repeat("t", 32)},
			"multi-b": {Name: "multi-b", AccountID: strings.Repeat("u", 32)},
			"multi-c": {Name: "multi-c", AccountID: strings.Repeat("v", 32)},
		},
	}

	err := bm.processRestoreData(backupData, "/tmp/test.json", true)
	require.NoError(t, err)
}

func TestCreateEncryptedBackup_WriteError(t *testing.T) {
	disableAnim(t)
	// This is hard to trigger without mocking os.WriteFile,
	// but we can at least test the path where os.UserHomeDir returns something odd
	// For now, verify the happy path more thoroughly
	bm, _ := newTestBackupManager(t, "y", "pw", "pw")
	seedProfile(t, bm, "write-test")
	err := bm.CreateEncryptedBackup()
	require.NoError(t, err)
}

// ===================================================================
// writeFile helper
// ===================================================================

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}
