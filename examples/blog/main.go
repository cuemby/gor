package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cuemby/gor/pkg/gor"
)

// Base Model struct for all models
type Model struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// H is a shortcut for map[string]interface{}
type H = map[string]interface{}

// Models
type User struct {
	Model
	Email        string    `db:"email" validate:"required,email"`
	Name         string    `db:"name" validate:"required"`
	PasswordHash string    `db:"password_hash"`
	Bio          string    `db:"bio"`
	Posts        []Post    `db:"-"`
	Comments     []Comment `db:"-"`
}

type Post struct {
	Model
	Title       string    `db:"title" validate:"required,max=255"`
	Slug        string    `db:"slug" validate:"required"`
	Body        string    `db:"body" validate:"required"`
	Excerpt     string    `db:"excerpt"`
	Published   bool      `db:"published" default:"false"`
	PublishedAt time.Time `db:"published_at"`
	ViewCount   int       `db:"view_count" default:"0"`
	AuthorID    int       `db:"author_id"`
	Author      *User     `db:"-"`
	Comments    []Comment `db:"-"`
	Tags        []Tag     `db:"-"`
}

type Comment struct {
	Model
	Body     string `db:"body" validate:"required"`
	PostID   int    `db:"post_id"`
	UserID   int    `db:"user_id"`
	Approved bool   `db:"approved" default:"true"`
	Post     *Post  `db:"-"`
	User     *User  `db:"-"`
}

type Tag struct {
	Model
	Name  string `db:"name" validate:"required"`
	Slug  string `db:"slug" validate:"required"`
	Posts []Post `db:"-"`
}

// Controllers
type HomeController struct{}

func (c *HomeController) Index(ctx *gor.Context) error {
	// Get recent posts
	// In a real implementation, fetch from database
	var posts []Post
	var err error

	if err != nil {
		return err
	}

	// Get popular tags
	// In a real implementation, fetch popular tags
	var tags []Tag

	return ctx.HTML(http.StatusOK, fmt.Sprintf(`
		<h1>Blog - Home</h1>
		<p>Posts: %d</p>
		<p>Tags: %d</p>
	`, len(posts), len(tags)))
}

type PostsController struct{}

func (c *PostsController) Index(ctx *gor.Context) error {
	page := ctx.QueryParam("page")
	if page == "" {
		page = "1"
	}
	tag := ctx.QueryParam("tag")

	// In a real implementation, fetch posts with pagination
	var posts []Post
	_ = tag

	return ctx.HTML(http.StatusOK, fmt.Sprintf(`
		<h1>All Posts</h1>
		<p>Page: %s</p>
		<p>Posts: %d</p>
	`, page, len(posts)))
}

func (c *PostsController) Show(ctx *gor.Context) error {
	slug := ctx.Param("slug")

	// In a real implementation, check cache
	_ = slug

	// Load from database
	// In a real implementation, fetch from database
	post := &Post{
		Model: Model{ID: 1},
		Title: "Sample Post",
		Slug: slug,
	}

	// In a real implementation, increment view count and cache
	_ = post

	// Load related posts
	// In a real implementation, fetch related posts
	var related []Post

	return ctx.HTML(http.StatusOK, fmt.Sprintf(`
		<h1>%s</h1>
		<p>Related posts: %d</p>
	`, post.Title, len(related)))
}

func (c *PostsController) New(ctx *gor.Context) error {
	if ctx.User == nil {
		return ctx.Redirect(http.StatusFound, "/login")
	}

	return ctx.HTML(http.StatusOK, `
		<h1>New Post</h1>
		<form method="post" action="/posts">
			<input type="text" name="title" placeholder="Title">
			<textarea name="body" placeholder="Content"></textarea>
			<button type="submit">Create Post</button>
		</form>
	`)
}

func (c *PostsController) Create(ctx *gor.Context) error {
	if ctx.User == nil {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Login required",
		})
	}

	// Get form data
	title := ctx.Request.FormValue("title")
	body := ctx.Request.FormValue("body")

	post := Post{
		Model: Model{ID: 1},
		Title: title,
		Body: body,
		Slug: slugify(title),
	}

	// In a real implementation, save to database
	_ = post

	return ctx.Redirect(http.StatusFound, "/posts/"+post.Slug)
}

type CommentsController struct{}

func (c *CommentsController) Create(ctx *gor.Context) error {
	if ctx.User == nil {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Login required",
		})
	}

	postID := ctx.Param("post_id")
	body := ctx.Request.FormValue("body")

	comment := Comment{
		Model: Model{ID: 1},
		Body: body,
		PostID: atoi(postID),
	}

	// In a real implementation, save to database
	_ = comment

	return ctx.JSON(201, comment)
}

type AuthController struct{}

func (c *AuthController) LoginForm(ctx *gor.Context) error {
	return ctx.HTML(http.StatusOK, `
		<h1>Login</h1>
		<form method="post" action="/login">
			<input type="email" name="email" placeholder="Email">
			<input type="password" name="password" placeholder="Password">
			<button type="submit">Login</button>
		</form>
	`)
}

func (c *AuthController) Login(ctx *gor.Context) error {
	email := ctx.Request.FormValue("email")
	password := ctx.Request.FormValue("password")

	// In a real implementation, authenticate user
	_ = email
	_ = password

	returnTo := ctx.QueryParam("return_to")
	if returnTo == "" {
		returnTo = "/"
	}
	return ctx.Redirect(http.StatusFound, returnTo)
}

func (c *AuthController) Logout(ctx *gor.Context) error {
	// In a real implementation, logout user
	return ctx.Redirect(http.StatusFound, "/")
}

func (c *AuthController) RegisterForm(ctx *gor.Context) error {
	return ctx.HTML(http.StatusOK, `
		<h1>Register</h1>
		<form method="post" action="/register">
			<input type="text" name="name" placeholder="Name">
			<input type="email" name="email" placeholder="Email">
			<input type="password" name="password" placeholder="Password">
			<button type="submit">Register</button>
		</form>
	`)
}

func (c *AuthController) Register(ctx *gor.Context) error {
	name := ctx.Request.FormValue("name")
	email := ctx.Request.FormValue("email")
	password := ctx.Request.FormValue("password")

	user := User{
		Model: Model{ID: 1},
		Name: name,
		Email: email,
		PasswordHash: password, // In real implementation, hash this
	}

	// In a real implementation, save to database
	_ = user

	return ctx.Redirect(http.StatusFound, "/")
}

// Background Jobs - simplified for example
type IncrementViewJob struct {
	PostID int `json:"post_id"`
}

func (j *IncrementViewJob) Perform(ctx context.Context) error {
	// In real implementation, increment view count
	log.Printf("Incrementing view count for post %d", j.PostID)
	return nil
}

// Application setup
var app *BlogApplication

type BlogApplication struct {
	orm    gor.ORM
	router gor.Router
	queue  gor.Queue
	cache  gor.Cache
	cable  gor.Cable
	auth   gor.Auth
}

func (a *BlogApplication) Start(ctx context.Context) error {
	// Start HTTP server
	server := &http.Server{
		Addr: ":3000",
		Handler: a.router,
	}

	log.Println("Starting server on :3000")
	return server.ListenAndServe()
}

func (a *BlogApplication) Router() gor.Router { return a.router }
func (a *BlogApplication) ORM() gor.ORM       { return a.orm }
func (a *BlogApplication) Queue() gor.Queue   { return a.queue }
func (a *BlogApplication) Cache() gor.Cache   { return a.cache }
func (a *BlogApplication) Cable() gor.Cable   { return a.cable }
func (a *BlogApplication) Auth() gor.Auth     { return a.auth }

func main() {
	// For this simplified example, create minimal app
	app = &BlogApplication{
		orm: nil, // Would be initialized with real ORM
		router: &SimpleRouter{
			routes: make(map[string]map[string]gor.HandlerFunc),
		},
		queue: nil, // Would be initialized with real queue
		cache: nil, // Would be initialized with real cache
		cable: nil, // Would be initialized with real cable
		auth: nil, // Would be initialized with real auth
	}

	// Configure routes
	configureRoutes(app.router)

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// Start application
	if err := app.Start(ctx); err != nil {
		log.Fatal(err)
	}
}

func configureRoutes(r gor.Router) {
	// Create controller instances
	homeController := &HomeController{}
	postsController := &PostsController{}
	commentsController := &CommentsController{}
	authController := &AuthController{}

	// Public routes
	r.GET("/", func(ctx *gor.Context) error {
		return homeController.Index(ctx)
	})

	r.GET("/posts", func(ctx *gor.Context) error {
		return postsController.Index(ctx)
	})

	r.GET("/posts/:slug", func(ctx *gor.Context) error {
		return postsController.Show(ctx)
	})

	// Authentication routes
	r.GET("/login", func(ctx *gor.Context) error {
		return authController.LoginForm(ctx)
	})

	r.POST("/login", func(ctx *gor.Context) error {
		return authController.Login(ctx)
	})

	r.GET("/register", func(ctx *gor.Context) error {
		return authController.RegisterForm(ctx)
	})

	r.POST("/register", func(ctx *gor.Context) error {
		return authController.Register(ctx)
	})

	r.POST("/logout", func(ctx *gor.Context) error {
		return authController.Logout(ctx)
	})

	// Protected routes
	r.GET("/posts/new", func(ctx *gor.Context) error {
		return postsController.New(ctx)
	})

	r.POST("/posts", func(ctx *gor.Context) error {
		return postsController.Create(ctx)
	})

	r.POST("/posts/:post_id/comments", func(ctx *gor.Context) error {
		return commentsController.Create(ctx)
	})
}

// Simple router implementation for the example
type SimpleRouter struct {
	routes map[string]map[string]gor.HandlerFunc
}

func (r *SimpleRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Create context
	ctx := &gor.Context{
		Request: req,
		Response: w,
		Params: make(map[string]string),
		Query: req.URL.Query(),
	}

	// Find and execute handler
	if methods, ok := r.routes[req.URL.Path]; ok {
		if handler, ok := methods[req.Method]; ok {
			if err := handler(ctx); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}

	http.NotFound(w, req)
}

func (r *SimpleRouter) Resources(name string, controller gor.Controller) gor.Router { return r }
func (r *SimpleRouter) Resource(name string, controller gor.Controller) gor.Router { return r }

func (r *SimpleRouter) GET(path string, handler gor.HandlerFunc) gor.Router {
	if r.routes[path] == nil {
		r.routes[path] = make(map[string]gor.HandlerFunc)
	}
	r.routes[path][http.MethodGet] = handler
	return r
}

func (r *SimpleRouter) POST(path string, handler gor.HandlerFunc) gor.Router {
	if r.routes[path] == nil {
		r.routes[path] = make(map[string]gor.HandlerFunc)
	}
	r.routes[path][http.MethodPost] = handler
	return r
}

func (r *SimpleRouter) PUT(path string, handler gor.HandlerFunc) gor.Router { return r }
func (r *SimpleRouter) PATCH(path string, handler gor.HandlerFunc) gor.Router { return r }
func (r *SimpleRouter) DELETE(path string, handler gor.HandlerFunc) gor.Router { return r }
func (r *SimpleRouter) Namespace(prefix string, fn func(gor.Router)) gor.Router { return r }
func (r *SimpleRouter) Group(middleware ...gor.MiddlewareFunc) gor.Router { return r }
func (r *SimpleRouter) Use(middleware ...gor.MiddlewareFunc) gor.Router { return r }
func (r *SimpleRouter) Named(name string) gor.Router { return r }

// Helpers
func slugify(s string) string {
	// Simple slugification
	return strings.ToLower(strings.ReplaceAll(s, " ", "-"))
}

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}