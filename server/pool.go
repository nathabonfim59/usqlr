package server

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/xo/dburl"
	"github.com/xo/usql/drivers"
)

// ConnectionInterface defines the interface for database connections.
type ConnectionInterface interface {
	ExecuteQuery(ctx context.Context, query string, args ...interface{}) (*QueryResult, error)
	ExecuteStatement(ctx context.Context, query string, args ...interface{}) (*StatementResult, error)
}

// ConnectionPool manages multiple database connections.
type ConnectionPool struct {
	mu          sync.RWMutex
	connections map[string]*Connection
	maxConns    int
	config      *Config
}

// Connection represents a database connection with its associated handler.
type Connection struct {
	ID       string
	URL      *dburl.URL
	DB       *sql.DB
	Created  time.Time
	LastUsed time.Time
	mu       sync.RWMutex
}

// NewConnectionPool creates a new connection pool.
func NewConnectionPool(config *Config) *ConnectionPool {
	return &ConnectionPool{
		connections: make(map[string]*Connection),
		maxConns:    config.Server.MaxConnections,
		config:      config,
	}
}

// CreateConnection creates a new database connection and adds it to the pool.
func (cp *ConnectionPool) CreateConnection(ctx context.Context, id, dsn string) (ConnectionInterface, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Check if connection already exists
	if _, exists := cp.connections[id]; exists {
		return nil, fmt.Errorf("connection with ID %s already exists", id)
	}

	// Check pool size limit
	if len(cp.connections) >= cp.maxConns {
		return nil, fmt.Errorf("connection pool limit reached (max: %d)", cp.maxConns)
	}

	// Parse DSN
	u, err := dburl.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	// Open database connection using drivers directly
	db, err := drivers.Open(ctx, u, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create connection object
	conn := &Connection{
		ID:       id,
		URL:      u,
		DB:       db,
		Created:  time.Now(),
		LastUsed: time.Now(),
	}


	// Add to pool
	cp.connections[id] = conn

	return conn, nil
}

// GetConnection retrieves a connection from the pool.
func (cp *ConnectionPool) GetConnection(id string) (ConnectionInterface, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	conn, exists := cp.connections[id]
	if !exists {
		return nil, fmt.Errorf("connection with ID %s not found", id)
	}

	// Update last used time
	conn.mu.Lock()
	conn.LastUsed = time.Now()
	conn.mu.Unlock()

	return conn, nil
}

// CloseConnection closes and removes a connection from the pool.
func (cp *ConnectionPool) CloseConnection(id string) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	conn, exists := cp.connections[id]
	if !exists {
		return fmt.Errorf("connection with ID %s not found", id)
	}

	// Close database connection
	if conn.DB != nil {
		conn.DB.Close()
	}


	// Remove from pool
	delete(cp.connections, id)

	return nil
}

// ListConnections returns a list of all connection IDs and their basic info.
func (cp *ConnectionPool) ListConnections() map[string]ConnectionInfo {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	result := make(map[string]ConnectionInfo, len(cp.connections))
	for id, conn := range cp.connections {
		conn.mu.RLock()
		result[id] = ConnectionInfo{
			ID:       conn.ID,
			Driver:   conn.URL.Driver,
			Host:     conn.URL.Host,
			Database: conn.URL.Path,
			Created:  conn.Created,
			LastUsed: conn.LastUsed,
		}
		conn.mu.RUnlock()
	}

	return result
}

// ConnectionInfo provides basic information about a connection.
type ConnectionInfo struct {
	ID       string    `json:"id"`
	Driver   string    `json:"driver"`
	Host     string    `json:"host"`
	Database string    `json:"database"`
	Created  time.Time `json:"created"`
	LastUsed time.Time `json:"last_used"`
}

// CheckConnection tests if a connection is still alive.
func (cp *ConnectionPool) CheckConnection(ctx context.Context, id string) error {
	cp.mu.RLock()
	conn, exists := cp.connections[id]
	cp.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("connection with ID %s not found", id)
	}

	return conn.DB.PingContext(ctx)
}

// Close closes all connections in the pool.
func (cp *ConnectionPool) Close() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	var lastErr error
	for id, conn := range cp.connections {
		if err := conn.DB.Close(); err != nil {
			lastErr = err
		}
		delete(cp.connections, id)
	}

	return lastErr
}

// Size returns the current number of connections in the pool.
func (cp *ConnectionPool) Size() int {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	return len(cp.connections)
}

// ExecuteQuery executes a SQL query on the specified connection.
func (conn *Connection) ExecuteQuery(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.LastUsed = time.Now()

	// Execute query directly on database
	rows, err := conn.DB.QueryContext(ctx, query, args...)
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

// ExecuteStatement executes a non-query SQL statement (INSERT, UPDATE, DELETE, etc.).
func (conn *Connection) ExecuteStatement(ctx context.Context, statement string, args ...interface{}) (*StatementResult, error) {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	conn.LastUsed = time.Now()

	result, err := conn.DB.ExecContext(ctx, statement, args...)
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