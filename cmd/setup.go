/*
Package cmd provides the setup command for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	"github.com/CosmoLabs-org/cosmoflare/internal/interactive"
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard for R2Go2 configuration",
	Long: `Interactive setup wizard for R2Go2 configuration.

This command guides you through the complete setup process with:
• Secure password-style input for API tokens
• Smart auto-detection of account information
• Real-time token validation
• Visual progress indicators
• Beautiful error handling and recovery guidance
• Connection testing and verification

The wizard creates a secure configuration profile that you can use
immediately with all R2Go2 commands.

Example:
  cosmoflare setup                    # Interactive setup
  cosmoflare setup --profile=prod     # Create specific profile
  cosmoflare setup --quiet            # Non-interactive setup with env vars`,
	RunE: runSetup,
}

var (
	setupProfile     string
	setupQuiet       bool
	setupSkipTest    bool
	setupAutoDetect  bool
	setupSwitch      bool
	setupWelcome     bool
	setupBackup      bool
	setupRestore     bool
)

func init() {
	rootCmd.AddCommand(setupCmd)

	setupCmd.Flags().StringVar(&setupProfile, "profile", "", "Profile name to create (default: 'production')")
	setupCmd.Flags().BoolVar(&setupQuiet, "quiet", false, "Run in quiet mode (use environment variables)")
	setupCmd.Flags().BoolVar(&setupSkipTest, "skip-test", false, "Skip connection testing")
	setupCmd.Flags().BoolVar(&setupAutoDetect, "auto-detect", true, "Auto-detect account information")
	setupCmd.Flags().BoolVar(&setupSwitch, "switch", false, "Switch profiles interactively")
	setupCmd.Flags().BoolVar(&setupWelcome, "welcome", false, "Show welcome message for new users")
	setupCmd.Flags().BoolVar(&setupBackup, "backup", false, "Backup profiles")
	setupCmd.Flags().BoolVar(&setupRestore, "restore", false, "Restore profiles")
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Handle profile switching
	if setupSwitch {
		profileManager, err := interactive.NewProfileManager()
		if err != nil {
			return fmt.Errorf("failed to create profile manager: %w", err)
		}
		return profileManager.ShowProfileSwitcher()
	}

	// Handle welcome message
	if setupWelcome {
		interactive.ShowWelcomeForNewUser()
		return nil
	}

	// Handle backup
	if setupBackup {
		backupManager, err := interactive.NewBackupManager()
		if err != nil {
			return fmt.Errorf("failed to create backup manager: %w", err)
		}
		return backupManager.ShowBackupInterface()
	}

	// Handle restore
	if setupRestore {
		backupManager, err := interactive.NewBackupManager()
		if err != nil {
			return fmt.Errorf("failed to create backup manager: %w", err)
		}
		return backupManager.ShowRestoreInterface()
	}

	// Initialize the setup wizard
	wizard := interactive.NewSetupWizard()
	wizard.Quiet = setupQuiet

	if setupQuiet {
		return runQuietSetup(wizard)
	}

	return runInteractiveSetup(wizard)
}

func runInteractiveSetup(wizard *interactive.SetupWizard) error {
	// Show welcome screen
	wizard.Welcome()

	// Step 1: Authentication Method
	authMethod, err := wizard.Step1_AuthMethod()
	if err != nil {
		return fmt.Errorf("authentication selection failed: %w", err)
	}

	var apiToken string
	switch authMethod {
	case "env":
		// Use environment variables
		apiToken = ""
		interactive.ShowSpinner("Checking environment variables...", 2*time.Second)

		token := os.Getenv("CLOUDFLARE_API_TOKEN")
		if token == "" {
			printError("CLOUDFLARE_API_TOKEN environment variable not found")
			return fmt.Errorf("environment variable CLOUDFLARE_API_TOKEN is not set")
		}

		interactive.ShowSpinner("Validating token...", 3*time.Second)

	case "api_token", "service_key":
		// Get token interactively
		interactive.ShowSpinner("Preparing secure input...", 1*time.Second)
		apiToken, err = wizard.Step2_APIToken()
		if err != nil {
			return fmt.Errorf("token input failed: %w", err)
		}

		// Validate the token with visual feedback
		interactive.ShowSpinner("Validating API token...", 3*time.Second)
		tokenInfo, err := interactive.ValidateAPIToken(apiToken)
		if err != nil {
			interactive.HandleError(interactive.AuthError("Token validation", err))

			// Ask if user wants to continue anyway
			if !interactive.ConfirmYesNo("Continue with this token anyway?", false) {
				return fmt.Errorf("setup cancelled due to invalid token")
			}
		} else {
			interactive.SuccessMessage("Token validated successfully!",
				fmt.Sprintf("Account ID: %s", config.MaskAccountID(tokenInfo.AccountID)))
		}
	}

	// Step 3: Account Information
	interactive.ShowSpinner("Preparing account setup...", 1*time.Second)
	var accountID, accountName string

	if setupAutoDetect && apiToken != "" {
		accountID, accountName, _ = wizard.Step3_AccountInfo(apiToken)
	} else {
		accountID, accountName, _ = wizard.Step3_AccountInfo("")
	}

	// Validate account ID
	if err := interactive.ValidateAccountID(accountID); err != nil {
		printError("Invalid account ID: %v", err)
		return fmt.Errorf("account ID validation failed: %w", err)
	}

	// Step 4: Profile Setup
	interactive.ShowSpinner("Preparing profile configuration...", 1*time.Second)
	profileName, description, err := wizard.Step4_ProfileSetup()
	if err != nil {
		return fmt.Errorf("profile setup failed: %w", err)
	}

	// Override with command line profile if provided
	if setupProfile != "" {
		profileName = setupProfile
	}

	// Test connection if not skipped
	if !setupSkipTest && apiToken != "" {
		interactive.ShowSpinner("Testing connection to Cloudflare R2...", 3*time.Second)
		if err := interactive.TestConnection(accountID, apiToken); err != nil {
			interactive.HandleError(interactive.NetworkError("Connection test", err))

			if !interactive.ConfirmYesNo("Continue despite connection failure?", false) {
				return fmt.Errorf("setup cancelled due to connection failure")
			}
		} else {
			interactive.SuccessMessage("Connection test passed!", "Your credentials are working correctly.")
		}
	}

	// Save configuration
	interactive.ShowSpinner("Saving configuration...", 2*time.Second)

	configMgr, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	profile := &config.Profile{
		Name:        profileName,
		Description: description,
		AccountID:   accountID,
		APIToken:    apiToken,
		Region:      "auto",
	}

	if err := configMgr.SetProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	// Set as current profile
	if err := configMgr.SetCurrent(profileName); err != nil {
		printWarning("Failed to set profile as current: %v", err)
	}

	// Show completion screen
	wizard.Complete(profileName, accountName)

	// Additional helpful information
	printInfo("📚 Next steps:")
	printInfo("  cosmoflare config list                    # View all profiles")
	printInfo("  cosmoflare bucket list                    # List existing buckets")
	printInfo("  cosmoflare bucket create my-bucket        # Create a new bucket")
	printInfo("  cosmoflare upload ./file.txt my-bucket    # Upload files")
	printInfo("")
	printInfo("📖 For more help: cosmoflare --help")
	printInfo("🌐 Documentation: https://github.com/CosmoLabs-org/cosmoflare")

	return nil
}

func runQuietSetup(wizard *interactive.SetupWizard) error {
	printInfo("🚀 Running quiet setup using environment variables...")

	// Check required environment variables
	token := os.Getenv("CLOUDFLARE_API_TOKEN")
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")

	if token == "" {
		return fmt.Errorf("CLOUDFLARE_API_TOKEN environment variable is required in quiet mode")
	}

	if accountID == "" {
		return fmt.Errorf("CLOUDFLARE_ACCOUNT_ID environment variable is required in quiet mode")
	}

	// Use defaults
	profileName := "default"
	if setupProfile != "" {
		profileName = setupProfile
	}
	description := "Auto-created profile via quiet setup"

	// Validate inputs
	interactive.ShowSpinner("Validating configuration...", 2*time.Second)

	if err := interactive.ValidateAccountID(accountID); err != nil {
		return fmt.Errorf("account ID validation failed: %w", err)
	}

	tokenInfo, err := interactive.ValidateAPIToken(token)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	printSuccess("✅ Environment variables validated")

	// Test connection if not skipped
	if !setupSkipTest {
		interactive.ShowSpinner("Testing connection...", 3*time.Second)
		if err := interactive.TestConnection(accountID, token); err != nil {
			return fmt.Errorf("connection test failed: %w", err)
		}
		printSuccess("✅ Connection test passed")
	}

	// Save configuration
	interactive.ShowSpinner("Saving configuration...", 2*time.Second)

	configMgr, err := config.NewConfigManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	profile := &config.Profile{
		Name:        profileName,
		Description: description,
		AccountID:   accountID,
		APIToken:    token,
		Region:      "auto",
	}

	if err := configMgr.SetProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	if err := configMgr.SetCurrent(profileName); err != nil {
		return fmt.Errorf("failed to set current profile: %w", err)
	}

	printSuccess("✅ Setup completed successfully!")
	printInfo("Profile '%s' has been created and set as current", profileName)

	if tokenInfo.AccountName != "" {
		printInfo("Account: %s", tokenInfo.AccountName)
	}

	return nil
}