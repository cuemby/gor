package main

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	gortest "github.com/cuemby/gor/internal/testing"
	"github.com/cuemby/gor/pkg/gor"
	_ "github.com/mattn/go-sqlite3"
)

// MockORM implements gor.ORM interface for testing
type MockORM struct {
	db *sql.DB
}

func (m *MockORM) Connect(ctx context.Context, config gor.DatabaseConfig) error {
	return nil
}

func (m *MockORM) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func (m *MockORM) DB() *sql.DB {
	return m.db
}

func (m *MockORM) Migrate(ctx context.Context) error {
	return nil
}

func (m *MockORM) Rollback(ctx context.Context, steps int) error {
	return nil
}

func (m *MockORM) MigrationStatus(ctx context.Context) ([]gor.Migration, error) {
	return nil, nil
}

func (m *MockORM) Register(models ...interface{}) error {
	return nil
}

func (m *MockORM) Table(name string) gor.Table {
	return nil
}

func (m *MockORM) Transaction(ctx context.Context, fn func(tx gor.Transaction) error) error {
	return nil
}

func (m *MockORM) Query(model interface{}) gor.QueryBuilder {
	return nil
}

func (m *MockORM) Find(model interface{}, id interface{}) error {
	return nil
}

func (m *MockORM) FindAll(models interface{}) error {
	return nil
}

func (m *MockORM) Create(model interface{}) error {
	return nil
}

func (m *MockORM) Update(model interface{}) error {
	return nil
}

func (m *MockORM) Delete(model interface{}) error {
	return nil
}

// MockRouter implements gor.Router interface for testing
type MockRouter struct {
	handlers map[string]http.HandlerFunc
}

func NewMockRouter() *MockRouter {
	return &MockRouter{
		handlers: make(map[string]http.HandlerFunc),
	}
}

func (m *MockRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.Method + ":" + r.URL.Path
	if handler, ok := m.handlers[key]; ok {
		handler(w, r)
	} else if r.URL.Path == "/articles" && r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<h1>Articles</h1>"))
	} else if r.URL.Path == "/articles/1" && r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<h1>Test Article</h1>"))
	} else if r.URL.Path == "/articles" && r.Method == http.MethodPost {
		w.WriteHeader(http.StatusCreated)
	} else if r.URL.Path == "/articles/1" && r.Method == http.MethodPut {
		w.WriteHeader(http.StatusOK)
	} else if r.URL.Path == "/articles/1" && r.Method == http.MethodDelete {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *MockRouter) Resources(name string, controller gor.Controller) gor.Router {
	return m
}

func (m *MockRouter) Resource(name string, controller gor.Controller) gor.Router {
	return m
}

func (m *MockRouter) GET(path string, handler gor.HandlerFunc) gor.Router {
	return m
}

func (m *MockRouter) POST(path string, handler gor.HandlerFunc) gor.Router {
	return m
}

func (m *MockRouter) PUT(path string, handler gor.HandlerFunc) gor.Router {
	return m
}

func (m *MockRouter) PATCH(path string, handler gor.HandlerFunc) gor.Router {
	return m
}

func (m *MockRouter) DELETE(path string, handler gor.HandlerFunc) gor.Router {
	return m
}

func (m *MockRouter) Namespace(prefix string, fn func(gor.Router)) gor.Router {
	return m
}

func (m *MockRouter) Group(middleware ...gor.MiddlewareFunc) gor.Router {
	return m
}

func (m *MockRouter) Use(middleware ...gor.MiddlewareFunc) gor.Router {
	return m
}

func (m *MockRouter) Named(name string) gor.Router {
	return m
}

// MockCable implements gor.Cable interface for testing
type MockCable struct{}

func (m *MockCable) HandleWebSocket(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (m *MockCable) HandleSSE(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (m *MockCable) Subscribe(ctx context.Context, connectionID, channel string, params map[string]interface{}) error {
	return nil
}

func (m *MockCable) Unsubscribe(ctx context.Context, connectionID, channel string) error {
	return nil
}

func (m *MockCable) UnsubscribeAll(ctx context.Context, connectionID string) error {
	return nil
}

func (m *MockCable) Broadcast(ctx context.Context, channel string, message interface{}) error {
	return nil
}

func (m *MockCable) BroadcastTo(ctx context.Context, connectionIDs []string, message interface{}) error {
	return nil
}

func (m *MockCable) BroadcastToUser(ctx context.Context, userID string, message interface{}) error {
	return nil
}

func (m *MockCable) ConnectionCount(ctx context.Context) (int, error) {
	return 0, nil
}

func (m *MockCable) ChannelConnections(ctx context.Context, channel string) ([]string, error) {
	return nil, nil
}

func (m *MockCable) UserConnections(ctx context.Context, userID string) ([]string, error) {
	return nil, nil
}

func (m *MockCable) Start(ctx context.Context) error {
	return nil
}

func (m *MockCable) Stop(ctx context.Context) error {
	return nil
}

func (m *MockCable) Stats(ctx context.Context) (gor.CableStats, error) {
	return gor.CableStats{}, nil
}

// MockQueueAdapter wraps MockQueue to implement gor.Queue interface
type MockQueueAdapter struct {
	*gortest.MockQueue
}

func (m *MockQueueAdapter) Enqueue(ctx context.Context, job gor.Job) error {
	m.MockQueue.Enqueue(job)
	return nil
}

func (m *MockQueueAdapter) EnqueueAt(ctx context.Context, job gor.Job, at time.Time) error {
	m.MockQueue.Enqueue(job)
	return nil
}

func (m *MockQueueAdapter) EnqueueIn(ctx context.Context, job gor.Job, delay time.Duration) error {
	m.MockQueue.Enqueue(job)
	return nil
}

func (m *MockQueueAdapter) Cancel(ctx context.Context, jobID string) error {
	return nil
}

func (m *MockQueueAdapter) Retry(ctx context.Context, jobID string) error {
	return nil
}

func (m *MockQueueAdapter) Delete(ctx context.Context, jobID string) error {
	return nil
}

func (m *MockQueueAdapter) Start(ctx context.Context) error {
	return nil
}

func (m *MockQueueAdapter) Stop(ctx context.Context) error {
	return nil
}

func (m *MockQueueAdapter) Pause(ctx context.Context, queueName string) error {
	return nil
}

func (m *MockQueueAdapter) Resume(ctx context.Context, queueName string) error {
	return nil
}

func (m *MockQueueAdapter) Stats(ctx context.Context) (gor.QueueStats, error) {
	return gor.QueueStats{}, nil
}

func (m *MockQueueAdapter) JobStatus(ctx context.Context, jobID string) (gor.JobStatus, error) {
	return gor.JobPending, nil
}

func (m *MockQueueAdapter) ListJobs(ctx context.Context, opts gor.ListJobsOptions) ([]gor.JobInfo, error) {
	return nil, nil
}

func (m *MockQueueAdapter) RegisterWorker(name string, worker gor.Worker) error {
	return nil
}

func (m *MockQueueAdapter) UnregisterWorker(name string) error {
	return nil
}

func (m *MockQueueAdapter) Workers() map[string]gor.Worker {
	return nil
}

// MockCacheAdapter wraps MockCache to implement gor.Cache interface
type MockCacheAdapter struct {
	*gortest.MockCache
}

func (m *MockCacheAdapter) Get(ctx context.Context, key string) (interface{}, error) {
	return m.MockCache.Get(key)
}

func (m *MockCacheAdapter) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	m.MockCache.Set(key, value, ttl)
	return nil
}

func (m *MockCacheAdapter) Delete(ctx context.Context, key string) error {
	m.MockCache.Delete(key)
	return nil
}

func (m *MockCacheAdapter) Exists(ctx context.Context, key string) (bool, error) {
	_, err := m.MockCache.Get(key)
	return err == nil, nil
}

func (m *MockCacheAdapter) GetMulti(ctx context.Context, keys []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for _, key := range keys {
		if val, err := m.MockCache.Get(key); err == nil {
			result[key] = val
		}
	}
	return result, nil
}

func (m *MockCacheAdapter) SetMulti(ctx context.Context, items map[string]gor.CacheItem) error {
	for key, item := range items {
		m.MockCache.Set(key, item.Value, item.TTL)
	}
	return nil
}

func (m *MockCacheAdapter) DeleteMulti(ctx context.Context, keys []string) error {
	for _, key := range keys {
		m.MockCache.Delete(key)
	}
	return nil
}

func (m *MockCacheAdapter) DeletePattern(ctx context.Context, pattern string) error {
	return nil
}

func (m *MockCacheAdapter) Keys(ctx context.Context, pattern string) ([]string, error) {
	return nil, nil
}

func (m *MockCacheAdapter) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	return 0, nil
}

func (m *MockCacheAdapter) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	return 0, nil
}

func (m *MockCacheAdapter) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
	if val, err := m.MockCache.Get(key); err == nil {
		return val, nil
	}
	val, err := fn()
	if err != nil {
		return nil, err
	}
	m.MockCache.Set(key, val, ttl)
	return val, nil
}

func (m *MockCacheAdapter) Touch(ctx context.Context, key string, ttl time.Duration) error {
	return nil
}

func (m *MockCacheAdapter) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

func (m *MockCacheAdapter) Clear(ctx context.Context) error {
	m.MockCache.Clear()
	return nil
}

func (m *MockCacheAdapter) Stats(ctx context.Context) (gor.CacheStats, error) {
	return gor.CacheStats{}, nil
}

func (m *MockCacheAdapter) Size(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *MockCacheAdapter) Namespace(prefix string) gor.Cache {
	return m
}

func (m *MockCacheAdapter) Tagged(tags ...string) gor.TaggedCache {
	return nil
}

// MockAuth implements a simple auth interface for testing
type MockAuth struct{}

func (m *MockAuth) Authenticate(username, password string) (interface{}, error) {
	return nil, nil
}

func (m *MockAuth) Authorize(user interface{}, permission string) bool {
	return true
}

// TestApplication is a minimal test application for testing
type TestApplication struct {
	orm    gor.ORM
	router gor.Router
	queue  gor.Queue
	cache  gor.Cache
	cable  gor.Cable
	auth   interface{}
	config gor.Config
}

func (a *TestApplication) Start(ctx context.Context) error {
	return nil
}

func (a *TestApplication) Stop(ctx context.Context) error {
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

func (a *TestApplication) Auth() interface{} {
	return a.auth
}

func (a *TestApplication) Config() gor.Config {
	return a.config
}

// MockConfig implements gor.Config interface for testing
type MockConfig struct{}

func (m *MockConfig) Environment() string {
	return "test"
}

func (m *MockConfig) Get(key string) interface{} {
	return nil
}

func (m *MockConfig) Set(key string, value interface{}) {}

func (m *MockConfig) Database() gor.DatabaseConfig {
	return gor.DatabaseConfig{
		Driver:   "sqlite3",
		Database: ":memory:",
	}
}

func (m *MockConfig) Server() gor.ServerConfig {
	return gor.ServerConfig{
		Host:         "localhost",
		Port:         8080,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// createTestApp creates a test application with all components initialized
func createTestApp() gor.Application {
	// Create in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}

	// Use mock implementations for testing
	// We need to wrap the testing mocks to implement the full interfaces
	app := &TestApplication{
		orm:    &MockORM{db: db},
		router: NewMockRouter(),
		queue:  &MockQueueAdapter{MockQueue: gortest.NewMockQueue()},
		cache:  &MockCacheAdapter{MockCache: gortest.NewMockCache()},
		cable:  &MockCable{},
		auth:   &MockAuth{},
		config: &MockConfig{},
	}

	return app
}