# Gor Testing Guide

Comprehensive guide to testing Gor applications.

## Testing Philosophy

Gor embraces test-driven development (TDD) and provides built-in testing utilities to make testing as painless as possible. Our testing philosophy:

1. **Test at multiple levels**: Unit, integration, and end-to-end tests
2. **Fast feedback loops**: Tests should run quickly
3. **Isolated tests**: Each test should be independent
4. **Readable tests**: Tests serve as documentation
5. **High coverage**: Aim for 80%+ code coverage

## Test Organization

```
test/
├── controllers/     # Controller tests
├── models/         # Model tests
├── jobs/           # Background job tests
├── mailers/        # Mailer tests
├── channels/       # Cable channel tests
├── integration/    # Integration tests
├── fixtures/       # Test data fixtures
├── helpers/        # Test helper utilities
└── test_helper.go  # Test configuration
```

## Running Tests

### Basic Commands

```bash
# Run all tests
gor test

# Run specific test file
gor test test/models/user_test.go

# Run tests matching pattern
gor test --pattern User

# Run with coverage
gor test --coverage

# Run with verbose output
gor test --verbose

# Stop on first failure
gor test --failfast
```

### Using Go Test Directly

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Unit Testing

### Testing Models

```go
package models_test

import (
    "testing"
    "github.com/cuemby/gor/pkg/gor"
    "myapp/app/models"
)

func TestUser_Validation(t *testing.T) {
    user := &models.User{
        Name:  "",
        Email: "invalid",
    }

    err := user.Validate()
    if err == nil {
        t.Error("Expected validation error")
    }

    validationErr, ok := err.(*gor.ValidationError)
    if !ok {
        t.Error("Expected ValidationError type")
    }

    if validationErr.Field != "email" {
        t.Errorf("Expected email error, got %s", validationErr.Field)
    }
}

func TestUser_Save(t *testing.T) {
    // Setup test database
    db := gor.TestDB(t)
    defer db.Close()

    user := &models.User{
        Name:  "John Doe",
        Email: "john@example.com",
    }

    err := db.Save(user)
    if err != nil {
        t.Fatalf("Failed to save user: %v", err)
    }

    if user.ID == 0 {
        t.Error("Expected user ID to be set")
    }
}

func TestUser_Associations(t *testing.T) {
    db := gor.TestDB(t)
    defer db.Close()

    user := &models.User{Name: "Jane"}
    db.Save(user)

    post := &models.Post{
        Title:  "Test Post",
        UserID: user.ID,
    }
    db.Save(post)

    // Test loading association
    var loadedUser models.User
    db.Preload("Posts").Find(&loadedUser, user.ID)

    if len(loadedUser.Posts) != 1 {
        t.Errorf("Expected 1 post, got %d", len(loadedUser.Posts))
    }
}
```

### Testing Controllers

```go
package controllers_test

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/cuemby/gor/pkg/gor"
    "myapp/app/controllers"
)

func TestPostsController_Index(t *testing.T) {
    // Setup
    app := gor.NewTestApp()
    controller := &controllers.PostsController{}

    // Create test request
    req := httptest.NewRequest("GET", "/posts", nil)
    w := httptest.NewRecorder()

    ctx := app.NewContext(req, w)

    // Execute
    err := controller.Index(ctx)
    if err != nil {
        t.Fatalf("Index failed: %v", err)
    }

    // Assert
    if w.Code != http.StatusOK {
        t.Errorf("Expected status 200, got %d", w.Code)
    }

    body := w.Body.String()
    if !strings.Contains(body, "Posts") {
        t.Error("Response should contain 'Posts'")
    }
}

func TestPostsController_Create(t *testing.T) {
    app := gor.NewTestApp()
    controller := &controllers.PostsController{}

    // Prepare form data
    form := url.Values{}
    form.Add("title", "Test Post")
    form.Add("body", "Test content")

    req := httptest.NewRequest("POST", "/posts", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    w := httptest.NewRecorder()

    ctx := app.NewContext(req, w)

    // Execute
    err := controller.Create(ctx)
    if err != nil {
        t.Fatalf("Create failed: %v", err)
    }

    // Check redirect
    if w.Code != http.StatusSeeOther {
        t.Errorf("Expected redirect, got %d", w.Code)
    }

    // Verify post was created
    var post models.Post
    app.DB().Where("title = ?", "Test Post").First(&post)

    if post.ID == 0 {
        t.Error("Post was not created")
    }
}

func TestPostsController_Authentication(t *testing.T) {
    app := gor.NewTestApp()
    controller := &controllers.PostsController{}

    // Request without authentication
    req := httptest.NewRequest("GET", "/posts/new", nil)
    w := httptest.NewRecorder()

    ctx := app.NewContext(req, w)

    err := controller.New(ctx)

    // Should redirect to login
    if w.Code != http.StatusUnauthorized {
        t.Errorf("Expected 401, got %d", w.Code)
    }
}
```

### Testing Background Jobs

```go
package jobs_test

import (
    "testing"
    "context"
    "github.com/cuemby/gor/pkg/gor"
    "myapp/app/jobs"
)

func TestEmailJob_Perform(t *testing.T) {
    job := &jobs.EmailJob{
        To:      "test@example.com",
        Subject: "Test Email",
        Body:    "Test content",
    }

    ctx := context.Background()
    err := job.Perform(ctx)

    if err != nil {
        t.Errorf("Job failed: %v", err)
    }

    // Verify email was sent (check test mail server)
    emails := gor.TestMailer.SentEmails()
    if len(emails) != 1 {
        t.Errorf("Expected 1 email, got %d", len(emails))
    }

    if emails[0].To != "test@example.com" {
        t.Errorf("Wrong recipient: %s", emails[0].To)
    }
}

func TestPaymentJob_Retry(t *testing.T) {
    job := &jobs.PaymentJob{
        OrderID: 123,
        Amount:  99.99,
    }

    // Simulate failure
    job.MockPaymentGateway = func() error {
        return errors.New("payment failed")
    }

    ctx := context.Background()
    err := job.Perform(ctx)

    if err == nil {
        t.Error("Expected job to fail")
    }

    // Check retry was scheduled
    queue := gor.TestQueue()
    retries := queue.RetryJobs()

    if len(retries) != 1 {
        t.Errorf("Expected 1 retry, got %d", len(retries))
    }
}
```

### Testing Real-time Features

```go
package channels_test

import (
    "testing"
    "github.com/cuemby/gor/pkg/gor"
    "myapp/app/channels"
)

func TestChatChannel_Subscribe(t *testing.T) {
    cable := gor.NewTestCable()
    channel := &channels.ChatChannel{}

    // Create test client
    client := cable.NewClient("test-user")

    // Subscribe to channel
    err := channel.Subscribe(client, "room:123")
    if err != nil {
        t.Fatalf("Subscribe failed: %v", err)
    }

    // Verify subscription
    if !client.IsSubscribed("room:123") {
        t.Error("Client should be subscribed")
    }
}

func TestChatChannel_Broadcast(t *testing.T) {
    cable := gor.NewTestCable()
    channel := &channels.ChatChannel{}

    // Create multiple clients
    client1 := cable.NewClient("user1")
    client2 := cable.NewClient("user2")

    channel.Subscribe(client1, "room:123")
    channel.Subscribe(client2, "room:123")

    // Broadcast message
    msg := map[string]interface{}{
        "text": "Hello",
        "user": "user1",
    }

    channel.Broadcast("room:123", msg)

    // Both clients should receive
    msg1 := <-client1.Messages
    msg2 := <-client2.Messages

    if msg1["text"] != "Hello" {
        t.Error("Client1 didn't receive message")
    }

    if msg2["text"] != "Hello" {
        t.Error("Client2 didn't receive message")
    }
}
```

## Integration Testing

Integration tests verify that different parts of your application work together correctly.

```go
package integration_test

import (
    "testing"
    "github.com/cuemby/gor/pkg/gor"
)

func TestUserRegistration(t *testing.T) {
    app := gor.NewTestApp()

    // Step 1: Register user
    req := httptest.NewRequest("POST", "/users", strings.NewReader(`{
        "name": "Jane Doe",
        "email": "jane@example.com",
        "password": "password123"
    }`))
    req.Header.Set("Content-Type", "application/json")

    w := httptest.NewRecorder()
    app.ServeHTTP(w, req)

    if w.Code != http.StatusCreated {
        t.Fatalf("Registration failed: %d", w.Code)
    }

    // Step 2: Verify email was sent
    emails := gor.TestMailer.SentEmails()
    if len(emails) != 1 {
        t.Error("Welcome email not sent")
    }

    // Step 3: Verify user can login
    loginReq := httptest.NewRequest("POST", "/login", strings.NewReader(`{
        "email": "jane@example.com",
        "password": "password123"
    }`))
    loginReq.Header.Set("Content-Type", "application/json")

    w2 := httptest.NewRecorder()
    app.ServeHTTP(w2, loginReq)

    if w2.Code != http.StatusOK {
        t.Error("Login failed")
    }

    // Step 4: Verify session/token
    cookie := w2.Header().Get("Set-Cookie")
    if cookie == "" {
        t.Error("Session not created")
    }
}
```

## Test Helpers

### Database Helpers

```go
// test/helpers/database.go
package helpers

import (
    "testing"
    "github.com/cuemby/gor/pkg/gor"
)

// SetupTestDB creates a clean test database
func SetupTestDB(t *testing.T) *gor.DB {
    db := gor.TestDB(t)

    // Run migrations
    db.AutoMigrate(&models.User{}, &models.Post{})

    // Cleanup after test
    t.Cleanup(func() {
        db.DropTable(&models.User{}, &models.Post{})
    })

    return db
}

// CreateTestUser creates a user for testing
func CreateTestUser(t *testing.T, db *gor.DB) *models.User {
    user := &models.User{
        Name:  "Test User",
        Email: "test@example.com",
    }

    if err := db.Save(user); err != nil {
        t.Fatalf("Failed to create test user: %v", err)
    }

    return user
}
```

### Request Helpers

```go
// test/helpers/requests.go
package helpers

import (
    "net/http/httptest"
    "testing"
)

// AuthenticatedRequest creates a request with authentication
func AuthenticatedRequest(t *testing.T, method, path string, user *models.User) *http.Request {
    req := httptest.NewRequest(method, path, nil)

    // Add authentication token
    token, err := gor.GenerateToken(user.ID)
    if err != nil {
        t.Fatalf("Failed to generate token: %v", err)
    }

    req.Header.Set("Authorization", "Bearer "+token)
    return req
}

// JSONRequest creates a JSON request
func JSONRequest(t *testing.T, method, path string, body interface{}) *http.Request {
    data, err := json.Marshal(body)
    if err != nil {
        t.Fatalf("Failed to marshal JSON: %v", err)
    }

    req := httptest.NewRequest(method, path, bytes.NewReader(data))
    req.Header.Set("Content-Type", "application/json")

    return req
}
```

## Test Fixtures

### Using Fixtures

```go
// test/fixtures/users.go
package fixtures

var Users = []map[string]interface{}{
    {
        "name":  "Alice",
        "email": "alice@example.com",
        "role":  "admin",
    },
    {
        "name":  "Bob",
        "email": "bob@example.com",
        "role":  "user",
    },
}

// test/models/user_test.go
func TestUser_Fixtures(t *testing.T) {
    db := helpers.SetupTestDB(t)

    // Load fixtures
    for _, data := range fixtures.Users {
        user := &models.User{
            Name:  data["name"].(string),
            Email: data["email"].(string),
            Role:  data["role"].(string),
        }
        db.Save(user)
    }

    // Test with fixtures
    var users []models.User
    db.Find(&users)

    if len(users) != 2 {
        t.Errorf("Expected 2 users, got %d", len(users))
    }
}
```

### Factory Pattern

```go
// test/factories/user_factory.go
package factories

import (
    "fmt"
    "github.com/cuemby/gor/pkg/gor"
    "myapp/app/models"
)

type UserFactory struct {
    counter int
}

func NewUserFactory() *UserFactory {
    return &UserFactory{}
}

func (f *UserFactory) Build(attrs ...map[string]interface{}) *models.User {
    f.counter++

    user := &models.User{
        Name:  fmt.Sprintf("User %d", f.counter),
        Email: fmt.Sprintf("user%d@example.com", f.counter),
    }

    // Apply custom attributes
    if len(attrs) > 0 {
        for key, value := range attrs[0] {
            switch key {
            case "name":
                user.Name = value.(string)
            case "email":
                user.Email = value.(string)
            }
        }
    }

    return user
}

func (f *UserFactory) Create(db *gor.DB, attrs ...map[string]interface{}) *models.User {
    user := f.Build(attrs...)
    db.Save(user)
    return user
}
```

## Test Coverage

### Measuring Coverage

```bash
# Generate coverage report
gor test --coverage

# Generate HTML report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# View coverage in terminal
go tool cover -func=coverage.out
```

### Coverage Goals

- **Models**: 90%+ coverage
- **Controllers**: 80%+ coverage
- **Jobs**: 80%+ coverage
- **Helpers**: 70%+ coverage
- **Overall**: 80%+ coverage

### Improving Coverage

1. **Identify gaps**: Use coverage reports to find untested code
2. **Test edge cases**: Don't just test the happy path
3. **Test error conditions**: Verify error handling
4. **Test validations**: Ensure data integrity
5. **Test associations**: Verify relationships

## Testing Best Practices

### 1. Test Structure

Follow the AAA pattern:

```go
func TestExample(t *testing.T) {
    // Arrange - Set up test data
    user := &User{Name: "Test"}

    // Act - Perform the action
    err := user.Save()

    // Assert - Verify the result
    if err != nil {
        t.Errorf("Save failed: %v", err)
    }
}
```

### 2. Table-Driven Tests

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid email", "test@example.com", false},
        {"invalid email", "invalid", true},
        {"empty email", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail(%s) error = %v, wantErr %v",
                    tt.input, err, tt.wantErr)
            }
        })
    }
}
```

### 3. Test Isolation

```go
func TestIsolated(t *testing.T) {
    // Use separate database transaction
    tx := db.Begin()
    defer tx.Rollback()

    // Test operations use transaction
    user := &User{Name: "Test"}
    tx.Save(user)

    // Changes are rolled back after test
}
```

### 4. Mock External Services

```go
type MockPaymentGateway struct {
    ChargeFunc func(amount float64) error
}

func (m *MockPaymentGateway) Charge(amount float64) error {
    if m.ChargeFunc != nil {
        return m.ChargeFunc(amount)
    }
    return nil
}

func TestPayment(t *testing.T) {
    gateway := &MockPaymentGateway{
        ChargeFunc: func(amount float64) error {
            if amount > 1000 {
                return errors.New("amount too large")
            }
            return nil
        },
    }

    err := ProcessPayment(gateway, 1500)
    if err == nil {
        t.Error("Expected error for large amount")
    }
}
```

### 5. Benchmark Tests

```go
func BenchmarkUserSave(b *testing.B) {
    db := setupBenchDB(b)
    user := &User{Name: "Bench User"}

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        user.ID = 0 // Reset ID
        db.Save(user)
    }
}
```

## Continuous Integration

### GitHub Actions Example

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:14
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'

    - name: Install dependencies
      run: go mod download

    - name: Run tests
      run: |
        go test -v -race -coverprofile=coverage.out ./...
        go tool cover -func=coverage.out

    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

## Debugging Tests

### Using Delve

```bash
# Debug a specific test
dlv test -- -test.run TestUserSave

# Set breakpoint and run
(dlv) break TestUserSave
(dlv) continue
```

### Verbose Output

```go
func TestDebug(t *testing.T) {
    t.Log("Starting test...")

    result := SomeFunction()
    t.Logf("Result: %+v", result)

    if testing.Verbose() {
        t.Logf("Detailed info: %#v", result)
    }
}
```

### Test Timeouts

```go
func TestWithTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    done := make(chan bool)

    go func() {
        // Long running operation
        time.Sleep(10 * time.Second)
        done <- true
    }()

    select {
    case <-done:
        t.Log("Operation completed")
    case <-ctx.Done():
        t.Fatal("Test timed out")
    }
}
```

## See Also

- [CLI Reference](./cli-reference.md) - Test command details
- [API Reference](./api.md) - Testing API documentation
- [Getting Started](./getting-started.md) - Basic setup
- [Examples](../examples/) - Sample test implementations