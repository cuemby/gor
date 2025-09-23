package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/cuemby/gor/pkg/gor"
)

// Mock application for testing
type mockApp struct{}

func (m *mockApp) Start(ctx context.Context) error   { return nil }
func (m *mockApp) Stop(ctx context.Context) error    { return nil }
func (m *mockApp) Router() gor.Router                 { return nil }
func (m *mockApp) ORM() gor.ORM                       { return nil }
func (m *mockApp) Queue() gor.Queue                   { return nil }
func (m *mockApp) Cache() gor.Cache                   { return nil }
func (m *mockApp) Cable() gor.Cable                   { return nil }
func (m *mockApp) Auth() interface{}                  { return nil }
func (m *mockApp) Config() gor.Config                 { return nil }

// Mock controller for testing
type mockController struct {
	indexCalled   bool
	newCalled     bool
	createCalled  bool
	showCalled    bool
	editCalled    bool
	updateCalled  bool
	destroyCalled bool
	lastParams    map[string]string
}

func (m *mockController) Index(ctx *gor.Context) error {
	m.indexCalled = true
	m.lastParams = ctx.Params
	return nil
}

func (m *mockController) New(ctx *gor.Context) error {
	m.newCalled = true
	m.lastParams = ctx.Params
	return nil
}

func (m *mockController) Create(ctx *gor.Context) error {
	m.createCalled = true
	m.lastParams = ctx.Params
	return nil
}

func (m *mockController) Show(ctx *gor.Context) error {
	m.showCalled = true
	m.lastParams = ctx.Params
	return nil
}

func (m *mockController) Edit(ctx *gor.Context) error {
	m.editCalled = true
	m.lastParams = ctx.Params
	return nil
}

func (m *mockController) Update(ctx *gor.Context) error {
	m.updateCalled = true
	m.lastParams = ctx.Params
	return nil
}

func (m *mockController) Destroy(ctx *gor.Context) error {
	m.destroyCalled = true
	m.lastParams = ctx.Params
	return nil
}

// Test helper to create a test handler
func testHandler(message string) gor.HandlerFunc {
	return func(ctx *gor.Context) error {
		ctx.Response.WriteHeader(http.StatusOK)
		ctx.Response.Write([]byte(message))
		return nil
	}
}

// Test helper to create an error handler
func errorHandler() gor.HandlerFunc {
	return func(ctx *gor.Context) error {
		return fmt.Errorf("test error")
	}
}

// Test helper to create middleware
func testMiddleware(prefix string) gor.MiddlewareFunc {
	return func(next gor.HandlerFunc) gor.HandlerFunc {
		return func(ctx *gor.Context) error {
			ctx.Response.Write([]byte(prefix + ":"))
			return next(ctx)
		}
	}
}

func TestNewRouter(t *testing.T) {
	app := &mockApp{}
	router := NewRouter(app)

	gorRouter, ok := router.(*GorRouter)
	if !ok {
		t.Fatal("NewRouter() should return *GorRouter")
	}

	if gorRouter.app != app {
		t.Error("NewRouter() should set application reference")
	}

	if gorRouter.routes == nil {
		t.Error("NewRouter() should initialize routes slice")
	}

	if gorRouter.middlewares == nil {
		t.Error("NewRouter() should initialize middlewares slice")
	}

	if gorRouter.namedRoutes == nil {
		t.Error("NewRouter() should initialize namedRoutes map")
	}

	if gorRouter.prefix != "" {
		t.Error("NewRouter() should initialize with empty prefix")
	}
}

func TestRouter_BasicRouting(t *testing.T) {
	app := &mockApp{}
	router := NewRouter(app)

	t.Run("GET", func(t *testing.T) {
		router.GET("/test", testHandler("GET response"))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GET route should return 200, got %d", w.Code)
		}

		if w.Body.String() != "GET response" {
			t.Errorf("GET route response = %v, want 'GET response'", w.Body.String())
		}
	})

	t.Run("POST", func(t *testing.T) {
		router.POST("/post", testHandler("POST response"))

		req := httptest.NewRequest("POST", "/post", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("POST route should return 200, got %d", w.Code)
		}

		if w.Body.String() != "POST response" {
			t.Errorf("POST route response = %v, want 'POST response'", w.Body.String())
		}
	})

	t.Run("PUT", func(t *testing.T) {
		router.PUT("/put", testHandler("PUT response"))

		req := httptest.NewRequest("PUT", "/put", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("PUT route should return 200, got %d", w.Code)
		}
	})

	t.Run("PATCH", func(t *testing.T) {
		router.PATCH("/patch", testHandler("PATCH response"))

		req := httptest.NewRequest("PATCH", "/patch", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("PATCH route should return 200, got %d", w.Code)
		}
	})

	t.Run("DELETE", func(t *testing.T) {
		router.DELETE("/delete", testHandler("DELETE response"))

		req := httptest.NewRequest("DELETE", "/delete", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("DELETE route should return 200, got %d", w.Code)
		}
	})
}

func TestRouter_NotFound(t *testing.T) {
	app := &mockApp{}
	router := NewRouter(app)

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Nonexistent route should return 404, got %d", w.Code)
	}
}

func TestRouter_Parameters(t *testing.T) {
	app := &mockApp{}
	router := NewRouter(app)

	handler := func(ctx *gor.Context) error {
		id := ctx.Params["id"]
		slug := ctx.Params["slug"]
		ctx.Response.WriteHeader(http.StatusOK)
		ctx.Response.Write([]byte("id:" + id + ",slug:" + slug))
		return nil
	}

	router.GET("/posts/:id/:slug", handler)

	req := httptest.NewRequest("GET", "/posts/123/hello-world", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Parameterized route should return 200, got %d", w.Code)
	}

	expected := "id:123,slug:hello-world"
	if w.Body.String() != expected {
		t.Errorf("Parameterized route response = %v, want %v", w.Body.String(), expected)
	}
}

func TestRouter_Named(t *testing.T) {
	app := &mockApp{}
	router := NewRouter(app).(*GorRouter)

	router.GET("/users/:id", testHandler("user")).Named("user_show")

	t.Run("NamedRouteRegistered", func(t *testing.T) {
		route, exists := router.namedRoutes["user_show"]
		if !exists {
			t.Error("Named route should be registered")
		}

		if route.Name != "user_show" {
			t.Errorf("Route name = %v, want user_show", route.Name)
		}
	})

	t.Run("URLFor", func(t *testing.T) {
		url, err := router.URLFor("user_show", map[string]string{"id": "42"})
		if err != nil {
			t.Fatalf("URLFor() should not return error: %v", err)
		}

		expected := "/users/42"
		if url != expected {
			t.Errorf("URLFor() = %v, want %v", url, expected)
		}
	})

	t.Run("URLForNonexistent", func(t *testing.T) {
		_, err := router.URLFor("nonexistent", nil)
		if err == nil {
			t.Error("URLFor() should return error for nonexistent route")
		}
	})
}

func TestRouter_Middleware(t *testing.T) {
	app := &mockApp{}
	router := NewRouter(app)

	t.Run("GlobalMiddleware", func(t *testing.T) {
		router.Use(testMiddleware("global"))
		router.GET("/middleware", testHandler("content"))

		req := httptest.NewRequest("GET", "/middleware", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Middleware route should return 200, got %d", w.Code)
		}

		expected := "global:content"
		if w.Body.String() != expected {
			t.Errorf("Middleware response = %v, want %v", w.Body.String(), expected)
		}
	})

	t.Run("Group", func(t *testing.T) {
		group := router.Group(testMiddleware("group"))
		group.GET("/group", testHandler("content"))

		req := httptest.NewRequest("GET", "/group", nil)
		w := httptest.NewRecorder()

		// Use the group router, not the main router
		group.ServeHTTP(w, req)

		expected := "global:group:content"
		if w.Body.String() != expected {
			t.Errorf("Group middleware response = %v, want %v", w.Body.String(), expected)
		}
	})

	t.Run("MultipleMiddlewares", func(t *testing.T) {
		// Create a fresh router for this test to avoid middleware pollution
		freshRouter := NewRouter(app)
		freshRouter.Use(testMiddleware("first"), testMiddleware("second"))
		freshRouter.GET("/multi", testHandler("end"))

		req := httptest.NewRequest("GET", "/multi", nil)
		w := httptest.NewRecorder()

		freshRouter.ServeHTTP(w, req)

		// Middleware should be applied in order
		expected := "first:second:end"
		if w.Body.String() != expected {
			t.Errorf("Multiple middleware response = %v, want %v", w.Body.String(), expected)
		}
	})
}

func TestRouter_Namespace(t *testing.T) {
	app := &mockApp{}
	router := NewRouter(app)

	router.Namespace("/api/v1", func(r gor.Router) {
		r.GET("/users", testHandler("users"))
		r.POST("/posts", testHandler("posts"))
	})

	t.Run("NamespaceGET", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/users", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Namespace GET should return 200, got %d", w.Code)
		}

		if w.Body.String() != "users" {
			t.Errorf("Namespace GET response = %v, want users", w.Body.String())
		}
	})

	t.Run("NamespacePOST", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/posts", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Namespace POST should return 200, got %d", w.Code)
		}

		if w.Body.String() != "posts" {
			t.Errorf("Namespace POST response = %v, want posts", w.Body.String())
		}
	})

	t.Run("NestedNamespace", func(t *testing.T) {
		router.Namespace("/admin", func(r gor.Router) {
			r.Namespace("/v2", func(r2 gor.Router) {
				r2.GET("/dashboard", testHandler("dashboard"))
			})
		})

		req := httptest.NewRequest("GET", "/admin/v2/dashboard", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Nested namespace should return 200, got %d", w.Code)
		}

		if w.Body.String() != "dashboard" {
			t.Errorf("Nested namespace response = %v, want dashboard", w.Body.String())
		}
	})
}

func TestRouter_Resources(t *testing.T) {
	app := &mockApp{}
	router := NewRouter(app)
	controller := &mockController{}

	router.Resources("users", controller)

	tests := []struct {
		method   string
		path     string
		expected func(*mockController) bool
	}{
		{"GET", "/users", func(c *mockController) bool { return c.indexCalled }},
		{"GET", "/users/new", func(c *mockController) bool { return c.newCalled }},
		{"POST", "/users", func(c *mockController) bool { return c.createCalled }},
		{"GET", "/users/123", func(c *mockController) bool { return c.showCalled }},
		{"GET", "/users/123/edit", func(c *mockController) bool { return c.editCalled }},
		{"PUT", "/users/123", func(c *mockController) bool { return c.updateCalled }},
		{"PATCH", "/users/123", func(c *mockController) bool { return c.updateCalled }},
		{"DELETE", "/users/123", func(c *mockController) bool { return c.destroyCalled }},
	}

	for _, test := range tests {
		t.Run(test.method+"_"+test.path, func(t *testing.T) {
			// Reset controller state
			*controller = mockController{}

			req := httptest.NewRequest(test.method, test.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("%s %s should return 200, got %d", test.method, test.path, w.Code)
			}

			if !test.expected(controller) {
				t.Errorf("%s %s should call expected controller method", test.method, test.path)
			}

			// Check parameters for routes with :id
			if strings.Contains(test.path, "123") {
				if controller.lastParams["id"] != "123" {
					t.Errorf("Parameter id should be '123', got '%v'", controller.lastParams["id"])
				}
			}
		})
	}
}

func TestRouter_Resource(t *testing.T) {
	app := &mockApp{}
	router := NewRouter(app)
	controller := &mockController{}

	router.Resource("profile", controller)

	tests := []struct {
		method   string
		path     string
		expected func(*mockController) bool
	}{
		{"GET", "/profile/new", func(c *mockController) bool { return c.newCalled }},
		{"POST", "/profile", func(c *mockController) bool { return c.createCalled }},
		{"GET", "/profile", func(c *mockController) bool { return c.showCalled }},
		{"GET", "/profile/edit", func(c *mockController) bool { return c.editCalled }},
		{"PUT", "/profile", func(c *mockController) bool { return c.updateCalled }},
		{"PATCH", "/profile", func(c *mockController) bool { return c.updateCalled }},
		{"DELETE", "/profile", func(c *mockController) bool { return c.destroyCalled }},
	}

	for _, test := range tests {
		t.Run(test.method+"_"+test.path, func(t *testing.T) {
			// Reset controller state
			*controller = mockController{}

			req := httptest.NewRequest(test.method, test.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("%s %s should return 200, got %d", test.method, test.path, w.Code)
			}

			if !test.expected(controller) {
				t.Errorf("%s %s should call expected controller method", test.method, test.path)
			}
		})
	}

	// Test that index route is not created for singular resource
	t.Run("NoIndexRoute", func(t *testing.T) {
		// Reset controller state
		*controller = mockController{}

		req := httptest.NewRequest("GET", "/profiles", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Singular resource should not have index route, got %d", w.Code)
		}

		if controller.indexCalled {
			t.Error("Index should not be called for singular resource")
		}
	})
}

func TestRouter_ErrorHandling(t *testing.T) {
	app := &mockApp{}
	router := NewRouter(app)

	router.GET("/error", errorHandler())

	req := httptest.NewRequest("GET", "/error", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Error handler should return 500, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "test error") {
		t.Error("Error response should contain error message")
	}
}

func TestPathToRegexp(t *testing.T) {
	tests := []struct {
		path           string
		expectedParams []string
		testPath       string
		shouldMatch    bool
		expectedValues map[string]string
	}{
		{
			path:           "/users/:id",
			expectedParams: []string{"id"},
			testPath:       "/users/123",
			shouldMatch:    true,
			expectedValues: map[string]string{"id": "123"},
		},
		{
			path:           "/posts/:id/:slug",
			expectedParams: []string{"id", "slug"},
			testPath:       "/posts/42/hello-world",
			shouldMatch:    true,
			expectedValues: map[string]string{"id": "42", "slug": "hello-world"},
		},
		{
			path:           "/static",
			expectedParams: []string{},
			testPath:       "/static",
			shouldMatch:    true,
			expectedValues: map[string]string{},
		},
		{
			path:           "/users/:id",
			expectedParams: []string{"id"},
			testPath:       "/users/123/edit",
			shouldMatch:    false,
			expectedValues: nil,
		},
		{
			path:           "/api/v:version/users/:id",
			expectedParams: []string{"version", "id"},
			testPath:       "/api/v1/users/456",
			shouldMatch:    true,
			expectedValues: map[string]string{"version": "1", "id": "456"},
		},
	}

	for _, test := range tests {
		t.Run(test.path+"->"+test.testPath, func(t *testing.T) {
			pattern, params := pathToRegexp(test.path)

			// Check parameters
			if len(params) != len(test.expectedParams) {
				t.Errorf("Expected %d params, got %d", len(test.expectedParams), len(params))
			}

			for i, expectedParam := range test.expectedParams {
				if i >= len(params) || params[i] != expectedParam {
					t.Errorf("Expected param %d to be %s, got %s", i, expectedParam, params[i])
				}
			}

			// Check pattern matching
			matches := pattern.FindStringSubmatch(test.testPath)
			if test.shouldMatch && matches == nil {
				t.Errorf("Pattern should match %s", test.testPath)
			}

			if !test.shouldMatch && matches != nil {
				t.Errorf("Pattern should not match %s", test.testPath)
			}

			if test.shouldMatch && matches != nil {
				// Check extracted values
				for i, param := range params {
					if i+1 >= len(matches) {
						t.Errorf("Missing match for parameter %s", param)
						continue
					}
					expectedValue := test.expectedValues[param]
					actualValue := matches[i+1]
					if actualValue != expectedValue {
						t.Errorf("Parameter %s = %s, want %s", param, actualValue, expectedValue)
					}
				}
			}
		})
	}
}

func TestRouter_Routes(t *testing.T) {
	app := &mockApp{}
	router := NewRouter(app).(*GorRouter)

	router.GET("/test", testHandler("test"))
	router.POST("/api/users", testHandler("users"))

	routes := router.Routes()
	if len(routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routes))
	}

	// Check first route
	if routes[0].Method != "GET" {
		t.Errorf("First route method = %s, want GET", routes[0].Method)
	}
	if routes[0].Path != "/test" {
		t.Errorf("First route path = %s, want /test", routes[0].Path)
	}

	// Check second route
	if routes[1].Method != "POST" {
		t.Errorf("Second route method = %s, want POST", routes[1].Method)
	}
	if routes[1].Path != "/api/users" {
		t.Errorf("Second route path = %s, want /api/users", routes[1].Path)
	}
}

func TestRouter_PrintRoutes(t *testing.T) {
	app := &mockApp{}
	router := NewRouter(app).(*GorRouter)

	router.GET("/test", testHandler("test")).Named("test_route")
	router.POST("/api/users", testHandler("users"))

	// This test just ensures PrintRoutes doesn't panic
	// In a real scenario, you might capture stdout to test the output
	router.PrintRoutes()
}

func TestRoute_Structure(t *testing.T) {
	route := &Route{
		Method:      "GET",
		Path:        "/test/:id",
		Pattern:     regexp.MustCompile("^/test/([^/]+)$"),
		Handler:     testHandler("test"),
		Middlewares: []gor.MiddlewareFunc{},
		Name:        "test_route",
		Params:      []string{"id"},
	}

	if route.Method != "GET" {
		t.Errorf("Route.Method = %s, want GET", route.Method)
	}

	if route.Path != "/test/:id" {
		t.Errorf("Route.Path = %s, want /test/:id", route.Path)
	}

	if len(route.Params) != 1 || route.Params[0] != "id" {
		t.Errorf("Route.Params = %v, want [id]", route.Params)
	}

	if route.Name != "test_route" {
		t.Errorf("Route.Name = %s, want test_route", route.Name)
	}
}

func TestWrapControllerAction(t *testing.T) {
	controller := &mockController{}
	wrapped := wrapControllerAction(controller.Index)

	ctx := &gor.Context{
		Params: map[string]string{"test": "value"},
	}

	err := wrapped(ctx)
	if err != nil {
		t.Errorf("Wrapped controller action should not return error: %v", err)
	}

	if !controller.indexCalled {
		t.Error("Wrapped controller action should call original method")
	}

	if controller.lastParams["test"] != "value" {
		t.Error("Wrapped controller action should pass context correctly")
	}
}

func TestRouter_MethodChaining(t *testing.T) {
	app := &mockApp{}
	router := NewRouter(app)

	// Test that all methods return the router for chaining
	result := router.GET("/test", testHandler("test")).
		Named("test").
		Use(testMiddleware("middleware"))

	if result != router {
		t.Error("Router methods should return the router for chaining")
	}
}

func TestRouter_Context(t *testing.T) {
	app := &mockApp{}
	router := NewRouter(app)

	handler := func(ctx *gor.Context) error {
		// Test context fields
		if ctx.Request == nil {
			t.Error("Context.Request should be set")
		}
		if ctx.Response == nil {
			t.Error("Context.Response should be set")
		}
		if ctx.Params == nil {
			t.Error("Context.Params should be initialized")
		}
		if ctx.Query == nil {
			t.Error("Context.Query should be initialized")
		}
		if ctx.Flash == nil {
			t.Error("Context.Flash should be initialized")
		}

		ctx.Response.WriteHeader(http.StatusOK)
		ctx.Response.Write([]byte("context_test"))
		return nil
	}

	router.GET("/context/:id", handler)

	req := httptest.NewRequest("GET", "/context/123?foo=bar", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Context test should return 200, got %d", w.Code)
	}

	if w.Body.String() != "context_test" {
		t.Errorf("Context test response = %v, want context_test", w.Body.String())
	}
}