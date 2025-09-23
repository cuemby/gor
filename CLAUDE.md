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

## Common Development Tasks

### Building and Running

```bash
# Build the CLI tool
go build -o bin/gor ./cmd/gor
# OR use Makefile
make build

# Run the CLI
./bin/gor --help

# Development server with hot reload
make dev

# Run examples
make run-webapp
make run-auth
make run-solid
make run-blog
make run-realtime
```

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

### Test Coverage Status (Current)

**High Coverage (80%+)**:
- **pkg/middleware**: 93.1% ✅
- **internal/router**: 98.9% ✅
- **internal/cache**: 88.6% ✅
- **internal/auth**: 86.6% ✅
- **internal/cable**: 85.1% ✅
- **internal/sse**: 84.5% ✅
- **internal/queue**: 84.4% ✅
- **internal/views**: 82.8% ✅
- **internal/cli**: 81.3% ✅
- **internal/app/controllers**: 100.0% ✅

**Medium Coverage (50-80%)**:
- **pkg/gor**: 66.3%
- **internal/config**: 48.2%
- **internal/websocket**: 47.9%

**Low Coverage (<50%)**:
- **internal/assets**: 43.2%
- **internal/plugin**: 28.5%
- **internal/orm**: 27.9%
- **internal/dev**: 19.0%
- **internal/testing**: 11.0%
- **internal/deploy**: 6.5%

**Overall Coverage**: ~75% and improving

### Coverage Improvement Priorities
1. **internal/orm** - Core data layer needs comprehensive testing
2. **internal/deploy** - Production deployment reliability
3. **internal/testing** - Test framework itself needs tests
4. **internal/dev** - Development tools testing

### Example Applications

1. **webapp**: Full-featured web application with templates and database
2. **auth_demo**: Authentication system demonstration
3. **solid_trifecta**: Queue, Cache, and Cable systems demo
4. **blog**: Simple blog application
5. **testing_demo**: Testing framework examples
6. **template_app**: Template rendering demonstration

## The Solid Trifecta (No Redis Required)

Gor features a complete \"Solid Trifecta\" implementation inspired by Rails 8, providing Queue, Cache, and Cable functionality without external dependencies:

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
```
app/
├── controllers/     # HTTP request handlers
├── models/         # Data models and business logic
├── views/          # Templates and view logic
├── jobs/           # Background job definitions
└── middleware/     # Custom middleware

config/             # Configuration files
db/                 # Database migrations and seeds
public/             # Static assets
```

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
7. Commit changes: `git add . && git commit -m \"feat: description\"`
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
```"}

## Dependencies

### Runtime Dependencies
- **Go 1.21+** (minimum version for Go features used)
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
