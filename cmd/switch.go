/*
Package cmd provides the profile switch command for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"fmt"
	"os"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
	"github.com/CosmoLabs-org/cosmoflare/internal/interactive"
	"github.com/spf13/cobra"
)

// switchCmd represents the profile switch command
var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Switch between configuration profiles",
	Long: `Interactive profile switching with a beautiful interface.

This command provides an intuitive interface for managing and switching
between multiple Cloudflare profiles. You can view all available profiles,
see the current active profile, and switch between them seamlessly.

Features:
• Interactive profile selection with keyboard navigation
• Visual indicators for current profile
• Profile creation and deletion
• Detailed profile information display

Examples:
  cosmoflare switch                    # Interactive profile switcher
  cosmoflare config list              # List all profiles
  cosmoflare setup --profile=new      # Create new profile`,
	RunE: runSwitch,
}

var (
	switchDetails bool
	switchDelete  bool
)

func init() {
	rootCmd.AddCommand(switchCmd)

	switchCmd.Flags().BoolVar(&switchDetails, "details", false, "Show details for current profile")
	switchCmd.Flags().BoolVar(&switchDelete, "delete", false, "Delete a profile interactively")
}

func runSwitch(cmd *cobra.Command, args []string) error {
	profileManager, err := interactive.NewProfileManager()
	if err != nil {
		return fmt.Errorf("failed to create profile manager: %w", err)
	}

	if switchDelete {
		return profileManager.DeleteProfileInteractive()
	}

	if switchDetails {
		// Show details for current profile
		configMgr, err := config.NewConfigManager()
		if err != nil {
			return fmt.Errorf("failed to get config manager: %w", err)
		}

		currentProfile, err := configMgr.GetCurrent()
		if err != nil {
			printError("No current profile set: %v", err)
			printInfo("Use 'cosmoflare switch' to select a profile")
			return nil
		}

		return profileManager.ShowProfileDetails(currentProfile.Name)
	}

	// Default: show interactive switcher
	if err := profileManager.ShowProfileSwitcher(); err != nil {
		if os.IsNotExist(err) {
			printInfo("No profiles found. Run 'cosmoflare setup' to create your first profile.")
			return nil
		}
		return err
	}

	return nil
}
