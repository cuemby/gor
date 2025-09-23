package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cuemby/gor/internal/assets"
	"github.com/cuemby/gor/internal/auth"
	"github.com/cuemby/gor/internal/cable"
	"github.com/cuemby/gor/internal/cache"
	"github.com/cuemby/gor/internal/config"
	"github.com/cuemby/gor/internal/orm"
	"github.com/cuemby/gor/internal/queue"
	"github.com/cuemby/gor/internal/router"
	"github.com/cuemby/gor/internal/views"
	"github.com/cuemby/gor/pkg/gor"
)

// Models
type User struct {
	orm.Model
	Email        string    `db:"email" validate:"required,email"`
	Name         string    `db:"name" validate:"required"`
	PasswordHash string    `db:"password_hash"`
	Bio          string    `db:"bio"`
	Posts        []Post    `db:"-"`
	Comments     []Comment `db:"-"`
}

type Post struct {
	orm.Model
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
	orm.Model
	Body     string `db:"body" validate:"required"`
	PostID   int    `db:"post_id"`
	UserID   int    `db:"user_id"`
	Approved bool   `db:"approved" default:"true"`
	Post     *Post  `db:"-"`
	User     *User  `db:"-"`
}

type Tag struct {
	orm.Model
	Name  string `db:"name" validate:"required"`
	Slug  string `db:"slug" validate:"required"`
	Posts []Post `db:"-"`
}

// Controllers
type HomeController struct {
	gor.BaseController
}

func (c *HomeController) Index(ctx gor.Context) error {
	// Get recent posts
	posts, err := c.App().ORM().
		Where("published = ?", true).
		Order("published_at DESC").
		Limit(10).
		Includes("Author").
		All(ctx.Request().Context(), &Post{})

	if err != nil {
		return err
	}

	// Get popular tags
	tags, _ := c.App().ORM().
		Select("tags.*, COUNT(post_tags.post_id) as post_count").
		Joins("post_tags").
		Group("tags.id").
		Order("post_count DESC").
		Limit(20).
		All(ctx.Request().Context(), &Tag{})

	return ctx.Render("home/index", gor.H{
		"Title": "Blog - Home",
		"Posts": posts,
		"Tags":  tags,
	})
}

type PostsController struct {
	gor.BaseController
}

func (c *PostsController) Index(ctx gor.Context) error {
	page := ctx.QueryDefault("page", "1")
	tag := ctx.Query("tag")

	query := c.App().ORM().Where("published = ?", true)

	if tag != "" {
		query = query.
			Joins("post_tags").
			Joins("tags").
			Where("tags.slug = ?", tag)
	}

	posts, _ := query.
		Order("published_at DESC").
		Paginate(page, 20).
		Includes("Author", "Tags").
		All(ctx.Request().Context(), &Post{})

	return ctx.Render("posts/index", gor.H{
		"Title": "All Posts",
		"Posts": posts,
		"Tag":   tag,
	})
}

func (c *PostsController) Show(ctx gor.Context) error {
	slug := ctx.Param("slug")

	// Try cache first
	cacheKey := "post:" + slug
	if cached, exists := c.App().Cache().Get(cacheKey); exists {
		post := cached.(*Post)
		return ctx.Render("posts/show", gor.H{
			"Title": post.Title,
			"Post":  post,
		})
	}

	// Load from database
	post, err := c.App().ORM().
		Where("slug = ? AND published = ?", slug, true).
		Includes("Author", "Comments.User", "Tags").
		First(ctx.Request().Context(), &Post{})

	if err != nil {
		return ctx.NotFound("Post not found")
	}

	// Increment view count asynchronously
	c.App().Queue().Enqueue(&IncrementViewJob{PostID: post.ID})

	// Cache for 1 hour
	c.App().Cache().Set(cacheKey, post, 1*time.Hour)

	// Load related posts
	related, _ := c.App().ORM().
		Where("id != ? AND published = ?", post.ID, true).
		Order("RANDOM()").
		Limit(3).
		All(ctx.Request().Context(), &Post{})

	return ctx.Render("posts/show", gor.H{
		"Title":   post.Title,
		"Post":    post,
		"Related": related,
	})
}

func (c *PostsController) New(ctx gor.Context) error {
	if !c.RequireAuth(ctx) {
		return ctx.Redirect("/login")
	}

	return ctx.Render("posts/new", gor.H{
		"Title": "New Post",
	})
}

func (c *PostsController) Create(ctx gor.Context) error {
	if !c.RequireAuth(ctx) {
		return ctx.Unauthorized("Login required")
	}

	var post Post
	if err := ctx.Bind(&post); err != nil {
		return ctx.BadRequest(err.Error())
	}

	post.AuthorID = c.CurrentUser(ctx).ID
	post.Slug = slugify(post.Title)

	if err := c.App().ORM().Create(ctx.Request().Context(), &post); err != nil {
		c.Flash(ctx, "error", "Failed to create post")
		return ctx.Render("posts/new", gor.H{
			"Title": "New Post",
			"Post":  post,
			"Error": err.Error(),
		})
	}

	// Clear cache
	c.App().Cache().Tagged("posts").Clear()

	// Send notification to subscribers
	c.App().Queue().Enqueue(&NewPostNotificationJob{PostID: post.ID})

	c.Flash(ctx, "success", "Post created successfully")
	return ctx.Redirect("/posts/" + post.Slug)
}

type CommentsController struct {
	gor.BaseController
}

func (c *CommentsController) Create(ctx gor.Context) error {
	if !c.RequireAuth(ctx) {
		return ctx.Unauthorized("Login required")
	}

	postID := ctx.Param("post_id")
	var comment Comment
	if err := ctx.Bind(&comment); err != nil {
		return ctx.JSON(400, gor.H{"error": err.Error()})
	}

	comment.PostID = atoi(postID)
	comment.UserID = c.CurrentUser(ctx).ID

	if err := c.App().ORM().Create(ctx.Request().Context(), &comment); err != nil {
		return ctx.JSON(500, gor.H{"error": "Failed to create comment"})
	}

	// Broadcast new comment via WebSocket
	c.App().Cable().Broadcast("post:"+postID, gor.H{
		"type":    "new_comment",
		"comment": comment,
	})

	return ctx.JSON(201, comment)
}

type AuthController struct {
	gor.BaseController
}

func (c *AuthController) LoginForm(ctx gor.Context) error {
	return ctx.Render("auth/login", gor.H{
		"Title": "Login",
	})
}

func (c *AuthController) Login(ctx gor.Context) error {
	email := ctx.FormValue("email")
	password := ctx.FormValue("password")

	user, err := c.App().Auth().Authenticate(email, password)
	if err != nil {
		c.Flash(ctx, "error", "Invalid email or password")
		return ctx.Render("auth/login", gor.H{
			"Title": "Login",
			"Email": email,
		})
	}

	c.App().Auth().Login(ctx, user)
	c.Flash(ctx, "success", "Welcome back, "+user.Name)

	returnTo := ctx.QueryDefault("return_to", "/")
	return ctx.Redirect(returnTo)
}

func (c *AuthController) Logout(ctx gor.Context) error {
	c.App().Auth().Logout(ctx)
	c.Flash(ctx, "info", "You have been logged out")
	return ctx.Redirect("/")
}

func (c *AuthController) RegisterForm(ctx gor.Context) error {
	return ctx.Render("auth/register", gor.H{
		"Title": "Register",
	})
}

func (c *AuthController) Register(ctx gor.Context) error {
	var user User
	if err := ctx.Bind(&user); err != nil {
		return ctx.BadRequest(err.Error())
	}

	// Hash password
	hashedPassword, _ := auth.HashPassword(ctx.FormValue("password"))
	user.PasswordHash = hashedPassword

	if err := c.App().ORM().Create(ctx.Request().Context(), &user); err != nil {
		c.Flash(ctx, "error", "Registration failed")
		return ctx.Render("auth/register", gor.H{
			"Title": "Register",
			"User":  user,
			"Error": err.Error(),
		})
	}

	// Send welcome email
	c.App().Queue().Enqueue(&WelcomeEmailJob{UserID: user.ID})

	c.App().Auth().Login(ctx, &user)
	c.Flash(ctx, "success", "Welcome to our blog!")
	return ctx.Redirect("/")
}

// Background Jobs
type IncrementViewJob struct {
	gor.Job
	PostID int `json:"post_id"`
}

func (j *IncrementViewJob) Perform(ctx context.Context) error {
	// Increment view count
	return app.ORM().
		Where("id = ?", j.PostID).
		UpdateColumn("view_count", "view_count + 1").
		Error
}

type NewPostNotificationJob struct {
	gor.Job
	PostID int `json:"post_id"`
}

func (j *NewPostNotificationJob) Perform(ctx context.Context) error {
	// Send notifications to subscribers
	// Implementation here...
	return nil
}

type WelcomeEmailJob struct {
	gor.Job
	UserID int `json:"user_id"`
}

func (j *WelcomeEmailJob) Perform(ctx context.Context) error {
	// Send welcome email
	// Implementation here...
	return nil
}

// Application setup
var app *BlogApplication

type BlogApplication struct {
	orm      gor.ORM
	router   gor.Router
	queue    gor.Queue
	cache    gor.Cache
	cable    gor.Cable
	auth     gor.Auth
	template *views.TemplateEngine
	assets   *assets.Pipeline
	config   *config.Config
}

func (a *BlogApplication) Start(ctx context.Context) error {
	// Start queue workers
	go a.queue.Start(ctx)

	// Start cable server
	go a.cable.Start(ctx)

	// Start HTTP server
	return a.router.Start(":3000")
}

func (a *BlogApplication) Router() gor.Router { return a.router }
func (a *BlogApplication) ORM() gor.ORM       { return a.orm }
func (a *BlogApplication) Queue() gor.Queue   { return a.queue }
func (a *BlogApplication) Cache() gor.Cache   { return a.cache }
func (a *BlogApplication) Cable() gor.Cable   { return a.cable }
func (a *BlogApplication) Auth() gor.Auth     { return a.auth }

func main() {
	// Load configuration
	cfg, err := config.New("config")
	if err != nil {
		log.Fatal(err)
	}

	// Initialize ORM
	database, err := orm.New(&orm.Config{
		Driver:   cfg.GetString("database.driver"),
		Host:     cfg.GetString("database.host"),
		Database: cfg.GetString("database.database"),
		User:     cfg.GetString("database.user"),
		Password: cfg.GetString("database.password"),
	})
	if err != nil {
		log.Fatal(err)
	}

	// Run migrations
	database.AutoMigrate(&User{}, &Post{}, &Comment{}, &Tag{})

	// Initialize components
	app = &BlogApplication{
		orm:    database,
		router: router.New(),
		queue: queue.New(&queue.Config{
			Driver:  "database",
			Workers: cfg.GetInt("queue.workers"),
		}),
		cache: cache.New(&cache.Config{
			Driver: cfg.GetString("cache.driver"),
		}),
		cable: cable.New(&cable.Config{
			Driver: "database",
		}),
		auth: auth.New(&auth.Config{
			SessionKey: cfg.GetString("app.secret_key"),
		}),
		template: views.NewTemplateEngine("views", true),
		assets: assets.New(&assets.Config{
			PublicPath: "public",
		}),
		config: cfg,
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
	// Static files
	r.Static("/assets", "public")

	// Public routes
	r.GET("/", &HomeController{}, "Index")
	r.GET("/posts", &PostsController{}, "Index")
	r.GET("/posts/:slug", &PostsController{}, "Show")

	// Authentication routes
	r.GET("/login", &AuthController{}, "LoginForm")
	r.POST("/login", &AuthController{}, "Login")
	r.GET("/register", &AuthController{}, "RegisterForm")
	r.POST("/register", &AuthController{}, "Register")
	r.POST("/logout", &AuthController{}, "Logout")

	// Protected routes
	r.Group(func(auth gor.Router) {
		auth.Use(RequireAuth())

		auth.GET("/posts/new", &PostsController{}, "New")
		auth.POST("/posts", &PostsController{}, "Create")
		auth.POST("/posts/:post_id/comments", &CommentsController{}, "Create")
	})

	// WebSocket endpoint
	r.GET("/cable", func(ctx gor.Context) error {
		return app.cable.HandleWebSocket(ctx)
	})
}

// Middleware
func RequireAuth() gor.MiddlewareFunc {
	return func(next gor.HandlerFunc) gor.HandlerFunc {
		return func(ctx gor.Context) error {
			if app.auth.CurrentUser(ctx) == nil {
				ctx.Set("return_to", ctx.Request().URL.Path)
				return ctx.Redirect("/login")
			}
			return next(ctx)
		}
	}
}

// Helpers
func slugify(s string) string {
	// Simple slugification
	return strings.ToLower(strings.ReplaceAll(s, " ", "-"))
}

func atoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}