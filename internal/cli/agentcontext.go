/*
Package cli — agent-context detection (FEAT-pN68ZRM).

Copyright © 2025-2026 CosmoLabs (https://cosmolabs.org)
License: MIT
*/

package cli

import "os"

// agentEnvVars: harness markers set by AI coding agents. Mirrors cf's
// agent-context detection (packages/cli/src/lib/agent-context.ts in
// cloudflare/cf): each harness exports a stable marker when a command runs
// inside it.
var agentEnvVars = []string{
	"CLAUDECODE",            // Claude Code
	"CLAUDE_CODE_ENTRYPOINT",// Claude Code (alternate entrypoints)
	"CURSOR_AGENT",          // Cursor's agent mode
	"GEMINI_CLI",            // Gemini CLI
	"CODEX_SANDBOX",         // OpenAI Codex CLI
	"AGENT",                 // generic convention
}

// IsAgentEnvironment reports whether the CLI is being invoked from inside
// an AI agent harness (checked via environment markers).
func IsAgentEnvironment() bool {
	for _, key := range agentEnvVars {
		if os.Getenv(key) != "" {
			return true
		}
	}
	return false
}

// AgentHelpPreamble is prepended to --help output when an agent harness is
// detected (FEAT-pN68ZRM): the shortest path to correct, machine-readable
// usage. Human terminals never see it.
const AgentHelpPreamble = `Running inside an AI agent — quick contract:
  • Every command accepts --json for machine-readable output.
  • cosmoflare search "<query>" finds commands by name or description.
  • In --json mode failures carry a stable error_code field.
  • Exit codes: 0 success, 1 failure (deterministic, scriptable).

`
