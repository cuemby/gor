package dev

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	watcher := NewWatcher("/test/path")

	if watcher == nil {
		t.Fatal("NewWatcher returned nil")
	}

	if watcher.root != "/test/path" {
		t.Errorf("Expected root /test/path, got %s", watcher.root)
	}

	// Check default exclude paths
	expectedExcludes := []string{".git", "node_modules", "vendor", "tmp", "log", ".gor"}
	if len(watcher.excludePaths) != len(expectedExcludes) {
		t.Errorf("Expected %d exclude paths, got %d", len(expectedExcludes), len(watcher.excludePaths))
	}

	// Check default include extensions
	expectedExts := []string{".go", ".html", ".css", ".js", ".json", ".yml", ".yaml", ".env"}
	if len(watcher.includeExts) != len(expectedExts) {
		t.Errorf("Expected %d include extensions, got %d", len(expectedExts), len(watcher.includeExts))
	}

	if watcher.buildCmd != "go build -o tmp/main ./cmd/app" {
		t.Errorf("Unexpected build command: %s", watcher.buildCmd)
	}

	if watcher.runCmd != "./tmp/main" {
		t.Errorf("Unexpected run command: %s", watcher.runCmd)
	}

	if watcher.debounceDelay != 500*time.Millisecond {
		t.Errorf("Expected debounce delay 500ms, got %v", watcher.debounceDelay)
	}

	if watcher.lastModTimes == nil {
		t.Error("lastModTimes map not initialized")
	}

	if watcher.logger == nil {
		t.Error("Logger not initialized")
	}
}

func TestWatcher_ShouldExclude(t *testing.T) {
	watcher := NewWatcher("/test")

	tests := []struct {
		path     string
		expected bool
	}{
		{"/test/.git/config", true},
		{"/test/node_modules/package.json", true},
		{"/test/vendor/github.com/lib", true},
		{"/test/tmp/build", true},
		{"/test/log/development.log", true},
		{"/test/.gor/cache", true},
		{"/test/src/main.go", false},
		{"/test/app/views/index.html", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := watcher.shouldExclude(tt.path)
			if result != tt.expected {
				t.Errorf("shouldExclude(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestWatcher_ShouldInclude(t *testing.T) {
	watcher := NewWatcher("/test")

	tests := []struct {
		path     string
		expected bool
	}{
		{"main.go", true},
		{"index.html", true},
		{"style.css", true},
		{"app.js", true},
		{"config.json", true},
		{"docker-compose.yml", true},
		{"config.yaml", true},
		{".env", true},
		{"README.md", false},
		{"Makefile", false},
		{"script.sh", false},
		{"image.png", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := watcher.shouldInclude(tt.path)
			if result != tt.expected {
				t.Errorf("shouldInclude(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestWatcher_SetBuildCommand(t *testing.T) {
	watcher := NewWatcher("/test")

	newCmd := "go build -tags debug -o bin/app"
	watcher.SetBuildCommand(newCmd)

	if watcher.buildCmd != newCmd {
		t.Errorf("Expected build command %s, got %s", newCmd, watcher.buildCmd)
	}
}

func TestWatcher_SetRunCommand(t *testing.T) {
	watcher := NewWatcher("/test")

	newCmd := "./bin/app --debug"
	watcher.SetRunCommand(newCmd)

	if watcher.runCmd != newCmd {
		t.Errorf("Expected run command %s, got %s", newCmd, watcher.runCmd)
	}
}

func TestWatcher_AddExcludePath(t *testing.T) {
	watcher := NewWatcher("/test")

	initialCount := len(watcher.excludePaths)
	watcher.AddExcludePath("coverage")

	if len(watcher.excludePaths) != initialCount+1 {
		t.Error("Exclude path was not added")
	}

	found := false
	for _, path := range watcher.excludePaths {
		if path == "coverage" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Added exclude path 'coverage' not found")
	}
}

func TestWatcher_AddIncludeExt(t *testing.T) {
	watcher := NewWatcher("/test")

	initialCount := len(watcher.includeExts)
	watcher.AddIncludeExt(".tsx")

	if len(watcher.includeExts) != initialCount+1 {
		t.Error("Include extension was not added")
	}

	found := false
	for _, ext := range watcher.includeExts {
		if ext == ".tsx" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Added include extension '.tsx' not found")
	}
}

func TestWatcher_TriggerRebuild(t *testing.T) {
	watcher := NewWatcher("/test")

	// Override commands to prevent actual execution
	watcher.buildCmd = "echo 'build'"
	watcher.runCmd = "echo 'run'"

	// Test debouncing
	watcher.triggerRebuild()

	// First timer should be set
	if watcher.debounceTimer == nil {
		t.Error("Debounce timer not set")
	}

	// Trigger again immediately
	watcher.triggerRebuild()

	// Should have cancelled and reset timer
	// (Hard to test the actual cancellation without time.Sleep)
}

func TestWatcher_FileChangeDetection(t *testing.T) {
	// Create temporary test directory
	tmpDir, err := os.MkdirTemp("", "watcher_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	watcher := NewWatcher(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	// Record initial modification time
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}

	watcher.mu.Lock()
	watcher.lastModTimes[testFile] = info.ModTime()
	watcher.mu.Unlock()

	// Sleep to ensure different modification time
	time.Sleep(10 * time.Millisecond)

	// Modify the file
	if err := os.WriteFile(testFile, []byte("package main\n// changed"), 0644); err != nil {
		t.Fatal(err)
	}

	// Check if change would be detected
	newInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}

	watcher.mu.RLock()
	lastMod := watcher.lastModTimes[testFile]
	watcher.mu.RUnlock()

	if !newInfo.ModTime().After(lastMod) {
		t.Error("File modification time should be newer")
	}
}

func TestWatcher_ExcludePathsIntegration(t *testing.T) {
	// Create temporary test directory structure
	tmpDir, err := os.MkdirTemp("", "watcher_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create directories
	dirs := []string{
		filepath.Join(tmpDir, ".git"),
		filepath.Join(tmpDir, "node_modules"),
		filepath.Join(tmpDir, "src"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	watcher := NewWatcher(tmpDir)

	// Test that .git and node_modules are excluded
	gitPath := filepath.Join(tmpDir, ".git", "config")
	if !watcher.shouldExclude(gitPath) {
		t.Error(".git path should be excluded")
	}

	nodePath := filepath.Join(tmpDir, "node_modules", "package.json")
	if !watcher.shouldExclude(nodePath) {
		t.Error("node_modules path should be excluded")
	}

	// Test that src is not excluded
	srcPath := filepath.Join(tmpDir, "src", "main.go")
	if watcher.shouldExclude(srcPath) {
		t.Error("src path should not be excluded")
	}
}

func TestWatcher_ConcurrentAccess(t *testing.T) {
	watcher := NewWatcher("/test")

	// Test concurrent access to lastModTimes
	done := make(chan bool)

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			watcher.mu.Lock()
			watcher.lastModTimes[strings.Repeat("a", i)] = time.Now()
			watcher.mu.Unlock()
		}
		done <- true
	}()

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			watcher.mu.RLock()
			_ = len(watcher.lastModTimes)
			watcher.mu.RUnlock()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}

// Benchmark tests
func BenchmarkWatcher_ShouldExclude(b *testing.B) {
	watcher := NewWatcher("/test")
	testPath := "/test/node_modules/some/deep/path/file.js"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = watcher.shouldExclude(testPath)
	}
}

func BenchmarkWatcher_ShouldInclude(b *testing.B) {
	watcher := NewWatcher("/test")
	testPath := "/test/src/main.go"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = watcher.shouldInclude(testPath)
	}
}

func BenchmarkWatcher_TriggerRebuild(b *testing.B) {
	watcher := NewWatcher("/test")
	// Override commands to prevent actual execution
	watcher.buildCmd = "true"
	watcher.runCmd = "true"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		watcher.triggerRebuild()
		// Cancel timer to prevent buildup
		if watcher.debounceTimer != nil {
			watcher.debounceTimer.Stop()
		}
	}
}