package cmd

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestAuthPermissionsCommandRegistered(t *testing.T) {
	found := false
	for _, c := range authCmd.Commands() {
		if c.Name() == "permissions" {
			found = true
		}
	}
	if !found {
		t.Fatal("permissions command not registered on auth")
	}

	listFound := false
	for _, c := range authPermissionsCmd.Commands() {
		if c.Name() == "list" {
			listFound = true
		}
	}
	if !listFound {
		t.Fatal("list command not registered on auth permissions")
	}
}

func resetAuthPermissionsFlags() {
	authPermissionsScope = ""
	authPermissionsGroup = ""
}

func TestRunAuthPermissionsList_All(t *testing.T) {
	resetAuthPermissionsFlags()
	defer resetAuthPermissionsFlags()

	r, restore := captureStdout(t)
	err := runAuthPermissionsList(authPermissionsListCmd, nil)
	restore()
	if err != nil {
		t.Fatalf("runAuthPermissionsList() error = %v", err)
	}

	out, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("read captured stdout: %v", readErr)
	}
	text := string(out)
	if !strings.Contains(text, "ID") || !strings.Contains(text, "SCOPE") {
		t.Errorf("table output missing headers, got: %s", text)
	}
	if !strings.Contains(text, "account.") {
		t.Errorf("table output missing account families, got: %s", text)
	}
	if !strings.Contains(text, "zone.") {
		t.Errorf("table output missing zone families, got: %s", text)
	}
	if !strings.Contains(text, "user.") {
		t.Errorf("table output missing user families, got: %s", text)
	}
}

func TestRunAuthPermissionsList_ScopeFilter(t *testing.T) {
	resetAuthPermissionsFlags()
	defer resetAuthPermissionsFlags()
	authPermissionsScope = "account"

	r, restore := captureStdout(t)
	err := runAuthPermissionsList(authPermissionsListCmd, nil)
	restore()
	if err != nil {
		t.Fatalf("runAuthPermissionsList() error = %v", err)
	}

	out, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("read captured stdout: %v", readErr)
	}
	text := string(out)
	if strings.Contains(text, "zone.") || strings.Contains(text, "user.") {
		t.Errorf("--scope account leaked non-account families, got: %s", text)
	}
	if !strings.Contains(text, "account.") {
		t.Errorf("--scope account missing account families, got: %s", text)
	}
}

func TestRunAuthPermissionsList_InvalidScope(t *testing.T) {
	resetAuthPermissionsFlags()
	defer resetAuthPermissionsFlags()
	authPermissionsScope = "bogus"

	err := runAuthPermissionsList(authPermissionsListCmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid --scope, got nil")
	}
}

func TestRunAuthPermissionsList_GroupLookup(t *testing.T) {
	resetAuthPermissionsFlags()
	defer resetAuthPermissionsFlags()
	authPermissionsGroup = "deploy"

	r, restore := captureStdout(t)
	err := runAuthPermissionsList(authPermissionsListCmd, nil)
	restore()
	if err != nil {
		t.Fatalf("runAuthPermissionsList() error = %v", err)
	}

	out, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("read captured stdout: %v", readErr)
	}
	text := string(out)
	if !strings.Contains(text, "deploy") {
		t.Errorf("group output missing command group name, got: %s", text)
	}
}

func TestRunAuthPermissionsList_UnknownGroup(t *testing.T) {
	resetAuthPermissionsFlags()
	defer resetAuthPermissionsFlags()
	authPermissionsGroup = "not-a-real-group"

	err := runAuthPermissionsList(authPermissionsListCmd, nil)
	if err == nil {
		t.Fatal("expected error for unknown --group, got nil")
	}
}

func TestRunAuthPermissionsList_JSONShape(t *testing.T) {
	resetAuthPermissionsFlags()
	defer resetAuthPermissionsFlags()

	savedJSON := JSONOutput
	defer func() { JSONOutput = savedJSON }()
	JSONOutput = true

	r, restore := captureStdout(t)
	err := runAuthPermissionsList(authPermissionsListCmd, nil)
	restore()
	if err != nil {
		t.Fatalf("runAuthPermissionsList() error = %v", err)
	}

	out, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("read captured stdout: %v", readErr)
	}

	var resp OutputResponse
	if jsonErr := json.Unmarshal(out, &resp); jsonErr != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", jsonErr, out)
	}
	if !resp.Success {
		t.Errorf("resp.Success = false, want true")
	}
	if resp.Data == nil {
		t.Error("resp.Data is nil")
	}
}
