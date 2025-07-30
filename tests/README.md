# usqlr Testing Framework

This directory contains comprehensive testing tools for usqlr, including database environment setup and integration tests.

## Quick Start

```bash
# Build usqlr
make build

# Run all tests
make test

# Start a PostgreSQL database for testing
make db-start DB=postgres

# Test the PostgreSQL connection
make db-test DB=postgres

# Stop all databases
make db-stop DB=all
```

## File Structure

```
tests/
├── README.md              # This file
├── common.sh               # Shared functions for all test scripts
├── setup_databases.sh      # Database environment management
├── test_sqlite_simple.sh   # SQLite integration test
├── test_drivers_simple.sh  # Multi-database driver test
└── test_connection.sh      # Single connection test
```

## Database Environment Setup

The `setup_databases.sh` script provides easy management of database containers using the existing usql contrib configurations.

### Available Commands

```bash
./setup_databases.sh <command> [options]
```

**Commands:**
- `start <database>` - Start a specific database container
- `start test` - Start primary testing databases (mysql, postgres, sqlserver, oracle, clickhouse, cassandra)
- `start all` - Start all available database containers
- `stop <database>` - Stop a specific database container
- `stop all` - Stop all database containers
- `list` - List available database configurations
- `list-running` - List currently running containers
- `status` - Show status of all database containers
- `dsn <database>` - Show the DSN for a database
- `test <database>` - Test connection to a database

**Options:**
- `-u, --update` - Pull/update container images before starting

### Supported Databases

The script supports all databases configured in `contrib/`:

- **SQL Databases**: postgres, mysql, sqlserver, oracle, oracle-enterprise
- **NoSQL**: cassandra, couchbase
- **Analytics**: clickhouse, presto, trino, vertica
- **Specialized**: ydb, h2, ignite, hive, exasol, firebird, flightsql
- **Big Data**: db2

## Makefile Integration

The Makefile provides convenient shortcuts:

### Build and Test
```bash
make build                    # Build usqlr binary
make test                     # Run all tests
make test-sqlite              # Run SQLite integration test
make test-drivers             # Run multi-database driver test
make test-db DSN="<dsn>"      # Test specific database connection
```

### Database Management
```bash
make db-list                  # List available databases
make db-status                # Show container status
make db-start DB=<database>   # Start database container
make db-stop DB=<database>    # Stop database container
make db-test DB=<database>    # Test database connection
```

### Development
```bash
make dev                      # Run development server on port 8080
make run-config CONFIG=<file> # Run with custom config
make clean                    # Clean build artifacts
make help                     # Show all available targets
```

## Test Scripts

### SQLite Integration Test (`test_sqlite_simple.sh`)

Comprehensive test that verifies:
- Server startup and health checks
- MCP protocol initialization
- SQLite connection creation
- Table creation and data insertion
- Query execution (SELECT with results)
- Parameterized queries
- Statement execution (INSERT, UPDATE)
- Connection management and cleanup
- Error handling

### Multi-Database Driver Test (`test_drivers_simple.sh`)

Tests DSN parsing and connection creation for all 70+ supported database drivers:
- File-based databases (work without servers)
- Network databases (verify DSN parsing)
- Cloud databases (BigQuery, Snowflake, etc.)
- NoSQL databases (Cassandra, CouchBase, etc.)

### Single Connection Test (`test_connection.sh`)

Simple test for validating a specific database connection:
```bash
./test_connection.sh "postgres://user:pass@localhost/db"
```

## Container Runtime Support

The setup automatically detects and uses:
- **Podman** (preferred if available)
- **Docker** (fallback)

## Usage Examples

### Start PostgreSQL and test it
```bash
# Start PostgreSQL container
make db-start DB=postgres

# Check if it's running
make db-status

# Test the connection
make db-test DB=postgres

# Stop when done
make db-stop DB=postgres
```

### Start multiple databases for comprehensive testing
```bash
# Start primary test databases
make db-start DB=test UPDATE=-u

# Run tests against multiple databases
make db-test DB=postgres
make db-test DB=mysql
make db-test DB=clickhouse

# Stop all when done
make db-stop DB=all
```

### Test a custom database connection
```bash
# Test SQLite (no container needed)
make test-db DSN="sqlite3://mytest.db"

# Test remote database
make test-db DSN="postgres://user:pass@remote-host:5432/mydb"
```

## Troubleshooting

### Container Issues
```bash
# Check container status
make db-status

# View running containers
./setup_databases.sh list-running

# Restart a problematic container
make db-stop DB=postgres
make db-start DB=postgres UPDATE=-u
```

### Connection Issues
```bash
# Get the DSN for a database
./setup_databases.sh dsn postgres

# Test connection manually
./test_connection.sh "postgres://postgres:P4ssw0rd@localhost"

# Check server logs
tail -f tests/*.log
```

### Build Issues
```bash
# Clean and rebuild
make clean
make build

# Check for missing dependencies
go mod tidy
```

## Security Note

The default passwords in the contrib configurations are for testing only:
- PostgreSQL: `P4ssw0rd`
- MySQL: `P4ssw0rd`
- SQL Server: `P4ssw0rd`

Never use these in production environments.

## Contributing

When adding new tests:
1. Use functions from `common.sh` to avoid code duplication
2. Follow the existing naming convention
3. Add appropriate cleanup in trap handlers
4. Update this README with new functionality
5. Test with both Docker and Podman if possible

## Related Documentation

- [usqlr Implementation Plan](../USQLR_IMPLEMENTATION_PLAN.md)
- [usql contrib configurations](../contrib/)
- [MCP Protocol Specification](https://spec.modelcontextprotocol.io/)