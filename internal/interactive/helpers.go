/*
Package interactive provides helper functions for beautiful CLI interactions

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package interactive

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/CosmoLabs-org/CosmoDev-R2Go2/internal/config"
)

// Color definitions for consistent theming
var (
	Bold       = color.New(color.Bold).SprintFunc()
	Success    = color.New(color.FgGreen, color.Bold).SprintFunc()
	Error      = color.New(color.FgRed, color.Bold).SprintFunc()
	Warning    = color.New(color.FgYellow, color.Bold).SprintFunc()
	Info       = color.New(color.FgCyan, color.Bold).SprintFunc()
	Dim        = color.New(color.Faint).SprintFunc()
	Muted      = color.New(color.FgHiBlack).SprintFunc()
	Reset      = "\033[0m"
	Red        = color.RedString
	Green      = color.GreenString
	Yellow     = color.YellowString
	Blue       = color.BlueString
	Magenta    = color.MagentaString
	Cyan       = color.CyanString
	White      = color.WhiteString
)

// Print functions for consistent output
func PrintSuccess(format string, args ...interface{}) {
	fmt.Printf("✅ %s\n", fmt.Sprintf(format, args...))
}

func PrintError(format string, args ...interface{}) {
	fmt.Printf("❌ %s\n", Error(fmt.Sprintf(format, args...)))
}

func PrintWarning(format string, args ...interface{}) {
	fmt.Printf("⚠️  %s\n", Warning(fmt.Sprintf(format, args...)))
}

func PrintInfo(format string, args ...interface{}) {
	fmt.Printf("ℹ️  %s\n", Info(fmt.Sprintf(format, args...)))
}

// GetConfigManager returns a new config manager instance
func GetConfigManager() (*config.ConfigManager, error) {
	return config.NewConfigManager()
}

// ClearScreen clears the terminal screen
func ClearScreen() {
	fmt.Print("\033[H\033[2J")
}

// CenterText centers text within a given width
func CenterText(text string, width int) string {
	padding := width - len(text)
	if padding <= 0 {
		return text
	}
	left := padding / 2
	right := padding - left
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
}

// BoxText creates a nice box around text
func BoxText(text string) string {
	lines := strings.Split(text, "\n")
	maxWidth := 0
	for _, line := range lines {
		if len(line) > maxWidth {
			maxWidth = len(line)
		}
	}

	border := strings.Repeat("─", maxWidth+4)
	result := "┌" + border + "┐\n"

	for _, line := range lines {
		result += "│ " + line + strings.Repeat(" ", maxWidth-len(line)) + " │\n"
	}

	result += "└" + border + "┘"
	return result
}

// TruncateText truncates text to fit within max length
func TruncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	if maxLength < 3 {
		return strings.Repeat("●", maxLength)
	}
	return text[:maxLength-3] + "..."
}

// ConfirmYesNo prompts for yes/no confirmation
func ConfirmYesNo(prompt string, defaultYes bool) bool {
	return ConfirmWithReader(prompt, defaultYes, DefaultInput())
}

// PauseAndWait displays a message and waits for user to press Enter
func PauseAndWait(message string) {
	PauseAndWaitWithReader(message, DefaultInput())
}

// PauseAndWaitWithReader displays a message and waits using the given InputReader
func PauseAndWaitWithReader(message string, reader InputReader) {
	if message == "" {
		message = "Press Enter to continue..."
	}
	fmt.Print(message)
	reader.ReadLine()
}

// CheckFirstRun determines if this is the first time the user is running R2Go2
func CheckFirstRun() (bool, error) {
	configMgr, err := GetConfigManager()
	if err != nil {
		return false, err
	}

	profiles := configMgr.ListProfiles()
	return len(profiles) == 0, nil
}

// ShowWelcomeForNewUser displays a special welcome for first-time users
func ShowWelcomeForNewUser() {
	ShowWelcomeForNewUserWithReader(DefaultInput())
}

// ShowWelcomeForNewUserWithReader displays welcome using the given InputReader
func ShowWelcomeForNewUserWithReader(reader InputReader) {
	ClearScreen()
	fmt.Println()
	fmt.Println(Bold("🎉 Welcome to R2Go2!"))
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()
	fmt.Println("It looks like this is your first time using R2Go2.")
	fmt.Println("Let's get you set up quickly and easily.")
	fmt.Println()
	fmt.Println("R2Go2 is a professional Cloudflare R2 management tool")
	fmt.Println("that helps you store and manage files in the cloud.")
	fmt.Println()

	if ConfirmWithReader("Would you like to run the setup wizard now?", true, reader) {
		fmt.Println()
		fmt.Println("🚀 Starting setup wizard...")
	}
}

// FormatFileSize formats a byte size as human readable
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ShowCommandTip shows helpful command tips
func ShowCommandTip(tips ...string) {
	fmt.Println()
	Info("💡 Tip:")
	for _, tip := range tips {
		fmt.Printf("  • %s\n", tip)
	}
}

// ExitWithError displays an error message and exits
func ExitWithError(err error) {
	PrintError("%v", err)
	os.Exit(1)
}