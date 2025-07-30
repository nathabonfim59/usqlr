#!/bin/bash

# Common functions for usqlr tests
# Source this file in your test scripts: source ./common.sh

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Global variables
SERVER_PID=""
DEFAULT_TIMEOUT=10

# Function to print colored output
print_status() {
    echo -e "${GREEN}[TEST]${NC} $1"
}

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Function to check if usqlr binary exists
check_binary() {
    if [ ! -f ../usqlr ]; then
        print_error "usqlr binary not found. Run 'make build' first."
        exit 1
    fi
}

# Function to make MCP requests
mcp_request() {
    local method="$1"
    local params="$2"
    local id="$3"
    local server_port="${4:-8080}"
    
    if [ -z "$params" ]; then
        local json_data="{\"jsonrpc\":\"2.0\",\"method\":\"$method\",\"id\":$id}"
    else
        local json_data="{\"jsonrpc\":\"2.0\",\"method\":\"$method\",\"params\":$params,\"id\":$id}"
    fi
    
    curl -s -X POST -H "Content-Type: application/json" \
         -d "$json_data" \
         "http://localhost:$server_port/mcp"
}

# Function to wait for server to be ready
wait_for_server() {
    local server_port="${1:-8080}"
    local timeout="${2:-$DEFAULT_TIMEOUT}"
    local count=0
    
    print_info "Waiting for server on port $server_port..."
    while [ $count -lt $timeout ]; do
        if curl -s "http://localhost:$server_port/health" > /dev/null 2>&1; then
            print_status "Server is ready!"
            return 0
        fi
        sleep 1
        count=$((count + 1))
    done
    
    print_error "Server failed to start within $timeout seconds"
    return 1
}

# Function to start usqlr server
start_server() {
    local port="$1"
    local log_file="$2"
    local pid_file="$3"
    
    print_status "Starting usqlr server on port $port..."
    ../usqlr --port $port > "$log_file" 2>&1 &
    SERVER_PID=$!
    echo $SERVER_PID > "$pid_file"
    
    wait_for_server "$port"
}

# Function to stop server
stop_server() {
    local pid_file="$1"
    
    if [ -n "$SERVER_PID" ]; then
        print_info "Stopping server (PID: $SERVER_PID)..."
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        kill $pid 2>/dev/null || true
        rm -f "$pid_file"
    fi
}

# Function to cleanup test files
cleanup_test_files() {
    local log_file="$1"
    local pid_file="$2"
    
    stop_server "$pid_file"
    rm -f "$log_file" "$pid_file" *.db
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

# Function to test database connection creation
test_connection_creation() {
    local connection_id="$1"
    local dsn="$2"
    local request_id="$3"
    local server_port="${4:-8080}"
    
    print_info "Testing connection creation: $connection_id"
    
    local create_params="{\"name\":\"create_connection\",\"arguments\":{\"connection_id\":\"$connection_id\",\"dsn\":\"$dsn\"}}"
    local response=$(mcp_request "tools/call" "$create_params" "$request_id" "$server_port")
    
    if echo "$response" | grep -q '"error"'; then
        print_warning "Connection creation failed: $connection_id"
        echo "  DSN: $dsn"
        echo "  Error: $(echo "$response" | jq -r '.error.message' 2>/dev/null || echo "$response")"
        return 1
    else
        print_status "Connection created successfully: $connection_id ✅"
        return 0
    fi
}

# Function to execute a query
execute_query() {
    local connection_id="$1"
    local query="$2"
    local request_id="$3"
    local server_port="${4:-8080}"
    
    local query_params="{\"name\":\"execute_query\",\"arguments\":{\"connection_id\":\"$connection_id\",\"query\":\"$query\"}}"
    local response=$(mcp_request "tools/call" "$query_params" "$request_id" "$server_port")
    
    echo "$response"
}

# Function to execute a statement
execute_statement() {
    local connection_id="$1"
    local statement="$2"
    local request_id="$3"
    local server_port="${4:-8080}"
    
    local stmt_params="{\"name\":\"execute_statement\",\"arguments\":{\"connection_id\":\"$connection_id\",\"statement\":\"$statement\"}}"
    local response=$(mcp_request "tools/call" "$stmt_params" "$request_id" "$server_port")
    
    echo "$response"
}

# Function to close connection
close_connection() {
    local connection_id="$1"
    local request_id="$2"
    local server_port="${3:-8080}"
    
    local close_params="{\"name\":\"close_connection\",\"arguments\":{\"connection_id\":\"$connection_id\"}}"
    local response=$(mcp_request "tools/call" "$close_params" "$request_id" "$server_port")
    
    echo "$response"
}

# Function to list connections
list_connections() {
    local request_id="$1"
    local server_port="${2:-8080}"
    
    local list_params="{\"uri\":\"connections://list\"}"
    local response=$(mcp_request "resources/read" "$list_params" "$request_id" "$server_port")
    
    echo "$response"
}

# Function to check connection status
check_connection_status() {
    local request_id="$1"
    local server_port="${2:-8080}"
    
    local status_params="{\"uri\":\"connections://status\"}"
    local response=$(mcp_request "resources/read" "$status_params" "$request_id" "$server_port")
    
    echo "$response"
}

# Function to print test header
print_test_header() {
    local test_name="$1"
    echo
    print_status "========================================="
    print_status "$test_name"
    print_status "========================================="
}

# Function to print test summary
print_test_summary() {
    local total="$1"
    local successful="$2"
    local failed="$3"
    
    echo
    print_status "========================================="
    print_status "Test Summary"
    print_status "========================================="
    print_info "Total tests: $total"
    print_status "Successful: $successful"
    if [ "$failed" -gt 0 ]; then
        print_warning "Failed: $failed"
    else
        print_status "Failed: $failed"
    fi
}

# Function to initialize MCP protocol
initialize_mcp() {
    local request_id="$1"
    local server_port="${2:-8080}"
    
    print_info "Initializing MCP protocol..."
    local response=$(mcp_request "initialize" "" "$request_id" "$server_port")
    
    if check_response "$response" "MCP Initialize"; then
        print_status "MCP protocol initialized successfully"
        return 0
    else
        return 1
    fi
}

# Function to get available tools
get_tools() {
    local request_id="$1"
    local server_port="${2:-8080}"
    
    local response=$(mcp_request "tools/list" "" "$request_id" "$server_port")
    echo "$response"
}