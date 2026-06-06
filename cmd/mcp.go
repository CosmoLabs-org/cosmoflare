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

Available MCP tools:
  cosmoflare_bucket_list    List R2 storage buckets
  cosmoflare_worker_list    List Cloudflare Workers
  cosmoflare_worker_deploy  Deploy a Worker script
  cosmoflare_dns_list       List DNS records for a zone
  cosmoflare_kv_list        List KV namespaces
  cosmoflare_zone_list      List Cloudflare zones
  cosmoflare_cache_purge    Purge cache for a zone
  cosmoflare_doctor         Run domain diagnostics

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

func runMCPServe(cmd *cobra.Command, args []string) error {
	server := cosmoflare.NewMCPServer(AccountID, APIToken)
	server.RegisterDefaultTools()

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
		printInfo("MCP server starting on stdio (%d tools registered)", len(server.Tools()))
	}

	return server.Serve(ctx, os.Stdin, os.Stdout)
}

func runMCPTools(cmd *cobra.Command, args []string) error {
	server := cosmoflare.NewMCPServer("", "")
	server.RegisterDefaultTools()

	tools := server.Tools()

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
