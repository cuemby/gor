package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/cuemby/gor/internal/app/controllers"
	"github.com/cuemby/gor/internal/orm"
	"github.com/cuemby/gor/internal/router"
	"github.com/cuemby/gor/pkg/gor"
	"github.com/cuemby/gor/pkg/middleware"
)

// WebUser model (renamed to avoid conflict with orm_example.go)
type WebUser struct {
	gor.BaseModel
	Name  string `json:"name"`
	Email string `json:"email"`
	Bio   string `json:"bio"`
}

func (u WebUser) TableName() string {
	return "web_users"
}

// UsersController handles user-related requests
type UsersController struct {
	controllers.ApplicationController
	db gor.ORM
}

// NewUsersController creates a new users controller
func NewUsersController(db gor.ORM) *UsersController {
	return &UsersController{db: db}
}

// Index lists all users
func (c *UsersController) Index(ctx *gor.Context) error {
	var users []WebUser
	if err := c.db.FindAll(&users); err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch users",
		})
	}

	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

// Show displays a specific user
func (c *UsersController) Show(ctx *gor.Context) error {
	id := ctx.Param("id")

	userID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID",
		})
	}

	var user WebUser
	if err := c.db.Find(&user, uint(userID)); err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	return ctx.JSON(http.StatusOK, user)
}

// Create creates a new user
func (c *UsersController) Create(ctx *gor.Context) error {
	var user WebUser

	// Parse request body
	decoder := json.NewDecoder(ctx.Request.Body)
	if err := decoder.Decode(&user); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Create user in database
	if err := c.db.Create(&user); err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create user",
		})
	}

	return ctx.JSON(http.StatusCreated, user)
}

// Update updates an existing user
func (c *UsersController) Update(ctx *gor.Context) error {
	id := ctx.Param("id")

	userID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID",
		})
	}

	// Find existing user
	var user WebUser
	if err := c.db.Find(&user, uint(userID)); err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	// Parse request body
	var updates map[string]interface{}
	decoder := json.NewDecoder(ctx.Request.Body)
	if err := decoder.Decode(&updates); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Update user fields
	if name, ok := updates["name"].(string); ok {
		user.Name = name
	}
	if email, ok := updates["email"].(string); ok {
		user.Email = email
	}
	if bio, ok := updates["bio"].(string); ok {
		user.Bio = bio
	}

	// Save updates
	if err := c.db.Update(&user); err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update user",
		})
	}

	return ctx.JSON(http.StatusOK, user)
}

// Destroy deletes a user
func (c *UsersController) Destroy(ctx *gor.Context) error {
	id := ctx.Param("id")

	userID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID",
		})
	}

	// Find user
	var user WebUser
	if err := c.db.Find(&user, uint(userID)); err != nil {
		return ctx.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	// Delete user
	if err := c.db.Delete(&user); err != nil {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete user",
		})
	}

	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "User deleted successfully",
	})
}

// HomeController handles home page
type HomeController struct {
	controllers.ApplicationController
}

// Index renders the home page
func (c *HomeController) Index(ctx *gor.Context) error {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Gor Framework</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
               margin: 0; padding: 20px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
               min-height: 100vh; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 40px;
                    border-radius: 10px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
        h1 { color: #333; margin: 0 0 10px 0; }
        p { color: #666; line-height: 1.6; }
        .badge { display: inline-block; padding: 5px 10px; background: #667eea; color: white;
                border-radius: 5px; font-size: 12px; margin-left: 10px; }
        .routes { margin-top: 30px; }
        .route { background: #f8f9fa; padding: 10px; margin: 10px 0; border-radius: 5px;
                font-family: monospace; }
        .method { font-weight: bold; color: #667eea; display: inline-block; width: 70px; }
        a { color: #667eea; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 Gor Framework <span class="badge">v0.1.0</span></h1>
        <p>Welcome to <strong>Gor</strong> - a Rails-inspired web framework for Go!</p>

        <div class="routes">
            <h2>API Routes</h2>
            <div class="route">
                <span class="method">GET</span> <a href="/api/users">/api/users</a> - List all users
            </div>
            <div class="route">
                <span class="method">GET</span> /api/users/:id - Get a specific user
            </div>
            <div class="route">
                <span class="method">POST</span> /api/users - Create a new user
            </div>
            <div class="route">
                <span class="method">PUT</span> /api/users/:id - Update a user
            </div>
            <div class="route">
                <span class="method">DELETE</span> /api/users/:id - Delete a user
            </div>
        </div>

        <div class="routes">
            <h2>Features</h2>
            <ul>
                <li>✅ Rails-style routing with RESTful resources</li>
                <li>✅ Type-safe ORM with migrations</li>
                <li>✅ Middleware support (Logger, CORS, Recovery)</li>
                <li>✅ JSON API responses</li>
                <li>✅ Database-backed models</li>
                <li>🔄 Template engine (coming soon)</li>
                <li>🔄 Background jobs (coming soon)</li>
                <li>🔄 WebSocket support (coming soon)</li>
            </ul>
        </div>

        <p style="margin-top: 30px; padding-top: 20px; border-top: 1px solid #e0e0e0; font-size: 14px; color: #999;">
            Built with Go | Inspired by Rails | Made for Developer Happiness
        </p>
    </div>
</body>
</html>
`
	return ctx.HTML(http.StatusOK, html)
}

// Simplified Application struct for this example
type SimpleApp struct {
	router gor.Router
	orm    gor.ORM
}

func (a *SimpleApp) Start(ctx context.Context) error { return nil }
func (a *SimpleApp) Stop(ctx context.Context) error  { return nil }
func (a *SimpleApp) Router() gor.Router              { return a.router }
func (a *SimpleApp) ORM() gor.ORM                    { return a.orm }
func (a *SimpleApp) Queue() gor.Queue                { return nil }
func (a *SimpleApp) Cache() gor.Cache                { return nil }
func (a *SimpleApp) Cable() gor.Cable                { return nil }
func (a *SimpleApp) Auth() interface{}               { return nil }
func (a *SimpleApp) Config() gor.Config              { return nil }

func main() {
	// Initialize ORM
	config := gor.DatabaseConfig{
		Driver:   "sqlite3",
		Database: "./webapp.db",
	}

	gorORM := orm.NewORM(config)
	ctx := context.Background()

	if err := gorORM.Connect(ctx, config); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer gorORM.Close()

	// Register models
	if err := gorORM.Register(&WebUser{}); err != nil {
		log.Fatal("Failed to register models:", err)
	}

	// Create sample users
	sampleUsers := []WebUser{
		{Name: "John Doe", Email: "john@example.com", Bio: "Software Developer"},
		{Name: "Jane Smith", Email: "jane@example.com", Bio: "Product Manager"},
		{Name: "Bob Wilson", Email: "bob@example.com", Bio: "Designer"},
	}

	for _, user := range sampleUsers {
		// Check if user already exists
		var existingUser WebUser
		err := gorORM.Query(&WebUser{}).Where("email = ?", user.Email).First(&existingUser)
		if err != nil {
			// User doesn't exist, create it
			if err := gorORM.Create(&user); err != nil {
				log.Printf("Failed to create sample user: %v", err)
			}
		}
	}

	// Create application
	app := &SimpleApp{
		orm: gorORM,
	}

	// Initialize router
	appRouter := router.NewRouter(app)

	// Add middleware
	appRouter.Use(
		middleware.Logger(),
		middleware.Recovery(),
		middleware.CORS(middleware.CORSOptions{
			AllowOrigin:  "*",
			AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
			AllowHeaders: "Content-Type,Authorization",
		}),
		middleware.RequestID(),
	)

	// Define routes
	homeController := &HomeController{}
	appRouter.GET("/", homeController.Index)

	// API routes
	appRouter.Namespace("/api", func(r gor.Router) {
		// Users resource
		usersController := NewUsersController(gorORM)
		r.Resources("users", usersController)
	})

	// Static files (if any)
	appRouter.Use(middleware.Static("/static", "./static"))

	// Update app router reference
	app.router = appRouter

	// Print registered routes
	if r, ok := appRouter.(*router.GorRouter); ok {
		r.PrintRoutes()
	}

	// Start server
	fmt.Println("\n🚀 Gor server starting on http://localhost:8080")
	fmt.Println("   Press Ctrl+C to stop")

	server := &http.Server{
		Addr:              ":8080",
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
