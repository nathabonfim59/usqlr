package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// handleToolsList handles requests to list available tools.
func (h *Handler) handleToolsList(ctx context.Context, w http.ResponseWriter, req *JSONRPCRequest) error {
	tools := []Tool{
		{
			Name:        "execute_query",
			Description: "Execute a SQL query on a database connection",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"connection_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the database connection to use",
					},
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The SQL query to execute",
					},
					"args": map[string]interface{}{
						"type":        "array",
						"description": "Optional query arguments for parameterized queries",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"connection_id", "query"},
			},
		},
		{
			Name:        "create_connection",
			Description: "Create a new database connection",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"connection_id": map[string]interface{}{
						"type":        "string",
						"description": "A unique identifier for the connection",
					},
					"dsn": map[string]interface{}{
						"type":        "string",
						"description": "The database connection string (DSN)",
					},
				},
				"required": []string{"connection_id", "dsn"},
			},
		},
		{
			Name:        "close_connection",
			Description: "Close an existing database connection",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"connection_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the connection to close",
					},
				},
				"required": []string{"connection_id"},
			},
		},
		{
			Name:        "execute_statement",
			Description: "Execute a SQL statement (INSERT, UPDATE, DELETE, etc.)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"connection_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the database connection to use",
					},
					"statement": map[string]interface{}{
						"type":        "string",
						"description": "The SQL statement to execute",
					},
					"args": map[string]interface{}{
						"type":        "array",
						"description": "Optional statement arguments for parameterized statements",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"connection_id", "statement"},
			},
		},
	}

	result := map[string]interface{}{
		"tools": tools,
	}

	return h.sendSuccessResponse(w, req.ID, result)
}

// handleToolsCall handles tool invocation requests.
func (h *Handler) handleToolsCall(ctx context.Context, w http.ResponseWriter, req *JSONRPCRequest) error {
	// Parse parameters
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", "params must be an object")
	}

	name, ok := params["name"].(string)
	if !ok {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", "name is required")
	}

	arguments, ok := params["arguments"].(map[string]interface{})
	if !ok {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", "arguments is required")
	}

	// Route to appropriate tool handler
	switch name {
	case "execute_query":
		return h.toolExecuteQuery(ctx, w, req, arguments)
	case "create_connection":
		return h.toolCreateConnection(ctx, w, req, arguments)
	case "close_connection":
		return h.toolCloseConnection(ctx, w, req, arguments)
	case "execute_statement":
		return h.toolExecuteStatement(ctx, w, req, arguments)
	default:
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", fmt.Sprintf("unknown tool: %s", name))
	}
}

// toolExecuteQuery implements the execute_query tool.
func (h *Handler) toolExecuteQuery(ctx context.Context, w http.ResponseWriter, req *JSONRPCRequest, args map[string]interface{}) error {
	connectionID, ok := args["connection_id"].(string)
	if !ok {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", "connection_id is required")
	}

	query, ok := args["query"].(string)
	if !ok {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", "query is required")
	}

	// Get connection
	conn, err := h.pool.GetConnection(connectionID)
	if err != nil {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", fmt.Sprintf("connection not found: %s", connectionID))
	}

	// Parse query arguments if provided
	var queryArgs []interface{}
	if argsInterface, exists := args["args"]; exists {
		if argSlice, ok := argsInterface.([]interface{}); ok {
			queryArgs = argSlice
		}
	}

	// Execute query
	result, err := conn.ExecuteQuery(ctx, query, queryArgs...)
	if err != nil {
		return h.sendErrorResponse(w, req.ID, -32603, "Query execution failed", err.Error())
	}

	// Format result as JSON
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return h.sendErrorResponse(w, req.ID, -32603, "Internal error", err.Error())
	}

	response := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": string(resultJSON),
			},
		},
	}

	return h.sendSuccessResponse(w, req.ID, response)
}

// toolCreateConnection implements the create_connection tool.
func (h *Handler) toolCreateConnection(ctx context.Context, w http.ResponseWriter, req *JSONRPCRequest, args map[string]interface{}) error {
	connectionID, ok := args["connection_id"].(string)
	if !ok {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", "connection_id is required")
	}

	dsn, ok := args["dsn"].(string)
	if !ok {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", "dsn is required")
	}

	// Create connection
	_, err := h.pool.CreateConnection(ctx, connectionID, dsn)
	if err != nil {
		return h.sendErrorResponse(w, req.ID, -32603, "Connection creation failed", err.Error())
	}

	response := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Successfully created connection: %s", connectionID),
			},
		},
	}

	return h.sendSuccessResponse(w, req.ID, response)
}

// toolCloseConnection implements the close_connection tool.
func (h *Handler) toolCloseConnection(ctx context.Context, w http.ResponseWriter, req *JSONRPCRequest, args map[string]interface{}) error {
	connectionID, ok := args["connection_id"].(string)
	if !ok {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", "connection_id is required")
	}

	// Close connection
	err := h.pool.CloseConnection(connectionID)
	if err != nil {
		return h.sendErrorResponse(w, req.ID, -32603, "Connection close failed", err.Error())
	}

	response := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("Successfully closed connection: %s", connectionID),
			},
		},
	}

	return h.sendSuccessResponse(w, req.ID, response)
}

// toolExecuteStatement implements the execute_statement tool.
func (h *Handler) toolExecuteStatement(ctx context.Context, w http.ResponseWriter, req *JSONRPCRequest, args map[string]interface{}) error {
	connectionID, ok := args["connection_id"].(string)
	if !ok {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", "connection_id is required")
	}

	statement, ok := args["statement"].(string)
	if !ok {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", "statement is required")
	}

	// Get connection
	conn, err := h.pool.GetConnection(connectionID)
	if err != nil {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", fmt.Sprintf("connection not found: %s", connectionID))
	}

	// Parse statement arguments if provided
	var stmtArgs []interface{}
	if argsInterface, exists := args["args"]; exists {
		if argSlice, ok := argsInterface.([]interface{}); ok {
			stmtArgs = argSlice
		}
	}

	// Execute statement
	result, err := conn.ExecuteStatement(ctx, statement, stmtArgs...)
	if err != nil {
		return h.sendErrorResponse(w, req.ID, -32603, "Statement execution failed", err.Error())
	}

	// Format result as JSON
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return h.sendErrorResponse(w, req.ID, -32603, "Internal error", err.Error())
	}

	response := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": string(resultJSON),
			},
		},
	}

	return h.sendSuccessResponse(w, req.ID, response)
}

// Tool represents an MCP tool.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"inputSchema,omitempty"`
}