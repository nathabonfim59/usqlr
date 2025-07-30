# Makefile for usqlr - Server version of usql with MCP support

.PHONY: build test test-sqlite test-drivers clean help

# Binary names
BINARY_NAME=usqlr
BUILD_DIR=.
TESTS_DIR=tests

# Go build flags
LDFLAGS=-ldflags "-s -w"

# Default target
all: build

# Build the usqlr binary
build:
	@echo "Building usqlr..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/usqlr

# Run all tests
test: build test-sqlite test-drivers

# Run SQLite integration test
test-sqlite: build
	@echo "Running SQLite integration test..."
	cd $(TESTS_DIR) && ./test_sqlite.sh

# Run multi-database driver test
test-drivers: build
	@echo "Running multi-database driver test..."
	cd $(TESTS_DIR) && ./test_drivers.sh

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
	@echo "  test         - Run all tests"
	@echo "  test-sqlite  - Run SQLite integration test"
	@echo "  test-drivers - Run multi-database driver test"
	@echo "  test-db      - Test specific database (requires DSN parameter)"
	@echo "  dev          - Run development server on port 8080"
	@echo "  run-config   - Run server with custom config (requires CONFIG parameter)"
	@echo "  clean        - Clean build artifacts and test files"
	@echo "  help         - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make test"
	@echo "  make test-db DSN=\"sqlite3://test.db\""
	@echo "  make run-config CONFIG=\"config/usqlr.yaml\""
	@echo "  make dev"