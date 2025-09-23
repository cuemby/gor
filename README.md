# Gor - Rails-Inspired Web Framework for Go

Gor is a "batteries included" web framework for Go, inspired by Ruby on Rails's productivity-focused approach while leveraging Go's performance, type safety, and concurrency advantages.

## Overview

Gor aims to provide solo developers and small teams with the same rapid development experience as Rails, but with Go's superior performance and deployment characteristics.

### Core Philosophy

- **Convention Over Configuration**: Sensible defaults for 80% of use cases
- **Batteries Included**: Essential components integrated out of the box
- **Type Safety**: Leverage Go's compile-time checking for reliability
- **Performance**: 10x+ faster than Rails while maintaining productivity
- **Simplicity**: Single binary deployment with embedded assets

## Architecture Overview

### Core Components

#### 1. Framework Core (`pkg/gor/framework.go`)

- **Application**: Central application instance coordinating all components
- **Router**: Rails-style RESTful routing with middleware support
- **Controller**: MVC pattern with standard CRUD actions
- **Context**: Enhanced request/response context with Gor-specific features
- **Config**: Environment-based configuration management

#### 2. ORM Layer (`pkg/gor/orm.go`)

- **Type-safe ORM**: Inspired by ActiveRecord but leveraging Go's type system
- **Query Builder**: Fluent interface for building database queries
- **Migrations**: Database schema versioning and management
- **Associations**: Support for relationships between models
- **Validation**: Model validation with custom rules
- **Scopes**: Reusable query logic

#### 3. Solid Trifecta (Rails Inspired)

##### Queue System (`pkg/gor/queue.go`)

- **Database-backed**: SQLite/PostgreSQL backed job processing
- **Worker Management**: Concurrent job processing with goroutines
- **Job Types**: Support for various job patterns (email, webhooks, cleanup)
- **Monitoring**: Real-time job statistics and monitoring
- **Recurring Jobs**: Cron-like scheduled job support

##### Cache System (`pkg/gor/cache.go`)

- **Multi-tier**: Memory → Disk → Database caching strategy
- **Tagged Caching**: Group invalidation with cache tags
- **Fragment Caching**: Partial template caching support
- **Performance**: High-performance caching with compression and encryption
- **Monitoring**: Cache hit/miss statistics and performance metrics

##### Cable System (`pkg/gor/cable.go`)

- **Real-time Messaging**: WebSocket and SSE support
- **Channel System**: Organized real-time communication
- **Presence**: User presence tracking
- **Broadcasting**: Efficient message broadcasting
- **Horizontal Scaling**: Cluster support for multiple nodes

#### 4. Authentication System (`pkg/gor/auth.go`)

- **Built-in Auth**: Complete authentication system out of the box
- **JWT & Sessions**: Support for both token and session-based auth
- **Role-based Access**: Flexible role and permission system
- **MFA Support**: Multi-factor authentication with TOTP/SMS
- **OAuth Integration**: Support for external OAuth providers
- **Security**: Password policies, rate limiting, audit logging

## Directory Structure

```
gor/
├── cmd/gor/                 # CLI tool and generators
├── pkg/
│   ├── gor/                 # Core framework interfaces
│   ├── middleware/          # HTTP middleware components
│   └── generators/          # Code generation tools
├── internal/
│   ├── app/
│   │   ├── controllers/     # Application controllers
│   │   ├── models/          # Data models
│   │   └── views/           # Templates and views
│   ├── config/              # Configuration management
│   ├── router/              # HTTP routing implementation
│   ├── orm/                 # ORM implementation
│   ├── queue/               # Background job system
│   ├── cache/               # Caching system
│   ├── cable/               # Real-time messaging
│   ├── auth/                # Authentication system
│   ├── assets/              # Asset pipeline
│   ├── testing/             # Testing framework
│   └── deployment/          # Deployment tools
├── examples/                # Example applications
└── docs/                    # Documentation
```

## Key Features Planned

### Developer Experience

- **Code Generators**: Scaffold controllers, models, and complete CRUD operations
- **Hot Reload**: Sub-second rebuild times during development
- **Built-in Testing**: Comprehensive testing utilities and fixtures
- **CLI Tool**: Rails-like command-line interface for common tasks

### Production Ready

- **Single Binary**: Deploy as a single executable with embedded assets
- **Docker Integration**: Containerized deployment with zero-downtime updates
- **Monitoring**: Built-in metrics, logging, and health checks
- **Security**: Rate limiting, CSRF protection, and security headers

### Performance

- **Goroutine-based**: Leverage Go's excellent concurrency model
- **Connection Pooling**: Efficient database connection management
- **Caching**: Multi-tier caching for optimal performance
- **Asset Optimization**: Minification, compression, and CDN support

## Current Status

✅ **Framework Architecture**: Core interfaces and principles defined
🔄 **ORM Layer**: In progress
⏳ **Router System**: Planned
⏳ **Template Engine**: Planned
⏳ **Solid Trifecta**: Planned
⏳ **Authentication**: Planned
⏳ **CLI Tools**: Planned

## Getting Started (Planned)

```bash
# Install Gor CLI
go install github.com/cuemby/gor/cmd/gor@latest

# Create new application
gor new myapp
cd myapp

# Generate a model
gor generate model User name:string email:string

# Generate a controller
gor generate controller Users

# Run the application
gor server

# Run database migrations
gor db migrate
```

## Comparison with Rails

| Feature | Rails | Gor | Advantage |
|---------|---------|-----|-----------|
| **Performance** | Ruby | Go | 10x+ faster |
| **Type Safety** | Dynamic | Static | Compile-time error detection |
| **Deployment** | Multiple files | Single binary | Simpler deployment |
| **Memory Usage** | High | Low | Better resource efficiency |
| **Concurrency** | Threads | Goroutines | Superior concurrency model |
| **Startup Time** | Slow | Fast | Quick application startup |
| **Learning Curve** | Moderate | Moderate | Similar ease of use |

## Goals

1. **Match Rails Productivity**: Achieve similar development speed as Rails
2. **Exceed Rails Performance**: 10x+ performance improvement
3. **Simplify Deployment**: Single binary with zero external dependencies
4. **Maintain Type Safety**: Leverage Go's compile-time guarantees
5. **Modern Architecture**: Cloud-native, container-ready applications

## Contributing

This framework is currently in early development. Contributions and feedback are welcome!

## License

MIT License - see LICENSE file for details.
