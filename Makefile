# Makefile for usqlr - Server version of usql with MCP support

.PHONY: build build-musl test test-sqlite test-drivers test-db clean help db-start db-stop db-test db-list

# Binary names
BINARY_NAME=usqlr
BUILD_DIR=.
TESTS_DIR=tests

# Go build flags
LDFLAGS=-ldflags "-s -w"
MUSL_LDFLAGS=-ldflags "-s -w -linkmode external -extldflags '-static'"

# Default target
all: build

# Build the usqlr binary
build:
	@echo "Building usqlr..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/usqlr

# Build the usqlr binary with musl (static linking)
build-musl:
	@echo "Building usqlr with musl (static linking)..."
	CC=musl-gcc CGO_ENABLED=1 go build $(MUSL_LDFLAGS) -o $(BINARY_NAME) ./cmd/usqlr

# Run all tests
test: build test-sqlite test-drivers

# Run SQLite integration test
test-sqlite: build
	@echo "Running SQLite integration test..."
	cd $(TESTS_DIR) && ./test_sqlite_simple.sh

# Run multi-database driver test
test-drivers: build
	@echo "Running multi-database driver test..."
	cd $(TESTS_DIR) && ./test_drivers_simple.sh

# Test with a specific database (example: make test-db DSN="postgres://user:pass@localhost/db")
test-db: build
	@if [ -z "$(DSN)" ]; then \
		echo "Usage: make test-db DSN=\"your-database-dsn\""; \
		echo "Example: make test-db DSN=\"postgres://user:pass@localhost/mydb\""; \
		exit 1; \
	fi
	@echo "Testing connection to: $(DSN)"
	@cd $(TESTS_DIR) && ./test_connection.sh "$(DSN)"

# Run development server
dev: build
	@echo "Starting usqlr development server on port 8080..."
	./$(BINARY_NAME) --port 8080

# Run server with custom config
run-config: build
	@if [ -z "$(CONFIG)" ]; then \
		echo "Usage: make run-config CONFIG=path/to/config.yaml"; \
		exit 1; \
	fi
	@echo "Starting usqlr server with config: $(CONFIG)"
	./$(BINARY_NAME) --config $(CONFIG)

# Database environment management
db-start:
	@if [ -z "$(DB)" ]; then \
		echo "Usage: make db-start DB=<database>"; \
		echo "       make db-start DB=test (starts primary test databases)"; \
		echo "       make db-start DB=all (starts all databases)"; \
		echo "Example: make db-start DB=postgres"; \
		cd $(TESTS_DIR) && ./setup_databases.sh list; \
	else \
		cd $(TESTS_DIR) && ./setup_databases.sh start $(DB) $(UPDATE); \
	fi

db-stop:
	@if [ -z "$(DB)" ]; then \
		echo "Usage: make db-stop DB=<database>"; \
		echo "       make db-stop DB=all (stops all databases)"; \
	else \
		cd $(TESTS_DIR) && ./setup_databases.sh stop $(DB); \
	fi

db-test: build
	@if [ -z "$(DB)" ]; then \
		echo "Usage: make db-test DB=<database>"; \
		echo "Example: make db-test DB=postgres"; \
	else \
		cd $(TESTS_DIR) && ./setup_databases.sh test $(DB); \
	fi

db-list:
	@echo "Available database configurations:"
	@cd $(TESTS_DIR) && ./setup_databases.sh list

db-status:
	@cd $(TESTS_DIR) && ./setup_databases.sh status

# Clean build artifacts and test files
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -f $(TESTS_DIR)/*.db
	rm -f $(TESTS_DIR)/*.log
	rm -f $(TESTS_DIR)/*.pid

# Show available targets
help:
	@echo "Available targets:"
	@echo "  build        - Build the usqlr binary"
	@echo "  build-musl   - Build the usqlr binary with musl (static linking)"
	@echo "  test         - Run all tests"
	@echo "  test-sqlite  - Run SQLite integration test"
	@echo "  test-drivers - Run multi-database driver test"
	@echo "  test-db      - Test specific database (requires DSN parameter)"
	@echo "  dev          - Run development server on port 8080"
	@echo "  run-config   - Run server with custom config (requires CONFIG parameter)"
	@echo "  db-start     - Start database containers (requires DB parameter)"
	@echo "  db-stop      - Stop database containers (requires DB parameter)"
	@echo "  db-test      - Test database connection (requires DB parameter)"
	@echo "  db-list      - List available database configurations"
	@echo "  db-status    - Show status of database containers"
	@echo "  clean        - Clean build artifacts and test files"
	@echo "  help         - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make build-musl"
	@echo "  make test"
	@echo "  make test-db DSN=\"sqlite3://test.db\""
	@echo "  make db-start DB=postgres"
	@echo "  make db-start DB=test UPDATE=-u"
	@echo "  make db-test DB=mysql"
	@echo "  make db-stop DB=all"
	@echo "  make run-config CONFIG=\"config/usqlr.yaml\""
	@echo "  make dev"