/*
Package visual provides enhanced visual effects and animations for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package visual

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/utils"
	"github.com/charmbracelet/lipgloss"
)

// Animation types
type AnimationType int

const (
	AnimationTypeSpinner AnimationType = iota
	AnimationTypeDots
	AnimationTypePulse
	AnimationTypeWave
	AnimationTypeProgress
	AnimationTypeRainbow
	AnimationTypeGlow
)

// VisualTheme defines color schemes and styles
type VisualTheme struct {
	Primary     lipgloss.Color
	Secondary   lipgloss.Color
	Success     lipgloss.Color
	Warning     lipgloss.Color
	Error       lipgloss.Color
	Info        lipgloss.Color
	Background  lipgloss.Color
	Foreground  lipgloss.Color
}

// DefaultTheme returns a professional color theme
func DefaultTheme() *VisualTheme {
	return &VisualTheme{
		Primary:    lipgloss.Color("#007ACC"),  // Cloudflare blue
		Secondary:  lipgloss.Color("#4A90E2"),  // Light blue
		Success:    lipgloss.Color("#00C851"),  // Green
		Warning:    lipgloss.Color("#FF8800"),  // Orange
		Error:      lipgloss.Color("#FF4444"),  // Red
		Info:       lipgloss.Color("#17A2B8"),  // Cyan
		Background: lipgloss.Color("#1E1E1E"),  // Dark background
		Foreground: lipgloss.Color("#FFFFFF"),  // White text
	}
}

// DarkTheme returns a dark professional theme
func DarkTheme() *VisualTheme {
	return &VisualTheme{
		Primary:    lipgloss.Color("#64B5F6"),  // Light blue
		Secondary:  lipgloss.Color("#42A5F5"),  // Medium blue
		Success:    lipgloss.Color("#66BB6A"),  // Light green
		Warning:    lipgloss.Color("#FFA726"),  // Light orange
		Error:      lipgloss.Color("#EF5350"),  // Light red
		Info:       lipgloss.Color("#26C6DA"),  // Light cyan
		Background: lipgloss.Color("#121212"),  // Very dark
		Foreground: lipgloss.Color("#F5F5F5"),  // Light white
	}
}

// TerminalEffects provides advanced terminal animations
type TerminalEffects struct {
	theme       *VisualTheme
	animating   bool
	currentStep int
	startTime   time.Time
}

// NewTerminalEffects creates a new terminal effects manager
func NewTerminalEffects(theme *VisualTheme) *TerminalEffects {
	if theme == nil {
		theme = DefaultTheme()
	}
	return &TerminalEffects{
		theme:     theme,
		animating: false,
		startTime: time.Now(),
	}
}

// ShowStartupAnimation displays a beautiful startup animation
func (te *TerminalEffects) ShowStartupAnimation() {
	te.animating = true
	defer func() { te.animating = false }()

	// Clear screen
	fmt.Print("\033[2J\033[H")

	// Animate logo or title
	logos := []string{
		"🚀 R2Go2 - Enhanced CLI",
		"⚡ Professional File Operations",
		"🎯 Real-time Progress Monitoring",
	}

	// Animated title display
	for i, logo := range logos {
		te.animatedPrint(logo, 100*time.Millisecond)
		te.newLine()
		if i < len(logos)-1 {
			time.Sleep(200 * time.Millisecond)
		}
	}

	te.newLine()
	te.animatedPrint("Initializing enhanced features...", 50*time.Millisecond)
	time.Sleep(1 * time.Second)

	for i := 0; i < 3; i++ {
		te.printColored(".", te.theme.Info)
		time.Sleep(300 * time.Millisecond)
	}

	te.printColoredLn(" Ready!", te.theme.Success)
	te.newLine()
	time.Sleep(500 * time.Millisecond)
}

// AnimatedSpinner creates an animated spinner with custom styling
func (te *TerminalEffects) AnimatedSpinner(message string, duration time.Duration) {
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	start := time.Now()
	for time.Since(start) < duration {
		for _, char := range spinner {
			if time.Since(start) >= duration {
				break
			}

			fmt.Printf("\r%s %s",
				te.styleText(char, te.theme.Primary, true),
				te.styleText(message, te.theme.Foreground))

			time.Sleep(100 * time.Millisecond)
		}
	}
	fmt.Printf("\r%s %s %s\n",
		te.styleText("✅", te.theme.Success, true),
		te.styleText(message, te.theme.Foreground),
		te.styleText("Done!", te.theme.Success))
}

// ProgressBar creates an animated progress bar with custom styling
func (te *TerminalEffects) AnimatedProgress(current, total int64, message string) {
	percentage := float64(current) / float64(total) * 100

	// Bar styling
	barWidth := 35
	filled := int(float64(barWidth) * percentage / 100)
	bar := te.createProgressBar(filled, barWidth)

	// Calculate speed (simplified)
	speed := float64(current) / (1024 * 1024) // Approximate MB

	// Format the output professionally
	icon := te.styleText("⚡", te.theme.Primary, true)
	msgStyled := te.styleText(message, te.theme.Foreground)
	sizeProgress := te.styleText(fmt.Sprintf("%s/%s", utils.FormatBytes(current), utils.FormatBytes(total)), te.theme.Secondary)
	pctStyled := te.styleText(fmt.Sprintf("%5.1f%%", percentage), te.theme.Success, true)

	// Main progress line with better formatting
	fmt.Printf("\r%s %s\n   %s %s  %s",
		icon,
		msgStyled,
		bar,
		sizeProgress,
		pctStyled)

	if current >= total {
		// Clear and show completion
		fmt.Printf("\r%s %s %s\n",
			te.styleText("✅", te.theme.Success, true),
			te.styleText(message, te.theme.Success),
			te.styleText("[Complete]", te.theme.Success))
	} else {
		// Move cursor back up for next update
		fmt.Print("\033[1A\r")
	}
	_ = speed // Placeholder for future speed display
}

// createProgressBar generates a styled progress bar
func (te *TerminalEffects) createProgressBar(filled, total int) string {
	var bar strings.Builder
	bar.WriteString("[")

	// Filled portion
	filledBar := strings.Repeat("█", filled)
	bar.WriteString(te.styleText(filledBar, te.theme.Primary))

	// Empty portion
	emptyBar := strings.Repeat("░", total-filled)
	bar.WriteString(te.styleText(emptyBar, te.theme.Secondary))
	bar.WriteString("]")

	return bar.String()
}

// SuccessAnimation displays a beautiful success animation
func (te *TerminalEffects) SuccessAnimation(message string) {
	successChars := []string{"✨", "⭐", "🌟", "💫", "✅"}

	for i, char := range successChars {
		delay := time.Duration(i*100) * time.Millisecond
		time.Sleep(delay)
		fmt.Printf("%s ", te.styleText(char, te.theme.Success, true))
	}

	fmt.Printf("%s %s\n",
		te.styleText("🎉", te.theme.Success, true),
		te.styleText(message, te.theme.Success, true))
}

// ErrorAnimation displays an error with animation
func (te *TerminalEffects) ErrorAnimation(message string) {
	fmt.Printf("%s %s\n",
		te.styleText("❌", te.theme.Error, true),
		te.styleText(message, te.theme.Error))
}

// RainbowText creates rainbow-colored text
func (te *TerminalEffects) RainbowText(text string) {
	colors := []lipgloss.Color{
		lipgloss.Color("#FF0000"), // Red
		lipgloss.Color("#FF7F00"), // Orange
		lipgloss.Color("#FFFF00"), // Yellow
		lipgloss.Color("#00FF00"), // Green
		lipgloss.Color("#0000FF"), // Blue
		lipgloss.Color("#4B0082"), // Indigo
		lipgloss.Color("#9400D3"), // Violet
	}

	for i, char := range text {
		color := colors[i%len(colors)]
		fmt.Print(te.styleText(string(char), color, true))
	}
	fmt.Println()
}

// GlowingText creates a glowing text effect
func (te *TerminalEffects) GlowingText(text string) {
	if len(text) == 0 {
		return
	}
	for i := 0; i < 3; i++ {
		delay := time.Duration(i*200) * time.Millisecond
		time.Sleep(delay)

		intensity := []string{"", "━━", "══"}[i]
		fmt.Printf("\r%s%s%s",
			te.styleText(intensity, te.theme.Primary),
			te.styleText(text, te.theme.Primary, true),
			te.styleText(intensity, te.theme.Primary))
	}
	fmt.Println()
}

// ShowResult displays formatted operation results
func (te *TerminalEffects) ShowResult(title string, details map[string]interface{}) {
	// Box drawing characters
	const (
		topLeft     = "┌"
		topRight    = "┐"
		bottomLeft  = "└"
		bottomRight = "┘"
		horizontal  = "─"
		vertical    = "│"
	)

	boxWidth := 56
	fmt.Println()

	// Top border
	topBorder := topLeft + strings.Repeat(horizontal, boxWidth-2) + topRight
	fmt.Println(te.styleText(topBorder, te.theme.Primary))

	// Title
	titleLine := fmt.Sprintf("%s  %s", vertical, title)
	for len(titleLine) < boxWidth-1 {
		titleLine += " "
	}
	titleLine += vertical
	fmt.Println(te.styleText(titleLine, te.theme.Primary, true))

	// Separator
	fmt.Println(te.styleText(vertical+strings.Repeat(horizontal, boxWidth-2)+vertical, te.theme.Secondary))

	// Details with aligned formatting
	for key, value := range details {
		keyStyled := te.styleText(fmt.Sprintf("%-14s", key+":"), te.theme.Info)
		valueStyled := te.styleText(fmt.Sprintf("%v", value), te.theme.Foreground)
		fmt.Printf("%s  %s %s", te.styleText(vertical, te.theme.Primary), keyStyled, valueStyled)
		// Pad to box width
		content := fmt.Sprintf("  %-14s %v", key+":", value)
		for len(content) < boxWidth-2 {
			fmt.Print(" ")
			content += " "
		}
		fmt.Println(te.styleText(vertical, te.theme.Primary))
	}

	// Bottom border
	bottomBorder := bottomLeft + strings.Repeat(horizontal, boxWidth-2) + bottomRight
	fmt.Println(te.styleText(bottomBorder, te.theme.Primary))
	fmt.Println()
}

// LiveDashboard creates a real-time dashboard display
func (te *TerminalEffects) LiveDashboard(title string, metrics map[string]interface{}) {
	// Box drawing characters for clean borders
	const (
		topLeft     = "╭"
		topRight    = "╮"
		bottomLeft  = "╰"
		bottomRight = "╯"
		horizontal  = "─"
		vertical    = "│"
	)

	boxWidth := 60

	// Top border
	topBorder := topLeft + strings.Repeat(horizontal, boxWidth-2) + topRight
	fmt.Println(te.styleText(topBorder, te.theme.Primary))

	// Title with centered text
	titlePadded := fmt.Sprintf("%s%s%s", vertical, centerText(title, boxWidth-2), vertical)
	fmt.Println(te.styleText(titlePadded, te.theme.Primary, true))

	// Separator
	separator := te.styleText(vertical, te.theme.Primary) + te.styleText(strings.Repeat(horizontal, boxWidth-2), te.theme.Secondary) + te.styleText(vertical, te.theme.Primary)
	fmt.Println(separator)

	// Metrics
	for key, value := range metrics {
		line := fmt.Sprintf(" %-20s │ %-33v", key, value)
		if len(line) > boxWidth-2 {
			line = line[:boxWidth-2]
		}
		for len(line) < boxWidth-2 {
			line += " "
		}
		fmt.Printf("%s%s%s%s%s\n",
			te.styleText(vertical, te.theme.Primary),
			te.styleText(fmt.Sprintf(" %-20s", key), te.theme.Info),
			te.styleText(" │ ", te.theme.Secondary),
			te.styleText(fmt.Sprintf("%-32v ", value), te.theme.Foreground),
			te.styleText(vertical, te.theme.Primary))
	}

	// Bottom border
	bottomBorder := bottomLeft + strings.Repeat(horizontal, boxWidth-2) + bottomRight
	fmt.Println(te.styleText(bottomBorder, te.theme.Primary))
}

// centerText centers text within a given width
func centerText(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-len(text)-padding)
}

// AnimatedTypewriter creates typewriter effect
func (te *TerminalEffects) AnimatedTypewriter(text string, speed time.Duration) {
	for _, char := range text {
		fmt.Print(te.styleText(string(char), te.theme.Foreground))
		time.Sleep(speed)
	}
	fmt.Println()
}

// WaveAnimation creates a wave effect on text
func (te *TerminalEffects) WaveAnimation(text string) {
	for i := 0; i < len(text); i++ {
		line := ""
		for j, char := range text {
			if j == i {
				line += te.styleText(string(char), te.theme.Primary, true)
			} else {
				line += te.styleText(string(char), te.theme.Secondary)
			}
		}
		fmt.Printf("\r%s", line)
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Println()
}

// PulseAnimation creates a pulsing effect
func (te *TerminalEffects) PulseAnimation(message string, cycles int) {
	for i := 0; i < cycles; i++ {
		// Fade in
		for brightness := 0; brightness <= 100; brightness += 10 {
			te.printPulsedText(message, brightness)
			time.Sleep(20 * time.Millisecond)
		}
		// Fade out
		for brightness := 100; brightness >= 0; brightness -= 10 {
			te.printPulsedText(message, brightness)
			time.Sleep(20 * time.Millisecond)
		}
	}
}

// Helper methods

func (te *TerminalEffects) animatedPrint(text string, delay time.Duration) {
	for _, char := range text {
		fmt.Print(te.styleText(string(char), te.theme.Foreground))
		time.Sleep(delay)
	}
}

func (te *TerminalEffects) printColored(text string, color lipgloss.Color) {
	fmt.Print(te.styleText(text, color))
}

func (te *TerminalEffects) printColoredLn(text string, color lipgloss.Color) {
	fmt.Println(te.styleText(text, color))
}

func (te *TerminalEffects) newLine() {
	fmt.Println()
}

func (te *TerminalEffects) printSeparator(char string, color lipgloss.Color) {
	separator := strings.Repeat(char, 60)
	fmt.Println(te.styleText(separator, color))
}

func (te *TerminalEffects) styleText(text string, color lipgloss.Color, style ...interface{}) string {
	styleText := lipgloss.NewStyle().
		Foreground(color)

	// Apply bold if requested
	if len(style) > 0 {
		if bold, ok := style[0].(bool); ok && bold {
			styleText = styleText.Bold(true)
		}
	}

	return styleText.Render(text)
}

func (te *TerminalEffects) printPulsedText(message string, brightness int) {
	// Simple brightness simulation with spacing
	spacing := strings.Repeat(" ", (100-brightness)/20)
	fmt.Printf("\r%s%s%s", spacing, te.styleText(message, te.theme.Primary), spacing)
	time.Sleep(10 * time.Millisecond)
}

// Global convenience functions

var defaultEffects *TerminalEffects
var quiet bool

func init() {
	defaultEffects = NewTerminalEffects(DefaultTheme())
}

// DisableAnimations suppresses all visual output and sleeps.
func DisableAnimations() {
	quiet = true
}

// EnableAnimations re-enables visual output.
func EnableAnimations() {
	quiet = false
}

// IsQuiet returns whether animations are suppressed.
func IsQuiet() bool {
	return quiet
}

// ShowStartupAnimation displays the default startup animation
func ShowStartupAnimation() {
	if quiet {
		return
	}
	defaultEffects.ShowStartupAnimation()
}

// ShowSuccess displays a success message with animation
func ShowSuccess(message string) {
	if quiet {
		return
	}
	defaultEffects.SuccessAnimation(message)
}

// ShowError displays an error message with animation
func ShowError(message string) {
	if quiet {
		return
	}
	defaultEffects.ErrorAnimation(message)
}

// ShowProgress displays an animated progress bar
func ShowProgress(current, total int64, message string) {
	if quiet {
		return
	}
	defaultEffects.AnimatedProgress(current, total, message)
}

// ShowSpinner displays an animated spinner
func ShowSpinner(message string, duration time.Duration) {
	if quiet {
		return
	}
	defaultEffects.AnimatedSpinner(message, duration)
}

// ShowResult displays formatted results
func ShowResult(title string, details map[string]interface{}) {
	if quiet {
		return
	}
	defaultEffects.ShowResult(title, details)
}

// ShowDashboard displays a live dashboard
func ShowDashboard(title string, metrics map[string]interface{}) {
	if quiet {
		return
	}
	defaultEffects.LiveDashboard(title, metrics)
}

// SetTheme changes the visual theme
func SetTheme(theme *VisualTheme) {
	defaultEffects = NewTerminalEffects(theme)
}

// Random animations for fun
func RandomAnimation() {
	animations := []AnimationType{
		AnimationTypeSpinner,
		AnimationTypeDots,
		AnimationTypePulse,
		AnimationTypeWave,
	}

	selected := animations[rand.Intn(len(animations))]

	switch selected {
	case AnimationTypeSpinner:
		ShowSpinner("Processing...", 2*time.Second)
	case AnimationTypeDots:
		for i := 0; i < 3; i++ {
			fmt.Print(".")
			time.Sleep(500 * time.Millisecond)
		}
		fmt.Println(" Done!")
	case AnimationTypePulse:
		defaultEffects.PulseAnimation("Pulsing...", 3)
	case AnimationTypeWave:
		defaultEffects.WaveAnimation("Wave Animation!")
	}
}