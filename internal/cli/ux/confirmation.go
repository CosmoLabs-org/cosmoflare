/*
Package ux provides user experience enhancements for R2Go2

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package ux

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

// ConfirmationType defines different types of confirmations
type ConfirmationType int

const (
	ConfirmationTypeYesNo ConfirmationType = iota
	ConfirmationTypeYesNoAll
	ConfirmationTypeYesNoCancel
	ConfirmationTypeContinue
	ConfirmationTypeRetry
	ConfirmationTypeOverwrite
)

// ConfirmationOptions contains options for confirmation dialogs
type ConfirmationOptions struct {
	Type        ConfirmationType
	Default     bool
	Timeout     int // Timeout in seconds, 0 for no timeout
	ShowDetails bool
	Message     string
	Details     string
	Warning     string
}

// ConfirmationResult contains the result of a confirmation
type ConfirmationResult struct {
	Answer      bool
	All         bool
	Cancelled   bool
	Timeout     bool
	Confidence  string // "high", "medium", "low"
	UserInput   string
}

// DefaultConfirmationOptions returns default confirmation options
func DefaultConfirmationOptions() *ConfirmationOptions {
	return &ConfirmationOptions{
		Type:        ConfirmationTypeYesNo,
		Default:     false,
		Timeout:     0,
		ShowDetails: false,
		Message:     "Do you want to continue?",
	}
}

// Confirm prompts the user for confirmation
func Confirm(message string) bool {
	opts := DefaultConfirmationOptions()
	opts.Message = message
	result := PromptConfirmation(opts)
	return result.Answer
}

// ConfirmWithDetails prompts the user with detailed information
func ConfirmWithDetails(message, details string) bool {
	opts := DefaultConfirmationOptions()
	opts.Message = message
	opts.Details = details
	opts.ShowDetails = true
	result := PromptConfirmation(opts)
	return result.Answer
}

// ConfirmDestructive prompts for destructive operations
func ConfirmDestructive(operation string, targets []string) bool {
	opts := &ConfirmationOptions{
		Type:        ConfirmationTypeYesNo,
		Default:     false,
		ShowDetails: true,
		Message:     fmt.Sprintf("Are you sure you want to %s?", operation),
		Warning:     "This action cannot be undone!",
	}

	if len(targets) > 0 {
		details := fmt.Sprintf("Targets (%d):\n", len(targets))
		for i, target := range targets {
			if i >= 10 { // Limit display to first 10 items
				details += fmt.Sprintf("  ... and %d more items\n", len(targets)-10)
				break
			}
			details += fmt.Sprintf("  • %s\n", target)
		}
		opts.Details = details
	}

	result := PromptConfirmation(opts)
	return result.Answer
}

// ConfirmOverwrite prompts for file overwrite
func ConfirmOverwrite(filename string, isNewer bool) bool {
	opts := &ConfirmationOptions{
		Type:    ConfirmationTypeOverwrite,
		Default: false,
		Message: fmt.Sprintf("Overwrite existing file: %s?", filename),
	}

	if isNewer {
		opts.Details = "Source file is newer than destination"
	}

	result := PromptConfirmation(opts)
	return result.Answer
}

// PromptConfirmation displays a confirmation dialog and returns the result
func PromptConfirmation(opts *ConfirmationOptions) *ConfirmationResult {
	if opts == nil {
		opts = DefaultConfirmationOptions()
	}

	result := &ConfirmationResult{}

	// Print the confirmation message
	printConfirmationMessage(opts)

	// Get user input
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		result.Cancelled = true
		return result
	}

	input = strings.TrimSpace(strings.ToLower(input))

	// Parse response
	result.UserInput = input
	result.Answer, result.All, result.Cancelled = parseConfirmationInput(input, opts.Type)

	// Set confidence level
	result.Confidence = determineConfidence(input, opts.Type)

	return result
}

// printConfirmationMessage formats and prints the confirmation dialog
func printConfirmationMessage(opts *ConfirmationOptions) {
	fmt.Println()

	// Print warning if present
	if opts.Warning != "" {
		color.Yellow("⚠️  %s", opts.Warning)
		fmt.Println()
	}

	// Print main message
	color.Cyan("❓ %s", opts.Message)

	// Print details if available
	if opts.ShowDetails && opts.Details != "" {
		fmt.Println()
		color.White("%s", opts.Details)
	}

	// Print prompt options
	var prompt string
	switch opts.Type {
	case ConfirmationTypeYesNo:
		if opts.Default {
			prompt = " [Y/n]: "
		} else {
			prompt = " [y/N]: "
		}
	case ConfirmationTypeYesNoAll:
		prompt = " [y/n/a/l]: "
	case ConfirmationTypeYesNoCancel:
		prompt = " [y/n/c]: "
	case ConfirmationTypeContinue:
		prompt = " [Continue/C/Cancel]: "
	case ConfirmationTypeRetry:
		prompt = " [Retry/R/Cancel]: "
	case ConfirmationTypeOverwrite:
		if opts.Default {
			prompt = " [Y/n/o/s]: "
		} else {
			prompt = " [y/N/o/s]: "
		}
	default:
		prompt = " [y/N]: "
	}

	color.White(prompt)
}

// parseConfirmationInput parses user input for confirmation dialogs
func parseConfirmationInput(input string, confType ConfirmationType) (answer, all, cancelled bool) {
	// Handle empty input (default)
	if input == "" {
		switch confType {
		case ConfirmationTypeYesNo, ConfirmationTypeOverwrite:
			return false, false, false // Default to No
		default:
			return false, false, false
		}
	}

	switch confType {
	case ConfirmationTypeYesNo:
		return parseYesNoInput(input)

	case ConfirmationTypeYesNoAll:
		return parseYesNoAllInput(input)

	case ConfirmationTypeYesNoCancel:
		return parseYesNoCancelInput(input)

	case ConfirmationTypeContinue:
		return parseContinueInput(input)

	case ConfirmationTypeRetry:
		return parseRetryInput(input)

	case ConfirmationTypeOverwrite:
		return parseOverwriteInput(input)

	default:
		return input == "y" || input == "yes", false, false
	}
}

// parseYesNoInput parses input for yes/no confirmations
func parseYesNoInput(input string) (answer, all, cancelled bool) {
	return input == "y" || input == "yes", false, false
}

// parseYesNoAllInput parses input for yes/no/all confirmations
func parseYesNoAllInput(input string) (answer, all, cancelled bool) {
	switch input {
	case "y", "yes":
		return true, false, false
	case "n", "no":
		return false, false, false
	case "a", "all":
		return true, true, false
	case "l", "less":
		return false, false, false
	default:
		return false, false, false
	}
}

// parseYesNoCancelInput parses input for yes/no/cancel confirmations
func parseYesNoCancelInput(input string) (answer, all, cancelled bool) {
	switch input {
	case "y", "yes":
		return true, false, false
	case "n", "no":
		return false, false, false
	case "c", "cancel":
		return false, false, true
	default:
		return false, false, false
	}
}

// parseContinueInput parses input for continue/cancel confirmations
func parseContinueInput(input string) (answer, all, cancelled bool) {
	switch input {
	case "continue", "c":
		return true, false, false
	case "cancel":
		return false, false, true
	default:
		return false, false, false
	}
}

// parseRetryInput parses input for retry/cancel confirmations
func parseRetryInput(input string) (answer, all, cancelled bool) {
	switch input {
	case "retry", "r":
		return true, false, false
	case "cancel":
		return false, false, true
	default:
		return false, false, false
	}
}

// parseOverwriteInput parses input for overwrite/skip confirmations
func parseOverwriteInput(input string) (answer, all, cancelled bool) {
	switch input {
	case "y", "yes":
		return true, false, false
	case "n", "no":
		return false, false, false
	case "o", "overwrite":
		return true, false, false
	case "s", "skip":
		return false, false, false
	default:
		return false, false, false
	}
}

// determineConfidence determines confidence level based on user input
func determineConfidence(input string, confType ConfirmationType) string {
	// Full words indicate higher confidence
	if len(input) > 2 {
		return "high"
	}

	// Single letters indicate medium confidence
	if len(input) == 1 {
		return "medium"
	}

	// Empty or unclear input indicates low confidence
	return "low"
}

// InteractiveConfirmation provides an interactive confirmation with timeout
func InteractiveConfirmation(opts *ConfirmationOptions, timeout int) *ConfirmationResult {
	// For now, just call the standard confirmation
	// In a real implementation, this would handle timeout with goroutines
	return PromptConfirmation(opts)
}

// BatchConfirmation confirms multiple operations at once
func BatchConfirmation(operations []string, operationType string) *ConfirmationResult {
	opts := &ConfirmationOptions{
		Type:        ConfirmationTypeYesNoAll,
		Default:     false,
		ShowDetails: true,
		Message:     fmt.Sprintf("Perform %s operation on %d items?", operationType, len(operations)),
		Warning:     "This will affect multiple files/folders",
	}

	if len(operations) > 0 {
		details := fmt.Sprintf("Operations:\n")
		maxShow := 5
		for i, op := range operations {
			if i >= maxShow {
				details += fmt.Sprintf("  ... and %d more operations\n", len(operations)-maxShow)
				break
			}
			details += fmt.Sprintf("  • %s\n", op)
		}
		opts.Details = details
	}

	return PromptConfirmation(opts)
}

// AutoConfirm provides automatic confirmation for non-interactive mode
func AutoConfirm(defaultAnswer bool) *ConfirmationResult {
	return &ConfirmationResult{
		Answer:     defaultAnswer,
		All:        false,
		Cancelled:  false,
		Timeout:    false,
		Confidence: "auto",
		UserInput:  "",
	}
}