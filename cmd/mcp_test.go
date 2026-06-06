package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	cosmoflare "github.com/CosmoLabs-org/cosmoflare/pkg/cosmoflare"
)

func findMCPCmd() *cobra.Command {
	for _, c := range rootCmd.Commands() {
		if c.Use == "mcp" {
			return c
		}
	}
	return nil
}

func TestMCPCmdExists(t *testing.T) {
	if findMCPCmd() == nil {
		t.Fatal("expected 'mcp' subcommand on root")
	}
}

func TestMCPServeSubcommand(t *testing.T) {
	mcpParent := findMCPCmd()
	if mcpParent == nil {
		t.Fatal("mcp command not found")
	}

	subNames := make(map[string]bool)
	for _, c := range mcpParent.Commands() {
		subNames[c.Use] = true
	}
	if !subNames["serve"] {
		t.Fatal("expected 'serve' subcommand under mcp")
	}
	if !subNames["tools"] {
		t.Fatal("expected 'tools' subcommand under mcp")
	}
}

func TestMCPToolsCount(t *testing.T) {
	srv := cosmoflare.NewMCPServer("", "")
	srv.RegisterDefaultTools()
	tools := srv.Tools()
	if len(tools) != 8 {
		t.Fatalf("expected 8 MCP tools, got %d", len(tools))
	}
}

func TestMCPToolNames(t *testing.T) {
	srv := cosmoflare.NewMCPServer("", "")
	srv.RegisterDefaultTools()

	expected := []string{
		"cosmoflare_bucket_list",
		"cosmoflare_cache_purge",
		"cosmoflare_dns_list",
		"cosmoflare_doctor",
		"cosmoflare_kv_list",
		"cosmoflare_worker_deploy",
		"cosmoflare_worker_list",
		"cosmoflare_zone_list",
	}

	tools := srv.Tools()
	if len(tools) != len(expected) {
		t.Fatalf("expected %d tools, got %d", len(expected), len(tools))
	}
	for i, tool := range tools {
		if tool.Name != expected[i] {
			t.Errorf("tool[%d]: expected %s, got %s", i, expected[i], tool.Name)
		}
	}
}

func TestMCPHelpText(t *testing.T) {
	mcpParent := findMCPCmd()
	if mcpParent == nil {
		t.Fatal("mcp command not found")
	}

	long := mcpParent.Long
	if !strings.Contains(long, "MCP") {
		t.Fatal("expected MCP in long description")
	}
	if !strings.Contains(long, "cosmoflare_bucket_list") {
		t.Fatal("expected tool names in help text")
	}
}

func TestMCPServeHelpText(t *testing.T) {
	mcpParent := findMCPCmd()
	if mcpParent == nil {
		t.Fatal("mcp command not found")
	}

	var serveCmd *cobra.Command
	for _, c := range mcpParent.Commands() {
		if c.Use == "serve" {
			serveCmd = c
			break
		}
	}
	if serveCmd == nil {
		t.Fatal("serve subcommand not found")
	}
	if !strings.Contains(serveCmd.Long, "JSON-RPC") {
		t.Fatal("expected JSON-RPC in serve help text")
	}
}

func TestMCPToolsHaveDescriptions(t *testing.T) {
	srv := cosmoflare.NewMCPServer("", "")
	srv.RegisterDefaultTools()

	for _, tool := range srv.Tools() {
		if tool.Description == "" {
			t.Errorf("tool %s has empty description", tool.Name)
		}
	}
}

func TestMCPToolsHaveSchemas(t *testing.T) {
	srv := cosmoflare.NewMCPServer("", "")
	srv.RegisterDefaultTools()

	for _, tool := range srv.Tools() {
		if len(tool.InputSchema) == 0 {
			t.Errorf("tool %s has empty input schema", tool.Name)
		}
	}
}
