package server

import (
	"context"
	"database/sql"
	"fmt"
	"os/user"
	"sync"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/xo/dburl"
	"github.com/xo/usql/handler"
)

// MultiHandler extends the basic handler functionality for multi-connection support.
type MultiHandler struct {
	mu       sync.RWMutex
	handlers map[string]*handler.Handler
	config   *Config
}

// NewMultiHandler creates a new multi-connection handler.
func NewMultiHandler(config *Config) *MultiHandler {
	return &MultiHandler{
		handlers: make(map[string]*handler.Handler),
		config:   config,
	}
}

// CreateHandler creates and stores a new handler for a connection.
func (mh *MultiHandler) CreateHandler(ctx context.Context, id string, u *dburl.URL, db *sql.DB) (*handler.Handler, error) {
	mh.mu.Lock()
	defer mh.mu.Unlock()

	// Check if handler already exists
	if _, exists := mh.handlers[id]; exists {
		return nil, fmt.Errorf("handler with ID %s already exists", id)
	}

	// Get current user for handler initialization
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// Create memory filesystem for handler
	fs := memfs.New()

	// Create new handler instance
	h, err := handler.New(nil, currentUser, fs, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create handler: %w", err)
	}

	// Set connection information
	if err := h.SetURL(u); err != nil {
		return nil, fmt.Errorf("failed to set URL: %w", err)
	}

	if err := h.SetDB(db); err != nil {
		return nil, fmt.Errorf("failed to set database: %w", err)
	}

	// Store handler
	mh.handlers[id] = h

	return h, nil
}

// GetHandler retrieves a handler by connection ID.
func (mh *MultiHandler) GetHandler(id string) (*handler.Handler, error) {
	mh.mu.RLock()
	defer mh.mu.RUnlock()

	h, exists := mh.handlers[id]
	if !exists {
		return nil, fmt.Errorf("handler with ID %s not found", id)
	}

	return h, nil
}

// RemoveHandler removes a handler for a connection.
func (mh *MultiHandler) RemoveHandler(id string) error {
	mh.mu.Lock()
	defer mh.mu.Unlock()

	if _, exists := mh.handlers[id]; !exists {
		return fmt.Errorf("handler with ID %s not found", id)
	}

	delete(mh.handlers, id)
	return nil
}

// ListHandlers returns a list of all handler IDs.
func (mh *MultiHandler) ListHandlers() []string {
	mh.mu.RLock()
	defer mh.mu.RUnlock()

	ids := make([]string, 0, len(mh.handlers))
	for id := range mh.handlers {
		ids = append(ids, id)
	}

	return ids
}

// ExecuteQuery executes a query using the specified handler.
func (mh *MultiHandler) ExecuteQuery(ctx context.Context, connectionID, query string, args ...interface{}) (*QueryResult, error) {
	h, err := mh.GetHandler(connectionID)
	if err != nil {
		return nil, err
	}

	// Get database connection from handler
	db := h.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection available")
	}

	// Execute query directly on the database
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Get column information
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to get column types: %w", err)
	}

	// Prepare result structure
	result := &QueryResult{
		Columns:     columns,
		ColumnTypes: make([]string, len(columnTypes)),
		Rows:        [][]interface{}{},
	}

	for i, ct := range columnTypes {
		result.ColumnTypes[i] = ct.DatabaseTypeName()
	}

	// Read all rows
	for rows.Next() {
		// Create slice of interface{} to hold row values
		values := make([]interface{}, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert byte arrays to strings for JSON serialization
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				values[i] = string(b)
			}
		}

		result.Rows = append(result.Rows, values)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return result, nil
}

// ExecuteStatement executes a statement using the specified handler.
func (mh *MultiHandler) ExecuteStatement(ctx context.Context, connectionID, query string, args ...interface{}) (*StatementResult, error) {
	h, err := mh.GetHandler(connectionID)
	if err != nil {
		return nil, err
	}

	// Get database connection from handler
	db := h.DB()
	if db == nil {
		return nil, fmt.Errorf("no database connection available")
	}

	// Execute statement directly on the database
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("statement execution failed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// Some drivers don't support RowsAffected
		rowsAffected = -1
	}

	lastInsertId, err := result.LastInsertId()
	if err != nil {
		// Some drivers don't support LastInsertId
		lastInsertId = -1
	}

	return &StatementResult{
		RowsAffected: rowsAffected,
		LastInsertId: lastInsertId,
	}, nil
}

// Close closes all handlers.
func (mh *MultiHandler) Close() error {
	mh.mu.Lock()
	defer mh.mu.Unlock()

	var lastErr error
	for id := range mh.handlers {
		delete(mh.handlers, id)
	}

	return lastErr
}