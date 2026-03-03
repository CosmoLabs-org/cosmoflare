package utils

import (
	"fmt"
	"strings"
)

// FormatBytes converts a byte count to a human-readable string using binary units (1024).
func FormatBytes(bytes int64) string {
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

// MaskAccountID masks sensitive account ID information, showing only
// the first 4 and last 4 characters with asterisks in between.
func MaskAccountID(accountID string) string {
	if len(accountID) <= 8 {
		return strings.Repeat("*", len(accountID))
	}
	return accountID[:4] + strings.Repeat("*", len(accountID)-8) + accountID[len(accountID)-4:]
}
