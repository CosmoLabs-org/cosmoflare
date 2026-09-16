package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
)

// envFlagTestEnv isolates the --env resolution globals against a temp HOME.
// It snapshots and restores EnvProfile, ActiveProfile, the credential
// package vars, and the limits plan flag so tests never leak state.
func envFlagTestEnv(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	oldEnv, oldActive := EnvProfile, ActiveProfile
	oldID, oldToken, oldPlan := AccountID, APIToken, limitsPlan
	t.Cleanup(func() {
		EnvProfile, ActiveProfile = oldEnv, oldActive
		AccountID, APIToken, limitsPlan = oldID, oldToken, oldPlan
	})
	EnvProfile, ActiveProfile = "", nil
	AccountID, APIToken, limitsPlan = "", "", ""
	return os.Getenv("HOME")
}

// envFlagWriteConfig writes a two-profile config.yaml into the isolated
// HOME and fails the test on I/O error.
func envFlagWriteConfig(t *testing.T, home string) {
	t.Helper()
	const yaml = `current: default
profiles:
  prod:
    name: prod
    account_id: a1b2c3d4e5f60718293a4b5c6d7e8f90
    api_token: tok-prod-abcdef1234567890
    plan_tier: paid
  dev:
    name: dev
    account_id: 0f9e8d7c6b5a4938271605948372615a4
    api_token: tok-dev-abcdef1234567890
    plan_tier: free
`
	dir := filepath.Join(home, ".cosmoflare")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("creating config dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0600); err != nil {
		t.Fatalf("writing config.yaml failed: %v", err)
	}
}

// TestEnvFlag_RegisteredOnRoot verifies the persistent flag exists with the
// invocation-scoped wording.
func TestEnvFlag_RegisteredOnRoot(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("env")
	if flag == nil {
		t.Fatal("--env persistent flag is not registered on root")
	}
	if !strings.Contains(flag.Usage, "does not change the current profile") {
		t.Errorf("--env usage %q should promise not to change the current profile", flag.Usage)
	}
}

// TestEnvFlag_UnknownNameErrors verifies the actionable listing error for an
// unknown environment name.
func TestEnvFlag_UnknownNameErrors(t *testing.T) {
	home := envFlagTestEnv(t)
	envFlagWriteConfig(t, home)
	EnvProfile = "staging"

	err := resolveActiveEnv()
	if err == nil {
		t.Fatal("expected error for unknown environment")
	}
	want := `unknown environment "staging": known environments are`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q missing prefix %q", err.Error(), want)
	}
	for _, name := range []string{"prod", "dev"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q should list known environment %q", err.Error(), name)
		}
	}
	if ActiveProfile != nil {
		t.Error("ActiveProfile must stay nil after a failed resolution")
	}
}

// TestEnvFlag_KnownSelectsCredentials verifies that a known profile's
// AccountID/APIToken take precedence over the ambient environment variables
// for the invocation and that ActiveProfile is populated.
func TestEnvFlag_KnownSelectsCredentials(t *testing.T) {
	home := envFlagTestEnv(t)
	envFlagWriteConfig(t, home)

	// Ambient environment variables that the profile must outrank.
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "ambient-account-id")
	t.Setenv("CLOUDFLARE_API_TOKEN", "ambient-api-token")
	EnvProfile = "prod"

	if err := resolveActiveEnv(); err != nil {
		t.Fatalf("resolveActiveEnv failed: %v", err)
	}
	if ActiveProfile == nil || ActiveProfile.Name != "prod" {
		t.Fatalf("ActiveProfile = %+v, want the prod profile", ActiveProfile)
	}
	if AccountID != "a1b2c3d4e5f60718293a4b5c6d7e8f90" {
		t.Errorf("AccountID = %q, want the prod profile account (profile outranks ambient env)", AccountID)
	}
	if APIToken != "tok-prod-abcdef1234567890" {
		t.Errorf("APIToken = %q, want the prod profile token (profile outranks ambient env)", APIToken)
	}
}

// TestEnvFlag_ExplicitFlagsWin verifies explicit --account-id/--api-token
// flags still outrank the profile: the profile only fills unset variables.
func TestEnvFlag_ExplicitFlagsWin(t *testing.T) {
	home := envFlagTestEnv(t)
	envFlagWriteConfig(t, home)
	AccountID, APIToken = "flag-account-id", "flag-api-token-123"
	EnvProfile = "prod"

	if err := resolveActiveEnv(); err != nil {
		t.Fatalf("resolveActiveEnv failed: %v", err)
	}
	if AccountID != "flag-account-id" || APIToken != "flag-api-token-123" {
		t.Errorf("explicit flags were overridden: AccountID=%q APIToken=%q", AccountID, APIToken)
	}
	if ActiveProfile == nil || ActiveProfile.Name != "prod" {
		t.Errorf("ActiveProfile = %+v, want the prod profile", ActiveProfile)
	}
}

// TestEnvFlag_UnsetKeepsCurrentBehavior verifies that without --env nothing
// is resolved, ActiveProfile is cleared, and the credential vars are left
// untouched for the existing ambient-env fallback.
func TestEnvFlag_UnsetKeepsCurrentBehavior(t *testing.T) {
	envFlagTestEnv(t)
	ActiveProfile = &config.Profile{Name: "stale"}
	EnvProfile = ""

	if err := resolveActiveEnv(); err != nil {
		t.Fatalf("resolveActiveEnv with no --env should be a no-op: %v", err)
	}
	if ActiveProfile != nil {
		t.Errorf("ActiveProfile = %+v, want nil when --env is not given", ActiveProfile)
	}
	if AccountID != "" || APIToken != "" {
		t.Errorf("credential vars changed without --env: AccountID=%q APIToken=%q", AccountID, APIToken)
	}
}

// TestEnvFlag_InvalidPlanTierFails verifies a profile with an invalid
// plan_tier fails resolution before any command runs.
func TestEnvFlag_InvalidPlanTierFails(t *testing.T) {
	home := envFlagTestEnv(t)
	envFlagWriteConfig(t, home)
	yamlPath := filepath.Join(home, ".cosmoflare", "config.yaml")
	bad, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("reading config failed: %v", err)
	}
	bad = []byte(strings.Replace(string(bad), "plan_tier: paid", "plan_tier: enterprise", 1))
	if err := os.WriteFile(yamlPath, bad, 0600); err != nil {
		t.Fatalf("rewriting config failed: %v", err)
	}
	EnvProfile = "prod"

	err = resolveActiveEnv()
	if err == nil {
		t.Fatal("expected error for invalid plan_tier")
	}
	if !strings.Contains(err.Error(), "plan_tier") || !strings.Contains(err.Error(), `"free" or "paid"`) {
		t.Errorf("error %q should explain the accepted plan tiers", err.Error())
	}
}

// TestEnvFlag_LimitsPlanTier verifies the limits plan-tier passthrough:
// the profile tier is used when --plan is not given, and the --plan flag
// still wins when provided.
func TestEnvFlag_LimitsPlanTier(t *testing.T) {
	envFlagTestEnv(t)

	ActiveProfile = nil
	limitsPlan = ""
	if got := effectiveFlagPlan(); got != "" {
		t.Errorf("no profile, no flag: got %q, want empty", got)
	}

	ActiveProfile = &config.Profile{Name: "prod", PlanTier: "paid"}
	if got := effectiveFlagPlan(); got != "paid" {
		t.Errorf("profile tier: got %q, want %q", got, "paid")
	}

	limitsPlan = "free"
	if got := effectiveFlagPlan(); got != "free" {
		t.Errorf("--plan flag: got %q, want %q (flag outranks profile)", got, "free")
	}
}
