/*
Package interactive provides beautiful CLI interactions for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package interactive

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/utils"
	"github.com/fatih/color"
)

// Color definitions
var (
	colorBold    = color.New(color.Bold)
	colorSuccess = color.New(color.FgGreen, color.Bold)
	colorError   = color.New(color.FgRed, color.Bold)
	colorWarning = color.New(color.FgYellow, color.Bold)
	colorInfo    = color.New(color.FgCyan, color.Bold)
	colorDim     = color.New(color.Faint)
	colorMuted   = color.New(color.FgHiBlack)
)

// SetupWizard represents the interactive setup wizard
type SetupWizard struct {
	Step       int
	TotalSteps int
	Quiet      bool
}

// NewSetupWizard creates a new setup wizard instance
func NewSetupWizard() *SetupWizard {
	return &SetupWizard{
		Step:       0,
		TotalSteps: 4,
		Quiet:      false,
	}
}

// Welcome displays the welcome screen
func (w *SetupWizard) Welcome() {
	w.clearScreen()

	fmt.Println()
	colorBold.Println("🚀 Welcome to R2Go2 Setup!")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()

	colorInfo.Println("Let's configure your Cloudflare R2 access:")
	fmt.Println()

	colorMuted.Println("This wizard will guide you through:")
	fmt.Println("  • Authentication configuration")
	fmt.Println("  • Account verification")
	fmt.Println("  • Profile setup")
	fmt.Println("  • Connection testing")
	fmt.Println()

	w.showProgress()
	fmt.Println()
}

// showProgress displays the current progress
func (w *SetupWizard) showProgress() {
	fmt.Printf("📊 Progress: ")

	for i := 1; i <= w.TotalSteps; i++ {
		if i <= w.Step {
			colorSuccess.Printf("● ")
		} else {
			colorDim.Printf("○ ")
		}
	}

	colorMuted.Printf("(Step %d/%d)", w.Step, w.TotalSteps)
	fmt.Println()
}

// Step1_AuthMethod handles authentication method selection
func (w *SetupWizard) Step1_AuthMethod() (string, error) {
	w.Step = 1
	w.clearScreen()

	fmt.Println()
	colorInfo.Println("📋 Step 1/4: Authentication Method")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()

	colorMuted.Println("Choose how you want to authenticate with Cloudflare:")
	fmt.Println()

	fmt.Println("  [1] API Token (recommended)")
	fmt.Println("       • Most secure method")
	fmt.Println("       • Scoped permissions")
	fmt.Println("       • Easy to manage")
	fmt.Println()

	fmt.Println("  [2] Service Key")
	fmt.Println("       • Full account access")
	fmt.Println("       • Use with caution")
	fmt.Println("       • Good for automation")
	fmt.Println()

	fmt.Println("  [3] Environment Variables")
	fmt.Println("       • Use existing CLOUDFLARE_API_TOKEN")
	fmt.Println("       • Already configured")
	fmt.Println()

	for {
		fmt.Print("Choose method [1]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response == "" || response == "1" {
			return "api_token", nil
		} else if response == "2" {
			return "service_key", nil
		} else if response == "3" {
			return "env", nil
		}

		colorError.Println("  ❌ Please enter 1, 2, or 3")
	}
}

// Step2_APIToken handles API token input with validation
func (w *SetupWizard) Step2_APIToken() (string, error) {
	w.Step = 2
	w.clearScreen()

	fmt.Println()
	colorInfo.Println("🔐 Step 2/4: API Token")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()

	colorMuted.Println("Enter your Cloudflare API Token:")
	fmt.Println()
	fmt.Println("  📝 Need a token? Visit: https://dash.cloudflare.com/profile/api-tokens")
	fmt.Println("  🔒 Required permissions: R2:Read, R2:Write")
	fmt.Println()

	// Try environment variable first
	envToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	if envToken != "" {
		fmt.Printf("Found token in environment variable (%s characters). Use this? [Y/n]: ",
			colorMuted.Sprintf("%d", len(envToken)))
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "" || response == "y" || response == "yes" {
			return envToken, nil
		}
	}

	// Interactive token input with masking
	for {
		fmt.Print("Cloudflare API Token: ")
		token, err := readPassword()
		if err != nil {
			return "", fmt.Errorf("failed to read token: %w", err)
		}

		token = strings.TrimSpace(token)
		if token == "" {
			colorError.Println("  ❌ Token cannot be empty")
			continue
		}

		// Basic token format validation
		if len(token) < 20 {
			colorError.Println("  ❌ Token seems too short (minimum 20 characters)")
			continue
		}

		// Show masked token for confirmation
		masked := maskToken(token)
		fmt.Printf("Token entered: %s\n", colorMuted.Sprintf(masked))

		fmt.Printf("Does this look correct? [Y/n]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "" || response == "y" || response == "yes" {
			return token, nil
		}

		colorWarning.Println("  ⚠️ Let's try again...")
	}
}

// Step3_AccountInfo handles account information and auto-detection
func (w *SetupWizard) Step3_AccountInfo(token string) (string, string, error) {
	w.Step = 3
	w.clearScreen()

	fmt.Println()
	colorInfo.Println("🏢 Step 3/4: Account Information")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()

	// Try to auto-detect account info
	accountID, accountName := autoDetectAccountInfo(token)

	if accountID != "" {
		colorSuccess.Println("  ✅ Auto-detected account information!")
		fmt.Printf("  Account ID: %s\n", colorMuted.Sprintf(utils.MaskAccountID(accountID)))
		if accountName != "" {
			fmt.Printf("  Account Name: %s\n", accountName)
		}
		fmt.Println()

		fmt.Printf("Use this account? [Y/n]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "" || response == "y" || response == "yes" {
			return accountID, accountName, nil
		}
	}

	// Manual account ID input
	fmt.Println("Please enter your Cloudflare Account ID:")
	fmt.Println()

	// Try environment variable first
	envAccountID := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	if envAccountID != "" {
		fmt.Printf("Found account ID in environment: %s. Use this? [Y/n]: ",
			colorMuted.Sprintf(utils.MaskAccountID(envAccountID)))
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "" || response == "y" || response == "yes" {
			return envAccountID, "", nil
		}
	}

	for {
		fmt.Print("Account ID: ")
		reader := bufio.NewReader(os.Stdin)
		accountID, _ := reader.ReadString('\n')
		accountID = strings.TrimSpace(accountID)

		if accountID == "" {
			colorError.Println("  ❌ Account ID cannot be empty")
			continue
		}

		// Validate account ID format (should be 32 hex characters)
		if len(accountID) != 32 {
			colorError.Println("  ❌ Account ID should be 32 characters long")
			continue
		}

		return accountID, "", nil
	}
}

// Step4_ProfileSetup handles profile configuration
func (w *SetupWizard) Step4_ProfileSetup() (string, string, error) {
	w.Step = 4
	w.clearScreen()

	fmt.Println()
	colorInfo.Println("💾 Step 4/4: Profile Setup")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Profile name
	defaultProfileName := "production"
	fmt.Printf("Profile name [%s]: ", colorMuted.Sprintf(defaultProfileName))
	profileName, _ := reader.ReadString('\n')
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		profileName = defaultProfileName
	}

	// Profile description
	fmt.Print("Description (optional): ")
	description, _ := reader.ReadString('\n')
	description = strings.TrimSpace(description)

	return profileName, description, nil
}

// Complete shows the completion screen
func (w *SetupWizard) Complete(profileName, accountName string) {
	w.Step = w.TotalSteps
	w.clearScreen()

	fmt.Println()
	colorSuccess.Println("🎉 Setup completed successfully!")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Println()

	fmt.Printf("✅ Profile: %s\n", colorBold.Sprintf(profileName))
	if accountName != "" {
		fmt.Printf("✅ Account: %s\n", accountName)
	}
	fmt.Println()

	colorMuted.Println("Your configuration has been saved securely.")
	fmt.Println()

	colorInfo.Println("Next steps:")
	fmt.Printf("  • Run: %s\n", colorBold.Sprintf("r2go2 config list"))
	fmt.Printf("  • Test: %s\n", colorBold.Sprintf("r2go2 bucket list"))
	fmt.Printf("  • Create: %s\n", colorBold.Sprintf("r2go2 bucket create my-bucket"))
	fmt.Println()

	w.showProgress()
	fmt.Println()
}

// clearScreen clears the terminal screen
func (w *SetupWizard) clearScreen() {
	if !w.Quiet {
		fmt.Print("\033[H\033[2J")
	}
}

// maskToken masks a token for display
func maskToken(token string) string {
	if len(token) <= 8 {
		return strings.Repeat("*", len(token))
	}

	visible := 4
	masked := strings.Repeat("*", len(token)-visible*2)
	return token[:visible] + masked + token[len(token)-visible:]
}

// ShowProgressBar displays a progress bar
func ShowProgressBar(current, total int, prefix string) {
	const barWidth = 40

	percent := float64(current) / float64(total)
	filled := int(percent * barWidth)

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	fmt.Printf("\r%s %s %d/%d (%.1f%%)",
		prefix,
		colorInfo.Sprint("["+bar+"]"),
		current,
		total,
		percent*100)

	if current == total {
		fmt.Println()
	}
}

// PromptWithDefault prompts the user with a default value
func PromptWithDefault(prompt, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, colorMuted.Sprintf(defaultValue))
	} else {
		fmt.Printf("%s: ", prompt)
	}

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if response == "" {
		return defaultValue, nil
	}

	return response, nil
}

// SelectFromList prompts the user to select from a list
func SelectFromList(prompt string, options []string, defaultIndex int) (int, error) {
	fmt.Println(prompt)
	fmt.Println()

	for i, option := range options {
		marker := " "
		if i == defaultIndex {
			marker = "►"
			colorInfo.Printf("%s [%d] %s\n", marker, i+1, option)
		} else {
			fmt.Printf("  [%d] %s\n", i+1, option)
		}
	}
	fmt.Println()

	for {
		if defaultIndex >= 0 {
			fmt.Printf("Select option [%d]: ", defaultIndex+1)
		} else {
			fmt.Print("Select option: ")
		}

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(response)

		if response == "" && defaultIndex >= 0 {
			return defaultIndex, nil
		}

		index, err := strconv.Atoi(response)
		if err != nil || index < 1 || index > len(options) {
			colorError.Printf("  Please enter a number between 1 and %d\n", len(options))
			continue
		}

		return index - 1, nil
	}
}