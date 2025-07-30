package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/xo/usql/server/mcp"
)

// Server represents the usqlr HTTP server.
type Server struct {
	pool       *ConnectionPool
	config     *Config
	httpServer *http.Server
	mcpHandler *mcp.Handler
}

// New creates a new server instance.
func New(config *Config) (*Server, error) {
	pool := NewConnectionPool(config)
	adapter := NewPoolAdapter(pool)
	
	mcpHandler, err := mcp.New(adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP handler: %w", err)
	}

	return &Server{
		pool:       pool,
		config:     config,
		mcpHandler: mcpHandler,
	}, nil
}

// Listen starts the HTTP server on the specified address.
func (s *Server) Listen(ctx context.Context, addr string) error {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)

	// MCP endpoint (JSON-RPC 2.0)
	if s.config.Server.EnableMCP {
		mux.HandleFunc("/mcp", s.handleMCP)
	}

	// CORS middleware
	var handler http.Handler = mux
	if s.config.Server.EnableCORS {
		handler = s.corsMiddleware(handler)
	}

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		return s.httpServer.Shutdown(context.Background())
	case err := <-errChan:
		return err
	}
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	// Close connection pool
	if err := s.pool.Close(); err != nil {
		log.Printf("Error closing connection pool: %v", err)
	}

	// Shutdown HTTP server
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}

	return nil
}

// handleHealth handles health check requests.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := struct {
		Status      string `json:"status"`
		Connections int    `json:"connections"`
		Timestamp   string `json:"timestamp"`
	}{
		Status:      "healthy",
		Connections: s.pool.Size(),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleMCP handles MCP (JSON-RPC 2.0) requests.
func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set content type for JSON-RPC
	w.Header().Set("Content-Type", "application/json")

	// Create request context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), s.config.Server.RequestTimeout)
	defer cancel()

	// Handle the MCP request
	if err := s.mcpHandler.ServeHTTP(ctx, w, r); err != nil {
		log.Printf("MCP handler error: %v", err)
		
		// Send JSON-RPC error response
		errorResp := map[string]interface{}{
			"jsonrpc": "2.0",
			"error": map[string]interface{}{
				"code":    -32603,
				"message": "Internal error",
			},
			"id": nil,
		}
		json.NewEncoder(w).Encode(errorResp)
	}
}

// corsMiddleware adds CORS headers to responses.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
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
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{} `json:"id,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// validateJSONRPCRequest validates a JSON-RPC 2.0 request.
func validateJSONRPCRequest(req *JSONRPCRequest) error {
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