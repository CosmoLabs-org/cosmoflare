/*
Package interactive provides test functions for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package interactive

import (
	"fmt"
	"strings"
	"time"
)

// ColorTest demonstrates the color output capabilities
func ColorTest() {
	fmt.Println()
	colorBold.Println("🎨 Color Output Test")
	fmt.Println(strings.Repeat("─", 40))

	colorSuccess.Println("✅ Success message")
	colorError.Println("❌ Error message")
	colorWarning.Println("⚠️ Warning message")
	colorInfo.Println("ℹ️ Info message")
	colorDim.Println("💡 Dimmed text")
	colorMuted.Println("🔇 Muted text")
	fmt.Println()
}

// ShowProgressBarTest demonstrates progress bar functionality
func ShowProgressBarTest() {
	fmt.Println()
	colorInfo.Println("📊 Progress Bar Test")
	fmt.Println(strings.Repeat("─", 40))

	for i := 0; i <= 10; i++ {
		ShowProgressBar(i, 10, "Processing")
		time.Sleep(200 * time.Millisecond)
	}
	fmt.Println()
}

// InputTest demonstrates input interaction capabilities
func InputTest() {
	fmt.Println()
	colorInfo.Println("📝 Input Interaction Test")
	fmt.Println(strings.Repeat("─", 40))

	// Test confirm
	confirmed := ConfirmYesNo("Would you like to continue with this test?", true)
	fmt.Printf("User response: %v\n", confirmed)

	// Test selection
	options := []string{
		"API Token (recommended)",
		"Service Key",
		"Environment Variables",
	}
	selected, err := SelectFromList("Choose authentication method:", options, 0)
	if err != nil {
		colorError.Println("Error:", err)
	} else {
		fmt.Printf("User selected: %s\n", colorBold.Sprintf(options[selected]))
	}

	// Test prompt with default
	name, err := PromptWithDefault("Enter your name", "John Doe")
	if err != nil {
		colorError.Println("Error:", err)
	} else {
		fmt.Printf("User entered: %s\n", colorBold.Sprintf(name))
	}

	fmt.Println()
}