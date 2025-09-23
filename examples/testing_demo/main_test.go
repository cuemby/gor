package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	gortest "github.com/cuemby/gor/internal/testing"
)

// Example model for testing
type Article struct {
	ID          int       `db:"id"`
	Title       string    `db:"title"`
	Content     string    `db:"content"`
	AuthorID    int       `db:"author_id"`
	PublishedAt time.Time `db:"published_at"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// TestArticleController tests the article controller
func TestArticleController(t *testing.T) {
	// Create test app
	app := createTestApp()
	tc := gortest.NewTestCase(t, app)
	defer tc.TearDown()

	// Load fixtures
	tc.LoadFixtures("articles", "users")

	t.Run("GET /articles", func(t *testing.T) {
		resp := tc.GET("/articles")
		tc.Assert().HTTPStatusCode(http.StatusOK, resp.Code)
		tc.Assert().Contains(resp.Body.String(), "Articles")
	})

	t.Run("GET /articles/:id", func(t *testing.T) {
		resp := tc.GET("/articles/1")
		tc.Assert().HTTPStatusCode(http.StatusOK, resp.Code)
		tc.Assert().Contains(resp.Body.String(), "Test Article")
	})

	t.Run("POST /articles", func(t *testing.T) {
		article := map[string]interface{}{
			"title":   "New Article",
			"content": "Article content",
		}

		resp := tc.POST("/articles", article)
		tc.Assert().HTTPStatusCode(http.StatusCreated, resp.Code)
	})

	t.Run("PUT /articles/:id", func(t *testing.T) {
		update := map[string]interface{}{
			"title": "Updated Title",
		}

		resp := tc.PUT("/articles/1", update)
		tc.Assert().HTTPStatusCode(http.StatusOK, resp.Code)
	})

	t.Run("DELETE /articles/:id", func(t *testing.T) {
		resp := tc.DELETE("/articles/1")
		tc.Assert().HTTPStatusCode(http.StatusNoContent, resp.Code)
	})
}

// TestArticleValidation tests article validation
func TestArticleValidation(t *testing.T) {
	app := createTestApp()
	tc := gortest.NewTestCase(t, app)
	defer tc.TearDown()

	t.Run("requires title", func(t *testing.T) {
		article := map[string]interface{}{
			"content": "Content without title",
		}

		resp := tc.POST("/articles", article)
		tc.Assert().HTTPStatusCode(http.StatusBadRequest, resp.Code)
		tc.Assert().Contains(resp.Body.String(), "Title is required")
	})

	t.Run("title length validation", func(t *testing.T) {
		article := map[string]interface{}{
			"title":   "ab", // Too short
			"content": "Valid content",
		}

		resp := tc.POST("/articles", article)
		tc.Assert().HTTPStatusCode(http.StatusBadRequest, resp.Code)
		tc.Assert().Contains(resp.Body.String(), "Title must be at least 3 characters")
	})
}

// TestArticleFactories tests using factories
func TestArticleFactories(t *testing.T) {
	factory := gortest.DefaultFactories()

	// Define article factory
	factory.Define("article", Article{}, map[string]gortest.AttributeFunc{
		"ID": gortest.FixedValue(1),
		"Title": func(f *gortest.Factory) interface{} {
			return fmt.Sprintf("Article %d", time.Now().UnixNano())
		},
		"Content": func(f *gortest.Factory) interface{} {
			return "This is article content"
		},
		"AuthorID":    gortest.FixedValue(1),
		"PublishedAt": gortest.CurrentTime(),
		"CreatedAt":   gortest.CurrentTime(),
		"UpdatedAt":   gortest.CurrentTime(),
	})

	t.Run("build single article", func(t *testing.T) {
		article, err := factory.Build("article")
		if err != nil {
			t.Fatal(err)
		}

		a := article.(*Article)
		if a.Title == "" {
			t.Error("Article title should not be empty")
		}
	})

	t.Run("build multiple articles", func(t *testing.T) {
		articles, err := factory.BuildList("article", 5)
		if err != nil {
			t.Fatal(err)
		}

		if len(articles) != 5 {
			t.Errorf("Expected 5 articles, got %d", len(articles))
		}
	})

	t.Run("override attributes", func(t *testing.T) {
		article, err := factory.Build("article", map[string]interface{}{
			"Title": "Custom Title",
		})
		if err != nil {
			t.Fatal(err)
		}

		a := article.(*Article)
		if a.Title != "Custom Title" {
			t.Errorf("Expected custom title, got %s", a.Title)
		}
	})
}

// TestAssertions tests the assertion helpers
func TestAssertions(t *testing.T) {
	assert := gortest.NewAssertions(t)

	t.Run("equality assertions", func(t *testing.T) {
		assert.Equal("hello", "hello")
		assert.NotEqual("hello", "world")
	})

	t.Run("nil assertions", func(t *testing.T) {
		var ptr *string
		assert.Nil(ptr)

		str := "test"
		assert.NotNil(&str)
	})

	t.Run("boolean assertions", func(t *testing.T) {
		assert.True(true)
		assert.False(false)
	})

	t.Run("string assertions", func(t *testing.T) {
		assert.Contains("hello world", "world")
		assert.NotContains("hello world", "foo")
	})

	t.Run("collection assertions", func(t *testing.T) {
		assert.Empty([]string{})
		assert.NotEmpty([]string{"item"})
		assert.Len([]int{1, 2, 3}, 3)
	})

	t.Run("error assertions", func(t *testing.T) {
		var err error
		assert.NoError(err)

		err = fmt.Errorf("test error")
		assert.Error(err)
	})

	t.Run("panic assertions", func(t *testing.T) {
		assert.Panics(func() {
			panic("test panic")
		})

		assert.NotPanics(func() {
			// Normal execution
		})
	})
}

// TestMocks tests mock objects
func TestMocks(t *testing.T) {
	t.Run("mock HTTP client", func(t *testing.T) {
		client := gortest.NewMockHTTPClient()
		client.SetResponse("GET", "https://api.example.com/data", 200, map[string]interface{}{
			"message": "success",
		})

		req, _ := http.NewRequest("GET", "https://api.example.com/data", nil)
		resp, err := client.Do(req)

		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		requests := client.GetRequests()
		if len(requests) != 1 {
			t.Errorf("Expected 1 request, got %d", len(requests))
		}
	})

	t.Run("mock cache", func(t *testing.T) {
		cache := gortest.NewMockCache()

		// Set and get
		cache.Set("key1", "value1", 0)
		val, err := cache.Get("key1")
		if err != nil {
			t.Fatal(err)
		}
		if val != "value1" {
			t.Errorf("Expected value1, got %v", val)
		}

		// Test TTL
		cache.Set("key2", "value2", 100*time.Millisecond)
		time.Sleep(150 * time.Millisecond)
		_, err = cache.Get("key2")
		if err == nil {
			t.Error("Expected key to be expired")
		}
	})

	t.Run("mock queue", func(t *testing.T) {
		queue := gortest.NewMockQueue()

		// Enqueue jobs
		queue.Enqueue("job1")
		queue.Enqueue("job2")

		jobs := queue.GetJobs()
		if len(jobs) != 2 {
			t.Errorf("Expected 2 jobs, got %d", len(jobs))
		}

		// Process jobs
		queue.Process()
		processed := queue.GetProcessed()
		if len(processed) != 2 {
			t.Errorf("Expected 2 processed jobs, got %d", len(processed))
		}
	})

	t.Run("mock mailer", func(t *testing.T) {
		mailer := gortest.NewMockMailer()

		// Send email
		mailer.Send([]string{"user@example.com"}, "noreply@example.com", "Test Subject", "Test Body", false)

		emails := mailer.GetSentEmails()
		if len(emails) != 1 {
			t.Errorf("Expected 1 email, got %d", len(emails))
		}

		email := emails[0]
		if email.Subject != "Test Subject" {
			t.Errorf("Expected subject 'Test Subject', got %s", email.Subject)
		}
	})
}

// TestHelpers tests test helper functions
func TestHelpers(t *testing.T) {
	helpers := gortest.NewTestHelpers()
	defer helpers.Cleanup()

	t.Run("temporary files", func(t *testing.T) {
		// Create temp directory
		dir, err := helpers.CreateTempDir("test")
		if err != nil {
			t.Fatal(err)
		}

		// Create temp file
		file, err := helpers.CreateTempFile(dir, "test*.txt", []byte("test content"))
		if err != nil {
			t.Fatal(err)
		}

		// Verify file exists
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Error("Temp file should exist")
		}
	})

	t.Run("test server", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test response"))
		})

		server := helpers.StartTestServer(handler)
		resp, err := http.Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("random generators", func(t *testing.T) {
		// Random string
		str := gortest.RandomString(10)
		if len(str) != 10 {
			t.Errorf("Expected string of length 10, got %d", len(str))
		}

		// Random email
		email := gortest.RandomEmail()
		if !strings.Contains(email, "@") {
			t.Errorf("Expected valid email, got %s", email)
		}

		// Random int
		num := gortest.RandomInt(1, 10)
		if num < 1 || num > 10 {
			t.Errorf("Expected number between 1 and 10, got %d", num)
		}
	})

	t.Run("wait for condition", func(t *testing.T) {
		counter := 0
		go func() {
			time.Sleep(100 * time.Millisecond)
			counter = 1
		}()

		success := gortest.WaitFor(func() bool {
			return counter == 1
		}, 500*time.Millisecond)

		if !success {
			t.Error("Condition should have been met")
		}
	})
}

// BenchmarkExample demonstrates benchmarking
func BenchmarkExample(b *testing.B) {
	factory := gortest.DefaultFactories()

	b.Run("BuildArticle", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			factory.Build("user")
		}
	})

	b.Run("RandomString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			gortest.RandomString(20)
		}
	})
}

// TestSuite demonstrates test suite usage
func TestArticleSuite(t *testing.T) {
	suite := gortest.NewTestSuite("ArticleSuite")

	// Setup
	suite.SetSetup(func() {
		// Global setup
		gortest.SetupTestEnvironment()
	})

	// Teardown
	suite.SetTeardown(func() {
		// Global cleanup
	})

	// Add test cases
	suite.AddTest("CreateArticle", func(tc *gortest.TestCase) {
		tc.LoadFixtures("users")

		resp := tc.POST("/articles", map[string]interface{}{
			"title":   "Test Article",
			"content": "Test Content",
		})

		tc.Assert().HTTPSuccess(resp.Code)
	})

	suite.AddTest("ListArticles", func(tc *gortest.TestCase) {
		tc.LoadFixtures("articles")

		resp := tc.GET("/articles")
		tc.Assert().HTTPSuccess(resp.Code)
	})

	// Run suite
	app := createTestApp()
	suite.Run(t, app)
}

// TestTransactions tests database transactions
func TestTransactions(t *testing.T) {
	app := createTestApp()
	tc := gortest.NewTestCase(t, app)
	defer tc.TearDown()

	t.Run("rollback on failure", func(t *testing.T) {
		tc.RunInTransaction(func() {
			// Create article in transaction
			resp := tc.POST("/articles", map[string]interface{}{
				"title":   "Transactional Article",
				"content": "This will be rolled back",
			})

			tc.Assert().HTTPSuccess(resp.Code)

			// Transaction will be rolled back
			// Article should not exist after this
		})

		// Verify article doesn't exist
		resp := tc.GET("/articles")
		tc.Assert().NotContains(resp.Body.String(), "Transactional Article")
	})
}

// createTestApp is now defined in test_app.go