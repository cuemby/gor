package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Config manages application configuration
type Config struct {
	data       map[string]interface{}
	envPrefix  string
	configPath string
	env        string
	mu         sync.RWMutex
	watchers   []func(key string, oldValue, newValue interface{})
}

// New creates a new configuration instance
func New(configPath string) (*Config, error) {
	c := &Config{
		data:       make(map[string]interface{}),
		envPrefix:  "GOR",
		configPath: configPath,
		env:        getEnvironment(),
		watchers:   make([]func(string, interface{}, interface{}), 0),
	}

	// Load configuration files
	if err := c.load(); err != nil {
		return nil, err
	}

	return c, nil
}

// load loads configuration from files and environment
func (c *Config) load() error {
	// Load base configuration
	baseFile := filepath.Join(c.configPath, "config.yml")
	if err := c.loadFile(baseFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Load environment-specific configuration
	envFile := filepath.Join(c.configPath, fmt.Sprintf("config.%s.yml", c.env))
	if err := c.loadFile(envFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Load local configuration (not committed to version control)
	localFile := filepath.Join(c.configPath, "config.local.yml")
	if err := c.loadFile(localFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Override with environment variables
	c.loadEnvironmentVariables()

	return nil
}

// loadFile loads configuration from a file
func (c *Config) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var config map[string]interface{}

	// Detect file format
	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse JSON config %s: %w", path, err)
		}
	case ".yml", ".yaml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse YAML config %s: %w", path, err)
		}
	default:
		return fmt.Errorf("unsupported config format: %s", ext)
	}

	// Merge configuration
	c.merge(config)

	return nil
}

// loadEnvironmentVariables loads configuration from environment variables
func (c *Config) loadEnvironmentVariables() {
	prefix := c.envPrefix + "_"

	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			continue
		}

		key := pair[0]
		value := pair[1]

		if !strings.HasPrefix(key, prefix) {
			continue
		}

		// Convert environment variable to config key
		// GOR_DATABASE_HOST -> database.host
		configKey := strings.ToLower(strings.TrimPrefix(key, prefix))
		configKey = strings.ReplaceAll(configKey, "_", ".")

		c.Set(configKey, parseValue(value))
	}
}

// merge merges configuration data
func (c *Config) merge(source map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, value := range source {
		if existing, ok := c.data[key]; ok {
			// Merge maps recursively
			if existingMap, ok := existing.(map[string]interface{}); ok {
				if valueMap, ok := value.(map[string]interface{}); ok {
					c.data[key] = mergeMaps(existingMap, valueMap)
					continue
				}
			}
		}
		c.data[key] = value
	}
}

// Get retrieves a configuration value
func (c *Config) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.getNestedValue(key)
}

// GetString retrieves a string configuration value
func (c *Config) GetString(key string) string {
	val := c.Get(key)
	if val == nil {
		return ""
	}

	switch v := val.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetInt retrieves an integer configuration value
func (c *Config) GetInt(key string) int {
	val := c.Get(key)
	if val == nil {
		return 0
	}

	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		i, _ := strconv.Atoi(v)
		return i
	default:
		return 0
	}
}

// GetBool retrieves a boolean configuration value
func (c *Config) GetBool(key string) bool {
	val := c.Get(key)
	if val == nil {
		return false
	}

	switch v := val.(type) {
	case bool:
		return v
	case string:
		b, _ := strconv.ParseBool(v)
		return b
	case int:
		return v != 0
	default:
		return false
	}
}

// GetFloat retrieves a float configuration value
func (c *Config) GetFloat(key string) float64 {
	val := c.Get(key)
	if val == nil {
		return 0
	}

	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return f
	default:
		return 0
	}
}

// GetDuration retrieves a duration configuration value
func (c *Config) GetDuration(key string) time.Duration {
	val := c.Get(key)
	if val == nil {
		return 0
	}

	switch v := val.(type) {
	case time.Duration:
		return v
	case string:
		d, _ := time.ParseDuration(v)
		return d
	case int:
		return time.Duration(v) * time.Second
	default:
		return 0
	}
}

// GetStringSlice retrieves a string slice configuration value
func (c *Config) GetStringSlice(key string) []string {
	val := c.Get(key)
	if val == nil {
		return []string{}
	}

	switch v := val.(type) {
	case []string:
		return v
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	case string:
		return strings.Split(v, ",")
	default:
		return []string{}
	}
}

// GetMap retrieves a map configuration value
func (c *Config) GetMap(key string) map[string]interface{} {
	val := c.Get(key)
	if val == nil {
		return map[string]interface{}{}
	}

	switch v := val.(type) {
	case map[string]interface{}:
		return v
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for k, v := range v {
			result[fmt.Sprintf("%v", k)] = v
		}
		return result
	default:
		return map[string]interface{}{}
	}
}

// Set sets a configuration value
func (c *Config) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldValue := c.getNestedValue(key)
	c.setNestedValue(key, value)

	// Notify watchers
	for _, watcher := range c.watchers {
		watcher(key, oldValue, value)
	}
}

// Has checks if a configuration key exists
func (c *Config) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.getNestedValue(key) != nil
}

// Watch registers a watcher for configuration changes
func (c *Config) Watch(fn func(key string, oldValue, newValue interface{})) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.watchers = append(c.watchers, fn)
}

// Reload reloads configuration from files
func (c *Config) Reload() error {
	c.mu.Lock()
	c.data = make(map[string]interface{})
	c.mu.Unlock()

	return c.load()
}

// Environment returns the current environment
func (c *Config) Environment() string {
	return c.env
}

// IsProduction returns true if in production environment
func (c *Config) IsProduction() bool {
	return c.env == "production" || c.env == "prod"
}

// IsDevelopment returns true if in development environment
func (c *Config) IsDevelopment() bool {
	return c.env == "development" || c.env == "dev"
}

// IsTest returns true if in test environment
func (c *Config) IsTest() bool {
	return c.env == "test"
}

// getNestedValue gets a nested value using dot notation
func (c *Config) getNestedValue(key string) interface{} {
	parts := strings.Split(key, ".")
	value := c.data

	for _, part := range parts {
		if m, ok := value[part]; ok {
			if mapValue, ok := m.(map[string]interface{}); ok && len(parts) > 1 {
				value = mapValue
				parts = parts[1:]
			} else {
				return m
			}
		} else {
			return nil
		}
	}

	return value
}

// setNestedValue sets a nested value using dot notation
func (c *Config) setNestedValue(key string, value interface{}) {
	parts := strings.Split(key, ".")

	if len(parts) == 1 {
		c.data[key] = value
		return
	}

	current := c.data
	for _, part := range parts[:len(parts)-1] {
		if _, ok := current[part]; !ok {
			current[part] = make(map[string]interface{})
		}

		if m, ok := current[part].(map[string]interface{}); ok {
			current = m
		} else {
			// Can't set nested value if parent is not a map
			return
		}
	}

	current[parts[len(parts)-1]] = value
}

// Helper functions

// getEnvironment gets the current environment
func getEnvironment() string {
	env := os.Getenv("GOR_ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}
	if env == "" {
		env = "development"
	}
	return env
}

// parseValue parses a string value to appropriate type
func parseValue(value string) interface{} {
	// Try boolean
	if b, err := strconv.ParseBool(value); err == nil {
		return b
	}

	// Try integer
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i
	}

	// Try float
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}

	// Try duration
	if d, err := time.ParseDuration(value); err == nil {
		return d
	}

	// Return as string
	return value
}

// mergeMaps recursively merges two maps
func mergeMaps(dst, src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy dst
	for k, v := range dst {
		result[k] = v
	}

	// Merge src
	for k, v := range src {
		if existing, ok := result[k]; ok {
			// Merge maps recursively
			if existingMap, ok := existing.(map[string]interface{}); ok {
				if srcMap, ok := v.(map[string]interface{}); ok {
					result[k] = mergeMaps(existingMap, srcMap)
					continue
				}
			}
		}
		result[k] = v
	}

	return result
}

// Bind binds configuration to a struct
func (c *Config) Bind(key string, dest interface{}) error {
	val := c.Get(key)
	if val == nil {
		return fmt.Errorf("configuration key %s not found", key)
	}

	// Use reflection to set struct fields
	return bindValue(val, dest)
}

// bindValue binds a value to a destination using reflection
func bindValue(src interface{}, dest interface{}) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}

	destValue = destValue.Elem()

	// Convert source to appropriate type
	srcValue := reflect.ValueOf(src)

	if !srcValue.Type().AssignableTo(destValue.Type()) {
		// Try to convert
		if srcValue.Type().ConvertibleTo(destValue.Type()) {
			srcValue = srcValue.Convert(destValue.Type())
		} else {
			return fmt.Errorf("cannot bind %v to %v", srcValue.Type(), destValue.Type())
		}
	}

	destValue.Set(srcValue)
	return nil
}
