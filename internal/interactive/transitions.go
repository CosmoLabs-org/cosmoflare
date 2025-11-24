/*
Package interactive provides smooth transition functionality

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package interactive

import (
	"fmt"
	"strings"
	"time"
)

// TransitionType defines different transition types
type TransitionType string

const (
	TransitionFade    TransitionType = "fade"
	TransitionSlide   TransitionType = "slide"
	TransitionWipe    TransitionType = "wipe"
	TransitionZoom    TransitionType = "zoom"
	TransitionReplace TransitionType = "replace"
)

// Transition defines a transition between two screens or states
type Transition struct {
	Type       TransitionType
	Duration   time.Duration
	EasingFunc func(float64) float64
	From       TransitionState
	To         TransitionState
}

// TransitionState represents a screen state
type TransitionState struct {
	Title       string
	Content     []string
	Progress    int
	MaxProgress int
	Theme       string
}

// TransitionManager manages screen transitions
type TransitionManager struct {
	animator    *Animator
	Disabled    bool
	CurrentTheme string
}

// NewTransitionManager creates a new transition manager
func NewTransitionManager() *TransitionManager {
	return &TransitionManager{
		animator:    NewAnimator(),
		Disabled:    false,
		CurrentTheme: "cosmic",
	}
}

// SetTheme sets the visual theme for transitions
func (tm *TransitionManager) SetTheme(theme string) {
	tm.CurrentTheme = theme
}

// Disable disables all transition effects
func (tm *TransitionManager) Disable() {
	tm.Disabled = true
	tm.animator.Disabled = true
}

// Execute performs a transition between two states
func (tm *TransitionManager) Execute(transitionType TransitionType, from, to TransitionState) {
	if tm.Disabled {
		tm.renderInstant(to)
		return
	}

	switch transitionType {
	case TransitionFade:
		tm.fadeTransition(from, to)
	case TransitionSlide:
		tm.slideTransition(from, to)
	case TransitionWipe:
		tm.wipeTransition(from, to)
	case TransitionZoom:
		tm.zoomTransition(from, to)
	default:
		tm.replaceTransition(from, to)
	}
}

// fadeTransition creates a fade in/out effect
func (tm *TransitionManager) fadeTransition(from, to TransitionState) {
	duration := 800 * time.Millisecond
	frames := 20
	frameDuration := duration / time.Duration(frames)

	// Fade out from state
	for i := 0; i <= frames/2; i++ {
		progress := float64(i) / float64(frames/2)
		alpha := 1.0 - progress

		tm.renderWithAlpha(from, alpha)
		time.Sleep(frameDuration)
	}

	// Fade in to state
	for i := 0; i <= frames/2; i++ {
		progress := float64(i) / float64(frames/2)
		alpha := progress

		tm.renderWithAlpha(to, alpha)
		time.Sleep(frameDuration)
	}

	tm.renderInstant(to)
}

// slideTransition creates a sliding effect
func (tm *TransitionManager) slideTransition(from, to TransitionState) {
	duration := 600 * time.Millisecond
	frames := 30
	frameDuration := duration / time.Duration(frames)

	for i := 0; i <= frames; i++ {
		progress := float64(i) / float64(frames)
		easedProgress := easeInOutCubic(progress)

		tm.renderSlide(from, to, easedProgress)
		time.Sleep(frameDuration)
	}

	tm.renderInstant(to)
}

// wipeTransition creates a wipe effect
func (tm *TransitionManager) wipeTransition(from, to TransitionState) {
	duration := 500 * time.Millisecond
	frames := 25
	frameDuration := duration / time.Duration(frames)

	for i := 0; i <= frames; i++ {
		progress := float64(i) / float64(frames)
		easedProgress := easeOutQuad(progress)

		tm.renderWipe(from, to, easedProgress)
		time.Sleep(frameDuration)
	}

	tm.renderInstant(to)
}

// zoomTransition creates a zoom in/out effect
func (tm *TransitionManager) zoomTransition(from, to TransitionState) {
	duration := 700 * time.Millisecond
	frames := 35
	frameDuration := duration / time.Duration(frames)

	// Zoom out
	for i := 0; i <= frames/2; i++ {
		progress := float64(i) / float64(frames/2)
		scale := 1.0 - progress*0.5

		tm.renderWithScale(from, scale)
		time.Sleep(frameDuration)
	}

	// Zoom in
	for i := 0; i <= frames/2; i++ {
		progress := float64(i) / float64(frames/2)
		scale := 0.5 + progress*0.5

		tm.renderWithScale(to, scale)
		time.Sleep(frameDuration)
	}

	tm.renderInstant(to)
}

// replaceTransition performs a simple replace
func (tm *TransitionManager) replaceTransition(from, to TransitionState) {
	time.Sleep(200 * time.Millisecond)
	tm.renderInstant(to)
}

// Render functions

func (tm *TransitionManager) renderInstant(state TransitionState) {
	ClearScreen()
	tm.renderHeader(state.Title)
	tm.renderContent(state.Content)
	if state.Progress > 0 {
		tm.renderProgress(state.Progress, state.MaxProgress)
	}
}

func (tm *TransitionManager) renderWithAlpha(state TransitionState, alpha float64) {
	// Alpha blending simulation using brightness
	brightness := int(alpha * 255)
	brightnessStr := fmt.Sprintf("\033[38;2;%d;%d;%dm", brightness, brightness, brightness)

	fmt.Printf("%s", brightnessStr)
	tm.renderInstant(state)
	fmt.Print(Reset)
}

func (tm *TransitionManager) renderSlide(from, to TransitionState, progress float64) {
	ClearScreen()

	// Slide content
	if progress < 0.5 {
		// Show from state sliding out
		slideOffset := int((1.0 - progress*2) * 10)
		tm.renderWithOffset(from, slideOffset)
	} else {
		// Show to state sliding in
		slideOffset := int((progress-0.5)*2 * 10)
		tm.renderWithOffset(to, -10+slideOffset)
	}
}

func (tm *TransitionManager) renderWipe(from, to TransitionState, progress float64) {
	ClearScreen()

	// Wipe from left to right
	wipeWidth := int(progress * 50)
	if wipeWidth > 0 {
		fmt.Printf("%s", strings.Repeat("█", wipeWidth))
	}

	// Show content based on wipe position
	if progress < 0.5 {
		tm.renderInstant(from)
	} else {
		tm.renderInstant(to)
	}

	if wipeWidth > 0 && wipeWidth < 50 {
		fmt.Printf("\r%s", strings.Repeat(" ", wipeWidth))
	}
}

func (tm *TransitionManager) renderWithScale(state TransitionState, scale float64) {
	ClearScreen()

	// Scale simulation using spacing
	if scale < 1.0 {
		padding := int((1.0 - scale) * 10)
		spacer := strings.Repeat(" ", padding)

		fmt.Println(spacer + tm.scaleText(state.Title, scale))
		for _, line := range state.Content {
			fmt.Println(spacer + tm.scaleText(line, scale))
		}
	} else {
		tm.renderInstant(state)
	}
}

func (tm *TransitionManager) renderWithOffset(state TransitionState, offset int) {
	spacer := strings.Repeat(" ", max(0, offset))

	fmt.Println(spacer + state.Title)
	for _, line := range state.Content {
		fmt.Println(spacer + line)
	}
}

func (tm *TransitionManager) renderHeader(title string) {
	fmt.Println()
	fmt.Println(Bold(title))
	fmt.Println(strings.Repeat("─", len(title)+2))
	fmt.Println()
}

func (tm *TransitionManager) renderContent(content []string) {
	for _, line := range content {
		fmt.Println(line)
	}
	fmt.Println()
}

func (tm *TransitionManager) renderProgress(current, max int) {
	percentage := float64(current) / float64(max)
	filled := int(percentage * 30)
	bar := strings.Repeat("█", filled) + strings.Repeat("○", 30-filled)

	fmt.Printf("Progress: [%s] %d/%d (%.0f%%)\n", bar, current, max, percentage*100)
}

// Helper functions

func (tm *TransitionManager) scaleText(text string, scale float64) string {
	if scale >= 1.0 {
		return text
	}

	// Simple text scaling by truncation
	length := int(float64(len(text)) * scale)
	if length < len(text) && length > 3 {
		return text[:length] + "..."
	}
	return text[:length]
}

// SetupStepTransition handles transitions between setup steps
type SetupStepTransition struct {
	Manager *TransitionManager
	Current int
	Total   int
}

// NewSetupStepTransition creates a new setup step transition
func NewSetupStepTransition(totalSteps int) *SetupStepTransition {
	return &SetupStepTransition{
		Manager: NewTransitionManager(),
		Current: 0,
		Total:   totalSteps,
	}
}

// NextStep transitions to the next setup step
func (sst *SetupStepTransition) NextStep(title, content string) {
	fromState := TransitionState{
		Progress:    sst.Current,
		MaxProgress: sst.Total,
	}

	sst.Current++

	toState := TransitionState{
		Title:       title,
		Content:     strings.Split(content, "\n"),
		Progress:    sst.Current,
		MaxProgress: sst.Total,
		Theme:       sst.Manager.CurrentTheme,
	}

	sst.Manager.Execute(TransitionFade, fromState, toState)
}

// CompleteStep shows completion transition
func (sst *SetupStepTransition) CompleteStep(title, content string) {
	toState := TransitionState{
		Title:       title,
		Content:     strings.Split(content, "\n"),
		Progress:    sst.Total,
		MaxProgress: sst.Total,
		Theme:       sst.Manager.CurrentTheme,
	}

	ClearScreen()
	sst.Manager.renderInstant(toState)

	// Success animation
	globalAnimator.ShowSuccessAnimation("Setup Complete!")
}

// Global transition manager
var globalTransitionManager = NewTransitionManager()

// SetupWizardTransition provides easy access to setup transitions
type SetupWizardTransition struct {
	steps []TransitionState
	current int
}

// NewSetupWizardTransition creates a new wizard transition
func NewSetupWizardTransition() *SetupWizardTransition {
	return &SetupWizardTransition{
		steps:   make([]TransitionState, 0),
		current: -1,
	}
}

// AddStep adds a step to the wizard
func (swt *SetupWizardTransition) AddStep(title string, content []string) {
	step := TransitionState{
		Title:   title,
		Content: content,
	}
	swt.steps = append(swt.steps, step)
}

// Next moves to the next step
func (swt *SetupWizardTransition) Next() {
	if swt.current >= 0 && swt.current < len(swt.steps)-1 {
		from := swt.steps[swt.current]
		to := swt.steps[swt.current+1]
		globalTransitionManager.Execute(TransitionFade, from, to)
	}
	swt.current++
}

// Show shows the current step
func (swt *SetupWizardTransition) Show(stepIndex int) {
	if stepIndex >= 0 && stepIndex < len(swt.steps) {
		globalTransitionManager.renderInstant(swt.steps[stepIndex])
		swt.current = stepIndex
	}
}