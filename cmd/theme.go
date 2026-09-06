/*
Package cmd provides the theme command for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/CosmoLabs-org/cosmoflare/internal/interactive"
)

// themeCmd represents the theme command
var themeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Configure visual themes and appearance",
	Long: `Configure visual themes and appearance settings for R2Go2.

This command provides a beautiful interface for customizing the visual
appearance of R2Go2 with multiple built-in themes and support for
creating custom themes.

Available Themes:
• Cosmic    - Purple and blue gradients (default)
• Forest    - Green and earth tones
• Ocean     - Deep blues and teals
• Sunset    - Warm oranges and reds
• Monochrome- Black and white only

Features:
• Interactive theme selection
• Custom theme creation
• Theme previews
• Accessibility themes
• Animation controls

Examples:
  cosmoflare theme                      # Interactive theme menu
  cosmoflare theme --list              # List available themes
  cosmoflare theme --set=ocean         # Set ocean theme
  cosmoflare theme --create            # Create custom theme`,
	RunE: runTheme,
}

var (
	themeList   bool
	themeSet    string
	themeCreate bool
	themeInfo   bool
)

func init() {
	rootCmd.AddCommand(themeCmd)

	themeCmd.Flags().BoolVar(&themeList, "list", false, "List available themes")
	themeCmd.Flags().StringVar(&themeSet, "set", "", "Set theme by name")
	themeCmd.Flags().BoolVar(&themeCreate, "create", false, "Create custom theme")
	themeCmd.Flags().BoolVar(&themeInfo, "info", false, "Show current theme info")
}

func runTheme(cmd *cobra.Command, args []string) error {
	themeManager := interactive.GetThemeManager()

	if themeList {
		return listThemes(themeManager)
	}

	if themeSet != "" {
		return setTheme(themeManager, themeSet)
	}

	if themeCreate {
		return createTheme(themeManager)
	}

	if themeInfo {
		return showThemeInfo(themeManager)
	}

	// Default: show interactive theme menu
	return themeManager.ShowThemeMenu()
}

// listThemes lists all available themes
func listThemes(themeManager *interactive.ThemeManager) error {
	themes := themeManager.ListThemes()
	currentThemeName := themeManager.GetCurrentTheme().Name

	fmt.Println("Available Themes:")
	fmt.Println(strings.Repeat("─", 20))

	for _, theme := range themes {
		marker := " "
		if theme.Name == currentThemeName {
			marker = "●"
		}
		fmt.Printf("  %s %-15s %s%s\n", marker, theme.Name, "", theme.Description)
	}

	fmt.Println()
	fmt.Printf("Current: %s\n", interactive.Info(currentThemeName))
	return nil
}

// setTheme sets a specific theme
func setTheme(themeManager *interactive.ThemeManager, themeName string) error {
	if err := themeManager.SetTheme(themeName); err != nil {
		printError("Failed to set theme: %v", err)
		return err
	}

	theme := themeManager.GetCurrentTheme()
	printSuccess("Theme set to: %s", theme.Name)
	printInfo("Description: %s", theme.Description)
	return nil
}

// createTheme creates a new custom theme
func createTheme(themeManager *interactive.ThemeManager) error {
	if err := themeManager.CreateCustomTheme(); err != nil {
		return fmt.Errorf("failed to create theme: %w", err)
	}
	return nil
}

// showThemeInfo shows information about the current theme
func showThemeInfo(themeManager *interactive.ThemeManager) error {
	theme := themeManager.GetCurrentTheme()
	if theme == nil {
		printError("No theme is currently set")
		return fmt.Errorf("no theme set")
	}

	fmt.Printf("Current Theme: %s\n", interactive.Bold(theme.Name))
	fmt.Printf("Description: %s\n", theme.Description)
	fmt.Println()

	if theme.Colors.Name != "" {
		fmt.Printf("Color Scheme: %s\n", theme.Colors.Description)
	}

	if theme.Animations.Enabled {
		fmt.Printf("Animations: Enabled (Speed: %dms)\n", theme.Animations.Speed)
	} else {
		fmt.Printf("Animations: Disabled\n")
	}

	if theme.Icons.UseEmojis {
		fmt.Printf("Icons: Emoji enabled\n")
	} else {
		fmt.Printf("Icons: Text based\n")
	}

	fmt.Println()
	printInfo("Run 'cosmoflare theme' to change themes")

	return nil
}