package testing

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ar4mirez/gor/pkg/gor"
)

// TestCase provides a base structure for test cases
type TestCase struct {
	t          *testing.T
	app        gor.Application
	db         *sql.DB
	dbPath     string // Track database path for cleanup
	fixtures   map[string]interface{}
	assertions *Assertions
}

// NewTestCase creates a new test case
func NewTestCase(t *testing.T, app gor.Application) *TestCase {
	tc := &TestCase{
		t:          t,
		app:        app,
		fixtures:   make(map[string]interface{}),
		assertions: NewAssertions(t),
	}

	// Set up test database if ORM is available
	if orm := app.ORM(); orm != nil {
		tc.setupTestDatabase()
	}

	return tc
}

// setupTestDatabase sets up a test database
func (tc *TestCase) setupTestDatabase() {
	// Use test database configuration
	testDBPath := "test.db"
	if env := os.Getenv("GO_ENV"); env == "test" {
		testDBPath = fmt.Sprintf("test_%d.db", time.Now().UnixNano())
	}

	db, err := sql.Open("sqlite3", testDBPath)
	if err != nil {
		tc.t.Fatalf("Failed to open test database: %v", err)
	}

	tc.db = db
	tc.dbPath = testDBPath

	// Run migrations
	if orm := tc.app.ORM(); orm != nil {
		if err := orm.Migrate(); err != nil {
			tc.t.Fatalf("Failed to run migrations: %v", err)
		}
	}
}

// TearDown cleans up after tests
func (tc *TestCase) TearDown() {
	if tc.db != nil {
		tc.db.Close()
		// Remove test database file
		if tc.dbPath != "" && strings.HasPrefix(tc.dbPath, "test_") {
			os.Remove(tc.dbPath)
		}
	}
}

// LoadFixtures loads test fixtures from files
func (tc *TestCase) LoadFixtures(names ...string) {
	for _, name := range names {
		fixtureFile := filepath.Join("test", "fixtures", name+".json")
		data, err := os.ReadFile(fixtureFile)
		if err != nil {
			tc.t.Fatalf("Failed to load fixture %s: %v", name, err)
		}

		var fixture interface{}
		if err := json.Unmarshal(data, &fixture); err != nil {
			tc.t.Fatalf("Failed to parse fixture %s: %v", name, err)
		}

		tc.fixtures[name] = fixture

		// Insert fixture data into database if it's a model fixture
		if orm := tc.app.ORM(); orm != nil {
			tc.insertFixtureData(name, fixture)
		}
	}
}

// insertFixtureData inserts fixture data into the database
func (tc *TestCase) insertFixtureData(name string, data interface{}) {
	orm := tc.app.ORM()
	if orm == nil {
		return
	}

	// Handle different fixture formats
	switch v := data.(type) {
	case []interface{}:
		for _, item := range v {
			if err := orm.Create(item); err != nil {
				tc.t.Errorf("Failed to insert fixture %s: %v", name, err)
			}
		}
	case map[string]interface{}:
		if err := orm.Create(v); err != nil {
			tc.t.Errorf("Failed to insert fixture %s: %v", name, err)
		}
	}
}

// Assert returns the assertions helper
func (tc *TestCase) Assert() *Assertions {
	return tc.assertions
}

// HTTPRequest makes an HTTP request to the test server
func (tc *TestCase) HTTPRequest(method, path string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	rec := httptest.NewRecorder()

	if router := tc.app.Router(); router != nil {
		router.ServeHTTP(rec, req)
	}

	return rec
}

// GET makes a GET request
func (tc *TestCase) GET(path string) *httptest.ResponseRecorder {
	return tc.HTTPRequest(http.MethodGet, path, nil)
}

// POST makes a POST request
func (tc *TestCase) POST(path string, body interface{}) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reader = bytes.NewReader(data)
	}
	return tc.HTTPRequest(http.MethodPost, path, reader)
}

// PUT makes a PUT request
func (tc *TestCase) PUT(path string, body interface{}) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reader = bytes.NewReader(data)
	}
	return tc.HTTPRequest(http.MethodPut, path, reader)
}

// DELETE makes a DELETE request
func (tc *TestCase) DELETE(path string) *httptest.ResponseRecorder {
	return tc.HTTPRequest(http.MethodDelete, path, nil)
}

// RunInTransaction runs a test in a database transaction that gets rolled back
func (tc *TestCase) RunInTransaction(fn func()) {
	if tc.db == nil {
		fn()
		return
	}

	tx, err := tc.db.Begin()
	if err != nil {
		tc.t.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Replace the database connection temporarily
	origDB := tc.db
	tc.db = tx
	defer func() { tc.db = origDB }()

	fn()
}

// Benchmark runs a benchmark
func (tc *TestCase) Benchmark(name string, fn func(*testing.B)) {
	tc.t.Run("Benchmark_"+name, func(t *testing.T) {
		if b, ok := t.(*testing.B); ok {
			fn(b)
		}
	})
}