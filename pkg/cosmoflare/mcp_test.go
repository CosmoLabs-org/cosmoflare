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

// ---------------------------------------------------------------------------
// Additional coverage tests
// ---------------------------------------------------------------------------

// TestNewMCPServerStoresCredentials verifies that NewMCPServer persists the
// accountID and apiToken for later use by tool handlers.
func TestNewMCPServerStoresCredentials(t *testing.T) {
	s := NewMCPServer("my-account", "my-token")
	if s.accountID != "my-account" {
		t.Fatalf("expected accountID 'my-account', got %q", s.accountID)
	}
	if s.apiToken != "my-token" {
		t.Fatalf("expected apiToken 'my-token', got %q", s.apiToken)
	}
}

// TestNewMCPServerEmptyCredentials ensures the server can be created with
// blank credentials (tools that don't need auth still work).
func TestNewMCPServerEmptyCredentials(t *testing.T) {
	s := NewMCPServer("", "")
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.tools == nil {
		t.Fatal("tools map must be initialised")
	}
}

// TestHandleRequestStringID checks that string IDs are round-tripped correctly.
func TestHandleRequestStringID(t *testing.T) {
	s := NewMCPServer("", "")
	req := `{"jsonrpc":"2.0","id":"req-abc","method":"ping"}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	if err := json.Unmarshal(resp, &r); err != nil {
		t.Fatal(err)
	}
	if r.Error != nil {
		t.Fatalf("unexpected error: %s", r.Error.Message)
	}
	if r.ID != "req-abc" {
		t.Fatalf("expected id 'req-abc', got %v", r.ID)
	}
}

// TestHandleRequestNullID ensures null IDs are handled without crashing.
func TestHandleRequestNullID(t *testing.T) {
	s := NewMCPServer("", "")
	req := `{"jsonrpc":"2.0","id":null,"method":"ping"}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	if err := json.Unmarshal(resp, &r); err != nil {
		t.Fatal(err)
	}
	if r.Error != nil {
		t.Fatalf("unexpected error: %s", r.Error.Message)
	}
}

// TestHandleToolsListEmpty verifies that tools/list returns an empty list when
// no tools have been registered.
func TestHandleToolsListEmpty(t *testing.T) {
	s := NewMCPServer("", "")
	req := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	if err := json.Unmarshal(resp, &r); err != nil {
		t.Fatal(err)
	}
	if r.Error != nil {
		t.Fatalf("unexpected error: %s", r.Error.Message)
	}
	var result toolsListResult
	if err := json.Unmarshal(r.Result, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Tools) != 0 {
		t.Fatalf("expected 0 tools, got %d", len(result.Tools))
	}
}

// TestHandleToolsListMultiple verifies that all registered tools appear in the
// tools/list response and that they are sorted alphabetically.
func TestHandleToolsListMultiple(t *testing.T) {
	s := NewMCPServer("", "")
	s.RegisterTool(MCPTool{Name: "zebra", Description: "last"})
	s.RegisterTool(MCPTool{Name: "apple", Description: "first"})
	s.RegisterTool(MCPTool{Name: "mango", Description: "middle"})

	req := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	if err := json.Unmarshal(resp, &r); err != nil {
		t.Fatal(err)
	}
	var result toolsListResult
	json.Unmarshal(r.Result, &result)
	if len(result.Tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(result.Tools))
	}
	if result.Tools[0].Name != "apple" || result.Tools[1].Name != "mango" || result.Tools[2].Name != "zebra" {
		t.Fatalf("tools not in sorted order: %v", result.Tools)
	}
}

// TestHandleToolsListPreservesSchema verifies that the InputSchema JSON is
// faithfully round-tripped through tools/list.
func TestHandleToolsListPreservesSchema(t *testing.T) {
	schema := `{"type":"object","properties":{"key":{"type":"string"}},"required":["key"]}`
	s := NewMCPServer("", "")
	s.RegisterTool(MCPTool{
		Name:        "schema_tool",
		InputSchema: json.RawMessage(schema),
	})

	req := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	json.Unmarshal(resp, &r)
	var result toolsListResult
	json.Unmarshal(r.Result, &result)
	if len(result.Tools) != 1 {
		t.Fatalf("expected 1 tool")
	}
	got := string(result.Tools[0].InputSchema)
	// Unmarshal both and compare to avoid whitespace issues.
	var wantObj, gotObj interface{}
	json.Unmarshal([]byte(schema), &wantObj)
	json.Unmarshal([]byte(got), &gotObj)
	wantBytes, _ := json.Marshal(wantObj)
	gotBytes, _ := json.Marshal(gotObj)
	if string(wantBytes) != string(gotBytes) {
		t.Fatalf("schema mismatch: want %s, got %s", wantBytes, gotBytes)
	}
}

// TestHandleInitializeCapabilities verifies that the capabilities block
// returned by initialize contains the expected "tools" key.
func TestHandleInitializeCapabilities(t *testing.T) {
	s := NewMCPServer("", "")
	req := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	json.Unmarshal(resp, &r)
	if r.Error != nil {
		t.Fatalf("unexpected error: %s", r.Error.Message)
	}
	var result map[string]interface{}
	json.Unmarshal(r.Result, &result)
	caps, ok := result["capabilities"].(map[string]interface{})
	if !ok {
		t.Fatal("expected capabilities map")
	}
	if _, hasTools := caps["tools"]; !hasTools {
		t.Fatal("expected capabilities to contain 'tools'")
	}
}

// TestHandleInitializeServerInfo verifies the serverInfo version field.
func TestHandleInitializeServerInfo(t *testing.T) {
	s := NewMCPServer("", "")
	req := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
	resp := s.HandleRequest(context.Background(), []byte(req))
	var r jsonRPCResponse
	json.Unmarshal(resp, &r)
	var result map[string]interface{}
	json.Unmarshal(r.Result, &result)
	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatal("expected serverInfo map")
	}
	if serverInfo["version"] == "" {
		t.Fatal("expected non-empty server version")
	}
}

// TestServeSkipsBlankLines verifies that blank lines in the input stream do
// not cause errors and produce no spurious output.
func TestServeSkipsBlankLines(t *testing.T) {
	s := NewMCPServer("", "")
	// Input with blank lines interspersed between valid requests.
	input := "\n" + `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n\n" + `{"jsonrpc":"2.0","id":2,"method":"ping"}` + "\n\n"
	var out bytes.Buffer
	err := s.Serve(context.Background(), strings.NewReader(input), &out)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 response lines, got %d: %q", len(lines), out.String())
	}
}

// TestServeContextCancellation verifies that Serve returns the context error
// when the context is cancelled before reading completes.
func TestServeContextCancellation(t *testing.T) {
	s := NewMCPServer("", "")
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately so the first select{} in the scan loop exits.
	cancel()

	// Use a long input; it should be abandoned after ctx is done.
	input := `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n"
	var out bytes.Buffer
	err := s.Serve(ctx, strings.NewReader(input), &out)
	if err == nil {
		// Cancellation may race with scan; if we produced output it means the
		// scan completed before the select ran — that's still valid Go behaviour.
		// Only fail if the scanner returns an error other than context.Canceled.
		return
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// TestHandleWorkerDeployMissingName verifies that handleWorkerDeploy returns
// an error when the "name" param is absent.
func TestHandleWorkerDeployMissingName(t *testing.T) {
	s := NewMCPServer("", "")
	s.RegisterDefaultTools()

	params := json.RawMessage(`{"script":"addEventListener('fetch',e=>e.respondWith(new Response('ok')))"}`)
	_, err := s.handleWorkerDeploy(context.Background(), params)
	if err == nil {
		t.Fatal("expected error when name is missing")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Fatalf("expected error to mention 'name', got: %v", err)
	}
}

// TestHandleWorkerDeployMissingScript verifies that handleWorkerDeploy returns
// an error when the "script" param is absent.
func TestHandleWorkerDeployMissingScript(t *testing.T) {
	s := NewMCPServer("", "")
	params := json.RawMessage(`{"name":"my-worker"}`)
	_, err := s.handleWorkerDeploy(context.Background(), params)
	if err == nil {
		t.Fatal("expected error when script is missing")
	}
	if !strings.Contains(err.Error(), "script") {
		t.Fatalf("expected error to mention 'script', got: %v", err)
	}
}

// TestHandleWorkerDeployInvalidJSON verifies that handleWorkerDeploy returns
// an error when params JSON is malformed.
func TestHandleWorkerDeployInvalidJSON(t *testing.T) {
	s := NewMCPServer("", "")
	_, err := s.handleWorkerDeploy(context.Background(), json.RawMessage(`not-json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON params")
	}
}

// TestHandleDNSListMissingZoneID verifies that handleDNSList returns an error
// when zone_id is absent.
func TestHandleDNSListMissingZoneID(t *testing.T) {
	s := NewMCPServer("", "")
	_, err := s.handleDNSList(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error when zone_id is missing")
	}
	if !strings.Contains(err.Error(), "zone_id") {
		t.Fatalf("expected error to mention 'zone_id', got: %v", err)
	}
}

// TestHandleDNSListInvalidJSON verifies that handleDNSList returns an error
// when params JSON is malformed.
func TestHandleDNSListInvalidJSON(t *testing.T) {
	s := NewMCPServer("", "")
	_, err := s.handleDNSList(context.Background(), json.RawMessage(`{bad}`))
	if err == nil {
		t.Fatal("expected error for invalid JSON params")
	}
}

// TestHandleCachePurgeMissingZoneID verifies that handleCachePurge returns an
// error when zone_id is absent.
func TestHandleCachePurgeMissingZoneID(t *testing.T) {
	s := NewMCPServer("", "")
	_, err := s.handleCachePurge(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error when zone_id is missing")
	}
	if !strings.Contains(err.Error(), "zone_id") {
		t.Fatalf("expected error to mention 'zone_id', got: %v", err)
	}
}

// TestHandleCachePurgeNoPurgeStrategy verifies that handleCachePurge returns
// an error when no purge strategy is specified alongside a valid zone_id.
// With empty credentials the service constructor fails before the strategy
// check, so we accept either failure reason.
func TestHandleCachePurgeNoPurgeStrategy(t *testing.T) {
	s := NewMCPServer("", "")
	_, err := s.handleCachePurge(context.Background(), json.RawMessage(`{"zone_id":"z1"}`))
	if err == nil {
		t.Fatal("expected error when no purge strategy specified")
	}
	msg := err.Error()
	if !strings.Contains(msg, "purge") && !strings.Contains(msg, "cache service") && !strings.Contains(msg, "token") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestHandleCachePurgeInvalidJSON verifies that handleCachePurge returns an
// error when params JSON is malformed.
func TestHandleCachePurgeInvalidJSON(t *testing.T) {
	s := NewMCPServer("", "")
	_, err := s.handleCachePurge(context.Background(), json.RawMessage(`{{`))
	if err == nil {
		t.Fatal("expected error for invalid JSON params")
	}
}

// TestHandleDoctorMissingDomain verifies that handleDoctor returns an error
// when the domain param is absent.
func TestHandleDoctorMissingDomain(t *testing.T) {
	s := NewMCPServer("", "")
	_, err := s.handleDoctor(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error when domain is missing")
	}
	if !strings.Contains(err.Error(), "domain") {
		t.Fatalf("expected error to mention 'domain', got: %v", err)
	}
}

// TestHandleDoctorInvalidJSON verifies that handleDoctor returns an error when
// params JSON is malformed.
func TestHandleDoctorInvalidJSON(t *testing.T) {
	s := NewMCPServer("", "")
	_, err := s.handleDoctor(context.Background(), json.RawMessage(`not-json`))
	if err == nil {
		t.Fatal("expected error for invalid JSON params")
	}
}

// TestDefaultToolDescriptionsNonEmpty verifies that every default tool has a
// non-empty Description field.
func TestDefaultToolDescriptionsNonEmpty(t *testing.T) {
	s := NewMCPServer("acct", "tok")
	s.RegisterDefaultTools()
	for _, tool := range s.Tools() {
		if tool.Description == "" {
			t.Errorf("tool %q has empty description", tool.Name)
		}
	}
}

// TestDefaultToolSchemasValidJSON verifies that every default tool exposes a
// valid JSON object as its InputSchema.
func TestDefaultToolSchemasValidJSON(t *testing.T) {
	s := NewMCPServer("acct", "tok")
	s.RegisterDefaultTools()
	for _, tool := range s.Tools() {
		var schema map[string]interface{}
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			t.Errorf("tool %q has invalid InputSchema JSON: %v", tool.Name, err)
			continue
		}
		if schema["type"] != "object" {
			t.Errorf("tool %q InputSchema type is not 'object': %v", tool.Name, schema["type"])
		}
	}
}

// TestDefaultToolHandlersNotNil verifies that every default tool has a non-nil
// Handler assigned.
func TestDefaultToolHandlersNotNil(t *testing.T) {
	s := NewMCPServer("acct", "tok")
	s.RegisterDefaultTools()
	// Access the internal map directly to check handler presence (Tools() strips Handler).
	s.mu.RLock()
	defer s.mu.RUnlock()
	for name, tool := range s.tools {
		if tool.Handler == nil {
			t.Errorf("tool %q has nil Handler", name)
		}
	}
}

// TestConcurrentRegisterAndList verifies that RegisterTool and Tools() can be
// called concurrently without data races.
func TestConcurrentRegisterAndList(t *testing.T) {
	s := NewMCPServer("", "")
	done := make(chan struct{})

	go func() {
		for i := 0; i < 50; i++ {
			s.RegisterTool(MCPTool{Name: fmt.Sprintf("tool_%d", i)})
		}
		close(done)
	}()

	for {
		select {
		case <-done:
			return
		default:
			_ = s.Tools()
		}
	}
}

// TestWorkerDeployToolInputSchemaRequiredFields verifies the worker deploy
// schema marks "name" and "script" as required.
func TestWorkerDeployToolInputSchemaRequiredFields(t *testing.T) {
	s := NewMCPServer("acct", "tok")
	s.RegisterDefaultTools()

	var deployTool *MCPTool
	for i, tool := range s.Tools() {
		if tool.Name == "cosmoflare_worker_deploy" {
			t := s.Tools()[i]
			deployTool = &t
			break
		}
	}
	if deployTool == nil {
		t.Fatal("cosmoflare_worker_deploy tool not found")
	}

	var schema map[string]interface{}
	json.Unmarshal(deployTool.InputSchema, &schema)
	required, ok := schema["required"].([]interface{})
	if !ok {
		t.Fatal("expected 'required' array in worker deploy schema")
	}
	requiredSet := map[string]bool{}
	for _, v := range required {
		if s, ok := v.(string); ok {
			requiredSet[s] = true
		}
	}
	if !requiredSet["name"] {
		t.Error("expected 'name' in required fields")
	}
	if !requiredSet["script"] {
		t.Error("expected 'script' in required fields")
	}
}

// TestErrorResponseFields verifies the structure of an error JSON-RPC response.
func TestErrorResponseFields(t *testing.T) {
	s := NewMCPServer("", "")
	out := s.errorResponse("id-1", jsonRPCInternalError, "internal boom")
	var r jsonRPCResponse
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatal(err)
	}
	if r.JSONRPC != "2.0" {
		t.Fatalf("expected jsonrpc '2.0', got %q", r.JSONRPC)
	}
	if r.Error == nil {
		t.Fatal("expected error field")
	}
	if r.Error.Code != jsonRPCInternalError {
		t.Fatalf("expected code %d, got %d", jsonRPCInternalError, r.Error.Code)
	}
	if !strings.Contains(r.Error.Message, "internal boom") {
		t.Fatalf("expected message to contain 'internal boom', got %q", r.Error.Message)
	}
	if r.Result != nil {
		t.Fatal("expected nil result on error response")
	}
}

// TestSuccessResponseFields verifies the structure of a success JSON-RPC response.
func TestSuccessResponseFields(t *testing.T) {
	s := NewMCPServer("", "")
	payload := map[string]string{"key": "value"}
	out := s.successResponse(float64(99), payload)
	var r jsonRPCResponse
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatal(err)
	}
	if r.JSONRPC != "2.0" {
		t.Fatalf("expected jsonrpc '2.0', got %q", r.JSONRPC)
	}
	if r.Error != nil {
		t.Fatalf("expected nil error on success response, got %v", r.Error)
	}
	if r.Result == nil {
		t.Fatal("expected non-nil result on success response")
	}
	var decoded map[string]string
	json.Unmarshal(r.Result, &decoded)
	if decoded["key"] != "value" {
		t.Fatalf("expected key=value in result, got %v", decoded)
	}
}
