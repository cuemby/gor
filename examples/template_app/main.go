package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/cuemby/gor/internal/router"
	"github.com/cuemby/gor/internal/views"
	"github.com/cuemby/gor/pkg/gor"
	"github.com/cuemby/gor/pkg/middleware"
)

// PageData represents data passed to templates
type PageData struct {
	Title       string
	Description string
	Users       []User
	User        *User
	Flash       map[string]string
}

// User represents a user (simplified for demo)
type User struct {
	ID        int
	Name      string
	Email     string
	Bio       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Global view renderer
var viewRenderer *views.ViewRenderer

// HomeController handles homepage
type HomeController struct{}

func (c *HomeController) Index(ctx *gor.Context) error {
	data := PageData{
		Title:       "Home",
		Description: "Welcome to Gor Framework",
	}

	return renderTemplate(ctx, "home/index", data)
}

// AboutController handles about page
type AboutController struct{}

func (c *AboutController) Index(ctx *gor.Context) error {
	html := `
<h1>About Gor Framework</h1>

<p>Gor is a Rails-inspired web framework for Go that brings the productivity and conventions
of Rails to the performance and type safety of Go.</p>

<h2>Key Features</h2>
<ul>
    <li><strong>Convention over Configuration</strong> - Sensible defaults that just work</li>
    <li><strong>Batteries Included</strong> - Everything you need in one framework</li>
    <li><strong>Type Safety</strong> - Leverage Go's compile-time checking</li>
    <li><strong>High Performance</strong> - 10x+ faster than Rails</li>
    <li><strong>Rails-style Routing</strong> - RESTful resources out of the box</li>
    <li><strong>Template Engine</strong> - Layouts, partials, and helpers</li>
    <li><strong>ORM with Migrations</strong> - ActiveRecord-inspired database layer</li>
    <li><strong>Background Jobs</strong> - Database-backed job processing</li>
    <li><strong>WebSockets</strong> - Real-time features built-in</li>
</ul>

<h2>Philosophy</h2>
<p>We believe that developer happiness and productivity don't have to be sacrificed for performance.
Gor brings the best of both worlds - the elegant conventions of Rails with the raw power of Go.</p>

<h2>Getting Started</h2>
<pre><code>go get github.com/cuemby/gor

gor new myapp
cd myapp
gor server</code></pre>

<p>That's it! You're ready to build amazing web applications with Gor.</p>
`

	// For now, render HTML directly since we don't have a template
	return ctx.HTML(http.StatusOK, wrapInLayout("About", html))
}

// UsersController handles user pages
type UsersController struct {
	users []User
}

func NewUsersController() *UsersController {
	// Create some sample users
	return &UsersController{
		users: []User{
			{ID: 1, Name: "Alice Johnson", Email: "alice@example.com", Bio: "Software Engineer", CreatedAt: time.Now().AddDate(0, -1, 0), UpdatedAt: time.Now()},
			{ID: 2, Name: "Bob Smith", Email: "bob@example.com", Bio: "Product Manager", CreatedAt: time.Now().AddDate(0, 0, -15), UpdatedAt: time.Now()},
			{ID: 3, Name: "Charlie Brown", Email: "charlie@example.com", Bio: "Designer", CreatedAt: time.Now().AddDate(0, 0, -7), UpdatedAt: time.Now()},
		},
	}
}

func (c *UsersController) Index(ctx *gor.Context) error {
	data := PageData{
		Title: "Users",
		Users: c.users,
	}

	// Try to use template engine if views exist
	if viewRenderer != nil {
		return viewRenderer.Render(ctx, "users/index", data)
	}

	// Fallback to HTML
	html := `<h1>Users</h1><table><thead><tr><th>ID</th><th>Name</th><th>Email</th><th>Bio</th></tr></thead><tbody>`
	for _, user := range c.users {
		html += fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td></tr>", user.ID, user.Name, user.Email, user.Bio)
	}
	html += `</tbody></table>`

	return ctx.HTML(http.StatusOK, wrapInLayout("Users", html))
}

func (c *UsersController) Show(ctx *gor.Context) error {
	// In a real app, we'd get the ID from ctx.Param("id") and look up the user
	user := c.users[0] // Demo: just show first user

	data := PageData{
		Title: fmt.Sprintf("User: %s", user.Name),
		User:  &user,
	}

	if viewRenderer != nil {
		return viewRenderer.Render(ctx, "users/show", data)
	}

	html := fmt.Sprintf(`
<h1>%s</h1>
<p><strong>Email:</strong> %s</p>
<p><strong>Bio:</strong> %s</p>
<p><strong>Member Since:</strong> %s</p>
`, user.Name, user.Email, user.Bio, user.CreatedAt.Format("January 2, 2006"))

	return ctx.HTML(http.StatusOK, wrapInLayout(data.Title, html))
}

// Helper functions
func renderTemplate(ctx *gor.Context, template string, data interface{}) error {
	if viewRenderer != nil {
		return viewRenderer.Render(ctx, template, data)
	}
	// Fallback to JSON if no template engine
	return ctx.JSON(http.StatusOK, data)
}

func wrapInLayout(title, content string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>%s - Gor Framework</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
               margin: 0; padding: 20px; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); min-height: 100vh; }
        .container { max-width: 1000px; margin: 0 auto; background: white; padding: 40px;
                    border-radius: 10px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
        h1 { color: #333; }
        p { color: #666; line-height: 1.6; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
        code { font-family: 'Courier New', monospace; }
        table { width: 100%%; border-collapse: collapse; margin-top: 20px; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #e0e0e0; }
        th { background: #f5f5f5; }
        nav { margin-bottom: 30px; padding-bottom: 20px; border-bottom: 1px solid #e0e0e0; }
        nav a { margin-right: 20px; color: #667eea; text-decoration: none; }
        nav a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <nav>
            <a href="/">Home</a>
            <a href="/about">About</a>
            <a href="/users">Users</a>
            <a href="/docs">Documentation</a>
        </nav>
        %s
    </div>
</body>
</html>`, title, content)
}

// Simplified Application for this example
type SimpleApp struct {
	router gor.Router
}

func (a *SimpleApp) Start(ctx context.Context) error { return nil }
func (a *SimpleApp) Stop(ctx context.Context) error  { return nil }
func (a *SimpleApp) Router() gor.Router              { return a.router }
func (a *SimpleApp) ORM() gor.ORM                    { return nil }
func (a *SimpleApp) Queue() gor.Queue                { return nil }
func (a *SimpleApp) Cache() gor.Cache                { return nil }
func (a *SimpleApp) Cable() gor.Cable                { return nil }
func (a *SimpleApp) Auth() interface{}               { return nil }
func (a *SimpleApp) Config() gor.Config              { return nil }

func main() {
	// Try to initialize template engine if views directory exists
	viewsPath := filepath.Join(".", "views")
	if _, err := filepath.Abs(viewsPath); err == nil {
		viewRenderer = views.NewViewRenderer(viewsPath, true) // debug mode = true
		fmt.Println("✅ Template engine initialized with views from:", viewsPath)
	} else {
		fmt.Println("⚠️  No views directory found, using inline HTML")
	}

	// Create application
	app := &SimpleApp{}

	// Initialize router
	appRouter := router.NewRouter(app)

	// Add middleware
	appRouter.Use(
		middleware.Logger(),
		middleware.Recovery(),
	)

	// Define routes
	homeController := &HomeController{}
	aboutController := &AboutController{}
	usersController := NewUsersController()

	appRouter.GET("/", func(ctx *gor.Context) error {
		return homeController.Index(ctx)
	})

	appRouter.GET("/about", func(ctx *gor.Context) error {
		return aboutController.Index(ctx)
	})

	appRouter.GET("/users", func(ctx *gor.Context) error {
		return usersController.Index(ctx)
	})

	appRouter.GET("/users/:id", func(ctx *gor.Context) error {
		return usersController.Show(ctx)
	})

	appRouter.GET("/docs", func(ctx *gor.Context) error {
		return ctx.HTML(http.StatusOK, wrapInLayout("Documentation", `
<h1>Gor Framework Documentation</h1>
<p>Complete documentation coming soon!</p>
<p>For now, check out the <a href="https://github.com/cuemby/gor">GitHub repository</a> for examples and guides.</p>
`))
	})

	// Update app router
	app.router = appRouter

	// Start server
	fmt.Println("\n🚀 Template demo server starting on http://localhost:8081")
	fmt.Println("   Visit http://localhost:8081 to see the template engine in action")
	fmt.Println("   Press Ctrl+C to stop")

	server := &http.Server{
		Addr:              ":8081",
		Handler:           appRouter,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}
