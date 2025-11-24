/*
Simple R2Go2 Setup Demo

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/interactive"
)

func main() {
	fmt.Println("🚀 Welcome to R2Go2 Setup Demo!")
	fmt.Println("================================")
	fmt.Println()

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  ./simple-setup interactive    # Run interactive setup demo")
		fmt.Println("  ./simple-setup error         # Test beautiful error handling")
		fmt.Println("  ./simple-setup validation     # Test token validation demo")
		return
	}

	switch os.Args[1] {
	case "interactive":
		runInteractiveDemo()
	case "error":
		runErrorDemo()
	case "validation":
		runValidationDemo()
	default:
		fmt.Println("Unknown command:", os.Args[1])
	}
}

func runInteractiveDemo() {
	wizard := interactive.NewSetupWizard()

	// Show welcome screen
	wizard.Welcome()

	// Step 1: Authentication Method
	authMethod, err := wizard.Step1_AuthMethod()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Selected authentication method: %s\n", authMethod)

	// Step 2: API Token (simulated)
	interactive.ShowSpinner("Preparing secure input...", 1*time.Second)
	fmt.Printf("✅ Would normally collect API token securely here\n")

	// Step 3: Account Information (simulated)
	interactive.ShowSpinner("Preparing account setup...", 1*time.Second)
	fmt.Printf("✅ Would normally auto-detect account information here\n")

	// Step 4: Profile Setup (simulated)
	interactive.ShowSpinner("Preparing profile configuration...", 1*time.Second)
	fmt.Printf("✅ Would normally collect profile information here\n")

	// Connection test (simulated)
	interactive.ShowSpinner("Testing connection to Cloudflare R2...", 2*time.Second)
	interactive.SuccessMessage("Connection test passed!", "Your credentials are working correctly.")

	// Save configuration (simulated)
	interactive.ShowSpinner("Saving configuration...", 1*time.Second)

	// Show completion screen
	wizard.Complete("production", "My Company")

	fmt.Println("🎉 Interactive setup demo completed successfully!")
}

func runErrorDemo() {
	fmt.Println("🧪 Testing Beautiful Error Handling")
	fmt.Println("===================================")

	// Test different error types
	errors := []interactive.ErrorContext{
		interactive.NetworkError("Connection test", fmt.Errorf("timeout connecting to api.cloudflare.com")),
		interactive.AuthError("Token validation", fmt.Errorf("invalid API token format")),
		interactive.ConfigError("Profile loading", fmt.Errorf("configuration file not found")),
		interactive.InputError("Bucket name validation", fmt.Errorf("bucket name too short")),
	}

	for i, errCtx := range errors {
		if i > 0 {
			fmt.Println(strings.Repeat("=", 60))
		}
		interactive.HandleError(errCtx)
		time.Sleep(1 * time.Second)
	}

	fmt.Println("✅ Error handling demo completed!")
}

func runValidationDemo() {
	fmt.Println("🔐 Testing Token Validation Simulation")
	fmt.Println("=====================================")

	// Simulate different token scenarios
	testCases := []struct {
		token string
		name  string
	}{
		{"", "Empty token"},
		{"short", "Too short token"},
		{"0123456789abcdef0123456789ABCDEF", "Valid format token"},
		{"this-is-not-a-real-cloudflare-token-but-its-long-enough-to-look-plausible", "Invalid real token"},
	}

	for _, testCase := range testCases {
		fmt.Printf("\n🧪 Testing: %s\n", testCase.name)

		interactive.ShowSpinner("Validating token...", 2*time.Second)

		if testCase.token == "" {
			interactive.HandleError(interactive.AuthError("Token validation", fmt.Errorf("token cannot be empty")))
			continue
		}

		if len(testCase.token) < 20 {
			interactive.HandleError(interactive.AuthError("Token validation", fmt.Errorf("token is too short (minimum 20 characters)")))
			continue
		}

		// Simulate API call validation
		if testCase.token == "0123456789abcdef0123456789ABCDEF" {
			interactive.SuccessMessage("Token validated successfully!", "Account ID: 0123abcd********ef8912")
		} else {
			interactive.HandleError(interactive.AuthError("Token validation", fmt.Errorf("token rejected by Cloudflare API")))
		}
	}

	fmt.Println("\n✅ Token validation demo completed!")
}