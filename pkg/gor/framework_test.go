package gor_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cuemby/gor/pkg/gor"
)

// MockApplication implements gor.Application for testing
type MockApplication struct {
	router gor.Router
	orm    gor.ORM
	queue  gor.Queue
	cache  gor.Cache
	cable  gor.Cable
	auth   interface{}
	config gor.Config
}

func (a *MockApplication) Start(ctx context.Context) error {
	return nil
}

func (a *MockApplication) Stop(ctx context.Context) error {
	return nil
}

func (a *MockApplication) Router() gor.Router {
	return a.router
}

func (a *MockApplication) ORM() gor.ORM {
	return a.orm
}

func (a *MockApplication) Queue() gor.Queue {
	return a.queue
}

func (a *MockApplication) Cache() gor.Cache {
	return a.cache
}

func (a *MockApplication) Cable() gor.Cable {
	return a.cable
}

func (a *MockApplication) Auth() interface{} {
	return a.auth
}

func (a *MockApplication) Config() gor.Config {
	return a.config
}

// MockRouter implements gor.Router for testing
type MockRouter struct {
	handlers   map[string]gor.HandlerFunc
	middleware []gor.MiddlewareFunc
	routes     []string
}

func NewMockRouter() *MockRouter {
	return &MockRouter{
		handlers:   make(map[string]gor.HandlerFunc),
		middleware: make([]gor.MiddlewareFunc, 0),
		routes:     make([]string, 0),
	}
}

func (r *MockRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Simple implementation for testing
	key := req.Method + ":" + req.URL.Path
	if handler, ok := r.handlers[key]; ok {
		ctx := &gor.Context{
			Context:  context.Background(),
			Request:  req,
			Response: w,
			Params:   make(map[string]string),
			Query:    req.URL.Query(),
			Flash:    make(map[string]interface{}),
		}
		handler(ctx)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (r *MockRouter) Resources(name string, controller gor.Controller) gor.Router {
	r.routes = append(r.routes, "resources:"+name)
	return r
}

func (r *MockRouter) Resource(name string, controller gor.Controller) gor.Router {
	r.routes = append(r.routes, "resource:"+name)
	return r
}

func (r *MockRouter) GET(path string, handler gor.HandlerFunc) gor.Router {
	r.handlers["GET:"+path] = handler
	return r
}

func (r *MockRouter) POST(path string, handler gor.HandlerFunc) gor.Router {
	r.handlers["POST:"+path] = handler
	return r
}

func (r *MockRouter) PUT(path string, handler gor.HandlerFunc) gor.Router {
	r.handlers["PUT:"+path] = handler
	return r
}

func (r *MockRouter) PATCH(path string, handler gor.HandlerFunc) gor.Router {
	r.handlers["PATCH:"+path] = handler
	return r
}

func (r *MockRouter) DELETE(path string, handler gor.HandlerFunc) gor.Router {
	r.handlers["DELETE:"+path] = handler
	return r
}

func (r *MockRouter) Namespace(prefix string, fn func(gor.Router)) gor.Router {
	// Call the function with the router for testing
	fn(r)
	return r
}

func (r *MockRouter) Group(middleware ...gor.MiddlewareFunc) gor.Router {
	newRouter := &MockRouter{
		handlers:   make(map[string]gor.HandlerFunc),
		middleware: append(r.middleware, middleware...),
		routes:     make([]string, len(r.routes)),
	}
	copy(newRouter.routes, r.routes)
	return newRouter
}

func (r *MockRouter) Use(middleware ...gor.MiddlewareFunc) gor.Router {
	r.middleware = append(r.middleware, middleware...)
	return r
}

func (r *MockRouter) Named(name string) gor.Router {
	return r
}

// MockController implements gor.Controller for testing
type MockController struct {
	lastAction string
	lastCtx    *gor.Context
}

func (c *MockController) Index(ctx *gor.Context) error {
	c.lastAction = "index"
	c.lastCtx = ctx
	return nil
}

func (c *MockController) Show(ctx *gor.Context) error {
	c.lastAction = "show"
	c.lastCtx = ctx
	return nil
}

func (c *MockController) New(ctx *gor.Context) error {
	c.lastAction = "new"
	c.lastCtx = ctx
	return nil
}

func (c *MockController) Create(ctx *gor.Context) error {
	c.lastAction = "create"
	c.lastCtx = ctx
	return nil
}

func (c *MockController) Edit(ctx *gor.Context) error {
	c.lastAction = "edit"
	c.lastCtx = ctx
	return nil
}

func (c *MockController) Update(ctx *gor.Context) error {
	c.lastAction = "update"
	c.lastCtx = ctx
	return nil
}

func (c *MockController) Destroy(ctx *gor.Context) error {
	c.lastAction = "destroy"
	c.lastCtx = ctx
	return nil
}

// MockConfig implements gor.Config for testing
type MockConfig struct {
	environment  string
	values       map[string]interface{}
	dbConfig     gor.DatabaseConfig
	serverConfig gor.ServerConfig
}

func NewMockConfig() *MockConfig {
	return &MockConfig{
		environment: "test",
		values:      make(map[string]interface{}),
		dbConfig: gor.DatabaseConfig{
			Driver:   "sqlite3",
			Database: ":memory:",
		},
		serverConfig: gor.ServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

func (c *MockConfig) Environment() string {
	return c.environment
}

func (c *MockConfig) Get(key string) interface{} {
	return c.values[key]
}

func (c *MockConfig) Set(key string, value interface{}) {
	c.values[key] = value
}

func (c *MockConfig) Database() gor.DatabaseConfig {
	return c.dbConfig
}

func (c *MockConfig) Server() gor.ServerConfig {
	return c.serverConfig
}

// TestApplication tests the Application interface
func TestApplication(t *testing.T) {
	router := NewMockRouter()
	config := NewMockConfig()

	app := &MockApplication{
		router: router,
		config: config,
	}

	t.Run("Start", func(t *testing.T) {
		ctx := context.Background()
		err := app.Start(ctx)
		if err != nil {
			t.Errorf("Start() returned error: %v", err)
		}
	})

	t.Run("Stop", func(t *testing.T) {
		ctx := context.Background()
		err := app.Stop(ctx)
		if err != nil {
			t.Errorf("Stop() returned error: %v", err)
		}
	})

	t.Run("Router", func(t *testing.T) {
		result := app.Router()
		if result != router {
			t.Error("Router() should return the configured router")
		}
		if result == nil {
			t.Error("Router() should not return nil")
		}
	})

	t.Run("ORM", func(t *testing.T) {
		result := app.ORM()
		// ORM can be nil in this mock
		if result != app.orm {
			t.Error("ORM() should return the configured ORM")
		}
	})

	t.Run("Queue", func(t *testing.T) {
		result := app.Queue()
		// Queue can be nil in this mock
		if result != app.queue {
			t.Error("Queue() should return the configured queue")
		}
	})

	t.Run("Cache", func(t *testing.T) {
		result := app.Cache()
		// Cache can be nil in this mock
		if result != app.cache {
			t.Error("Cache() should return the configured cache")
		}
	})

	t.Run("Cable", func(t *testing.T) {
		result := app.Cable()
		// Cable can be nil in this mock
		if result != app.cable {
			t.Error("Cable() should return the configured cable")
		}
	})

	t.Run("Auth", func(t *testing.T) {
		result := app.Auth()
		// Auth can be nil in this mock
		if result != app.auth {
			t.Error("Auth() should return the configured auth")
		}
	})

	t.Run("Config", func(t *testing.T) {
		result := app.Config()
		if result != config {
			t.Error("Config() should return the configured config")
		}
		if result == nil {
			t.Error("Config() should not return nil")
		}
	})
}

// TestRouter tests the Router interface
func TestRouter(t *testing.T) {
	router := NewMockRouter()

	t.Run("GET", func(t *testing.T) {
		handler := func(ctx *gor.Context) error {
			ctx.Text(200, "test")
			return nil
		}

		result := router.GET("/test", handler)
		if result != router {
			t.Error("GET() should return the router for chaining")
		}

		// Test that the handler was registered
		if _, ok := router.handlers["GET:/test"]; !ok {
			t.Error("GET() should register the handler")
		}
	})

	t.Run("POST", func(t *testing.T) {
		handler := func(ctx *gor.Context) error { return nil }
		result := router.POST("/test", handler)
		if result != router {
			t.Error("POST() should return the router for chaining")
		}
		if _, ok := router.handlers["POST:/test"]; !ok {
			t.Error("POST() should register the handler")
		}
	})

	t.Run("PUT", func(t *testing.T) {
		handler := func(ctx *gor.Context) error { return nil }
		result := router.PUT("/test", handler)
		if result != router {
			t.Error("PUT() should return the router for chaining")
		}
		if _, ok := router.handlers["PUT:/test"]; !ok {
			t.Error("PUT() should register the handler")
		}
	})

	t.Run("PATCH", func(t *testing.T) {
		handler := func(ctx *gor.Context) error { return nil }
		result := router.PATCH("/test", handler)
		if result != router {
			t.Error("PATCH() should return the router for chaining")
		}
		if _, ok := router.handlers["PATCH:/test"]; !ok {
			t.Error("PATCH() should register the handler")
		}
	})

	t.Run("DELETE", func(t *testing.T) {
		handler := func(ctx *gor.Context) error { return nil }
		result := router.DELETE("/test", handler)
		if result != router {
			t.Error("DELETE() should return the router for chaining")
		}
		if _, ok := router.handlers["DELETE:/test"]; !ok {
			t.Error("DELETE() should register the handler")
		}
	})

	t.Run("Resources", func(t *testing.T) {
		controller := &MockController{}
		result := router.Resources("articles", controller)
		if result != router {
			t.Error("Resources() should return the router for chaining")
		}

		found := false
		for _, route := range router.routes {
			if route == "resources:articles" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Resources() should register the resource routes")
		}
	})

	t.Run("Resource", func(t *testing.T) {
		controller := &MockController{}
		result := router.Resource("profile", controller)
		if result != router {
			t.Error("Resource() should return the router for chaining")
		}

		found := false
		for _, route := range router.routes {
			if route == "resource:profile" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Resource() should register the resource routes")
		}
	})

	t.Run("Namespace", func(t *testing.T) {
		called := false
		result := router.Namespace("api", func(r gor.Router) {
			called = true
			if r != router {
				t.Error("Namespace() should pass the router to the function")
			}
		})

		if result != router {
			t.Error("Namespace() should return the router for chaining")
		}

		if !called {
			t.Error("Namespace() should call the provided function")
		}
	})

	t.Run("Group", func(t *testing.T) {
		middleware := func(next gor.HandlerFunc) gor.HandlerFunc {
			return func(ctx *gor.Context) error {
				return next(ctx)
			}
		}

		result := router.Group(middleware)
		if result == router {
			t.Error("Group() should return a new router instance")
		}

		groupRouter, ok := result.(*MockRouter)
		if !ok {
			t.Error("Group() should return a MockRouter instance")
		}

		if len(groupRouter.middleware) != 1 {
			t.Error("Group() should add middleware to the new router")
		}
	})

	t.Run("Use", func(t *testing.T) {
		middleware := func(next gor.HandlerFunc) gor.HandlerFunc {
			return func(ctx *gor.Context) error {
				return next(ctx)
			}
		}

		initialMiddlewareCount := len(router.middleware)
		result := router.Use(middleware)

		if result != router {
			t.Error("Use() should return the router for chaining")
		}

		if len(router.middleware) != initialMiddlewareCount+1 {
			t.Error("Use() should add middleware to the router")
		}
	})

	t.Run("Named", func(t *testing.T) {
		result := router.Named("test-route")
		if result != router {
			t.Error("Named() should return the router for chaining")
		}
	})
}

// TestController tests the Controller interface
func TestController(t *testing.T) {
	controller := &MockController{}
	ctx := &gor.Context{
		Context: context.Background(),
		Params:  make(map[string]string),
		Query:   make(map[string][]string),
		Flash:   make(map[string]interface{}),
	}

	tests := []struct {
		name     string
		action   func() error
		expected string
	}{
		{
			name:     "Index",
			action:   func() error { return controller.Index(ctx) },
			expected: "index",
		},
		{
			name:     "Show",
			action:   func() error { return controller.Show(ctx) },
			expected: "show",
		},
		{
			name:     "New",
			action:   func() error { return controller.New(ctx) },
			expected: "new",
		},
		{
			name:     "Create",
			action:   func() error { return controller.Create(ctx) },
			expected: "create",
		},
		{
			name:     "Edit",
			action:   func() error { return controller.Edit(ctx) },
			expected: "edit",
		},
		{
			name:     "Update",
			action:   func() error { return controller.Update(ctx) },
			expected: "update",
		},
		{
			name:     "Destroy",
			action:   func() error { return controller.Destroy(ctx) },
			expected: "destroy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action()
			if err != nil {
				t.Errorf("%s() returned error: %v", tt.name, err)
			}

			if controller.lastAction != tt.expected {
				t.Errorf("%s() should set lastAction to %s, got %s",
					tt.name, tt.expected, controller.lastAction)
			}

			if controller.lastCtx != ctx {
				t.Errorf("%s() should receive the context", tt.name)
			}
		})
	}
}

// TestContext tests the Context methods
func TestContext(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?foo=bar&foo=baz", nil)
	rec := httptest.NewRecorder()

	ctx := &gor.Context{
		Context:  context.Background(),
		Request:  req,
		Response: rec,
		Params:   map[string]string{"id": "123"},
		Query:    req.URL.Query(),
		Flash:    make(map[string]interface{}),
	}

	t.Run("Param", func(t *testing.T) {
		result := ctx.Param("id")
		if result != "123" {
			t.Errorf("Param() should return '123', got %s", result)
		}

		result = ctx.Param("nonexistent")
		if result != "" {
			t.Errorf("Param() should return empty string for nonexistent param, got %s", result)
		}
	})

	t.Run("QueryParam", func(t *testing.T) {
		result := ctx.QueryParam("foo")
		if result != "bar" {
			t.Errorf("QueryParam() should return first value 'bar', got %s", result)
		}

		result = ctx.QueryParam("nonexistent")
		if result != "" {
			t.Errorf("QueryParam() should return empty string for nonexistent param, got %s", result)
		}
	})

	t.Run("JSON", func(t *testing.T) {
		data := map[string]interface{}{
			"message": "hello",
			"status":  "ok",
		}

		err := ctx.JSON(200, data)
		if err != nil {
			t.Errorf("JSON() returned error: %v", err)
		}

		if rec.Code != 200 {
			t.Errorf("JSON() should set status code 200, got %d", rec.Code)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("JSON() should set Content-Type to application/json, got %s", contentType)
		}
	})

	t.Run("HTML", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx.Response = rec

		html := "<h1>Hello World</h1>"
		err := ctx.HTML(200, html)
		if err != nil {
			t.Errorf("HTML() returned error: %v", err)
		}

		if rec.Code != 200 {
			t.Errorf("HTML() should set status code 200, got %d", rec.Code)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "text/html; charset=utf-8" {
			t.Errorf("HTML() should set Content-Type to text/html; charset=utf-8, got %s", contentType)
		}

		if rec.Body.String() != html {
			t.Errorf("HTML() should write the HTML content, got %s", rec.Body.String())
		}
	})

	t.Run("Text", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx.Response = rec

		text := "Hello World"
		err := ctx.Text(200, text)
		if err != nil {
			t.Errorf("Text() returned error: %v", err)
		}

		if rec.Code != 200 {
			t.Errorf("Text() should set status code 200, got %d", rec.Code)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "text/plain; charset=utf-8" {
			t.Errorf("Text() should set Content-Type to text/plain; charset=utf-8, got %s", contentType)
		}

		if rec.Body.String() != text {
			t.Errorf("Text() should write the text content, got %s", rec.Body.String())
		}
	})

	t.Run("Redirect", func(t *testing.T) {
		rec := httptest.NewRecorder()
		ctx.Response = rec
		ctx.Request = httptest.NewRequest("GET", "/", nil)

		err := ctx.Redirect(302, "/redirected")
		if err != nil {
			t.Errorf("Redirect() returned error: %v", err)
		}

		if rec.Code != 302 {
			t.Errorf("Redirect() should set status code 302, got %d", rec.Code)
		}

		location := rec.Header().Get("Location")
		if location != "/redirected" {
			t.Errorf("Redirect() should set Location header to /redirected, got %s", location)
		}
	})

	t.Run("App", func(t *testing.T) {
		app := &MockApplication{}
		ctx.SetApp(app)

		result := ctx.App()
		if result != app {
			t.Error("App() should return the application set by SetApp()")
		}
	})
}

// TestConfig tests the Config interface
func TestConfig(t *testing.T) {
	config := NewMockConfig()

	t.Run("Environment", func(t *testing.T) {
		result := config.Environment()
		if result != "test" {
			t.Errorf("Environment() should return 'test', got %s", result)
		}
	})

	t.Run("Get", func(t *testing.T) {
		config.Set("test_key", "test_value")
		result := config.Get("test_key")
		if result != "test_value" {
			t.Errorf("Get() should return 'test_value', got %v", result)
		}

		result = config.Get("nonexistent")
		if result != nil {
			t.Errorf("Get() should return nil for nonexistent key, got %v", result)
		}
	})

	t.Run("Set", func(t *testing.T) {
		config.Set("new_key", 42)
		result := config.Get("new_key")
		if result != 42 {
			t.Errorf("Set/Get should work with integer values, got %v", result)
		}
	})

	t.Run("Database", func(t *testing.T) {
		dbConfig := config.Database()
		if dbConfig.Driver != "sqlite3" {
			t.Errorf("Database() should return sqlite3 driver, got %s", dbConfig.Driver)
		}
		if dbConfig.Database != ":memory:" {
			t.Errorf("Database() should return :memory: database, got %s", dbConfig.Database)
		}
	})

	t.Run("Server", func(t *testing.T) {
		serverConfig := config.Server()
		if serverConfig.Host != "localhost" {
			t.Errorf("Server() should return localhost host, got %s", serverConfig.Host)
		}
		if serverConfig.Port != 8080 {
			t.Errorf("Server() should return 8080 port, got %d", serverConfig.Port)
		}
		if serverConfig.ReadTimeout != 15*time.Second {
			t.Errorf("Server() should return 15s read timeout, got %v", serverConfig.ReadTimeout)
		}
		if serverConfig.WriteTimeout != 15*time.Second {
			t.Errorf("Server() should return 15s write timeout, got %v", serverConfig.WriteTimeout)
		}
		if serverConfig.IdleTimeout != 60*time.Second {
			t.Errorf("Server() should return 60s idle timeout, got %v", serverConfig.IdleTimeout)
		}
	})
}

// TestDatabaseConfig tests the DatabaseConfig struct
func TestDatabaseConfig(t *testing.T) {
	config := gor.DatabaseConfig{
		Driver:          "postgres",
		Host:            "localhost",
		Port:            5432,
		Database:        "testdb",
		Username:        "testuser",
		Password:        "testpass",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	t.Run("AllFieldsSet", func(t *testing.T) {
		if config.Driver != "postgres" {
			t.Errorf("Driver should be postgres, got %s", config.Driver)
		}
		if config.Host != "localhost" {
			t.Errorf("Host should be localhost, got %s", config.Host)
		}
		if config.Port != 5432 {
			t.Errorf("Port should be 5432, got %d", config.Port)
		}
		if config.Database != "testdb" {
			t.Errorf("Database should be testdb, got %s", config.Database)
		}
		if config.Username != "testuser" {
			t.Errorf("Username should be testuser, got %s", config.Username)
		}
		if config.Password != "testpass" {
			t.Errorf("Password should be testpass, got %s", config.Password)
		}
		if config.SSLMode != "disable" {
			t.Errorf("SSLMode should be disable, got %s", config.SSLMode)
		}
		if config.MaxOpenConns != 25 {
			t.Errorf("MaxOpenConns should be 25, got %d", config.MaxOpenConns)
		}
		if config.MaxIdleConns != 5 {
			t.Errorf("MaxIdleConns should be 5, got %d", config.MaxIdleConns)
		}
		if config.ConnMaxLifetime != 5*time.Minute {
			t.Errorf("ConnMaxLifetime should be 5m, got %v", config.ConnMaxLifetime)
		}
	})
}

// TestServerConfig tests the ServerConfig struct
func TestServerConfig(t *testing.T) {
	config := gor.ServerConfig{
		Host:         "0.0.0.0",
		Port:         3000,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	t.Run("AllFieldsSet", func(t *testing.T) {
		if config.Host != "0.0.0.0" {
			t.Errorf("Host should be 0.0.0.0, got %s", config.Host)
		}
		if config.Port != 3000 {
			t.Errorf("Port should be 3000, got %d", config.Port)
		}
		if config.ReadTimeout != 30*time.Second {
			t.Errorf("ReadTimeout should be 30s, got %v", config.ReadTimeout)
		}
		if config.WriteTimeout != 30*time.Second {
			t.Errorf("WriteTimeout should be 30s, got %v", config.WriteTimeout)
		}
		if config.IdleTimeout != 120*time.Second {
			t.Errorf("IdleTimeout should be 120s, got %v", config.IdleTimeout)
		}
	})
}
