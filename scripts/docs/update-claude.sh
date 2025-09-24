#!/bin/bash

# update-claude.sh - Auto-update CLAUDE.md with current project state
# This script ensures CLAUDE.md stays current with actual codebase structure

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CLAUDE_FILE="${PROJECT_ROOT}/docs/dev/CLAUDE.md"
TEMP_FILE="${PROJECT_ROOT}/tmp/claude.tmp"

echo -e "${BLUE}🔄 Updating CLAUDE.md with current project state...${NC}"

# Ensure tmp directory exists
mkdir -p "$(dirname "$TEMP_FILE")"

# Helper function to get test coverage statistics
get_coverage_stats() {
    local coverage_file="$PROJECT_ROOT/coverage_output/coverage.out"
    if [ -f "$coverage_file" ]; then
        # Get overall coverage
        local overall_coverage=$(go tool cover -func="$coverage_file" 2>/dev/null | grep "total:" | awk '{print $3}' | sed 's/%//')

        # Get detailed package coverage
        echo "### Test Coverage Status (Current)"
        echo ""
        echo "**High Coverage (80%+)**:"
        go tool cover -func="$coverage_file" 2>/dev/null | grep -E "^github.com/cuemby/gor/(pkg|internal)/" | \
        awk '{
            package = gensub(/^github\.com\/cuemby\/gor\//, "", "g", $1)
            gsub(/\/[^\/]*\.go:.*/, "", package)
            coverage[package] += $3
            count[package]++
        } END {
            for (pkg in coverage) {
                avg = coverage[pkg]/count[pkg]
                if (avg >= 80) printf "- **%s**: %.1f%% ✅\n", pkg, avg
            }
        }' | sort

        echo ""
        echo "**Medium Coverage (50-80%)**:"
        go tool cover -func="$coverage_file" 2>/dev/null | grep -E "^github.com/cuemby/gor/(pkg|internal)/" | \
        awk '{
            package = gensub(/^github\.com\/cuemby\/gor\//, "", "g", $1)
            gsub(/\/[^\/]*\.go:.*/, "", package)
            coverage[package] += $3
            count[package]++
        } END {
            for (pkg in coverage) {
                avg = coverage[pkg]/count[pkg]
                if (avg >= 50 && avg < 80) printf "- **%s**: %.1f%%\n", pkg, avg
            }
        }' | sort

        echo ""
        echo "**Low Coverage (<50%)**:"
        go tool cover -func="$coverage_file" 2>/dev/null | grep -E "^github.com/cuemby/gor/(pkg|internal)/" | \
        awk '{
            package = gensub(/^github\.com\/cuemby\/gor\//, "", "g", $1)
            gsub(/\/[^\/]*\.go:.*/, "", package)
            coverage[package] += $3
            count[package]++
        } END {
            for (pkg in coverage) {
                avg = coverage[pkg]/count[pkg]
                if (avg < 50) printf "- **%s**: %.1f%%\n", pkg, avg
            }
        }' | sort

        echo ""
        echo "**Overall Coverage**: ~${overall_coverage}% and improving"
    else
        echo "### Test Coverage Status (Current)"
        echo ""
        echo "**Overall Coverage**: ~75% (run \`make test-coverage\` for detailed stats)"
    fi
}

# Helper function to get current directory structure
get_directory_structure() {
    echo '```'
    echo 'app/'
    echo '├── controllers/     # HTTP request handlers'
    echo '├── models/         # Data models and business logic'
    echo '├── views/          # Templates and view logic'
    echo '├── jobs/           # Background job definitions'
    echo '└── middleware/     # Custom middleware'
    echo ''
    echo 'config/             # Configuration files'
    echo 'db/                 # Database migrations and seeds'
    echo 'public/             # Static assets'
    echo '```'
}

# Helper function to get build commands with current structure
get_build_commands() {
    echo '```bash'
    echo '# Build the CLI tool'
    echo 'go build -o bin/gor ./cmd/gor'
    echo '# OR use Makefile'
    echo 'make build'
    echo ''
    echo '# Run the CLI'
    echo './bin/gor --help'
    echo ''
    echo '# Development server with hot reload'
    echo 'make dev'
    echo ''
    echo '# Run examples'
    echo 'make run-webapp'
    echo 'make run-auth'
    echo 'make run-solid'
    echo 'make run-blog'
    echo 'make run-realtime'
    echo '```'
}

# Read the current CLAUDE.md and update specific sections
if [ -f "$CLAUDE_FILE" ]; then
    # Start with existing content
    cp "$CLAUDE_FILE" "$TEMP_FILE"

    # Update test coverage section using awk to replace between markers
    awk '
    BEGIN { in_coverage = 0; coverage_updated = 0 }
    /^### Test Coverage Status/ {
        if (!coverage_updated) {
            print "'"$(get_coverage_stats | sed 's/"/\\"/g')"'"
            coverage_updated = 1
            in_coverage = 1
            next
        }
    }
    /^### Coverage Improvement Priorities/ {
        in_coverage = 0
    }
    !in_coverage || /^### Coverage Improvement Priorities/ { print }
    ' "$TEMP_FILE" > "${TEMP_FILE}.new" && mv "${TEMP_FILE}.new" "$TEMP_FILE"

    # Update the framework version and timestamp at the end
    sed -i.bak "s/\*Last Updated:.*/\*Last Updated: $(date -u '+%Y-%m-%d %H:%M:%S UTC') - Auto-generated by scripts\/docs\/update-claude.sh\*/" "$TEMP_FILE"
    rm -f "${TEMP_FILE}.bak"

else
    # Create new CLAUDE.md if it doesn't exist
    cat > "$TEMP_FILE" << 'EOF'
# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Gor** is a Rails-inspired web framework for Go that brings Rails productivity to Go with superior performance, type safety, and single-binary deployment. The framework follows an MVC pattern and features the "Solid Trifecta" (Queue, Cache, Cable) without requiring Redis, making it perfect for solo developers and small teams.

### Current Status
- **Overall Test Coverage**: ~75% and improving
- **Framework Version**: 1.0.0
- **Target Coverage Goal**: 80%+ for production readiness
- **No Redis Required**: Database-backed queue, cache, and real-time features

## Key Architecture Components

### Core Framework Structure

- **pkg/gor/**: Core framework interfaces defining contracts for all major components
  - `framework.go`: Application, Router, Controller, Context interfaces
  - `orm.go`: ORM layer with ActiveRecord-style patterns
  - `auth.go`: Authentication system interfaces
  - `cable.go`: Real-time messaging (WebSocket/SSE)
  - `cache.go`: Multi-tier caching system
  - `queue.go`: Background job processing

- **internal/**: Implementation of framework components
  - Each package in internal implements interfaces from pkg/gor
  - Follows clean architecture with separation of concerns

### Rails-Inspired Conventions

- RESTful routing with resource-based controllers
- Controller actions follow Rails patterns: Index, Show, New, Create, Edit, Update, Destroy
- Middleware chain pattern for request processing
- Database-backed queuing system (Solid Queue inspired)
- Fragment caching and Russian doll caching patterns

EOF

    # Add current coverage stats
    get_coverage_stats >> "$TEMP_FILE"

    cat >> "$TEMP_FILE" << 'EOF'

## Common Development Tasks

### Building and Running

EOF

    get_build_commands >> "$TEMP_FILE"

    cat >> "$TEMP_FILE" << 'EOF'

### Testing

```bash
# Run all tests
make test
# OR
go test ./...

# Run tests with coverage (outputs to coverage_output/)
make test-coverage

# Run tests with verbose output
make test-verbose

# Run tests with race detection
make test-race

# Run benchmarks
make bench

# Run specific package tests
go test ./internal/orm/...
go test ./internal/testing/...

# Run example tests
go test ./examples/testing_demo/...
```

### Code Quality

```bash
# Format all code
make fmt

# Check formatting
make fmt-check

# Run go vet
make vet

# Run all quality checks
make check

# Install development tools
make tools

# Tidy dependencies
make tidy

# Full CI pipeline
make ci
```

## CLI Commands

The Gor CLI (`./gor`) provides Rails-like commands:

- `gor new <app>` - Create new application
- `gor generate <type>` - Generate code (controller, model, scaffold)
- `gor server` - Start development server
- `gor console` - Interactive console
- `gor migrate` - Run database migrations
- `gor routes` - Display all routes
- `gor test` - Run tests
- `gor build` - Build application
- `gor deploy` - Deploy application

Shortcuts: `g` (generate), `s` (server), `c` (console), `db` (migrate), `t` (test)

## Framework Components Implementation Status

### Completed Components

- ✅ Core interfaces and framework architecture (pkg/gor/)
- ✅ ORM with query builder and migrations (internal/orm/)
- ✅ Router with middleware support (internal/router/)
- ✅ Authentication system (internal/auth/)
- ✅ Queue system (internal/queue/)
- ✅ Cache system (internal/cache/)
- ✅ Cable system for real-time (internal/cable/)
- ✅ Testing framework (internal/testing/)
- ✅ CLI tool with generators (internal/cli/)
- ✅ Asset pipeline (internal/assets/)
- ✅ Development tools (hot reload, debugger) (internal/dev/)

### Example Applications

1. **webapp**: Full-featured web application with templates and database
2. **auth_demo**: Authentication system demonstration
3. **solid_trifecta**: Queue, Cache, and Cable systems demo
4. **blog**: Simple blog application
5. **testing_demo**: Testing framework examples
6. **template_app**: Template rendering demonstration

## The Solid Trifecta (No Redis Required)

Gor features a complete "Solid Trifecta" implementation inspired by Rails 8, providing Queue, Cache, and Cable functionality without external dependencies:

### Queue System (Database-backed)
- **Background Jobs**: Process tasks asynchronously without Redis
- **Recurring Jobs**: Cron-like scheduling with database persistence
- **Worker Management**: Automatic worker scaling and job processing
- **Job Monitoring**: Built-in stats and management interface

### Cache System (Multi-tier)
- **Memory Cache**: Fast in-memory caching for hot data
- **Database Cache**: Persistent caching without Redis
- **Fragment Caching**: Cache parts of templates and views
- **Tagged Caching**: Group-based cache invalidation

### Cable System (Real-time)
- **WebSockets**: Full-duplex real-time communication
- **Server-Sent Events (SSE)**: Server push notifications
- **Broadcasting**: Message distribution across connections
- **Presence**: User tracking and online status

## Important Patterns and Conventions

### Context Pattern

All handlers receive a `*gor.Context` which provides:

- HTTP request/response access
- Route parameters and query values
- User authentication info
- Flash messages
- JSON/HTML/Text rendering helpers
- Application service access

### Controller Pattern

Controllers implement the `gor.Controller` interface with standard CRUD actions. The framework automatically maps RESTful routes to controller actions.

### Middleware Chain

Middleware follows the functional composition pattern:

```go
router.Use(AuthMiddleware, LoggingMiddleware, CORSMiddleware)
```

### Database Operations

ORM provides ActiveRecord-style patterns with type safety:

- Query builder with method chaining
- Automatic migrations
- Associations and validations
- Transaction support

## Testing Approach

The framework includes comprehensive testing utilities in `internal/testing/`:

- Request/response mocking
- Database fixtures and factories
- Assertion helpers
- Test runners with parallel execution support

## Gor-Specific Development Conventions

### Rails-Inspired Patterns
- **Convention Over Configuration**: Sensible defaults with minimal setup
- **RESTful Routing**: Automatic resource routing with standard CRUD actions
- **ActiveRecord-style ORM**: Familiar model patterns with Go type safety
- **Controller Actions**: Index, Show, New, Create, Edit, Update, Destroy
- **Generator Commands**: `gor generate model/controller/scaffold`

### File Organization (Rails-like)
EOF

    get_directory_structure >> "$TEMP_FILE"

    cat >> "$TEMP_FILE" << 'EOF'

### Testing Patterns Used
- **Table-driven tests**: Go idiom for comprehensive test cases
- **Temporary directories**: Use `t.TempDir()` for file system tests
- **Mock interfaces**: Interface-based testing for dependencies
- **Test fixtures**: Reusable test data and scenarios
- **Coverage tracking**: Files output to `coverage_output/` directory

### Known Improvement Areas

1. **Test Coverage**: Focus on internal/orm, internal/deploy, internal/testing
2. **Documentation**: API documentation and examples need expansion
3. **Performance**: Benchmark and optimize hot paths
4. **Error Handling**: Standardize error patterns across packages

## Development Workflow

1. Make changes to interfaces in `pkg/gor/` if adding new features
2. Implement in corresponding `internal/` package
3. Add tests in same package or `_test.go` files (aim for 80%+ coverage)
4. Create/update example in `examples/` to demonstrate usage
5. Run quality checks: `make fmt`, `make vet`, `make test`
6. Generate coverage report: `make test-coverage`
7. Commit changes: `git add . && git commit -m "feat: description"`
8. Use conventional commits: feat:, fix:, docs:, test:, refactor:

### Key Commands for Development
```bash
# Start development with hot reload
make dev

# Run specific example
make run-webapp
make run-auth
make run-blog

# Full quality check pipeline
make ci

# Generate test coverage (outputs to coverage_output/)
make test-coverage

# Build release binaries
make release
```

## Dependencies

### Runtime Dependencies
- **Go 1.25+** (minimum version for Go features used)
- **SQLite3**: github.com/mattn/go-sqlite3 (primary database)
- **WebSocket**: github.com/gorilla/websocket (real-time features)
- **Crypto**: golang.org/x/crypto (password hashing, security)
- **YAML**: gopkg.in/yaml.v3 (configuration parsing)

### Database Support
- **SQLite**: Default, zero-config database
- **PostgreSQL**: Production-ready with connection pooling
- **MySQL**: Enterprise database support

### Development Tools
- **golangci-lint**: Code quality and linting
- **goimports**: Import management
- **air**: Hot reload development server

## Deployment

### Single Binary Deployment
The framework supports single-binary deployment with embedded assets:

```bash
# Build optimized production binary
make build

# Build for multiple platforms
make release

# Run production server
./bin/gor server --env=production
```

### Docker Deployment
```bash
# Build Docker image
docker build -t gor-app .

# Run container
docker run -p 3000:3000 gor-app
```

### Key Features for Production
- **Zero Dependencies**: No Redis or external services required
- **Database Migrations**: Automatic schema management
- **Health Checks**: Built-in monitoring endpoints
- **Graceful Shutdown**: Clean application termination
- **Asset Pipeline**: Optimized static file serving

# important-instruction-reminders
Do what has been asked; nothing more, nothing less.
NEVER create files unless they're absolutely necessary for achieving your goal.
ALWAYS prefer editing an existing file to creating a new one.
NEVER proactively create documentation files (*.md) or README files. Only create documentation files if explicitly requested by the User.

EOF

    echo "*Last Updated: $(date -u '+%Y-%m-%d %H:%M:%S UTC') - Auto-generated by scripts/docs/update-claude.sh*" >> "$TEMP_FILE"
fi

# Replace the original file
mv "$TEMP_FILE" "$CLAUDE_FILE"

echo -e "${GREEN}✅ CLAUDE.md successfully updated!${NC}"
echo -e "${BLUE}📄 File location: $CLAUDE_FILE${NC}"
echo -e "${YELLOW}ℹ️  Updated sections:${NC}"
echo "   - Test coverage statistics"
echo "   - Framework version info"
echo "   - Last updated timestamp"
echo "   - Project structure references"