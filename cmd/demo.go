/*
Package cmd provides demo command for enhanced CLI visual features

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cmd

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/spf13/cobra"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/cli/visual"
)

// demoCmd represents the demo command
var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Showcase enhanced CLI visual features",
	Long: `Demonstrates the beautiful visual features of R2Go2 CLI including
animations, progress bars, and interactive effects.

Available demos:
  - startup     - Show startup animation
  - spinner      - Show animated spinner
  - progress     - Show progress bar animation
  - success      - Show success animation
  - rainbow      - Show rainbow text effect
  - dashboard    - Show sample dashboard
  - typewriter   - Show typewriter effect
  - pulse        - Show pulsing text effect
  - random       - Show random animation

Examples:
  r2go2 demo startup
  r2go2 demo spinner
  r2go2 demo dashboard
  r2go2 demo random`,
	Run: runDemo,
}

var demoType string

func init() {
	rootCmd.AddCommand(demoCmd)

	demoCmd.Flags().StringVarP(&demoType, "type", "t", "startup", "Demo type to show")
}

func runDemo(cmd *cobra.Command, args []string) {
	fmt.Println()
	visual.ShowSuccess("R2Go2 Enhanced CLI Demo")
	fmt.Println()

	switch demoType {
	case "startup":
		demoStartup()
	case "spinner":
		demoSpinner()
	case "progress":
		demoProgress()
	case "success":
		demoSuccess()
	case "rainbow":
		demoRainbow()
	case "dashboard":
		demoDashboard()
	case "typewriter":
		demoTypewriter()
	case "pulse":
		demoPulse()
	case "random":
		demoRandom()
	default:
		fmt.Println("Available demo types:")
		fmt.Println("  startup     - Show startup animation")
		fmt.Println("  spinner      - Show animated spinner")
		fmt.Println("  progress     - Show progress bar animation")
		fmt.Println("  success      - Show success animation")
		fmt.Println("  rainbow      - Show rainbow text effect")
		fmt.Println("  dashboard    - Show sample dashboard")
		fmt.Println("  typewriter   - Show typewriter effect")
		fmt.Println("  pulse        - Show pulsing text effect")
		fmt.Println("  random       - Show random animation")
		fmt.Println()
		fmt.Printf("Use: r2go2 demo --type <demo_type>\n")
	}
}

func demoStartup() {
	visual.ShowStartupAnimation()
}

func demoSpinner() {
	visual.ShowSpinner("Processing your request...", 3*time.Second)
}

func demoProgress() {
	fmt.Println("Upload progress demonstration:")
	total := int64(100000000) // 100MB
	for i := 0; i <= 10; i++ {
		current := int64(i) * total / 10
		visual.ShowProgress(current, total, "Uploading file.zip")
		time.Sleep(200 * time.Millisecond)
	}
	visual.ShowSuccess("Upload completed successfully!")
}

func demoSuccess() {
	visual.ShowSuccess("Operation completed successfully!")
	visual.ShowSuccess("All files processed!")
	visual.ShowSuccess("Deployment ready!")
}

func demoRainbow() {
	effects := visual.NewTerminalEffects(visual.DefaultTheme())
	texts := []string{
		"🚀 R2Go2 - Enhanced CLI",
		"⚡ Professional Operations",
		"🎯 Real-time Monitoring",
		"✨ Visual Experience",
	}

	for _, text := range texts {
		effects.RainbowText(text)
		time.Sleep(500 * time.Millisecond)
	}
}

func demoDashboard() {
	metrics := map[string]interface{}{
		"Status":      "Running",
		"Uptime":      "2h 15m 32s",
		"Operations":  "1,247",
		"Speed":       "87.3 MB/s",
		"Completed":   "1,198",
		"Failed":      "3",
		"CPU Usage":   "12%",
		"Memory":      "245 MB",
	}

	visual.ShowDashboard("📊 Live Dashboard", metrics)

	// Update dashboard live
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)
		metrics["Completed"] = fmt.Sprintf("%d", 1198+i)
		metrics["Speed"] = fmt.Sprintf("%.1f MB/s", 87.3+rand.Float64()*5)
	}
}

func demoTypewriter() {
	effects := visual.NewTerminalEffects(visual.DefaultTheme())
	texts := []string{
		"Welcome to R2Go2 Enhanced CLI",
		"Experience the power of visual feedback",
		"Professional file operations with real-time monitoring",
		"Beautiful animations and progress tracking",
	}

	for _, text := range texts {
		effects.AnimatedTypewriter(text, 50*time.Millisecond)
		time.Sleep(500 * time.Millisecond)
	}
}

func demoPulse() {
	effects := visual.NewTerminalEffects(visual.DefaultTheme())
	effects.PulseAnimation("PULSING EFFECT", 3)
}

func demoRandom() {
	fmt.Println("🎲 Showing random animations...")
	time.Sleep(1 * time.Second)

	for i := 0; i < 3; i++ {
		fmt.Printf("\n--- Animation %d ---\n", i+1)
		time.Sleep(500 * time.Millisecond)
		visual.RandomAnimation()
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\n✨ Demo completed! Try 'r2go2 demo --type <type>' for specific animations")
}