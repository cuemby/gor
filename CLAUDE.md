# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Gor** is a Rails-inspired web framework for Go, designed to provide rapid development with strong conventions and type safety. The framework follows an MVC pattern and includes integrated components for ORM, authentication, caching, queuing, and real-time messaging.

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
go build -o gor ./cmd/gor

# Run the CLI
./gor --help

# Run examples
go run examples/webapp/main.go
go run examples/auth_demo/main.go
go run examples/solid_trifecta/main.go
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/orm/...
go test ./internal/testing/...

# Run example tests
go test ./examples/testing_demo/...
```

### Code Quality

```bash
# Format code
gofmt -w .

# Run go vet
go vet ./...

# Check for formatting issues
gofmt -l .

# Tidy dependencies
go mod tidy
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

## Known Issues to Fix

1. **Formatting**: Multiple files need `gofmt` formatting
2. **Vet Issues**: Several `fmt.Println` calls have redundant newlines
3. **Build Failures**: Some example applications have build issues
4. **Test Coverage**: Need to increase test coverage across internal packages

## Development Workflow

1. Make changes to interfaces in `pkg/gor/` if adding new features
2. Implement in corresponding `internal/` package
3. Add tests in same package or `_test.go` files
4. Create/update example in `examples/` to demonstrate usage
5. Run `gofmt -w .` before committing
6. Run `go vet ./...` to check for issues
7. Run `go test ./...` to ensure tests pass

## Dependencies

- Go 1.24.0+
- sqlite3 (github.com/mattn/go-sqlite3)
- gorilla/websocket for WebSocket support
- golang.org/x/crypto for security features
- gopkg.in/yaml.v3 for configuration

## Deployment

The framework supports single-binary deployment with embedded assets. Use `gor build` to create a production binary and `gor deploy` for orchestrated deployments.
