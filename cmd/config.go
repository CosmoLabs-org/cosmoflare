/*
Package cmd provides configuration commands for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"context"
	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	"github.com/CosmoLabs-org/cosmoflare/internal/utils"
)

var (
	configProfile     string
	configDescription string
	configAccountID   string
	configAPIToken    string
	configEndpoint    string
	configAccessKey   string
	configSecretKey   string
	configRegion      string
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage R2Go2 configuration profiles",
	Long: `Configuration management for R2Go2 profiles.

Commands:
  init      Initialize configuration
  validate  Validate current configuration
  list      List all profiles
  show      Show profile details
  set       Create or update a profile
  delete    Delete a profile
  switch    Switch current profile
  export    Export profile as environment variables

Profiles are stored in ~/.cosmoflare/config.yaml with secure permissions.`,
}

// configInitCmd represents the config init command
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize R2Go2 configuration",
	Long: `Initialize the R2Go2 configuration file with default settings.
This will create ~/.cosmoflare/config.yaml if it doesn't exist.`,
	RunE: runConfigInit,
}

// configValidateCmd represents the config validate command
var configValidateCmd = &cobra.Command{
	Use:   "validate [profile-name]",
	Short: "Validate a profile configuration",
	Long: `Validate the specified profile configuration.
If no profile name is provided, validates the current profile.

This command checks:
- Profile exists
- Required fields are present
- Account ID format is valid
- API token format is reasonable
- Connection to Cloudflare API works`,
	RunE: runConfigValidate,
}

// configListCmd represents the config list command
var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration profiles",
	Long: `List all configured profiles with their basic information.
Sensitive information like API tokens and secret keys are masked.`,
	RunE: runConfigList,
}

// configShowCmd represents the config show command
var configShowCmd = &cobra.Command{
	Use:   "show [profile-name]",
	Short: "Show profile details",
	Long: `Show detailed information about a profile.
If no profile name is provided, shows the current profile.

Sensitive information like API tokens and secret keys are masked unless --show-secrets is used.`,
	RunE: runConfigShow,
}

// configSetCmd represents the config set command
var configSetCmd = &cobra.Command{
	Use:   "set [profile-name]",
	Short: "Create or update a profile",
	Long: `Create a new profile or update an existing one.

You can provide values via flags or interactively if --interactive is used.

Example:
  cosmoflare config set my-profile --account-id=1234567890abcdef1234567890abcdef --interactive

  cosmoflare config set my-profile \
    --account-id=1234567890abcdef1234567890abcdef \
    --api-token=your_api_token_here \
    --description="Production account"`,
	RunE: runConfigSet,
}

// configDeleteCmd represents the config delete command
var configDeleteCmd = &cobra.Command{
	Use:   "delete [profile-name]",
	Short: "Delete a profile",
	Long: `Delete a profile from the configuration.
You cannot delete the currently active profile.`,
	RunE: runConfigDelete,
}

// configSwitchCmd represents the config switch command
var configSwitchCmd = &cobra.Command{
	Use:   "switch [profile-name]",
	Short: "Switch to a different profile",
	Long: `Set the specified profile as the current active profile.
All subsequent commands will use this profile unless overridden.`,
	RunE: runConfigSwitch,
}

// configExportCmd represents the config export command
var configExportCmd = &cobra.Command{
	Use:   "export [profile-name]",
	Short: "Export profile as environment variables",
	Long: `Export a profile's configuration as shell environment variables.
This is useful for:
- Setting up environment in scripts
- Debugging configuration
- Using with other tools

Example:
  eval $(cosmoflare config export my-profile)`,
	RunE: runConfigExport,
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configDeleteCmd)
	configCmd.AddCommand(configSwitchCmd)
	configCmd.AddCommand(configExportCmd)

	// Flags for config set command
	configSetCmd.Flags().StringVar(&configDescription, "description", "", "Profile description")
	configSetCmd.Flags().StringVar(&configAccountID, "account-id", "", "Cloudflare Account ID")
	configSetCmd.Flags().StringVar(&configAPIToken, "api-token", "", "Cloudflare API Token")
	configSetCmd.Flags().StringVar(&configEndpoint, "endpoint", "", "R2 endpoint URL")
	configSetCmd.Flags().StringVar(&configAccessKey, "access-key", "", "S3 Access Key ID")
	configSetCmd.Flags().StringVar(&configSecretKey, "secret-key", "", "S3 Secret Access Key")
	configSetCmd.Flags().StringVar(&configRegion, "region", "auto", "AWS Region")
	configSetCmd.Flags().Bool("interactive", true, "Interactive mode for missing values")
	configSetCmd.Flags().Bool("test-connection", false, "Test connection after setting profile")

	// Flags for config show command
	configShowCmd.Flags().Bool("show-secrets", false, "Show sensitive information (API tokens, secret keys)")

	// Mark required flags
	configSetCmd.MarkFlagRequired("account-id")
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	printInfo("Initializing R2Go2 configuration...")

	configMgr, err := getConfigManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	// Check if config already exists
	if _, err := os.Stat(configMgr.GetConfigPath()); err == nil {
		printWarning("Configuration file already exists at %s", configMgr.GetConfigPath())
		printInfo("Use 'cosmoflare config list' to see existing profiles")
		return nil
	}

	// Create default profile interactively
	profile, err := createProfileInteractive("default")
	if err != nil {
		return fmt.Errorf("failed to create default profile: %w", err)
	}

	if err := configMgr.SetProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	printSuccess("Configuration initialized successfully!")
	printInfo("Config file: %s", configMgr.GetConfigPath())
	printInfo("Default profile: %s", profile.Name)
	printInfo("Use 'cosmoflare config list' to see all profiles")

	return nil
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	var profileName string
	if len(args) > 0 {
		profileName = args[0]
	}

	configMgr, err := getConfigManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	// Get profile to validate
	var profile *config.Profile
	if profileName != "" {
		profile, err = configMgr.GetProfile(profileName)
	} else {
		profile, err = configMgr.GetCurrent()
	}

	if err != nil {
		printError("Profile validation failed: %v", err)
		return err
	}

	printInfo("Validating profile: %s", profile.Name)

	// Basic validation
	if err := configMgr.ValidateProfile(profile); err != nil {
		printError("Basic validation failed: %v", err)
		return err
	}

	// Test connection to Cloudflare API
	client, err := cosmoflare.NewClient(
		cosmoflare.WithAccountID(profile.AccountID),
		cosmoflare.WithAPIToken(profile.APIToken),
	)
	if err != nil {
		printError("Failed to create client: %v", err)
		return err
	}

	if err := client.TestConnection(context.Background()); err != nil {
		printError("Connection test failed: %v", err)
		return err
	}

	printSuccess("Profile '%s' is valid and connected!", profile.Name)
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	configMgr, err := getConfigManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	profiles := configMgr.ListProfiles()
	currentProfile, _ := configMgr.GetCurrent()

	if len(profiles) == 0 {
		printInfo("No profiles configured. Use 'cosmoflare config init' to get started.")
		return nil
	}

	if JSONOutput {
		data := map[string]interface{}{
			"current":  currentProfile,
			"profiles": configMgr.SanitizeForOutput().Profiles,
		}
		return printJSON(data)
	}

	// Tabular output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROFILE\tCURRENT\tACCOUNT ID\tDESCRIPTION")
	for _, name := range profiles {
		profile, _ := configMgr.GetProfile(name)
		current := ""
		if currentProfile != nil && currentProfile.Name == name {
			current = "✓"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			name,
			current,
			utils.MaskAccountID(profile.AccountID),
			profile.Description,
		)
	}
	w.Flush()

	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	var profileName string
	if len(args) > 0 {
		profileName = args[0]
	}

	configMgr, err := getConfigManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	// Get profile to show
	var profile *config.Profile
	if profileName != "" {
		profile, err = configMgr.GetProfile(profileName)
	} else {
		profile, err = configMgr.GetCurrent()
	}

	if err != nil {
		printError("Failed to get profile: %v", err)
		return err
	}

	if JSONOutput {
		showSecrets, _ := cmd.Flags().GetBool("show-secrets")
		if !showSecrets {
			profile.APIToken = ""
			profile.SecretKey = ""
		}
		return printJSON(profile)
	}

	// Human-readable output
	printInfo("Profile: %s", profile.Name)
	if profile.Description != "" {
		printInfo("Description: %s", profile.Description)
	}
	printInfo("Account ID: %s", utils.MaskAccountID(profile.AccountID))

	showSecrets, _ := cmd.Flags().GetBool("show-secrets")
	if showSecrets && profile.APIToken != "" {
		printInfo("API Token: %s", profile.APIToken)
	} else if profile.APIToken != "" {
		printInfo("API Token: %s", config.MaskKey(profile.APIToken))
	}

	if profile.Endpoint != "" {
		printInfo("Endpoint: %s", profile.Endpoint)
	}
	if profile.AccessKey != "" {
		printInfo("Access Key: %s", config.MaskKey(profile.AccessKey))
	}
	if showSecrets && profile.SecretKey != "" {
		printInfo("Secret Key: %s", profile.SecretKey)
	} else if profile.SecretKey != "" {
		printInfo("Secret Key: %s", config.MaskKey(profile.SecretKey))
	}
	if profile.Region != "" {
		printInfo("Region: %s", profile.Region)
	}

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	var profileName string
	if len(args) > 0 {
		profileName = args[0]
	}

	if profileName == "" {
		return fmt.Errorf("profile name is required")
	}

	configMgr, err := getConfigManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	// Check if profile exists and load existing values
	profile, _ := configMgr.GetProfile(profileName)
	interactive, _ := cmd.Flags().GetBool("interactive")

	if profile == nil {
		profile = &config.Profile{Name: profileName}
	}

	// Set values from flags
	description, _ := cmd.Flags().GetString("description")
	accountID, _ := cmd.Flags().GetString("account-id")
	apiToken, _ := cmd.Flags().GetString("api-token")
	endpoint, _ := cmd.Flags().GetString("endpoint")
	accessKey, _ := cmd.Flags().GetString("access-key")
	secretKey, _ := cmd.Flags().GetString("secret-key")
	region, _ := cmd.Flags().GetString("region")

	if description != "" {
		profile.Description = description
	}
	if accountID != "" {
		profile.AccountID = accountID
	}
	if apiToken != "" {
		profile.APIToken = apiToken
	}
	if endpoint != "" {
		profile.Endpoint = endpoint
	}
	if accessKey != "" {
		profile.AccessKey = accessKey
	}
	if secretKey != "" {
		profile.SecretKey = secretKey
	}
	if region != "" {
		profile.Region = region
	}

	// Interactive mode for missing values
	if interactive {
		if err := fillProfileInteractively(profile); err != nil {
			return fmt.Errorf("interactive input failed: %w", err)
		}
	}

	// Validate profile
	if err := configMgr.ValidateProfile(profile); err != nil {
		return fmt.Errorf("profile validation failed: %w", err)
	}

	// Save profile
	if err := configMgr.SetProfile(profile); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	printSuccess("Profile '%s' saved successfully!", profileName)

	// Test connection if requested
	testConnection, _ := cmd.Flags().GetBool("test-connection")
	if testConnection {
		printInfo("Testing connection...")
		client, err := getAPIClient()
		if err != nil {
			printError("Failed to create client: %v", err)
			return err
		}

		if err := client.TestConnection(context.Background()); err != nil {
			printError("Connection test failed: %v", err)
			return err
		}

		printSuccess("Connection test passed!")
	}

	return nil
}

func runConfigDelete(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("profile name is required")
	}
	profileName := args[0]

	configMgr, err := getConfigManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	if !configMgr.ProfileExists(profileName) {
		return fmt.Errorf("profile '%s' does not exist", profileName)
	}

	// Confirm deletion
	if !DryRun {
		fmt.Printf("Are you sure you want to delete profile '%s'? [y/N]: ", profileName)
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			printInfo("Profile deletion cancelled")
			return nil
		}
	}

	if err := configMgr.DeleteProfile(profileName); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	if DryRun {
		printInfo("DRY RUN: Would delete profile '%s'", profileName)
	} else {
		printSuccess("Profile '%s' deleted successfully!", profileName)
	}

	return nil
}

func runConfigSwitch(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("profile name is required")
	}
	profileName := args[0]

	configMgr, err := getConfigManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	if !configMgr.ProfileExists(profileName) {
		return fmt.Errorf("profile '%s' does not exist", profileName)
	}

	if err := configMgr.SetCurrent(profileName); err != nil {
		return fmt.Errorf("failed to switch profile: %w", err)
	}

	printSuccess("Switched to profile '%s'", profileName)
	return nil
}

func runConfigExport(cmd *cobra.Command, args []string) error {
	var profileName string
	if len(args) > 0 {
		profileName = args[0]
	}

	configMgr, err := getConfigManager()
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	// Get profile to export
	if profileName != "" {
		_, err := configMgr.GetProfile(profileName)
		if err != nil {
			return fmt.Errorf("failed to get profile: %w", err)
		}

		export, err := configMgr.ExportProfile(profileName)
		if err != nil {
			return fmt.Errorf("failed to export profile: %w", err)
		}

		fmt.Print(export)
	} else {
		// Export current profile
		export, err := configMgr.ExportProfile("")
		if err != nil {
			return fmt.Errorf("failed to export current profile: %w", err)
		}

		fmt.Print(export)
	}

	return nil
}

// Helper functions

func getConfigManager() (*config.ConfigManager, error) {
	return config.NewConfigManager()
}

func createProfileInteractive(name string) (*config.Profile, error) {
	reader := bufio.NewReader(os.Stdin)

	printInfo("Creating profile: %s", name)
	printInfo("Press Enter to use default values or environment variables.")

	// Account ID
	fmt.Print("Cloudflare Account ID: ")
	accountID, _ := reader.ReadString('\n')
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		accountID = os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	}

	// API Token
	fmt.Print("Cloudflare API Token: ")
	apiToken, _ := reader.ReadString('\n')
	apiToken = strings.TrimSpace(apiToken)
	if apiToken == "" {
		apiToken = os.Getenv("CLOUDFLARE_API_TOKEN")
	}

	// Description
	fmt.Print("Description (optional): ")
	description, _ := reader.ReadString('\n')
	description = strings.TrimSpace(description)

	profile := &config.Profile{
		Name:        name,
		AccountID:   accountID,
		APIToken:    apiToken,
		Description: description,
		Region:      "auto",
	}

	return profile, nil
}

func fillProfileInteractively(profile *config.Profile) error {
	reader := bufio.NewReader(os.Stdin)

	printInfo("Filling missing values for profile: %s", profile.Name)

	if profile.AccountID == "" {
		fmt.Print("Cloudflare Account ID: ")
		accountID, _ := reader.ReadString('\n')
		profile.AccountID = strings.TrimSpace(accountID)
		if profile.AccountID == "" {
			profile.AccountID = os.Getenv("CLOUDFLARE_ACCOUNT_ID")
		}
	}

	if profile.APIToken == "" {
		fmt.Print("Cloudflare API Token: ")
		apiToken, _ := reader.ReadString('\n')
		profile.APIToken = strings.TrimSpace(apiToken)
		if profile.APIToken == "" {
			profile.APIToken = os.Getenv("CLOUDFLARE_API_TOKEN")
		}
	}

	if profile.Description == "" {
		fmt.Print("Description (optional): ")
		description, _ := reader.ReadString('\n')
		profile.Description = strings.TrimSpace(description)
	}

	return nil
}