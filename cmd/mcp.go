package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Model Context Protocol (MCP) server for AI agents",
	Long: `Expose Cosmoflare operations as MCP tools so AI agents can manage Cloudflare.

The MCP server speaks JSON-RPC 2.0 over stdin/stdout, following the Model
Context Protocol specification. AI agents (Claude Code, etc.) connect to this
server to list and invoke Cloudflare management tools.

Tools come from two sources:

1. Curated tools (always available):
  cosmoflare_bucket_list    List R2 storage buckets
  cosmoflare_worker_list    List Cloudflare Workers
  cosmoflare_worker_deploy  Deploy a Worker script
  cosmoflare_dns_list       List DNS records for a zone
  cosmoflare_kv_list        List KV namespaces
  cosmoflare_zone_list      List Cloudflare zones
  cosmoflare_cache_purge    Purge cache for a zone
  cosmoflare_doctor         Run domain diagnostics

2. Generated tools — one per eligible CLI command, covering the full
  command surface (R2, DNS, D1, Pages, Queues, and every other service).
  Each tool name is the command path with underscores (for example
  kv_namespace_list, dns_record_create), its input schema is derived from
  the command's positional arguments and flags, and calling it executes the
  underlying command with --json and returns the JSON result.

Mutation gating (FAIL-CLOSED):
  Commands that mutate your Cloudflare account (create, update, delete,
  upload, put, deploy, apply, sync, purge, import, and similar — or any
  command with a --force/--confirm flag) are NOT exposed as tools by
  default. To expose them, opt in via the project config (.cosmoflare.yaml):

      mcp:
        allow_mutations: true

  Without that key, tools/list omits every mutating command and read-only
  commands work normally.

Commands:
  serve   Start the MCP server (JSON-RPC over stdio)
  tools   List available MCP tools (for debugging)

Examples:
  cosmoflare mcp serve                        # Start MCP server on stdio
  cosmoflare mcp tools                        # List available tools
  cosmoflare mcp tools --json                 # List tools as JSON

  # Claude Code MCP configuration:
  # claude mcp add cosmoflare cosmoflare mcp serve`,
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server on stdio",
	Long: `Start the Model Context Protocol server.

The server reads JSON-RPC 2.0 requests from stdin and writes responses to
stdout, one JSON object per line. It supports the standard MCP methods:

  initialize   — handshake with protocol version and capabilities
  tools/list   — enumerate available tools with JSON Schema inputs
  tools/call   — invoke a tool by name with arguments
  ping         — health check

The server exposes the curated tools plus one generated tool per eligible CLI
command (full command surface). Generated tools execute the underlying CLI
command with --json and return its JSON output.

Mutation gating is FAIL-CLOSED: mutating commands (create, update, delete,
upload, put, deploy, apply, sync, purge, import, and similar, or commands
with a --force/--confirm flag) are omitted from tools/list unless the project
config (.cosmoflare.yaml) opts in:

    mcp:
      allow_mutations: true

The server runs until stdin is closed or SIGINT/SIGTERM is received.

Environment Variables:
  CLOUDFLARE_API_TOKEN    Your Cloudflare API token (required)
  CLOUDFLARE_ACCOUNT_ID   Your Cloudflare Account ID (required)

Examples:
  cosmoflare mcp serve
  echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | cosmoflare mcp serve`,
	RunE: runMCPServe,
}

var mcpToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available MCP tools",
	Long: `List all tools that the MCP server exposes.

This is a debugging aid — it shows what an AI agent would see after
calling tools/list. Each tool has a name, description, and input schema.

Examples:
  cosmoflare mcp tools              # Human-readable table
  cosmoflare mcp tools --json       # JSON output with full schemas`,
	RunE: runMCPTools,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpServeCmd)
	mcpCmd.AddCommand(mcpToolsCmd)
}

// newMCPCLIServer builds the MCP server the CLI serves: the 8 curated tools
// plus one generated tool per eligible cobra command (BUG-035), versioned
// from the build info. Mutating commands are included only when the project
// config opts in (fail-closed).
func newMCPCLIServer() (*cosmoflare.MCPServer, mcpGenerateStats) {
	server := cosmoflare.NewMCPServer(AccountID, APIToken)
	server.SetVersion(AppVersion)
	server.RegisterDefaultTools()
	stats := registerGeneratedCLITools(server, rootCmd, loadMCPMutationAllowance("."))
	return server, stats
}

func runMCPServe(cmd *cobra.Command, args []string) error {
	server, stats := newMCPCLIServer()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if Verbose {
		printInfo("MCP server starting on stdio (%d tools: %d generated, %d mutating commands excluded; mutations %s)",
			len(server.Tools()), stats.Registered, stats.MutatingExcluded,
			map[bool]string{true: "allowed", false: "excluded (fail-closed; set mcp.allow_mutations: true in .cosmoflare.yaml)"}[loadMCPMutationAllowance(".")])
	}

	return server.Serve(ctx, os.Stdin, os.Stdout)
}

func runMCPTools(cmd *cobra.Command, args []string) error {
	server, stats := newMCPCLIServer()

	tools := server.Tools()

	if !JSONOutput {
		if stats.MutatingExcluded > 0 {
			printInfo("%d mutating command(s) excluded (fail-closed); set mcp.allow_mutations: true in .cosmoflare.yaml to expose them", stats.MutatingExcluded)
		}
	}

	if JSONOutput {
		type toolInfo struct {
			Name        string      `json:"name"`
			Description string      `json:"description"`
			InputSchema interface{} `json:"input_schema"`
		}
		out := make([]toolInfo, len(tools))
		for i, t := range tools {
			var schema interface{}
			if err := json.Unmarshal(t.InputSchema, &schema); err != nil {
				schema = string(t.InputSchema)
			}
			out[i] = toolInfo{
				Name:        t.Name,
				Description: t.Description,
				InputSchema: schema,
			}
		}
		return printSuccessJSON("MCP tools", out)
	}

	fmt.Printf("Available MCP Tools (%d)\n\n", len(tools))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "TOOL\tDESCRIPTION\n")
	fmt.Fprintf(w, "----\t-----------\n")
	for _, t := range tools {
		fmt.Fprintf(w, "%s\t%s\n", t.Name, t.Description)
	}
	return w.Flush()
}
