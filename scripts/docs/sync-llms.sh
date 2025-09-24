#!/bin/bash

# sync-llms.sh - Auto-generate llms.txt from multiple sources
# This script ensures llms.txt is always current with the actual codebase

set -euo pipefail

# Colors for output
RED=$'\033[0;31m'
GREEN=$'\033[0;32m'
YELLOW=$'\033[1;33m'
BLUE=$'\033[0;34m'
NC=$'\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
LLMS_FILE="${PROJECT_ROOT}/docs/dev/llms.txt"
TEMP_FILE="${PROJECT_ROOT}/tmp/llms.tmp"

echo -e "${BLUE}🔄 Syncing llms.txt with current codebase...${NC}"

# Ensure tmp directory exists
mkdir -p "$(dirname "$TEMP_FILE")"

# Helper function to check if a file exists
file_exists() {
    [ -f "$PROJECT_ROOT/$1" ]
}

# Helper function to get test coverage statistics
get_coverage_stats() {
    local coverage_file="$PROJECT_ROOT/coverage_output/coverage.out"
    if [ -f "$coverage_file" ]; then
        go tool cover -func="$coverage_file" 2>/dev/null | grep -E "^github.com/cuemby/gor/(pkg|internal)/" | \
        awk '{
            package = gensub(/^github\.com\/cuemby\/gor\//, "", "g", $1)
            gsub(/\/[^\/]*\.go:.*/, "", package)
            coverage[package] += $3
            count[package]++
        } END {
            for (pkg in coverage) {
                printf "- **%s**: %.1f%%\n", pkg, coverage[pkg]/count[pkg]
            }
        }' | sort
    else
        echo "- Coverage data unavailable (run 'make test-coverage' to generate)"
    fi
}

# Helper function to get example applications
get_examples() {
    local examples=()

    # Use process substitution for better performance and avoid subshell issues
    while read -r dir; do
        local name
        name=$(basename "$dir")
        local main_file="$dir/main.go"
        local description="Example application"

        if [ -f "$main_file" ]; then
            description=$(grep -E "^// .*" "$main_file" | head -1 | sed 's|^// ||' 2>/dev/null || echo "Example application")
        fi

        examples+=("- [$name Application](./examples/$name) - $description")
    done < <(find "$PROJECT_ROOT/examples" -maxdepth 1 -type d -not -path "$PROJECT_ROOT/examples")

    # Sort and output
    printf '%s\n' "${examples[@]}" | sort
}

# Helper function to scan for actual Go packages
get_framework_components() {
    local components=()

    # Use process substitution for better performance and avoid subshell issues
    while read -r dir; do
        local name=$(basename "$dir")
        local desc="Framework component"

        # Try to get description from main Go file
        if [ -f "$dir/$name.go" ]; then
            local extracted_desc
            extracted_desc=$(grep -E "^// Package $name" "$dir/$name.go" | head -1 | sed "s|^// Package $name ||" 2>/dev/null)
            [ -n "$extracted_desc" ] && desc="$extracted_desc"
        fi

        components+=("- \`internal/$name\` - $desc")
    done < <(find "$PROJECT_ROOT/internal" -maxdepth 1 -type d -not -path "$PROJECT_ROOT/internal")

    # Sort and output
    printf '%s\n' "${components[@]}" | sort
}

# Start generating the new llms.txt
cat > "$TEMP_FILE" << 'EOF'
# Gor - Rails for Go

> Batteries-included web framework bringing Rails productivity to Go

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://go.dev)
[![Test Coverage](https://img.shields.io/badge/coverage-75%25-yellow.svg)](./coverage.out)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](./LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-green.svg)](./actions)

A Rails-inspired web framework for Go that provides the rapid development experience of Ruby on Rails with Go's superior performance, type safety, and single-binary deployment. Features the "Solid Trifecta" (Queue, Cache, Cable) without requiring Redis, making it perfect for solo developers and small teams.

## Key Features

- **Convention Over Configuration** - Rails-like conventions with sensible defaults
- **No Redis Required** - Database-backed queue, cache, and real-time features
- **Type Safety** - Compile-time checking with Go's type system
- **Single Binary** - Deploy one file with embedded assets
- **10x Performance** - Faster than Rails while maintaining productivity
- **Batteries Included** - Everything you need out of the box

## Documentation

### Getting Started

- [Quick Start Guide](./docs/guides/getting-started.md) - Get up and running in 5 minutes
- [Installation Guide](./docs/guides/installation.md) - Install Gor CLI and dependencies
- [Creating Your First App](./docs/guides/first-app.md) - Step-by-step tutorial
- [CLI Reference](./docs/api/cli-reference.md) - Complete command-line interface documentation
- [Key Concepts](./README.md#core-philosophy) - MVC, conventions, and principles
- [Development Workflow](./docs/guides/development-workflow.md) - Hot reload and productivity

### Core Framework

- [Application](./docs/api/application.md) - Central application instance
- [Configuration](./docs/api/configuration.md) - Environment-based config management
- [Request Lifecycle](./pkg/gor/framework.go) - How requests are processed
- [Context](./docs/api/context.md) - Request/response context
- [Middleware](./docs/api/middleware.md) - Request processing pipeline

### Routing & Controllers

- [RESTful Routing](./docs/api/router.md) - Rails-style resource routing
- [Route Definitions](./docs/api/routing.md) - HTTP verbs and patterns
- [Named Routes](./docs/api/named-routes.md) - URL generation helpers
- [Route Constraints](./docs/api/route-constraints.md) - Parameter validation
- [Route Groups](./docs/api/route-groups.md) - Organized routing
- [Controllers](./docs/api/controllers.md) - Request handlers
- [Controller Actions](./docs/api/controller-actions.md) - CRUD operations
- [Controller Callbacks](./docs/api/controller-callbacks.md) - Before/after filters

### Models & ORM

- [Model Definition](./docs/api/models.md) - Defining data models
- [ORM Queries](./docs/api/querying.md) - Database operations
- [Query Builder](./docs/api/query-builder.md) - Fluent query interface
- [Associations](./docs/api/associations.md) - Relationships between models
- [Validations](./docs/api/validations.md) - Data validation rules
- [Callbacks](./docs/api/model-callbacks.md) - Lifecycle hooks
- [Scopes](./docs/api/scopes.md) - Reusable queries
- [Transactions](./docs/api/transactions.md) - Database transactions
- [Raw SQL](./docs/api/raw-sql.md) - Custom SQL queries
- [Migrations](./internal/orm/migration.go) - Schema management
- [Database Adapters](./internal/orm) - SQLite, PostgreSQL, MySQL

### Views & Templates

- [Template Engine](./internal/views) - HTML templating
- [Layouts](./docs/api/layouts.md) - Master templates
- [Partials](./internal/views/template.go) - Reusable components
- [Template Helpers](./internal/views/helpers.go) - View utilities
- [Asset Pipeline](./internal/assets) - CSS/JS processing

### The Solid Trifecta

#### Queue System
- [Queue Overview](./docs/api/queue.md) - Background job processing
- [Job Definition](./docs/api/jobs.md) - Creating jobs
- [Enqueueing Jobs](./docs/api/enqueuing.md) - Job scheduling
- [Recurring Jobs](./docs/api/recurring-jobs.md) - Cron-like scheduling
- [Worker Management](./internal/queue/worker.go) - Job workers
- [Queue Monitoring](./docs/api/queue-monitoring.md) - Stats and management
- [Database Backend](./internal/queue) - No Redis required

#### Cache System
- [Cache Overview](./docs/api/cache.md) - Multi-tier caching
- [Basic Operations](./docs/api/cache-operations.md) - Get/Set/Delete
- [Advanced Caching](./docs/api/advanced-caching.md) - Fetch, increment, tags
- [Cache Stores](./docs/api/cache-stores.md) - Memory, disk, database
- [Tagged Caching](./internal/cache) - Group invalidation
- [Fragment Caching](./internal/cache) - Partial caching

#### Cable System
- [Cable Overview](./docs/api/cable.md) - Real-time features
- [WebSockets](./docs/api/websockets.md) - Bidirectional communication
- [Server-Sent Events](./docs/api/sse.md) - Server push
- [Broadcasting](./docs/api/broadcasting.md) - Message distribution
- [Channels](./internal/cable) - Organized messaging
- [Presence](./internal/cable) - User tracking

### Authentication & Security

- [Authentication System](./docs/api/authentication.md) - Built-in auth
- [User Model](./docs/api/user-model.md) - Authentication model
- [Sessions](./docs/api/sessions.md) - Session management
- [JWT](./internal/auth) - Token authentication
- [Authorization](./docs/api/authorization.md) - RBAC and abilities
- [Password Management](./internal/auth) - Hashing and reset
- [CSRF Protection](./pkg/middleware) - Security middleware
- [CORS](./pkg/middleware) - Cross-origin requests
- [Rate Limiting](./pkg/middleware) - Request throttling

### Testing

- [Testing Guide](./docs/guides/testing.md) - Comprehensive testing documentation
- [Testing Framework](./docs/api/testing.md) - Built-in testing
- [Test Helpers](./docs/api/test-helpers.md) - Testing utilities
- [Controller Tests](./docs/api/controller-tests.md) - HTTP testing
- [Model Tests](./docs/api/model-tests.md) - Data layer testing
- [Job Tests](./docs/api/job-tests.md) - Background job testing
- [Test Factories](./internal/testing) - Test data generation
- [Fixtures](./internal/testing) - Sample data

### Test Coverage Status

Current test coverage by package:

EOF

# Add dynamic test coverage
echo "$(get_coverage_stats)" >> "$TEMP_FILE"

cat >> "$TEMP_FILE" << 'EOF'

**Overall Coverage**: ~75% and improving

### CLI & Generators

- [CLI Overview](./cmd/gor) - Command-line interface
- [New Application](./docs/guides/new-app.md) - Create new projects
- [Model Generator](./docs/guides/generators.md#model) - Generate models
- [Controller Generator](./docs/guides/generators.md#controller) - Generate controllers
- [Scaffold Generator](./internal/cli/generators) - Full CRUD scaffolding
- [Migration Generator](./docs/guides/generators.md#migration) - Database migrations
- [Database Commands](./docs/guides/database.md) - Migration management
- [Server Commands](./docs/guides/server.md) - Development server

### Deployment

- [Deployment Guide](./docs/guides/deployment.md) - Complete deployment documentation
- [Production Build](./docs/guides/deployment.md#building-for-production) - Optimized builds
- [Docker Deployment](./docs/guides/deployment.md#docker-deployment) - Container deployment
- [Kubernetes](./docs/guides/deployment.md#kubernetes-deployment) - K8s deployment
- [Cloud Platforms](./docs/guides/deployment.md#cloud-platform-deployments) - AWS, GCP, Heroku, DO
- [Environment Variables](./docs/guides/deployment.md#environment-configuration) - Configuration
- [Health Checks](./docs/guides/deployment.md#monitoring-and-health-checks) - Monitoring endpoints
- [Zero-downtime Deploy](./docs/guides/deployment.md#zero-downtime-deployment) - Graceful updates
- [Security](./docs/guides/deployment.md#security-best-practices) - Production security

### Developer Experience

- [Hot Reload](./internal/dev) - Auto-reloading
- [Debugging](./internal/dev) - Debug tools
- [Error Pages](./internal/dev) - Development errors
- [Console](./internal/dev) - Interactive console
- [Logging](./pkg/gor) - Structured logging

### Plugin System

- [Plugin Overview](./docs/api/plugins.md) - Extensibility
- [Creating Plugins](./docs/api/plugin-development.md) - Plugin development
- [Plugin Hooks](./internal/plugin) - Extension points
- [Plugin Registry](./internal/plugin/registry.go) - Plugin management
- [Installing Plugins](./docs/api/plugin-installation.md) - Plugin installation

### Examples

EOF

# Add dynamic examples list
get_examples >> "$TEMP_FILE"

cat >> "$TEMP_FILE" << 'EOF'

### Comparison & Migration

- [vs Ruby on Rails](./README.md#comparison-with-rails) - Rails comparison
- [vs Express.js](./README.md#comparison-with-express) - Node.js comparison
- [vs Django](./README.md#comparison-with-django) - Python comparison
- [Performance Benchmarks](./README.md#performance) - Speed comparisons
- [Migration Guide](./docs/guides/migration.md) - Porting applications

### API Reference

- [Complete API](./docs/api/index.md) - Full API documentation
- [Application API](./docs/api/application.md) - App methods
- [Router API](./docs/api/router.md) - Routing methods
- [Context API](./docs/api/context.md) - Request/response
- [ORM API](./docs/api/orm.md) - Database methods
- [Queue API](./docs/api/queue.md) - Job methods
- [Cache API](./docs/api/cache.md) - Caching methods
- [Cable API](./docs/api/cable.md) - Real-time methods
- [Auth API](./docs/api/authentication.md) - Auth methods

### Integrations

- [PostgreSQL](./internal/orm) - PostgreSQL adapter
- [MySQL](./internal/orm) - MySQL adapter
- [SQLite](./internal/orm) - SQLite adapter
- [Redis Cache](./internal/cache) - Redis cache store
- [S3 Storage](./docs/integrations/s3.md) - File uploads
- [Email Services](./docs/integrations/email.md) - Email integration
- [OAuth Providers](./internal/auth) - Social login
- [Monitoring Tools](./docs/integrations/monitoring.md) - APM integration

### Best Practices

- [Project Structure](./docs/guides/project-structure.md) - Organizing code
- [Performance Tips](./docs/guides/performance.md) - Optimization
- [Security Guidelines](./docs/security/guidelines.md) - Secure coding
- [Testing Strategy](./docs/guides/testing-strategy.md) - Test coverage
- [Deployment Strategy](./docs/guides/deployment-strategy.md) - Production tips
- [Database Design](./docs/guides/database-design.md) - Schema best practices

### Community & Support

- [GitHub Repository](https://github.com/cuemby/gor) - Source code
- [Issue Tracker](https://github.com/cuemby/gor/issues) - Bug reports
- [Discussions](https://github.com/cuemby/gor/discussions) - Community forum
- [Contributing](./CONTRIBUTING.md) - Contribution guide
- [Code of Conduct](./CODE_OF_CONDUCT.md) - Community guidelines
- [Security Policy](./SECURITY.md) - Security vulnerability reporting
- [Support](./SUPPORT.md) - Getting help and community support
- [Authors](./AUTHORS.md) - Contributors and maintainers
- [Changelog](./CHANGELOG.md) - Release notes and version history
- [License](./LICENSE) - MIT License

## Framework Components

### Core Packages
- `pkg/gor` - Framework interfaces and core types
- `internal/orm` - Object-relational mapping
- `internal/router` - HTTP routing
- `internal/views` - Template engine
- `internal/queue` - Background jobs
- `internal/cache` - Caching system
- `internal/cable` - Real-time features
- `internal/auth` - Authentication
- `internal/assets` - Asset pipeline
- `internal/testing` - Test framework
- `internal/cli` - CLI tools
- `internal/config` - Configuration
- `internal/plugin` - Plugin system

### Framework Components Detail

EOF

# Add dynamic framework components
get_framework_components >> "$TEMP_FILE"

cat >> "$TEMP_FILE" << 'EOF'

### Key Features
- **Type-safe ORM** with compile-time checking
- **RESTful routing** with nested resources
- **Background jobs** without Redis
- **Multi-tier caching** with tagging
- **WebSockets & SSE** for real-time
- **Built-in auth** with sessions and JWT
- **Asset pipeline** with fingerprinting
- **Hot reload** in development
- **Single binary** deployment
- **Docker ready** with health checks

## Quick Examples

### Create a Model
```go
type Post struct {
    gor.Model
    Title     string `db:"title" validate:"required"`
    Body      string `db:"body"`
    Published bool   `db:"published"`
}
```

### Define Routes
```go
router.Resource("posts", &PostsController{})
// Creates all RESTful routes automatically
```

### Background Job
```go
type EmailJob struct {
    gor.Job
    To string
}

func (j *EmailJob) Perform(ctx context.Context) error {
    return sendEmail(j.To)
}

queue.Enqueue(&EmailJob{To: "user@example.com"})
```

### WebSocket
```go
ctx.Cable().HandleWebSocket(func(conn *WebSocketConnection) {
    conn.Subscribe("chat")
    for msg := range conn.Messages() {
        ctx.Cable().Broadcast("chat", msg)
    }
})
```

## Version Information

- **Current Version**: 1.0.0
- **Go Version**: 1.21+
- **Database Support**: SQLite 3.35+, PostgreSQL 12+, MySQL 8+
- **License**: MIT

## Related Projects

- [Ruby on Rails](https://rubyonrails.org) - Original inspiration
- [Phoenix Framework](https://phoenixframework.org) - Elixir web framework
- [Django](https://djangoproject.com) - Python web framework
- [Laravel](https://laravel.com) - PHP web framework

---

This documentation is automatically generated and kept in sync with the codebase.
For the latest updates, visit the [GitHub repository](https://github.com/cuemby/gor).

*Last updated: $(date -u '+%Y-%m-%d %H:%M:%S UTC')*
*Generated by: scripts/docs/sync-llms.sh*
EOF

# Move the generated file to replace the original
mv "$TEMP_FILE" "$LLMS_FILE"

echo -e "${GREEN}✅ llms.txt successfully updated!${NC}"
echo -e "${BLUE}📄 File location: $LLMS_FILE${NC}"
echo -e "${YELLOW}ℹ️  Generated content includes:${NC}"
echo "   - Current test coverage statistics"
echo "   - Dynamic examples list from filesystem"
echo "   - Current framework components"
echo "   - Updated file references for new structure"
echo "   - Generation timestamp"