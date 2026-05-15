/*
Package interactive provides enhanced profile management functionality

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package interactive

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/config"
)

// ProfileManager handles interactive profile operations
type ProfileManager struct {
	configMgr *config.ConfigManager
	Input     InputReader
}

// NewProfileManager creates a new profile manager
func NewProfileManager() (*ProfileManager, error) {
	configMgr, err := config.NewConfigManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	return &ProfileManager{
		configMgr: configMgr,
		Input:     DefaultInput(),
	}, nil
}

// ShowProfileSwitcher displays the interactive profile switching interface
func (pm *ProfileManager) ShowProfileSwitcher() error {
	profiles := pm.configMgr.ListProfiles()
	currentProfile, _ := pm.configMgr.GetCurrent()

	if len(profiles) == 0 {
		PrintWarning("No profiles found. Run 'r2go2 setup' to create one first.")
		return nil
	}

	fmt.Println("🔄 Profile Manager")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()

	if currentProfile != nil {
		fmt.Printf("Current: %s\n", FormatProfileName(currentProfile.Name, currentProfile.Description))
		fmt.Println()
	}

	fmt.Println("📋 Available Profiles:")
	for i, profileName := range profiles {
		profile, err := pm.configMgr.GetProfile(profileName)
		if err != nil {
			continue
		}

		status := "  "
		if currentProfile != nil && profile.Name == currentProfile.Name {
			status = "● "
		}

		icon := getProfileIcon(profile.Description)
		fmt.Printf("  [%d] %s%s%-20s %s%s\n",
			i+1, status, icon, profileName,
			getProfileDescription(profile.Description),
			Reset)
	}

	fmt.Println()
	fmt.Printf("Select profile to switch to [%d-%d], or 'c' to create new: ", 1, len(profiles))

	input, _ := pm.Input.ReadLine()

	// Handle creation
	if strings.ToLower(input) == "c" {
		return pm.createNewProfileFromSwitcher()
	}

	// Handle selection
	choice, err := strconv.Atoi(input)
	if err != nil {
		PrintError("Invalid selection. Please enter a number or 'c'.")
		return pm.ShowProfileSwitcher() // Show again
	}

	if choice < 1 || choice > len(profiles) {
		PrintError("Please select a valid profile number.")
		return pm.ShowProfileSwitcher() // Show again
	}

	selectedProfileName := profiles[choice-1]

	// Switch to selected profile
	if err := pm.configMgr.SetCurrent(selectedProfileName); err != nil {
		PrintError("Failed to switch profile: %v", err)
		return err
	}

	profile, _ := pm.configMgr.GetProfile(selectedProfileName)
	PrintSuccess("✅ Switched to profile: %s", FormatProfileName(profile.Name, profile.Description))

	return nil
}

// ShowProfileDetails displays detailed information about a profile
func (pm *ProfileManager) ShowProfileDetails(profileName string) error {
	profile, err := pm.configMgr.GetProfile(profileName)
	if err != nil {
		PrintError("Profile '%s' not found", profileName)
		return err
	}

	fmt.Printf("📊 Profile Details: %s\n", FormatProfileName(profile.Name, profile.Description))
	fmt.Println(strings.Repeat("─", 50))

	fmt.Printf("%s     %s\n", Bold("Account ID:"), config.MaskAccountID(profile.AccountID))
	fmt.Printf("%s         %s\n", Bold("Region:"), formatRegion(profile.Region))
	fmt.Printf("%s    %s\n", Bold("Description:"), profile.Description)

	if profile.Endpoint != "" {
		fmt.Printf("%s       %s\n", Bold("Endpoint:"), profile.Endpoint)
	}

	if profile.AccessKey != "" {
		fmt.Printf("%s     %s\n", Bold("Access Key:"), config.MaskKey(profile.AccessKey))
	}

	// Show quick stats
	fmt.Println()
	fmt.Printf("%s\n", Bold("Quick Stats:"))
	fmt.Printf("  📁 Configuration: %s\n", pm.configMgr.GetConfigPath())
	fmt.Printf("  🔐 API Token: %s\n", formatTokenStatus(profile.APIToken))
	fmt.Printf("  🌐 Account ID: %s\n", formatAccountIDStatus(profile.AccountID))

	return nil
}

// DeleteProfileInteractive guides user through profile deletion
func (pm *ProfileManager) DeleteProfileInteractive() error {
	profiles := pm.configMgr.ListProfiles()
	currentProfile, _ := pm.configMgr.GetCurrent()

	if len(profiles) == 0 {
		PrintWarning("No profiles found.")
		return nil
	}

	fmt.Println("🗑️  Delete Profile")
	fmt.Println(strings.Repeat("─", 30))

	fmt.Println("Available profiles:")
	for i, profileName := range profiles {
		profile, err := pm.configMgr.GetProfile(profileName)
		if err != nil {
			continue
		}

		status := ""
		if currentProfile != nil && profile.Name == currentProfile.Name {
			status = " (current - cannot delete)"
		}

		fmt.Printf("  [%d] %s%s\n", i+1, FormatProfileName(profileName, profile.Description), status)
	}

	fmt.Println()
	fmt.Printf("Select profile to delete [%d-%d]: ", 1, len(profiles))

	input, _ := pm.Input.ReadLine()

	choice, err := strconv.Atoi(input)
	if err != nil {
		PrintError("Invalid selection.")
		return err
	}

	if choice < 1 || choice > len(profiles) {
		PrintError("Invalid selection.")
		return err
	}

	selectedProfileName := profiles[choice-1]

	// Check if it's the current profile
	if currentProfile != nil && selectedProfileName == currentProfile.Name {
		PrintError("Cannot delete the current profile. Switch to another profile first.")
		return fmt.Errorf("cannot delete current profile")
	}

	// Confirm deletion
	profile, _ := pm.configMgr.GetProfile(selectedProfileName)
	fmt.Printf("Are you sure you want to delete profile '%s'? This action cannot be undone.\n",
		FormatProfileName(profile.Name, profile.Description))
	fmt.Printf("Type 'DELETE' to confirm: ")

	confirmation, _ := pm.Input.ReadLine()

	if strings.ToUpper(confirmation) != "DELETE" {
		PrintInfo("Profile deletion cancelled.")
		return nil
	}

	// Delete the profile
	if err := pm.configMgr.DeleteProfile(selectedProfileName); err != nil {
		PrintError("Failed to delete profile: %v", err)
		return err
	}

	PrintSuccess("✅ Profile '%s' deleted successfully.", profile.Name)
	return nil
}

// createNewProfileFromSwitcher creates a new profile from the switcher interface
func (pm *ProfileManager) createNewProfileFromSwitcher() error {
	fmt.Println("➕ Create New Profile")
	fmt.Println(strings.Repeat("─", 30))

	var profileName, description string

	// Get profile name
	for {
		fmt.Print("Profile name: ")
		profileName, _ = pm.Input.ReadLine()

		if profileName == "" {
			PrintError("Profile name cannot be empty.")
			continue
		}

		if pm.configMgr.ProfileExists(profileName) {
			PrintError("Profile '%s' already exists. Choose a different name.", profileName)
			continue
		}

		break
	}

	// Get description
	fmt.Print("Description (optional): ")
	description, _ = pm.Input.ReadLine()

	// Create a basic profile that user can configure later
	profile := &config.Profile{
		Name:        profileName,
		Description: description,
		Region:      "auto",
	}

	if err := pm.configMgr.SetProfile(profile); err != nil {
		PrintError("Failed to create profile: %v", err)
		return err
	}

	PrintSuccess("✅ Profile '%s' created successfully!", profileName)
	PrintInfo("Run 'r2go2 setup --profile=%s' to configure this profile.", profileName)

	return nil
}

// Helper functions

func getProfileIcon(description string) string {
	desc := strings.ToLower(description)
	switch {
	case strings.Contains(desc, "prod") || strings.Contains(desc, "production"):
		return "🏭 "
	case strings.Contains(desc, "stag") || strings.Contains(desc, "dev"):
		return "⚡ "
	case strings.Contains(desc, "test"):
		return "🧪 "
	case strings.Contains(desc, "personal"):
		return "💻 "
	case strings.Contains(desc, "work") || strings.Contains(desc, "company"):
		return "🏢 "
	default:
		return "📁 "
	}
}

func getProfileDescription(description string) string {
	if description == "" {
		return "(no description)"
	}
	if len(description) > 30 {
		return "(" + description[:27] + "...)"
	}
	return "(" + description + ")"
}

func FormatProfileName(name, description string) string {
	if description != "" {
		return fmt.Sprintf("%s %s", name, Dim(description))
	}
	return name
}

func formatRegion(region string) string {
	if region == "" || region == "auto" {
		return "Auto-detect"
	}
	return strings.ToUpper(region)
}

func formatTokenStatus(token string) string {
	if token == "" {
		return Red("Not set")
	}
	if len(token) < 10 {
		return Red("Invalid")
	}
	return Green("Configured")
}

func formatAccountIDStatus(accountID string) string {
	if accountID == "" {
		return Red("Not set")
	}
	if len(accountID) != 32 {
		return Yellow("Invalid format")
	}
	return Green("Valid format")
}