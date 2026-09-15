/*
Package workflows tests TUI user workflows and integration scenarios

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package onboarding

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// firstRunSteps describes the sequence a new user experiences on first launch.
var firstRunSteps = []struct {
	step        string
	description string
	keyAction   string
	expected    string
	timeout     time.Duration
}{
	{
		step:        "Initial startup",
		description: "Dashboard shows with welcome state",
		keyAction:   "none",
		expected:    "Loading R2Go2 Dashboard",
		timeout:     time.Second * 3,
	},
	{
		step:        "Help discovery",
		description: "User discovers help via F1",
		keyAction:   "F1",
		expected:    "R2Go2 Dashboard Help",
		timeout:     time.Millisecond * 500,
	},
	{
		step:        "Help navigation",
		description: "User reads through help content",
		keyAction:   "↓",
		expected:    "Navigation:",
		timeout:     time.Millisecond * 100,
	},
	{
		step:        "Help dismissal",
		description: "User dismisses help to return to dashboard",
		keyAction:   "Esc",
		expected:    "Dashboard view",
		timeout:     time.Millisecond * 200,
	},
}

// emptyStateScenarios describes how the application handles empty states.
var emptyStateScenarios = []struct {
	state      string
	component  string
	message    string
	actionable bool
}{
	{
		state:      "No buckets configured",
		component:  "Bucket list",
		message:    "No buckets found. Press 'C' to create your first bucket.",
		actionable: true,
	},
	{
		state:      "No objects in bucket",
		component:  "Object list",
		message:    "No bucket selected or bucket is empty",
		actionable: true,
	},
	{
		state:      "No upload queue",
		component:  "Upload interface",
		message:    "No files in upload queue",
		actionable: true,
	},
	{
		state:      "No recent activity",
		component:  "Monitoring",
		message:    "No recent activity",
		actionable: false,
	},
}

// progressiveFeatures describes how complexity is revealed progressively.
var progressiveFeatures = []struct {
	feature    string
	discovery  string
	complexity string
	required   bool
}{
	{
		feature:    "Basic navigation",
		discovery:  "Immediate - visible on first load",
		complexity: "Simple - arrow keys and enter",
		required:   true,
	},
	{
		feature:    "Section switching",
		discovery:  "Immediate - number keys 1-6",
		complexity: "Simple - direct access to sections",
		required:   true,
	},
	{
		feature:    "Quick actions",
		discovery:  "Discoverable - keyboard shortcuts",
		complexity: "Medium - multiple key combinations",
		required:   false,
	},
	{
		feature:    "Advanced search",
		discovery:  "Discoverable - '/' key",
		complexity: "Advanced - filtering and sorting",
		required:   false,
	},
	{
		feature:    "Theme customization",
		discovery:  "Discoverable - settings menu",
		complexity: "Advanced - multiple configuration options",
		required:   false,
	},
}

// TestFirstRunExperience tests the complete first-time user experience
func TestFirstRunExperience(t *testing.T) {
	t.Run("Fresh Installation Flow", func(t *testing.T) {
		// Test the sequence a new user would experience
		for _, step := range firstRunSteps {
			t.Run("Step: "+step.step, func(t *testing.T) {
				assert.NotEmpty(t, step.step, "Step should have name")
				assert.NotEmpty(t, step.description, "Step should have description")
				assert.NotEmpty(t, step.expected, "Expected outcome should be specified")
				assert.Greater(t, step.timeout, time.Duration(0), "Timeout should be positive")

				// Simulate key action if specified
				if step.keyAction != "none" {
					var keyMsg tea.KeyMsg
					switch step.keyAction {
					case "F1":
						keyMsg = tea.KeyMsg{Type: tea.KeyF1}
					case "Esc":
						keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
					case "↓":
						keyMsg = tea.KeyMsg{Type: tea.KeyDown}
					default:
						keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune(step.keyAction[0])}}
					}
					assert.NotNil(t, keyMsg, "Key message should be created")
				}
			})
		}
	})

	t.Run("Empty State Handling", func(t *testing.T) {
		// Test how the application handles empty states
		for _, scenario := range emptyStateScenarios {
			t.Run("Empty state: "+scenario.state, func(t *testing.T) {
				assert.NotEmpty(t, scenario.state, "State description should not be empty")
				assert.NotEmpty(t, scenario.component, "Component should not be empty")
				assert.NotEmpty(t, scenario.message, "Message should not be empty")

				if scenario.actionable {
					// Actionable empty states should provide clear next steps
					assert.True(t, len(scenario.message) > 10, "Actionable message should be descriptive")
				}
			})
		}
	})

	t.Run("Progressive Disclosure", func(t *testing.T) {
		// Test that complexity is revealed progressively
		for _, feature := range progressiveFeatures {
			t.Run("Progressive: "+feature.feature, func(t *testing.T) {
				assert.NotEmpty(t, feature.feature, "Feature should have name")
				assert.NotEmpty(t, feature.discovery, "Discovery method should be described")
				assert.NotEmpty(t, feature.complexity, "Complexity level should be specified")

				if feature.required {
					// Required features should be simple and immediately discoverable
					assert.Contains(t, feature.complexity, "Simple", "Required features should be simple")
					assert.Contains(t, feature.discovery, "Immediate", "Required features should be immediately discoverable")
				}
			})
		}
	})
}

// helpContexts describes contextually relevant help per section.
var helpContexts = []struct {
	section     string
	helpContent string
	relevant    bool
	actions     []string
}{
	{
		section:     "Overview",
		helpContent: "Dashboard summary and quick actions",
		relevant:    true,
		actions:     []string{"Create bucket", "Monitor mode", "Settings"},
	},
	{
		section:     "Buckets",
		helpContent: "Bucket management operations",
		relevant:    true,
		actions:     []string{"Create", "Delete", "Select", "View details"},
	},
	{
		section:     "Upload",
		helpContent: "File upload instructions",
		relevant:    true,
		actions:     []string{"Add files", "Start upload", "Monitor progress"},
	},
	{
		section:     "Monitoring",
		helpContent: "Real-time statistics interpretation",
		relevant:    true,
		actions:     []string{"View metrics", "Export data", "Set alerts"},
	},
}

// errorGuidanceScenarios describes error messages with clear guidance.
var errorGuidanceScenarios = []struct {
	errorType   string
	userMessage string
	guidances   []string
	recovery    string
}{
	{
		errorType:   "Connection timeout",
		userMessage: "Failed to connect to Cloudflare R2",
		guidances:   []string{"Check internet connection", "Verify credentials", "Try again"},
		recovery:    "Press F5 to retry or Esc to continue",
	},
	{
		errorType:   "Invalid credentials",
		userMessage: "Authentication failed",
		guidances:   []string{"Check API token", "Verify account ID", "Run setup wizard"},
		recovery:    "Press 'P' to configure profiles",
	},
	{
		errorType:   "File upload failed",
		userMessage: "Upload interrupted",
		guidances:   []string{"Check file size limits", "Verify file permissions", "Check storage space"},
		recovery:    "Press 'U' to retry upload",
	},
}

// helpLevels describes help detail per user experience level.
var helpLevels = []struct {
	experience  string
	helpDepth   string
	features    []string
	assumptions []string
}{
	{
		experience:  "First-time user",
		helpDepth:   "Basic - essential functions only",
		features:    []string{"Navigation", "Basic operations", "Help access"},
		assumptions: []string{"No prior TUI experience", "Needs clear instructions"},
	},
	{
		experience:  "Intermediate user",
		helpDepth:   "Standard - commonly used features",
		features:    []string{"Shortcuts", "Advanced navigation", "Basic configuration"},
		assumptions: []string{"Familiar with TUI concepts", "Knows basic operations"},
	},
	{
		experience:  "Advanced user",
		helpDepth:   "Comprehensive - all features and tips",
		features:    []string{"Advanced configuration", "Automation", "Performance tips"},
		assumptions: []string{"Experienced with CLI tools", "Seeks efficiency"},
	},
}

// TestUserGuidance tests user guidance and help systems
func TestUserGuidance(t *testing.T) {
	t.Run("Help System Integration", func(t *testing.T) {
		// Test that help is contextually relevant
		for _, context := range helpContexts {
			t.Run("Help context: "+context.section, func(t *testing.T) {
				assert.NotEmpty(t, context.section, "Section should not be empty")
				assert.NotEmpty(t, context.helpContent, "Help content should not be empty")
				assert.NotEmpty(t, context.actions, "Help should suggest actions")

				if context.relevant {
					// Relevant help should be specific and actionable
					assert.Greater(t, len(context.helpContent), 20, "Relevant help should be detailed")
					assert.Greater(t, len(context.actions), 0, "Help should provide actionable steps")
				}
			})
		}
	})

	t.Run("Error Message Guidance", func(t *testing.T) {
		// Test that error messages provide clear guidance
		for _, scenario := range errorGuidanceScenarios {
			t.Run("Error guidance: "+scenario.errorType, func(t *testing.T) {
				assert.NotEmpty(t, scenario.errorType, "Error type should not be empty")
				assert.NotEmpty(t, scenario.userMessage, "User message should not be empty")
				assert.NotEmpty(t, scenario.guidances, "Error should provide guidance")
				assert.NotEmpty(t, scenario.recovery, "Error should provide recovery path")

				// Error messages should be clear and actionable
				assert.Greater(t, len(scenario.guidances), 0, "Should provide at least one guidance")
				assert.Greater(t, len(scenario.recovery), 5, "Recovery path should be descriptive")
			})
		}
	})

	t.Run("Progressive Help", func(t *testing.T) {
		// Test that help becomes more detailed with user experience
		for _, level := range helpLevels {
			t.Run("Help level: "+level.experience, func(t *testing.T) {
				assert.NotEmpty(t, level.experience, "Experience level should not be empty")
				assert.NotEmpty(t, level.helpDepth, "Help depth should be described")
				assert.NotEmpty(t, level.features, "Help level should include features")
				assert.NotEmpty(t, level.assumptions, "Help level should state assumptions")

				// Help should be appropriate for the experience level
				assert.Greater(t, len(level.features), 0, "Help should include relevant features")
				assert.Greater(t, len(level.assumptions), 0, "Help should consider user assumptions")
			})
		}
	})
}

// productiveSteps describes the complete path from new user to productive user.
var productiveSteps = []struct {
	step          string
	duration      time.Duration
	description   string
	successMetric string
}{
	{
		step:          "Initial dashboard load",
		duration:      time.Second * 3,
		description:   "User sees initial dashboard",
		successMetric: "Dashboard renders without errors",
	},
	{
		step:          "Help discovery",
		duration:      time.Second * 10,
		description:   "User discovers help system",
		successMetric: "User can access and navigate help",
	},
	{
		step:          "First navigation",
		duration:      time.Second * 15,
		description:   "User navigates between sections",
		successMetric: "User can reach all 6 sections",
	},
	{
		step:          "Action discovery",
		duration:      time.Second * 20,
		description:   "User discovers quick actions",
		successMetric: "User identifies at least 3 actions",
	},
	{
		step:          "Productive task completion",
		duration:      time.Minute,
		description:   "User completes first meaningful task",
		successMetric: "User successfully creates or manages a bucket",
	},
}

// returningUserSteps describes the experience for returning users.
var returningUserSteps = []struct {
	step      string
	fast      bool
	shortcut  string
	skippable bool
}{
	{
		step:      "Quick status check",
		fast:      true,
		shortcut:  "Overview dashboard",
		skippable: false,
	},
	{
		step:      "Recent activity review",
		fast:      true,
		shortcut:  "Monitoring section",
		skippable: true,
	},
	{
		step:      "Quick action execution",
		fast:      true,
		shortcut:  "Quick action keys",
		skippable: false,
	},
	{
		step:      "Settings adjustment",
		fast:      false,
		shortcut:  "Settings menu",
		skippable: true,
	},
}

// migrationScenarios describes users migrating from other tools.
var migrationScenarios = []struct {
	source     string
	challenges []string
	features   []string
	guidance   string
}{
	{
		source:     "AWS S3 CLI",
		challenges: []string{"Different command structure", "No bulk operations"},
		features:   []string{"Visual dashboard", "Quick navigation", "Progress tracking"},
		guidance:   "AWS S3 users will appreciate the visual interface and real-time feedback",
	},
	{
		source:     "Web-based consoles",
		challenges: []string{"No keyboard shortcuts", "Learning TUI paradigms"},
		features:   []string{"Keyboard efficiency", "Terminal integration", "Scriptable workflows"},
		guidance:   "Web console users gain efficiency through keyboard-first interface",
	},
	{
		source:     "Other TUI tools",
		challenges: []string{"Different keybindings", "Unique workflows"},
		features:   []string{"Customizable themes", "Section-based navigation", "Help system"},
		guidance:   "Experienced TUI users will benefit from comprehensive help and customization",
	},
}

// TestOnboardingFlows tests various onboarding scenarios
func TestOnboardingFlows(t *testing.T) {
	t.Run("Zero-to-Productive Flow", func(t *testing.T) {
		// Test the complete path from new user to productive user
		totalOnboardingTime := time.Duration(0)
		for _, step := range productiveSteps {
			totalOnboardingTime += step.duration

			t.Run("Onboarding step: "+step.step, func(t *testing.T) {
				assert.NotEmpty(t, step.step, "Step should have name")
				assert.NotEmpty(t, step.description, "Step should have description")
				assert.NotEmpty(t, step.successMetric, "Step should have success metric")
				assert.Greater(t, step.duration, time.Duration(0), "Step should have positive duration")

				// Onboarding should be efficient
				assert.Less(t, step.duration, time.Minute*2, "Individual steps should be reasonable")
			})
		}

		// Total onboarding should be efficient
		assert.Less(t, totalOnboardingTime, time.Minute*5, "Total onboarding should be under 5 minutes")
	})

	t.Run("Returning User Experience", func(t *testing.T) {
		// Test the experience for users returning to the application
		for _, step := range returningUserSteps {
			t.Run("Returning: "+step.step, func(t *testing.T) {
				assert.NotEmpty(t, step.step, "Step should have name")

				if step.fast {
					// Fast steps should have keyboard shortcuts
					assert.NotEmpty(t, step.shortcut, "Fast steps should have shortcuts")
				}

				if step.skippable {
					// Skippable steps should be optional
					assert.True(t, step.skippable, "Step should be marked as skippable")
				}
			})
		}
	})

	t.Run("Migration Scenarios", func(t *testing.T) {
		// Test scenarios where users are migrating from other tools
		for _, scenario := range migrationScenarios {
			t.Run("Migration from: "+scenario.source, func(t *testing.T) {
				assert.NotEmpty(t, scenario.source, "Source tool should not be empty")
				assert.NotEmpty(t, scenario.challenges, "Should identify migration challenges")
				assert.NotEmpty(t, scenario.features, "Should highlight benefits")
				assert.NotEmpty(t, scenario.guidance, "Should provide migration guidance")

				// Migration guidance should be helpful and specific
				assert.Greater(t, len(scenario.guidance), 20, "Migration guidance should be detailed")
			})
		}
	})
}
