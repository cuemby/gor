package testing

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand" // #nosec G404 - This is for test data generation, not cryptographic use
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TestHelpers provides utility functions for testing
type TestHelpers struct {
	tempDirs []string
	servers  []*httptest.Server
}

// NewTestHelpers creates a new test helpers instance
func NewTestHelpers() *TestHelpers {
	return &TestHelpers{
		tempDirs: make([]string, 0),
		servers:  make([]*httptest.Server, 0),
	}
}

// CreateTempDir creates a temporary directory
func (th *TestHelpers) CreateTempDir(prefix string) (string, error) {
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", err
	}

	th.tempDirs = append(th.tempDirs, dir)
	return dir, nil
}

// CreateTempFile creates a temporary file
func (th *TestHelpers) CreateTempFile(dir, pattern string, content []byte) (string, error) {
	if dir == "" {
		var err error
		dir, err = th.CreateTempDir("test")
		if err != nil {
			return "", err
		}
	}

	file, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if len(content) > 0 {
		if _, err := file.Write(content); err != nil {
			return "", err
		}
	}

	return file.Name(), nil
}

// StartTestServer starts a test HTTP server
func (th *TestHelpers) StartTestServer(handler http.Handler) *httptest.Server {
	server := httptest.NewServer(handler)
	th.servers = append(th.servers, server)
	return server
}

// Cleanup cleans up all created resources
func (th *TestHelpers) Cleanup() {
	// Close all test servers
	for _, server := range th.servers {
		server.Close()
	}

	// Remove all temporary directories
	for _, dir := range th.tempDirs {
		os.RemoveAll(dir)
	}
}

// LoadJSON loads JSON data from a file
func LoadJSON(filename string, v interface{}) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// SaveJSON saves data as JSON to a file
func SaveJSON(filename string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// RandomString generates a random string
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	// rand.Seed is deprecated as of Go 1.20; automatic seeding is now done by default

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))] // #nosec G404 - This is for test data, not cryptographic use
	}
	return string(b)
}

// RandomEmail generates a random email
func RandomEmail() string {
	return fmt.Sprintf("%s@test.com", RandomString(10))
}

// RandomInt generates a random integer
func RandomInt(min, max int) int {
	// rand.Seed is deprecated as of Go 1.20; automatic seeding is now done by default
	return rand.Intn(max-min+1) + min // #nosec G404 - This is for test data, not cryptographic use
}

// CreateTestDatabase creates a test database
func CreateTestDatabase(name string) (*sql.DB, error) {
	dbPath := fmt.Sprintf("%s_test.db", name)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Run initial setup
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// DropTestDatabase drops a test database
func DropTestDatabase(name string) error {
	dbPath := fmt.Sprintf("%s_test.db", name)
	return os.Remove(dbPath)
}

// CompareJSON compares two JSON strings
func CompareJSON(expected, actual string) (bool, error) {
	var expectedData, actualData interface{}

	if err := json.Unmarshal([]byte(expected), &expectedData); err != nil {
		return false, fmt.Errorf("failed to parse expected JSON: %w", err)
	}

	if err := json.Unmarshal([]byte(actual), &actualData); err != nil {
		return false, fmt.Errorf("failed to parse actual JSON: %w", err)
	}

	expectedJSON, _ := json.Marshal(expectedData)
	actualJSON, _ := json.Marshal(actualData)

	return string(expectedJSON) == string(actualJSON), nil
}

// ParseFormValues parses form values from a request body
func ParseFormValues(body io.Reader) (map[string]string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	values := make(map[string]string)
	pairs := strings.Split(string(data), "&")

	for _, pair := range pairs {
		parts := strings.Split(pair, "=")
		if len(parts) == 2 {
			values[parts[0]] = parts[1]
		}
	}

	return values, nil
}

// WaitFor waits for a condition to be true
func WaitFor(condition func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}

	return false
}

// CaptureOutput captures stdout and stderr
func CaptureOutput(fn func()) (string, string, error) {
	// Create temp files for output
	stdoutFile, err := os.CreateTemp("", "stdout")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(stdoutFile.Name())

	stderrFile, err := os.CreateTemp("", "stderr")
	if err != nil {
		return "", "", err
	}
	defer os.Remove(stderrFile.Name())

	// Save original stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Redirect stdout and stderr
	os.Stdout = stdoutFile
	os.Stderr = stderrFile

	// Run the function
	fn()

	// Read captured output
	_, _ = stdoutFile.Seek(0, 0)
	stdoutData, err := io.ReadAll(stdoutFile)
	if err != nil {
		return "", "", err
	}

	_, _ = stderrFile.Seek(0, 0)
	stderrData, err := io.ReadAll(stderrFile)
	if err != nil {
		return "", "", err
	}

	return string(stdoutData), string(stderrData), nil
}

// SetupTestEnvironment sets up the test environment
func SetupTestEnvironment() {
	os.Setenv("GO_ENV", "test")
	os.Setenv("LOG_LEVEL", "debug")
}

// CreateTestConfig creates a test configuration file
func CreateTestConfig(dir string, config map[string]interface{}) (string, error) {
	configPath := filepath.Join(dir, "config.test.json")
	return configPath, SaveJSON(configPath, config)
}
