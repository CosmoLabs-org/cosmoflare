package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/internal/config"
)

// configRunEnv isolates the config runner against a temp HOME with the OS
// keychain disabled and snapshots the output-mode globals.
func configRunEnv(t *testing.T) string {
	t.Helper()
	t.Setenv("COSMOFLARE_NO_KEYCHAIN", "1") // never probe the OS keychain
	home := t.TempDir()
	t.Setenv("HOME", home)

	oldDry, oldJSON := DryRun, JSONOutput
	t.Cleanup(func() {
		DryRun, JSONOutput = oldDry, oldJSON
	})
	return home
}

// configRunSeedProfile writes one valid profile into the isolated config file
// and returns a manager sharing that file.
func configRunSeedProfile(t *testing.T, name string) *config.ConfigManager {
	t.Helper()
	cm, err := config.NewConfigManager()
	if err != nil {
		t.Fatalf("NewConfigManager failed: %v", err)
	}
	if err := cm.SetProfile(&config.Profile{
		Name:        name,
		AccountID:   "12345678901234567890123456789012",
		APIToken:    "test-token-abcdef1234567890",
		Description: "seeded profile",
	}); err != nil {
		t.Fatalf("SetProfile failed: %v", err)
	}
	return cm
}

// configRunReadAll drains a captured stdout pipe after restoration.
func configRunReadAll(t *testing.T, r *os.File) string {
	t.Helper()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout failed: %v", err)
	}
	return string(out)
}

// TestGetConfigManager_PointsAtIsolatedHome verifies the manager helper reads
// its config path from $HOME so tests never touch the real user config.
func TestGetConfigManager_PointsAtIsolatedHome(t *testing.T) {
	home := configRunEnv(t)

	cm, err := getConfigManager()
	if err != nil {
		t.Fatalf("getConfigManager failed: %v", err)
	}
	want := filepath.Join(home, ".cosmoflare", "config.yaml")
	if cm.GetConfigPath() != want {
		t.Fatalf("config path = %q, want %q", cm.GetConfigPath(), want)
	}
}

// TestRunConfigInit_AlreadyExists verifies init is a no-op (success) when a
// config file already exists, instead of overwriting it.
func TestRunConfigInit_AlreadyExists(t *testing.T) {
	home := configRunEnv(t)
	if err := os.MkdirAll(filepath.Join(home, ".cosmoflare"), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".cosmoflare", "config.yaml"), []byte("current: default\n"), 0600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if err := runConfigInit(configInitCmd, nil); err != nil {
		t.Fatalf("init on existing config should succeed: %v", err)
	}
}

// TestRunConfigValidate_ErrorPaths verifies validation failures for a missing
// named profile, a missing current profile, and an invalid stored profile.
func TestRunConfigValidate_ErrorPaths(t *testing.T) {
	t.Run("named profile missing", func(t *testing.T) {
		configRunEnv(t)
		err := runConfigValidate(configValidateCmd, []string{"ghost"})
		if err == nil {
			t.Fatal("validating a nonexistent profile should fail")
		}
	})
	t.Run("no current profile", func(t *testing.T) {
		configRunEnv(t)
		err := runConfigValidate(configValidateCmd, nil)
		if err == nil {
			t.Fatal("validating with no profiles configured should fail")
		}
	})
	t.Run("stored profile invalid", func(t *testing.T) {
		configRunEnv(t)
		cm, err := config.NewConfigManager()
		if err != nil {
			t.Fatalf("NewConfigManager failed: %v", err)
		}
		if err := cm.SetProfile(&config.Profile{Name: "broken", AccountID: "tooshort", APIToken: "test-token-abcdef1234567890"}); err != nil {
			t.Fatalf("SetProfile failed: %v", err)
		}
		err = runConfigValidate(configValidateCmd, []string{"broken"})
		if err == nil {
			t.Fatal("validating a profile with a malformed account ID should fail")
		}
	})
}

// TestRunConfigList verifies list output for empty and populated configs.
func TestRunConfigList(t *testing.T) {
	t.Run("empty config succeeds", func(t *testing.T) {
		configRunEnv(t)
		if err := runConfigList(configListCmd, nil); err != nil {
			t.Fatalf("listing an empty config should succeed: %v", err)
		}
	})
	t.Run("populated config prints profiles", func(t *testing.T) {
		configRunEnv(t)
		configRunSeedProfile(t, "work")
		configRunSeedProfile(t, "personal")

		r, restore := captureStdout(t)
		err := runConfigList(configListCmd, nil)
		restore()
		if err != nil {
			t.Fatalf("list should succeed: %v", err)
		}
		out := configRunReadAll(t, r)
		for _, want := range []string{"work", "personal", "PROFILE"} {
			if !strings.Contains(out, want) {
				t.Errorf("list output missing %q;\noutput:\n%s", want, out)
			}
		}
	})
}

// TestRunConfigShow verifies profile display, secret masking by default, and
// the --show-secrets override in JSON mode.
func TestRunConfigShow(t *testing.T) {
	t.Run("missing profile errors", func(t *testing.T) {
		configRunEnv(t)
		if err := runConfigShow(configShowCmd, []string{"ghost"}); err == nil {
			t.Fatal("showing a nonexistent profile should fail")
		}
	})
	t.Run("json masks secrets by default", func(t *testing.T) {
		env := configRunEnv(t)
		configRunSeedProfile(t, "work")
		JSONOutput = true
		_ = env

		r, restore := captureStdout(t)
		err := runConfigShow(configShowCmd, []string{"work"})
		restore()
		if err != nil {
			t.Fatalf("show should succeed: %v", err)
		}
		out := configRunReadAll(t, r)
		if strings.Contains(out, "test-token-abcdef1234567890") {
			t.Error("raw API token must not appear in masked JSON output")
		}
	})
	t.Run("json show-secrets reveals token", func(t *testing.T) {
		configRunEnv(t)
		configRunSeedProfile(t, "work")
		JSONOutput = true
		if err := configShowCmd.Flags().Set("show-secrets", "true"); err != nil {
			t.Fatalf("setting show-secrets failed: %v", err)
		}
		t.Cleanup(func() {
			configShowCmd.Flags().Set("show-secrets", "false")
			if f := configShowCmd.Flags().Lookup("show-secrets"); f != nil {
				f.Changed = false
			}
		})

		r, restore := captureStdout(t)
		err := runConfigShow(configShowCmd, []string{"work"})
		restore()
		if err != nil {
			t.Fatalf("show should succeed: %v", err)
		}
		out := configRunReadAll(t, r)
		if !strings.Contains(out, "test-token-abcdef1234567890") {
			t.Errorf("raw API token should appear with --show-secrets;\noutput:\n%s", out)
		}
	})
}

// TestRunConfigSet verifies set's guards, validation, persistence, and
// merging of unspecified fields from an existing profile.
func TestRunConfigSet(t *testing.T) {
	t.Run("profile name required", func(t *testing.T) {
		configRunEnv(t)
		err := runConfigSet(configSetCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "profile name is required") {
			t.Fatalf("expected name-required error, got %v", err)
		}
	})
	t.Run("validation failure surfaces", func(t *testing.T) {
		configRunEnv(t)
		if err := configSetCmd.Flags().Set("interactive", "false"); err != nil {
			t.Fatalf("setting interactive=false failed: %v", err)
		}
		t.Cleanup(func() {
			configSetCmd.Flags().Set("interactive", "true")
			if f := configSetCmd.Flags().Lookup("interactive"); f != nil {
				f.Changed = false
			}
		})
		if err := configSetCmd.Flags().Set("account-id", "tooshort"); err != nil {
			t.Fatalf("setting account-id failed: %v", err)
		}

		err := runConfigSet(configSetCmd, []string{"bad"})
		if err == nil || !strings.Contains(err.Error(), "profile validation failed") {
			t.Fatalf("expected validation error, got %v", err)
		}
	})
	t.Run("persists a valid profile", func(t *testing.T) {
		configRunEnv(t)
		if err := configSetCmd.Flags().Set("interactive", "false"); err != nil {
			t.Fatalf("setting interactive=false failed: %v", err)
		}
		t.Cleanup(func() {
			configSetCmd.Flags().Set("interactive", "true")
			// Reset value flags so later subtests do not inherit them
			// (runConfigSet applies any non-empty flag value).
			configSetCmd.Flags().Set("account-id", "")
			configSetCmd.Flags().Set("api-token", "")
			configSetCmd.Flags().Set("description", "")
			for _, name := range []string{"interactive", "account-id", "api-token", "description"} {
				if f := configSetCmd.Flags().Lookup(name); f != nil {
					f.Changed = false
				}
			}
		})
		if err := configSetCmd.Flags().Set("account-id", "123456789012345678901234567890ab"); err != nil {
			t.Fatalf("setting account-id failed: %v", err)
		}
		if err := configSetCmd.Flags().Set("api-token", "fresh-token-1234567890"); err != nil {
			t.Fatalf("setting api-token failed: %v", err)
		}
		if err := configSetCmd.Flags().Set("description", "from test"); err != nil {
			t.Fatalf("setting description failed: %v", err)
		}

		if err := runConfigSet(configSetCmd, []string{"fresh"}); err != nil {
			t.Fatalf("set should succeed: %v", err)
		}

		cm, err := config.NewConfigManager()
		if err != nil {
			t.Fatalf("reloading config failed: %v", err)
		}
		p, err := cm.GetProfile("fresh")
		if err != nil {
			t.Fatalf("saved profile missing: %v", err)
		}
		if p.AccountID != "123456789012345678901234567890ab" || p.APIToken != "fresh-token-1234567890" || p.Description != "from test" {
			t.Errorf("saved profile mismatch: %+v", p)
		}
	})
	t.Run("merges unspecified fields from existing profile", func(t *testing.T) {
		configRunEnv(t)
		configRunSeedProfile(t, "keep")
		if err := configSetCmd.Flags().Set("interactive", "false"); err != nil {
			t.Fatalf("setting interactive=false failed: %v", err)
		}
		t.Cleanup(func() {
			configSetCmd.Flags().Set("interactive", "true")
			if f := configSetCmd.Flags().Lookup("interactive"); f != nil {
				f.Changed = false
			}
		})
		if err := configSetCmd.Flags().Set("api-token", "rotated-token-1234567890"); err != nil {
			t.Fatalf("setting api-token failed: %v", err)
		}

		if err := runConfigSet(configSetCmd, []string{"keep"}); err != nil {
			t.Fatalf("set should succeed: %v", err)
		}

		cm, err := config.NewConfigManager()
		if err != nil {
			t.Fatalf("reloading config failed: %v", err)
		}
		p, err := cm.GetProfile("keep")
		if err != nil {
			t.Fatalf("profile missing after update: %v", err)
		}
		if p.AccountID != "12345678901234567890123456789012" {
			t.Errorf("account ID should be preserved, got %q", p.AccountID)
		}
		if p.APIToken != "rotated-token-1234567890" {
			t.Errorf("api token should be rotated, got %q", p.APIToken)
		}
	})
}

// TestRunConfigDelete verifies delete guards and the dry-run no-op.
func TestRunConfigDelete(t *testing.T) {
	t.Run("profile name required", func(t *testing.T) {
		configRunEnv(t)
		err := runConfigDelete(configDeleteCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "profile name is required") {
			t.Fatalf("expected name-required error, got %v", err)
		}
	})
	t.Run("missing profile errors", func(t *testing.T) {
		configRunEnv(t)
		err := runConfigDelete(configDeleteCmd, []string{"ghost"})
		if err == nil || !strings.Contains(err.Error(), "does not exist") {
			t.Fatalf("expected does-not-exist error, got %v", err)
		}
	})
	t.Run("dry run keeps profile", func(t *testing.T) {
		configRunEnv(t)
		configRunSeedProfile(t, "doomed")
		DryRun = true

		if err := runConfigDelete(configDeleteCmd, []string{"doomed"}); err != nil {
			t.Fatalf("dry-run delete should succeed: %v", err)
		}

		cm, err := config.NewConfigManager()
		if err != nil {
			t.Fatalf("reloading config failed: %v", err)
		}
		if !cm.ProfileExists("doomed") {
			t.Error("dry-run delete must not remove the profile")
		}
	})
}

// TestRunConfigSwitch verifies switch guards and that a successful switch
// persists a new current profile.
func TestRunConfigSwitch(t *testing.T) {
	t.Run("profile name required", func(t *testing.T) {
		configRunEnv(t)
		err := runConfigSwitch(configSwitchCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "profile name is required") {
			t.Fatalf("expected name-required error, got %v", err)
		}
	})
	t.Run("missing profile errors", func(t *testing.T) {
		configRunEnv(t)
		err := runConfigSwitch(configSwitchCmd, []string{"ghost"})
		if err == nil || !strings.Contains(err.Error(), "does not exist") {
			t.Fatalf("expected does-not-exist error, got %v", err)
		}
	})
	t.Run("switch persists current profile", func(t *testing.T) {
		configRunEnv(t)
		configRunSeedProfile(t, "alpha")
		configRunSeedProfile(t, "beta")

		if err := runConfigSwitch(configSwitchCmd, []string{"beta"}); err != nil {
			t.Fatalf("switch should succeed: %v", err)
		}

		cm, err := config.NewConfigManager()
		if err != nil {
			t.Fatalf("reloading config failed: %v", err)
		}
		cur, err := cm.GetCurrent()
		if err != nil {
			t.Fatalf("GetCurrent failed: %v", err)
		}
		if cur.Name != "beta" {
			t.Errorf("current profile = %q, want %q", cur.Name, "beta")
		}
	})
}

// TestRunConfigExport verifies export guards and the emitted env-var lines.
func TestRunConfigExport(t *testing.T) {
	t.Run("missing profile errors", func(t *testing.T) {
		configRunEnv(t)
		err := runConfigExport(configExportCmd, []string{"ghost"})
		if err == nil || !strings.Contains(err.Error(), "failed to get profile") {
			t.Fatalf("expected get-profile error, got %v", err)
		}
	})
	t.Run("exports named profile as shell exports", func(t *testing.T) {
		configRunEnv(t)
		configRunSeedProfile(t, "work")

		r, restore := captureStdout(t)
		err := runConfigExport(configExportCmd, []string{"work"})
		restore()
		if err != nil {
			t.Fatalf("export should succeed: %v", err)
		}
		out := configRunReadAll(t, r)
		if !strings.Contains(out, "export CLOUDFLARE_API_TOKEN=") {
			t.Errorf("export output missing API token line;\noutput:\n%s", out)
		}
		if !strings.Contains(out, "export CLOUDFLARE_ACCOUNT_ID=") {
			t.Errorf("export output missing account ID line;\noutput:\n%s", out)
		}
	})
}

// configRunWithStdin replaces os.Stdin with a pipe pre-loaded with input and
// restores it when the test ends.
func configRunWithStdin(t *testing.T, input string) {
	t.Helper()
	saved := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("writing stdin fixture failed: %v", err)
	}
	w.Close()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = saved })
}

// TestCreateProfileInteractive verifies interactive prompting reads fields
// from stdin and falls back to environment variables for empty answers.
func TestCreateProfileInteractive(t *testing.T) {
	t.Run("reads values from stdin", func(t *testing.T) {
		env := configRunEnv(t)
		_ = env
		configRunWithStdin(t, "123456789012345678901234567890ff\ntyped-token-abcdef\nMy profile\n")

		p, err := createProfileInteractive("typed")
		if err != nil {
			t.Fatalf("createProfileInteractive failed: %v", err)
		}
		if p.Name != "typed" || p.AccountID != "123456789012345678901234567890ff" || p.APIToken != "typed-token-abcdef" || p.Description != "My profile" {
			t.Errorf("profile mismatch: %+v", p)
		}
		if p.Region != "auto" {
			t.Errorf("region = %q, want auto", p.Region)
		}
	})
	t.Run("falls back to environment variables", func(t *testing.T) {
		env := configRunEnv(t)
		t.Setenv("CLOUDFLARE_ACCOUNT_ID", "env-account-000000000000000000000000")
		t.Setenv("CLOUDFLARE_API_TOKEN", "env-token-abcdef123456")
		_ = env
		configRunWithStdin(t, "\n\n\n")

		p, err := createProfileInteractive("envy")
		if err != nil {
			t.Fatalf("createProfileInteractive failed: %v", err)
		}
		if p.AccountID != "env-account-000000000000000000000000" || p.APIToken != "env-token-abcdef123456" {
			t.Errorf("env fallback mismatch: %+v", p)
		}
	})
}

// TestFillProfileInteractively verifies only missing fields are prompted for
// and already-populated fields are left untouched.
func TestFillProfileInteractively(t *testing.T) {
	configRunEnv(t)
	configRunWithStdin(t, "filled-token-abcdef123\nFilled description\n")

	p := &config.Profile{
		Name:      "partial",
		AccountID: "12345678901234567890123456789012",
	}
	if err := fillProfileInteractively(p); err != nil {
		t.Fatalf("fillProfileInteractively failed: %v", err)
	}
	if p.AccountID != "12345678901234567890123456789012" {
		t.Errorf("account ID should be untouched, got %q", p.AccountID)
	}
	if p.APIToken != "filled-token-abcdef123" || p.Description != "Filled description" {
		t.Errorf("prompted fields mismatch: %+v", p)
	}
}
