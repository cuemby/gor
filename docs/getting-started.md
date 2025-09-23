# Getting Started with Gor

This guide will walk you through creating your first Gor application.

## Prerequisites

- Go 1.21 or later
- SQLite, PostgreSQL, or MySQL (optional)

## Installation

### Install the Gor CLI

```bash
go install github.com/ar4mirez/gor/cmd/gor@latest
```

### Verify Installation

```bash
gor version
```

## Creating Your First Application

### 1. Generate a New Application

```bash
gor new blog
cd blog
```

This creates a new Gor application with the following structure:

```
blog/
├── app/
│   ├── controllers/     # Request handlers
│   ├── models/          # Data models
│   ├── views/           # HTML templates
│   ├── jobs/            # Background jobs
│   └── middleware/      # Custom middleware
├── config/
│   ├── config.yml       # Base configuration
│   └── routes.go        # Route definitions
├── db/
│   ├── migrations/      # Database migrations
│   └── seeds/           # Seed data
├── public/              # Static assets
├── test/                # Tests
├── go.mod               # Dependencies
└── main.go              # Application entry
```

### 2. Configure Database

Edit `config/config.yml`:

```yaml
development:
  database:
    driver: sqlite3
    database: blog_development.db
    pool: 5
    log: true

production:
  database:
    driver: postgres
    host: ${DB_HOST}
    port: 5432
    database: blog_production
    user: ${DB_USER}
    password: ${DB_PASSWORD}
    pool: 20
```

### 3. Generate a Model

```bash
gor generate model Post title:string body:text published:bool author_id:int
```

This generates:
- `app/models/post.go` - Model definition
- `db/migrations/[timestamp]_create_posts.go` - Migration file

### 4. Run Migrations

```bash
gor db:migrate
```

### 5. Generate a Controller

```bash
gor generate controller Posts
```

Or generate a complete scaffold (model + controller + views):

```bash
gor generate scaffold Post title:string body:text published:bool
```

### 6. Define Routes

Edit `config/routes.go`:

```go
func ConfigureRoutes(app *gor.Application) {
    router := app.Router()

    // RESTful resource
    router.Resource("posts", &controllers.PostsController{})

    // Custom routes
    router.GET("/", &controllers.HomeController{}, "Index")
    router.GET("/about", &controllers.PagesController{}, "About")

    // API routes
    router.Namespace("/api", func(api gor.Router) {
        api.Resource("posts", &api.PostsController{})
    })
}
```

### 7. Create a Controller

`app/controllers/posts_controller.go`:

```go
package controllers

import (
    "github.com/ar4mirez/gor/pkg/gor"
    "blog/app/models"
)

type PostsController struct {
    gor.BaseController
}

// GET /posts
func (c *PostsController) Index(ctx gor.Context) error {
    posts, err := c.ORM().All[models.Post](ctx.Request().Context())
    if err != nil {
        return err
    }

    return ctx.Render("posts/index", gor.H{
        "Title": "All Posts",
        "posts": posts,
    })
}

// GET /posts/:id
func (c *PostsController) Show(ctx gor.Context) error {
    id := ctx.Param("id")
    post, err := c.ORM().Find[models.Post](ctx.Request().Context(), id)
    if err != nil {
        return ctx.NotFound("Post not found")
    }

    return ctx.Render("posts/show", gor.H{
        "Title": post.Title,
        "post":  post,
    })
}

// POST /posts
func (c *PostsController) Create(ctx gor.Context) error {
    var post models.Post
    if err := ctx.Bind(&post); err != nil {
        return ctx.JSON(400, gor.H{"error": err.Error()})
    }

    if err := c.ORM().Create(ctx.Request().Context(), &post); err != nil {
        return err
    }

    return ctx.Redirect("/posts/" + fmt.Sprint(post.ID))
}
```

### 8. Create Views

`app/views/layouts/application.html`:

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}} - Blog</title>
    <link rel="stylesheet" href="/assets/application.css">
</head>
<body>
    <nav>
        <a href="/">Home</a>
        <a href="/posts">Posts</a>
    </nav>

    <main>
        {{template "content" .}}
    </main>

    <script src="/assets/application.js"></script>
</body>
</html>
```

`app/views/posts/index.html`:

```html
{{define "content"}}
<h1>All Posts</h1>

<a href="/posts/new">New Post</a>

{{range .posts}}
<article>
    <h2><a href="/posts/{{.ID}}">{{.Title}}</a></h2>
    <p>{{.Body | truncate 200}}</p>
    {{if .Published}}
        <span class="badge">Published</span>
    {{else}}
        <span class="badge draft">Draft</span>
    {{end}}
</article>
{{end}}
{{end}}
```

### 9. Start the Server

```bash
gor server
```

Visit http://localhost:3000 to see your application!

## Development Workflow

### Hot Reload

Gor automatically reloads when you change files:

```bash
gor server --dev
```

### Running Tests

```bash
gor test
```

### Database Console

```bash
gor db:console
```

### Generate Migration

```bash
gor generate migration AddViewsCountToPosts views_count:int:default=0
```

### Rollback Migration

```bash
gor db:rollback
```

## Working with Models

### Defining Models

```go
type Post struct {
    gor.Model
    Title     string    `db:"title" validate:"required"`
    Body      string    `db:"body" validate:"required"`
    Published bool      `db:"published" default:"false"`
    AuthorID  int       `db:"author_id"`
    Author    *User     `db:"-" belongs_to:"users"`
    Comments  []Comment `db:"-" has_many:"comments"`
    Tags      []Tag     `db:"-" many_to_many:"tags,post_tags"`
}
```

### Querying Data

```go
// Find by ID
post, err := orm.Find[Post](ctx, 1)

// Find with conditions
posts, err := orm.Where("published = ?", true).
    Order("created_at DESC").
    Limit(10).
    All[Post](ctx)

// Complex queries
posts, err := orm.
    Joins("Author").
    Where("author_id = ? AND published = ?", userID, true).
    All[Post](ctx)

// Scopes
type Post struct {
    gor.Model
    // fields...
}

func (Post) Scopes() map[string]func(gor.Query) gor.Query {
    return map[string]func(gor.Query) gor.Query{
        "published": func(q gor.Query) gor.Query {
            return q.Where("published = ?", true)
        },
        "recent": func(q gor.Query) gor.Query {
            return q.Order("created_at DESC").Limit(10)
        },
    }
}

// Use scopes
posts, err := orm.Scope("published", "recent").All[Post](ctx)
```

## Background Jobs

### Define a Job

```go
type EmailJob struct {
    gor.Job
    To      string
    Subject string
    Body    string
}

func (j *EmailJob) Perform(ctx context.Context) error {
    return mailer.Send(j.To, j.Subject, j.Body)
}
```

### Enqueue Jobs

```go
// Immediate execution
app.Queue().Enqueue(&EmailJob{
    To:      "user@example.com",
    Subject: "Welcome!",
    Body:    "Thanks for signing up",
})

// Delayed execution
app.Queue().EnqueueIn(5*time.Minute, &EmailJob{...})

// Scheduled execution
app.Queue().EnqueueAt(tomorrow, &EmailJob{...})
```

## Real-time Features

### WebSocket Connection

```go
func (c *ChatController) Connect(ctx gor.Context) error {
    return ctx.Cable().HandleWebSocket(func(conn *gor.WebSocketConnection) {
        // Subscribe to channel
        conn.Subscribe("chat:lobby")

        // Handle messages
        for msg := range conn.Messages() {
            // Broadcast to all subscribers
            ctx.Cable().Broadcast("chat:lobby", msg)
        }
    })
}
```

### Server-Sent Events

```go
func (c *NotificationsController) Stream(ctx gor.Context) error {
    return ctx.SSE(func(stream *gor.SSEStream) {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ticker.C:
                notifications := getNewNotifications()
                stream.Send("notification", notifications)
            case <-stream.Done():
                return
            }
        }
    })
}
```

## Authentication

### Setup Authentication

```bash
gor generate auth
```

This generates:
- User model with authentication fields
- Sessions controller for login/logout
- Registration controller
- Password reset functionality
- Authentication middleware

### Protect Routes

```go
func ConfigureRoutes(app *gor.Application) {
    router := app.Router()

    // Public routes
    router.GET("/login", &SessionsController{}, "New")
    router.POST("/login", &SessionsController{}, "Create")

    // Protected routes
    router.Group(func(r gor.Router) {
        r.Use(middleware.RequireAuth())

        r.Resource("posts", &PostsController{})
        r.GET("/profile", &UsersController{}, "Profile")
    })
}
```

### Current User

```go
func (c *PostsController) Create(ctx gor.Context) error {
    user := ctx.CurrentUser()

    post := &Post{
        Title:    ctx.FormValue("title"),
        Body:     ctx.FormValue("body"),
        AuthorID: user.ID,
    }

    return c.ORM().Create(ctx, post)
}
```

## Deployment

### Building for Production

```bash
gor build --production
```

### Docker Deployment

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o app

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/app .
COPY --from=builder /app/public ./public
COPY --from=builder /app/config ./config
CMD ["./app"]
```

### Environment Variables

```bash
export GOR_ENV=production
export GOR_DATABASE_URL=postgres://user:pass@host/db
export GOR_SECRET_KEY=your-secret-key
export GOR_PORT=8080
```

## Next Steps

- Read the [API Documentation](api.md) for detailed reference
- Check out [Example Applications](../examples/) for real-world usage
- Join the [Community Forum](https://forum.gor.dev) for help and discussions