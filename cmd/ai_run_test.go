package cmd

import (
	"strings"
	"testing"
)

// aiRunGlobals snapshots and restores every package-level variable the AI run
// functions read, so tests cannot leak state.
func aiRunGlobals(t *testing.T) {
	t.Helper()
	oldFilter := aiModelsFilter
	oldPrompt, oldSystem := aiRunPrompt, aiRunSystem
	oldTTL, oldLimit, oldWindow := aiGatewayCacheTTL, aiGatewayRateLimit, aiGatewayRateWindow
	oldCollect, oldForce, oldLogs := aiGatewayCollectLogs, aiGatewayForce, aiGatewayLogsLimit
	oldAcct, oldToken := AccountID, APIToken
	oldJSON, oldDry := JSONOutput, DryRun
	t.Cleanup(func() {
		aiModelsFilter = oldFilter
		aiRunPrompt, aiRunSystem = oldPrompt, oldSystem
		aiGatewayCacheTTL, aiGatewayRateLimit, aiGatewayRateWindow = oldTTL, oldLimit, oldWindow
		aiGatewayCollectLogs, aiGatewayForce, aiGatewayLogsLimit = oldCollect, oldForce, oldLogs
		AccountID, APIToken = oldAcct, oldToken
		JSONOutput, DryRun = oldJSON, oldDry
	})
}

// aiRunDefaults installs the pristine flag state each subtest needs.
func aiRunDefaults() {
	aiModelsFilter = ""
	aiRunPrompt = ""
	aiRunSystem = ""
	aiGatewayCacheTTL = 0
	aiGatewayRateLimit = 0
	aiGatewayRateWindow = 60
	aiGatewayCollectLogs = false
	aiGatewayForce = false
	aiGatewayLogsLimit = 25
	AccountID = "test-account"
	APIToken = ""
	JSONOutput = false
	DryRun = false
}

// aiRunCmdFns returns every AI run function that requires live credentials,
// so table-driven tests can cover the shared validation branches uniformly.
func aiRunCmdFns() map[string]func() error {
	return map[string]func() error{
		"models list":  func() error { return runAIModelsList(aiModelsListCmd, nil) },
		"models get":   func() error { return runAIModelsGet(aiModelsGetCmd, []string{"@cf/meta/llama-3-8b-instruct"}) },
		"gateway list": func() error { return runAIGatewayList(aiGatewayListCmd, nil) },
		"gateway create": func() error {
			return runAIGatewayCreate(aiGatewayCreateCmd, []string{"my-gateway"})
		},
		"gateway delete": func() error {
			return runAIGatewayDelete(aiGatewayDeleteCmd, []string{"my-gateway"})
		},
		"gateway logs": func() error {
			return runAIGatewayLogs(aiGatewayLogsCmd, []string{"my-gateway"})
		},
	}
}

// TestRunAIRun_RequiresPrompt verifies the argument guard fires before any
// service or network interaction.
func TestRunAIRun_RequiresPrompt(t *testing.T) {
	aiRunGlobals(t)
	aiRunDefaults()
	DryRun = true // even dry-run must not skip the prompt guard

	err := runAIRun(aiRunCmd, []string{"@cf/meta/llama-3-8b-instruct"})
	if err == nil || !strings.Contains(err.Error(), "--prompt is required") {
		t.Fatalf("expected prompt-required error, got %v", err)
	}
	if !strings.Contains(err.Error(), "Usage: cosmoflare ai run") {
		t.Fatalf("expected usage hint in error, got %v", err)
	}
}

// TestRunAIRun_DryRun verifies the dry-run short-circuit needs no credentials.
func TestRunAIRun_DryRun(t *testing.T) {
	cases := []struct {
		name   string
		system string
	}{
		{"plain prompt", ""},
		{"with system message", "You are a pirate"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			aiRunGlobals(t)
			aiRunDefaults()
			aiRunPrompt = "What is Go?"
			aiRunSystem = tc.system
			DryRun = true

			if err := runAIRun(aiRunCmd, []string{"@cf/meta/llama-3-8b-instruct"}); err != nil {
				t.Fatalf("dry-run inference should succeed: %v", err)
			}
		})
	}
}

// TestRunAIRun_DryRunJSON verifies the JSON dry-run envelope path.
func TestRunAIRun_DryRunJSON(t *testing.T) {
	aiRunGlobals(t)
	aiRunDefaults()
	aiRunPrompt = "Hello"
	DryRun = true
	JSONOutput = true

	if err := runAIRun(aiRunCmd, []string{"@cf/openai/whisper"}); err != nil {
		t.Fatalf("dry-run JSON inference should succeed: %v", err)
	}
}

// TestRunAIRun_MissingAccountID verifies the service-construction error path
// when no account ID is set (non-dry-run only).
func TestRunAIRun_MissingAccountID(t *testing.T) {
	aiRunGlobals(t)
	aiRunDefaults()
	aiRunPrompt = "Hello"
	AccountID = ""

	err := runAIRun(aiRunCmd, []string{"@cf/meta/llama-3-8b-instruct"})
	if err == nil || !strings.Contains(err.Error(), "failed to create AI service") {
		t.Fatalf("expected service creation error, got %v", err)
	}
	if !strings.Contains(err.Error(), "account ID is required") {
		t.Fatalf("expected wrapped account ID error, got %v", err)
	}
}

// TestRunAI_MissingAccountID verifies every credential-backed AI command
// fails fast on a missing account ID.
func TestRunAI_MissingAccountID(t *testing.T) {
	fns := aiRunCmdFns()
	names := []string{"models list", "models get", "gateway list", "gateway create", "gateway delete", "gateway logs"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			aiRunGlobals(t)
			aiRunDefaults()
			AccountID = ""

			err := fns[name]()
			if err == nil || !strings.Contains(err.Error(), "failed to create AI service") {
				t.Fatalf("expected service creation error, got %v", err)
			}
			if !strings.Contains(err.Error(), "account ID is required") {
				t.Fatalf("expected wrapped account ID error, got %v", err)
			}
		})
	}
}

// TestRunAI_MissingAPIToken verifies the token guard surfaces through service
// construction for every command.
func TestRunAI_MissingAPIToken(t *testing.T) {
	fns := aiRunCmdFns()
	names := []string{"models list", "models get", "gateway list", "gateway create", "gateway delete", "gateway logs"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			aiRunGlobals(t)
			aiRunDefaults()
			APIToken = ""

			err := fns[name]()
			if err == nil || !strings.Contains(err.Error(), "API token is required") {
				t.Fatalf("expected API token error, got %v", err)
			}
		})
	}
}

// TestRunAIGatewayCreate_DryRun verifies the dry-run envelope, including the
// cache/rate-limit hint lines.
func TestRunAIGatewayCreate_DryRun(t *testing.T) {
	cases := []struct {
		name  string
		ttl   int
		limit int
	}{
		{"defaults", 0, 0},
		{"with cache and rate limit", 300, 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			aiRunGlobals(t)
			aiRunDefaults()
			aiGatewayCacheTTL = tc.ttl
			aiGatewayRateLimit = tc.limit
			DryRun = true

			if err := runAIGatewayCreate(aiGatewayCreateCmd, []string{"prod-gw"}); err != nil {
				t.Fatalf("dry-run gateway create should succeed: %v", err)
			}
		})
	}
}

// TestRunAIGatewayCreate_DryRunJSON verifies the JSON dry-run envelope for
// gateway creation with log collection enabled.
func TestRunAIGatewayCreate_DryRunJSON(t *testing.T) {
	aiRunGlobals(t)
	aiRunDefaults()
	aiGatewayCacheTTL = 300
	aiGatewayCollectLogs = true
	DryRun = true
	JSONOutput = true

	if err := runAIGatewayCreate(aiGatewayCreateCmd, []string{"my-gateway"}); err != nil {
		t.Fatalf("dry-run JSON gateway create should succeed: %v", err)
	}
}

// TestRunAIGatewayDelete_DryRun verifies the delete dry-run short-circuit
// needs no credentials and never prompts.
func TestRunAIGatewayDelete_DryRun(t *testing.T) {
	aiRunGlobals(t)
	aiRunDefaults()
	DryRun = true
	aiGatewayForce = false

	if err := runAIGatewayDelete(aiGatewayDeleteCmd, []string{"old-gw"}); err != nil {
		t.Fatalf("dry-run gateway delete should succeed: %v", err)
	}
}
