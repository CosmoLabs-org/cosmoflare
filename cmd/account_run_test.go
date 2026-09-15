package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// accountRunEnv isolates the account runners against a temp HOME and
// snapshots the account flag variables plus output-mode globals.
func accountRunEnv(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	oldID, oldToken, oldEmail := accountAddAccountID, accountAddAPIToken, accountAddEmail
	oldForce, oldJSON, oldDry := accountRemoveForce, JSONOutput, DryRun
	t.Cleanup(func() {
		accountAddAccountID, accountAddAPIToken, accountAddEmail = oldID, oldToken, oldEmail
		accountRemoveForce, JSONOutput, DryRun = oldForce, oldJSON, oldDry
	})
	accountAddAccountID, accountAddAPIToken, accountAddEmail = "", "", ""
	accountRemoveForce, JSONOutput, DryRun = false, false, false
	return os.Getenv("HOME")
}

// accountRunAdd invokes the add runner with the given flags set.
func accountRunAdd(t *testing.T, name, id, token string) error {
	t.Helper()
	accountAddAccountID, accountAddAPIToken = id, token
	return runAccountAdd(accountAddCmd, []string{name})
}

// accountRunSeedAdded adds one account and fails the test on error.
func accountRunSeedAdded(t *testing.T, name string) {
	t.Helper()
	accountAddAccountID, accountAddAPIToken = name+"-id-1234567890abcdef", "tok-"+name+"-abcdef123456"
	if err := runAccountAdd(accountAddCmd, []string{name}); err != nil {
		t.Fatalf("seeding account %q failed: %v", name, err)
	}
}

// accountRunFilePath returns the expected path of a file in the isolated
// ~/.cosmoflare config directory.
func accountRunFilePath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(os.Getenv("HOME"), ".cosmoflare", name)
}

func TestRunAccountList_Empty(t *testing.T) {
	accountRunEnv(t)

	if err := runAccountList(accountListCmd, nil); err != nil {
		t.Fatalf("listing with no accounts should succeed: %v", err)
	}
}

func TestRunAccountList_EmptyJSON(t *testing.T) {
	accountRunEnv(t)
	JSONOutput = true

	if err := runAccountList(accountListCmd, nil); err != nil {
		t.Fatalf("JSON listing with no accounts should succeed: %v", err)
	}
}

func TestRunAccountAdd_Success(t *testing.T) {
	home := accountRunEnv(t)

	if err := accountRunAdd(t, "prod", "prod-id-1234567890abcdef", "tok-prod-abcdef123456"); err != nil {
		t.Fatalf("add should succeed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".cosmoflare", "accounts.yaml")); err != nil {
		t.Fatalf("accounts.yaml should exist after add: %v", err)
	}
	if err := runAccountList(accountListCmd, nil); err != nil {
		t.Fatalf("list after add should succeed: %v", err)
	}
}

func TestRunAccountAdd_JSON(t *testing.T) {
	accountRunEnv(t)
	JSONOutput = true

	if err := accountRunAdd(t, "prod", "prod-id-1234567890abcdef", "tok-prod-abcdef123456"); err != nil {
		t.Fatalf("JSON add should succeed: %v", err)
	}
}

func TestRunAccountAdd_DryRun(t *testing.T) {
	accountRunEnv(t)
	DryRun = true

	if err := accountRunAdd(t, "prod", "prod-id-1234567890abcdef", "tok-prod-abcdef123456"); err != nil {
		t.Fatalf("dry-run add should succeed: %v", err)
	}
	if _, err := os.Stat(accountRunFilePath(t, "accounts.yaml")); !os.IsNotExist(err) {
		t.Fatalf("dry-run add must not write accounts.yaml (stat err: %v)", err)
	}
}

func TestRunAccountAdd_Validation(t *testing.T) {
	cases := []struct {
		name      string
		acctName  string
		id        string
		token     string
		wantError string
	}{
		{"invalid-name", "bad name!", "id-1234", "tok-1234", "invalid account name"},
		{"missing-id", "prod", "", "tok-1234", "account-id is required"},
		{"missing-token", "prod", "id-1234", "", "api-token is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			accountRunEnv(t)

			err := accountRunAdd(t, tc.acctName, tc.id, tc.token)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected error containing %q, got %v", tc.wantError, err)
			}
		})
	}
}

func TestRunAccountAdd_Duplicate(t *testing.T) {
	accountRunEnv(t)
	accountRunSeedAdded(t, "prod")

	err := accountRunAdd(t, "prod", "other-id-1234567890ab", "tok-other-abcdef123")
	if err == nil || !strings.Contains(err.Error(), `account "prod" already exists`) {
		t.Fatalf("expected duplicate-account error, got %v", err)
	}
}

func TestRunAccountSwitch_NotFound(t *testing.T) {
	accountRunEnv(t)

	err := runAccountSwitch(accountSwitchCmd, []string{"ghost"})
	if err == nil || !strings.Contains(err.Error(), `account "ghost" not found`) {
		t.Fatalf("expected not-found error, got %v", err)
	}
}

func TestRunAccountSwitch_Success(t *testing.T) {
	accountRunEnv(t)
	accountRunSeedAdded(t, "prod")
	accountRunSeedAdded(t, "staging")

	if err := runAccountSwitch(accountSwitchCmd, []string{"staging"}); err != nil {
		t.Fatalf("switch should succeed: %v", err)
	}
	data, err := os.ReadFile(accountRunFilePath(t, "active-account"))
	if err != nil {
		t.Fatalf("reading active-account: %v", err)
	}
	if strings.TrimSpace(string(data)) != "staging" {
		t.Fatalf("active-account = %q, want %q", data, "staging")
	}
	if err := runAccountCurrent(accountCurrentCmd, nil); err != nil {
		t.Fatalf("current after switch should succeed: %v", err)
	}
}

func TestRunAccountCurrent_NoneAndActive(t *testing.T) {
	cases := []struct {
		name string
		seed bool
		json bool
	}{
		{"none", false, false},
		{"none-json", false, true},
		{"active", true, false},
		{"active-json", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			accountRunEnv(t)
			if tc.seed {
				accountRunSeedAdded(t, "prod")
			}
			JSONOutput = tc.json

			if err := runAccountCurrent(accountCurrentCmd, nil); err != nil {
				t.Fatalf("current should succeed: %v", err)
			}
		})
	}
}

func TestRunAccountRemove_Guards(t *testing.T) {
	cases := []struct {
		name      string
		target    string
		force     bool
		wantError string
	}{
		{"not-found", "ghost", false, `account "ghost" not found`},
		{"active-without-force", "prod", false, "is the active account"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			accountRunEnv(t)
			accountRunSeedAdded(t, "prod")
			accountRemoveForce = tc.force

			err := runAccountRemove(accountRemoveCmd, []string{tc.target})
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("expected error containing %q, got %v", tc.wantError, err)
			}
		})
	}
}

func TestRunAccountRemove_ForceAndInactive(t *testing.T) {
	cases := []struct {
		name   string
		target string
		force  bool
	}{
		{"active-with-force", "prod", true},
		{"inactive", "staging", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			accountRunEnv(t)
			accountRunSeedAdded(t, "prod")
			accountRunSeedAdded(t, "staging")
			accountRemoveForce = tc.force

			if err := runAccountRemove(accountRemoveCmd, []string{tc.target}); err != nil {
				t.Fatalf("remove should succeed: %v", err)
			}
			data, err := os.ReadFile(accountRunFilePath(t, "accounts.yaml"))
			if err != nil {
				t.Fatalf("reading accounts.yaml: %v", err)
			}
			if strings.Contains(string(data), "name: "+tc.target) {
				t.Fatalf("account %q should be gone from accounts.yaml", tc.target)
			}
		})
	}
}

func TestRunAccountRemove_DryRun(t *testing.T) {
	accountRunEnv(t)
	accountRunSeedAdded(t, "prod")
	DryRun = true

	if err := runAccountRemove(accountRemoveCmd, []string{"prod"}); err != nil {
		t.Fatalf("dry-run remove should succeed: %v", err)
	}
	data, err := os.ReadFile(accountRunFilePath(t, "accounts.yaml"))
	if err != nil {
		t.Fatalf("reading accounts.yaml: %v", err)
	}
	if !strings.Contains(string(data), "name: prod") {
		t.Fatalf("dry-run remove must not delete the account:\n%s", data)
	}
}
