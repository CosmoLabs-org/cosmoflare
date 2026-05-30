/*
Package interactive provides first-run detection and auto-setup functionality

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package interactive

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/config"
)

const (
	Second = 1 * time.Second
)

// FirstRunDetector handles first-time user detection and setup
type FirstRunDetector struct {
	configMgr *config.ConfigManager
}

// NewFirstRunDetector creates a new first-run detector
func NewFirstRunDetector() (*FirstRunDetector, error) {
	configMgr, err := config.NewConfigManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	return &FirstRunDetector{
		configMgr: configMgr,
	}, nil
}

// IsFirstRun checks if this is the first time the user is running R2Go2
func (frd *FirstRunDetector) IsFirstRun() bool {
	profiles := frd.configMgr.ListProfiles()
	return len(profiles) == 0
}

// AutoTriggerSetup automatically shows setup for first-time users
func (frd *FirstRunDetector) AutoTriggerSetup() error {
	if !frd.IsFirstRun() {
		return nil
	}

	// Check if there's a flag to skip first-run detection
	if os.Getenv("R2GO2_SKIP_FIRST_RUN") == "true" {
		return nil
	}

	ClearScreen()
	fmt.Println()
	fmt.Println(Bold("🚀 Welcome to R2Go2!"))
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()

	fmt.Println("It looks like this is your first time using R2Go2.")
	fmt.Println("Let's get you set up quickly and easily.")
	fmt.Println()

	fmt.Println("R2Go2 is a professional Cloudflare R2 management tool")
	fmt.Println("that helps you store and manage files in the cloud.")
	fmt.Println()

	fmt.Println(Muted("Features include:"))
	fmt.Println("  • Beautiful CLI with interactive setup")
	fmt.Println("  • Multiple profile support")
	fmt.Println("  • Secure token management")
	fmt.Println("  • Real-time upload progress")
	fmt.Println("  • Bucket and object management")
	fmt.Println()

	if ConfirmYesNo("Would you like to run the setup wizard now?", true) {
		fmt.Println()
		ShowSpinner("Starting setup wizard...", 1*Second)
		return nil // Let the main function handle setup
	} else {
		fmt.Println()
		PrintInfo("No problem! You can run setup anytime with:")
		fmt.Printf("  %s\n", Info("cosmoflare setup"))
		fmt.Println()
		PrintInfo("For help getting started, see:")
		fmt.Printf("  %s\n", Info("cosmoflare --help"))
		os.Exit(0)
	}

	return nil
}

// ShowQuickStart shows a quick start guide for new users
func (frd *FirstRunDetector) ShowQuickStart() {
	ClearScreen()
	fmt.Println()
	fmt.Println(Bold("🎓 R2Go2 Quick Start Guide"))
	fmt.Println(strings.Repeat("─", 35))
	fmt.Println()

	fmt.Println(Info("1. Setup Your First Profile"))
	fmt.Println(Muted("   Configure your Cloudflare credentials"))
	fmt.Printf("   %s\n", Info("cosmoflare setup"))
	fmt.Println()

	fmt.Println(Info("2. Create a Bucket"))
	fmt.Println(Muted("   Buckets store your files in the cloud"))
	fmt.Printf("   %s\n", Info("cosmoflare bucket create my-first-bucket"))
	fmt.Println()

	fmt.Println(Info("3. Upload Files"))
	fmt.Println(Muted("   Upload local files to your bucket"))
	fmt.Printf("   %s\n", Info("cosmoflare upload ./local-file.txt my-first-bucket"))
	fmt.Println()

	fmt.Println(Info("4. List Your Content"))
	fmt.Println(Muted("   See what's in your buckets"))
	fmt.Printf("   %s\n", Info("cosmoflare object list my-first-bucket"))
	fmt.Println()

	fmt.Println(Info("5. Manage Profiles"))
	fmt.Println(Muted("   Switch between different accounts"))
	fmt.Printf("   %s\n", Info("cosmoflare setup --switch"))
	fmt.Println()

	PrintInfo("📚 Ready to learn more?")
	fmt.Printf("  • Documentation: %s\n", Info("https://github.com/CosmoLabs-org/CosmoDev-R2Go2"))
	fmt.Printf("  • Advanced help: %s\n", Info("cosmoflare --help"))
	fmt.Println()

	PauseAndWait("Press Enter to continue...")
}

// DetectAndSetup checks for first run and triggers appropriate setup
func DetectAndSetup() error {
	detector, err := NewFirstRunDetector()
	if err != nil {
		return fmt.Errorf("failed to create first-run detector: %w", err)
	}

	if detector.IsFirstRun() {
		return detector.AutoTriggerSetup()
	}

	return nil
}

// ShowFirstRunWelcome shows a comprehensive welcome for new users
func ShowFirstRunWelcome() {
	ClearScreen()
	fmt.Println()
	fmt.Println(Bold("🌟 Welcome to R2Go2 - Professional Cloudflare R2 Management"))
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()

	fmt.Println(Bold("What is R2Go2?"))
	fmt.Println("R2Go2 is a command-line tool that makes managing Cloudflare R2")
	fmt.Println("simple, secure, and professional. Think of it as the 'aws-cli' for")
	fmt.Println("Cloudflare R2, but with better UX and designed for modern workflows.")
	fmt.Println()

	fmt.Println(Bold("Why R2Go2?"))
	fmt.Printf("%s Beautiful, intuitive interface\n", Success("✓"))
	fmt.Printf("%s Multiple profile support\n", Success("✓"))
	fmt.Printf("%s Secure credential management\n", Success("✓"))
	fmt.Printf("%s Real-time progress tracking\n", Success("✓"))
	fmt.Printf("%s Professional error handling\n", Success("✓"))
	fmt.Printf("%s Works great over SSH\n", Success("✓"))
	fmt.Println()

	fmt.Println(Bold("Getting Started is Easy:"))
	fmt.Println("1. Run the setup wizard to configure your Cloudflare account")
	fmt.Println("2. Create buckets to organize your files")
	fmt.Println("3. Upload, download, and manage your content")
	fmt.Println()

	fmt.Println(Bold("Quick Commands to Know:"))
	fmt.Printf("  %s - Run setup wizard\n", Info("cosmoflare setup"))
	fmt.Printf("  %s - Switch between profiles\n", Info("cosmoflare setup --switch"))
	fmt.Printf("  %s - List all buckets\n", Info("cosmoflare bucket list"))
	fmt.Printf("  %s - Create a new bucket\n", Info("cosmoflare bucket create <name>"))
	fmt.Printf("  %s - Upload files\n", Info("cosmoflare upload <file> <bucket>"))
	fmt.Println()

	PrintInfo("🚀 Let's get you set up!")
	if ConfirmYesNo("Start the interactive setup wizard?", true) {
		ShowSpinner("Starting setup...", 1*Second)
	} else {
		fmt.Println()
		PrintInfo("You can start setup anytime with 'cosmoflare setup'")
		os.Exit(0)
	}
}