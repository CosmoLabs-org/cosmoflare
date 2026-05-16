/*
Package interactive provides animation and transition functionality

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package interactive

import (
	"fmt"
	"strings"
	"time"
)

// AnimationState represents the current animation state
type AnimationState struct {
	Progress float64
	Message  string
	Speed    time.Duration
	Complete bool
}

// Easing functions for smooth animations

// easeInOutCubic provides smooth acceleration and deceleration
func easeInOutCubic(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	if t < 0.5 {
		return 4 * t * t * t
	}
	p := -2*t + 2
	result := 1 - p*p*p/2
	if result > 1 {
		result = 1
	}
	if result < 0 {
		result = 0
	}
	return result
}

// easeOutQuad provides smooth deceleration
func easeOutQuad(t float64) float64 {
	return 1 - (1-t)*(1-t)
}

// easeInQuad provides smooth acceleration
func easeInQuad(t float64) float64 {
	return t * t
}

// Animation frames and timing
const (
	DefaultFrameRate = 60 * time.Millisecond
	FastFrameRate    = 30 * time.Millisecond
	SlowFrameRate    = 100 * time.Millisecond
)

// SpinnerCharacters contains different spinner styles
var SpinnerCharacters = []string{
	"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
}

// DotsSpinnerCharacters contains a simple dots spinner
var DotsSpinnerCharacters = []string{
	"⠁", "⠈", "⠐", "⠠", "⠄", "⠂",
}

// ProgressBarCharacters contains progress bar characters
var ProgressBarCharacters = map[string]struct {
	Empty  string
	Fill   string
	Start  string
	End    string
	Head   string
}{
	"standard": {" ", "█", " ", " ", "█"},
	"blocks":   {"░", "▓", " ", " ", "▓"},
	"dots":     {" ", "●", " ", " ", "●"},
	"arrows":   {"-", ">", " ", " ", ">"},
}

// Animator handles various animations
type Animator struct {
	Style       string
	Speed       time.Duration
	Disabled    bool
	frameCount  int
	lastUpdate  time.Time
}

// NewAnimator creates a new animator
func NewAnimator() *Animator {
	return &Animator{
		Style:    "standard",
		Speed:    DefaultFrameRate,
		Disabled: false,
	}
}

// ShowSpinner displays an animated spinner
func (a *Animator) ShowSpinner(message string, duration time.Duration) {
	if a.Disabled {
		fmt.Printf("%s %s\n", message, "✓")
		return
	}

	start := time.Now()
	spinnerIndex := 0

	fmt.Printf("\r%s %s ", message, SpinnerCharacters[0])

	for time.Since(start) < duration {
		if time.Since(a.lastUpdate) >= a.Speed {
			fmt.Printf("\r%s %s ", message, SpinnerCharacters[spinnerIndex])
			spinnerIndex = (spinnerIndex + 1) % len(SpinnerCharacters)
			a.lastUpdate = time.Now()
		}
		time.Sleep(10 * time.Millisecond)
	}

	fmt.Printf("\r%s %s ✓\n", message, strings.Repeat(" ", 10))
}

// ShowProgress displays an animated progress bar
func (a *Animator) ShowProgress(message string, steps []string) {
	if a.Disabled {
		fmt.Printf("%s %s\n", message, "✓")
		return
	}

	fmt.Println()
	fmt.Printf("%s\n", Bold(message))
	fmt.Println(strings.Repeat("─", len(message)+2))

	totalSteps := len(steps)
	barStyle := ProgressBarCharacters["standard"]

	for i, step := range steps {
		progress := float64(i+1) / float64(totalSteps)
		a.drawProgressBar(progress, barStyle, step)
		time.Sleep(a.Speed * 2)
	}

	fmt.Println()
}

// drawProgressBar draws a single progress bar frame
func (a *Animator) drawProgressBar(progress float64, style struct{ Empty, Fill, Start, End, Head string }, label string) {
	width := 40
	filled := int(progress * float64(width))
	empty := width - filled

	bar := style.Start + strings.Repeat(style.Fill, filled) + style.Head + strings.Repeat(style.Empty, empty) + style.End
	percentage := int(progress * 100)

	fmt.Printf("\r  %s %3d%% %s", label, percentage, bar)
	if progress >= 1.0 {
		fmt.Println()
	}
}

// AnimateTransition creates smooth transitions between states
func (a *Animator) AnimateTransition(from, to string, duration time.Duration) {
	if a.Disabled {
		fmt.Printf("%s → %s\n", from, to)
		return
	}

	steps := 20
	stepDuration := duration / time.Duration(steps)

	fmt.Printf("\r%s", from)

	for i := 0; i <= steps; i++ {
		progress := float64(i) / float64(steps)
		easedProgress := easeInOutCubic(progress)

		// Create fade effect
		fadeOut := 1.0 - easedProgress
		fadeIn := easedProgress

		// Mix the two texts
		mixed := a.mixTexts(from, to, fadeOut, fadeIn)
		fmt.Printf("\r%s", mixed)

		time.Sleep(stepDuration)
	}

	fmt.Printf("\r%s\n", to)
}

// mixTexts creates a fade effect between two text strings
func (a *Animator) mixTexts(from, to string, fadeOut, fadeIn float64) string {
	if len(from) != len(to) {
		// Pad shorter string
		maxLen := max(len(from), len(to))
		from += strings.Repeat(" ", maxLen-len(from))
		to += strings.Repeat(" ", maxLen-len(to))
	}

	result := make([]rune, len(from))
	for i, r := range from {
		if i < len(to) {
			// Simple character mixing - in a real implementation,
			// you might use color codes or other visual effects
			if fadeIn > 0.5 {
				result[i] = []rune(to)[i]
			} else {
				result[i] = r
			}
		} else {
			result[i] = r
		}
	}

	return string(result)
}

// ShowLoadingSkeleton displays a loading skeleton animation
func (a *Animator) ShowLoadingSkeleton(lines []string, duration time.Duration) {
	if a.Disabled {
		fmt.Printf("%s\n", strings.Join(lines, "\n"))
		return
	}

	start := time.Now()
	frame := 0

	for time.Since(start) < duration {
		if time.Since(a.lastUpdate) >= a.Speed*2 {
			a.drawSkeleton(lines, frame)
			frame = (frame + 1) % 4
			a.lastUpdate = time.Now()
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Show final result
	fmt.Printf("\r%s\n", strings.Join(lines, "\n"))
}

// drawSkeleton draws a loading skeleton frame
func (a *Animator) drawSkeleton(lines []string, frame int) {
	// Move cursor up to overwrite skeleton
	for range lines {
		fmt.Printf("\033[A")
	}

	skeletonChars := []string{"░", "▒", "▓", "█"}
	char := skeletonChars[frame%len(skeletonChars)]

	for _, line := range lines {
		skeletonLine := strings.Repeat(char, len(line))
		fmt.Printf("\r%s\n", skeletonLine)
	}

	time.Sleep(50 * time.Millisecond)
}

// TypewriterEffect creates a typewriter animation for text
func (a *Animator) TypewriterEffect(text string, speed time.Duration) {
	if a.Disabled {
		fmt.Printf("%s\n", text)
		return
	}

	for _, char := range text {
		fmt.Printf("%c", char)
		time.Sleep(speed)
	}
	fmt.Println()
}

// PulseText creates a pulsing effect for important text
func (a *Animator) PulseText(text string, pulses int) {
	if a.Disabled {
		fmt.Printf("%s\n", text)
		return
	}

	for i := 0; i < pulses; i++ {
		// Pulse in
		for j := 0; j <= 10; j++ {
			intensity := float64(j) / 10.0
			a.drawPulsedText(text, intensity)
			time.Sleep(30 * time.Millisecond)
		}

		// Pulse out
		for j := 10; j >= 0; j-- {
			intensity := float64(j) / 10.0
			a.drawPulsedText(text, intensity)
			time.Sleep(30 * time.Millisecond)
		}
	}

	fmt.Printf("\r%s\n", text)
}

// drawPulsedText draws text with pulsing intensity
func (a *Animator) drawPulsedText(text string, intensity float64) {
	// In a real implementation, this would use color codes
	// or terminal escape sequences for visual effects
	bold := intensity > 0.5
	if bold {
		fmt.Printf("\r%s", Bold(text))
	} else {
		fmt.Printf("\r%s", text)
	}
}

// ShowStepTransition creates smooth transitions between setup steps
func (a *Animator) ShowStepTransition(fromStep, toStep int, fromTitle, toTitle string) {
	if a.Disabled {
		fmt.Printf("\nStep %d → %d\n", fromStep, toStep)
		return
	}

	duration := 500 * time.Millisecond
	steps := 10

	fmt.Println()

	for i := 0; i <= steps; i++ {
		progress := float64(i) / float64(steps)
		easedProgress := easeInOutCubic(progress)

		// Create visual transition
		dots := int(easedProgress * 20)
		progressBar := strings.Repeat("●", dots) + strings.Repeat("○", 20-dots)

		fmt.Printf("\rStep %d → %d [%s] %s", fromStep, toStep, progressBar, "")
		time.Sleep(duration / time.Duration(steps))
	}

	fmt.Printf("\rStep %d → %d [%s] %s\n", fromStep, toStep, strings.Repeat("●", 20), toTitle)
	time.Sleep(200 * time.Millisecond)
}

// ShowSuccessAnimation displays a success animation
func (a *Animator) ShowSuccessAnimation(message string) {
	if a.Disabled {
		PrintSuccess("%s", message)
		return
	}

	animations := []string{
		"✓          ",
		"✓          ",
		"✓✓         ",
		"✓✓✓        ",
		"✓✓✓✓       ",
		"✓✓✓✓✓      ",
		"✨✓✓✓✓✨     ",
		" 🎉✓✓✓✓🎉    ",
		"   ✨✓✓✨    ",
		"     ✨✓✨    ",
		"       ✨✨   ",
		"         ✨✨ ",
		"           ✨✨",
	}

	for _, animation := range animations {
		fmt.Printf("\r%s %s", message, animation)
		time.Sleep(80 * time.Millisecond)
	}

	PrintSuccess("%s", message)
}

// Configurable animation settings
func (a *Animator) SetStyle(style string) {
	switch style {
	case "fast":
		a.Speed = FastFrameRate
	case "slow":
		a.Speed = SlowFrameRate
	case "disabled":
		a.Disabled = true
	default:
		a.Speed = DefaultFrameRate
	}
}

// Helper functions

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Global animator instance
var globalAnimator = NewAnimator()

// ShowSpinner is a convenience function using the global animator
func ShowSpinner(message string, duration time.Duration) {
	globalAnimator.ShowSpinner(message, duration)
}

// ShowProgress is a convenience function using the global animator
func ShowProgress(message string, steps []string) {
	globalAnimator.ShowProgress(message, steps)
}

// AnimateTransition is a convenience function using the global animator
func AnimateTransition(from, to string, duration time.Duration) {
	globalAnimator.AnimateTransition(from, to, duration)
}

// SetAnimationStyle sets the global animation style
func SetAnimationStyle(style string) {
	globalAnimator.SetStyle(style)
}