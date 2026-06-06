package cosmoflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestNewMCPServer(t *testing.T) {
	s := NewMCPServer("acct-123", "tok-abc")
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if len(s.Tools()) != 0 {
		t.Fatalf("expected 0 tools, got %d", len(s.Tools()))
	}
}

func TestRegisterTool(t *testing.T) {
	s := NewMCPServer("", "")
	s.RegisterTool(MCPTool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		Handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			return map[string]string{"status": "ok"}, nil
		},
	})

	tools := s.Tools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools))
	}
	if tools[0].Name != "test_tool" {
		t.Fatalf("expected name test_tool, got %s", tools[0].Name)
	}
	if tools[0].Description != "A test tool" {
		t.Fatalf("expected description 'A test tool', got %s", tools[0].Description)
	}
}

func TestRegisterToolOverwrite(t *testing.T) {
	s := NewMCPServer("", "")
	s.RegisterTool(MCPTool{Name: "dup", Description: "first"})
	s.RegisterTool(MCPTool{Name: "dup", Description: "second"})

	tools := s.Tools()
	if len(tools) != 1 {
		t.Fatalf("expected 1 tool after overwrite, got %d", len(tools))
	}
	if tools[0].Description != "second" {
		t.Fatalf("expected overwritten description 'second', got %s", tools[0].Description)
	}
}

func TestToolsSortedOrder(t *testing.T) {
	s := NewMCPServer("", "")
	s.RegisterTool(MCPTool{Name: "zeta"})
	s.RegisterTool(MCPTool{Name: "alpha"})
	s.RegisterTool(MCPTool{Name: "middle"})

	tools := s.Tools()
	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}
	if tools[0].Name != "alpha" || tools[1].Name != "middle" || tools[2].Name != "zeta" {
		t.Fatalf("tools not sorted: %s, %s, %s", tools[0].Name, tools[1].Name, tools[2].Name)
	}
}

func TestRegisterDefaultTools(t *testing.T) {
	s := NewMCPServer("acct", "tok")
	s.RegisterDefaultTools()

	tools := s.Tools()
	if len(tools) != 8 {
		t.Fatalf("expected 8 default tools, got %d", len(tools))
	}

	expected := map[string]bool{
		"cosmoflare_bucket_list":  false,
		"cosmoflare_worker_list":  false,
		"cosmoflare_worker_deploy": false,
		"cosmoflare_dns_list":     false,
		"cosmoflare_kv_list":      false,
		"cosmoflare_zone_list":    false,
		"cosmoflare_cache_purge":  false,
		"cosmoflare_doctor":       false,
	}
	for _, tool := range tools {
		if _, ok := expected[tool.Name]; !ok {
			t.Errorf("unexpected tool: %s", tool.Name)
		}
		expected[tool.Name] = true
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing expected tool: %s", name)
		}
	}
}

func TestHandleRequestParseError(t *testing.T) {
	s := NewMCPServer("", "")
	resp := s.HandleRequest(context.Background(), []byte("not json"))
	var r jsonRPCResponse
	if err := json.Unmarshal(resp, &r); err != nil {
		t.Fatal(err)
	}
	if r.Error == nil {
		t.Fatal("expected error response")
	}
	if r.Error.Code != jsonRPCParseError {
		t.Fatalf("expected parse error code %d, got %d", jsonRPCParseError, r.Error.Code)
	}
}

func TestHandleRequestInvalidVersion(t *testing.T) {
	s := NewMCPServer("", "")
	req := `{"jsonrpc":"1.0","id":1,"method":"ping"}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	json.Unmarshal(resp, &r)
	if r.Error == nil || r.Error.Code != jsonRPCInvalidRequest {
		t.Fatal("expected invalid request error for wrong jsonrpc version")
	}
}

func TestHandleRequestUnknownMethod(t *testing.T) {
	s := NewMCPServer("", "")
	req := `{"jsonrpc":"2.0","id":1,"method":"unknown/method"}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	json.Unmarshal(resp, &r)
	if r.Error == nil || r.Error.Code != jsonRPCMethodNotFound {
		t.Fatal("expected method not found error")
	}
}

func TestHandleInitialize(t *testing.T) {
	s := NewMCPServer("", "")
	req := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	json.Unmarshal(resp, &r)
	if r.Error != nil {
		t.Fatalf("unexpected error: %s", r.Error.Message)
	}

	var result map[string]interface{}
	json.Unmarshal(r.Result, &result)
	if result["protocolVersion"] != "2024-11-05" {
		t.Fatalf("expected protocolVersion 2024-11-05, got %v", result["protocolVersion"])
	}
	serverInfo := result["serverInfo"].(map[string]interface{})
	if serverInfo["name"] != "cosmoflare" {
		t.Fatalf("expected server name cosmoflare, got %v", serverInfo["name"])
	}
}

func TestHandlePing(t *testing.T) {
	s := NewMCPServer("", "")
	req := `{"jsonrpc":"2.0","id":42,"method":"ping"}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	json.Unmarshal(resp, &r)
	if r.Error != nil {
		t.Fatalf("unexpected error: %s", r.Error.Message)
	}
	// ID should be preserved
	if r.ID != float64(42) {
		t.Fatalf("expected id 42, got %v", r.ID)
	}
}

func TestHandleToolsList(t *testing.T) {
	s := NewMCPServer("", "")
	s.RegisterTool(MCPTool{
		Name:        "my_tool",
		Description: "does stuff",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"x":{"type":"string"}}}`),
	})

	req := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	json.Unmarshal(resp, &r)
	if r.Error != nil {
		t.Fatalf("unexpected error: %s", r.Error.Message)
	}

	var result toolsListResult
	json.Unmarshal(r.Result, &result)
	if len(result.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result.Tools))
	}
	if result.Tools[0].Name != "my_tool" {
		t.Fatalf("expected my_tool, got %s", result.Tools[0].Name)
	}
}

func TestHandleToolsCallSuccess(t *testing.T) {
	s := NewMCPServer("", "")
	s.RegisterTool(MCPTool{
		Name:        "echo",
		Description: "echoes input",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		Handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			return map[string]string{"echo": string(params)}, nil
		},
	})

	req := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"msg":"hello"}}}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	json.Unmarshal(resp, &r)
	if r.Error != nil {
		t.Fatalf("unexpected error: %s", r.Error.Message)
	}

	var result toolsCallResult
	json.Unmarshal(r.Result, &result)
	if result.IsError {
		t.Fatal("expected success, got isError=true")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(result.Content))
	}
	if result.Content[0].Type != "text" {
		t.Fatalf("expected type text, got %s", result.Content[0].Type)
	}
	if !strings.Contains(result.Content[0].Text, "hello") {
		t.Fatalf("expected content to contain 'hello', got %s", result.Content[0].Text)
	}
}

func TestHandleToolsCallError(t *testing.T) {
	s := NewMCPServer("", "")
	s.RegisterTool(MCPTool{
		Name:    "fail_tool",
		Handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			return nil, fmt.Errorf("something went wrong")
		},
	})

	req := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"fail_tool","arguments":{}}}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	json.Unmarshal(resp, &r)
	// Tool errors are returned as success with isError=true per MCP spec.
	if r.Error != nil {
		t.Fatalf("unexpected JSON-RPC error: %s", r.Error.Message)
	}

	var result toolsCallResult
	json.Unmarshal(r.Result, &result)
	if !result.IsError {
		t.Fatal("expected isError=true for tool failure")
	}
	if !strings.Contains(result.Content[0].Text, "something went wrong") {
		t.Fatalf("expected error message in content, got %s", result.Content[0].Text)
	}
}

func TestHandleToolsCallUnknownTool(t *testing.T) {
	s := NewMCPServer("", "")
	req := `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"nonexistent","arguments":{}}}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	json.Unmarshal(resp, &r)
	if r.Error == nil {
		t.Fatal("expected error for unknown tool")
	}
	if r.Error.Code != jsonRPCMethodNotFound {
		t.Fatalf("expected method not found code, got %d", r.Error.Code)
	}
}

func TestServe(t *testing.T) {
	s := NewMCPServer("", "")
	s.RegisterTool(MCPTool{
		Name: "greet",
		Handler: func(ctx context.Context, params json.RawMessage) (interface{}, error) {
			return map[string]string{"greeting": "hello"}, nil
		},
	})

	// Two requests, newline-delimited.
	input := `{"jsonrpc":"2.0","id":1,"method":"ping"}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"greet","arguments":{}}}
`
	var out bytes.Buffer
	err := s.Serve(context.Background(), strings.NewReader(input), &out)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 response lines, got %d: %q", len(lines), out.String())
	}

	// Verify first response (ping).
	var r1 jsonRPCResponse
	json.Unmarshal([]byte(lines[0]), &r1)
	if r1.Error != nil {
		t.Fatalf("ping error: %s", r1.Error.Message)
	}

	// Verify second response (tools/call).
	var r2 jsonRPCResponse
	json.Unmarshal([]byte(lines[1]), &r2)
	if r2.Error != nil {
		t.Fatalf("greet error: %s", r2.Error.Message)
	}
	var result toolsCallResult
	json.Unmarshal(r2.Result, &result)
	if result.IsError {
		t.Fatal("expected greet success")
	}
	if !strings.Contains(result.Content[0].Text, "hello") {
		t.Fatalf("expected greeting, got %s", result.Content[0].Text)
	}
}

func TestHandleToolsCallInvalidParams(t *testing.T) {
	s := NewMCPServer("", "")
	req := `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":"not an object"}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	json.Unmarshal(resp, &r)
	if r.Error == nil || r.Error.Code != jsonRPCInvalidParams {
		t.Fatal("expected invalid params error")
	}
}

func TestDoctorToolInputSchema(t *testing.T) {
	s := NewMCPServer("", "")
	s.RegisterDefaultTools()

	tools := s.Tools()
	var doctorTool *MCPTool
	for i, tool := range tools {
		if tool.Name == "cosmoflare_doctor" {
			doctorTool = &tools[i]
			break
		}
	}
	if doctorTool == nil {
		t.Fatal("cosmoflare_doctor tool not found")
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(doctorTool.InputSchema, &schema); err != nil {
		t.Fatalf("invalid input schema JSON: %v", err)
	}
	if schema["type"] != "object" {
		t.Fatalf("expected type object, got %v", schema["type"])
	}
	required, ok := schema["required"].([]interface{})
	if !ok || len(required) != 1 || required[0] != "domain" {
		t.Fatalf("expected required [domain], got %v", schema["required"])
	}
}
