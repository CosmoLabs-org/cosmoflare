package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

// ---------------------------------------------------------------------------
// BUG-035: MCP tools generated from the cobra command tree
//
// The MCP server previously exposed 8 hand-written tools while the CLI has
// 300+ commands (2% of the surface): an agent could not create a DNS record,
// put an R2 object, or touch D1/Pages/Queues. This generator walks the cobra
// root command tree and registers one MCP tool per eligible leaf command, so
// the full CLI surface is agent-reachable.
//
// The library (pkg/cosmoflare) cannot import cmd (import cycle), so the
// generator lives here and registers tools through the library's public
// RegisterToolFunc API.
//
// MUTATION GATING IS FAIL-CLOSED. A command is classified MUTATING when any
// word of its command path matches a mutation keyword (create, update,
// delete, remove, upload, put, copy, move, rename, deploy, apply, sync,
// watch, purge, import, restore, export-write, enable, disable, revoke,
// rotate, put-object) or it defines a --force/--confirm flag. MUTATING
// commands are registered ONLY when the project config opts in with
//
//	mcp:
//	  allow_mutations: true
//
// in .cosmoflare.yaml. With no such config key, tools/list simply omits
// them. Read-only commands register unconditionally.
//
// Generation is STATIC: the walk only reads command metadata (name, help
// text, flags); nothing executes. Long-running/interactive commands are
// denylisted because executing them would block the stdio server forever.
// ---------------------------------------------------------------------------

// mcpMutationKeywords marks a command path word as mutating.
var mcpMutationKeywords = map[string]bool{
	"create":       true,
	"update":       true,
	"delete":       true,
	"remove":       true,
	"upload":       true,
	"put":          true,
	"copy":         true,
	"move":         true,
	"rename":       true,
	"deploy":       true,
	"apply":        true,
	"sync":         true,
	"watch":        true,
	"purge":        true,
	"import":       true,
	"restore":      true,
	"export-write": true,
	"enable":       true,
	"disable":      true,
	"revoke":       true,
	"rotate":       true,
	"put-object":   true,
}

// mcpBlockedCommands never become tools: the MCP server itself, cobra
// builtins, and long-running/interactive commands that would block the stdio
// server (a server, TUI, wizard, or file watcher never returns).
var mcpBlockedCommands = map[string]bool{
	"mcp":        true,
	"help":       true,
	"completion": true,
	"serve":      true,
	"dev":        true,
	"dashboard":  true,
	"init":       true,
	"setup":      true,
	"watch":      true,
}

// mcpGenerateStats reports what a generator walk produced.
type mcpGenerateStats struct {
	Registered       int // tools registered
	MutatingExcluded int // mutating commands omitted (policy or opt-in state)
	Skipped          int // eligible commands skipped (name collision, blocked)
}

// commandPathWords returns the command path below root, e.g.
// ["kv", "namespace", "list"] for `cosmoflare kv namespace list`.
func commandPathWords(root, cmd *cobra.Command) []string {
	path := cmd.CommandPath()
	prefix := root.Name() + " "
	if strings.HasPrefix(path, prefix) {
		path = strings.TrimPrefix(path, prefix)
	}
	return strings.Fields(path)
}

// mcpToolNameFromWords converts command path words into an MCP tool name,
// e.g. ["kv", "namespace", "list"] → "kv_namespace_list".
var mcpUnsafeNameChars = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

func mcpToolNameFromWords(words []string) string {
	return mcpUnsafeNameChars.ReplaceAllString(strings.Join(words, "_"), "_")
}

// commandIsMutating applies the fail-closed classification: whole path words
// are compared (so "output" never matches "put"), plus a --force/--confirm
// flag anywhere on the command counts as mutating.
func commandIsMutating(cmd *cobra.Command) bool {
	for _, word := range commandPathWords(cmd.Root(), cmd) {
		if mcpMutationKeywords[word] {
			return true
		}
	}
	for _, flags := range []*pflag.FlagSet{cmd.LocalFlags(), cmd.InheritedFlags()} {
		for _, name := range []string{"force", "confirm"} {
			if f := flags.Lookup(name); f != nil {
				return true
			}
		}
	}
	return false
}

// commandRequiresCredentials reports whether the command's PersistentPreRun
// chain validates Cloudflare credentials (mirrors root's skip list). Used as
// a fail-closed guard so a missing-credential tool call returns a tool error
// instead of letting os.Exit(1) in rootCmd.PersistentPreRun kill the MCP
// server process.
func commandRequiresCredentials(cmd *cobra.Command) bool {
	for _, skip := range noCredentialRequiredCommands {
		for p := cmd; p != nil; p = p.Parent() {
			if p.Name() == skip {
				return false
			}
		}
	}
	return true
}

// mcpCredentialsAvailable mirrors rootCmd.PersistentPreRun's validation.
func mcpCredentialsAvailable() bool {
	token := APIToken
	if token == "" {
		token = os.Getenv("CLOUDFLARE_API_TOKEN")
	}
	if len(token) < 10 {
		return false
	}
	account := AccountID
	if account == "" {
		account = os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	}
	return account != ""
}

// loadMCPMutationAllowance reads the fail-closed mutation switch from the
// project config in dir. Any error (including "no project config found")
// means mutations stay excluded.
func loadMCPMutationAllowance(dir string) bool {
	cfg, err := cosmoflare.LoadProjectConfig(dir)
	if err != nil {
		return false
	}
	return cfg.MCP.AllowMutations
}

// registerGeneratedCLITools walks root's cobra tree and registers one MCP
// tool per eligible leaf command. Commands whose generated name collides
// with an already-registered (curated) tool are skipped — curated wins.
func registerGeneratedCLITools(server *cosmoflare.MCPServer, root *cobra.Command, allowMutations bool) mcpGenerateStats {
	var stats mcpGenerateStats

	var walk func(c *cobra.Command, path []string)
	walk = func(c *cobra.Command, path []string) {
		name := c.Name()

		// Blocked or hidden commands are skipped together with their subtree.
		if c.Hidden || mcpBlockedCommands[name] {
			stats.Skipped++
			return
		}

		next := path
		if c != root {
			next = append(append([]string{}, path...), name)
		}

		if c.Runnable() && c != root {
			toolName := mcpToolNameFromWords(next)
			switch {
			case server.HasTool(toolName):
				// A curated tool already owns this name; it wins.
				stats.Skipped++
			case commandIsMutating(c) && !allowMutations:
				// Fail-closed mutation policy: omit from tools/list.
				stats.MutatingExcluded++
			default:
				server.RegisterToolFunc(toolName, commandToolDescription(c), buildToolInputSchema(c), func(target *cobra.Command, words []string) cosmoflare.MCPToolHandlerFunc {
					return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
						return executeCommandTool(ctx, root, target, words, args)
					}
				}(c, next))
				stats.Registered++
			}
		}

		for _, sub := range c.Commands() {
			walk(sub, next)
		}
	}
	walk(root, nil)

	return stats
}

// commandToolDescription picks the command's Short text, falling back to the
// first line of Long, then to the tool name.
func commandToolDescription(c *cobra.Command) string {
	if desc := strings.TrimSpace(c.Short); desc != "" {
		return desc
	}
	for _, line := range strings.Split(c.Long, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed
		}
	}
	return mcpToolNameFromWords(commandPathWords(c.Root(), c))
}

// positionalHint describes one positional argument parsed from the command's
// Use string (e.g. "put [namespace-id] [key]" → two hints).
type positionalHint struct {
	Name     string // placeholder as written, without brackets
	Variadic bool   // declared with a "..." suffix
}

// positionalHints extracts argument placeholders from the command's Use
// string. Words after the command name become hints; surrounding [] or <>
// is stripped, "[flags]" is ignored, and a "..." suffix marks variadic.
func positionalHints(c *cobra.Command) []positionalHint {
	use := strings.TrimSpace(c.Use)
	if i := strings.IndexAny(use, " \t"); i >= 0 {
		use = use[i+1:]
	} else {
		return nil
	}

	var hints []positionalHint
	for _, field := range strings.Fields(use) {
		if field == "[flags]" {
			continue
		}
		name := strings.Trim(field, "[]<>")
		variadic := strings.HasSuffix(name, "...")
		name = strings.TrimSuffix(name, "...")
		if name == "" {
			continue
		}
		hints = append(hints, positionalHint{Name: name, Variadic: variadic})
	}
	return hints
}

// buildToolInputSchema derives a JSON Schema object from the command's
// positional argument placeholders and its own flags: positional args become
// required string properties arg1..argN (variadic args become an optional
// array of strings), each flag becomes a property of the matching type, and
// additionalProperties is closed.
func buildToolInputSchema(c *cobra.Command) map[string]interface{} {
	properties := map[string]interface{}{}
	var required []string

	for i, hint := range positionalHints(c) {
		name := fmt.Sprintf("arg%d", i+1)
		if hint.Variadic {
			properties[name] = map[string]interface{}{
				"type":        "array",
				"items":       map[string]interface{}{"type": "string"},
				"description": fmt.Sprintf("%s (repeatable positional argument %d)", hint.Name, i+1),
			}
			continue
		}
		properties[name] = map[string]interface{}{
			"type":        "string",
			"description": fmt.Sprintf("%s (positional argument %d)", hint.Name, i+1),
		}
		required = append(required, name)
	}

	// Only the command's own flags (LocalFlags) are exposed; inherited root
	// flags (--json, --api-token, --account-id, ...) stay internal.
	c.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden || f.Deprecated != "" {
			return
		}
		properties[f.Name] = mcpFlagProperty(f)
	})

	schema := map[string]interface{}{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// mcpFlagProperty maps a pflag flag to a JSON Schema property.
func mcpFlagProperty(f *pflag.Flag) map[string]interface{} {
	prop := map[string]interface{}{"description": f.Usage}
	switch f.Value.Type() {
	case "bool":
		prop["type"] = "boolean"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "count":
		prop["type"] = "integer"
	case "float32", "float64":
		prop["type"] = "number"
	case "stringSlice", "stringArray":
		prop["type"] = "array"
		prop["items"] = map[string]interface{}{"type": "string"}
	case "stringToString":
		prop["type"] = "object"
		prop["additionalProperties"] = map[string]interface{}{"type": "string"}
	default: // string, duration, and anything else textual
		prop["type"] = "string"
	}
	return prop
}

// mcpExecMu serializes programmatic command execution: cobra flags and the
// os.Stdout/os.Stdin swaps are process-global state.
var mcpExecMu sync.Mutex

// executeCommandTool invokes the command programmatically with --json set,
// the tool args mapped to positional args plus --flag=value pairs, captures
// the JSON the command prints, and returns it as the tool result. Errors
// from the command surface as tool errors (isError=true), not JSON-RPC
// errors.
func executeCommandTool(ctx context.Context, root, target *cobra.Command, pathWords []string, args map[string]interface{}) (interface{}, error) {
	mcpExecMu.Lock()
	defer mcpExecMu.Unlock()

	// Fail-closed credential guard: rootCmd.PersistentPreRun would call
	// os.Exit(1) on missing credentials, killing the whole MCP server.
	if commandRequiresCredentials(target) && !mcpCredentialsAvailable() {
		return nil, fmt.Errorf("command %q requires Cloudflare credentials: set CLOUDFLARE_API_TOKEN and CLOUDFLARE_ACCOUNT_ID in the MCP server environment", strings.Join(pathWords, " "))
	}

	mcpResetCommandFlags(target)
	argv := mcpBuildToolArgv(target, pathWords, args)

	out, err := mcpExecuteCaptured(root, target, argv)
	if err != nil {
		return nil, err
	}
	return mcpParseCommandOutput(out)
}

// mcpResetCommandFlags resets the target's flags to defaults so state from a
// previous tool call never leaks into this one. pflag renders an empty slice
// default as "[]", which Set() would parse as a one-element slice ["[]"]; use
// "" for slice types instead (it parses back to an empty slice).
// TASK-009: extracted from executeCommandTool.
func mcpResetCommandFlags(target *cobra.Command) {
	target.LocalFlags().VisitAll(func(f *pflag.Flag) {
		switch f.Value.Type() {
		case "stringSlice", "stringArray", "stringToString":
			if f.DefValue == "[]" || f.DefValue == "" {
				_ = f.Value.Set("")
				return
			}
		}
		_ = f.Value.Set(f.DefValue)
	})
}

// mcpBuildToolArgv maps tool args onto a cobra argv: the command path, --json,
// positional arg1..argN in order, then --flag=value pairs for the flags the
// command itself declares. TASK-009: extracted from executeCommandTool.
func mcpBuildToolArgv(target *cobra.Command, pathWords []string, args map[string]interface{}) []string {
	argv := append([]string{}, pathWords...)
	argv = append(argv, "--json")

	// Positional args, in order.
	for i := 0; ; i++ {
		key := fmt.Sprintf("arg%d", i+1)
		raw, ok := args[key]
		if !ok {
			break
		}
		argv = append(argv, mcpFormatArgValue(raw)...)
	}

	// Flags, only those the command itself declares.
	target.LocalFlags().VisitAll(func(f *pflag.Flag) {
		raw, ok := args[f.Name]
		if !ok {
			return
		}
		argv = append(argv, mcpFormatFlagValue(f.Name, raw)...)
	})

	return argv
}

// mcpExecuteCaptured runs root with argv while capturing stdout (commands
// print their JSON envelope there) and detaching stdin from the MCP channel
// so an interactive prompt cannot block the server forever, then restores
// process state. It returns the captured stdout; when the command failed
// without printing anything, it returns a wrapped error instead.
// TASK-009: extracted from executeCommandTool.
func mcpExecuteCaptured(root, target *cobra.Command, argv []string) ([]byte, error) {
	savedStdout := os.Stdout
	savedStdin := os.Stdin
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("mcp: stdout pipe: %w", err)
	}
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		pr.Close()
		pw.Close()
		return nil, fmt.Errorf("mcp: open %s: %w", os.DevNull, err)
	}

	var outBuf bytes.Buffer
	copyDone := make(chan struct{})
	os.Stdout = pw
	os.Stdin = devNull
	go func() {
		_, _ = io.Copy(&outBuf, pr)
		close(copyDone)
	}()

	savedJSON, savedVerbose, savedDryRun := JSONOutput, Verbose, DryRun
	savedSilenceUsage, savedSilenceErrors := target.SilenceUsage, target.SilenceErrors
	target.SilenceUsage, target.SilenceErrors = true, true

	root.SetArgs(argv)
	_, execErr := root.ExecuteC()

	// Restore process state before reading the captured output.
	os.Stdout = savedStdout
	os.Stdin = savedStdin
	pw.Close()
	devNull.Close()
	<-copyDone
	pr.Close()
	JSONOutput, Verbose, DryRun = savedJSON, savedVerbose, savedDryRun
	target.SilenceUsage, target.SilenceErrors = savedSilenceUsage, savedSilenceErrors
	root.SetArgs(nil)

	if execErr != nil && outBuf.Len() == 0 {
		return nil, fmt.Errorf("command failed: %w", execErr)
	}
	return outBuf.Bytes(), nil
}

// mcpFormatArgValue renders a positional argument value as one or more argv
// entries (arrays flatten for variadic positionals).
func mcpFormatArgValue(raw interface{}) []string {
	switch v := raw.(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, fmt.Sprint(item))
		}
		return out
	case []string:
		return append([]string{}, v...)
	default:
		return []string{fmt.Sprint(v)}
	}
}

// mcpFormatFlagValue renders a tool argument as --flag=value argv entries
// (repeated for array values, which pflag string slices accept).
func mcpFormatFlagValue(name string, raw interface{}) []string {
	prefix := "--" + name + "="
	switch v := raw.(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, prefix+fmt.Sprint(item))
		}
		return out
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, prefix+item)
		}
		return out
	default:
		return []string{prefix + fmt.Sprint(v)}
	}
}

// mcpParseCommandOutput converts captured command stdout into a tool result:
// the OutputResponse envelope's data when the command printed one (an error
// envelope becomes a tool error), the raw JSON document when it printed bare
// JSON, or the text as-is otherwise.
func mcpParseCommandOutput(out []byte) (interface{}, error) {
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return map[string]interface{}{"success": true, "message": "command completed with no output"}, nil
	}

	var envelope OutputResponse
	if err := json.Unmarshal(trimmed, &envelope); err == nil &&
		(envelope.Message != "" || envelope.Data != nil || envelope.Error != "" || envelope.DryRun) {
		if !envelope.Success {
			return nil, fmt.Errorf("%s", envelope.Error)
		}
		// Return the full envelope so agents see message, data, and dry_run.
		return envelope, nil
	}

	if json.Valid(trimmed) {
		return json.RawMessage(trimmed), nil
	}
	return string(trimmed), nil
}
