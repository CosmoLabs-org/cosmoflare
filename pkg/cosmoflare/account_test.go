package cosmoflare

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// helper: create a temp dir and return a fresh AccountService rooted there.
func newTestAccountService(t *testing.T) *AccountService {
	t.Helper()
	dir := t.TempDir()
	svc, err := NewAccountService(dir)
	if err != nil {
		t.Fatalf("NewAccountService: %v", err)
	}
	// stub out real HTTP calls
	svc.verifyFunc = func(_ context.Context, _, _ string) error { return nil }
	return svc
}

// --- NewAccountService ---

func TestNewAccountService_DefaultDir(t *testing.T) {
	svc, err := NewAccountService("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.ConfigDir() == "" {
		t.Error("ConfigDir() should not be empty")
	}
}

func TestNewAccountService_CustomDir(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewAccountService(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc.ConfigDir() != dir {
		t.Errorf("ConfigDir() = %q, want %q", svc.ConfigDir(), dir)
	}
}

// --- Add ---

func TestAccountService_Add_Success(t *testing.T) {
	svc := newTestAccountService(t)

	acct, err := svc.Add("prod", "abc123", "tok_xxx", "user@example.com")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if acct.Name != "prod" {
		t.Errorf("Name = %q, want %q", acct.Name, "prod")
	}
	if acct.AccountID != "abc123" {
		t.Errorf("AccountID = %q, want %q", acct.AccountID, "abc123")
	}
	if acct.APIToken != "tok_xxx" {
		t.Errorf("APIToken = %q, want %q", acct.APIToken, "tok_xxx")
	}
	if acct.Email != "user@example.com" {
		t.Errorf("Email = %q, want %q", acct.Email, "user@example.com")
	}
	if acct.CreatedAt == "" {
		t.Error("CreatedAt should be set")
	}
}

func TestAccountService_Add_FirstAccountBecomesActive(t *testing.T) {
	svc := newTestAccountService(t)

	_, err := svc.Add("first", "id1", "tok1", "")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	cur, err := svc.Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if cur == nil || cur.Name != "first" {
		t.Errorf("first account should be active, got %v", cur)
	}
}

func TestAccountService_Add_DuplicateName(t *testing.T) {
	svc := newTestAccountService(t)

	_, _ = svc.Add("dup", "id1", "tok1", "")
	_, err := svc.Add("dup", "id2", "tok2", "")
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestAccountService_Add_EmptyName(t *testing.T) {
	svc := newTestAccountService(t)
	_, err := svc.Add("", "id", "tok", "")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestAccountService_Add_InvalidName(t *testing.T) {
	svc := newTestAccountService(t)
	_, err := svc.Add("bad name!", "id", "tok", "")
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestAccountService_Add_MissingAccountID(t *testing.T) {
	svc := newTestAccountService(t)
	_, err := svc.Add("ok", "", "tok", "")
	if err == nil {
		t.Fatal("expected error for missing account-id")
	}
}

func TestAccountService_Add_MissingAPIToken(t *testing.T) {
	svc := newTestAccountService(t)
	_, err := svc.Add("ok", "id", "", "")
	if err == nil {
		t.Fatal("expected error for missing api-token")
	}
}

// --- List ---

func TestAccountService_List_Empty(t *testing.T) {
	svc := newTestAccountService(t)

	list, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(list))
	}
}

func TestAccountService_List_Multiple(t *testing.T) {
	svc := newTestAccountService(t)
	_, _ = svc.Add("a", "id1", "tok1", "")
	_, _ = svc.Add("b", "id2", "tok2", "")

	list, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(list))
	}
}

// --- Switch ---

func TestAccountService_Switch_Success(t *testing.T) {
	svc := newTestAccountService(t)
	_, _ = svc.Add("a", "id1", "tok1", "")
	_, _ = svc.Add("b", "id2", "tok2", "")

	acct, err := svc.Switch("b")
	if err != nil {
		t.Fatalf("Switch: %v", err)
	}
	if acct.Name != "b" {
		t.Errorf("switched to %q, want %q", acct.Name, "b")
	}

	cur, _ := svc.Current()
	if cur == nil || cur.Name != "b" {
		t.Errorf("Current should be 'b', got %v", cur)
	}
}

func TestAccountService_Switch_NotFound(t *testing.T) {
	svc := newTestAccountService(t)
	_, err := svc.Switch("nope")
	if err == nil {
		t.Fatal("expected error for missing account")
	}
}

func TestAccountService_Switch_EmptyName(t *testing.T) {
	svc := newTestAccountService(t)
	_, err := svc.Switch("")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

// --- Remove ---

func TestAccountService_Remove_Success(t *testing.T) {
	svc := newTestAccountService(t)
	_, _ = svc.Add("a", "id1", "tok1", "")
	_, _ = svc.Add("b", "id2", "tok2", "")
	_ = svc.writeActiveAccount("b")

	err := svc.Remove("a", false)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	list, _ := svc.List()
	if len(list) != 1 {
		t.Errorf("expected 1 account after remove, got %d", len(list))
	}
}

func TestAccountService_Remove_ActiveWithoutForce(t *testing.T) {
	svc := newTestAccountService(t)
	_, _ = svc.Add("active", "id1", "tok1", "")

	err := svc.Remove("active", false)
	if err == nil {
		t.Fatal("expected error removing active account without --force")
	}
}

func TestAccountService_Remove_ActiveWithForce(t *testing.T) {
	svc := newTestAccountService(t)
	_, _ = svc.Add("active", "id1", "tok1", "")

	err := svc.Remove("active", true)
	if err != nil {
		t.Fatalf("Remove with force: %v", err)
	}

	list, _ := svc.List()
	if len(list) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(list))
	}
}

func TestAccountService_Remove_NotFound(t *testing.T) {
	svc := newTestAccountService(t)
	err := svc.Remove("nope", false)
	if err == nil {
		t.Fatal("expected error for missing account")
	}
}

func TestAccountService_Remove_EmptyName(t *testing.T) {
	svc := newTestAccountService(t)
	err := svc.Remove("", false)
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

// --- Current ---

func TestAccountService_Current_None(t *testing.T) {
	svc := newTestAccountService(t)

	cur, err := svc.Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if cur != nil {
		t.Errorf("expected nil when no active account, got %v", cur)
	}
}

func TestAccountService_Current_Stale(t *testing.T) {
	svc := newTestAccountService(t)
	// Write pointer to a non-existent account
	_ = svc.writeActiveAccount("ghost")

	_, err := svc.Current()
	if err == nil {
		t.Fatal("expected error for stale active-account pointer")
	}
}

// --- Verify ---

func TestAccountService_Verify_Valid(t *testing.T) {
	svc := newTestAccountService(t)
	_, _ = svc.Add("ok", "id1", "tok1", "")

	res, err := svc.Verify("ok")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !res.Valid {
		t.Errorf("expected Valid=true, got false: %s", res.Message)
	}
}

func TestAccountService_Verify_Invalid(t *testing.T) {
	svc := newTestAccountService(t)
	svc.verifyFunc = func(_ context.Context, _, _ string) error {
		return fmt.Errorf("401 Unauthorized")
	}
	_, _ = svc.Add("bad", "id1", "tok1", "")

	res, err := svc.Verify("bad")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if res.Valid {
		t.Error("expected Valid=false for bad credentials")
	}
}

func TestAccountService_Verify_NotFound(t *testing.T) {
	svc := newTestAccountService(t)
	_, err := svc.Verify("nope")
	if err == nil {
		t.Fatal("expected error for missing account")
	}
}

func TestAccountService_Verify_EmptyName(t *testing.T) {
	svc := newTestAccountService(t)
	_, err := svc.Verify("")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

// --- File permissions ---

func TestAccountService_FilePermissions(t *testing.T) {
	svc := newTestAccountService(t)
	_, _ = svc.Add("test", "id1", "tok1", "")

	info, err := os.Stat(svc.accountsPath())
	if err != nil {
		t.Fatalf("stat accounts.yaml: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("accounts.yaml permissions = %o, want 0600", perm)
	}
}

// --- isValidAccountName ---

func TestIsValidAccountName(t *testing.T) {
	valid := []string{"prod", "staging-1", "my_account", "ABC123", "a-b_c"}
	for _, n := range valid {
		if !isValidAccountName(n) {
			t.Errorf("expected %q to be valid", n)
		}
	}

	invalid := []string{"", "bad name", "no@at", "has.dot", "sp ace", "semi;colon"}
	for _, n := range invalid {
		if isValidAccountName(n) {
			t.Errorf("expected %q to be invalid", n)
		}
	}
}

// --- Persistence round-trip ---

func TestAccountService_PersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()

	// Write with one service instance
	svc1, _ := NewAccountService(dir)
	svc1.verifyFunc = func(_ context.Context, _, _ string) error { return nil }
	_, _ = svc1.Add("x", "id-x", "tok-x", "x@example.com")
	_ = svc1.writeActiveAccount("x")

	// Read with a fresh service instance
	svc2, _ := NewAccountService(dir)
	svc2.verifyFunc = func(_ context.Context, _, _ string) error { return nil }

	list, err := svc2.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].Name != "x" {
		t.Errorf("expected 1 account named 'x', got %v", list)
	}

	cur, err := svc2.Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if cur == nil || cur.Name != "x" {
		t.Errorf("expected active='x', got %v", cur)
	}
}

// --- activeAccountPath / accountsPath ---

func TestAccountService_Paths(t *testing.T) {
	svc := newTestAccountService(t)
	dir := svc.ConfigDir()

	if svc.accountsPath() != filepath.Join(dir, "accounts.yaml") {
		t.Errorf("accountsPath = %q", svc.accountsPath())
	}
	if svc.activeAccountPath() != filepath.Join(dir, "active-account") {
		t.Errorf("activeAccountPath = %q", svc.activeAccountPath())
	}
}
