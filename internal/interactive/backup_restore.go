/*
Package interactive provides backup and restore functionality for profiles

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package interactive

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/config"
)

// BackupData represents the backup structure
type BackupData struct {
	Version     string                    `json:"version"`
	CreatedAt   time.Time                 `json:"created_at"`
	Description string                    `json:"description,omitempty"`
	Profiles    map[string]BackupProfile  `json:"profiles"`
	Metadata    map[string]interface{}    `json:"metadata,omitempty"`
}

// BackupProfile represents a backed up profile
type BackupProfile struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	AccountID   string `json:"account_id"`
	Endpoint    string `json:"endpoint,omitempty"`
	AccessKey   string `json:"access_key,omitempty"`
	Region      string `json:"region,omitempty"`
	// Note: APIToken is never included in backups for security
	// SecretKey is never included in backups for security
}

// BackupManager handles backup and restore operations
type BackupManager struct {
	configMgr *config.ConfigManager
}

// NewBackupManager creates a new backup manager
func NewBackupManager() (*BackupManager, error) {
	configMgr, err := config.NewConfigManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	return &BackupManager{
		configMgr: configMgr,
	}, nil
}

// ShowBackupInterface displays the backup interface
func (bm *BackupManager) ShowBackupInterface() error {
	ClearScreen()
	fmt.Println()
	fmt.Println(Bold("💾 Backup Profiles"))
	fmt.Println(strings.Repeat("─", 30))
	fmt.Println()

	fmt.Println("Select backup format:")
	fmt.Println("  [1] Encrypted file (recommended) - Password protected")
	fmt.Println("  [2] Plain JSON file            - Not password protected")
	fmt.Println("  [3] Environment variables       - Export as shell script")
	fmt.Println()

	fmt.Printf("Select format [1]: ")
	var format string
	fmt.Scanln(&format)
	format = strings.TrimSpace(format)

	switch format {
	case "1", "":
		return bm.CreateEncryptedBackup()
	case "2":
		return bm.CreatePlainBackup()
	case "3":
		return bm.CreateEnvironmentBackup()
	default:
		PrintError("Invalid selection.")
		return bm.ShowBackupInterface()
	}
}

// CreateEncryptedBackup creates an encrypted backup
func (bm *BackupManager) CreateEncryptedBackup() error {
	fmt.Println()
	fmt.Println(Bold("🔐 Encrypted Backup"))
	fmt.Println(strings.Repeat("─", 20))
	fmt.Println()

	// Get profiles to backup
	profiles := bm.configMgr.ListProfiles()
	if len(profiles) == 0 {
		PrintWarning("No profiles found to backup.")
		return nil
	}

	fmt.Printf("Found %d profile(s):\n", len(profiles))
	for _, name := range profiles {
		profile, err := bm.configMgr.GetProfile(name)
		if err != nil {
			continue
		}
		fmt.Printf("  • %s %s\n", FormatProfileName(name, profile.Description), getProfileIcon(profile.Description))
	}

	fmt.Println()

	if !ConfirmYesNo("Continue with backup?", true) {
		return nil
	}

	// Get password
	fmt.Print("Enter backup password: ")
	password, err := readPassword()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	fmt.Print("Confirm password: ")
	confirmPassword, err := readPassword()
	if err != nil {
		return fmt.Errorf("failed to read confirmation password: %w", err)
	}

	if password != confirmPassword {
		PrintError("Passwords do not match.")
		return fmt.Errorf("password mismatch")
	}

	// Create backup data
	backupData := &BackupData{
		Version:     "1.0",
		CreatedAt:   time.Now(),
		Description: "R2Go2 profile backup",
		Profiles:    make(map[string]BackupProfile),
		Metadata: map[string]interface{}{
			"created_by": "R2Go2 CLI",
			"os":         os.Getenv("GOOS"),
		},
	}

	// Add profiles (without sensitive data)
	for _, name := range profiles {
		profile, err := bm.configMgr.GetProfile(name)
		if err != nil {
			continue
		}

		backupData.Profiles[name] = BackupProfile{
			Name:        profile.Name,
			Description: profile.Description,
			AccountID:   profile.AccountID,
			Endpoint:    profile.Endpoint,
			AccessKey:   profile.AccessKey,
			Region:      profile.Region,
		}
	}

	// Generate filename
	timestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf(".r2go2-backup-%s.enc", timestamp)
	homeDir, _ := os.UserHomeDir()
	filepath := filepath.Join(homeDir, filename)

	// Encrypt and save
	ShowSpinner("Creating encrypted backup...", 2*Second)

	encryptedData, err := bm.encryptBackupData(backupData, password)
	if err != nil {
		return fmt.Errorf("failed to encrypt backup: %w", err)
	}

	if err := os.WriteFile(filepath, encryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	PrintSuccess("✅ Backup completed successfully!")
	fmt.Printf("Backup saved to: %s\n", Info(filepath))
	PrintWarning("⚠️  Remember your password - it cannot be recovered!")

	return nil
}

// CreatePlainBackup creates a plain JSON backup
func (bm *BackupManager) CreatePlainBackup() error {
	fmt.Println()
	fmt.Println(Bold("📄 Plain JSON Backup"))
	fmt.Println(strings.Repeat("─", 25))
	fmt.Println()

	PrintWarning("⚠️  Plain backup will NOT include API tokens for security reasons.")
	fmt.Println()

	// Create backup data
	backupData := &BackupData{
		Version:     "1.0",
		CreatedAt:   time.Now(),
		Description: "R2Go2 profile backup (plain JSON)",
		Profiles:    make(map[string]BackupProfile),
	}

	profiles := bm.configMgr.ListProfiles()
	for _, name := range profiles {
		profile, err := bm.configMgr.GetProfile(name)
		if err != nil {
			continue
		}

		backupData.Profiles[name] = BackupProfile{
			Name:        profile.Name,
			Description: profile.Description,
			AccountID:   profile.AccountID,
			Endpoint:    profile.Endpoint,
			AccessKey:   profile.AccessKey,
			Region:      profile.Region,
		}
	}

	// Generate filename
	timestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf(".r2go2-backup-%s.json", timestamp)
	homeDir, _ := os.UserHomeDir()
	filepath := filepath.Join(homeDir, filename)

	// Save as JSON
	ShowSpinner("Creating JSON backup...", 1*Second)

	jsonData, err := json.MarshalIndent(backupData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal backup data: %w", err)
	}

	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	PrintSuccess("✅ Backup completed successfully!")
	fmt.Printf("Backup saved to: %s\n", Info(filepath))
	PrintInfo("💡 Note: You'll need to re-enter API tokens when restoring.")

	return nil
}

// CreateEnvironmentBackup creates environment variable backup
func (bm *BackupManager) CreateEnvironmentBackup() error {
	fmt.Println()
	fmt.Println(Bold("🌍 Environment Variables Backup"))
	fmt.Println(strings.Repeat("─", 35))
	fmt.Println()

	PrintWarning("⚠️  This creates a shell script with environment variables.")
	fmt.Println(Muted("API tokens will be included - keep this file secure!"))
	fmt.Println()

	if !ConfirmYesNo("Continue with environment backup?", false) {
		return nil
	}

	// Generate filename
	timestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf(".r2go2-backup-%s.sh", timestamp)
	homeDir, _ := os.UserHomeDir()
	filepath := filepath.Join(homeDir, filename)

	// Create shell script
	ShowSpinner("Creating shell script backup...", 1*Second)

	var script strings.Builder
	script.WriteString("#!/bin/bash\n")
	script.WriteString(fmt.Sprintf("# R2Go2 Backup - %s\n", time.Now().Format("2006-01-02 15:04:05")))
	script.WriteString("#\n")
	script.WriteString("# Usage: source this file to export environment variables\n")
	script.WriteString("#\n\n")

	profiles := bm.configMgr.ListProfiles()
	for _, name := range profiles {
		profile, err := bm.configMgr.GetProfile(name)
		if err != nil {
			continue
		}

		script.WriteString(fmt.Sprintf("# Profile: %s\n", name))
		if profile.Description != "" {
			script.WriteString(fmt.Sprintf("# %s\n", profile.Description))
		}
		script.WriteString(fmt.Sprintf("export CLOUDFLARE_ACCOUNT_ID_%s=\"%s\"\n", strings.ToUpper(name), profile.AccountID))
		if profile.APIToken != "" {
			script.WriteString(fmt.Sprintf("export CLOUDFLARE_API_TOKEN_%s=\"%s\"\n", strings.ToUpper(name), profile.APIToken))
		}
		if profile.Endpoint != "" {
			script.WriteString(fmt.Sprintf("export R2_ENDPOINT_%s=\"%s\"\n", strings.ToUpper(name), profile.Endpoint))
		}
		if profile.AccessKey != "" {
			script.WriteString(fmt.Sprintf("export AWS_ACCESS_KEY_ID_%s=\"%s\"\n", strings.ToUpper(name), profile.AccessKey))
		}
		if profile.Region != "" {
			script.WriteString(fmt.Sprintf("export AWS_REGION_%s=\"%s\"\n", strings.ToUpper(name), profile.Region))
		}
		script.WriteString("\n")
	}

	if err := os.WriteFile(filepath, []byte(script.String()), 0600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	PrintSuccess("✅ Backup completed successfully!")
	fmt.Printf("Backup saved to: %s\n", Info(filepath))
	PrintInfo("💡 Usage: source " + filepath)

	return nil
}

// ShowRestoreInterface displays the restore interface
func (bm *BackupManager) ShowRestoreInterface() error {
	ClearScreen()
	fmt.Println()
	fmt.Println(Bold("📂 Restore Profiles"))
	fmt.Println(strings.Repeat("─", 30))
	fmt.Println()

	fmt.Print("Enter backup file path: ")
	var filepath string
	fmt.Scanln(&filepath)
	filepath = strings.TrimSpace(filepath)

	if filepath == "" {
		// Try to find latest backup
		homeDir, _ := os.UserHomeDir()
		latestBackup := bm.findLatestBackup(homeDir)
		if latestBackup != "" {
			fmt.Printf("Found latest backup: %s\n", Info(latestBackup))
			if ConfirmYesNo("Use this backup file?", true) {
				filepath = latestBackup
			} else {
				PrintInfo("Restore cancelled.")
				return nil
			}
		} else {
			PrintError("No backup file specified and no backups found.")
			return fmt.Errorf("no backup file")
		}
	}

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		PrintError("Backup file not found: %s", filepath)
		return fmt.Errorf("backup file not found")
	}

	// Determine file type
	if strings.HasSuffix(filepath, ".enc") {
		return bm.restoreFromEncrypted(filepath)
	} else if strings.HasSuffix(filepath, ".json") {
		return bm.restoreFromJSON(filepath)
	} else if strings.HasSuffix(filepath, ".sh") {
		return bm.restoreFromShell(filepath)
	} else {
		PrintError("Unsupported backup file format.")
		return fmt.Errorf("unsupported format")
	}
}

// restoreFromEncrypted restores from encrypted backup
func (bm *BackupManager) restoreFromEncrypted(filepath string) error {
	fmt.Println()
	fmt.Println(Bold("🔓 Encrypted Restore"))
	fmt.Println(strings.Repeat("─", 25))
	fmt.Println()

	fmt.Print("Enter backup password: ")
	password, err := readPassword()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	ShowSpinner("Reading and decrypting backup...", 2*Second)

	// Read and decrypt
	encryptedData, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	backupData, err := bm.decryptBackupData(encryptedData, password)
	if err != nil {
		PrintError("Failed to decrypt backup. Check your password.")
		return fmt.Errorf("decryption failed: %w", err)
	}

	return bm.processRestoreData(backupData, filepath, true)
}

// restoreFromJSON restores from JSON backup
func (bm *BackupManager) restoreFromJSON(filepath string) error {
	fmt.Println()
	fmt.Println(Bold("📄 JSON Restore"))
	fmt.Println(strings.Repeat("─", 20))
	fmt.Println()

	ShowSpinner("Reading JSON backup...", 1*Second)

	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	var backupData BackupData
	if err := json.Unmarshal(data, &backupData); err != nil {
		return fmt.Errorf("failed to parse backup file: %w", err)
	}

	return bm.processRestoreData(&backupData, filepath, false)
}

// processRestoreData processes restore data and restores profiles
func (bm *BackupManager) processRestoreData(backupData *BackupData, filepath string, hasTokens bool) error {
	fmt.Printf("📋 Found %d profile(s) in backup:\n", len(backupData.Profiles))
	for name, profile := range backupData.Profiles {
		fmt.Printf("  • %s %s\n", FormatProfileName(name, profile.Description), getProfileIcon(profile.Description))
	}

	fmt.Println()
	fmt.Println("Select restore option:")
	fmt.Println("  [1] All profiles")
	fmt.Println("  [2] Select individual profiles")
	fmt.Printf("Choice [1]: ")

	var choice string
	fmt.Scanln(&choice)
	choice = strings.TrimSpace(choice)

	var profilesToRestore []string

	switch choice {
	case "1", "":
		for name := range backupData.Profiles {
			profilesToRestore = append(profilesToRestore, name)
		}
	case "2":
		profileList := make([]string, 0, len(backupData.Profiles))
		for name := range backupData.Profiles {
			profileList = append(profileList, name)
		}

		fmt.Println("Select profiles to restore (comma-separated numbers):")
		for i, name := range profileList {
			profile := backupData.Profiles[name]
			fmt.Printf("  [%d] %s %s\n", i+1, name, getProfileIcon(profile.Description))
		}

		fmt.Print("Selection: ")
		var selection string
		fmt.Scanln(&selection)
		selection = strings.TrimSpace(selection)

		numbers := strings.Split(selection, ",")
		for _, num := range numbers {
			if idx, err := strconv.Atoi(strings.TrimSpace(num)); err == nil {
				if idx >= 1 && idx <= len(profileList) {
					profilesToRestore = append(profilesToRestore, profileList[idx-1])
				}
			}
		}

	default:
		PrintError("Invalid selection.")
		return fmt.Errorf("invalid choice")
	}

	if len(profilesToRestore) == 0 {
		PrintInfo("No profiles selected for restore.")
		return nil
	}

	fmt.Println()
	fmt.Printf("Will restore %d profile(s): %s\n", len(profilesToRestore), strings.Join(profilesToRestore, ", "))

	if !ConfirmYesNo("Continue with restore?", true) {
		return nil
	}

	// Restore profiles
	ShowSpinner("Restoring profiles...", 2*Second)

	for _, name := range profilesToRestore {
		backupProfile := backupData.Profiles[name]

		// Check if profile already exists
		if bm.configMgr.ProfileExists(name) {
			PrintWarning("Profile '%s' already exists.", name)
			if !ConfirmYesNo(fmt.Sprintf("Overwrite profile '%s'?", name), false) {
				continue
			}
		}

		// Create profile (API tokens need to be re-entered for security)
		profile := &config.Profile{
			Name:        backupProfile.Name,
			Description: backupProfile.Description,
			AccountID:   backupProfile.AccountID,
			Endpoint:    backupProfile.Endpoint,
			AccessKey:   backupProfile.AccessKey,
			Region:      backupProfile.Region,
			// APIToken is not restored for security
		}

		if err := bm.configMgr.SetProfile(profile); err != nil {
			PrintError("Failed to restore profile '%s': %v", name, err)
			continue
		}

		if !hasTokens {
			fmt.Printf("🔑 Enter API token for profile '%s': ", name)
			token, err := readPassword()
			if err != nil {
				PrintWarning("Failed to read token for profile '%s'", name)
				continue
			}

			profile.APIToken = token
			if err := bm.configMgr.SetProfile(profile); err != nil {
				PrintWarning("Failed to save token for profile '%s'", name)
			}
		}

		PrintSuccess("✅ Restored profile: %s", name)
	}

	fmt.Println()
	PrintSuccess("🎉 Restore completed successfully!")

	if !hasTokens {
		PrintInfo("💡 Remember to test your restored profiles with 'r2go2 bucket list'")
	}

	return nil
}

// Helper functions

func (bm *BackupManager) encryptBackupData(data *BackupData, password string) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// Derive key from password
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	key := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, jsonData, nil)

	// Prepend salt
	result := append(salt, ciphertext...)

	return result, nil
}

func (bm *BackupManager) decryptBackupData(encryptedData []byte, password string) (*BackupData, error) {
	if len(encryptedData) < 16 {
		return nil, fmt.Errorf("invalid encrypted data")
	}

	// Extract salt
	salt := encryptedData[:16]
	ciphertext := encryptedData[16:]

	// Derive key
	key := pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	jsonData, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	var backupData BackupData
	if err := json.Unmarshal(jsonData, &backupData); err != nil {
		return nil, err
	}

	return &backupData, nil
}

func (bm *BackupManager) findLatestBackup(dir string) string {
	files, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var latestFile string
	var latestTime time.Time

	for _, file := range files {
		name := file.Name()
		if (strings.HasPrefix(name, ".r2go2-backup-") &&
			(strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".enc"))) {

			info, err := file.Info()
			if err != nil {
				continue
			}

			if info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latestFile = filepath.Join(dir, name)
			}
		}
	}

	return latestFile
}

func readPassword() (string, error) {
	// In a real implementation, this would use term.NewTerminal or similar
	// to read password without echo. For now, we'll use a simple approach
	var password string
	fmt.Scanln(&password)
	return strings.TrimSpace(password), nil
}

func (bm *BackupManager) restoreFromShell(filepath string) error {
	PrintInfo("Shell script backups cannot be automatically restored.")
	PrintInfo("Please manually source the file to export environment variables:")
	fmt.Printf("  source %s\n", Info(filepath))
	return nil
}