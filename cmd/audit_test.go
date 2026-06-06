package cmd

import (
	"strings"
	"testing"
)

// --- Command registration ---

func TestAuditCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "audit" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("auditCmd not registered on rootCmd")
	}
}

func TestAuditCmd_Metadata(t *testing.T) {
	if auditCmd.Use != "audit" {
		t.Errorf("auditCmd.Use = %q, want %q", auditCmd.Use, "audit")
	}
	if auditCmd.Short == "" {
		t.Error("auditCmd.Short is empty")
	}
	if auditCmd.Long == "" {
		t.Error("auditCmd.Long is empty")
	}
}

// --- Subcommand registration ---

func TestAuditCmd_HasSubcommands(t *testing.T) {
	expected := []string{"log", "search", "export", "clear"}
	subs := auditCmd.Commands()
	names := make(map[string]bool)
	for _, sub := range subs {
		names[sub.Name()] = true
	}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("subcommand %q not found on auditCmd", name)
		}
	}
}

func TestAuditCmd_SubcommandCount(t *testing.T) {
	if len(auditCmd.Commands()) != 4 {
		t.Errorf("auditCmd has %d subcommands, want 4", len(auditCmd.Commands()))
	}
}

// --- audit log ---

func TestAuditLogCmd_Metadata(t *testing.T) {
	if auditLogCmd.Use != "log" {
		t.Errorf("Use = %q, want %q", auditLogCmd.Use, "log")
	}
	if auditLogCmd.Short == "" {
		t.Error("Short is empty")
	}
	if auditLogCmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestAuditLogCmd_Flags(t *testing.T) {
	flags := []string{"limit", "since"}
	for _, name := range flags {
		f := auditLogCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("flag --%s not registered", name)
		}
	}
}

func TestAuditLogCmd_LimitDefault(t *testing.T) {
	f := auditLogCmd.Flags().Lookup("limit")
	if f == nil {
		t.Fatal("flag --limit not found")
	}
	if f.DefValue != "50" {
		t.Errorf("--limit default = %q, want %q", f.DefValue, "50")
	}
}

// --- audit search ---

func TestAuditSearchCmd_Metadata(t *testing.T) {
	if auditSearchCmd.Use != "search <query>" {
		t.Errorf("Use = %q, want %q", auditSearchCmd.Use, "search <query>")
	}
	if auditSearchCmd.Short == "" {
		t.Error("Short is empty")
	}
	if auditSearchCmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestAuditSearchCmd_RequiresArg(t *testing.T) {
	// cobra.ExactArgs(1) should be set
	if auditSearchCmd.Args == nil {
		t.Error("Args validator is nil, expected ExactArgs(1)")
	}
}

func TestAuditSearchCmd_TypeFlag(t *testing.T) {
	f := auditSearchCmd.Flags().Lookup("type")
	if f == nil {
		t.Fatal("flag --type not registered")
	}
	if f.DefValue != "" {
		t.Errorf("--type default = %q, want empty", f.DefValue)
	}
}

// --- audit export ---

func TestAuditExportCmd_Metadata(t *testing.T) {
	if auditExportCmd.Use != "export [file]" {
		t.Errorf("Use = %q, want %q", auditExportCmd.Use, "export [file]")
	}
	if auditExportCmd.Short == "" {
		t.Error("Short is empty")
	}
	if auditExportCmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

// --- audit clear ---

func TestAuditClearCmd_Metadata(t *testing.T) {
	if auditClearCmd.Use != "clear" {
		t.Errorf("Use = %q, want %q", auditClearCmd.Use, "clear")
	}
	if auditClearCmd.Short == "" {
		t.Error("Short is empty")
	}
	if auditClearCmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestAuditClearCmd_Flags(t *testing.T) {
	flags := []string{"force", "before"}
	for _, name := range flags {
		f := auditClearCmd.Flags().Lookup(name)
		if f == nil {
			t.Errorf("flag --%s not registered", name)
		}
	}
}

func TestAuditClearCmd_ForceDefault(t *testing.T) {
	f := auditClearCmd.Flags().Lookup("force")
	if f == nil {
		t.Fatal("flag --force not found")
	}
	if f.DefValue != "false" {
		t.Errorf("--force default = %q, want %q", f.DefValue, "false")
	}
}

// --- Long description content ---

func TestAuditCmd_LongDescriptionKeywords(t *testing.T) {
	keywords := []string{"audit", "mutation", "JSON", "log", "search", "export", "clear"}
	for _, kw := range keywords {
		if !strings.Contains(strings.ToLower(auditCmd.Long), strings.ToLower(kw)) {
			t.Errorf("auditCmd.Long should mention %q", kw)
		}
	}
}

func TestAuditLogCmd_LongDescriptionKeywords(t *testing.T) {
	keywords := []string{"audit", "log", "limit", "since"}
	for _, kw := range keywords {
		if !strings.Contains(strings.ToLower(auditLogCmd.Long), strings.ToLower(kw)) {
			t.Errorf("auditLogCmd.Long should mention %q", kw)
		}
	}
}

// --- parseDate helper ---

func TestParseDate_YYYYMMDD(t *testing.T) {
	d, err := parseDate("2026-06-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Year() != 2026 || d.Month() != 6 || d.Day() != 1 {
		t.Errorf("parsed = %v, want 2026-06-01", d)
	}
}

func TestParseDate_RFC3339(t *testing.T) {
	d, err := parseDate("2026-06-01T10:30:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Year() != 2026 || d.Hour() != 10 || d.Minute() != 30 {
		t.Errorf("parsed = %v, want 2026-06-01T10:30:00Z", d)
	}
}

func TestParseDate_Invalid(t *testing.T) {
	_, err := parseDate("not-a-date")
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

// --- truncate helper ---

func TestTruncate_Short(t *testing.T) {
	result := truncate("hello", 10)
	if result != "hello" {
		t.Errorf("truncate = %q, want %q", result, "hello")
	}
}

func TestTruncate_Long(t *testing.T) {
	result := truncate("this is a very long resource name", 15)
	if len(result) > 15 {
		t.Errorf("truncate result len = %d, want <= 15", len(result))
	}
	if !strings.HasSuffix(result, "...") {
		t.Errorf("truncated string should end with '...', got %q", result)
	}
}
