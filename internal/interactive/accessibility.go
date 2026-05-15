/*
Package interactive provides accessibility features for inclusive CLI usage

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package interactive

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// AccessibilityMode defines different accessibility modes
type AccessibilityMode int

const (
	AccessibilityNone AccessibilityMode = iota
	AccessibilityScreenReader
	AccessibilityHighContrast
	AccessibilityLargeText
	AccessibilityReducedMotion
	AccessibilityFull
)

// AccessibilityConfig holds accessibility settings
type AccessibilityConfig struct {
	Mode             AccessibilityMode
	HighContrast     bool
	ScreenReader     bool
	LargeText        bool
	ReducedMotion    bool
	Verbose          bool
	KeyboardOnly     bool
	ColorBlind       bool
	AnnounceActions  bool
	ExtendedTimeouts bool
}

// AccessibilityManager manages accessibility features
type AccessibilityManager struct {
	config AccessibilityConfig
	Input  InputReader
}

// NewAccessibilityManager creates a new accessibility manager
func NewAccessibilityManager() *AccessibilityManager {
	return &AccessibilityManager{
		Input: DefaultInput(),
		config: AccessibilityConfig{
			Mode:             AccessibilityNone,
			HighContrast:     false,
			ScreenReader:     false,
			LargeText:        false,
			ReducedMotion:    false,
			Verbose:          false,
			KeyboardOnly:     false,
			ColorBlind:       false,
			AnnounceActions:  false,
			ExtendedTimeouts: false,
		},
	}
}

// SetMode sets the accessibility mode
func (am *AccessibilityManager) SetMode(mode AccessibilityMode) {
	am.config.Mode = mode

	switch mode {
	case AccessibilityScreenReader:
		am.config.ScreenReader = true
		am.config.Verbose = true
		am.config.AnnounceActions = true
		am.config.ReducedMotion = true
		am.config.ExtendedTimeouts = true
	case AccessibilityHighContrast:
		am.config.HighContrast = true
		am.config.ColorBlind = true
	case AccessibilityLargeText:
		am.config.LargeText = true
		am.config.Verbose = true
	case AccessibilityReducedMotion:
		am.config.ReducedMotion = true
	case AccessibilityFull:
		am.config.HighContrast = true
		am.config.ScreenReader = true
		am.config.LargeText = true
		am.config.ReducedMotion = true
		am.config.Verbose = true
		am.config.KeyboardOnly = true
		am.config.ColorBlind = true
		am.config.AnnounceActions = true
		am.config.ExtendedTimeouts = true
	}
}

// IsEnabled checks if any accessibility features are enabled
func (am *AccessibilityManager) IsEnabled() bool {
	return am.config.Mode != AccessibilityNone
}

// ShowAccessibilityMenu displays the accessibility configuration menu
func (am *AccessibilityManager) ShowAccessibilityMenu() error {
	ClearScreen()
	fmt.Println()
	fmt.Println(Bold("🔧 Accessibility Mode Configuration"))
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()

	fmt.Println("Select accessibility mode:")
	fmt.Println("  [1] Screen Reader      - Optimized for screen readers")
	fmt.Println("  [2] High Contrast      - High contrast colors")
	fmt.Println("  [3] Large Text         - Larger text and spacing")
	fmt.Println("  [4] Reduced Motion     - Disable animations")
	fmt.Println("  [5] Full Accessibility  - Enable all features")
	fmt.Println("  [6] Disable            - Turn off all features")
	fmt.Println()

	fmt.Printf("Select option [1]: ")
	input, _ := am.Input.ReadLine()

	switch input {
	case "1", "":
		am.SetMode(AccessibilityScreenReader)
	case "2":
		am.SetMode(AccessibilityHighContrast)
	case "3":
		am.SetMode(AccessibilityLargeText)
	case "4":
		am.SetMode(AccessibilityReducedMotion)
	case "5":
		am.SetMode(AccessibilityFull)
	case "6":
		am.SetMode(AccessibilityNone)
	default:
		PrintWarning("Invalid selection. Using default (Screen Reader).")
		am.SetMode(AccessibilityScreenReader)
	}

	am.applySettings()
	am.showCurrentSettings()

	return nil
}

// applySettings applies accessibility settings to the global systems
func (am *AccessibilityManager) applySettings() {
	// Apply settings to animation system
	if am.config.ReducedMotion {
		SetAnimationStyle("disabled")
	}

	// Apply screen reader settings
	if am.config.ScreenReader {
		// Disable complex visual effects
		globalAnimator.SetStyle("disabled")
		globalTransitionManager.Disable()
	}

	// Apply high contrast mode
	if am.config.HighContrast {
		// This would update color schemes in a real implementation
		// For now, we'll use the built-in color constants
	}

	// Apply large text
	if am.config.LargeText {
		// This would increase text size in a real implementation
	}
}

// showCurrentSettings displays the current accessibility settings
func (am *AccessibilityManager) showCurrentSettings() {
	fmt.Println()
	fmt.Println(Bold("📋 Current Accessibility Settings"))
	fmt.Println(strings.Repeat("─", 35))

	fmt.Printf("Mode: %s\n", am.getModeName())
	fmt.Printf("High Contrast: %s\n", formatBool(am.config.HighContrast))
	fmt.Printf("Screen Reader: %s\n", formatBool(am.config.ScreenReader))
	fmt.Printf("Large Text: %s\n", formatBool(am.config.LargeText))
	fmt.Printf("Reduced Motion: %s\n", formatBool(am.config.ReducedMotion))
	fmt.Printf("Verbose Output: %s\n", formatBool(am.config.Verbose))
	fmt.Printf("Keyboard Only: %s\n", formatBool(am.config.KeyboardOnly))
	fmt.Printf("Color Blind Support: %s\n", formatBool(am.config.ColorBlind))
	fmt.Printf("Announce Actions: %s\n", formatBool(am.config.AnnounceActions))
	fmt.Printf("Extended Timeouts: %s\n", formatBool(am.config.ExtendedTimeouts))

	fmt.Println()
	PrintSuccess("✅ Accessibility settings applied!")
}

// getModeName returns a human-readable mode name
func (am *AccessibilityManager) getModeName() string {
	switch am.config.Mode {
	case AccessibilityScreenReader:
		return "Screen Reader"
	case AccessibilityHighContrast:
		return "High Contrast"
	case AccessibilityLargeText:
		return "Large Text"
	case AccessibilityReducedMotion:
		return "Reduced Motion"
	case AccessibilityFull:
		return "Full Accessibility"
	default:
		return "Disabled"
	}
}

// Enhanced display methods for accessibility

// PrintAccessible prints text with accessibility considerations
func (am *AccessibilityManager) PrintAccessible(message string) {
	if am.config.ScreenReader {
		am.announceToScreenReader(message)
	}

	if am.config.Verbose {
		fmt.Printf("%s %s\n", "ℹ️", message)
	} else {
		fmt.Println(message)
	}
}

// PrintAccessibleSuccess prints a success message with accessibility
func (am *AccessibilityManager) PrintAccessibleSuccess(message string) {
	if am.config.ScreenReader {
		am.announceToScreenReader(fmt.Sprintf("Success: %s", message))
	}

	if am.config.Verbose {
		fmt.Printf("%s %s\n", "✅ Success:", message)
	} else {
		PrintSuccess("%s", message)
	}
}

// PrintAccessibleError prints an error message with accessibility
func (am *AccessibilityManager) PrintAccessibleError(message string) {
	if am.config.ScreenReader {
		am.announceToScreenReader(fmt.Sprintf("Error: %s", message))
	}

	if am.config.Verbose {
		fmt.Printf("%s %s\n", "❌ Error:", message)
	} else {
		PrintError("%s", message)
	}
}

// announceToScreenReader announces text to screen readers
func (am *AccessibilityManager) announceToScreenReader(text string) {
	// In a real implementation, this would use system APIs or screen reader protocols
	// For now, we'll use screen reader friendly formatting
	fmt.Printf("🔊 %s\n", text)
}

// formatBool formats boolean values for accessibility
func formatBool(value bool) string {
	if value {
		return Green("Enabled")
	}
	return Red("Disabled")
}

// AccessibilityHelper provides utility functions for accessible interactions

// AccessibilityHelper provides accessibility-aware user interaction methods
type AccessibilityHelper struct {
	manager *AccessibilityManager
	Input   InputReader
}

// NewAccessibilityHelper creates a new accessibility helper
func NewAccessibilityHelper() *AccessibilityHelper {
	return &AccessibilityHelper{
		manager: NewAccessibilityManager(),
		Input:   DefaultInput(),
	}
}

// ShowAccessibleMenu displays an accessible menu interface
func (ah *AccessibilityHelper) ShowAccessibleMenu(title string, options []string, defaultIndex int) int {
	if ah.manager.config.ScreenReader {
		return ah.showScreenReaderMenu(title, options, defaultIndex)
	}

	if ah.manager.config.LargeText {
		return ah.showLargeTextMenu(title, options, defaultIndex)
	}

	return ah.showStandardMenu(title, options, defaultIndex)
}

// showScreenReaderMenu displays a menu optimized for screen readers
func (ah *AccessibilityHelper) showScreenReaderMenu(title string, options []string, defaultIndex int) int {
	fmt.Println()
	fmt.Println(Bold(title))
	fmt.Println(strings.Repeat("=", len(title)+2))
	fmt.Println()

	ah.manager.PrintAccessible("Menu options:")

	for i, option := range options {
		marker := " "
		if i == defaultIndex {
			marker = "*"
		}
		fmt.Printf("  %s Option %d: %s\n", marker, i+1, option)
	}

	fmt.Println()
	fmt.Printf("Enter option number 1-%d [%d]: ", len(options), defaultIndex+1)

	input, _ := ah.Input.ReadLine()

	if input == "" {
		return defaultIndex
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(options) {
		ah.manager.PrintAccessibleError("Invalid selection. Please enter a number.")
		return ah.showScreenReaderMenu(title, options, defaultIndex)
	}

	return choice - 1
}

// showLargeTextMenu displays a menu with large text and spacing
func (ah *AccessibilityHelper) showLargeTextMenu(title string, options []string, defaultIndex int) int {
	fmt.Println()
	fmt.Println()
	fmt.Println(Bold(title))
	fmt.Println(strings.Repeat("=", len(title)+2))
	fmt.Println()

	for i, option := range options {
		prefix := "  "
		if i == defaultIndex {
			prefix = "► "
		}
		fmt.Printf("\n%s%s\n\n", prefix, option)
	}

	fmt.Printf("Select option (1-%d) [%d]: ", len(options), defaultIndex+1)

	input, _ := ah.Input.ReadLine()

	if input == "" {
		return defaultIndex
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(options) {
		PrintError("Invalid selection. Please enter a number.")
		return ah.showLargeTextMenu(title, options, defaultIndex)
	}

	return choice - 1
}

// showStandardMenu displays a standard menu
func (ah *AccessibilityHelper) showStandardMenu(title string, options []string, defaultIndex int) int {
	fmt.Println()
	fmt.Println(Bold(title))
	fmt.Println(strings.Repeat("─", len(title)+2))

	for i, option := range options {
		marker := " "
		if i == defaultIndex {
			marker = "●"
		}
		fmt.Printf("  %s [%d] %s\n", marker, i+1, option)
	}

	fmt.Printf("Select [%d]: ", defaultIndex+1)

	input, _ := ah.Input.ReadLine()

	if input == "" {
		return defaultIndex
	}

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 1 || choice > len(options) {
		PrintError("Invalid selection.")
		return ah.showStandardMenu(title, options, defaultIndex)
	}

	return choice - 1
}

// GetAccessibleInput gets user input with accessibility considerations
func (ah *AccessibilityHelper) GetAccessibleInput(prompt string, sensitive bool) string {
	if ah.manager.config.ScreenReader {
		ah.manager.announceToScreenReader(prompt)
	}

	if sensitive {
		fmt.Printf("%s: ", prompt)
		input, _ := ah.Input.ReadLine()
		return input
	}

	fmt.Printf("%s: ", prompt)
	input, _ := ah.Input.ReadLine()
	return input
}

// ConfirmAccessibleYesNo gets confirmation with accessibility
func (ah *AccessibilityHelper) ConfirmAccessibleYesNo(prompt string, defaultYes bool) bool {
	if ah.manager.config.ScreenReader {
		ah.manager.announceToScreenReader(fmt.Sprintf("Confirmation: %s", prompt))
	}

	defaultText := "Y/n"
	if !defaultYes {
		defaultText = "y/N"

		if ah.manager.config.Verbose {
			fmt.Printf("%s\n", prompt)
			defaultAnswer := "no"
	if defaultYes {
		defaultAnswer = "yes"
	}
	fmt.Printf("Press Enter for '%s', or type 'y' for yes, 'n' for no: ", defaultAnswer)
		} else {
			fmt.Printf("%s [%s]: ", prompt, defaultText)
		}
	} else {
		if ah.manager.config.Verbose {
			fmt.Printf("%s\n", prompt)
			defaultAnswer := "no"
	if defaultYes {
		defaultAnswer = "yes"
	}
	fmt.Printf("Press Enter for '%s', or type 'y' for yes, 'n' for no: ", defaultAnswer)
		} else {
			fmt.Printf("%s [%s]: ", prompt, defaultText)
		}
	}

	input, _ := ah.Input.ReadLine()
	input = strings.ToLower(input)

	if input == "" {
		return defaultYes
	}

	result := input == "y" || input == "yes"

	if ah.manager.config.AnnounceActions {
		action := "no"
		if result {
			action = "yes"
		}
		ah.manager.announceToScreenReader(fmt.Sprintf("Response: %s", action))
	}

	return result
}

// ShowAccessibleProgress displays progress with accessibility
func (ah *AccessibilityHelper) ShowAccessibleProgress(message string, current, total int) {
	if ah.manager.config.ScreenReader {
		ah.manager.announceToScreenReader(fmt.Sprintf("%s: %d of %d complete", message, current, total))
	}

	if ah.manager.config.Verbose {
		percentage := float64(current) / float64(total) * 100
		fmt.Printf("Progress: %s - %d/%d (%.1f%%)\n", message, current, total, percentage)
	} else {
		// Simple progress bar
		width := 30
		filled := int(float64(current) / float64(total) * float64(width))
		bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
		fmt.Printf("\r%s [%s] %d/%d", message, bar, current, total)
	}

	if current == total {
		fmt.Println()
	}
}

// AutoDetectAccessibility tries to auto-detect accessibility needs
func AutoDetectAccessibility() *AccessibilityManager {
	manager := NewAccessibilityManager()

	// Check for common accessibility environment variables
	if os.Getenv("SCREEN_READER") != "" || os.Getenv("ACCESSIBILITY") == "1" {
		manager.SetMode(AccessibilityScreenReader)
	}

	if os.Getenv("HIGH_CONTRAST") != "" {
		manager.SetMode(AccessibilityHighContrast)
	}

	if os.Getenv("LARGE_TEXT") != "" {
		manager.SetMode(AccessibilityLargeText)
	}

	if os.Getenv("REDUCED_MOTION") != "" {
		manager.SetMode(AccessibilityReducedMotion)
	}

	// Check if running in a terminal that supports accessibility
	if os.Getenv("TERM_PROGRAM") == "vscode" && os.Getenv("ACCESSIBILITY") == "1" {
		manager.SetMode(AccessibilityFull)
	}

	return manager
}

// Global accessibility manager instance
var globalAccessibilityManager = NewAccessibilityManager()

// GetAccessibilityManager returns the global accessibility manager
func GetAccessibilityManager() *AccessibilityManager {
	return globalAccessibilityManager
}

// EnableAccessibilityMode enables accessibility mode with the specified type
func EnableAccessibilityMode(mode AccessibilityMode) {
	globalAccessibilityManager.SetMode(mode)
}

// IsAccessibilityEnabled checks if accessibility is enabled
func IsAccessibilityEnabled() bool {
	return globalAccessibilityManager.IsEnabled()
}

// Add import for strconv if not already imported
// This function should be moved to the appropriate place in the imports section