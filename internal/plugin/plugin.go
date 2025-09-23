package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/cuemby/gor/pkg/gor"
)

// Plugin represents a Gor plugin
type Plugin interface {
	// Metadata returns plugin metadata
	Metadata() Metadata

	// Initialize initializes the plugin
	Initialize(app gor.Application) error

	// Start starts the plugin
	Start(ctx context.Context) error

	// Stop stops the plugin
	Stop(ctx context.Context) error

	// Hooks returns plugin hooks
	Hooks() []Hook

	// Commands returns CLI commands
	Commands() []Command

	// Routes returns HTTP routes
	Routes() []Route

	// Middleware returns HTTP middleware
	Middleware() []Middleware
}

// Metadata contains plugin metadata
type Metadata struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	License     string   `json:"license"`
	Dependencies []string `json:"dependencies"`
	Tags        []string `json:"tags"`
}

// Hook represents a plugin hook
type Hook struct {
	Name     string
	Priority int
	Handler  func(context.Context, interface{}) error
}

// Command represents a CLI command
type Command struct {
	Name        string
	Description string
	Usage       string
	Flags       []Flag
	Action      func(ctx *CommandContext) error
}

// Flag represents a command flag
type Flag struct {
	Name     string
	Short    string
	Usage    string
	Default  interface{}
	Required bool
}

// CommandContext contains command execution context
type CommandContext struct {
	Args  []string
	Flags map[string]interface{}
	App   gor.Application
}

// Route represents an HTTP route
type Route struct {
	Method  string
	Path    string
	Handler func(ctx gor.Context) error
}

// Middleware represents HTTP middleware
type Middleware struct {
	Name     string
	Priority int
	Handler  func(next gor.HandlerFunc) gor.HandlerFunc
}

// Manager manages plugins
type Manager struct {
	plugins   map[string]Plugin
	loaded    map[string]bool
	hooks     map[string][]Hook
	commands  map[string]Command
	routes    []Route
	middleware []Middleware
	app       gor.Application
	mu        sync.RWMutex
}

// NewManager creates a new plugin manager
func NewManager(app gor.Application) *Manager {
	return &Manager{
		plugins:    make(map[string]Plugin),
		loaded:     make(map[string]bool),
		hooks:      make(map[string][]Hook),
		commands:   make(map[string]Command),
		routes:     make([]Route, 0),
		middleware: make([]Middleware, 0),
		app:        app,
	}
}

// Register registers a plugin
func (m *Manager) Register(p Plugin) error {
	metadata := p.Metadata()

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[metadata.Name]; exists {
		return fmt.Errorf("plugin %s already registered", metadata.Name)
	}

	// Check dependencies
	for _, dep := range metadata.Dependencies {
		if _, exists := m.plugins[dep]; !exists {
			return fmt.Errorf("dependency %s not found for plugin %s", dep, metadata.Name)
		}
	}

	m.plugins[metadata.Name] = p

	// Register hooks
	for _, hook := range p.Hooks() {
		m.hooks[hook.Name] = append(m.hooks[hook.Name], hook)
	}

	// Register commands
	for _, cmd := range p.Commands() {
		m.commands[cmd.Name] = cmd
	}

	// Register routes
	m.routes = append(m.routes, p.Routes()...)

	// Register middleware
	m.middleware = append(m.middleware, p.Middleware()...)

	return nil
}

// Load loads a plugin from a file
func (m *Manager) Load(path string) error {
	// Load plugin using Go's plugin package
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to load plugin %s: %w", path, err)
	}

	// Look for the exported Plugin symbol
	symbol, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin %s does not export Plugin symbol: %w", path, err)
	}

	// Assert that it implements the Plugin interface
	gorPlugin, ok := symbol.(Plugin)
	if !ok {
		return fmt.Errorf("plugin %s does not implement Plugin interface", path)
	}

	// Register the plugin
	return m.Register(gorPlugin)
}

// LoadDirectory loads all plugins from a directory
func (m *Manager) LoadDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Check if it's a plugin file (.so on Unix, .dll on Windows)
		if filepath.Ext(entry.Name()) == ".so" || filepath.Ext(entry.Name()) == ".dll" {
			path := filepath.Join(dir, entry.Name())
			if err := m.Load(path); err != nil {
				return err
			}
		}
	}

	return nil
}

// Initialize initializes all plugins
func (m *Manager) Initialize() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, p := range m.plugins {
		if err := p.Initialize(m.app); err != nil {
			return fmt.Errorf("failed to initialize plugin %s: %w", name, err)
		}
		m.loaded[name] = true
	}

	return nil
}

// Start starts all plugins
func (m *Manager) Start(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, p := range m.plugins {
		if !m.loaded[name] {
			continue
		}

		if err := p.Start(ctx); err != nil {
			return fmt.Errorf("failed to start plugin %s: %w", name, err)
		}
	}

	return nil
}

// Stop stops all plugins
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var errors []error

	// Stop in reverse order
	for name, p := range m.plugins {
		if !m.loaded[name] {
			continue
		}

		if err := p.Stop(ctx); err != nil {
			errors = append(errors, fmt.Errorf("failed to stop plugin %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping plugins: %v", errors)
	}

	return nil
}

// ExecuteHook executes a hook
func (m *Manager) ExecuteHook(ctx context.Context, name string, data interface{}) error {
	m.mu.RLock()
	hooks := m.hooks[name]
	m.mu.RUnlock()

	// Sort hooks by priority
	sortHooksByPriority(hooks)

	for _, hook := range hooks {
		if err := hook.Handler(ctx, data); err != nil {
			return fmt.Errorf("hook %s failed: %w", name, err)
		}
	}

	return nil
}

// GetCommand gets a command
func (m *Manager) GetCommand(name string) (Command, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd, ok := m.commands[name]
	return cmd, ok
}

// GetCommands gets all commands
func (m *Manager) GetCommands() []Command {
	m.mu.RLock()
	defer m.mu.RUnlock()

	commands := make([]Command, 0, len(m.commands))
	for _, cmd := range m.commands {
		commands = append(commands, cmd)
	}

	return commands
}

// GetRoutes gets all routes
func (m *Manager) GetRoutes() []Route {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.routes
}

// GetMiddleware gets all middleware
func (m *Manager) GetMiddleware() []Middleware {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Sort middleware by priority
	sortMiddlewareByPriority(m.middleware)

	return m.middleware
}

// GetPlugin gets a plugin by name
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.plugins[name]
	return p, ok
}

// List lists all registered plugins
func (m *Manager) List() []Metadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metadata := make([]Metadata, 0, len(m.plugins))
	for _, p := range m.plugins {
		metadata = append(metadata, p.Metadata())
	}

	return metadata
}

// Helper functions

func sortHooksByPriority(hooks []Hook) {
	// Simple bubble sort for small arrays
	for i := 0; i < len(hooks); i++ {
		for j := i + 1; j < len(hooks); j++ {
			if hooks[i].Priority > hooks[j].Priority {
				hooks[i], hooks[j] = hooks[j], hooks[i]
			}
		}
	}
}

func sortMiddlewareByPriority(middleware []Middleware) {
	// Simple bubble sort for small arrays
	for i := 0; i < len(middleware); i++ {
		for j := i + 1; j < len(middleware); j++ {
			if middleware[i].Priority > middleware[j].Priority {
				middleware[i], middleware[j] = middleware[j], middleware[i]
			}
		}
	}
}

// BasePlugin provides a base implementation of Plugin
type BasePlugin struct {
	metadata   Metadata
	hooks      []Hook
	commands   []Command
	routes     []Route
	middleware []Middleware
}

// NewBasePlugin creates a new base plugin
func NewBasePlugin(metadata Metadata) *BasePlugin {
	return &BasePlugin{
		metadata:   metadata,
		hooks:      make([]Hook, 0),
		commands:   make([]Command, 0),
		routes:     make([]Route, 0),
		middleware: make([]Middleware, 0),
	}
}

// Metadata returns plugin metadata
func (p *BasePlugin) Metadata() Metadata {
	return p.metadata
}

// Initialize initializes the plugin
func (p *BasePlugin) Initialize(app gor.Application) error {
	return nil
}

// Start starts the plugin
func (p *BasePlugin) Start(ctx context.Context) error {
	return nil
}

// Stop stops the plugin
func (p *BasePlugin) Stop(ctx context.Context) error {
	return nil
}

// Hooks returns plugin hooks
func (p *BasePlugin) Hooks() []Hook {
	return p.hooks
}

// Commands returns CLI commands
func (p *BasePlugin) Commands() []Command {
	return p.commands
}

// Routes returns HTTP routes
func (p *BasePlugin) Routes() []Route {
	return p.routes
}

// Middleware returns HTTP middleware
func (p *BasePlugin) Middleware() []Middleware {
	return p.middleware
}

// AddHook adds a hook
func (p *BasePlugin) AddHook(hook Hook) {
	p.hooks = append(p.hooks, hook)
}

// AddCommand adds a command
func (p *BasePlugin) AddCommand(cmd Command) {
	p.commands = append(p.commands, cmd)
}

// AddRoute adds a route
func (p *BasePlugin) AddRoute(route Route) {
	p.routes = append(p.routes, route)
}

// AddMiddleware adds middleware
func (p *BasePlugin) AddMiddleware(mw Middleware) {
	p.middleware = append(p.middleware, mw)
}