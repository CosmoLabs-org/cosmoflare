/*
Package interactive provides onboarding tutorial functionality

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package interactive

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// TutorialState represents the current tutorial state
type TutorialState struct {
	CurrentLesson int
	TotalLessons  int
	Completed     []int
	Skipped       []int
	Progress      float64
	StartTime     time.Time
}

// Tutorial represents a single tutorial lesson
type Tutorial struct {
	ID          int
	Title       string
	Description string
	Content     []string
	Actions     []TutorialAction
	Interactive bool
	Required    bool
}

// TutorialAction represents an action the user can take in a tutorial
type TutorialAction struct {
	ID          string
	Label       string
	Description string
	Handler     func() error
	SkipAllowed bool
}

// TutorialManager manages the onboarding tutorial system
type TutorialManager struct {
	tutorials []Tutorial
	state     *TutorialState
	config    *TutorialConfig
}

// TutorialConfig holds tutorial configuration
type TutorialConfig struct {
	AutoStart     bool
	AllowSkip     bool
	SaveProgress  bool
	Timeout       time.Duration
	ShowHints     bool
	VerboseMode   bool
	Interactive   bool
}

// NewTutorialManager creates a new tutorial manager
func NewTutorialManager() *TutorialManager {
	return &TutorialManager{
		tutorials: createDefaultTutorials(),
		state: &TutorialState{
			CurrentLesson: 0,
			TotalLessons:  0,
			Completed:     make([]int, 0),
			Skipped:       make([]int, 0),
			Progress:      0.0,
			StartTime:     time.Now(),
		},
		config: &TutorialConfig{
			AutoStart:     true,
			AllowSkip:     true,
			SaveProgress:  true,
			Timeout:       5 * time.Minute,
			ShowHints:     true,
			VerboseMode:   false,
			Interactive:   true,
		},
	}
}

// createDefaultTutorials creates the default tutorial lessons
func createDefaultTutorials() []Tutorial {
	return []Tutorial{
		{
			ID:          1,
			Title:       "Understanding Profiles",
			Description: "Learn how profiles work and why they're important",
			Content: []string{
				"Profiles store your Cloudflare account information securely.",
				"",
				"Each profile contains:",
				"• Account ID for your Cloudflare account",
				"• API token for authentication",
				"• Region preferences",
				"• Custom settings",
				"",
				"You can have multiple profiles for different accounts or environments.",
				"This is perfect for separating work and personal projects!",
			},
			Actions: []TutorialAction{
				{
					ID:          "view_profiles",
					Label:       "View Your Profiles",
					Description: "List all configured profiles",
					SkipAllowed: true,
				},
				{
					ID:          "create_profile",
					Label:       "Practice Creating",
					Description: "Try creating a test profile",
					SkipAllowed: true,
				},
			},
			Interactive: true,
			Required:    true,
		},
		{
			ID:          2,
			Title:       "Your First Bucket",
			Description: "Learn about R2 buckets and how to create them",
			Content: []string{
				"Buckets are containers for your files in Cloudflare R2.",
				"",
				"Think of them like folders in the cloud:",
				"• Organize files by project or type",
				"• Set access permissions",
				"• Configure custom domains",
				"• Monitor storage usage",
				"",
				"Best practices:",
				"• Use descriptive names",
				"• Keep related files together",
				"• Regular cleanup of old files",
			},
			Actions: []TutorialAction{
				{
					ID:          "list_buckets",
					Label:       "List Buckets",
					Description: "See your existing buckets",
					SkipAllowed: true,
				},
				{
					ID:          "create_bucket",
					Label:       "Create Test Bucket",
					Description: "Practice creating a bucket",
					SkipAllowed: true,
				},
			},
			Interactive: true,
			Required:    true,
		},
		{
			ID:          3,
			Title:       "Uploading Files",
			Description: "Learn how to upload files to your buckets",
			Content: []string{
				"Uploading files is easy with R2Go2!",
				"",
				"Upload features:",
				"• Single file uploads",
				"• Batch directory uploads",
				"• Progress tracking",
				"• Automatic retries",
				"• Checksum verification",
				"",
				"Common upload patterns:",
				"• Website assets (images, CSS, JS)",
				"• Application binaries",
				"• Data backups",
				"• Media files",
			},
			Actions: []TutorialAction{
				{
					ID:          "upload_file",
					Label:       "Upload Test File",
					Description: "Try uploading a file",
					SkipAllowed: true,
				},
				{
					ID:          "view_upload_help",
					Label:       "View Upload Help",
					Description: "Learn upload options",
					SkipAllowed: true,
				},
			},
			Interactive: true,
			Required:    true,
		},
		{
			ID:          4,
			Title:       "Managing Objects",
			Description: "Learn to manage files (objects) in your buckets",
			Content: []string{
				"Objects are individual files stored in your buckets.",
				"",
				"Object operations:",
				"• List and search objects",
				"• Download files",
				"• Delete objects",
				"• Copy between buckets",
				"• Generate signed URLs",
				"",
				"Advanced features:",
				"• Metadata management",
				"• Object versioning",
				"• Lifecycle policies",
				"• Access control",
			},
			Actions: []TutorialAction{
				{
					ID:          "list_objects",
					Label:       "List Objects",
					Description: "Browse files in a bucket",
					SkipAllowed: true,
				},
				{
					ID:          "object_info",
					Label:       "Object Details",
					Description: "View file information",
					SkipAllowed: true,
				},
			},
			Interactive: true,
			Required:    false,
		},
		{
			ID:          5,
			Title:       "Advanced Tips & Tricks",
			Description: "Learn advanced R2Go2 features",
			Content: []string{
				"Take your R2Go2 skills to the next level!",
				"",
				"Advanced features:",
				"• Profile switching and management",
				"• Backup and restore configurations",
				"• Custom themes and accessibility",
				"• Command completion",
				"• Output formatting (JSON, table)",
				"",
				"Productivity tips:",
				"• Use shell aliases for frequent commands",
				"• Create bash functions for complex workflows",
				"• Enable verbose output for debugging",
				"• Use profiles for different environments",
			},
			Actions: []TutorialAction{
				{
					ID:          "try_switching",
					Label:       "Try Profile Switching",
					Description: "Practice switching profiles",
					SkipAllowed: true,
				},
				{
					ID:          "view_help",
					Label:       "Explore Help",
					Description: "Browse help topics",
					SkipAllowed: true,
				},
			},
			Interactive: true,
			Required:    false,
		},
	}
}

// StartTutorial begins the tutorial sequence
func (tm *TutorialManager) StartTutorial() error {
	tm.state.TotalLessons = len(tm.tutorials)
	tm.state.StartTime = time.Now()

	// Show tutorial introduction
	tm.showTutorialIntro()

	for i, tutorial := range tm.tutorials {
		tm.state.CurrentLesson = i + 1

		// Check if tutorial should be shown
		if tm.shouldSkipTutorial(tutorial.ID) {
			continue
		}

		// Show the tutorial
		if err := tm.showTutorialLesson(tutorial); err != nil {
			if err == ErrTutorialSkipped {
				tm.state.Skipped = append(tm.state.Skipped, tutorial.ID)
				continue
			}
			return err
		}

		tm.state.Completed = append(tm.state.Completed, tutorial.ID)
		tm.updateProgress()
	}

	// Show tutorial completion
	tm.showTutorialCompletion()

	return nil
}

// showTutorialIntro displays the tutorial introduction
func (tm *TutorialManager) showTutorialIntro() {
	ClearScreen()
	fmt.Println()
	fmt.Println(Bold("🎓 R2Go2 Interactive Tutorial"))
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()

	fmt.Println("Welcome to the R2Go2 tutorial! This will help you get started")
	fmt.Println("with managing your Cloudflare R2 storage effectively.")
	fmt.Println()

	fmt.Printf("📚 %d lessons planned\n", len(tm.tutorials))
	fmt.Printf("⏱️  Estimated time: %d minutes\n", len(tm.tutorials)*2)
	fmt.Println()

	fmt.Println("What you'll learn:")
	for _, tutorial := range tm.tutorials {
		if tutorial.Required {
			fmt.Printf("  • %s\n", tutorial.Title)
		}
	}
	fmt.Println()

	if tm.config.AllowSkip {
		fmt.Println("💡 Tips:")
		fmt.Println("  • Press Enter to continue through content")
		fmt.Println("  • Type 'skip' to skip optional lessons")
		fmt.Println("  • Type 'quit' to exit tutorial anytime")
		fmt.Println()
	}

	fmt.Println("Ready to start learning?")
	if ConfirmYesNo("Begin tutorial now?", true) {
		fmt.Println()
		ShowSpinner("Starting tutorial...", 1*Second)
	} else {
		PrintInfo("You can start the tutorial anytime with:")
		fmt.Printf("  %s\n", Info("r2go2 tutorial"))
		os.Exit(0)
	}
}

// showTutorialLesson displays a single tutorial lesson
func (tm *TutorialManager) showTutorialLesson(tutorial Tutorial) error {
	// Show lesson header
	ClearScreen()
	fmt.Println()
	fmt.Printf("📚 Lesson %d/%d: %s\n", tm.state.CurrentLesson, tm.state.TotalLessons, tutorial.Title)
	fmt.Printf("%s\n", Dim(tutorial.Description))
	fmt.Println(strings.Repeat("─", 60))

	// Display content with typing effect
	if tm.config.VerboseMode {
		for _, line := range tutorial.Content {
			fmt.Println()
			globalAnimator.TypewriterEffect(line, 20*time.Millisecond)
		}
	} else {
		for _, line := range tutorial.Content {
			fmt.Printf("  %s\n", line)
		}
	}

	fmt.Println()

	// Show interactive actions if available
	if tutorial.Interactive && len(tutorial.Actions) > 0 {
		return tm.handleTutorialActions(tutorial)
	}

	// Simple continue for non-interactive lessons
	fmt.Println()
	PauseAndWait("▶️ Press Enter to continue...")
	return nil
}

// handleTutorialActions handles interactive tutorial actions
func (tm *TutorialManager) handleTutorialActions(tutorial Tutorial) error {
	fmt.Println(Bold("🎯 Practice Actions:"))
	for i, action := range tutorial.Actions {
		fmt.Printf("  [%d] %s\n", i+1, action.Label)
		if action.Description != "" {
			fmt.Printf("      %s\n", Dim(action.Description))
		}
	}

	if tm.config.AllowSkip {
		fmt.Printf("  [s] Skip this lesson\n")
	}
	fmt.Printf("  [q] Quit tutorial\n")

	fmt.Println()
	fmt.Printf("Choose action [1]: ")

	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(strings.ToLower(input))

	// Handle input
	switch input {
	case "":
		// Default to first action
		if len(tutorial.Actions) > 0 {
			return tm.executeAction(tutorial.Actions[0])
		}
	case "s":
		if tm.config.AllowSkip {
			PrintInfo("Lesson skipped.")
			return ErrTutorialSkipped
		}
	case "q":
		PrintInfo("Tutorial exited.")
		os.Exit(0)
	default:
		// Try to parse as number
		choice, err := strconv.Atoi(input)
		if err == nil && choice >= 1 && choice <= len(tutorial.Actions) {
			return tm.executeAction(tutorial.Actions[choice-1])
		}
		PrintError("Invalid selection.")
		return tm.handleTutorialActions(tutorial)
	}

	return nil
}

// executeAction executes a tutorial action
func (tm *TutorialManager) executeAction(action TutorialAction) error {
	fmt.Printf("\n🚀 %s\n", action.Label)
	fmt.Println(strings.Repeat("─", len(action.Label)+4))
	fmt.Println()

	switch action.ID {
	case "view_profiles":
		PrintInfo("To view your profiles, run:")
		fmt.Printf("  %s\n", Info("r2go2 config list"))
		fmt.Println()
		PrintInfo("Try it in another terminal window!")

	case "create_profile":
		PrintInfo("To create a new profile, run:")
		fmt.Printf("  %s\n", Info("r2go2 setup --profile=tutorial-test"))
		fmt.Println()
		PrintInfo("Use your test credentials or create a mock profile for practice.")

	case "list_buckets":
		PrintInfo("To list your buckets, run:")
		fmt.Printf("  %s\n", Info("r2go2 bucket list"))
		fmt.Println()

	case "create_bucket":
		PrintInfo("To create a bucket, run:")
		fmt.Printf("  %s\n", Info("r2go2 bucket create tutorial-test"))
		fmt.Println()

	case "upload_file":
		PrintInfo("To upload a file, run:")
		fmt.Printf("  %s\n", Info("r2go2 upload ./test.txt your-bucket"))
		fmt.Println()
		PrintInfo("Create a test file first: echo 'test' > test.txt")

	case "list_objects":
		PrintInfo("To list objects in a bucket, run:")
		fmt.Printf("  %s\n", Info("r2go2 object list your-bucket"))
		fmt.Println()

	case "try_switching":
		PrintInfo("To switch profiles, run:")
		fmt.Printf("  %s\n", Info("r2go2 setup --switch"))
		fmt.Println()

	case "view_help":
		PrintInfo("To see all help topics, run:")
		fmt.Printf("  %s\n", Info("r2go2 --help"))
		fmt.Println()

	default:
		PrintInfo("This action demonstrates an important concept.")
	}

	fmt.Println()
	PauseAndWait("Press Enter when done...")
	return nil
}

// shouldSkipTutorial checks if a tutorial should be skipped
func (tm *TutorialManager) shouldSkipTutorial(tutorialID int) bool {
	// Skip if already completed
	for _, completed := range tm.state.Completed {
		if completed == tutorialID {
			return true
		}
	}
	return false
}

// updateProgress updates the tutorial progress
func (tm *TutorialManager) updateProgress() {
	completed := len(tm.state.Completed)
	total := tm.state.TotalLessons
	tm.state.Progress = float64(completed) / float64(total)
}

// showTutorialCompletion displays the tutorial completion screen
func (tm *TutorialManager) showTutorialCompletion() {
	duration := time.Since(tm.state.StartTime)
	ClearScreen()

	fmt.Println()
	fmt.Println("🎉 " + Bold("Tutorial Complete!"))
	fmt.Println(strings.Repeat("─", 30))
	fmt.Println()

	fmt.Printf("✅ Completed lessons: %d/%d\n", len(tm.state.Completed), tm.state.TotalLessons)
	fmt.Printf("⏱️  Time spent: %v\n", duration.Round(time.Second))
	fmt.Printf("📊 Progress: %.1f%%\n", tm.state.Progress*100)

	if len(tm.state.Skipped) > 0 {
		fmt.Printf("⏭️  Skipped lessons: %d\n", len(tm.state.Skipped))
	}

	fmt.Println()
	fmt.Println(Bold("🚀 You're ready to use R2Go2!"))
	fmt.Println()

	fmt.Println("Quick reference:")
	fmt.Printf("  • %s - List buckets\n", Info("r2go2 bucket list"))
	fmt.Printf("  • %s - Upload files\n", Info("r2go2 upload"))
	fmt.Printf("  • %s - Switch profiles\n", Info("r2go2 setup --switch"))
	fmt.Printf("  • %s - View all commands\n", Info("r2go2 --help"))
	fmt.Println()

	fmt.Println("Need more help?")
	fmt.Printf("  • Documentation: %s\n", Info("https://github.com/CosmoLabs-org/CosmoDev-R2Go2"))
	fmt.Printf("  • Tutorial replay: %s\n", Info("r2go2 tutorial"))
	fmt.Printf("  • Advanced help: %s\n", Info("r2go2 help advanced"))

	globalAnimator.ShowSuccessAnimation("Tutorial completed successfully!")
}

// ShowQuickStart shows a quick start guide
func (tm *TutorialManager) ShowQuickStart() {
	ClearScreen()
	fmt.Println()
	fmt.Println(Bold("🚀 R2Go2 Quick Start"))
	fmt.Println(strings.Repeat("─", 25))
	fmt.Println()

	fmt.Println("Essential commands to get you started:")
	fmt.Println()

	fmt.Println("1. Setup your first profile")
	fmt.Printf("   %s\n", Info("r2go2 setup"))
	fmt.Println()

	fmt.Println("2. Create a bucket for your files")
	fmt.Printf("   %s\n", Info("r2go2 bucket create my-first-bucket"))
	fmt.Println()

	fmt.Println("3. Upload files to your bucket")
	fmt.Printf("   %s\n", Info("r2go2 upload ./file.txt my-first-bucket"))
	fmt.Println()

	fmt.Println("4. List your files")
	fmt.Printf("   %s\n", Info("r2go2 object list my-first-bucket"))
	fmt.Println()

	fmt.Println("5. Switch between profiles")
	fmt.Printf("   %s\n", Info("r2go2 setup --switch"))
	fmt.Println()

	PrintInfo("💡 Want to learn more? Run 'r2go2 tutorial' for the full guide!")
}

// Custom error for skipped tutorials
var ErrTutorialSkipped = fmt.Errorf("tutorial skipped")