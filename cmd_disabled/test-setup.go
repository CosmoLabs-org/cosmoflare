/*
Test setup for R2Go2 interactive wizard

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package main

import (
	"fmt"
	"os"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/interactive"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "test" {
		testInteractiveSetup()
		return
	}

	fmt.Println("R2Go2 Test Setup")
	fmt.Println("================")
	fmt.Println("This is a test version to demonstrate the interactive setup wizard")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  ./test-setup test    # Run the interactive setup wizard test")
}

func testInteractiveSetup() {
	// Test the interactive setup components
	fmt.Println("🚀 Testing R2Go2 Interactive Setup Components")
	fmt.Println("================================================")

	// Test color outputs
	interactive.ColorTest()

	// Test spinner
	interactive.ShowSpinner("Testing spinner animation", 3)

	// Test progress bar
	interactive.ShowProgressBarTest()

	// Test input functions
	interactive.InputTest()

	fmt.Println()
	fmt.Println("✅ Interactive setup component test completed!")
}