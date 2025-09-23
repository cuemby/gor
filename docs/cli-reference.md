# Gor CLI Reference

Complete reference for the Gor command-line interface.

## Installation

```bash
go install github.com/cuemby/gor/cmd/gor@latest
```

## Global Options

```
gor [command] [options]

Global Options:
  -h, --help      Show help
  -v, --version   Show version
  --env ENV       Set environment (default: development)
  --config FILE   Specify configuration file
```

## Commands Overview

- **Application Commands**
  - `new` - Create a new Gor application
  - `server` - Start the development server
  - `console` - Start interactive console
  - `routes` - Display application routes
  - `build` - Build for production
  - `deploy` - Deploy to production

- **Generator Commands**
  - `generate` (or `g`) - Generate code
    - `model` - Generate a model
    - `controller` - Generate a controller
    - `scaffold` - Generate complete CRUD
    - `migration` - Generate a migration
    - `job` - Generate a background job
    - `mailer` - Generate a mailer
    - `channel` - Generate a cable channel

- **Database Commands**
  - `db` - Database management
    - `create` - Create database
    - `drop` - Drop database
    - `migrate` - Run migrations
    - `rollback` - Rollback migrations
    - `seed` - Load seed data
    - `status` - Show migration status
    - `reset` - Drop and recreate database

- **Testing Commands**
  - `test` (or `t`) - Run tests

## Command Details

### Application Commands

#### `gor new`

Create a new Gor application.

```bash
gor new APP_NAME [options]

Options:
  --database DB    Database adapter (sqlite3, postgres, mysql) [default: sqlite3]
  --skip-bundle    Skip bundle installation
  --api            API-only application
  --no-git         Skip git initialization
  --template URL   Use custom template

Examples:
  # Create a standard application
  gor new blog

  # Create with PostgreSQL
  gor new blog --database postgres

  # Create API-only application
  gor new api_app --api
```

#### `gor server`

Start the development server with hot reload.

```bash
gor server [options]
gor s       # Short alias

Options:
  -p, --port PORT     Port to listen on [default: 3000]
  -b, --binding IP    IP to bind to [default: 0.0.0.0]
  --pid FILE          PID file location
  --no-reload         Disable hot reload
  --debug             Enable debug mode

Examples:
  # Start on default port 3000
  gor server

  # Start on custom port
  gor server -p 8080

  # Bind to localhost only
  gor server -b 127.0.0.1
```

#### `gor console`

Start an interactive console with your application loaded.

```bash
gor console [options]
gor c       # Short alias

Options:
  --sandbox    Rollback database changes on exit

Examples:
  # Start console
  gor console

  # Start in sandbox mode
  gor console --sandbox
```

#### `gor routes`

Display all application routes.

```bash
gor routes [options]

Options:
  --grep PATTERN    Filter routes by pattern
  --controller C    Show only routes for controller
  --format FORMAT   Output format (table, json, csv) [default: table]

Examples:
  # Show all routes
  gor routes

  # Filter by pattern
  gor routes --grep user

  # Show only PostsController routes
  gor routes --controller Posts
```

#### `gor build`

Build the application for production.

```bash
gor build [options]

Options:
  -o, --output FILE    Output binary name [default: app]
  --static            Include static assets in binary
  --compress          Compress binary with UPX
  --ldflags FLAGS     Custom ldflags for build

Examples:
  # Basic build
  gor build

  # Build with custom name
  gor build -o myapp

  # Build with embedded assets
  gor build --static
```

#### `gor deploy`

Deploy application to production.

```bash
gor deploy [ENVIRONMENT] [options]

Options:
  --host HOST       Deployment host
  --user USER       SSH user
  --path PATH       Remote path
  --restart         Restart after deploy

Examples:
  # Deploy to production
  gor deploy production

  # Deploy to staging
  gor deploy staging --host staging.example.com
```

### Generator Commands

#### `gor generate model`

Generate a model with optional fields.

```bash
gor generate model NAME [field:type ...] [options]
gor g model NAME    # Short alias

Field Types:
  string      - String field
  text        - Text field (long string)
  integer     - Integer field
  float       - Float field
  decimal     - Decimal field
  boolean     - Boolean field
  date        - Date field
  datetime    - DateTime field
  time        - Time field
  json        - JSON field
  uuid        - UUID field
  references  - Foreign key reference

Modifiers:
  :index      - Add database index
  :unique     - Add unique constraint
  :null       - Allow NULL values

Options:
  --skip-migration    Don't generate migration
  --skip-tests       Don't generate tests

Examples:
  # Basic model
  gor generate model User name:string email:string

  # With associations
  gor generate model Post title:string body:text user:references

  # With modifiers
  gor generate model User email:string:unique:index

  # Complex example
  gor generate model Article \
    title:string \
    slug:string:unique:index \
    body:text \
    published_at:datetime:null \
    author:references \
    tags:json \
    view_count:integer
```

#### `gor generate controller`

Generate a controller with specified actions.

```bash
gor generate controller NAME [actions...] [options]
gor g controller NAME    # Short alias

Standard Actions:
  index    - List resources
  show     - Show single resource
  new      - New resource form
  create   - Create resource
  edit     - Edit resource form
  update   - Update resource
  destroy  - Delete resource

Options:
  --skip-tests     Don't generate tests
  --skip-views     Don't generate views
  --parent NAME    Nested under parent controller

Examples:
  # Controller with all RESTful actions
  gor generate controller Posts

  # Controller with specific actions
  gor generate controller Posts index show

  # API controller (no views)
  gor generate controller API::Posts --skip-views

  # Nested controller
  gor generate controller Comments --parent Posts
```

#### `gor generate scaffold`

Generate complete CRUD scaffolding (model + controller + views).

```bash
gor generate scaffold NAME [field:type ...] [options]
gor g scaffold NAME    # Short alias

Options:
  --skip-tests        Don't generate tests
  --skip-migration    Don't generate migration
  --api              Generate API controller (no views)

Examples:
  # Blog scaffold
  gor generate scaffold Post \
    title:string \
    body:text \
    published:boolean

  # User scaffold with associations
  gor generate scaffold User \
    name:string \
    email:string:unique \
    posts:has_many

  # API scaffold
  gor generate scaffold Product \
    name:string \
    price:decimal \
    --api
```

#### `gor generate migration`

Generate a database migration.

```bash
gor generate migration NAME [field:type ...] [options]
gor g migration NAME    # Short alias

Migration Patterns:
  create_TABLE       - Create new table
  add_COLUMN_to_TABLE - Add column
  remove_COLUMN_from_TABLE - Remove column
  change_COLUMN_in_TABLE - Modify column
  add_index_to_TABLE - Add index
  remove_index_from_TABLE - Remove index

Examples:
  # Create table
  gor generate migration create_users \
    name:string \
    email:string:unique

  # Add column
  gor generate migration add_age_to_users age:integer

  # Remove column
  gor generate migration remove_age_from_users age:integer

  # Add index
  gor generate migration add_index_to_users email:index

  # Custom migration
  gor generate migration update_user_roles
```

#### `gor generate job`

Generate a background job.

```bash
gor generate job NAME [options]
gor g job NAME    # Short alias

Options:
  --queue QUEUE    Queue name [default: default]
  --priority N     Job priority (1-10)

Examples:
  # Basic job
  gor generate job SendEmail

  # With custom queue
  gor generate job ProcessImage --queue images

  # High priority job
  gor generate job CriticalTask --priority 10
```

#### `gor generate mailer`

Generate a mailer for sending emails.

```bash
gor generate mailer NAME [methods...] [options]
gor g mailer NAME    # Short alias

Options:
  --skip-tests    Don't generate tests
  --skip-views    Don't generate email templates

Examples:
  # Basic mailer
  gor generate mailer UserMailer

  # With specific methods
  gor generate mailer UserMailer welcome password_reset

  # Notification mailer
  gor generate mailer NotificationMailer \
    comment_added \
    post_published \
    weekly_digest
```

#### `gor generate channel`

Generate a cable channel for real-time features.

```bash
gor generate channel NAME [options]
gor g channel NAME    # Short alias

Options:
  --skip-tests       Don't generate tests
  --connection TYPE  Connection type (websocket, sse) [default: websocket]

Examples:
  # Basic channel
  gor generate channel Chat

  # With custom connection
  gor generate channel Notifications --connection sse

  # Room channel
  gor generate channel Room
```

### Database Commands

#### `gor db create`

Create the database for current environment.

```bash
gor db create [options]

Options:
  --all    Create all databases (development, test, production)

Examples:
  # Create development database
  gor db create

  # Create all databases
  gor db create --all
```

#### `gor db drop`

Drop the database for current environment.

```bash
gor db drop [options]

Options:
  --all    Drop all databases
  --force  Skip confirmation

Examples:
  # Drop development database
  gor db drop

  # Force drop without confirmation
  gor db drop --force
```

#### `gor db migrate`

Run pending database migrations.

```bash
gor db migrate [options]

Options:
  --version VERSION    Migrate to specific version
  --step N            Run N migrations
  --dry-run           Show SQL without running

Examples:
  # Run all pending migrations
  gor db migrate

  # Migrate to specific version
  gor db migrate --version 20240101120000

  # Run next 3 migrations
  gor db migrate --step 3

  # Preview migrations
  gor db migrate --dry-run
```

#### `gor db rollback`

Rollback database migrations.

```bash
gor db rollback [options]

Options:
  --step N     Rollback N migrations [default: 1]
  --to VERSION Rollback to specific version

Examples:
  # Rollback last migration
  gor db rollback

  # Rollback last 3 migrations
  gor db rollback --step 3

  # Rollback to specific version
  gor db rollback --to 20240101120000
```

#### `gor db seed`

Load seed data into database.

```bash
gor db seed [options]

Options:
  --file FILE    Specific seed file to run

Examples:
  # Run all seeds
  gor db seed

  # Run specific seed file
  gor db seed --file users
```

#### `gor db status`

Show migration status.

```bash
gor db status [options]

Options:
  --pending    Show only pending migrations
  --applied    Show only applied migrations

Examples:
  # Show all migrations
  gor db status

  # Show pending only
  gor db status --pending
```

#### `gor db reset`

Drop, create, and migrate database.

```bash
gor db reset [options]

Options:
  --seed    Also run seed data
  --force   Skip confirmation

Examples:
  # Reset database
  gor db reset

  # Reset and seed
  gor db reset --seed
```

### Testing Commands

#### `gor test`

Run application tests.

```bash
gor test [PATH] [options]
gor t        # Short alias

Options:
  --coverage        Generate coverage report
  --verbose         Verbose output
  --failfast        Stop on first failure
  --pattern PATTERN Run tests matching pattern
  --exclude PATTERN Exclude tests matching pattern

Examples:
  # Run all tests
  gor test

  # Run specific test file
  gor test test/models/user_test.go

  # Run with coverage
  gor test --coverage

  # Run tests matching pattern
  gor test --pattern User

  # Run verbose with failfast
  gor test --verbose --failfast
```

## Environment Variables

```bash
# Set environment
GOR_ENV=production gor server

# Database URL
DATABASE_URL=postgres://user:pass@localhost/myapp gor db migrate

# Custom config
GOR_CONFIG=/path/to/config.yml gor server
```

## Configuration Files

Gor looks for configuration in these locations:
1. `./config/config.yml`
2. `./Gorfile`
3. `~/.gor/config.yml`
4. Environment variables

## Examples

### Complete Workflow

```bash
# Create new app
gor new blog
cd blog

# Generate models
gor g model User name:string email:string:unique
gor g model Post title:string body:text user:references

# Run migrations
gor db migrate

# Generate controllers
gor g controller Users
gor g scaffold Comments body:text post:references

# Start server
gor server
```

### API Development

```bash
# Create API app
gor new api --api

# Generate API resources
gor g scaffold Product name:string price:decimal --api
gor g scaffold Order user:references total:decimal --api

# Run tests
gor test --coverage
```

### Background Jobs

```bash
# Generate jobs
gor g job SendWelcomeEmail
gor g job ProcessPayment --queue payments
gor g job CleanupOldRecords --queue maintenance

# Run worker
gor jobs work --queue payments,maintenance
```

## Tips and Tricks

1. **Use aliases**: `g` for generate, `s` for server, `c` for console, `t` for test
2. **Tab completion**: Install shell completions with `gor completion`
3. **Custom templates**: Create `.gor/templates/` for custom generators
4. **Environment shortcuts**: `development`, `test`, and `production` can be shortened to `dev`, `test`, and `prod`

## Getting Help

```bash
# General help
gor help

# Command-specific help
gor help generate
gor generate help model

# Version information
gor version --verbose
```

## See Also

- [Getting Started Guide](./getting-started.md)
- [API Reference](./api.md)
- [Testing Guide](./testing-guide.md)
- [Deployment Guide](./deployment.md)