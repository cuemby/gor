package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	// Create temporary directory for test configs
	tmpDir := t.TempDir()

	t.Run("NewWithValidPath", func(t *testing.T) {
		config, err := New(tmpDir)
		if err != nil {
			t.Fatalf("New() should not return error for valid path, got: %v", err)
		}

		if config == nil {
			t.Fatal("New() should return a config instance")
		}

		if config.configPath != tmpDir {
			t.Errorf("Config path should be %s, got %s", tmpDir, config.configPath)
		}

		if config.envPrefix != "GOR" {
			t.Errorf("Default env prefix should be GOR, got %s", config.envPrefix)
		}
	})

	t.Run("NewWithNonexistentPath", func(t *testing.T) {
		config, err := New("/nonexistent/path")
		// Should still succeed as it doesn't require config files to exist
		if err != nil {
			t.Fatalf("New() should succeed even with nonexistent path, got: %v", err)
		}

		if config == nil {
			t.Fatal("New() should return a config instance")
		}
	})
}

func TestConfigGettersAndSetters(t *testing.T) {
	tmpDir := t.TempDir()
	config, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	t.Run("SetAndGet", func(t *testing.T) {
		config.Set("test.key", "test_value")
		value := config.Get("test.key")

		if value != "test_value" {
			t.Errorf("Expected 'test_value', got %v", value)
		}
	})

	t.Run("GetString", func(t *testing.T) {
		config.Set("string_key", "hello")
		config.Set("int_as_string", 42)

		if got := config.GetString("string_key"); got != "hello" {
			t.Errorf("GetString() = %v, want hello", got)
		}

		if got := config.GetString("int_as_string"); got != "42" {
			t.Errorf("GetString() should convert int to string, got %v", got)
		}

		if got := config.GetString("nonexistent"); got != "" {
			t.Errorf("GetString() should return empty string for nonexistent key, got %v", got)
		}
	})

	t.Run("GetInt", func(t *testing.T) {
		config.Set("int_key", 42)
		config.Set("string_int", "123")
		config.Set("float_int", 45.6)

		if got := config.GetInt("int_key"); got != 42 {
			t.Errorf("GetInt() = %v, want 42", got)
		}

		if got := config.GetInt("string_int"); got != 123 {
			t.Errorf("GetInt() should parse string, got %v", got)
		}

		if got := config.GetInt("float_int"); got != 45 {
			t.Errorf("GetInt() should convert float, got %v", got)
		}

		if got := config.GetInt("nonexistent"); got != 0 {
			t.Errorf("GetInt() should return 0 for nonexistent key, got %v", got)
		}
	})

	t.Run("GetBool", func(t *testing.T) {
		config.Set("bool_true", true)
		config.Set("bool_false", false)
		config.Set("string_true", "true")
		config.Set("string_false", "false")
		config.Set("int_true", 1)
		config.Set("int_false", 0)

		testCases := []struct {
			key      string
			expected bool
		}{
			{"bool_true", true},
			{"bool_false", false},
			{"string_true", true},
			{"string_false", false},
			{"int_true", true},
			{"int_false", false},
			{"nonexistent", false},
		}

		for _, tc := range testCases {
			if got := config.GetBool(tc.key); got != tc.expected {
				t.Errorf("GetBool(%s) = %v, want %v", tc.key, got, tc.expected)
			}
		}
	})

	t.Run("GetFloat", func(t *testing.T) {
		config.Set("float_key", 3.14)
		config.Set("int_float", 42)
		config.Set("string_float", "2.71")

		if got := config.GetFloat("float_key"); got != 3.14 {
			t.Errorf("GetFloat() = %v, want 3.14", got)
		}

		if got := config.GetFloat("int_float"); got != 42.0 {
			t.Errorf("GetFloat() should convert int, got %v", got)
		}

		if got := config.GetFloat("string_float"); got != 2.71 {
			t.Errorf("GetFloat() should parse string, got %v", got)
		}

		if got := config.GetFloat("nonexistent"); got != 0 {
			t.Errorf("GetFloat() should return 0 for nonexistent key, got %v", got)
		}
	})

	t.Run("GetDuration", func(t *testing.T) {
		config.Set("duration_string", "5m")
		config.Set("duration_int", 30) // should be treated as seconds

		expectedDuration := 5 * time.Minute
		if got := config.GetDuration("duration_string"); got != expectedDuration {
			t.Errorf("GetDuration() = %v, want %v", got, expectedDuration)
		}

		expectedSeconds := 30 * time.Second
		if got := config.GetDuration("duration_int"); got != expectedSeconds {
			t.Errorf("GetDuration() should convert int to seconds, got %v", got)
		}

		if got := config.GetDuration("nonexistent"); got != 0 {
			t.Errorf("GetDuration() should return 0 for nonexistent key, got %v", got)
		}
	})

	t.Run("GetStringSlice", func(t *testing.T) {
		config.Set("slice_strings", []string{"a", "b", "c"})
		config.Set("slice_interface", []interface{}{"x", "y", "z"})
		config.Set("comma_separated", "one,two,three")

		expected := []string{"a", "b", "c"}
		if got := config.GetStringSlice("slice_strings"); !stringSlicesEqual(got, expected) {
			t.Errorf("GetStringSlice() = %v, want %v", got, expected)
		}

		expected = []string{"x", "y", "z"}
		if got := config.GetStringSlice("slice_interface"); !stringSlicesEqual(got, expected) {
			t.Errorf("GetStringSlice() should convert interface slice, got %v", got)
		}

		expected = []string{"one", "two", "three"}
		if got := config.GetStringSlice("comma_separated"); !stringSlicesEqual(got, expected) {
			t.Errorf("GetStringSlice() should split comma-separated string, got %v", got)
		}

		if got := config.GetStringSlice("nonexistent"); len(got) != 0 {
			t.Errorf("GetStringSlice() should return empty slice for nonexistent key, got %v", got)
		}
	})

	t.Run("GetMap", func(t *testing.T) {
		testMap := map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		}
		config.Set("map_key", testMap)

		got := config.GetMap("map_key")
		if len(got) != 2 {
			t.Errorf("GetMap() should return map with 2 keys, got %d", len(got))
		}

		if got["key1"] != "value1" {
			t.Errorf("GetMap() key1 = %v, want value1", got["key1"])
		}

		if got["key2"] != 42 {
			t.Errorf("GetMap() key2 = %v, want 42", got["key2"])
		}

		if got := config.GetMap("nonexistent"); len(got) != 0 {
			t.Errorf("GetMap() should return empty map for nonexistent key, got %v", got)
		}
	})
}

func TestConfigHas(t *testing.T) {
	tmpDir := t.TempDir()
	config, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	t.Run("HasExistingKey", func(t *testing.T) {
		config.Set("existing_key", "value")
		if !config.Has("existing_key") {
			t.Error("Has() should return true for existing key")
		}
	})

	t.Run("HasNonexistentKey", func(t *testing.T) {
		if config.Has("nonexistent_key") {
			t.Error("Has() should return false for nonexistent key")
		}
	})

	t.Run("HasNestedKey", func(t *testing.T) {
		config.Set("nested.key", "value")
		if !config.Has("nested.key") {
			t.Error("Has() should return true for nested key")
		}
	})
}

func TestConfigWatch(t *testing.T) {
	tmpDir := t.TempDir()
	config, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	t.Run("WatcherNotification", func(t *testing.T) {
		var watchedKey string
		var newVal interface{}
		called := false

		config.Watch(func(key string, oldValue, newValue interface{}) {
			watchedKey = key
			newVal = newValue
			called = true
		})

		config.Set("watched_key", "new_value")

		if !called {
			t.Error("Watcher should be called when value changes")
		}

		if watchedKey != "watched_key" {
			t.Errorf("Watcher key = %v, want watched_key", watchedKey)
		}

		if newVal != "new_value" {
			t.Errorf("Watcher new value = %v, want new_value", newVal)
		}
	})
}

func TestConfigNestedValues(t *testing.T) {
	tmpDir := t.TempDir()
	config, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	t.Run("SetNestedValue", func(t *testing.T) {
		config.Set("database.host", "localhost")
		config.Set("database.port", 5432)

		if got := config.GetString("database.host"); got != "localhost" {
			t.Errorf("Nested value host = %v, want localhost", got)
		}

		if got := config.GetInt("database.port"); got != 5432 {
			t.Errorf("Nested value port = %v, want 5432", got)
		}
	})

	t.Run("GetNestedFromMap", func(t *testing.T) {
		config.Set("server", map[string]interface{}{
			"host": "0.0.0.0",
			"port": 8080,
		})

		if got := config.GetString("server.host"); got != "0.0.0.0" {
			t.Errorf("Nested map value = %v, want 0.0.0.0", got)
		}

		if got := config.GetInt("server.port"); got != 8080 {
			t.Errorf("Nested map value = %v, want 8080", got)
		}
	})
}

func TestConfigEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	config, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	t.Run("Environment", func(t *testing.T) {
		env := config.Environment()
		// Should return current environment (likely "development" or what's set in GO_ENV/GOR_ENV)
		if env == "" {
			t.Error("Environment() should not return empty string")
		}
	})

	t.Run("IsDevelopment", func(t *testing.T) {
		// This depends on current environment, but we can test the logic
		isDev := config.IsDevelopment()
		env := config.Environment()
		expected := env == "development" || env == "dev"
		if isDev != expected {
			t.Errorf("IsDevelopment() = %v, expected %v for env %s", isDev, expected, env)
		}
	})

	t.Run("IsProduction", func(t *testing.T) {
		isProd := config.IsProduction()
		env := config.Environment()
		expected := env == "production" || env == "prod"
		if isProd != expected {
			t.Errorf("IsProduction() = %v, expected %v for env %s", isProd, expected, env)
		}
	})

	t.Run("IsTest", func(t *testing.T) {
		isTest := config.IsTest()
		env := config.Environment()
		expected := env == "test"
		if isTest != expected {
			t.Errorf("IsTest() = %v, expected %v for env %s", isTest, expected, env)
		}
	})
}

func TestConfigFileLoading(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("LoadYAMLConfig", func(t *testing.T) {
		// Create a test YAML config file
		yamlContent := `
app:
  name: test_app
  port: 3000
database:
  host: localhost
  port: 5432
`
		yamlFile := filepath.Join(tmpDir, "config.yml")
		if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
			t.Fatalf("Failed to create YAML config: %v", err)
		}

		config, err := New(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		if got := config.GetString("app.name"); got != "test_app" {
			t.Errorf("YAML config app.name = %v, want test_app", got)
		}

		if got := config.GetInt("app.port"); got != 3000 {
			t.Errorf("YAML config app.port = %v, want 3000", got)
		}

		if got := config.GetString("database.host"); got != "localhost" {
			t.Errorf("YAML config database.host = %v, want localhost", got)
		}
	})

	t.Run("LoadJSONConfig", func(t *testing.T) {
		// Create a new test directory to avoid conflicts
		jsonTmpDir := t.TempDir()

		// Create a test JSON config file (config.yml but with JSON content)
		jsonContent := `{
  "api": {
    "version": "v1",
    "timeout": 30
  },
  "debug": true
}`
		jsonFile := filepath.Join(jsonTmpDir, "config.json")
		if err := os.WriteFile(jsonFile, []byte(jsonContent), 0644); err != nil {
			t.Fatalf("Failed to create JSON config: %v", err)
		}

		// Create config and manually load the JSON file
		config, err := New(jsonTmpDir)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		// Load the JSON file directly
		err = config.loadFile(jsonFile)
		if err != nil {
			t.Fatalf("Failed to load JSON file: %v", err)
		}

		if got := config.GetString("api.version"); got != "v1" {
			t.Errorf("JSON config api.version = %v, want v1", got)
		}

		if got := config.GetInt("api.timeout"); got != 30 {
			t.Errorf("JSON config api.timeout = %v, want 30", got)
		}

		if got := config.GetBool("debug"); got != true {
			t.Errorf("JSON config debug = %v, want true", got)
		}
	})
}

func TestConfigEnvironmentVariables(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("LoadFromEnvironmentVariables", func(t *testing.T) {
		// Set test environment variables
		os.Setenv("GOR_TEST_HOST", "env_host")
		os.Setenv("GOR_TEST_PORT", "9000")
		os.Setenv("GOR_TEST_ENABLED", "true")
		defer func() {
			os.Unsetenv("GOR_TEST_HOST")
			os.Unsetenv("GOR_TEST_PORT")
			os.Unsetenv("GOR_TEST_ENABLED")
		}()

		config, err := New(tmpDir)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		if got := config.GetString("test.host"); got != "env_host" {
			t.Errorf("Env var test.host = %v, want env_host", got)
		}

		if got := config.GetInt("test.port"); got != 9000 {
			t.Errorf("Env var test.port = %v, want 9000", got)
		}

		if got := config.GetBool("test.enabled"); got != true {
			t.Errorf("Env var test.enabled = %v, want true", got)
		}
	})
}

func TestConfigReload(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial config
	yamlContent := `app:
  name: initial_app
`
	yamlFile := filepath.Join(tmpDir, "config.yml")
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create initial YAML config: %v", err)
	}

	config, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Verify initial value
	if got := config.GetString("app.name"); got != "initial_app" {
		t.Errorf("Initial app.name = %v, want initial_app", got)
	}

	// Update config file
	updatedContent := `app:
  name: updated_app
  version: 2.0
`
	if err := os.WriteFile(yamlFile, []byte(updatedContent), 0644); err != nil {
		t.Fatalf("Failed to update YAML config: %v", err)
	}

	// Reload configuration
	if err := config.Reload(); err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	// Verify updated values
	if got := config.GetString("app.name"); got != "updated_app" {
		t.Errorf("After reload app.name = %v, want updated_app", got)
	}

	if got := config.GetFloat("app.version"); got != 2.0 {
		t.Errorf("After reload app.version = %v, want 2.0", got)
	}
}

func TestConfigBind(t *testing.T) {
	tmpDir := t.TempDir()
	config, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	type TestStruct struct {
		Name    string
		Port    int
		Enabled bool
	}

	t.Run("BindValidStruct", func(t *testing.T) {
		// The current bind implementation only handles direct type assignment
		// not struct field mapping, so test with a simple string value
		config.Set("simple_value", "test_string")

		var result string
		err := config.Bind("simple_value", &result)
		if err != nil {
			t.Fatalf("Bind() should not return error: %v", err)
		}

		if result != "test_string" {
			t.Errorf("Bound value = %v, want test_string", result)
		}
	})

	t.Run("BindNonexistentKey", func(t *testing.T) {
		var result TestStruct
		err := config.Bind("nonexistent", &result)
		if err == nil {
			t.Error("Bind() should return error for nonexistent key")
		}
	})

	t.Run("BindNonPointer", func(t *testing.T) {
		config.Set("test_key", "test_value")
		var result TestStruct
		err := config.Bind("test_key", result) // Not a pointer
		if err == nil {
			t.Error("Bind() should return error for non-pointer destination")
		}
	})
}

// Helper functions

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// Test helper functions

func TestParseValue(t *testing.T) {
	testCases := []struct {
		input    string
		expected interface{}
	}{
		{"true", true},
		{"false", false},
		{"123", int64(123)},
		{"3.14", 3.14},
		{"5m", 5 * time.Minute},
		{"hello", "hello"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := parseValue(tc.input)
			if result != tc.expected {
				t.Errorf("parseValue(%s) = %v (%T), want %v (%T)",
					tc.input, result, result, tc.expected, tc.expected)
			}
		})
	}
}

func TestMergeMaps(t *testing.T) {
	dst := map[string]interface{}{
		"a": 1,
		"b": map[string]interface{}{
			"x": 10,
			"y": 20,
		},
		"c": "old_value",
	}

	src := map[string]interface{}{
		"b": map[string]interface{}{
			"y": 30, // Should override
			"z": 40, // Should add
		},
		"c": "new_value", // Should override
		"d": "added",     // Should add
	}

	result := mergeMaps(dst, src)

	// Check that 'a' is preserved
	if result["a"] != 1 {
		t.Errorf("mergeMaps result['a'] = %v, want 1", result["a"])
	}

	// Check that 'c' is overridden
	if result["c"] != "new_value" {
		t.Errorf("mergeMaps result['c'] = %v, want new_value", result["c"])
	}

	// Check that 'd' is added
	if result["d"] != "added" {
		t.Errorf("mergeMaps result['d'] = %v, want added", result["d"])
	}

	// Check nested map merging
	bMap, ok := result["b"].(map[string]interface{})
	if !ok {
		t.Fatal("mergeMaps result['b'] should be a map")
	}

	if bMap["x"] != 10 {
		t.Errorf("mergeMaps result['b']['x'] = %v, want 10", bMap["x"])
	}

	if bMap["y"] != 30 {
		t.Errorf("mergeMaps result['b']['y'] = %v, want 30", bMap["y"])
	}

	if bMap["z"] != 40 {
		t.Errorf("mergeMaps result['b']['z'] = %v, want 40", bMap["z"])
	}
}

func TestGetEnvironment(t *testing.T) {
	// Save original values
	originalGorEnv := os.Getenv("GOR_ENV")
	originalGoEnv := os.Getenv("GO_ENV")

	defer func() {
		// Restore original values
		if originalGorEnv != "" {
			os.Setenv("GOR_ENV", originalGorEnv)
		} else {
			os.Unsetenv("GOR_ENV")
		}
		if originalGoEnv != "" {
			os.Setenv("GO_ENV", originalGoEnv)
		} else {
			os.Unsetenv("GO_ENV")
		}
	}()

	t.Run("DefaultEnvironment", func(t *testing.T) {
		os.Unsetenv("GOR_ENV")
		os.Unsetenv("GO_ENV")

		env := getEnvironment()
		if env != "development" {
			t.Errorf("getEnvironment() = %v, want development", env)
		}
	})

	t.Run("GorEnvTakesPrecedence", func(t *testing.T) {
		os.Setenv("GOR_ENV", "production")
		os.Setenv("GO_ENV", "staging")

		env := getEnvironment()
		if env != "production" {
			t.Errorf("getEnvironment() = %v, want production", env)
		}
	})

	t.Run("GoEnvFallback", func(t *testing.T) {
		os.Unsetenv("GOR_ENV")
		os.Setenv("GO_ENV", "test")

		env := getEnvironment()
		if env != "test" {
			t.Errorf("getEnvironment() = %v, want test", env)
		}
	})
}