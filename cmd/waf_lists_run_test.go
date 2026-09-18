package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// wafListRunSnapshot snapshots the credential, output and WAF-list flag
// variables so tests cannot leak state into each other.
func wafListRunSnapshot(t *testing.T) {
	t.Helper()
	savedAccount, savedToken := AccountID, APIToken
	savedDry, savedJSON := DryRun, JSONOutput
	savedKind, savedDesc := wafListKind, wafListDescription
	savedName, savedForce := wafListName, wafListForce
	savedIP, savedASN := wafListItemIP, wafListItemASN
	savedComment, savedFile := wafListItemComment, wafListItemFile
	savedMode := wafManagedMode
	t.Cleanup(func() {
		AccountID, APIToken = savedAccount, savedToken
		DryRun, JSONOutput = savedDry, savedJSON
		wafListKind, wafListDescription = savedKind, savedDesc
		wafListName, wafListForce = savedName, savedForce
		wafListItemIP, wafListItemASN = savedIP, savedASN
		wafListItemComment, wafListItemFile = savedComment, savedFile
		wafManagedMode = savedMode
	})
}

// wafListRunResetVars clears every variable the WAF-list runners read so each
// subtest starts from a pristine state.
func wafListRunResetVars() {
	AccountID, APIToken = "", ""
	DryRun, JSONOutput = false, false
	wafListKind, wafListDescription = "", ""
	wafListName, wafListForce = "", false
	wafListItemIP, wafListItemASN, wafListItemComment, wafListItemFile = "", 0, "", ""
	wafManagedMode = ""
}

// TestGetWAFListService_CredentialValidation verifies the service factory
// rejects empty credentials with distinct errors and succeeds once both the
// account ID and API token are present.
func TestGetWAFListService_CredentialValidation(t *testing.T) {
	wafListRunSnapshot(t)
	cases := []struct {
		name        string
		accountID   string
		apiToken    string
		wantErr     string
		wantService bool
	}{
		{"missing account id", "", "token", "account ID is required", false},
		{"missing api token", "acct", "", "API token is required", false},
		{"missing both", "", "", "account ID is required", false},
		{"valid credentials", "acct", "token", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wafListRunResetVars()
			AccountID, APIToken = tc.accountID, tc.apiToken
			svc, err := getWAFListService()
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if svc == nil {
				t.Fatal("expected non-nil service")
			}
		})
	}
}

// TestRunWAFListLs_ServiceCreationFailsOffline verifies the list runner aborts
// with a wrapped service error before any network call when credentials are
// empty.
func TestRunWAFListLs_ServiceCreationFailsOffline(t *testing.T) {
	wafListRunSnapshot(t)
	wafListRunResetVars()

	err := runWAFListLs(wafListLsCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create WAF list service") {
		t.Fatalf("expected service creation error, got %v", err)
	}
}

// TestRunWAFListCreate_Validation verifies the create runner rejects a missing
// name argument and a missing --kind flag before building a service.
func TestRunWAFListCreate_Validation(t *testing.T) {
	wafListRunSnapshot(t)
	cases := []struct {
		name    string
		args    []string
		kind    string
		wantErr string
	}{
		{"no name", nil, "ip", "list name is required"},
		{"no kind", []string{"blocked-ips"}, "", "--kind is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wafListRunResetVars()
			wafListKind = tc.kind
			err := runWAFListCreate(wafListCreateCmd, tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRunWAFListCreate_DryRun verifies dry-run mode reports the would-be list
// payload without contacting the API, in both human and JSON output modes.
func TestRunWAFListCreate_DryRun(t *testing.T) {
	wafListRunSnapshot(t)
	cases := []struct {
		name string
		json bool
	}{
		{"human output", false},
		{"json output", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wafListRunResetVars()
			DryRun = true
			JSONOutput = tc.json
			wafListKind = "ip"

			r, restore := captureStdout(t)
			err := runWAFListCreate(wafListCreateCmd, []string{"blocked-ips"})
			restore()
			if err != nil {
				t.Fatalf("dry-run create returned error: %v", err)
			}
			out, readErr := io.ReadAll(r)
			if readErr != nil {
				t.Fatalf("reading captured stdout: %v", readErr)
			}
			if !strings.Contains(string(out), "blocked-ips") {
				t.Errorf("output missing list name, got: %s", string(out))
			}
		})
	}
}

// TestRunWAFListUpdate_Validation verifies the update runner requires a list
// ID and a --description flag, and rejects unsupported renames.
func TestRunWAFListUpdate_Validation(t *testing.T) {
	wafListRunSnapshot(t)
	cases := []struct {
		name        string
		args        []string
		listName    string
		description string
		wantErr     string
	}{
		{"no list id", nil, "", "new desc", "list ID is required"},
		{"rename rejected", []string{"list-1"}, "new-name", "new desc", "not supported by the Cloudflare API"},
		{"no description", []string{"list-1"}, "", "", "--description is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wafListRunResetVars()
			wafListName = tc.listName
			wafListDescription = tc.description
			err := runWAFListUpdate(wafListUpdateCmd, tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRunWAFListUpdate_DryRun verifies dry-run mode reports the pending
// description update without contacting the API.
func TestRunWAFListUpdate_DryRun(t *testing.T) {
	wafListRunSnapshot(t)
	wafListRunResetVars()
	DryRun = true
	wafListDescription = "updated description"
	AccountID, APIToken = "acct", "token"

	if err := runWAFListUpdate(wafListUpdateCmd, []string{"list-1"}); err != nil {
		t.Fatalf("dry-run update returned error: %v", err)
	}
}

// TestRunWAFListDelete_RequiresListID verifies the delete runner rejects a
// missing list ID before any prompt or service creation.
func TestRunWAFListDelete_RequiresListID(t *testing.T) {
	wafListRunSnapshot(t)
	wafListRunResetVars()

	err := runWAFListDelete(wafListDeleteCmd, nil)
	if err == nil || !strings.Contains(err.Error(), "list ID is required") {
		t.Fatalf("expected list-ID-required error, got %v", err)
	}
}

// TestRunWAFListDelete_DryRunSkipsConfirmation verifies dry-run mode skips the
// interactive confirmation prompt (which would otherwise block on stdin) and
// reports the deletion payload.
func TestRunWAFListDelete_DryRunSkipsConfirmation(t *testing.T) {
	wafListRunSnapshot(t)
	wafListRunResetVars()
	DryRun = true
	wafListForce = false
	AccountID, APIToken = "acct", "token"

	r, restore := captureStdout(t)
	err := runWAFListDelete(wafListDeleteCmd, []string{"list-1"})
	restore()
	if err != nil {
		t.Fatalf("dry-run delete returned error: %v", err)
	}
	out, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("reading captured stdout: %v", readErr)
	}
	if !strings.Contains(string(out), "list-1") {
		t.Errorf("output missing list id, got: %s", string(out))
	}
}

// TestRunWAFListItemLs_Validation verifies the item-list runner requires a
// list ID and aborts offline with a wrapped service error.
func TestRunWAFListItemLs_Validation(t *testing.T) {
	wafListRunSnapshot(t)
	t.Run("no list id", func(t *testing.T) {
		wafListRunResetVars()
		err := runWAFListItemLs(wafListItemLsCmd, nil)
		if err == nil || !strings.Contains(err.Error(), "list ID is required") {
			t.Fatalf("expected list-ID-required error, got %v", err)
		}
	})
	t.Run("service creation fails offline", func(t *testing.T) {
		wafListRunResetVars()
		err := runWAFListItemLs(wafListItemLsCmd, []string{"list-1"})
		if err == nil || !strings.Contains(err.Error(), "failed to create WAF list service") {
			t.Fatalf("expected service creation error, got %v", err)
		}
	})
}

// TestRunWAFListItemAdd_Validation verifies the item-add runner requires a
// list ID and at least one of --ip or --asn before doing anything else.
func TestRunWAFListItemAdd_Validation(t *testing.T) {
	wafListRunSnapshot(t)
	cases := []struct {
		name    string
		args    []string
		ip      string
		asn     uint32
		wantErr string
	}{
		{"no list id", nil, "1.2.3.4", 0, "list ID is required"},
		{"no items", []string{"list-1"}, "", 0, "at least one of --ip or --asn is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wafListRunResetVars()
			wafListItemIP = tc.ip
			wafListItemASN = tc.asn
			err := runWAFListItemAdd(wafListItemAddCmd, tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRunWAFListItemAdd_DryRun verifies dry-run mode reports the pending items
// (IP, ASN, or both) without contacting the API.
func TestRunWAFListItemAdd_DryRun(t *testing.T) {
	wafListRunSnapshot(t)
	cases := []struct {
		name string
		ip   string
		asn  uint32
	}{
		{"ip only", "192.168.0.0/24", 0},
		{"asn only", "", 13335},
		{"both ip and asn", "1.2.3.4", 13335},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wafListRunResetVars()
			DryRun = true
			wafListItemIP = tc.ip
			wafListItemASN = tc.asn
			wafListItemComment = "test"
			AccountID, APIToken = "acct", "token"

			if err := runWAFListItemAdd(wafListItemAddCmd, []string{"list-1"}); err != nil {
				t.Fatalf("dry-run item add returned error: %v", err)
			}
		})
	}
}

// TestRunWAFListItemReplace_Validation verifies the item-replace runner checks
// the list ID, the --items-file flag, file readability, JSON shape and a
// non-empty item array before building a service.
func TestRunWAFListItemReplace_Validation(t *testing.T) {
	wafListRunSnapshot(t)

	dir := t.TempDir()
	validFile := filepath.Join(dir, "items.json")
	if err := os.WriteFile(validFile, []byte(`[{"ip":"1.2.3.4"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	badJSONFile := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(badJSONFile, []byte(`{"not":"an array"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	emptyArrayFile := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(emptyArrayFile, []byte(`[]`), 0o600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name      string
		args      []string
		itemsFile string
		wantErr   string
	}{
		{"no list id", nil, validFile, "list ID is required"},
		{"no items file", []string{"list-1"}, "", "--items-file is required"},
		{"missing file", []string{"list-1"}, filepath.Join(dir, "nope.json"), "failed to read items file"},
		{"invalid json shape", []string{"list-1"}, badJSONFile, "failed to parse items file"},
		{"empty item array", []string{"list-1"}, emptyArrayFile, "items file contains no items"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wafListRunResetVars()
			wafListItemFile = tc.itemsFile
			err := runWAFListItemReplace(wafListItemReplaceCmd, tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRunWAFListItemReplace_DryRun verifies dry-run mode reports the pending
// replacement count for a well-formed items file without contacting the API.
func TestRunWAFListItemReplace_DryRun(t *testing.T) {
	wafListRunSnapshot(t)
	wafListRunResetVars()
	DryRun = true
	AccountID, APIToken = "acct", "token"

	file := filepath.Join(t.TempDir(), "items.json")
	if err := os.WriteFile(file, []byte(`[{"ip":"1.2.3.4","comment":"scanner"},{"asn":13335}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	wafListItemFile = file

	if err := runWAFListItemReplace(wafListItemReplaceCmd, []string{"list-1"}); err != nil {
		t.Fatalf("dry-run item replace returned error: %v", err)
	}
}

// TestRunWAFManagedRulesetUpdate_Validation verifies the managed-ruleset
// runner requires a phase argument and a --mode of exactly "on" or "off".
func TestRunWAFManagedRulesetUpdate_Validation(t *testing.T) {
	wafListRunSnapshot(t)
	cases := []struct {
		name    string
		args    []string
		mode    string
		wantErr string
	}{
		{"no phase", nil, "on", "ruleset phase is required"},
		{"empty mode", []string{"http_request_firewall_managed"}, "", "--mode is required"},
		{"invalid mode", []string{"http_request_firewall_managed"}, "maybe", "--mode is required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wafListRunResetVars()
			wafManagedMode = tc.mode
			err := runWAFManagedRulesetUpdate(wafManagedRulesetUpdateCmd, tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected %q error, got %v", tc.wantErr, err)
			}
		})
	}
}

// TestRunWAFManagedRulesetUpdate_DryRun verifies dry-run mode reports the
// pending phase/mode change without contacting the API.
func TestRunWAFManagedRulesetUpdate_DryRun(t *testing.T) {
	wafListRunSnapshot(t)
	cases := []struct {
		name string
		mode string
	}{
		{"mode on", "on"},
		{"mode off", "off"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wafListRunResetVars()
			DryRun = true
			wafManagedMode = tc.mode
			AccountID, APIToken = "acct", "token"

			if err := runWAFManagedRulesetUpdate(wafManagedRulesetUpdateCmd, []string{"http_request_firewall_managed"}); err != nil {
				t.Fatalf("dry-run managed ruleset update returned error: %v", err)
			}
		})
	}
}
