# Gor API Reference

Complete API documentation for the Gor web framework.

## Table of Contents

- [Application](#application)
- [Router](#router)
- [Controllers](#controllers)
- [Context](#context)
- [ORM](#orm)
- [Queue](#queue)
- [Cache](#cache)
- [Cable](#cable)
- [Authentication](#authentication)
- [Middleware](#middleware)
- [Testing](#testing)
- [Configuration](#configuration)

## Application

### Creating an Application

```go
import "github.com/ar4mirez/gor/pkg/gor"

app := gor.NewApplication(&gor.Config{
    Port:        3000,
    Environment: "development",
    Database: gor.DatabaseConfig{
        Driver:   "postgres",
        Host:     "localhost",
        Database: "myapp",
    },
})
```

### Application Methods

```go
// Start the application
app.Start(context.Background())

// Access components
router := app.Router()
orm := app.ORM()
queue := app.Queue()
cache := app.Cache()
cable := app.Cable()

// Configuration
config := app.Config()
env := app.Environment()
```

## Router

### Basic Routing

```go
router := app.Router()

// HTTP verbs
router.GET("/path", handler, "action")
router.POST("/path", handler, "action")
router.PUT("/path", handler, "action")
router.PATCH("/path", handler, "action")
router.DELETE("/path", handler, "action")
router.OPTIONS("/path", handler, "action")
router.HEAD("/path", handler, "action")
```

### RESTful Resources

```go
// Full resource (all 7 actions)
router.Resource("posts", &PostsController{})

// Limited resource
router.Resource("posts", &PostsController{}, gor.Only("index", "show"))
router.Resource("posts", &PostsController{}, gor.Except("destroy"))

// Nested resources
router.Resource("posts", &PostsController{}, func(r gor.Router) {
    r.Resource("comments", &CommentsController{})
})
```

### Route Groups

```go
// Group with prefix
router.Group("/api", func(api gor.Router) {
    api.GET("/users", handler, "index")
    api.GET("/posts", handler, "index")
})

// Group with middleware
router.Group(func(r gor.Router) {
    r.Use(middleware.Auth())
    r.Resource("admin/users", &AdminUsersController{})
})
```

### Named Routes

```go
// Define named route
router.GET("/users/:id", handler, "user")

// Generate URL
url := router.Path("user", gor.H{"id": 123}) // "/users/123"

// In templates
// {{path "user" "id" .UserID}}
```

### Route Constraints

```go
// Parameter constraints
router.GET("/posts/:id", handler, "show").Where("id", "[0-9]+")

// Custom constraints
router.GET("/users/:username", handler, "profile").
    Where("username", "[a-z][a-z0-9_-]{2,15}")
```

## Controllers

### Basic Controller

```go
type PostsController struct {
    gor.BaseController
}

// RESTful actions
func (c *PostsController) Index(ctx gor.Context) error
func (c *PostsController) Show(ctx gor.Context) error
func (c *PostsController) New(ctx gor.Context) error
func (c *PostsController) Create(ctx gor.Context) error
func (c *PostsController) Edit(ctx gor.Context) error
func (c *PostsController) Update(ctx gor.Context) error
func (c *PostsController) Destroy(ctx gor.Context) error
```

### Controller Callbacks

```go
func (c *PostsController) BeforeAction(ctx gor.Context, action string) error {
    // Run before every action
    return nil
}

func (c *PostsController) AfterAction(ctx gor.Context, action string) error {
    // Run after every action
    return nil
}

func (c *PostsController) BeforeFilter(actions ...string) gor.MiddlewareFunc {
    // Apply to specific actions
    return func(next gor.HandlerFunc) gor.HandlerFunc {
        return func(ctx gor.Context) error {
            // Filter logic
            return next(ctx)
        }
    }
}
```

### Controller Helpers

```go
func (c *PostsController) Show(ctx gor.Context) error {
    // Access ORM
    post, err := c.ORM().Find[Post](ctx.Request().Context(), ctx.Param("id"))

    // Access cache
    cached := c.Cache().Get("post:" + ctx.Param("id"))

    // Access current user
    user := c.CurrentUser(ctx)

    // Flash messages
    c.Flash(ctx, "success", "Post updated successfully")

    // Redirect with flash
    return c.RedirectWithFlash(ctx, "/posts", "info", "Redirected")
}
```

## Context

### Request Data

```go
// URL parameters
id := ctx.Param("id")

// Query parameters
page := ctx.Query("page")
sort := ctx.QueryDefault("sort", "created_at")

// Form data
title := ctx.FormValue("title")

// JSON binding
var post Post
err := ctx.Bind(&post)

// File uploads
file, header, err := ctx.FormFile("avatar")

// Headers
auth := ctx.Header("Authorization")

// Cookies
session := ctx.Cookie("session_id")
```

### Response Methods

```go
// HTML rendering
ctx.Render("template", data)

// JSON response
ctx.JSON(200, gor.H{"status": "success"})

// Text response
ctx.Text(200, "OK")

// File download
ctx.File("path/to/file.pdf")

// Redirect
ctx.Redirect("/posts")

// Status codes
ctx.NotFound("Page not found")
ctx.Unauthorized("Login required")
ctx.Forbidden("Access denied")
ctx.BadRequest("Invalid input")
ctx.InternalServerError("Something went wrong")
```

### Context Storage

```go
// Set value
ctx.Set("user_id", 123)

// Get value
userID := ctx.Get("user_id").(int)

// Check existence
if val, exists := ctx.GetOk("user_id"); exists {
    // Use val
}
```

## ORM

### Model Definition

```go
type Post struct {
    gor.Model
    Title       string    `db:"title" validate:"required,max=255"`
    Body        string    `db:"body" validate:"required"`
    Published   bool      `db:"published" default:"false"`
    PublishedAt time.Time `db:"published_at"`
    AuthorID    int       `db:"author_id"`

    // Associations
    Author   *User      `belongs_to:"users"`
    Comments []Comment  `has_many:"comments"`
    Tags     []Tag      `many_to_many:"tags,post_tags"`
}

// Callbacks
func (p *Post) BeforeSave(ctx context.Context) error
func (p *Post) AfterSave(ctx context.Context) error
func (p *Post) BeforeCreate(ctx context.Context) error
func (p *Post) AfterCreate(ctx context.Context) error
func (p *Post) BeforeUpdate(ctx context.Context) error
func (p *Post) AfterUpdate(ctx context.Context) error
func (p *Post) BeforeDelete(ctx context.Context) error
func (p *Post) AfterDelete(ctx context.Context) error

// Validations
func (p *Post) Validate() error {
    if p.Title == "" {
        return errors.New("title is required")
    }
    return nil
}

// Scopes
func (Post) Scopes() map[string]func(gor.Query) gor.Query {
    return map[string]func(gor.Query) gor.Query{
        "published": func(q gor.Query) gor.Query {
            return q.Where("published = ?", true)
        },
    }
}
```

### Querying

```go
orm := app.ORM()

// Find by ID
post, err := orm.Find[Post](ctx, 1)

// Find by attribute
post, err := orm.FindBy[Post](ctx, "slug", "hello-world")

// First record
post, err := orm.First[Post](ctx)

// Last record
post, err := orm.Last[Post](ctx)

// All records
posts, err := orm.All[Post](ctx)

// Where conditions
posts, err := orm.Where("published = ?", true).All[Post](ctx)
posts, err := orm.Where("created_at > ?", time.Now().AddDate(0, -1, 0)).All[Post](ctx)

// Multiple conditions
posts, err := orm.
    Where("published = ?", true).
    Where("author_id = ?", userID).
    All[Post](ctx)

// OR conditions
posts, err := orm.
    Where("title LIKE ?", "%golang%").
    Or("body LIKE ?", "%golang%").
    All[Post](ctx)

// Ordering
posts, err := orm.Order("created_at DESC").All[Post](ctx)
posts, err := orm.Order("published DESC, created_at DESC").All[Post](ctx)

// Limit and Offset
posts, err := orm.Limit(10).Offset(20).All[Post](ctx)

// Select specific columns
posts, err := orm.Select("id", "title", "created_at").All[Post](ctx)

// Distinct
titles, err := orm.Distinct("title").All[Post](ctx)

// Count
count, err := orm.Where("published = ?", true).Count[Post](ctx)

// Exists
exists, err := orm.Where("slug = ?", "hello-world").Exists[Post](ctx)

// Group and Having
stats, err := orm.
    Select("author_id", "COUNT(*) as post_count").
    Group("author_id").
    Having("COUNT(*) > ?", 5).
    All[PostStats](ctx)
```

### Associations

```go
// Eager loading
posts, err := orm.Includes("Author", "Comments").All[Post](ctx)

// Nested eager loading
posts, err := orm.Includes("Author", "Comments.User").All[Post](ctx)

// Join queries
posts, err := orm.
    Joins("Author").
    Where("users.name = ?", "John").
    All[Post](ctx)

// Has many
user, _ := orm.Find[User](ctx, 1)
posts, err := orm.HasMany(&user, "Posts").All[Post](ctx)

// Belongs to
post, _ := orm.Find[Post](ctx, 1)
author, err := orm.BelongsTo(&post, "Author").First[User](ctx)

// Many to many
post, _ := orm.Find[Post](ctx, 1)
tags, err := orm.ManyToMany(&post, "Tags").All[Tag](ctx)
```

### Transactions

```go
err := orm.Transaction(func(tx gor.Transaction) error {
    // Create user
    user := &User{Name: "John"}
    if err := tx.Create(ctx, user); err != nil {
        return err
    }

    // Create post
    post := &Post{
        Title:    "Hello",
        AuthorID: user.ID,
    }
    if err := tx.Create(ctx, post); err != nil {
        return err
    }

    return nil
})
```

### Raw SQL

```go
// Raw query
var posts []Post
err := orm.Raw("SELECT * FROM posts WHERE published = ?", true).Scan(&posts)

// Exec
result, err := orm.Exec("UPDATE posts SET views = views + 1 WHERE id = ?", postID)
```

## Queue

### Job Definition

```go
type EmailJob struct {
    gor.Job
    To      string `json:"to"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}

func (j *EmailJob) Perform(ctx context.Context) error {
    // Send email
    return mailer.Send(j.To, j.Subject, j.Body)
}

// Optional: configure job options
func (j *EmailJob) Options() gor.JobOptions {
    return gor.JobOptions{
        Queue:    "mailers",
        Priority: gor.PriorityHigh,
        Retries:  3,
        Timeout:  30 * time.Second,
    }
}
```

### Enqueueing Jobs

```go
queue := app.Queue()

// Enqueue immediately
err := queue.Enqueue(&EmailJob{
    To:      "user@example.com",
    Subject: "Welcome",
    Body:    "Thanks for signing up!",
})

// Enqueue with delay
err := queue.EnqueueIn(5*time.Minute, &EmailJob{...})

// Enqueue at specific time
err := queue.EnqueueAt(time.Now().Add(24*time.Hour), &EmailJob{...})

// Enqueue with options
err := queue.EnqueueWithOptions(&EmailJob{...}, gor.JobOptions{
    Queue:    "critical",
    Priority: gor.PriorityUrgent,
})
```

### Recurring Jobs

```go
// Schedule recurring job
queue.Schedule("0 0 * * *", &DailyReportJob{}) // Daily at midnight
queue.Schedule("*/5 * * * *", &HealthCheckJob{}) // Every 5 minutes

// Named schedules
queue.ScheduleNamed("daily_report", "0 0 * * *", &DailyReportJob{})

// Remove scheduled job
queue.Unschedule("daily_report")
```

### Queue Management

```go
// Get queue stats
stats := queue.Stats()
fmt.Printf("Pending: %d, Processing: %d, Failed: %d\n",
    stats.Pending, stats.Processing, stats.Failed)

// Clear queue
queue.Clear("default")

// Retry failed jobs
queue.RetryFailed()

// Pause/Resume processing
queue.Pause()
queue.Resume()
```

## Cache

### Basic Operations

```go
cache := app.Cache()

// Set value
err := cache.Set("key", "value", 1*time.Hour)

// Get value
value, exists := cache.Get("key")

// Delete value
cache.Delete("key")

// Check existence
exists := cache.Exists("key")

// Clear all
cache.Clear()
```

### Advanced Caching

```go
// Fetch with callback
value, err := cache.Fetch("expensive_operation", 1*time.Hour, func() (interface{}, error) {
    // Expensive operation
    return computeExpensiveValue(), nil
})

// Increment/Decrement
cache.Increment("counter", 1)
cache.Decrement("counter", 1)

// Tagged caching
cache.Tagged("users", "posts").Set("key", value, 1*time.Hour)
cache.Tagged("users").Clear() // Clear all user-related cache

// Multi-get/set
values := cache.MGet("key1", "key2", "key3")
cache.MSet(map[string]interface{}{
    "key1": "value1",
    "key2": "value2",
})
```

### Cache Stores

```go
// Configure different stores
cache := gor.NewCache(gor.CacheConfig{
    DefaultStore: "memory",
    Stores: map[string]gor.CacheStore{
        "memory": gor.NewMemoryStore(100), // Max 100 items
        "redis":  gor.NewRedisStore("localhost:6379"),
        "database": gor.NewDatabaseStore(db),
    },
})

// Use specific store
cache.Store("redis").Set("key", value, 1*time.Hour)
```

## Cable

### WebSocket Server

```go
// Controller action
func (c *ChatController) Connect(ctx gor.Context) error {
    return ctx.Cable().HandleWebSocket(func(conn *gor.WebSocketConnection) {
        // Authentication
        user := ctx.CurrentUser()
        if user == nil {
            conn.Close()
            return
        }

        // Subscribe to channels
        conn.Subscribe("chat:lobby")
        conn.Subscribe("user:" + fmt.Sprint(user.ID))

        // Handle incoming messages
        for msg := range conn.Messages() {
            switch msg.Type {
            case "chat":
                // Broadcast to channel
                ctx.Cable().Broadcast("chat:lobby", gor.H{
                    "user":    user.Name,
                    "message": msg.Data,
                })
            case "typing":
                // Notify others
                ctx.Cable().BroadcastExcept("chat:lobby", conn.ID, gor.H{
                    "user":   user.Name,
                    "typing": true,
                })
            }
        }
    })
}
```

### Server-Sent Events

```go
func (c *NotificationsController) Stream(ctx gor.Context) error {
    return ctx.SSE(func(stream *gor.SSEStream) {
        // Subscribe to notifications
        notifications := make(chan Notification)
        go subscribeToNotifications(ctx.CurrentUser().ID, notifications)

        for notification := range notifications {
            stream.Send(gor.SSEEvent{
                Event: "notification",
                Data:  notification,
                ID:    fmt.Sprint(notification.ID),
            })
        }
    })
}
```

### Broadcasting

```go
cable := app.Cable()

// Broadcast to channel
cable.Broadcast("channel_name", data)

// Broadcast to specific connections
cable.BroadcastTo([]string{connID1, connID2}, data)

// Broadcast except certain connections
cable.BroadcastExcept("channel_name", excludeConnID, data)

// Direct message
cable.Send(connectionID, data)
```

## Authentication

### User Model

```go
type User struct {
    gor.Model
    gor.Authenticatable // Adds password_digest, remember_token, etc.

    Email    string `db:"email" validate:"required,email"`
    Name     string `db:"name"`
    Role     string `db:"role" default:"user"`
}

// Password management
user.SetPassword("secret123")
valid := user.CheckPassword("secret123")

// Token generation
token := user.GenerateToken()
user.GeneratePasswordResetToken()
```

### Authentication Methods

```go
auth := app.Auth()

// Login
user, err := auth.Authenticate(email, password)
if err != nil {
    // Invalid credentials
}

// Create session
err = auth.Login(ctx, user)

// Logout
auth.Logout(ctx)

// Current user
user := auth.CurrentUser(ctx)

// Check if logged in
if auth.IsLoggedIn(ctx) {
    // User is authenticated
}

// Remember me
auth.LoginWithRemember(ctx, user, 30*24*time.Hour)
```

### Authorization

```go
// Define abilities
auth.DefineAbility("admin", func(user *User, resource interface{}) bool {
    return user.Role == "admin"
})

auth.DefineAbility("edit_post", func(user *User, post *Post) bool {
    return user.ID == post.AuthorID || user.Role == "admin"
})

// Check abilities
if auth.Can(user, "edit_post", post) {
    // User can edit
}

if auth.Cannot(user, "delete_post", post) {
    // User cannot delete
}

// In controllers
func (c *PostsController) Update(ctx gor.Context) error {
    post, _ := c.ORM().Find[Post](ctx, ctx.Param("id"))

    if !c.Authorize(ctx, "edit_post", post) {
        return ctx.Forbidden("You cannot edit this post")
    }

    // Update post...
}
```

## Middleware

### Built-in Middleware

```go
router.Use(middleware.Logger())           // Request logging
router.Use(middleware.Recovery())         // Panic recovery
router.Use(middleware.RequestID())        // Request ID generation
router.Use(middleware.CORS())            // CORS headers
router.Use(middleware.RateLimit(100))    // Rate limiting
router.Use(middleware.Compress())        // Response compression
router.Use(middleware.Static("public"))  // Static file serving
router.Use(middleware.CSRF())           // CSRF protection
router.Use(middleware.Session())        // Session management
router.Use(middleware.Auth())          // Authentication
```

### Custom Middleware

```go
func TimingMiddleware() gor.MiddlewareFunc {
    return func(next gor.HandlerFunc) gor.HandlerFunc {
        return func(ctx gor.Context) error {
            start := time.Now()

            // Process request
            err := next(ctx)

            // Log timing
            duration := time.Since(start)
            ctx.Logger().Info("Request processed",
                "method", ctx.Request().Method,
                "path", ctx.Request().URL.Path,
                "duration", duration,
            )

            return err
        }
    }
}

// Use middleware
router.Use(TimingMiddleware())
```

### Conditional Middleware

```go
// Apply to specific routes
router.Group("/admin", func(r gor.Router) {
    r.Use(middleware.RequireAdmin())
    r.Resource("users", &AdminUsersController{})
})

// Skip middleware for certain paths
router.Use(middleware.Auth().Except("/login", "/register"))

// Only apply to certain paths
router.Use(middleware.RateLimit(10).Only("/api/*"))
```

## Testing

### Test Helpers

```go
import "github.com/ar4mirez/gor/pkg/testing"

func TestPostsController_Index(t *testing.T) {
    app := testing.NewTestApp(t)

    // Create test data
    app.Factory(&Post{Title: "Test Post"}).Create()

    // Make request
    resp := app.Get("/posts")

    // Assertions
    resp.AssertStatus(200)
    resp.AssertContains("Test Post")
    resp.AssertHeader("Content-Type", "text/html")
}
```

### Controller Tests

```go
func TestPostsController_Create(t *testing.T) {
    app := testing.NewTestApp(t)

    // Login user
    user := app.Factory(&User{}).Create()
    app.LoginAs(user)

    // Make POST request
    resp := app.Post("/posts", gor.H{
        "title": "New Post",
        "body":  "Post content",
    })

    // Assert redirect
    resp.AssertRedirect("/posts/1")

    // Verify database
    var post Post
    app.DB().First(&post)
    assert.Equal(t, "New Post", post.Title)
}
```

### Model Tests

```go
func TestPost_Validation(t *testing.T) {
    post := &Post{
        Title: "", // Empty title
        Body:  "Content",
    }

    err := post.Validate()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "title is required")
}
```

### Job Tests

```go
func TestEmailJob(t *testing.T) {
    app := testing.NewTestApp(t)

    job := &EmailJob{
        To:      "test@example.com",
        Subject: "Test",
        Body:    "Test email",
    }

    // Perform job
    err := job.Perform(context.Background())
    assert.NoError(t, err)

    // Verify email was sent
    assert.Equal(t, 1, app.Mailer().SentCount())
}
```

## Configuration

### Configuration Files

```yaml
# config/config.yml
app:
  name: MyApp
  port: 3000
  secret_key: ${SECRET_KEY}

database:
  driver: postgres
  host: ${DB_HOST}
  port: 5432
  database: myapp_${GOR_ENV}

cache:
  driver: redis
  host: localhost
  port: 6379

queue:
  driver: database
  workers: 10
```

### Accessing Configuration

```go
config := app.Config()

// Get values
port := config.GetInt("app.port")
dbHost := config.GetString("database.host")
workers := config.GetInt("queue.workers")

// Get with default
timeout := config.GetDurationDefault("http.timeout", 30*time.Second)

// Get nested config
dbConfig := config.GetMap("database")

// Check existence
if config.Has("feature.enabled") {
    // Feature is configured
}

// Watch for changes
config.Watch("app.debug", func(old, new interface{}) {
    fmt.Printf("Debug mode changed from %v to %v\n", old, new)
})
```

### Environment Configuration

```go
// Different configs per environment
// config/config.development.yml
// config/config.test.yml
// config/config.production.yml

env := app.Environment()

if app.IsDevelopment() {
    // Development specific code
}

if app.IsProduction() {
    // Production specific code
}

if app.IsTest() {
    // Test specific code
}
```

## Plugins

### Creating a Plugin

```go
type MetricsPlugin struct {
    gor.BasePlugin
}

func NewMetricsPlugin() *MetricsPlugin {
    return &MetricsPlugin{
        BasePlugin: gor.NewBasePlugin(gor.PluginMetadata{
            Name:        "metrics",
            Version:     "1.0.0",
            Description: "Application metrics collection",
        }),
    }
}

func (p *MetricsPlugin) Initialize(app gor.Application) error {
    // Setup metrics collection
    return nil
}

func (p *MetricsPlugin) Routes() []gor.Route {
    return []gor.Route{
        {Method: "GET", Path: "/metrics", Handler: p.metricsHandler},
    }
}

func (p *MetricsPlugin) Middleware() []gor.Middleware {
    return []gor.Middleware{
        {Name: "metrics", Handler: p.metricsMiddleware},
    }
}
```

### Using Plugins

```go
// Register plugin
app.Plugins().Register(NewMetricsPlugin())

// Load plugin from file
app.Plugins().Load("path/to/plugin.so")

// Install from registry
app.Plugins().Install("github.com/user/gor-metrics")
```