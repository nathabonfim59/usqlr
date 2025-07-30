#!/bin/bash

# SQLite Integration Test for usqlr
# This script tests the full MCP workflow with a SQLite database

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SERVER_PORT=8083
SERVER_PID=""
TEST_DB="test_usqlr.db"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[TEST]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to make MCP requests
mcp_request() {
    local method="$1"
    local params="$2"
    local id="$3"
    
    if [ -z "$params" ]; then
        local json_data="{\"jsonrpc\":\"2.0\",\"method\":\"$method\",\"id\":$id}"
    else
        local json_data="{\"jsonrpc\":\"2.0\",\"method\":\"$method\",\"params\":$params,\"id\":$id}"
    fi
    
    curl -s -X POST -H "Content-Type: application/json" \
         -d "$json_data" \
         "http://localhost:$SERVER_PORT/mcp"
}

# Function to check if server is running
wait_for_server() {
    local timeout=10
    local count=0
    
    print_status "Waiting for server to start..."
    while [ $count -lt $timeout ]; do
        if curl -s "http://localhost:$SERVER_PORT/health" > /dev/null 2>&1; then
            print_status "Server is ready!"
            return 0
        fi
        sleep 1
        count=$((count + 1))
    done
    
    print_error "Server failed to start within $timeout seconds"
    return 1
}

# Function to start server
start_server() {
    print_status "Starting usqlr server on port $SERVER_PORT..."
    ../usqlr --port $SERVER_PORT > server_test.log 2>&1 &
    SERVER_PID=$!
    echo $SERVER_PID > server_test.pid
    
    wait_for_server
}

# Function to stop server
stop_server() {
    if [ -n "$SERVER_PID" ]; then
        print_status "Stopping server (PID: $SERVER_PID)..."
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    
    if [ -f server_test.pid ]; then
        local pid=$(cat server_test.pid)
        kill $pid 2>/dev/null || true
        rm -f server_test.pid
    fi
    
    rm -f server_test.log
}

# Function to cleanup test files
cleanup() {
    print_status "Cleaning up test files..."
    stop_server
    rm -f "$TEST_DB" server_test.log server_test.pid
}

# Function to check JSON response for errors
check_response() {
    local response="$1"
    local step="$2"
    
    if echo "$response" | grep -q '"error"'; then
        print_error "Step '$step' failed:"
        echo "$response" | jq '.' 2>/dev/null || echo "$response"
        return 1
    fi
    
    return 0
}

# Setup trap for cleanup
trap cleanup EXIT

print_status "========================================="
print_status "SQLite Integration Test for usqlr"
print_status "========================================="

# Check if usqlr binary exists
if [ ! -f ../usqlr ]; then
    print_error "usqlr binary not found. Run 'make build' first."
    exit 1
fi

# Start server
start_server

print_status "Testing health endpoint..."
health_response=$(curl -s "http://localhost:$SERVER_PORT/health")
echo "Health: $health_response"

print_status "Step 1: Initialize MCP protocol..."
init_response=$(mcp_request "initialize" "" 1)
check_response "$init_response" "Initialize"
echo "Initialize response: $init_response" | jq '.' 2>/dev/null || echo "$init_response"

print_status "Step 2: List available tools..."
tools_response=$(mcp_request "tools/list" "" 2)
check_response "$tools_response" "Tools list"
echo "Tools available: $(echo "$tools_response" | jq -r '.result.tools[].name' 2>/dev/null | tr '\n' ', ')"

print_status "Step 3: Create SQLite database connection..."
create_params="{\"arguments\":{\"connection_id\":\"test_sqlite\",\"dsn\":\"sqlite3://$TEST_DB\"}}"
create_response=$(mcp_request "tools/call" "{\"name\":\"create_connection\",\"arguments\":{\"connection_id\":\"test_sqlite\",\"dsn\":\"sqlite3://$TEST_DB\"}}" 3)
check_response "$create_response" "Create connection"
echo "Connection created: $create_response"

print_status "Step 4: Create test table..."
create_table_query="CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT UNIQUE, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)"
table_params="{\"name\":\"execute_statement\",\"arguments\":{\"connection_id\":\"test_sqlite\",\"statement\":\"$create_table_query\"}}"
table_response=$(mcp_request "tools/call" "$table_params" 4)
check_response "$table_response" "Create table"
echo "Table created: $table_response"

print_status "Step 5: Insert test data..."
insert_queries=(
    "INSERT INTO users (name, email) VALUES ('Alice Johnson', 'alice@example.com')"
    "INSERT INTO users (name, email) VALUES ('Bob Smith', 'bob@example.com')"
    "INSERT INTO users (name, email) VALUES ('Carol Davis', 'carol@example.com')"
)

for i in "${!insert_queries[@]}"; do
    insert_params="{\"name\":\"execute_statement\",\"arguments\":{\"connection_id\":\"test_sqlite\",\"statement\":\"${insert_queries[$i]}\"}}"
    insert_response=$(mcp_request "tools/call" "$insert_params" $((5 + i)))
    check_response "$insert_response" "Insert data $((i + 1))"
done

print_status "Step 6: Query test data..."
query_params="{\"name\":\"execute_query\",\"arguments\":{\"connection_id\":\"test_sqlite\",\"query\":\"SELECT * FROM users ORDER BY id\"}}"
query_response=$(mcp_request "tools/call" "$query_params" 8)
check_response "$query_response" "Query data"
echo "Query results:"
echo "$query_response" | jq -r '.result.content[0].text' 2>/dev/null | jq '.' || echo "$query_response"

print_status "Step 7: Test parameterized query..."
param_query_params="{\"name\":\"execute_query\",\"arguments\":{\"connection_id\":\"test_sqlite\",\"query\":\"SELECT * FROM users WHERE name LIKE ?\",\"args\":[\"A%\"]}}"
param_response=$(mcp_request "tools/call" "$param_query_params" 9)
check_response "$param_response" "Parameterized query"
echo "Parameterized query results:"
echo "$param_response" | jq -r '.result.content[0].text' 2>/dev/null | jq '.' || echo "$param_response"

print_status "Step 8: Test schema information..."
schema_params="{\"uri\":\"schema://info\",\"connection_id\":\"test_sqlite\"}"
schema_response=$(mcp_request "resources/read" "$schema_params" 10)
check_response "$schema_response" "Schema info"
echo "Schema info:"
echo "$schema_response" | jq -r '.result.contents[0].text' 2>/dev/null || echo "$schema_response"

print_status "Step 9: List connections..."
list_params="{\"uri\":\"connections://list\"}"
list_response=$(mcp_request "resources/read" "$list_params" 11)
check_response "$list_response" "List connections"
echo "Active connections:"
echo "$list_response" | jq -r '.result.contents[0].text' 2>/dev/null || echo "$list_response"

print_status "Step 10: Check connection status..."
status_params="{\"uri\":\"connections://status\"}"
status_response=$(mcp_request "resources/read" "$status_params" 12)
check_response "$status_response" "Connection status"
echo "Connection status:"
echo "$status_response" | jq -r '.result.contents[0].text' 2>/dev/null || echo "$status_response"

print_status "Step 11: Update data..."
update_params="{\"name\":\"execute_statement\",\"arguments\":{\"connection_id\":\"test_sqlite\",\"statement\":\"UPDATE users SET email = 'alice.johnson@example.com' WHERE name = 'Alice Johnson'\"}}"
update_response=$(mcp_request "tools/call" "$update_params" 13)
check_response "$update_response" "Update data"
echo "Update results:"
echo "$update_response" | jq -r '.result.content[0].text' 2>/dev/null | jq '.' || echo "$update_response"

print_status "Step 12: Verify update..."
verify_params="{\"name\":\"execute_query\",\"arguments\":{\"connection_id\":\"test_sqlite\",\"query\":\"SELECT * FROM users WHERE name = 'Alice Johnson'\"}}"
verify_response=$(mcp_request "tools/call" "$verify_params" 14)
check_response "$verify_response" "Verify update"
echo "Updated record:"
echo "$verify_response" | jq -r '.result.content[0].text' 2>/dev/null | jq '.' || echo "$verify_response"

print_status "Step 13: Close connection..."
close_params="{\"name\":\"close_connection\",\"arguments\":{\"connection_id\":\"test_sqlite\"}}"
close_response=$(mcp_request "tools/call" "$close_params" 15)
check_response "$close_response" "Close connection"
echo "Connection closed: $close_response"

print_status "========================================="
print_status "All tests completed successfully! ✅"
print_status "========================================="

# Test summary
echo
print_status "Test Summary:"
echo "- ✅ Server startup and health check"
echo "- ✅ MCP protocol initialization"
echo "- ✅ SQLite connection creation"
echo "- ✅ Table creation and data insertion"
echo "- ✅ Query execution (SELECT)"
echo "- ✅ Parameterized queries"
echo "- ✅ Statement execution (INSERT, UPDATE)"
echo "- ✅ Schema information retrieval"
echo "- ✅ Connection management"
echo "- ✅ Resource listing and status checks"
echo "- ✅ Connection cleanup"

print_status "SQLite database created: $TEST_DB (will be cleaned up)"
print_status "usqlr successfully integrates with SQLite! 🎉"