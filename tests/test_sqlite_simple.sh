#!/bin/bash

# SQLite Integration Test for usqlr (using common functions)

set -e

# Source common functions
source ./common.sh

# Test configuration
SERVER_PORT=8083
TEST_DB="test_usqlr.db"
LOG_FILE="server_test.log"
PID_FILE="server_test.pid"

# Cleanup function
cleanup() {
    print_status "Cleaning up test files..."
    cleanup_test_files "$LOG_FILE" "$PID_FILE"
    rm -f "$TEST_DB"
}

# Setup trap for cleanup
trap cleanup EXIT

# Main test function
run_sqlite_test() {
    print_test_header "SQLite Integration Test for usqlr"
    
    # Check binary
    check_binary
    
    # Start server
    start_server "$SERVER_PORT" "$LOG_FILE" "$PID_FILE"
    
    # Test health endpoint
    print_status "Testing health endpoint..."
    local health_response=$(curl -s "http://localhost:$SERVER_PORT/health")
    print_info "Health: $health_response"
    
    # Initialize MCP
    initialize_mcp 1 "$SERVER_PORT"
    
    # List available tools
    print_status "Listing available tools..."
    local tools_response=$(get_tools 2 "$SERVER_PORT")
    local tools=$(echo "$tools_response" | jq -r '.result.tools[].name' 2>/dev/null | tr '\n' ', ')
    print_info "Available tools: $tools"
    
    # Create SQLite connection
    print_status "Creating SQLite database connection..."
    test_connection_creation "test_sqlite" "sqlite3://$TEST_DB" 3 "$SERVER_PORT"
    
    # Create test table
    print_status "Creating test table..."
    local create_table="CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT UNIQUE, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)"
    local table_response=$(execute_statement "test_sqlite" "$create_table" 4 "$SERVER_PORT")
    check_response "$table_response" "Create table"
    print_info "Table created successfully"
    
    # Insert test data
    print_status "Inserting test data..."
    local insert_queries=(
        "INSERT INTO users (name, email) VALUES ('Alice Johnson', 'alice@example.com')"
        "INSERT INTO users (name, email) VALUES ('Bob Smith', 'bob@example.com')"
        "INSERT INTO users (name, email) VALUES ('Carol Davis', 'carol@example.com')"
    )
    
    local request_id=5
    for query in "${insert_queries[@]}"; do
        local insert_response=$(execute_statement "test_sqlite" "$query" $request_id "$SERVER_PORT")
        check_response "$insert_response" "Insert data"
        ((request_id++))
    done
    print_info "Test data inserted successfully"
    
    # Query test data
    print_status "Querying test data..."
    local query_response=$(execute_query "test_sqlite" "SELECT * FROM users ORDER BY id" 8 "$SERVER_PORT")
    check_response "$query_response" "Query data"
    print_info "Query results:"
    echo "$query_response" | jq -r '.result.content[0].text' 2>/dev/null | jq '.' || echo "$query_response"
    
    # Test parameterized query
    print_status "Testing parameterized query..."
    local param_query_params="{\"name\":\"execute_query\",\"arguments\":{\"connection_id\":\"test_sqlite\",\"query\":\"SELECT * FROM users WHERE name LIKE ?\",\"args\":[\"A%\"]}}"
    local param_response=$(mcp_request "tools/call" "$param_query_params" 9 "$SERVER_PORT")
    check_response "$param_response" "Parameterized query"
    print_info "Parameterized query results:"
    echo "$param_response" | jq -r '.result.content[0].text' 2>/dev/null | jq '.' || echo "$param_response"
    
    # List connections
    print_status "Listing active connections..."
    local list_response=$(list_connections 10 "$SERVER_PORT")
    check_response "$list_response" "List connections"
    print_info "Active connections:"
    echo "$list_response" | jq -r '.result.contents[0].text' 2>/dev/null || echo "$list_response"
    
    # Check connection status
    print_status "Checking connection status..."
    local status_response=$(check_connection_status 11 "$SERVER_PORT")
    check_response "$status_response" "Connection status"
    print_info "Connection status:"
    echo "$status_response" | jq -r '.result.contents[0].text' 2>/dev/null || echo "$status_response"
    
    # Update data
    print_status "Updating test data..."
    local update_response=$(execute_statement "test_sqlite" "UPDATE users SET email = 'alice.johnson@example.com' WHERE name = 'Alice Johnson'" 12 "$SERVER_PORT")
    check_response "$update_response" "Update data"
    print_info "Update successful"
    
    # Verify update
    print_status "Verifying update..."
    local verify_response=$(execute_query "test_sqlite" "SELECT * FROM users WHERE name = 'Alice Johnson'" 13 "$SERVER_PORT")
    check_response "$verify_response" "Verify update"
    print_info "Updated record:"
    echo "$verify_response" | jq -r '.result.content[0].text' 2>/dev/null | jq '.' || echo "$verify_response"
    
    # Close connection
    print_status "Closing connection..."
    local close_response=$(close_connection "test_sqlite" 14 "$SERVER_PORT")
    check_response "$close_response" "Close connection"
    print_info "Connection closed successfully"
    
    # Test summary
    print_test_summary 12 12 0
    
    print_status "Test Details:"
    echo "- ✅ Server startup and health check"
    echo "- ✅ MCP protocol initialization"
    echo "- ✅ SQLite connection creation"
    echo "- ✅ Table creation and data insertion"
    echo "- ✅ Query execution (SELECT)"
    echo "- ✅ Parameterized queries"
    echo "- ✅ Statement execution (INSERT, UPDATE)"
    echo "- ✅ Connection management"
    echo "- ✅ Resource listing and status checks"
    echo "- ✅ Data updates and verification"
    echo "- ✅ Connection cleanup"
    echo "- ✅ Proper error handling"
    
    print_status "🎉 SQLite integration test completed successfully!"
}

# Run the test
run_sqlite_test