/*
Package interactive provides theming system for customizable visual appearance

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package interactive

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ColorScheme represents a color scheme
type ColorScheme struct {
	Name        string
	Description string
	Primary     string
	Secondary   string
	Success     string
	Warning     string
	Error       string
	Info        string
	Accent      string
	Background  string
	Foreground  string
	Emoji       map[string]string
}

// Theme represents a complete visual theme
type Theme struct {
	ID          string
	Name        string
	Description string
	Colors      ColorScheme
	Spacing     ThemeSpacing
	Icons       ThemeIcons
	Animations  ThemeAnimations
}

// ThemeSpacing defines spacing for the theme
type ThemeSpacing struct {
	LineSpacing   int
	SectionGap    int
	IndentSize    int
	ProgressBarWidth int
}

// ThemeIcons defines icon usage
type ThemeIcons struct {
	UseEmojis     bool
	SpinnerChars  []string
	ProgressChars string
	BulletChars   []string
}

// ThemeAnimations defines animation settings
type ThemeAnimations struct {
	Enabled      bool
	Speed        int // milliseconds
	Easing       string
	Transitions  bool
}

// ThemeManager manages theme loading and application
type ThemeManager struct {
	currentTheme *Theme
	themes       map[string]*Theme
	themePath    string
	Input        InputReader
}

// NewThemeManager creates a new theme manager
func NewThemeManager() *ThemeManager {
	manager := &ThemeManager{
		themes:    make(map[string]*Theme),
		Input:     DefaultInput(),
	}
	manager.themePath = manager.getThemePath()

	// Load built-in themes
	manager.loadBuiltinThemes()

	// Load custom themes
	manager.loadCustomThemes()

	// Set default theme
	manager.SetTheme("cosmic")

	return manager
}

// getThemePath returns the path to user themes
func (tm *ThemeManager) getThemePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".cosmoflare", "themes")
}

// loadBuiltinThemes loads the built-in themes
func (tm *ThemeManager) loadBuiltinThemes() {
	// Cosmic Theme (default)
	tm.themes["cosmic"] = &Theme{
		ID:          "cosmic",
		Name:        "Cosmic",
		Description: "🌌 Purple and blue gradients",
		Colors: ColorScheme{
			Name:        "cosmic",
			Description: "Purple and blue gradient theme",
			Primary:     "\033[38;5;147m",  // Light purple
			Secondary:   "\033[38;5;75m",   // Light blue
			Success:     "\033[38;5;84m",   // Green
			Warning:     "\033[38;5;221m",  // Yellow
			Error:       "\033[38;5;203m",  // Red
			Info:        "\033[38;5;123m",  // Cyan
			Accent:      "\033[38;5;177m",  // Pink
			Background:  "\033[48;5;235m",  // Dark background
			Foreground:  "\033[38;5;255m",  // White
			Emoji: map[string]string{
				"success": "✨",
				"error":   "💫",
				"warning": "⚡",
				"info":    "🔮",
				"loading": "🌌",
			},
		},
		Spacing: ThemeSpacing{
			LineSpacing:        1,
			SectionGap:         2,
			IndentSize:         2,
			ProgressBarWidth:   40,
		},
		Icons: ThemeIcons{
			UseEmojis:    true,
			SpinnerChars: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
			ProgressChars: "█",
			BulletChars:  []string{"•", "◦", "○"},
		},
		Animations: ThemeAnimations{
			Enabled:     true,
			Speed:       60,
			Easing:      "smooth",
			Transitions: true,
		},
	}

	// Forest Theme
	tm.themes["forest"] = &Theme{
		ID:          "forest",
		Name:        "Forest",
		Description: "🌲 Green and earth tones",
		Colors: ColorScheme{
			Name:        "forest",
			Description: "Nature-inspired green theme",
			Primary:     "\033[38;5;106m",  // Green
			Secondary:   "\033[38;5;100m",  // Olive
			Success:     "\033[38;5;112m",  // Bright green
			Warning:     "\033[38;5;172m",  // Orange
			Error:       "\033[38;5;124m",  // Red
			Info:        "\033[38;5;108m",  // Cyan-green
			Accent:      "\033[38;5;143m",  // Brown
			Background:  "\033[48;5;234m",  // Dark green background
			Foreground:  "\033[38;5;255m",  // White
			Emoji: map[string]string{
				"success": "🌿",
				"error":   "🍂",
				"warning": "🌻",
				"info":    "🍄",
				"loading": "🌲",
			},
		},
		Spacing: ThemeSpacing{
			LineSpacing:        1,
			SectionGap:         2,
			IndentSize:         2,
			ProgressBarWidth:   40,
		},
		Icons: ThemeIcons{
			UseEmojis:    true,
			SpinnerChars: []string{"🌲", "🌳", "🌴", "🎋"},
			ProgressChars: "▓",
			BulletChars:  []string{"🍃", "🌱", "🌿"},
		},
		Animations: ThemeAnimations{
			Enabled:     true,
			Speed:       80,
			Easing:      "organic",
			Transitions: true,
		},
	}

	// Ocean Theme
	tm.themes["ocean"] = &Theme{
		ID:          "ocean",
		Name:        "Ocean",
		Description: "🌊 Deep blues and teals",
		Colors: ColorScheme{
			Name:        "ocean",
			Description: "Ocean-inspired blue theme",
			Primary:     "\033[38;5;39m",   // Blue
			Secondary:   "\033[38;5;80m",   // Cyan
			Success:     "\033[38;5;43m",   // Aqua
			Warning:     "\033[38;5;220m",  // Gold
			Error:       "\033[38;5;196m",  // Red
			Info:        "\033[38;5;51m",   // Light blue
			Accent:      "\033[38;5;225m",  // Light cyan
			Background:  "\033[48;5;236m",  // Dark blue background
			Foreground:  "\033[38;5;255m",  // White
			Emoji: map[string]string{
				"success": "🐚",
				"error":   "🌊",
				"warning": "⚓",
				"info":    "🐠",
				"loading": "🌊",
			},
		},
		Spacing: ThemeSpacing{
			LineSpacing:        1,
			SectionGap:         2,
			IndentSize:         2,
			ProgressBarWidth:   40,
		},
		Icons: ThemeIcons{
			UseEmojis:    true,
			SpinnerChars: []string{"🌊", "🌊", "💧", "💦", "🌊"},
			ProgressChars: "▓",
			BulletChars:  []string{"🐚", "🦀", "⚓"},
		},
		Animations: ThemeAnimations{
			Enabled:     true,
			Speed:       50,
			Easing:      "wave",
			Transitions: true,
		},
	}

	// Sunset Theme
	tm.themes["sunset"] = &Theme{
		ID:          "sunset",
		Name:        "Sunset",
		Description: "🌅 Warm oranges and reds",
		Colors: ColorScheme{
			Name:        "sunset",
			Description: "Sunset-inspired warm theme",
			Primary:     "\033[38;5;208m",  // Orange
			Secondary:   "\033[38;5;196m",  // Red
			Success:     "\033[38;5;214m",  // Gold
			Warning:     "\033[38;5;226m",  // Yellow
			Error:       "\033[38;5;160m",  // Dark red
			Info:        "\033[38;5;179m",  // Peach
			Accent:      "\033[38;5;203m",  // Pink
			Background:  "\033[48;5;94m",   // Warm background
			Foreground:  "\033[38;5;255m",  // White
			Emoji: map[string]string{
				"success": "🌅",
				"error":   "🌆",
				"warning": "🌇",
				"info":    "🌄",
				"loading": "🌅",
			},
		},
		Spacing: ThemeSpacing{
			LineSpacing:        1,
			SectionGap:         2,
			IndentSize:         2,
			ProgressBarWidth:   40,
		},
		Icons: ThemeIcons{
			UseEmojis:    true,
			SpinnerChars: []string{"🌅", "🌆", "🌇", "🌄"},
			ProgressChars: "█",
			BulletChars:  []string{"🔥", "✨", "💫"},
		},
		Animations: ThemeAnimations{
			Enabled:     true,
			Speed:       70,
			Easing:      "warm",
			Transitions: true,
		},
	}

	// Monochrome Theme
	tm.themes["monochrome"] = &Theme{
		ID:          "monochrome",
		Name:        "Monochrome",
		Description: "⚪ Black and white only",
		Colors: ColorScheme{
			Name:        "monochrome",
			Description: "Simple black and white theme",
			Primary:     "\033[38;5;255m", // White
			Secondary:   "\033[38;5;245m", // Light gray
			Success:     "\033[38;5;255m", // White
			Warning:     "\033[38;5;255m", // White
			Error:       "\033[38;5;255m", // White
			Info:        "\033[38;5;255m", // White
			Accent:      "\033[38;5;255m", // White
			Background:  "\033[48;5;16m",  // Black
			Foreground:  "\033[38;5;255m", // White
			Emoji: map[string]string{
				"success": "✓",
				"error":   "✗",
				"warning": "⚠",
				"info":    "ℹ",
				"loading": "◐",
			},
		},
		Spacing: ThemeSpacing{
			LineSpacing:        1,
			SectionGap:         1,
			IndentSize:         4,
			ProgressBarWidth:   50,
		},
		Icons: ThemeIcons{
			UseEmojis:    false,
			SpinnerChars: []string{"|", "/", "-", "\\"},
			ProgressChars: "=",
			BulletChars:  []string{"•", "◦", "○"},
		},
		Animations: ThemeAnimations{
			Enabled:     false,
			Speed:       0,
			Easing:      "none",
			Transitions: false,
		},
	}
}

// loadCustomThemes loads user-defined themes from the theme directory
func (tm *ThemeManager) loadCustomThemes() {
	// In a real implementation, this would scan the theme directory
	// for YAML/JSON theme files and load them
	// For now, we'll create the directory if it doesn't exist
	if tm.themePath != "" {
		os.MkdirAll(tm.themePath, 0755)
	}
}

// SetTheme applies a theme by name
func (tm *ThemeManager) SetTheme(themeName string) error {
	theme, exists := tm.themes[themeName]
	if !exists {
		return fmt.Errorf("theme '%s' not found", themeName)
	}

	tm.currentTheme = theme
	tm.applyTheme()
	return nil
}

// applyTheme applies the current theme to the display system
func (tm *ThemeManager) applyTheme() {
	if tm.currentTheme == nil {
		return
	}

	// Apply animation settings
	if !tm.currentTheme.Animations.Enabled {
		SetAnimationStyle("disabled")
	} else {
		switch tm.currentTheme.Animations.Speed {
		case 0, 30:
			SetAnimationStyle("fast")
		case 80, 100:
			SetAnimationStyle("slow")
		default:
			SetAnimationStyle("normal")
		}
	}

	// In a real implementation, this would also update color constants
	// and other visual elements
}

// GetCurrentTheme returns the currently active theme
func (tm *ThemeManager) GetCurrentTheme() *Theme {
	return tm.currentTheme
}

// ListThemes returns all available themes
func (tm *ThemeManager) ListThemes() []*Theme {
	themes := make([]*Theme, 0, len(tm.themes))
	for _, theme := range tm.themes {
		themes = append(themes, theme)
	}
	return themes
}

// ShowThemeMenu displays the theme selection interface
func (tm *ThemeManager) ShowThemeMenu() error {
	ClearScreen()
	fmt.Println()
	fmt.Println(Bold("🎨 Theme Configuration"))
	fmt.Println(strings.Repeat("─", 30))
	fmt.Println()

	themes := tm.ListThemes()
	currentThemeName := tm.GetCurrentTheme().Name

	fmt.Println("Available Themes:")
	for i, theme := range themes {
		marker := " "
		if theme.Name == currentThemeName {
			marker = "●"
		}
		fmt.Printf("  [%d] %s %-15s %s\n", i+1, marker, theme.Name, Dim(theme.Description))
	}

	fmt.Println()
	fmt.Printf("Select theme [1]: ")

	input, _ := tm.Input.ReadLine()

	if input == "" {
		input = "1"
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(themes) {
		PrintError("Invalid selection.")
		return tm.ShowThemeMenu()
	}

	selectedTheme := themes[choice-1]
	if err := tm.SetTheme(selectedTheme.ID); err != nil {
		PrintError("Failed to apply theme: %v", err)
		return err
	}

	PrintSuccess("✅ Theme applied: %s", selectedTheme.Name)

	// Show theme preview
	tm.showThemePreview(selectedTheme)

	return nil
}

// showThemePreview displays a preview of the selected theme
func (tm *ThemeManager) showThemePreview(theme *Theme) {
	fmt.Println()
	fmt.Println(Bold("🎭 Theme Preview"))
	fmt.Println(strings.Repeat("─", 25))
	fmt.Println()

	fmt.Printf("Theme: %s\n", theme.Name)
	fmt.Printf("Description: %s\n", theme.Description)
	fmt.Println()

	fmt.Println("Color Samples:")
	fmt.Printf("  Primary:   %s%s%s\n", theme.Colors.Primary, "Primary Text", Reset)
	fmt.Printf("  Secondary: %s%s%s\n", theme.Colors.Secondary, "Secondary Text", Reset)
	fmt.Printf("  Success:   %s%s%s\n", theme.Colors.Success, "Success Message", Reset)
	fmt.Printf("  Warning:   %s%s%s\n", theme.Colors.Warning, "Warning Message", Reset)
	fmt.Printf("  Error:     %s%s%s\n", theme.Colors.Error, "Error Message", Reset)
	fmt.Printf("  Info:      %s%s%s\n", theme.Colors.Info, "Info Message", Reset)
	fmt.Println()

	if theme.Icons.UseEmojis {
		fmt.Println("Icon Samples:")
		fmt.Printf("  ✨ Success  💫 Error  ⚡ Warning  🔮 Info\n")
		fmt.Println()
	}

	fmt.Println("UI Elements:")
	tm.showProgressBar(theme, 75)
	tm.showSpinnerDemo(theme)
	fmt.Println()

	fmt.Println("Typography:")
	fmt.Printf("  %s\n", Bold("Bold Text"))
	fmt.Printf("  %s\n", Dim("Dim Text"))
	fmt.Printf("  %sRegular Text%s\n", theme.Colors.Primary, Reset)
}

// showProgressBar displays a progress bar in the theme style
func (tm *ThemeManager) showProgressBar(theme *Theme, percentage int) {
	width := theme.Spacing.ProgressBarWidth
	filled := int(float64(percentage) / 100.0 * float64(width))
	empty := width - filled

	bar := strings.Repeat(theme.Icons.ProgressChars, filled) + strings.Repeat(" ", empty)
	fmt.Printf("  Progress: [%s] %d%%\n", bar, percentage)
}

// showSpinnerDemo displays a spinner animation
func (tm *ThemeManager) showSpinnerDemo(theme *Theme) {
	fmt.Printf("  Spinner: ")
	for i, char := range theme.Icons.SpinnerChars {
		if i > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		fmt.Printf("\r  Spinner: %s", char)
	}
	fmt.Printf("\r  Spinner: %s (demo complete)\n", theme.Icons.SpinnerChars[0])
}

// CreateCustomTheme allows users to create a custom theme
func (tm *ThemeManager) CreateCustomTheme() error {
	ClearScreen()
	fmt.Println()
	fmt.Println(Bold("🎨 Custom Theme Editor"))
	fmt.Println(strings.Repeat("─", 35))
	fmt.Println()

	fmt.Print("Theme name: ")
	name, _ := tm.Input.ReadLine()

	if name == "" {
		PrintError("Theme name cannot be empty.")
		return fmt.Errorf("empty theme name")
	}

	if _, exists := tm.themes[name]; exists {
		PrintError("Theme '%s' already exists.", name)
		return fmt.Errorf("theme exists")
	}

	fmt.Print("Description: ")
	description, _ := tm.Input.ReadLine()

	// Create custom theme based on cosmic theme
	customTheme := &Theme{
		ID:          strings.ToLower(name),
		Name:        name,
		Description: description,
		Colors:      tm.themes["cosmic"].Colors, // Start with cosmic base
		Spacing:     tm.themes["cosmic"].Spacing,
		Icons:       tm.themes["cosmic"].Icons,
		Animations:  tm.themes["cosmic"].Animations,
	}

	// Apply customizations
	tm.customizeTheme(customTheme)

	// Save the theme
	tm.themes[customTheme.ID] = customTheme

	PrintSuccess("✅ Custom theme '%s' created!", name)
	return nil
}

// customizeTheme guides user through theme customization
func (tm *ThemeManager) customizeTheme(theme *Theme) {
	fmt.Println()
	fmt.Println("Customize your theme:")
	fmt.Println()

	// Animation preference
	animations := ConfirmWithReader("Enable animations?", theme.Animations.Enabled, tm.Input)
	theme.Animations.Enabled = animations

	if animations {
		fmt.Print("Animation speed (1-100): ")
		speed, _ := tm.Input.ReadLine()
		if speedInt, err := strconv.Atoi(speed); err == nil {
			theme.Animations.Speed = speedInt
		}
	}

	// Emoji preference
	useEmojis := ConfirmWithReader("Use emoji icons?", theme.Icons.UseEmojis, tm.Input)
	theme.Icons.UseEmojis = useEmojis

	// Progress bar width
	fmt.Printf("Progress bar width [%d]: ", theme.Spacing.ProgressBarWidth)
	width, _ := tm.Input.ReadLine()
	if widthInt, err := strconv.Atoi(width); err == nil {
		theme.Spacing.ProgressBarWidth = widthInt
	}
}

// SaveTheme saves a theme to disk
func (tm *ThemeManager) SaveTheme(theme *Theme) error {
	// In a real implementation, this would save the theme as YAML/JSON
	// to the theme directory for persistence
	return nil
}

// LoadTheme loads a theme from disk
func (tm *ThemeManager) LoadTheme(themeID string) error {
	// In a real implementation, this would load a theme from YAML/JSON
	// file in the theme directory
	return nil
}

// GetThemePath returns the path to theme files
func (tm *ThemeManager) GetThemePath() string {
	return tm.themePath
}

// Global theme manager instance
var globalThemeManager = NewThemeManager()

// GetThemeManager returns the global theme manager
func GetThemeManager() *ThemeManager {
	return globalThemeManager
}

// SetGlobalTheme sets the global theme
func SetGlobalTheme(themeName string) error {
	return globalThemeManager.SetTheme(themeName)
}

// GetCurrentThemeName returns the current theme name
func GetCurrentThemeName() string {
	theme := globalThemeManager.GetCurrentTheme()
	if theme != nil {
		return theme.Name
	}
	return "unknown"
}

// ApplyThemeSettings applies theme-based settings to display functions
func ApplyThemeSettings() {
	theme := globalThemeManager.GetCurrentTheme()
	if theme == nil {
		return
	}

	// Update spinner characters
	if len(theme.Icons.SpinnerChars) > 0 {
		// In a real implementation, this would update global spinner arrays
	}

	// Update animation settings
	if !theme.Animations.Enabled {
		SetAnimationStyle("disabled")
	}
}