package main

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/cuemby/gor/internal/auth"
	"github.com/cuemby/gor/internal/cable"
	"github.com/cuemby/gor/internal/cache"
	"github.com/cuemby/gor/internal/orm"
	"github.com/cuemby/gor/internal/queue"
	"github.com/cuemby/gor/internal/router"
	"github.com/cuemby/gor/pkg/gor"
	_ "github.com/mattn/go-sqlite3"
)

// TestApplication is a minimal test application for testing
type TestApplication struct {
	orm    gor.ORM
	router gor.Router
	queue  gor.Queue
	cache  gor.Cache
	cable  gor.Cable
	auth   gor.Auth
	db     *sql.DB
}

func (a *TestApplication) Start(ctx context.Context) error {
	return nil
}

func (a *TestApplication) Router() gor.Router {
	return a.router
}

func (a *TestApplication) ORM() gor.ORM {
	return a.orm
}

func (a *TestApplication) Queue() gor.Queue {
	return a.queue
}

func (a *TestApplication) Cache() gor.Cache {
	return a.cache
}

func (a *TestApplication) Cable() gor.Cable {
	return a.cable
}

func (a *TestApplication) Auth() gor.Auth {
	return a.auth
}

// createTestApp creates a test application with all components initialized
func createTestApp() gor.Application {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}

	// Initialize ORM with test database
	testORM := orm.New(&orm.Config{
		Driver:   "sqlite3",
		Database: ":memory:",
	})

	// Initialize router
	testRouter := router.New()
	configureTestRoutes(testRouter)

	// Initialize queue (memory-based for testing)
	testQueue := queue.New(&queue.Config{
		Driver:  "memory",
		Workers: 1,
	})

	// Initialize cache (memory-based for testing)
	testCache := cache.New(&cache.Config{
		Driver: "memory",
		TTL:    300,
	})

	// Initialize cable (memory-based for testing)
	testCable := cable.New(&cable.Config{
		Driver: "memory",
	})

	// Initialize auth
	testAuth := auth.New(&auth.Config{
		SessionKey: "test-secret-key",
	})

	app := &TestApplication{
		orm:    testORM,
		router: testRouter,
		queue:  testQueue,
		cache:  testCache,
		cable:  testCable,
		auth:   testAuth,
		db:     db,
	}

	// Run migrations for test models
	if err := testORM.Migrate(); err != nil {
		panic(err)
	}

	return app
}

// configureTestRoutes sets up routes for testing
func configureTestRoutes(r gor.Router) {
	// Articles routes
	r.GET("/articles", &ArticlesController{}, "Index")
	r.GET("/articles/:id", &ArticlesController{}, "Show")
	r.POST("/articles", &ArticlesController{}, "Create")
	r.PUT("/articles/:id", &ArticlesController{}, "Update")
	r.DELETE("/articles/:id", &ArticlesController{}, "Destroy")
}

// ArticlesController is a test controller
type ArticlesController struct {
	gor.BaseController
}

func (c *ArticlesController) Index(ctx gor.Context) error {
	return ctx.HTML(http.StatusOK, "<h1>Articles</h1>")
}

func (c *ArticlesController) Show(ctx gor.Context) error {
	return ctx.HTML(http.StatusOK, "<h1>Test Article</h1>")
}

func (c *ArticlesController) Create(ctx gor.Context) error {
	var article map[string]interface{}
	if err := ctx.Bind(&article); err != nil {
		return ctx.JSON(http.StatusBadRequest, gor.H{
			"error": "Title is required",
		})
	}

	// Validate
	if title, ok := article["title"].(string); !ok || len(title) < 3 {
		return ctx.JSON(http.StatusBadRequest, gor.H{
			"error": "Title must be at least 3 characters",
		})
	}

	return ctx.JSON(http.StatusCreated, article)
}

func (c *ArticlesController) Update(ctx gor.Context) error {
	return ctx.JSON(http.StatusOK, gor.H{
		"message": "Updated",
	})
}

func (c *ArticlesController) Destroy(ctx gor.Context) error {
	return ctx.Status(http.StatusNoContent)
}