package cosmoflare

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

// MCPServer implements a Model Context Protocol (MCP) tool server.
// It registers tools with JSON Schema input definitions and dispatches
// JSON-RPC 2.0 requests to the appropriate handler.
type MCPServer struct {
	mu    sync.RWMutex
	tools map[string]MCPTool

	// Service factories — called lazily on first tool invocation.
	// Each returns the concrete service using the caller's credentials.
	accountID string
	apiToken  string
}

// MCPTool describes a single tool exposed over MCP.
type MCPTool struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	InputSchema json.RawMessage   `json:"inputSchema"`
	Handler     MCPToolHandler    `json:"-"`
}

// MCPToolHandler processes a tool call and returns a result or error.
type MCPToolHandler func(ctx context.Context, params json.RawMessage) (interface{}, error)

// jsonRPCRequest is a minimal JSON-RPC 2.0 request envelope.
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonRPCResponse is a minimal JSON-RPC 2.0 response envelope.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCError is the error object inside a JSON-RPC 2.0 response.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCP protocol method constants.
const (
	mcpMethodInitialize   = "initialize"
	mcpMethodToolsList    = "tools/list"
	mcpMethodToolsCall    = "tools/call"
	mcpMethodPing         = "ping"
)

// JSON-RPC error codes.
const (
	jsonRPCParseError     = -32700
	jsonRPCInvalidRequest = -32600
	jsonRPCMethodNotFound = -32601
	jsonRPCInvalidParams  = -32602
	jsonRPCInternalError  = -32603
)

// toolsListResult is the response payload for tools/list.
type toolsListResult struct {
	Tools []mcpToolInfo `json:"tools"`
}

// mcpToolInfo is the wire format for a tool in tools/list.
type mcpToolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// toolsCallParams is the params payload for tools/call.
type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// toolsCallResult is the response payload for tools/call.
type toolsCallResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// mcpContent represents a single content block returned by a tool.
type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewMCPServer creates an MCPServer. Pass empty strings if credentials
// will be supplied per-request or if only listing tools.
func NewMCPServer(accountID, apiToken string) *MCPServer {
	return &MCPServer{
		tools:     make(map[string]MCPTool),
		accountID: accountID,
		apiToken:  apiToken,
	}
}

// RegisterTool adds a tool to the server. Duplicate names overwrite.
func (s *MCPServer) RegisterTool(tool MCPTool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[tool.Name] = tool
}

// Tools returns all registered tools in deterministic order (sorted by name).
func (s *MCPServer) Tools() []MCPTool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]MCPTool, 0, len(s.tools))
	for _, t := range s.tools {
		out = append(out, t)
	}
	// Sort by name for deterministic output.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[i].Name > out[j].Name {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// HandleRequest processes a single JSON-RPC 2.0 request and returns a response.
func (s *MCPServer) HandleRequest(ctx context.Context, raw []byte) []byte {
	var req jsonRPCRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return s.errorResponse(nil, jsonRPCParseError, "parse error: "+err.Error())
	}
	if req.JSONRPC != "2.0" {
		return s.errorResponse(req.ID, jsonRPCInvalidRequest, "jsonrpc must be \"2.0\"")
	}

	switch req.Method {
	case mcpMethodInitialize:
		return s.handleInitialize(req)
	case mcpMethodToolsList:
		return s.handleToolsList(req)
	case mcpMethodToolsCall:
		return s.handleToolsCall(ctx, req)
	case mcpMethodPing:
		return s.handlePing(req)
	default:
		return s.errorResponse(req.ID, jsonRPCMethodNotFound, "unknown method: "+req.Method)
	}
}

// Serve reads newline-delimited JSON-RPC requests from r, dispatches each,
// and writes responses to w. It blocks until r is exhausted or ctx is cancelled.
func (s *MCPServer) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	// Allow up to 1 MB per line (generous for JSON-RPC).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		resp := s.HandleRequest(ctx, line)
		resp = append(resp, '\n')
		if _, err := w.Write(resp); err != nil {
			return fmt.Errorf("mcp: write response: %w", err)
		}
	}
	return scanner.Err()
}

// --- internal handlers ---

func (s *MCPServer) handleInitialize(req jsonRPCRequest) []byte {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "cosmoflare",
			"version": "1.0.0",
		},
	}
	return s.successResponse(req.ID, result)
}

func (s *MCPServer) handlePing(req jsonRPCRequest) []byte {
	return s.successResponse(req.ID, map[string]interface{}{})
}

func (s *MCPServer) handleToolsList(req jsonRPCRequest) []byte {
	tools := s.Tools()
	infos := make([]mcpToolInfo, len(tools))
	for i, t := range tools {
		infos[i] = mcpToolInfo{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		}
	}
	return s.successResponse(req.ID, toolsListResult{Tools: infos})
}

func (s *MCPServer) handleToolsCall(ctx context.Context, req jsonRPCRequest) []byte {
	var params toolsCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, jsonRPCInvalidParams, "invalid params: "+err.Error())
	}

	s.mu.RLock()
	tool, ok := s.tools[params.Name]
	s.mu.RUnlock()
	if !ok {
		return s.errorResponse(req.ID, jsonRPCMethodNotFound, "unknown tool: "+params.Name)
	}

	result, err := tool.Handler(ctx, params.Arguments)
	if err != nil {
		errResult := toolsCallResult{
			Content: []mcpContent{{Type: "text", Text: err.Error()}},
			IsError: true,
		}
		return s.successResponse(req.ID, errResult)
	}

	text, err := json.Marshal(result)
	if err != nil {
		return s.errorResponse(req.ID, jsonRPCInternalError, "marshal result: "+err.Error())
	}

	callResult := toolsCallResult{
		Content: []mcpContent{{Type: "text", Text: string(text)}},
	}
	return s.successResponse(req.ID, callResult)
}

// --- response helpers ---

func (s *MCPServer) successResponse(id interface{}, result interface{}) []byte {
	raw, _ := json.Marshal(result)
	resp := jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: raw}
	out, _ := json.Marshal(resp)
	return out
}

func (s *MCPServer) errorResponse(id interface{}, code int, message string) []byte {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonRPCError{Code: code, Message: message},
	}
	out, _ := json.Marshal(resp)
	return out
}

// ---------------------------------------------------------------------------
// Default Cosmoflare tools
// ---------------------------------------------------------------------------

// RegisterDefaultTools adds the standard set of Cosmoflare MCP tools.
func (s *MCPServer) RegisterDefaultTools() {
	s.RegisterTool(MCPTool{
		Name:        "cosmoflare_bucket_list",
		Description: "List all R2 storage buckets in the Cloudflare account",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		Handler:     s.handleBucketList,
	})

	s.RegisterTool(MCPTool{
		Name:        "cosmoflare_worker_list",
		Description: "List all Cloudflare Workers in the account",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		Handler:     s.handleWorkerList,
	})

	s.RegisterTool(MCPTool{
		Name:        "cosmoflare_worker_deploy",
		Description: "Deploy a Cloudflare Worker script",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"name":{"type":"string","description":"Worker name"},"script":{"type":"string","description":"JavaScript source code"},"compatibility_date":{"type":"string","description":"Compatibility date (YYYY-MM-DD)"}},"required":["name","script"]}`),
		Handler:     s.handleWorkerDeploy,
	})

	s.RegisterTool(MCPTool{
		Name:        "cosmoflare_dns_list",
		Description: "List DNS records for a Cloudflare zone",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"zone_id":{"type":"string","description":"Cloudflare Zone ID"}},"required":["zone_id"]}`),
		Handler:     s.handleDNSList,
	})

	s.RegisterTool(MCPTool{
		Name:        "cosmoflare_kv_list",
		Description: "List KV namespaces in the Cloudflare account",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		Handler:     s.handleKVList,
	})

	s.RegisterTool(MCPTool{
		Name:        "cosmoflare_zone_list",
		Description: "List all Cloudflare zones in the account",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		Handler:     s.handleZoneList,
	})

	s.RegisterTool(MCPTool{
		Name:        "cosmoflare_cache_purge",
		Description: "Purge cache for a Cloudflare zone (all files, or by URLs/tags/hosts)",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"zone_id":{"type":"string","description":"Cloudflare Zone ID"},"purge_everything":{"type":"boolean","description":"Purge all cached files"},"files":{"type":"array","items":{"type":"string"},"description":"Specific URLs to purge"},"tags":{"type":"array","items":{"type":"string"},"description":"Cache tags to purge"},"hosts":{"type":"array","items":{"type":"string"},"description":"Hostnames to purge"}},"required":["zone_id"]}`),
		Handler:     s.handleCachePurge,
	})

	s.RegisterTool(MCPTool{
		Name:        "cosmoflare_doctor",
		Description: "Run diagnostic health checks on a domain (DNS propagation, SSL, HTTP, nameservers)",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"domain":{"type":"string","description":"Domain to diagnose"}},"required":["domain"]}`),
		Handler:     s.handleDoctor,
	})
}

// --- tool handlers ---

func (s *MCPServer) handleBucketList(ctx context.Context, _ json.RawMessage) (interface{}, error) {
	client, err := NewClient(WithAccountID(s.accountID), WithAPIToken(s.apiToken))
	if err != nil {
		return nil, fmt.Errorf("failed to create R2 client: %w", err)
	}
	return client.ListBuckets(ctx)
}

func (s *MCPServer) handleWorkerList(ctx context.Context, _ json.RawMessage) (interface{}, error) {
	svc, err := NewWorkerServiceFromCreds(s.accountID, s.apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker service: %w", err)
	}
	return svc.List(ctx)
}

type workerDeployParams struct {
	Name              string `json:"name"`
	Script            string `json:"script"`
	CompatibilityDate string `json:"compatibility_date"`
}

func (s *MCPServer) handleWorkerDeploy(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p workerDeployParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	if p.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if p.Script == "" {
		return nil, fmt.Errorf("script is required")
	}

	svc, err := NewWorkerServiceFromCreds(s.accountID, s.apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker service: %w", err)
	}

	var opts []WorkerOption
	if p.CompatibilityDate != "" {
		opts = append(opts, WithWorkerCompatibilityDate(p.CompatibilityDate))
	}

	return svc.Deploy(ctx, p.Name, strings.NewReader(p.Script), opts...)
}

func (s *MCPServer) handleDNSList(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p struct {
		ZoneID string `json:"zone_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	if p.ZoneID == "" {
		return nil, fmt.Errorf("zone_id is required")
	}

	svc, err := NewDNSServiceFromCreds(p.ZoneID, s.apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create DNS service: %w", err)
	}
	return svc.List(ctx)
}

func (s *MCPServer) handleKVList(ctx context.Context, _ json.RawMessage) (interface{}, error) {
	svc, err := NewKVServiceFromCreds(s.accountID, s.apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create KV service: %w", err)
	}
	return svc.ListNamespaces(ctx)
}

func (s *MCPServer) handleZoneList(ctx context.Context, _ json.RawMessage) (interface{}, error) {
	svc, err := NewZoneServiceFromCreds(s.accountID, s.apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create zone service: %w", err)
	}
	return svc.List(ctx)
}

type cachePurgeParams struct {
	ZoneID          string   `json:"zone_id"`
	PurgeEverything bool     `json:"purge_everything"`
	Files           []string `json:"files"`
	Tags            []string `json:"tags"`
	Hosts           []string `json:"hosts"`
}

func (s *MCPServer) handleCachePurge(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p cachePurgeParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	if p.ZoneID == "" {
		return nil, fmt.Errorf("zone_id is required")
	}

	svc, err := NewCacheServiceFromCreds(p.ZoneID, s.apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache service: %w", err)
	}

	if p.PurgeEverything {
		return svc.PurgeAll(ctx)
	}
	if len(p.Files) > 0 {
		return svc.PurgeByURLs(ctx, p.Files)
	}
	if len(p.Tags) > 0 {
		return svc.PurgeByTags(ctx, p.Tags)
	}
	if len(p.Hosts) > 0 {
		return svc.PurgeByHosts(ctx, p.Hosts)
	}
	return nil, fmt.Errorf("specify purge_everything, files, tags, or hosts")
}

func (s *MCPServer) handleDoctor(ctx context.Context, params json.RawMessage) (interface{}, error) {
	var p struct {
		Domain string `json:"domain"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	if p.Domain == "" {
		return nil, fmt.Errorf("domain is required")
	}

	doctor := NewDoctorService(0)
	return doctor.RunDiagnostics(ctx, p.Domain, nil)
}
