package dev

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Watcher watches for file changes and triggers rebuilds
type Watcher struct {
	root          string
	excludePaths  []string
	includeExts   []string
	buildCmd      string
	runCmd        string
	process       *exec.Cmd
	lastModTimes  map[string]time.Time
	mu            sync.RWMutex
	debounceDelay time.Duration
	debounceTimer *time.Timer
	logger        *log.Logger
}

// NewWatcher creates a new file watcher
func NewWatcher(root string) *Watcher {
	return &Watcher{
		root: root,
		excludePaths: []string{
			".git",
			"node_modules",
			"vendor",
			"tmp",
			"log",
			".gor",
		},
		includeExts: []string{
			".go",
			".html",
			".css",
			".js",
			".json",
			".yml",
			".yaml",
			".env",
		},
		buildCmd:      "go build -o tmp/main ./cmd/app",
		runCmd:        "./tmp/main",
		lastModTimes:  make(map[string]time.Time),
		debounceDelay: 500 * time.Millisecond,
		logger:        log.New(os.Stdout, "[watcher] ", log.LstdFlags),
	}
}

// Start starts watching for file changes
func (w *Watcher) Start() error {
	w.logger.Println("Starting file watcher...")

	// Initial build and run
	if err := w.rebuild(); err != nil {
		return fmt.Errorf("initial build failed: %w", err)
	}

	// Start watching
	go w.watch()

	// Keep the watcher running
	select {}
}

// watch monitors for file changes
func (w *Watcher) watch() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		changed := false

		err := filepath.WalkDir(w.root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			// Skip excluded paths
			if w.shouldExclude(path) {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip directories
			if d.IsDir() {
				return nil
			}

			// Check file extension
			if !w.shouldInclude(path) {
				return nil
			}

			// Check modification time
			info, err := d.Info()
			if err != nil {
				return nil
			}

			modTime := info.ModTime()

			w.mu.RLock()
			lastMod, exists := w.lastModTimes[path]
			w.mu.RUnlock()

			if !exists || modTime.After(lastMod) {
				w.mu.Lock()
				w.lastModTimes[path] = modTime
				w.mu.Unlock()

				if exists {
					w.logger.Printf("File changed: %s\n", path)
					changed = true
				}
			}

			return nil
		})

		if err != nil {
			w.logger.Printf("Error walking directory: %v\n", err)
		}

		if changed {
			w.triggerRebuild()
		}
	}
}

// triggerRebuild triggers a rebuild with debouncing
func (w *Watcher) triggerRebuild() {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Cancel existing timer
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}

	// Set new timer
	w.debounceTimer = time.AfterFunc(w.debounceDelay, func() {
		if err := w.rebuild(); err != nil {
			w.logger.Printf("Rebuild failed: %v\n", err)
		}
	})
}

// rebuild rebuilds and restarts the application
func (w *Watcher) rebuild() error {
	w.logger.Println("Rebuilding...")

	// Stop existing process
	if w.process != nil {
		w.logger.Println("Stopping existing process...")
		if err := w.process.Process.Kill(); err != nil {
			w.logger.Printf("Failed to kill process: %v\n", err)
		}
		w.process.Wait()
		w.process = nil
	}

	// Build
	w.logger.Println("Building...")
	buildCmd := exec.Command("sh", "-c", w.buildCmd)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Run
	w.logger.Println("Starting application...")
	w.process = exec.Command("sh", "-c", w.runCmd)
	w.process.Stdout = os.Stdout
	w.process.Stderr = os.Stderr

	if err := w.process.Start(); err != nil {
		return fmt.Errorf("failed to start application: %w", err)
	}

	w.logger.Printf("Application started (PID: %d)\n", w.process.Process.Pid)
	return nil
}

// shouldExclude checks if a path should be excluded
func (w *Watcher) shouldExclude(path string) bool {
	for _, exclude := range w.excludePaths {
		if strings.Contains(path, exclude) {
			return true
		}
	}
	return false
}

// shouldInclude checks if a file should be included
func (w *Watcher) shouldInclude(path string) bool {
	for _, ext := range w.includeExts {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// SetBuildCommand sets the build command
func (w *Watcher) SetBuildCommand(cmd string) {
	w.buildCmd = cmd
}

// SetRunCommand sets the run command
func (w *Watcher) SetRunCommand(cmd string) {
	w.runCmd = cmd
}

// AddExcludePath adds a path to exclude from watching
func (w *Watcher) AddExcludePath(path string) {
	w.excludePaths = append(w.excludePaths, path)
}

// AddIncludeExt adds a file extension to include in watching
func (w *Watcher) AddIncludeExt(ext string) {
	w.includeExts = append(w.includeExts, ext)
}
