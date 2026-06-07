package cmd

import (
	"strings"
	"testing"

	"github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
	"github.com/spf13/cobra"
)

// --- Command registration ---

func TestCostCmd_RegisteredOnRoot(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "cost" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("costCmd not registered on rootCmd")
	}
}

func TestCostCmd_Metadata(t *testing.T) {
	if costCmd.Use != "cost" {
		t.Errorf("costCmd.Use = %q, want %q", costCmd.Use, "cost")
	}
	if costCmd.Short == "" {
		t.Error("costCmd.Short is empty")
	}
}

func TestCostCmd_ShortDescriptionContent(t *testing.T) {
	short := strings.ToLower(costCmd.Short)
	if !strings.Contains(short, "cost") && !strings.Contains(short, "estimate") {
		t.Errorf("costCmd.Short %q does not describe cost estimation", costCmd.Short)
	}
}

func TestCostCmd_LongDescription(t *testing.T) {
	if costCmd.Long == "" {
		t.Error("costCmd.Long is empty")
	}
}

func TestCostCmd_LongDescriptionMentionsServices(t *testing.T) {
	for _, svc := range []string{"R2", "Workers", "KV"} {
		if !strings.Contains(costCmd.Long, svc) {
			t.Errorf("costCmd.Long does not mention service %q", svc)
		}
	}
}

func TestCostCmd_LongDescriptionMentionsSubcommands(t *testing.T) {
	for _, sub := range []string{"r2", "workers", "kv", "detail"} {
		if !strings.Contains(costCmd.Long, sub) {
			t.Errorf("costCmd.Long does not mention subcommand %q", sub)
		}
	}
}

func TestCostCmd_LongDescriptionHasExamples(t *testing.T) {
	if !strings.Contains(costCmd.Long, "cosmoflare cost") {
		t.Error("costCmd.Long should contain usage examples")
	}
}

func TestCostCmd_HasParentCommand(t *testing.T) {
	if costCmd.Parent() == nil {
		t.Error("costCmd has no parent — must be a subcommand of rootCmd")
	}
	if costCmd.Parent().Name() != rootCmd.Name() {
		t.Errorf("costCmd parent = %q, want %q", costCmd.Parent().Name(), rootCmd.Name())
	}
}

func TestCostCmd_NotHidden(t *testing.T) {
	if costCmd.Hidden {
		t.Error("costCmd.Hidden is true — cost should be visible in help")
	}
}

// --- Subcommand registration ---

func TestCostCmd_Subcommands(t *testing.T) {
	expected := []string{"r2", "workers", "kv", "detail"}
	for _, name := range expected {
		found := false
		for _, sub := range costCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("cost subcommand %q not registered", name)
		}
	}
}

func TestCostCmd_ExactSubcommandCount(t *testing.T) {
	// r2, workers, kv, detail — exactly 4 subcommands
	got := len(costCmd.Commands())
	if got != 4 {
		t.Errorf("costCmd has %d subcommands, want 4", got)
	}
}

func TestCostCmd_SubcommandMetadata(t *testing.T) {
	cases := []struct {
		cmd   *cobra.Command
		use   string
		short string
	}{
		{costR2Cmd, "r2", ""},
		{costWorkersCmd, "workers", ""},
		{costKVCmd, "kv", ""},
		{costDetailCmd, "detail", ""},
	}
	for _, tc := range cases {
		if tc.cmd.Use != tc.use {
			t.Errorf("cmd.Use = %q, want %q", tc.cmd.Use, tc.use)
		}
		if tc.cmd.Short == "" {
			t.Errorf("subcommand %q has empty Short description", tc.use)
		}
		if tc.cmd.Long == "" {
			t.Errorf("subcommand %q has empty Long description", tc.use)
		}
	}
}

func TestCostCmd_SubcommandLongDescriptionsHaveExamples(t *testing.T) {
	cmds := []*cobra.Command{costR2Cmd, costWorkersCmd, costKVCmd, costDetailCmd}
	for _, c := range cmds {
		if !strings.Contains(c.Long, "cosmoflare cost") {
			t.Errorf("subcommand %q Long description missing usage examples", c.Use)
		}
	}
}

// --- RunE handlers wired ---

func TestCostCmd_AllRunE(t *testing.T) {
	cmds := []*cobra.Command{
		costCmd,
		costR2Cmd,
		costWorkersCmd,
		costKVCmd,
		costDetailCmd,
	}
	for _, c := range cmds {
		if c.RunE == nil {
			t.Errorf("%q has nil RunE", c.Use)
		}
	}
}

func TestCostCmd_NoRunField(t *testing.T) {
	// cost uses RunE (not Run)
	cmds := []*cobra.Command{costCmd, costR2Cmd, costWorkersCmd, costKVCmd, costDetailCmd}
	for _, c := range cmds {
		if c.Run != nil {
			t.Errorf("%q has non-nil Run — should use RunE for error propagation", c.Use)
		}
	}
}

// --- Flag registration ---

func TestCostCmd_PeriodFlag(t *testing.T) {
	f := costCmd.PersistentFlags().Lookup("period")
	if f == nil {
		t.Fatal("--period flag not registered on costCmd")
	}
	if f.DefValue != "30d" {
		t.Errorf("--period default = %q, want %q", f.DefValue, "30d")
	}
}

func TestCostCmd_PeriodFlagIsString(t *testing.T) {
	f := costCmd.PersistentFlags().Lookup("period")
	if f == nil {
		t.Fatal("--period flag not found")
	}
	if f.Value.Type() != "string" {
		t.Errorf("--period type = %q, want string", f.Value.Type())
	}
}

func TestCostCmd_PeriodFlagHasUsageText(t *testing.T) {
	f := costCmd.PersistentFlags().Lookup("period")
	if f == nil {
		t.Fatal("--period flag not found")
	}
	if f.Usage == "" {
		t.Error("--period flag has empty usage text")
	}
}

func TestCostCmd_FormatFlag(t *testing.T) {
	f := costCmd.PersistentFlags().Lookup("format")
	if f == nil {
		t.Fatal("--format flag not registered on costCmd")
	}
	if f.DefValue != "table" {
		t.Errorf("--format default = %q, want %q", f.DefValue, "table")
	}
}

func TestCostCmd_FormatFlagIsString(t *testing.T) {
	f := costCmd.PersistentFlags().Lookup("format")
	if f == nil {
		t.Fatal("--format flag not found")
	}
	if f.Value.Type() != "string" {
		t.Errorf("--format type = %q, want string", f.Value.Type())
	}
}

func TestCostCmd_FormatFlagHasUsageText(t *testing.T) {
	f := costCmd.PersistentFlags().Lookup("format")
	if f == nil {
		t.Fatal("--format flag not found")
	}
	if f.Usage == "" {
		t.Error("--format flag has empty usage text")
	}
}

// --- Flag inheritance ---

func TestCostCmd_SubcommandsInheritFlags(t *testing.T) {
	subs := []*cobra.Command{costR2Cmd, costWorkersCmd, costKVCmd, costDetailCmd}
	for _, sub := range subs {
		pf := sub.InheritedFlags()
		if pf.Lookup("period") == nil {
			t.Errorf("subcommand %q does not inherit --period flag", sub.Name())
		}
		if pf.Lookup("format") == nil {
			t.Errorf("subcommand %q does not inherit --format flag", sub.Name())
		}
	}
}

// --- Period validation ---

func TestCostCmd_ValidatePeriod(t *testing.T) {
	cases := []struct {
		input string
		valid bool
	}{
		{"7d", true},
		{"30d", true},
		{"90d", true},
		{"1d", false},
		{"365d", false},
		{"", false},
		{"invalid", false},
		{"7D", false},  // case-sensitive
		{"30", false},  // missing 'd'
		{"d30", false}, // wrong order
	}
	for _, tc := range cases {
		err := validatePeriod(tc.input)
		if tc.valid && err != nil {
			t.Errorf("validatePeriod(%q) returned error %v, want nil", tc.input, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("validatePeriod(%q) returned nil, want error", tc.input)
		}
	}
}

func TestCostCmd_ValidatePeriod_ErrorMessageContainsInput(t *testing.T) {
	err := validatePeriod("bogus")
	if err == nil {
		t.Fatal("expected error for invalid period")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error message %q does not mention invalid input", err.Error())
	}
}

func TestCostCmd_ValidatePeriod_ErrorMessageListsValid(t *testing.T) {
	err := validatePeriod("bogus")
	if err == nil {
		t.Fatal("expected error for invalid period")
	}
	msg := err.Error()
	for _, valid := range []string{"7d", "30d", "90d"} {
		if !strings.Contains(msg, valid) {
			t.Errorf("error message %q does not list valid option %q", msg, valid)
		}
	}
}

// --- Format validation ---

func TestCostCmd_ValidateFormat(t *testing.T) {
	cases := []struct {
		input string
		valid bool
	}{
		{"table", true},
		{"json", true},
		{"csv", true},
		{"xml", false},
		{"", false},
		{"yaml", false},
		{"TABLE", false}, // case-sensitive
		{"JSON", false},
		{"CSV", false},
	}
	for _, tc := range cases {
		err := validateCostFormat(tc.input)
		if tc.valid && err != nil {
			t.Errorf("validateCostFormat(%q) returned error %v, want nil", tc.input, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("validateCostFormat(%q) returned nil, want error", tc.input)
		}
	}
}

func TestCostCmd_ValidateFormat_ErrorMessageContainsInput(t *testing.T) {
	err := validateCostFormat("bogus")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error message %q does not mention invalid input", err.Error())
	}
}

func TestCostCmd_ValidateFormat_ErrorMessageListsValid(t *testing.T) {
	err := validateCostFormat("bogus")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	msg := err.Error()
	for _, valid := range []string{"table", "json", "csv"} {
		if !strings.Contains(msg, valid) {
			t.Errorf("error message %q does not list valid option %q", msg, valid)
		}
	}
}

// ---------------------------------------------------------------------------
// CostService estimation logic (no API credentials needed)
// ---------------------------------------------------------------------------

func newTestCostService(t *testing.T) *cosmoflare.CostService {
	t.Helper()
	svc, err := cosmoflare.NewCostService("test-account", "test-token")
	if err != nil {
		t.Fatalf("NewCostService: %v", err)
	}
	return svc
}

// --- R2 cost estimation ---

func TestCostService_EstimateR2Cost_ZeroUsage(t *testing.T) {
	svc := newTestCostService(t)
	est := svc.EstimateR2Cost(cosmoflare.R2Usage{})
	if est.TotalCost != 0 {
		t.Errorf("EstimateR2Cost zero usage: TotalCost = %.6f, want 0", est.TotalCost)
	}
	if est.StorageCost != 0 {
		t.Errorf("StorageCost = %.6f, want 0", est.StorageCost)
	}
	if est.ClassACost != 0 {
		t.Errorf("ClassACost = %.6f, want 0", est.ClassACost)
	}
	if est.ClassBCost != 0 {
		t.Errorf("ClassBCost = %.6f, want 0", est.ClassBCost)
	}
}

func TestCostService_EstimateR2Cost_StorageOnly(t *testing.T) {
	svc := newTestCostService(t)
	usage := cosmoflare.R2Usage{StorageGB: 100}
	est := svc.EstimateR2Cost(usage)
	// 100 GB * $0.015/GB = $1.50
	want := 1.50
	if est.StorageCost != want {
		t.Errorf("StorageCost = %.4f, want %.4f", est.StorageCost, want)
	}
	if est.TotalCost != want {
		t.Errorf("TotalCost = %.4f, want %.4f (storage only)", est.TotalCost, want)
	}
}

func TestCostService_EstimateR2Cost_ClassAOpsOnly(t *testing.T) {
	svc := newTestCostService(t)
	usage := cosmoflare.R2Usage{ClassAOps: 1_000_000}
	est := svc.EstimateR2Cost(usage)
	// 1M Class A ops * $4.50/M = $4.50
	want := 4.50
	if est.ClassACost != want {
		t.Errorf("ClassACost = %.4f, want %.4f", est.ClassACost, want)
	}
}

func TestCostService_EstimateR2Cost_ClassBOpsOnly(t *testing.T) {
	svc := newTestCostService(t)
	usage := cosmoflare.R2Usage{ClassBOps: 1_000_000}
	est := svc.EstimateR2Cost(usage)
	// 1M Class B ops * $0.36/M = $0.36
	want := 0.36
	if est.ClassBCost != want {
		t.Errorf("ClassBCost = %.4f, want %.4f", est.ClassBCost, want)
	}
}

func TestCostService_EstimateR2Cost_TotalIsSum(t *testing.T) {
	svc := newTestCostService(t)
	usage := cosmoflare.R2Usage{StorageGB: 50, ClassAOps: 500_000, ClassBOps: 2_000_000}
	est := svc.EstimateR2Cost(usage)
	wantTotal := est.StorageCost + est.ClassACost + est.ClassBCost
	if est.TotalCost != wantTotal {
		t.Errorf("TotalCost = %.6f, want %.6f (sum of components)", est.TotalCost, wantTotal)
	}
}

func TestCostService_EstimateR2Cost_UsagePreserved(t *testing.T) {
	svc := newTestCostService(t)
	usage := cosmoflare.R2Usage{StorageGB: 12.5, ClassAOps: 300, ClassBOps: 600}
	est := svc.EstimateR2Cost(usage)
	if est.Usage.StorageGB != usage.StorageGB {
		t.Errorf("Usage.StorageGB = %.2f, want %.2f", est.Usage.StorageGB, usage.StorageGB)
	}
	if est.Usage.ClassAOps != usage.ClassAOps {
		t.Errorf("Usage.ClassAOps = %d, want %d", est.Usage.ClassAOps, usage.ClassAOps)
	}
	if est.Usage.ClassBOps != usage.ClassBOps {
		t.Errorf("Usage.ClassBOps = %d, want %d", est.Usage.ClassBOps, usage.ClassBOps)
	}
}

// --- Workers cost estimation ---

func TestCostService_EstimateWorkersCost_ZeroUsage(t *testing.T) {
	svc := newTestCostService(t)
	est := svc.EstimateWorkersCost(cosmoflare.WorkersUsage{})
	if est.TotalCost != 0 {
		t.Errorf("TotalCost = %.6f, want 0", est.TotalCost)
	}
	if est.BillableRequests != 0 {
		t.Errorf("BillableRequests = %d, want 0", est.BillableRequests)
	}
}

func TestCostService_EstimateWorkersCost_BelowFreeTier(t *testing.T) {
	svc := newTestCostService(t)
	// Free tier is 100,000 requests — anything at or below costs $0
	usage := cosmoflare.WorkersUsage{Requests: 50_000}
	est := svc.EstimateWorkersCost(usage)
	if est.TotalCost != 0 {
		t.Errorf("TotalCost = %.6f, want 0 (below free tier)", est.TotalCost)
	}
	if est.BillableRequests != 0 {
		t.Errorf("BillableRequests = %d, want 0 (below free tier)", est.BillableRequests)
	}
}

func TestCostService_EstimateWorkersCost_ExactlyFreeTier(t *testing.T) {
	svc := newTestCostService(t)
	usage := cosmoflare.WorkersUsage{Requests: int64(cosmoflare.WorkersFreeRequests)}
	est := svc.EstimateWorkersCost(usage)
	if est.TotalCost != 0 {
		t.Errorf("TotalCost = %.6f, want 0 (exactly at free tier)", est.TotalCost)
	}
	if est.BillableRequests != 0 {
		t.Errorf("BillableRequests = %d, want 0", est.BillableRequests)
	}
}

func TestCostService_EstimateWorkersCost_AboveFreeTier(t *testing.T) {
	svc := newTestCostService(t)
	// 1.1M requests → 1M billable → $0.50
	usage := cosmoflare.WorkersUsage{Requests: 1_100_000}
	est := svc.EstimateWorkersCost(usage)
	wantBillable := int64(1_100_000 - cosmoflare.WorkersFreeRequests)
	if est.BillableRequests != wantBillable {
		t.Errorf("BillableRequests = %d, want %d", est.BillableRequests, wantBillable)
	}
	if est.TotalCost <= 0 {
		t.Errorf("TotalCost = %.6f, want > 0 for usage above free tier", est.TotalCost)
	}
}

func TestCostService_EstimateWorkersCost_OneMillion(t *testing.T) {
	svc := newTestCostService(t)
	// 1M + free tier requests → exactly 1M billable → $0.50
	usage := cosmoflare.WorkersUsage{Requests: 1_000_000 + int64(cosmoflare.WorkersFreeRequests)}
	est := svc.EstimateWorkersCost(usage)
	want := 0.50
	if est.TotalCost != want {
		t.Errorf("TotalCost = %.4f, want %.4f", est.TotalCost, want)
	}
}

func TestCostService_EstimateWorkersCost_UsagePreserved(t *testing.T) {
	svc := newTestCostService(t)
	usage := cosmoflare.WorkersUsage{Requests: 500_000, CPUTimeMs: 1234}
	est := svc.EstimateWorkersCost(usage)
	if est.Usage.Requests != usage.Requests {
		t.Errorf("Usage.Requests = %d, want %d", est.Usage.Requests, usage.Requests)
	}
	if est.Usage.CPUTimeMs != usage.CPUTimeMs {
		t.Errorf("Usage.CPUTimeMs = %d, want %d", est.Usage.CPUTimeMs, usage.CPUTimeMs)
	}
}

// --- KV cost estimation ---

func TestCostService_EstimateKVCost_ZeroUsage(t *testing.T) {
	svc := newTestCostService(t)
	est := svc.EstimateKVCost(cosmoflare.KVUsage{})
	if est.TotalCost != 0 {
		t.Errorf("TotalCost = %.6f, want 0", est.TotalCost)
	}
}

func TestCostService_EstimateKVCost_ReadsOnly(t *testing.T) {
	svc := newTestCostService(t)
	usage := cosmoflare.KVUsage{Reads: 1_000_000}
	est := svc.EstimateKVCost(usage)
	// 1M reads * $0.50/M = $0.50
	want := 0.50
	if est.ReadsCost != want {
		t.Errorf("ReadsCost = %.4f, want %.4f", est.ReadsCost, want)
	}
}

func TestCostService_EstimateKVCost_WritesOnly(t *testing.T) {
	svc := newTestCostService(t)
	usage := cosmoflare.KVUsage{Writes: 1_000_000}
	est := svc.EstimateKVCost(usage)
	// 1M writes * $5.00/M = $5.00
	want := 5.00
	if est.WritesCost != want {
		t.Errorf("WritesCost = %.4f, want %.4f", est.WritesCost, want)
	}
}

func TestCostService_EstimateKVCost_StorageOnly(t *testing.T) {
	svc := newTestCostService(t)
	usage := cosmoflare.KVUsage{StorageGB: 10}
	est := svc.EstimateKVCost(usage)
	// 10 GB * $0.50/GB = $5.00
	want := 5.00
	if est.StorageCost != want {
		t.Errorf("StorageCost = %.4f, want %.4f", est.StorageCost, want)
	}
}

func TestCostService_EstimateKVCost_TotalIsSum(t *testing.T) {
	svc := newTestCostService(t)
	usage := cosmoflare.KVUsage{Reads: 2_000_000, Writes: 500_000, StorageGB: 5}
	est := svc.EstimateKVCost(usage)
	wantTotal := est.ReadsCost + est.WritesCost + est.StorageCost
	if est.TotalCost != wantTotal {
		t.Errorf("TotalCost = %.6f, want %.6f (sum of components)", est.TotalCost, wantTotal)
	}
}

func TestCostService_EstimateKVCost_UsagePreserved(t *testing.T) {
	svc := newTestCostService(t)
	usage := cosmoflare.KVUsage{Reads: 100, Writes: 200, StorageGB: 3.5}
	est := svc.EstimateKVCost(usage)
	if est.Usage.Reads != usage.Reads {
		t.Errorf("Usage.Reads = %d, want %d", est.Usage.Reads, usage.Reads)
	}
	if est.Usage.Writes != usage.Writes {
		t.Errorf("Usage.Writes = %d, want %d", est.Usage.Writes, usage.Writes)
	}
	if est.Usage.StorageGB != usage.StorageGB {
		t.Errorf("Usage.StorageGB = %.2f, want %.2f", est.Usage.StorageGB, usage.StorageGB)
	}
}

// --- EstimateTotal ---

func TestCostService_EstimateTotal_ZeroUsage(t *testing.T) {
	svc := newTestCostService(t)
	total := svc.EstimateTotal(cosmoflare.R2Usage{}, cosmoflare.WorkersUsage{}, cosmoflare.KVUsage{})
	if total.TotalMonthlyCost != 0 {
		t.Errorf("TotalMonthlyCost = %.6f, want 0", total.TotalMonthlyCost)
	}
	if total.R2 == nil {
		t.Error("total.R2 is nil")
	}
	if total.Workers == nil {
		t.Error("total.Workers is nil")
	}
	if total.KV == nil {
		t.Error("total.KV is nil")
	}
}

func TestCostService_EstimateTotal_SumsComponents(t *testing.T) {
	svc := newTestCostService(t)
	r2 := cosmoflare.R2Usage{StorageGB: 100}
	workers := cosmoflare.WorkersUsage{Requests: 2_000_000 + int64(cosmoflare.WorkersFreeRequests)}
	kv := cosmoflare.KVUsage{Reads: 1_000_000, Writes: 1_000_000}
	total := svc.EstimateTotal(r2, workers, kv)
	wantTotal := total.R2.TotalCost + total.Workers.TotalCost + total.KV.TotalCost
	if total.TotalMonthlyCost != wantTotal {
		t.Errorf("TotalMonthlyCost = %.6f, want %.6f", total.TotalMonthlyCost, wantTotal)
	}
}

func TestCostService_EstimateTotal_DefaultPeriod(t *testing.T) {
	svc := newTestCostService(t)
	total := svc.EstimateTotal(cosmoflare.R2Usage{}, cosmoflare.WorkersUsage{}, cosmoflare.KVUsage{})
	if total.Period != "30d" {
		t.Errorf("Period = %q, want %q", total.Period, "30d")
	}
}

func TestCostService_EstimateTotal_HasDisclaimer(t *testing.T) {
	svc := newTestCostService(t)
	total := svc.EstimateTotal(cosmoflare.R2Usage{}, cosmoflare.WorkersUsage{}, cosmoflare.KVUsage{})
	if total.Disclaimer == "" {
		t.Error("TotalCostEstimate.Disclaimer is empty")
	}
}

// --- NewCostService validation ---

func TestCostService_New_RequiresAccountID(t *testing.T) {
	_, err := cosmoflare.NewCostService("", "token")
	if err == nil {
		t.Error("NewCostService with empty accountID should return error")
	}
}

func TestCostService_New_RequiresAPIToken(t *testing.T) {
	_, err := cosmoflare.NewCostService("account", "")
	if err == nil {
		t.Error("NewCostService with empty apiToken should return error")
	}
}

func TestCostService_New_BothEmptyReturnsError(t *testing.T) {
	_, err := cosmoflare.NewCostService("", "")
	if err == nil {
		t.Error("NewCostService with both empty fields should return error")
	}
}

func TestCostService_New_ValidInputReturnsService(t *testing.T) {
	svc, err := cosmoflare.NewCostService("acct-123", "tok-abc")
	if err != nil {
		t.Errorf("NewCostService with valid inputs returned error: %v", err)
	}
	if svc == nil {
		t.Error("NewCostService returned nil service with no error")
	}
}

// --- Pricing constants sanity checks ---

func TestCostConstants_R2Pricing(t *testing.T) {
	if cosmoflare.R2StoragePerGB <= 0 {
		t.Errorf("R2StoragePerGB = %.4f, must be > 0", cosmoflare.R2StoragePerGB)
	}
	if cosmoflare.R2ClassAPerMillion <= 0 {
		t.Errorf("R2ClassAPerMillion = %.4f, must be > 0", cosmoflare.R2ClassAPerMillion)
	}
	if cosmoflare.R2ClassBPerMillion <= 0 {
		t.Errorf("R2ClassBPerMillion = %.4f, must be > 0", cosmoflare.R2ClassBPerMillion)
	}
	// Class A (write) ops cost more than Class B (read) ops — this is a CF invariant
	if cosmoflare.R2ClassAPerMillion <= cosmoflare.R2ClassBPerMillion {
		t.Errorf("R2ClassAPerMillion (%.2f) should be > R2ClassBPerMillion (%.2f)", cosmoflare.R2ClassAPerMillion, cosmoflare.R2ClassBPerMillion)
	}
}

func TestCostConstants_WorkersPricing(t *testing.T) {
	if cosmoflare.WorkersRequestsPerMillion <= 0 {
		t.Errorf("WorkersRequestsPerMillion = %.4f, must be > 0", cosmoflare.WorkersRequestsPerMillion)
	}
	if cosmoflare.WorkersFreeRequests <= 0 {
		t.Errorf("WorkersFreeRequests = %d, must be > 0", cosmoflare.WorkersFreeRequests)
	}
}

func TestCostConstants_KVPricing(t *testing.T) {
	if cosmoflare.KVReadsPerMillion <= 0 {
		t.Errorf("KVReadsPerMillion = %.4f, must be > 0", cosmoflare.KVReadsPerMillion)
	}
	if cosmoflare.KVWritesPerMillion <= 0 {
		t.Errorf("KVWritesPerMillion = %.4f, must be > 0", cosmoflare.KVWritesPerMillion)
	}
	if cosmoflare.KVStoragePerGB <= 0 {
		t.Errorf("KVStoragePerGB = %.4f, must be > 0", cosmoflare.KVStoragePerGB)
	}
	// Writes cost more than reads — this is a CF invariant
	if cosmoflare.KVWritesPerMillion <= cosmoflare.KVReadsPerMillion {
		t.Errorf("KVWritesPerMillion (%.2f) should be > KVReadsPerMillion (%.2f)", cosmoflare.KVWritesPerMillion, cosmoflare.KVReadsPerMillion)
	}
}
