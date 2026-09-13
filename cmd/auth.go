/*
Package cmd provides authentication commands for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	"github.com/CosmoLabs-org/cosmoflare/internal/utils"
	"github.com/spf13/cobra"
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication with Cloudflare",
	Long: `Authentication management for Cloudflare API access.

Commands:
  login     Authenticate with Cloudflare
  rotate    Rotate API tokens
  status    Show current authentication status
  logout    Clear current authentication

R2Go2 supports multiple authentication methods:
- API Token authentication (recommended)
- Service Key authentication
- Environment variables`,
}

// authLoginCmd represents the auth login command
var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Cloudflare",
	Long: `Login to Cloudflare and obtain API credentials.

Interactive mode will guide you through:
1. Choosing authentication method
2. Providing required credentials
3. Testing the connection
4. Saving to a profile

You can also provide credentials directly via flags for automation.

Example:
  cosmoflare auth login --token=your_api_token --account-id=your_account_id

  cosmoflare auth login --profile=production --interactive`,
	RunE: runAuthLogin,
}

// authRotateCmd represents the auth rotate command
var authRotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Rotate API tokens",
	Long: `Rotate API tokens for enhanced security.

This command helps you:
1. Generate new API tokens
2. Update existing profiles
3. Revoke old tokens
4. Test new credentials

Example:
  cosmoflare auth rotate --profile=production`,
	RunE: runAuthRotate,
}

// authStatusCmd represents the auth status command
var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	Long: `Display information about the current authentication status.

Shows:
- Current profile
- Account information
- Token details (masked)
- Connection status
- Permissions scope`,
	RunE: runAuthStatus,
}

// authLogoutCmd represents the auth logout command
var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear current authentication",
	Long: `Clear current authentication credentials.
This removes stored tokens from the current session.

Note: This only clears in-memory credentials.
Profile configurations remain stored in ~/.cosmoflare/config.yaml`,
	RunE: runAuthLogout,
}

var (
	authProfile     string
	authToken       string
	authAccountID   string
	authEmail       string
	authMethod      string
	authInteractive bool
	authScope       string
)

func init() {
	rootCmd.AddCommand(authCmd)

	// Add subcommands
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authRotateCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)

	// Flags for auth login
	authLoginCmd.Flags().StringVar(&authProfile, "profile", "", "Save credentials to this profile")
	authLoginCmd.Flags().StringVar(&authToken, "token", "", "Cloudflare API token")
	authLoginCmd.Flags().StringVar(&authAccountID, "account-id", "", "Cloudflare Account ID")
	authLoginCmd.Flags().StringVar(&authEmail, "email", "", "Cloudflare account email")
	authLoginCmd.Flags().StringVar(&authMethod, "method", "token", "Authentication method (token, key)")
	authLoginCmd.Flags().BoolVar(&authInteractive, "interactive", true, "Interactive mode")
	authLoginCmd.Flags().StringVar(&authScope, "scope", "", "Token scope permissions")

	// Flags for auth rotate
	authRotateCmd.Flags().StringVar(&authProfile, "profile", "", "Profile to rotate token for")
	authRotateCmd.Flags().Bool("revoke-old", false, "Revoke the old token after rotation")
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	printInfo("🔐 Cloudflare Authentication")

	// Get authentication method
	method, _ := cmd.Flags().GetString("method")
	interactive, _ := cmd.Flags().GetBool("interactive")

	if interactive {
		method = selectAuthMethod()
	}

	var credentials *AuthCredentials
	var err error

	switch method {
	case "token":
		credentials, err = getAPITokenCredentials(cmd)
	case "key":
		credentials, err = getServiceKeyCredentials(cmd)
	default:
		return fmt.Errorf("unsupported authentication method: %s", method)
	}

	if err != nil {
		return fmt.Errorf("failed to get credentials: %w", err)
	}

	// Test credentials
	if err := testCredentials(credentials); err != nil {
		return fmt.Errorf("credential validation failed: %w", err)
	}

	// Save to profile if requested
	profileName, _ := cmd.Flags().GetString("profile")
	if profileName != "" {
		if err := saveCredentialsToProfile(profileName, credentials); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}
		printSuccess("Credentials saved to profile '%s'", profileName)
	}

	printSuccess("✅ Authentication successful!")
	printInfo("Account ID: %s", utils.MaskAccountID(credentials.AccountID))
	printInfo("Authentication method: %s", method)

	return nil
}

func runAuthRotate(cmd *cobra.Command, args []string) error {
	profileName, _ := cmd.Flags().GetString("profile")
	if profileName == "" {
		return fmt.Errorf("profile name is required. Use --profile to specify")
	}

	printInfo("🔄 Rotating API token for profile: %s", profileName)

	// Get current profile
	configMgr, err := getConfigManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	profile, err := configMgr.GetProfile(profileName)
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	printInfo("Current token (masked): %s", config.MaskKey(profile.APIToken))

	// BUG-044: capture the pre-rotation token BEFORE the profile is
	// overwritten — --revoke-old must revoke this token, never the new one.
	oldToken := profile.APIToken

	// Generate new token
	printInfo("Generating new API token...")
	newToken, err := generateNewToken(profile)
	if err != nil {
		return fmt.Errorf("failed to generate new token: %w", err)
	}

	// Test new token
	newCredentials := &AuthCredentials{
		APIToken:  newToken,
		AccountID: profile.AccountID,
	}

	if err := testCredentials(newCredentials); err != nil {
		return fmt.Errorf("new token validation failed: %w", err)
	}

	// Save new token
	profile.APIToken = newToken
	if err := configMgr.SetProfile(profile); err != nil {
		return fmt.Errorf("failed to save updated profile: %w", err)
	}

	printSuccess("✅ Token rotation successful!")
	printInfo("New token saved to profile '%s'", profileName)

	// Revoke old token if requested
	revokeOld, _ := cmd.Flags().GetBool("revoke-old")
	if revokeOld {
		printInfo("Revoking old token...")
		if err := revokeOldToken(oldToken); err != nil {
			printWarning("Failed to revoke old token: %v", err)
		} else {
			printSuccess("Old token revoked successfully")
		}
	}

	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	printInfo("🔍 Authentication Status")

	// Check environment variables
	envToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	envAccountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")

	if envToken != "" && envAccountID != "" {
		printInfo("Environment variables configured:")
		printInfo("  Account ID: %s", utils.MaskAccountID(envAccountID))
		printInfo("  API Token: %s", config.MaskKey(envToken))
	}

	// Check current profile
	configMgr, err := getConfigManager()
	if err == nil {
		if currentProfile, err := configMgr.GetCurrent(); err == nil {
			printInfo("Current profile: %s", currentProfile.Name)
			if currentProfile.Description != "" {
				printInfo("  Description: %s", currentProfile.Description)
			}
			printInfo("  Account ID: %s", utils.MaskAccountID(currentProfile.AccountID))
			printInfo("  API Token: %s", config.MaskKey(currentProfile.APIToken))
		} else {
			printInfo("No current profile set")
		}
	}

	// Test connection
	printInfo("Testing connection...")
	client, err := newClientFromEnv()
	if err != nil {
		printError("Failed to create client: %v", err)
		return err
	}

	if err := client.TestConnection(context.Background()); err != nil {
		printError("Connection test failed: %v", err)
		return err
	}

	printSuccess("✅ Connection successful!")
	printInfo("Account details available")

	// Show token scope if available
	if envToken != "" || (configMgr != nil) {
		printInfo("Token permissions: R2 management (assumed)")
		printInfo("Note: Use Cloudflare dashboard to view exact token permissions")
	}

	return nil
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	printInfo("🚪 Logging out...")

	// Clear environment variables in current process
	os.Unsetenv("CLOUDFLARE_API_TOKEN")
	os.Unsetenv("CLOUDFLARE_ACCOUNT_ID")
	os.Unsetenv("R2_ENDPOINT")
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Unsetenv("AWS_REGION")

	printSuccess("Cleared environment variables")

	// Note: We don't delete profile configurations as they may be needed later
	printInfo("Note: Profile configurations remain saved in ~/.cosmoflare/config.yaml")
	printInfo("Use 'cosmoflare auth login' to re-authenticate")

	return nil
}

// Helper types and functions

type AuthCredentials struct {
	APIToken    string
	AccountID   string
	Email       string
	AccessKey   string
	SecretKey   string
	Method      string
	Scope       string
	ExpiresAt   time.Time
}

func selectAuthMethod() string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nSelect authentication method:")
	fmt.Println("1. API Token (recommended)")
	fmt.Println("2. Service Key (Global API Key)")
	fmt.Print("Choose method [1-2]: ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1", "token":
		return "token"
	case "2", "key":
		return "key"
	default:
		printWarning("Invalid choice, defaulting to API Token")
		return "token"
	}
}

func getAPITokenCredentials(cmd *cobra.Command) (*AuthCredentials, error) {
	// Try to get from flags first
	token, _ := cmd.Flags().GetString("token")
	accountID, _ := cmd.Flags().GetString("account-id")

	// If not provided, try interactive or environment
	interactive, _ := cmd.Flags().GetBool("interactive")
	if interactive && token == "" {
		token = promptForAPIToken()
	}
	if token == "" {
		token = os.Getenv("CLOUDFLARE_API_TOKEN")
	}

	if interactive && accountID == "" {
		accountID = promptForAccountID()
	}
	if accountID == "" {
		accountID = os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	}

	if token == "" {
		return nil, fmt.Errorf("API token is required")
	}
	if accountID == "" {
		return nil, fmt.Errorf("Account ID is required")
	}

	return &AuthCredentials{
		APIToken:  strings.TrimSpace(token),
		AccountID: strings.TrimSpace(accountID),
		Method:    "token",
	}, nil
}

func getServiceKeyCredentials(cmd *cobra.Command) (*AuthCredentials, error) {
	email, _ := cmd.Flags().GetString("email")
	interactive, _ := cmd.Flags().GetBool("interactive")

	if interactive && email == "" {
		email = promptForEmail()
	}
	if email == "" {
		email = os.Getenv("CLOUDFLARE_EMAIL")
	}

	if email == "" {
		return nil, fmt.Errorf("email is required for service key authentication")
	}

	printWarning("Service key authentication requires Global API Key")
	printWarning("This is less secure than API tokens")
	printWarning("Consider using API tokens instead")

	return &AuthCredentials{
		Email:  strings.TrimSpace(email),
		Method: "key",
	}, nil
}

func promptForAPIToken() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Cloudflare API Token: ")
	token, _ := reader.ReadString('\n')
	return strings.TrimSpace(token)
}

func promptForAccountID() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Cloudflare Account ID: ")
	accountID, _ := reader.ReadString('\n')
	return strings.TrimSpace(accountID)
}

func promptForEmail() string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Cloudflare Email: ")
	email, _ := reader.ReadString('\n')
	return strings.TrimSpace(email)
}

// Test seams (BUG-044): these are package-level so tests can stub token
// generation/revocation/validation without network access.
var testCredentials = func(credentials *AuthCredentials) error {
	client, err := cosmoflare.NewClient(
		cosmoflare.WithAccountID(credentials.AccountID),
		cosmoflare.WithAPIToken(credentials.APIToken),
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	return client.TestConnection(context.Background())
}

func newClientFromEnv() (cosmoflare.R2Client, error) {
	accountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	if apiToken == "" || accountID == "" {
		return nil, fmt.Errorf("missing CLOUDFLARE_API_TOKEN or CLOUDFLARE_ACCOUNT_ID environment variables")
	}

	return cosmoflare.NewClient(
		cosmoflare.WithAccountID(accountID),
		cosmoflare.WithAPIToken(apiToken),
	)
}

func saveCredentialsToProfile(profileName string, credentials *AuthCredentials) error {
	configMgr, err := getConfigManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	profile := &config.Profile{
		Name:      profileName,
		AccountID: credentials.AccountID,
		APIToken:  credentials.APIToken,
		Region:    "auto",
	}

	return configMgr.SetProfile(profile)
}

var generateNewToken = func(profile *config.Profile) (string, error) {
	// This is a placeholder implementation
	// In practice, you would use the Cloudflare API to generate new tokens
	// For now, return an error indicating manual token generation is required
	return "", fmt.Errorf("automatic token generation not implemented. Please generate a new token manually in the Cloudflare dashboard")
}

var revokeOldToken = func(token string) error {
	// This is a placeholder implementation
	// In practice, you would use the Cloudflare API to revoke tokens
	// For now, return an error indicating manual revocation is required
	return fmt.Errorf("automatic token revocation not implemented. Please revoke the old token manually in the Cloudflare dashboard")
}