package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Handler handles MCP (Model Context Protocol) requests.
type Handler struct {
	pool ConnectionPool
}

// ConnectionPool interface for dependency injection.
type ConnectionPool interface {
	CreateConnection(ctx context.Context, id, dsn string) (Connection, error)
	GetConnection(id string) (Connection, error)
	CloseConnection(id string) error
	ListConnections() map[string]ConnectionInfo
	CheckConnection(ctx context.Context, id string) error
}

// Connection interface for database connections.
type Connection interface {
	ExecuteQuery(ctx context.Context, query string, args ...interface{}) (*QueryResult, error)
	ExecuteStatement(ctx context.Context, query string, args ...interface{}) (*StatementResult, error)
}

// ConnectionInfo provides basic information about a connection.
type ConnectionInfo struct {
	ID       string `json:"id"`
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Database string `json:"database"`
}

// QueryResult represents the result of a SQL query.
type QueryResult struct {
	Columns     []string        `json:"columns"`
	ColumnTypes []string        `json:"column_types"`
	Rows        [][]interface{} `json:"rows"`
}

// StatementResult represents the result of a SQL statement execution.
type StatementResult struct {
	RowsAffected int64 `json:"rows_affected"`
	LastInsertId int64 `json:"last_insert_id"`
}

// New creates a new MCP handler.
func New(pool ConnectionPool) (*Handler, error) {
	return &Handler{
		pool: pool,
	}, nil
}

// ServeHTTP handles MCP HTTP requests.
func (h *Handler) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return h.sendErrorResponse(w, nil, -32700, "Parse error", nil)
	}

	// Validate JSON-RPC request
	if err := h.validateRequest(&req); err != nil {
		return h.sendErrorResponse(w, req.ID, -32600, "Invalid Request", err.Error())
	}

	// Route the request based on method
	switch req.Method {
	case "initialize":
		return h.handleInitialize(ctx, w, &req)
	case "capabilities":
		return h.handleCapabilities(ctx, w, &req)
	case "resources/list":
		return h.handleResourcesList(ctx, w, &req)
	case "resources/read":
		return h.handleResourcesRead(ctx, w, &req)
	case "tools/list":
		return h.handleToolsList(ctx, w, &req)
	case "tools/call":
		return h.handleToolsCall(ctx, w, &req)
	default:
		return h.sendErrorResponse(w, req.ID, -32601, "Method not found", nil)
	}
}

// validateRequest validates a JSON-RPC 2.0 request.
func (h *Handler) validateRequest(req *JSONRPCRequest) error {
	if req.JSONRPC != "2.0" {
		return fmt.Errorf("invalid JSON-RPC version: %s", req.JSONRPC)
	}
	
	if req.Method == "" {
		return fmt.Errorf("missing method")
	}

	if strings.HasPrefix(req.Method, "rpc.") {
		return fmt.Errorf("method name cannot start with 'rpc.'")
	}

	return nil
}

// handleInitialize handles MCP initialization.
func (h *Handler) handleInitialize(ctx context.Context, w http.ResponseWriter, req *JSONRPCRequest) error {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"resources": map[string]interface{}{
				"subscribe": false,
				"listChanged": false,
			},
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "usqlr",
			"version": "1.0.0",
		},
	}

	return h.sendSuccessResponse(w, req.ID, result)
}

// handleCapabilities returns server capabilities.
func (h *Handler) handleCapabilities(ctx context.Context, w http.ResponseWriter, req *JSONRPCRequest) error {
	capabilities := map[string]interface{}{
		"resources": []string{
			"list_databases",
			"schema_info",
			"connection_status",
		},
		"tools": []string{
			"execute_query",
			"create_connection",
			"close_connection",
		},
	}

	return h.sendSuccessResponse(w, req.ID, capabilities)
}

// sendSuccessResponse sends a successful JSON-RPC response.
func (h *Handler) sendSuccessResponse(w http.ResponseWriter, id interface{}, result interface{}) error {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(response)
}

// sendErrorResponse sends an error JSON-RPC response.
func (h *Handler) sendErrorResponse(w http.ResponseWriter, id interface{}, code int, message string, data interface{}) error {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(response)
}

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{}   `json:"id,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}