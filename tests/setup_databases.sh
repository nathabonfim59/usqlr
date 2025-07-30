#!/bin/bash

# Database Environment Setup for usqlr Testing
# This script uses the existing contrib configurations from usql

set -e

# Source common functions
source ./common.sh

# Configuration
CONTRIB_DIR="../contrib"
CONTAINER_RUNTIME=""

# Function to detect container runtime
detect_runtime() {
    if command -v podman &> /dev/null; then
        CONTAINER_RUNTIME="podman"
        print_status "Using Podman as container runtime"
    elif command -v docker &> /dev/null; then
        CONTAINER_RUNTIME="docker"
        print_status "Using Docker as container runtime"
    else
        print_error "Neither Docker nor Podman found. Please install one of them."
        exit 1
    fi
}

# Function to show usage
show_usage() {
    echo "Usage: $0 <command> [options]"
    echo ""
    echo "Commands:"
    echo "  start <database>    Start a specific database container"
    echo "  start test          Start primary testing databases (mysql, postgres, sqlserver, oracle, clickhouse, cassandra)"
    echo "  start all           Start all available database containers"
    echo "  stop <database>     Stop a specific database container" 
    echo "  stop all            Stop all database containers"
    echo "  list                List available database configurations"
    echo "  list-running        List currently running containers"
    echo "  status              Show status of all database containers"
    echo "  dsn <database>      Show the DSN for a database"
    echo "  test <database>     Test connection to a database"
    echo ""
    echo "Options:"
    echo "  -u, --update        Pull/update container images before starting"
    echo ""
    echo "Examples:"
    echo "  $0 start postgres"
    echo "  $0 start test -u"
    echo "  $0 dsn mysql"
    echo "  $0 test postgres"
    echo "  $0 stop all"
}

# Function to get available databases
get_available_databases() {
    find "$CONTRIB_DIR" -name "podman-config" -type f | \
        sed "s|$CONTRIB_DIR/||g" | \
        sed 's|/podman-config||g' | \
        sort
}

# Function to check if database config exists
check_database_config() {
    local db="$1"
    if [ ! -f "$CONTRIB_DIR/$db/podman-config" ]; then
        print_error "Database configuration not found: $db"
        print_info "Available databases:"
        get_available_databases | sed 's/^/  /'
        exit 1
    fi
}

# Function to get DSN for a database
get_dsn() {
    local db="$1"
    check_database_config "$db"
    
    if [ -f "$CONTRIB_DIR/$db/usql-config" ]; then
        local dsn=$(grep "^DB=" "$CONTRIB_DIR/$db/usql-config" | cut -d'"' -f2)
        echo "$dsn"
    else
        print_warning "No usql-config found for $db"
        return 1
    fi
}

# Function to start databases using contrib script
start_databases() {
    local target="$1"
    local update_flag="$2"
    
    print_status "Starting database containers: $target"
    
    # Change to contrib directory and run the existing script
    cd "$CONTRIB_DIR"
    
    if [ "$update_flag" = "-u" ]; then
        ./podman-run.sh "$target" -u
    else
        ./podman-run.sh "$target"
    fi
    
    cd - > /dev/null
    
    # Wait a bit for containers to fully start
    print_info "Waiting for containers to fully initialize..."
    sleep 5
}

# Function to stop containers
stop_containers() {
    local target="$1"
    
    if [ "$target" = "all" ]; then
        print_status "Stopping all database containers..."
        
        # Get all running containers with database names
        local containers
        if [ "$CONTAINER_RUNTIME" = "podman" ]; then
            containers=$(podman ps --format "{{.Names}}" | grep -E "(postgres|mysql|sqlserver|oracle|clickhouse|cassandra|mongo|redis)" || true)
        else
            containers=$(docker ps --format "{{.Names}}" | grep -E "(postgres|mysql|sqlserver|oracle|clickhouse|cassandra|mongo|redis)" || true)
        fi
        
        if [ -n "$containers" ]; then
            echo "$containers" | while read -r container; do
                print_info "Stopping container: $container"
                $CONTAINER_RUNTIME stop "$container" || true
            done
        else
            print_info "No database containers currently running"
        fi
    else
        check_database_config "$target"
        print_status "Stopping container: $target"
        $CONTAINER_RUNTIME stop "$target" || print_warning "Container $target was not running"
    fi
}

# Function to list running containers
list_running() {
    print_status "Currently running database containers:"
    
    if [ "$CONTAINER_RUNTIME" = "podman" ]; then
        podman ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}" | \
            head -1
        podman ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}" | \
            grep -E "(postgres|mysql|sqlserver|oracle|clickhouse|cassandra|mongo|redis)" || print_info "No database containers running"
    else
        docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}" | \
            head -1
        docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}" | \
            grep -E "(postgres|mysql|sqlserver|oracle|clickhouse|cassandra|mongo|redis)" || print_info "No database containers running"
    fi
}

# Function to show container status
show_status() {
    print_status "Database container status:"
    
    local databases
    databases=$(get_available_databases)
    
    for db in $databases; do
        local status
        if [ "$CONTAINER_RUNTIME" = "podman" ]; then
            status=$(podman ps -a --filter "name=$db" --format "{{.Status}}" 2>/dev/null || echo "Not created")
        else
            status=$(docker ps -a --filter "name=$db" --format "{{.Status}}" 2>/dev/null || echo "Not created")
        fi
        
        if [[ "$status" == *"Up"* ]]; then
            echo "  ✅ $db: $status"
        elif [[ "$status" == "Not created" ]]; then
            echo "  ⚪ $db: $status"
        else
            echo "  ❌ $db: $status"
        fi
    done
}

# Function to test database connection
test_connection() {
    local db="$1"
    check_database_config "$db"
    
    local dsn
    dsn=$(get_dsn "$db")
    
    if [ $? -eq 0 ]; then
        print_status "Testing connection to $db..."
        print_info "DSN: $dsn"
        
        # Use the connection test script
        ./test_connection.sh "$dsn"
    else
        print_error "Could not get DSN for $db"
        exit 1
    fi
}

# Main script
main() {
    detect_runtime
    
    if [ $# -eq 0 ]; then
        show_usage
        exit 1
    fi
    
    local command="$1"
    shift
    
    case "$command" in
        "start")
            if [ $# -eq 0 ]; then
                print_error "Please specify which database(s) to start"
                show_usage
                exit 1
            fi
            
            local target="$1"
            local update_flag=""
            
            if [ $# -gt 1 ] && [[ "$2" =~ ^(-u|--update)$ ]]; then
                update_flag="-u"
            fi
            
            start_databases "$target" "$update_flag"
            ;;
            
        "stop")
            if [ $# -eq 0 ]; then
                print_error "Please specify which database(s) to stop"
                show_usage
                exit 1
            fi
            
            stop_containers "$1"
            ;;
            
        "list")
            print_status "Available database configurations:"
            get_available_databases | sed 's/^/  /'
            ;;
            
        "list-running")
            list_running
            ;;
            
        "status")
            show_status
            ;;
            
        "dsn")
            if [ $# -eq 0 ]; then
                print_error "Please specify database name"
                exit 1
            fi
            
            local dsn
            dsn=$(get_dsn "$1")
            if [ $? -eq 0 ]; then
                echo "$dsn"
            fi
            ;;
            
        "test")
            if [ $# -eq 0 ]; then
                print_error "Please specify database name"
                exit 1
            fi
            
            test_connection "$1"
            ;;
            
        "help"|"-h"|"--help")
            show_usage
            ;;
            
        *)
            print_error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"