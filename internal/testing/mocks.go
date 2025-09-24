package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// MockHTTPClient provides a mock HTTP client for testing
type MockHTTPClient struct {
	responses map[string]*http.Response
	requests  []*http.Request
	mu        sync.Mutex
}

// NewMockHTTPClient creates a new mock HTTP client
func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		responses: make(map[string]*http.Response),
		requests:  make([]*http.Request, 0),
	}
}

// Do executes a mock HTTP request
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.requests = append(m.requests, req)

	key := fmt.Sprintf("%s:%s", req.Method, req.URL.String())
	if resp, ok := m.responses[key]; ok {
		return resp, nil
	}

	// Return a default 404 response
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
	}, nil
}

// SetResponse sets a mock response for a specific request
func (m *MockHTTPClient) SetResponse(method, url string, statusCode int, body interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var bodyReader io.ReadCloser
	switch v := body.(type) {
	case string:
		bodyReader = io.NopCloser(bytes.NewBufferString(v))
	case []byte:
		bodyReader = io.NopCloser(bytes.NewBuffer(v))
	default:
		data, _ := json.Marshal(v)
		bodyReader = io.NopCloser(bytes.NewBuffer(data))
	}

	key := fmt.Sprintf("%s:%s", method, url)
	m.responses[key] = &http.Response{
		StatusCode: statusCode,
		Body:       bodyReader,
		Header:     make(http.Header),
	}
}

// GetRequests returns all captured requests
func (m *MockHTTPClient) GetRequests() []*http.Request {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requests
}

// MockCache provides a mock cache for testing
type MockCache struct {
	data map[string]interface{}
	ttls map[string]time.Time
	mu   sync.RWMutex
}

// NewMockCache creates a new mock cache
func NewMockCache() *MockCache {
	return &MockCache{
		data: make(map[string]interface{}),
		ttls: make(map[string]time.Time),
	}
}

// Get retrieves a value from the mock cache
func (m *MockCache) Get(key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if expiry, ok := m.ttls[key]; ok && time.Now().After(expiry) {
		delete(m.data, key)
		delete(m.ttls, key)
		return nil, fmt.Errorf("key not found")
	}

	if val, ok := m.data[key]; ok {
		return val, nil
	}
	return nil, fmt.Errorf("key not found")
}

// Set stores a value in the mock cache
func (m *MockCache) Set(key string, value interface{}, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	if ttl > 0 {
		m.ttls[key] = time.Now().Add(ttl)
	}
	return nil
}

// Delete removes a value from the mock cache
func (m *MockCache) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	delete(m.ttls, key)
	return nil
}

// Clear clears all values from the mock cache
func (m *MockCache) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string]interface{})
	m.ttls = make(map[string]time.Time)
	return nil
}

// MockQueue provides a mock queue for testing
type MockQueue struct {
	jobs      []interface{}
	processed []interface{}
	failures  []interface{}
	mu        sync.Mutex
}

// NewMockQueue creates a new mock queue
func NewMockQueue() *MockQueue {
	return &MockQueue{
		jobs:      make([]interface{}, 0),
		processed: make([]interface{}, 0),
		failures:  make([]interface{}, 0),
	}
}

// Enqueue adds a job to the mock queue
func (m *MockQueue) Enqueue(job interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.jobs = append(m.jobs, job)
	return nil
}

// Process simulates processing jobs
func (m *MockQueue) Process() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.processed = append(m.processed, m.jobs...)
	m.jobs = make([]interface{}, 0)
	return nil
}

// GetJobs returns all pending jobs
func (m *MockQueue) GetJobs() []interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.jobs
}

// GetProcessed returns all processed jobs
func (m *MockQueue) GetProcessed() []interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.processed
}

// MockMailer provides a mock mailer for testing
type MockMailer struct {
	sentEmails []Email
	mu         sync.Mutex
}

// Email represents a mock email
type Email struct {
	To      []string
	From    string
	Subject string
	Body    string
	HTML    bool
}

// NewMockMailer creates a new mock mailer
func NewMockMailer() *MockMailer {
	return &MockMailer{
		sentEmails: make([]Email, 0),
	}
}

// Send sends a mock email
func (m *MockMailer) Send(to []string, from, subject, body string, html bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	email := Email{
		To:      to,
		From:    from,
		Subject: subject,
		Body:    body,
		HTML:    html,
	}

	m.sentEmails = append(m.sentEmails, email)
	return nil
}

// GetSentEmails returns all sent emails
func (m *MockMailer) GetSentEmails() []Email {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sentEmails
}

// Clear clears all sent emails
func (m *MockMailer) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentEmails = make([]Email, 0)
}

// MockDatabase provides a mock database for testing
type MockDatabase struct {
	tables  map[string][]map[string]interface{}
	queries []string
	mu      sync.RWMutex
}

// NewMockDatabase creates a new mock database
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		tables:  make(map[string][]map[string]interface{}),
		queries: make([]string, 0),
	}
}

// Query simulates a database query
func (m *MockDatabase) Query(query string, args ...interface{}) ([]map[string]interface{}, error) {
	m.mu.Lock()
	m.queries = append(m.queries, query)
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Simple mock: return all data from the first mentioned table
	for tableName, data := range m.tables {
		if bytes.Contains([]byte(query), []byte(tableName)) {
			return data, nil
		}
	}

	return []map[string]interface{}{}, nil
}

// Insert simulates inserting data
func (m *MockDatabase) Insert(table string, data map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tables[table] == nil {
		m.tables[table] = make([]map[string]interface{}, 0)
	}

	m.tables[table] = append(m.tables[table], data)
	return nil
}

// GetQueries returns all executed queries
func (m *MockDatabase) GetQueries() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.queries
}

// Clear clears all data and queries
func (m *MockDatabase) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tables = make(map[string][]map[string]interface{})
	m.queries = make([]string, 0)
}
