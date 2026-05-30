/*
Package interactive provides beautiful error handling for R2Go2

Copyright © 2025 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package interactive

import (
	"fmt"
	"strings"
)

// ErrorType represents different types of errors
type ErrorType int

const (
	ErrorTypeNetwork ErrorType = iota
	ErrorTypeAuth
	ErrorTypeConfig
	ErrorTypeInput
	ErrorTypePermission
	ErrorTypeNotFound
	ErrorTypeValidation
	ErrorTypeUnknown
)

// ErrorContext provides context for error handling
type ErrorContext struct {
	Error       error
	Type        ErrorType
	Operation   string
	UserAction  string
	Troubleshoot []string
	NextSteps   []string
}

// HandleError displays a beautiful error message with helpful guidance
func HandleError(ctx ErrorContext) {
	fmt.Println()

	// Error header with appropriate icon and color
	switch ctx.Type {
	case ErrorTypeNetwork:
		colorError.Println("🌐 Network Error")
	case ErrorTypeAuth:
		colorError.Println("🔐 Authentication Error")
	case ErrorTypeConfig:
		colorError.Println("⚙️ Configuration Error")
	case ErrorTypeInput:
		colorError.Println("📝 Input Error")
	case ErrorTypePermission:
		colorError.Println("🚫 Permission Error")
	case ErrorTypeNotFound:
		colorError.Println("🔍 Not Found Error")
	case ErrorTypeValidation:
		colorError.Println("✅ Validation Error")
	default:
		colorError.Println("❌ Error")
	}

	fmt.Println(strings.Repeat("─", 50))

	// Main error message
	fmt.Printf("❌ %s", colorBold.Sprintf("%s", ctx.Operation))
	if ctx.Error != nil {
		fmt.Printf(": %s", ctx.Error.Error())
	}
	fmt.Println()

	// User action if available
	if ctx.UserAction != "" {
		fmt.Println()
		colorInfo.Printf("💡 What to do: %s\n", ctx.UserAction)
	}

	// Troubleshooting steps
	if len(ctx.Troubleshoot) > 0 {
		fmt.Println()
		colorInfo.Println("🔧 Troubleshooting:")
		for i, step := range ctx.Troubleshoot {
			fmt.Printf("  %d. %s\n", i+1, step)
		}
	}

	// Next steps
	if len(ctx.NextSteps) > 0 {
		fmt.Println()
		colorInfo.Println("👣 Next steps:")
		for _, step := range ctx.NextSteps {
			fmt.Printf("  • %s\n", step)
		}
	}

	fmt.Println()
}

// NetworkError creates a network error context
func NetworkError(operation string, err error) ErrorContext {
	return ErrorContext{
		Error:     err,
		Type:      ErrorTypeNetwork,
		Operation: operation,
		UserAction: "Check your internet connection and try again.",
		Troubleshoot: []string{
			"Verify your internet connection is working",
			"Check if you can access https://cloudflare.com",
			"Try again in a few moments",
			"Check if Cloudflare services are operational",
		},
		NextSteps: []string{
			"Run the command again when connection is restored",
			"Use --dry-run to test without making changes",
		},
	}
}

// AuthError creates an authentication error context
func AuthError(operation string, err error) ErrorContext {
	return ErrorContext{
		Error:     err,
		Type:      ErrorTypeAuth,
		Operation: operation,
		UserAction: "Please verify your API token and account ID.",
		Troubleshoot: []string{
			"Check that your API token hasn't expired",
			"Verify the token has R2 permissions",
			"Ensure your account ID is correct (32 hex characters)",
			"Try creating a new API token at https://dash.cloudflare.com/profile/api-tokens",
		},
		NextSteps: []string{
			"Run 'cosmoflare setup' to reconfigure your credentials",
			"Update your profile with 'cosmoflare config set <profile>'",
			"Test with 'cosmoflare config validate'",
		},
	}
}

// ConfigError creates a configuration error context
func ConfigError(operation string, err error) ErrorContext {
	return ErrorContext{
		Error:     err,
		Type:      ErrorTypeConfig,
		Operation: operation,
		UserAction: "Your configuration needs to be updated.",
		Troubleshoot: []string{
			"Run 'cosmoflare config list' to see your profiles",
			"Check if a profile is set as current",
			"Verify the configuration file exists and is readable",
			"Ensure file permissions are correct (should be 600)",
		},
		NextSteps: []string{
			"Run 'cosmoflare setup' to create a new configuration",
			"Use 'cosmoflare config switch' to change profiles",
			"Delete and recreate the problematic profile",
		},
	}
}

// InputError creates an input validation error context
func InputError(operation string, err error) ErrorContext {
	return ErrorContext{
		Error:     err,
		Type:      ErrorTypeInput,
		Operation: operation,
		UserAction: "Please check your input and try again.",
		Troubleshoot: []string{
			"Bucket names must be 3-63 characters",
			"Use only lowercase letters, numbers, hyphens, and dots",
			"File paths must be valid and accessible",
			"Check for typos in your input",
		},
		NextSteps: []string{
			"Run 'cosmoflare --help' for command usage",
			"Check the documentation for correct input formats",
			"Use 'cosmoflare bucket list' to see existing bucket names",
		},
	}
}

// PermissionError creates a permission error context
func PermissionError(operation string, err error) ErrorContext {
	return ErrorContext{
		Error:     err,
		Type:      ErrorTypePermission,
		Operation: operation,
		UserAction: "Your API token lacks required permissions.",
		Troubleshoot: []string{
			"Ensure your API token has 'R2:Read' and 'R2:Write' permissions",
			"Check if your account has R2 access enabled",
			"Verify you're not trying to access another account's resources",
			"Contact your account administrator if needed",
		},
		NextSteps: []string{
			"Create a new API token with proper permissions",
			"Run 'cosmoflare setup' to reconfigure with correct permissions",
		},
	}
}

// NotFoundError creates a not found error context
func NotFoundError(operation string, err error) ErrorContext {
	return ErrorContext{
		Error:     err,
		Type:      ErrorTypeNotFound,
		Operation: operation,
		UserAction: "The requested resource was not found.",
		Troubleshoot: []string{
			"Check if the bucket name is spelled correctly",
			"Verify you're using the correct account",
			"Run 'cosmoflare bucket list' to see available buckets",
			"Ensure the resource hasn't been deleted",
		},
		NextSteps: []string{
			"Create the bucket with 'cosmoflare bucket create'",
			"Switch to the correct account profile",
		},
	}
}

// ValidationError creates a validation error context
func ValidationError(operation string, err error) ErrorContext {
	return ErrorContext{
		Error:     err,
		Type:      ErrorTypeValidation,
		Operation: operation,
		UserAction: "The input validation failed.",
		Troubleshoot: []string{
			"Check that all required fields are provided",
			"Verify the format of your input",
			"Ensure values meet the required constraints",
			"Look for any hidden characters or formatting issues",
		},
		NextSteps: []string{
			"Run 'cosmoflare config validate' to check your profile",
			"Use 'cosmoflare setup' for guided configuration",
		},
	}
}

// SuccessMessage displays a beautiful success message
func SuccessMessage(operation, details string) {
	fmt.Println()
	colorSuccess.Println("🎉 Operation Successful!")
	fmt.Println(strings.Repeat("─", 30))
	fmt.Printf("✅ %s\n", colorBold.Sprintf("%s", operation))

	if details != "" {
		fmt.Printf("📋 %s\n", details)
	}

	fmt.Println()
}

// WarningMessage displays a warning message
func WarningMessage(operation, details string) {
	fmt.Println()
	colorWarning.Println("⚠️ Warning")
	fmt.Println(strings.Repeat("─", 30))
	fmt.Printf("⚠️ %s\n", colorBold.Sprintf("%s", operation))

	if details != "" {
		fmt.Printf("💡 %s\n", details)
	}

	fmt.Println()
}