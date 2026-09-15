package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// ---------------------------------------------------------------------------
// BUG-035: MCP tools generated from the cobra command tree
//
// These tests exercise the generator over a small in-memory cobra tree plus
// the real rootCmd (counting only — nothing executes during generation).
// ---------------------------------------------------------------------------

// mcpInvocation records one programmatic command invocation.
type mcpInvocation struct {
	Args    []string
	Flags   map[string]string
	JSONSet bool
}

// mcpTestRoot builds a small in-memory command tree:
//
//	cftest read <path>              (read-only, 4 flags of distinct types)
//	cftest delete [name]            (mutating: keyword "delete" AND a --force flag)
//	cftest upload [file]            (mutating: keyword "upload", no --force flag)
//	cftest secret [x]               (hidden)
//	cftest mcp serve                (the MCP server itself — must be skipped)
//	cftest grp leaf [id]            (nested path → grp_leaf)
//	cftest broken <req>             (RunE always errors)
func mcpTestRoot(t *testing.T) (*cobra.Command, *[]mcpInvocation) {
	t.Helper()
	invocations := &[]mcpInvocation{}

	var jsonFlag bool
	root := &cobra.Command{Use: "cftest"}
	root.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output in JSON format")

	root.AddCommand(
		mcpTestReadCmd(t, invocations, &jsonFlag),
		mcpTestDeleteCmd(t, invocations),
		mcpTestStubCmd("upload [file]", "Upload a thing", "uploaded", false),
		mcpTestStubCmd("secret [x]", "Hidden thing", "secret", true),
		mcpTestMcpLikeCmd(),
		mcpTestGroupCmd(),
		mcpTestBrokenCmd(),
	)
	return root, invocations
}

// mcpTestReadCmd builds the flag-rich read-only command: it records every
// invocation (args, resolved flags, and the persistent --json state) and
// echoes its arguments back inside its JSON envelope.
func mcpTestReadCmd(t *testing.T, invocations *[]mcpInvocation, jsonFlag *bool) *cobra.Command {
	t.Helper()
	var readFormat string
	var readLimit int
	var readVerbose bool
	var readTags []string
	readCmd := &cobra.Command{
		Use:   "read <path>",
		Short: "Read a thing",
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			limit, _ := cmd.Flags().GetInt("limit")
			verbose, _ := cmd.Flags().GetBool("verbose")
			tags, _ := cmd.Flags().GetStringSlice("tags")
			*invocations = append(*invocations, mcpInvocation{
				Args:    args,
				Flags:   map[string]string{"format": format, "limit": fmt.Sprint(limit), "verbose": fmt.Sprint(verbose), "tags": strings.Join(tags, ",")},
				JSONSet: *jsonFlag,
			})
			return printTestEnvelope("read ok", map[string]interface{}{"read": args})
		},
	}
	readCmd.Flags().StringVar(&readFormat, "format", "table", "Output format")
	readCmd.Flags().IntVar(&readLimit, "limit", 10, "Maximum rows")
	readCmd.Flags().BoolVar(&readVerbose, "verbose", false, "Verbose output")
	readCmd.Flags().StringSliceVar(&readTags, "tags", nil, "Tags to filter")
	return readCmd
}

// mcpTestDeleteCmd builds the mutating delete command: it records each
// invocation together with the resolved --force flag.
func mcpTestDeleteCmd(t *testing.T, invocations *[]mcpInvocation) *cobra.Command {
	t.Helper()
	var delForce bool
	deleteCmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a thing",
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			*invocations = append(*invocations, mcpInvocation{
				Args:  args,
				Flags: map[string]string{"force": fmt.Sprint(force)},
			})
			return printTestEnvelope("deleted", nil)
		},
	}
	deleteCmd.Flags().BoolVar(&delForce, "force", false, "Skip confirmation")
	return deleteCmd
}

// mcpTestStubCmd returns a plain command that always replies with a fixed
// envelope message and no payload; hidden controls cobra visibility.
func mcpTestStubCmd(use, short, message string, hidden bool) *cobra.Command {
	return &cobra.Command{
		Use:    use,
		Short:  short,
		Hidden: hidden,
		RunE: func(cmd *cobra.Command, args []string) error {
			return printTestEnvelope(message, nil)
		},
	}
}

// mcpTestMcpLikeCmd builds the "mcp serve" subtree that MCP generation must
// exclude from the generated tool list.
func mcpTestMcpLikeCmd() *cobra.Command {
	mcpLikeCmd := &cobra.Command{Use: "mcp", Short: "MCP server"}
	mcpLikeCmd.AddCommand(mcpTestStubCmd("serve", "Serve", "served", false))
	return mcpLikeCmd
}

// mcpTestGroupCmd builds the "grp leaf" subtree; the leaf echoes its
// arguments back inside its JSON envelope.
func mcpTestGroupCmd() *cobra.Command {
	groupCmd := &cobra.Command{Use: "grp", Short: "Group"}
	leafCmd := &cobra.Command{
		Use:   "leaf [id]",
		Short: "Leaf under group",
		RunE: func(cmd *cobra.Command, args []string) error {
			return printTestEnvelope("leaf ok", map[string]interface{}{"leaf": args})
		},
	}
	groupCmd.AddCommand(leafCmd)
	return groupCmd
}

// mcpTestBrokenCmd returns a command whose RunE always fails.
func mcpTestBrokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "broken <req>",
		Short: "Always fails",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("command failed on purpose")
		},
	}
}

// printTestEnvelope prints an OutputResponse-shaped JSON envelope to stdout,
// mirroring what real commands emit under --json.
func printTestEnvelope(message string, data interface{}) error {
	response := OutputResponse{Success: true, Message: message, Data: data}
	raw, err := json.Marshal(response)
	if err != nil {
		return err
	}
	fmt.Println(string(raw))
	return nil
}

func generatedToolNames(server *cosmoflare.MCPServer) []string {
	var names []string
	for _, tool := range server.Tools() {
		names = append(names, tool.Name)
	}
	return names
}

func findGeneratedTool(server *cosmoflare.MCPServer, name string) *cosmoflare.MCPTool {
	tools := server.Tools()
	for i := range tools {
		if tools[i].Name == name {
			return &tools[i]
		}
	}
	return nil
}

func TestGenerateToolNames(t *testing.T) {
	root, _ := mcpTestRoot(t)
	server := cosmoflare.NewMCPServer("", "")
	stats := registerGeneratedCLITools(server, root, false)

	if stats.MutatingExcluded == 0 {
		t.Error("expected at least one mutating command excluded")
	}
	want := []string{"broken", "grp_leaf", "read"}
	got := generatedToolNames(server)
	if len(got) != len(want) {
		t.Fatalf("expected tools %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected tools %v, got %v", want, got)
		}
	}
}

func TestGenerateToolSchemaShape(t *testing.T) {
	root, _ := mcpTestRoot(t)
	server := cosmoflare.NewMCPServer("", "")
	registerGeneratedCLITools(server, root, false)

	tool := findGeneratedTool(server, "read")
	if tool == nil {
		t.Fatal("read tool not found")
	}
	if tool.Description != "Read a thing" {
		t.Fatalf("expected Short text as description, got %q", tool.Description)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
		t.Fatalf("invalid schema JSON: %v", err)
	}
	if schema["type"] != "object" {
		t.Fatalf("expected type object, got %v", schema["type"])
	}
	if ap, ok := schema["additionalProperties"].(bool); !ok || ap {
		t.Fatalf("expected additionalProperties=false, got %v", schema["additionalProperties"])
	}
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected properties map, got %T", schema["properties"])
	}

	// Positional arg becomes a required string property named arg1.
	arg1, ok := props["arg1"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected arg1 property, got %v", props["arg1"])
	}
	if arg1["type"] != "string" {
		t.Fatalf("expected arg1 type string, got %v", arg1["type"])
	}
	if !strings.Contains(fmt.Sprint(arg1["description"]), "path") {
		t.Fatalf("expected arg1 description to mention the placeholder 'path', got %v", arg1["description"])
	}
	required, ok := schema["required"].([]interface{})
	if !ok || len(required) != 1 || required[0] != "arg1" {
		t.Fatalf("expected required [arg1], got %v", schema["required"])
	}

	// Flags map to typed properties.
	for prop, wantType := range map[string]string{
		"format":  "string",
		"limit":   "integer",
		"verbose": "boolean",
		"tags":    "array",
	} {
		p, ok := props[prop].(map[string]interface{})
		if !ok {
			t.Fatalf("expected %s property, got %v", prop, props[prop])
		}
		if p["type"] != wantType {
			t.Fatalf("expected %s type %s, got %v", prop, wantType, p["type"])
		}
	}
	tags, ok := props["tags"].(map[string]interface{})
	if !ok {
		t.Fatal("expected tags property")
	}
	items, ok := tags["items"].(map[string]interface{})
	if !ok || items["type"] != "string" {
		t.Fatalf("expected tags items type string, got %v", tags["items"])
	}

	// No inherited root flags leaked into the schema.
	for _, banned := range []string{"json", "verbose_root", "api-token", "account-id", "dry-run"} {
		if _, present := props[banned]; present {
			t.Fatalf("inherited flag %q must not be exposed in the schema", banned)
		}
	}
}

func TestGenerateDescriptionFallsBackToLong(t *testing.T) {
	root := &cobra.Command{Use: "cftest"}
	noShort := &cobra.Command{
		Use:  "describe",
		Long: "First line of long text.\nSecond line.",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	root.AddCommand(noShort)

	server := cosmoflare.NewMCPServer("", "")
	registerGeneratedCLITools(server, root, false)
	tool := findGeneratedTool(server, "describe")
	if tool == nil {
		t.Fatal("describe tool not found")
	}
	if tool.Description != "First line of long text." {
		t.Fatalf("expected Long first line as description, got %q", tool.Description)
	}
}

func TestGenerateHiddenExcluded(t *testing.T) {
	root, _ := mcpTestRoot(t)
	server := cosmoflare.NewMCPServer("", "")
	registerGeneratedCLITools(server, root, false)
	if findGeneratedTool(server, "secret") != nil {
		t.Fatal("hidden command must not be registered as a tool")
	}
}

func TestGenerateMCPCommandTreeExcluded(t *testing.T) {
	root, _ := mcpTestRoot(t)
	server := cosmoflare.NewMCPServer("", "")
	registerGeneratedCLITools(server, root, false)
	if findGeneratedTool(server, "mcp") != nil || findGeneratedTool(server, "mcp_serve") != nil {
		t.Fatal("the mcp command tree must not be registered as tools")
	}
}

func TestGenerateMutatingGating(t *testing.T) {
	// Excluded by default (fail-closed): the keyword+flag command and the
	// keyword-only command alike.
	root, _ := mcpTestRoot(t)
	server := cosmoflare.NewMCPServer("", "")
	registerGeneratedCLITools(server, root, false)
	if findGeneratedTool(server, "delete") != nil {
		t.Fatal("mutating command (keyword + --force) must be excluded by default")
	}
	if findGeneratedTool(server, "upload") != nil {
		t.Fatal("mutating command (keyword) must be excluded by default")
	}

	// Included when the project config opts in.
	server2 := cosmoflare.NewMCPServer("", "")
	stats := registerGeneratedCLITools(server2, root, true)
	if findGeneratedTool(server2, "delete") == nil {
		t.Fatal("mutating command must be registered when allow_mutations is enabled")
	}
	if findGeneratedTool(server2, "upload") == nil {
		t.Fatal("keyword-mutating command must be registered when allow_mutations is enabled")
	}
	if stats.MutatingExcluded != 0 {
		t.Fatalf("expected 0 mutating excluded when enabled, got %d", stats.MutatingExcluded)
	}
}

func TestGenerateCuratedToolsWin(t *testing.T) {
	root, _ := mcpTestRoot(t)
	server := cosmoflare.NewMCPServer("", "")
	// Pre-register a "curated" tool whose name a generated tool would collide
	// with — the curated registration must survive.
	server.RegisterToolFunc("read", "Curated read tool", nil, func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return "curated", nil
	})
	registerGeneratedCLITools(server, root, false)

	tool := findGeneratedTool(server, "read")
	if tool == nil {
		t.Fatal("read tool missing")
	}
	if tool.Description != "Curated read tool" {
		t.Fatalf("curated tool must win over generated tool, got %q", tool.Description)
	}
}

// TestExecuteGeneratedToolMapsArgsAndFlags drives a full tools/call round
// trip: the tool args must map onto positional args and --flag=value pairs,
// and the JSON captured from the OutputResponse envelope must come back as
// the tool result.
func TestExecuteGeneratedToolMapsArgsAndFlags(t *testing.T) {
	root, invocations := mcpTestRoot(t)
	server := cosmoflare.NewMCPServer("", "")
	registerGeneratedCLITools(server, root, false)

	req := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read","arguments":{"arg1":"a/b.txt","format":"json","limit":5,"verbose":true,"tags":["x","y"]}}}`
	resp := server.HandleRequest(context.Background(), []byte(req))

	var r struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(resp, &r); err != nil {
		t.Fatal(err)
	}
	if r.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %s", r.Error.Message)
	}
	if r.Result.IsError {
		t.Fatalf("expected success, got error content: %s", r.Result.Content[0].Text)
	}
	if !strings.Contains(r.Result.Content[0].Text, "read ok") {
		t.Fatalf("expected envelope message in result, got %s", r.Result.Content[0].Text)
	}
	if !strings.Contains(r.Result.Content[0].Text, "a/b.txt") {
		t.Fatalf("expected envelope data in result, got %s", r.Result.Content[0].Text)
	}

	if len(*invocations) != 1 {
		t.Fatalf("expected 1 invocation, got %d", len(*invocations))
	}
	inv := (*invocations)[0]
	if len(inv.Args) != 1 || inv.Args[0] != "a/b.txt" {
		t.Fatalf("expected positional args [a/b.txt], got %v", inv.Args)
	}
	if inv.Flags["format"] != "json" {
		t.Fatalf("expected format=json, got %v", inv.Flags["format"])
	}
	if inv.Flags["limit"] != "5" {
		t.Fatalf("expected limit=5, got %v", inv.Flags["limit"])
	}
	if inv.Flags["verbose"] != "true" {
		t.Fatalf("expected verbose=true, got %v", inv.Flags["verbose"])
	}
	if inv.Flags["tags"] != "x,y" {
		t.Fatalf("expected tags=x,y, got %v", inv.Flags["tags"])
	}
	if !inv.JSONSet {
		t.Fatal("expected --json to be set for the executed command")
	}
}

// TestExecuteGeneratedToolResetsStaleFlagState verifies a flag set by one
// tool call does not leak into the next call (cobra flags are global state).
func TestExecuteGeneratedToolResetsStaleFlagState(t *testing.T) {
	root, invocations := mcpTestRoot(t)
	server := cosmoflare.NewMCPServer("", "")
	registerGeneratedCLITools(server, root, false)

	server.HandleRequest(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read","arguments":{"arg1":"one","limit":5}}}`))
	server.HandleRequest(context.Background(), []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"read","arguments":{"arg1":"two"}}}`))

	if len(*invocations) != 2 {
		t.Fatalf("expected 2 invocations, got %d", len(*invocations))
	}
	second := (*invocations)[1]
	if second.Flags["limit"] != "10" {
		t.Fatalf("expected stale --limit reset to default 10, got %v", second.Flags["limit"])
	}
	if second.Flags["format"] != "table" {
		t.Fatalf("expected stale --format reset to default table, got %v", second.Flags["format"])
	}
}

// TestExecuteGeneratedToolReturnsError verifies a failing command surfaces as
// a tool error result instead of a JSON-RPC protocol error.
func TestExecuteGeneratedToolReturnsError(t *testing.T) {
	root, _ := mcpTestRoot(t)
	server := cosmoflare.NewMCPServer("", "")
	registerGeneratedCLITools(server, root, false)

	resp := server.HandleRequest(context.Background(), []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"broken","arguments":{"arg1":"x"}}}`))
	var r struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(resp, &r); err != nil {
		t.Fatal(err)
	}
	if r.Error != nil {
		t.Fatalf("command failure must be a tool error, not a JSON-RPC error: %s", r.Error.Message)
	}
	if !r.Result.IsError {
		t.Fatalf("expected isError=true, got content %s", r.Result.Content[0].Text)
	}
	if !strings.Contains(r.Result.Content[0].Text, "command failed on purpose") {
		t.Fatalf("expected command error surfaced, got %s", r.Result.Content[0].Text)
	}
}

// TestExecuteGeneratedToolCredentialGuard verifies the fail-closed guard: a
// credential-requiring command returns a tool error instead of reaching
// rootCmd.PersistentPreRun, which would os.Exit(1) and kill the MCP server.
func TestExecuteGeneratedToolCredentialGuard(t *testing.T) {
	savedToken, savedAccount := APIToken, AccountID
	defer func() { APIToken, AccountID = savedToken, savedAccount }()
	APIToken, AccountID = "", ""
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")

	root := &cobra.Command{Use: "cftest"}
	ran := false
	kvList := &cobra.Command{
		Use:  "kv",
		RunE: func(cmd *cobra.Command, args []string) error { ran = true; return nil },
	}
	root.AddCommand(kvList)

	_, err := executeCommandTool(context.Background(), root, kvList, []string{"kv"}, map[string]interface{}{})
	if err == nil {
		t.Fatal("expected credential error, got nil")
	}
	if !strings.Contains(err.Error(), "requires Cloudflare credentials") {
		t.Fatalf("expected credential error message, got %v", err)
	}
	if ran {
		t.Fatal("command must not execute without credentials")
	}
}

func TestCommandIsMutating(t *testing.T) {
	root := &cobra.Command{Use: "cftest"}
	cases := []struct {
		use      string
		mutating bool
	}{
		{"list", false},
		{"get [key]", false},
		{"status", false},
		{"output [fmt]", false}, // contains "put" as substring but not as a word
		{"create [x]", true},
		{"update [x]", true},
		{"delete [x]", true},
		{"remove [x]", true},
		{"upload [x]", true},
		{"put [k] [v]", true},
		{"copy [src]", true},
		{"move [src]", true},
		{"rename [src]", true},
		{"deploy [name]", true},
		{"apply", true},
		{"sync [dir]", true},
		{"watch [dir]", true},
		{"purge [zone]", true},
		{"import [file]", true},
		{"restore [file]", true},
		{"enable [x]", true},
		{"disable [x]", true},
		{"revoke [x]", true},
		{"rotate [x]", true},
		{"put-object [k]", true},
		{"export-write [f]", true},
	}
	for _, tc := range cases {
		c := &cobra.Command{Use: tc.use, RunE: func(cmd *cobra.Command, args []string) error { return nil }}
		root.AddCommand(c)
		if got := commandIsMutating(c); got != tc.mutating {
			t.Errorf("%s: expected mutating=%v, got %v", tc.use, tc.mutating, got)
		}
	}

	// A --force or --confirm flag alone marks a command as mutating.
	forced := &cobra.Command{
		Use:  "nuke",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	forced.Flags().Bool("force", false, "Skip confirmation")
	if !commandIsMutating(forced) {
		t.Error("expected --force flag to mark command as mutating")
	}
	confirmed := &cobra.Command{
		Use:  "blast",
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}
	confirmed.Flags().Bool("confirm", false, "Skip confirmation")
	if !commandIsMutating(confirmed) {
		t.Error("expected --confirm flag to mark command as mutating")
	}
}

func TestCommandRequiresCredentials(t *testing.T) {
	// Fake tree: config subtree requires nothing, kv does.
	root := &cobra.Command{Use: "cftest"}
	cfgGroup := &cobra.Command{Use: "config"}
	cfgList := &cobra.Command{Use: "list", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	cfgGroup.AddCommand(cfgList)
	kvList := &cobra.Command{Use: "kv", RunE: func(cmd *cobra.Command, args []string) error { return nil }}
	root.AddCommand(cfgGroup, kvList)

	if commandRequiresCredentials(cfgList) {
		t.Error("commands under config must not require credentials")
	}
	if !commandRequiresCredentials(kvList) {
		t.Error("kv command must require credentials")
	}

	// Real tree spot checks.
	if commandRequiresCredentials(findRealSub(t, "config")) {
		t.Error("real config command must not require credentials")
	}
	if !commandRequiresCredentials(findRealSub(t, "kv")) {
		t.Error("real kv command must require credentials")
	}
}

func findRealSub(t *testing.T, name string) *cobra.Command {
	t.Helper()
	for _, c := range rootCmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	t.Fatalf("command %q not found on real root", name)
	return nil
}

func TestLoadMCPMutationAllowance(t *testing.T) {
	// Explicit opt-in.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".cosmoflare.yaml"), []byte("mcp:\n  allow_mutations: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if !loadMCPMutationAllowance(dir) {
		t.Fatal("expected allow_mutations=true to be honoured")
	}

	// Present config without the key (fail-closed).
	dir2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir2, ".cosmoflare.yaml"), []byte("bucket: b\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if loadMCPMutationAllowance(dir2) {
		t.Fatal("expected allow_mutations to default to false")
	}

	// Absent config (fail-closed).
	if loadMCPMutationAllowance(t.TempDir()) {
		t.Fatal("expected no config to mean mutations excluded")
	}
}

// TestGenerateRealTreeCounts walks the REAL command tree. Generation is
// static: nothing executes, so this is safe and fast.
func TestGenerateRealTreeCounts(t *testing.T) {
	server := cosmoflare.NewMCPServer("", "")
	server.RegisterDefaultTools()
	curated := len(server.Tools())

	stats := registerGeneratedCLITools(server, rootCmd, false)
	total := len(server.Tools())
	generated := total - curated

	t.Logf("real tree: generated=%d (read-only), mutating-excluded=%d, skipped=%d, curated=%d",
		generated, stats.MutatingExcluded, stats.Skipped, curated)

	if generated <= 100 {
		t.Errorf("expected the generator to cover a large fraction of the CLI surface, got %d tools", generated)
	}
	if stats.MutatingExcluded == 0 {
		t.Error("expected mutating commands to be excluded by default")
	}

	for _, tool := range server.Tools() {
		for _, banned := range []string{"mcp", "completion", "help", "serve", "dev", "dashboard", "init", "setup", "watch"} {
			if tool.Name == banned || strings.HasPrefix(tool.Name, banned+"_") {
				if banned != "mcp" || tool.Name != "cosmoflare_kv_list" {
					t.Errorf("tool %q must not be generated (banned prefix %q)", tool.Name, banned)
				}
			}
		}
		if !strings.HasPrefix(tool.Name, "cosmoflare_") {
			// Generated tool: verify every schema is a closed object.
			var schema map[string]interface{}
			if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
				t.Errorf("tool %q has invalid schema: %v", tool.Name, err)
				continue
			}
			if ap, ok := schema["additionalProperties"].(bool); !ok || ap {
				t.Errorf("tool %q schema must set additionalProperties=false", tool.Name)
			}
		}
	}

	// With mutations allowed, the generated count grows by the excluded ones.
	server2 := cosmoflare.NewMCPServer("", "")
	stats2 := registerGeneratedCLITools(server2, rootCmd, true)
	if stats2.MutatingExcluded != 0 {
		t.Errorf("expected 0 excluded when allowed, got %d", stats2.MutatingExcluded)
	}
	if len(server2.Tools()) != generated+stats.MutatingExcluded {
		t.Errorf("expected %d tools with mutations allowed, got %d", generated+stats.MutatingExcluded, len(server2.Tools()))
	}
}
