/*
Package palette provides a VS Code-style command palette for the TUI dashboard

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package palette

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// Command represents an executable action in the command palette.
type Command struct {
	Name        string         // Display name, e.g. "bucket create"
	Description string         // Short help text
	Category    string         // Grouping: "R2", "DNS", "KV", "Navigation", "Settings"
	Icon        string         // Emoji icon for visual distinction
	Action      func() tea.Cmd // What happens on selection
	ContextFunc func() bool    // Optional: show only when this returns true
}

// String returns the searchable text for fuzzy matching.
func (c Command) String() string {
	return c.Name + " " + c.Description
}

// BuildRegistryFromCobra walks a Cobra command tree and creates palette
// commands from every non-hidden leaf (or branch with its own RunE).
func BuildRegistryFromCobra(root *cobra.Command) []Command {
	var commands []Command
	walkCobra(root, "", "", &commands)
	return commands
}

// walkCobra recursively walks the Cobra command tree.
func walkCobra(cmd *cobra.Command, parentPath, parentCategory string, out *[]Command) {
	if cmd.Hidden {
		return
	}

	fullPath := cmd.Name()
	if parentPath != "" {
		fullPath = parentPath + " " + cmd.Name()
	}

	category := parentCategory
	if category == "" && cmd.Parent() != nil {
		category = cmd.Name()
	}

	subs := cmd.Commands()
	if len(subs) == 0 {
		// Leaf command — always add
		*out = append(*out, Command{
			Name:        fullPath,
			Description: cmd.Short,
			Category:    category,
			Icon:        iconForCategory(category),
		})
	} else {
		// Branch command — recurse into children
		for _, sub := range subs {
			walkCobra(sub, fullPath, category, out)
		}
	}
}

// iconForCategory returns an emoji icon for a command category.
func iconForCategory(category string) string {
	switch category {
	case "bucket":
		return "🪣"
	case "object":
		return "📦"
	case "worker":
		return "⚡"
	case "kv":
		return "🔑"
	case "dns":
		return "🌐"
	case "zone":
		return "🏷️"
	case "ssl":
		return "🔒"
	case "cache":
		return "💨"
	case "config":
		return "⚙️"
	case "doctor":
		return "🩺"
	case "domains":
		return "🌍"
	case "analytics":
		return "📊"
	default:
		return "▸"
	}
}

// DefaultDashboardActions returns hardcoded navigation and dashboard actions
// that are always available in the palette.
func DefaultDashboardActions() []Command {
	return []Command{
		{
			Name:        "Go to Overview",
			Description: "Switch to the overview section",
			Category:    "Navigation",
			Icon:        "🏠",
		},
		{
			Name:        "Go to Browser",
			Description: "Switch to the bucket/object browser",
			Category:    "Navigation",
			Icon:        "🪣",
		},
		{
			Name:        "Go to Upload",
			Description: "Switch to the upload section",
			Category:    "Navigation",
			Icon:        "📤",
		},
		{
			Name:        "Go to Monitoring",
			Description: "Switch to monitoring dashboard",
			Category:    "Navigation",
			Icon:        "📈",
		},
		{
			Name:        "Go to Settings",
			Description: "Switch to settings",
			Category:    "Navigation",
			Icon:        "⚙️",
		},
		{
			Name:        "Toggle Help",
			Description: "Show or hide the help panel",
			Category:    "Dashboard",
			Icon:        "❓",
		},
		{
			Name:        "Refresh Data",
			Description: "Reload data from the API",
			Category:    "Dashboard",
			Icon:        "🔄",
		},
		{
			Name:        "Quit",
			Description: "Exit the dashboard",
			Category:    "Dashboard",
			Icon:        "🚪",
		},
	}
}
