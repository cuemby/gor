package main

import (
	"testing"
	"time"

	gortest "github.com/cuemby/gor/internal/testing"
)

// TestSimpleAssertions demonstrates basic assertion helpers
func TestSimpleAssertions(t *testing.T) {
	assert := gortest.NewAssertions(t)

	t.Run("equality", func(t *testing.T) {
		assert.Equal("hello", "hello")
		assert.NotEqual("hello", "world")
		assert.Equal(42, 42)
		assert.NotEqual(1, 2)
	})

	t.Run("nil checks", func(t *testing.T) {
		var ptr *string
		assert.Nil(ptr)

		str := "test"
		assert.NotNil(&str)
	})

	t.Run("boolean", func(t *testing.T) {
		assert.True(true)
		assert.False(false)
		assert.True(1+1 == 2)
	})

	t.Run("strings", func(t *testing.T) {
		assert.Contains("hello world", "world")
		assert.NotContains("hello world", "foo")
		assert.Contains("testing is fun", "test")
	})

	t.Run("collections", func(t *testing.T) {
		assert.Empty([]string{})
		assert.NotEmpty([]string{"item"})
		assert.Len([]int{1, 2, 3}, 3)
		assert.Len(map[string]int{"a": 1, "b": 2}, 2)
	})
}

// TestFactories demonstrates test data factories
func TestFactories(t *testing.T) {
	factory := gortest.DefaultFactories()

	// Define a simple user factory
	type User struct {
		ID    int
		Name  string
		Email string
	}

	factory.Define("user", User{}, map[string]gortest.AttributeFunc{
		"ID":    gortest.FixedValue(1),
		"Name":  gortest.FixedValue("Test User"),
		"Email": gortest.SequentialEmail("example.com"),
	})

	t.Run("build single user", func(t *testing.T) {
		user, err := factory.Build("user")
		if err != nil {
			t.Fatal(err)
		}

		u := user.(*User)
		if u.Name == "" {
			t.Error("User name should not be empty")
		}
		if u.Email == "" {
			t.Error("User email should not be empty")
		}
	})

	t.Run("build multiple users", func(t *testing.T) {
		users, err := factory.BuildList("user", 3)
		if err != nil {
			t.Fatal(err)
		}

		if len(users) != 3 {
			t.Errorf("Expected 3 users, got %d", len(users))
		}
	})

	t.Run("override attributes", func(t *testing.T) {
		user, err := factory.Build("user", map[string]interface{}{
			"Name": "John Doe",
		})
		if err != nil {
			t.Fatal(err)
		}

		u := user.(*User)
		if u.Name != "John Doe" {
			t.Errorf("Expected name 'John Doe', got %s", u.Name)
		}
	})
}

// TestHelpers demonstrates test helper functions
func TestSimpleHelpers(t *testing.T) {
	t.Run("random generators", func(t *testing.T) {
		// Random string
		str := gortest.RandomString(10)
		if len(str) != 10 {
			t.Errorf("Expected string of length 10, got %d", len(str))
		}

		// Random email
		email := gortest.RandomEmail()
		if email == "" {
			t.Error("Email should not be empty")
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
			// Simulate async operation
			time.Sleep(50 * time.Millisecond)
			counter = 1
		}()

		// Wait up to 500ms for condition
		success := gortest.WaitFor(func() bool {
			return counter == 1
		}, 500*time.Millisecond)

		if !success {
			t.Error("Condition should have been met")
		}
	})
}

// TestMocks demonstrates mock objects
func TestSimpleMocks(t *testing.T) {
	t.Run("mock cache", func(t *testing.T) {
		cache := gortest.NewMockCache()

		// Set and get values
		cache.Set("key1", "value1", 0)
		val, err := cache.Get("key1")
		if err != nil {
			t.Fatal(err)
		}
		if val != "value1" {
			t.Errorf("Expected 'value1', got %v", val)
		}

		// Delete value
		cache.Delete("key1")
		_, err = cache.Get("key1")
		if err == nil {
			t.Error("Expected error for deleted key")
		}
	})

	t.Run("mock queue", func(t *testing.T) {
		queue := gortest.NewMockQueue()

		// Enqueue jobs
		queue.Enqueue("job1")
		queue.Enqueue("job2")
		queue.Enqueue("job3")

		jobs := queue.GetJobs()
		if len(jobs) != 3 {
			t.Errorf("Expected 3 jobs, got %d", len(jobs))
		}

		// Process jobs
		queue.Process()
		processed := queue.GetProcessed()
		if len(processed) != 3 {
			t.Errorf("Expected 3 processed jobs, got %d", len(processed))
		}

		// Queue should be empty now
		jobs = queue.GetJobs()
		if len(jobs) != 0 {
			t.Errorf("Expected 0 jobs after processing, got %d", len(jobs))
		}
	})

	t.Run("mock mailer", func(t *testing.T) {
		mailer := gortest.NewMockMailer()

		// Send emails
		mailer.Send([]string{"user1@example.com"}, "sender@example.com", "Subject 1", "Body 1", false)
		mailer.Send([]string{"user2@example.com"}, "sender@example.com", "Subject 2", "Body 2", true)

		emails := mailer.GetSentEmails()
		if len(emails) != 2 {
			t.Errorf("Expected 2 emails, got %d", len(emails))
		}

		// Check first email
		email := emails[0]
		if email.Subject != "Subject 1" {
			t.Errorf("Expected subject 'Subject 1', got %s", email.Subject)
		}
		if email.HTML {
			t.Error("First email should not be HTML")
		}

		// Clear sent emails
		mailer.Clear()
		if len(mailer.GetSentEmails()) != 0 {
			t.Error("Mailer should be empty after clear")
		}
	})
}

// BenchmarkRandomString demonstrates benchmarking
func BenchmarkRandomString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		gortest.RandomString(20)
	}
}

// BenchmarkFactoryBuild benchmarks factory building
func BenchmarkFactoryBuild(b *testing.B) {
	factory := gortest.DefaultFactories()

	type Item struct {
		ID   int
		Name string
	}

	factory.Define("item", Item{}, map[string]gortest.AttributeFunc{
		"ID":   gortest.SequentialID("item"),
		"Name": gortest.RandomName(),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factory.Build("item")
	}
}