package cli

import (
	"fmt"
	"path/filepath"
)

// NewCommand creates a new Gor application
type NewCommand struct{}

// NewNewCommand creates a new 'new' command
func NewNewCommand() *NewCommand {
	return &NewCommand{}
}

// Name returns the command name
func (c *NewCommand) Name() string {
	return "new"
}

// Description returns the command description
func (c *NewCommand) Description() string {
	return "Create a new Gor application"
}

// Usage returns the command usage
func (c *NewCommand) Usage() string {
	return "gor new <app_name> [options]"
}

// Run executes the command
func (c *NewCommand) Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("application name is required")
	}

	appName := args[0]
	appPath := filepath.Join(".", appName)

	// Check if directory already exists
	if FileExists(appPath) {
		return fmt.Errorf("directory %s already exists", appName)
	}

	fmt.Printf("🚀 Creating new Gor application: %s\n\n", appName)

	// Create directory structure
	if err := c.createDirectoryStructure(appPath); err != nil {
		return err
	}

	// Generate files
	if err := c.generateFiles(appPath, appName); err != nil {
		return err
	}

	fmt.Printf(`
✅ Application created successfully!

Next steps:
  cd %s
  go mod tidy
  gor db create
  gor db migrate
  gor server

Visit http://localhost:3000
`, appName)

	return nil
}

func (c *NewCommand) createDirectoryStructure(appPath string) error {
	dirs := []string{
		"app/controllers",
		"app/models",
		"app/views/layouts",
		"app/views/shared",
		"app/views/home",
		"app/jobs",
		"app/mailers",
		"app/channels",
		"app/helpers",
		"assets/images",
		"assets/javascripts",
		"assets/stylesheets",
		"config/environments",
		"config/initializers",
		"config/locales",
		"db/migrations",
		"db/seeds",
		"lib/tasks",
		"log",
		"public",
		"test/controllers",
		"test/models",
		"test/fixtures",
		"test/integration",
		"tmp/cache",
		"tmp/sessions",
		"tmp/sockets",
		"vendor",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(appPath, dir)
		if err := CreateDirectory(fullPath); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("  ✓ Created %s\n", dir)
	}

	return nil
}

func (c *NewCommand) generateFiles(appPath, appName string) error {
	files := map[string]string{
		"go.mod":                                    c.goModContent(appName),
		"main.go":                                   c.mainGoContent(),
		"config/application.go":                     c.applicationContent(),
		"config/database.yml":                       c.databaseYmlContent(),
		"config/routes.go":                          c.routesContent(),
		"config/environments/development.go":        c.developmentEnvContent(),
		"config/environments/production.go":         c.productionEnvContent(),
		"config/environments/test.go":               c.testEnvContent(),
		"app/controllers/application_controller.go": c.applicationControllerContent(),
		"app/controllers/home_controller.go":        c.homeControllerContent(),
		"app/models/application_record.go":          c.applicationRecordContent(),
		"app/views/layouts/application.html":        c.layoutContent(),
		"app/views/home/index.html":                 c.homeViewContent(),
		"db/seeds/seeds.go":                         c.seedsContent(),
		"Gorfile":                                   c.gorfileContent(),
		".gitignore":                                c.gitignoreContent(),
		"README.md":                                 c.readmeContent(appName),
	}

	for path, content := range files {
		fullPath := filepath.Join(appPath, path)
		if err := WriteFile(fullPath, content); err != nil {
			return fmt.Errorf("failed to create file %s: %w", path, err)
		}
		fmt.Printf("  ✓ Created %s\n", path)
	}

	return nil
}

// File content generators

func (c *NewCommand) goModContent(appName string) string {
	return fmt.Sprintf(`module %s

go 1.21

require (
	github.com/cuemby/gor v1.0.0
)
`, appName)
}

func (c *NewCommand) mainGoContent() string {
	return `package main

import (
	"log"
	"os"

	"github.com/cuemby/gor/pkg/gor"
	"./config"
)

func main() {
	// Initialize application
	app := config.NewApplication()

	// Get port from environment or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Start server
	log.Printf("Starting Gor application on port %s", port)
	if err := app.Start(":" + port); err != nil {
		log.Fatal("Failed to start application:", err)
	}
}
`
}

func (c *NewCommand) applicationContent() string {
	return `package config

import (
	"context"

	"github.com/cuemby/gor/pkg/gor"
	"github.com/cuemby/gor/internal/orm"
	"github.com/cuemby/gor/internal/router"
	"github.com/cuemby/gor/internal/queue"
	"github.com/cuemby/gor/internal/cache"
	"github.com/cuemby/gor/internal/cable"
	"github.com/cuemby/gor/internal/auth"
)

type Application struct {
	router gor.Router
	orm    gor.ORM
	queue  *queue.SolidQueue
	cache  *cache.SolidCache
	cable  *cable.SolidCable
	auth   *auth.Authenticator
	config gor.Config
}

func NewApplication() *Application {
	app := &Application{}

	// Initialize components
	app.initializeDatabase()
	app.initializeCache()
	app.initializeQueue()
	app.initializeCable()
	app.initializeAuth()
	app.initializeRouter()

	return app
}

func (a *Application) initializeDatabase() {
	// Initialize ORM
	ormInstance, err := orm.NewORM("sqlite", "db/development.db")
	if err != nil {
		panic("Failed to initialize database: " + err.Error())
	}
	a.orm = ormInstance
}

func (a *Application) initializeCache() {
	// Initialize cache
	cacheInstance, err := cache.NewSolidCache("db/cache.db", 100)
	if err != nil {
		panic("Failed to initialize cache: " + err.Error())
	}
	a.cache = cacheInstance
}

func (a *Application) initializeQueue() {
	// Initialize queue
	queueInstance, err := queue.NewSolidQueue("db/queue.db", 5)
	if err != nil {
		panic("Failed to initialize queue: " + err.Error())
	}
	a.queue = queueInstance
}

func (a *Application) initializeCable() {
	// Initialize cable
	cableInstance, err := cable.NewSolidCable("db/cable.db")
	if err != nil {
		panic("Failed to initialize cable: " + err.Error())
	}
	a.cable = cableInstance
}

func (a *Application) initializeAuth() {
	// Initialize authenticator
	authInstance, err := auth.NewAuthenticator("db/auth.db")
	if err != nil {
		panic("Failed to initialize auth: " + err.Error())
	}
	a.auth = authInstance
}

func (a *Application) initializeRouter() {
	// Initialize router and load routes
	a.router = router.NewRouter(a)
	LoadRoutes(a.router)
}

func (a *Application) Start(addr string) error {
	ctx := context.Background()

	// Start background services
	a.queue.Start(ctx)

	// Start HTTP server
	return a.router.Listen(addr)
}

func (a *Application) Router() gor.Router { return a.router }
func (a *Application) ORM() gor.ORM { return a.orm }
func (a *Application) Queue() gor.Queue { return nil }
func (a *Application) Cache() gor.Cache { return nil }
func (a *Application) Cable() gor.Cable { return nil }
func (a *Application) Auth() interface{} { return a.auth }
func (a *Application) Config() gor.Config { return a.config }
func (a *Application) Stop(ctx context.Context) error { return nil }
`
}

func (c *NewCommand) databaseYmlContent() string {
	return `# Database configuration

default: &default
  adapter: sqlite3
  pool: 5
  timeout: 5000

development:
  <<: *default
  database: db/development.db

test:
  <<: *default
  database: db/test.db

production:
  <<: *default
  database: db/production.db
  pool: 25
`
}

func (c *NewCommand) routesContent() string {
	return `package config

import (
	"github.com/cuemby/gor/pkg/gor"
	"../app/controllers"
)

// LoadRoutes defines all application routes
func LoadRoutes(r gor.Router) {
	// Root route
	r.GET("/", controllers.HomeIndex)

	// RESTful resources
	// r.Resources("posts", &controllers.PostsController{})
	// r.Resources("users", &controllers.UsersController{})

	// API namespace
	// r.Namespace("/api", func(api gor.Router) {
	//     api.Namespace("/v1", func(v1 gor.Router) {
	//         v1.Resources("posts", &controllers.API.V1.PostsController{})
	//     })
	// })

	// Admin namespace
	// r.Namespace("/admin", func(admin gor.Router) {
	//     admin.Use(middleware.RequireAdmin())
	//     admin.Resources("users", &controllers.Admin.UsersController{})
	// })

	// Static files
	r.Static("/assets", "./public/assets")
}
`
}

func (c *NewCommand) developmentEnvContent() string {
	return `package environments

// Development environment configuration
var Development = map[string]interface{}{
	"debug":           true,
	"log_level":       "debug",
	"cache_enabled":   false,
	"asset_pipeline":  false,
	"hot_reload":      true,
	"database_log":    true,
}
`
}

func (c *NewCommand) productionEnvContent() string {
	return `package environments

// Production environment configuration
var Production = map[string]interface{}{
	"debug":           false,
	"log_level":       "info",
	"cache_enabled":   true,
	"asset_pipeline":  true,
	"hot_reload":      false,
	"database_log":    false,
}
`
}

func (c *NewCommand) testEnvContent() string {
	return `package environments

// Test environment configuration
var Test = map[string]interface{}{
	"debug":           false,
	"log_level":       "error",
	"cache_enabled":   false,
	"asset_pipeline":  false,
	"hot_reload":      false,
	"database_log":    false,
}
`
}

func (c *NewCommand) applicationControllerContent() string {
	return `package controllers

import (
	"github.com/cuemby/gor/pkg/gor"
)

// ApplicationController is the base controller for all controllers
type ApplicationController struct {
	gor.BaseController
}

// BeforeAction runs before every action
func (c *ApplicationController) BeforeAction(ctx *gor.Context) error {
	// Add common before filters here
	return nil
}

// AfterAction runs after every action
func (c *ApplicationController) AfterAction(ctx *gor.Context) error {
	// Add common after filters here
	return nil
}
`
}

func (c *NewCommand) homeControllerContent() string {
	return `package controllers

import (
	"net/http"
	"github.com/cuemby/gor/pkg/gor"
)

// HomeController handles the home page
type HomeController struct {
	ApplicationController
}

// HomeIndex renders the home page
func HomeIndex(ctx *gor.Context) error {
	return ctx.Render("home/index", map[string]interface{}{
		"title": "Welcome to Gor",
		"message": "Your Rails-inspired Go framework is ready!",
	})
}
`
}

func (c *NewCommand) applicationRecordContent() string {
	return `package models

import (
	"time"
)

// ApplicationRecord is the base model for all models
type ApplicationRecord struct {
	ID        uint      ` + "`gorm:\"primarykey\"`" + `
	CreatedAt time.Time
	UpdatedAt time.Time
}
`
}

func (c *NewCommand) layoutContent() string {
	return `<!DOCTYPE html>
<html>
<head>
    <title>{{.title}} | Gor Application</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            padding: 60px;
            text-align: center;
            max-width: 600px;
        }
        h1 { color: #333; margin-bottom: 20px; font-size: 3em; }
        p { color: #666; font-size: 1.2em; line-height: 1.6; }
        .logo { font-size: 5em; margin-bottom: 20px; }
    </style>
</head>
<body>
    {{template "content" .}}
</body>
</html>
`
}

func (c *NewCommand) homeViewContent() string {
	return `{{define "content"}}
<div class="container">
    <div class="logo">🚀</div>
    <h1>{{.title}}</h1>
    <p>{{.message}}</p>
    <p style="margin-top: 30px; font-size: 0.9em; color: #999;">
        Edit this page at: app/views/home/index.html
    </p>
</div>
{{end}}
`
}

func (c *NewCommand) seedsContent() string {
	return `package seeds

import (
	"log"
)

// Run executes all seed data
func Run() error {
	log.Println("Seeding database...")

	// Add your seed data here
	// Example:
	// user := &models.User{
	//     Name: "Admin User",
	//     Email: "admin@example.com",
	// }
	// db.Create(user)

	log.Println("Database seeded successfully")
	return nil
}
`
}

func (c *NewCommand) gorfileContent() string {
	return `# Gorfile - Gor application configuration

# Application settings
app_name: "` + "My Gor App" + `"
version: "1.0.0"

# Server settings
server:
  port: 3000
  host: "localhost"
  timeout: 30s

# Database settings
database:
  adapter: "sqlite3"
  database: "db/development.db"
  pool: 5
  timeout: 5000

# Background jobs
jobs:
  workers: 5
  queues:
    - default
    - mailers
    - urgent

# Cache settings
cache:
  store: "database"
  expires_in: 3600

# Session settings
session:
  store: "cookie"
  key: "_gor_session"
  secret: "change_me_in_production"
  expires_in: 86400

# Asset pipeline
assets:
  compile: true
  compress: true
  fingerprint: true
`
}

func (c *NewCommand) gitignoreContent() string {
	return `# Gor application

# Dependencies
/vendor/

# Database
*.db
*.db-journal
*.sqlite
*.sqlite3

# Logs
/log/*
*.log

# Temporary files
/tmp/*

# Cache
/cache/*

# Compiled assets
/public/assets/*

# Environment variables
.env
.env.local
.env.*.local

# IDE
.idea/
.vscode/
*.swp
*.swo
*~
.DS_Store

# Test coverage
coverage/
*.coverprofile

# Binary
/bin/
*.exe
`
}

func (c *NewCommand) readmeContent(appName string) string {
	return fmt.Sprintf(`# %s

A Gor Framework application.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- SQLite3 (for development)

### Installation

1. Install dependencies:
   `+"```bash\n   go mod tidy\n   ```"+`

2. Create and migrate the database:
   `+"```bash\n   gor db create\n   gor db migrate\n   ```"+`

3. Start the development server:
   `+"```bash\n   gor server\n   ```"+`

4. Visit http://localhost:3000

## Development

### Generate a new scaffold

`+"```bash\ngor generate scaffold Post title:string body:text published:boolean\n```"+`

### Run tests

`+"```bash\ngor test\n```"+`

### Console

`+"```bash\ngor console\n```"+`

## Deployment

`+"```bash\ngor build\ngor deploy production\n```"+`

## License

MIT
`, appName)
}
