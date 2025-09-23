package router

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/cuemby/gor/pkg/gor"
)

// GorRouter implements the gor.Router interface
type GorRouter struct {
	routes      []*Route
	middlewares []gor.MiddlewareFunc
	prefix      string
	namedRoutes map[string]*Route
	app         gor.Application
}

// Route represents a single route definition
type Route struct {
	Method      string
	Path        string
	Pattern     *regexp.Regexp
	Handler     gor.HandlerFunc
	Middlewares []gor.MiddlewareFunc
	Name        string
	Params      []string // Parameter names extracted from the path
}

// NewRouter creates a new router instance
func NewRouter(app gor.Application) gor.Router {
	return &GorRouter{
		routes:      make([]*Route, 0),
		middlewares: make([]gor.MiddlewareFunc, 0),
		namedRoutes: make(map[string]*Route),
		app:         app,
	}
}

// Resources creates RESTful routes for a resource
func (r *GorRouter) Resources(name string, controller gor.Controller) gor.Router {
	// Generate standard RESTful routes
	// GET    /users          -> Index
	// GET    /users/new      -> New
	// POST   /users          -> Create
	// GET    /users/:id      -> Show
	// GET    /users/:id/edit -> Edit
	// PUT    /users/:id      -> Update
	// PATCH  /users/:id      -> Update
	// DELETE /users/:id      -> Destroy

	basePath := r.prefix + "/" + name

	// Index route
	r.GET(basePath, wrapControllerAction(controller.Index)).Named(name + "_index")

	// New route
	r.GET(basePath+"/new", wrapControllerAction(controller.New)).Named(name + "_new")

	// Create route
	r.POST(basePath, wrapControllerAction(controller.Create)).Named(name + "_create")

	// Show route
	r.GET(basePath+"/:id", wrapControllerAction(controller.Show)).Named(name + "_show")

	// Edit route
	r.GET(basePath+"/:id/edit", wrapControllerAction(controller.Edit)).Named(name + "_edit")

	// Update routes (both PUT and PATCH)
	r.PUT(basePath+"/:id", wrapControllerAction(controller.Update)).Named(name + "_update")
	r.PATCH(basePath+"/:id", wrapControllerAction(controller.Update))

	// Destroy route
	r.DELETE(basePath+"/:id", wrapControllerAction(controller.Destroy)).Named(name + "_destroy")

	return r
}

// Resource creates routes for a singular resource (no index action)
func (r *GorRouter) Resource(name string, controller gor.Controller) gor.Router {
	// Generate singular resource routes
	// GET    /profile/new    -> New
	// POST   /profile        -> Create
	// GET    /profile        -> Show
	// GET    /profile/edit   -> Edit
	// PUT    /profile        -> Update
	// PATCH  /profile        -> Update
	// DELETE /profile        -> Destroy

	basePath := r.prefix + "/" + name

	// New route
	r.GET(basePath+"/new", wrapControllerAction(controller.New)).Named(name + "_new")

	// Create route
	r.POST(basePath, wrapControllerAction(controller.Create)).Named(name + "_create")

	// Show route
	r.GET(basePath, wrapControllerAction(controller.Show)).Named(name + "_show")

	// Edit route
	r.GET(basePath+"/edit", wrapControllerAction(controller.Edit)).Named(name + "_edit")

	// Update routes (both PUT and PATCH)
	r.PUT(basePath, wrapControllerAction(controller.Update)).Named(name + "_update")
	r.PATCH(basePath, wrapControllerAction(controller.Update))

	// Destroy route
	r.DELETE(basePath, wrapControllerAction(controller.Destroy)).Named(name + "_destroy")

	return r
}

// GET registers a GET route
func (r *GorRouter) GET(path string, handler gor.HandlerFunc) gor.Router {
	r.addRoute("GET", path, handler)
	return r
}

// POST registers a POST route
func (r *GorRouter) POST(path string, handler gor.HandlerFunc) gor.Router {
	r.addRoute("POST", path, handler)
	return r
}

// PUT registers a PUT route
func (r *GorRouter) PUT(path string, handler gor.HandlerFunc) gor.Router {
	r.addRoute("PUT", path, handler)
	return r
}

// PATCH registers a PATCH route
func (r *GorRouter) PATCH(path string, handler gor.HandlerFunc) gor.Router {
	r.addRoute("PATCH", path, handler)
	return r
}

// DELETE registers a DELETE route
func (r *GorRouter) DELETE(path string, handler gor.HandlerFunc) gor.Router {
	r.addRoute("DELETE", path, handler)
	return r
}

// Namespace creates a route namespace with a prefix
func (r *GorRouter) Namespace(prefix string, fn func(gor.Router)) gor.Router {
	// Create a sub-router with the namespace prefix
	subRouter := &GorRouter{
		routes:      make([]*Route, 0),
		middlewares: r.middlewares, // Inherit parent middlewares
		prefix:      r.prefix + prefix,
		namedRoutes: r.namedRoutes, // Share named routes map
		app:         r.app,
	}

	// Execute the namespace function with the sub-router
	fn(subRouter)

	// Merge sub-router routes back to parent
	r.routes = append(r.routes, subRouter.routes...)

	return r
}

// Group creates a route group with shared middleware
func (r *GorRouter) Group(middleware ...gor.MiddlewareFunc) gor.Router {
	// Create a sub-router that inherits parent routes but adds new middleware
	return &GorRouter{
		routes:      r.routes,      // Share routes
		middlewares: append(r.middlewares, middleware...), // Add new middleware
		prefix:      r.prefix,
		namedRoutes: r.namedRoutes,
		app:         r.app,
	}
}

// Use adds middleware to all routes
func (r *GorRouter) Use(middleware ...gor.MiddlewareFunc) gor.Router {
	r.middlewares = append(r.middlewares, middleware...)
	return r
}

// Named assigns a name to the last added route
func (r *GorRouter) Named(name string) gor.Router {
	if len(r.routes) > 0 {
		lastRoute := r.routes[len(r.routes)-1]
		lastRoute.Name = name
		r.namedRoutes[name] = lastRoute
	}
	return r
}

// ServeHTTP implements the http.Handler interface
func (r *GorRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Find matching route
	route, params := r.findRoute(req.Method, req.URL.Path)
	if route == nil {
		http.NotFound(w, req)
		return
	}

	// Create context
	ctx := &gor.Context{
		Context:  context.Background(),
		Request:  req,
		Response: w,
		Params:   params,
		Query:    req.URL.Query(),
		Flash:    make(map[string]interface{}),
		// app field is private, set through reflection or a setter
	}

	// Set application reference if possible
	if r.app != nil {
		ctx.SetApp(r.app)
	}

	// Build middleware chain
	handler := route.Handler

	// Apply route-specific middleware
	for i := len(route.Middlewares) - 1; i >= 0; i-- {
		handler = route.Middlewares[i](handler)
	}

	// Apply global middleware
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}

	// Execute handler
	if err := handler(ctx); err != nil {
		// Handle error (could be customized)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Helper methods

func (r *GorRouter) addRoute(method, path string, handler gor.HandlerFunc) {
	// Convert path with parameters to regex pattern
	pattern, params := pathToRegexp(r.prefix + path)

	route := &Route{
		Method:      method,
		Path:        r.prefix + path,
		Pattern:     pattern,
		Handler:     handler,
		Middlewares: make([]gor.MiddlewareFunc, 0),
		Params:      params,
	}

	r.routes = append(r.routes, route)
}

func (r *GorRouter) findRoute(method, path string) (*Route, map[string]string) {
	for _, route := range r.routes {
		if route.Method != method {
			continue
		}

		if matches := route.Pattern.FindStringSubmatch(path); matches != nil {
			// Extract parameters
			params := make(map[string]string)
			for i, name := range route.Params {
				if i+1 < len(matches) {
					params[name] = matches[i+1]
				}
			}
			return route, params
		}
	}
	return nil, nil
}

// pathToRegexp converts a path with parameters to a regular expression
func pathToRegexp(path string) (*regexp.Regexp, []string) {
	var params []string
	pattern := path

	// Find all parameters in the path (e.g., :id, :slug)
	re := regexp.MustCompile(`:(\w+)`)
	matches := re.FindAllStringSubmatch(path, -1)

	for _, match := range matches {
		params = append(params, match[1])
		// Replace :param with regex capture group
		pattern = strings.Replace(pattern, match[0], `([^/]+)`, 1)
	}

	// Escape special regex characters except for our capture groups
	pattern = "^" + pattern + "$"

	return regexp.MustCompile(pattern), params
}

// wrapControllerAction wraps a controller action to match the HandlerFunc signature
func wrapControllerAction(action func(*gor.Context) error) gor.HandlerFunc {
	return action
}

// URLFor generates a URL for a named route
func (r *GorRouter) URLFor(name string, params map[string]string) (string, error) {
	route, exists := r.namedRoutes[name]
	if !exists {
		return "", fmt.Errorf("no route named %s", name)
	}

	url := route.Path
	for param, value := range params {
		url = strings.Replace(url, ":"+param, value, 1)
	}

	return url, nil
}

// Routes returns all registered routes (useful for debugging)
func (r *GorRouter) Routes() []*Route {
	return r.routes
}

// PrintRoutes prints all registered routes (useful for debugging)
func (r *GorRouter) PrintRoutes() {
	fmt.Println("Registered Routes:")
	fmt.Println("==================")
	for _, route := range r.routes {
		fmt.Printf("%-7s %-30s", route.Method, route.Path)
		if route.Name != "" {
			fmt.Printf(" [%s]", route.Name)
		}
		fmt.Println()
	}
}