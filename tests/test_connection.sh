#!/bin/bash

# Single Database Connection Test for usqlr (using common functions)
# Usage: ./test_connection.sh "database_dsn"

set -e

# Source common functions
source ./common.sh

# Test configuration
SERVER_PORT=8085
LOG_FILE="test_connection.log"
PID_FILE="test_connection.pid"

# Cleanup function
cleanup() {
    cleanup_test_files "$LOG_FILE" "$PID_FILE"
}

# Setup trap for cleanup
trap cleanup EXIT

# Check arguments
if [ $# -eq 0 ]; then
    print_error "Usage: $0 \"database_dsn\""
    print_error "Example: $0 \"sqlite3://test.db\""
    print_error "Example: $0 \"postgres://user:pass@localhost/dbname\""
    exit 1
fi

DSN="$1"

# Main test function
run_connection_test() {
    print_test_header "Testing Database Connection"
    print_info "DSN: $DSN"
    
    # Check binary
    check_binary
    
    # Start server
    start_server "$SERVER_PORT" "$LOG_FILE" "$PID_FILE"
    
    # Initialize MCP
    initialize_mcp 1 "$SERVER_PORT"
    
    # Test connection
    print_status "Testing database connection..."
    if test_connection_creation "test_conn" "$DSN" 2 "$SERVER_PORT"; then
        
        # Try a simple query if it's SQLite
        if [[ "$DSN" == sqlite3://* ]] || [[ "$DSN" == moderncsqlite://* ]]; then
            print_status "Testing simple query..."
            local query_response=$(execute_query "test_conn" "SELECT 1 as test_column" 3 "$SERVER_PORT")
            
            if check_response "$query_response" "Simple query"; then
                print_status "Query test successful! ✅"
                print_info "Query result: $(echo "$query_response" | jq -r '.result.content[0].text' 2>/dev/null | jq -c '.' || echo "Could not parse")"
            else
                print_warning "Query test failed, but connection was successful"
            fi
        fi
        
        # Close connection
        print_status "Closing connection..."
        local close_response=$(close_connection "test_conn" 4 "$SERVER_PORT")
        check_response "$close_response" "Close connection" || true
        
        print_status "Connection test completed successfully! 🎉"
    else
        print_error "Connection test failed!"
        exit 1
    fi
}

# Run the test
run_connection_test