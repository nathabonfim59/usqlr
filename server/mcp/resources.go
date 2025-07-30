package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// handleResourcesList handles requests to list available resources.
func (h *Handler) handleResourcesList(ctx context.Context, w http.ResponseWriter, req *JSONRPCRequest) error {
	resources := []Resource{
		{
			URI:         "connections://list",
			Name:        "Database Connections",
			Description: "List all active database connections",
			MimeType:    "application/json",
		},
		{
			URI:         "connections://status",
			Name:        "Connection Status",
			Description: "Check the health status of database connections",
			MimeType:    "application/json",
		},
		{
			URI:         "schema://info",
			Name:        "Schema Information",
			Description: "Get database schema information for a connection",
			MimeType:    "application/json",
		},
	}

	result := map[string]interface{}{
		"resources": resources,
	}

	return h.sendSuccessResponse(w, req.ID, result)
}

// handleResourcesRead handles requests to read a specific resource.
func (h *Handler) handleResourcesRead(ctx context.Context, w http.ResponseWriter, req *JSONRPCRequest) error {
	// Parse parameters
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", "params must be an object")
	}

	uri, ok := params["uri"].(string)
	if !ok {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", "uri is required")
	}

	// Route based on URI
	switch {
	case uri == "connections://list":
		return h.readConnectionsList(ctx, w, req)
	case uri == "connections://status":
		return h.readConnectionsStatus(ctx, w, req)
	case uri == "schema://info":
		connectionID, ok := params["connection_id"].(string)
		if !ok {
			return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", "connection_id is required for schema info")
		}
		return h.readSchemaInfo(ctx, w, req, connectionID)
	default:
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", fmt.Sprintf("unknown resource URI: %s", uri))
	}
}

// readConnectionsList returns the list of active connections.
func (h *Handler) readConnectionsList(ctx context.Context, w http.ResponseWriter, req *JSONRPCRequest) error {
	connections := h.pool.ListConnections()

	result := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"uri":      "connections://list",
				"mimeType": "application/json",
				"text":     formatConnectionsList(connections),
			},
		},
	}

	return h.sendSuccessResponse(w, req.ID, result)
}

// readConnectionsStatus returns the health status of connections.
func (h *Handler) readConnectionsStatus(ctx context.Context, w http.ResponseWriter, req *JSONRPCRequest) error {
	connections := h.pool.ListConnections()
	status := make(map[string]interface{})

	for id := range connections {
		err := h.pool.CheckConnection(ctx, id)
		status[id] = map[string]interface{}{
			"healthy": err == nil,
			"error":   nil,
		}
		if err != nil {
			status[id].(map[string]interface{})["error"] = err.Error()
		}
	}

	statusJSON, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return h.sendErrorResponse(w, req.ID, -32603, "Internal error", err.Error())
	}

	result := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"uri":      "connections://status",
				"mimeType": "application/json",
				"text":     string(statusJSON),
			},
		},
	}

	return h.sendSuccessResponse(w, req.ID, result)
}

// readSchemaInfo returns schema information for a specific connection.
func (h *Handler) readSchemaInfo(ctx context.Context, w http.ResponseWriter, req *JSONRPCRequest, connectionID string) error {
	conn, err := h.pool.GetConnection(connectionID)
	if err != nil {
		return h.sendErrorResponse(w, req.ID, -32602, "Invalid params", fmt.Sprintf("connection not found: %s", connectionID))
	}

	// Get schema information using a basic query
	// This is a simplified approach - in production, you'd want to use the metadata package
	result, err := conn.ExecuteQuery(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema NOT IN ('information_schema', 'performance_schema', 'mysql', 'sys') LIMIT 100")
	if err != nil {
		// Fallback for databases that don't support information_schema
		result = &QueryResult{
			Columns:     []string{"note"},
			ColumnTypes: []string{"text"},
			Rows:        [][]interface{}{{"Schema information not available for this database type"}},
		}
	}

	schemaJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return h.sendErrorResponse(w, req.ID, -32603, "Internal error", err.Error())
	}

	response := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"uri":      "schema://info",
				"mimeType": "application/json",
				"text":     string(schemaJSON),
			},
		},
	}

	return h.sendSuccessResponse(w, req.ID, response)
}

// formatConnectionsList formats the connections list as a JSON string.
func formatConnectionsList(connections map[string]ConnectionInfo) string {
	data, err := json.MarshalIndent(connections, "", "  ")
	if err != nil {
		return "{\"error\": \"failed to format connections list\"}"
	}
	return string(data)
}

// Resource represents an MCP resource.
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}