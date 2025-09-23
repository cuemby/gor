// Package gor provides the core framework interfaces and types for the Gor web framework.
// Gor is a Rails-inspired "batteries included" web framework built in Go, designed for
// rapid development with strong conventions and type safety.
package gor

import (
	"context"
	"net/http"
	"time"
)

// Application represents the main Gor application instance.
// This is the central hub that coordinates all framework components.
type Application interface {
	// Start launches the application server
	Start(ctx context.Context) error

	// Stop gracefully shuts down the application
	Stop(ctx context.Context) error

	// Router returns the HTTP router instance
	Router() Router

	// ORM returns the database/ORM instance
	ORM() ORM

	// Queue returns the background job queue
	Queue() Queue

	// Cache returns the caching layer
	Cache() Cache

	// Cable returns the real-time messaging system
	Cable() Cable

	// Auth returns the authentication system
	Auth() interface{} // Will be Auth interface once defined

	// Config returns the application configuration
	Config() Config
}

// Router defines the HTTP routing interface following Rails-style conventions.
type Router interface {
	// RESTful resource routing
	Resources(name string, controller Controller) Router
	Resource(name string, controller Controller) Router

	// Manual route definitions
	GET(path string, handler HandlerFunc) Router
	POST(path string, handler HandlerFunc) Router
	PUT(path string, handler HandlerFunc) Router
	PATCH(path string, handler HandlerFunc) Router
	DELETE(path string, handler HandlerFunc) Router

	// Route groups and namespaces
	Namespace(prefix string, fn func(Router)) Router
	Group(middleware ...MiddlewareFunc) Router

	// Middleware
	Use(middleware ...MiddlewareFunc) Router

	// Named routes for URL generation
	Named(name string) Router
}

// Controller defines the base interface for all controllers.
// Controllers handle HTTP requests and coordinate between models and views.
type Controller interface {
	// Index lists all resources (GET /)
	Index(ctx *Context) error

	// Show displays a specific resource (GET /:id)
	Show(ctx *Context) error

	// New displays form for creating new resource (GET /new)
	New(ctx *Context) error

	// Create processes creation of new resource (POST /)
	Create(ctx *Context) error

	// Edit displays form for editing resource (GET /:id/edit)
	Edit(ctx *Context) error

	// Update processes resource updates (PUT/PATCH /:id)
	Update(ctx *Context) error

	// Destroy deletes a resource (DELETE /:id)
	Destroy(ctx *Context) error
}

// HandlerFunc represents a Gor handler function.
type HandlerFunc func(ctx *Context) error

// MiddlewareFunc represents middleware that can be applied to routes.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// Context provides request/response context with Gor-specific enhancements.
type Context struct {
	// Embedded standard context for Go patterns
	context.Context

	// HTTP request and response
	Request  *http.Request
	Response http.ResponseWriter

	// Route parameters and query values
	Params map[string]string
	Query  map[string][]string

	// User and authentication info
	User interface{}

	// Flash messages for redirects
	Flash map[string]interface{}

	// Application services
	app Application
}

// Param returns a route parameter value.
func (c *Context) Param(key string) string {
	return c.Params[key]
}

// QueryParam returns a query parameter value.
func (c *Context) QueryParam(key string) string {
	values := c.Query[key]
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// JSON renders a JSON response.
func (c *Context) JSON(status int, data interface{}) error {
	// Implementation will be added in router package
	return nil
}

// Render renders a template with data.
func (c *Context) Render(template string, data interface{}) error {
	// Implementation will be added in views package
	return nil
}

// Redirect performs an HTTP redirect.
func (c *Context) Redirect(status int, url string) error {
	http.Redirect(c.Response, c.Request, url, status)
	return nil
}

// Config defines the application configuration interface.
type Config interface {
	// Environment returns the current environment (development, test, production)
	Environment() string

	// Get retrieves a configuration value
	Get(key string) interface{}

	// Set stores a configuration value
	Set(key string, value interface{})

	// Database returns database configuration
	Database() DatabaseConfig

	// Server returns server configuration
	Server() ServerConfig
}

// DatabaseConfig holds database-specific configuration.
type DatabaseConfig struct {
	Driver   string
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string

	// Connection pool settings
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// Framework lifecycle hooks
type LifecycleHook interface {
	BeforeStart(app Application) error
	AfterStart(app Application) error
	BeforeStop(app Application) error
	AfterStop(app Application) error
}

// Plugin interface for extending framework functionality
type Plugin interface {
	Name() string
	Version() string
	Install(app Application) error
	Uninstall(app Application) error
}