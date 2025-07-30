package server

import (
	"context"

	"github.com/xo/usql/server/mcp"
)

// PoolAdapter adapts ConnectionPool to implement the mcp.ConnectionPool interface.
type PoolAdapter struct {
	pool *ConnectionPool
}

// NewPoolAdapter creates a new pool adapter.
func NewPoolAdapter(pool *ConnectionPool) *PoolAdapter {
	return &PoolAdapter{pool: pool}
}

// CreateConnection implements mcp.ConnectionPool interface.
func (pa *PoolAdapter) CreateConnection(ctx context.Context, id, dsn string) (mcp.Connection, error) {
	conn, err := pa.pool.CreateConnection(ctx, id, dsn)
	if err != nil {
		return nil, err
	}
	
	// Return an adapter that implements mcp.Connection
	return &ConnectionAdapter{conn: conn.(*Connection)}, nil
}

// GetConnection implements mcp.ConnectionPool interface.
func (pa *PoolAdapter) GetConnection(id string) (mcp.Connection, error) {
	conn, err := pa.pool.GetConnection(id)
	if err != nil {
		return nil, err
	}
	
	// Return an adapter that implements mcp.Connection
	return &ConnectionAdapter{conn: conn.(*Connection)}, nil
}

// CloseConnection implements mcp.ConnectionPool interface.
func (pa *PoolAdapter) CloseConnection(id string) error {
	return pa.pool.CloseConnection(id)
}

// ListConnections implements mcp.ConnectionPool interface.
func (pa *PoolAdapter) ListConnections() map[string]mcp.ConnectionInfo {
	connections := pa.pool.ListConnections()
	result := make(map[string]mcp.ConnectionInfo, len(connections))
	
	for id, conn := range connections {
		result[id] = mcp.ConnectionInfo{
			ID:       conn.ID,
			Driver:   conn.Driver,
			Host:     conn.Host,
			Database: conn.Database,
		}
	}
	
	return result
}

// CheckConnection implements mcp.ConnectionPool interface.
func (pa *PoolAdapter) CheckConnection(ctx context.Context, id string) error {
	return pa.pool.CheckConnection(ctx, id)
}

// ConnectionAdapter adapts Connection to implement the mcp.Connection interface.
type ConnectionAdapter struct {
	conn *Connection
}

// ExecuteQuery implements mcp.Connection interface.
func (ca *ConnectionAdapter) ExecuteQuery(ctx context.Context, query string, args ...interface{}) (*mcp.QueryResult, error) {
	result, err := ca.conn.ExecuteQuery(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	
	return &mcp.QueryResult{
		Columns:     result.Columns,
		ColumnTypes: result.ColumnTypes,
		Rows:        result.Rows,
	}, nil
}

// ExecuteStatement implements mcp.Connection interface.
func (ca *ConnectionAdapter) ExecuteStatement(ctx context.Context, query string, args ...interface{}) (*mcp.StatementResult, error) {
	result, err := ca.conn.ExecuteStatement(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	
	return &mcp.StatementResult{
		RowsAffected: result.RowsAffected,
		LastInsertId: result.LastInsertId,
	}, nil
}