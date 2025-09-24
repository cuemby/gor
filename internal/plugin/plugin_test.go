package plugin

import (
	"context"
	"errors"
	"testing"

	"github.com/cuemby/gor/pkg/gor"
)

// MockPlugin implements the Plugin interface for testing
type MockPlugin struct {
	metadata   Metadata
	initCalled bool
	initErr    error
	startErr   error
	stopErr    error
	hooks      []Hook
	commands   []Command
	routes     []Route
	middleware []Middleware
}

func (m *MockPlugin) Metadata() Metadata {
	return m.metadata
}

func (m *MockPlugin) Initialize(app gor.Application) error {
	m.initCalled = true
	return m.initErr
}

func (m *MockPlugin) Start(ctx context.Context) error {
	return m.startErr
}

func (m *MockPlugin) Stop(ctx context.Context) error {
	return m.stopErr
}

func (m *MockPlugin) Hooks() []Hook {
	return m.hooks
}

func (m *MockPlugin) Commands() []Command {
	return m.commands
}

func (m *MockPlugin) Routes() []Route {
	return m.routes
}

func (m *MockPlugin) Middleware() []Middleware {
	return m.middleware
}

// Mock Application for testing
type MockApplication struct{}

func (m *MockApplication) Start(ctx context.Context) error { return nil }
func (m *MockApplication) Stop(ctx context.Context) error  { return nil }
func (m *MockApplication) Router() gor.Router              { return nil }
func (m *MockApplication) ORM() gor.ORM                    { return nil }
func (m *MockApplication) Queue() gor.Queue                { return nil }
func (m *MockApplication) Cache() gor.Cache                { return nil }
func (m *MockApplication) Cable() gor.Cable                { return nil }
func (m *MockApplication) Auth() interface{}               { return nil }
func (m *MockApplication) Config() gor.Config              { return nil }

func TestMetadata(t *testing.T) {
	metadata := Metadata{
		Name:         "test-plugin",
		Version:      "1.0.0",
		Description:  "Test plugin",
		Author:       "Test Author",
		License:      "MIT",
		Dependencies: []string{"dep1", "dep2"},
		Tags:         []string{"test", "example"},
	}

	if metadata.Name != "test-plugin" {
		t.Errorf("Expected name test-plugin, got %s", metadata.Name)
	}

	if metadata.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", metadata.Version)
	}

	if len(metadata.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(metadata.Dependencies))
	}

	if len(metadata.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(metadata.Tags))
	}
}

func TestNewManager(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.app != app {
		t.Error("App not set correctly")
	}

	if manager.plugins == nil {
		t.Error("Plugins map not initialized")
	}

	if manager.loaded == nil {
		t.Error("Loaded map not initialized")
	}

	if manager.hooks == nil {
		t.Error("Hooks map not initialized")
	}

	if manager.commands == nil {
		t.Error("Commands map not initialized")
	}

	if manager.routes == nil {
		t.Error("Routes slice not initialized")
	}

	if manager.middleware == nil {
		t.Error("Middleware slice not initialized")
	}
}

func TestManager_Register(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	plugin := &MockPlugin{
		metadata: Metadata{
			Name:        "test-plugin",
			Version:     "1.0.0",
			Description: "Test plugin",
		},
		hooks: []Hook{
			{Name: "before_request", Priority: 1, Handler: func(ctx context.Context, data interface{}) error { return nil }},
		},
		commands: []Command{
			{Name: "test", Description: "Test command"},
		},
		routes: []Route{
			{Method: "GET", Path: "/test"},
		},
		middleware: []Middleware{
			{Name: "test-middleware", Priority: 1},
		},
	}

	err := manager.Register(plugin)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	// Check plugin was registered
	if _, exists := manager.plugins["test-plugin"]; !exists {
		t.Error("Plugin was not registered")
	}

	// Check hooks were registered
	if len(manager.hooks["before_request"]) != 1 {
		t.Error("Hook was not registered")
	}

	// Check commands were registered
	if _, exists := manager.commands["test"]; !exists {
		t.Error("Command was not registered")
	}

	// Check routes were registered
	if len(manager.routes) != 1 {
		t.Error("Route was not registered")
	}

	// Check middleware was registered
	if len(manager.middleware) != 1 {
		t.Error("Middleware was not registered")
	}
}

func TestManager_RegisterDuplicate(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	plugin := &MockPlugin{
		metadata: Metadata{
			Name: "test-plugin",
		},
	}

	// Register once
	err := manager.Register(plugin)
	if err != nil {
		t.Errorf("First Register() error = %v", err)
	}

	// Try to register again
	err = manager.Register(plugin)
	if err == nil {
		t.Error("Expected error for duplicate plugin registration")
	}
}

func TestManager_RegisterWithDependencies(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	// Register dependency first
	dep := &MockPlugin{
		metadata: Metadata{
			Name: "dep-plugin",
		},
	}
	err := manager.Register(dep)
	if err != nil {
		t.Errorf("Register dependency error = %v", err)
	}

	// Register plugin with dependency
	plugin := &MockPlugin{
		metadata: Metadata{
			Name:         "main-plugin",
			Dependencies: []string{"dep-plugin"},
		},
	}
	err = manager.Register(plugin)
	if err != nil {
		t.Errorf("Register with dependency error = %v", err)
	}
}

func TestManager_RegisterMissingDependency(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	plugin := &MockPlugin{
		metadata: Metadata{
			Name:         "main-plugin",
			Dependencies: []string{"missing-dep"},
		},
	}

	err := manager.Register(plugin)
	if err == nil {
		t.Error("Expected error for missing dependency")
	}
}

func TestManager_Initialize(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	plugin := &MockPlugin{
		metadata: Metadata{
			Name: "test-plugin",
		},
	}

	err := manager.Register(plugin)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	err = manager.Initialize()
	if err != nil {
		t.Errorf("Initialize() error = %v", err)
	}

	if !plugin.initCalled {
		t.Error("Plugin Initialize was not called")
	}

	if !manager.loaded["test-plugin"] {
		t.Error("Plugin was not marked as loaded")
	}
}

func TestManager_InitializeError(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	plugin := &MockPlugin{
		metadata: Metadata{
			Name: "test-plugin",
		},
		initErr: errors.New("init failed"),
	}

	err := manager.Register(plugin)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	err = manager.Initialize()
	if err == nil {
		t.Error("Expected error from Initialize")
	}
}

func TestManager_Start(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	plugin := &MockPlugin{
		metadata: Metadata{
			Name: "test-plugin",
		},
	}

	err := manager.Register(plugin)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	err = manager.Initialize()
	if err != nil {
		t.Errorf("Initialize() error = %v", err)
	}

	ctx := context.Background()
	err = manager.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}
}

func TestManager_Stop(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	plugin := &MockPlugin{
		metadata: Metadata{
			Name: "test-plugin",
		},
	}

	err := manager.Register(plugin)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	err = manager.Initialize()
	if err != nil {
		t.Errorf("Initialize() error = %v", err)
	}

	ctx := context.Background()
	err = manager.Start(ctx)
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	err = manager.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

func TestManager_ExecuteHook(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	hookExecuted := false
	plugin := &MockPlugin{
		metadata: Metadata{
			Name: "test-plugin",
		},
		hooks: []Hook{
			{
				Name:     "test-hook",
				Priority: 1,
				Handler: func(ctx context.Context, data interface{}) error {
					hookExecuted = true
					return nil
				},
			},
		},
	}

	err := manager.Register(plugin)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	ctx := context.Background()
	err = manager.ExecuteHook(ctx, "test-hook", nil)
	if err != nil {
		t.Errorf("ExecuteHook() error = %v", err)
	}

	if !hookExecuted {
		t.Error("Hook was not executed")
	}
}

func TestManager_GetCommands(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	plugin := &MockPlugin{
		metadata: Metadata{
			Name: "test-plugin",
		},
		commands: []Command{
			{Name: "cmd1", Description: "Command 1"},
			{Name: "cmd2", Description: "Command 2"},
		},
	}

	err := manager.Register(plugin)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	commands := manager.GetCommands()
	if len(commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(commands))
	}
}

func TestManager_GetRoutes(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	plugin := &MockPlugin{
		metadata: Metadata{
			Name: "test-plugin",
		},
		routes: []Route{
			{Method: "GET", Path: "/route1"},
			{Method: "POST", Path: "/route2"},
		},
	}

	err := manager.Register(plugin)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	routes := manager.GetRoutes()
	if len(routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routes))
	}
}

func TestManager_GetMiddleware(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	plugin := &MockPlugin{
		metadata: Metadata{
			Name: "test-plugin",
		},
		middleware: []Middleware{
			{Name: "middleware1", Priority: 1},
			{Name: "middleware2", Priority: 2},
		},
	}

	err := manager.Register(plugin)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	middleware := manager.GetMiddleware()
	if len(middleware) != 2 {
		t.Errorf("Expected 2 middleware, got %d", len(middleware))
	}
}

func TestManager_GetCommand(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	testCmd := Command{
		Name:        "test",
		Description: "Test command",
	}

	plugin := &MockPlugin{
		metadata: Metadata{
			Name: "test-plugin",
		},
		commands: []Command{testCmd},
	}

	err := manager.Register(plugin)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	cmd, exists := manager.GetCommand("test")
	if !exists {
		t.Error("Command not found")
	}

	if cmd.Name != testCmd.Name {
		t.Errorf("Expected command name %s, got %s", testCmd.Name, cmd.Name)
	}
}

func TestManager_GetPlugin(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	plugin := &MockPlugin{
		metadata: Metadata{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
	}

	err := manager.Register(plugin)
	if err != nil {
		t.Errorf("Register() error = %v", err)
	}

	// Test getting existing plugin
	p, exists := manager.GetPlugin("test-plugin")
	if !exists {
		t.Error("Plugin should exist")
	}
	if p == nil {
		t.Error("Plugin should not be nil")
	}

	// Test getting non-existent plugin
	_, exists = manager.GetPlugin("non-existent")
	if exists {
		t.Error("Non-existent plugin should not exist")
	}
}

func TestManager_List(t *testing.T) {
	app := &MockApplication{}
	manager := NewManager(app)

	plugin1 := &MockPlugin{
		metadata: Metadata{
			Name:    "plugin1",
			Version: "1.0.0",
		},
	}
	plugin2 := &MockPlugin{
		metadata: Metadata{
			Name:    "plugin2",
			Version: "2.0.0",
		},
	}

	manager.Register(plugin1)
	manager.Register(plugin2)

	metadataList := manager.List()
	if len(metadataList) != 2 {
		t.Errorf("Expected 2 plugins, got %d", len(metadataList))
	}
}

func TestHook(t *testing.T) {
	hook := Hook{
		Name:     "test-hook",
		Priority: 10,
		Handler: func(ctx context.Context, data interface{}) error {
			return nil
		},
	}

	if hook.Name != "test-hook" {
		t.Errorf("Expected hook name test-hook, got %s", hook.Name)
	}

	if hook.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", hook.Priority)
	}

	if hook.Handler == nil {
		t.Error("Handler should not be nil")
	}
}

func TestCommand(t *testing.T) {
	cmd := Command{
		Name:        "test",
		Description: "Test command",
		Usage:       "test [flags]",
		Flags: []Flag{
			{
				Name:     "verbose",
				Short:    "v",
				Usage:    "Enable verbose output",
				Default:  false,
				Required: false,
			},
		},
	}

	if cmd.Name != "test" {
		t.Errorf("Expected command name test, got %s", cmd.Name)
	}

	if len(cmd.Flags) != 1 {
		t.Errorf("Expected 1 flag, got %d", len(cmd.Flags))
	}

	if cmd.Flags[0].Short != "v" {
		t.Errorf("Expected flag short v, got %s", cmd.Flags[0].Short)
	}
}

func TestCommandContext(t *testing.T) {
	ctx := CommandContext{
		Args:  []string{"arg1", "arg2"},
		Flags: map[string]interface{}{"verbose": true},
		App:   &MockApplication{},
	}

	if len(ctx.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(ctx.Args))
	}

	if verbose, ok := ctx.Flags["verbose"].(bool); !ok || !verbose {
		t.Error("Verbose flag not set correctly")
	}

	if ctx.App == nil {
		t.Error("App should not be nil")
	}
}

func TestRoute(t *testing.T) {
	route := Route{
		Method: "GET",
		Path:   "/test",
		Handler: func(ctx gor.Context) error {
			return nil
		},
	}

	if route.Method != "GET" {
		t.Errorf("Expected method GET, got %s", route.Method)
	}

	if route.Path != "/test" {
		t.Errorf("Expected path /test, got %s", route.Path)
	}

	if route.Handler == nil {
		t.Error("Handler should not be nil")
	}
}

func TestMiddleware(t *testing.T) {
	middleware := Middleware{
		Name:     "test-middleware",
		Priority: 5,
		Handler: func(next gor.HandlerFunc) gor.HandlerFunc {
			return func(ctx *gor.Context) error {
				return next(ctx)
			}
		},
	}

	if middleware.Name != "test-middleware" {
		t.Errorf("Expected middleware name test-middleware, got %s", middleware.Name)
	}

	if middleware.Priority != 5 {
		t.Errorf("Expected priority 5, got %d", middleware.Priority)
	}

	if middleware.Handler == nil {
		t.Error("Handler should not be nil")
	}
}

// Benchmark tests
func BenchmarkManager_Register(b *testing.B) {
	app := &MockApplication{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager := NewManager(app)
		plugin := &MockPlugin{
			metadata: Metadata{
				Name: "test-plugin",
			},
		}
		_ = manager.Register(plugin)
	}
}

func BenchmarkManager_ExecuteHook(b *testing.B) {
	app := &MockApplication{}
	manager := NewManager(app)

	plugin := &MockPlugin{
		metadata: Metadata{
			Name: "test-plugin",
		},
		hooks: []Hook{
			{
				Name:     "test-hook",
				Priority: 1,
				Handler: func(ctx context.Context, data interface{}) error {
					return nil
				},
			},
		},
	}

	_ = manager.Register(plugin)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.ExecuteHook(ctx, "test-hook", nil)
	}
}
